package service

import (
	"errors"

	"github.com/xz1220/agent-vm/internal/infra/agentstore"
)

// Error is the typed error every service method should return for
// failures the presentation/UI layer is expected to inspect.
//
// Code is the stable, machine-readable identifier (see the Code*
// constants). Message is a human-readable string. Details carries
// structured context per Code (e.g. {"name": "alpha"} for AGENT_CONFLICT).
//
// The Code/Message/Details shape is the public contract surfaced to
// JSON output and external consumers (TS/UI). It is part of the AVM
// CLI protocol — changing field names or removing codes is a breaking
// change.
type Error struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`

	// cause is unwrappable for errors.Is/As compatibility but is
	// intentionally not exported and not serialised.
	cause error
}

// Error returns the human message; satisfies the error interface.
func (e *Error) Error() string { return e.Message }

// Unwrap returns the underlying cause so existing errors.Is/As checks
// against agentstore.ErrConflict, agentstore.ErrNotFound, etc. keep
// working through a service-layer wrap.
func (e *Error) Unwrap() error { return e.cause }

// NewError constructs a typed error with no underlying cause.
func NewError(code, message string, details map[string]any) *Error {
	return &Error{Code: code, Message: message, Details: details}
}

// WrapError constructs a typed error that wraps cause; errors.Is on
// the result will also match cause.
func WrapError(code string, cause error, message string, details map[string]any) *Error {
	return &Error{Code: code, Message: message, Details: details, cause: cause}
}

// AsError extracts a *service.Error from err if any link in the chain
// is one. Otherwise returns nil.
func AsError(err error) *Error {
	if err == nil {
		return nil
	}
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	return nil
}

// Stable error codes. These values are part of the public CLI/JSON
// protocol — see docs/api/cli-protocol.md.
const (
	// Agent CRUD.
	CodeAgentConflict    = "AGENT_CONFLICT"     // Save would overwrite an existing Agent without explicit OnConflict.
	CodeAgentNotFound    = "AGENT_NOT_FOUND"    // Named Agent does not exist.
	CodeAgentInvalidName = "AGENT_INVALID_NAME" // Name fails validation regex.

	// Runtime resolution & launch.
	CodeRuntimeNotFound      = "RUNTIME_NOT_FOUND"      // Named runtime not registered.
	CodeRuntimeAmbiguous     = "RUNTIME_AMBIGUOUS"      // Agent has multiple runtimes; --runtime not given.
	CodeRuntimeMissing       = "RUNTIME_MISSING"        // Agent has no configured runtimes at all.
	CodeRuntimeBinaryMissing = "RUNTIME_BINARY_MISSING" // Driver reports binary not available.
	CodeRuntimePlanFailure   = "RUNTIME_PLAN_FAILURE"   // Driver.Plan returned an error.
	CodeDriftDetected        = "DRIFT_DETECTED"         // Managed config drift detected; --drift not given.

	// Package install / export.
	CodePackageNotFound        = "PACKAGE_NOT_FOUND"
	CodePackageInvalidManifest = "PACKAGE_INVALID_MANIFEST"
	CodePackageChecksum        = "PACKAGE_CHECKSUM_MISMATCH"

	// Capabilities.
	CodeCapabilityNotFound = "CAPABILITY_NOT_FOUND"
	CodeCapabilityConflict = "CAPABILITY_CONFLICT" // Same (kind, name) under multiple sources.

	// Generic.
	CodeMissingInput = "MISSING_INPUT"  // Required field absent in non-interactive call.
	CodeValidation   = "VALIDATION"     // Generic validation failure.
	CodeIOFailure    = "IO_FAILURE"     // Underlying infra IO error.
	CodeInternal     = "INTERNAL_ERROR" // Catch-all when nothing better fits.
)

// Sentinel errors retained for backward compatibility with code that
// still uses errors.Is(err, service.ErrAgentXxx). New code should use
// the Code* constants on *Error instead.
//
// These alias the underlying agentstore sentinels so service helpers
// that wrap with WrapError(CodeAgentConflict, agentstore.ErrConflict, ...)
// continue to satisfy both legacy and new checks.
var (
	ErrAgentConflict = agentstore.ErrConflict
	ErrAgentNotFound = agentstore.ErrNotFound
)

// ============================================================================
// Convenience constructors for common error patterns.
//
// These exist to make call sites read clearly and to keep the Code/
// Details payload consistent across the service. Add a new constructor
// here only when 3+ call sites would benefit; for one-off errors,
// inline NewError/WrapError is fine.
// ============================================================================

// AgentConflictError wraps agentstore.ErrConflict with details {"name": name}.
// `errors.Is(err, agentstore.ErrConflict)` and `errors.Is(err, ErrAgentConflict)`
// still match the returned error.
func AgentConflictError(name, message string) *Error {
	return WrapError(CodeAgentConflict, agentstore.ErrConflict,
		message,
		map[string]any{"name": name},
	)
}

// AgentNotFoundError wraps a not-found error from infra with details
// {"name": name}. Pass cause = nil if there is no underlying error.
func AgentNotFoundError(name string, cause error) *Error {
	if cause == nil {
		cause = agentstore.ErrNotFound
	}
	return WrapError(CodeAgentNotFound, cause,
		"agent \""+name+"\" not found",
		map[string]any{"name": name},
	)
}

// MissingInputError signals that a required field was absent.
// Hint is a human-readable instruction (e.g. "pass --runtime").
func MissingInputError(field, hint string) *Error {
	return NewError(CodeMissingInput,
		"missing required input: "+field+"; "+hint,
		map[string]any{"field": field, "hint": hint},
	)
}
