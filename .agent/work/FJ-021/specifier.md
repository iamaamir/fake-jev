---
stage: specifier
task: FJ-021
inputFingerprint: 8dacde273d16cf959da358666963b3afdbec066ac5dd4ebea125c094c4796740
outputFingerprint: 8dacde273d16cf959da358666963b3afdbec066ac5dd4ebea125c094c4796740
taskFingerprint: 4559c4633576304d61378f9e05d4fd68a8a1d9db7fddfed3188c00e4a9adc84e
gitHead: a76d577
generatedAt: 2026-09-29T17:02:00Z
author: worker/FJ-021-specifier-refresh-3
---

# Specifier — FJ-021: request history, request-history clearing, verify, and full reset

## Observable criteria

### Request history — `GET /__fake/v1/requests`

- Returns HTTP `200` with a JSON object whose only member is `requests`, an
  array of interaction records ordered by `sequence` ascending.
- Each record has exactly the members `sequence` (integer), `timestamp`
  (string), `method` (string), `path` (string), `profile` (string or `null`),
  `operation` (string or `null`), `outcome` (string), `matchedStubId` (string or
  `null`), `status` (integer), `error` (string or `null`), `requestBody` (any
  JSON value, a string, or `null`).
- `timestamp` is UTC RFC3339 with whatever fractional precision the
  implementation naturally records; it is diagnostic and does not influence
  matching or golden semantic comparisons.
- `outcome` is one of exactly `matched`, `unmatched`, `unknown_route`,
  `validation_error`, `payload_too_large`, `sequence_exhausted`,
  `invalid_stub_response`, `internal_error`; `journal_full` never appears as an
  interaction outcome.
- `profile` and `operation` are strings whenever the server identified the
  active profile/operation for that request, and `null` when routing failed
  before that information existed.
- `requestBody` rules: the parsed JSON value when the request body was valid
  JSON; the UTF-8-decoded raw body string (with replacement characters for
  invalid UTF-8) when the body was malformed JSON; `null` when there was no body
  and when an oversized request's body was intentionally not retained.
- `matchedStubId` is the selected stub's id when a stub was selected and `null`
  otherwise. `error` is `null` for matched requests, including an intentionally
  configured raw non-2xx response, and is otherwise the fake failure code for
  that outcome. `status` is the data-plane HTTP status returned for that
  request.
- Only admitted data-plane interactions appear: control-plane requests are never
  recorded, and a request refused for journal capacity (which receives no
  sequence) has no record.
- Falsifiable anchors: a successful `GET /v1/models` appears as
  `profile: "jev/v1"`, `operation: "models"`, `outcome: "matched"`,
  `matchedStubId: null`, `status: 200`, `error: null`, `requestBody: null`; an
  unmatched stub request appears as `outcome: "unmatched"`, `status: 501`,
  `error: "fake_jev_unmatched_request"`; an unknown route as
  `outcome: "unknown_route"`, `status: 404`; a malformed body as
  `outcome: "validation_error"`; an oversized data-plane body as
  `outcome: "payload_too_large"`, `status: 413`, `requestBody: null`.

### Request-history clearing — `DELETE /__fake/v1/requests`

- Returns HTTP `204` with an empty body.
- Afterwards `GET /__fake/v1/requests` returns an empty `requests` array, and the
  next admitted data-plane request is recorded with `sequence: 1`.
- Interaction-derived verification failures are gone: a following
  `GET /__fake/v1/verify` reports `failures: []` unless unsatisfied invocation
  expectations remain, which are not interaction-derived and are preserved.
- Stub invocation counts and response sequence positions are not reset: listed
  `invocations` are unchanged and a sequenced stub's next selected call continues
  from the position it had reached.
- The sticky `journal_full` failure state ends (it is kept only until request
  history is cleared), so a data-plane request after the clear is admitted with
  `sequence: 1` instead of returning `507`.

### Verification — `GET /__fake/v1/verify`

- Returns HTTP `200` with exactly one of two shapes: `{"passed": true,
  "failures": []}`, or `{"passed": false, "failures": [ ... ]}` with at least one
  failure item.
- Every failure item has exactly the members `code` (string), `message`
  (string), `requestSequence` (integer or `null`), `stubId` (string or `null`).
  `message` is non-empty human-readable text; the specification does not fix its
  wording.
- `code` is one of exactly the eleven required codes: `unmatched_request`,
  `unknown_route`, `validation_error`, `payload_too_large`,
  `sequence_exhausted`, `invalid_stub_response`, `journal_full`,
  `internal_error`, `expect_exactly`, `expect_at_least`, `expect_at_most`. No
  other code appears, and no failure is derived that §41.9 does not define.
- Ordering: interaction-derived failures first in ascending `requestSequence`,
  then expectation failures in ascending stub registration index. An
  interaction-derived failure carries the sequence assigned to its request; a
  failure with no assigned sequence (notably `journal_full`) carries
  `requestSequence: null`.
