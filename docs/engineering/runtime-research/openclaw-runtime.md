# OpenClaw 运行时调研

> 目的：用源码事实评估 AVM PRD 的每项能力在 OpenClaw 上能否承接，以及 AVM adapter 必须兜底的行为边界。本文不提产品建议，只记录源码事实与对 PRD 的映射。
>
> 代码范围：`frameworks/openclaw/`（pnpm 10.33.0，node ≥ 22.12）。本文所有 `src/...` 路径相对 `frameworks/openclaw/`。

## 摘要

核心判断：**OpenClaw 能承接 AVM PRD 的主要能力，但几乎没有默认安全兜底，都要 AVM adapter 显式启用或约束。**

- OpenClaw 是 Node/TS CLI，本质是 `@mariozechner/pi-coding-agent` 的壳：CLI → `pi` session runtime → agent loop。
- 默认执行路径是 **Gateway RPC**；`--local` 走 embedded；Gateway 失败会 fallback 到本地。
- 默认 **非隔离**：sandbox=off、workspaceOnly=false、approval=off；host 文件工具可写任意路径，exec 落到 host/gateway。
- 状态默认集中在 `~/.openclaw`：sessions / auth / plugin index / cache / logs / OAuth。workspace 默认 `~/.openclaw/workspace`。
- Skills / Plugins / MCP server **都能执行 host command 或加载进程内代码**。AVM 若不做白名单，等于把 host shell 暴露给模型。
- AVM 可以干净映射的 PRD 字段：Agent identity、instructions、skills、MCP、runtime config、package 安装、run 透明度。需要额外兜底的：能力来源发现、isolation、memory 边界、并发进程状态。

详细证据见 §7 附录。

## 1. OpenClaw 运行时模型

### 1.1 启动链路

```
npm bin `openclaw`
 └─ openclaw.mjs           # node ≥ 22.12, ESM; 检查 dist
    └─ dist/entry.(m)js    # 需要 pnpm build 产物
       └─ src/entry.ts     # profile / container / auth 初始化
          └─ runMainOrRootHelp()
             └─ runCli()   # 加载 .env / normalize proxy
                └─ commander program (agent / gateway / skills / plugins / mcp)
```

证据：`package.json:16`、`openclaw.mjs:8,128,191,197`、`src/entry.ts:45,67,100,175`、`src/cli/run-main.ts:173,180,286`、`src/cli/program/build-program.ts:9`。

### 1.2 Agent 执行的两条路径

```
openclaw agent ...
        │
    ┌───┴────────────────────┐
    │ 默认：Gateway RPC      │  --local：embedded 短进程
    └───┬────────────────────┘
        │
  gateway/server-methods/agent.ts
        │
  agentCommandFromIngress → agentCommand
        │
  runWithModelFallback → runAgentAttempt
        │
    ┌───┴─────────────────┐
    │ runCliAgent         │ runEmbeddedPiAgent
    │ (provider=cli)      │ (provider=llm)
    └───┬─────────────────┘
        │
  runEmbeddedAttempt
    - load workspace / sandbox ctx
    - resolve skills / tools
    - materialize MCP / LSP tools
    - open session manager
    - activeSession.prompt(prompt)
```

Gateway 路径默认，`--local` 显式本地；Gateway 失败会 fallback 到本地。非 loopback bind 要求 auth + token bootstrap / trusted proxy。

证据：`src/cli/program/register.agent.ts:26,44,60`、`src/commands/agent-via-gateway.ts:189,198`、`src/cli/gateway-cli/run.ts:418,561,644`、`src/gateway/server-methods/agent.ts:387,442,1140`、`src/agents/agent-command.ts:901,1154,1174`、`src/agents/pi-embedded-runner/run/attempt.ts:1380,1540,2637`。

### 1.3 状态与工作目录默认值

