/*
 * Tencent is pleased to support the open source community by making TKEStack
 * available.
 *
 * Copyright (C) 2012-2019 Tencent. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the “License”); you may not use
 * this file except in compliance with the License. You may obtain a copy of the
 * License at
 *
 * https://opensource.org/licenses/Apache-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an “AS IS” BASIS, WITHOUT
 * WARRANTIES OF ANY KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations under the License.
 */

package cluster

import (
	clusterprovider "pml.io/april/pkg/platform/provider/cluster"
	"pml.io/april/pkg/util/log"
)

func init() {
	p, err := NewProvider()
	if err != nil {
		log.Errorf("init cluster provider error: %s", err)
		return
	}
	clusterprovider.Register(p.Name(), p)
}

type Provider struct {
	*clusterprovider.DelegateProvider
}

var _ clusterprovider.Provider = &Provider{}

func NewProvider() (*Provider, error) {
	p := new(Provider)

	p.DelegateProvider = &clusterprovider.DelegateProvider{
		ProviderName: "Imported",
		CreateHandlers: []clusterprovider.Handler{
			p.EnsureClusterFittness,
			p.EnsureVKInstalled,
		},
		DeleteHandlers: []clusterprovider.Handler{
			p.EnsureCleanClusterMark,
		},
	}
	return p, nil
}
