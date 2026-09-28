---
stage: coder
task: FJ-012
inputFingerprint: 51091883561b6bf71f2238f41dca36138f47df3a1ceeddd451d63c4565c7dbb1
outputFingerprint: 51091883561b6bf71f2238f41dca36138f47df3a1ceeddd451d63c4565c7dbb1
taskFingerprint: ab72443cf7b377355d104172139b3a0b07decb20252a42b978316e820ed86f2c
gitHead: f866a06
generatedAt: 2026-09-28T20:56:30Z
author: subagent/worker-FJ-012-coder
---

## Changes

Added provider-neutral response-sequence state and selection lifecycle. Selected
stubs now increment their invocation count before consuming a response action;
sequence positions begin at zero, advance once per selected element, and report
an explicit exhaustion result without repeating the final action. Unmatched
exchanges do not mutate stub state. Added validated exact and bounded invocation
expectations with registration-ordered `expect_exactly`, `expect_at_least`, and
`expect_at_most` evaluation.

## Changed files

- `internal/engine/engine.go`
- `internal/engine/matcher.go`
- `internal/engine/stub.go`
- `internal/engine/sequence.go`
- `internal/engine/sequence_test.go`

## Evidence

- The amended allowed scope explicitly includes `internal/engine/matcher.go` for
  the `Registry.Select` integration that commits sequence consumption at
  selection time.
- The §44.7 sequence vector test consumes two actions at positions 0 and 1,
  returns an explicit exhaustion result on call 3, and observes invocation count
  3 with no implicit repeat-last behavior.
- Selection tests cover count commitment before later response failure and no
  sequence advancement for unmatched exchanges.
- Table-driven expectation tests cover exactly, atLeast, atMost, lower and upper
  range failures, satisfied ranges, invalid forms, and registration ordering.
- Engine source remains provider-neutral; response actions are opaque and no
  HTTP, Jev, route, or schema types were added.

## Validation

- `gofmt -w internal/engine/*.go`
- `go test ./...`
- `go test -race ./...`
- `go vet ./...`
- `./scripts/verify-candidate FJ-012` — repository, product Go, architecture,
  lint, trace, fuzz, build, vet, and race checks were executed; downstream
  cleaner, hardener, and QA artifacts were not present at coder stage.
- `./scripts/candidate-fingerprint candidate .` —
  `51091883561b6bf71f2238f41dca36138f47df3a1ceeddd451d63c4565c7dbb`

## Remaining

HTTP mapping of sequence exhaustion, interaction-derived verification failures,
configuration decoding, and reset lifecycle integration remain outside this
provider-neutral engine stage. Downstream cleaner, hardener, and QA stage
artifacts remain to be produced.
