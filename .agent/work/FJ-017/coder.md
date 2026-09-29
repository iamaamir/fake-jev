---
stage: coder
task: FJ-017
inputFingerprint: b36ecf5e1da6ff235dbfb96eab8ac7959c74e72db5e7034edc9f7056ff04c954
outputFingerprint: e7bf3ac7918e394e8334d7d0a08f9f20c3039486bc6d0183e6ec76d9d26f8cfc
taskFingerprint: e56486e8ee7c2c403e987715b5cd6c1cb6aff7bce6387ecbd053b7a256bea81e
gitHead: 49263a2
generatedAt: 2026-09-29T08:40:00Z
author: subagent/worker-FJ-017-coder
---

# Coder — FJ-017

## Changes

- Added a real net/http host and listener-backed Serve method for the active jev/v1 router.
- Added separate bounded data/control body admission with specification defaults and exact fake failure envelopes.
- Wired data-plane transitions through the existing serialized engine journal, including unknown routes, payload-too-large outcomes, journal-full handling, unmatched requests, and response selection.
- Added validated configuration compilation for static stubs, matchers, sequences, and expectations without moving compatibility behavior into the engine.
- Added table-driven httptest coverage for exact route failures, query handling, payload limits, journal capacity, control exclusion, successful content responses, and concurrent admission.

## Changed files

- `internal/host/http/server.go`
- `internal/host/http/router.go`
- `internal/host/http/server_test.go`

## Evidence

- `gofmt -w internal/host/http/server.go internal/host/http/router.go internal/host/http/server_test.go`
- `go test ./...`
- `go test -race ./...`
- `go vet ./...`
- `go run ./cmd/guard lint`
- `./scripts/verify-candidate FJ-017`
- `./scripts/candidate-fingerprint candidate .` — `e7bf3ac7918e394e8334d7d0a08f9f20c3039486bc6d0183e6ec76d9d26f8cfc`

## Remaining

- Control API endpoints, lifecycle signals, ready-file handling, and control bookkeeping remain downstream work.
- Cleaner, hardener, and QA stage artifacts remain downstream evidence.
