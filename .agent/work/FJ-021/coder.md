---
stage: coder
task: FJ-021
inputFingerprint: 8dacde273d16cf959da358666963b3afdbec066ac5dd4ebea125c094c4796740
outputFingerprint: 4acd6dd6c828af008d216a52f543aa571806355cf45bac0e71937bf5588d49ab
taskFingerprint: 4559c4633576304d61378f9e05d4fd68a8a1d9db7fddfed3188c00e4a9adc84e
gitHead: a76d577
generatedAt: 2026-09-29T17:03:32Z
author: worker/FJ-021-coder-vector-rebind
---

# Coder — FJ-021: request history, request-history clearing, verify, and full reset

## What changed

### `internal/control/handlers.go`

- Wired four routes into the existing `ServeHTTP`, alongside health, meta, and
  stubs, preserving their exact contracts:
  - `GET /__fake/v1/requests` → `listRequests`
  - `DELETE /__fake/v1/requests` → `clearRequests`
  - `GET /__fake/v1/verify` → `verify` (defined in `verify.go`)
  - `POST /__fake/v1/reset` → `reset`
  Every other method on those paths uses the existing
  `fake_jev_control_method_not_allowed` 405 envelope; unknown paths keep the
  existing `fake_jev_control_not_found` 404.
- Added the exact §41.7 record shape (`requestRecord`) with member order
  `sequence, timestamp, method, path, profile, operation, outcome, matchedStubId,
  status, error, requestBody`; nullable string members are pointers so absence
  renders as JSON `null`. `GET` always renders a non-nil `requests` array, so an
  empty journal is exactly `{"requests":[]}`.
- `profile`/`operation` are `null` when the engine recorded no value (for
  example `operation: null` for an unknown route); `profile` stays `"jev/v1"` for
  an unknown route because the host identified the active profile before routing
  failed.
- `error` is `null` only for `matched`; otherwise it is the §43 fake failure code
  (`failureCodeForOutcome`). Every outcome maps to `fake_jev_` + outcome, except
  `unmatched` → `fake_jev_unmatched_request` (§43.1). This keeps an intentionally
  configured raw non-2xx response (`outcome: matched`, `error: null`) exempt.
- `requestBody` implements §41.7 literally (`requestBodyValue`): the parsed JSON
  value (decoded with `UseNumber`) for a valid single JSON body; the UTF-8 decoded
  raw string with invalid sequences replaced by U+FFFD for a malformed body; and
  `null` for no body and for a `payload_too_large` body the host did not retain.
- `clearRequests` calls `engine.ClearRequestHistory()` and returns 204 with an
  empty body and no content type.
- `reset` calls `engine.Reset()` under the existing `stubMutations` lock, so a
  concurrent control-plane registration cannot capture an index across the reset,
  and the engine's own mutex serializes it against in-flight data-plane
  transitions. Returns 204 with an empty body and no content type.
- `method`/`path` are read from `record.Request.Metadata` under keys `method` and
  `target` (`metadataString`), with an empty-string fallback when a record carries
  no metadata.

### `internal/control/verify.go` (new)

- `verify` always returns HTTP 200 with the fixed §41.9 envelope: exactly
  `{"passed":true,"failures":[]}` when nothing failed, otherwise
  `{"passed":false,"failures":[...]}` with a non-nil array.
- Each failure has exactly the four §41.9 members
  (`code`, `message`, `requestSequence`, `stubId`); `requestSequence` and
  `stubId` render `null` when absent.
- Failures are assembled from engine state only, never derived from interaction
  records, so the §11.4 raw-error exemption holds automatically: a configured raw
  non-2xx response records `matched` and produces no engine failure.
- Ordering is interaction-derived failures first, by `requestSequence` ascending
  (stable), then expectation failures in stub registration-index order as
  returned by `Registry.EvaluateExpectations`. Sticky failures with no sequence
  (`journal_full`) are ordered after the sequenced interactions.

### `internal/host/http/server.go`

- Refactored the data-plane decision closure into a named `decide` and wrapped
  the `engine.Transition` call so every decision's exchange is stamped by
  `stampRequestMetadata` before it is recorded.
- `stampRequestMetadata` writes `Exchange.Metadata["method"] = request.Method`
  and `Exchange.Metadata["target"] = request.URL.Path`. Path is the URL path, so
  a query string is excluded (see observation on `path`). This is the only
  channel available: §8.6 gives the provider-neutral journal no HTTP fields.
- `target` is stamped on route, validation, unknown-route, payload, and internal
  error exchanges as well as matches.

### `internal/engine/journal.go` (scope amendment 1)

