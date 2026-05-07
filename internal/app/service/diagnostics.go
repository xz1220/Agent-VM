package service

import (
	"context"

	"github.com/xz1220/agent-vm/internal/app/model"
	"github.com/xz1220/agent-vm/internal/infra/agentstore"
	"github.com/xz1220/agent-vm/internal/infra/runlog"
	"github.com/xz1220/agent-vm/internal/runtime"
)

// DiagnosticsService implements `avm doctor` and `avm status`.
type DiagnosticsService interface {
	Doctor(ctx context.Context) (*model.DoctorReport, error)
	Status(ctx context.Context, agent string) (*model.StatusReport, error)
}

// Diagnostics is the default DiagnosticsService.
type Diagnostics struct {
	Agents   agentstore.Repository
	Runtimes runtime.Registry
	Log      runlog.Log
}

func NewDiagnostics(agents agentstore.Repository, registry runtime.Registry, log runlog.Log) *Diagnostics {
	return &Diagnostics{Agents: agents, Runtimes: registry, Log: log}
}

func (s *Diagnostics) Doctor(ctx context.Context) (*model.DoctorReport, error) {
	return &model.DoctorReport{}, nil
}

func (s *Diagnostics) Status(ctx context.Context, agent string) (*model.StatusReport, error) {
	return &model.StatusReport{}, nil
}
