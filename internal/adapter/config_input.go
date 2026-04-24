package adapter

import (
	"fmt"
	"sort"

	"github.com/xz1220/agent-vm/internal/config"
)

type RenderInputOptions struct {
	ProjectRoot string
	ActiveDir   string
}

func RenderInputFromResolved(resolved *config.ResolvedActivation, runtime string, opts RenderInputOptions) (RenderInput, error) {
	if resolved == nil {
		return RenderInput{}, fmt.Errorf("resolved activation is nil")
	}
	if runtime == "" {
		return RenderInput{}, fmt.Errorf("runtime is required")
	}

	agent, ok := resolved.RuntimeAgents[runtime]
	if !ok {
		return RenderInput{}, fmt.Errorf("runtime %q has no resolved agent", runtime)
	}
	if opts.ActiveDir == "" {
		opts.ActiveDir = config.ActiveDir()
	}

	return RenderInput{
		Active:       activeRefFromConfig(resolved.Active),
		Runtime:      runtime,
		Agent:        agentFromConfig(agent),
		Capabilities: capabilitySetFromConfig(resolved.Capabilities[runtime]),
		Memory:       portableMemoryFromConfig(resolved.Memory[runtime]),
		ProjectRoot:  opts.ProjectRoot,
		ActiveDir:    opts.ActiveDir,
	}, nil
}

func RenderInputsFromResolved(resolved *config.ResolvedActivation, opts RenderInputOptions) ([]RenderInput, error) {
	if resolved == nil {
		return nil, fmt.Errorf("resolved activation is nil")
	}

	runtimes := resolvedRuntimeOrder(resolved)
	inputs := make([]RenderInput, 0, len(runtimes))
	for _, runtime := range runtimes {
		input, err := RenderInputFromResolved(resolved, runtime, opts)
		if err != nil {
			return nil, err
		}
		inputs = append(inputs, input)
	}
	return inputs, nil
}

func activeRefFromConfig(ref config.ActiveRef) ActiveRef {
	return ActiveRef{
		Kind: ref.Kind,
		Name: ref.Name,
	}
}

func agentFromConfig(agent config.AgentProfile) Agent {
	return Agent{
		Name:        agent.Name,
		Description: agent.Description,
		SourceScope: agent.SourceScope,
		Instructions: Instructions{
			System:     agent.Instructions.System,
			Developer:  agent.Instructions.Developer,
			References: append([]string(nil), agent.Instructions.References...),
		},
		Model: ModelConfig{
			Model:           agent.ModelRun.Model,
			ReasoningEffort: agent.ModelRun.ReasoningEffort,
			Verbosity:       agent.ModelRun.Verbosity,
			Temperature:     agent.ModelRun.Temperature,
		},
		Permissions: PermissionConfig{
			Approval:              agent.Permissions.Approval,
			Sandbox:               agent.Permissions.Sandbox,
			Allow:                 append([]string(nil), agent.Permissions.Allow...),
			Deny:                  append([]string(nil), agent.Permissions.Deny...),
			AdditionalDirectories: append([]string(nil), agent.Permissions.AdditionalDirectories...),
		},
		MemoryRefs: memoryRefsFromConfig(agent.MemoryRefs),
	}
}

func capabilitySetFromConfig(capabilities config.ResolvedCapabilities) CapabilitySet {
	return CapabilitySet{
		Skills:     capabilityRefs(capabilities.Skills),
		MCPServers: mcpServers(capabilities.MCPs),
		Commands:   capabilityRefs(capabilities.Commands),
		Hooks:      capabilityRefs(capabilities.Hooks),
		Toolsets:   toolsets(capabilities.Toolsets),
	}
}

func capabilityRefs(names []string) []CapabilityRef {
	refs := make([]CapabilityRef, 0, len(names))
	for _, name := range names {
		if name != "" {
			refs = append(refs, CapabilityRef{Name: name})
		}
	}
	return refs
}

func mcpServers(names []string) []MCPServer {
	servers := make([]MCPServer, 0, len(names))
	for _, name := range names {
		if name != "" {
			servers = append(servers, MCPServer{Name: name})
		}
	}
	return servers
}

func toolsets(values map[string]string) []Toolset {
	if len(values) == 0 {
		return nil
	}

	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)

	items := make([]Toolset, 0, len(names))
	for _, name := range names {
		items = append(items, Toolset{Name: name, Mode: values[name]})
	}
	return items
}

func memoryRefsFromConfig(refs []config.MemoryRef) []MemoryRef {
	out := make([]MemoryRef, 0, len(refs))
	for _, ref := range refs {
		out = append(out, MemoryRef{
			ID:    ref.ID,
			Scope: ref.Scope,
			Path:  ref.Path,
			Mode:  ref.Mode,
		})
	}
	return out
}

func portableMemoryFromConfig(memory []config.PortableMemory) []PortableMemory {
	out := make([]PortableMemory, 0, len(memory))
	for _, item := range memory {
		out = append(out, PortableMemory{
			ID:    item.ID,
			Scope: item.Scope,
			Path:  item.Path,
			Mode:  item.Mode,
		})
	}
	return out
}

func resolvedRuntimeOrder(resolved *config.ResolvedActivation) []string {
	if resolved == nil || len(resolved.RuntimeAgents) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(resolved.RuntimeAgents))
	runtimes := make([]string, 0, len(resolved.RuntimeAgents))
	for _, runtime := range resolved.Targets {
		if _, ok := resolved.RuntimeAgents[runtime]; !ok {
			continue
		}
		if _, ok := seen[runtime]; ok {
			continue
		}
		seen[runtime] = struct{}{}
		runtimes = append(runtimes, runtime)
	}

	var remaining []string
	for runtime := range resolved.RuntimeAgents {
		if _, ok := seen[runtime]; !ok {
			remaining = append(remaining, runtime)
		}
	}
	sort.Strings(remaining)
	return append(runtimes, remaining...)
}
