package model

import "time"

// PackageManifest is the metadata at the top of an .avm.zip package.
// Package contents (Agent, capability blobs) are referenced from the
// manifest but not embedded here.
type PackageManifest struct {
	SchemaVersion string            `yaml:"schema_version"`
	Name          string            `yaml:"name"`
	Version       string            `yaml:"version"`
	Description   string            `yaml:"description,omitempty"`
	Author        string            `yaml:"author,omitempty"`
	CreatedAt     time.Time         `yaml:"created_at"`
	Agents        []PackageAgentRef `yaml:"agents,omitempty"`
	Capabilities  []PackageCapBlob  `yaml:"capabilities,omitempty"`
}

// PackageAgentRef points at an Agent file within the package zip.
type PackageAgentRef struct {
	Name string `yaml:"name"`
	Path string `yaml:"path"`
}

// PackageCapBlob describes a capability blob shipped inside the package.
// The capability is imported into the AVM capability store on install;
// it is not referenced from a package-private directory at runtime.
type PackageCapBlob struct {
	Kind     CapabilityKind `yaml:"kind"`
	Name     string         `yaml:"name"`
	Path     string         `yaml:"path"`
	Checksum string         `yaml:"checksum"`
}

// PackageSummary is the list projection of a package.
type PackageSummary struct {
	Name        string
	Version     string
	Description string
	Source      string // file path or registry slug
}

// PackageDetail is the show/inspect projection.
type PackageDetail struct {
	Manifest PackageManifest
	Files    []string
	Source   string
}

// InstallRequest captures install intent. Conflicts (same name) are
// resolved via Resolution; non-interactive callers must pre-decide.
type InstallRequest struct {
	Source         string // file path or registry slug
	Resolution     ConflictResolution
	NonInteractive bool
}

// ConflictResolution mirrors PRD §4.5.
type ConflictResolution string

const (
	ResolveAsk       ConflictResolution = ""
	ResolveRename    ConflictResolution = "rename"
	ResolveSkip      ConflictResolution = "skip"
	ResolveOverwrite ConflictResolution = "overwrite"
	ResolveCancel    ConflictResolution = "cancel"
)

// InstallResult reports what was actually written.
type InstallResult struct {
	Manifest        PackageManifest
	InstalledAgents []string
	ImportedCaps    []CapabilityID
	Skipped         []string
	Renamed         map[string]string // old -> new
}

// ExportRequest captures the user's intent to export an Agent.
type ExportRequest struct {
	Agent         string
	IncludeSkills bool
	IncludeMCP    bool
	OutputPath    string
}

// ExportResult reports the artifact written to disk.
type ExportResult struct {
	Manifest PackageManifest
	Path     string
}
