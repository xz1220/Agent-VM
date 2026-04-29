package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/xz1220/agent-vm/internal/packageio"
)

func newImportCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "import <file.avm.zip>",
		Short: "Import an AVM package",
		Args:  cobra.ExactArgs(1),
		RunE:  runImport,
	}
}

func runImport(cmd *cobra.Command, args []string) error {
	result, err := installPackageFromPath(args[0], false)
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "imported %s %s: added %d, skipped %d\n", result.Manifest.Kind, result.Manifest.Name, len(result.Added), len(result.Skipped))
	return nil
}

func installPackageFromPath(packagePath string, dryRun bool) (*packageio.ImportResult, error) {
	if !dryRun {
		if err := ensureInitialized(); err != nil {
			return nil, err
		}
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	result, err := packageio.ImportPackage(packageio.ImportOptions{
		PackagePath: packagePath,
		CWD:         cwd,
		DryRun:      dryRun,
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
