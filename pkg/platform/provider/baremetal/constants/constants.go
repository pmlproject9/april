package constants

import (
	"time"
)

const (
	AuditPolicyConfigName  = "audit-policy.yaml"
	AuthzWebhookConfigName = "tke-authz-webhook.yaml"
	OIDCCACertName         = "oidc-ca.crt"
	AdminCertName          = "admin.crt"
	AdminKeyName           = "admin.key"
	WebhookCertName        = "webhook.crt"
	WebhookKeyName         = "webhook.key"
	// Kubernetes Config
	KubernetesDir                       = "/etc/kubernetes/"
	KubernetesSchedulerPolicyConfigFile = KubernetesDir + "scheduler-policy-config.json"
	KubernetesAuditWebhookConfigFile    = KubernetesDir + "audit-api-client-config.yaml"
	TokenFile                           = KubernetesDir + "known_tokens.csv"
	KubernetesAuditPolicyConfigFile     = KubernetesDir + AuditPolicyConfigName
	KubernetesAuthzWebhookConfigFile    = KubernetesDir + AuthzWebhookConfigName
	KubeadmConfigFileName               = KubernetesDir + "kubeadm-config.yaml"
	KubeletKubeConfigFileName           = KubernetesDir + "kubelet.conf"

	KubeletPodManifestDir                = KubernetesDir + "manifests/"
	EtcdPodManifestFile                  = KubeletPodManifestDir + "etcd.yaml"
	KubeAPIServerPodManifestFile         = KubeletPodManifestDir + "kube-apiserver.yaml"
	KubeControllerManagerPodManifestFile = KubeletPodManifestDir + "kube-controller-manager.yaml"
	KubeSchedulerPodManifestFile         = KubeletPodManifestDir + "kube-scheduler.yaml"
	KeepavlivedManifestFile              = KubeletPodManifestDir + "keepalived.yaml"

	KubeadmPathInNodePackge = "kubernetes/node/bin/kubeadm"
	KubeletPathInNodePackge = "kubernetes/node/bin/kubelet"
	KubectlPathInNodePackge = "kubernetes/node/bin/kubectl"

	DstTmpDir  = "/tmp/k8s/"
	DstBinDir  = "/usr/bin/"
	CNIBinDir  = "/opt/cni/bin/"
	CNIDataDir = "/var/lib/cni/"
	CNIConfDIr = "/etc/cni"
	AppCertDir = "/app/certs/"

	// AppCert
	AppAdminCertFile = AppCertDir + AdminCertName
	AppAdminKeyFile  = AppCertDir + AdminKeyName

	// ETC
	EtcdDataDir          = "/var/lib/etcd"
	KubectlConfigFile    = "/root/.kube/config"
	KeepavliedConfigFile = "/etc/keepalived/keepalived.conf"

	// PKI
	CertificatesDir = KubernetesDir + "pki/"
	OIDCCACertFile  = CertificatesDir + OIDCCACertName
	WebhookCertFile = CertificatesDir + WebhookCertName
	WebhookKeyFile  = CertificatesDir + WebhookKeyName
	AdminCertFile   = CertificatesDir + AdminCertName
	AdminKeyFile    = CertificatesDir + AdminKeyName

	// CACertName defines certificate name
	CACertName = CertificatesDir + "ca.crt"
	// CAKeyName defines certificate name
	CAKeyName = CertificatesDir + "ca.key"
	// APIServerCertName defines API's server certificate name
	APIServerCertName = CertificatesDir + "apiserver.crt"
	// APIServerKeyName defines API's server key name
	APIServerKeyName = CertificatesDir + "apiserver.key"
	// KubeletClientCurrent defines kubelet rotate certificates
	KubeletClientCurrent = "/var/lib/kubelet/pki/kubelet-client-current.pem"
	// EtcdCACertName defines etcd's CA certificate name
	EtcdCACertName = CertificatesDir + "etcd/ca.crt"
	// EtcdCAKeyName defines etcd's CA key name
	EtcdCAKeyName = CertificatesDir + "etcd/ca.key"
	// EtcdListenClientPort defines the port etcd listen on for client traffic
	EtcdListenClientPort = 2379
	// EtcdListenPeerPort defines the port etcd listen on for peer traffic
	EtcdListenPeerPort = 2380
	// APIServerEtcdClientCertName defines apiserver's etcd client certificate name
	APIServerEtcdClientCertName = CertificatesDir + "apiserver-etcd-client.crt"
	// APIServerEtcdClientKeyName defines apiserver's etcd client key name
	APIServerEtcdClientKeyName = CertificatesDir + "apiserver-etcd-client.key"

	// LabelNodeRoleMaster specifies that a node is a control-plane
	// This is a duplicate definition of the constant in pkg/controller/service/service_controller.go
	LabelNodeRoleMaster = "node-role.kubernetes.io/master"

	// Provider
	ProviderDir           = "provider/baremetal/"
	SrcDir                = ProviderDir + "res/"
	ConfDir               = ProviderDir + "conf/"
	ConfigFile            = ConfDir + "config.yaml"
	AuditPolicyConfigFile = ConfDir + AuditPolicyConfigName
	OIDCConfigFile        = ConfDir + OIDCCACertName
	ManifestsDir          = ProviderDir + "manifests/"
	GPUManagerManifest    = ManifestsDir + "gpu-manager/gpu-manager.yaml"
	CSIOperatorManifest   = ManifestsDir + "csi-operator/csi-operator.yaml"
	MetricsServerManifest = ManifestsDir + "metrics-server/metrics-server.yaml"
	CiliumManifest        = ManifestsDir + "cilium/cilium.yaml"

	KUBERNETES               = 1
	DNSIPIndex               = 10
	GPUQuotaAdmissionIPIndex = 9
	GalaxyIPAMIPIndex        = 8

	// RenewCertsTimeThreshold control how long time left to renew certs
	RenewCertsTimeThreshold = 30 * 24 * time.Hour

	// MinNumCPU mininum cpu number.
	MinNumCPU = 2

	APIServerHostName = "pml.io"

	NeedUpgradeCoreDNSK8sVersion = "1.19.0"

	LabelMachineIPV4 = "pml.io/machine-ip"
)
