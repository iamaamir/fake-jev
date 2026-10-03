---
stage: coder
task: FJ-040
inputFingerprint: 8c80ab298a252de531dbf9f61a0f67e866fdf5e091cbf8fb80d272d988305c29
outputFingerprint: 525ee08845fbb9098ba37a5146877529863cddd57dcb263e9050628d63b9a84a
taskFingerprint: ae3ccc43bc6a790f3329233e8585379262d6e5b49e2751da583353742d06c6e4
gitHead: e60433b
generatedAt: 2026-10-03T11:00:16Z
author: worker/FJ-040-coder
---

# FJ-040 coder: race and malformed-input robustness coverage

**outputFingerprint: `525ee08845fbb9098ba37a5146877529863cddd57dcb263e9050628d63b9a84a`**

Change set: exactly the four new files named by `allowedFiles`. No existing
file, product source, existing test, script, guard, or `testdata` file was
created or modified. `/Users/mak/git/fake-jev-FJ-040` at `e60433b`, four new
untracked test files, zero staged paths.

## C1 — `internal/engine/race_test.go` (C-HOST-009 engine half, C-QUAL-002)

Package `engine`, imports `sync` and `testing` only (arch guard forbids
`net/http`, `os/cli`, `internal/control`, `internal/compat/jev*` there).
Synchronization: one closed start channel per test plus one `sync.WaitGroup`
join, and per-goroutine result slots. No `time.Sleep`, `time.After`, timer, or
unseeded randomness appears.

`TestConcurrentRegistrationSelectionJournalAndResetAreRaceFree` — three static
stubs (`static-alpha` with a two-element sequence, `static-beta` with
`NewExactExpectation(0)`, `static-gamma` matching only the unused `gamma`
operation), journal capacity 128, 97 goroutines released together: 64 selectors
(half `alpha`, half `beta`), 8 readers, 8 dynamic registrations, 8
`RemoveDynamic`, 8 `ClearRequestHistory`, 8 `Transition`-then
`UpdateInteraction`/`RecordFailure` workers, 1 `Reset`. What each assertion
actually establishes rather than restating:

- every selector got a non-nil `*Selection` **and** the stub ID that owns the
  operation it submitted. This is not a restatement: the only way it can fail is
  a lost, duplicated, or mis-ordered registry mutation, because matching is pure
  and the operation uniquely identifies the stub. (The runtime stubs are
  registered with `Matcher{Operation: "dynamic"}` and priority 40 — a method
  choice not fixed by the specifier artifact: a wildcard runtime stub at that
  priority would legally win the selection and make the identity assertion
  false, so the runtime stubs deliberately match an operation no selector
  submits.)
- both error-slot arrays (`RegisterDynamic`, `Transition`) are all nil. With
  distinct IDs and 72 possible admissions against capacity 128, "no error" is
  exactly the admission-conservation property, not a tautology.
- registry envelope: unique IDs, strictly increasing `RegistrationIndex`,
  `Source` in {static, dynamic}, `SequencePosition <= len(Sequence)` and
  `SequencePosition <= InvocationCount` for every returned stub. The last two
  are the real "count before/after a consumed action" invariant that a lost
  increment or a double-advanced sequence position breaks.
- journal envelope: `len(Interactions()) <= 128`, strictly increasing
  `Sequence` (reserve and append happen under one mutex, so a mid-window
  `Reset` can only truncate the prefix, never reorder it), `Outcome` in
  {matched, unmatched, sequence_exhausted, invalid_stub_response},
  `ResponseStatus` in {0, 200, 409, 500}. An unset or corrupted outcome or
  status fails here.
- failure envelope: every failure has a non-empty code and message, at most one
  `journal_full` failure, and it alone may carry a nil `RequestSequence`; every
  other failure (including `sequence_exhausted` and the injected
  `invalid_stub_response`) must carry one.
- expectation envelope: every reported `StubID` names a currently returned
  stub and every code is a documented `expect_*` code.
- the final `Reset()` runs after all workers have joined, so its post-state is
  deterministic: statics only, every counter and sequence position zero,
  journal empty, failures empty, expectations satisfied. This is the assertion
  that a reset cannot leave a half-reverted registry or derived state.

