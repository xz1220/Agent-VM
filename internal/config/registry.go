package config

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func SkillRegistryPath(name string) string {
	return filepath.Join(RegistryKindDir("skills"), name)
}

func SkillRegistryFilePath(name string) string {
	return filepath.Join(SkillRegistryPath(name), "SKILL.md")
}

func MCPRegistryPath(name string) string {
	return filepath.Join(RegistryKindDir("mcps"), name+".yaml")
}

func resolvedSkill(name string) (ResolvedSkill, string, error) {
	if !validName(name) {
		return ResolvedSkill{Name: name}, "", fieldError("", "name", "invalid name %q", name)
	}

	sourceDir := SkillRegistryPath(name)
	sourcePath := filepath.Join(sourceDir, "SKILL.md")
	if _, err := os.Stat(sourcePath); err != nil {
		return ResolvedSkill{Name: name}, sourcePath, err
	}
	return ResolvedSkill{
		Name:       name,
		SourceDir:  sourceDir,
		SourcePath: sourcePath,
	}, sourcePath, nil
}

func ReadMCPRegistryEntry(name string) (*MCPRegistryEntry, string, error) {
	if !validName(name) {
		return nil, "", fieldError("", "name", "invalid name %q", name)
	}

	path := MCPRegistryPath(name)
	var entry MCPRegistryEntry
	if err := readYAML(path, &entry); err != nil {
		return nil, path, err
	}
	if strings.TrimSpace(entry.Name) == "" {
		entry.Name = name
	}
	if entry.Name != name {
		return nil, path, fieldError(path, "name", "expected %q, got %q", name, entry.Name)
	}
	if entry.Kind != "" && entry.Kind != "mcp" {
		return nil, path, fieldError(path, "kind", "expected %q, got %q", "mcp", entry.Kind)
	}
	return &entry, path, nil
}

func WriteMCPRegistryEntry(entry *MCPRegistryEntry) error {
	if entry == nil {
		return fieldError("", "", "mcp registry entry is nil")
	}
	if !validName(entry.Name) {
		return fieldError("", "name", "invalid name %q", entry.Name)
	}
	if entry.Kind == "" {
		entry.Kind = "mcp"
	}
	if entry.Kind != "mcp" {
		return fieldError("", "kind", "expected %q, got %q", "mcp", entry.Kind)
	}
	return writeYAML(MCPRegistryPath(entry.Name), entry)
}

func DeleteMCPRegistryEntry(name string) error {
	if !validName(name) {
		return fieldError("", "name", "invalid name %q", name)
	}
	return os.Remove(MCPRegistryPath(name))
}

func ListMCPRegistryEntries() ([]MCPRegistrySummary, error) {
	paths, err := listYAMLFiles(RegistryKindDir("mcps"))
	if err != nil {
		return nil, err
	}

	summaries := make([]MCPRegistrySummary, 0, len(paths))
	for _, path := range paths {
		name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		var entry MCPRegistryEntry
		if err := readYAML(path, &entry); err != nil {
			return nil, err
		}
		if strings.TrimSpace(entry.Name) == "" {
			entry.Name = name
		}
		if entry.Name != name {
			return nil, fieldError(path, "name", "expected %q, got %q", name, entry.Name)
		}
		if entry.Kind != "" && entry.Kind != "mcp" {
			return nil, fieldError(path, "kind", "expected %q, got %q", "mcp", entry.Kind)
		}
		summaries = append(summaries, MCPRegistrySummary{
			Name:        entry.Name,
			Description: entry.Description,
			Type:        entry.Server.Type,
			Command:     entry.Server.Command,
			URL:         entry.Server.URL,
			Path:        path,
		})
	}
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Name < summaries[j].Name
	})
	return summaries, nil
}

func resolvedMCPServer(name string) (ResolvedMCPServer, string, error) {
	entry, path, err := ReadMCPRegistryEntry(name)
	if err != nil {
		if os.IsNotExist(err) {
			return ResolvedMCPServer{Name: name}, path, err
		}
		return ResolvedMCPServer{Name: name}, path, err
	}
	return ResolvedMCPServer{
		Name:       entry.Name,
		Type:       entry.Server.Type,
		Command:    entry.Server.Command,
		Args:       cloneStringSlice(entry.Server.Args),
		Env:        cloneStringMap(entry.Server.Env),
		URL:        entry.Server.URL,
		Headers:    cloneStringMap(entry.Server.Headers),
		SourcePath: path,
	}, path, nil
}
