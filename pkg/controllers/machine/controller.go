package machine

import (
	"admiralty.io/multicluster-scheduler/pkg/controller"
	multiclusterClientset "admiralty.io/multicluster-scheduler/pkg/generated/clientset/versioned"
	informers "admiralty.io/multicluster-scheduler/pkg/generated/informers/externalversions/multicluster/v1alpha1"
	listers "admiralty.io/multicluster-scheduler/pkg/generated/listers/multicluster/v1alpha1"
	"context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
	"pml.io/april/pkg/apis/platform/v1alpha1"
	"pml.io/april/pkg/config/agent"
	platformClientset "pml.io/april/pkg/generated/clientset/versioned"
	platforminformers "pml.io/april/pkg/generated/informers/externalversions/platform/v1alpha1"
	platformlisters "pml.io/april/pkg/generated/listers/platform/v1alpha1"
	_ "pml.io/april/pkg/platform/provider/baremetal/machine"
	machineprovider "pml.io/april/pkg/platform/provider/machine"
	innertypesv1 "pml.io/april/pkg/platform/provider/type"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"time"
)

const singletonName = "singleton"

type reconciler struct {
	kubeclientset         *kubernetes.Clientset
	platformClientset     platformClientset.Interface
	multiclusterclientset multiclusterClientset.Interface
	machineLister         platformlisters.MachineLister
	clusterSummaryListers map[string]listers.ClusterSummaryLister
}

// NewController returns a new Machine controller
func NewController(kubeclientset *kubernetes.Clientset,
	config *rest.Config,
	clusterSummaryInformers map[string]informers.ClusterSummaryInformer,
	machineInformer platforminformers.MachineInformer) *controller.Controller {

	platformClientset, err := platformClientset.NewForConfig(config)
	utilruntime.Must(err)

	multiClientset, err := multiclusterClientset.NewForConfig(config)
	utilruntime.Must(err)

	// 1. construct Machine Reconciler
	r := &reconciler{
		kubeclientset:         kubeclientset,
		platformClientset:     platformClientset,
		multiclusterclientset: multiClientset,
		machineLister:         machineInformer.Lister(),
		clusterSummaryListers: make(map[string]listers.ClusterSummaryLister, len(clusterSummaryInformers)),
	}

	//2. construct informer sync
	informersSynced := make([]cache.InformerSynced, len(clusterSummaryInformers)+1)
	informersSynced[0] = machineInformer.Informer().HasSynced
	i := 1
	for targetName, informer := range clusterSummaryInformers {
		r.clusterSummaryListers[targetName] = informer.Lister()
		informersSynced[i] = informer.Informer().HasSynced
		i++
	}

	//3. construct machine controller
	c := controller.New("machine-reconcile", r, informersSynced...)
	machineInformer.Informer().AddEventHandler(controller.HandleAllWith(func(obj interface{}) {
		machine := obj.(*v1alpha1.Machine)
		c.EnqueueKey(machine.Name)
	}))

	return c
}

func (r reconciler) Handle(key interface{}) (requeueAfter *time.Duration, err error) {
	ctx := context.Background()
	//1. Get Machine Object
	machineName := key.(string)
	machine, err := r.machineLister.Get(machineName)
	if err != nil {
		// The machine resource may no longer exist, in which case we stop processing.
		if errors.IsNotFound(err) {
			klog.Infof("machine '%s' in work queue no longer exists", key)
			// in this case, we'll handle the machine no longer.
			return nil, nil
		}
		return nil, err
	}
	//2. Add default setting.
	if machine.Status.Phase == "" {
		machine.Status.Phase = v1alpha1.MachineInitializing
	}
	//3. Schedule One cluster as the target to join into, Get the target cluster configuration.
	targetConfig, err := r.getTargetClusterConfig(ctx, machine)
	if err != nil {
		klog.Info("can't get target cluster or get one cluster so we need go to next loop")
		// In this case we'll handle the machine obj later
		return nil, err
	}
	//4. into handle chains
	switch machine.Status.Phase {
	case v1alpha1.MachineInitializing:
		err = r.onCreate(ctx, machine, targetConfig)
	case v1alpha1.MachineRunning, v1alpha1.MachineFailed, v1alpha1.MachineUpgrading:
		// TODO here. FIX ME FIX ME FIX ME!!!
		klog.Info("Now finished", " phase is ", machine.Status.Phase)
	case v1alpha1.MachineTerminating:
		// TODO something here.
		if err == nil {
			klog.Info("Machine has been successfully deleted")
		}
	default:
		klog.Info("unknown machine phase", "status.phase", machine.Status.Phase)
	}

	return nil, err
}

func (r reconciler) getTargetClusterConfig(ctx context.Context,
	machine *v1alpha1.Machine) (*rest.Config, error) {
	// there is no cluster for the machine for now.
	if machine.Spec.ClusterName == "" {
		// 1. Note !@! Note !@!  Scheduler logic here.
		// TODO It's Too Simple NOW. FIX ME!!!!
		var maxResource int64 = 0
		var targetName string
		for clusterName, lister := range r.clusterSummaryListers {
			clusterSummary, err := lister.Get(singletonName)
			utilruntime.Must(err)
			if cpu, inf := clusterSummary.Capacity.Cpu().AsInt64(); inf {
				if maxResource < cpu {
					targetName = clusterName
				}
			}
		}
		machine.Spec.ClusterName = targetName
		if _, err := r.platformClientset.PlatformV1alpha1().Machines().Update(ctx, machine, metav1.UpdateOptions{}); err != nil {
			return nil, err
		} else {
			return nil, errors.NewBadRequest("now has a new cluster")
		}
	}

	target, err := r.multiclusterclientset.MulticlusterV1alpha1().Targets(corev1.NamespaceDefault).Get(ctx, machine.Spec.ClusterName, metav1.GetOptions{})
	if err != nil {
		klog.Infof("can't get '%s' target in global", machine.Spec.ClusterName)
		return nil, err
	}
	if kcfg := target.Spec.KubeconfigSecret; kcfg != nil {
		return agent.GetConfigFromKubeconfigSecretOrDie(ctx, r.kubeclientset, target.Namespace, kcfg.Name, kcfg.Key, kcfg.Context)
	} else {
		return config.GetConfigOrDie(), nil
	}
}

func (r reconciler) onCreate(ctx context.Context, machine *v1alpha1.Machine, targetconfig *rest.Config) error {
	provider, err := machineprovider.GetProvider(machine.Spec.Type)
	if err != nil {
		return err
	}
	clusterWrapper, err := innertypesv1.GetClusterByName(ctx, machine.Spec.ClusterName, targetconfig, r.kubeclientset)
	if err != nil {
		return err
	}

	for machine.Status.Phase == v1alpha1.MachineInitializing {
		err = provider.OnCreate(ctx, machine, clusterWrapper)
		if err != nil {
			// Update status
			_, err = r.platformClientset.PlatformV1alpha1().Machines().UpdateStatus(ctx, machine, metav1.UpdateOptions{})
			return err
		}
	}
	if _, err := r.platformClientset.PlatformV1alpha1().Machines().UpdateStatus(ctx, machine, metav1.UpdateOptions{}); err != nil {
		return err
	} else {
		return nil
	}
}