| 状态 | 默认路径 | 覆盖方式 |
| --- | --- | --- |
| state dir | `~/.openclaw` | `OPENCLAW_STATE_DIR` |
| config | `<stateDir>/openclaw.json` | `OPENCLAW_CONFIG_PATH` |
| agent dir | `<stateDir>/agents/<agentId>/agent` | `OPENCLAW_AGENT_DIR` / `PI_CODING_AGENT_DIR` |
| workspace | `~/.openclaw/workspace`（profile 时加后缀） | `--workspace` / ingress workspace |
| sessions | `<stateDir>/agents/<agentId>/sessions/` | — |
| OAuth 凭据 | `<stateDir>/credentials/oauth.json` | — |
| transcript | session 目录 JSONL，mode 0600 | — |
| tmp | `/tmp/openclaw`（非 symlink, 0700） | uid-owned fallback |

证据：`src/config/paths.ts:55,101,227`、`src/utils.ts:130`、`src/agents/workspace-default.ts:6,12`、`src/agents/agent-paths.ts:6`、`src/config/sessions/paths.ts:10,240`、`src/config/sessions/transcript.ts:19`、`src/infra/tmp-openclaw-dir.ts:5,33,75`。

## 2. PRD 能力在 OpenClaw 上的可行性

按 PRD 的产品要求逐条判断 OpenClaw 是否提供原生承接，以及 AVM adapter 需要做什么。承接度沿用 PRD 的 mapping 语义（native / rendered_as_instructions / ignored / unsupported）。

### 2.1 PRD §2.1 & §4.2：Agent 配置 CRUD

**AVM 要求**：用户管理 Agent（identity、instructions、skills、MCP、runtime config）；skills/MCP 候选列表来自"全量实时发现"，包括 runtime 全局目录和 AVM 未管理的资源。

**OpenClaw 上的事实**：

| AVM 字段 | OpenClaw 承接 | 细节 |
| --- | --- | --- |
| identity (name / desc) | native | agent id 落在 `<stateDir>/agents/<agentId>/agent`；OpenClaw 本身没有 "role" 字段 |
| instructions | native | session startup 把指令构造进 system prompt |
| skills | native | 多 root 发现 + precedence 合并（见 §3.2） |
| MCP servers | native | config `mcp.servers`（stdio / http / sse / streamable-http） |
| runtime config | native | model / provider / approval / sandbox / workspace 是一等字段 |
| role 等 AVM 扩展元数据 | unsupported | OpenClaw 没有对应概念，需 AVM rendered_as_instructions 或 ignored |

**全量发现**：skills loader 合并 7 个 root（bundled / managed / workspace / extraDirs / plugin / personal `~/.agents/skills` / project `<ws>/.agents/skills`），按 precedence 排序。AVM 在 UI 上要做的：
- 把 candidate list 归一成 AVM Agent 字段，保留 origin（AVM-managed / package / runtime-global）。
- 同名冲突必须显式呈现。OpenClaw 是 last-wins，不能直接透传给用户。

证据：`src/agents/skills/workspace.ts:533,561,579,645`、`src/config/types.mcp.ts:1,9`。

**结论**：✅ 可实现。AVM adapter 把 OpenClaw skills/plugins/MCP 发现结果渲染进 Agent create/edit UI 即可。

### 2.2 PRD §2.2 & §4.4：用户选 Agent，不选 runtime

**AVM 要求**：`avm run <agent>`；runtime 由 AVM 解析或让用户确认；不静默选择。

**OpenClaw 上的事实**：
- OpenClaw 自己只有一个 runtime（就是它自己），runtime 多选的问题在 AVM 层解决，不在 OpenClaw 内部。
- OpenClaw 的 session/agentId 是**状态 key**，不是 AVM 意义上的用户可复用配置。AVM 要决定把 AVM Agent 稳定映射到哪个 OpenClaw `agentId`。

**AVM adapter 职责**：
1. 把 `avm run <agent>` 映射为 `openclaw agent ...`，并注入：
   - `OPENCLAW_STATE_DIR` / `OPENCLAW_CONFIG_PATH` / `OPENCLAW_AGENT_DIR`（per-Agent 隔离，见 §3.3）
   - `--workspace` / `--session` / `--provider` / `--model`
