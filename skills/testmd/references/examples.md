# testmd — Examples

## Simple: single file, no labels

````markdown
# Login page renders

```yaml
watch: ./src/auth/**
```

1. Open /login
2. Verify the form has email and password fields
3. Verify "Forgot password" link is present
````

```
$ testmd status
TEST.md
  Login page renders
    … a1b2c3e3b0c45ab752  pending

$ testmd resolve a1b2c3
Resolved: Login page renders

$ testmd status
TEST.md
  Login page renders
    ✓ a1b2c3e3b0c45ab752  resolved  (5s ago)
```

## Labels from filesystem (each with glob)

````markdown
# {service} healthcheck

```yaml
each:
  service: ./services/*/
watch: ./services/{service}/**
```

Verify `{service}` responds to GET /health with 200.
````

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
TEST.md
  {service} healthcheck
    … ed4be2fe0c315ab752  service=auth     pending
    … ed4be2c9054d5ab752  service=billing  pending
    … ed4be2ab12345ab752  service=gateway  pending

$ testmd resolve ed4be2
Resolved: auth healthcheck (service=auth)
Resolved: billing healthcheck (service=billing)
Resolved: gateway healthcheck (service=gateway)
```

Adding a new service (`services/payments/`) automatically creates a new pending test instance.

## Each with explicit values

````markdown
# API compatibility

```yaml
each:
  version: [v1, v2, v3]
watch: ./api/**
```

Verify the API contract for version `{version}`.
````

```
$ testmd status
TEST.md
  API compatibility
    … abc1231111115ab752  version=v1  pending
    … abc1232222225ab752  version=v2  pending
    … abc1233333335ab752  version=v3  pending
```

## Each with glob + explicit (cartesian product)

````markdown
# Deploy smoke test

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

## Combinations for irregular sets

````markdown
# Database migrations

```yaml
combinations:
  - db: [postgres, mysql]
    suite: [full]
  - db: [sqlite]
    suite: [basic]
watch: ./migrations/{db}/**
```

Run `{suite}` migration tests for `{db}`.
````

## Multiple watch patterns

````markdown
# Config validation

```yaml
each:
  env: ./config/*/
watch:
  - ./config/{env}/**
  - ./schema/config.json
```

Verify that `{env}` config validates against the JSON schema.
````

Changes to any config directory OR `schema/config.json` will mark the test as outdated.

## Multiple TEST.md files

TEST.md files are automatically discovered under the project root (the directory containing `.testmd.yaml`). Just create TEST.md files wherever they make sense:

```
project/
  .testmd.yaml
  TEST.md                    # root-level tests
  tests/integration/TEST.md  # integration tests
  tests/e2e/TEST.md          # e2e tests
```

```
$ testmd status
TEST.md
  Unit test sanity
    … aaa111e3b0c45ab752  pending

tests/integration/TEST.md
  Integration: database
    … bbb222e3b0c4112233  pending

tests/e2e/TEST.md
  E2E: checkout flow
    … ccc333e3b0c4445566  pending
```

All state is stored in a single `.testmd.lock` file in the project root. Same test title in different files produces different IDs (the source file path is part of the ID hash).

## Project configuration

`.testmd.yaml` in the project root:

```yaml
ignorefile: .gitignore
```

If `ignorefile` is not specified, `.gitignore` is used by default. The ignorefile filters out directories from TEST.md discovery, variable discovery, and file hashing.

To start a new project:
```
$ testmd init
Created .testmd.yaml
```

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

````markdown
# OAuth flow

```yaml
id: oauth
watch: ./services/auth/**
```

This test has a stable id based on `oauth` (not the title) that won't change if the title is renamed.
````
