package service

import (
	"context"
	"errors"

	"github.com/xz1220/agent-vm/internal/app/model"
	"github.com/xz1220/agent-vm/internal/infra/capstore"
	"github.com/xz1220/agent-vm/internal/runtime"
)

// CapabilityService unifies AVM-managed and runtime-global capability
// candidates per PRD §4.2. The list MUST be live: every call must
// reflect runtime-global state at the moment the user sees it.
type CapabilityService interface {
	Discover(ctx context.Context, req model.DiscoverRequest) ([]model.CapabilityCandidate, error)
}

// Capabilities is the default CapabilityService.
type Capabilities struct {
	Store    capstore.Store
	Runtimes runtime.Registry
}

func NewCapabilities(store capstore.Store, registry runtime.Registry) *Capabilities {
	return &Capabilities{Store: store, Runtimes: registry}
}

func (s *Capabilities) Discover(ctx context.Context, req model.DiscoverRequest) ([]model.CapabilityCandidate, error) {
	return nil, errors.New("capabilities: Discover not yet implemented")
}
