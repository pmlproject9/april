package machine

import (
	//multiclusterv1alpha1 "admiralty.io/multicluster-scheduler/pkg/apis/multicluster/v1alpha1"
	"bytes"
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	bootstrapapi "k8s.io/cluster-bootstrap/token/api"
	api "k8s.io/kubernetes/pkg/apis/core"

	"os"
	"path"
	platformv1 "pml.io/april/pkg/apis/platform/v1alpha1"
	"pml.io/april/pkg/platform/provider/baremetal/constants"
	"pml.io/april/pkg/platform/provider/baremetal/phases/addons/cniplugins"
	"pml.io/april/pkg/platform/provider/baremetal/phases/docker"
	"pml.io/april/pkg/platform/provider/baremetal/phases/gpu"
	"pml.io/april/pkg/platform/provider/baremetal/phases/kubeadm"
	"pml.io/april/pkg/platform/provider/baremetal/phases/kubeconfig"
	"pml.io/april/pkg/platform/provider/baremetal/phases/kubelet"
	"pml.io/april/pkg/platform/provider/baremetal/preflight"
	"pml.io/april/pkg/platform/provider/baremetal/res"
	typesv1 "pml.io/april/pkg/platform/provider/type"
	"pml.io/april/pkg/util/apiclient"
	"pml.io/april/pkg/util/cmdstring"
	"pml.io/april/pkg/util/hosts"
	"strings"
	"time"
)

const (
	sysctlFile       = "/etc/sysctl.conf"
	sysctlCustomFile = "/etc/sysctl.d/99-tke.conf"
	moduleFile       = "/etc/modules-load.d/tke.conf"
)

func (p *Provider) EnsureCopyFiles(ctx context.Context, machine *platformv1.Machine, cluster *typesv1.Cluster) error {
	_, err := machine.Spec.SSH()
	if err != nil {
		return err
	}

	return nil
}

func (p *Provider) EnsurePreInstallHook(ctx context.Context, machine *platformv1.Machine, cluster *typesv1.Cluster) error {
	return nil
}

func (p *Provider) EnsurePostInstallHook(ctx context.Context, machine *platformv1.Machine, cluster *typesv1.Cluster) error {
	return nil
}

func (p *Provider) EnsureClean(ctx context.Context, machine *platformv1.Machine, cluster *typesv1.Cluster) error {
	machineSSH, err := machine.Spec.SSH()
	if err != nil {
		return err
	}

	_, err = machineSSH.CombinedOutput(fmt.Sprintf("rm -rf %s", constants.KubernetesDir))
	if err != nil {
		return err
	}

	_, err = machineSSH.CombinedOutput(fmt.Sprintf("rm -rf %s", constants.CNIConfDIr))
	if err != nil {
		return err
	}

	return nil
}

func (p *Provider) EnsurePreflight(ctx context.Context, machine *platformv1.Machine, cluster *typesv1.Cluster) error {
	machineSSH, err := machine.Spec.SSH()
	if err != nil {
		return err
	}

	err = preflight.RunNodeChecks(cluster, machineSSH)
	if err != nil {
		return err
	}

	return nil
}

