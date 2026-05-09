// Package model defines AVM's stable product semantics: the structures
// AVM promises to understand, validate, render, package, and migrate.
//
// What belongs here:
//   - Agent, Identity, Instructions, CapabilityRef, RuntimePreference
//   - PackageManifest
//   - MappingStatus enum
//   - Run preview/result and diagnostics value objects
//
// What does NOT belong here:
//   - Runtime-native config DTOs (Codex TOML, Claude config JSON, ...)
//   - Adapter intermediate structures
//   - CLI request DTOs that are not part of the durable model
//
// JSON tags are part of the public CLI protocol (see
// docs/api/cli-protocol.md). YAML tags drive on-disk Agent files.
package model

import (
	"errors"
	"regexp"
)

var agentNameRE = regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}$`)

// Agent is the user's primary object: created, edited, deleted, cloned,
// renamed, run, and shared. It captures a stable AVM-owned definition
// that is independent of any specific runtime's config files.
type Agent struct {
	Identity     Identity        `yaml:"identity"           json:"identity"`
	Instructions Instructions    `yaml:"instructions"       json:"instructions"`
	Skills       []CapabilityRef `yaml:"skills,omitempty"   json:"skills,omitempty"`
	MCP          []CapabilityRef `yaml:"mcp,omitempty"      json:"mcp,omitempty"`
	Runtimes     []RuntimePref   `yaml:"runtimes,omitempty" json:"runtimes,omitempty"`
}

// Identity is the user-facing identity of the Agent.
type Identity struct {
	Name        string `yaml:"name"                  json:"name"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	Role        string `yaml:"role,omitempty"        json:"role,omitempty"`
}

// Instructions captures system/developer instructions and referenced files.
type Instructions struct {
	System string   `yaml:"system,omitempty" json:"system,omitempty"`
	Files  []string `yaml:"files,omitempty"  json:"files,omitempty"`
	Inline string   `yaml:"inline,omitempty" json:"inline,omitempty"`
}

// CapabilityRef is an AVM-stable reference to a capability stored in the
// AVM capability store. It does not encode runtime-specific paths or
// package provenance — those are tracked separately as audit metadata.
type CapabilityRef struct {
	ID   CapabilityID   `yaml:"id"   json:"id"`
	Kind CapabilityKind `yaml:"kind" json:"kind"`
}

// RuntimePref expresses which runtimes an Agent is intended to run on
// and the AVM-level preferences for each. It is not the runtime's own
// config — it is the AVM-side intent that drives adapter mapping.
type RuntimePref struct {
	Runtime string `yaml:"runtime"           json:"runtime"`
	Default bool   `yaml:"default,omitempty" json:"default,omitempty"`
}

// AgentSummary is a list-friendly projection used by AgentService.List.
type AgentSummary struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Runtimes    []string `json:"runtimes,omitempty"`
}

// AgentDetail is the show-friendly projection used by AgentService.Show.
type AgentDetail struct {
	Agent      Agent                   `json:"agent"`
	SourcePath string                  `json:"source_path,omitempty"`
	Mapping    []RuntimeMappingSummary `json:"mapping,omitempty"`
}

// RuntimeMappingSummary describes per-runtime mapping status for "agent show".
type RuntimeMappingSummary struct {
	Runtime  string                `json:"runtime"`
	Fields   []FieldMappingSummary `json:"fields,omitempty"`
	Warnings []string              `json:"warnings,omitempty"`
}

// FieldMappingSummary is the model-layer projection of a runtime field mapping.
type FieldMappingSummary struct {
	Field  string        `json:"field"`
	Status MappingStatus `json:"status"`
	Note   string        `json:"note,omitempty"`
}

// Validate performs minimal AVM-layer validation. Runtime-specific
// validation is performed by RuntimeDriver.Plan.
func (a *Agent) Validate() error {
	if a == nil {
		return errors.New("agent: nil")
	}
	if !agentNameRE.MatchString(a.Identity.Name) {
		return errors.New("agent: identity.name must match [a-z][a-z0-9-]{0,62}")
	}
	return nil
}
