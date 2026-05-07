package model

// CreateAgentRequest captures the user's intent to create an Agent.
// Source describes blank/default vs existing-package origin (see PRD §4.2).
type CreateAgentRequest struct {
	Name           string
	Description    string
	Role           string
	Instructions   Instructions
	Skills         []CapabilityRef
	MCP            []CapabilityRef
	Runtimes       []RuntimePref
	Source         CreateSource
	OnConflict     ConflictResolution
	NonInteractive bool
}

// CreateSource enumerates create origins per PRD §4.2.
type CreateSource string

const (
	CreateSourceBlank   CreateSource = "blank"
	CreateSourceDefault CreateSource = "default"
	CreateSourcePackage CreateSource = "package"
)

// EditAgentRequest captures partial-edit intent. Nil pointers mean
// "keep existing"; non-nil pointers replace the field.
type EditAgentRequest struct {
	Name           string
	Identity       *Identity
	Instructions   *Instructions
	Skills         *[]CapabilityRef
	MCP            *[]CapabilityRef
	Runtimes       *[]RuntimePref
	NonInteractive bool
}

// DeleteAgentRequest controls deletion behavior.
type DeleteAgentRequest struct {
	Name           string
	Confirm        bool // required in non-interactive mode
	NonInteractive bool
}

// DiscoverRequest scopes a capability discovery call.
type DiscoverRequest struct {
	Kinds    []CapabilityKind // empty = all
	Runtimes []string         // empty = all detected
}
