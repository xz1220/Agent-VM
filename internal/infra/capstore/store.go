// Package capstore is the AVM capability store. It owns the durable
// content of skills/MCPs that Agents reference by stable ID. Package
// installs and explicit imports route through this store; runtime
// global discoveries do NOT enter here automatically.
package capstore

import (
	"errors"
	"io"

	"github.com/xz1220/agent-vm/internal/app/model"
)

// Store is the contract for AVM capability persistence.
type Store interface {
	List() ([]model.CapabilityRecord, error)
	Get(id model.CapabilityID) (model.CapabilityRecord, error)
	// Add writes payload under a new ID derived from the record. If
	// the same name+kind already exists with identical content,
	// implementations may dedupe and return the existing ID.
	Add(record model.CapabilityRecord, payload io.Reader) (model.CapabilityID, error)
	// Materialize prepares the given capabilities under target dir
	// (e.g. via symlink) so a runtime boundary can reference them.
	Materialize(ids []model.CapabilityID, target string) error
	Remove(id model.CapabilityID) error
}

// ErrNotFound is returned when an ID is unknown.
var ErrNotFound = errors.New("capstore: not found")

// FSStore is a placeholder filesystem-backed Store.
type FSStore struct{ Dir string }

func New(dir string) *FSStore { return &FSStore{Dir: dir} }

func (s *FSStore) List() ([]model.CapabilityRecord, error) { return nil, nil }
func (s *FSStore) Get(id model.CapabilityID) (model.CapabilityRecord, error) {
	return model.CapabilityRecord{}, ErrNotFound
}
func (s *FSStore) Add(rec model.CapabilityRecord, payload io.Reader) (model.CapabilityID, error) {
	return "", errors.New("capstore: Add not yet implemented")
}
func (s *FSStore) Materialize(ids []model.CapabilityID, target string) error {
	return errors.New("capstore: Materialize not yet implemented")
}
func (s *FSStore) Remove(id model.CapabilityID) error {
	return errors.New("capstore: Remove not yet implemented")
}
