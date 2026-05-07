package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/xz1220/agent-vm/internal/app/model"
	"github.com/xz1220/agent-vm/internal/infra/home"
)

// SystemService owns AVM-lifecycle actions: init/uninstall the home
// directory and resolve shell-completion paths. Presentation layer
// depends on this so it never has to import infra/home directly.
type SystemService interface {
	Init(ctx context.Context) (*model.InitResult, error)
	UninstallHome(ctx context.Context) (*model.UninstallResult, error)
	CompletionPath(ctx context.Context, shell string) (string, error)
	RemoveCompletion(ctx context.Context, shell string) error
	// HomeRoot returns the AVM home root path. It is read-only: callers
	// that need to *display* the root without mutating state (e.g. the
	// uninstall dry-run) use this instead of UninstallHome.
	HomeRoot(ctx context.Context) (string, error)
}

// System is the default SystemService backed by a home.Layout.
type System struct {
	Layout home.Layout
}

// NewSystem constructs a System bound to layout. cmd/avm/main.go is
// the production composition root; tests can pass a Layout pointing
// at t.TempDir() to exercise real file effects in isolation.
func NewSystem(layout home.Layout) *System { return &System{Layout: layout} }

var supportedShells = map[string]struct{}{
	"bash": {},
	"zsh":  {},
	"fish": {},
}

func validateShell(shell string) error {
	if shell == "" {
		return MissingInputError("shell", "shell name is required (bash|zsh|fish)")
	}
	if _, ok := supportedShells[shell]; !ok {
		return NewError(CodeValidation,
			fmt.Sprintf("unsupported shell %q (want bash|zsh|fish)", shell),
			map[string]any{"shell": shell})
	}
	return nil
}

// Init creates the AVM home if absent. If the home already exists
// (detected by AgentsDir presence), AlreadyExists is true and no
// directories are touched.
func (s *System) Init(ctx context.Context) (*model.InitResult, error) {
	res := &model.InitResult{Root: s.Layout.Root}
	if _, err := os.Stat(s.Layout.AgentsDir()); err == nil {
		res.AlreadyExists = true
		return res, nil
	}
	if err := s.Layout.EnsureDirs(); err != nil {
		return nil, WrapError(CodeIOFailure, err,
			"init AVM home: "+err.Error(),
			map[string]any{"root": s.Layout.Root})
	}
	res.CreatedPaths = []string{
		s.Layout.Root,
		s.Layout.AgentsDir(),
		s.Layout.CapabilitiesDir(),
		s.Layout.PackagesDir(),
		s.Layout.RunLogDir(),
		s.Layout.BoundaryDir(),
	}
	return res, nil
}

// UninstallHome removes the AVM home directory tree. It is idempotent:
// removing a non-existent home returns Removed=false with no error.
func (s *System) UninstallHome(ctx context.Context) (*model.UninstallResult, error) {
	res := &model.UninstallResult{Root: s.Layout.Root}
	if _, err := os.Stat(s.Layout.Root); errors.Is(err, os.ErrNotExist) {
		return res, nil
	}
	if err := os.RemoveAll(s.Layout.Root); err != nil {
		return res, WrapError(CodeIOFailure, err,
			"remove AVM home: "+err.Error(),
			map[string]any{"root": s.Layout.Root})
	}
	res.Removed = true
	return res, nil
}

// CompletionPath returns the path where shell-completion for the given
// shell should be written. The parent directory is created lazily so
// callers can write the file in one step.
func (s *System) CompletionPath(ctx context.Context, shell string) (string, error) {
	if err := validateShell(shell); err != nil {
		return "", err
	}
	dir := filepath.Join(s.Layout.Root, "shell")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", WrapError(CodeIOFailure, err,
			"create shell completion dir: "+err.Error(),
			map[string]any{"dir": dir})
	}
	return filepath.Join(dir, "avm-completion."+shell), nil
}

// HomeRoot returns the AVM home root path without touching the
// filesystem. Read-only: safe for dry-run displays.
func (s *System) HomeRoot(ctx context.Context) (string, error) {
	return s.Layout.Root, nil
}

// RemoveCompletion removes the completion file for shell. Missing
// files are not an error (idempotent uninstall).
func (s *System) RemoveCompletion(ctx context.Context, shell string) error {
	if err := validateShell(shell); err != nil {
		return err
	}
	p := filepath.Join(s.Layout.Root, "shell", "avm-completion."+shell)
	if err := os.Remove(p); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return WrapError(CodeIOFailure, err,
			"remove shell completion file: "+err.Error(),
			map[string]any{"path": p})
	}
	return nil
}
