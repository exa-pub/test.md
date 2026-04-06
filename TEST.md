---
include: [src/testmd/TEST.md]
---

# Go implementation matches Python

```yaml
on_change:
  - ./internal/**
  - ./cmd/**
```

Go code changed — verify it still matches the Python reference:
1. Run both on the same TEST.md: `testmd status --report-json /tmp/py.json` vs `./testmd-go status --report-json /tmp/go.json`
2. Compare IDs and content hashes: `diff <(jq -S '.tests[]|{id,status}' /tmp/py.json) <(jq -S '.tests[]|{id,status}' /tmp/go.json)`
3. Resolve a test with Go, read state with Python — status is consistent
4. State JSON written by Go is parseable by Python and vice versa
5. Go tests pass: `go test ./internal/...`

# Python implementation matches Go

```yaml
on_change:
  - ./src/testmd/**
  - ./tests/**
```

Python code or tests changed — verify it still matches the Go implementation:
1. Run both on the same TEST.md and compare report JSON (same as above)
2. Resolve a test with Python, read state with Go — status is consistent
3. Python tests pass: `python -m pytest tests/`
4. If a new feature was added to Python, the Go implementation needs the same feature
5. If a test was added to Python, an equivalent Go test is needed

# Documentation is accurate

```yaml
on_change:
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

<!-- State
```testmd
{
  "tests": {
    "38fe20-e3b0c4": {
      "content_hash": "64a3d49434306e4ec874e6cad0d0826b350ebe72203e6843498a5e9602aa7947",
      "failed_at": null,
      "files": {
        "src/testmd/TEST.md": "5f9504463a72e780d985904f87863f8114da12acec5188d6a5818917de7046f7",
        "src/testmd/__init__.py": "3f8ac0eaaf86d35f2b93f63eacb541d6f20a69ebc68c5affdf42b1016872c0fe",
        "src/testmd/cli.py": "b569ee631a03c1b63d431e7cc40a72aaa80bf74eedb80c294940b9647ff77553",
        "src/testmd/hashing.py": "5c406850aa878191fa716c0116a8de7338c9fc45f600e82d259c5c32cd82a007",
        "src/testmd/models.py": "80cea856a9911183a9a390c8720e5b5b105a4d8513c8bcc4089123874db8a5f6",
        "src/testmd/parser.py": "36a1b6997b297b7b5207c45fd5f7ef70eb2412031cc3bfccfed5a48ef06c0c21",
        "src/testmd/patterns.py": "0c01c3ef3ae968bea814503d2dd7a82ce578e1848e42f0ea641f88531fc37cfc",
        "src/testmd/report.py": "1b7a0b467a01c25e5964b56187e61e6c820410c8a895b7314f4810ccf112bb7c",
        "src/testmd/resolver.py": "e1af32a06cb7a6e74bd909dcc0487c9d0d12bff8ea38ad9221a3e623b91f055b",
        "src/testmd/state.py": "7e1b426792a1f1fef6ee9d3185017929e9c0e51bb02abf80baff4e6b1f824136",
        "tests/__init__.py": "ad539f0c07f27eaeb3317c06188b337f1c8e906b99a028fee249ec075369863a",
        "tests/test_hashing.py": "9d6c65400367e239ac8b1b21fe6ca3e6c6f2632bf1e4ff7ed6274171691c779e",
        "tests/test_parser.py": "2c929d15048d16b2ce3e916bb12432b3b16eb101e9346a2e90632ffb7a1b9f2e",
        "tests/test_patterns.py": "e9b5bfeb50d7a835d52f15816210411f313330331238e7d67927e55a1ad193dc",
        "tests/test_resolver.py": "77e41e7fe7abb96841cd7abdf310491c4db8ff1b3db8ab5a95942b011c3a649b",
        "tests/test_state.py": "171bf8013e0977cd72baebeac35885103cae980719a4b8bd9bce5aea9d34ab25"
      },
      "labels": {},
      "message": null,
      "resolved_at": "2026-04-06T21:59:08.140222+00:00",
      "status": "resolved",
      "title": "Python implementation matches Go"
    },
    "7a4284-e3b0c4": {
      "content_hash": "e7a1c5d306851f00952c93e3a21e20ac11b0512d6608d1602aa7f57eb7316eb5",
      "failed_at": null,
      "files": {
        "README.md": "200102d6092dcecaea1d2912080e7b5eca2d61562fe9784574d76232267182da",
        "docs/architecture.md": "bf31e22d52e6314d4e4ed026d387cf76e01eb45bf8d82e932b0e22dc89a920c4",
        "docs/cli.md": "58906b73f4d2f62296c59c2cc80fdd4d38a1184fd41c0d0265107f63d6342a85",
        "docs/examples.md": "c385819d26fee494ab5ea5b3914aff79ffe870738299f1d5e9ac6f3722f35d09",
        "docs/specification.md": "8ca6de64ca06c9a79e745c697ea2c869a221c08129c1cf2079010dca69938701"
      },
      "labels": {},
      "message": null,
      "resolved_at": "2026-04-06T22:42:29.035149+00:00",
      "status": "resolved",
      "title": "Documentation is accurate"
    },
    "a68d3b-e3b0c4": {
      "content_hash": "cde7ce192c7f17a31ef8049d8ff38a2177c70e3f89447a1b3061770ff0e87f08",
      "failed_at": null,
      "files": {
        "cmd/testmd/main.go": "f68f17819d1ec13d0b8a1480a9999348e0825c1583a4d93eeebd2d28d33944af",
        "internal/cli/cli.go": "9f33d01df3759e3c53cd899c4ec43f4077609be1471bdf89b87eb19ce9cdaa27",
        "internal/hashing/hashing.go": "21020a5d6c7cfdbf6729851b0386ff8f8caa3c2e3171898ae6856612349be6e7",
        "internal/hashing/hashing_test.go": "d8bc3caefd0cf353c7c19457bcf3ac8299c6abfaa609cede8a89b4af31cc3063",
        "internal/models/models.go": "63b0317011a1c4c1a3164e9ec7477d1a1c19e2a1043ca9e428e7079d0a2fc58c",
        "internal/parser/parser.go": "79b4885b152f6d321418aad95e9d32b497792c42313aef26ccca5008ae9dfeb2",
        "internal/parser/parser_test.go": "8354b84af738b3fd7288ab24d67e45aeb98eb561a2ed8567395aa53de6aad608",
        "internal/patterns/patterns.go": "47560fd0ed402001754c998fb11fe15d01ca9255db75baa77d7fb18afe814654",
        "internal/patterns/patterns_test.go": "b26b7ae4278cc6d1804827068e0f3bd40239d89628c65e488401b23d1ad6edba",
        "internal/report/report.go": "86bc8ccbc5b84e1acc1c63005dff32253c90b7d5a09bb167e6c615759168b19e",
        "internal/resolver/resolver.go": "881471c88cb51a61935a198426fb52a60cbf5c2c952f44c08e2bf1a5c09d64be",
        "internal/resolver/resolver_test.go": "87b4b72e0c506d9dafae3048efc6b5b3ddd557871956d871eb71592969a0bbdf",
        "internal/state/state.go": "dde505bd57199b78426b540f372160a1a7949773c84435efcc74533722f9ecd4",
        "internal/state/state_test.go": "8edc2a95cb2036debd4461cfd90a9061413ca3811c6d4e93ef585949c23ae343"
      },
      "labels": {},
      "message": null,
      "resolved_at": "2026-04-06T21:59:08.055295+00:00",
      "status": "resolved",
      "title": "Go implementation matches Python"
    }
  },
  "version": 1
}
```
-->
