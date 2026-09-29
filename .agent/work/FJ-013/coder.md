---
stage: coder
task: FJ-013
inputFingerprint: 06d1e29f3bf779c3f69c2067ce6251da6491a25f787824aa019becac9c440e47
outputFingerprint: d84f77ff9b4f9d3bf8278a18d9e1f37b4e5a59b4ba11e40617ea8e65fe0cdb3e
taskFingerprint: 7d8bd46b210c54295204609b3375cf8457dbdd048d367c0cf830120a6e85ad8d
gitHead: 734ff6f
generatedAt: 2026-09-29T05:58:08Z
author: subagent/worker-FJ-013-coder
---

# Coder — FJ-013

## Changes

Implemented a provider-neutral bounded interaction journal and serialized
engine transition boundary. Admissions reserve the next one-based interaction
sequence before the route/decode/validation callback, retain sequence-ordered
records without eviction, and expose a sticky journal-full failure when the
capacity is exhausted. Full admission transitions serialize selection,
invocation/response-sequence mutation, and recording against registrations,
request-history clear, and full reset.

Added atomic request-history clear and full reset primitives. Clear resets
journal sequence and interaction-derived failures while preserving stub state;
reset removes dynamic stubs, rewinds static counters/sequences, resets journal
state, and resumes registration indices immediately after static stubs.
Detached journal snapshots copy normalized exchanges and raw bodies. The engine
remains independent of HTTP, control APIs, and compatibility-profile types.

## Changed files

- `internal/engine/journal.go`
- `internal/engine/engine.go`
- `internal/engine/journal_test.go`

## Evidence

- Table-driven transition tests verify sequence assignment before rejected
  route/decode/validation outcomes, including malformed and unknown requests.
- Bounded-journal tests verify no sequence, selection, invocation, or sequence
  mutation after exhaustion; sticky `journal_full` carries a nil request
  sequence and is cleared by history clear.
- Reset tests verify dynamic removal, static state rewind, journal clearing, and
  registration-index restoration.
- Concurrent transition/reset coverage exercises serialized state under the Go
  race detector; detached-record tests cover request ownership.

## Validation

- `gofmt -w internal/engine/engine.go internal/engine/journal.go internal/engine/journal_test.go`
- `go test ./...`
- `go test -race ./...`
- `go vet ./...`
- `./scripts/verify-candidate FJ-013` — repository, product Go, architecture,
  lint, trace, fuzz, build, vet, and race checks passed; downstream stage
  artifacts were not present at coder stage.
- `./scripts/candidate-fingerprint candidate .` —
  `d84f77ff9b4f9d3bf8278a18d9e1f37b4e5a59b4ba11e40617ea8e65fe0cdb3e`

## Remaining

HTTP status/error mapping, control API exposure, and compatibility-profile
routing remain outside this engine change. Downstream cleaner, hardener, and QA
stage artifacts remain to be produced.
