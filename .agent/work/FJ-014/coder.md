---
stage: coder
task: FJ-014
inputFingerprint: a63a3d6fedef600dcdcdb8aed573eef907f8fea92e52605686e2133a0dea1b4c
outputFingerprint: ab4294d2021797bf919b58dbe3c2a61d068e4f998715ec42e28a82aac96b3db2
taskFingerprint: 1df19a608667d96667c2059da5f0a35005334f81ca89b7ab6c1307c5440d2021
gitHead: 2944771
generatedAt: 2026-09-29T06:45:18Z
author: subagent/worker-FJ-014-coder
---

# Coder — FJ-014

## Changes

Implemented the jev/v1 exact data-plane route lookup and deterministic request
validation boundary. Route recognition accepts only `GET /v1/models` and
`POST /v1/systemone`, ignores query strings, and rejects trailing slashes and
wrong methods without redirect aliases. Added the unknown-route 404 response
shape.

Added frozen request decoding and first-error validation for state, model,
questions, lexicographically ordered question names, supported question types,
instructions, and noul/choice/score criteria rules. Unknown fields and raw JSON
values are preserved. Validation failures expose a single deterministic detail
in the 422 envelope, including the malformed-json, missing-field, and
unsupported-type forms.

## Changed files

- `internal/compat/jev/v1/routes.go`
- `internal/compat/jev/v1/request.go`
- `internal/compat/jev/v1/validation.go`
- `internal/compat/jev/v1/validation_test.go`

## Evidence

- Table-driven exact-route tests cover query handling, trailing slashes, wrong
  methods, and unknown paths.
- Golden validation tests cover malformed JSON, missing state, unsupported
  question type, exact detail locations/types/messages, and wire envelope
  serialization.
- Validation-order tests cover required-field precedence, lexicographic
  question ordering, top-level object enforcement, and first-error-only output.
- Criteria tests cover valid supported question forms, unknown-field
  preservation, optional noul criteria, non-empty choice objects, non-empty
  score arrays, element kinds, and instruction kinds.

## Validation

- `gofmt -w internal/compat/jev/v1/routes.go internal/compat/jev/v1/request.go internal/compat/jev/v1/validation.go internal/compat/jev/v1/validation_test.go`
- `go test ./...`
- `go test -race ./...`
- `go vet ./...`
- `./scripts/verify-candidate FJ-014` — repository, Go, build, architecture,
  lint, trace, fuzz, and race checks executed; downstream stage artifacts were
  not present in this coder-stage workspace.
- `./scripts/candidate-fingerprint candidate .` —
  `ab4294d2021797bf919b58dbe3c2a61d068e4f998715ec42e28a82aac96b3db2`

## Remaining

Answer generation, stub selection, HTTP host integration, and framework prose
emulation remain outside this validation and route-recognition slice.