- Fixed `cloneExchange` so `Exchange.Metadata` is actually deep-copied. The
  previous code reassigned the field to a fresh map and then ranged over that new
  empty map, so every journal record silently lost its metadata (including the
  host's pre-existing `Metadata["error"]`). The copy now builds a local map and
  assigns it afterwards, keeping the nil-stays-nil and `cloneValue` deep-copy
  contract. This was required for the §41.7 method/path seam to work at all; it is
  a defect fix, not a design change, and no existing test referenced `Metadata`.

## Tests added (`internal/control/handlers_test.go`)

All new tests bind the real HTTP host (`internal/host/http`) and the control API
to one engine, so history and verification observe genuine data-plane recording.

- `TestControlRequestHistoryRecordRules` — table-driven over models, unmatched,
  unknown route, malformed body, valid-body-rejected-by-validation, invalid UTF-8,
  payload-too-large, invalid stub response, and configured raw 429. Asserts the
  full §41.7 member set, the timestamp parses as RFC3339, and each outcome's
  status/error/requestBody rule. Also asserts control traffic never enters the
  journal.
- `TestControlRequestHistorySequenceAscending` — sequence order plus
  `sequence_exhausted` and wrong-method unknown-route records.
- `TestControlRequestHistoryEmptyShape` — exact `{"requests":[]}`.
- `TestControlRequestHistoryWithoutHostMetadata` — the empty-string fallback.
- `TestEngineExchangeMetadataSurvivesTransition` — regression for the
  `cloneExchange` fix: metadata survives into the stored record and later
  mutation of the caller's map does not leak.
- `TestControlVerifyShapes`, `TestControlVerifyFailureOrdering`,
  `TestControlVerifyExpectationCodes`, `TestControlVerifyRawErrorExemption` —
  §41.9 shapes, ordering, all three `expect_*` codes, stability across calls, and
  the raw-error exemption.
- `TestControlClearRequestsEffects` — §41.8: 204/empty, history cleared, sequence
  restart at 1, interaction-derived failures gone, unsatisfied expectation
  preserved, and stub counters/sequence positions untouched (the post-clear
  sequenced stub continues at element 1).
- `TestControlClearRequestsEndsJournalFull` — sticky `journal_full` ends at clear
  and the next request is admitted with sequence 1.
- `TestControlFullResetVector` — the §44.8/C-GOLD-008 vector as an executable
  test: dynamic stub absent, static counters zero, history empty, next sequence 1,
  next dynamic index `N+1 = 3`, and post-reset `expect_exactly` failure for the
  `expect.exactly: 2` stub.
- `TestControlResetRewindsSequenceAndClearsFailures` — C-SEQ-006 sequence rewind
  (element 0 returned again after reset), interaction failure clearing, and next
  dynamic index `N+1 = 2`.
- `TestControlNewEndpointMethods` — 405 envelopes for wrong methods, no engine
  mutation.
- `TestControlResetRacesDataPlane`, `TestControlClearRacesDataPlane` —
  concurrency-safety for the two mutating endpoints; only the two legal outcomes
  occur, no partial reset state.

## Commands run and outcomes

- `gofmt -w` on all changed files; `gofmt -l` on the packages is empty.
- `go test ./internal/control -count=1` — ok
- `go test ./... -count=1` — ok (all packages)
- `go test -race ./... -count=1` — ok (all packages)
- `go vet ./...` — ok
- `go run ./cmd/guard arch` — exit 0, `findings: []`
- `go run ./cmd/guard lint` — exit 0, `findings: []`
- `go run ./cmd/guard trace` — exit 0, `active: 0`, `covered: 17`
- `go run ./cmd/guard fuzz` — exit 0, `targets: 0`
- `./scripts/verify-candidate FJ-021` — `RESULT: PASS`, 20 passed / 0 failed /
  0 skipped / 0 not_applicable; report
  `.agent/reports/FJ-021/report-20260929T160947Z.json`
- `./scripts/candidate-fingerprint candidate` —
  `8b96804bcfd74e200491d856317e9d93606c3244e46492caf7cfbccdfb5d5494`

## Observations / ambiguities

- **`path` is the URL path, query stripped; recorded under metadata key
  `target`.** `spec/control-api.openapi.yaml` and §41.7 both call the member
  `path`, and the committed §44 vector `matching/05-routes-exact-paths.json`
  records `GET /v1/models?x=1` as `path: "/v1/models"`. I therefore stamp
  `request.URL.Path` (not `RequestURI`) under `Metadata["target"]`.
- **`requestBody` for valid JSON bodies: §41.7 vs committed vectors
  (UNRESOLVED needs-human blocker).** §41.7's prose ("MUST contain the parsed JSON
  value for valid JSON bodies") and its example (`"requestBody": {}`) and
  `spec/control-api.openapi.yaml` all require the parsed value, so this
  implementation renders the parsed value (e.g. the object) and, for a
  valid-body validation failure, the parsed object. However, 15 committed
  `journal` assertions across 13 vectors assert `"requestBody": null` for
  valid-JSON request bodies, contradicting §41.7. Listed for adjudication (the
  vectors are outside this item's allowedFiles and were not edited):
  `matching/01-exact-question-set.json`,
  `matching/02-priority-then-registration.json`,
  `matching/03-dynamic-no-implicit-override.json`,
  `matching/06-priority-equal-registration-order.json`,
  `mixed/02-partial-answers-rejected.json`, `reset/01-full-reset.json`,
  `sequences/01-exhaustion.json` (3 records),
  `validation/02-missing-state.json`,
  `validation/03-unsupported-question-type.json`,
  `validation/04-empty-choice-criteria.json`,
  `verification/01-raw-error-not-a-failure.json`,
  `verification/02-journal-full.json`,
  `verification/04-unknown-choice-in-fixture.json`.
  (`verification/03-payload-too-large.json` asserts `null` and is consistent with
  §41.7, because that body was intentionally not retained; it is not part of the
  conflict.) Consequently the §44.8 executable test asserts the §41.7-correct
  parsed body, not the vector's `null`. This needs owner adjudication before any
  runner consumes those vectors.
- **Ordering of `null`-sequence sticky failures.** §41.9 fixes only
  "interaction-derived failures in request sequence ascending; then expectation
  failures". It does not place a failure with no sequence. `journal_full` is
  ordered after sequenced interaction failures, which also matches its
  chronological position (the journal is full, so no later interaction exists).
- **`path`/`method` seam.** §8.6 deliberately gives the journal no HTTP fields,
  so the host records `method`/`path` in `Exchange.Metadata` and the control API
  reads them, with an empty-string fallback (never a panic). `internal/control`
  and `internal/host/http` declare the same key literals; the values are pinned
  by tests on the control side and by the host-stamping tests.
- **Verify when the result cannot be calculated.** §41.9 defines only the 200
  case; no status or body is defined otherwise, so the endpoint always returns
  200 and aggregates whatever the engine reports (a nil engine yields the pass
  shape with no failures).
- **`timestamp`** is rendered as `Received.UTC()` in RFC3339 with nanosecond
  precision; it is diagnostic only.
- **Host control routing is still not wired.** As in FJ-020, the HTTP listener
  still answers every `/__fake/` path with the unknown-control-endpoint 404; that
  wiring is a separate slice and was explicitly out of scope. The observable
  surface for these endpoints in this item is the control API handler, exercised
  in tests alongside a real data-plane host sharing one engine.

## Residual risks

- The §41.7-vs-vectors `requestBody` divergence must be adjudicated by the owner
  before FJ-046 consumes `testdata/contracts/**`. Until then, any runner that
  compares `journal.requestBody` strictly will disagree with this implementation
  for valid JSON bodies.
- `internal_error` is required by §41.9, but the host's internal-error decision
  paths (`exchange.Payload` type-assert failure and a non-journal `Transition`
  error) do not record an engine failure, so such a case would not surface in
  `/verify`. This is a pre-existing host gap outside FJ-021's scope; the control
  layer aggregates engine failures only and does not invent failures.
- If a future host bypasses `serveData`, request-history `method`/`path` would
  render as empty strings rather than failing.

## Vector re-bind (PM decision)

- The `requestBody` divergence recorded above is resolved by owner ruling: the
  derived golden vectors were corrected to §41.7, so every assertion for a
  valid-JSON request body now carries the parsed request body instead of `null`.
  15 assertions across 13 files under `testdata/contracts/` were changed
  (`matching/01`, `matching/02`, `matching/03`, `matching/06`,
  `mixed/02-partial-answers-rejected`, `reset/01-full-reset`,
  `sequences/01-exhaustion` (3 records), `validation/02-missing-state`,
  `validation/03-unsupported-question-type`,
  `validation/04-empty-choice-criteria`,
  `verification/01-raw-error-not-a-failure`, `verification/02-journal-full`,
  `verification/04-unknown-choice-in-fixture`). Each corrected assertion now
  equals the step's own sent `request.body`; `verification/03-payload-too-large`
  keeps `null` (body intentionally not retained, §41.7/§44.15) and
  `validation/01-malformed-json` keeps the raw string `"{"`.
- No Go code changed in this revision: the diff is `testdata/contracts/` only.
- Candidate fingerprint `4acd6dd6c828af008d216a52f543aa571806355cf45bac0e71937bf5588d49ab`;
  task fingerprint `4559c4633576304d61378f9e05d4fd68a8a1d9db7fddfed3188c00e4a9adc84e`.

## Changed files

- `internal/control/handlers.go`
- `internal/control/verify.go` (new)
- `internal/control/handlers_test.go`
- `internal/host/http/server.go`
- `internal/engine/journal.go`
