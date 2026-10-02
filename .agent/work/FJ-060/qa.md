---
stage: qa
task: FJ-060
inputFingerprint: 325768fd9665e78ee4264d86eb09e1dc651a4918c6ad1e140276a5b89208de31
outputFingerprint: 325768fd9665e78ee4264d86eb09e1dc651a4918c6ad1e140276a5b89208de31
taskFingerprint: 41ba2cd97f56c930d6ac57fdd4e662aebc1be3c12067c6cc2bd7dac7194f93a1
gitHead: 9c43594
generatedAt: 2026-10-02T19:15:00Z
author: worker/FJ-060-qa
---

# QA — FJ-060 (fuzz worker failures are not target crashes)

This stage edited nothing under version control. Every probe was built under
`/private/tmp/fj060-qa`; the repository working tree was read-only for this
stage. No verdict is expressed here; only `./scripts/verify-candidate` emits
results.

Inputs read: `./scripts/agent-context FJ-060` (via `state.json`),
`docs/agents/roles/qa.md`, `docs/agents/README.md`,
`.agent/work/FJ-060/{specifier,coder,cleaner,hardener}.md`,
`cmd/guard/fuzz.go`, `cmd/guard/fuzz_test.go`, `cmd/guard/main.go`,
`cmd/guard/pkgscan.go`, the guard rows of `scripts/verify-candidate`,
`docs/superpowers/specs/2026-09-26-deterministic-guard-stack-design.md` §3.3,
`docs/development/acceptance-catalog.md` (C-QUAL-005 and neighbours), and the
installed toolchain (`go1.26.3`, `/usr/local/go`, darwin/arm64).

## Probe trees (author independence)

| tree | construction | discriminator |
|---|---|---|
| `/private/tmp/fj060-qa/base` | `git archive 9c43594 \| tar -x` (base revision, no `.git`) | pre-fix product code |
| `/private/tmp/fj060-qa/fixed` | `rsync -a --exclude .git --exclude .agent` of the working tree | candidate |
| `/private/tmp/fj060-qa/mut1..3` | copies of `fixed` with throwaway mutations | mutation probes |

`diff -r -q base/cmd/guard fixed/cmd/guard` reports exactly two differing
files: `fuzz.go`, `fuzz_test.go`. Guard binaries built from each tree:
`guard-base`, `guard-fixed`.

## Criterion map (observations, no verdicts)

| criterion | observation in this stage |
|---|---|
| AC-1 worker start failure is operational | 1a-1f: base fabricates a crash, candidate returns rc=2 with empty stdout; the captured `pipe:`/`open /dev/null:`/`fork/exec:` shapes classify kind=2 |
| AC-2 worker communication failure is operational | 1e `communicating with fuzzing process` -> kind=2; 1f `TestMain` stand-in -> rc=2 |
| AC-3 worker termination without attributable input is operational | 1e signal/without-fuzzing/internal-failure rows -> kind=2 |
| AC-4 genuine target crash reported with preserved input | section 2 row 1: rc=1, `guard.fuzz.crash` at `p/testdata/fuzz/FuzzAlways/771e938e4458e983`, file on disk; identical at base in a pristine module |
| AC-5 seed-corpus failure reported | section 2 row 2: rc=1, `guard.fuzz.crash` at the declaring `fuzz_p_test.go:5` |
| AC-6 genuine timeouts reported | section 2 rows 3-4 (`fuzztime 20s`): live hang -> rc=1 with preserved input; seed hang -> rc=1 at the declaring source |
| AC-7 bare `--- FAIL: <Name>` is not a crash | 1e bare row -> kind=2; 1f bare stand-in -> base rc=1/finding, candidate rc=2/no finding |
| AC-8 fail closed, never a silent pass | section 3: four listed operational shapes plus the fd-limited repository run in 1c, all rc=2 with empty stdout; trace to `main.run`'s `return 2` and `verify-candidate`'s `rc not in (0, 1)` -> `fail guard.tool.failed`; base's bare-`--- FAIL:` path was rc=1, so the candidate's disposition is the stricter one |
| AC-9 clean bounded run unchanged | `go run ./cmd/guard fuzz` rc=0, `stats.targets: 0`; repository `go test ./cmd/guard -count=1` rc=0 (97); scratch non-crashing run in 1d-1f clean when not fd-starved |
| AC-10 uncompilable package is an operational error naming the import path | `gbuild` probe rc=2, stderr names `example.test/m/p` and `[build failed]`; `TestRunFuzzBuildFailureEchoingSeedMarkerIsNotCrash` observed in the repository suite |
| AC-11 scope preserved | `.agent/guards.json` `fuzz.fuzztime` is still `30s`; zero-target runs still perform no `go test` invocation (`fuzz` row above); `main.run` still has exactly the four implemented checks; `runOneFuzz` consumes only the subprocess's retained output; `git diff` touches only the two allowed files |
| C-QUAL-005 (item acceptance) / §23 offline baseline | `verify-candidate FJ-060` rc=0 over the pinned module environment (`GOFLAGS=-mod=readonly GOPROXY=off GOWORK=off`), plus `go test`, `go test -race`, `go vet`, `go build`; no credential, model, or network call in any probe |
| C-QUAL-001/002/003/004 | repository `go test ./...` rc=0 (552), `go test -race ./...` rc=0 (552), `go vet ./...` rc=0; section 2 shows the fuzz path still reports a panicking target rather than passing it |

