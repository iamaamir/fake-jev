---
stage: cleaner
task: FJ-018
inputFingerprint: 42e8fb43dcfeb2f77cb2af3211b84f449af80fcff92a1ce24ddb729e9b57df0f
outputFingerprint: b6ee5d27a9832f6beabdb4fbe9f33a162fa776e685da6fff84eee8f94b3db4a2
taskFingerprint: aa19104f2114a6952c92e7fff2055c9f448e3079a3d2e8f317f7f4ca63eb48f8
gitHead: 25e53aa
generatedAt: 2026-09-29T09:31:00Z
author: subagent/worker-FJ-018-cleaner
---

# Cleaner — FJ-018

## Structure changed

- Reused `engine.DefaultMaxInteractions` as the HTTP host's exported default instead of maintaining a second literal. The engine remains the owner of the journal capacity default; the host-facing constant name and value remain available.
- Removed unused server assignments in the priority/registration integration case. The test still starts two independently configured ephemeral listeners and exercises both mappings; only the returned state handle is omitted where it is not inspected.
- No wiring, lifecycle, compatibility, engine, or public HTTP behavior was moved or redesigned.

## Behavior unchanged

Configuration loading still precedes host construction; `NewServerFromConfig` still compiles static stubs into the provider-neutral engine and binds the `jev/v1` profile in the HTTP router. Each integration case still loads its own validated document, listens on `127.0.0.1:0`, uses a real `net/http.Client`, and closes its listener while waiting for `Serve` to return. No sleeps, fixed ports, model/provider calls, credentials, or network egress were introduced.

The executable mappings remain the in-scope §44 vectors: models and query handling, noul, choice, score, exact question sets, priority/registration order, sequence exhaustion, configured raw 429, validation, journal and body limits, and exact routes. JSON comparison remains semantic after decoding, and request order remains explicit for sequence and priority cases. Control traffic remains outside this data-plane integration path. The engine continues to expose only normalized exchanges and generic interaction/failure state; Jev route and wire handling remains in `internal/compat/jev/v1` and the HTTP host.

## Evidence

- Candidate chain: coder output `42e8fb43dcfeb2f77cb2af3211b84f449af80fcff92a1ce24ddb729e9b57df0f` to current cleaner candidate `b6ee5d27a9832f6beabdb4fbe9f33a162fa776e685da6fff84eee8f94b3db4a2`.
- Task fingerprint: `aa19104f2114a6952c92e7fff2055c9f448e3079a3d2e8f317f7f4ca63eb48f8`.
- `gofmt -w internal/host/http/server.go test/integration/dataplane_test.go` — completed.
- `go test ./test/integration` — exit 0.
- `go test ./...` — exit 0.
- `go test -race ./...` — exit 0.
- `go vet ./...` — exit 0.
- `go run ./cmd/guard lint` — exit 0.
- `go run ./cmd/guard arch` — exit 0.
- `go run ./cmd/guard trace` — exit 0.
- `go run ./cmd/guard fuzz` — exit 0.
- `./scripts/verify-candidate FJ-018` — repository/product checks exit 0; the report also records the expected absence of downstream hardener and QA artifacts.
