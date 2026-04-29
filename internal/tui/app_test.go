package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xz1220/agent-vm/internal/config"
)

func TestModelSwitchesTabs(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	project := t.TempDir()

	model, err := NewModel(Options{CWD: project})
	if err != nil {
		t.Fatalf("NewModel returned error: %v", err)
	}
	if model.tab != tabStatus {
		t.Fatalf("initial tab = %v, want status", model.tab)
	}

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyRight})
	updated := next.(Model)
	if updated.tab != tabAgents {
		t.Fatalf("tab after right = %v, want agents", updated.tab)
	}

	next, _ = updated.Update(tea.KeyMsg{Type: tea.KeyLeft})
	updated = next.(Model)
	if updated.tab != tabStatus {
		t.Fatalf("tab after left = %v, want status", updated.tab)
	}
}

func TestAgentFormSavesProfile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	project := t.TempDir()

	model, err := NewModel(Options{CWD: project})
	if err != nil {
		t.Fatalf("NewModel returned error: %v", err)
	}
	model.tab = tabAgents
	model.startNew()
	setFormValue(&model.form, "name", "api-coder")
	setFormValue(&model.form, "scope", "global")
	setFormValue(&model.form, "runtime", "codex")
	setFormValue(&model.form, "skills", "git,test")
	setFormValue(&model.form, "mcps", "github")
	setFormValue(&model.form, "memory_refs", "standards:project:/memory/standards.md:read")

	if err := model.saveForm(); err != nil {
		t.Fatalf("saveForm returned error: %v", err)
	}

	agent, err := config.ReadAgent("api-coder", config.ScopeGlobal, project)
	if err != nil {
		t.Fatalf("ReadAgent returned error: %v", err)
	}
	if agent.Runtime.Preferred != "codex" {
		t.Fatalf("runtime = %q, want codex", agent.Runtime.Preferred)
	}
	if got := join(agent.Capabilities.Skills); got != "git,test" {
		t.Fatalf("skills = %q, want git,test", got)
	}
	if len(agent.MemoryRefs) != 1 || agent.MemoryRefs[0].ID != "standards" {
		t.Fatalf("memory refs = %#v, want standards ref", agent.MemoryRefs)
	}
}

func TestRuntimeCandidateCreateCallback(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	project := t.TempDir()
	var gotRef string

	model, err := NewModel(Options{
		CWD: project,
		LoadRuntimes: func(cwd string) ([]RuntimeRow, string, error) {
			return []RuntimeRow{
				{
					Runtime: "claude-code",
					Found:   true,
					Candidates: []RuntimeCandidate{
						{Runtime: "claude-code", Name: "reviewer", Description: "Review code"},
					},
				},
			}, "/tmp/report.json", nil
		},
		CreateImportAgent: func(ref string) (string, error) {
			gotRef = ref
			return "reviewer", nil
		},
	})
	if err != nil {
		t.Fatalf("NewModel returned error: %v", err)
	}
	model.tab = tabRuntimes
	model.selected[tabRuntimes] = 1
	model.startNew()

	if gotRef != "claude-code/reviewer" {
		t.Fatalf("create callback ref = %q, want claude-code/reviewer", gotRef)
	}
	if model.message != "created agent reviewer" {
		t.Fatalf("message = %q, want created agent reviewer", model.message)
	}
}

func join(values []string) string {
	if len(values) == 0 {
		return ""
	}
	out := values[0]
	for _, value := range values[1:] {
		out += "," + value
	}
	return out
}
