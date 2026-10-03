---
stage: qa
task: FJ-040
inputFingerprint: 4d9e1ad756b11f6771e4e2d24cb495181300a120125791cad14551ca17ff5f5e
outputFingerprint: 4d9e1ad756b11f6771e4e2d24cb495181300a120125791cad14551ca17ff5f5e
taskFingerprint: ae3ccc43bc6a790f3329233e8585379262d6e5b49e2751da583353742d06c6e4
gitHead: 433fc97
generatedAt: 2026-10-03T21:07:00Z
author: worker/FJ-040-qa-4
---

# FJ-040 — QA (final pass, author `worker/FJ-040-qa-4`)

This pass states observations, measurements, and limits. It declares no verdict and
uses no pass/fail wording of its own; where a tool line contains such wording it is
quoted verbatim and attributed to the tool. It was deliberately scoped small: the
expensive repository-wide gates for this candidate were already run by the PM on a
warm Go build cache and are cited from `/tmp/gates40.log` rather than re-run here.

Related artifacts (not modified by this pass):

- `.agent/work/FJ-040/qa-pass-2.md` — the restored full artifact of the previous QA
  pass (`worker/FJ-040-qa-2`, 405 lines). Its measurements and its falsification
  experiments are not duplicated below; they are summarised by reference. It is
  preserved under that name and was not modified.
- `.agent/work/FJ-040/hardener.md` — the fourth hardener pass
  (`worker/FJ-040-hardener-4`), which records the seed repair and the fingerprint
  accounting. Its front-matter `outputFingerprint` is read from the artifact below.

## Fingerprints and the one-path shift

| observation | value |
| --- | --- |
| `git rev-parse HEAD` | `433fc974544cbf912a93b3f50cd2011bdd9511fa` (`433fc97`) |
| `hardener.md` `outputFingerprint` (read from artifact front matter) | `4d9e1ad756b11f6771e4e2d24cb495181300a120125791cad14551ca17ff5f5e` |
| `./scripts/candidate-fingerprint candidate` (this pass) | `4d9e1ad756b11f6771e4e2d24cb495181300a120125791cad14551ca17ff5f5e` |
| `qa-pass-2.md` `inputFingerprint`/`outputFingerprint` (when written) | `8961ce46838962ae48d5bf900aa2013b9008df3c595b84cfa6e72c614d7cc434` |

The workspace digest and `hardener.md`'s recorded output agree, so this pass takes
`4d9e1ad7…` as both `inputFingerprint` and `outputFingerprint` (this pass changed no
file that the fingerprint covers; it wrote only this file, which lives under the
excluded `.agent/work/` prefix — see `EXCLUDED_PREFIXES` in
`scripts/candidate-fingerprint`).

The `8961ce46…` → `4d9e1ad7…` shift is attributable to exactly one covered path.
`candidate-fingerprint` hashes the four scoped test files plus product/go sources; of
the four files, three still hash to the values `qa-pass-2.md` recorded and exactly
`internal/control/fuzz_test.go` moved:

```text
534141def7a50ab65ee4f3428e7f6d5996ab6a958c59d1524b06416fc7d45233  internal/control/fuzz_test.go  (per qa-pass-2.md)
afb63fea7ea86572745d1d94f0c75c7c22e29da323ff06ae964a324eea3e48b2  internal/control/fuzz_test.go  (now)
932a8eec55e382a5b2adbf450e78c159f5c7f60296008a4db6c7a28071dec3c8  internal/config/fuzz_test.go       (unchanged)
c757a234551693962e1673b718ac584e2fae3eee672e32d7193022e213486911  internal/engine/race_test.go       (unchanged)
c72cf7e9619e9fdfe30ec3ee3e9dfacd5a2a5112a42ce839e7f1eb40edf854e1  internal/compat/jev/v1/fuzz_test.go (unchanged)
```

That single path is the home of the one-byte seed repair (the stray `"` after the
opening `{` removed from the `repeated-key-bomb` literal), so the whole digest shift
is explained by that repair and nothing else. `internal/control/fuzz_test.go` is the
only covered path that moved.

## Per-criterion observations

The item's acceptance contract is `C-QUAL-002`, `C-QUAL-004`, `C-HOST-009`
(`.agent/work/FJ-040/state.json`; catalog rows in
`docs/development/acceptance-catalog.md`). The four scoped files are the item's
deliverable: `internal/engine/race_test.go` and the concurrency test in
`internal/control/fuzz_test.go` carry the race-freedom half; the three fuzz targets
carry the malformed-input half.

