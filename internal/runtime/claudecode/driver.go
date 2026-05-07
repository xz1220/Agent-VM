// Package claudecode implements the Claude Code RuntimeDriver.
package claudecode

import (
	"context"
	"errors"

	"github.com/xz1220/agent-vm/internal/app/model"
	"github.com/xz1220/agent-vm/internal/runtime"
)

const Name = "claude-code"

type Driver struct{}

func New() *Driver { return &Driver{} }

func (d *Driver) Name() string { return Name }

func (d *Driver) Facts(ctx context.Context) (runtime.Facts, error) {
	return runtime.Facts{Name: Name}, nil
}

func (d *Driver) DiscoverGlobal(ctx context.Context) ([]model.GlobalCapability, error) {
	return nil, nil
}

func (d *Driver) Plan(ctx context.Context, agent *model.Agent) (*runtime.Plan, error) {
	return nil, errors.New("claude-code: Plan not yet implemented")
}

func (d *Driver) Boundary(ctx context.Context, agent *model.Agent) (runtime.Boundary, error) {
	return runtime.Boundary{}, errors.New("claude-code: Boundary not yet implemented")
}

func (d *Driver) LaunchSpec(ctx context.Context, agent *model.Agent, plan *runtime.Plan) (runtime.LaunchSpec, error) {
	return runtime.LaunchSpec{}, errors.New("claude-code: LaunchSpec not yet implemented")
}
