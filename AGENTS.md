# Repository Guidelines

## Project Structure & Module Organization

Agent VM is a Go CLI project. The executable entrypoint and Cobra commands live in `cmd/avm/`. Shared packages are under `internal/`, grouped by concern: `config`, `adapter`, `sync`, `memory`, `runtime`, `state`, and `packageio`. Long-form documentation lives in `docs/`, with engineering details in `docs/engineering/`. Use `fixtures/` for realistic sample AVM homes and runtime layouts, and `testdata/` for stable inputs and expected outputs. Visual README assets are in `assets/`; developer scripts are in `scripts/`.

## Build, Test, and Development Commands

- `make build`: builds the CLI to `bin/avm`.
- `make test`: runs `go test ./...`.
- `make fmt`: runs `gofmt -w ./cmd ./internal`.
- `make vet`: runs `go vet ./...`.
- `go run ./cmd/avm --help`: runs the CLI locally without installing it.
- `make clean`: removes generated build and coverage artifacts.

CI runs `go build ./...`, `go vet ./...`, `test -z "$(gofmt -l .)"`, and `go test ./...`.

## Coding Style & Naming Conventions

Target Go 1.23. Keep Go code `gofmt`-formatted and organized around small packages with clear ownership. Use idiomatic Go names: exported identifiers use `PascalCase`, unexported identifiers use `camelCase`, and test helpers stay unexported unless needed across files. CLI behavior belongs in `cmd/avm/`; reusable logic belongs in `internal/`. Avoid serializing secrets or machine-local paths.

## Testing Guidelines

Tests use the standard Go `testing` package and live beside implementation files as `*_test.go`. Prefer table-driven tests for validation, parsing, rendering, and CLI behavior. Put reusable golden inputs in `testdata/`; put human-readable scenario fixtures in `fixtures/`. Add tests for behavior changes, especially adapter mapping, config resolution, activation, sync, import/export, and error handling.

## Commit & Pull Request Guidelines

Recent history uses short imperative subjects, often with prefixes such as `fix:`, `feat:`, and `revert:`. Keep commits focused on one behavior or docs change.

PRs should include a summary, testing results, and any notes about docs, secrets, or machine-local paths. Follow `.github/pull_request_template.md`: check `go test ./...`, `go vet ./...`, update docs when behavior changes, and separate implemented behavior from planned behavior. Avoid broad rewrites or unrelated refactors unless already scoped.

## Security & Configuration Notes

Treat `~/.avm` as the source of truth. `avm init` must not modify runtime config files, and runtime writes should go through adapter-owned managed paths. Reference secrets rather than copying them into portable profiles.
