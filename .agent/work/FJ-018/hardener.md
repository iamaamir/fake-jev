---
stage: hardener
task: FJ-018
inputFingerprint: b6ee5d27a9832f6beabdb4fbe9f33a162fa776e685da6fff84eee8f94b3db4a2
outputFingerprint: 4f811c765e1cf47c069ff957b82979f307f7376776475789c31e8183d700b178
taskFingerprint: aa19104f2114a6952c92e7fff2055c9f448e3079a3d2e8f317f7f4ca63eb48f8
gitHead: 25e53aa
generatedAt: 2026-09-29T09:36:00Z
author: subagent/worker-FJ-018-hardener
---

# Hardener — FJ-018

## Chain and scope

The audit consumed cleaner output
`b6ee5d27a9832f6beabdb4fbe9f33a162fa776e685da6fff84eee8f94b3db4a2` and
produced candidate output
`4f811c765e1cf47c069ff957b82979f307f7376776475789c31e8183d700b178`. The task
fingerprint is
`aa19104f2114a6952c92e7fff2055c9f448e3079a3d2e8f317f7f4ca63eb48f8`.
The audit covered the allowed configuration, engine, `jev/v1`, HTTP host, and
integration seams against §§17.4, 19.4, 31, 32, and 44.

## Hardening actions and contract evidence

- `Server.ServeHTTP` now rejects an inactive/nil router-engine boundary with the
  deterministic internal-error body instead of entering `Transition` through a
  nil engine. This closes the direct-construction panic path while preserving
  the configured host route and provider-neutral engine boundary (§17.4; §32).
- Added a host regression check for a nil router construction. The check ensures
  malformed host wiring fails closed and does not mutate a journal (§17.4;
  §32's observable internal-failure requirement).
- The integration cleanup closes idle client transport connections before
  closing each ephemeral listener, then waits for `Serve` to return. Listener
  address selection remains `127.0.0.1:0`, with no fixed-port collision or
  sleep-based synchronization (§31; §44 execution requirement).
- Request bodies are closed on every host path, and `readBody` bounds unknown
  content lengths with `limit+1` while rejecting known oversized lengths before
  allocation. Data-plane oversize requests enter one serialized transition with
  null journal body, no matcher/invocation/sequence mutation, 413 response,
  and verification failure; control oversize requests remain outside the journal
  (§19.4; §44.15).
- Engine admission remains the serialization boundary for sequence assignment,
  route/decode/validation, selection, invocation and sequence consumption.
  Journal-full admission is rejected before sequence assignment; reset and
  registration methods share the engine mutex, preventing partial state
  transitions under concurrent requests (§17.4; §44.7 and §44.14).
- Post-encoding status/outcome updates and failure recording preserve
  cross-layer propagation: configured raw non-2xx responses remain matched and
  do not create fake failures, while sequence exhaustion, validation, unknown
  route, unmatched, payload-limit, and invalid-stub paths retain their
  interaction/failure state (§32; §44.5, §44.7, §44.9, §44.10, §44.14–§44.16).
- The HTTP/profile path only consumes fixture data. The profile does not inspect
  Authorization, the host has no provider client or dial/egress path, and the
  integration suite uses no model, API key, provider network, or outbound
  request (§19.2; §31; §32).
- Exact route recognition, deterministic matching order, semantic JSON vector
  assertions, bounded journal behavior, and sequence request order remain
  unchanged across the in-scope §44 vectors.

## Validation evidence

- `gofmt -w internal/host/http/server.go internal/host/http/server_test.go test/integration/dataplane_test.go` — completed.
- `go test ./...` — completed successfully.
- `go test -race ./...` — completed successfully.
- `go test ./test/integration` — completed successfully.
- `go test -count=1 ./...` and `go test -count=1 -race ./...` — completed successfully without cached results.
- `go vet ./...` — completed with no diagnostics.
- `go run ./cmd/guard lint` — no findings.
- `go run ./cmd/guard arch` — no findings.
- `go run ./cmd/guard trace` — no findings (`active: 0`, `covered: 17`).
- `go run ./cmd/guard fuzz` — no findings (`targets: 0`).
- `./scripts/candidate-fingerprint candidate .` — `4f811c765e1cf47c069ff957b82979f307f7376776475789c31e8183d700b178`.
- `./scripts/candidate-fingerprint task .agent/work/FJ-018/state.json` — task fingerprint `aa19104f2114a6952c92e7fff2055c9f448e3079a3d2e8f317f7f4ca63eb48f8`.
- `./scripts/verify-candidate FJ-018` — repository, product, race, vet, and guard checks recorded in `.agent/reports/FJ-018/report-20260929T093645Z.json`; the only reported stage failure is the not-yet-produced downstream QA artifact.

## Residual risks

- Mutation/hardening analysis tooling remains under the repository bootstrap
  exemption; race, vet, deterministic tests, and repository guards are the
  available executable evidence.
- `NewServer` accepts direct non-zero limit values without reproducing the full
  configuration validator; validated configuration remains the public loading
  boundary, while negative direct limits are bounded by request admission.
- `Serve` intentionally receives an externally owned listener in this phase;
  graceful shutdown timeout and full control-plane lifecycle remain outside the
  phase-1 non-goals.
- The repository has no fuzz targets yet; the bounded guard scan is not a
  substitute for generated malformed-input fuzz coverage.
