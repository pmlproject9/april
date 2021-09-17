/*
 * Tencent is pleased to support the open source community by making TKEStack
 * available.
 *
 * Copyright (C) 2012-2019 Tencent. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the “License”); you may not use
 * this file except in compliance with the License. You may obtain a copy of the
 * License at
 *
 * https://opensource.org/licenses/Apache-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an “AS IS” BASIS, WITHOUT
 * WARRANTIES OF ANY KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations under the License.
 */

package cluster

import (
	"context"
	"github.com/pkg/errors"
	appv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"pml.io/april/pkg/platform/provider/imported/constants"

	typesv1 "pml.io/april/pkg/platform/provider/type"
	apiclient "pml.io/april/pkg/util/apiclient"
)

func (p *Provider) EnsureClusterFittness(ctx context.Context, c *typesv1.Cluster) error {
	client, err := kubernetes.NewForConfig(c.TargetConfig)
	if err != nil {
		return err
	}
	//1.
	if apiclient.ClusterVersionIsBefore118(client) {
		return errors.New("Cluster version should >= 1.18")
	}
	//2.

	return nil
}

func (p *Provider) EnsureVKInstalled(ctx context.Context, c *typesv1.Cluster) error {
	VkDeployment := newDeployment(c.ClusterName, c.TargetCluster.Spec.KubeconfigSecret.Name)
	_, err := c.MasterKubeclientset.AppsV1().Deployments(constants.ClusterConfigNamespace).Create(ctx, VkDeployment, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	// TODO 讨论：我们应该等待deployment 部署成功么？
	// 可能会牺牲 并发性。
	//if err := waitForDeploymentCompleteAndMarkPodsReady(); err != nil {
	//	return err
	//}

	return nil
}

// newDeployment returns a Deployment with a tensile-kube/virtual-kubelet image
func newDeployment(deploymentName, cmName string) *appv1.Deployment {
	replicas := int32(1)
	return &appv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: constants.ClusterConfigNamespace,
			Name:      deploymentName,
			Labels:    map[string]string{"k8s-app": "kubelet"},
		},
		Spec: appv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"k8s-app": "virtual-kubelet"}},
			Strategy: appv1.DeploymentStrategy{
				Type:          appv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: new(appv1.RollingUpdateDeployment),
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"pod-type": "virtual-kubelet", "k8s-app": "virtual-kubelet"},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:            "virtual-kubelet",
							Image:           "lmxia/virtual-node:v0.1.1-21-ged34a840a4558a",
							ImagePullPolicy: v1.PullIfNotPresent,
							Env: []v1.EnvVar{
								{Name: "KUBELET_PORT", Value: "10450"},
								{Name: "DEFAULT_NODE_NAME", Value: deploymentName},
								{Name: "VKUBELET_POD_IP",
									ValueFrom: &v1.EnvVarSource{
										FieldRef: &v1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
							},
							Args: []string{
								"--provider=k8s",
								"--nodename=$(DEFAULT_NODE_NAME)",
								"--disable-taint=true",
								"--kube-api-qps=500",
								"--kube-api-burst=1000",
								"--client-qps=500",
								"--client-burst=1000",
								"--client-kubeconfig=/root/kube.config",
								"--klog.v=5",
								"--log-level=debug",
								"--metrics-addr=:10455",
							},
							LivenessProbe: &v1.Probe{
								Handler: v1.Handler{
									TCPSocket: &v1.TCPSocketAction{
										Port: intstr.FromInt(10455),
									},
								},
								InitialDelaySeconds: 20,
								PeriodSeconds:       20,
							},
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "kube",
									MountPath: "/root",
									ReadOnly:  true,
								},
							},
						},
					},
					HostNetwork: true,
					Tolerations: []v1.Toleration{{Key: "role", Value: "not-vk", Operator: "Equal", Effect: "NoSchedule"}},
					Volumes: []v1.Volume{
						{
							VolumeSource: v1.VolumeSource{
								ConfigMap: &v1.ConfigMapVolumeSource{
									LocalObjectReference: v1.LocalObjectReference{
										Name: cmName,
									},
									Items: []v1.KeyToPath{{Key: "kube.config", Path: "kube.config"}},
								},
							},
							Name: "kube",
						},
					},
					ServiceAccountName: "virtual-kubelet",
					Affinity: &v1.Affinity{
						NodeAffinity: &v1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
								NodeSelectorTerms: []v1.NodeSelectorTerm{
									{
										MatchExpressions: []v1.NodeSelectorRequirement{
											{
												Key:      "type",
												Operator: v1.NodeSelectorOpNotIn,
												Values:   []string{"virtual-kubelet"},
											},
										},
									},
								},
							},
						},
						PodAntiAffinity: &v1.PodAntiAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: []v1.PodAffinityTerm{
								{
									LabelSelector: &metav1.LabelSelector{
										MatchExpressions: []metav1.LabelSelectorRequirement{
											{
												Key:      "pod-type",
												Operator: metav1.LabelSelectorOpIn,
												Values:   []string{"virtual-kubelet"},
											},
										},
									},
									TopologyKey: "kubernetes.io/hostname",
								},
							},
						},
					},
				},
			},
		},
	}
}