`TestConcurrentTransitionsRetainExactlyTheAdmittedSequences` — capacity 64, 256
selectors. Proves exact conservation of the admission window: exactly 64
non-nil selections, exactly 64 retained records with `Sequence == i+1` in
order, `InvocationCount == 64`, and exactly one `journal_full` failure with a
nil `RequestSequence`. Any lost reservation, double admission, or reordered
append breaks at least one of these.

`TestConcurrentDuplicateRegistrationAdmitsExactlyOneStub` — 32 goroutines
register the same ID. Exactly one nil error, one stub, index 1.

`TestConcurrentClearHistoryPreservesInvocationCounters` — capacity 4096, 32×8
selections and 8×8 clears. Proves clears never drop stub mutations: all 256
selections succeed, `InvocationCount == 256`, the sequence position of a
sequence-less stub stays 0, every retained record is a matched `static` record,
and the retained window stays strictly sequence increasing across interleaved
clears.

Some invariants are deliberately **not** asserted: a forced interleaving of
`Reset` against one selection (no scheduling hook exists in product code and
none may be added here), and `InteractionRecord.Received` (populated from
`time.Now()` inside `Engine.transition`, hence wall-clock dependent).

## C2 — `internal/control/fuzz_test.go` (C-QUAL-004 control, C-HOST-009 control half)

`FuzzControlMalformedInput(method, path, contentType string, body []byte)` with
a first-statement cap guard (path ≤ 1024, content type ≤ 256, body ≤ 16 KiB) so
no work is input-proportional beyond the cap. Each iteration builds a fresh
`engine.NewEngineWithJournal(nil, 64)` and a fresh `API` over a constant
`config.Load({"schemaVersion":1,"stubs":[]})`, so no iteration can inherit
state. The request is hand-built (`&http.Request{Method, URL: &url.URL{Path},
Header, Body: io.NopCloser(...)}`) instead of `httptest.NewRequest` so a
malformed method/header/body cannot panic inside a harness parser — a panic in
the fuzzer's crash signal can therefore only come from product code.

Per iteration it asserts: status ∈ {200, 201, 204, 400, 404, 405, 409} (a
status ≥ 500 would mean untrusted input was reported as an internal fake-jev
failure); 204 ⇒ empty body; otherwise `Content-Type: application/json`, an
object body, and ≤ 64 KiB; non-2xx ⇒ a non-empty `error` field prefixed
`fake_jev_`; the engine journal is untouched (`len(Interactions()) == 0`) — the
control API never journals; the caller's body slice is byte-identical after the
call; and for a matched read-only route (health, meta, stubs GET, requests GET,
verify) a second identical request returns the same status and byte-identical
body. That last pair is the observable form of "no corrupted state": a mutation
hidden in a read path would show up as a differing repeat.

15 seeds, all non-crashing, covering health/meta/stubs/requests/verify/reset,
an empty stub ID, truncated JSON, `null`, two concatenated JSON values, an
unknown path, an unknown method, a 4 KiB `[` body, and a CRLF-injected
`Content-Type`; plus one valid create so the success path is exercised in the
seed corpus.

`TestControlStateRaceFree` (required by C-HOST-009's "control state ... race
free"; `internal/control` is otherwise outside `allowedFiles`, so the test lives
here — a recorded method choice, not a scope widening) drives one shared `API`
from 76 goroutines released together with per-goroutine result slots: 16 health
reads, 8 meta, 8 stub reads, 8 dynamic creates of distinct IDs, 8 stub clears, 8
request-history reads, 8 request-history clears, 8 verifies, 4 resets.
Assertions hold for any interleaving: fixed per-route status (GETs 200; creates
201/400/409; deletes and reset 204 with an empty body), every 2xx body is a JSON
object, every non-2xx body carries a `fake_jev_`-prefixed error with the JSON
content type, **all 8 distinct stubs were admitted** (`created == 8` — the real
conservation claim, since a lost registration index or a reset that eats a
registration would show up here), and a final quiescent `GET /stubs` reports
strictly increasing unique `registrationIndex` values with unique engine stub
IDs.

## C3 — `internal/compat/jev/v1/fuzz_test.go` (C-QUAL-004 compat)

