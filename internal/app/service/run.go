package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/xz1220/agent-vm/internal/app/model"
	"github.com/xz1220/agent-vm/internal/infra/agentstore"
	"github.com/xz1220/agent-vm/internal/infra/managedfile"
	"github.com/xz1220/agent-vm/internal/infra/process"
	"github.com/xz1220/agent-vm/internal/infra/runlog"
	"github.com/xz1220/agent-vm/internal/runtime"
)

// RunService implements PRD §4.4 run flow:
//
//	resolve runtime → render plan → drift check → write managed config
//	→ spawn runtime → record run log → return result/preview.
type RunService interface {
	Preview(ctx context.Context, req model.RunRequest) (*model.RunPreview, error)
	Run(ctx context.Context, req model.RunRequest) (*model.RunResult, error)
}

// Runner is the default RunService.
type Runner struct {
	Repo     agentstore.Repository
	Runtimes runtime.Registry
	Writer   managedfile.Writer
	Process  process.Runner
	Log      runlog.Log
}

func NewRunner(
	repo agentstore.Repository,
	registry runtime.Registry,
	writer managedfile.Writer,
	proc process.Runner,
	log runlog.Log,
) *Runner {
	return &Runner{Repo: repo, Runtimes: registry, Writer: writer, Process: proc, Log: log}
}

// resolveRuntime implements PRD §2.2: explicit override wins; single
// preference auto-resolves; multiple preferences require explicit
// selection. Callers must supply --runtime when ambiguous.
func (s *Runner) resolveRuntime(req model.RunRequest, agent *model.Agent) (string, error) {
	if req.Runtime != "" {
		if _, err := s.Runtimes.Resolve(req.Runtime); err != nil {
			return "", WrapError(CodeRuntimeNotFound, err,
				fmt.Sprintf("runtime %q not registered", req.Runtime),
				map[string]any{"runtime": req.Runtime})
		}
		return req.Runtime, nil
	}
	if len(agent.Runtimes) == 0 {
		return "", NewError(CodeRuntimeMissing,
			fmt.Sprintf("agent %q has no runtimes configured; pass --runtime", agent.Identity.Name),
			map[string]any{"agent": agent.Identity.Name})
	}
	if len(agent.Runtimes) == 1 {
		return agent.Runtimes[0].Runtime, nil
	}
	// Multiple. Prefer the explicit Default if exactly one is marked.
	var def string
	count := 0
	for _, r := range agent.Runtimes {
		if r.Default {
			def = r.Runtime
			count++
		}
	}
	if count == 1 {
		return def, nil
	}
	// Otherwise the caller (UI/CLI) must choose.
	options := make([]string, 0, len(agent.Runtimes))
	for _, r := range agent.Runtimes {
		options = append(options, r.Runtime)
	}
	return "", NewError(CodeRuntimeAmbiguous,
		fmt.Sprintf("agent %q has multiple runtimes; pass --runtime", agent.Identity.Name),
		map[string]any{"agent": agent.Identity.Name, "runtimes": options})
}

// loadPlan loads agent + driver and produces a Plan and Boundary.
func (s *Runner) loadPlan(ctx context.Context, req model.RunRequest) (*model.Agent, runtime.Driver, *runtime.Plan, runtime.Boundary, string, error) {
	if s.Repo == nil || s.Runtimes == nil {
		return nil, nil, nil, runtime.Boundary{}, "", errors.New("runner: missing dependencies")
	}
	agent, err := s.Repo.Get(req.Agent)
	if err != nil {
		if errors.Is(err, agentstore.ErrNotFound) {
			return nil, nil, nil, runtime.Boundary{}, "", AgentNotFoundError(req.Agent, err)
		}
		return nil, nil, nil, runtime.Boundary{}, "", WrapError(CodeIOFailure, err,
			"load agent: "+err.Error(), map[string]any{"agent": req.Agent})
	}
	rtName, err := s.resolveRuntime(req, agent)
	if err != nil {
		return agent, nil, nil, runtime.Boundary{}, "", err
	}
	drv, err := s.Runtimes.Resolve(rtName)
	if err != nil {
		return agent, nil, nil, runtime.Boundary{}, rtName, WrapError(CodeRuntimeNotFound, err,
			fmt.Sprintf("resolve runtime %q: %v", rtName, err),
			map[string]any{"runtime": rtName})
	}
	plan, err := drv.Plan(ctx, agent)
	if err != nil {
		return agent, drv, nil, runtime.Boundary{}, rtName, WrapError(CodeRuntimePlanFailure, err,
			fmt.Sprintf("runtime %q failed to render plan: %v", rtName, err),
			map[string]any{"runtime": rtName, "agent": agent.Identity.Name})
	}
	bnd, err := drv.Boundary(ctx, agent)
	if err != nil {
		return agent, drv, plan, runtime.Boundary{}, rtName, WrapError(CodeRuntimePlanFailure, err,
			fmt.Sprintf("runtime %q boundary failure: %v", rtName, err),
			map[string]any{"runtime": rtName, "agent": agent.Identity.Name})
	}
	return agent, drv, plan, bnd, rtName, nil
}