- `stubId` names the involved stub for sequence exhaustion and invalid stub
  responses (the failures whose wire bodies carry a `stubId`) and for expectation
  failures; it is `null` for interaction-derived failures that concern no
  specific stub (unmatched request, unknown route, validation failure, payload
  too large, journal full).
- Observed failure codes line up with the observed data-plane results:
  `unmatched_request` (501), `unknown_route` (404), `validation_error` (422),
  `payload_too_large` (413), `sequence_exhausted` (409), `invalid_stub_response`
  (500), `internal_error` (500), `journal_full` (sticky, no sequence assigned),
  and `expect_exactly` / `expect_at_least` / `expect_at_most` when a stub's
  invocation count misses its configured bound.
- Verification fails when, since the last clear/reset boundary, any of these
  occurred: an unmatched data-plane request, an unknown route, a profile
  validation failure, response sequence exhaustion, interaction journal
  overflow, a fake-server internal failure, or an unsatisfied stub invocation
  expectation. An intentionally configured raw/emulated provider error such as
  HTTP 429 does not itself fail verification.
- Repeated `GET /__fake/v1/verify` calls between lifecycle boundaries return the
  same result; failure state is cleared only by `DELETE /__fake/v1/requests` and
  by full reset, and neither of those removes unsatisfied invocation
  expectations (reset re-evaluates them against zeroed counters, so a previously
  satisfied `expect.exactly` can newly fail).

### Full reset — `POST /__fake/v1/reset`

- Returns HTTP `204` with an empty body.
- As one atomic control operation, all six effects are observable once it
  returns: (1) every dynamic stub is gone — absent from `GET /__fake/v1/stubs`
  and no longer matchable; (2) every static stub's invocation count is zero;
  (3) every static stub's response sequence is back at element zero, so the next
  selected call returns the first element; (4) `GET /__fake/v1/requests` is empty
  and interaction-derived verification failures are gone; (5) the next admitted
  data-plane request is recorded with `sequence: 1`; (6) the next dynamic
  registration index is `N+1` for `N` static stubs, so creating a dynamic stub
  after reset returns `registrationIndex: N+1` (with two static stubs, `3`).
- Static stubs remain registered at their original registration indexes with
  unchanged configuration.
- Atomicity is observable: a data-plane request racing a reset either completes
  before it (its interaction and stub effects are then wiped by the reset) or
  observes the fully reset state; no partially reset state is observable.
- After reset, expectations are evaluated against zeroed invocation counts: a
  static stub with `expect.exactly: 1` that was satisfied before the reset fails
  verification with `expect_exactly` until it is called once again. Verification
  passes after reset exactly when no expectation is unsatisfiable at zero
  invocations.

## Traces

- C-CTRL-006 — §41.7 exact `requests` record shape, outcome enumeration,
  sequence-ascending order, and `requestBody` rules.
- C-CTRL-007 — §41.8 204, history and interaction-derived failure clearing,
  sequence restart at `1`, stub counters and sequence positions untouched; §43.5
  for the end of the sticky `journal_full` state.
- C-CTRL-008 — §41.9 always-200 calculation, exact pass/fail shapes, fixed
  failure item shape, required codes, defined ordering; §11.4 failure conditions
  and the raw-error exemption.
- C-CTRL-009 — §41.10 six atomic reset effects including next dynamic index
  `N+1`; §44.8.
- C-SEQ-006 — reset rewinds every static sequence to element zero and removes
  all dynamic stubs (§41.10, §44.8).
- C-SEQ-008 — an unsatisfied invocation expectation fails verification with
  `expect_exactly` / `expect_at_least` / `expect_at_most` (§41.9).
- C-GOLD-008 — §44.8 golden reset vector, including the post-reset expectation
  evaluation.

## Explicit non-goals

- No CLI `verify` command, exit-code handling, or wrapper-side verification
  logic (FJ-032; §15.3, §42.1). This item stops at the HTTP endpoints.
- No interaction-derived failure condition, failure code, or failure item beyond
  §41.9: no new codes, no derived judgements, no wrapper-specific
  interpretation (§11.4, §0.1).
- No new control endpoints, query parameters, or configuration keys around these
  routes, and no change to how data-plane interactions are recorded (§0.1,
  §41.1).
- No durable persistence or cross-restart guarantees for history, sequence,
  counters, or verification state; only live-process state is defined.

## Scope amendment

- The `method` and `path` members of every request-history record (§41.7) are
  satisfied by host-populated exchange metadata written from
  `internal/host/http/server.go`, under the keys `"method"` and `"target"`.
  Precedent: the host already writes `exchange.Metadata["error"]`.
- The engine core and its journal record type are unchanged: `Exchange.Metadata`
  is defensively cloned into journal records, so the host-written keys reach the
  recorded interaction without an engine change.
