<p align="center">
  <img src="assets/avm-hero.svg" alt="Agent VM：一个 Profile，投射到所有 Agent Runtime" width="100%">
</p>

<h1 align="center">Agent VM</h1>

<p align="center">
  <strong>跨运行时管理 AI Agent 配置。</strong>
  <br>
  创建可复用的 Agent 配置，应用到 Codex、Claude Code、OpenCode、Cline 或 Cursor。
</p>

<p align="center">
  <a href="https://github.com/xz1220/Agent-VM/actions/workflows/ci.yml"><img src="https://github.com/xz1220/Agent-VM/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <img src="https://img.shields.io/badge/status-early_preview-0f766e" alt="状态：早期预览">
  <img src="https://img.shields.io/badge/runtime-Codex%20%7C%20Claude%20Code%20%7C%20OpenCode-1d4ed8" alt="目标 Runtime">
  <img src="https://img.shields.io/badge/language-Go-00ADD8" alt="Go">
</p>

<p align="center">
  <a href="README.md">English</a> | 简体中文
</p>

Agent VM，简称 `avm`，是一个面向 AI Coding Agent 的本地配置管理器。它应该让用户只理解几个稳定对象：

- **Agent**：可复用的 Agent 配置，包含 instructions、skills、MCP servers 和 runtime 配置。
- **Environment**：内部 default 上下文。用户在产品路径中不创建、不切换、不管理 Environment。
- **Package**：可安装、可分发的配置包，用来安装 Agent 以及它们引用的能力。
- **Runtime**：Agent 最终生效的工具，例如 Codex、Claude Code、OpenCode、Cline 或 Cursor。

日常使用时，你只需要创建或安装 Agent，然后运行这个 Agent。Skills 会在创建或编辑 Agent 时一起配置；AVM 会自动处理 runtime 检测和单次运行所需的受控配置。

## 日常路径

```bash
avm create
avm run backend-coder
```

目标用户路径应该足够直接：

1. 安装并初始化 AVM。
2. 通过当前 preview 的向导创建 Agent。
3. 运行某个 Agent。

```text
Blank/default / 现有 Package
  -> 创建 Agent
    -> run Agent
      -> runtime-specific managed config
        -> Codex / Claude Code / OpenCode / Cline / Cursor
```

## 用户模块

### 1. 安装、初始化和卸载

这个模块只负责 AVM 在用户机器上的生命周期。

当前 preview 已有：

```bash
curl -fsSL https://raw.githubusercontent.com/xz1220/Agent-VM/main/scripts/install.sh | sh
avm init
avm shell init zsh
```

安装脚本默认把 `avm` 放到 `$HOME/.local/bin`，把 shell integration 写入当前 shell 的 rc 文件，并初始化 `~/.avm`。如果只想安装二进制，可以设置 `AVM_SKIP_INIT=1`。

产品目标：

```bash
avm init
avm doctor
avm uninstall
avm shell install
avm shell uninstall
```

### 2. Agent 配置管理

Agent 是最核心的用户对象。Skills、MCP、instructions 和 runtime 配置都属于 Agent 配置。

当前 preview 已有：

```bash
avm create
avm create backend-coder
avm create --from default --name api-coder

avm agent create backend-coder --runtime codex --skills git,test
avm agent clone backend-coder --name backend-reviewer
avm agent edit backend-reviewer
avm agent rename backend-reviewer reviewer --update-refs
avm agent delete reviewer --force
avm agent list
avm agent show backend-coder
avm agent show backend-coder --runtime codex
```

Agent CRUD 能力：

```bash
avm agent create
avm agent list
avm agent show <name>
avm agent edit <name>
avm agent delete <name>
avm agent clone <name> --name <new-name>
avm agent rename <old-name> <new-name>
```

`avm create` 可以继续作为首次创建向导和快捷入口。它应该允许用户从这些来源创建 Agent：

- 空白/default Agent
- 用户自己创建或已经安装的现有 Package

