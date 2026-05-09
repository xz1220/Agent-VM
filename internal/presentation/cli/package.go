package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/xz1220/agent-vm/internal/app/model"
	"github.com/xz1220/agent-vm/internal/app/service"
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
			g := globalFlags(c)
			pkgs, err := deps.Services.Packages.List(c.Context())
			if err != nil {
				return err
			}
			if g.JSON {
				return jsonWrite(c.OutOrStdout(), pkgs)
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
			g := globalFlags(c)
			detail, err := deps.Services.Packages.Show(c.Context(), args[0])
			if err != nil {
				if errors.Is(err, service.ErrPackageRegistryNotSupported) {
					fmt.Fprintln(c.OutOrStdout(), "(installed-package registry not yet implemented; use 'avm package inspect <file>')")
					return nil
				}
				return err
			}
			if g.JSON {
				return jsonWrite(c.OutOrStdout(), detail)
			}
			return RenderPackageDetail(c.OutOrStdout(), detail)
		},
	}
}

func newPackageInstallCmd(deps Deps) *cobra.Command {
	var resolution string
	cmd := &cobra.Command{
		Use:   "install <package-or-file>",
		Short: "Install a package",
		Long: `Install a package. Pass --on-conflict {rename|skip|overwrite|cancel}
to handle agents that already exist; the default returns AGENT_CONFLICT
so the caller decides explicitly.`,
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			g := globalFlags(c)
			req := model.InstallRequest{
				Source:     args[0],
				Resolution: model.ConflictResolution(resolution),
			}
			res, err := deps.Services.Packages.Install(c.Context(), req)
			if err != nil {
				return err
			}
			if g.JSON {
				return jsonWrite(c.OutOrStdout(), res)
			}
			return renderInstallResult(c.OutOrStdout(), res)
		},
	}
	cmd.Flags().StringVar(&resolution, "on-conflict", "", "rename|skip|overwrite|cancel")
	return cmd
}

func renderInstallResult(w io.Writer, r *model.InstallResult) error {
	if r == nil {
		return nil
	}
	fmt.Fprintf(w, "Installed package %q\n", r.Manifest.Name)
	if len(r.InstalledAgents) > 0 {
		fmt.Fprintf(w, "  agents:  %s\n", strings.Join(r.InstalledAgents, ", "))
	}
	if len(r.ImportedCaps) > 0 {
		ids := make([]string, 0, len(r.ImportedCaps))
		for _, id := range r.ImportedCaps {
			ids = append(ids, string(id))
		}
		fmt.Fprintf(w, "  caps:    %s\n", strings.Join(ids, ", "))
	}
	if len(r.Skipped) > 0 {
		fmt.Fprintf(w, "  skipped: %s\n", strings.Join(r.Skipped, ", "))
	}
	if len(r.Renamed) > 0 {
		fmt.Fprintln(w, "  renamed:")
		for old, n := range r.Renamed {
			fmt.Fprintf(w, "    %s -> %s\n", old, n)
		}
	}
	return nil
}

func newPackageUninstallCmd(deps Deps) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "uninstall <name>",
		Short: "Uninstall a package (deletes the imported Agent)",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			if !yes {
				return service.MissingInputError("yes",
					"package uninstall is destructive; pass --yes to confirm")
			}
			if err := deps.Services.Packages.Uninstall(c.Context(), args[0]); err != nil {
				return err
			}
			fmt.Fprintf(c.OutOrStdout(), "Uninstalled %q\n", args[0])
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "confirm uninstall (required)")
	return cmd
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
			res, err := deps.Services.Packages.Export(c.Context(), model.ExportRequest{
				Agent:         args[0],
				IncludeSkills: withSkills,
				IncludeMCP:    withMCP,
				OutputPath:    out,
			})
			if err != nil {
				return err
			}
			g := globalFlags(c)
			if g.JSON {
				return jsonWrite(c.OutOrStdout(), res)
			}
			fmt.Fprintf(c.OutOrStdout(), "Wrote %s\n", res.Path)
			return nil
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
			g := globalFlags(c)
			detail, err := deps.Services.Packages.Inspect(c.Context(), args[0])
			if err != nil {
				return err
			}
			if g.JSON {
				return jsonWrite(c.OutOrStdout(), detail)
			}
			return RenderPackageDetail(c.OutOrStdout(), detail)
		},
	}
}