// buildPreview constructs a RunPreview without writing anything.
func buildPreview(agent *model.Agent, rtName string, plan *runtime.Plan, bnd runtime.Boundary, drift []model.DiffEntry) model.RunPreview {
	preview := model.RunPreview{
		Agent:   agent.Identity.Name,
		Runtime: rtName,
		Boundary: model.BoundarySummary{
			StateDir: bnd.StateDir,
			EnvKeys:  envKeys(bnd.Env),
		},
		Drift: drift,
	}
	for _, f := range plan.Files {
		preview.WritePaths = append(preview.WritePaths, f.Path)
	}
	for _, m := range plan.Mappings {
		preview.Mapping = append(preview.Mapping, model.FieldMappingSummary{
			Field:  m.Field,
			Status: m.Status,
			Note:   m.Note,
		})
	}
	preview.Warnings = append(preview.Warnings, plan.Warnings...)
	return preview
}

func envKeys(env map[string]string) []string {
	if len(env) == 0 {
		return nil
	}
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	// Stable order for deterministic rendering.
	sortStrings(keys)
	return keys
}

func sortStrings(s []string) {
	// Tiny insertion sort to avoid pulling sort import noise; small N.
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}

// Preview renders what `avm run --preview` would do without any writes.
func (s *Runner) Preview(ctx context.Context, req model.RunRequest) (*model.RunPreview, error) {
	agent, _, plan, bnd, rtName, err := s.loadPlan(ctx, req)
	if err != nil {
		return nil, err
	}
	var drift []model.DiffEntry
	if s.Writer != nil && len(plan.Files) > 0 {
		d, derr := s.Writer.DryRun(ctx, plan.Files)
		if derr != nil {
			return nil, WrapError(CodeIOFailure, derr,
				"managed-file dryrun: "+derr.Error(),
				map[string]any{"runtime": rtName, "agent": agent.Identity.Name})
		}
		drift = d
	}
	p := buildPreview(agent, rtName, plan, bnd, drift)
	return &p, nil
}

// applyDriftPolicy decides whether to proceed with apply given a drift
// list and a user-selected policy.
//
// Behavior (post-rewrite — pure plumbing, no interactive default):
//   - empty drift                      → proceed
//   - DriftAsk + drift > 0             → return CodeDriftDetected (caller must
//     re-issue with explicit --drift)
//   - DriftKeep / DriftMerge / Discard → proceed; managedfile.Apply overwrites
//
// Merge-back-into-Agent semantics is a TODO; today merge and discard
// both delegate to managedfile.Apply (which overwrites).
func applyDriftPolicy(drift []model.DiffEntry, req model.RunRequest, agentName, rtName string) error {
	if len(drift) == 0 {
		return nil
	}
	switch req.DriftPolicy {
	case model.DriftAsk:
		return NewError(CodeDriftDetected,
			fmt.Sprintf("drift detected for agent %q on runtime %q; pass --drift", agentName, rtName),
			map[string]any{
				"agent":   agentName,
				"runtime": rtName,
				"entries": drift,
			})
	case model.DriftKeep, model.DriftMerge, model.DriftDiscard:
		return nil
	default:
		return NewError(CodeValidation,
			fmt.Sprintf("unknown drift policy %q", req.DriftPolicy),
			map[string]any{"drift_policy": string(req.DriftPolicy)})
	}
}

// Run executes the full launch flow.
func (s *Runner) Run(ctx context.Context, req model.RunRequest) (*model.RunResult, error) {
	if s.Process == nil {
		return nil, errors.New("runner: missing process runner")
	}
	agent, drv, plan, bnd, rtName, err := s.loadPlan(ctx, req)
	if err != nil {
		return nil, err
	}

	// Drift gate (uses dryrun first so a buggy policy does not silently overwrite).
	var drift []model.DiffEntry
	if s.Writer != nil && len(plan.Files) > 0 {
		drift, err = s.Writer.DryRun(ctx, plan.Files)
		if err != nil {
			return nil, WrapError(CodeIOFailure, err,
				"managed-file dryrun: "+err.Error(),
				map[string]any{"runtime": rtName, "agent": agent.Identity.Name})
		}
		if err := applyDriftPolicy(drift, req, agent.Identity.Name, rtName); err != nil {
			return nil, err
		}
	}

	// Apply the managed files.
	if s.Writer != nil && len(plan.Files) > 0 {
		if _, err := s.Writer.Apply(ctx, plan.Files); err != nil {
			return nil, WrapError(CodeIOFailure, err,
				"managed-file apply: "+err.Error(),
				map[string]any{"runtime": rtName, "agent": agent.Identity.Name})
		}
	}

	spec, err := drv.LaunchSpec(ctx, agent, plan)
	if err != nil {
		return nil, WrapError(CodeRuntimePlanFailure, err,
			fmt.Sprintf("runtime %q launch spec: %v", rtName, err),
			map[string]any{"runtime": rtName, "agent": agent.Identity.Name})
	}

	preview := buildPreview(agent, rtName, plan, bnd, drift)

	startedAt := time.Now()
	res, runErr := s.Process.Run(ctx, spec)
	endedAt := time.Now()

	exitCode := res.ExitCode
	if runErr != nil && exitCode == 0 {
		exitCode = -1
	}

	if s.Log != nil {
		_ = s.Log.Append(model.RunRecord{
			Agent:     agent.Identity.Name,
			Runtime:   rtName,
			StartedAt: startedAt,
			EndedAt:   endedAt,
			ExitCode:  exitCode,
			Drift:     drift,
			Warnings:  plan.Warnings,
		})
	}

	if runErr != nil {
		return nil, WrapError(CodeRuntimeBinaryMissing, runErr,
			"runtime process failed: "+runErr.Error(),
			map[string]any{"runtime": rtName, "agent": agent.Identity.Name})
	}
	return &model.RunResult{
		Preview:   preview,
		StartedAt: startedAt,
		EndedAt:   endedAt,
		ExitCode:  exitCode,
	}, nil
}
