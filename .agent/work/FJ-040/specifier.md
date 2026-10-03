---
stage: specifier
task: FJ-040
inputFingerprint: 8c80ab298a252de531dbf9f61a0f67e866fdf5e091cbf8fb80d272d988305c29
outputFingerprint: 8c80ab298a252de531dbf9f61a0f67e866fdf5e091cbf8fb80d272d988305c29
taskFingerprint: ae3ccc43bc6a790f3329233e8585379262d6e5b49e2751da583353742d06c6e4
gitHead: e60433b
generatedAt: 2026-10-03T10:44:42Z
author: worker/FJ-040-specifier
---

# FJ-040 specifier: race and malformed-input robustness coverage

Objective: engine and control state are race-free, and malformed untrusted input
produces controlled errors instead of panics or corrupted state (§17.4, §22.1,
§22.2, §23, §32).

This artifact fixes WHAT each of the four allowed files contains and which
observable property it establishes. It contains no test code. It is evidence,
not a verdict.

## Binding boundaries

- Change set: only the four NEW files listed under `allowedFiles` in
  `.agent/work/FJ-040/state.json`. No existing file, product source, existing
  test, or script is edited. Evidence writes: `.agent/work/FJ-040/**` and
  `.agent/reports/FJ-040/**` (verifier-owned).
- Determinism: seed corpus entries are literal constants; no wall-clock
  dependence; no `time.Sleep`, no `time.After`, no timer-based synchronization;
  no unseeded randomness; no assertion that depends on Go map iteration order,
  goroutine scheduling order, host load, or the number of CPUs.
- Synchronization vocabulary used by the design: a release channel closed once
  ("start barrier"), `sync.WaitGroup`, `sync.Mutex`/`sync.RWMutex`, and
  per-goroutine result slots. A `time.Sleep` used to wait for another goroutine
  is a defect of the implementation of this item.
- Bounded work: every fuzz target returns early when its input exceeds the
  stated cap, so neither allocation nor parse work is driven by the fuzzer
  beyond the cap. No fuzz target may loop on unbounded input or allocate in
  input-proportional size without a cap.
- Guard constraints that constrain the test source (checked by
  `go run ./cmd/guard`): `arch` guards test-only imports exactly like product
  imports (so `internal/engine/race_test.go` may not import `net/http`,
  `os/exec`, `text/template`, `html/template`, `plugin`, `fake-jev/internal/cli`,
  `fake-jev/internal/control`, or `fake-jev/internal/compat/jev`;
  `internal/config` and `internal/compat/jev/v1` may not import `net`,
  `net/http`, `os/exec`, `text/template`, `html/template`, `plugin`);
  `lint` forbids discarded error results and unchecked type assertions;
  `trace` flags any acceptance-catalog ID appearing in a string literal of a
  `_test.go` file that the catalog does not define (the only IDs used here,
  C-HOST-009, C-QUAL-002, C-QUAL-004, are catalog rows).

## C1 — concurrent engine state transitions are race-free and conserve state

File: `internal/engine/race_test.go`, package `engine`.
Imports: `sync`, `testing` only.

Test 1, mixed concurrent window:
`TestConcurrentRegistrationSelectionJournalAndResetAreRaceFree`.
Fixed goroutine set, released together through a closed start channel, joined
with one `sync.WaitGroup`:

- 64 × `SelectInvocation(Exchange{Profile: "profile"})` against a matching
  static stub;
- 8 × `RegisterDynamic` with distinct IDs `dynamic-0..dynamic-7`;
- 8 × `RemoveDynamic`;
- 8 × `ClearRequestHistory`;
- 8 × reader goroutines calling `Interactions()`, `VerificationFailures()`,
  `EvaluateExpectations()`;
- 8 × goroutines that call `Transition` with a valid `InteractionDecision` and
  then `UpdateInteraction(sequence, "fake_jev_invalid_stub_response", 500)` and
  `RecordFailure(sequence, "fake_jev_invalid_stub_response", "...", "", 500)`;
- 1 × `Reset`.

Engine built with `NewEngineWithJournal` and a capacity of 128, three static
stubs (one with `Sequence` of length 2, one with `Expect` = exact 0 to exercise
`EvaluateExpectations`, distinct priorities). Post-quiescent assertions (after
Wait, plus one final deterministic `Reset()` after all goroutines have joined):

- `Registry.Stubs()`: IDs unique; `RegistrationIndex` strictly increasing in
  returned order; every `Source` ∈ {`static`, `dynamic`};
