---
stage: hardener
task: FJ-021
inputFingerprint: 4acd6dd6c828af008d216a52f543aa571806355cf45bac0e71937bf5588d49ab
outputFingerprint: 4acd6dd6c828af008d216a52f543aa571806355cf45bac0e71937bf5588d49ab
taskFingerprint: 4559c4633576304d61378f9e05d4fd68a8a1d9db7fddfed3188c00e4a9adc84e
gitHead: a76d577
generatedAt: 2026-09-29T17:06:19Z
author: worker/FJ-021-hardener-rebind-2
---

# Hardener — FJ-021: request history, clear, verify, full reset

## Candidate fingerprint

`./scripts/candidate-fingerprint candidate` after this stage:

```text
59598e94499eea493d9aff2662f2a91689a2a461b2e3202fd0070fc60a13b79e
```

Input candidate was `8b96804bcfd74e200491d856317e9d93606c3244e46492caf7cfbccdfb5d5494`.
Re-running the fingerprint after writing this artifact is unchanged: `.agent/work/`
is excluded from the candidate digest.

## Hardening actions taken

### H1 (required fix, cleaner F1) — `internal/control/verify.go` null-sequence ordering

`interactionFailures` documented "Sticky failures with no assigned sequence, such
as `journal_full` ... are ordered after the sequenced failures" while its
comparator did the opposite: with `if left == nil || right == nil { return
right != nil }`, `less(nilSeq, seq)` returned `true`, so a `requestSequence: null`
failure sorted *before* every sequenced failure. The engine appends `journal_full`
last, so the sort silently reinterpreted the engine's own chronological order.

Change (the only behavior-affecting line touched by this stage):

```go
if left == nil || right == nil {
	// A sequenced failure sorts before a null-sequence one; equal keys
	// (including two null keys) keep their recorded order.
	return left != nil
}
```

Rationale:

- `return left != nil` is the minimal predicate with the wanted totality:
  `nil` vs non-`nil` -> non-`nil` first; non-`nil` vs `nil` -> non-`nil` first;
  both `nil` / both non-`nil` -> `false`, so `sort.SliceStable` keeps the engine's
  recorded order for equal keys (the same property the two equal-sequence case
  already relied on).
- Resulting group order in `/verify` is now: sequenced interaction-derived
  failures ascending, then null-sequence interaction-derived sticky failures,
  then expectation failures by registration index — i.e. exactly what the doc
  comment and the coder artifact described, and the order that preserves the
  engine's chronological insertion order for `journal_full`.
