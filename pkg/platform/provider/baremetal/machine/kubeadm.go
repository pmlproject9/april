///*
// * Tencent is pleased to support the open source community by making TKEStack
// * available.
// *
// * Copyright (C) 2012-2020 Tencent. All Rights Reserved.
// *
// * Licensed under the Apache License, Version 2.0 (the “License”); you may not use
// * this file except in compliance with the License. You may obtain a copy of the
// * License at
// *
// * https://opensource.org/licenses/Apache-2.0
// *
// * Unless required by applicable law or agreed to in writing, software
// * distributed under the License is distributed on an “AS IS” BASIS, WITHOUT
// * WARRANTIES OF ANY KIND, either express or implied.  See the License for the
// * specific language governing permissions and limitations under the License.
// */
//
package machine

import (
	"fmt"
	kubeadmv1beta2 "pml.io/april/pkg/platform/provider/baremetal/apis/kubeadm/v1beta2"
	"pml.io/april/pkg/platform/provider/baremetal/constants"
	typesv1 "pml.io/april/pkg/platform/provider/type"
)

func (p *Provider) getKubeadmJoinConfig(c *typesv1.Cluster, machineIP string) *kubeadmv1beta2.JoinConfiguration {
	apiServerEndpoint := c.MasterIp

	nodeRegistration := kubeadmv1beta2.NodeRegistrationOptions{}
	kubeletExtraArgs := p.getKubeletExtraArgs(c)

	kubeletExtraArgs["node-labels"] = fmt.Sprintf("%s=%s", constants.LabelMachineIPV4, machineIP)

	// add node ip for single stack ipv6 clusters.
	if _, ok := kubeletExtraArgs["node-ip"]; !ok {
		kubeletExtraArgs["node-ip"] = machineIP
	}
	//if _, ok := kubeletExtraArgs["hostname-override"]; !ok {
	//	nodeRegistration.Name = machineIP
	//}
	nodeRegistration.KubeletExtraArgs = kubeletExtraArgs

	return &kubeadmv1beta2.JoinConfiguration{
		NodeRegistration: nodeRegistration,
		Discovery: kubeadmv1beta2.Discovery{
			BootstrapToken: &kubeadmv1beta2.BootstrapTokenDiscovery{
				Token:                    *c.ClusterCredential.BootstrapToken,
				APIServerEndpoint:        apiServerEndpoint,
				UnsafeSkipCAVerification: true,
			},
			TLSBootstrapToken: *c.ClusterCredential.BootstrapToken,
		},
	}
}

func (p *Provider) getKubeletExtraArgs(c *typesv1.Cluster) map[string]string {
	//args := map[string]string{
	//	"pod-infra-container-image": images.Get().Pause.FullName(),
	//}
	args := map[string]string{
		"pod-infra-container-image": "k8s.gcr.io/pause:3.4.1",
	}
	//utilruntime.Must(mergo.Merge(&args, p.config.Kubelet.ExtraArgs))

	return args
}
