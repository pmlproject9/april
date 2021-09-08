/*
Copyright 2021 The April Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	agentconfig "admiralty.io/multicluster-scheduler/pkg/config/agent"
	"admiralty.io/multicluster-scheduler/pkg/generated/clientset/versioned"
	"admiralty.io/multicluster-scheduler/pkg/generated/informers/externalversions/multicluster/v1alpha1"
	"context"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
	"pml.io/april/pkg/controllers/cluster"
	machinecontroller "pml.io/april/pkg/controllers/machine"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"time"

	admiraltytclientset "admiralty.io/multicluster-scheduler/pkg/generated/clientset/versioned"
	admiraltyinformers "admiralty.io/multicluster-scheduler/pkg/generated/informers/externalversions"
	clientset "pml.io/april/pkg/generated/clientset/versioned"
	informers "pml.io/april/pkg/generated/informers/externalversions"
	"pml.io/april/pkg/signals"
)

func main() {

	klog.InitFlags(nil)
	//var (
	//	masterURL  string
	//	kubeconfig string
	//)
	//flag.StringVar(&kubeconfig, "kubeconfig", "/Users/xialingming/.kube/config", "Path to a kubeconfig. Only required if out-of-cluster.")
	//flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	//
	//flag.Parse()
	// 1. set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-stopCh
		cancel()
	}()

	// 2. make self and remote configurations
	cfg := config.GetConfigOrDie()

	agentCfg := agentconfig.NewFromCRD(ctx)

	//3. start controllers
	startControllers(ctx, stopCh, agentCfg, cfg)

	// down.
	<-stopCh
}

// TODO !!!!  So of course we need a NEW controller manager, to handle multi-cluster controllers.
func startControllers(ctx context.Context, stopCh <-chan struct{}, agentCfg agentconfig.Config, cfg *rest.Config) {
	//1. construct local clients: local k8s client and local platform client
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	platformClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building example clientset: %s", err.Error())
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Minute)
	// 我们是不是选择一个更高的重复处理时间。
	platformInformerFactory := informers.NewSharedInformerFactory(platformClient, time.Hour)

	// 2. construct remote config and client set
	n := len(agentCfg.Targets)
	targetKubeClients := make(map[string]kubernetes.Interface, n)
	targetCustomClients := make(map[string]admiraltytclientset.Interface, n)
	targetCustomInformerFactories := make(map[string]admiraltyinformers.SharedInformerFactory, n)
	targetClusterSummaryInformers := make(map[string]v1alpha1.ClusterSummaryInformer, n)
	// all targets are remote clusters.
	for _, target := range agentCfg.Targets {
		k, err := kubernetes.NewForConfig(target.ClientConfig)
		utilruntime.Must(err)
		targetKubeClients[target.GetKey()] = k
		c, err := versioned.NewForConfig(target.ClientConfig)
		utilruntime.Must(err)
		targetCustomClients[target.GetKey()] = c
		f := admiraltyinformers.NewSharedInformerFactoryWithOptions(c, time.Minute, admiraltyinformers.WithNamespace(target.Namespace))
		targetCustomInformerFactories[target.Name] = f
		targetClusterSummaryInformers[target.Name] = f.Multicluster().V1alpha1().ClusterSummaries()
	}

	machineController := machinecontroller.NewController(kubeClient, cfg, targetClusterSummaryInformers,
		platformInformerFactory.Platform().V1alpha1().Machines())

	clusterController := cluster.NewController(kubeClient, cfg, platformInformerFactory.Platform().V1alpha1().Clusters())

	// notice that there is no need to run Start methods in a separate goroutine. (i.e. go kubeInformerFactory.Start(stopCh)
	// Start method is non-blocking and runs all registered informers in a dedicated goroutine.
	kubeInformerFactory.Start(stopCh)
	platformInformerFactory.Start(stopCh)

	for _, f := range targetCustomInformerFactories {
		f.Start(stopCh)
	}

	go func() { utilruntime.Must(machineController.Run(2, stopCh)) }()
	go func() { utilruntime.Must(clusterController.Run(2, stopCh)) }()

}
