package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newInstallCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "install <file.avm.zip>",
		Short: "Install an AVM package",
		Args:  cobra.ExactArgs(1),
		RunE:  runInstall,
	}
}

func runInstall(cmd *cobra.Command, args []string) error {
	result, err := installPackageFromPath(args[0])
	if err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "installed %s %s: added %d, skipped %d\n", result.Manifest.Kind, result.Manifest.Name, len(result.Added), len(result.Skipped))
	return nil
}