- per stub: `Sequence == nil || SequencePosition <= uint64(len(Sequence))`, and
  `SequencePosition <= InvocationCount`;
- after the final `Reset()`: no stub with `Source == dynamic` remains, every
  static counter and sequence position is 0, `Interactions()` is empty, and
  `VerificationFailures()` is empty;
- `Interactions()`: length ≤ 128; `Sequence` strictly increasing;
- `EvaluateExpectations()`: every `StubID` names a stub currently returned by
  `Stubs()`, and every `Code` ∈ {`expect_exactly`, `expect_at_least`,
  `expect_at_most`};
- `VerificationFailures()`: at most one entry with `Code == "journal_full"` and
  `RequestSequence == nil`; every `invalid_stub_response` entry has a non-nil
  `RequestSequence`.

Observable: `go test` and `go test -race` exit 0 for `./internal/engine/`, and
the race detector prints nothing for this test.
Command: `go test -race -count=1 -run '^TestConcurrentRegistrationSelectionJournalAndResetAreRaceFree$' ./internal/engine/`
Expected: `ok  fake-jev/internal/engine  <d>` and no `WARNING: DATA RACE` line.
Traces: C-HOST-009 (engine half), C-QUAL-002.

Test 2, admission window conservation:
`TestConcurrentTransitionsRetainExactlyTheAdmittedSequences`.
One static stub, journal capacity 64, 256 goroutines released together, each
calling `SelectInvocation` with the matching exchange and recording into its own
result slot whether the returned `*Selection` was nil.

- `len(Interactions()) == 64`;
- `Interactions()[i].Sequence == uint64(i+1)` for every i — the engine mutex
  makes `journal.reserve` strictly ordered, so the retained records are exactly
  the first 64 admitted sequences, independent of scheduling;
- non-nil selection count == 64, and the selected stub's `InvocationCount` == 64
  (exact conservation: no lost, duplicated, or over-counted mutation);
- `VerificationFailures()` == exactly one `journal_full` entry with nil
  `RequestSequence`.

Command: `go test -race -count=1 -run '^TestConcurrentTransitionsRetainExactlyTheAdmittedSequences$' ./internal/engine/`
Expected: `ok  fake-jev/internal/engine  <d>`, no `WARNING: DATA RACE`.
Traces: C-HOST-009 (engine half), C-QUAL-002.

Test 3, duplicate registration:
`TestConcurrentDuplicateRegistrationAdmitsExactlyOneStub`.
32 goroutines register the same ID into a fresh engine; each writes its own error
slot (no shared map). Assertions: exactly one `nil` error; `len(Stubs()) == 1`;
that stub's `RegistrationIndex == 1`.

Command: `go test -race -count=1 -run '^TestConcurrentDuplicateRegistrationAdmitsExactlyOneStub$' ./internal/engine/`
Expected: `ok  fake-jev/internal/engine  <d>`.
Traces: C-HOST-009 (engine half), C-QUAL-002.

Test 4, clear does not drop stub mutations:
`TestConcurrentClearHistoryPreservesInvocationCounters`.
Journal capacity 4096; 32 goroutines × 8 `SelectInvocation` each (each goroutine
counts its own non-nil results), 8 goroutines × 8 `ClearRequestHistory`, released
together. Assertions: total non-nil selections == 256 == selected stub's
`InvocationCount`; every retained record is `Matched`, `MatchedStubID == "static"`,
`Outcome == "matched"`, sequences strictly increasing.

Command: `go test -race -count=1 -run '^TestConcurrentClearHistoryPreservesInvocationCounters$' ./internal/engine/`
Expected: `ok  fake-jev/internal/engine  <d>`, no `WARNING: DATA RACE`.
Traces: C-HOST-009 (engine half), C-QUAL-002.

Invariants that ARE deterministically checkable and therefore asserted: the four
above. Invariants deliberately NOT asserted: any specific interleaving of one
`Reset` against one `SelectInvocation` (cannot be forced without a scheduling
hook that product code does not expose), and the exact value of
`InteractionRecord.Received` (derived from `time.Now()` inside
`Engine.transition`, hence wall-clock dependent). §17.4 allows exactly two legal
outcomes for a reset racing a request; C1 observes the resulting state envelope
(identity, bounds, monotonicity, conservation) rather than a forced schedule.

## C2 — malformed control-plane input yields controlled HTTP errors

