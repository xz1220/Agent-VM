package model

// CreateAgentRequest captures the user's intent to create an Agent.
// Source describes blank/default vs existing-package origin (see PRD §4.2).
type CreateAgentRequest struct {
	Name         string             `json:"name"`
	Description  string             `json:"description,omitempty"`
	Role         string             `json:"role,omitempty"`
	Instructions Instructions       `json:"instructions,omitempty"`
	Skills       []CapabilityRef    `json:"skills,omitempty"`
	MCP          []CapabilityRef    `json:"mcp,omitempty"`
	Runtimes     []RuntimePref      `json:"runtimes,omitempty"`
	Source       CreateSource       `json:"source,omitempty"`
	OnConflict   ConflictResolution `json:"on_conflict,omitempty"`
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
	Name         string           `json:"name"`
	Identity     *Identity        `json:"identity,omitempty"`
	Instructions *Instructions    `json:"instructions,omitempty"`
	Skills       *[]CapabilityRef `json:"skills,omitempty"`
	MCP          *[]CapabilityRef `json:"mcp,omitempty"`
	Runtimes     *[]RuntimePref   `json:"runtimes,omitempty"`
}

// DeleteAgentRequest controls deletion behavior.
type DeleteAgentRequest struct {
	Name    string `json:"name"`
	Confirm bool   `json:"confirm,omitempty"`
}

// DiscoverRequest scopes a capability discovery call.
type DiscoverRequest struct {
	Kinds    []CapabilityKind `json:"kinds,omitempty"`    // empty = all
	Runtimes []string         `json:"runtimes,omitempty"` // empty = all detected
}
