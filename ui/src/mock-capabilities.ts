import type {CapabilityCandidate, CapabilityKind} from "./protocol.js";

const baseCandidates: CapabilityCandidate[] = [
  {
    kind: "skill",
    name: "git-review",
    source: "avm",
    record: {
      id: "cap_mock_skill_git_review",
      kind: "skill",
      name: "git-review",
      source: "avm",
      version: "mock"
    }
  },
  {
    kind: "skill",
    name: "repo-map",
    source: "avm",
    record: {
      id: "cap_mock_skill_repo_map",
      kind: "skill",
      name: "repo-map",
      source: "avm",
      version: "mock"
    }
  },
  {
    kind: "mcp",
    name: "filesystem",
    source: "avm",
    record: {
      id: "cap_mock_mcp_filesystem",
      kind: "mcp",
      name: "filesystem",
      source: "avm",
      version: "mock"
    }
  }
];

const runtimeNames = ["codex", "claude-code", "opencode"];

export function mockCapabilityCandidates(input: {
  kinds?: CapabilityKind[];
  runtimes?: string[];
}): CapabilityCandidate[] {
  const runtimes = input.runtimes && input.runtimes.length > 0 ? input.runtimes : runtimeNames;
  const runtimeCandidates = runtimes.flatMap((runtime) => [
    {
      kind: "skill" as const,
      name: `${runtime}-global-skill`,
      source: "runtime" as const,
      global: {
        runtime,
        kind: "skill" as const,
        name: `${runtime}-global-skill`,
        path: `mock://${runtime}/skills/${runtime}-global-skill`
      }
    },
    {
      kind: "mcp" as const,
      name: `${runtime}-mcp`,
      source: "runtime" as const,
      global: {
        runtime,
        kind: "mcp" as const,
        name: `${runtime}-mcp`,
        path: `mock://${runtime}/mcp/${runtime}-mcp`
      }
    }
  ]);

  const kinds = new Set(input.kinds ?? []);
  return [...baseCandidates, ...runtimeCandidates].filter((candidate) => {
    return kinds.size === 0 || kinds.has(candidate.kind);
  });
}

export function capabilityRefID(candidate: CapabilityCandidate): string {
  if (candidate.record?.id) {
    return candidate.record.id;
  }
  const runtime = candidate.global?.runtime ?? "runtime";
  return `mock_${candidate.kind}_${runtime}_${candidate.name}`.replace(/[^a-zA-Z0-9_-]/g, "_");
}
