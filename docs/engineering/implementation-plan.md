# Agent VM — Coding Implementation Plan

> 日期：2026-04-24
> 范围：Phase 1 MVP
> 目标：支持多个 coding Agent 并发实现，同时避免文件冲突和架构漂移

---

## 总结

可以进入 coding，但不能一上来全并发。初始仓库需要先由一个 Lead Agent 做最小 scaffold，锁定 module、目录、根命令、测试命令和公共 package 边界。scaffold 完成后，再按文件所有权拆成多个并发 lane。

重要约束：**“推荐 Agent 分工”不是启动顺序**。启动顺序以 Stage/Round 为准；分工表只定义文件所有权和任务边界。任何公共接口、根命令、module 依赖、核心 struct 变更，都必须由对应 owner 串行落地后，其他 Agent 才能基于它继续。

Phase 1 的实现主线：

```text
Scaffold
  -> config/data model
  -> agent create/list/show
  -> memory import --dry-run
  -> use/sync render plan
  -> concrete runtime adapters
  -> acceptance fixtures
```

PRD 已明确：Phase 1 不做 `sync --watch`，不做 `workspace_isolation` 主字段，不静默同步 runtime native memory。

---

## 当前执行状态

### Round 0：Lead Scaffold

状态：`DONE`

Lead 验收结果：

- `go test ./...` 通过。
- `go run ./cmd/avm --help` 有稳定输出。
- Go module、Makefile、CI 占位、root command、version 包、公共 package 占位目录已存在。

后续动作：

- 可以启动 Round 1 的 4 个并发 Codex：Config、Adapter Contract、CLI Skeleton、Fixtures。
- Round 1 Agent 不应修改 `go.mod`、`go.sum`、`cmd/avm/root.go`，除非在交付说明中声明并交给 Lead 合并。

---

## 并发规则

### 启动规则

1. Round 0 只能启动 1 个 Lead Agent，负责 repo scaffold 和公共骨架。
2. Round 1 在 Round 0 合并后启动，建议并发 4 个 Codex：Config、Adapter Contract、CLI Skeleton、Fixtures。
3. Round 2 在 Config + CLI Skeleton 可编译后启动，建议并发 2 个 Codex：Memory、Agent CLI。
4. Round 3 在 Adapter contract 和 first vertical slice 稳定后启动，建议并发 Sync + runtime adapters。
5. 每个 Codex 建议使用独立 `git worktree` 和独立 branch，不建议多个 Codex 直接在同一个 worktree 并发写。
6. 任何 Agent 需要改 owner 之外的文件时，必须停止并在交付说明里声明，不要自行跨边界修改。

推荐 worktree 形态：

```bash
git worktree add ../agent-vm-scaffold -b feat/scaffold
git worktree add ../agent-vm-config -b feat/config
git worktree add ../agent-vm-adapter-contract -b feat/adapter-contract
git worktree add ../agent-vm-cli -b feat/cli-skeleton
git worktree add ../agent-vm-fixtures -b feat/fixtures
```

### 必须串行

这些任务会定义公共接口或全局结构，不能多人同时改：

| 标记 | 任务 | 原因 | Owner |
|------|------|------|-------|
| `S0` | Go module、目录结构、测试命令、CI 占位 | 所有后续任务依赖 | Lead Agent |
| `S1` | `internal/config` 核心 struct 命名 | adapter、sync、memory 都依赖 | Config Agent |
| `S2` | `internal/adapter` interface | concrete adapter 和 sync 都依赖 | Adapter Contract Agent |
| `S3` | CLI root command 和错误输出格式 | 所有命令共享 | CLI Agent |
| `S4` | integration 合并和最终验收 | 需要统一行为 | Lead Agent |

### 可以并发

这些任务有清晰文件边界，可以并行：

| 标记 | 任务 | 写入范围 | 依赖 |
|------|------|----------|------|
| `P1` | Config YAML 读写 + validation | `internal/config/**` | `S0` |
| `P2` | Portable Memory import dry-run | `internal/memory/**`, `cmd/avm/memory*.go` | `S1`, `S3` |
| `P3` | Adapter contract + fake adapter | `internal/adapter/**` | `S0` |
| `P4` | Sync/state/backup | `internal/sync/**`, `internal/state/**` | `S1`, `S2` |
| `P5` | CLI agent/env/status commands | `cmd/avm/**` | `S1`, `S3` |
| `P6` | Runtime fixtures | `testdata/**`, `fixtures/**` | `S0` |
| `P7` | Claude Code adapter | `internal/adapter/claude/**` | `S2` |
| `P8` | Codex adapter | `internal/adapter/codex/**` | `S2` |
| `P9` | Cline adapter | `internal/adapter/cline/**` | `S2` |
| `P10` | Cursor PoC adapter | `internal/adapter/cursor/**` | `S2` |

