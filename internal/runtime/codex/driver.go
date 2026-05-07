// Package codex implements the Codex RuntimeDriver. It is responsible for
// translating AVM Agent semantics into CODEX_HOME-managed config files,
// detecting the codex binary, computing the per-Agent isolation boundary,
// and producing a launch spec.
package codex

import (
	"context"
	"errors"

	"github.com/xz1220/agent-vm/internal/app/model"
	"github.com/xz1220/agent-vm/internal/runtime"
)

// Name is the canonical Registry key for this driver.
const Name = "codex"

// Driver is the Codex runtime adapter. Construction is via New so we
// can later inject filesystem helpers, env probes, etc.
type Driver struct{}

// New returns a Codex driver.
func New() *Driver { return &Driver{} }

func (d *Driver) Name() string { return Name }

func (d *Driver) Facts(ctx context.Context) (runtime.Facts, error) {
	// TODO(stage:runtime): probe binary path + version.
	return runtime.Facts{Name: Name}, nil
}

func (d *Driver) DiscoverGlobal(ctx context.Context) ([]model.GlobalCapability, error) {
	// TODO(stage:runtime): scan ~/.codex skills/MCPs.
	return nil, nil
}

func (d *Driver) Plan(ctx context.Context, agent *model.Agent) (*runtime.Plan, error) {
	return nil, errors.New("codex: Plan not yet implemented")
}

func (d *Driver) Boundary(ctx context.Context, agent *model.Agent) (runtime.Boundary, error) {
	return runtime.Boundary{}, errors.New("codex: Boundary not yet implemented")
}

func (d *Driver) LaunchSpec(ctx context.Context, agent *model.Agent, plan *runtime.Plan) (runtime.LaunchSpec, error) {
	return runtime.LaunchSpec{}, errors.New("codex: LaunchSpec not yet implemented")
}
