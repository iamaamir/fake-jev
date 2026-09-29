---
stage: cleaner
task: FJ-021
inputFingerprint: 4acd6dd6c828af008d216a52f543aa571806355cf45bac0e71937bf5588d49ab
outputFingerprint: 4acd6dd6c828af008d216a52f543aa571806355cf45bac0e71937bf5588d49ab
taskFingerprint: 4559c4633576304d61378f9e05d4fd68a8a1d9db7fddfed3188c00e4a9adc84e
gitHead: a76d577
generatedAt: 2026-09-29T17:05:13Z
author: worker/FJ-021-cleaner-rebind-2
---

# Cleaner — FJ-021: request history, clear, verify, full reset

## Changes applied

**None.** No file outside `.agent/work/FJ-021/cleaner.md` was created, edited,
or deleted by this stage. The candidate fingerprint is unchanged
(`8b96804bc...`, re-checked after the review; see commands below).

Reason: the parent pinned the candidate fingerprint for this stage, and every
cleanup considered below lands in an in-fingerprint file (`internal/**`), so any
edit would invalidate the pinned `inputFingerprint` and stale the coder artifact
as well as the downstream hardener/QA artifacts. Per the cleaner stage pack a
structural edit is only made when it is justified *and* re-verified against a new
candidate; with the fingerprint frozen, the review is reported instead of applied.

## Files inspected

- `internal/control/handlers.go` (new routes, `requestRecord`,
  `listRequests`, `clearRequests`, `reset`, `metadataString`, `stringPointer`,
  `failureCodeForOutcome`, `requestBodyValue`, metadata-key consts)
- `internal/control/verify.go` (new: `verificationFailure`,
  `verificationResult`, `verify`, `interactionFailures`, `expectationFailures`)
- `internal/control/handlers_test.go` (all added tests and helpers)
- `internal/control/api.go` (unchanged; read for package conventions)
- `internal/host/http/server.go` (`decide` closure split,
  `stampRequestMetadata`, metadata-key consts)
- `internal/engine/journal.go` (`cloneExchange`)

## Findings

### F1 — `verify.go` ordering doc comment disagrees with the comparator (comment accuracy)

`interactionFailures` documents "Sticky failures with no assigned sequence, such
as `journal_full`, carry `requestSequence: null` and are ordered after the
sequenced failures." The comparator is:

```go
if left == nil || right == nil {
    return right != nil
}
```

`less(nilSeq, seq)` returns `true` (because `right != nil`), so a
`requestSequence: null` failure sorts **before** sequenced failures, not after.
Reproduced outside the repository with the same comparator over
`{unknown_route(1), journal_full(nil), unmatched_request(2)}`: the output order is
`journal_full, unknown_route, unmatched_request`. The engine itself appends
`journal_full` last (`interactionJournal.setJournalFull` appends to
`j.failures`; `verificationFailures` copies in slice order), so the sort reverses
the engine's natural chronological order for that sticky failure.

- §41.9 fixes only "interaction-derived failures in request sequence ascending;
  then expectation failures", and explicitly allows `requestSequence: null` when
  no sequence was assigned, so neither order is forbidden by the specification
  text.
- No test pins the relative position of a null-sequence failure: the only test
  containing `journal_full` (`TestControlClearRequestsEndsJournalFull`) asserts a
  single-element `failures` array. `TestControlVerifyFailureOrdering` has four
  sequenced/expectation failures only.
