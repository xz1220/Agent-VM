package runtime

import (
	"context"

	"github.com/xz1220/agent-vm/internal/app/model"
)

// Driver is the per-runtime aggregate entry point. The application
// layer only depends on this interface and Registry; how a driver
// internally splits facts/adapter/boundary/launcher is its own detail.
type Driver interface {
	// Name returns the canonical runtime name (codex, claude-code, opencode...).
	Name() string

	// Facts probes runtime presence/version/capabilities. Cheap and
	// safe to call multiple times; drivers may cache internally.
	Facts(ctx context.Context) (Facts, error)

	// DiscoverGlobal returns capabilities found in the runtime's own
	// global directory (e.g. ~/.codex/skills). These are NOT AVM
	// managed; application layer must show their source clearly.
	DiscoverGlobal(ctx context.Context) ([]model.GlobalCapability, error)

	// Plan renders the Agent into the runtime's managed config.
	Plan(ctx context.Context, agent *model.Agent) (*Plan, error)

	// Boundary returns the per-Agent×runtime isolation boundary.
	Boundary(ctx context.Context, agent *model.Agent) (Boundary, error)

	// LaunchSpec describes how to spawn the runtime for this Agent.
	LaunchSpec(ctx context.Context, agent *model.Agent, plan *Plan) (LaunchSpec, error)
}