创建或编辑 Agent 时，skills 和 MCP 选择器应该展示当前全量能力：AVM 管理的能力，以及从支持的 runtime 中实时发现的用户全局安装能力。

### 3. Default Environment

Environment 管理不是当前产品路径里的用户模块。AVM 只保留一个内部 default Environment。

用户不应该创建、切换、删除、导出或安装 Environment。Environment 不负责 runtime 到 Agent 的映射，因为 Agent 自己已经拥有 runtime 配置。

Package 不安装、不导出、不携带 Environment 元数据。

### 4. 运行 Agent

这是用户每天真正会用的执行入口。

```bash
avm run backend-coder
avm run backend-coder --runtime codex
```

`avm run` 是命令级行为，不产生需要用户管理的长期状态，也不需要额外清理。

### 5. Package

Package 用于分发和复用。用户安装 package 后得到 Agent 和它们引用的能力；日常不会直接运行 package。Package 不安装、不导出、不携带 Environment 元数据。

当前 preview 已有：

```bash
avm package list
avm package show reviewer
avm package inspect backend-coder.avm.zip
avm export backend-coder --output backend-coder.avm.zip
avm install backend-coder.avm.zip
```

产品目标：

```bash
avm package list
avm package show <package>
avm package install <package-or-file>
avm package uninstall <package>
avm package export <agent>
avm package inspect <file.avm.zip>
```

## Runtime 支持

AVM 会把选中的 Agent 渲染成各 runtime 的受控配置。

| Runtime | 状态 | 说明 |
| --- | --- | --- |
| Codex | 支持 | 尽量原生映射 profile、model、reasoning |
| Claude Code | 支持 | 映射 agent frontmatter、MCP 和 skills |
| OpenCode | 支持 | 映射 config、agent、skills 和 MCP |
| Cline | 兼容 | 主要渲染为 rules 和 MCP settings |
| Cursor | 兼容 | 保守的 rules/MCP PoC |

Adapter 必须报告字段映射状态：`native`、`rendered_as_instructions`、`ignored` 或 `unsupported`。AVM 不能假装所有 runtime 都支持完全相同的能力。

## 当前 Preview 缺口

产品模块还没有完全收敛。

| 模块 | 当前已有 | 缺口 |
| --- | --- | --- |
| Agent | `create`、`list`、`show`、`edit`、`delete`、`rename`、`clone` | 还需要更完整的首次创建 / package-backed create 流程和交互体验打磨 |
| Environment | 内部 default | 不做面向用户的 Environment 模块 |
| 安装生命周期 | installer、`init`、`shell init` | 缺一等的 doctor/uninstall 命令 |
| Package | list/show/inspect/export/install | install/export 命令归属还需要统一 |
| Skills | `skill list` | 应主要出现在 Agent create/edit 中 |
| Sync | `sync` | 应隐藏在 `run` 背后 |

## 安全模型

AVM 默认应保守：

- installer 初始化和 `avm init` 只写入 `~/.avm`。
- Agent 应成为显式 CRUD 资源，避免隐式覆盖。
- Runtime-native 文件只能通过 adapter 声明的 managed paths 写入。
- Runtime 无法表达的字段必须报告，不能静默丢弃。
- Secrets 应通过环境变量引用，不应以明文导出到 portable profile。

## 开发

```bash
make test
make vet
make fmt
make build
```

主包是 `cmd/avm`。核心包位于 `internal/config`、`internal/adapter`、`internal/sync`、`internal/runtime`、`internal/state` 和 `internal/packageio`。

相关文档：

- [产品需求文档](docs/product/prd.md)
- [技术设计](docs/design/tech-design.md)
- [架构](docs/engineering/architecture.md)
- [数据模型](docs/engineering/data-model.md)
- [实现计划](docs/engineering/implementation-plan.md)
- [验收标准](docs/engineering/acceptance.md)

## License

项目还没有选择开源协议。在 license 被添加之前，代码可以阅读，但不能被默认视作具备广泛复用权利的开源项目。