---

## 推荐 Agent 分工

### Lead Agent

Owner:

- `go.mod`, `go.sum`
- `cmd/avm/root.go`
- `internal/version/**`
- `Makefile` 或 `justfile`
- integration merge

交付：

- `go test ./...` 可运行。
- `avm --help` 可运行。
- 所有公共 package 有空实现或接口占位。

不要做：

- 不实现具体 runtime adapter。
- 不把业务逻辑塞进 `cmd/avm`。

### Config Agent

Owner:

- `internal/config/**`
- `testdata/config/**`

交付：

- `GlobalConfig`
- `AgentProfile`
- `Environment`
- `PortableMemory`
- `Read/Write/List/Validate`
- `ResolveActivation`

验收：

- YAML round-trip 字段稳定。
- `workspace_isolation` 不存在于 `AgentProfile`。
- Environment 不接受 `capabilities` 或 `memory_layers`。

### Memory Agent

Owner:

- `internal/memory/**`
- `cmd/avm/memory*.go`
- `testdata/memory/**`

交付：

- `avm memory import --from <path|runtime> --dry-run`
- `MemoryImportPlan`
- `MemoryDiff`
- `memory-import-report.json`

验收：

- dry-run 不写 runtime 文件。
- dry-run 不写正式 `~/.avm/memory/` 文件。
- 输出 `new | changed | conflict | skipped`。

### Adapter Contract Agent

Owner:

- `internal/adapter/adapter.go`
- `internal/adapter/fake/**`
- `internal/adapter/renderplan/**`

交付：

- `Adapter` interface
- `RenderPlan`
- `FieldMapping`
- `ManagedPath`
- optional `MemoryImportCapable`
- fake adapter for sync tests

验收：

- mapping status 只允许 `native | rendered_as_instructions | ignored | unsupported`。
- fake adapter 能生成 deterministic render plan。

### Sync Agent

Owner:

- `internal/sync/**`
- `internal/state/**`
- `internal/backup/**`

交付：

- `RebuildActive`
- `SyncActivation`
- conflict detection
- backup
- `sync-state.json`

验收：

- active 重建失败不破坏旧 active。
- 只检测 `ManagedPaths`。
- 不实现 watch。

### CLI Agent

Owner:

- `cmd/avm/agent*.go`
- `cmd/avm/env*.go`
- `cmd/avm/use*.go`
- `cmd/avm/status*.go`
- `cmd/avm/shell*.go`

交付：

- `avm init`
- `avm agent create/list/show`
- `avm env create`
- `avm use`
- `avm status`
- `avm shell init`
- `avm deactivate`

验收：

- 命令输出稳定，适合 golden tests。
- 命令层只编排，不直接写 runtime config。

### Runtime Adapter Agents

并发启动条件：`Adapter` interface 和 `RenderInput` 已锁定。

| Agent | Owner | 交付 |
|------|-------|------|
| Claude Agent | `internal/adapter/claude/**` | `.claude/agents/*.md`, `.mcp.json`, memory refs 渲染 |
| Codex Agent | `internal/adapter/codex/**` | `config.toml`, role TOML, skills/memory instructions |
| Cline Agent | `internal/adapter/cline/**` | MCP settings, `.clinerules/avm/*.md` |
| Cursor Agent | `internal/adapter/cursor/**` | `.cursor/mcp.json`, rules PoC, partial status |

共同验收：

- 不覆盖用户 instruction 文件。
- 不展开 `${ENV_VAR}`。
- unsupported/ignored/rendered 都写入 mapping status。

---

## 执行阶段

### Stage 0：Scaffold 串行

标记：`SERIAL`

任务：

1. 初始化 Go module。
2. 建目录：

```text
cmd/avm/
internal/config/
internal/adapter/
internal/memory/
internal/sync/
internal/state/
internal/backup/
internal/runtime/
testdata/
```

3. 建 `README` 里的本地开发命令。
4. 建最小 `go test ./...`。

退出条件：

- `go test ./...` 通过。
- `go run ./cmd/avm --help` 有输出。

### Stage 1：Core Contract 并发

