# testmd — Specification

testmd encodes cross-cutting rules — "if you changed X, verify Y" — as executable contracts in `TEST.md` files. Every codebase has implicit knowledge: rename an API field and the docs break, change a schema and the migration needs updating. testmd makes these rules explicit, trackable, and enforceable in CI.

This is especially valuable when code is written by AI agents, which have no way of knowing a project's unwritten rules. An agent runs `testmd ci`, sees which contracts its changes have broken, and either fixes the issues or reports what it cannot resolve.

## Core loop

1. Developer or agent changes code
2. `testmd status` / `testmd ci` shows which contracts are affected (file hashes changed)
3. The author verifies each flagged area and runs `testmd resolve <id>` or `testmd fail <id> <message>`
4. CI calls `testmd ci` and fails if there are unresolved tests

## Project configuration

testmd is configured via a `.testmd.yaml` (or `.testmd.yml`) file at the project root. This file is **required** — its presence defines the project root.

```yaml
ignorefile: .gitignore
```

| Field        | Type   | Default      | Description                                                  |
|--------------|--------|--------------|--------------------------------------------------------------|
| `ignorefile` | string | `.gitignore` | Path to a gitignore-format file (relative to root). Matching entries are excluded from TEST.md discovery, label discovery, and file hashing. |

An empty `.testmd.yaml` file is valid — all fields use their defaults.

### Root discovery

Commands search upward from the current working directory for `.testmd.yaml` or `.testmd.yml`. The directory containing the config file is the project root. If neither is found, testmd exits with an error.

### TEST.md discovery

All files named `TEST.md` under the project root are automatically discovered:

1. Walk the directory tree from root
2. Skip directories excluded by the ignorefile (e.g. `node_modules`, `.git`)
3. Collect all files named exactly `TEST.md`
4. Sort by relative path for determinism

## TEST.md format

A TEST.md file contains **test definitions** — sections starting with `# Title`. No frontmatter.