func (p *Provider) EnsureRegistryHosts(ctx context.Context, machine *platformv1.Machine, cluster *typesv1.Cluster) error {
	if !p.config.Registry.NeedSetHosts() {
		return nil
	}

	machineSSH, err := machine.Spec.SSH()
	if err != nil {
		return err
	}

	domains := []string{
		p.config.Registry.Domain,
	}

	for _, one := range domains {
		remoteHosts := hosts.RemoteHosts{Host: one, SSH: machineSSH}
		err := remoteHosts.Set(p.config.Registry.IP)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Provider) EnsureKernelModule(ctx context.Context, machine *platformv1.Machine, cluster *typesv1.Cluster) error {
	s, err := machine.Spec.SSH()
	if err != nil {
		return err
	}

	modules := []string{"iptable_nat", "ip_vs", "ip_vs_rr", "ip_vs_wrr", "ip_vs_sh"}
	if _, err := s.CombinedOutput("modinfo br_netfilter"); err == nil {
		modules = append(modules, "br_netfilter")
	}
	var data bytes.Buffer
	for _, m := range modules {
		_, err := s.CombinedOutput(fmt.Sprintf("modprobe %s", m))
		if err != nil {
			return err
		}
		data.WriteString(m + "\n")
	}
	err = s.WriteFile(strings.NewReader(data.String()), moduleFile)
	if err != nil {
		return err
	}

	return nil
}

func (p *Provider) EnsureSysctl(ctx context.Context, machine *platformv1.Machine, cluster *typesv1.Cluster) error {
	machineSSH, err := machine.Spec.SSH()
	if err != nil {
		return err
	}

	_, err = machineSSH.CombinedOutput(cmdstring.SetFileContent(sysctlFile, "^net.ipv4.ip_forward.*", "net.ipv4.ip_forward = 1"))
	if err != nil {
		return err
	}

	_, err = machineSSH.CombinedOutput(cmdstring.SetFileContent(sysctlFile, "^net.bridge.bridge-nf-call-iptables.*", "net.bridge.bridge-nf-call-iptables = 1"))
	if err != nil {
		return err
	}

	f, err := os.Open(path.Join(constants.ConfDir, "sysctl.conf"))
	if err == nil {
		err = machineSSH.WriteFile(f, sysctlCustomFile)
		if err != nil {
			return err
		}
	}

	_, err = machineSSH.CombinedOutput("sysctl --system")
	if err != nil {
		return err
	}
	return nil
}

func (p *Provider) EnsureDisableSwap(ctx context.Context, machine *platformv1.Machine, cluster *typesv1.Cluster) error {
	machineSSH, err := machine.Spec.SSH()
	if err != nil {
		return err
	}

	_, err = machineSSH.CombinedOutput(`swapoff -a && sed -i "s/^[^#]*swap/#&/" /etc/fstab`)
	if err != nil {
		return err
	}

	return nil
}

func (p *Provider) EnsureDisableOffloading(ctx context.Context, machine *platformv1.Machine, cluster *typesv1.Cluster) error {
	machineSSH, err := machine.Spec.SSH()
	if err != nil {
		return err
	}

	_, err = machineSSH.CombinedOutput(`ethtool --offload flannel.1 rx off tx off || true`)
	if err != nil {
		return err
	}

	return nil
}

func (p *Provider) EnsureManifestDir(ctx context.Context, machine *platformv1.Machine, cluster *typesv1.Cluster) error {
	machineSSH, err := machine.Spec.SSH()
	if err != nil {
		return err
	}
	_, err = machineSSH.CombinedOutput("mkdir -p /etc/kubernetes/manifests")
	if err != nil {
		return err
	}

	return nil
}

func (p *Provider) EnsureKubeconfig(ctx context.Context, machine *platformv1.Machine, cluster *typesv1.Cluster) error {
	masterEndpoint := fmt.Sprintf("https://%s:%d", cluster.MasterIp, 6443)

	machineSSH, err := machine.Spec.SSH()
	if err != nil {
		return err
	}

	option := &kubeconfig.Option{
		MasterEndpoint: masterEndpoint,
		ClusterName:    cluster.ClusterName,
		CACert:         cluster.ClusterCredential.CACert,
		Token:          *cluster.ClusterCredential.Token,
	}
	err = kubeconfig.Install(machineSSH, option)
	if err != nil {
		return err
	}

	return nil
}

//func (p *Provider) EnsureNvidiaDriver(ctx context.Context, machine *platformv1.Machine, cluster string) error {
//	if !gpu.IsEnable(machine.Spec.Labels) {
//		return nil
//	}
//
//	machineSSH, err := machine.Spec.SSH()
//	if err != nil {
//		return err
//	}
//
//	return gpu.InstallNvidiaDriver(machineSSH, &gpu.NvidiaDriverOption{})
//}

func (p *Provider) EnsureNvidiaContainerRuntime(ctx context.Context, machine *platformv1.Machine, cluster *typesv1.Cluster) error {
	if !gpu.IsEnable(machine.Spec.Labels) {
		return nil
	}

	machineSSH, err := machine.Spec.SSH()
	if err != nil {
		return err
	}

	return gpu.InstallNvidiaContainerRuntime(machineSSH, &gpu.NvidiaContainerRuntimeOption{})
}

func (p *Provider) EnsureNvidiaDevicePlugin(ctx context.Context, machine *platformv1.Machine, cluster *typesv1.Cluster) error {
	option := &gpu.NvidiaDevicePluginOption{
		Image: "nvcr.io/nvidia/k8s-device-plugin:v0.9.0",
	}

	targetClientset, err := kubernetes.NewForConfig(cluster.TargetConfig)
	if err != nil {
		return err
	}

	err = gpu.InstallNvidiaDevicePlugin(ctx, targetClientset, option)
	if err != nil {
		return err
	}

	return nil
}

func (p *Provider) EnsureDocker(ctx context.Context, machine *platformv1.Machine, cluster *typesv1.Cluster) error {
	machineSSH, err := machine.Spec.SSH()
	if err != nil {
		return err
	}

	insecureRegistries := fmt.Sprintf(`"%s"`, "p.config.Registry.Domain")

	//extraArgs := cluster.Spec.DockerExtraArgs
	//utilruntime.Must(mergo.Merge(&extraArgs, p.config.Docker.ExtraArgs))
	option := &docker.Option{
		InsecureRegistries: insecureRegistries,
		RegistryDomain:     insecureRegistries,
		IsGPU:              gpu.IsEnable(machine.Spec.Labels),
		ExtraArgs:          nil, // TODO
	}
	err = docker.Install(machineSSH, option)
	if err != nil {
		return err
	}

	return nil
}

func (p *Provider) EnsureKubelet(ctx context.Context, machine *platformv1.Machine, cluster *typesv1.Cluster) error {
	machineSSH, err := machine.Spec.SSH()
	if err != nil {
		return err
	}

	err = kubelet.Install(machineSSH, cluster.K8sVersionsWithV)

	if err != nil {
		return err
	}

	return nil
}

func (p *Provider) EnsureCNIPlugins(ctx context.Context, machine *platformv1.Machine, cluster *typesv1.Cluster) error {
	machineSSH, err := machine.Spec.SSH()
	if err != nil {
		return err
	}

	option := &cniplugins.Option{}
	err = cniplugins.Install(machineSSH, option)
	if err != nil {
		return err
	}
	return nil
}

func (p *Provider) EnsureConntrackTools(ctx context.Context, machine *platformv1.Machine, cluster *typesv1.Cluster) error {
	machineSSH, err := machine.Spec.SSH()
	if err != nil {
		return err
	}

	err = res.ConntrackTools.InstallWithDefault(machineSSH)
	if err != nil {
		return err
	}

	return nil
}

func (p *Provider) EnsureKubeadm(ctx context.Context, machine *platformv1.Machine, cluster *typesv1.Cluster) error {
	machineSSH, err := machine.Spec.SSH()
	if err != nil {
		return err
	}

	err = kubeadm.Install(machineSSH, cluster.K8sVersionsWithV)
	if err != nil {
		return err
	}

	return nil
}

func (p *Provider) EnsureJoinPhasePreflight(ctx context.Context, machine *platformv1.Machine, cluster *typesv1.Cluster) error {
	machineSSH, err := machine.Spec.SSH()
	if err != nil {
		return err
	}
	// make sure bootstrap token exist.
	if err := completeCredential(cluster); err != nil {
		return err
	}
	err = kubeadm.Join(machineSSH, p.getKubeadmJoinConfig(cluster, machine.Spec.IP), "preflight", []string{cluster.MasterIp})
	if err != nil {
		return err
	}

	return nil
}

func (p *Provider) EnsureJoinPhaseKubeletStart(ctx context.Context, machine *platformv1.Machine, cluster *typesv1.Cluster) error {
	machineSSH, err := machine.Spec.SSH()
	if err != nil {
		return err
	}
	// make sure bootstrap token exist.
	if err := completeCredential(cluster); err != nil {
		return err
	}
	err = kubeadm.Join(machineSSH, p.getKubeadmJoinConfig(cluster, machine.Spec.IP), "kubelet-start", []string{cluster.MasterIp})
	if err != nil {
		return err
	}
	return nil
}

func (p *Provider) EnsureMarkNode(ctx context.Context, machine *platformv1.Machine, cluster *typesv1.Cluster) error {
	clientset, err := kubernetes.NewForConfig(cluster.TargetConfig)
	if err != nil {
		return err
	}

	node, err := apiclient.GetNodeByMachineIP(ctx, clientset, machine.Spec.IP)
	if err != nil {
		return err
	}
	err = apiclient.MarkNode(ctx, clientset, node.Name, machine.Spec.Labels, machine.Spec.Taints)
	if err != nil {
		return err
	}
	return nil
}

func (p *Provider) EnsureNodeReady(ctx context.Context, machine *platformv1.Machine, cluster *typesv1.Cluster) error {
	clientset, err := kubernetes.NewForConfig(cluster.TargetConfig)
	if err != nil {
		return err
	}

	return wait.PollImmediate(5*time.Second, 5*time.Minute, func() (bool, error) {
		node, err := apiclient.GetNodeByMachineIP(ctx, clientset, machine.Spec.IP)
		if err != nil {
			return false, nil
		}

		for _, one := range node.Status.Conditions {
			if one.Type == corev1.NodeReady && one.Status == corev1.ConditionTrue {
				return true, nil
			}
		}

		return false, nil
	})
}

func (p *Provider) EnsureInitAPIServerHost(ctx context.Context, machine *platformv1.Machine, cluster *typesv1.Cluster) error {
	machineSSH, err := machine.Spec.SSH()
	if err != nil {
		return err
	}
	remoteHosts := hosts.RemoteHosts{Host: constants.APIServerHostName, SSH: machineSSH}
	return remoteHosts.Set(cluster.MasterIp)
}

func completeCredential(cluster *typesv1.Cluster) error {
	client, err := kubernetes.NewForConfig(cluster.TargetConfig)
	if err != nil {
		return err
	}
	// 1. first check if bootstrap token type secret exist and valid.
	secrets, err := client.CoreV1().Secrets(api.NamespaceSystem).List(context.Background(), metav1.ListOptions{LabelSelector: labels.Everything().String()})
	for _, secret := range secrets.Items {
		if secret.Type == bootstrapapi.SecretTypeBootstrapToken {
			// check secret validation
			if tempTokenStr, ok := kubeadm.ValidateSecretForSigning(&secret); ok {
				cluster.ClusterCredential.BootstrapToken = &tempTokenStr
				return nil
			}
		}
	}
	// 2. no valid bootstrap token, then create one!
	tempTokenStr, err := kubeadm.CreateShortLivedBootstrapToken(client)
	cluster.ClusterCredential.BootstrapToken = &tempTokenStr
	return nil
}
