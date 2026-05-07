package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xz1220/agent-vm/internal/infra/home"
)

func newTestSystem(t *testing.T) *System {
	t.Helper()
	root := filepath.Join(t.TempDir(), ".avm")
	return NewSystem(home.Layout{Root: root})
}

func TestSystem_Init_FreshAndIdempotent(t *testing.T) {
	sys := newTestSystem(t)
	res, err := sys.Init(context.Background())
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	if res.AlreadyExists {
		t.Fatalf("expected fresh init, got AlreadyExists=true")
	}
	if res.Root != sys.Layout.Root {
		t.Fatalf("root mismatch: %q vs %q", res.Root, sys.Layout.Root)
	}
	if len(res.CreatedPaths) == 0 {
		t.Fatalf("expected CreatedPaths populated")
	}
	for _, p := range res.CreatedPaths {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("expected %s to exist: %v", p, err)
		}
	}

	// Second call must report already initialised.
	res2, err := sys.Init(context.Background())
	if err != nil {
		t.Fatalf("init re-run: %v", err)
	}
	if !res2.AlreadyExists {
		t.Fatalf("expected AlreadyExists=true on re-run")
	}
	if len(res2.CreatedPaths) != 0 {
		t.Fatalf("expected empty CreatedPaths on re-run, got %v", res2.CreatedPaths)
	}
}

func TestSystem_UninstallHome_RemovesAndIsIdempotent(t *testing.T) {
	sys := newTestSystem(t)
	if _, err := sys.Init(context.Background()); err != nil {
		t.Fatalf("init: %v", err)
	}

	res, err := sys.UninstallHome(context.Background())
	if err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	if !res.Removed {
		t.Fatalf("expected Removed=true")
	}
	if _, err := os.Stat(sys.Layout.Root); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected root gone, got %v", err)
	}

	// Re-running on a missing home is a no-op, not an error.
	res2, err := sys.UninstallHome(context.Background())
	if err != nil {
		t.Fatalf("uninstall re-run: %v", err)
	}
	if res2.Removed {
		t.Fatalf("expected Removed=false on missing home")
	}
}

func TestSystem_HomeRoot(t *testing.T) {
	sys := newTestSystem(t)
	got, err := sys.HomeRoot(context.Background())
	if err != nil {
		t.Fatalf("HomeRoot: %v", err)
	}
	if got != sys.Layout.Root {
		t.Fatalf("HomeRoot mismatch: %q vs %q", got, sys.Layout.Root)
	}
}

func TestSystem_CompletionPath(t *testing.T) {
	sys := newTestSystem(t)
	for _, sh := range []string{"bash", "zsh", "fish"} {
		p, err := sys.CompletionPath(context.Background(), sh)
		if err != nil {
			t.Fatalf("CompletionPath(%s): %v", sh, err)
		}
		if !strings.HasPrefix(p, sys.Layout.Root) {
			t.Fatalf("path not under root: %q", p)
		}
		if !strings.HasSuffix(p, "avm-completion."+sh) {
			t.Fatalf("unexpected suffix: %q", p)
		}
		if _, err := os.Stat(filepath.Dir(p)); err != nil {
			t.Fatalf("expected parent dir created: %v", err)
		}
	}
}

func TestSystem_CompletionPath_Rejects(t *testing.T) {
	sys := newTestSystem(t)
	if _, err := sys.CompletionPath(context.Background(), ""); err == nil {
		t.Fatalf("expected error for empty shell")
	}
	if _, err := sys.CompletionPath(context.Background(), "tcsh"); err == nil {
		t.Fatalf("expected error for unsupported shell")
	}
}

func TestSystem_RemoveCompletion_Idempotent(t *testing.T) {
	sys := newTestSystem(t)
	p, err := sys.CompletionPath(context.Background(), "bash")
	if err != nil {
		t.Fatalf("CompletionPath: %v", err)
	}
	// Create the file, then ensure RemoveCompletion deletes it.
	if err := os.WriteFile(p, []byte("# completion"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := sys.RemoveCompletion(context.Background(), "bash"); err != nil {
		t.Fatalf("RemoveCompletion: %v", err)
	}
	if _, err := os.Stat(p); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected file gone, got %v", err)
	}
	// Second call (file already gone) must not error.
	if err := sys.RemoveCompletion(context.Background(), "bash"); err != nil {
		t.Fatalf("RemoveCompletion (idempotent): %v", err)
	}
}
