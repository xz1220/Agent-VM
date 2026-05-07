# AVM UI (placeholder)

This directory is reserved for the AVM interactive frontend. The Go CLI
in `../cmd/avm/` is **plumbing only** — it does not prompt, render
wizards, or use a TUI library. All interactive UX lives here as a
TS/JS application that shells out to the Go binary.

## Status

**Not yet started.** This README exists so the Go side can develop
against a stable contract while the UI is built independently.

## Contract

Read [`../docs/api/cli-protocol.md`](../docs/api/cli-protocol.md) before
writing any UI code. It documents:

- All command-line flags and arguments the Go CLI accepts.
- The `--json` output shape for every command.
- The structured error envelope (`{"error": {code, message, details}}`)
  and the full list of error codes.
- The two-step Preview → Run flow for `avm run` (multi-runtime and
  drift handling).

## Suggested toolchain (pick when you start)

The UI directory is intentionally empty — the agent that builds the UI
should choose its own stack. Reasonable starting points:

- **Terminal UI**: [Ink](https://github.com/vadimdemedes/ink) (React for
  terminal) + [inquirer](https://github.com/SBoudrias/Inquirer.js).
- **Web UI**: Vite + React + TypeScript, calling Go via spawn or HTTP
  bridge.
- **VS Code extension**: TypeScript + the VS Code extension API.

## Calling the Go CLI

The Go binary is at `../bin/avm` after `make build`. From the UI:

```ts
import { spawn } from "node:child_process";

const child = spawn("avm", ["agent", "list", "--json"], {
  stdio: ["ignore", "pipe", "pipe"],
});
// Parse stdout as JSON. On failure, parse stdout's {"error": ...}
// envelope and read child.exitCode.
```

For `avm run <agent>`, the Go process inherits stdin/stdout/stderr and
hands the TTY to the runtime (codex/claude/opencode); the UI should
spawn it with `stdio: "inherit"` and just wait for exit.

## Layering invariant

The UI consumes the CLI's `--json` contract. It does **not** import any
Go code or reach into AVM home directly. If a feature requires UI ←→ Go
coordination beyond what CLI commands expose, add a new command on the
Go side first, document it in `cli-protocol.md`, then consume from
here.
