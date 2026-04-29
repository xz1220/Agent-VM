package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/xz1220/agent-vm/internal/packageio"
)

func newInstallCommand() *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "install <file.avm.zip>",
		Short: "Install an AVM package",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstall(cmd, args, dryRun)
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview package install without writing files")
	return cmd
}

func runInstall(cmd *cobra.Command, args []string, dryRun bool) error {
	result, err := installPackageFromPath(args[0], dryRun)
	if err != nil {
		return err
	}
	if dryRun {
		printInstallDryRun(cmd, result)
		return nil
	}
	fmt.Fprintf(cmd.OutOrStdout(), "installed %s %s: added %d, skipped %d\n", result.Manifest.Kind, result.Manifest.Name, len(result.Added), len(result.Skipped))
	return nil
}

func printInstallDryRun(cmd *cobra.Command, result *packageio.ImportResult) {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "install plan for %s %s: add %d, skip %d, conflict %d\n", result.Manifest.Kind, result.Manifest.Name, len(result.Added), len(result.Skipped), len(result.Conflicts))
	printImportActions(out, "would add", result.Added)
	printImportActions(out, "would skip", result.Skipped)
	printImportActions(out, "conflicts", result.Conflicts)
}

func printImportActions(out io.Writer, label string, actions []packageio.ImportAction) {
	if len(actions) == 0 {
		return
	}
	fmt.Fprintf(out, "%s:\n", label)
	for _, action := range actions {
		fmt.Fprintf(out, "  %s -> %s\n", action.PackagePath, action.TargetPath)
	}
}