`FuzzProfileUntrustedInput(method, target, authorization string, body,
configured []byte)` with a cap guard (64/512/512/16 KiB/16 KiB) and a fresh
`DefaultProfile()` per iteration. 13 seeds include the canonical valid
systemone request with a valid and with an out-of-range configured answer,
`null`/`[]`/`{}` answers, truncated JSON, `null` and `[]` bodies, a body with a
duplicate `questions` key, a 64-question body, invalid UTF-8 bytes, and an
unknown target.

Properties asserted per iteration:

- route parity with the profile's own validator: on the systemone route,
  `Decode` succeeds **iff** `ValidateRequest(body)` succeeds, and on rejection
  the two error strings are identical, so a fuzzed request can never be accepted
  by one path and rejected by the other;
- accepted exchanges carry `Profile == "jev/v1"` and the operation the route
  names;
- determinism: a second identical `Decode` yields the same accept/reject, the
  same diagnostic, and a `reflect.DeepEqual` exchange;
- the models route is body independent: decoding with and without a body gives
  deep-equal exchanges (a real property — `Decode(models)` must not read the
  body);
- the unknown route is always a controlled error with a non-empty diagnostic;
- when the request validates and declares ≤ 32 questions, `GenerateAnswers` is
  run twice: identical accept/reject, byte-identical marshalled output (no
  hidden generator state), a key set equal to the question name set, and every
  answer object whose `type` equals the question type and which contains the
  matching `noul`/`choice`/`score` helper key;
- the `body` and `configured` caller slices are byte-identical afterwards.

Recorded strengthening: the specifier artifact asked for `errors.As` "when it
is an `*InvalidStubResponseError`"; the test asserts unconditionally that every
generation failure is reachable through `errors.As` (i.e. is an
`*InvalidStubResponseError`). That is the current documented contract of
`GenerateAnswers` (its failures are the type the control layer maps to
`invalid_stub_response`), and the seed corpus covers both the accept and the
reject branch. It is stricter than the artifact's wording, so it is flagged here
for the cleaner and hardener rather than left implicit.

The pre-existing `FuzzValidateFixtureAnswers` in `fixture_test.go` (FJ-059) is
untouched and still passes.

## C4 — `internal/config/fuzz_test.go` (C-QUAL-004 config)

`FuzzConfigMalformedInput(data []byte)` with a 64 KiB cap guard. Only `Load`
(bytes) is called — never `LoadFile` — so the known unbounded-read defect (D1) is
unreachable by construction. Properties: `Load` never panics; two identical
calls agree on accept/reject and on the exact error string; an error is never
returned together with a configuration and never has an empty message; a success
is `reflect.DeepEqual`-reproducible, has `SchemaVersion == 1`, passes
`Validate`, and carries the §12 defaults (`DefaultHost`, `DefaultPort`,
`DefaultMode`, at least one compatibility profile, all five limits positive, at
least one model, non-nil stubs), and never declares more stubs than the bytes it
was read from (non-amplification); the caller's slice is unchanged.

18 seeds cover `{}`, `null`, `[]`, `"x"`, `0`, a YAML flow document, two YAML
documents, a duplicate JSON key, an unknown field, `schemaVersion` 2, a negative
limit, a stub with a raw body and no `then`, `mode: auto`, 1 KiB of `<`, valid
multi-byte UTF-8, an embedded NUL, and a deeply nested 256-level array.

## C5 — repository-wide battery (C-QUAL-002, C-HOST-009)

All commands below were run from the repository root on the change set above.

| command | result |
| --- | --- |
| `gofmt -w` on the four new files, then `gofmt -l` | list empty (already formatted) |
| `go test ./internal/engine ./internal/control ./internal/compat/jev/v1 ./internal/config -count=1` | exit 0 |
| `go test -race ./... -count=1` | exit 0, 702 tests across 10 packages, no `WARNING: DATA RACE` |
| `go test ./... -count=1` | exit 0, 702 tests across 10 packages |
| `go vet ./...` | exit 0, no diagnostics |
| `go run ./cmd/guard arch` | exit 0, `findings: []`, 64 files / 10 packages |
| `go run ./cmd/guard lint` | exit 0, `findings: []` |
| `go run ./cmd/guard trace` | exit 0, `findings: []`, 17 covered markers, 26 test files |
| `go run ./cmd/guard fuzz` | exit 0, `findings: []`, `targets: 4` |
| `go build ./cmd/fake-jev` | exit 0 |
| `./scripts/candidate-fingerprint candidate` | `525ee08845fbb9098ba37a5146877529863cddd57dcb263e9050628d63b9a84a` |

