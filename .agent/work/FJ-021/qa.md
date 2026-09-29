---
stage: qa
task: FJ-021
inputFingerprint: 4acd6dd6c828af008d216a52f543aa571806355cf45bac0e71937bf5588d49ab
outputFingerprint: 4acd6dd6c828af008d216a52f543aa571806355cf45bac0e71937bf5588d49ab
taskFingerprint: 4559c4633576304d61378f9e05d4fd68a8a1d9db7fddfed3188c00e4a9adc84e
gitHead: a76d577
generatedAt: 2026-09-29T17:12:35Z
author: worker/FJ-021-qa-rebind-2
---

# QA — FJ-021: request history, request-history clearing, verify, and full reset

This artifact records independent observations against the acceptance criteria
and the specification. It is evidence, not a verdict: acceptance is emitted only
by `./scripts/verify-candidate` (AGENTS.md completion rule, `docs/agents/README.md`
authority hierarchy).

## Method

- Read `docs/agents/roles/qa.md`, `docs/agents/README.md`,
  `.agent/work/FJ-021/{specifier,coder,cleaner,hardener}.md`,
  spec §§41.7–41.10, 11.4, 17.4, 43.1–43.7, 44.8, and catalog rows
  C-CTRL-006/007/008/009, C-SEQ-006/008, C-GOLD-008.
- Did not rely on the repository's own tests. Wrote an independent probe suite
  outside the repository and injected it with `go test -overlay` so no file is
  added to the repository.
  - Probe source: `/private/tmp/fj021-qa-final/qa_final_probe_test.go`
    (package `control_test`, 23 top-level `TestQA*` functions plus 16 subtests;
    45,140 bytes).
  - Overlay: `/private/tmp/fj021-qa-final/overlay.json` maps
    `/Users/mak/git/fake-jev-FJ-021/internal/control/zz_qa_final_probe_test.go`
    to the probe source. A smoke overlay (`overlay-smoke.json`) confirmed the
    overlay mechanism before the real probes ran.
  - The probes exercise the public surface only: real `internal/host/http`
    data-plane server + real `internal/control` handler sharing one engine, plus
    `engine`/`control` exported APIs where the host surface cannot reach a case
    (called out per probe below).
- Candidate fingerprints re-derived from the worktree:
  `./scripts/candidate-fingerprint candidate` =
  `59598e94499eea493d9aff2662f2a91689a2a461b2e3202fd0070fc60a13b79e`
  (matches the front-matter `inputFingerprint`/`outputFingerprint`);
  `./scripts/candidate-fingerprint task .agent/work/FJ-021/state.json` =
  `taskFingerprint 44caf21dca360d2cfe46347601c2250e25a6181e2015b36830e535f50e95c0a5`.

## Per-criterion observations

### C-CTRL-006 — §41.7 request history

- **Exact record shape.** `TestQARequestsRecordFieldRules` decodes a record and
  requires exactly the 11 members `sequence, timestamp, method, path, profile,
  operation, outcome, matchedStubId, status, error, requestBody`; `timestamp`
  parsed as RFC3339Nano; `sequence`/`status` JSON numbers.
- **Empty history shape.** `TestQARequestsEmptyExactShape` observes the body is
  exactly `{"requests":[]}` with `Content-Type: application/json`.
- **Sequence ascending.** `TestQARequestsSequenceAscending` issues four matched
  `/v1/systemone` calls plus one unknown route and observes `sequence` = 1..5 in
  list order.
- **method/path from host-stamped Metadata.** `TestQARequestHistoryMethodPathFromHost`
  issues `DELETE /v1/systemone` (routes as unknown route) and observes
  `method:"DELETE"`, `path:"/v1/systemone"`, `outcome:"unknown_route"`. The
  matched-query case `GET /v1/models?x=1` records `path:"/v1/models"` (query
  stripped). The values travel through `Exchange.Metadata` (the host writes
  `"method"`/`"target"`; the control handler reads them).
- **profile/operation null when routing failed before identification.**
  `GET /v1/nope` records `operation:null`, `profile:"jev/v1"`.
  `TestQANullProfileAndOperationRendering` drives the engine boundary with an
  empty `Exchange{}`/empty profile/operation and observes `profile:null,
  operation:null`, pinning the empty-string→JSON-null rendering. Through the
  host surface `profile` is never null because the host identifies the active
  profile before routing (see evidence limits).
