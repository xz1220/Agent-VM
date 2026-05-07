package model

// CapabilityKind enumerates the kinds of capabilities AVM understands.
// Future kinds (commands, hooks, toolsets) are added here.
type CapabilityKind string

const (
	CapabilityKindSkill CapabilityKind = "skill"
	CapabilityKindMCP   CapabilityKind = "mcp"
)

// CapabilityID is the stable identity of a capability inside the AVM
// capability store. Agents reference capabilities by ID; package names,
// import sources, and checksums are audit metadata, not identity.
type CapabilityID string

// CapabilitySource describes where a discovered capability came from.
// AVM must show this to users at create/edit time so users can tell
// AVM-managed capabilities apart from runtime global discoveries.
type CapabilitySource string

const (
	SourceAVM           CapabilitySource = "avm"     // managed by AVM capability store
	SourcePackage       CapabilitySource = "package" // installed via AVM package
	SourceRuntimeGlobal CapabilitySource = "runtime" // discovered in runtime global dir
)

// CapabilityRecord is an entry in the AVM capability store.
type CapabilityRecord struct {
	ID         CapabilityID
	Kind       CapabilityKind
	Name       string
	Version    string
	Source     CapabilitySource
	Checksum   string
	ImportFrom string // optional audit: package name or runtime path
}

// GlobalCapability is what a runtime reports during global discovery.
// It does not enter Agent references directly — users must explicitly
// import it into the AVM capability store first.
type GlobalCapability struct {
	Runtime string
	Kind    CapabilityKind
	Name    string
	Path    string
	Version string
}

// CapabilityCandidate is what AVM presents to users at create/edit time.
// It unifies AVM-managed records with runtime global discoveries and
// must always carry a Source so the UI can explain provenance.
type CapabilityCandidate struct {
	Kind     CapabilityKind
	Name     string
	Source   CapabilitySource
	Record   *CapabilityRecord // set when Source != SourceRuntimeGlobal
	Global   *GlobalCapability // set when Source == SourceRuntimeGlobal
	Conflict bool              // true when name collides across sources
}