File: `internal/control/fuzz_test.go`, package `control`.
The file contains exactly one fuzz target plus one concurrency test. The second
test is required by C-HOST-009 ("control state ... race-free") and is placed here
because `internal/control` is otherwise outside `allowedFiles`; this is a
recorded method choice, not a scope widening (see "Spec-silent observations").

Fuzz target: `FuzzControlMalformedInput(f *testing.F)`.
Inputs: `method, path, contentType string, body []byte`. Cap guard first:
`len(path) > 1024 || len(contentType) > 256 || len(body) > 16<<10` ⇒ return.
Per iteration a fresh `engine.NewEngineWithJournal(nil, 64)` plus a fresh
`API` is built (`NewAPI` over a constant `config.Load` of
`{"schemaVersion":1,"stubs":[]}`), so iterations cannot inherit state from
previous iterations.
The request is constructed directly as `&http.Request{Method: method,
URL: &url.URL{Path: path}, Header: ..., Body: io.NopCloser(bytes.NewReader(body))}`
and driven through `api.ServeHTTP(recorder, req)` with an `httptest.ResponseRecorder`.
Hand-building the request (instead of `httptest.NewRequest`) is deliberate: the
product surface under test is `API.ServeHTTP`, which reads only `Method`,
`URL.Path`, headers, and `Body`; this keeps harness parsing panics out of the
fuzzer's signal so a panic can only come from product code.
Assertions per iteration:

- `recorder.Code` ∈ {200, 201, 204, 400, 404, 405, 409} — never ≥ 500, so a
  malformed request can never be reported as an internal fake-jev failure;
- 204 ⇒ empty body; every other status ⇒ body is a JSON object, `Content-Type`
  is `application/json`, and the body length is ≤ 64 KiB;
- non-2xx ⇒ `error` field present, non-empty, prefix `fake_jev_`;
- the engine journal is untouched by every control request:
  `len(engine.Interactions()) == 0` after the call (the `API` doc comment: the
  control API never uses the engine journal);
- the fuzzed request body slice is byte-identical after the call (no aliasing
  corruption of caller input);
- for a read-only route that matched (health, meta, stubs GET, requests GET,
  verify) a second identical request returns the same status and byte-identical
  body.
Seed corpus (registered with `f.Add`, all non-crashing so the ordinary
`go test` seed run is clean): `GET /__fake/v1/health`, `GET /__fake/v1/meta`,
`GET /__fake/v1/stubs`, `GET /__fake/v1/requests`, `GET /__fake/v1/verify`,
`POST /__fake/v1/reset`, `POST /__fake/v1/stubs` with `{"id":""}`,
`POST /__fake/v1/stubs` with `{`, `POST /__fake/v1/stubs` with `null`,
`POST /__fake/v1/stubs` with two concatenated JSON values, `POST` on an unknown
path, `BREW /__fake/v1/reset`, `POST /__fake/v1/stubs` with a 4 KiB body of
`[`, and `GET /__fake/v1/health` with a hostile `Content-Type` value.

Concurrency test: `TestControlStateRaceFree`.
One `API` shared by all goroutines (the state under test); each goroutine builds
its own request and its own `httptest.ResponseRecorder`, fixed counts released
together and joined with one `sync.WaitGroup`: 16 × `GET /__fake/v1/health`,
8 × `GET /__fake/v1/meta`, 8 × `GET /__fake/v1/stubs`, 8 × `POST /__fake/v1/stubs`
with distinct valid dynamic stub IDs, 8 × `DELETE /__fake/v1/stubs`,
8 × `GET /__fake/v1/requests`, 8 × `DELETE /__fake/v1/requests`,
8 × `GET /__fake/v1/verify`, 4 × `POST /__fake/v1/reset`.
Assertions after Wait: every status is in the allowed set for its route
(GETs/verify ⇒ 200, POST stubs ⇒ 201|400|409, DELETEs and reset ⇒ 204); every
2xx body parses as JSON and every non-2xx body carries a `fake_jev_`-prefixed
error with `Content-Type: application/json`; `GET /stubs` item
`registrationIndex` values are unique and strictly increasing; engine
`Registry.Stubs()` IDs are unique. No assertion depends on which goroutine won.

