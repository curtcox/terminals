package appruntime

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Definitions returns one app definition per exported entrypoint.
func (r *Runtime) Definitions() []AppDefinition {
	if r == nil {
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]AppDefinition, 0, len(r.packages))
	names := make([]string, 0, len(r.packages))
	for name := range r.packages {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, packageName := range names {
		pkg := r.packages[packageName]
		exports := pkg.Manifest.Exports
		if len(exports) == 0 {
			exports = []string{pkg.Manifest.Name}
		}
		for _, exportName := range exports {
			exportName = strings.TrimSpace(exportName)
			if exportName == "" {
				continue
			}
			out = append(out, &manifestDefinition{
				runtime:    r,
				packageRef: pkg.Manifest.Name,
				exportName: exportName,
			})
		}
	}
	return out
}

type manifestDefinition struct {
	runtime    *Runtime
	packageRef string
	exportName string
}

func (d *manifestDefinition) Name() string {
	if strings.EqualFold(d.packageRef, d.exportName) {
		return fmt.Sprintf("app.%s", d.packageRef)
	}
	return fmt.Sprintf("app.%s.%s", d.packageRef, d.exportName)
}

func (d *manifestDefinition) Match(req ActivationRequest) bool {
	intent := strings.TrimSpace(req.Intent)
	if intent == "" {
		return false
	}
	return strings.EqualFold(intent, d.exportName) ||
		strings.EqualFold(intent, d.Name()) ||
		strings.EqualFold(intent, "app."+d.exportName)
}

func (d *manifestDefinition) NewActivation(req ActivationRequest) (AppActivation, error) {
	if !d.Match(req) {
		return nil, nil
	}
	d.runtime.mu.RLock()
	pkg, ok := d.runtime.packages[d.packageRef]
	d.runtime.mu.RUnlock()
	if !ok {
		return nil, ErrPackageNotFound
	}
	return &manifestActivation{
		id:             activationID(pkg.Manifest.Name, d.exportName, req.DeviceID, pkg.Revision),
		definitionName: d.Name(),
		pinnedRevision: pkg.Revision,
		createdAt:      time.Now().UTC(),
	}, nil
}

type manifestActivation struct {
	id             string
	definitionName string
	pinnedRevision uint64
	createdAt      time.Time
}

func (a *manifestActivation) ID() string {
	return a.id
}

func (a *manifestActivation) DefinitionName() string {
	return a.definitionName
}

func (a *manifestActivation) Start(ctx context.Context, env *Environment) error {
	_ = ctx
	_ = env
	return nil
}

func (a *manifestActivation) Handle(ctx context.Context, env *Environment, trigger Trigger) error {
	_ = ctx
	_ = env
	_ = trigger
	return nil
}

func (a *manifestActivation) Stop(ctx context.Context, env *Environment) error {
	_ = ctx
	_ = env
	return nil
}

func (a *manifestActivation) Suspend(ctx context.Context, env *Environment) error {
	_ = ctx
	_ = env
	return nil
}

func (a *manifestActivation) Resume(ctx context.Context, env *Environment) error {
	_ = ctx
	_ = env
	return nil
}

func activationID(packageName, exportName, deviceID string, revision uint64) string {
	packageName = strings.TrimSpace(packageName)
	exportName = strings.TrimSpace(exportName)
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		deviceID = "global"
	}
	return fmt.Sprintf("app:%s:%s:%s:r%d", packageName, exportName, deviceID, revision)
}
