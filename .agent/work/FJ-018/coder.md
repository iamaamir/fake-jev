---
stage: coder
task: FJ-018
inputFingerprint: 35e197abcaf1232556d96fbea5a9fc08ead097834fc3063daa85935685723864
outputFingerprint: 42e8fb43dcfeb2f77cb2af3211b84f449af80fcff92a1ce24ddb729e9b57df0f
taskFingerprint: aa19104f2114a6952c92e7fff2055c9f448e3079a3d2e8f317f7f4ca63eb48f8
gitHead: 25e53aa
generatedAt: 2026-09-29T09:26:00Z
author: subagent/worker-FJ-018-coder
---

# Coder — FJ-018

## Changes

- Added the integration suite at `test/integration/dataplane_test.go`, loading validated YAML/JSON configuration, starting each host on `127.0.0.1:0`, exercising the resolved URL with `net/http.Client`, and deterministically closing the listener.
- Covered all in-scope data-plane §44 vectors: models, noul, choice, score, exact question sets, priority/registration ordering, sequences, raw responses, validation, journal and payload limits, and exact routes.
- Completed engine transition failure bookkeeping for validation, unmatched, unknown-route, payload-limit, sequence-exhaustion, journal-full, and invalid-stub outcomes while preserving provider-neutral exchange and response boundaries.
- Added post-encoding interaction completion so configured raw non-2xx responses remain matched and retain their actual status.
- Added a host engine accessor for in-process deterministic integration-state assertions without adding a control endpoint.

## Changed files

- `internal/engine/engine.go`
- `internal/engine/journal.go`
- `internal/host/http/server.go`
- `test/integration/dataplane_test.go`

## Evidence

- `gofmt -w test/integration/dataplane_test.go internal/engine/engine.go internal/engine/journal.go internal/host/http/server.go`
- `go test ./test/integration`
- `go test ./...`
- `go test -race ./...`
- `go vet ./...`
- `go run ./cmd/guard lint`
- `go run ./cmd/guard arch`
- `go run ./cmd/guard trace`
- `go run ./cmd/guard fuzz`
- `./scripts/verify-candidate FJ-018` — product checks passed; downstream stage artifacts are not yet present
- `./scripts/candidate-fingerprint candidate .` — `42e8fb43dcfeb2f77cb2af3211b84f449af80fcff92a1ce24ddb729e9b57df0f`

## Remaining

- Downstream cleaner, hardener, and QA stage evidence remains to be produced.
