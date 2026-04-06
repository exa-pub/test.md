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
