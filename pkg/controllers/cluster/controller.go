package cluster

import (
	"admiralty.io/multicluster-scheduler/pkg/controller"
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
	"pml.io/april/pkg/apis/platform/v1alpha1"
	platformClientset "pml.io/april/pkg/generated/clientset/versioned"
	platforminformers "pml.io/april/pkg/generated/informers/externalversions/platform/v1alpha1"
	platformlisters "pml.io/april/pkg/generated/listers/platform/v1alpha1"
	clusterprovider "pml.io/april/pkg/platform/provider/cluster"
	"pml.io/april/pkg/platform/provider/imported/constants"
	typesv1 "pml.io/april/pkg/platform/provider/type"
	"pml.io/april/pkg/util/log"
	"time"
)

type reconciler struct {
	kubeclientset     *kubernetes.Clientset
	platformClientset platformClientset.Interface
	clusterLister     platformlisters.ClusterLister
}

// NewController returns a new Machine controller
func NewController(kubeclientset *kubernetes.Clientset, config *rest.Config,
	clusterInformer platforminformers.ClusterInformer) *controller.Controller {
	platformClientset, err := platformClientset.NewForConfig(config)
	utilruntime.Must(err)
	// 1. construct TargetCluster Reconciler
	r := &reconciler{
		kubeclientset:     kubeclientset,
		platformClientset: platformClientset,
		clusterLister:     clusterInformer.Lister(),
	}

	//2. construct informer sync
	informersSynced := make([]cache.InformerSynced, 1)
	informersSynced[0] = clusterInformer.Informer().HasSynced

	//3. construct machine controller
	c := controller.New("machine-reconcile", r, informersSynced...)
	clusterInformer.Informer().AddEventHandler(controller.HandleAllWith(func(obj interface{}) {
		cluster := obj.(*v1alpha1.Cluster)
		c.EnqueueKey(cluster.Name)
	}))

	return c
}

func (r reconciler) Handle(key interface{}) (requeueAfter *time.Duration, err error) {
	ctx := context.Background()
	//1. Get Machine Object
	clusterName := key.(string)
	targetCluster, err := r.clusterLister.Get(clusterName)
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
	if targetCluster.Status.Phase == "" {
		targetCluster.Status.Phase = v1alpha1.ClusterInitializing
	}

	switch targetCluster.Status.Phase {
	case v1alpha1.ClusterInitializing:
		err = r.onCreate(ctx, targetCluster)
	case v1alpha1.ClusterRunning, v1alpha1.ClusterFailed:
		err = r.onUpdate(ctx, targetCluster)
	case v1alpha1.ClusterUpgrading:
		err = r.onUpdate(ctx, targetCluster)
	case v1alpha1.ClusterTerminating:
		log.FromContext(ctx).Info("TargetCluster has been terminated. Attempting to cleanup resources")
	default:
		log.FromContext(ctx).Info("unknown targetCluster phase", "status.phase", targetCluster.Status.Phase)
	}

	return nil, err
}

func (r reconciler) onCreate(ctx context.Context, targetCluster *v1alpha1.Cluster) error {
	targetCfg, err := r.getConfigFromKubeconfigConfigmapOrDie(ctx, targetCluster)
	if err != nil {
		return fmt.Errorf("ensureClusterCredentialExsit error: %w", err)
	}
	provider, err := clusterprovider.GetProvider(targetCluster.Spec.Type)
	if err != nil {
		return err
	}
	clusterWrapper, err := typesv1.GetCluster(targetCfg, targetCluster)
	if err != nil {
		return err
	}

	for targetCluster.Status.Phase == v1alpha1.ClusterInitializing {
		err = provider.OnCreate(ctx, clusterWrapper)
		if err != nil {
			// Update status, ignore failure
			_, _ = r.platformClientset.PlatformV1alpha1().Clusters().UpdateStatus(ctx, clusterWrapper.TargetCluster, metav1.UpdateOptions{})
			return err
		}
		clusterWrapper.TargetCluster, err = r.platformClientset.PlatformV1alpha1().Clusters().UpdateStatus(ctx, clusterWrapper.TargetCluster, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func (r reconciler) onUpdate(ctx context.Context, cluster *v1alpha1.Cluster) error {
	return nil
}

// getConfigFromKubeconfigConfigmapOrDie creates ClusterCredential for cluster if ClusterCredentialRef is nil.
// TODO: add gc collector for clean non reference ClusterCredential.
func (r reconciler) getConfigFromKubeconfigConfigmapOrDie(ctx context.Context, cluster *v1alpha1.Cluster) (*rest.Config, error) {
	key := cluster.Spec.KubeconfigSecret.Key
	if key == "" {
		key = "kube.config"
	}

	credentialConfig, err := r.kubeclientset.CoreV1().ConfigMaps(constants.ClusterConfigNamespace).Get(ctx, cluster.Spec.KubeconfigSecret.Name, metav1.GetOptions{})
	cfg0, err := clientcmd.Load([]byte(credentialConfig.Data[key]))
	if err != nil {
		return nil, err
	}

	cfg1 := clientcmd.NewDefaultClientConfig(*cfg0, &clientcmd.ConfigOverrides{CurrentContext: cluster.Spec.KubeconfigSecret.Context})

	cfg2, err := cfg1.ClientConfig()
	if err != nil {
		return nil, err
	}

	return cfg2, nil
}
