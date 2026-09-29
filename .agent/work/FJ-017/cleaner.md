---
stage: cleaner
task: FJ-017
inputFingerprint: e7bf3ac7918e394e8334d7d0a08f9f20c3039486bc6d0183e6ec76d9d26f8cfc
outputFingerprint: 6929bb38a12d3f0a5137180715368be877a0d74d12ce5b5f71cd84b793aa66f8
taskFingerprint: e56486e8ee7c2c403e987715b5cd6c1cb6aff7bce6387ecbd053b7a256bea81e
gitHead: 49263a2
generatedAt: 2026-09-29T08:47:01Z
author: subagent/worker-FJ-017-cleaner
---

# Cleaner — FJ-017

## Structure changed

- Centralized zero-value limit completion in `applyLimitDefaults`, so direct server construction and config-derived limits use the same defaulting path. Configured non-zero values and all specification defaults remain unchanged.
- Replaced the repeated fake failure envelope types and custom marshalers with one ordered `failureBody` carrying optional profile, operation, and stub fields. The exact error codes, messages, status paths, and JSON field order remain unchanged.
- Reviewed `Router` and `Server` layering without moving compatibility routing, engine transitions, bounded body admission, response writing, or listener ownership. No test files were changed.

## Behavior unchanged

Data/control classification and separate bounded body reads remain before data dispatch. Data-plane transitions still assign and serialize journal state through the engine, including payload-too-large, unknown-route, unmatched, and journal-full paths; control requests remain outside the journal. Response status, headers, body envelopes, route recognition, stub selection, and sequence/invocation mutation ordering are unchanged. The engine remains the concurrency serialization boundary; no new locking or response-side state mutation was introduced.

## Evidence

- Candidate chain: coder output `e7bf3ac7918e394e8334d7d0a08f9f20c3039486bc6d0183e6ec76d9d26f8cfc` to cleaner output `6929bb38a12d3f0a5137180715368be877a0d74d12ce5b5f71cd84b793aa66f8`.
- Task fingerprint: `e56486e8ee7c2c403e987715b5cd6c1cb6aff7bce6387ecbd053b7a256bea81e`.
- `gofmt -w internal/host/http/server.go internal/host/http/router.go internal/host/http/server_test.go` — completed.
- `go test ./...` — passed.
- `go test -race ./...` — passed.
- `go vet ./...` — passed.
- `go run ./cmd/guard arch` — passed.
- `go run ./cmd/guard lint` — passed.
- `go run ./cmd/guard trace` — passed.
- `go run ./cmd/guard fuzz` — passed.
- `./scripts/verify-candidate FJ-017` — report `.agent/reports/FJ-017/report-20260929T085054Z.json` recorded the cleaner artifact, candidate fingerprint, repository tests, build, vet, race, architecture, lint, trace, and fuzz checks; downstream hardener and QA artifacts remain absent.
