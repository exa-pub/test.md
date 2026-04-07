---
name: testmd
description: Work with testmd — a tool for tracking manual/semi-automated tests described in TEST.md files. Helps understand the workflow, run commands, and resolve tests correctly.
---

# testmd skill

testmd encodes cross-cutting rules — "if you changed X, verify Y" — as executable contracts in `TEST.md` files. When code changes, testmd detects which contracts are affected via file hashing and requires explicit resolution.

## Setup

If the `testmd` command is not available, install it:

```bash
curl -fsSL https://raw.githubusercontent.com/exa-pub/test.md/main/install.sh | sh
```

This auto-detects OS/arch, downloads the latest release binary, verifies checksums, and installs to `/usr/local/bin` (or `~/.local/bin` if no root access). You can pin a version with `TESTMD_VERSION=v1.0.0` or change the install path with `TESTMD_INSTALL_DIR=/path`.

To initialize a new project:

```bash
testmd init   # creates .testmd.yaml in the current directory
```

## Core workflow

1. After changing code, run `testmd status` to see which tests are affected
2. For each affected test, run `testmd get <id>` to read the test description and verification steps
3. Perform the verification (manually or by running described checks)
4. Mark each test: `testmd resolve <id>` (passed) or `testmd fail <id> "reason"` (failed)

In CI, `testmd ci` exits non-zero if any test is not `resolved`.

## Commands

| Command | Purpose |
|---|---|
| `testmd init` | Create `.testmd.yaml` in the current directory |
| `testmd status [--report-md F] [--report-json F]` | Show all tests and their statuses |
| `testmd get <id>` | Show full test details: title, labels, description, watched files, status |
| `testmd resolve <id>` | Mark test as resolved (verified and passing) |
| `testmd fail <id> "msg"` | Mark test as failed with a reason |
| `testmd reset <id>` | Reset test to pending (remove stored state) |
| `testmd ci [--report-md F] [--report-json F]` | CI gate — fails if any test is not resolved |
| `testmd gc` | Remove orphaned state records |

All commands accept `--root PATH` to specify the project root explicitly.

IDs are 18 hex characters (`aabbccddeeff112233`) with three 6-char segments: hash of title, hash of labels, hash of source path. Prefix matching is supported: 6 chars matches all instances of a test, 12 chars matches specific labels, 18 chars is exact match.

## Critical principles

### Use `testmd get`, not file reading

TEST.md files can be very large. **Always use `testmd get <id>` to read test details** instead of reading TEST.md directly. The `get` command:
- Substitutes label variables (`{var}`) with actual values in titles and descriptions
- Shows only the relevant test, not the entire file
- Includes current status, watched files, and labels
- For outdated tests, shows which files changed

### Use `testmd status` for discovery

Don't parse TEST.md to figure out what tests exist or which are affected. Run `testmd status` — it computes file hashes and shows exactly which tests need attention.

### Resolve after verification, not before

Only run `testmd resolve <id>` after you have actually verified that the test passes. Never resolve without performing the actual check described in the test. The resolve command records the current file hashes — resolving prematurely means the test won't trigger again if you make more changes.

### Understand statuses

| Status | Meaning | Action needed |
|---|---|---|
| `pending` | New test, never verified | Perform full verification per description |
| `resolved` | Verified and passing | None |
| `failed` | Verified and failing | Check if previous failure reason is now fixed, then re-verify |
| `outdated` | Was resolved/failed, but watched files changed | Check changed files specifically (use `testmd get` to see which), re-verify |

### Resolve and fail correctly

- For `outdated` tests: focus on the changed files (shown by `testmd get`), not the whole test from scratch
- For `pending` tests: perform the full verification described in the test
- For `failed` tests: check whether the previous failure reason has been addressed
- Fail messages must be specific — include the file, error, and what was tried. Not just "test failed"
- Each test is verified and resolved individually, not in bulk

### State lives in `.testmd.lock`

All state is stored in a single `.testmd.lock` file (YAML) in the project root (next to `.testmd.yaml`). One file for the entire project, regardless of how many TEST.md files exist. Don't modify it manually — always use `testmd resolve` / `testmd fail` / `testmd reset`.

### When you change code

