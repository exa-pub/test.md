# testmd — CLI Reference

## Global option

All commands accept `--root PATH`:

```
testmd [--root PATH] <command> [args]
```

| `--root`          | Project root                                  |
|-------------------|-----------------------------------------------|
| not specified     | search upward from cwd for `.testmd.yaml`/`.testmd.yml` |
| path to directory | must contain `.testmd.yaml` or `.testmd.yml`  |

**Upward search:** when `--root` is not specified, the tool searches for `.testmd.yaml` or `.testmd.yml` in the current directory, then parent, then grandparent, etc. Error if not found.

---

## Commands

### `testmd init`

Create a `.testmd.yaml` file in the current directory with default settings.

```
$ testmd init
Created .testmd.yaml
```

Errors if `.testmd.yaml` or `.testmd.yml` already exists:
```
$ testmd init
Error: .testmd.yaml already exists
```

---

### `testmd status`

Show the status of all tests, grouped by source TEST.md file.

```
testmd status [--report-md FILE] [--report-json FILE]
```

Output:
```
TEST.md
  OAuth login flow
    ✓ abc123def456111222  provider=google env=prod  resolved  (2h ago)
    ✗ abc123789abc222333  provider=github env=prod  failed    "Redirect broken"
    ⟳ abc123112233333444  provider=apple  env=prod  outdated

sub/TEST.md
  Database migrations
    … 445566e3b0c4aabbcc  pending

Summary: 1 resolved, 1 failed, 1 outdated, 1 pending
```

Flags:
- `--report-md FILE` — save report as markdown
- `--report-json FILE` — save report as JSON

---

### `testmd resolve <id>`

Mark test(s) as resolved. Saves the current content hash.

```
$ testmd resolve abc123def456111222
Resolved: OAuth login flow (provider=google env=prod)
```

Using a 6-char prefix resolves all instances of that test:
```
$ testmd resolve abc123
Resolved: OAuth login flow (provider=google env=prod)
Resolved: OAuth login flow (provider=github env=prod)
Resolved: OAuth login flow (provider=apple env=prod)
```

---

### `testmd fail <id> <message>`

Mark test(s) as failed with a message.

```
$ testmd fail abc123789abc222333 "Redirect returns 500 on staging"
Failed: OAuth login flow (provider=github env=prod)
  Message: Redirect returns 500 on staging
```

---

### `testmd reset <id>`

Reset test(s) to pending — removes the stored state record, as if the test was never resolved or failed.

```
$ testmd reset abc123def456111222
Reset: OAuth login flow (provider=google env=prod)
```

Using a prefix resets all matching instances:
```
$ testmd reset abc123
Reset: OAuth login flow (provider=google env=prod)
Reset: OAuth login flow (provider=github env=prod)
Reset: OAuth login flow (provider=apple env=prod)
```

---

### `testmd get <id>`

Show test details: status, labels, watched patterns, files, and the full description.

```
$ testmd get abc123789abc222333

# OAuth login flow
Labels: provider=github env=prod
Status: failed
Failed at: 2026-04-05T10:00:00Z
Message: Redirect returns 500 on staging
Patterns: ./services/github/prod/**
Files: 3
---
Verify that OAuth works for each provider:
1. Navigate to /login
2. Click "Sign in with github"
...
```

For outdated tests, shows which files changed:
```
Changed:
  services/google/handler.go
  services/google/config.yaml
```

Label variables in the title and description are substituted with actual values.

---

### `testmd gc`

Remove orphaned state records — tests that no longer exist in TEST.md or whose label values no longer match the filesystem.

```
$ testmd gc
Removed 2 orphaned record(s).
```

---

### `testmd ci`

Like `status`, but exits with code 1 if any test is not resolved.

```
$ testmd ci
FAIL: 2 test(s) require attention

TEST.md
  OAuth login flow
    ✗ abc123789abc222333  provider=github env=prod  failed  "Redirect broken"
    ⟳ abc123112233333444  provider=apple  env=prod  outdated
```

Exit codes:
- `0` — all tests resolved
- `1` — at least one test is pending, outdated, or failed

Supports `--report-md` and `--report-json` flags.

---

## ID resolution

Test IDs are 18 hex characters with three implicit segments:

| Chars 1–6 | Chars 7–12 | Chars 13–18 |
|-----------|------------|-------------|
| hash of title/id | hash of labels | hash of source file path |

When specifying `<id>` in commands, you can use any prefix:

| Input              | Matches                                          |
|--------------------|--------------------------------------------------|
| `abc123def456111222` | Exact match (full 18 chars)                    |
| `abc123def456`     | All instances with that title+labels (12 chars)  |
| `abc123`           | All instances of that test (6 chars)             |
| `abc`              | All instances whose id starts with `abc`         |

Resolution: pure prefix match. Longer prefix = more specific.