- Scope amendment 2 (PM, 2026-09-29): `internal/engine/journal.go` is added to
  `allowedFiles` because its `cloneExchange` helper does not preserve
  `Exchange.Metadata` — it reassigns the field to a fresh map and then ranges
  over that now-empty map, so every recorded exchange silently loses its
  metadata (the host's pre-existing `Metadata["error"]` included). Approved as a
  minimal correctness fix inside `cloneExchange`: build a local map, then assign
  it to the record, so the deep-copy (`cloneValue`) semantics and the
  nil-stays-nil contract are preserved. No new engine fields, no change to the
  journal record type, and no change to the public engine surface.
- Host delegation of `/__fake/` requests to `control.API` is still not wired
  (`internal/host/http/server.go` answers those paths with its own unknown
  control endpoint response). That remains a separate recorded gap, out of scope
  for FJ-021.
- Scope amendment 3 (PM, 2026-09-29; owner ruling + system_one second opinion):
  §41.7's `requestBody` rule is authoritative, so the derived vectors were
  corrected rather than the specification. 15 journal assertions across 13
  vectors under `testdata/contracts/` that asserted `"requestBody": null` for
  valid-JSON bodies now assert the parsed request body. `testdata/contracts/` is
  added to `allowedFiles` for that correction. No criterion above is changed by
  this amendment; the criteria already described the parsed-value rule.

## Owner rulings

- §41.7 wins over the derived vectors. The §41.7 prose rule — `requestBody` is
  the parsed JSON value for valid JSON, the decoded raw string for malformed
  JSON, and `null` for no body or an intentionally unretained oversized body —
  is the normative behavior. The 15 journal assertions across 13
  `testdata/contracts/` vectors were the derived artifact in error and have been
  corrected to the §41.7 rule (amendment 3).
- Null-sequence ordering: §41.9 is silent on where a null-`requestSequence`
  interaction-derived failure (notably `journal_full`) sorts. The adopted reading
  is that null-sequence interaction-derived failures sort after all sequenced
  interaction-derived failures and before expectation failures. Recommended
  §41.9 sentence wording: "Within the interaction-derived group, failures with an
  assigned `requestSequence` are ordered ascending by that sequence and precede
  any interaction-derived failure with `requestSequence: null`; all
  interaction-derived failures precede expectation failures."
- Separately recorded, unpatched defect: `testdata/contracts/verification/
  03-payload-too-large.json` expects `413` with `fake_jev_payload_too_large` for
  a step body of roughly 88 bytes against `limits.dataPlaneBodyBytes: 1024`, so
  the vector cannot produce its own expected status (its `PADDING` placeholder
  was never expanded). This is recorded but not corrected here; it needs a
  dedicated vector-audit ticket.

## Observations (specification ambiguity recorded, not resolved)

- `method` and `path` (§41.7) appear only in the example record; the section
  never states whether `path` is the request path alone or the full request
  target. §39.8 makes query strings irrelevant to routing, so the value recorded
  for a request carrying a query string is undetermined. The criteria therefore
  require only that the record identifies the data-plane request by method and
  path.
- `profile` / `operation` null-ness (§41.7) is given as a rule, not as a
  per-outcome table: the specification does not say whether an unknown route
  records the configured active profile. The criteria state the rule only.
- `error` for a `validation_error` outcome is underdetermined: §41.7 says "the
  fake failure code", §21.2 lists `fake_jev_validation_error` as a "verification
  classification", while §39.3 / §44.10 return a Jev-shaped 422 detail envelope
  with no `error` member.
- The `status` member (§41.7) is not defined explicitly; it is read here as the
  data-plane HTTP status returned for that request, which §43 defines per failure
  and the example illustrates as `200`.
- Failure `message` wording (§41.9) is a free human-readable string; only the
  item shape is fixed.
- The `stubId` null rule is assembled from §41.9's item shape plus the
  §43.3 / §43.4 wire bodies rather than stated in §41.9 itself.
- §41.9 fixes the `200` case only ("always returns HTTP `200` when verification
  can be calculated"); no status or body is defined for the case where
  verification cannot be calculated, so nothing is claimed for it.
- That `GET /__fake/v1/verify` does not itself clear failure state is an
  inference from §41.9 naming only `DELETE /__fake/v1/requests` and full reset as
  clearing conditions; §41.9 does not state it directly.
- Response media type and the 204 body rule for the new endpoints follow §11 and
  are already covered by catalog row C-CTRL-011 (FJ-020); they are inherited, not
  re-claimed, by this item.
- Integration seam, not a spec gap: the HTTP host still answers every `/__fake/`
  request with the "unknown control endpoint" 404 (`internal/host/http/server.go`),
  because wiring the control package into the host is outside this item's
  allowed files. As in FJ-020, the observable surface for these endpoints is the
  control API handler itself; host wiring belongs to a separate slice.

This artifact records observable behavior and traceability only; it declares no
implementation or acceptance verdict.
