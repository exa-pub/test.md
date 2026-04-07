# Go implementation works correctly

```yaml
watch:
  - ./internal/**
  - ./cmd/**
```

Go code changed — verify it works correctly:
1. Go tests pass: `go test ./internal/...`
2. Build succeeds: `go build -o ./bin/ ./cmd/...`
3. Run `./testmd-go status` on a sample TEST.md and verify output is correct

# Documentation is accurate

```yaml
watch:
  - ./docs/specification.md
  - ./docs/cli.md
  - ./docs/examples.md
  - ./docs/architecture.md
  - ./README.md
```

Read through each doc and verify:
1. All documented commands actually work as described
2. All examples are copy-pasteable and produce expected output
3. No references to removed features or old behavior
4. Architecture doc matches actual module structure

# Markdown code blocks are balanced

```yaml
watch:
  - ./docs/**
  - ./README.md
  - ./skills/testmd/SKILL.md
```

Markdown fences must be balanced: an opening fence (N backticks) is closed only by a line starting with exactly N backticks. Inner fences with fewer backticks are content, not structure.

Verify for each watched `.md` file:
1. Parse top-to-bottom: track a stack of open fences by backtick count
2. A line with N backticks either closes the current block (if N == top of stack) or is content (if N < top of stack) or opens a new block (if no block is open, or N > top of stack)
3. At end of file, the stack must be empty — no unclosed blocks
4. Outer fences wrapping examples (e.g. in README, SKILL.md) must use strictly more backticks (`````) than inner fences (```) — never the same count

# Agent skill is up to date

```yaml
watch:
  - ./skills/testmd/SKILL.md
  - ./docs/specification.md
  - ./docs/cli.md
  - ./internal/cli/cli.go
```

The agent skill (`skills/testmd/SKILL.md`) must accurately describe the current CLI:
1. All commands and flags match the implementation
2. ID format matches specification (18 hex, no dashes, prefix matching)
3. State file format and location are correct (.testmd.lock, YAML)
4. Project configuration described correctly (.testmd.yaml, no frontmatter)
5. No references to removed features (include, per-file lock files, JSON state, --testmd flag)

