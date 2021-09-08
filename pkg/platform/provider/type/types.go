package types

import (
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
	"net/url"
	platform "pml.io/april/pkg/apis/platform/v1alpha1"
)
import "context"

type Cluster struct {
	K8sVersionsWithV  string
	MasterIp          string
	ClusterName       string
	TargetCluster     *platform.Cluster
	TargetConfig      *rest.Config
	ClusterCredential *ClusterCredential
}

// ClusterCredential records the credential information needed to access the cluster.
type ClusterCredential struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	TenantID    string `json:"tenantID" protobuf:"bytes,2,opt,name=tenantID"`
	ClusterName string `json:"clusterName" protobuf:"bytes,3,opt,name=clusterName"`

	// +optional
	ETCDCACert []byte `json:"etcdCACert,omitempty" protobuf:"bytes,4,opt,name=etcdCACert"`
	// +optional
	ETCDCAKey []byte `json:"etcdCAKey,omitempty" protobuf:"bytes,5,opt,name=etcdCAKey"`
	// +optional
	ETCDAPIClientCert []byte `json:"etcdAPIClientCert,omitempty" protobuf:"bytes,6,opt,name=etcdAPIClientCert"`
	// +optional
	ETCDAPIClientKey []byte `json:"etcdAPIClientKey,omitempty" protobuf:"bytes,7,opt,name=etcdAPIClientKey"`

	// For connect the cluster
	// +optional
	CACert []byte `json:"caCert,omitempty" protobuf:"bytes,8,opt,name=caCert"`
	// +optional
	CAKey []byte `json:"caKey,omitempty" protobuf:"bytes,9,opt,name=caKey"`
	// For kube-apiserver X509 auth
	// +optional
	ClientCert []byte `json:"clientCert,omitempty" protobuf:"bytes,10,opt,name=clientCert"`
	// For kube-apiserver X509 auth
	// +optional
	ClientKey []byte `json:"clientKey,omitempty" protobuf:"bytes,11,opt,name=clientKey"`
	// For kube-apiserver token auth
	// +optional
	Token *string `json:"token,omitempty" protobuf:"bytes,12,opt,name=token"`
	// For kubeadm init or join
	// +optional
	BootstrapToken *string `json:"bootstrapToken,omitempty" protobuf:"bytes,13,opt,name=bootstrapToken"`
	// For kubeadm init or join
	// +optional
	CertificateKey *string `json:"certificateKey,omitempty" protobuf:"bytes,14,opt,name=certificateKey"`
}

func GetClusterByName(ctx context.Context, clusterName string, targetConfig *rest.Config) (*Cluster, error) {
	kubeClient, err := kubernetes.NewForConfig(targetConfig)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}
	versionInfo, err := kubeClient.ServerVersion()
	if err != nil {
		return nil, fmt.Errorf("couldn't retrieve API server's version: %v", err)
	}

	credential := &ClusterCredential{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "clustercredential",
		},
		CACert: targetConfig.CAData,
		Token:  &targetConfig.BearerToken,
	}

	u, err := url.Parse(targetConfig.Host)
	if err != nil {
		return nil, err
	}

	return &Cluster{
		K8sVersionsWithV:  versionInfo.String(),
		MasterIp:          u.Hostname(),
		TargetConfig:      targetConfig,
		ClusterName:       clusterName,
		ClusterCredential: credential,
	}, nil
}

func GetCluster(cfg *rest.Config, cluster *platform.Cluster) (*Cluster, error) {
	result := new(Cluster)
	result.ClusterName = cluster.Name
	result.TargetConfig = cfg
	result.TargetCluster = cluster
	return result, nil
}
