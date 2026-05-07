// Package home owns the AVM home directory layout. It is the single
// place that knows where Agents/Capabilities/RunLog/etc live.
package home

import (
	"errors"
	"os"
	"path/filepath"
)

// Layout describes the on-disk paths AVM owns under ~/.avm.
//
// Paths are computed from a single Root so tests can swap to a tmp dir.
type Layout struct {
	Root string
}

// EnvVar is the override for AVM home root.
const EnvVar = "AVM_HOME"

// DefaultLayout returns the user's default AVM home, honoring $AVM_HOME.
func DefaultLayout() (Layout, error) {
	if v := os.Getenv(EnvVar); v != "" {
		return Layout{Root: v}, nil
	}
	hd, err := os.UserHomeDir()
	if err != nil {
		return Layout{}, err
	}
	if hd == "" {
		return Layout{}, errors.New("home: empty user home dir")
	}
	return Layout{Root: filepath.Join(hd, ".avm")}, nil
}

// AgentsDir is where Agent YAMLs live.
func (l Layout) AgentsDir() string { return filepath.Join(l.Root, "agents") }

// CapabilitiesDir is the AVM capability store root.
func (l Layout) CapabilitiesDir() string { return filepath.Join(l.Root, "capabilities") }

// PackagesDir caches installed packages' metadata.
func (l Layout) PackagesDir() string { return filepath.Join(l.Root, "packages") }

// RunLogDir holds run history records.
func (l Layout) RunLogDir() string { return filepath.Join(l.Root, "runlog") }

// BoundaryDir is the per-Agent×runtime isolation parent.
func (l Layout) BoundaryDir() string { return filepath.Join(l.Root, "boundaries") }

// EnsureDirs creates all subdirs lazily; safe to call repeatedly.
func (l Layout) EnsureDirs() error {
	for _, p := range []string{l.AgentsDir(), l.CapabilitiesDir(), l.PackagesDir(), l.RunLogDir(), l.BoundaryDir()} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			return err
		}
	}
	return nil
}
