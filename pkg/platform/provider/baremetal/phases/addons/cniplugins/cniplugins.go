package cniplugins

import (
	"fmt"
	"pml.io/april/pkg/platform/provider/baremetal/constants"
	"pml.io/april/pkg/platform/provider/baremetal/res"
	"pml.io/april/pkg/util/ssh"
)

type Option struct {
}

func Install(s ssh.Interface, option *Option) error {
	dstFile, err := res.CNIPlugins.CopyToNodeWithDefault(s)
	if err != nil {
		return err
	}

	_, stderr, exit, err := s.Execf("[ -d %s ] || mkdir -p %s", constants.CNIBinDir, constants.CNIBinDir)
	if exit != 0 || err != nil {
		return fmt.Errorf("clean %s failed:exit %d:stderr %s:error %s", constants.CNIBinDir, exit, stderr, err)
	}

	cmd := "tar xvaf %s -C %s"
	_, stderr, exit, err = s.Execf(cmd, dstFile, constants.CNIBinDir)
	if err != nil {
		return fmt.Errorf("exec %q failed:exit %d:stderr %s:error %s", cmd, exit, stderr, err)
	}

	return nil
}