2. 选定 Gateway 还是 `--local`，并保持稳定——混用会导致 session/auth 分叉。
3. 把 AVM Agent identity 映射成稳定 `agentId`，让 sessions/auth 可持续复用。

**结论**：✅ 可实现，前提是 AVM 为 OpenClaw 选定一种调用模式。

### 2.3 PRD §2.3 & §4.2：能力边界与 mapping status

**AVM 要求**：每个能力解释来源；adapter 无法表达时明确标注 native / rendered_as_instructions / ignored / unsupported。

**OpenClaw 的字段承接**：

| AVM 能力 | OpenClaw 状态 | 说明 |
| --- | --- | --- |
| system/developer instructions | native | 支持 reference files |
| skills 选择 | native | frontmatter + SKILL.md prompt 注入 |
| MCP server 连接 | native | 多 transport，OAuth |
| 工具权限 allowlist | native | `tool-policy` owner/subagent/leaf filter |
| 审批策略 (approval) | native | full / moderate / strict |
| 写入范围 (workspaceOnly) | native | tool-fs-policy |
| sandbox (docker) | native | `sandbox.mode=docker` |
| env override | native (per-skill) | `skills/env-overrides` |
| 启动/安装脚本 | native | skill frontmatter `install` |
| role / team / tenant 等 AVM 扩展字段 | unsupported | 只能 rendered_as_instructions 或 ignored |

mapping status 必须由 AVM 在 run preview 中展示，这是 OpenClaw 自己不会做的事。

证据：`src/agents/tool-policy.ts:22,54`、`src/agents/pi-tools.policy.ts:34,49`、`src/agents/exec-defaults.ts:43,91,124`、`src/agents/tool-fs-policy.ts:11,29`、`src/agents/sandbox/config.ts:220`、`src/agents/skills/env-overrides.ts:24`、`src/agents/skills/frontmatter.ts:112`。

**结论**：✅ 主要能力有 native 映射，少数 AVM 扩展字段要 rendered_as_instructions / ignored。

### 2.4 PRD §2.4 & §4.5：可信复用（Package）

**AVM 要求**：Package 是分发单元；安装前展示写入内容；冲突处理清楚；export 可选包含 skills/MCP；不把 runtime 原生文件伪装成 AVM 对象。

**OpenClaw 上的事实**：
- OpenClaw 有 **plugin** 概念（manifest + runtime registry），但 plugin ≠ AVM Package。plugin manifest 可以声明 skills / mcpServers / commands / agents / hooks。
- Plugin 安装：archive/package 下载 → security scan → 索引到 `<stateDir>/plugins/installs.json`。可被 `--dangerously-force-unsafe-install` 绕过。
- Skill 可以通过 ClawHub slug 安装到 `<workspace>/skills/<slug>`。
- Skill install recipe 支持 `brew / node / go / uv / download`，download 限定 http/https，落到 per-skill tools root。

**AVM adapter 职责**：
- AVM Package 不直接等同于 OpenClaw plugin：AVM Package 跨 runtime，plugin 只在 OpenClaw 生效。
- 安装 AVM Package 时，落到 OpenClaw 的部分走 OpenClaw 的 skill/mcp/plugin 安装路径；AVM 记录来源。
- export 时不要把 OpenClaw 管理的 skills/plugins 当作 AVM 所有物——这是 PRD §2.4 明确禁止的"伪装"行为。

证据：`src/plugins/loader.ts:2603,2788`、`src/plugins/install.ts:39,769`、`src/plugins/installed-plugin-index-store-path.ts:4,20`、`src/cli/plugins-cli.ts:760`、`src/agents/skills-clawhub.ts:18,122,144`、`src/agents/skills/frontmatter.ts:93,112`、`src/agents/skills-install.ts:457,503,549`、`src/agents/skills-install-download.ts:31`。