标记：`PARALLEL`

可同时启动：

- `P1 Config`
- `P3 Adapter Contract`
- `P6 Runtime Fixtures`
- `P5 CLI skeleton`

退出条件：

- config structs 可编译。
- adapter fake 可编译。
- CLI commands 可注册但允许返回 `not implemented`。

### Stage 2：First Vertical Slice

标记：`JOIN`

目标：先打通一个非 runtime 写入路径。

任务：

1. `avm init`
2. `avm agent create/list/show`
3. `avm memory import --from <file> --dry-run`

退出条件：

- 能在临时 HOME 下跑 e2e。
- 不写任何 runtime 配置文件。
- `memory import --dry-run` 有 diff 输出。

### Stage 3：Activation Pipeline

标记：`PARALLEL_AFTER_JOIN`

可同时启动：

- `P4 Sync/state/backup`
- `P5 use/status/deactivate`
- `P7/P8/P9/P10 adapter implementations`

退出条件：

- `avm use backend-coder` 能重建 active。
- fake adapter 能写 managed path 并记录 mapping status。
- concrete adapter 至少 Codex 或 Claude Code 打通一个。

### Stage 4：Runtime Adapter Completion

标记：`PARALLEL`

任务：

- Claude Code adapter
- Codex adapter
- Cline adapter
- Cursor PoC

退出条件：

- 每个 adapter 有 fixture。
- 每个 adapter 有 `Plan` 和 `Render` 单测。
- `avm status` 能显示 partial/ignored/unsupported。

### Stage 5：Acceptance Hardening

标记：`SERIAL_JOIN`

任务：

1. 按 [acceptance.md](./acceptance.md) 跑全量 e2e。
2. 修冲突检测、路径权限、export/import 边界。
3. 补 README 和示例。

退出条件：

- `go test ./...` 通过。
- Phase 1 acceptance 核心路径通过。
- 没有 silent field drop。

---

## 文件所有权

并发时按 owner 控制写入：

| 路径 | Owner | 备注 |
|------|-------|------|
| `go.mod`, `go.sum` | Lead Agent | 其他 Agent 需要依赖时先记录，合并时统一添加 |
| `cmd/avm/root.go` | Lead Agent | 子命令文件可由 CLI/Memory Agent 写 |
| `cmd/avm/memory*.go` | Memory Agent | 不和 CLI Agent 混改 |
| `cmd/avm/agent*.go`, `env*.go`, `use*.go`, `status*.go` | CLI Agent | 只做命令编排 |
| `internal/config/**` | Config Agent | 其他 Agent 不改 config struct |
| `internal/adapter/adapter.go` | Adapter Contract Agent | concrete adapter 不改 interface |
| `internal/adapter/<runtime>/**` | 对应 Runtime Agent | 每个 runtime 独立 |
| `internal/memory/**` | Memory Agent | 包含 import dry-run 和 diff |
| `internal/sync/**` | Sync Agent | 不改 adapter |
| `internal/state/**` | Sync Agent | memory report 可由 Memory Agent 加独立文件 |
| `testdata/<area>/**` | 对应 Agent | fixture 按模块拆分 |

---

## 禁止并发修改

以下文件需要单 owner 串行合并：

- `go.mod`
- `go.sum`
- `cmd/avm/root.go`
- `internal/config/models.go`
- `internal/adapter/adapter.go`
- `internal/config/resolve.go`
- `internal/sync/syncer.go`

如果必须改，先在任务说明里声明，等 owner 合并后再继续。

---

## 第一轮推荐任务队列

```text
Round 0
  Lead Agent:
    S0 scaffold repo, go.mod, root command, go test ./...

Round 1
  Config Agent:
    P1 internal/config models + read/write + validation
  Adapter Contract Agent:
    P3 internal/adapter interface + fake adapter
  CLI Agent:
    P5 cobra subcommand skeleton
  Fixture Agent:
    P6 testdata layout and fixture conventions

Round 2
  Memory Agent:
    P2 memory import --dry-run from file
  CLI Agent:
    agent create/list/show
  Config Agent:
    ResolveActivation

Round 3
  Sync Agent:
    active rebuild + state + backup + fake adapter render
  Runtime Agents:
    Claude / Codex / Cline / Cursor adapters in parallel

Round 4
  Lead Agent:
    integration, acceptance, docs cleanup
```

## 手动 Codex 启动 Prompts

