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

## Core workflow

1. After changing code, run `testmd status` to see which tests are affected
2. For each affected test, run `testmd get <id>` to read the test description and verification steps
3. Perform the verification (manually or by running described checks)
4. Mark each test: `testmd resolve <id>` (passed) or `testmd fail <id> "reason"` (failed)

In CI, `testmd ci` exits non-zero if any test is not `resolved`.

## Commands

| Command | Purpose |
|---|---|
| `testmd status` | Show all tests and their statuses |
| `testmd get <id>` | Show full test details: title, labels, description, watched files, status |
| `testmd resolve <id>` | Mark test as resolved (verified and passing) |
| `testmd fail <id> "msg"` | Mark test as failed with a reason |
| `testmd ci` | CI gate — fails if any test is not resolved |
| `testmd gc` | Remove orphaned state records |

IDs support abbreviations: full `aabbcc-ddeeff`, first-part `aabbcc`, or unambiguous prefix `aab`.

## Critical principles

### Use `testmd get`, not file reading

TEST.md files can be very large and contain embedded state blocks with hashes. **Always use `testmd get <id>` to read test details** instead of reading TEST.md directly. The `get` command:
- Substitutes label variables (`$var`) with actual values in titles and descriptions
- Shows only the relevant test, not the entire file
- Includes current status, watched files, and labels
- Is far more efficient than parsing markdown yourself

### Use `testmd status` for discovery

Don't parse TEST.md to figure out what tests exist or which are affected. Run `testmd status` — it computes file hashes and shows exactly which tests need attention.

### Resolve after verification, not before

Only run `testmd resolve <id>` after you have actually verified that the test passes. The resolve command records the current file hashes — resolving prematurely means the test won't trigger again if you make more changes.

### Understand statuses

| Status | Meaning | Action needed |
|---|---|---|
| `pending` | New test, never verified | Verify and resolve/fail |
| `resolved` | Verified and passing | None |
| `failed` | Verified and failing | Fix the issue, then resolve |
| `outdated` | Was resolved/failed, but watched files changed | Re-verify and resolve/fail |

### State lives in TEST.md

All state is stored inline in TEST.md files as HTML comments (invisible in rendered markdown). There are no external state files. Don't modify state blocks manually — always use `testmd resolve` / `testmd fail`.

### When you change code

After making code changes:
1. Run `testmd status` to check if any tests became `outdated` or are `pending`
2. For each non-resolved test, run `testmd get <id>` to understand what to verify
3. Perform the verification steps described in the test
4. Resolve or fail each test

This is especially important before committing or opening a PR — `testmd ci` will block merges if tests are unresolved.

## TEST.md file structure

A TEST.md file has three optional sections, in order:

1. **Frontmatter** — YAML between `---` delimiters at the very beginning
2. **Test definitions** — sections starting with `# Title`
3. **State block** — auto-managed by testmd, at the end of the file (do not edit manually)

### Frontmatter

Optional YAML block at the top of the file:

````markdown
---
include: [tests/integration/TEST.md, tests/e2e/TEST.md]
ignorefile: .testmdignore
---
````

| Field | Default | Description |
|---|---|---|
| `include` | `[]` | Paths to other TEST.md files (relative to current). Tests are merged; each file stores its own state. |
| `ignorefile` | `.gitignore` | Gitignore-format file. Matching entries are excluded from label discovery and file hashing. |

### Test definition

Each test is an h1 heading + a YAML config block + a free-form description:

````markdown
# API returns valid JSON

```yaml
on_change: ./src/api/**
```

Send GET /users and verify response is valid JSON with correct schema.
````

Config fields:

| Field | Type | Required | Description |
|---|---|---|---|
| `on_change` | string or list | **yes** | Glob pattern(s) for watched files |
| `id` | string | no | Explicit stable id (so renaming the title doesn't change the id) |
| `matrix` | list | no | Label combinations (see below) |

### Patterns in `on_change`

```yaml
on_change: ./src/auth/**          # recursive glob
on_change: ./config/*.yaml        # single-level wildcard
on_change:                        # multiple patterns
  - ./src/api/**
  - ./schema/openapi.yaml
on_change: ./services/$name/**    # label variable — auto-discovers from filesystem
```

Special segments:
- `$identifier` — label variable, each unique directory at that position produces a test instance
- `*` — matches any name at one path level
- `**` — matches any sub-path, zero or more levels

### Labels: auto-discovery from filesystem

When a pattern contains `$var` and no `matrix` is specified, testmd discovers values from the filesystem:

````markdown
# $service healthcheck

```yaml
on_change: ./services/$service/**
```

Verify `$service` responds to GET /health with 200.
````

With filesystem `services/{auth,billing,gateway}/`, this creates three test instances:
```
$ testmd status
$service healthcheck
  … ed4be2-fe0c31  service=auth     pending
  … ed4be2-c9054d  service=billing  pending
  … ed4be2-ab1234  service=gateway  pending
```

Adding a new directory (e.g. `services/payments/`) automatically creates a new pending test.

### Labels: explicit matrix

Matrix decouples label generation from file structure. Useful for fixed combinations (environments, versions) or mixing auto-discovery with explicit values.

**Const — explicit values (cartesian product):**

````markdown
# API compatibility

```yaml
on_change: ./api/**
matrix:
  - const:
      version: [v1, v2, v3]
```

Verify the API contract for version `$version`.
````

**Match — discover from filesystem:**

```yaml
matrix:
  - match:
      - ./services/$name/
```

**Match + const — cartesian product of discovered and explicit:**

````markdown
# Deploy smoke test

```yaml
on_change:
  - ./services/$service/**
  - ./deploy/$env.yaml
matrix:
  - match:
      - ./services/$service/
    const:
      env: [prod, staging]
```

After deploying `$service` to `$env`:
1. Verify the service starts
2. Check /health returns 200
````

**Multiple entries — union (not cartesian):**

```yaml
matrix:
  - const: {db: [postgres, mysql]}
  - const: {db: [sqlite]}
# produces: db=postgres, db=mysql, db=sqlite
```

### Includes: splitting tests across files

````markdown
---
include: [tests/integration/TEST.md, tests/e2e/TEST.md]
---

# Unit test sanity

```yaml
on_change: ./src/**
```

Run `make test` and verify all unit tests pass.
````

`testmd status` shows tests from all included files. Each file stores its own state independently. Nested includes (an included file including another) are not supported.

### Full example

A complete TEST.md showing multiple features together:

````markdown
---
include: [src/testmd/TEST.md]
ignorefile: .gitignore
---

# Go implementation matches Python

```yaml
on_change:
  - ./internal/**
  - ./cmd/**
```

Go code changed — verify it still matches the Python reference:
1. Run both implementations on the same TEST.md
2. Compare report JSON output
3. Go tests pass: `go test ./internal/...`

# Deploy smoke test for $service

```yaml
on_change:
  - ./services/$service/**
  - ./deploy/$env.yaml
matrix:
  - match:
      - ./services/$service/
    const:
      env: [prod, staging]
```

After deploying `$service` to `$env`:
1. Verify the service starts
2. Check /health returns 200
3. Run basic smoke test
````