**结论**：✅ 可实现，关键是 AVM 明确区分 "AVM 分发" vs "OpenClaw 原生分发"。

### 2.5 PRD §2.5 & §4.4：运行透明

**AVM 要求**：每次 run 能解释 Agent、runtime、managed paths、memory boundary、warnings；adapter mapping 在 preview/run 输出展示。

**OpenClaw 暴露的运行元数据**：
- session key / agentId / workspace：session store 可读。
- transcript：`<stateDir>/agents/<agentId>/sessions/*.jsonl`，0600。
- skill snapshot：session 创建/stale 时存 snapshot。
- mapping diagnostics：OpenClaw 对 unsupported MCP transport / bundle MCP 会产生 warning。

**AVM adapter 职责**：
- Run preview 显示 agentId、workspace、agentDir、session key、启用 skills/MCP 列表、mapping status。
- 把 OpenClaw 的 warning（MCP transport / plugin compat / skill loader truncation）翻译为 AVM warning 暴露给用户。
- AVM 内部同步（config 写入、skill 拷贝）对用户透明。

证据：`src/config/sessions/paths.ts:10,240`、`src/config/sessions/transcript.ts:19`、`src/config/sessions/store.ts:232,411`、`src/agents/agent-command.ts:591,631`、`src/agents/skills/workspace.ts:814`、`src/agents/mcp-transport-config.ts:95`、`src/plugins/loader.ts:2635`。

**结论**：✅ 可实现，数据全在 OpenClaw 侧可读。

### 2.6 PRD §2.6 & §4.6：不管理运行记忆，只隔离

**AVM 要求**：不提供 memory CRUD；保证同一 Agent/runtime 的 memory 在稳定 boundary 里，不跨 Agent 串。

**OpenClaw 上的事实**：
- Workspace 内 memory：`<workspace>/MEMORY.md`、`<workspace>/memory/**/*.md`、event log `memory/.dreams/events.jsonl`。canonical memory file 禁 symlink。
- Memory backend 可额外加载 agent workspace / config extra paths / sessions。
- **Memory plugin state 是进程级全局对象**——这是最大的隐患。

**AVM adapter 职责**：
- 用 per-Agent workspace 就能天然把 memory 隔离开。
- 不要把 memory 拷到 AVM 侧作为 "AVM memory"。
- **不要复用同一个 OpenClaw 进程跑不同 AVM Agent**。推荐每个 AVM run 起独立 OpenClaw 进程（见 §3.1）。

证据：`src/memory/root-memory-files.ts:4,33`、`src/memory-host-sdk/host/backend-config.ts:329,350`、`src/memory-host-sdk/events.ts:5`、`src/plugins/memory-state.ts:159,202`。

**结论**：✅ 可实现，但约束是"不要跨 Agent 共享 OpenClaw 进程"。

## 3. 适配关键数据边界

### 3.1 CLI / Gateway 调用模式对比

| 模式 | 优点 | 缺点 | 适合 |
| --- | --- | --- | --- |
| `--local` embedded | 进程自持，状态明确；无网络 | 每次冷启；无法跨进程共享 session | AVM 每次 run 单独启 |
| Gateway 常驻 | 可复用 session；支持 RPC | auth 管理复杂；plugin/memory 进程级状态会跨 agent 串 | 多 client 并发场景 |

两条路径都被 OpenClaw 支持；AVM 选哪条是 PRD 未决问题（§5）。

### 3.2 Skills 发现与安装

**发现 root 优先级**（升序，后者覆盖前者）：

```
extraDirs < bundled < managed < personal (~/.agents/skills) < project (<ws>/.agents/skills) < workspace
```

**安装可执行动作**：
- `brew / node / go / uv / download`（HTTP/HTTPS 限定）
- 写入 per-skill tools root，禁止跳出
- 进程级 env override，带屏蔽危险变量清单