### C-HOST-009 — engine and control state race-free under `go test -race`

- PM-run, cited from `/tmp/gates40.log` (`=== go test -race ./... ===`), labelled
  PM-run: eight packages reported `ok` (`cmd/guard`, `internal/cli`,
  `internal/compat/jev/v1`, `internal/config`, `internal/control`, `internal/engine`,
  `internal/host/http`, `test/integration`) with no `WARNING: DATA RACE` line; the
  two packages with no test files reported `[no test files]`. This pass did not
  re-run `-race` (explicitly out of scope for this pass).
- The race-engineered cases themselves live in `internal/engine/race_test.go`
  (`TestConcurrentRegistrationSelectionJournalAndResetAreRaceFree`,
  `TestConcurrentTransitionsRetainExactlyTheAdmittedSequences`,
  `TestConcurrentDuplicateRegistrationAdmitsExactlyOneStub`,
  `TestConcurrentClearHistoryPreservesInvocationCounters`) and in the control
  concurrency test `TestControlStateRaceFree`. Their falsifiability is not something
  this pass re-measured; `qa-pass-2.md` § *Non-vacuity experiments* recorded it
  (fault M1 produced 35 `WARNING: DATA RACE` blocks and an admission-conservation
  failure; fault M2 produced the conservation failure without a race).
- Scoped cheap check this pass ran:
  `go vet ./internal/control` → exit status 0, no diagnostics.

### C-QUAL-002 — `go test -race ./...` on supported runners

- PM-run, cited from `/tmp/gates40.log`, labelled PM-run: `=== go test ./... ===`
  reported `ok` for the same eight packages and `[no test files]` for `cmd/fake-jev`
  and `examples/go`; `=== go vet ===` printed `vet rc=0`.
- The candidate's own verifier row set (PM-run, `/tmp/gates40.log`,
  `=== verify-candidate FJ-040 ===`) is quoted verbatim below as tool output,
  including its `RESULT:` line:

  ```text
  report: .agent/reports/FJ-040/report-20261003T210524Z.json
  summary: 35 passed, 0 failed, 1 skipped, 0 not_applicable

  RESULT: PASS
  ```

  The report is bound to the candidate fingerprint by the verifier; per
  `qa-pass-2.md`, this verifier revision's row set contains **no** `qa`/`hardener`
  stage row, **no** `go test -race` row, and **no** guard-fuzz row, so this
  artifact's evidence and the race/fuzz batteries are not bound by a verifier row.

### C-QUAL-004 — malformed untrusted input yields a controlled error, no panic, no corrupted state

- This pass's seed-corpus run of the control target, exact command and output:

  ```text
  $ go test ./internal/control -run FuzzControlMalformedInput -count=1
  ok  	fake-jev/internal/control	0.243s
  ```

  (`-run FuzzControlMalformedInput` executes the registered seed corpus under the
  ordinary runner; `seed#10` is `repeated-key-bomb`. The seed corpus run is the
  deterministic part; the fuzzing window is not what this pass ran.)
- PM-run guard fuzz, cited from `/tmp/gates40.log` (`=== guard fuzz ===`), labelled
  PM-run — findings and target count quoted exactly:

  ```json
  {
    "check": "fuzz",
    "findings": [],
    "stats": {
      "targets": 4
    }
  }
  ```

  Four targets at this revision (the three new ones plus the pre-existing
  `cmd/guard` fuzz target); the guard reported an empty findings list.
- The live-route contrast below is this pass's own evidence that the control target's
  regression seed is not inert (and, by extension, that the assertion it feeds the
  fuzzer — response body ≤ 64 KiB — is reachable).

## Seed-repair proof: the 9,607-byte body reaches the duplicate-key path

A binary was built once (`go build -o /private/tmp/fj040-qa4/fake-jev ./cmd/fake-jev`)
and started as `serve --port 0 --host 127.0.0.1 --ready-file …`; a Python probe POSTed
to `/__fake/v1/stubs` on the reported ephemeral port. Bodies are constructed exactly
as the seed source constructs them:

- A = `{` + 1600 × `"a":1,` + `"a":1}` → **9,607 bytes**, valid JSON, one key name
  (`a`) repeated 1,601 times → 1,600 duplicate keys.
- B = `{"` + 1600 × `"a":1,` + `"a":1}` → **9,608 bytes** (the pre-repair literal
  shape), invalid JSON.

