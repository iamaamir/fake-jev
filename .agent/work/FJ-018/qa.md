---
stage: qa
task: FJ-018
inputFingerprint: 4f811c765e1cf47c069ff957b82979f307f7376776475789c31e8183d700b178
outputFingerprint: 4f811c765e1cf47c069ff957b82979f307f7376776475789c31e8183d700b178
taskFingerprint: aa19104f2114a6952c92e7fff2055c9f448e3079a3d2e8f317f7f4ca63eb48f8
gitHead: 25e53aa
generatedAt: 2026-09-29T09:41:22Z
author: subagent/worker-FJ-018-qa
---

# QA observations — FJ-018

## Candidate and chain identity

- Current candidate fingerprint: `4f811c765e1cf47c069ff957b82979f307f7376776475789c31e8183d700b178`.
- Current task fingerprint: `aa19104f2114a6952c92e7fff2055c9f448e3079a3d2e8f317f7f4ca63eb48f8`.
- Hardener input (cleaner output): `b6ee5d27a9832f6beabdb4fbe9f33a162fa776e685da6fff84eee8f94b3db4a2`.
- Hardener output/current candidate: `4f811c765e1cf47c069ff957b82979f307f7376776475789c31e8183d700b178`.
- Hardener artifact author: `subagent/worker-FJ-018-hardener`.
- Coder artifact author: `subagent/worker-FJ-018-coder`.
- QA artifact author: `subagent/worker-FJ-018-qa` (distinct from coder).
- QA made no production-file edits and added no persistent probe or test file.

## Real-listener data-plane observations

The focused integration suite starts `net.Listen("tcp", "127.0.0.1:0")`, serves the configured host, uses a real `net/http.Client` against the resolved ephemeral address, closes idle client connections, closes the listener, and waits for `Serve` to return. JSON bodies were compared after decoding, so object-key order and insignificant numeric spelling were ignored as required by §44; each expected status and JSON value was asserted.

The following in-scope data-plane §44 vectors were exercised through that listener:

- **C-GOLD-001 / §44.1 models:** `GET /v1/models?x=1` returned `200` and the exact default `jev-latest` model list, description, and `1970-01-01` release date.
- **C-GOLD-002 / §44.2 noul:** the configured `spam` stub returned `200`, echoed `jev-latest`, returned `spam.noul: 0.9`, and emitted zero-token usage.
- **C-GOLD-003 / §44.3 choice:** the configured `route` answer selected `backend`, confidence was `1.0`, and probabilities were the one-hot map over `frontend`, `backend`, and `infra`.
- **C-GOLD-004 / §44.4 score:** the configured fractional score returned `1.5`, confidence `1.0`, the request-derived three-entry legend, and probabilities `0.0/0.5/0.5`.
- **C-GOLD-005 / §44.5 exact question set:** adding `urgent` to the configured `route` request returned `501` with the exact `fake_jev_unmatched_request` envelope; engine state recorded `unmatched_request` verification failure.
- **C-GOLD-006 / §44.6 ordering:** priority `10` stub B beat priority `0` A and equal-priority C; an independent configuration without B selected C, demonstrating registration-order tie breaking rather than specificity scoring.
- **C-GOLD-007 / §44.7 sequence:** calls one and two returned the configured `1.0` and `0.0` answers. Call three returned `409` with the exact `fake_jev_sequence_exhausted` envelope; invocation count was `3`, sequence position was exhausted, and engine state recorded the sequence failure at request sequence `3`.
- **C-GOLD-009 / §44.9 raw error:** configured raw `429` returned status `429` and its JSON body, recorded a matched interaction with response status `429`, and left verification failures empty.
- **C-GOLD-010 / §44.10 malformed request:** body `{` returned `422` with exactly the `json_invalid` detail envelope and recorded a validation interaction/failure.
- **C-GOLD-011 / §44.11 missing state:** the otherwise valid request without `state` returned `422` with exactly `loc: ["body", "state"]`, `msg: "Field required"`, and `type: "missing"`; it recorded a validation interaction/failure.
- **C-GOLD-014 / §44.14 journal limit:** with `maxInteractions: 1`, the first models request returned `200`; the second returned `507` with the exact `fake_jev_journal_full` envelope, no second interaction/sequence was created, and the sticky failure had null request sequence. No stub mutation was possible or performed.
- **C-GOLD-015 / §44.15 payload limit:** with a 1024-byte data-plane limit, the oversized request returned `413` with the exact `fake_jev_payload_too_large` envelope; the journal outcome was `payload_too_large`, raw request body was nil, verification failure was recorded, and the configured stub invocation count remained zero.
- **C-GOLD-016 / §44.16 exact routes:** query-bearing `/v1/models` was handled; `/v1/models/`, `POST /v1/models`, and `GET /v1/systemone` returned `404` with the exact `fake_jev_unknown_route` envelope, were journaled, and produced verification failures.

