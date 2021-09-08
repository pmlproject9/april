/*
 * Copyright 2020 The Multicluster-Scheduler Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status
// Cluster is the Schema for the Clusters API
// +k8s:openapi-gen=trues
type Cluster struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ClusterSpec `json:"spec,omitempty"`
	// +optional
	Status ClusterStatus `json:"status,omitempty"`
}

type ClusterSpec struct {
	Type             string            `json:"type" protobuf:"bytes,4,opt,name=type"`
	KubeconfigSecret *KubeconfigSecret `json:"kubeconfigSecret,omitempty"`
}

type KubeconfigSecret struct {
	Name string `json:"name"`
	// +optional
	Key string `json:"key,omitempty"`
	// +optional
	Context string `json:"context,omitempty"`
}

// ClusterPhase defines the phases of platform constructor
type ClusterPhase string

const (
	// MachineInitializing is the initialize phases
	ClusterInitializing ClusterPhase = "Initializing"
	// MachineRunning is the normal running phases
	ClusterRunning ClusterPhase = "Running"
	// MachineFailed is the failed phases
	ClusterFailed ClusterPhase = "Failed"
	// MachineUpgrading means that the platform is in upgrading process.
	ClusterUpgrading ClusterPhase = "Upgrading"
	// MachineTerminating is the terminating phases
	ClusterTerminating ClusterPhase = "Terminating"
)

// ClusterCondition contains details for the current condition of this Cluster.
type ClusterCondition struct {
	// Type is the type of the condition.
	Type string `json:"type" protobuf:"bytes,1,opt,name=type"`
	// Status is the status of the condition.
	// Can be True, False, Unknown.
	Status ConditionStatus `json:"status" protobuf:"bytes,2,opt,name=status,casttype=ConditionStatus"`
	// Last time we probed the condition.
	// +optional
	LastProbeTime metav1.Time `json:"lastProbeTime,omitempty" protobuf:"bytes,3,opt,name=lastProbeTime"`
	// Last time the condition transitioned from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty" protobuf:"bytes,4,opt,name=lastTransitionTime"`
	// Unique, one-word, CamelCase reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty" protobuf:"bytes,5,opt,name=reason"`
	// Human-readable message indicating details about last transition.
	// +optional
	Message string `json:"message,omitempty" protobuf:"bytes,6,opt,name=message"`
}

type ClusterStatus struct {
	// +optional
	Phase ClusterPhase `json:"phases,omitempty" protobuf:"bytes,2,opt,name=phases,casttype=ClusterPhase"`
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []ClusterCondition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,3,rep,name=conditions"`
	// A human readable message indicating details about why the platform is in this condition.
	// +optional
	Message string `json:"message,omitempty" protobuf:"bytes,4,opt,name=message"`
	// A brief CamelCase message indicating details about why the platform is in this state.
	// +optional
	Reason string `json:"reason,omitempty" protobuf:"bytes,5,opt,name=reason"`
}

// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// ClusterList contains a list of Cluster
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cluster `json:"items"`
}

func (in *Cluster) KeepHistory(keepHistory bool, condition ClusterCondition) bool {
	if !keepHistory {
		return false
	}
	return condition.Status == ConditionTrue
}

func (in *Cluster) SetCondition(newCondition ClusterCondition, keepHistory bool) {
	var conditions []ClusterCondition

	exist := false

	if newCondition.LastProbeTime.IsZero() {
		newCondition.LastProbeTime = metav1.Now()
	}
	for _, condition := range in.Status.Conditions {
		if condition.Type == newCondition.Type && !in.KeepHistory(keepHistory, condition) {
			exist = true
			if newCondition.LastTransitionTime.IsZero() {
				newCondition.LastTransitionTime = condition.LastTransitionTime
			}
			condition = newCondition
		}
		conditions = append(conditions, condition)
	}

	if !exist {
		if newCondition.LastTransitionTime.IsZero() {
			newCondition.LastTransitionTime = metav1.Now()
		}
		conditions = append(conditions, newCondition)
	}

	in.Status.Conditions = conditions
	switch newCondition.Status {
	case ConditionFalse:
		in.Status.Reason = newCondition.Reason
		in.Status.Message = newCondition.Message
	default:
		in.Status.Reason = ""
		in.Status.Message = ""
	}
}
