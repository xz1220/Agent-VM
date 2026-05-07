package service

import (
	"context"
	"errors"

	"github.com/xz1220/agent-vm/internal/app/model"
	"github.com/xz1220/agent-vm/internal/infra/agentstore"
	"github.com/xz1220/agent-vm/internal/infra/capstore"
	"github.com/xz1220/agent-vm/internal/infra/packageio"
)

// PackageService implements PRD §4.5 package install/export/inspect.
type PackageService interface {
	List(ctx context.Context) ([]model.PackageSummary, error)
	Show(ctx context.Context, name string) (*model.PackageDetail, error)
	Install(ctx context.Context, req model.InstallRequest) (*model.InstallResult, error)
	Uninstall(ctx context.Context, name string) error
	Export(ctx context.Context, req model.ExportRequest) (*model.ExportResult, error)
	Inspect(ctx context.Context, file string) (*model.PackageDetail, error)
}

// Packages is the default PackageService.
type Packages struct {
	Agents agentstore.Repository
	Caps   capstore.Store
	IO     packageio.IO
}

func NewPackages(agents agentstore.Repository, caps capstore.Store, io packageio.IO) *Packages {
	return &Packages{Agents: agents, Caps: caps, IO: io}
}

func (s *Packages) List(ctx context.Context) ([]model.PackageSummary, error) { return nil, nil }

func (s *Packages) Show(ctx context.Context, name string) (*model.PackageDetail, error) {
	return nil, errors.New("packages: Show not yet implemented")
}

func (s *Packages) Install(ctx context.Context, req model.InstallRequest) (*model.InstallResult, error) {
	return nil, errors.New("packages: Install not yet implemented")
}

func (s *Packages) Uninstall(ctx context.Context, name string) error {
	return errors.New("packages: Uninstall not yet implemented")
}

func (s *Packages) Export(ctx context.Context, req model.ExportRequest) (*model.ExportResult, error) {
	return nil, errors.New("packages: Export not yet implemented")
}

func (s *Packages) Inspect(ctx context.Context, file string) (*model.PackageDetail, error) {
	return nil, errors.New("packages: Inspect not yet implemented")
}
