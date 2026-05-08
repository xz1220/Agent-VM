# AVM UI

This directory contains the AVM full-screen terminal UI. The Go CLI in
`../cmd/avm/` is **plumbing only** — it does not prompt, render wizards, or
use a TUI library. All interactive UX lives here as a TypeScript application
that shells out to the Go binary.

## Status

Initial Agent CRUD skeleton:

- Agent list + search.
- Agent detail panel.
- Create wizard.
- Edit workspace.
- Delete confirmation.
- Mocked Skills/MCP candidates behind `AvmClient.discoverCapabilities()`.

See [`INTEGRATION-GAPS.md`](./INTEGRATION-GAPS.md) for Go protocol gaps that
are intentionally mocked or worked around during the first UI pass.

## Development

Install dependencies:

```bash
npm install
```

Run against an installed `avm`:

```bash
npm run dev
```

Run against the repo build:

```bash
make -C .. build
npm run dev -- --avm ../bin/avm
```

Build the packaged executable:

```bash
npm run build
node dist/avm-ui.js --help
```

The package exposes the executable name `avm-ui`.

## Contract

Read [`../docs/api/cli-protocol.md`](../docs/api/cli-protocol.md) before
writing any UI code. It documents:

- All command-line flags and arguments the Go CLI accepts.
- The `--json` output shape for every command.
- The structured error envelope (`{"error": {code, message, details}}`)
  and the full list of error codes.
- The two-step Preview → Run flow for `avm run` (multi-runtime and
  drift handling).

## Toolchain

- TypeScript + Node.js.
- Ink + React for the full-screen TUI.
- Zod for CLI protocol parsing.
- Fuse.js for list search/filtering.
- tsup for single-file ESM output.

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