Observed response:

```text
bodyA len=9607 bodyB len=9608
A(valid-dup-key,9607B) -> status=400 content-type=application/json bodylen=106
A(valid-dup-key,9607B) -> body={"error":"fake_jev_bad_control_request","message":"decode JSON configuration: duplicate object key \"a\""}
B(malformed,9608B) -> status=400 content-type=application/json bodylen=77
B(malformed,9608B) -> body={"error":"fake_jev_bad_control_request","message":"Invalid control request."}
```

Reading: the two bodies differ by one byte and by 29 response bytes. The 9,607-byte
body is answered by the **constant 106-byte** duplicate-key diagnostic
(`decode JSON configuration: duplicate object key "a"`), which is produced only after
the body has been parsed and has reached `config.Load`'s duplicate-key scan — i.e.
the repaired seed reaches the duplicate-key path that FJ-065 changed. The 9,608-byte
body is rejected earlier, by the generic **77-byte** `Invalid control request.`
response, before that path. That contrast — not the plain seed run alone — is the
proof that the repaired seed is not inert; it is the shape `qa-pass-2.md`'s
§ *Headline observation* identified as missing (that pass measured 9,608 B → 77 B and
9,607 B → 106 B and concluded the then-committed seed could not reach the regression
it documents). The repair recorded in `hardener.md` matches what is now in the file
(`internal/control/fuzz_test.go` lines 124–125).

## Scope confirmation

`git status --short` at the end of this pass (verbatim, two columns preserved):

```text
 M .agent/work/FJ-040/state.json
?? .agent/reports/FJ-040/report-20261003T110212Z.json
?? .agent/reports/FJ-040/report-20261003T151405Z.json
?? .agent/reports/FJ-040/report-20261003T161105Z.json
?? .agent/reports/FJ-040/report-20261003T191741Z.json
?? .agent/reports/FJ-040/report-20261003T193026Z.json
?? .agent/reports/FJ-040/report-20261003T193457Z.json
?? .agent/reports/FJ-040/report-20261003T193955Z.json
?? .agent/reports/FJ-040/report-20261003T210524Z.json
?? .agent/work/FJ-040/cleaner.md
?? .agent/work/FJ-040/coder.md
?? .agent/work/FJ-040/hardener.md
?? .agent/work/FJ-040/qa-pass-2.md
?? .agent/work/FJ-040/qa.md
?? .agent/work/FJ-040/specifier.md
?? internal/compat/jev/v1/fuzz_test.go
?? internal/config/fuzz_test.go
?? internal/control/fuzz_test.go
?? internal/engine/race_test.go
```

Reading, scoped to `433fc97`:

- The only tracked-path modification is `.agent/work/FJ-040/state.json`
  (work-item metadata). `git diff --stat 433fc97 -- .` reports that one path and only
  that one path: `1 file changed, 35 insertions(+), 10 deletions(-)`.
