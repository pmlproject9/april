package kubelet

import (
	"bytes"
	"fmt"
	"path"
	"pml.io/april/pkg/platform/provider/baremetal/constants"
	"pml.io/april/pkg/platform/provider/baremetal/res"
	"pml.io/april/pkg/util/ssh"
	"pml.io/april/pkg/util/supervisor"
	"pml.io/april/pkg/util/template"
)

type ServiceOperation string

var (
	Start ServiceOperation = "start"
	Stop  ServiceOperation = "stop"
)

func Install(s ssh.Interface, version string) (err error) {
	dstFile, err := res.KubernetesNode.CopyToNode(s, version)
	if err != nil {
		return err
	}

	for _, file := range []string{"kubelet", "kubectl"} {
		file = path.Join(constants.DstBinDir, file)
		if ok, err := s.Exist(file); err == nil && ok {
			backupFile, err := ssh.BackupFile(s, file)
			if err != nil {
				return fmt.Errorf("backup file %q error: %w", file, err)
			}
			defer func() {
				if err == nil {
					return
				}
				if err = ssh.RestoreFile(s, backupFile); err != nil {
					err = fmt.Errorf("restore file %q error: %w", backupFile, err)
				}
			}()
		}
	}

	cmd := "tar xvaf %s -C %s --strip-components=3"
	_, stderr, exit, err := s.Execf(cmd, dstFile, constants.DstBinDir)
	if err != nil {
		return fmt.Errorf("exec %q failed:exit %d:stderr %s:error %s", cmd, exit, stderr, err)
	}

	serviceData, err := template.ParseFile(path.Join(constants.ConfDir, "kubelet/kubelet.service"), nil)
	if err != nil {
		return err
	}

	ss := &supervisor.SystemdSupervisor{Name: "kubelet", SSH: s}
	err = ss.Deploy(bytes.NewReader(serviceData))
	if err != nil {
		return err
	}

	err = ss.Start()
	if err != nil {
		return err
	}

	cmd = "kubectl completion bash > /etc/bash_completion.d/kubectl"
	_, err = s.CombinedOutput(cmd)
	if err != nil {
		return err
	}

	return nil
}

func ServiceOperate(s ssh.Interface, op ServiceOperation) (err error) {
	cmd := fmt.Sprintf("systemctl %s kubelet", string(op))
	_, err = s.CombinedOutput(cmd)
	if err != nil {
		return err
	}
	return nil
}
