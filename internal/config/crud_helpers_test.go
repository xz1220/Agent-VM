package config

import (
	"os"
	"testing"
)

func TestDeleteHelpersRemoveConfigFiles(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	t.Setenv("HOME", home)

	if err := WriteAgent(&AgentProfile{Name: "delete-me", Runtime: RuntimePreferences{Preferred: "codex"}}, ScopeGlobal, project); err != nil {
		t.Fatalf("WriteAgent returned error: %v", err)
	}
	if err := DeleteAgent("delete-me", ScopeGlobal, project); err != nil {
		t.Fatalf("DeleteAgent returned error: %v", err)
	}
	if _, err := os.Stat(AgentPath("delete-me")); !os.IsNotExist(err) {
		t.Fatalf("agent file still exists or stat failed unexpectedly: %v", err)
	}

	if err := WriteEnvironment(&Environment{
		Name: "delete-env",
		RuntimeAgents: map[string]RuntimeAgent{
			"codex": {Primary: "delete-me"},
		},
		Targets: []string{"codex"},
	}); err != nil {
		t.Fatalf("WriteEnvironment returned error: %v", err)
	}
	if err := DeleteEnvironment("delete-env"); err != nil {
		t.Fatalf("DeleteEnvironment returned error: %v", err)
	}
	if _, err := os.Stat(EnvPath("delete-env")); !os.IsNotExist(err) {
		t.Fatalf("env file still exists or stat failed unexpectedly: %v", err)
	}

	if err := WritePortableMemory(&PortableMemory{ID: "delete-memory", Scope: string(ScopeProject), Path: "/tmp/delete-memory.md"}); err != nil {
		t.Fatalf("WritePortableMemory returned error: %v", err)
	}
	if err := DeletePortableMemory("delete-memory", ScopeProject); err != nil {
		t.Fatalf("DeletePortableMemory returned error: %v", err)
	}
	if _, err := os.Stat(MemoryPath("delete-memory", ScopeProject)); !os.IsNotExist(err) {
		t.Fatalf("memory file still exists or stat failed unexpectedly: %v", err)
	}
}

func TestMCPRegistryListWriteDelete(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	entry := &MCPRegistryEntry{
		Name:        "github",
		Description: "GitHub MCP",
		Server: MCPServerConfig{
			Type:    "stdio",
			Command: "github-mcp",
			Args:    []string{"--stdio"},
		},
	}
	if err := WriteMCPRegistryEntry(entry); err != nil {
		t.Fatalf("WriteMCPRegistryEntry returned error: %v", err)
	}

	summaries, err := ListMCPRegistryEntries()
	if err != nil {
		t.Fatalf("ListMCPRegistryEntries returned error: %v", err)
	}
	if len(summaries) != 1 || summaries[0].Name != "github" || summaries[0].Command != "github-mcp" {
		t.Fatalf("unexpected summaries: %#v", summaries)
	}

	got, _, err := ReadMCPRegistryEntry("github")
	if err != nil {
		t.Fatalf("ReadMCPRegistryEntry returned error: %v", err)
	}
	if got.Kind != "mcp" {
		t.Fatalf("kind = %q, want mcp", got.Kind)
	}

	if err := DeleteMCPRegistryEntry("github"); err != nil {
		t.Fatalf("DeleteMCPRegistryEntry returned error: %v", err)
	}
	if _, err := os.Stat(MCPRegistryPath("github")); !os.IsNotExist(err) {
		t.Fatalf("mcp file still exists or stat failed unexpectedly: %v", err)
	}
}
