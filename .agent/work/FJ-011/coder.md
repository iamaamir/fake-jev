---
stage: coder
task: FJ-011
inputFingerprint: dfbe154ac2a066ce29e4e356e521685ab5e42f972a5bda3b1f1c64cf1e7d7775
outputFingerprint: 7d24ad5a06f13ec9404686209d64ad43704aa9c8383e2c21738e876c15aa0e5a
taskFingerprint: b788644f51a266d22e3a0d6efc84fe854e22b1fba53fbc99cc44c97f32ab94bb
gitHead: 4d6dff0
generatedAt: 2026-09-28T19:41:57Z
author: subagent/worker-FJ-011-coder
---

## Changes

Implemented a provider-neutral engine boundary with normalized exchanges,
opaque response actions, static/dynamic stub provenance, duplicate-safe
registration, and one-based registration indices assigned by successful
registration order. Selection filters by active profile, combines present
matcher fields with logical AND, treats omitted fields as wildcards, compares
question name/type sets exactly, compares model strings exactly, and compares
JSON state values recursively with key-order-independent objects and
precision-preserving numeric equality. Candidate ordering is priority
descending followed by registration index ascending; no source-specific or
specificity precedence is used.

## Changed files

- `internal/engine/engine.go`
- `internal/engine/exchange.go`
- `internal/engine/stub.go`
- `internal/engine/matcher.go`
- `internal/engine/matcher_test.go`

## Evidence

- Table-driven matcher tests cover profile filtering, wildcard omissions,
  AND-combined fields, exact model and question matching, ignored payload
  contents, JSON null/array/object semantics, numeric exponent equivalence, and
  large-integer precision.
- Golden matching/order tests cover priority selection, equal-priority
  registration order, profile exclusion, removal of a higher-priority
  candidate, equal-priority static-versus-dynamic behavior, and explicit
  greater-priority dynamic override.
- Registration tests cover static/dynamic source provenance, one-based indices,
  duplicate rejection, and failure not consuming an index.

## Validation

- `gofmt -w internal/engine/*.go`
- `go test ./...`
- `go test -race ./...`
- `go vet ./...`
- `./scripts/candidate-fingerprint candidate .` —
  `7d24ad5a06f13ec9404686209d64ad43704aa9c8383e2c21738e876c15aa0e5a`
- `./scripts/verify-candidate FJ-011` — repository integrity, product Go,
  architecture, lint, trace, fuzz, and race checks completed; the aggregate
  gate remains blocked only by missing downstream cleaner, hardener, and QA
  artifacts.

## Remaining

No sequence, journal, specificity scoring, HTTP, CLI, process, or
compatibility-profile behavior was added. The broader engine integration and
response lifecycle remain for later work items.