Commands:
`go test -race -count=1 -run '^TestControlStateRaceFree$' ./internal/control/`
Expected: `ok  fake-jev/internal/control  <d>`, no `WARNING: DATA RACE`.
`go test -count=1 -run '^$' ./internal/control/`
Expected: `ok  fake-jev/internal/control  <d>` (seed corpus only, deterministic).
`go test -count=1 -run '^$' -fuzz '^FuzzControlMalformedInput$' -fuzztime 30s ./internal/control/`
Expected: `ok  fake-jev/internal/control  <d>` (elapsed ≈ 30s), no crash input
written, no `--- FAIL:` line.
Traces: C-QUAL-004 (control payloads), C-HOST-009 (control half), C-QUAL-002.

## C3 — untrusted jev/v1 profile input yields controlled errors

File: `internal/compat/jev/v1/fuzz_test.go`, package `v1`.
Second fuzz target in this package; `FuzzValidateFixtureAnswers` in
`fixture_test.go` (FJ-059) is untouched.

Fuzz target: `FuzzProfileUntrustedInput(f *testing.F)`.
Inputs: `method, target, authorization string, body, configured []byte`.
Cap guard first: `len(method) > 64 || len(target) > 512 ||
len(authorization) > 512 || len(body) > 16<<10 || len(configured) > 16<<10` ⇒
return. A fresh `DefaultProfile()` is built per iteration.
Assertions per iteration:

- `profile.Decode(HostRequest{Meta: RequestMeta{Method: method, Target: target},
  Headers: map[string]string{"Authorization": authorization}, Body: body})`
  never panics, and either returns an error or returns an exchange with
  `Profile == "jev/v1"` and `Operation` ∈ {`models`, `systemone`};
- route parity with the profile's own validator: for the systemone route,
  `Decode` succeeds if and only if `ValidateRequest(body)` succeeds, and on
  rejection the two error strings are equal (`Decode` returns the validator's
  error unchanged), so a fuzzed request can never be accepted by one path and
  rejected by the other;
- determinism: a second identical `Decode` yields the same accept/reject, the
  same error text, and an exchange that `reflect.DeepEqual`s the first;
- when `ValidateRequest` succeeds and `len(request.Questions) <= 32`:
  `GenerateAnswers(request, json.RawMessage(configured))` never panics; a second
  identical call produces byte-identical JSON (`json.Marshal` of both answer
  maps compared) proving no hidden state in generation;
- on generation success: the answer key set equals the question name set, and
  every answer is a JSON object whose `type` equals the question's type and
  which contains the matching helper key (`noul`/`choice`/`score`);
- on generation failure: the error is non-nil and its message is non-empty, and
  when it is an `*InvalidStubResponseError` it is reachable through
  `errors.As`;
- the `body` and `configured` input slices are byte-identical after the calls
  (generation and validation must not corrupt caller-owned input the caller may
  reuse).
Questions-count and input-length caps are the loop/alloc bounds: no code path in
this target iterates input-proportional work beyond `len(request.Questions) <= 32`.
Seed corpus: `GET /v1/models` with an empty body; systemone route with the
canonical `{"state":{},"model":"jev-latest","questions":{"urgent":{"type":"noul"}}}`
paired with `{"urgent":{"noul":0.9}}`; the same request with configured
`{"urgent":{"noul":1.5}}`; configured `null`, `[]`, and `{}`; body `{`, body
`null`, body `[]`, body with a duplicate `questions` key, body with 64 questions
(above the cap branch), a body containing invalid UTF-8 bytes, and an unknown
target `POST /v1/other` with a body.

Command: `go test -count=1 -run '^$' -fuzz '^FuzzProfileUntrustedInput$' -fuzztime 30s ./internal/compat/jev/v1/`
Expected: `ok  fake-jev/internal/compat/jev/v1  <d>` (elapsed ≈ 30s), no crash
input written.
Command: `go test -count=1 -run '^$' ./internal/compat/jev/v1/`
Expected: `ok  fake-jev/internal/compat/jev/v1  <d>` (both fuzz targets' seed
corpora only).
Traces: C-QUAL-004 (compat decoder robustness, matcher input), C-QUAL-002.

## C4 — malformed configuration bytes yield controlled decode errors

File: `internal/config/fuzz_test.go`, package `config`.

Fuzz target: `FuzzConfigMalformedInput(f *testing.F)`.
Input: `data []byte`. Cap guard first: `len(data) > 64<<10` ⇒ return.
Only `Load(data)` is called — never `LoadFile` — so the known unbounded
`os.ReadFile` defect (D1 below) is unreachable in this target by construction.
Assertions per iteration:

- `Load` never panics;
- determinism: a second `Load` on the same bytes gives the same accept/reject and
  an identical error string, and on success `reflect.DeepEqual(cfg1, cfg2)`;
