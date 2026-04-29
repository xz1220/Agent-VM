package main

import (
	"github.com/spf13/cobra"
	"github.com/xz1220/agent-vm/internal/tui"
	"github.com/xz1220/agent-vm/internal/version"
)

func Execute() error {
	return newRootCommand().Execute()
}

func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "avm",
		Short:         "Manage portable AI agent profiles across runtimes",
		Long:          "Agent VM (AVM) manages portable AI agent profiles, capabilities, memory refs, and runtime render plans.",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version.String(),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !isTerminalFile(cmd.InOrStdin()) || !isTerminalFile(cmd.OutOrStdout()) {
				return cmd.Help()
			}
			return runTUI(cmd, args)
		},
	}
	cmd.SetVersionTemplate("avm {{.Version}}\n")
	addCommands(cmd)
	return cmd
}

func runTUI(cmd *cobra.Command, args []string) error {
	if err := ensureInitialized(); err != nil {
		return err
	}
	opts, err := tuiOptions(cmd)
	if err != nil {
		return err
	}
	return tui.Run(opts)
}
