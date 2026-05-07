package model

import "time"

// MappingStatus is the AVM-stable status of how an Agent field is
// represented in a target runtime. It is rendered to users and stored
// in run logs, so it is a model-layer concern even though the actual
// per-field mapping is computed by Runtime Integration.
type MappingStatus string

const (
	MappingNative                 MappingStatus = "native"
	MappingRenderedAsInstructions MappingStatus = "rendered_as_instructions"
	MappingIgnored                MappingStatus = "ignored"
	MappingUnsupported            MappingStatus = "unsupported"
)

// Warning is a structured warning surfaced from runtime adapters or
// drift checks. Presentation layer formats it; application layer
// records it; it is not free-form text downstream of the driver.
type Warning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// RunRequest is the application-layer request to run an Agent.
type RunRequest struct {
	Agent       string      `json:"agent"`
	Runtime     string      `json:"runtime,omitempty"`
	DriftPolicy DriftPolicy `json:"drift_policy,omitempty"`
}

// DriftPolicy describes how the user wants AVM to react when the
// runtime managed config has drifted from the AVM Agent definition.
type DriftPolicy string

const (
	DriftAsk     DriftPolicy = "" // unset; service will reject if drift detected
	DriftMerge   DriftPolicy = "merge"
	DriftDiscard DriftPolicy = "discard"
	DriftKeep    DriftPolicy = "keep" // keep this run only
)

// RunPreview is what `avm run --preview` returns: the plan AVM intends
// to apply, without actually launching the runtime.
type RunPreview struct {
	Agent      string                `json:"agent"`
	Runtime    string                `json:"runtime"`
	WritePaths []string              `json:"write_paths,omitempty"`
	Boundary   BoundarySummary       `json:"boundary,omitempty"`
	Mapping    []FieldMappingSummary `json:"mapping,omitempty"`
	Warnings   []Warning             `json:"warnings,omitempty"`
	Drift      []DiffEntry           `json:"drift,omitempty"`
}

// BoundarySummary is a model-layer projection of runtime.Boundary so
// that callers above runtime layer don't import runtime types directly.
type BoundarySummary struct {
	StateDir string   `json:"state_dir,omitempty"`
	EnvKeys  []string `json:"env_keys,omitempty"` // names only; values are not exported
}

// RunResult is the outcome of a launch.
type RunResult struct {
	Preview   RunPreview `json:"preview"`
	StartedAt time.Time  `json:"started_at"`
	EndedAt   time.Time  `json:"ended_at"`
	ExitCode  int        `json:"exit_code"`
}

// RunRecord is what RunLog persists.
type RunRecord struct {
	Agent     string      `json:"agent"`
	Runtime   string      `json:"runtime"`
	StartedAt time.Time   `json:"started_at"`
	EndedAt   time.Time   `json:"ended_at"`
	ExitCode  int         `json:"exit_code"`
	Drift     []DiffEntry `json:"drift,omitempty"`
	Warnings  []Warning   `json:"warnings,omitempty"`
}

// DiffEntry describes a drift between Agent definition and runtime
// managed config (or between desired and existing managed file).
type DiffEntry struct {
	Path   string `json:"path"`
	Field  string `json:"field,omitempty"`
	Reason string `json:"reason"`
}