- §41.9 fixes only the two numbered groups ("interaction-derived failures in
  request sequence ascending"; "expectation failures in stub registration index
  ascending") and is silent on where a `requestSequence: null` failure belongs,
  so the comparator was contradicting documented intent without violating the
  numbered groups. See the observation below for the recorded ambiguity.
- No committed vector distinguishes the two orders:
  `testdata/contracts/verification/02-journal-full.json` is the only vector
  carrying a `journal_full` failure and its `verify.expectFailures` has exactly
  one element (`journal_full`, `requestSequence: null`), so the fix leaves every
  golden expectation untouched.

Clauses served: §41.9 (failure ordering), C-CTRL-008, §11.4.

### H2 (required test) — `TestControlVerifyNullSequenceFailureOrdering`

Added to `internal/control/handlers_test.go`. The test builds an engine with a
zero-capacity journal (`engine.NewEngineWithJournal(stubs, 0)`) over two static
stubs carrying `expect.exactly: 1`, injects two sequenced failures in
non-ascending order through the public `Engine.RecordFailure` boundary
(`unknown_route` at sequence 7 first, `validation_error` at sequence 3 second),
then drives one `Transition` to force the sticky `journal_full` failure with
`requestSequence: null`. It asserts through the real `GET /__fake/v1/verify`
handler:

- exactly five failures, each with the fixed four-member shape, and positionally
  `validation_error` (sequence 3, `stubId: null`), `unknown_route` (sequence 7,
  `stubId: null`), `journal_full` (`requestSequence: null`, `stubId: null`),
  `expect_exactly` (`stubId: "first"`), `expect_exactly` (`stubId: "second"`) —
  i.e. the exact failure code order
  `validation_error,unknown_route,journal_full,expect_exactly,expect_exactly`,
  and expectation failures in registration-index order;
- no sequenced failure follows a null-sequence failure, and the sequenced
  values are strictly ascending.

The zero-capacity journal is what makes the sequenced input genuinely
out of order relative to natural recording; the host can never produce that
order (sequences are monotonic), which is precisely why the comparator needs the
`RecordFailure` path to exercise it.

Regression value confirmed empirically: with the old `return right != nil`
comparator restored, the test fails at
`handlers_test.go: failure code = journal_full, want validation_error`; with the
fix in place it passes. The comparator line is therefore pinned, not just
observed.

## Audit of the rest of the candidate (no further defect fix applied)

Each focus area below was checked against §§11.4, 17.4, 41.7-41.10, 43.x, 44.8
and the C-CTRL-006/007/008/009, C-SEQ-006/008, C-GOLD-008 rows. Where a concrete
defect existed it was fixed (H1); where the behavior already matched the clause,
no edit was made.

1. **Full reset atomicity vs in-flight transitions (§17.4).** `Engine.Reset`
   runs under `Engine.mu`, the same mutex every data-plane `transition` holds
   across reserve/route/select/mutate/record, so a racing request either
   completes before the reset or observes the reset state; the lock order
   `Engine.mu -> Registry.mu` is used identically by `Reset`,
   `RegisterDynamic`, and the transition handler, so the candidate introduces no
   lock inversion. The control handler (`internal/control/handlers.go reset`)
   additionally holds the package-level `stubMutations` lock that `createStub`
   holds across index capture, so a concurrent registration cannot capture a
   registration index across the reset. Re-verified under `-race`
   (`go test -race ./... -count=1`), plus the candidate's own
   `TestControlResetRacesDataPlane` / `TestControlClearRacesDataPlane`.
2. **All six §41.10 effects.** `Engine.Reset` filters dynamic stubs out, zeroes
   each static `InvocationCount` and `SequencePosition`, sets
   `nextRegistrationIndex = staticCount+1`, and `interactionJournal.clear()`
   wipes entries, the next sequence (back to 1) and the failure list; §44.8's
   `registrationIndex: 3` for two static stubs follows from `staticCount+1`.
   Pinned by `TestControlFullResetVector` (all six effects plus the post-reset
   `expect_exactly` re-evaluation) and `TestControlResetRewindsSequenceAndClearsFailures`.
   The candidate does not touch `internal/engine/engine.go`.
3. **`DELETE /__fake/v1/requests` (§41.8).** The handler only calls
   `Engine.ClearRequestHistory`, which only calls `interactionJournal.clear()`;
   no registry path is reachable, so stub counters and response-sequence
   positions are untouched while the interaction sequence restarts at 1 and the
   interaction-derived failures (including the sticky `journal_full`) end.
   Pinned by `TestControlClearRequestsEffects` and
   `TestControlClearRequestsEndsJournalFull`.
4. **Verify raw-error exemption (§11.4).** `verify` derives failures exclusively
   from `Engine.VerificationFailures()` (sticky, engine-recorded) and
   `Engine.EvaluateExpectations()`; it never derives a failure from an
   interaction record, so an intentionally configured raw 429 stays
   `outcome: matched` / `error: null` and produces no failure. Pinned by
   `TestControlVerifyRawErrorExemption` and the raw-429 row of
   `TestControlRequestHistoryRecordRules`.
5. **Control traffic never enters the journal (§11.1, §41.1).** The control
   API never calls `Transition`, and the host returns before `serveData` for
   every `/__fake/` path. Pinned by the journal-length assertions in
   `TestControlHealthMetaAndUnsupportedRequests`,
   `TestControlRequestHistoryRecordRules`, `TestControlBodyLimit`, and
   `TestControlNewEndpointMethods`.
6. **Nil safety.** `listRequests` and `verify` call `a.engine` methods without an
   explicit guard; `Interactions`, `VerificationFailures`, and
   `EvaluateExpectations` all check `e == nil` first, so the calls are safe.
   Confirmed with a temporary in-package probe (created, run, deleted; not part
   of the candidate): a nil-engine API answers `200 {"requests":[]}` and
   `200 {"passed":true,"failures":[]}` with `Content-Type: application/json`, and
   a non-prefixed path answers the ordinary `404 fake_jev_control_not_found`.
   `metadataString` reads a missing key from a nil map and falls back to `""`;
   `stampRequestMetadata` nil-checks the exchange and request and allocates
   `Metadata` before writing.
7. **Body-limit handling (§41.1, §11).** The control handler wraps the body in
   `http.MaxBytesReader` with `metadata.Limits.ControlPlaneBodyBytes` (default
   `2097152` when unset); an over-limit body surfaces as `*http.MaxBytesError`,
   which `createStub` maps to `413 fake_jev_payload_too_large` while a malformed
   in-limit body maps to `400 fake_jev_bad_control_request`. Pinned for both
   known and unknown `Content-Length`, with a read-count assertion bounded by
   `limit+1`, by `TestControlBodyLimit`.
8. **`cloneExchange` defensive-copy contract (`internal/engine/journal.go`).**
   The amendment-2 fix builds a local map and assigns it after the range, so the
   nil guard is unchanged (nil `Metadata` stays nil), every value still goes
   through `cloneValue` (deep copy of `json.RawMessage`/`[]byte` and, via
   marshal + `UseNumber` decode, of nested containers), and the returned struct's
   map is private to the record because the receiver is a struct copy. Pinned by
   `TestEngineExchangeMetadataSurvivesTransition`, which observes the stored
   `method`/`target` values after mutating the caller's map. Not widened by this
   stage: the contract holds as written, so no further assertion was added.

No other concrete defect was found inside the allowedFiles; nothing was staged
(`git diff --cached --name-only` is empty).

## Observations / ambiguities (recorded, not adjudicated)

- **§41.9 is silent on the position of null-sequence interaction-derived
  failures.** The section numbers two groups only ("interaction-derived failures
  in request sequence ascending"; "expectation failures in stub registration
  index ascending") and separately permits `requestSequence: null` "when no
  journal sequence was assigned", without placing that third case. Both group
  orders (null-sequence first or last) satisfy the numbered groups, and no
  committed vector discriminates between them. F1 was resolved by applying the
  behavior the code comment and the coder artifact already documented
  (sequenced, then null-sequence, then expectations), which also matches the
  engine's insertion order for `journal_full`. If the owner prefers the other
  reading, the comparator predicate and `TestControlVerifyNullSequenceFailureOrdering`
  are the two places to change.
- **Naming overlap for "target".** In `internal/host/http/server.go` the
  metadata key `"target"` carries `request.URL.Path` (§41.7 `path`, query
  excluded) while `v1.RequestMeta.Target` carries `requestTarget(request)` =
  `URL.RequestURI()`. Both are intentional and documented at their own sites;
  noted only because the same word means two things in one file.
- **Unreachable internal-error branch does not fail verification.**
  `internal/host/http` records `outcome: internal_error` for the
  `exchange.Payload.(v1.Request)` type-assert failure without attaching an
  engine `VerificationFailure`, so that branch would not surface in `/verify`
  even though §11.4 lists "a fake-server internal failure" as a failure
  condition and §41.9 lists `internal_error` as a required code. The branch is
  not reachable through the public surface (the profile always populates
  `Payload` for `systemone`), the behavior predates this item, and adding the
  failure would assert a condition the specifier did not claim, so it is
  recorded here rather than changed under a frozen scope.

## Commands run and outcomes

| command | result |
| --- | --- |
| `gofmt -w internal/control/verify.go internal/control/handlers_test.go` | applied |
| `gofmt -l internal/control internal/engine internal/host/http` | empty output, exit 0 |
| `go test ./internal/control -count=1` | `ok fake-jev/internal/control 0.281s` |
| `go test ./internal/engine -count=1` | `ok fake-jev/internal/engine 2.837s` |
| `go test ./... -count=1` | all packages `ok` (cmd/guard, compat/jev/v1, config, control, engine, host/http, test/integration) |
| `go test -race ./... -count=1` | all packages `ok`, no race reports |
| `go vet ./...` | exit 0, no findings |
| `go run ./cmd/guard arch` | exit 0, `findings: []`, 48 files / 9 packages |
| `go run ./cmd/guard lint` | exit 0, `findings: []`, 48 files / 9 packages |
| `go run ./cmd/guard trace` | exit 0, `findings: []`, `active: 0`, `covered: 17` |
| `go run ./cmd/guard fuzz` | exit 0, `findings: []`, `targets: 0` |
| `./scripts/candidate-fingerprint candidate` | `59598e94499eea493d9aff2662f2a91689a2a461b2e3202fd0070fc60a13b79e` |
| old-comparator control run (`return right != nil` restored) | `--- FAIL: TestControlVerifyNullSequenceFailureOrdering`, then reverted |

`./scripts/verify-candidate FJ-021` was not run by this stage (verification
evidence is bound to a revision by the verification tooling, and this stage does
not own `state.json` or the reports directory).

## Residual risks

- **Open `needs-human` blocker: `testdata/contracts/**` vs §41.7 `requestBody`.**
  §41.7's prose, its example, and `spec/control-api.openapi.yaml` require the
  parsed JSON value for valid JSON bodies, while ~15 `journal` assertions in 13
  committed vectors (including `reset/01-full-reset.json`,
  `verification/01-raw-error-not-a-failure.json`, `verification/02-journal-full.json`)
  assert `"requestBody": null`. The recorded position implements §41.7 and defers
  vector corrections to the repo owner; the divergence is unchanged by this stage
  and remains a separate open blocker for whoever builds the vector runner.
- **§41.9 null-sequence placement.** F1 applies the documented intent; a
  different owner reading would require reverting the comparator and the new
  test together (see the observation above).
- **Post-reset failure recording for a racing invalid-stub-response request.**
  `serveData` records `invalid_stub_response` through `Engine.RecordFailure`
  *after* `Transition` returns, i.e. in a second engine mutation outside the
  transition's serialized section. A reset landing between the two calls clears
  the journal and the failure list, and the subsequent `RecordFailure` then
  re-adds an `invalid_stub_response` failure for a sequence the reset already
  cleared, so `/verify` could report a failure that the reset intended to wipe.
  This predates the candidate (the call sites are unchanged by the diff,
  `record.Sequence`'s `UpdateInteraction` path is a harmless no-op after a clear)
  and cannot be fixed inside the allowedFiles: it needs either the outcome
  recording moved inside the engine transition or an engine-side guard, both in
  `internal/engine/engine.go`. Recorded for a follow-up item rather than patched
  under frozen scope. The candidate's `TestControlResetRacesDataPlane` exercises
  only the matched path, which is the case this does not affect.
- **`method`/`path` seam duplication (cleaner F2).** The `"method"`/`"target"`
  literals are declared in both `internal/control/handlers.go` and
  `internal/host/http/server.go`; they must agree or §41.7 renders empty strings.
  Both sides are pinned by tests, but the coupling remains.
- **Host `/__fake/` delegation is still unwired.** The listener answers every
  control path with its own `404 fake_jev_control_not_found`; the endpoints
  hardened here are reachable only through `control.API`. This is the recorded
  separate gap, unchanged by this stage.
- **`internal/control/verify.go` is a new untracked file.** It is part of the
  candidate fingerprint via the untracked listing; a later `git add` does not
  change the digest, but a stray `git clean` would.

## Re-bind (metadata-only refresh, 2026-09-29T17:06:19Z)

Re-bound after the `testdata/contracts` correction: 15 `journal` `requestBody`
assertions across 13 vectors were changed from `"requestBody": null` to the
parsed JSON value (§41.7). The earlier F1 ordering fix was already in place; no
source edit was made in this stage (metadata-only, per the approved direction).

Re-checked:

- **F1 ordering fix present** in `internal/control/verify.go`: the
  `interactionFailures` comparator still returns `left != nil` for the
  nil-sequence branch, so sequenced interaction-derived failures sort before
  null-sequence ones (e.g. `journal_full`) and both precede the expectation
  failures appended by `verify`. Unchanged.
- **F1 pinning test passes**:
  `go test ./internal/control -count=1 -run TestControlVerifyNullSequenceFailureOrdering`
  — green (2 focused tests passed together with the cloneExchange pin).
- **`cloneExchange` Metadata fix intact** in `internal/engine/journal.go`: the
  nil guard is unchanged and the private map is still built locally and assigned
  after the range; `TestEngineExchangeMetadataSurvivesTransition` passes.
- **Focused suites re-run**: all green (see table below).
- **Candidate fingerprint re-run**: `4acd6dd6c828af008d216a52f543aa571806355cf45bac0e71937bf5588d49ab`
  — matches the binding recorded in the front matter.

| command | result |
| --- | --- |
| `go test ./internal/control -count=1` | `ok` (81 passed) |
| `go test -race ./internal/control -count=1` | `ok` (81 passed), no race reports |
| `go vet ./...` | exit 0, no findings |
| `./scripts/candidate-fingerprint candidate` | `4acd6dd6c828af008d216a52f543aa571806355cf45bac0e71937bf5588d49ab` |

Residual risks are unchanged from the stage above. In particular the recorded
`needs-human` blocker remains: **`testdata/contracts/verification/03-payload-too-large.json`
is defective and unpatched** — line 71 still asserts `"requestBody": null` for a
journaled interaction, contradicting §41.7's parsed-JSON requirement. It was not
corrected by the 13-file pass and stays open for the repo owner / vector runner.
