# testmd — Examples

## Simple: single file, no labels

```markdown
# Login page renders

```yaml
on_change: ./src/auth/**
```

1. Open /login
2. Verify the form has email and password fields
3. Verify "Forgot password" link is present
```

```
$ testmd status
Login page renders
  … a1b2c3-e3b0c4  pending

$ testmd resolve a1b2c3
Resolved: Login page renders

$ testmd status
Login page renders
  ✓ a1b2c3-e3b0c4  resolved  (5s ago)
```

## Labels from filesystem

```markdown
# $service healthcheck

```yaml
on_change: ./services/$service/**
```

Verify `$service` responds to GET /health with 200.
```

With filesystem:
```
services/
  auth/
    main.go
  billing/
    main.go
  gateway/
    main.go
```

```
$ testmd status
$service healthcheck
  … ed4be2-fe0c31  service=auth     pending
  … ed4be2-c9054d  service=billing  pending
  … ed4be2-ab1234  service=gateway  pending

$ testmd resolve ed4be2
Resolved: $service healthcheck (service=auth)
Resolved: $service healthcheck (service=billing)
Resolved: $service healthcheck (service=gateway)
```

Adding a new service (`services/payments/`) automatically creates a new pending test instance.

## Matrix: const values

```markdown
# API compatibility

```yaml
on_change: ./api/**
matrix:
  - const:
      version: [v1, v2, v3]
```

Verify the API contract for version `$version`.
```

```
$ testmd status
API compatibility
  … abc123-111111  version=v1  pending
  … abc123-222222  version=v2  pending
  … abc123-333333  version=v3  pending
```

## Matrix: match + const

```markdown
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
3. Run basic smoke test
```

## Matrix: union for irregular combinations

```markdown
# Database migrations

```yaml
on_change: ./migrations/$db/**
matrix:
  - const:
      db: [postgres, mysql]
  - const:
      db: [sqlite]
```

Run `migrate up` and `migrate down` for `$db`.

Note: sqlite only needs basic tests, while postgres and mysql need full tests.
```

## Multiple on_change patterns

```markdown
# Config validation

```yaml
on_change:
  - ./config/$env.yaml
  - ./schema/config.json
```

Verify that `$env` config validates against the JSON schema.
```

Changes to any `config/*.yaml` OR `schema/config.json` will mark the test as outdated.

## Include files

Root `TEST.md`:
```yaml
---
include: [tests/integration/TEST.md, tests/e2e/TEST.md]
---

# Unit test sanity

```yaml
on_change: ./src/**
```

Run `make test` and verify all unit tests pass.
```

`tests/integration/TEST.md`:
```markdown
# Integration: database

```yaml
on_change: ./src/db/**
```

Run integration tests against a real database.
```

```
$ testmd status
Unit test sanity
  … aaa111-e3b0c4  pending

Integration: database
  … bbb222-e3b0c4  pending
```

Each file stores its own state. Resolving "Integration: database" writes state to `tests/integration/TEST.md`, not to the root.

## Custom ignorefile

```yaml
---
ignorefile: .testmdignore
---
```

`.testmdignore`:
```gitignore
__pycache__/
*.pyc
node_modules/
dist/
*.min.js
```

If `ignorefile` is not specified, `.gitignore` is used by default.

## CI integration

```yaml
# GitHub Actions
- name: Check manual tests
  run: testmd ci --report-md test-report.md

# GitLab CI
test:manual:
  script:
    - testmd ci --report-json report.json
  artifacts:
    paths: [report.json]
```

## Explicit ID for stable references

```markdown
# OAuth flow

```yaml
id: oauth 
on_change: ./services/auth/**
```

This test has a stable id `oauth-e3b0c4` that won't change if the title is renamed.
```
