# Parser correctness

```yaml
on_change: ./src/testmd/parser.py
```

Verify that TEST.md parsing handles edge cases:
1. Multiple tests in one file
2. Missing yaml block raises an error
3. on_change as string and as list both work

# Pattern expansion

```yaml
on_change: ./src/testmd/patterns.py
```

Verify label expansion with $variables:
1. Single $var enumerates directory entries
2. Nested $var1/$var2 produces cartesian combinations
3. Patterns without $vars return a single empty label set

# CLI commands

```yaml
on_change:
  - ./src/testmd/cli.py
  - ./src/testmd/report.py
```

Verify all CLI commands work end-to-end:
1. `testmd status` shows correct output
2. `testmd resolve` / `testmd fail` update state
3. `testmd ci` exits 1 when tests are pending
4. `testmd gc` removes orphaned records

# Service health

```yaml
on_change: ./services/$name/**
```

Verify that service `$name` starts and responds to healthcheck.

<!-- State
```testmd
{
  "version": 1,
  "tests": {
    "c9fdbc-e3b0c4": {
      "title": "Parser correctness",
      "labels": {},
      "content_hash": "06cdffd728b2f5aeb6e7b40cb82836766afab008b8ef3bdb823018f90f8cb019",
      "files": {
        "src/testmd/parser.py": "36a1b6997b297b7b5207c45fd5f7ef70eb2412031cc3bfccfed5a48ef06c0c21"
      },
      "status": "resolved",
      "resolved_at": "2026-04-06T20:48:33.043731+00:00",
      "failed_at": null,
      "message": null
    }
  }
}
```
-->
