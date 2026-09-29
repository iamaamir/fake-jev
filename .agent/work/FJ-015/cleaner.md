---
stage: cleaner
task: FJ-015
inputFingerprint: d2c3dbdd16b4a19a61aba2150ec84e5fa188c7bb3a4dc5edf1f851111e5d9849
outputFingerprint: 66f35765afb2948730b0bac0c54bc1b08852900e356f16423d15d9ad2690710e
taskFingerprint: c4409d7a1fbba4b732c5b1d9cc70c922f2a2f717e6b47adce19907a99d646528
gitHead: 4515668
generatedAt: 2026-09-29T07:43:53Z
author: subagent/worker-FJ-015-cleaner
---

# Cleaner — FJ-015

## Structure changed

- Added one shared `validateHelperFields` path for the identical choice and
  score required/allowed-field checks. Existing helper-specific error wording
  and accepted fields remain unchanged.
- Added sorted key traversal for request answer coverage, probability parsing
  and validation, and score expected-value accumulation. This makes validation
  order, error selection, and floating-point accumulation deterministic without
  changing accepted values or emitted answer data.
- Reused the generic `sortedKeys` helper for the existing criteria-key set;
  score level keys remain generated in request criteria order. No provider,
  engine, HTTP, or public compatibility boundary moved.

## Behavior unchanged

Exact answer-map coverage remains enforced before generation. Noul conversion,
choice one-hot generation and selected-maximum validation, score interpolation
and request-derived legends, finite `[0,1]` checks, `1e-6` probability and
expected-value tolerances, helper/request type matching, and supplied-legend
rejection remain as specified by §13.

The only product-file changes are in `internal/compat/jev/v1/fixture.go`.
`internal/compat/jev/v1/fixture_test.go` was gofmt-checked but not changed.

## Evidence

- Candidate chain: coder output
  `d2c3dbdd16b4a19a61aba2150ec84e5fa188c7bb3a4dc5edf1f851111e5d9849` to
  cleaner output
  `66f35765afb2948730b0bac0c54bc1b08852900e356f16423d15d9ad2690710e`.
- Task fingerprint:
  `c4409d7a1fbba4b732c5b1d9cc70c922f2a2f717e6b47adce19907a99d646528`.
- `gofmt -w internal/compat/jev/v1/fixture.go internal/compat/jev/v1/fixture_test.go` — completed.
- `go test ./...` — passed.
- `go test -race ./...` — passed.
- `go vet ./...` — passed.
- `go run ./cmd/guard arch` — passed.
- `go run ./cmd/guard lint` — passed.
- `go run ./cmd/guard trace` — passed.
- `go run ./cmd/guard fuzz` — passed.
- `./scripts/verify-candidate FJ-015` — product tests, vet, build,
  architecture, lint, trace, fuzz, and race checks passed; the downstream
  stage artifacts are not yet present.
- G-L complexity/CRAP tooling is not implemented and the repository's cleaner
  tooling exemption was reported by verification.