这些 prompt 用于人工启动多个 Codex。每个 Codex 从主仓库启动后，必须自己创建独立 worktree，进入 worktree 开发，验收通过后 push 到远程分支，然后停止。除非 prompt 明确允许，不要修改 owner 之外的文件。

通用规则：

- 初始目录可以是主仓库 `/Users/danielxing/code/agent-vm`。
- 不要在主仓库 worktree 里改业务文件。
- 如果目标 worktree 已存在，进入已有 worktree 继续；不要删除目录。
- 如果远程不存在或 push 失败，保留本地 branch，并在最终说明中明确阻塞原因。
- 开发完成后运行 `git status --short`，确认只包含本任务允许的文件。
- push 命令使用 `git push -u origin <branch>`。

### Prompt 0：Lead Scaffold（先单独跑）

```text
你是 Agent VM Phase 1 的 Lead Scaffold Agent。

目标：
- 完成 Round 0 / Stage 0 scaffold。
- 当前仓库是空 Go 仓库或接近空仓库。

写入范围：
- go.mod, go.sum
- cmd/avm/root.go
- internal/version/**
- Makefile 或 justfile
- 必要的空 package 占位文件

任务：
1. 初始化 Go module。
2. 建立目录：
   - cmd/avm/
   - internal/config/
   - internal/adapter/
   - internal/memory/
   - internal/sync/
   - internal/state/
   - internal/backup/
   - internal/runtime/
   - testdata/
3. 建最小 CLI root command，使 `go run ./cmd/avm --help` 有稳定输出。
4. 建本地测试命令，使 `go test ./...` 可运行并通过。
5. 不实现具体业务逻辑，不实现 runtime adapter，不把业务逻辑塞进 cmd/avm。

验收：
- `go test ./...` 通过。
- `go run ./cmd/avm --help` 有输出。
- 公共 package 有空实现或接口占位。
- 最终说明列出修改文件、测试命令和结果。
```

### Prompt 1：Config Agent（Round 1）

```text
你是 Agent VM Phase 1 的 Config Agent。

前置条件：
- Lead Scaffold 已合并。
- 你从主仓库启动，但必须先自己创建并进入独立 worktree。

启动步骤：
1. 在当前目录运行：
   - pwd
   - git branch --show-current
   - git remote -v
   - git worktree list
   - git status --short
2. 创建或进入 worktree：
   - 目标目录：../agent-vm-config
   - 目标 branch：feat/config
   - 如果 ../agent-vm-config 不存在且 feat/config branch 不存在，运行：git worktree add ../agent-vm-config -b feat/config
   - 如果 ../agent-vm-config 不存在但 feat/config branch 已存在，运行：git worktree add ../agent-vm-config feat/config
   - 如果目录已存在，进入该目录并确认 branch 是 feat/config。
3. 进入 worktree 后再次运行：
   - pwd
   - git branch --show-current
   - git status --short
4. 只有当 pwd 位于 ../agent-vm-config 且 branch 是 feat/config 时，才允许修改文件。否则立即停止。

写入范围：
- internal/config/**
- testdata/config/**

允许修改：
- go.mod/go.sum 仅当需要引入 YAML 依赖，例如 gopkg.in/yaml.v3。若修改，最终说明必须单独列出依赖变更，交给 Lead 合并。

不要修改：
- cmd/avm/**
- internal/adapter/**
- internal/sync/**
- docs/engineering/implementation-plan.md

任务：
1. 实现 Phase 1 config data model：
   - ActiveRef
   - GlobalConfig
   - AgentProfile
   - Environment
   - PortableMemory
2. 使用 yaml.v3，不要自研 YAML parser/encoder。
3. 实现 Read / Write / List / Validate。
4. 添加 YAML round-trip 和 validation tests。
5. 若 ResolveActivation 依赖其他未完成模块，可以预留清晰接口并说明。

收尾步骤：
1. 运行 `go test ./...`。
2. 运行 `git status --short`，确认只包含允许文件。
3. 提交本 branch。
4. 运行 `git push -u origin feat/config`。
5. push 完成后停止，不要继续做 Round 2。

验收：
- `go test ./...` 通过。
- YAML round-trip 字段稳定。
- `workspace_isolation` 不存在于 AgentProfile。
- Environment 不接受 `capabilities` 或 `memory_layers`。
- 最终说明列出修改文件、测试命令、结果、是否改了 go.mod/go.sum、远程 branch。
```

### Prompt 2：Adapter Contract Agent（Round 1）

