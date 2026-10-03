---
stage: cleaner
task: FJ-040
inputFingerprint: 525ee08845fbb9098ba37a5146877529863cddd57dcb263e9050628d63b9a84a
outputFingerprint: 525ee08845fbb9098ba37a5146877529863cddd57dcb263e9050628d63b9a84a
taskFingerprint: ae3ccc43bc6a790f3329233e8585379262d6e5b49e2751da583353742d06c6e4
gitHead: e60433b
generatedAt: 2026-10-03T11:38:29Z
author: worker/FJ-040-cleaner
---

# FJ-040 cleaner: race and malformed-input coverage review

Scope of this stage: read `./scripts/agent-context FJ-040`, `docs/agents/roles/cleaner.md`,
`.agent/work/FJ-040/specifier.md`, `.agent/work/FJ-040/coder.md`, and the four new test
files; audit for sleeps-as-sync, unseeded randomness, wall-clock logic, map-iteration
order reliance; prove non-vacuity by mutation in a throwaway copy outside the
repository; check fuzz boundedness/termination; check that only the four allowed files
changed; check comments for over-claims. Cleanup applied: **none**. No file other than
this artifact was written.

## Change-set discipline

- `git status --short` (revision e60433b): ` M .agent/work/FJ-040/state.json`,
  `?? .agent/reports/FJ-040/report-20261003T110212Z.json`, `?? .agent/work/FJ-040/coder.md`,
  `?? .agent/work/FJ-040/specifier.md`, `?? internal/compat/jev/v1/fuzz_test.go`,
  `?? internal/config/fuzz_test.go`, `?? internal/control/fuzz_test.go`,
  `?? internal/engine/race_test.go`. `git diff --cached --stat` is empty: zero staged paths.
- The four test files were byte-compared against the untouched snapshot taken before this
  review (`/tmp/fj040-pristine`): all four identical, i.e. this stage edited nothing.
- `./scripts/candidate-fingerprint candidate` = `525ee08845fbb9098ba37a5146877529863cddd57dcb263e9050628d63b9a84a`,
  equal to the coder `outputFingerprint`; the candidate revision therefore still covers
  untracked new paths.

## Determinism audit (hard rules)

`grep -nE "time\.Sleep|time\.After|time\.Tick|NewTimer|rand\.|time\.Now|select \{"` over the
four files returns no match (exit 1): no sleep, no timer, no `time.Now`, no `rand`, and no
`select` statement at all — so there is no timeout-standing-in-for-synchronisation anywhere.

- Synchronisation vocabulary in every concurrency test is one `close(start)` barrier plus one
  `sync.WaitGroup` join, with results written to per-goroutine slots (`selectionSlots[slot]`,
  `readerSlots[slot]`, `registrationErrors[slot]`, `calls[index]` from a per-call closure).
  The tests add no shared mutable state of their own.
- No map-iteration-order dependence found. `stubIDs`, `legalOutcomes`, `legalStatuses`,
  `controlStatuses`, `seen` are lookup-only; the config assertion uses `reflect.DeepEqual`
  (map-content comparison); the compat answer checks use `len(first)` plus per-name lookups;
  `sort.Strings` in `ValidateRequest` never feeds an ordering assertion.
- Wall-clock independence: no assertion reads `InteractionRecord.Received`, which
  `Engine.transition` fills from `time.Now()`. The control fuzz target's byte-identical
  repeat comparison covers `/requests` while the engine journal is empty (the target also
  asserts the control plane never journals), so no timestamp can differ between the two
  calls. `newRequestRecord` formats the timestamp but is only reached with an empty journal.
- Repeat stability: `go test -race -count=5 -run '^TestConcurrent|^TestControlStateRaceFree'
  ./internal/engine ./internal/control` → 35 tests, exit 0, no race warning (five repeats of
  each concurrency test).

## Non-vacuity: mutation evidence

