package service

import (
	"context"
	"errors"

	"github.com/xz1220/agent-vm/internal/app/model"
	"github.com/xz1220/agent-vm/internal/infra/agentstore"
	"github.com/xz1220/agent-vm/internal/infra/managedfile"
	"github.com/xz1220/agent-vm/internal/infra/process"
	"github.com/xz1220/agent-vm/internal/infra/runlog"
	"github.com/xz1220/agent-vm/internal/runtime"
)

// RunService implements PRD §4.4 run flow:
//
//	resolve runtime → render plan → drift check → write managed config
//	→ spawn runtime → record run log → return result/preview.
type RunService interface {
	Preview(ctx context.Context, req model.RunRequest) (*model.RunPreview, error)
	Run(ctx context.Context, req model.RunRequest) (*model.RunResult, error)
}

// Runner is the default RunService.
type Runner struct {
	Repo     agentstore.Repository
	Runtimes runtime.Registry
	Writer   managedfile.Writer
	Process  process.Runner
	Log      runlog.Log
}

func NewRunner(
	repo agentstore.Repository,
	registry runtime.Registry,
	writer managedfile.Writer,
	proc process.Runner,
	log runlog.Log,
) *Runner {
	return &Runner{Repo: repo, Runtimes: registry, Writer: writer, Process: proc, Log: log}
}

func (s *Runner) Preview(ctx context.Context, req model.RunRequest) (*model.RunPreview, error) {
	return nil, errors.New("runner: Preview not yet implemented")
}

func (s *Runner) Run(ctx context.Context, req model.RunRequest) (*model.RunResult, error) {
	return nil, errors.New("runner: Run not yet implemented")
}
