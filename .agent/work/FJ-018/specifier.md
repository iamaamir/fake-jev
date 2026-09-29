---
stage: specifier
task: FJ-018
inputFingerprint: 35e197abcaf1232556d96fbea5a9fc08ead097834fc3063daa85935685723864
outputFingerprint: 35e197abcaf1232556d96fbea5a9fc08ead097834fc3063daa85935685723864
taskFingerprint: aa19104f2114a6952c92e7fff2055c9f448e3079a3d2e8f317f7f4ca63eb48f8
gitHead: 25e53aa
generatedAt: 2026-09-29T09:16:10Z
author: subagent/worker-FJ-018-specifier
---

# Specifier — FJ-018

## Observable criteria

- The integration test loads a validated YAML/JSON configuration, constructs the configured engine and `jev/v1` compatibility profile through the HTTP host, listens on `127.0.0.1:0`, and sends requests with a real `net/http` client to the resolved ephemeral URL. It must close the listener/server deterministically; it must not use a fixed port, sleeps as synchronization, a model, credentials, provider SDK, or outbound network.
- The configuration-to-host path is provider-neutral at the engine boundary: configuration owns parsing/defaults/validation, the host owns listener admission and HTTP dispatch, `jev/v1` owns Jev routes/request and response wire shapes, and the engine owns generic exchange matching, invocation/sequence state, journaling, and failure state. The engine must not gain Jev route, `noul`, `choice`, `score`, schema, or `net/http` knowledge.
- The ephemeral-port client covers every in-scope data-plane §44 vector, using exact JSON semantic comparison (object key order and insignificant numeric spelling do not matter):
  - `GET /v1/models` returns `200` and the exact default model list from §44.1.
  - The §44.2 `noul` fixture/request returns the exact `jev-latest` model echo, `spam` noul answer `0.9`, and zero-token usage.
  - The §44.3 choice fixture generates the exact one-hot `backend` answer over all request criteria keys, with confidence `1.0`.
  - The §44.4 fractional score fixture returns score `1.5`, confidence `1.0`, the request-derived legend, and probabilities `0.0/0.5/0.5`.
  - The §44.5 request containing the configured `route` choice plus an extra `urgent` question does not reuse the exact-question-set stub and returns the unmatched `501` fake error.
  - The §44.6 matching cases select priority `10` stub B over priority `0` A and equal-priority C; with B absent in a separate configured run, C is selected. No specificity scoring is introduced.
  - The §44.7 sequence returns its two configured answers on calls one and two; call three returns `409` with `fake_jev_sequence_exhausted`, increments the invocation count to three, and records the verification failure in engine state.
  - The §44.9 raw `429` fixture returns its configured raw response, records a matched interaction with no fake error, increments invocation count, and does not fail verification merely because the status is non-2xx.
  - The §44.10 malformed `{` body returns `422` with exactly the frozen `json_invalid` body envelope and records a `validation_error` verification failure.
  - The §44.11 request without `state` returns `422` with exactly the frozen `body.state` missing-field envelope.
  - The §44.14 `maxInteractions: 1` run processes and journals the first data-plane request; the second returns `507 fake_jev_journal_full`, does not assign a sequence or mutate stub invocation/response-sequence state, and records the sticky `journal_full` failure.
  - The §44.15 bounded-body run returns `413 fake_jev_payload_too_large` for an oversized data-plane body when journal capacity remains, records outcome `payload_too_large` with `requestBody: null`, does not route/invoke/consume a sequence, and records the verification failure.
  - The §44.16 route cases are exact: `GET /v1/models?x=1` is handled as models; `GET /v1/models/`, `POST /v1/models`, and `GET /v1/systemone` each return `404 fake_jev_unknown_route`, are journaled as data-plane interactions, and fail verification.
- Data-plane requests are deterministic and strict: no unmatched request receives a plausible default answer; configured raw non-2xx responses remain successful matches; validation, unknown-route, sequence, journal, and payload-limit failures use the frozen status/error contracts. Control requests do not enter this integration path's journal.
- The integration suite is offline by construction and satisfies the phase quality contract: `go test ./...` includes the real-listener suite, and the golden suite requires no model, API key, provider call, paid service, or network egress.
- No feature from §33 is added while wiring this slice: normal operation remains in-memory and deterministic, with no auto mode, scenario DSL, persistence, remote operation, control authentication, fault injection, record/replay, provider ingestion/evaluation, probabilistic generation, or generic HTTP mocking.

## Integration seam and allowed-file check

The approved seam is `config.Load`/validated `Config` -> `host/http.NewServerFromConfig` -> `host/http.Server.Serve(net.Listener)` -> `net/http.Client`. The compatibility profile is already injectable through the host router, and the engine already exposes the transition/journal state needed to retain verification failures after HTTP responses. `test/integration/dataplane_test.go` may be created under the explicitly allowed path; the host, engine, config, and `jev/v1` packages are all explicitly allowed for the wiring needed here. No CLI or control endpoint is needed for the in-scope vectors. No allowed-file gap or specification blocker was identified; do not expose a new control API or invent an alternate verification surface merely for this test.

## Traces

- C-GOLD-001 — §44.1 default model list.
- C-GOLD-002 — §44.2 noul answer.
- C-GOLD-003 — §44.3 choice one-hot generation.
- C-GOLD-004 — §44.4 fractional score generation.
- C-GOLD-005 — §44.5 exact question-set matching.
- C-GOLD-006 — §44.6 priority then registration order.
- C-GOLD-007 — §44.7 sequence exhaustion and verification failure.
- C-GOLD-009 — §44.9 configured raw error semantics.
- C-GOLD-010 — §44.10 malformed request.
- C-GOLD-011 — §44.11 missing state.
- C-GOLD-014 — §44.14 journal limit.
- C-GOLD-015 — §44.15 payload too large.
- C-GOLD-016 — §44.16 exact routes.
- C-QUAL-001 — §31 phase-1 Go test acceptance.
- C-QUAL-005 — §23 offline golden-contract requirement.
- C-ARCH-003 — §33 deferred decisions remain absent.
- C-ARCH-004 — §§4.9 and 19.2 prohibit model/provider calls and network egress in normal execution.

## Explicit non-goals

- No control API coverage or implementation, including reset, dynamic stub registration/removal, history/clear, verification endpoint, control schemas, or control-plane authentication. Consequently §44.8 reset and §44.12 dynamic/static precedence are not part of this slice.
- No CLI, `run`, child-process, ready-file, signal, graceful-shutdown, or exit-precedence coverage. Consequently §44.13 is out of scope; the test owns an ephemeral listener directly.
- No new public route, response, matcher, config field, profile, provider adapter, or verification API. Existing engine verification state is retained for later control-plane work; this slice observes the data-plane HTTP contract and required state transitions without inventing a control surface.
- No model execution, model quality simulation, API-key validation, provider/network access, external service, cross-repository client smoke test, or live upstream drift test.
- No §33 deferred feature, persistence, database, UI, remote multi-user mode, plugin ABI, WASM distribution, generic HTTP-mocking framework, or probabilistic/fuzz response generation.

This artifact records observable behavior and traceability only; it does not declare an implementation or acceptance verdict.