Method: a throwaway full copy at `/tmp/fj040-mut` (outside the repository, no `.git`
required) built from a pristine snapshot; one product-code fault injected per row with a
single-match text replacement; the focused test re-run with `/usr/local/go/bin/go test`
(mutated source only); the mutated file restored from the snapshot; `diff -rq` verified
clean after every row. Nothing in `/Users/mak/git/fake-jev-FJ-040` was mutated.

Race coverage rows (run with `-race`):

| mutation (product fault injected) | command target | observed |
| --- | --- | --- |
| remove `e.mu.Lock()/Unlock()` from `Engine.transition` | `-run '^TestConcurrentTransitionsRetainExactlyTheAdmittedSequences$' ./internal/engine/` | exit 1, `WARNING: DATA RACE` ×8 |
| remove the lock from `Engine.Interactions` (readers race appenders) | `-run '^TestConcurrentRegistrationSelectionJournalAndResetAreRaceFree$' ./internal/engine/` | exit 1, `WARNING: DATA RACE` ×2 |
| remove the mutex from `Registry.selectInvocation` | engine concurrency tests | exit 0 — the registry lock is redundant behind `Engine.mu`; that mutation is undetectable through the `Engine` API and is not claimed otherwise |

Conservation/envelope rows (each maps to a test whose claim it breaks):

