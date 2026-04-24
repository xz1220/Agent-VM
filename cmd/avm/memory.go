package main

import "github.com/spf13/cobra"

func newMemoryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "memory",
		Short: "Manage AVM portable memory",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newMemoryImportCommand())
	return cmd
}

func newMemoryImportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import portable memory candidates",
		Args:  cobra.NoArgs,
		RunE:  notImplemented,
	}
	cmd.Flags().String("from", "", "memory source path or runtime")
	cmd.Flags().Bool("dry-run", false, "preview memory import without writing")
	cmd.Flags().String("format", "text", "output format")
	return cmd
}