**AVM 注意**：
- skill install 可执行 host command。AVM 必须把 install 视为高风险动作，做用户确认。
- personal/project skills 会被 OpenClaw 自动发现，AVM 不能假设 "只有 AVM 管理的 skills 生效"。

证据：`src/agents/skills/workspace.ts:533,561,579`、`src/agents/skills/frontmatter.ts:93,112`、`src/agents/skills-install.ts:457,503,549`、`src/agents/skills-install-download.ts:31`、`src/agents/skills/tools-dir.ts:7`、`src/agents/skills/env-overrides.ts:24,37,83`。

### 3.3 Per-Agent 隔离的最小注入集

让 OpenClaw 在每次 `avm run <agent>` 使用独立状态：

```bash
OPENCLAW_STATE_DIR=~/.avm/runtimes/openclaw/<avm-agent-id>
OPENCLAW_CONFIG_PATH=~/.avm/runtimes/openclaw/<avm-agent-id>/openclaw.json
OPENCLAW_AGENT_DIR=~/.avm/runtimes/openclaw/<avm-agent-id>/agent
# workspace 通过 CLI flag
openclaw agent --workspace <avm-workspace> --session <avm-session-key>
```

这样 sessions / auth / plugin index / cache / logs / OAuth 都落到 per-Agent 目录，不污染 `~/.openclaw`。

### 3.4 Plugin / MCP 边界

- **Plugin**：AVM 控制 `plugin roots`（config extraRoots）和 enable list。默认发现 bundled / global `<configDir>/extensions` / workspace `<workspace>/.openclaw/extensions` / config 额外路径。
- **MCP client**：AVM 下发的 server 列表进 `mcp.servers`。stdio server 会 spawn configured command——视同授予 host 权限。
- **MCP server (OpenClaw 作为 server)**：暴露 built-in / plugin tools。若 AVM 需要这种形态，必须 audit 可暴露 tools 清单。

证据：`src/plugins/roots.ts:16,29`、`src/plugins/loader.ts:2131,2311,2788`、`src/agents/mcp-stdio-transport.ts:27`、`src/agents/mcp-http.ts:19`、`src/mcp/openclaw-tools-serve.ts:2`、`src/mcp/plugin-tools-serve.ts:2`、`src/mcp/plugin-tools-handlers.ts:21,53`。

## 4. 默认不安全项清单（AVM 必须兜底）

| OpenClaw 默认 | 风险 | AVM 兜底策略 |
| --- | --- | --- |
| `sandbox.mode=off` | host 级执行无 OS 隔离 | 依赖可用时默认 `sandbox.mode=docker`；否则 exec 降级到 preview-only |
| `workspaceOnly=false` | 文件工具可写 host 任意路径 | 默认注入 `workspaceOnly=true` |
| `approval=off` | 无交互批准即执行 | 注入 `approval=moderate` 或更高 |
| exec host=auto | 可能走 gateway/host shell | 强制 `host: sandbox` 或白名单 |
| 默认 workspace `~/.openclaw/workspace` | 跨 Agent 共享 | 每 Agent 独立 workspace |
| MCP stdio 可 spawn 任意 command | 等同给 model host shell | 白名单 + per-Agent env 注入 |
| plugin/skill install 可跑 host command | 同上 | install 动作需用户显式确认 |
| 全局 state in `~/.openclaw` | 跨 Agent 数据串扰 | `OPENCLAW_STATE_DIR` per-Agent |
| memory plugin state 进程级 | 同一进程跨 Agent 串 memory | per-run 独立 OpenClaw 进程 |

证据：`src/agents/sandbox/config.ts:220`、`src/agents/tool-fs-policy.ts:11`、`src/agents/pi-tools.read.ts:751,757,785`、`src/agents/exec-defaults.ts:91,124`、`src/agents/bash-tools.exec-runtime.ts:681,708`、`src/agents/mcp-stdio-transport.ts:27`、`src/agents/skills-install.ts:457`、`src/plugins/loader.ts:2788`、`src/plugins/memory-state.ts:202`。