After making code changes:
1. Run `testmd status` to find non-resolved tests
2. For each: run `testmd get <id>` to read what to verify
3. Perform the actual verification (run commands, read code, compare)
4. If the check passes — `testmd resolve <id>`
5. If a problem is found and you can fix it — fix, re-verify, then `testmd resolve <id>`
6. If a problem is found but you cannot fix it — `testmd fail <id> "specific reason"`

Never resolve without actual verification. This is especially important before committing or opening a PR — `testmd ci` will block merges if tests are unresolved.

## Project configuration

testmd is configured via `.testmd.yaml` (or `.testmd.yml`) at the project root. Its presence defines the root — commands search upward from cwd to find it.

```yaml
ignorefile: .gitignore
```

| Field | Default | Description |
|---|---|---|
| `ignorefile` | `.gitignore` | Gitignore-format file. Matching entries are excluded from TEST.md discovery, label discovery, and file hashing. |

An empty `.testmd.yaml` is valid — all fields use defaults.

TEST.md files are **auto-discovered**: all files named exactly `TEST.md` under the project root are found automatically (filtered by ignorefile). No configuration needed.

## Detailed documentation

For full reference on format, commands, and examples, read the files in the `references/` directory next to this skill:

- `references/specification.md` — complete format and behavior spec (read when writing or editing TEST.md files, or when you need to understand edge cases)
- `references/cli.md` — all commands, flags, and output formats (read when you need exact command syntax)
- `references/examples.md` — practical examples of contracts with variables, combinations, and CI (read when writing new contracts)
- `references/architecture.md` — internal design (read only if contributing to testmd itself)

## TEST.md file structure

A TEST.md file contains test definitions — sections starting with `# Title`. **No frontmatter.**

### Test definition

Each test is an h1 heading + a YAML config block + a free-form description:

````markdown
# API returns valid JSON

```yaml
watch: ./src/api/**
```

Send GET /users and verify response is valid JSON with correct schema.
````

Config fields:

| Field | Type | Required | Description |
|---|---|---|---|
| `watch` | string or list | **yes** | Glob pattern(s) for watched files |
| `id` | string | no | Explicit stable id (so renaming the title doesn't change the id) |
| `each` | object | no | Variable sources, cartesian product |
| `combinations` | list | no | Variable sources, union of entries |

`each` and `combinations` are mutually exclusive.

After writing or editing a TEST.md, run `testmd status` to verify parsing and file discovery. If 0 files are matched for a test, the watch pattern is wrong — fix it.

### Watch patterns

```yaml
watch: ./src/auth/**          # recursive glob
watch: ./config/*.yaml        # single-level wildcard
watch:                        # multiple patterns
  - ./src/api/**
  - ./schema/openapi.yaml
watch: ./services/{name}/**   # variable substitution
```

Variables use `{name}` syntax — substituted before globbing. Watch patterns should be specific — avoid overly broad patterns like `**/*`.

### Variables: `each` (cartesian product)

`each` defines variable sources. All are expanded as a cartesian product.

````markdown
# {service} healthcheck

```yaml
each:
  service: ./services/*/
watch: ./services/{service}/**
```

Verify `{service}` responds to GET /health with 200.
````

Source types:
- `./path/*/` — directory names (trailing `/` = dirs only)
- `./path/*.ext` — file names without extension
- `[a, b, c]` — explicit list

### Variables: `combinations` (union)

When cartesian product is not appropriate:

```yaml
combinations:
  - db: [postgres, mysql]
    suite: [full]
  - db: [sqlite]
    suite: [basic]
watch: ./migrations/{db}/**
```

Each entry is a cartesian product internally; entries are combined via union.

### Full example

A complete TEST.md showing multiple features together:

````markdown
# Go implementation works correctly

```yaml
watch:
  - ./internal/**
  - ./cmd/**
```

Go code changed — verify it works correctly:
1. Go tests pass: `go test ./internal/...`
2. Build succeeds: `go build -o ./bin/ ./cmd/...`
3. Run `testmd status` on a sample TEST.md and verify output

# Deploy smoke test for {service}

```yaml
each:
  service: ./services/*/
  env: [prod, staging]
watch:
  - ./services/{service}/**
  - ./deploy/{env}.yaml
```

After deploying `{service}` to `{env}`:
1. Verify the service starts
2. Check /health returns 200
3. Run basic smoke test
````