- **requestBody rules.** In `TestQARequestsRecordFieldRules`:
  - valid JSON body → parsed JSON value (`unmatched_keeps_parsed_body`,
    `valid_JSON_rejected_by_validation_keeps_parsed_value`,
    `raw_non-2xx_stays_matched_error_null`);
  - malformed body `{` → raw string `"{"`;
  - invalid UTF-8 `\xff` → `"\ufffd"` (replacement semantics);
  - no body → `null` (`no_body_requestBody_null`, and matched `GET /v1/models`);
  - oversized body (> `dataPlaneBodyBytes`) → `null`.
  `TestQARequestBodyTrailingTokensFallsBackToRaw` observes `{} {}` → raw string
  `"{} {}"` (not one valid JSON document).
- **outcome values.** Observed exactly the §41.7 enumeration through real
  data-plane traffic: `matched`, `unmatched`, `unknown_route`,
  `validation_error`, `payload_too_large`, `sequence_exhausted`,
  `invalid_stub_response`. `journal_full` was not observed as an interaction
  outcome; the capacity-refused request has no record
  (`TestQAJournalFullNoSequenceNoStubMutation`).
- **matchedStubId null when nothing selected.** Observed null for matched
  `/v1/models` with no stub, unknown route, unmatched, and payload-too-large.
- **error null for matched including raw non-2xx.**
  `TestQARequestsRecordFieldRules` observes a configured raw 429 as
  `outcome:"matched"`, `status:429`, `error:null`, `matchedStubId:"rate-limited"`;
  non-matched outcomes carry `fake_jev_<code>` (`unmatched` →
  `fake_jev_unmatched_request`).

### C-CTRL-007 — §41.8 clear request history

`TestQAClearRequestsEffects` observes `DELETE /__fake/v1/requests` returning
`204` with an empty body and no `Content-Type`. Afterwards:

- history is `{"requests":[]}` and the next admitted request is recorded with
  `sequence: 1`;
- the interaction-derived `unknown_route` failure is gone while the unsatisfied
  `expect_exactly` on the untouched stub `wide` is retained (verify reports only
  that expectation failure);
- stub invocation counts and sequence positions are unchanged: the sequenced
  stub `flow` still has `InvocationCount 1`, `SequencePosition 1`, and its next
  selected call returns element index 1 (`0`), not element zero (`1`) — proving
  the position was not rewound;
- `TestQAClearDoesNotRemoveDynamicStubs` observes a registered dynamic stub
  still present after the clear (only full reset removes it).

`TestQAClearRequestsEndsJournalFull` observes that after `maxInteractions: 1`
overflows (sticky `journal_full`, second request `507`), clearing history ends
the sticky state and the next request is admitted with `sequence: 1` (not `507`).

### C-CTRL-008 — §41.9 verify

- **Always 200 when calculable; exact shapes.**
  `TestQAVerifyPassShapeStableAnd200` observes `200 application/json` with body
  exactly `{"passed":true,"failures":[]}`, stable across repeated calls.
  `TestQAVerifyRepeatedCallsStableWithFailures` observes a non-clean result is
  also stable across three calls and does not clear the engine failure state.
- **Failure item shape.** `qaAssertFailureShape` requires exactly the four
  members `code, message, requestSequence, stubId`; `message` is a non-empty
  string; `requestSequence` is a number or null; `stubId` is a string or null.
  Applied to every failure observed in every verify probe.
- **Ordering groups (§41.9).** `TestQAVerifyOrderingGroups` observes
  `unknown_route(#1)`, `unmatched_request(#2)` (interaction-derived, ascending),
  then `expect_exactly(stubId A)`, `expect_at_least(stubId B)` (expectations by
  registration index).
- **Raw-error exemption (§11.4).** `TestQAVerifyRawErrorExemption` observes a
  configured raw 429 yields `passed:true, failures:[]`.