- The coder artifact states the opposite of the code
  (".agent/work/FJ-021/coder.md`: "Sticky failures with no sequence
  (`journal_full`) are ordered after the sequenced interactions"), so the
  written record is inconsistent with the shipped behavior.

Not changed: aligning code with the comment (or the comment with the code) is
behavior-affecting or at minimum fingerprint-affecting, and §41.9 does not
determine which is intended. Recorded for the parent/hardener rather than
resolved unilaterally.

### F2 — Metadata key literals declared twice, cross-referenced by comment (duplication, reviewed and accepted)

`requestMethodMetadataKey`/`requestTargetMetadataKey` are declared identically in
`internal/control/handlers.go` and `internal/host/http/server.go`, each with a
comment pointing at the other (`"internal/host/http writes the same keys"` /
`"Keep in sync with internal/control."`). The literals are pinned from both sides
by tests (`TestControlRequestHistoryRecordRules` asserts `method`/`path` values
through the control endpoint; the host side is exercised by the same harness).

Alternatives weighed and not applied:

- define in `internal/engine` — contradicts the approved scope amendment ("the
  engine core and its journal type stay untouched") and would put HTTP detail in
  the provider-neutral core;
- export from `internal/control` and import in `internal/host/http` — makes the
  host depend on the control package for a naming detail before the (separate,
  recorded-gap) `/__fake/` delegation is wired, i.e. a new coupling for two
  string literals;
- a shared neutral package — new package for two constants.

Two literals with mutual cross-references and test coverage is the smaller
change, and any consolidation changes the fingerprint. Reported as drift risk
only.

### F3 — `newRequestRecord` uses two spellings for one null rule (minor consistency)

`Profile`/`Operation` use the `stringPointer` helper; `MatchedStubID` re-implements
the same empty-string → `nil` rule inline:

```go
if record.MatchedStubID != "" {
    stubID := record.MatchedStubID
    item.MatchedStubID = &stubID
}
```

`item.MatchedStubID = stringPointer(record.MatchedStubID)` is behavior-identical
(empty value → `nil`, non-empty → pointer to a copy). Cosmetic only; not changed
(fingerprint).

### F4 — `requestBodyValue` resembles `decodeValue` but must not share it (reviewed, no action)

Both decode one JSON value with `UseNumber`, but `requestBodyValue` additionally
requires EOF after the first value so that a trailing token falls back to the raw
string, while `decodeValue` accepts only the first value. Reusing `decodeValue`
would change `requestBody` for bodies like `{} {}` from the raw string to `{}`, so
the apparent duplication is required by §41.7 and should not be collapsed.

### F5 — `decide` + wrapper closure in `serveData` (reviewed, no action)

The split into `decide` (route/validate/payload decisions) plus one wrapping
closure that stamps metadata is the lowest-duplication way to stamp all seven
decision returns before journaling; stamping inside each branch would repeat
`stampRequestMetadata` seven times, and stamping after `Transition` returns is not
possible (the decision is built inside the callback). `_ = sequence` is
pre-existing and retained. Control flow reads linearly.

### F6 — `metadataString` / nil-engine guards (reviewed, no action)

`metadataString` reads a missing key from a nil map safely and falls back to `""`
(never panics); `stampRequestMetadata` nil-checks both the exchange and the
request and initializes `Metadata` before writing. `listRequests`/`verify` call
`a.engine.Interactions()`/`VerificationFailures()` without the `a.engine != nil`
guard used by `clearRequests`/`reset`; both engine methods have nil-receiver
guards returning `nil`, so this is safe and `Interactions()` on a nil engine
already renders `{"requests":[]}`. The inconsistency with `listStubs` is
stylistic.

## `cloneExchange` defensive-copy contract (explicit check)

`internal/engine/journal.go` now reads:

```go
if exchange.Metadata != nil {
    metadata := make(map[string]Value, len(exchange.Metadata))
    for key, value := range exchange.Metadata {
        metadata[key] = cloneValue(value)
    }
    exchange.Metadata = metadata
}
```

- **nil stays nil**: the whole block is guarded by `exchange.Metadata != nil`; a
  nil input map leaves the copied struct's field nil. Matches the pre-change
  intent (the old code also only entered the block for non-nil maps).
- **nested Values deep-cloned**: each value goes through `cloneValue`
  (`internal/engine/matcher.go:164`), which copies `json.RawMessage`/`[]byte` and
  re-decodes containers via `json.Marshal` + `UseNumber` decode, so nested
  maps/slices/arrays in metadata are detached from the caller's tree. Same helper
  and same call site as before; only the destination map changed.
- **caller mutation isolation**: the receiver is a struct copy, so assigning
  `exchange.Metadata` cannot touch the caller's struct; the new map and its
  cloned values are private to the returned copy. `TestEngineExchangeMetadataSurvivesTransition`
  mutates the caller's map after `Transition` and observes the stored record
  unchanged.
- **What the fix changes**: the previous code reassigned the field to a fresh
  map *before* ranging over it, so the range iterated the new empty map and every
  non-nil `Metadata` was replaced by an empty map. The fix preserves the
  entry set and the deep-copy semantics; it does not add fields, does not change
  `InteractionRecord`, and does not touch the nil case. Previously dropped keys
  (host `Metadata["error"]`, now also `method`/`target`) survive into the record.
  `Metadata` is not read by the matcher/registry (grep: only `exchange.go`
  declares it and `journal.go` clones it), so no matching behavior depends on
  this.

## Commands run

| command | result |
| --- | --- |
| `gofmt -l internal/control/handlers.go internal/control/verify.go internal/control/handlers_test.go internal/host/http/server.go internal/engine/journal.go` | empty output, exit 0 |
| `go test ./internal/control -count=1` | `ok fake-jev/internal/control 0.568s` |
| `go test -race ./internal/control -count=1` | `ok fake-jev/internal/control 2.643s` |
| `go test ./internal/engine -count=1 -race` | `ok fake-jev/internal/engine 1.523s` |
| `go vet ./...` | exit 0, no findings |
| `./scripts/candidate-fingerprint candidate` | `8b96804bcfd74e200491d856317e9d93606c3244e46492caf7cfbccdfb5d5494` (unchanged, re-run after the review) |

`git diff --cached --name-only` is empty throughout; no file was staged.

## Residual risks

- F1 is the only finding that can be observed at runtime: a `journal_full`
  failure renders before sequenced interaction failures in `/verify`, which is
  the reverse of what `internal/control/verify.go` and the coder artifact
  document. Nothing in §41.9 or the acceptance catalog (`C-CTRL-008`) forbids
  either order and no test pins it, so it is a documentation-versus-behavior
  divergence to settle at the parent/hardener level, not a cleaner edit under the
  frozen fingerprint.
- F2 remains a maintenance coupling: the two metadata-key constant blocks must
  agree or §41.7 `method`/`path` silently render as `""`.
- The §41.7 `requestBody`-versus-committed-vectors divergence and the
  `needs-human` blocker recorded in the specifier/coder artifacts are unchanged
  by this stage and were not re-adjudicated here.

## Vector re-review (re-bind 2, fingerprint `4acd6dd6c...`)

Post-correction review of the 13 `testdata/contracts/` vectors touched by scope
amendment 3. No JSON file was changed: all of them are inside the pinned
candidate fingerprint (`4acd6dd6c...`), so any edit would invalidate the pinned
`inputFingerprint`/`outputFingerprint`, exactly as in the review above.

Checked:

- **Valid JSON / formatting.** All 13 parse cleanly and round-trip byte-identical
  through `json.dumps(..., indent=2)`; consistent 2-space indentation maintained.
- **No lost or reordered keys.** Structural diff old (`git show HEAD:`) versus new
  reports only `/steps/N/expect/journal/requestBody` changed (null → object) in
  every file; no key added, removed, or reordered anywhere else.
- **Corrected `requestBody` equals the step's own request body.** All 15 journal
  assertions across the 13 files compare equal to `steps[N].request.body`
  (including `validation/02`, which omits `state`; the corrected value omits it
  too). Key order within each `requestBody` mirrors its request body.
- **Malformed-JSON case unchanged.** `validation/01-malformed-json.json` still
  asserts `requestBody: "{"` (the raw decoded string), and it is not among the
  modified files.
- **Legitimate nulls remain null.** `matching/05-routes-exact-paths.json` and
  `models/01-default-models.json` still assert `requestBody: null` for requests
  with no body, and were not modified.

Findings (reported, not patched — pinned fingerprint):

- **V1 (stale prose, cosmetic).** Two corrected vectors keep a note that now
  contradicts their own value: `mixed/02-partial-answers-rejected.json` and
  `validation/02-missing-state.json` both still say ``journal.requestBody is null
  ... because the expectation records shape only``, while the value is now the
  parsed body. Prose-only; no behavior or assertion depends on it. Left for the
  parent/hardener since the vectors are in-fingerprint.
- `verification/03-payload-too-large.json` still asserts `requestBody: null` for a
  valid-JSON body, but it is the separately-recorded defective vector (unexpanded
  `PADDING` placeholder → cannot yield its own 413); not part of this correction
  and not patched.

Commands: `go test ./internal/control -count=1` → ok (81 tests);
`./scripts/candidate-fingerprint candidate` → `4acd6dd6c...` (matches the pinned
value). No file staged.
