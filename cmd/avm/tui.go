package main

import (
	"bytes"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/xz1220/agent-vm/internal/config"
	"github.com/xz1220/agent-vm/internal/memory"
	"github.com/xz1220/agent-vm/internal/tui"
)

func newTUICommand() *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Open the full-screen AVM console",
		Args:  cobra.NoArgs,
		RunE:  runTUI,
	}
}

func tuiOptions(cmd *cobra.Command) (tui.Options, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return tui.Options{}, err
	}
	return tui.Options{
		CWD:                cwd,
		Input:              cmd.InOrStdin(),
		Output:             cmd.OutOrStdout(),
		Activate:           tuiActivate(cwd),
		LoadRuntimes:       tuiLoadRuntimes,
		ScanRuntimes:       refreshInitImportReport,
		CreateImportAgent:  tuiCreateImportAgent(cwd),
		ImportMemoryDryRun: tuiMemoryImportDryRun,
	}, nil
}

func tuiActivate(cwd string) tui.ActivateFunc {
	return func(kind, name string) error {
		resolved, err := resolveActivationRef(config.ActiveRef{Kind: kind, Name: name}, cwd)
		if err != nil {
			return err
		}
		_, err = applyActivation(resolved, cwd)
		return err
	}
}

func tuiLoadRuntimes(cwd string) ([]tui.RuntimeRow, string, error) {
	if _, err := os.Stat(initImportReportPath()); os.IsNotExist(err) {
		if err := refreshInitImportReport(); err != nil {
			return nil, initImportReportPath(), err
		}
	} else if err != nil {
		return nil, initImportReportPath(), err
	}
	report, err := readCreateImportReport()
	if err != nil {
		return nil, initImportReportPath(), err
	}
	rows := make([]tui.RuntimeRow, 0, len(report.Runtimes))
	for _, runtimeReport := range report.Runtimes {
		row := tui.RuntimeRow{
			Runtime:   runtimeReport.Runtime,
			Found:     runtimeReport.Found,
			ConfigDir: runtimeReport.ConfigDir,
			Warnings:  append([]string(nil), runtimeReport.Warnings...),
		}
		for _, candidate := range runtimeReport.AgentCandidates {
			if candidate.Name == "" {
				continue
			}
			row.Candidates = append(row.Candidates, tui.RuntimeCandidate{
				Runtime:     runtimeReport.Runtime,
				Name:        candidate.Name,
				Description: candidate.Description,
			})
		}
		rows = append(rows, row)
	}
	return rows, initImportReportPath(), nil
}

func tuiCreateImportAgent(cwd string) tui.ImportAgentCreator {
	return func(ref string) (string, error) {
		source, err := createSourceFromImport(ref)
		if err != nil {
			return "", err
		}
		values := defaultCreateValues(source, createOptions{Scope: config.ScopeGlobal}, cwd)
		if _, err := config.ReadAgent(values.Name, values.Scope, cwd); err == nil {
			return "", fmt.Errorf("agent %q already exists", values.Name)
		} else if err != nil && !os.IsNotExist(err) {
			return "", err
		}
		agent := agentFromCreateSource(source, values)
		if err := config.WriteAgent(agent, values.Scope, cwd); err != nil {
			return "", err
		}
		return agent.Name, nil
	}
}

func tuiMemoryImportDryRun(source string) (string, error) {
	plan, err := memory.ImportDryRun(memory.ImportOptions{
		Source: source,
		DryRun: true,
	})
	if err != nil {
		return "", err
	}
	var out bytes.Buffer
	if err := memory.WriteTextReport(&out, plan); err != nil {
		return "", err
	}
	return out.String(), nil
}
