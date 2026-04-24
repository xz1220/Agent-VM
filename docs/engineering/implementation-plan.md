# Agent VM — Coding Implementation Plan

> 日期：2026-04-24
> 范围：Phase 1 MVP
> 目标：支持多个 coding Agent 并发实现，同时避免文件冲突和架构漂移

---

## 总结

当前 `main` 已完成 Round 0/1：Go scaffold、config model、adapter contract、CLI skeleton 和 Phase 1 fixtures 都已合并。下一阶段进入 **Round 2 / First Vertical Slice**，目标是在不写任何 runtime 配置文件的前提下，打通：

```text
avm init
  -> avm agent create/list/show
  -> avm memory import --from <file> --dry-run
```

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

### 已完成

| Round | 状态 | 合并内容 | 验收 |
|-------|------|----------|------|
| Round 0 | `DONE` | Go module、Makefile、CI 占位、root command、version 包、公共 package 占位目录 | `go test ./...`、`go run ./cmd/avm --help` |
| Round 1 | `DONE` | `origin/feat/config`、`origin/feat/adapter-contract`、`origin/feat/cli-skeleton`、`origin/feat/fixtures` | `go test ./...`、`go vet ./...`、关键 CLI help、fixture JSON/YAML 校验、路径扫描 |

Round 1 合并后的能力基线：

- Config 层已有 Phase 1 model、YAML read/write/list/validate、round-trip tests。
- Adapter 层已有 contract、mapping status、fake adapter、render plan normalization。
- CLI 层已有命令 skeleton，命令可稳定返回 `not implemented`。
- Lead 已完成 `R2-P0`：`cmd/avm/commands.go` 拥有 root command registration，`cmd/avm/memory.go` 已预注册 `memory import` skeleton。
- Fixtures 已有 Phase 1 minimal layout。

### 下一阶段：Round 2 First Vertical Slice

状态：`R2-P0 DONE`

Round 2 不启动 Sync Agent 或 runtime adapter agents。它只打通 AVM source-of-truth 内部路径和 read-only dry-run memory import。

执行顺序：

1. `R2-P0 Lead prep` 串行：`DONE`。
2. `R2-P1 Config Resolve` 并发：实现 `ResolveActivation`、project/global profile lookup、env runtime mapping。
3. `R2-P2 Agent CLI` 并发：实现 `avm init`、`avm agent create/list/show`、基础 `avm env create`。
4. `R2-P3 Memory Import` 并发：实现 `avm memory import --from <file> --dry-run`。
5. `R2-P4 Lead integration` 串行：合并三个分支，在临时 HOME 下跑 e2e。

Round 2 退出条件：

- `go test ./...` 和 `go vet ./...` 通过。
- 临时 HOME 下 `avm init` 只写 `~/.avm/**`。
- `avm agent create backend-coder --runtime codex` 创建 agent YAML。
- `avm agent list/show` 输出稳定。
- `avm memory import --from <file> --dry-run` 输出 `new | changed | conflict | skipped`，且不写 runtime 文件、不写正式 `~/.avm/memory/**`。
- 仍不实现 `sync --watch`，不写 concrete runtime 配置。

---

## 并发规则

### 启动规则

1. Round 0 和 Round 1 已完成，不再启动对应 Agent。
2. Round 2 先由 Lead 做 `R2-P0` 串行准备，完成后再并发启动 Config Resolve、Agent CLI、Memory Import 三个 Codex。
3. Round 2 的三个 Codex 必须从最新 `origin/main` 创建独立 worktree 和 branch。
4. Round 3 只能在 Round 2 vertical slice 合并并通过 e2e 后启动。
5. 每个 Codex 必须使用独立 `git worktree` 和独立 branch，不允许多个 Codex 直接在同一个 worktree 并发写。
6. 任何 Agent 需要改 owner 之外的文件时，必须停止并在交付说明里声明，不要自行跨边界修改。

Round 2 推荐 worktree 形态：