- on error: the message is non-empty;
- on success: `cfg != nil`, `cfg.SchemaVersion == 1`, `Validate(cfg)` returns nil,
  and the §12 defaults are present —
  `cfg.Server.Host == DefaultHost`, `cfg.Server.Port == DefaultPort`,
  `cfg.Mode == DefaultMode`, `len(cfg.Compatibility) >= 1`,
  `cfg.Limits.DataPlaneBodyBytes > 0`, `cfg.Limits.ControlPlaneBodyBytes > 0`,
  `cfg.Limits.MaxInteractions > 0`, `cfg.Limits.LogBodyBytes > 0`,
  `cfg.Limits.GracefulShutdownSeconds > 0`, `len(cfg.Models) >= 1`,
  `cfg.Stubs != nil`;
- no amplification: `len(cfg.Stubs) <= len(data)` for successful loads;
- the input slice is byte-identical after the call.
Seed corpus: `{"schemaVersion":1}`, `{}`, `null`, `[]`, `"x"`, `0`, the YAML flow
form `schemaVersion: 1\n`, a YAML document with two documents, JSON with a
duplicate `schemaVersion` key, JSON with an unknown field, `{"schemaVersion":2}`,
`{"schemaVersion":1,"limits":{"maxInteractions":-1}}`, a stub with a raw body and
no `then`, a configuration declaring `mode: auto`, a 1 KiB run of `<`, valid
UTF-8 multi-byte text, bytes containing NUL, and a deeply nested
`[[[[...]]]]` document.

Command: `go test -count=1 -run '^$' -fuzz '^FuzzConfigMalformedInput$' -fuzztime 30s ./internal/config/`
Expected: `ok  fake-jev/internal/config  <d>` (elapsed ≈ 30s), no crash input
written.
Command: `go test -count=1 -run '^$' ./internal/config/`
Expected: `ok  fake-jev/internal/config  <d>` (seed corpus only).
Traces: C-QUAL-004 (malformed untrusted input, controlled error, no corrupted
state), C-QUAL-002.

## C5 — repository-wide race battery

No file change; this criterion consumes C1–C4 plus the existing suite.
Command: `go test -race -count=1 ./...`
Expected: one `ok  <pkg>  <d>` line per package, exit 0, no
`WARNING: DATA RACE` line in the output.
Command: `go test -count=1 ./...` and `go vet ./...`
Expected: exit 0 each (the four new files must compile under the module's vet
settings; the `lint` guard row also forbids discarded error results in them).
Command: `./scripts/verify-candidate FJ-040`
Expected: exit 0; the `go test -race (repository packages)` row is required at
G-H and reads this battery's result.
Traces: C-QUAL-002, C-HOST-009.

## C6 — guard fuzz row cost consequence

`go run ./cmd/guard fuzz` runs one `go test -run '^$' -fuzz '^<Name>$' -fuzztime
<guards.json fuzz.fuzztime>` invocation per discovered target
(`cmd/guard/fuzz.go`, `discoverFuzzTargets`). `race_test.go` declares no fuzz
target, so this item takes the repository from 1 target
(`FuzzValidateFixtureAnswers`) to 4: `FuzzControlMalformedInput`,
`FuzzProfileUntrustedInput`, `FuzzConfigMalformedInput`.
Expected: the guard fuzz row grows from ≈30s to ≈120s (4 × 30s plus per-target
compile/seed overhead), exit 0, zero findings. This is the accepted price of real
coverage; fuzztime must not be shortened and no guard may be weakened.
Command: `go run ./cmd/guard fuzz`
Expected: exit 0 and a report with no findings for the fuzz row.

## Traces

- C-HOST-009 — engine and control state race-free under `go test -race`
  (§17.4, §32): C1 (engine registration/selection/journal/evaluation/clear/reset
  concurrency), C2 `TestControlStateRaceFree` (control mutations and reads),
  C5 (`go test -race ./...`).
- C-QUAL-002 — `go test -race ./...` on supported runners (§23): C1–C4 source
  (the new tests must be race-clean themselves), C5, and the G-H
  `go test -race (repository packages)` row in `./scripts/verify-candidate`.
- C-QUAL-004 — malformed untrusted input yields a controlled error and never a
  panic or corrupted state (§22.2): C2 fuzz target (control paths, bodies,
  headers), C3 fuzz target (profile request bodies and configured answers),
  C4 fuzz target (configuration bytes).

## Non-goals and out of scope

