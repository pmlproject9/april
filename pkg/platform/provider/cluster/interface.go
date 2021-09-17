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

package cluster

import (
	"context"
	"errors"
	"fmt"
	types "pml.io/april/pkg/platform/provider/type"
	"reflect"
	"runtime"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/server/mux"
	v1alpha1 "pml.io/april/pkg/apis/platform/v1alpha1"
	"pml.io/april/pkg/util/log"
)

const (
	ReasonWaiting      = "Waiting"
	ReasonSkip         = "Skip"
	ReasonFailedInit   = "FailedInit"
	ReasonFailedUpdate = "FailedUpdate"
	ReasonFailedDelete = "FailedDelete"

	ConditionTypeDone = "EnsureDone"
)

type APIProvider interface {
	RegisterHandler(mux *mux.PathRecorderMux)
	Validate(cluster *types.Cluster) field.ErrorList
	PreCreate(cluster *types.Cluster) error
	AfterCreate(cluster *types.Cluster) error
}

type ControllerProvider interface {
	// Setup called by controller to give an chance for plugin do some init work.
	Setup() error
	// Teardown called by controller for plugin do some clean job.
	Teardown() error

	OnCreate(ctx context.Context, cluster *types.Cluster) error
	OnUpdate(ctx context.Context, cluster *types.Cluster) error
	OnDelete(ctx context.Context, cluster *types.Cluster) error
	// OnFilter called by cluster controller informer for plugin
	// do the filter on the cluster obj for specific case:
	// return bool:
	//  false: drop the object to the queue
	//  true: add the object to queue, AddFunc and UpdateFunc will
	//  go through later
	OnFilter(ctx context.Context, cluster *types.Cluster) bool
	// OnRunning call on first running.
	OnRunning(ctx context.Context, cluster *types.Cluster) error
}

// Provider defines a set of response interfaces for specific cluster
// types in cluster management.
type Provider interface {
	Name() string

	APIProvider
	ControllerProvider
}

var _ Provider = &DelegateProvider{}

type Handler func(context.Context, *types.Cluster) error

func (h Handler) Name() string {
	name := runtime.FuncForPC(reflect.ValueOf(h).Pointer()).Name()
	i := strings.LastIndex(name, ".")
	if i == -1 {
		return "Unknown"
	}
	return strings.TrimSuffix(name[i+1:], "-fm")
}

type DelegateProvider struct {
	ProviderName string

	ValidateFunc    func(cluster *types.Cluster) field.ErrorList
	PreCreateFunc   func(cluster *types.Cluster) error
	AfterCreateFunc func(cluster *types.Cluster) error

	CreateHandlers    []Handler
	DeleteHandlers    []Handler
	UpdateHandlers    []Handler
	UpgradeHandlers   []Handler
	ScaleUpHandlers   []Handler
	ScaleDownHandlers []Handler
}

func (p *DelegateProvider) Name() string {
	if p.ProviderName == "" {
		return "unknown"
	}
	return p.ProviderName
}

func (p *DelegateProvider) Setup() error {
	return nil
}

func (p *DelegateProvider) Teardown() error {
	return nil
}

func (p *DelegateProvider) RegisterHandler(mux *mux.PathRecorderMux) {
}

func (p *DelegateProvider) Validate(cluster *types.Cluster) field.ErrorList {
	if p.ValidateFunc != nil {
		return p.ValidateFunc(cluster)
	}

	return nil
}

func (p *DelegateProvider) PreCreate(cluster *types.Cluster) error {
	if p.PreCreateFunc != nil {
		return p.PreCreateFunc(cluster)
	}

	return nil
}

func (p *DelegateProvider) AfterCreate(cluster *types.Cluster) error {
	if p.AfterCreateFunc != nil {
		return p.AfterCreateFunc(cluster)
	}

	return nil
}