## 1. Original defect reproduced at 9c43594, absent on the candidate

### 1a. `ulimit` path, repository at 9c43594 (base tree)

```text
cd /private/tmp/fj060-qa/base
bash -c 'ulimit -n 64; go test ./cmd/guard -run TestRunFuzzPass -count=1 -v'
rc=1
=== RUN   TestRunFuzzPass
    fuzz_test.go:144: findings=[{guard.fuzz.crash p/fuzz_p_test.go 5 fuzz target FuzzSafe failed}] stats=map[targets:1], want a clean bounded run
--- FAIL: TestRunFuzzPass (0.64s)
FAIL
FAIL	fake-jev/cmd/guard	0.837s
```

Observed: a `guard.fuzz.crash` finding against a target whose `f.Fuzz` body is
empty — the fabrication the item names.

### 1b. Same command, candidate tree

```text
cd /private/tmp/fj060-qa/fixed
bash -c 'ulimit -n 64; go test ./cmd/guard -run TestRunFuzzPass -count=1 -v'
rc=1
=== RUN   TestRunFuzzPass
    fuzz_test.go:141: fuzz: example.test/m/p: unexpected failure: ... --- FAIL: FuzzSafe (0.01s)     pipe: too many open files FAIL exit status 1 ...
--- FAIL: TestRunFuzzPass (0.93s)
```

Observed: no crash finding; the run surfaces as an operational error, and
`TestRunFuzzPass` (which asserts a clean bounded run) still reports a test
failure because the fd-limited environment cannot host a fuzz run. This is
specifier observation 8. The run-list expectation that this command exit 0 is
not met; see residual risk 1.

### 1c. Same command, repository working tree (candidate revision)

```text
cd /Users/mak/git/fake-jev-FJ-060
bash -c 'ulimit -n 64; go test ./cmd/guard -run TestRunFuzzPass -count=1 -v'
rc=1
=== RUN   TestRunFuzzPass
    fuzz_test.go:141: fuzz: example.test/m/p: unexpected failure: ... --- FAIL: FuzzSafe (0.01s)     open /dev/null: too many open files FAIL exit status 1 ...
```

Observed: same classification as 1b; the specific worker-start diagnostic
differed across runs (`pipe:` in the copy, `open /dev/null:` here), consistent
with the specifier's note that the wording is environment-dependent.

### 1d. Guard binary itself under the fd limit (scratch module, targets 1)

Scratch module `/private/tmp/fj060-qa/pfd` (`module example.test/m`, one
`FuzzSafe` with an empty body, repo-derived `.agent/guards.json`, `fuzztime 1s`).

Raw subprocess shape captured with
`bash -c 'ulimit -n 64; go test -count=1 -run ^$ -fuzz ^FuzzSafe$ -fuzztime 1s example.test/m/p'`
(saved at `/private/tmp/fj060-qa/captured-fd-limit-raw.txt`):

```text
fuzz: elapsed: 0s, gathering baseline coverage: 0/1 completed
fuzz: elapsed: 0s, gathering baseline coverage: 0/1 completed
--- FAIL: FuzzSafe (0.01s)
    pipe: too many open files
FAIL
exit status 1
FAIL	example.test/m/p	0.580s
```