- The only untracked files are the four in-scope test files
  (`internal/engine/race_test.go`, `internal/control/fuzz_test.go`,
  `internal/compat/jev/v1/fuzz_test.go`, `internal/config/fuzz_test.go`) plus
  `.agent/` evidence: the stage artifacts (`specifier.md`, `coder.md`, `cleaner.md`,
  `hardener.md`, `qa-pass-2.md`, and this `qa.md`) and the verifier reports under
  `.agent/reports/FJ-040/`. `qa-pass-2.md` and `hardener.md` are evidence additions
  (the restored previous QA artifact and the hardener's finalisation record). This
  pass added only `qa.md`; the verifier report `report-20261003T210524Z.json` is the
  PM-run `verify-candidate` output, not an artifact edit.
- No product code, no script, no guard config, and no other test file changed:
  `git diff 433fc97 --stat -- . ':(exclude).agent'` produces no output, and every
  listed `??` path outside `.agent/` is one of the four in-scope test files.
  `git diff --cached --stat` is empty (zero staged paths).

Limits of this scope check: the four new test files are untracked, so
`git diff 433fc97 -- <them>` cannot show their content; `candidate-fingerprint`
covers them, and the three unchanged hashes above are three of those files.

## Residual risks and evidence limits

- **The YAML duplicate-key quadratic remains in the product, and the fuzz guard walks
  around it rather than asserting against it.** `internal/config/fuzz_test.go`'s
  guard skips YAML-path documents above its separator bound; inside the bound the
  target still reaches multi-millisecond / multi-megabyte `config.Load` calls, and
  just above it the documented cost grows sharply (`qa-pass-2.md` and `hardener.md`
  record 74.6 ms / 57 MB at 1,000 colon-free entries and 17.4 s / 13.9 GiB at 13,100
  entries). Nothing in the four files asserts a cost bound; the item's protection on
  that path is a skip.
- **The Go-fuzzer coordinator exec-counter freeze is harness-side and produces no
  non-zero exit.** `qa-pass-2.md` observed the config target's reported exec counter
  stopping early across repeated runs while the process still exited 0; it makes that
  target's reported throughput unrepresentative and could mask a real slowdown. The
  control target did not show it in that pass's four runs. This pass did not
  re-observe it (re-running fuzz windows is outside this pass's command list).
- **The control-level `stubMutations` lock is not falsifiable through its test.**
  Removing it did not make any assertion in the four files fail as far as the prior QA
  and hardener passes established, and no test in scope can observe it. The engine-level
  mutex is falsifiable (M1/M2 in `qa-pass-2.md`); the control-level one is not
  established by this item.
- **FJ-030 (unbounded configuration read via `LoadFile`/`os.ReadFile`) and FJ-021
  (post-reset `invalid_stub_response` recording race) remain open.** Neither was
  touched here: the config target calls `config.Load([]byte)` only, and the engine
  test takes its after-reset assertion after all workers join and a final `Reset`, so
  neither is asserted away or fixed by this item.
- **The expensive gates for *this* candidate were run by the PM on a warm build
  cache, not independently by this pass.** `go test ./...`, `go test -race ./...`,
  `go vet ./...`, `go run ./cmd/guard fuzz`, and `./scripts/verify-candidate FJ-040`
  are PM-run and are cited from `/tmp/gates40.log`; this pass ran only the cheap
  commands listed in "Per-criterion observations" and the live-route probe. The
  harness timeouts that motivated this scoping (five earlier attempts at this item
  died to them; each subagent sandbox starts with a cold Go build cache) mean the
  repository-wide results here are single PM-run samples, not repetitions.
- **§22.2's matcher-input fuzzing area remains uncovered** because `allowedFiles`
  contains no `internal/engine` fuzz file; `internal/engine` declares only the race
  test, and the matcher-input surface is therefore not fuzzed by this item.
- **Fuzz evidence is a sample, not a proof.** The PM-run guard-fuzz row is one
  observation at this revision (`targets: 4`, `findings: []`); this pass adds one
  deterministic seed-corpus run of the control target and one live-route probe. None
  of these bounds the fuzzers' state space.
- **Corpus and cache state are mutable.** Any baseline/skip counts quoted in the
  referenced artifacts read a shared warm fuzz cache and are snapshots; this pass
  quotes none of them as its own measurement.
- **Single host.** All observations were made on one host; timings are
  load-sensitive. The assertion-level observations this pass owns — the 106-byte vs
  77-byte response contrast, `gofmt`/`vet` exit statuses, and the scope listing — are
  deterministic in form.

## Commands this pass ran, and what each returned

| command | observation |
| --- | --- |
| `go test ./internal/control -run FuzzControlMalformedInput -count=1` | exit 0, `ok fake-jev/internal/control 0.243s` |
| `gofmt -l internal/control/fuzz_test.go internal/engine/race_test.go internal/compat/jev/v1/fuzz_test.go internal/config/fuzz_test.go` | no output, exit 0 |
| `go vet ./internal/control` | no diagnostics, exit 0 |
| `git status --short` | the listing quoted under *Scope confirmation* |
| `./scripts/candidate-fingerprint candidate` | `4d9e1ad756b11f6771e4e2d24cb495181300a120125791cad14551ca17ff5f5e` |
| `go build -o /private/tmp/fj040-qa4/fake-jev ./cmd/fake-jev` + live probe | 9,607 B → 400 / 106 B duplicate-key body; 9,608 B → 400 / 77 B generic body |

PM-run results cited (not run by this pass), from `/tmp/gates40.log`: `go test ./...`
(eight packages `ok`), `go test -race ./...` (eight packages `ok`, no race lines),
`go vet ./...` (`vet rc=0`), `guard fuzz` (`findings: []`, `targets: 4`),
`verify-candidate FJ-040` (`summary: 35 passed, 0 failed, 1 skipped, 0 not_applicable`
against `report-20261003T210524Z.json`), and `candidate-fingerprint candidate`
(`4d9e1ad7…`).