| id | mutation | observed failure line |
| --- | --- | --- |
| E2 | `journal.reserve` no longer checks capacity | `race_test.go:311: admitted 256 of 256 calls, want exactly 64` |
| E3 | `journal.reserve` stops advancing `nextSequence` | `race_test.go:320: interaction 1 has sequence 1, want 2` |
| E4 | `Registry.register` drops the duplicate-ID check | `race_test.go:379: admitted 32 duplicate registrations, want exactly one` |
| E5 | `Engine.ClearRequestHistory` also rewinds invocation counts | `race_test.go:451: invocation count is 0, want 256` |
| E6 | `Engine.Reset` stops rewinding `InvocationCount` | `race_test.go:263: reset left stub "static-alpha" at count 32, position 0` |
| E7 | `Engine.Reset` stops removing dynamic stubs | detected in 7 of 10 runs (`reset left ... stubs`), not in 3 — see below |
| E8 | `badRequest` writes 500 instead of 400 | `fuzz_test.go:136: … returned status 500` (seed corpus) |
| E9 | health response carries a per-call counter | `fuzz_test.go:173: repeated control request GET "/__fake/v1/health" … then …` (seed corpus) |
| E10 | health journals an interaction | `fuzz_test.go:163: control request GET "/__fake/v1/health" admitted 1 interactions` |
| E11 | `Registry.register` stops appending the stub | `fuzz_test.go:285: call 32: create answered 500` (`TestControlStateRaceFree`) |
| E12 | `stubMutations` lock removed from `reset` + `registerStub` | detected in 1 of 20 `TestControlStateRaceFree` runs |
| E13 | `compileStub` always rejects (creates answer 400) | `fuzz_test.go:310: admitted 0 of 8 distinct dynamic stubs` |
| E14 | `API.ServeHTTP` returns 500 for paths longer than 120 bytes | seed run clean; fuzzer failed in 0.38 s on a generated input (`fuzz_test.go:136`) |
| E15 | models-route `Decode` depends on the request body | seed run clean; fuzzer failed in 0.20 s (`fuzz_test.go:188`) |
| E16 | `Decode` returns a different diagnostic than `ValidateRequest` | `fuzz_test.go:165: Decode diagnostic … differs from the validator diagnostic` |
| E17 | `Decode` accepts an unknown route | `fuzz_test.go:192: Decode accepted the unknown route POST "/v1/other"` |
| E18 | noul answer field renamed (`noul` → `noulValue`) | `fuzz_test.go:176: answer "urgent" carries no "noul" helper` |
| E19 | `Decode` writes to the caller's body slice | `fuzz_test.go:211: profile decoding mutated the caller input` |
| E20b | default host set to a wrong value | `fuzz_test.go:115: server defaults are "mutation-host":8787, want "127.0.0.1":8787` |
| E21 | `Load` returns a config together with an error | `fuzz_test.go:106: Load returned a configuration together with error …` |
| E22b | `Load` fabricates `len(data)+1` valid stubs | `fuzz_test.go:115: loaded 20 stubs from 19 configuration bytes` |
| E23 | `Load` embeds a wall-clock value | `fuzz_test.go:113: Load is not reproducible …` |
| E24 | `Load` writes to the caller's byte slice | `fuzz_test.go:119: Load mutated the caller's configuration bytes …` |

Mutations that did not produce a failure, and why (recorded so the negative results are not
mistaken for dead assertions):

- E20 `applyDefaults` made a no-op → exit 0, because `validateValues` then rejects `mode: ""`,
  so no successful load exists to assert defaults on. Defence in depth, not a dead assertion;
  E20b shows the same assertions do fire.
- E22 first form (`cfg.Stubs = append(cfg.Stubs, cfg.Stubs...)`) → exit 0, because no
  seed-corpus input that loads successfully declares a stub, so the doubling never ran.
  E22b isolates the non-amplification assertion with fabricated valid stubs.
- E7 and E12 are interleaving-dependent: E7's "reset removes dynamic stubs" half is only
  observable when a dynamic registration lands after the last concurrent `RemoveDynamic`
  (7/10 runs), and E12 needs a reset to land inside the two-call registration window
  (1/20 runs). Both are reachable, neither is deterministic in detection power.

## Assertions that cannot fail (disclosed plus newly identified)

- `journalFullFailures > 1` and `reader.journalFull == true` in the mixed engine test:
  capacity 128 versus 72 possible admissions means `reserve` never reports full there. The
  coder already recorded this as a vacuous envelope; the real journal-full proof is the
  dedicated capacity-64 test (E2 shows it is live).
- `stub.Source ∈ {static, dynamic}` in all engine assertions: `Registry.register` assigns only
  those two constants, so the check cannot fire while the registry has one writer. Envelope
  only.
- "every non-`journal_full` failure carries a non-nil `RequestSequence`": both `RecordFailure`
  and `transition` always set it, so this half cannot fire (the `journal_full` half can).
- Control concurrency `created == stubWriters`: its only falsification path is a create
  answering 400/409/500 (E11, E13). It cannot detect a registration that a reset removes,
  because such a create was already counted as created; the test's own comment ("every
  distinct valid stub admitted exactly once") matches what is asserted, while the coder
  artifact's stronger phrasing ("a reset that eats a registration would show up here") is not
  what the assertion proves.
- Config non-amplification (`len(cfg.Stubs) <= len(document)`) is live (E22b) but has no
  realistic product falsification path: every output stub needs at least ~2 bytes of source
  (block stub or YAML alias `*a`), so the ratio cannot exceed 1 through alias amplification.
- All other assertions checked (engine envelopes, control status/envelope/repeat/journal,
  compat route parity, answer contract, models body independence, config defaults/error shape/
  determinism/caller-slice) each have a mutation row above that makes them fail.

## Comment over-claims

- `internal/config/fuzz_test.go`, `configFuzzMaxBytes` comment: "configFuzzMaxBytes caps the
  input the fuzzer may hand to the decoder, so no assertion or parse cost can grow beyond the
  pre-agreed configuration limit." The input is capped; the parse cost is not. Measured with
  a probe in the throwaway copy: a 62,837-byte YAML document using anchors/aliases makes one
  `config.Load` allocate 51.7 MB in 97 ms (≈830× the input); a 109,821-byte variant allocates
  56.5 MB in 504 ms. yaml.v3 permits alias expansion up to roughly 400k decode operations
  (~100 MB) for a compliant document, and the target calls `Load` twice per iteration while
  the guard row runs 10 workers, so one fuzz iteration can be worth hundreds of MB of
  transient allocation. The comment's first clause is asserted by the guard; the second is not.
- `internal/control/fuzz_test.go` file comment says malformed input "is answered with a
  controlled error envelope"; the asserted property is the status set {200,201,204,400,404,
  405,409} plus envelope shape, so a malformed body on a read-only route legitimately returns
  200. Phrasing, not a false property.
- `internal/compat/jev/v1/fuzz_test.go`, `assertGeneratedAnswers` doc ("a rejection is always a
  non-empty InvalidStubResponseError") is stronger than the specifier artifact's `errors.As`
  wording, which the coder flagged. It matches the current product contract (every
  `GenerateAnswers` error path goes through `invalidStubResponse`) and E18 shows it can fail.
- All other doc comments checked (engine `race_test.go` file and per-test comments, control
  `readOnlyControlRoute`, the fuzz target preambles): each stated claim maps to an assertion
  with a firing mutation above.

## Fuzz boundedness and termination

- Caps are the first statement of each fuzz body, and the remaining work is
  input-proportional only up to the cap: control 1 KiB path / 256 B content type / 16 KiB
  body; compat 64 B method / 512 B target / 512 B authorization / 16 KiB body / 16 KiB
  configured, with answer generation gated on `len(request.Questions) <= 32`; config 64 KiB.
  No loop over unbounded input; no input-proportional allocation without a cap other than the
  YAML alias expansion recorded above.
- The control target's `method` argument is uncapped, but every use is an equality comparison
  against a literal (length-first), so its cost is O(1); no change is needed.
- Termination in the guard row's own mode (coverage-instrumented `go test -run '^$' -fuzz
  '^X$' -fuzztime 30s`, fresh `GOCACHE`): control exit 0 (3.8M execs), compat exit 0 (5.2M
  execs). The config target did not complete cleanly — see the next point.
- Config-target instability observed: the exec counter froze at 0/sec for 6–40 s in 5 of
  about 8 config-target runs (never in any control or compat run). One compiled-binary run
  reported `fuzzing process hung or terminated unexpectedly while minimizing: EOF`, and one
  `go test -fuzz` run from a fresh cache exited 1 with a failing input
  (`[]byte("0: \n01:")`) that does not reproduce — it passes standalone and as a corpus
  entry. A probe that printed whenever a `Load` call exceeded 50 ms printed nothing during a
  20 s / 262k-exec run, so the freeze could not be attributed to a slow `Load`; the mechanism
  was not determined.

## Genuine product defects found by this item's own fuzz targets (not fixed; no product source edited)

Both are in `internal/config`, outside `allowedFiles`, so neither is fixed here and no
product file was edited. Both need a PM-opened follow-up ticket.

F1 — `config.Load` accepts a case-variant duplicate `schemaVersion` key and returns a
configuration its own `Validate` rejects.

- Minimal reproduction (37 bytes):
  `config.Load([]byte("{\"schemaVersion\":1,\"sChemAVErsion\":0}"))` → `cfg != nil`, `err == nil`,
  `cfg.SchemaVersion == 0`, and `config.Validate(cfg)` reports `schemaVersion must equal 1`.
  The same holds for `SCHEMAVERSION`.
- Discovered by this item's `FuzzConfigMalformedInput`: on a cold fuzz cache the target
  minimized and wrote `[]byte("{\"schemaVersion\":1,\"sChemAVErsion\":0}")` at 1.5 s; a
  second fresh-cache run produced the variant
  `[]byte("{\"schemaVersion\":1,\"sCheMAVersion\":10}")` at 0.53 s. Both fail
  `fuzz_test.go:115: loaded schemaVersion is <n>, want 1`. Replaying the artifact fails in
  0.01 s.
- Mechanism: `encoding/json` matches object keys to struct fields case-insensitively, so the
  second key overwrites the decoded `Config.SchemaVersion`; `validateDocument` inspects the
  raw document by exact key (`root["schemaVersion"] == 1`) and `validateValues` never
  re-checks `cfg.SchemaVersion`.
- Effect reachable from untrusted input: a configuration document violating the schemaVersion
  contract loads with a nil error, and readers (host checks, control `/meta`
  `configSchemaVersion`) observe 0.

F2 — `config.Load`'s diagnostic is not reproducible for YAML input with non-string keys.

- Minimal reproduction (6 bytes): 200 calls of `config.Load([]byte("{07,2}"))` produced
  175 × `convert YAML configuration: object key 7 is not a string` and
  25 × `convert YAML configuration: object key 2 is not a string`.
- Discovered by `FuzzConfigMalformedInput` in the guard row's mode (fresh `GOCACHE`, exit 1 at
  26.5 s): `fuzz_test.go:103: Load diagnostics differ between identical calls:
  "convert YAML configuration: object key 2 is not a string" then "…object key 7 is not a string"`.
- Mechanism: `normalizeYAMLValue` iterates a `map[any]any` (randomised order) and reports the
  first non-string key it encounters.
- The determinism assertion that caught this is a method choice recorded by the specifier
  (§22.2 is silent on it), justified there by reproducible diagnostics.

## Command outcomes (repository, revision e60433b, four new files present)

| command | result |
| --- | --- |
| `gofmt -l` on the four files | no output (exit 0) |
| `git status --short` | the eight paths listed above; zero staged paths |
| `go test ./internal/engine ./internal/control ./internal/compat/jev/v1 ./internal/config -count=1` | exit 0 |
| `go test -race ./... -count=1` (run 1) | exit 0 |
| `go test -race ./... -count=1` (run 2, raw output captured) | exit 0, one `ok` line per package, 10 packages, no `WARNING: DATA RACE` — repeatable across the two runs |
| `go test -race -count=5 -run '^TestConcurrent\|^TestControlStateRaceFree' ./internal/engine ./internal/control` | exit 0, 35 test runs, no race warning |
| `go vet ./...` | exit 0, no diagnostics |
| `go run ./cmd/guard arch` | exit 0, `findings: []`, 64 files / 10 packages |
| `go run ./cmd/guard lint` | exit 0, `findings: []`, 64 files / 10 packages |
| `go run ./cmd/guard trace` (extra) | exit 0, `findings: []`, 17 covered markers, 26 test files |
| `./scripts/candidate-fingerprint candidate` | `525ee08845fbb9098ba37a5146877529863cddd57dcb263e9050628d63b9a84a` |

Fuzz rows were run in the throwaway copy, not in the repository, so that any crash artifact
would land outside `allowedFiles`: guard-row mode with a fresh cache gave control exit 0,
compat exit 0, and config exit 1 (F2). The full `go run ./cmd/guard fuzz` row was not re-run
in the repository for that same reason; its documented cost (coder: 2m12s for 4 targets every
30 s) is consistent with the per-target runs above. Consequence to record: with a cold fuzz
cache the config row fails within 30 s, so the row's green state depends on the shared warm
cache (`GOCACHE/fuzz/fake-jev/internal/config/...`).

## Cleanup applied: none

No edit was made, and this is deliberate. The four test files are byte-identical to the
pre-review snapshot, only the four allowed files exist as new content, and the candidate
fingerprint still equals the coder `outputFingerprint` (the required front matter pins both
fingerprints to that value, i.e. this stage is expected to be a no-change stage). The one
over-claiming comment (config `configFuzzMaxBytes`) is reported rather than reworded, because
the underlying behaviour — YAML alias amplification inside the cap — is a design question for
the PM, and a comment edit would both change the candidate fingerprint and stale the coder
and hardener artifacts.

## Residual risks

- F1 and F2 are real, reachable through this item's own fuzz target, and unfixed here; F1 is
  reachable from any configuration file.
- The config fuzz target is not allocation-bounded by its cap (≈830× measured amplification)
  and shows intermittent multi-second stalls plus one non-reproducing crash report; the guard
  fuzz row for this candidate is not stable across cold and warm cache states.
- §22.2's "matcher input" area remains unfuzzed: `allowedFiles` contains no
  `internal/engine` fuzz file, as the specifier recorded.
- Assertions listed under "cannot fail" (journal-full envelope, source envelope,
  `RequestSequence` envelope, control `created` semantics) are contained rather than removed;
  their stated properties are not falsifiable today.
- Fault-detection for the reset/dynamic and registration/reset atomicity claims is
  interleaving-dependent (7/10 and 1/20 in the mutation runs), so those claims are covered
  probabilistically rather than deterministically.