```text
bash -c 'ulimit -n 64; /private/tmp/fj060-qa/guard-base fuzz'
rc=1  stdout: {"check":"fuzz","findings":[{"code":"guard.fuzz.crash","path":"p/fuzz_p_test.go",
                "line":5,"message":"fuzz target FuzzSafe failed"}],"stats":{"targets":1}}  stderr: (empty)

bash -c 'ulimit -n 64; /private/tmp/fj060-qa/guard-fixed fuzz'
rc=2  stdout: (empty)
      stderr: guard: fuzz: example.test/m/p: unexpected failure: ... --- FAIL: FuzzSafe (0.01s)
              pipe: too many open files FAIL exit status 1 FAIL	example.test/m/p	0.249s
```

### 1e. Classifier driven directly with captured output shapes (candidate)

`/private/tmp/fj060-qa/fixed/cmd/guard/qa_probe_test.go` (QA-authored, exists
only in the copy) calls `classifyFuzzFailure` on shapes captured from the live
toolchain; `go test ./cmd/guard -run TestQAClassifyCapturedShapes -count=1 -v`
observes (kind 0 = written input, 1 = seed corpus, 2 = operational):

```text
QA captured: ulimit -n 64 pipe worker start failure        -> kind=2 input=""
QA captured: open /dev/null worker start failure           -> kind=2 input=""
QA captured: fork/exec worker start failure                -> kind=2 input=""
QA worker communication failure                            -> kind=2 input=""
QA worker terminated by unexpected signal                  -> kind=2 input=""
QA worker terminated without fuzzing                       -> kind=2 input=""
QA worker internal failure                                 -> kind=2 input=""
QA bare --- FAIL with no marker                            -> kind=2 input=""
QA unrecognized output                                     -> kind=2 input=""
QA genuine written input                                   -> kind=0 input="testdata/fuzz/FuzzSignal/0a1b2c3d"
QA genuine seed corpus entry                               -> kind=1 input=""
```

### 1f. Deterministic stand-in at the production dispatch site (both trees)

A scratch package whose `TestMain` prints the verbatim fd-pressure shape
(`--- FAIL: FuzzSafe (0.01s)`, `    pipe: too many open files`, `FAIL`,
`exit status 1`) and calls `os.Exit(1)` — no descriptor exhaustion — drives
`runFuzz` → `runOneFuzz` with no fd limit:

```text
BASE  (base/cmd/guard/qa_rf_probe_test.go, -run TestQAProbeBaseFabricatesCrash)
    BASE findings=[{Code:guard.fuzz.crash Path:p/fuzz_p_test.go Line:18 Message:fuzz target FuzzSafe failed}] stats=map[targets:1] err=<nil>
FIXED (fixed/cmd/guard/qa_probe_test.go, -run TestQAProbeFixedIsOperational)
    FIXED findings=[] stats=map[] err=fuzz: example.test/m/p: unexpected failure: ... --- FAIL: FuzzSafe (0.01s)     pipe: too many open files FAIL exit status 1 ...
```

A second stand-in printing only `--- FAIL: FuzzSafe (0.01s)` then exiting 1:

```text
base  gspoofbare: rc=1, {"code":"guard.fuzz.crash","path":"p/fuzz_p_test.go","line":15,...}   (fabricated)
fixed gspoofbare: rc=2, stdout empty, stderr guard: fuzz: example.test/m/p: unexpected failure: --- FAIL: FuzzSafe (0.01s) FAIL exit status 1 ...
```

AC-1, AC-2, AC-3, AC-7 and AC-8 observations: worker start, communication and
termination shapes all take the operational disposition; a bare
`--- FAIL: <Name>` with a worker diagnostic is not a crash on the candidate and
was a crash at base.

## 2. Genuine problems still produce a visible non-zero outcome

Each probe ran the `guard-fixed` binary in its own scratch module with
`fuzztime` as noted; scratch modules hold only the target under test.

