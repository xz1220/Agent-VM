package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newPackageCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "package",
		Short: "Discover AVM packages",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newPackageListCommand(), newPackageShowCommand())
	return cmd
}

func newPackageListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available AVM packages",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			fmt.Fprintln(out, "NAME\tMODES\tDEFAULT RUNTIME\tDESCRIPTION")
			for _, pkg := range listBuiltinPackages() {
				fmt.Fprintf(out, "%s\t%s\t%s\t%s\n", pkg.Name, stringsJoin(pkg.Modes, ","), pkg.DefaultRuntime, pkg.Description)
			}
			return nil
		},
	}
}

func newPackageShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show <package>",
		Short: "Show an AVM package",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pkg, ok := lookupBuiltinPackage(args[0])
			if !ok {
				return fmt.Errorf("package %q not found; run avm package list", args[0])
			}
			return encodeYAML(cmd, pkg)
		},
	}
}

func stringsJoin(values []string, sep string) string {
	if len(values) == 0 {
		return ""
	}
	out := values[0]
	for _, value := range values[1:] {
		out += sep + value
	}
	return out
}
