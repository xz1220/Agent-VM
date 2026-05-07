// Package process provides ProcessRunner: spawning runtime processes
// described by runtime.LaunchSpec and reporting their lifecycle.
package process

import (
	"context"
	"errors"

	"github.com/xz1220/agent-vm/internal/runtime"
)

// Result reports a finished process.
type Result struct {
	ExitCode int
}

// Runner is the contract Application layer uses to launch runtimes.
type Runner interface {
	Run(ctx context.Context, spec runtime.LaunchSpec) (Result, error)
}

// OS is the default os/exec-backed runner.
type OS struct{}

func New() *OS { return &OS{} }

func (OS) Run(ctx context.Context, spec runtime.LaunchSpec) (Result, error) {
	return Result{}, errors.New("process: Run not yet implemented")
}
