# AVM UI Integration Gaps

This file records UI-facing protocol gaps found while scaffolding `avm-ui`.
The TUI keeps these behind `AvmClient` mocks so the UI can be built before the
Go protocol is complete.

## Capability discovery

Application code already has `CapabilityService.Discover`, but the Cobra tree
does not expose a user-facing JSON command for it.

Needed shape:

```bash
avm capability discover --kind skill --kind mcp --runtime codex --json
```

Expected payload: `[]CapabilityCandidate`, using the model already documented
in `docs/api/cli-protocol.md`.

## Runtime-global capability import

`CapabilityCandidate` can point at runtime-global skills/MCP, but `Agent`
stores only AVM capability IDs. The UI needs a protocol to import a selected
runtime-global candidate into the AVM capability store before `agent create` or
`agent edit` persists the reference.

Possible shape:

```bash
avm capability import --runtime codex --kind skill --name repo-map --json
```

Expected payload: `CapabilityRecord` or an import result containing the new ID.

## Runtime requirement

`avm agent create --help` says at least one `--runtime` is required. The UI
enforces that, but the Go service/model currently allow an Agent with no
runtime. The backend should enforce the same invariant so scripts and UI behave
the same way.

## Delete JSON success

`docs/api/cli-protocol.md` says `agent delete` returns `null` in JSON mode.
The current command prints a human success line even with `--json`. The UI uses
a void command path for now, but this should be aligned before strict protocol
tests are added for the TUI.
