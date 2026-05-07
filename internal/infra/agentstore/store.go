// Package agentstore persists Agent definitions to disk under
// $AVM_HOME/agents/<name>.yaml. It implements AgentRepository.
package agentstore

import (
	"errors"

	"github.com/xz1220/agent-vm/internal/app/model"
)

// Repository is the contract Application layer uses to load/save Agents.
// It does not know anything about runtimes — it just persists model.Agent.
type Repository interface {
	Get(name string) (*model.Agent, error)
	List() ([]model.AgentSummary, error)
	Save(agent *model.Agent) error
	Delete(name string) error
	Exists(name string) bool
	SourcePath(name string) (string, error)
}

// ErrNotFound is returned when an Agent does not exist.
var ErrNotFound = errors.New("agentstore: not found")

// ErrConflict is returned when Save would overwrite without permission.
var ErrConflict = errors.New("agentstore: already exists")

// FSRepo is a placeholder filesystem-backed Repository. It is wired in
// the skeleton stage and filled in during the infra implementation pass.
type FSRepo struct {
	Dir string
}

// New returns an FSRepo bound to dir.
func New(dir string) *FSRepo { return &FSRepo{Dir: dir} }

func (r *FSRepo) Get(name string) (*model.Agent, error) {
	return nil, errors.New("agentstore: Get not yet implemented")
}

func (r *FSRepo) List() ([]model.AgentSummary, error) {
	return nil, nil
}

func (r *FSRepo) Save(agent *model.Agent) error {
	return errors.New("agentstore: Save not yet implemented")
}

func (r *FSRepo) Delete(name string) error {
	return errors.New("agentstore: Delete not yet implemented")
}

func (r *FSRepo) Exists(name string) bool { return false }

func (r *FSRepo) SourcePath(name string) (string, error) {
	return "", errors.New("agentstore: SourcePath not yet implemented")
}