```text
你是 Agent VM Phase 1 的 Adapter Contract Agent。

前置条件：
- Lead Scaffold 已合并。
- 你从主仓库启动，但必须先自己创建并进入独立 worktree。

启动步骤：
1. 在当前目录运行：
   - pwd
   - git branch --show-current
   - git remote -v
   - git worktree list
   - git status --short
2. 创建或进入 worktree：
   - 目标目录：../agent-vm-adapter-contract
   - 目标 branch：feat/adapter-contract
   - 如果 ../agent-vm-adapter-contract 不存在且 feat/adapter-contract branch 不存在，运行：git worktree add ../agent-vm-adapter-contract -b feat/adapter-contract
   - 如果 ../agent-vm-adapter-contract 不存在但 feat/adapter-contract branch 已存在，运行：git worktree add ../agent-vm-adapter-contract feat/adapter-contract
   - 如果目录已存在，进入该目录并确认 branch 是 feat/adapter-contract。
3. 进入 worktree 后再次运行：
   - pwd
   - git branch --show-current
   - git status --short
4. 只有当 pwd 位于 ../agent-vm-adapter-contract 且 branch 是 feat/adapter-contract 时，才允许修改文件。否则立即停止。

写入范围：
- internal/adapter/adapter.go
- internal/adapter/fake/**
- internal/adapter/renderplan/**

不要修改：
- concrete runtime adapter 目录，例如 internal/adapter/codex/**
- internal/config/**
- cmd/avm/**
- internal/sync/**
- docs/engineering/implementation-plan.md
- go.mod/go.sum，除非绝对必要；如必须修改，最终说明中声明。

任务：
1. 定义 Adapter interface。
2. 定义 RenderInput、RenderPlan、RenderOperation、FieldMapping、ManagedPath。
3. 定义 MappingStatus，合法值只能是：
   - native
   - rendered_as_instructions
   - ignored
   - unsupported
4. 定义 optional MemoryImportCapable。
5. 实现 fake adapter，供后续 sync tests 使用。
6. 添加 deterministic render plan tests。

约束：
- 当前 config types 可能还没合并，可以先让 adapter contract 自己可编译。
- 不要复制或重建完整 config.AgentProfile schema。
- 不要用泛滥的 map[string]any 代替核心合同。
- 等 Config Agent 合并后，Lead 会负责把 RenderInput 与 config.ResolvedActivation / config.AgentProfile 对齐。

收尾步骤：
1. 运行 `go test ./...`。
2. 运行 `git status --short`，确认只包含允许文件。
3. 提交本 branch。
4. 运行 `git push -u origin feat/adapter-contract`。
5. push 完成后停止，不要继续做 runtime adapter。

验收：
- `go test ./...` 通过。
- mapping status 只允许：
  - native
  - rendered_as_instructions
  - ignored
  - unsupported
- fake adapter 能生成 deterministic render plan。
- 最终说明列出修改文件、测试命令、结果、哪些点等待 config types 接入、远程 branch。
```

### Prompt 3：CLI Skeleton Agent（Round 1）

```text
你是 Agent VM Phase 1 的 CLI Skeleton Agent。

前置条件：
- Lead Scaffold 已合并。
- 你从主仓库启动，但必须先自己创建并进入独立 worktree。

启动步骤：
1. 在当前目录运行：
   - pwd
   - git branch --show-current
   - git remote -v
   - git worktree list
   - git status --short
2. 创建或进入 worktree：
   - 目标目录：../agent-vm-cli
   - 目标 branch：feat/cli-skeleton
   - 如果 ../agent-vm-cli 不存在且 feat/cli-skeleton branch 不存在，运行：git worktree add ../agent-vm-cli -b feat/cli-skeleton
   - 如果 ../agent-vm-cli 不存在但 feat/cli-skeleton branch 已存在，运行：git worktree add ../agent-vm-cli feat/cli-skeleton
   - 如果目录已存在，进入该目录并确认 branch 是 feat/cli-skeleton。
3. 进入 worktree 后再次运行：
   - pwd
   - git branch --show-current
   - git status --short
4. 只有当 pwd 位于 ../agent-vm-cli 且 branch 是 feat/cli-skeleton 时，才允许修改文件。否则立即停止。

写入范围：
- cmd/avm/agent*.go
- cmd/avm/env*.go
- cmd/avm/use*.go
- cmd/avm/status*.go
- cmd/avm/shell*.go
- cmd/avm/deactivate*.go
- cmd/avm/*_test.go only for CLI skeleton tests

不要修改：
- internal/config/**
- internal/adapter/**
- internal/sync/**
- go.mod/go.sum，除非绝对必要；如必须修改，最终说明中声明。
- docs/engineering/implementation-plan.md

关于 cmd/avm/root.go：
- 默认不要改。
- 如果现有 root command 没有子命令注册扩展点，允许做最小修改，例如增加 addCommands(root) hook。
- 如果改了 root.go，最终说明必须明确写出原因和改动范围。

任务：
1. 注册以下命令 skeleton：
   - avm init
   - avm agent create/list/show
   - avm env create
   - avm use
   - avm status
   - avm shell init
   - avm deactivate
2. 命令可以返回稳定的 `not implemented`。
3. 命令输出要稳定，便于 golden tests。
4. 命令层只做编排，不直接写 runtime config。

收尾步骤：
1. 运行 `go test ./...`。
2. 运行 `go run ./cmd/avm --help`。
3. 运行关键子命令 `--help`。
4. 运行 `git status --short`，确认只包含允许文件。
5. 提交本 branch。
6. 运行 `git push -u origin feat/cli-skeleton`。
7. push 完成后停止，不要实现业务逻辑。

验收：
- `go test ./...` 通过。
- `go run ./cmd/avm --help` 能看到命令。
- 关键子命令 help 可运行。
- 最终说明列出修改文件、测试命令、结果、是否改了 root.go、远程 branch。
```