- **Required failure codes reached.**
  `TestQAVerifyAllReachableCodes` observes each of: `unmatched_request`,
  `unknown_route`, `validation_error`, `payload_too_large`,
  `sequence_exhausted` (with `stubId`), `invalid_stub_response` (with `stubId`),
  `journal_full` (`requestSequence:null`), `expect_exactly`, `expect_at_least`,
  `expect_at_most`. `internal_error` is **not** reachable through the public
  host surface (the `jev/v1` profile always populates `exchange.Payload` so the
  `exchange.Payload.(v1.Request)` assertion in `serveData` cannot fail); the
  probe therefore drives `engine.RecordFailure(..., "fake_jev_internal_error", ...)`
  directly and observes `/verify` render `code:"internal_error"` with the request
  sequence. See residual risks.

### C-CTRL-009 — §41.10 full reset

- **204 empty body.** `TestQAResetSixEffects` observes `204`, empty body, no
  `Content-Type`.
- **Six effects.** Same probe observes, after one `POST /__fake/v1/reset`:
  (1) the dynamic stub is absent from the list and unmatchable; (2) static
  `Invocations` are 0; (3) static `SequencePosition` is 0 and the next selected
  call returns element index 0; (4) history is `{"requests":[]}` and
  interaction-derived failures are gone; (5) the next admitted request is
  recorded with `sequence: 1`; (6) the next dynamic registration returns
  `registrationIndex: 3` for two static stubs (`N+1`). Static stubs remain at
  registration indexes 1 and 2 with unchanged configuration.
- **Atomicity (§17.4).** `TestQAResetRacesDataPlane` runs 200 data-plane/reset
  races and observes only the two legal states: either the reset wiped the
  interaction and zeroed the stub, or the completed interaction is present with
  `sequence: 1`, `matchedStubId:"a"`, and `InvocationCount 1`. No partially reset
  state was observed. The full `-race` run (below) reported no data races.
- **§44.8 / C-GOLD-008 vector outcome.** `TestQAResetVectorPostResetExpectation`
  reproduces the vector: two static stubs (`static-a` `expect.exactly:2`), one
  dynamic stub, reset, then the post-reset calls and `verify` observing
  `{"passed":false}` with a single `expect_exactly` failure for `static-a`
  (`requestSequence:null`). Consistent with the committed
  `testdata/contracts/reset/01-full-reset.json` `verify.expectFailures`.

### C-SEQ-006 — reset rewinds sequences and removes dynamic stubs

`TestQAResetSixEffects` observes sequence position 0 after reset and the first
element returned again; the dynamic stub is removed. (Overlaps C-CTRL-009;
observed through the same probe.)

### C-SEQ-008 — unsatisfied invocation expectations

`TestQAVerifyAllReachableCodes/expect_exactly/at_least/at_most` observes
`expect_exactly`, `expect_at_least`, `expect_at_most` failure codes with
`stubId` set to the stub and `requestSequence:null`; satisfied `exactly:1` after
one call yields `passed:true, failures:[]`.

### C-GOLD-008 — §44.8 golden reset vector

`TestQAResetVectorPostResetExpectation` executes the vector's observable steps
and endpoint outcomes (dynamic removed, counters/positions zeroed, history
empty, next sequence 1, next dynamic index 3, post-reset `expect_exactly`
failure). No committed-vector runner exists yet (FJ-046); this is an
independent execution of the vector's semantics, not a runner.

## Clone-exchange fix observations

`TestQAMetadataSurvivesAndIsIsolated` constructs an exchange with
`Metadata["method"]="GET"`, `Metadata["target"]="/v1/models"`, and a nested map
value; after `Engine.Transition` the stored `InteractionRecord.Request.Metadata`
retains the keys, and mutating the caller's top-level map and nested container
afterwards does not change the stored record. `TestQANilMetadataStaysNil`
observes a nil `Metadata` map stays nil in the stored record. This independently
confirms the scope-amendment-2 `cloneExchange` fix.

## Control-plane isolation

`TestQAControlTrafficNotJournaled` issues health, meta, stubs (GET/POST/DELETE),
requests (GET/DELETE), verify, and reset through `control.API` and observes
`Engine.Interactions()` and `Engine.VerificationFailures()` both empty. It also
observes the host's own `/__fake/` handling returns `404` without journaling.

## Commands run and outcomes

