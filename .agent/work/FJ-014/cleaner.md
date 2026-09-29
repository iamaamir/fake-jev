---
stage: cleaner
task: FJ-014
inputFingerprint: ab4294d2021797bf919b58dbe3c2a61d068e4f998715ec42e28a82aac96b3db2
outputFingerprint: ab4294d2021797bf919b58dbe3c2a61d068e4f998715ec42e28a82aac96b3db2
taskFingerprint: 1df19a608667d96667c2059da5f0a35005334f81ca89b7ab6c1307c5440d2021
gitHead: 2944771
generatedAt: 2026-09-29T06:49:57Z
author: subagent/worker-FJ-014-cleaner
---

# Cleaner — FJ-014

## Review scope

Reviewed the coder artifact, §39.2 request validation, §39.3 deterministic
422 envelopes, §39.8 exact route matching, the four allowed `internal/compat/jev/v1`
files, and the staff-architect guidance. The review covered route recognition,
validation-layer boundaries, first-error ordering, envelope construction,
repeated helpers and aliases, naming, and preservation of the strict
compatibility boundary.

## Structure reviewed

No product-file cleanup was warranted. Route recognition is isolated in
`routes.go`; request and wire error types are in `request.go`; decoding and
schema validation are in `validation.go`; and contract coverage remains in
`validation_test.go`. The provider-neutral engine is not reached by these
files, and no HTTP, provider, or framework behavior was introduced.

- `RecognizeRoute` strips only the query suffix and compares exact method/path
  pairs, leaving trailing slashes and wrong methods on the unknown-route path.
  `MatchRoute` and `RouteFor` are thin compatibility-facing spellings and do
  not duplicate route logic.
- `ValidateRequest` performs decoding, then the frozen `state`, `model`, and
  `questions` checks, then lexicographically ordered question validation, and
  returns immediately on the first error.
- `validateQuestion` keeps question-specific checks at the compatibility
  boundary. Raw flexible values and unknown fields remain preserved rather
  than being generalized into provider-neutral types.
- Error constructors centralize the single-detail envelope and copy location
  slices; the malformed-JSON, missing-field, value, and unsupported-type forms
  therefore share one construction path without changing their wire fields.
- The apparent repeated exported entry points (`ParseRequest` and
  `DecodeRequest`) preserve distinct caller-facing names while delegating to
  the one validator. Removing either would be an unnecessary API change.

No naming, extraction, or alias removal was justified by the specification;
additional abstraction would widen the public surface without reducing a
meaningful behavior or ownership seam. No test files were changed.

## Behavior evidence

- Unknown top-level and question fields remain accepted and retained as raw
  JSON.
- Required-field precedence and lexicographic question-name ordering remain
  explicit in the validator and covered by table-driven tests.
- Validation errors contain exactly one `Detail`, with the specified location,
  message, and type; `ValidationError.MarshalJSON` serializes the same envelope.
- The route and validation code remains entirely within `internal/compat/jev/v1`;
  no provider-neutral package imports Jev-specific behavior.

## Validation evidence

- `gofmt -w internal/compat/jev/v1/routes.go internal/compat/jev/v1/request.go internal/compat/jev/v1/validation.go internal/compat/jev/v1/validation_test.go` — completed.
- `go test ./...` — passed.
- `go test -race ./...` — passed.
- `go vet ./...` — passed.
- `./scripts/selftest` — 75 scenarios passed.
- `./scripts/verify-candidate FJ-014` — product tests, vet, build, architecture,
  lint, trace, fuzz, and race checks completed; the run recorded the cleaner
  artifact and the two downstream artifacts that are not yet present
  (`37 passed, 2 failed, 3 skipped, 0 not_applicable`).
- `./scripts/candidate-fingerprint candidate .` —
  `ab4294d2021797bf919b58dbe3c2a61d068e4f998715ec42e28a82aac96b3db2`.
- `./scripts/candidate-fingerprint task .agent/work/FJ-014/state.json` —
  task fingerprint `1df19a608667d96667c2059da5f0a35005334f81ca89b7ab6c1307c5440d2021`.
- G-L complexity/CRAP tooling is absent and the repository records the
  bootstrap exemption described by the cleaner role.
