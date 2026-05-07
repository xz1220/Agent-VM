// Package managedfile applies runtime.Plan files to disk. It is the
// only place AVM writes runtime-managed config; no other layer pokes
// at runtime config paths directly.
package managedfile

import (
	"context"
	"errors"

	"github.com/xz1220/agent-vm/internal/app/model"
	"github.com/xz1220/agent-vm/internal/runtime"
)

// Writer applies a runtime Plan and reports drift between desired and
// existing files.
type Writer interface {
	Apply(ctx context.Context, files []runtime.ManagedFile) ([]model.DiffEntry, error)
	DryRun(ctx context.Context, files []runtime.ManagedFile) ([]model.DiffEntry, error)
}

// Default is an atomic-write-backed implementation.
type Default struct{}

func New() *Default { return &Default{} }

func (Default) Apply(ctx context.Context, files []runtime.ManagedFile) ([]model.DiffEntry, error) {
	return nil, errors.New("managedfile: Apply not yet implemented")
}

func (Default) DryRun(ctx context.Context, files []runtime.ManagedFile) ([]model.DiffEntry, error) {
	return nil, errors.New("managedfile: DryRun not yet implemented")
}
