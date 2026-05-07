package cli

import (
	"github.com/spf13/cobra"

	"github.com/xz1220/agent-vm/internal/app/model"
)

func newPackageCmd(deps Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "package",
		Short: "Manage AVM packages (PRD §4.5)",
	}
	cmd.AddCommand(newPackageListCmd(deps))
	cmd.AddCommand(newPackageShowCmd(deps))
	cmd.AddCommand(newPackageInstallCmd(deps))
	cmd.AddCommand(newPackageUninstallCmd(deps))
	cmd.AddCommand(newPackageExportCmd(deps))
	cmd.AddCommand(newPackageInspectCmd(deps))
	return cmd
}

func newPackageListCmd(deps Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List installed packages",
		RunE: func(c *cobra.Command, args []string) error {
			pkgs, err := deps.Services.Packages.List(c.Context())
			if err != nil {
				return err
			}
			return RenderPackageList(c.OutOrStdout(), pkgs)
		},
	}
}

func newPackageShowCmd(deps Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "show <name>",
		Short: "Show an installed package",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			detail, err := deps.Services.Packages.Show(c.Context(), args[0])
			if err != nil {
				return err
			}
			return RenderPackageDetail(c.OutOrStdout(), detail)
		},
	}
}

func newPackageInstallCmd(deps Deps) *cobra.Command {
	var (
		resolution     string
		nonInteractive bool
	)
	cmd := &cobra.Command{
		Use:   "install <package-or-file>",
		Short: "Install a package",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			_, err := deps.Services.Packages.Install(c.Context(), model.InstallRequest{
				Source:         args[0],
				Resolution:     model.ConflictResolution(resolution),
				NonInteractive: nonInteractive,
			})
			return err
		},
	}
	cmd.Flags().StringVar(&resolution, "on-conflict", "", "rename|skip|overwrite|cancel")
	cmd.Flags().BoolVar(&nonInteractive, "non-interactive", false, "fail instead of prompting")
	return cmd
}

func newPackageUninstallCmd(deps Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall <name>",
		Short: "Uninstall a package",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return deps.Services.Packages.Uninstall(c.Context(), args[0])
		},
	}
}

func newPackageExportCmd(deps Deps) *cobra.Command {
	var (
		out        string
		withSkills bool
		withMCP    bool
	)
	cmd := &cobra.Command{
		Use:   "export <agent>",
		Short: "Export an Agent as a package",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			_, err := deps.Services.Packages.Export(c.Context(), model.ExportRequest{
				Agent:         args[0],
				IncludeSkills: withSkills,
				IncludeMCP:    withMCP,
				OutputPath:    out,
			})
			return err
		},
	}
	cmd.Flags().StringVarP(&out, "output", "o", "", "output file")
	cmd.Flags().BoolVar(&withSkills, "with-skills", true, "embed referenced skills")
	cmd.Flags().BoolVar(&withMCP, "with-mcp", true, "embed referenced MCP servers")
	return cmd
}

func newPackageInspectCmd(deps Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "inspect <file.avm.zip>",
		Short: "Inspect a package file without installing",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			detail, err := deps.Services.Packages.Inspect(c.Context(), args[0])
			if err != nil {
				return err
			}
			return RenderPackageDetail(c.OutOrStdout(), detail)
		},
	}
}
