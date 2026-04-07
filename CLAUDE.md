# CLAUDE.md

## Project overview

testmd is a tool for tracking manual/semi-automated tests described in TEST.md files. It watches source files via hashing and tracks which tests need re-verification when code changes.

The canonical specification is in `docs/specification.md`. The architecture is described in `docs/architecture.md`. Read those before making changes.

## Implementation

- **Go** — `cmd/testmd/` + `internal/` (single binary)

## Key principles

- When modifying behavior, update `docs/specification.md` first, then the implementation.
- The specification and docs are **language-agnostic**.

## Architecture rules

- State is always stored in `.testmd.lock` (YAML, never inline in TEST.md)
- The lock file format is deterministic YAML (sorted keys, block-style only)
- Hashing must be deterministic: same files + same content = same hash
- Labels and files are always sorted before hashing or display
- Ignorefile defaults to `.gitignore`, parsed as gitignore format
- Project root is defined by `.testmd.yaml` / `.testmd.yml`
- TEST.md files are auto-discovered under root (no include/frontmatter)
- Test IDs are 18 hex chars: hash6(title) + hash6(labels) + hash6(source_path)
- State writes use atomic temp+rename and flock for concurrency

## Commands

```
testmd [--root PATH] init
testmd [--root PATH] status [--report-md F] [--report-json F]
testmd [--root PATH] resolve <id>
testmd [--root PATH] fail <id> <message>
testmd [--root PATH] get <id>
testmd [--root PATH] gc
testmd [--root PATH] ci [--report-md F] [--report-json F]
```

## Running tests

```
go build -o ./bin/ ./cmd/...
go test ./internal/...
```