## 5. 未决问题（PRD 需要决策）

1. **调用模式**：AVM 使用 OpenClaw Gateway 常驻服务还是每次 `--local`？源码两条路径都支持，但 session 生命周期、auth surface、并发模型不同。
2. **Docker sandbox**：AVM 自己提供外层 sandbox（VM / container），还是启用 OpenClaw 内置 Docker sandbox？后者需要部署前置检查 Docker。
3. **Skill / Plugin / MCP 白名单**：AVM 支持 OpenClaw 生态全量，还是只允许白名单？PRD §2.4 要求来源可追踪，倾向白名单 + 明示来源。
4. **遗留 `~/.openclaw` 数据迁移**：已有用户的 `~/.openclaw` 是否迁移到 per-Agent 目录？源码有 legacy/compat 路径。

## 6. 验证状态

- ✅ `node --version` → v22.22.0
- ✅ `pnpm --dir frameworks/openclaw --version` → 10.33.0
- ❌ `node frameworks/openclaw/openclaw.mjs --help` —— 缺 `dist/entry.(m)js`，对应 `openclaw.mjs:191,197` 的已知分支。pnpm build 后可再验证。

## 7. 源码证据索引

按主题归类，方便交叉引用；文中已在关键结论处内联。

### 入口与命令

- `package.json:16` —— bin 指向 `openclaw.mjs`
- `openclaw.mjs:8,128,191,197` —— node 版本检查 / 帮助快速路径 / dist 加载
- `src/entry.ts:45,67,100,175` —— profile / container / auth 初始化
- `src/cli/run-main.ts:173,180,286` —— `.env` 加载 / proxy normalize
- `src/cli/program/build-program.ts:9,16` —— 子命令注册
- `src/cli/program/register.agent.ts:26,44,60` —— agent 子命令

### Gateway / Agent dispatch

- `src/commands/agent-via-gateway.ts:189,198` —— default gateway + local fallback
- `src/cli/gateway-cli/run.ts:418,561,644` —— bind / auth
- `src/gateway/server-methods.ts:73,110` —— core handlers / 授权
- `src/gateway/server-methods/agent.ts:387,442,768,1054,1140` —— agent handler
- `src/agents/agent-command.ts:251,295,356,591,631,679,761,901,1154,1174` —— prepare / workspace / model / ingress

### Embedded runtime

- `src/agents/pi-embedded-runner/run.ts:239,253,287,317,421`
- `src/agents/pi-embedded-runner/run/attempt.ts:569,612,684,854,905,1037,1224,1380,1540,2637`

### Sandbox / exec

- `src/agents/sandbox/config.ts:101,220,240` —— mode / backend / workspace-access
- `src/agents/sandbox/docker.ts:391` —— docker 硬化
- `src/agents/sandbox/validate-sandbox-security.ts:20,39,359` —— mount 校验
- `src/agents/exec-defaults.ts:43,91,124` —— host / approval / security
- `src/agents/bash-tools.exec-host-gateway.ts:95,173` —— allowlist / approval
- `src/agents/bash-tools.exec.ts:1536`、`src/agents/bash-tools.exec-runtime.ts:681,708` —— runtime
- `src/infra/exec-approvals.ts:169` —— approval

### 文件工具

- `src/agents/tool-fs-policy.ts:11,29,36` —— workspaceOnly 默认 false
- `src/agents/pi-tools.ts:428,456,543` —— read / apply_patch
- `src/agents/pi-tools.read.ts:751,757,785` —— host write/edit 允许任意路径
- `src/agents/workspace.ts:27,54,467` —— workspace init
- `src/agents/workspace-default.ts:6,12` —— 默认 `~/.openclaw/workspace`
- `src/agents/workspace-run.ts:74,105` —— `--workspace` 解析
- `src/infra/boundary-file-read.ts:69`、`src/infra/fs-safe.ts:207` —— bounded read

