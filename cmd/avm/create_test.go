package main

import (
	"strings"
	"testing"

	"github.com/xz1220/agent-vm/internal/config"
)

func TestCreateFromBuiltinPackageWithYesLazyInitializes(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	t.Setenv("HOME", home)
	chdir(t, project)

	out, err := executeCommand("create", "backend-coder", "--name", "api-coder", "--runtime", "codex", "--yes")
	if err != nil {
		t.Fatalf("create returned error: %v\n%s", err, out)
	}
	for _, want := range []string{
		"created agent api-coder from package backend-coder",
		`eval "$(avm activate api-coder)"`,
		"codex",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("create output missing %q:\n%s", want, out)
		}
	}

	if _, err := config.ReadGlobalConfig(); err != nil {
		t.Fatalf("create should lazy initialize global config: %v", err)
	}
	agent, err := config.ReadAgent("api-coder", config.ScopeGlobal, project)
	if err != nil {
		t.Fatalf("read created agent: %v", err)
	}
	if agent.Runtime.Preferred != "codex" {
		t.Fatalf("runtime = %q, want codex", agent.Runtime.Preferred)
	}
	if !containsCreateTestString(agent.Capabilities.Skills, "git") || !containsCreateTestString(agent.Capabilities.Skills, "test") {
		t.Fatalf("unexpected skills: %#v", agent.Capabilities.Skills)
	}
}

func TestCreateInteractiveDefaults(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	t.Setenv("HOME", home)
	chdir(t, project)

	cmd := newRootCommand()
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetIn(strings.NewReader("\n\n\n"))
	cmd.SetArgs([]string{"create", "backend-coder"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("interactive create returned error: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "Agent name [backend-coder]") {
		t.Fatalf("interactive output missing name prompt:\n%s", out.String())
	}
	if _, err := config.ReadAgent("backend-coder", config.ScopeGlobal, project); err != nil {
		t.Fatalf("read interactive agent: %v", err)
	}
}

func TestPackageListShow(t *testing.T) {
	listOut, err := executeCommand("package", "list")
	if err != nil {
		t.Fatalf("package list returned error: %v", err)
	}
	if !strings.Contains(listOut, "backend-coder") || !strings.Contains(listOut, "reviewer") {
		t.Fatalf("package list missing builtins:\n%s", listOut)
	}

	showOut, err := executeCommand("package", "show", "backend-coder")
	if err != nil {
		t.Fatalf("package show returned error: %v", err)
	}
	for _, want := range []string{"name: backend-coder", "modes:", "default_runtime: codex"} {
		if !strings.Contains(showOut, want) {
			t.Fatalf("package show missing %q:\n%s", want, showOut)
		}
	}
}

func containsCreateTestString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
