# Architecture Overview

This is a short reader's guide that maps `docs/rewrite-architecture-proposal.md`
onto the actual code packages. Read the proposal first for the *why*; this file
is the *where*.

## Layer to package map

| Proposal layer | Package(s) | Responsibility |
| --- | --- | --- |
| Presentation | `internal/presentation/cli`, `internal/presentation/render` | cobra commands, huh prompts, rendering. Owns command/flag parsing, interactive UX, output formatting. |
| Application — model | `internal/app/model` | AVM-stable product model: `Agent`, `CapabilityRef`, `PackageManifest`, `MappingStatus`, `RunPreview`/`RunResult`/`RunRecord`, request DTOs. |
| Application — service | `internal/app/service` | `AgentService`, `RunService`, `PackageService`, `CapabilityService`, `DiagnosticsService`, plus a single `Container` so presentation depends on one struct. |
| Runtime Integration | `internal/runtime`, `internal/runtime/{codex,claudecode,opencode}` | `Driver` (Facts/DiscoverGlobal/Plan/Boundary/LaunchSpec) and `Registry`. Each runtime is a self-contained driver. |
| Infrastructure | `internal/infra/{home,agentstore,capstore,packageio,runlog,process,managedfile,fsutil}` | Side effects: home layout, Agent YAML persistence, capability content store, zip package IO, run history, process spawning, managed-file atomic writes. |

## Visual layer map

```
┌─────────────────────────────────────────────────────────────────┐
│ Layer 1: Presentation (internal/presentation/cli/)              │
│   root.go · agent.go · run.go · package.go · capability.go     │
│   init.go · render.go                                            │
└──────────────────────────┬──────────────────────────────────────┘
                           │ only imports service.Container + model
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│ Layer 2: Application                                             │
│                                                                  │
│  2a · model (pure data, no external deps)                        │
│      agent.go · capability.go · run.go · package.go              │
│      requests.go · diagnostics.go · system.go                    │
│                                                                  │
│  2b · service (business orchestration)                           │
│      AgentService       Repo + Runtimes                          │
│      RunService         Repo + Runtimes + Writer + Process + Log │
│      PackageService     Agents + Caps + IO                       │
│      CapabilityService  Store + Runtimes                         │
│      DiagnosticsService Runtimes + Log                           │
│      SystemService      Layout                                   │
│      container.go (DI aggregate) · errors.go (typed Error)       │
└─────────┬─────────────────────────────────────┬─────────────────┘
          │                                     │
          ▼                                     ▼
┌──────────────────────────────┐   ┌────────────────────────────┐
│ Layer 3: Runtime Integration │   │ Layer 4: Infrastructure    │
│  (internal/runtime/)         │   │  (internal/infra/)         │
│                              │   │                            │
│  driver.go (Driver iface,    │   │  agentstore/   Agent yaml  │
│            Exported,         │   │                  CRUD      │
│            MCPConfigV1)      │   │  capstore/     content-    │
│  registry.go (MapRegistry)   │   │                  addressed │
│  types.go (Plan, Boundary,   │   │                  cap store │
│            LaunchSpec,       │   │  managedfile/  DryRun +    │
│            ManagedFile,      │   │                  Apply     │
│            FieldMapping)     │   │  process/      fork/exec   │
│                              │   │  runlog/       JSONL log   │
│  codex/driver.go             │   │  packageio/    zip pack    │
│  claudecode/driver.go        │   │  home/         AVM_HOME    │
│  opencode/driver.go          │   │                  layout    │
│                              │   │  fsutil/       atomic      │
│                              │   │                  write     │
└──────────────────────────────┘   └────────────────────────────┘

Dependency direction:
- 1 → 2          (presentation only sees service.Container + model)
- 2b → 3, 2b → 4 (services hold driver + infra implementations via interfaces)
- 2b → 2a, 3 → 2a (services and drivers both speak in model types)
- 3 ↛ 4, 4 ↛ 3  (runtime and infra are siblings, never cross-reference)
- 4 ↛ 4         (infra packages are independent of each other)
```

## Service ↔ infra dependency matrix

Which service uses which lower-layer interface (Run is by far the heaviest):

| Service     | agentstore | capstore | managedfile | process | runlog | packageio | home | runtime |
| ---         | :-:        | :-:      | :-:         | :-:     | :-:    | :-:       | :-:  | :-:     |
| Agent       | ✓          |          |             |         |        |           |      | ✓       |
| Run         | ✓          |          | ✓           | ✓       | ✓      |           |      | ✓       |
| Package     | ✓          | ✓        |             |         |        | ✓         |      |         |
| Capability  |            | ✓        |             |         |        |           |      | ✓       |
| Diagnostics |            |          |             |         | ✓      |           |      | ✓       |
| System      |            |          |             |         |        |           | ✓    |         |

`cmd/avm/main.go` is the only composition root: it builds the registry,
instantiates each infra component, assembles a `service.Container`, and hands
the CLI a `cli.Deps`. No package below it imports anything above it.

## Call direction (top → bottom)

```
cmd/avm/main.go
  -> internal/presentation/cli            (parse args, call services, render)
       -> internal/app/service            (product rules)
            -> internal/runtime           (Driver/Registry)
            -> internal/infra/*           (filesystem & process)
            -> internal/app/model         (stable types)
```

Reverse imports are forbidden. If you ever feel like infra "needs to know"
something about a service, that's the signal to push the policy up, not the
data down.

## Key invariants

- The application layer never writes runtime-managed paths. It calls
  `runtime.Driver.Plan`, hands the resulting `runtime.ManagedFile` slice to
  `infra/managedfile`, and the writer is the only thing that touches disk.
- Capability identity is content-addressed. `capstore.Add` derives the ID from
  `sha256(kind + "\n" + name + "\n" + content_sha256)`, so package provenance
  (`ImportFrom`) is audit-only and never affects which Agent references which
  capability.
- Per-(Agent, runtime) isolation lives under `$AVM_HOME/boundaries/<runtime>/<agent-name>/`
  and the driver owns which env vars point the runtime at it.

## Where to start reading

- New to the codebase: `cmd/avm/main.go` -> `internal/app/service/container.go`
  -> any one service file (e.g. `run.go`) to see the typical orchestration shape.
- New to a runtime: read the matching research doc in
  `docs/engineering/runtime-research/`, then `internal/runtime/<name>/driver.go`.
- New to package format: `internal/app/model/package.go` for the manifest, then
  `internal/infra/packageio/packageio.go`, then `internal/app/service/package.go`.
