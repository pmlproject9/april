package machine

import (
	"pml.io/april/pkg/platform/provider/baremetal/config"
	machineprovider "pml.io/april/pkg/platform/provider/machine"
	"pml.io/april/pkg/util/containerregistry"

	"pml.io/april/pkg/util/log"
)

const (
	name = "Baremetal"
)

func init() {
	p, err := NewProvider()
	if err != nil {
		log.Errorf("init machine provider error: %s", err)
		return
	}
	machineprovider.Register(p.Name(), p)
}

type Provider struct {
	*machineprovider.DelegateProvider

	config *config.Config
}

func NewProvider() (*Provider, error) {
	p := new(Provider)

	p.DelegateProvider = &machineprovider.DelegateProvider{
		ProviderName: name,

		CreateHandlers: []machineprovider.Handler{
			p.EnsureCopyFiles,
			p.EnsurePreInstallHook,

			p.EnsureClean,
			//p.EnsureRegistryHosts,
			p.EnsureInitAPIServerHost,

			p.EnsureKernelModule,
			p.EnsureSysctl,
			p.EnsureDisableSwap,
			p.EnsureManifestDir,

			p.EnsurePreflight, // wait basic setting done

			//p.EnsureNvidiaDriver,
			//p.EnsureNvidiaContainerRuntime,
			p.EnsureDocker,         // 这是system service
			p.EnsureKubelet,        // 这个也是system service
			p.EnsureCNIPlugins,     // 解压一大堆的 二进制
			p.EnsureConntrackTools, //
			p.EnsureKubeadm,

			// should we support control node? I don't know.

			p.EnsureJoinPhasePreflight,
			p.EnsureJoinPhaseKubeletStart,
			//
			p.EnsureKubeconfig,
			p.EnsureMarkNode,
			p.EnsureNodeReady,
			p.EnsureDisableOffloading, // will remove it when upgrade to k8s v1.18.5
			p.EnsurePostInstallHook,
		},
		UpdateHandlers: []machineprovider.Handler{},
	}
	//cfg, err := config.New(constants.ConfigFile)
	//if err != nil {
	//	return nil, err
	//}
	//p.config = cfg

	//containerregistry.Init(cfg.Registry.Domain, cfg.Registry.Namespace)
	containerregistry.Init("cfg.Registry.Domain", "cfg.Registry.Namespace")
	// Run for compatibility with installer.
	//// TODO: Installer reuse platform components
	//if cfg.PlatformAPIClientConfig != "" {
	//	restConfig, err := clientcmd.BuildConfigFromFlags("", cfg.PlatformAPIClientConfig)
	//	if err != nil {
	//		return nil, err
	//	}
	//	p.platformClient, err = platformv1client.NewForConfig(restConfig)
	//	if err != nil {
	//		return nil, err
	//	}
	//}

	return p, nil
}

var _ machineprovider.Provider = &Provider{}