```bash
git fetch origin main
git worktree add ../agent-vm-config-resolve -b feat/config-resolve origin/main
git worktree add ../agent-vm-agent-cli -b feat/agent-cli origin/main
git worktree add ../agent-vm-memory-import -b feat/memory-import origin/main
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
| `P1` | Config YAML 读写 + validation | `internal/config/**` | `DONE` |
| `P2` | Portable Memory import dry-run | `internal/memory/**`, `cmd/avm/memory*.go` | `S1`, `S3` |
| `P3` | Adapter contract + fake adapter | `internal/adapter/**` | `DONE` |
| `P4` | Sync/state/backup | `internal/sync/**`, `internal/state/**` | `S1`, `S2` |
| `P5` | CLI agent/env/status commands | `cmd/avm/**` | `S1`, `S3` |
| `P6` | Runtime fixtures | `testdata/**`, `fixtures/**` | `DONE` |
| `P11` | Config ResolveActivation | `internal/config/resolve*.go`, `merge*.go`, tests | `S1` |
| `P12` | Agent CLI implementation | `cmd/avm/init*.go`, `agent*.go`, `env*.go`, tests | `S1`, `S3` |
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

标记：`DONE`

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

标记：`DONE`

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

标记：`NEXT`

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
| `cmd/avm/root.go` | Lead Agent | 不并发改 |
| `cmd/avm/commands.go` | Lead Agent | Round 2 prep 可新增；用于集中注册，避免 Agent 冲突 |
| `cmd/avm/memory*.go` | Memory Agent | 不和 Agent CLI 混改；registration 由 Lead 集成 |
| `cmd/avm/init*.go`, `agent*.go`, `env*.go` | Agent CLI | Round 2 只实现 init/agent/env create |
| `cmd/avm/use*.go`, `status*.go`, `shell*.go`, `deactivate*.go` | CLI Agent | Round 3 再实现 use/status/deactivate 行为 |
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
- `cmd/avm/commands.go`
- `internal/config/models.go`
- `internal/adapter/adapter.go`
- `internal/config/resolve.go`
- `internal/sync/syncer.go`

如果必须改，先在任务说明里声明，等 owner 合并后再继续。

---

## Round 2 任务队列

```text
R2-P0 Lead prep (DONE)
  - 整理 cmd/avm 命令注册边界。
  - cmd/avm/commands.go 负责注册，cmd/avm/memory.go 已预注册 memory import skeleton。
  - Agent CLI 和 Memory Import 不需要同时改 root command 或同一个 registration 函数。

R2-P1 Config Resolve Agent (PARALLEL)
  - internal/config/resolve*.go, merge*.go, tests
  - ResolveActivation(profile/env), project/global lookup, env runtime mapping

R2-P2 Agent CLI Agent (PARALLEL)
  - cmd/avm/init*.go, agent*.go, env*.go, tests
  - avm init, avm agent create/list/show, avm env create

R2-P3 Memory Import Agent (PARALLEL)
  - internal/memory/**, cmd/avm/memory*.go, testdata/memory/**
  - avm memory import --from <file> --dry-run

R2-P4 Lead integration (SERIAL)
  - merge branches
  - run temporary HOME e2e
  - update this plan with Round 2 result
```

## Round 2 Codex Prompts

这些 prompt 用于人工启动 Round 2 的多个 Codex。每个 Codex 从主仓库启动后，必须自己创建独立 worktree，进入 worktree 开发，验收通过后 push 到远程分支，然后停止。

通用规则：

- 启动前先 `git fetch origin main`，并基于最新 `origin/main` 创建分支。
- 不要在主仓库 worktree 里改业务文件。
- 如果目标 worktree 已存在，进入已有 worktree 继续；不要删除目录。
- 如果远程不存在或 push 失败，保留本地 branch，并在最终说明中明确阻塞原因。
- 开发完成后运行 `git status --short`，确认只包含本任务允许的文件。
- push 命令使用 `git push -u origin <branch>`。

### Prompt 1：Config Resolve Agent

```text
你是 Agent VM Phase 1 Round 2 的 Config Resolve Agent。

启动步骤：
1. 在主仓库运行：pwd、git branch --show-current、git fetch origin main、git worktree list、git status --short。
2. 目标目录：../agent-vm-config-resolve；目标 branch：feat/config-resolve。
3. 如果目录不存在且 branch 不存在：git worktree add ../agent-vm-config-resolve -b feat/config-resolve origin/main
4. 如果目录不存在但 branch 已存在：git worktree add ../agent-vm-config-resolve feat/config-resolve
5. cd ../agent-vm-config-resolve 后确认 pwd 和 branch。只有在 feat/config-resolve worktree 中才允许修改文件。

写入范围：
- internal/config/resolve*.go
- internal/config/merge*.go
- internal/config/*resolve*_test.go
- testdata/config/resolve/**

不要修改：
- cmd/avm/root.go
- cmd/avm/commands.go
- cmd/**
- internal/adapter/**
- internal/memory/**
- docs/engineering/implementation-plan.md
- go.mod/go.sum，除非绝对必要；如必须修改，最终说明中声明

任务：
1. 实现 ResolvedActivation 和 ResolvedCapabilities 的最小 Phase 1 类型。
2. 实现 ResolveActivation(ref ActiveRef, cwd string)：
   - profile active：读取同名 Agent Profile。
   - env active：读取 Environment，并按 runtime_agents.<runtime>.primary 展开 Agent Profile。
   - profile lookup 优先 project .avm/agents，再读全局 ~/.avm/agents。
   - targets 来自 env.targets、global defaults 或 profile runtime preferred。
3. 实现必要的 MergeEnvironment / project override 占位或最小行为。
4. 添加单测覆盖 profile active、env active、missing profile、project override 优先级。

验收：
- go test ./... 通过
- go vet ./... 通过
- 不改 AgentProfile 主模型中的 workspace_isolation
- Environment 仍不接受 capabilities 或 memory_layers
- 提交并 push：git push -u origin feat/config-resolve
- 最终说明列出修改文件、测试结果、远程 branch、未实现的 Phase 2 行为
```

### Prompt 2：Agent CLI Agent

```text
你是 Agent VM Phase 1 Round 2 的 Agent CLI Agent。

启动步骤：
1. 在主仓库运行：pwd、git branch --show-current、git fetch origin main、git worktree list、git status --short。
2. 目标目录：../agent-vm-agent-cli；目标 branch：feat/agent-cli。
3. 如果目录不存在且 branch 不存在：git worktree add ../agent-vm-agent-cli -b feat/agent-cli origin/main
4. 如果目录不存在但 branch 已存在：git worktree add ../agent-vm-agent-cli feat/agent-cli
5. cd ../agent-vm-agent-cli 后确认 pwd 和 branch。只有在 feat/agent-cli worktree 中才允许修改文件。

写入范围：
- cmd/avm/init*.go
- cmd/avm/agent*.go
- cmd/avm/env*.go
- cmd/avm/*_test.go
- testdata/cli/**

不要修改：
- cmd/avm/root.go
- cmd/avm/commands.go
- internal/adapter/**
- internal/memory/**
- internal/sync/**
- docs/engineering/implementation-plan.md
- go.mod/go.sum，除非绝对必要；如必须修改，最终说明中声明

任务：
1. 实现 avm init：
   - 只写 HOME 下的 ~/.avm/**
   - 创建 config.yaml、agents/default.yaml、envs/default.yaml 和必要目录
   - 支持临时 HOME 单测
2. 实现 avm agent create/list/show：
   - create 写 AgentProfile YAML
   - list/show 输出稳定，适合 tests
   - 支持 --runtime、--scope、--model、--reasoning、--skills、--mcps、--memory
3. 实现 avm env create 的最小版本：
   - 写 Environment YAML
   - Environment 不声明 capabilities 或 memory_layers
4. 命令层只调用 config 包，不写 runtime 配置。

验收：
- go test ./... 通过
- go vet ./... 通过
- 临时 HOME 下 avm init 只写 ~/.avm/**
- avm agent create/list/show 可用
- avm env create 可用
- 提交并 push：git push -u origin feat/agent-cli
- 最终说明列出修改文件、测试结果、远程 branch
```

### Prompt 3：Memory Import Agent

```text
你是 Agent VM Phase 1 Round 2 的 Memory Import Agent。

启动步骤：
1. 在主仓库运行：pwd、git branch --show-current、git fetch origin main、git worktree list、git status --short。
2. 目标目录：../agent-vm-memory-import；目标 branch：feat/memory-import。
3. 如果目录不存在且 branch 不存在：git worktree add ../agent-vm-memory-import -b feat/memory-import origin/main
4. 如果目录不存在但 branch 已存在：git worktree add ../agent-vm-memory-import feat/memory-import
5. cd ../agent-vm-memory-import 后确认 pwd 和 branch。只有在 feat/memory-import worktree 中才允许修改文件。

写入范围：
- internal/memory/**
- cmd/avm/memory*.go
- testdata/memory/**
- fixtures/phase1/** 中 memory import dry-run 相关文件

不要修改：
- cmd/avm/root.go
- cmd/avm/commands.go
- internal/config/models.go
- internal/adapter/**
- internal/sync/**
- cmd/avm/agent*.go
- cmd/avm/env*.go
- docs/engineering/implementation-plan.md
- go.mod/go.sum，除非绝对必要；如必须修改，最终说明中声明

任务：
1. 实现 memory import from file 的 dry-run：
   - 支持 markdown/yaml 输入
   - 生成 MemoryImportPlan、MemoryDiff
   - diff status 只能是 new、changed、conflict、skipped
2. 实现 avm memory import --from <path> --dry-run。
3. dry-run 可以打印 human-readable summary，也可以通过 flag 输出 JSON report；输出必须稳定。
4. dry-run 不写 runtime 文件，也不写正式 ~/.avm/memory/**。
5. 使用已存在的 `memory` command skeleton；不要修改 `cmd/avm/commands.go`。

验收：
- go test ./... 通过
- go vet ./... 通过
- 临时 HOME 下 dry-run 不写 runtime 文件、不写正式 ~/.avm/memory/**
- 输出包含 new | changed | conflict | skipped
- 提交并 push：git push -u origin feat/memory-import
- 最终说明列出修改文件、测试结果、远程 branch、是否需要 Lead wiring
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
