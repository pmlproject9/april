package machine

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"time"

	"pml.io/april/pkg/util/log"

	"k8s.io/apimachinery/pkg/util/validation/field"
	platform "pml.io/april/pkg/apis/platform/v1alpha1"
	typesv1 "pml.io/april/pkg/platform/provider/type"
)

const (
	ReasonWaiting      = "Waiting"
	ReasonSkip         = "Skip"
	ReasonFailedInit   = "FailedInit"
	ReasonFailedUpdate = "FailedUpdate"
	ReasonFailedDelete = "FailedDelete"

	ConditionTypeDone = "EnsureDone"
)

// Provider defines a set of response interfaces for specific machine
// types in machine management.
type Provider interface {
	Name() string

	Validate(machine *platform.Machine) field.ErrorList

	PreCreate(machine *platform.Machine) error
	AfterCreate(machine *platform.Machine) error

	OnCreate(ctx context.Context, machine *platform.Machine, cluster *typesv1.Cluster) error
	OnUpdate(ctx context.Context, machine *platform.Machine, cluster *typesv1.Cluster) error
	OnDelete(ctx context.Context, machine *platform.Machine, cluster *typesv1.Cluster) error
}

var _ Provider = &DelegateProvider{}

type Handler func(context.Context, *platform.Machine, *typesv1.Cluster) error

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

	ValidateFunc    func(machine *platform.Machine) field.ErrorList
	PreCreateFunc   func(machine *platform.Machine) error
	AfterCreateFunc func(machine *platform.Machine) error

	CreateHandlers []Handler
	DeleteHandlers []Handler
	UpdateHandlers []Handler
}

func (p *DelegateProvider) Name() string {
	if p.ProviderName == "" {
		return "unknown"
	}
	return p.ProviderName
}

func (p *DelegateProvider) Validate(machine *platform.Machine) field.ErrorList {
	if p.ValidateFunc != nil {
		return p.ValidateFunc(machine)
	}

	return nil
}

func (p *DelegateProvider) PreCreate(machine *platform.Machine) error {
	if p.PreCreateFunc != nil {
		return p.PreCreateFunc(machine)
	}

	return nil
}

func (p *DelegateProvider) AfterCreate(machine *platform.Machine) error {
	if p.AfterCreateFunc != nil {
		return p.AfterCreateFunc(machine)
	}

	return nil
}

func (p *DelegateProvider) OnCreate(ctx context.Context, machine *platform.Machine, cluster *typesv1.Cluster) error {
	condition, err := p.getCreateCurrentCondition(machine)
	if err != nil {
		return err
	}
	handler := p.getCreateHandler(condition.Type)
	if handler == nil {
		return fmt.Errorf("can't get handler by %s", condition.Type)
	}

	ctxLog := log.FromContext(ctx).WithName("MachineProvider.OnCreate").WithName(handler.Name()).WithContext(ctx)
	log.FromContext(ctxLog).Info("Doing")
	startTime := time.Now()
	err = handler(ctxLog, machine, cluster)
	log.FromContext(ctxLog).Info("Done", "error", err, "cost", time.Since(startTime).String())
	if err != nil {
		machine.SetCondition(platform.MachineCondition{
			Type:    condition.Type,
			Status:  platform.ConditionFalse,
			Message: err.Error(),
			Reason:  ReasonFailedInit,
		})
		return err
	}

	machine.SetCondition(platform.MachineCondition{
		Type:   condition.Type,
		Status: platform.ConditionTrue,
	})

	nextConditionType := p.getNextConditionType(condition.Type)
	if nextConditionType == ConditionTypeDone {
		machine.Status.Phase = platform.MachineRunning
	} else {
		machine.SetCondition(platform.MachineCondition{
			Type:    nextConditionType,
			Status:  platform.ConditionUnknown,
			Message: "waiting execute",
			Reason:  ReasonWaiting,
		})
	}

	return nil
}

func (p *DelegateProvider) OnUpdate(ctx context.Context, machine *platform.Machine, cluster *typesv1.Cluster) error {
	if machine.Status.Phase != platform.MachineUpgrading {
		return nil
	}
	for _, handler := range p.UpdateHandlers {
		ctx := log.FromContext(ctx).WithName("MachineProvider.OnUpdate").WithName(handler.Name()).WithContext(ctx)
		log.FromContext(ctx).Info("Doing")
		startTime := time.Now()
		err := handler(ctx, machine, cluster)
		log.FromContext(ctx).Info("Done", "error", err, "cost", time.Since(startTime).String())
		if err != nil {
			machine.Status.Reason = ReasonFailedUpdate
			machine.Status.Message = fmt.Sprintf("%s error: %v", handler.Name(), err)
			return err
		}
	}
	machine.Status.Reason = ""
	machine.Status.Message = ""

	return nil
}

func (p *DelegateProvider) OnDelete(ctx context.Context, machine *platform.Machine, cluster *typesv1.Cluster) error {
	for _, handler := range p.DeleteHandlers {
		ctx := log.FromContext(ctx).WithName("MachineProvider.OnDelete").WithName(handler.Name()).WithContext(ctx)
		log.FromContext(ctx).Info("Doing")
		startTime := time.Now()
		err := handler(ctx, machine, cluster)
		log.FromContext(ctx).Info("Done", "error", err, "cost", time.Since(startTime).String())
	}
	return nil
}

func (p *DelegateProvider) getNextConditionType(conditionType string) string {
	var (
		i       int
		handler Handler
	)
	for i, handler = range p.CreateHandlers {
		name := handler.Name()
		if name == conditionType {
			break
		}
	}
	if i == len(p.CreateHandlers)-1 {
		return ConditionTypeDone
	}
	next := p.CreateHandlers[i+1]

	return next.Name()
}

func (p *DelegateProvider) getCreateHandler(conditionType string) Handler {
	for _, f := range p.CreateHandlers {
		if conditionType == f.Name() {
			return f
		}
	}

	return nil
}

func (p *DelegateProvider) getCreateCurrentCondition(c *platform.Machine) (*platform.MachineCondition, error) {
	if c.Status.Phase == platform.MachineRunning {
		return nil, errors.New("machine phases is running now")
	}
	if len(p.CreateHandlers) == 0 {
		return nil, errors.New("no create handlers")
	}

	if len(c.Status.Conditions) == 0 {
		return &platform.MachineCondition{
			Type:    p.CreateHandlers[0].Name(),
			Status:  platform.ConditionUnknown,
			Message: "waiting process",
			Reason:  ReasonWaiting,
		}, nil
	}

	for _, condition := range c.Status.Conditions {
		if condition.Status == platform.ConditionFalse || condition.Status == platform.ConditionUnknown {
			return &condition, nil
		}
	}

	return nil, errors.New("no condition need process")
}