| command | outcome |
| --- | --- |
| `go test -overlay=/private/tmp/fj021-qa-final/overlay.json ./internal/control -run TestQA -count=1 -v` | exit 0; 23 top-level probes + 16 subtests, all matched their expectations |
| `go test -race -overlay=/private/tmp/fj021-qa-final/overlay.json ./internal/control -run TestQA -count=1` | exit 0; no race reports |
| `go test ./internal/control -count=1` | exit 0 (`ok fake-jev/internal/control`) |
| `go test ./internal/engine -count=1` | exit 0 (`ok fake-jev/internal/engine`) |
| `go test ./... -count=1` | exit 0 (all 8 packages `ok` or `[no test files]`) |
| `go test -race ./... -count=1` | exit 0; no race reports |
| `go vet ./...` | exit 0, no findings |
| `go run ./cmd/guard arch` | exit 0; `findings: []`, 48 files / 9 packages |
| `go run ./cmd/guard lint` | exit 0; `findings: []`, 48 files / 9 packages |
| `go run ./cmd/guard trace` | exit 0; `findings: []`, `active: 0`, `covered: 17`, `test_files: 16` |
| `go run ./cmd/guard fuzz` | exit 0; `findings: []`, `targets: 0` |
| `./scripts/verify-candidate FJ-021` | exit 0; its own summary reported 20 rows result=`pass`, 0 result=`fail`, 0 `skipped`, 0 `not_applicable`; report written to `.agent/reports/FJ-021/report-20260929T163211Z.json` |
| `./scripts/candidate-fingerprint candidate` | `59598e94499eea493d9aff2662f2a91689a2a461b2e3202fd0070fc60a13b79e` |
| `./scripts/candidate-fingerprint task .agent/work/FJ-021/state.json` | `taskFingerprint: 44caf21dca360d2cfe46347601c2250e25a6181e2015b36830e535f50e95c0a5` |

No repository source/test/state file was created or edited by this stage. The
overlay probes live only in `/private/tmp/fj021-qa-final/`; `git status` after
the probes showed no new or modified repository files. `git diff --cached
--name-only` is empty (nothing staged). The one repository write is the
verify-candidate report named in the table, produced by the required command
itself.

## Residual risks and evidence limits

1. **Unresolved `needs-human` blocker — `testdata/contracts/**` vs §41.7
   `requestBody`.** Independently re-confirmed: the 15 valid-JSON-body `journal`
   assertions listed by the coder across 13 vectors still assert
   `"requestBody": null`, while the implementation renders the parsed JSON value
   for those bodies (observed by the probes above and by
   `testdata/contracts/reset/01-full-reset.json` step 5, which the implementation
   would render as an object). §41.7's prose, its example, and
   `spec/control-api.openapi.yaml` require the parsed value; the committed
   vectors contradict that. This is the open `blockers[0]` in
   `.agent/work/FJ-021/state.json` (owner: repo owner) and remains unresolved;
   any runner that strictly compares `journal.requestBody` will disagree with
   the implementation until the owner adjudicates. QA did not edit the vectors
   (outside allowed files) and did not treat the divergence as a new failure.
2. **§41.9 null-sequence ordering is spec-silent.**
   `TestQAVerifyNullSequenceOrderingObserved` observes the order sequenced
   interaction failures ascending, then the `journal_full` null-sequence sticky
   failure, then expectation failures. §41.9 numbers only two groups
   (interaction-derived ascending; expectation failures by registration index)
   and separately permits `requestSequence:null` "when no journal sequence was
   assigned", without placing that third case. Both orders satisfy the numbered
   groups, and no committed vector discriminates (`verification/02-journal-full.json`
   has a single `journal_full` failure). Recorded as a specification silence,
   not adjudicated.
3. **`internal_error` is not reachable through the public surface.** The
   `serveData` internal-error decision (failed `exchange.Payload.(v1.Request)`
   assertion) cannot trigger because the `jev/v1` profile always populates
   `Payload` for `systemone`. The verify rendering of the `internal_error` code
   was observed only via a synthetic `engine.RecordFailure` call. The database
   code path is therefore covered by shape observation, not by a host-reachable
   reproduction.
4. **A racing `invalid_stub_response` failure can outlive a reset.** The host
   records that failure through `Engine.RecordFailure` after `Transition`
   returns, in a second engine mutation. A reset landing between the two calls
   would clear the journal and failure list and the subsequent `RecordFailure`
   would re-add a failure for a sequence the reset cleared. This is the
   hardener's recorded follow-up, pre-existing and outside `allowedFiles`; QA did
   not reproduce it deterministically (the data-plane/reset race probe used the
   matched path only).