| probe | `guard fuzz` observed | emitted on stdout |
|---|---|---|
| fuzzer panics inside `f.Fuzz`, `fuzztime 1s` (`/private/tmp/fj060-qa/gcrash`) | rc=1 | `{"check":"fuzz","findings":[{"code":"guard.fuzz.crash","path":"p/testdata/fuzz/FuzzAlways/771e938e4458e983","line":0,"message":"fuzz target FuzzAlways crashed; failing input preserved"}],"stats":{"targets":1}}`; the file `gcrash/p/testdata/fuzz/FuzzAlways/771e938e4458e983` exists on disk |
| committed seed panics during warmup, `fuzztime 1s` (`gseed`) | rc=1 | `{"check":"fuzz","findings":[{"code":"guard.fuzz.crash","path":"p/fuzz_p_test.go","line":5,"message":"fuzz target FuzzBoom failed"}],"stats":{"targets":1}}` |
| live `f.Fuzz` body sleeps 15 s (exceeds go's per-exec worker deadline), `fuzztime 20s` (`ghang`) | rc=1 | `{"code":"guard.fuzz.crash","path":"p/testdata/fuzz/FuzzHang/d69d388e7ef88217","line":0,"message":"fuzz target FuzzHang crashed; failing input preserved"}`; preserved file exists |
| committed seed sleeps 15 s, `fuzztime 20s` (`gseedhang`) | rc=1 | `{"code":"guard.fuzz.crash","path":"p/fuzz_p_test.go","line":8,"message":"fuzz target FuzzSeedHang failed"}` |

Raw `go test -fuzz` output for the same shapes (captured to
`/private/tmp/fj060-qa/raw-crash.txt`, `raw-seed.txt`, and the seed-hang run
above) carries the two markers the classifier consumes:

```text
crash:  --- FAIL: FuzzAlways (0.01s) /     Failing input written to testdata/fuzz/FuzzAlways/771e938e4458e983
seed:   failure while testing seed corpus entry: FuzzBoom/seed#0
seeding: failure while testing seed corpus entry: FuzzSeedHang/seed#0
        fuzzing process hung or terminated unexpectedly: exit status 2
hang:   --- FAIL: FuzzHang ... /     Failing input written to testdata/fuzz/FuzzHang/d69d388e7ef88217
```

Base/candidate comparison for the live crash in pristine scratch modules
(no pre-existing `testdata/fuzz`) is identical: both emit
`guard.fuzz.crash` with `path p/testdata/fuzz/FuzzAlways/771e938e4458e983`,
rc=1. §3.3's "failing input committed under `testdata/fuzz/<Target>/`" property
is therefore unchanged for genuine crashes.

**Race-visible failure — not reachable through this path.** `runFuzz` never
invokes `go test -race` (`cmd/guard/fuzz.go` builds `go test -count=1 -run ^$
-fuzz ...`), and the design's race row is a separate `verify-candidate` check
(`scripts/verify-candidate`, `check_go_race`, `go test -race $pkgs`). Observed
on a scratch target with a real data race (`gracerace`, four goroutines
incrementing a shared counter):

```text
guard-fixed fuzz            -> rc=0, {"findings":[],"stats":{"targets":1}}   (race not observed; no -race)
go test -race -fuzz ^FuzzRace$ -fuzztime 2s example.test/m/p
                            -> rc=1, testing.go:1712: race detected during execution of test,
                               preceded by failure while testing seed corpus entry: FuzzRace/seed#0
```

Evidence limit recorded: race detection is outside the fuzz classifier's
reach; a race in a fuzz target is not surfaced by `guard fuzz` and would be
surfaced by the separate race row only if the race reproduces under
`go test -race ./...`.

## 3. No false pass: an operational error exits non-zero

Observed operational dispositions, each with empty stdout and a `guard:`
diagnostic on stderr, and each with guard rc=2:

| shape | probe | rc | stderr (first 120 chars) |
|---|---|---|---|
| fd-starved worker start (guard binary under `ulimit -n 64`) | `pfd` | 2 | `guard: fuzz: example.test/m/p: unexpected failure: ... --- FAIL: FuzzSafe (0.01s) pipe: too many open files ...` |
| uncompilable package | `gbuild` | 2 | `guard: fuzz: example.test/m/p: unexpected failure: # example.test/m/p [example.test/m/p.test] ./p.go:3:13: syntax error ... FAIL [build failed]` |
| bare `--- FAIL:` with no marker | `gspoofbare` | 2 | `guard: fuzz: example.test/m/p: unexpected failure: --- FAIL: FuzzSafe (0.01s) FAIL exit status 1 ...` |
| fabricated written-input line with a nonexistent path | `gwirespoof` | 2 | `guard: fuzz: FuzzSafe: crash input vanished: lstat .../p/testdata/fuzz/FuzzSafe/deadbeef: no such file or directory` |

Trace from observed behaviour to the process exit code:

1. `runOneFuzz` returns a non-nil error for the operational branch
   (`cmd/guard/fuzz.go`, `return nil, fmt.Errorf("fuzz: %s: unexpected failure: %s", ...)`).
2. `runFuzz` propagates it unchanged (`if err != nil { return nil, nil, err }`).
3. `main.run` maps a handler error to `stderrf(stderr, "guard: %v\n", err)` and
   `return 2` before any `emitReport`, so stdout is empty
   (`cmd/guard/main.go:52-55`). `main` is `os.Exit(run(...))`.
4. `scripts/verify-candidate` `guard_row()` parses the guard's stdout and treats
   `rc not in (0, 1)` as `fail\tguard.tool.failed\tguard fuzz: exit 2: <stderr>`
   (`scripts/verify-candidate:1472-1479`), recorded through `record_row` as a
   failing guard row (`:1547-1553`). The row is emitted with `required=1` when
   the G-H fuzz row binds (hardener stage due, or task-less CI:
   `:1612-1638`), so when it is emitted a demoted classification fails
   verification rather than being ignored. At the current work-item state the
   hardener stage is not due, so the row is not emitted at all (see residual
   risk 9).

Consequence observed from the shapes above: the candidate can move a
classification from `guard.fuzz.crash` (rc=1, JSON report) to an operational
error (rc=2, no JSON), but no observed non-zero `go test -fuzz` exit produced
rc=0 or an empty finding list on rc=0. At base a bare `--- FAIL:` produced
rc=1 with a crash finding; on the candidate it produces rc=2 — both non-zero,
so the downgrade cannot hide a worker failure: the operational error is the
stricter of the two rows.

## 4. Marker-breadth check

Near-miss and spoofing text driven through `classifyFuzzFailure` in the copy
(`-run TestQAObserveMarkerBreadth`, no assertions):

```text
OBSERVE similar words, not go's seed line                          -> kind=2 ("...seed corpus entries...")
OBSERVE seed words with a non-blank prefix                         -> kind=2 ("note: failure while testing ...")
OBSERVE seed words quoted inside a compiler echo                   -> kind=2 ("./fuzz_p_test.go:6:2: \"failure while testing ...\"")
OBSERVE seed words emitted verbatim at line start by the package   -> kind=1
OBSERVE written-input words with a nonexistent path                -> kind=0 input="testdata/fuzz/FuzzSafe/does-not-exist"
OBSERVE written-input words embedded in unrelated prose            -> kind=0 input="nowhere"
```

Observations:

- The seed marker is line-anchored with an optional leading-blank tolerance, so
  text that merely *contains* the words, or quotes them behind a diagnostic
  prefix, is not read as a target failure. This is the direction the design
  intends (AC-10), and it held in the compiler-echo end-to-end probe (`gbuild`)
  and in `TestRunFuzzBuildFailureEchoingSeedMarkerIsNotCrash`.
- The line-start tolerance is the only breadth available: text that reproduces
  go's line verbatim at the start of a line is treated as a target failure. A
  package that prints the marker itself therefore yields a fabricated
  `guard.fuzz.crash` (probe `gspoofseed`: rc=1, `{"code":"guard.fuzz.crash",
  "path":"p/fuzz_p_test.go","line":16,"message":"fuzz target FuzzSafe failed"}`).
  This is a mislabel, not a silent pass (exit 1, failing row), and the same
  probe at base is also rc=1 with the same finding — base additionally matches
  the marker anywhere in the output and treats a bare `--- FAIL:` as a crash.
  Recorded as residual risk 3; no acceptance criterion covers it.
- The written-input marker is matched unanchored, so two prose/spoof shapes
  reach kind=0; `preserveCrashInput` then rejects them because the path does
  not exist inside the package (observed rc=2 above), and
  `TestPreserveCrashInputRejectsEscape` covers the escaping-path variant. No
  fabricated finding is produced by those shapes.

## 5. Mutation check on throwaway copies

| mutation | copy | observed |
|---|---|---|
| full product revert (`git show 9c43594:cmd/guard/fuzz.go` over the candidate, tests kept) | `/private/tmp/fj060-qa/mut1` | `go test ./cmd/guard -count=1` rc=1, `[build failed]`: `undefined: classifyFuzzFailure`, `undefined: fuzzFailureKind`, `undefined: fuzzFailureOperational`, `too many errors` |
| dispatch-only revert (old `strings.Contains(text, "--- FAIL: "+tgt.Name)` clause re-injected at the `runOneFuzz` call site, classifier left intact) | `/private/tmp/fj060-qa/mut2` | rc=1; `--- FAIL: TestRunFuzzWorkerStartFailureIsNotCrash` (`fuzz_test.go:381: worker-start failure must be an operational error, not a clean run`) and `--- FAIL: TestRunFuzzLiveCrash` (`fuzz_test.go:125: finding = {Code:guard.fuzz.crash Path:p/fuzz_p_test.go Line:5 ...}, want preserved input under testdata/fuzz`) |
| seed marker reverted to the bare substring (`strings.Contains(text, "failure while testing seed corpus entry")`) | `/private/tmp/fj060-qa/mut3` | rc=1; `--- FAIL: TestClassifyFuzzFailure` (subtest `bare_---_FAIL_with_no_marker`) and `--- FAIL: TestRunFuzzBuildFailureEchoingSeedMarkerIsNotCrash` |

The dispatch-only mutation is the case the cleaner recorded as an uncovered
gap; the hardener's `TestRunFuzzWorkerStartFailureIsNotCrash` was observed to
catch it here, so the call site is now pinned by a repository test.

## Commands run, with observed outcomes

All in `/Users/mak/git/fake-jev-FJ-060` unless noted; `go1.26.3 darwin/arm64`.

| command | observed |
|---|---|
| `gofmt -l cmd/guard/fuzz.go cmd/guard/fuzz_test.go` | no output, rc=0 |
| `go test ./cmd/guard -count=1` | rc=0, `97 passed in 1 packages` |
| `go test ./... -count=1` | rc=0, `552 passed in 9 packages` |
| `go test -race ./... -count=1` | rc=0, `552 passed in 9 packages` |
| `go vet ./...` | no output, rc=0 |
| `go run ./cmd/guard arch` | rc=0, `{"check":"arch","findings":[],"stats":{"files":59,"packages":9}}` |
| `go run ./cmd/guard lint` | rc=0, `{"check":"lint","findings":[],"stats":{"files":59,"packages":9}}` |
| `go run ./cmd/guard trace` | rc=0, `{"check":"trace","findings":[],"stats":{"active":0,"covered":17,"test_files":22}}` |
| `go run ./cmd/guard fuzz` | rc=0, `{"check":"fuzz","findings":[],"stats":{"targets":0}}` (zero targets = no `go test` invocation) |
| `bash -c 'ulimit -n 64; go test ./cmd/guard -run TestRunFuzzPass -count=1 -v'` | rc=1; `fuzz_test.go:141: fuzz: example.test/m/p: unexpected failure: ... --- FAIL: FuzzSafe (0.01s)     open /dev/null: too many open files ...`; no `guard.fuzz.crash` |
| `./scripts/verify-candidate FJ-060` (run twice: before and after this artifact was written) | rc=0 both times; reports `.agent/reports/FJ-060/report-20261002T191316Z.json` and `report-20261002T191544Z.json`, each `status: passed` at `revision 9c43594` with `candidateFingerprint 325768fd...`; text output `summary: 20 passed, 0 failed, 0 skipped, 0 not_applicable`, `RESULT: PASS`. The 20 emitted rows include `guard arch`, `guard lint`, `guard trace` and the repository `go test`/`go vet`/`go build` rows; no `guard fuzz` row, no race row and no stage-chain row were emitted at this candidate state |
| `./scripts/candidate-fingerprint candidate` | `325768fd9665e78ee4264d86eb09e1dc651a4918c6ad1e140276a5b89208de31`, equal to the hardener's `outputFingerprint`; writing this artifact does not move it (`.agent/work/` is excluded by `scripts/candidate-fingerprint`, untracked paths are not listed) |

Working-tree state at the end of this stage: `git status --short` shows
`M cmd/guard/fuzz.go`, `M cmd/guard/fuzz_test.go`, `M .agent/work/FJ-060/state.json`
(mtime 2026-10-02T18:11:04Z, i.e. written before this session), the four earlier
untracked stage artifacts, the new untracked `.agent/work/FJ-060/qa.md`, and the
untracked `.agent/reports/FJ-060/` directory written by the required
`verify-candidate` run. `git diff --cached --name-only` is empty. Nothing was
staged; no production code, repository test, or `state.json` was written by this
stage.

## Residual risks and evidence limits

1. **The `ulimit -n 64` command still exits non-zero on the candidate tree.**
   The fabricated finding is gone and the run is an operational error, but
   `TestRunFuzzPass` asserts a clean bounded run, so an fd-starved environment
   still reports a test failure (same observation as specifier obs 8, coder and
   cleaner residual risk 1). The run-list expectation that this command exit 0
   is therefore not satisfied by the candidate. Making it exit 0 in scope would
   require loosening that test (suppression) or raising `RLIMIT_NOFILE`
   (unix-only code outside `allowedFiles`); neither was done by any stage.
2. **Toolchain-string dependence.** The classifier's positive markers are
   go1.26.3 verbatim lines. A toolchain that re-frames either line sends a
   genuine crash/seed failure to the operational branch: exit 2, no finding —
   observable, non-zero, but with the crash attribution lost (specifier obs 2/3
   and hardener residual risk 1). Not exercised here beyond go1.26.3.
3. **A package can reproduce the seed marker verbatim and get a fabricated
   crash finding** (probe `gspoofseed`, rc=1 at both base and candidate). The
   written-input marker is protected by `preserveCrashInput`, the seed marker is
   not backed by a filesystem check. Effect is a mislabeled failing row, not a
   silent pass, and the same injection at base was equally accepted (base is
   broader: substring anywhere, plus the bare `--- FAIL:` clause). No
   acceptance criterion covers it; recorded, not fixed.
4. **`fuzzOutput.truncated` remains unconsulted.** A crash whose
   `Failing input written to` line falls beyond the 1 MiB retained bound is
   classified as an operational error (exit 2) rather than a crash finding.
   Fails closed; specifier obs 5, hardener residual risk 4.
5. **Race visibility is outside this code path.** `guard fuzz` does not pass
   `-race`; the observed race probe left `guard fuzz` at rc=0 with no findings
   while `go test -race -fuzz` reported `race detected during execution of
   test`. The task's "race-visible failure if you can" could therefore not be
   produced through the guard's fuzz classification; the equivalent guard-level
   evidence is the separate `check_go_race` row (`go test -race ./...`, observed
   rc=0 in this repository).
6. **Live-hang detection is bounded by the configured window.** Both hang
   probes used `fuzztime 20s` (above go's ~10 s per-exec worker deadline). With
   a window shorter than that deadline a hanging target can finish undetected —
   unchanged bounded-budget semantics (specifier obs 3).
7. **Probe environment.** All probes ran on this machine (darwin/arm64,
   `/usr/local/go`, fd limit 64) in `/private/tmp`; the base tree came from
   `git archive 9c43594` and the fixed tree from the working-tree copy, so
   neither probe tree carries the repository's `.git` metadata, and
   `verify-candidate`'s task-bound checks were exercised only in the repository.
   The copy touched nothing in the repository beyond the report directory
   written by that command.
8. **`TestRunFuzzPass` remains fd-sensitive by construction** — this stage did
   not attempt to make the fd-limited environment deterministic; the
   determinism evidence is 1e/1f (classifier and dispatch) plus the criterion
   probes, not a repeat of the pressure run.
9. **`./scripts/verify-candidate FJ-060` emits no `guard fuzz` row and no
   stage-chain row at this candidate state** (20 rows: repository/guard
   arch+lint+trace/tooling checks). Its result therefore does not itself
   exercise the fuzz classifier or validate this artifact's header fingerprints;
   the classifier evidence in this artifact is the independent probe set, and
   the header values were confirmed equal to
   `./scripts/candidate-fingerprint candidate`. The run was repeated after this
   artifact was written and recorded the same 20 rows and `status: passed`.