Focused race runs: `go test -race -count=1 -run '<the four engine race tests>'
./internal/engine/` → `ok fake-jev/internal/engine 1.329s`, all four PASS, no
race warning; `go test -race -count=1 -run '^TestControlStateRaceFree$'
./internal/control/` → `ok fake-jev/internal/control 1.237s`, PASS, no race
warning.

Seed corpora without fuzzing (deterministic, they run in plain `go test`):
`FuzzControlMalformedInput` 15 sub-cases, `FuzzProfileUntrustedInput` 13
sub-cases, `FuzzConfigMalformedInput` 18 sub-cases, `FuzzValidateFixtureAnswers`
(pre-existing) unchanged and passing.

## C6 — guard fuzz cost (measured, not estimated)

`go run ./cmd/guard fuzz` now discovers 4 targets
(`FuzzValidateFixtureAnswers`, `FuzzControlMalformedInput`,
`FuzzProfileUntrustedInput`, `FuzzConfigMalformedInput`) and runs each for the
unchanged `guards.json` `fuzz.fuzztime` of 30s. **Measured wall time: 2m12.257s**
(`real`, 4 targets × 30s plus per-target compile/seed overhead), exit 0, zero
findings. `fuzztime` was not shortened and no guard was weakened.

## Defects

**No genuine product defect was found, and no product source was edited.**
Neither fuzz target wrote a crash input (no `testdata/fuzz/**` path exists in
`git status`), and `go test -race` reported no data race.

Two known defects were kept out of scope and are neither fixed nor asserted:

- D1 unbounded configuration read (`LoadFile` → `os.ReadFile`, recorded after
  FJ-030). The config fuzz target calls `Load([]byte)` only, so the hanging path
  is unreachable from this change set. Not fixed.
- D2 post-reset `invalid_stub_response` recording race in
  `internal/engine/engine.go` (recorded after FJ-021). The mixed engine test does
  drive `RecordFailure` concurrently with `Reset`, exactly the D2 window: it
  tolerates a failure naming a cleared sequence (asserting only that
  `invalid_stub_response` failures carry a non-nil `RequestSequence`) and takes
  its "no failures remain" assertion only after the final `Reset()` with all
  workers joined, so no `RecordFailure` can be in flight. D2 is therefore
  observed as tolerated behaviour, not asserted away, and not fixed.

## Residual risks

- §22.2's "matcher input" area is still unfuzzed: `allowedFiles` contains no
  `internal/engine` fuzz file, so the engine matcher/value tree is not a fuzz
  target here (recorded by the specifier as an open coverage item, repeated
  honestly).
- 30s per target is short. Zero findings bound the risk, they do not eliminate
  it.
- The `journal_full` envelope assertion in the mixed engine test is vacuous at
  capacity 128 with only 72 possible admissions; it is kept as an envelope, and
  the real journal-full conservation proof is the dedicated capacity-64 test.
- The config non-amplification assertion (`len(cfg.Stubs) <= len(data)`) is
  argued, not proven, for pathological YAML anchors; 30s of fuzzing plus 18
  seeds found no violation.
- The control fuzz target pins the complete legal status set. A future
  legitimate status (for example 413 or 422 on the control plane) would have to
  widen that set deliberately.
- The four new files are untracked (not staged); the candidate fingerprint
  includes untracked non-`.agent` paths, so the recorded fingerprint does cover
  them.

## Spec-silent / method choices recorded

- Caps (16 KiB request/answer/control bodies, 64 KiB configuration), the 32-question
  generation bound, seed corpus contents, and the exact goroutine counts are
  method choices; §22.1/§22.2 and the acceptance catalog fix none of them.
- The `internal/engine/race_test.go` test names and the control concurrency
  test's placement in `internal/control` are method choices driven by
  `allowedFiles`.
- No test asserts anything about wall-clock time, host load, CPU count, map
  iteration order, or goroutine scheduling order.
