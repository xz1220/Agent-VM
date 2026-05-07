package cli

import (
	"errors"

	"github.com/spf13/cobra"
)

func newInitCmd(deps Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize ~/.avm",
		RunE: func(c *cobra.Command, args []string) error {
			return errors.New("init: not yet implemented")
		},
	}
}

func newDoctorCmd(deps Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose AVM and runtime state",
		RunE: func(c *cobra.Command, args []string) error {
			rep, err := deps.Services.Diagnostics.Doctor(c.Context())
			if err != nil {
				return err
			}
			return RenderDoctor(c.OutOrStdout(), rep)
		},
	}
}

func newStatusCmd(deps Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "status [agent]",
		Short: "Show AVM status",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			agent := ""
			if len(args) > 0 {
				agent = args[0]
			}
			rep, err := deps.Services.Diagnostics.Status(c.Context(), agent)
			if err != nil {
				return err
			}
			return RenderStatus(c.OutOrStdout(), rep)
		},
	}
}

func newUninstallCmd(deps Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall AVM (binary / shell integration / data)",
		RunE: func(c *cobra.Command, args []string) error {
			return errors.New("uninstall: not yet implemented")
		},
	}
}

func newShellCmd(deps Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "shell",
		Short: "Manage AVM shell integration",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "install",
		Short: "Install shell integration",
		RunE: func(c *cobra.Command, args []string) error {
			return errors.New("shell install: not yet implemented")
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall shell integration",
		RunE: func(c *cobra.Command, args []string) error {
			return errors.New("shell uninstall: not yet implemented")
		},
	})
	return cmd
}
