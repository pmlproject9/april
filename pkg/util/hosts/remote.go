/*
 * Tencent is pleased to support the open source community by making TKEStack
 * available.
 *
 * Copyright (C) 2012-2019 Tencent. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not use
 * this file except in compliance with the License. You may obtain a copy of the
 * License at
 *
 * https://opensource.org/licenses/Apache-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
 * WARRANTIES OF ANY KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations under the License.
 */

package hosts

import (
	"bytes"
	"pml.io/april/pkg/util/ssh"
)

// RemoteHosts for remote hosts
type RemoteHosts struct {
	Host string
	SSH  ssh.Interface
}

var _ Hostser = new(RemoteHosts)

// Data return hosts data
func (h RemoteHosts) Data() ([]byte, error) {
	return h.SSH.ReadFile(linuxHostfile)
}

// Set sets hosts
func (h RemoteHosts) Set(ip string) error {
	data, err := h.Data()
	if err != nil {
		return err
	}
	data, err = setHosts(data, h.Host, ip)
	if err != nil {
		return err
	}

	return h.SSH.WriteFile(bytes.NewReader(data), linuxHostfile)
}
