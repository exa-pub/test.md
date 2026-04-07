# testmd — Architecture

This document describes the internal architecture of testmd, independent of implementation language.

## Data flow

```
.testmd.yaml ──► parse config ──► Config (ignorefile)
                                      │
                                      ├──► discover TEST.md files
                                      │         │
                                      │         ▼
                                      │    parse each ──► TestDefinition[]
                                      │                        │
                                      │                   expand labels
                                      │                   (matrix or auto-discover)
                                      │                        │
                                      │                        ▼
                                      │                   TestInstance[]
                                      │                   (hash files, generate IDs)
                                      │                        │
                                      ├──► load state ◄────────┘
                                      │    (.testmd.lock)  compute statuses
                                      │                        │
                                      │                        ▼
                                      └──────────────────► CLI output
```

## Components

### Config loader

**Input:** `.testmd.yaml` or `.testmd.yml` file content
**Output:** Config struct (ignorefile)

Responsibilities:
1. Parse YAML config
2. Apply defaults (ignorefile → `.gitignore`)
3. Validate fields

### Root discovery

**Input:** current working directory
**Output:** project root path, config

Algorithm:
```
search upward from cwd for .testmd.yaml or .testmd.yml
if found → root = directory containing config
if not found → error
```

### TEST.md discovery

**Input:** root directory, ignorefile
**Output:** sorted list of TEST.md absolute paths

Algorithm:
```
walk root directory recursively
skip directories matching ignorefile
collect files named exactly "TEST.md"
sort by relative path
```

### Parser

**Input:** TEST.md file content, source file path
**Output:** list of TestDefinition

Responsibilities:
1. Split by `# ` headings (h1 only)
2. For each heading: extract the first ` ```yaml ` block as config, everything else as description
3. Validate: yaml block required, `watch` required
4. Track source line numbers

The parser does NOT handle config, discovery, or state loading — those are the caller's responsibility.

### Pattern resolver

Two responsibilities:

**1. Variable discovery** — resolve `each` / `combinations` sources to value lists.

For glob sources (e.g. `./services/*/`):
```
run standard glob (doublestar)
if trailing / → filter to directories only
extract basename of each match
if glob has explicit extension (*.yaml) → strip extension
filter: skip hidden files, skip ignorefile matches
sort and deduplicate
```

For explicit sources (e.g. `[prod, staging]`): use values as-is.

For `each`: cartesian product of all resolved sources.
For `combinations`: union across entries, cartesian within each entry.

**2. File resolution** — substitute `{var}` into watch patterns and glob for files.

Algorithm:
```
replace {var} with label values
strip leading ./
glob from root directory
filter: only files, exclude ignored
sort alphabetically
```

### Hasher

Computes content hashes for change detection:

- **File hash:** `sha256(relative_path + "\0" + file_content)` — path is included so renaming a file changes the hash
- **Content hash:** `sha256(concat(file_hashes))` — files must be sorted before concatenation
- **Test ID:** `sha256(title_or_explicit_id)[:6] + sha256(label_string)[:6] + sha256(source_path)[:6]` — 18 hex chars, no separators

### State manager

**Read:** read `.testmd.lock`, parse YAML. Missing file → empty state.
**Write:** serialize YAML (deterministic formatting), atomic write (temp + rename) to `.testmd.lock`. Empty state → delete file.
**Locking:** flock (LOCK_EX) for writes, ensuring concurrent access safety.

State is a single file per project in the project root.

### Resolver

Ties everything together:

1. For each TestDefinition, expand labels (matrix or auto-discovery)
2. For each label combination, resolve watch patterns to files
3. Compute content hash from matched files
4. Generate test ID from title, labels, and source path
5. Create TestInstance with all computed data

Status computation:
- No stored record → `pending`
- Stored hash ≠ current hash → `outdated`
- Otherwise → stored status (`resolved` or `failed`)

### Reporter

Formats output for terminal (with colors) and files (markdown, JSON).

Groups test instances by their source file, then by definition.

Label substitution: `{var}` in titles and descriptions is replaced with actual values in `get` output.

## Key invariants

1. **State is always in `.testmd.lock`** — never inline in TEST.md
2. **Hashing is deterministic** — same files with same content always produce the same hash
3. **Labels are sorted** — in IDs, state records, and display, labels are always sorted by key
4. **Files are sorted** — file lists are always sorted alphabetically before hashing
5. **Lock file is excluded from hashing** — `.testmd.lock` changes on every resolve and must not affect content hash
6. **Ignorefile applies to discovery and matching** — an ignored path never produces a label, a TEST.md, or contributes to a hash
7. **State is a single file** — all tests from all TEST.md files store state in one `.testmd.lock`
8. **ID is globally unique** — same title + same labels in different TEST.md files produce different IDs (source path is part of the ID)
9. **ID is deterministic** — same title + same labels + same source path always produce the same ID
10. **Atomic writes** — state is written via temp file + rename to prevent corruption
11. **TEST.md files are auto-discovered** — all files named `TEST.md` under root, filtered by ignorefile

## Module boundaries

| Module    | Depends on       | Responsibility                      |
|-----------|------------------|-------------------------------------|
| models    | —                | Data structures                     |
| parser    | models           | TEST.md → definitions, config parsing |
| patterns  | models,(filesystem) | Variable discovery, file globbing  |
| hashing   | (filesystem)     | SHA256, ID generation               |
| state     | (filesystem)     | Read/write state in `.testmd.lock`  |
| resolver  | patterns,hashing | Build instances, compute statuses   |
| report    | resolver         | Format output                       |
| cli       | all above        | CLI commands, root/config discovery |

The dependency graph is acyclic. `models` has no dependencies. `cli` depends on everything else but nothing depends on `cli`.
