package model

import "time"

// PackageManifest is the metadata at the top of an .avm.zip package.
// Package contents (Agent, capability blobs) are referenced from the
// manifest but not embedded here.
type PackageManifest struct {
	SchemaVersion string            `yaml:"schema_version"          json:"schema_version"`
	Name          string            `yaml:"name"                    json:"name"`
	Version       string            `yaml:"version"                 json:"version"`
	Description   string            `yaml:"description,omitempty"   json:"description,omitempty"`
	Author        string            `yaml:"author,omitempty"        json:"author,omitempty"`
	CreatedAt     time.Time         `yaml:"created_at"              json:"created_at"`
	Agents        []PackageAgentRef `yaml:"agents,omitempty"        json:"agents,omitempty"`
	Capabilities  []PackageCapBlob  `yaml:"capabilities,omitempty"  json:"capabilities,omitempty"`
}

// PackageAgentRef points at an Agent file within the package zip.
type PackageAgentRef struct {
	Name string `yaml:"name" json:"name"`
	Path string `yaml:"path" json:"path"`
}

// PackageCapBlob describes a capability blob shipped inside the package.
// The capability is imported into the AVM capability store on install;
// it is not referenced from a package-private directory at runtime.
type PackageCapBlob struct {
	Kind     CapabilityKind `yaml:"kind"     json:"kind"`
	Name     string         `yaml:"name"     json:"name"`
	SourceID CapabilityID   `yaml:"source_id,omitempty" json:"source_id,omitempty"`
	Format   string         `yaml:"format,omitempty" json:"format,omitempty"`
	Path     string         `yaml:"path"     json:"path"`
	Checksum string         `yaml:"checksum" json:"checksum"`
}

// PackageSummary is the list projection of a package.
type PackageSummary struct {
	Name        string `json:"name"`
	Version     string `json:"version,omitempty"`
	Description string `json:"description,omitempty"`
	Source      string `json:"source,omitempty"` // file path or registry slug
}

// PackageDetail is the show/inspect projection.
type PackageDetail struct {
	Manifest PackageManifest `json:"manifest"`
	Files    []string        `json:"files,omitempty"`
	Source   string          `json:"source,omitempty"`
}

// InstallRequest captures install intent. Conflicts (same name) are
// resolved via Resolution; non-interactive callers must pre-decide.
type InstallRequest struct {
	Source     string             `json:"source"`
	Resolution ConflictResolution `json:"resolution,omitempty"`
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
	Manifest        PackageManifest   `json:"manifest"`
	InstalledAgents []string          `json:"installed_agents,omitempty"`
	ImportedCaps    []CapabilityID    `json:"imported_caps,omitempty"`
	Skipped         []string          `json:"skipped,omitempty"`
	Renamed         map[string]string `json:"renamed,omitempty"` // old -> new
}

// ExportRequest captures the user's intent to export an Agent.
type ExportRequest struct {
	Agent         string `json:"agent"`
	IncludeSkills bool   `json:"include_skills,omitempty"`
	IncludeMCP    bool   `json:"include_mcp,omitempty"`
	OutputPath    string `json:"output_path,omitempty"`
}

// ExportResult reports the artifact written to disk.
type ExportResult struct {
	Manifest PackageManifest `json:"manifest"`
	Path     string          `json:"path"`
}
