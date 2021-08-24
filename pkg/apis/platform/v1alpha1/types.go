package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"pml.io/april/pkg/util/ssh"
	"time"
)

// FinalizerName is the name identifying a finalizer during cluster lifecycle.
type FinalizerName string

const (
	// ClusterFinalize is an internal finalizer values to Cluster.
	ClusterFinalize FinalizerName = "cluster"

	// MachineFinalize is an internal finalizer values to Machine.
	MachineFinalize FinalizerName = "platform"
)

// ConditionStatus defines the status of Condition.
type ConditionStatus string

// These are valid condition statuses.
// "ConditionTrue" means a resource is in the condition.
// "ConditionFalse" means a resource is not in the condition.
// "ConditionUnknown" means server can't decide if a resource is in the condition
// or not.
const (
	ConditionTrue    ConditionStatus = "True"
	ConditionFalse   ConditionStatus = "False"
	ConditionUnknown ConditionStatus = "Unknown"
)

// MachineCondition contains details for the current condition of this Machine.
type MachineCondition struct {
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

// MachinePhase defines the phases of platform constructor
type MachinePhase string

const (
	// MachineInitializing is the initialize phases
	MachineInitializing MachinePhase = "Initializing"
	// MachineRunning is the normal running phases
	MachineRunning MachinePhase = "Running"
	// MachineFailed is the failed phases
	MachineFailed MachinePhase = "Failed"
	// MachineUpgrading means that the platform is in upgrading process.
	MachineUpgrading MachinePhase = "Upgrading"
	// MachineTerminating is the terminating phases
	MachineTerminating MachinePhase = "Terminating"
)

type LocationType string

const (
	PlanetNode    LocationType = "planet"
	SatelliteNode LocationType = "satellite"
	MeteorNode    LocationType = "meteor"
)

type ResourceType string

const (
	TypeComputing ResourceType = "computing"
	TypeStorage   ResourceType = "storage"
	TypeHybrid    ResourceType = "hybrid"
)

type ProviderType string

const (
	TypePersonal   ProviderType = "personal"
	TypeEnterprise ProviderType = "enterprise"
	TypeAnonymous  ProviderType = "anonymous"
)

type PayType string

const (
	TypAuto    PayType = "auto"
	TypeStatic PayType = "static"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MachineSpec defines the desired state of Machine
type MachineSpec struct {
	// Finalizers is an opaque list of values that must be empty to permanently remove object from storage.
	// +optional
	Finalizers []FinalizerName `json:"finalizers,omitempty" protobuf:"bytes,1,rep,name=finalizers,casttype=FinalizerName"`
	// +optional
	ClusterName string `json:"clusterName" protobuf:"bytes,3,opt,name=clusterName"`
	Type        string `json:"type" protobuf:"bytes,4,opt,name=type"`
	IP          string `json:"ip" protobuf:"bytes,5,opt,name=ip"`
	Port        int32  `json:"port" protobuf:"varint,6,opt,name=port"`
	Username    string `json:"username" protobuf:"bytes,7,opt,name=username"`
	// +optional
	Password []byte `json:"password,omitempty" protobuf:"bytes,8,opt,name=password"`
	// +optional
	PrivateKey []byte `json:"privateKey,omitempty" protobuf:"bytes,9,opt,name=privateKey"`
	// +optional
	PassPhrase []byte `json:"passPhrase,omitempty" protobuf:"bytes,10,opt,name=passPhrase"`
	// +optional
	Labels map[string]string `json:"labels,omitempty" protobuf:"bytes,11,opt,name=labels"`
	// If specified, the node's taints.
	// +optional
	Taints []corev1.Taint `json:"taints,omitempty" protobuf:"bytes,12,opt,name=taints"`
	// +optional
	ResourceType ResourceType `json:"resourceType,omitempty" protobuf:"bytes,12,opt,name=resourceType"`
	// +optional
	LocationType LocationType `json:"locationType,omitempty" protobuf:"bytes,12,opt,name=locationType"`
	// +optional
	Location string `json:"location" protobuf:"bytes,7,opt,name=location"`
	// +optional
	ProviderType ProviderType `json:"providerType,omitempty" protobuf:"bytes,12,opt,name=providerType"`
	// +optional
	CpuCore int `json:"cpucore" protobuf:"varint,6,opt,name=cpucore"`
	// +optional
	MemSize int `json:"memsize" protobuf:"varint,6,opt,name=memsize"`
	// +optional
	StorageSize int `json:"storageSize" protobuf:"varint,6,opt,name=storageSize"`
	// +optional
	PayType PayType `json:"payType,omitempty" protobuf:"bytes,12,opt,name=payType"`
	// +optional
	PayPrice int `json:"payPrice" protobuf:"varint,6,opt,name=payPrice"`
}

// MachineAddress contains information for the platform's address.
type MachineAddress struct {
	// Machine address type, one of Public, ExternalIP or InternalIP.
	Type MachineAddressType `json:"type" protobuf:"bytes,1,opt,name=type,casttype=MachineAddressType"`
	// The platform address.
	Address string `json:"address" protobuf:"bytes,2,opt,name=address"`
}

// MachineAddressType represents the type of platform address.
type MachineAddressType string

// These are valid address type of platform.
const (
	MachineHostName    MachineAddressType = "Hostname"
	MachineExternalIP  MachineAddressType = "ExternalIP"
	MachineInternalIP  MachineAddressType = "InternalIP"
	MachineExternalDNS MachineAddressType = "ExternalDNS"
	MachineInternalDNS MachineAddressType = "InternalDNS"
)

// MachineStatus defines the observed state of Machine
type MachineStatus struct {
	// +optional
	Locked *bool `json:"locked,omitempty" protobuf:"varint,1,opt,name=locked"`
	// +optional
	Phase MachinePhase `json:"phases,omitempty" protobuf:"bytes,2,opt,name=phases,casttype=MachinePhase"`
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []MachineCondition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,3,rep,name=conditions"`
	// A human readable message indicating details about why the platform is in this condition.
	// +optional
	Message string `json:"message,omitempty" protobuf:"bytes,4,opt,name=message"`
	// A brief CamelCase message indicating details about why the platform is in this state.
	// +optional
	Reason string `json:"reason,omitempty" protobuf:"bytes,5,opt,name=reason"`
	// List of addresses reachable to the platform.
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	Addresses []MachineAddress `json:"addresses,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,6,rep,name=addresses"`
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:path=machine,scope=Cluster
// +kubebuilder:subresource:status
// Machine is the Schema for the machines API
type Machine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MachineSpec   `json:"spec,omitempty"`
	Status MachineStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient:nonNamespaced
// MachineList contains a list of Machine
type MachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Machine `json:"items"`
}

func (in *Machine) GetCondition(conditionType string) *MachineCondition {
	for _, condition := range in.Status.Conditions {
		if condition.Type == conditionType {
			return &condition
		}
	}

	return nil
}

func (in *Machine) SetCondition(newCondition MachineCondition) {
	var conditions []MachineCondition

	exist := false

	if newCondition.LastProbeTime.IsZero() {
		newCondition.LastProbeTime = metav1.Now()
	}
	for _, condition := range in.Status.Conditions {
		if condition.Type == newCondition.Type {
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

type HookType string

const (
	// node lifecycle hook
	HookPreInstall  HookType = "PreInstall"
	HookPostInstall HookType = "PostInstall"
	HookPreUpgrade  HookType = "PreUpgrade"
	HookPostUpgrade HookType = "PostUpgrade"

	// custer lifecycle hook
	HookPreClusterInstall  HookType = "PreClusterInstall"
	HookPostClusterInstall HookType = "PostClusterInstall"
	HookPreClusterUpgrade  HookType = "PreClusterUpgrade"
	HookPostClusterUpgrade HookType = "PostClusterUpgrade"
	HookPreClusterDelete   HookType = "PreClusterDelete"
	HookPostClusterDelete  HookType = "PostClusterDelete"
)

func (in *MachineSpec) SSH() (*ssh.SSH, error) {
	sshConfig := &ssh.Config{
		User:        in.Username,
		Host:        in.IP,
		Port:        int(in.Port),
		Password:    string(in.Password),
		PrivateKey:  in.PrivateKey,
		PassPhrase:  in.PassPhrase,
		DialTimeOut: time.Second,
		Retry:       0,
	}
	return ssh.New(sshConfig)
}
