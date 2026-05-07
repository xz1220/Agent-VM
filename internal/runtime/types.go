// Package runtime defines the Runtime Integration boundary: how AVM's
// stable Agent semantics get translated into a specific runtime's
// managed config, isolation boundary and launch invocation.
//
// A RuntimeDriver is a runtime's single aggregate entry point. The
// application layer never reaches into per-runtime adapters/launchers
// directly — it only depends on Registry/Driver here.
package runtime

import (
	"io/fs"

	"github.com/xz1220/agent-vm/internal/app/model"
)

// Facts answers: does this runtime exist, what version, what does it
// support, and what known risks should AVM warn about.
type Facts struct {
	Name         string
	Available    bool
	BinaryPath   string
	Version      string
	Capabilities []string
	Risks        []Risk
}

// Risk is a structured runtime-known caveat AVM surfaces to users.
type Risk struct {
	Code    string
	Message string
}

// Plan is the rendered managed-config plan plus per-field mapping
// status. Drivers produce it; application layer decides whether to
// apply it; infrastructure executes the writes.
type Plan struct {
	Files    []ManagedFile
	Mappings []FieldMapping
	Warnings []model.Warning
}

// ManagedFile describes a single file the runtime expects AVM to write.
type ManagedFile struct {
	Path     string
	Mode     fs.FileMode
	Contents []byte
}

// FieldMapping is the per-field translation result.
type FieldMapping struct {
	Field  string
	Status model.MappingStatus
	Note   string
}

// Boundary describes the isolation boundary for an Agent×runtime
// combination: a private state dir and the env vars that point the
// runtime at it. Boundary materialization (e.g. symlinking caps) is
// performed by infra, not the driver.
type Boundary struct {
	StateDir string
	Env      map[string]string
}

// LaunchSpec captures how to spawn the runtime.
type LaunchSpec struct {
	Bin     string
	Args    []string
	Env     map[string]string
	Workdir string
	Stdin   bool // true if stdin should be inherited from parent
}