func (p *DelegateProvider) OnCreate(ctx context.Context, cluster *types.Cluster) error {
	condition, err := p.getCurrentCondition(cluster.TargetCluster, v1alpha1.ClusterInitializing, p.CreateHandlers)
	if err != nil {
		return err
	}

	handler := p.getHandler(condition.Type, p.CreateHandlers)
	if handler == nil {
		return fmt.Errorf("can't get handler by %s", condition.Type)
	}
	ctx = log.FromContext(ctx).WithName("ClusterProvider.OnCreate").WithName(handler.Name()).WithContext(ctx)
	log.FromContext(ctx).Info("Doing")
	startTime := time.Now()
	err = handler(ctx, cluster)
	log.FromContext(ctx).Info("Done", "error", err, "cost", time.Since(startTime).String())
	if err != nil {
		cluster.TargetCluster.SetCondition(v1alpha1.ClusterCondition{
			Type:    condition.Type,
			Status:  v1alpha1.ConditionFalse,
			Message: err.Error(),
			Reason:  ReasonFailedInit,
		}, false)
		return nil
	}

	cluster.TargetCluster.SetCondition(v1alpha1.ClusterCondition{
		Type:   condition.Type,
		Status: v1alpha1.ConditionTrue,
	}, false)

	nextConditionType := p.getNextConditionType(condition.Type, p.CreateHandlers)
	if nextConditionType == ConditionTypeDone {
		cluster.TargetCluster.Status.Phase = v1alpha1.ClusterRunning
		if err := p.OnRunning(ctx, cluster); err != nil {
			return fmt.Errorf("%s.OnRunning error: %w", p.Name(), err)
		}
	} else {
		cluster.TargetCluster.SetCondition(v1alpha1.ClusterCondition{
			Type:    nextConditionType,
			Status:  v1alpha1.ConditionUnknown,
			Message: "waiting execute",
			Reason:  ReasonWaiting,
		}, false)
	}

	return nil
}

func (p *DelegateProvider) OnUpdate(ctx context.Context, cluster *types.Cluster) error {
	return nil
}

func (p *DelegateProvider) OnDelete(ctx context.Context, cluster *types.Cluster) error {
	return nil
}

func (p *DelegateProvider) OnRunning(ctx context.Context, cluster *types.Cluster) error {
	return nil
}

func (p *DelegateProvider) OnFilter(ctx context.Context, cluster *types.Cluster) (pass bool) {
	return true
}

func (p *DelegateProvider) getNextConditionType(conditionType string, handlers []Handler) string {
	var (
		i       int
		handler Handler
	)
	for i, handler = range handlers {
		if handler.Name() == conditionType {
			break
		}
	}
	if i == len(handlers)-1 {
		return ConditionTypeDone
	}
	next := handlers[i+1]

	return next.Name()
}

func (p *DelegateProvider) getHandler(conditionType string, handlers []Handler) Handler {
	for _, handler := range handlers {
		if conditionType == handler.Name() {
			return handler
		}
	}

	return nil
}

func (p *DelegateProvider) getCurrentCondition(c *v1alpha1.Cluster, phase v1alpha1.ClusterPhase, handlers []Handler) (*v1alpha1.ClusterCondition, error) {
	if c.Status.Phase != phase {
		return nil, fmt.Errorf("cluster phase is %s now", phase)
	}
	if len(handlers) == 0 {
		return nil, fmt.Errorf("no handlers")
	}

	if len(c.Status.Conditions) == 0 {
		return &v1alpha1.ClusterCondition{
			Type:    handlers[0].Name(),
			Status:  v1alpha1.ConditionUnknown,
			Message: "waiting process",
			Reason:  ReasonWaiting,
		}, nil
	}
	for _, condition := range c.Status.Conditions {
		if condition.Status == v1alpha1.ConditionFalse || condition.Status == v1alpha1.ConditionUnknown {
			return &condition, nil
		}
	}
	if c.Status.Phase == v1alpha1.ClusterUpgrading ||
		c.Status.Phase == v1alpha1.ClusterRunning {
		return &v1alpha1.ClusterCondition{
			Type:    handlers[0].Name(),
			Status:  v1alpha1.ConditionUnknown,
			Message: "waiting process",
			Reason:  ReasonWaiting,
		}, nil
	}
	return nil, errors.New("no condition need process")
}