- No product source, existing test, script, guard, or `allowedFiles` change.
- No benchmarking, no timing assertions, no load thresholds.
- No `time.Sleep` or other timer-based waiting anywhere.
- Matcher-input fuzzing of the engine's own `Matcher`/`Value` tree (§22.2 lists
  "matcher input") is NOT covered by this item: `allowedFiles` contains no
  `internal/engine` fuzz file. The compat target C3 exercises engine-side
  matching only indirectly. Recorded as an open coverage item, not a claim.
- Retry/backoff, provider error injection, and HTTP host wiring are untouched.
- §17.4's two legal reset/request outcomes are observed as a state envelope
  (C1), not as a forced interleaving; no scheduling hook exists in product code
  and none may be added here.

## Spec-silent observations (§22.1 / §22.2 leave the method open)

- §22.2 says to use Go's built-in fuzzing "where useful" and enumerates
  candidate areas; it fixes neither the number of targets, the per-package
  placement, the seed corpus, the input caps, nor the duration. The 30s figure
  lives in `.agent/guards.json` (`fuzz.fuzztime`) and was not chosen by this
  item. The caps chosen here (16 KiB request/answer/control bodies, 64 KiB
  configuration) are method choices, recorded so a later reviewer can see they
  were deliberate.
- §22.2 does not say fuzz targets must assert determinism between two identical
  calls; that assertion is added here because §42.5-style diagnostics require
  reproducible outcomes, and it is the cheapest observable form of "no corrupted
  state".
- §22.1 requires "concurrent safety" of the engine but does not fix invariant
  strength. C1 chooses conservation invariants (exact retained admission window,
  counter conservation, uniqueness of registration indices) that are stronger
  than the existing `TestConcurrentTransitionsAndResetAreRaceFree` and
  `TestConcurrentSelectionSerializesState`, while avoiding overlap with them:
  this item adds dynamic-registration, `RemoveDynamic`, `ClearRequestHistory`,
  evaluation, and journal-read concurrency, none of which the existing tests
  cover.
- §22.2 does not state that a controlled error must be distinguishable from a
  5xx. C2 pins status < 500 with a `fake_jev_`-prefixed `error` field because
  §41's documented failure shape is the only controlled-error shape for the
  control plane; the target asserts the shape, not a specific code per input.
- §17.4 requires serialized state transitions but does not prescribe the
  barrier style used to start goroutines; the closed-channel start barrier plus
  `WaitGroup` join is a method choice, deterministic and sleep-free.
- `InteractionRecord.Received` is populated from `time.Now()` inside
  `Engine.transition` and is therefore wall-clock dependent; no new test asserts
  anything about it.

## Known defects: out of scope, must not be fixed or asserted

- D1 unbounded configuration read (recorded after FJ-030): `LoadFile` reads the
  whole path with `os.ReadFile`, so a `/dev/zero`-style input can hang the
  validate path. No new test reaches `LoadFile`; C4 uses `Load([]byte)` only.
- D2 post-reset `invalid_stub_response` recording race in
  `internal/engine/engine.go` (recorded after FJ-021): `RecordFailure(sequence,
  ...)` after a `Reset()` records a failure naming a sequence the reset already
  cleared. Consequence for this design: the only "failures empty after reset"
  assertion (C1) is taken after the final `Reset()` with all workers already
  joined, so no `RecordFailure` can be in flight. No test asserts the absent
  re-ordering.

If race or fuzz execution surfaces either defect, or any other real defect: do
not edit product sources; record the minimal reproduction in
`.agent/work/FJ-040/state.json` notes as a suspected defect, leave the failing
case out of the committed test or mark it as a documented skipped case, and let
the PM open a dedicated ticket. A crash input written by `go test -fuzz` under
`testdata/fuzz/<Name>/` is a finding artifact: record its bytes in the item notes
and remove the generated file so the change set stays inside `allowedFiles`.

## Revision and hygiene record

- Base revision `e60433b`. At specifier time `git status --porcelain` reported
  one unstaged path, `.agent/work/FJ-040/state.json` (work-item metadata, not a
  product file); no product file was modified by this stage.
- No sleeps-as-sync and no unseeded randomness appear in this design: all fuzz
  seeds are literal constants, all concurrency is started by a closed channel and
  joined by `sync.WaitGroup`.
- Only the four allowed files change: `internal/engine/race_test.go`,
  `internal/control/fuzz_test.go`, `internal/compat/jev/v1/fuzz_test.go`,
  `internal/config/fuzz_test.go`.