### Prompt 4：Fixture Agent（Round 1）

```text
你是 Agent VM Phase 1 的 Runtime Fixture Agent。

前置条件：
- Lead Scaffold 已合并。
- 你从主仓库启动，但必须先自己创建并进入独立 worktree。

启动步骤：
1. 在当前目录运行：
   - pwd
   - git branch --show-current
   - git remote -v
   - git worktree list
   - git status --short
2. 创建或进入 worktree：
   - 目标目录：../agent-vm-fixtures
   - 目标 branch：feat/fixtures
   - 如果 ../agent-vm-fixtures 不存在且 feat/fixtures branch 不存在，运行：git worktree add ../agent-vm-fixtures -b feat/fixtures
   - 如果 ../agent-vm-fixtures 不存在但 feat/fixtures branch 已存在，运行：git worktree add ../agent-vm-fixtures feat/fixtures
   - 如果目录已存在，进入该目录并确认 branch 是 feat/fixtures。
3. 进入 worktree 后再次运行：
   - pwd
   - git branch --show-current
   - git status --short
4. 只有当 pwd 位于 ../agent-vm-fixtures 且 branch 是 feat/fixtures 时，才允许修改文件。否则立即停止。

写入范围：
- testdata/**
- fixtures/**
- docs 中与 fixture convention 直接相关的小段说明

不要修改：
- internal/**
- cmd/**
- go.mod/go.sum
- docs/engineering/implementation-plan.md

任务：
1. 建立 Phase 1 fixture layout。
2. 为以下场景预留 fixture convention：
   - config
   - memory import dry-run
   - adapter render plan
   - Codex adapter output
   - Claude Code adapter output
   - Cline adapter output
   - Cursor PoC output
3. 添加最小示例 fixture，覆盖 Codex、Claude Code、Cline、Cursor PoC 的目录形态。
4. 避免写入真实用户 runtime 配置路径。

收尾步骤：
1. 运行 `go test ./...`。
2. 运行 `git status --short`，确认只包含允许文件。
3. 提交本 branch。
4. 运行 `git push -u origin feat/fixtures`。
5. push 完成后停止，不要实现业务代码。

验收：
- fixture 路径命名稳定。
- README 或 docs 说明如何引用 fixture。
- 不需要业务代码也不能破坏 `go test ./...`。
- 最终说明列出修改文件、验证方式、远程 branch。
```

---

## 成功标准

Phase 1 coding 完成时至少满足：

1. `avm init` 只写 `~/.avm/**`。
2. `avm agent create/list/show` 可用。
3. `avm memory import --dry-run` 可用，且不写 runtime native memory。
4. `avm use <profile>` 可重建 active 并调用 adapter render。
5. `avm env create/use` 可按 runtime 选择不同 Agent Profile。
6. Codex、Claude Code、Cline 至少有 fixture 覆盖。
7. Cursor 明确标记 partial。
8. `workspace_isolation` 不存在于 Agent Profile 主模型。
9. mapping status 无 silent drop。
10. `go test ./...` 通过。