5. **`profile:null` is not reachable through the host.** The host always
   identifies the active profile before routing, so every host-journaled record
   carries `profile:"jev/v1"`; `profile:null` was observed only by driving the
   engine boundary with an empty exchange. Whether an unknown route *should*
   record the configured active profile is a specification silence noted by the
   specifier, not resolved here.
6. **`error` for `validation_error` is a spec ambiguity.** The implementation
   records `error:"fake_jev_validation_error"`; §21.2 lists that token as a
   verification classification while §39.3 returns a Jev-shaped 422 envelope
   with no `error` member. Observed as implementation behavior, not adjudicated.
7. **`path` excludes the query string.** Observed `GET /v1/models?x=1` →
   `path:"/v1/models"`, matching the committed
   `matching/05-routes-exact-paths.json`; §41.7 does not itself define the
   query-string treatment.
8. **Observable surface is `control.API`, not the HTTP listener.** As recorded
   by the specifier/coder/hardener, the HTTP host still answers every `/__fake/`
   path with its own `404 fake_jev_control_not_found`; the four endpoints are
   reachable only through `control.API`. The probes bind `control.API` to the
   real data-plane host, which is the same seam the repository's tests use.
9. **Timestamp precision.** Records render `timestamp` with RFC3339Nano
   precision; §41.7 declares it diagnostic with implementation-chosen
   fractional precision, so only parseability was checked.
10. **No committed-vector runner exists** (FJ-046/tracked G-Q exemption), so
    C-GOLD-008 was exercised against an in-process reproduction of the vector's
    semantics rather than the vector file.

This artifact records observable behavior and evidence limits only; it declares
no acceptance verdict.

## Re-bind re-verification — vector `requestBody` correction (candidate 4acd6dd6)

The only change since the previous QA stage is the correction of 15 journal
`requestBody` assertions across 13 `testdata/contracts` vectors: each now asserts
the parsed request body instead of `null`, per the authoritative §41.7. This
section records an independent re-verification of that correction; it supersedes
the `requestBody: null` premise of residual risk 1 for those 15 assertions.

### Method

- Probed entirely outside the repository under `/private/tmp/fj021-qa-rebind`; no
  repository source, test, vector, or state file was added or edited by the
  probes.
  - Static comparator `verify_vectors.py` walks every `testdata/contracts` vector
    and derives the §41.7 `requestBody` from each step's own request (parsed
    value for a valid JSON body, raw string for a malformed `rawBody`, `null` for
    no body or an oversized body) and compares it to the step's journal
    assertion.
  - Behavioral probe `qa_rebind_probe_test.go` (package `control`) is injected
    with `go test -overlay=overlay.json`; a smoke overlay confirmed the mechanism
    before the real probes ran. It replays every committed data-plane vector step
    through the real `internal/host/http` server + `internal/control` API sharing
    one engine and compares the emitted journal record's `requestBody` to the
    vector assertion.
- Candidate fingerprints re-derived from the worktree:
  `./scripts/candidate-fingerprint candidate` =
  `4acd6dd6c828af008d216a52f543aa571806355cf45bac0e71937bf5588d49ab`;
  `./scripts/candidate-fingerprint task .agent/work/FJ-021/state.json` =
  `taskFingerprint 4559c4633576304d61378f9e05d4fd68a8a1d9db7fddfed3188c00e4a9adc84e`.

### Observations

- **All 15 corrected assertions equal their step's own request body.** The static
  comparator found exactly 15 parsed-body assertions across the 13 listed files
  and each equals `step.request.body`; the behavioral replay reproduced the same
  15 vectors and observed the implementation emit exactly the asserted value.
  Files: `matching/{01,02,03,06}`, `mixed/02`, `reset/01`, `sequences/01` (three
  assertions: steps 0, 1, 2), `validation/{02,03,04}`,
  `verification/{01,02,04}`.
- **Legitimate nulls (requests with no body) unchanged.** The only `null`
  journal assertions are the five no-body steps in
  `matching/05-routes-exact-paths.json` (steps 0–3) and `models/01-default-models.json`;
  each carries no `body`/`rawBody`, and all remain `null`. The behavioral replay
  observed the implementation also emit `null` for those records.
