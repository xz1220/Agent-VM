// Package cli is the AVM presentation layer. It owns command and flag
// parsing, interactive UX, and rendering of structured results from
// the application services. It does not own product rules.
package cli

import (
	"github.com/spf13/cobra"

	"github.com/xz1220/agent-vm/internal/app/service"
)

// Deps is the wiring presentation needs from the composition root.
type Deps struct {
	Services service.Container
}

// NewRoot builds the cobra tree for `avm`.
func NewRoot(deps Deps) *cobra.Command {
	root := &cobra.Command{
		Use:           "avm",
		Short:         "Agent VM — local config manager for AI coding agents",
		SilenceUsage:  true,
		SilenceErrors: false,
	}

	root.AddCommand(newAgentCmd(deps))
	root.AddCommand(newRunCmd(deps))
	root.AddCommand(newPackageCmd(deps))
	root.AddCommand(newInitCmd(deps))
	root.AddCommand(newDoctorCmd(deps))
	root.AddCommand(newStatusCmd(deps))
	root.AddCommand(newUninstallCmd(deps))
	root.AddCommand(newShellCmd(deps))

	return root
}