The §44 reset (008), dynamic/static precedence (012), and `run` exit precedence (013) vectors are not data-plane vectors and are explicit FJ-018 non-goals; no claim is made for them here.

## Required quality and architecture observations

- **C-QUAL-001:** uncached `go test ./... -count=1` exercised all repository packages and completed without test failures.
- **C-QUAL-005:** the real-listener golden suite completed with `GOPROXY=off GOTOOLCHAIN=local`; it uses only in-process configuration plus loopback HTTP on an ephemeral port. No model, API key, provider SDK, paid service, or outbound network path was used.
- **C-ARCH-003:** the phase-1 source/guard review found no implementation of the §33 deferred features. `go run ./cmd/guard arch` and `go run ./cmd/guard lint` reported no findings; the phase remains in-memory strict data-plane wiring only.
- **C-ARCH-004:** normal execution uses fixture data and the local listener only. Dependency/source review found no provider client, dial/egress path, model invocation, API-key lookup, or credential validation in the product path. The engine remains provider-neutral; `go run ./cmd/guard arch` reported no findings.

## Determinism, cleanup, and repeatability

- Focused vectors: `go test ./test/integration -count=1` completed successfully.
- Focused vectors repeated 20 times: completed successfully.
- Focused vectors repeated 100 times: completed successfully.
- Focused vectors under race detection repeated 10 times: completed successfully.
- Every test-run server closed idle transport connections, closed its ephemeral listener, and waited for `Serve` completion; no fixed port or sleep-based synchronization was used.
- The repeated runs produced no intermittent failures, port collisions, or lifecycle leaks observable by the test process.

## Commands and results

- `./scripts/candidate-fingerprint candidate .` — `4f811c765e1cf47c069ff957b82979f307f7376776475789c31e8183d700b178`.
- `./scripts/candidate-fingerprint task .agent/work/FJ-018/state.json` — task fingerprint `aa19104f2114a6952c92e7fff2055c9f448e3079a3d2e8f317f7f4ca63eb48f8`.
- `go test ./test/integration -count=1 -run 'TestDataPlaneGoldenVectors|TestDataPlanePriorityAndRegistrationOrder|TestDataPlaneSequenceAndRawResponses|TestDataPlaneValidationLimitsAndRoutes' -v` — passed; all named in-scope vector groups and subtests passed.
- `go test ./test/integration -count=20 -run 'TestDataPlaneGoldenVectors|TestDataPlanePriorityAndRegistrationOrder|TestDataPlaneSequenceAndRawResponses|TestDataPlaneValidationLimitsAndRoutes'` — passed.
- `go test ./test/integration -count=100 -run 'TestDataPlaneGoldenVectors|TestDataPlanePriorityAndRegistrationOrder|TestDataPlaneSequenceAndRawResponses|TestDataPlaneValidationLimitsAndRoutes'` — passed.
- `go test -race ./test/integration -count=10 -run 'TestDataPlaneGoldenVectors|TestDataPlanePriorityAndRegistrationOrder|TestDataPlaneSequenceAndRawResponses|TestDataPlaneValidationLimitsAndRoutes'` — passed.
- `GOPROXY=off GOTOOLCHAIN=local go test ./test/integration -count=1` — passed.
- `go test ./... -count=1` — passed.
- `go test -race ./... -count=1` — passed.
- `go vet ./...` — passed with no diagnostics.
- `./scripts/selftest` — `summary: 75 passed, 0 failed`.
- `go run ./cmd/guard lint` — passed, `findings: []`.
- `go run ./cmd/guard arch` — passed, `findings: []`.
- `go run ./cmd/guard trace` — passed, `active: 0`, `covered: 17`.
- `go run ./cmd/guard fuzz` — passed, `targets: 0`.
- `./scripts/verify-candidate FJ-018` — completed; report `.agent/reports/FJ-018/report-20260929T094326Z.json` recorded `48 passed, 0 failed, 3 skipped, 0 not_applicable` with the QA chain and author checks bound to the candidate/task fingerprints above.

## Residual risks and observed gaps

- The repository's mutation/hardening tooling remains under the documented bootstrap exemption; race, vet, deterministic integration runs, and guard checks are the available executable evidence.
- The repository currently reports zero fuzz targets, so this stage provides no generated fuzz evidence beyond the guard's `targets: 0` result.
- Control API, reset, dynamic stubs, CLI lifecycle, graceful-shutdown policy, ready-file behavior, and `run` exit precedence remain outside FJ-018's allowed scope and were not exercised.
- The integration assertions inspect engine state through the phase-specific `Engine()` diagnostic accessor rather than a control API; this is the intended FJ-018 seam, not control-plane coverage.
- The test suite's body comparisons are semantic JSON comparisons as required; raw byte ordering is not independently asserted because §44 makes object-key order non-normative.
- No overall QA verdict is recorded here; this artifact records executed observations and remaining scope only.
