---
stage: coder
task: FJ-016
inputFingerprint: 66f35765afb2948730b0bac0c54bc1b08852900e356f16423d15d9ad2690710e
outputFingerprint: 9b41eb056a93671e363b2cd42cb0948511dec384fba10e9df96e8b7d693ad668
taskFingerprint: fe3ad8fed06766d931689414827eac15a4dfd515b391587f6f1a2bc1f7ffb72e
gitHead: c182855723ad6217e95b89eb7fb1baeaaf26fd11
generatedAt: 2026-09-29T08:08:00Z
author: subagent/worker-FJ-016-coder
---

# Coder — FJ-016

## Changes

- Added the jev/v1 successful non-raw response envelope with request-model fallback,
  configured model override, exact zero usage defaults, and configured usage values.
- Added deterministic models metadata and response encoding, preserving configured
  order and the normative `jev-latest` default.
- Added raw response encoding for validated statuses, cloned headers, optional JSON
  body serialization, and case-insensitive default `Content-Type` handling.
- Added the host-neutral jev/v1 profile boundary for route matching, request decoding,
  response encoding, models responses, and Authorization pass-through without reading,
  validating, or logging credentials.
- Added table-driven golden and behavior tests for response defaults, model metadata,
  raw responses, content types, and the three accepted Authorization cases.

## Changed files

- `internal/compat/jev/v1/response.go`
- `internal/compat/jev/v1/models.go`
- `internal/compat/jev/v1/profile.go`
- `internal/compat/jev/v1/response_test.go`

## Evidence

- `gofmt -w internal/compat/jev/v1/response.go internal/compat/jev/v1/models.go internal/compat/jev/v1/profile.go internal/compat/jev/v1/response_test.go`
- `go test ./...`
- `go test -race ./...`
- `go vet ./...`
- `go run ./cmd/guard arch`
- `go run ./cmd/guard lint`
- `go run ./cmd/guard trace`
- `go run ./cmd/guard fuzz`
- `./scripts/verify-candidate FJ-016`
- `./scripts/candidate-fingerprint candidate .` — `9b41eb056a93671e363b2cd42cb0948511dec384fba10e9df96e8b7d693ad668`

## Remaining

- HTTP host/listener behavior, request admission, journal transitions, and stub
  selection remain outside this coder slice.
- Cleaner, hardener, and QA stage artifacts remain downstream evidence.
