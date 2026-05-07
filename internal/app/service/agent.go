// Package service hosts AVM's application-layer use cases. Services
// orchestrate model, runtime and infra to fulfil PRD §4 user actions.
//
// Services own product rules. They do not own runtime-specific config
// file details, and they never write runtime-managed paths directly.
package service

import (
	"context"
	"errors"

	"github.com/xz1220/agent-vm/internal/app/model"
	"github.com/xz1220/agent-vm/internal/infra/agentstore"
	"github.com/xz1220/agent-vm/internal/runtime"
)

// AgentService implements PRD §4.2 Agent CRUD.
type AgentService interface {
	Create(ctx context.Context, req model.CreateAgentRequest) (*model.Agent, error)
	List(ctx context.Context) ([]model.AgentSummary, error)
	Show(ctx context.Context, name string) (*model.AgentDetail, error)
	Edit(ctx context.Context, req model.EditAgentRequest) (*model.Agent, error)
	Delete(ctx context.Context, req model.DeleteAgentRequest) error
	Clone(ctx context.Context, name, newName string) (*model.Agent, error)
	Rename(ctx context.Context, oldName, newName string) (*model.Agent, error)
}

// Agents is the default AgentService.
type Agents struct {
	Repo     agentstore.Repository
	Runtimes runtime.Registry
}

// NewAgents constructs the default AgentService.
func NewAgents(repo agentstore.Repository, registry runtime.Registry) *Agents {
	return &Agents{Repo: repo, Runtimes: registry}
}

func (s *Agents) Create(ctx context.Context, req model.CreateAgentRequest) (*model.Agent, error) {
	return nil, errors.New("agents: Create not yet implemented")
}

func (s *Agents) List(ctx context.Context) ([]model.AgentSummary, error) {
	if s.Repo == nil {
		return nil, errors.New("agents: missing repository")
	}
	return s.Repo.List()
}

func (s *Agents) Show(ctx context.Context, name string) (*model.AgentDetail, error) {
	return nil, errors.New("agents: Show not yet implemented")
}

func (s *Agents) Edit(ctx context.Context, req model.EditAgentRequest) (*model.Agent, error) {
	return nil, errors.New("agents: Edit not yet implemented")
}

func (s *Agents) Delete(ctx context.Context, req model.DeleteAgentRequest) error {
	return errors.New("agents: Delete not yet implemented")
}

func (s *Agents) Clone(ctx context.Context, name, newName string) (*model.Agent, error) {
	return nil, errors.New("agents: Clone not yet implemented")
}

func (s *Agents) Rename(ctx context.Context, oldName, newName string) (*model.Agent, error) {
	return nil, errors.New("agents: Rename not yet implemented")
}
