package spec

import (
	"github.com/thoas/go-funk"
	"pml.io/april/pkg/app/version"
)

var (
	TKEVersion    = version.Get().GitVersion
	Archs         = []string{"amd64", "arm64"}
	Arm64         = "arm64"
	Arm64Variants = []string{"v8", "unknown"}
	OSs           = []string{"linux"}

	K8sVersionConstraint = ">= 1.10"
	K8sVersions          = []string{"1.19.7", "1.18.3", "1.20.4", "1.20.4-tke.1"}
	K8sVersionsWithV     = funk.Map(K8sVersions, func(s string) string {
		return "v" + s
	}).([]string)

	DockerVersions                 = []string{"20.10.7"}
	CNIPluginsVersions             = []string{"v0.8.6"}
	ConntrackToolsVersions         = []string{"1.4.4"}
	NvidiaDriverVersions           = []string{"440.31"}
	NvidiaContainerRuntimeVersions = []string{"3.1.4"}
)
