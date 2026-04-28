package main

import (
	"context"
	"os"

	"github.com/spf13/cobra"
	"github.com/xz1220/agent-vm/internal/config"
	avmruntime "github.com/xz1220/agent-vm/internal/runtime"
	avmsync "github.com/xz1220/agent-vm/internal/sync"
)

func newSyncCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync the current AVM active profile or environment",
		Args:  cobra.NoArgs,
		RunE:  runSync,
	}
	cmd.Flags().StringArray("target", nil, "sync only the named runtime target")
	return cmd
}

func runSync(cmd *cobra.Command, args []string) error {
	if err := ensureInitialized(); err != nil {
		return err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	cfg, err := config.ReadGlobalConfig()
	if err != nil {
		return err
	}
	resolved, err := resolveActivationRef(cfg.Active, cwd)
	if err != nil {
		return err
	}

	targets, err := cmd.Flags().GetStringArray("target")
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		targets = append([]string(nil), resolved.Targets...)
	}

	syncer := avmsync.NewSyncer(avmruntime.NewRegistry())
	syncResult, syncErr := syncer.SyncActivation(context.Background(), resolved, avmsync.Options{
		ProjectRoot:  cwd,
		ActiveDir:    config.ActiveDir(),
		UpdateActive: false,
		Targets:      targets,
	})
	if syncResult == nil {
		return syncErr
	}

	currentErr := writeCurrentActive(syncResult.Active)
	result := activationResultFromSync(syncResult)
	if syncErr != nil {
		result.SyncStatus = "failed"
	}
	printActivationResult(cmd.OutOrStdout(), result)
	if currentErr != nil {
		return currentErr
	}
	return syncErr
}