- **`validation/01-malformed-json` still expects the raw string.** Its step 0
  asserts `"requestBody": "{"`; both the static comparator and the behavioral
  replay observed the malformed body render as the raw decoded string `"{"`,
  unchanged.
- **`testdata/contracts/verification/03-payload-too-large.json` is defective and
  unpatched.** Its step 0 request body is 88 bytes against a configured
  `dataPlaneBodyBytes` of 1024, so the body is neither malformed nor oversized
  and cannot produce the vector's own expected `413`; its `"requestBody": null`
  is therefore not justified by the §41.7 oversized rule. Independently
  re-confirmed: the implementation would emit the parsed value for that body.
  Left unpatched (outside this stage's scope; needs a dedicated vector-audit
  ticket). The behavioral replay excludes this single vector for that reason.
- **Host `/__fake/` delegation is unwired.** Independently re-confirmed: the
  HTTP host answers every `/__fake/` path with its own `404
  fake_jev_control_not_found` and does not delegate to the control API; the four
  §41.7–§41.10 endpoints are reachable only through `control.API`. Recorded
  (as specifier/coder/hardener do) as a separate gap, out of scope for FJ-021.
- **§41.7 endpoint behavior re-confirmed** by the behavioral probe: exact
  11-member record shape; `timestamp` parses as RFC3339Nano; `sequence`
  ascending 1..N; valid JSON body → parsed value; malformed `{` → raw string;
  no body → `null`; oversized body → `null` with `outcome:"payload_too_large"`;
  matched request carries `error:null` and a `matchedStubId`; `method`/`path`
  from host-stamped `Metadata` with the query string excluded
  (`GET /v1/models?x=1` → `path:"/v1/models"`); empty history body exactly
  `{"requests":[]}`.
- **§41.9 ordering re-confirmed** by the behavioral probe: interaction-derived
  failures ascending by request sequence (`unmatched_request` #1,
  `unknown_route` #2) come before expectation failures in stub registration
  index ascending (`expect_exactly` A, `expect_at_least` B); every failure item
  has exactly the four members. A `maxInteractions:1` overflow was re-observed to
  yield a `journal_full` failure with `requestSequence:null` placed after the
  sequenced interaction failures and before the expectation failures
  (spec-silent placement, as recorded in residual risk 2).

### Commands run and outcomes

| command | outcome |
| --- | --- |
| `python3 verify_vectors.py` (static comparator) | exit 0; 22 journal `requestBody` assertions scanned, 15 corrected parsed-body assertions equal their step request body, 5 no-body nulls unchanged, malformed raw string `"{"` unchanged, 1 defective null flagged, 0 mismatches within the 13 corrected files |
| `go test -overlay=/private/tmp/fj021-qa-rebind/overlay.json ./internal/control -run TestQA2 -count=1 -v` | exit 0; 6 probes pass; behavioral replay of 16 vectors matches the implementation, excluding the defective `verification/03-payload-too-large.json` |
| `go test -race -overlay=/private/tmp/fj021-qa-rebind/overlay.json ./internal/control -run TestQA2 -count=1` | exit 0; no race reports |
| `go test ./... -count=1` | exit 0 (all packages `ok` or `[no test files]`) |
| `go test -race ./... -count=1` | exit 0; no race reports |
| `go vet ./...` | exit 0, no findings |
| `./scripts/verify-candidate FJ-021` | exit 0; verifier summary `46 passed, 2 failed, 3 skipped, 0 not_applicable` before this artifact's front matter was refreshed, then `48 passed, 0 failed, 3 skipped, 0 not_applicable` after; report written to `.agent/reports/FJ-021/report-20260929T171338Z.json` |

The two pre-refresh failures were `stage qa task fingerprint` (stale
`taskFingerprint`) and `stage qa chain link` (stale `inputFingerprint`), both
the stale front matter this stage refreshes. No repository source/test/vector/
state file was created or edited by this stage; the probes live only in
`/private/tmp/fj021-qa-rebind/`. `git diff --cached --name-only` is empty
(nothing staged). The only repository writes are the two `verify-candidate`
reports produced by the required command (one per run, before and after the
front-matter refresh; the post-refresh one is named in the table).

This re-verification records observable behavior only; it declares no acceptance
verdict.
