---
stage: specifier
task: FJ-017
inputFingerprint: b36ecf5e1da6ff235dbfb96eab8ac7959c74e72db5e7034edc9f7056ff04c954
outputFingerprint: b36ecf5e1da6ff235dbfb96eab8ac7959c74e72db5e7034edc9f7056ff04c954
taskFingerprint: e56486e8ee7c2c403e987715b5cd6c1cb6aff7bce6387ecbd053b7a256bea81e
gitHead: 49263a2
generatedAt: 2026-09-29T08:28:39Z
author: subagent/worker-FJ-017-specifier
---

# Specifier — FJ-017

## Observable criteria

- The host serves the data plane through a real HTTP listener and dispatches requests to the active compatibility router. An admitted data-plane request is assigned its interaction sequence before route handling, body decoding, validation, or stub selection; the assign, dispatch/decode, selection, mutation, and outcome-recording transition is serialized against other data-plane requests and control mutations. Concurrent network arrival order is not promised, while an engine-assigned order is deterministic once admitted.
- Data-plane and control-plane admission use separate configured body limits and enforce the applicable limit before unbounded allocation. Defaults are `dataPlaneBodyBytes: 8388608` (8 MiB), `controlPlaneBodyBytes: 2097152` (2 MiB), `maxInteractions: 10000`, `logBodyBytes: 4096`, and `gracefulShutdownSeconds: 5`; configured limits retain the specification's valid ranges.
- With journal capacity available, an oversized data-plane body returns HTTP `413 Payload Too Large` with exactly `{"error":"fake_jev_payload_too_large","message":"Request body exceeds the configured limit."}`. It receives a normal request sequence and a journal outcome of `payload_too_large` with error `fake_jev_payload_too_large` and `requestBody: null`, fails verification, and does not route, match, invoke a stub, or advance response-sequence state.
- A request that discovers a full interaction journal returns HTTP `507 Insufficient Storage` with exactly `{"error":"fake_jev_journal_full","message":"Interaction journal limit reached. Reset or increase maxInteractions."}`. It receives no sequence, invokes no stub, mutates no invocation or response-sequence state, does not evict older interactions, and establishes the sticky `journal_full` verification failure with `requestSequence: null`.
- Control-plane requests are not data-plane interactions and never enter the journal. A control-plane body-limit response is HTTP 413 but does not affect verification; control API endpoint behavior itself is outside this slice.
- An unknown non-control data-plane route, including a trailing-slash variant or wrong method, returns HTTP `404 Not Found` with exactly `{"error":"fake_jev_unknown_route","message":"No active compatibility profile handles this route."}`. It is journaled as an interaction and fails verification. Query strings do not change recognition of an otherwise exact route.
- A recognized data-plane operation with no configured matching stub returns HTTP `501 Not Implemented` with exactly `{"error":"fake_jev_unmatched_request","message":"No configured stub matched this request.","profile":"jev/v1","operation":"systemone"}` and fails verification. This unmatched outcome is distinct from an unknown route and does not permit fallback matching.
- The golden vectors are preserved over HTTP: with `maxInteractions: 1`, the second data-plane request produces the §43.5/§44.14 journal-full behavior; with an incoming body exceeding the configured data-plane limit and available capacity, the request produces the §43.7/§44.15 payload-too-large status, outcome, error, null body, and unchanged stub state.

## Provider-neutral boundary

The HTTP host owns listener admission, plane classification, route dispatch, bounded body reading, and the serialized transition boundary. Compatibility-specific route claims and the exact `jev/v1` unmatched operation envelope remain in the compatibility router/profile. Journal records, verification failures, stub state, and sequence state remain engine concerns exposed through their existing interfaces; this slice wires their required observable transitions without defining a second journal implementation.

## Traces

- C-HOST-004 — §§19.4 and 43.7 define the data-plane 413 admission, sequence/journal outcome, null request body, no-stub mutation, and verification failure.
- C-HOST-006 — §§11 and 41.1 require control-plane exclusion from the interaction journal and exclusion of control-plane 413 responses from verification.
- C-HOST-007 — §43.2 defines the exact unknown-route 404 body, journaling, and verification failure.
- C-HOST-008 — §12.2 and §19.4 define the default body, journal, log-preview, and shutdown limits.
- C-MATCH-010 — §43.1 defines the exact unmatched data-plane 501 body and verification failure.
- C-GOLD-014 — §44.14 defines journal capacity exhaustion, no sequence/state mutation, and the sticky `journal_full` failure.
- C-GOLD-015 — §44.15 defines the exact payload-too-large outcome and unchanged stub/sequence state.

## Explicit non-goals

- No control API endpoints, control API schemas, or control-plane mutation implementation; those belong to FJ-020. The host only preserves the control/data admission boundary needed for the traced limit behavior.
- No signal handling, graceful-shutdown orchestration, ready-file creation/removal, CLI or `run` wrapper behavior; those belong to FJ-031 and related host lifecycle work.
- No journal bookkeeping implementation, journal schema redesign, request-history endpoint, verification runner, sticky-failure storage, reset, or interaction inspection; existing engine/state interfaces are used only at the required HTTP transition boundary.
- No changes to Jev request validation, stub matching or selection rules beyond routing unmatched requests to the specified failure, response generation, other compatibility profiles, provider calls, or network egress.
- No route redirects, trailing-slash aliases, wrong-method aliases, silent journal eviction, unbounded request-body allocation, or invented fake-server error bodies.

This artifact records observable behavior and traceability only; it does not declare an implementation or acceptance verdict.