### Skills

- `src/agents/skills/workspace.ts:268,533,561,579,645,814,910` —— 发现 / precedence / prompt / snapshot
- `src/agents/skills/local-loader.ts:21,44` —— SKILL.md / frontmatter
- `src/agents/skills/config.ts:26,58,73` —— entries config
- `src/agents/skills/frontmatter.ts:93,112` —— install spec
- `src/agents/skills-install.ts:457,503,549` —— install 动作
- `src/agents/skills-install-download.ts:31` —— download root guard
- `src/agents/skills/tools-dir.ts:7` —— per-skill tools root
- `src/agents/skills/env-overrides.ts:24,37,83` —— env override
- `src/agents/skills-clawhub.ts:18,122,144` —— ClawHub install
- `src/cli/skills-cli.ts:52,93` —— CLI

### Plugins

- `src/plugins/roots.ts:16,29` —— bundle / global / workspace / config roots
- `src/plugins/manifest-registry.ts:91` —— manifest precedence
- `src/plugins/loader.ts:2131,2311,2603,2635,2788,3148` —— 发现 / activate / runtime registry / bundle MCP warn
- `src/plugins/registry.ts:372,404,1438` —— 可注册 tool / hook / provider / gateway method
- `src/plugins/install.ts:39,769`、`src/cli/plugins-cli.ts:760` —— 安装 / security override
- `src/plugins/installed-plugin-index-store-path.ts:4,20` —— install index
- `src/plugins/memory-state.ts:159,202` —— 进程级 memory state

### MCP

- `src/config/types.mcp.ts:1,9` —— 配置
- `src/config/mcp-config.ts:29,59,105` —— CRUD
- `src/agents/bundle-mcp-config.ts:49` —— bundle 合并
- `src/agents/mcp-transport-config.ts:95` —— transport 解析
- `src/agents/mcp-stdio.ts:14`、`src/agents/mcp-stdio-transport.ts:27` —— stdio spawn
- `src/agents/mcp-http.ts:19` —— http
- `src/agents/pi-bundle-mcp-runtime.ts:181,228,351,483,580` —— runtime
- `src/agents/pi-bundle-mcp-materialize.ts:64` —— tools materialize
- `src/mcp/openclaw-tools-serve.ts:2`、`src/mcp/plugin-tools-serve.ts:2`、`src/mcp/plugin-tools-handlers.ts:21,53`、`src/mcp/tools-stdio-server.ts:9,24` —— OpenClaw 作为 MCP server

### 状态与存储

- `src/config/paths.ts:55,101,227` —— state dir / config / oauth
- `src/utils.ts:130` —— config dir 解析
- `src/agents/agent-paths.ts:6` —— agent dir
- `src/config/sessions/paths.ts:10,35,62,176,240` —— session store 路径
- `src/config/sessions/store.ts:232,411,493` —— atomic write / rotate
- `src/config/sessions/transcript.ts:19` —— transcript header
- `src/agents/auth-profiles/path-resolve.ts:12`、`src/agents/auth-profiles/store.ts:73,197`、`src/agents/auth-profiles/persisted.ts:95` —— auth
- `src/memory/root-memory-files.ts:4,33` —— MEMORY.md canonical
- `src/memory-host-sdk/host/backend-config.ts:329,350` —— memory backend
- `src/memory-host-sdk/events.ts:5` —— event log
- `src/logging/logger.ts:42,336,401,474` —— logger
- `src/infra/tmp-openclaw-dir.ts:5,33,75` —— tmp dir 安全性
- `src/agents/cache-trace.ts:81`、`src/agents/bootstrap-cache.ts:3`、`src/agents/pi-embedded-runner/openrouter-model-capabilities.ts:1,82,120` —— 缓存

### Tool policy

- `src/agents/tool-policy.ts:22,54` —— owner-only / non-owner filter
- `src/agents/pi-tools.policy.ts:34,49` —— subagent / leaf session deny