State is stored separately in `.testmd.lock` (see [State file](#state-file)).

### Test definition

Each test starts with a level-1 heading (`# Title`) followed by a YAML config block and a description:

````markdown
# OAuth login flow on {env}

```yaml
id: oauth
each:
  provider: ./services/*/
  env: [prod, staging]
watch:
  - ./services/{provider}/**
  - ./deploy/{env}.yaml
```

Verify that OAuth works for each provider:
1. Navigate to /login
2. Click "Sign in with {provider}"
3. Verify redirect and session creation
````

### Test config fields

| Field          | Type           | Required | Default | Description                                         |
|----------------|----------------|----------|---------|-----------------------------------------------------|
| `watch`        | string or list | **yes**  | —       | Glob pattern(s) for watched files                   |
| `id`           | string         | no       | —       | Explicit first part of the test id                  |
| `each`         | object         | no       | —       | Variable sources for cartesian product (see [Each](#each)) |
| `combinations` | list           | no       | —       | Variable sources for union of entries (see [Combinations](#combinations)) |

`each` and `combinations` are mutually exclusive — using both is an error.

## State file

State is stored in a single `.testmd.lock` file in the project root (next to `.testmd.yaml`). The lock file uses YAML format with deterministic formatting for merge-friendly git diffs:

```yaml
version: 1
tests:
  abc123def456789abc:
    title: OAuth login flow
    source: services/auth/TEST.md
    status: resolved
    content_hash: "a1b2c3..."
    resolved_at: "2026-04-06T12:00:00Z"
    failed_at: null
    message: null
    labels:
      env: prod
      provider: google
    files:
      services/auth/main.go: "d4e5f6..."
```

Formatting rules (for git merge friendliness):
- Test entries sorted by ID (lexicographic)
- Scalar fields before nested fields (labels, files)
- Labels sorted by key
- Files sorted by path
- Block style only (no flow `{}` or `[]`)

Implementations MUST:
- Read state from `.testmd.lock` in the project root
- Write state to `.testmd.lock` in the project root
- Delete the lock file when state is empty
- Never modify TEST.md or `.testmd.yaml` when saving state
- Use atomic writes (write to temp file, then rename)
- Use file locking (flock) for concurrent access

---

## Watch patterns

The `watch` field uses glob patterns with variable substitution:

```
./path/to/file.go            — exact file
./*.go                       — single-level wildcard
./services/{name}/**         — variable substitution + recursive glob
./services/{name}/api/*.go   — mixed
```

Variables use `{name}` syntax. Before globbing, `{var}` placeholders are replaced with the actual label values from `each` or `combinations`.

### Special segments

| Segment  | Meaning                                             |
|----------|-----------------------------------------------------|
| `{name}` | Variable placeholder, substituted before glob       |
| `*`      | Any name at one path level (standard glob)          |
| `**`     | Any sub-path, zero or more levels (standard glob)   |

---

## Each

`each` defines variable sources. All variables are expanded as a **cartesian product**, producing one test instance per combination.

```yaml
each:
  service: ./services/*/
  env: [prod, staging]
watch: ./services/{service}/**
```

Each value in the `each` map is a **source**:

| Syntax | Result | Example |
|---|---|---|
| `./path/*/` | Directory names (trailing `/` = dirs only) | `service: ./services/*/` → `[auth, billing]` |
| `./path/*` | File and directory names | `item: ./data/*` → `[foo.txt, bar]` |
| `./path/*.ext` | File names without extension | `config: ./configs/*.yaml` → `[app, db]` |
| `[a, b, c]` | Explicit list | `env: [prod, staging]` |

Glob sources use standard glob syntax (including `**`), are filtered by the ignorefile, and exclude hidden files (starting with `.`). Results are sorted and deduplicated.

Example: `each: {service: ./services/*/, env: [prod, staging]}` with `services/{auth, billing}/` produces 4 instances: `service=auth,env=prod`, `service=auth,env=staging`, `service=billing,env=prod`, `service=billing,env=staging`.

---

## Combinations

When a cartesian product is not appropriate, `combinations` provides explicit control over which label sets are generated. Each entry is an object of variable sources (same syntax as `each`), and entries are combined via **union**.

```yaml
combinations:
  - db: [postgres, mysql]
    suite: [full]
  - db: [sqlite]
    suite: [basic]
watch: ./migrations/{db}/**
```

This produces: `db=postgres,suite=full`, `db=mysql,suite=full`, `db=sqlite,suite=basic`.

Within each entry, sources are expanded as a cartesian product (same as `each`). Across entries, results are unioned.

Glob sources work inside `combinations` too:

```yaml
combinations:
  - service: ./services/*/
    env: [prod, staging]
  - service: [legacy-monolith]
    env: [prod]
```

---

## Test identification

### ID format: `aabbccddeeffgghhii` (18 hex characters)

```
aabbcc   — first 6 hex chars of sha256(title), or sha256(explicit_id) if set
ddeeff   — first 6 hex chars of sha256(label_string)
gghhii   — first 6 hex chars of sha256(relative_path_to_test_md)
```

The label string is a sorted, comma-separated list of `key=value` pairs. For tests without labels, it is an empty string (hash: `e3b0c4`).

The relative path is the path from the project root to the TEST.md file (e.g. `services/auth/TEST.md`).

The explicit `id` field provides a stable input to the hash, so renaming the title doesn't change the id.

The three segments are concatenated without separators, forming a single 18-character hex string.

### ID abbreviations

Users can refer to tests by prefix. Longer prefixes are more specific:

| Length | What it matches |
|--------|----------------|
| 18 chars | Exact match: specific test + labels + source file |
| 12 chars | Specific test + labels across all files |
| 6 chars | All instances of a test (all label combinations, all files) |
| < 6 chars | Prefix match (if unambiguous) |

Resolution: pure prefix match. Exact match is the special case of a full-length prefix.

---

## File hashing

For each test instance:

1. Expand `watch` patterns into a file list (with `{var}` substitution)
2. Exclude `.testmd.lock` — it changes on every resolve
3. Sort files alphabetically
4. For each file: `sha256(relative_path + "\0" + file_content)`
5. Content hash: `sha256(concat(all_file_hashes))`

The content hash is compared with the stored value. If different, the test status becomes `outdated`.

The individual file hashes are stored in state to show **which files** changed.

---

## Test statuses

| Status     | Meaning                                            |
|------------|----------------------------------------------------|
| `pending`  | New test, or test with no stored state              |
| `resolved` | Verified and passed                                 |
| `failed`   | Verified and failed (with message)                  |
| `outdated` | Was resolved/failed, but content hash changed       |

### Transitions

```
new test ──────────────────────────► pending
pending  ── resolve ──────────────► resolved
pending  ── fail ─────────────────► failed
resolved ── content hash changed ─► outdated
failed   ── content hash changed ─► outdated
outdated ── resolve ──────────────► resolved
outdated ── fail ─────────────────► failed
resolved ── reset ────────────────► pending
failed   ── reset ────────────────► pending
outdated ── reset ────────────────► pending
```

---

## Ignorefile

The `ignorefile` config field (default: `.gitignore`) specifies a gitignore-format file. Matching entries are excluded from:

1. **TEST.md discovery** — ignored directories are not searched for TEST.md files
2. **Variable discovery** — ignored directories are not enumerated as variable values
3. **File matching** — ignored files are not included in hash computation

This prevents `__pycache__`, `node_modules`, build artifacts, etc. from affecting tests.

For directory entries, the path is checked with a trailing `/` to match gitignore directory patterns correctly.

---

## State storage

### Location

State is stored in a single `.testmd.lock` file in the project root, as described in the [State file](#state-file) section.

### State record fields

| Field          | Type              | Description                              |
|----------------|-------------------|------------------------------------------|
| `title`        | string            | Test title (from `# Title`)              |
| `source`       | string            | Relative path to the TEST.md file        |
| `labels`       | object            | Label key-value pairs                     |
| `content_hash` | string            | Hash of all watched files at resolve time |
| `files`        | object            | `{relative_path: sha256_hash}` for each file |
| `status`       | string            | `resolved` or `failed` (`pending` and `outdated` are computed, not stored) |
| `resolved_at`  | string or null    | ISO 8601 timestamp                        |
| `failed_at`    | string or null    | ISO 8601 timestamp                        |
| `message`      | string or null    | Failure message                           |

---

## Variable substitution

Variables use `{var}` syntax and are substituted in:
- `watch` patterns (before file matching)
- Test titles (in `get` output)
- Test descriptions (in `get` output)

This allows descriptions to reference the current label values:

```markdown
# {service} health check

Verify that `{service}` responds to healthcheck on `{env}`.
```
