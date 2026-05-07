package service

import "github.com/xz1220/agent-vm/internal/infra/agentstore"

// Service-level error sentinels. Presentation layer must switch on
// these (via errors.Is) instead of importing infra packages directly.
//
// Implementation note: these alias the underlying agentstore sentinels
// so existing service wrapping (`fmt.Errorf("...: %w", agentstore.ErrXxx)`)
// keeps satisfying both names. The aliasing is deliberate — it lets us
// replace the infra source later without breaking presentation callers.
var (
	ErrAgentConflict = agentstore.ErrConflict
	ErrAgentNotFound = agentstore.ErrNotFound
)
