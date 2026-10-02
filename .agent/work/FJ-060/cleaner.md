---
stage: cleaner
task: FJ-060
inputFingerprint: 5fb3c895eceb9e3a37c42b94a3cf6aabf0ebab41e6ab3a2720dea0d0fba8943e
outputFingerprint: 5fb3c895eceb9e3a37c42b94a3cf6aabf0ebab41e6ab3a2720dea0d0fba8943e
taskFingerprint: 41ba2cd97f56c930d6ac57fdd4e662aebc1be3c12067c6cc2bd7dac7194f93a1
gitHead: 9c43594
generatedAt: 2026-10-02T18:52:26Z
author: worker/FJ-060-cleaner
---

# Cleaner — FJ-060 (fuzz worker failures are not target crashes)

`./scripts/candidate-fingerprint candidate` printed
`5fb3c895eceb9e3a37c42b94a3cf6aabf0ebab41e6ab3a2720dea0d0fba8943e`, exactly the
coder's `outputFingerprint`; it is carried unchanged as both fingerprints above.
No product or test file was edited in this stage, so the candidate fingerprint is
identical to the coder's.

Inputs read: `./scripts/agent-context FJ-060`, `docs/agents/roles/cleaner.md`,
`.agent/work/FJ-060/specifier.md`, `.agent/work/FJ-060/coder.md`, the working-tree
diff of `cmd/guard/fuzz.go` and `cmd/guard/fuzz_test.go`, the full content of both
files, `cmd/guard/main.go`, `cmd/guard/guards.go`, and the guard-row consumer in
`scripts/verify-candidate`.

## Consumer trace (how `runOneFuzz`'s two returns are used)

`runOneFuzz` returns `([]Finding, error)`; both returns are consumed, and the
error return is not a non-finding path.

1. `runOneFuzz` → `runFuzz`: a non-nil `err` makes `runFuzz` return `nil, nil, err`
   immediately (`cmd/guard/fuzz.go`, the `if err != nil { return nil, nil, err }`
   after the `runOneFuzz` call).
2. `runFuzz` is the `"fuzz"` handler in `cmd/guard/main.go`. `run()` maps a handler
   error to `stderrf(stderr, "guard: %v\n", err)` and `return 2`, before any
   `emitReport` call — stdout stays empty.
3. `scripts/verify-candidate` `guard_row()` runs the built guard and captures
   `rc`. Its Python verdict parser treats `rc not in (0, 1)` as
   `fail\tguard.tool.failed\tguard fuzz: exit 2: <stderr>`. Any `fail` row drives
   the report status away from `passed`, and the script ends `RESULT: FAIL;
   exit 1`.

So an operational error is a **tool failure that fails verification**, not a
"non-finding" that can be ignored. A crash finding (`guard.fuzz.crash`, exit 1)
and an operational error (exit 2) are both non-zero guard exits and both produce
a failing verify row; they differ only in the recorded code
(`guard.fuzz.failed` vs `guard.tool.failed`) and in whether a JSON report is
emitted.

## False-pass analysis

Question: can a genuine crash, seed failure, race, or timeout now be swallowed
into an "operational error" that the guard treats as a non-finding?

Conclusion: **no false pass is reachable through this change.** An operational
error is routed to exit 2 and `fail guard.tool.failed`, which fails
`verify-candidate`; it is stricter than the crash row, never weaker. The change
can demote a classification from `guard.fuzz.crash` (exit 1) to a tool error
(exit 2), but it cannot turn any failing run into exit 0 or into an empty
finding list on a zero exit.

Evidence, all run against a freshly built `cmd/guard` from the candidate revision
(`go1.26.3 darwin/arm64`), each in a scratch module with the repository
`.agent/guards.json` and only `fuzz.fuzztime` adjusted:

| probe | guard exit | classification | note |
|---|---|---|---|
| live `f.Fuzz` body panics | 1 | `guard.fuzz.crash`, `p/testdata/fuzz/FuzzAlways/<hash>` | Go writes `Failing input written to` |
| committed seed (`f.Add`) panics | 1 | `guard.fuzz.crash`, declaring `_test.go:5` | `failure while testing seed corpus entry` |
| hard process kill in body: `os.Exit(1)` | 1 | `guard.fuzz.crash`, preserved input | Go attributes it (written input **and** seed marker present) |
| hard signal in body: SIGSEGV self-kill | 1 | `guard.fuzz.crash`, preserved input | same attribution |
| committed seed sleeps 12 s (`fuzztime 20s`) | 1 | `guard.fuzz.crash`, declaring `_test.go:8` | seed-timeout kept |
| live body sleeps 12 s, no seed (`fuzztime 20s`) | 1 | `guard.fuzz.crash`, preserved input under `p/testdata/fuzz/FuzzHang/` | genuine timeout kept, not swallowed |
| SIGTERM self-kill (worker killed, no attributable input) | 2 | none; `guard: ... unexpected failure: ... fuzzing process terminated by unexpected signal; no crash will be recorded ...` | old code fabricated a crash here |
| uncompilable package (`[build failed]`) | 2 | none; stderr `... [build failed]` | `TestRunFuzzBuildFailure` covers this |

Race is out of reach of this change: `runFuzz`/`runOneFuzz` do not run
`go test -race`; the race row is a separate `verify-candidate` check
(`check_go_race`) with its own command. The SIGTERM row is the only "genuine
failure demoted from crash to tool error", and it is an externally induced kill
rather than a target crash; it still exits 2.

Residual exposure, recorded rather than fixed: an operational error emits no
JSON, so a hypothetical consumer that inspected only `findings` and ignored the
exit code would see an empty result. The repository's only consumer of guard
output is `scripts/verify-candidate`, which does check `rc`, so no such consumer
exists here.

## Cleaner findings

- **Duplication.** None introduced. The two corpus-attribution tests previously
  lived inline in `runOneFuzz`; they now live once in `classifyFuzzFailure`
  (`cmd/guard/fuzz.go:99`), and `runOneFuzz` dispatches on the returned kind
  (`cmd/guard/fuzz.go:321`). `writtenInputPattern` and `seedFailureMarker` each
  have exactly one use site (the classifier). No copy of the marker logic
  remains.
- **Dead code.** None. `fuzzFailureKind` and all three constants are referenced;
  `writtenInputPattern` and `seedFailureMarker` are both still read; no unused
  import or helper (confirmed by `go vet ./...` and `go build`).
- **Comment accuracy.** Verified against the installed toolchain and the code:
  - `writtenInputPattern`'s "go1.26" and the classifier's "go1.26.3" both match
    `go version` output (`go1.26.3`); the classifier's fd-pressure claim
    (`--- FAIL: FuzzSafe` followed by `fork/exec ...: too many open files`, no
    marker) is the reproduction output below.
  - The `runOneFuzz` doc comment ("a written crash input is the primary finding
    ... a seed-corpus failure points at the declaring source; anything else ...
    is an operational error") matches the switch exactly.
  - The `fuzzFailureSeedCorpus` and `fuzzFailureOperational` doc comments match
    the observed Go lines.
  - One cosmetic inconsistency only: "go1.26" in `writtenInputPattern`'s comment
    vs "go1.26.3" in the classifier comment. Not changed — a comment-only edit
    would move the candidate fingerprint and stale the coder artifact for no
    behavioural gain.
- **Infrastructure-marker list.** There is **no marker denylist in product
  code**. The classifier is positive-only: it recognises the two
  corpus-attribution lines and treats everything else as operational. Nothing is
  over-broad, and no `internal/fuzz` string is hard-coded, so the rule does not
  depend on a volatile toolchain message catalogue. The diagnostic strings
  (`fork/exec`, `pipe`, `open /dev/null`, `communicating with fuzzing process`,
  `fuzzing process terminated ...`, `fuzzing process exited unexpectedly ...`)
  appear only as fixture text in `fuzz_test.go`; they are not matched by name,
  they classify as operational by absence of the two positive markers. That is
  the intended, non-over-broad construction.
- **Would the tests fail on revert?** Tested in throwaway copies under `/tmp`
  (the working tree was not touched):
  - New tests + HEAD `fuzz.go` (product reverted, tests kept): build failure —
    `undefined: classifyFuzzFailure`, `undefined: fuzzFailureKind`,
    `undefined: fuzzFailureOperational`, `rc=1`.
  - Classifier behaviour reverted (a bare `--- FAIL:` treated as corpus-attributed,
    i.e. the old `--- FAIL: <Name>` clause reintroduced inside the classifier):
    `TestClassifyFuzzFailureFDPressureIsNotCrash` fails and 8 of the 12
    `TestClassifyFuzzFailure` rows fail, `rc=1`. The regression test does bite.
  - Coverage gap found: reverting **only** the `runOneFuzz` dispatch to the old
    `--- FAIL: `+`tgt.Name` clause while leaving `classifyFuzzFailure` defined
    makes the whole package pass (`ok  fake-jev/cmd/guard`) — the unit tests pin
    the classifier, not the call site. The operational fall-through path is still
    integration-covered by `TestRunFuzzBuildFailure`, and there is no hermetic way
    to drive a bare `--- FAIL:` through `runOneFuzz` (it executes a real
    `go test -fuzz`), so closing this gap would need a seam that does not exist
    and is not required by the objective. Recorded as a residual risk, no code
    change made.

## Cleanup applied

None. No edit was clearly justified and strictly behaviour-preserving; the
structure is already minimal (one pure classifier, one dispatch site, no dead
code, no duplicated marker logic), and every candidate edit available in this
stage would either be cosmetic (moving the fingerprint without behaviour change)
or speculative scope widening. The coder's fingerprint is kept unchanged.

## Command outcomes

| command | outcome |
|---|---|
| `gofmt -l cmd/guard/fuzz.go cmd/guard/fuzz_test.go` | no output, `rc=0` |
| `go test ./cmd/guard -count=1` | `rc=0` (93 passed, 1 package) |
| `go test ./... -count=1` | `rc=0` (548 passed, 9 packages) |
| `go test -race ./... -count=1` | `rc=0` (548 passed, 9 packages) |
| `go vet ./...` | `rc=0`, no output |
| `go run ./cmd/guard fuzz` | `{"check":"fuzz","findings":[],"stats":{"targets":0}}`, `rc=0` |
| `bash -c 'ulimit -n 64; go test ./cmd/guard -run TestRunFuzzPass -count=1 -v'` | `rc=1`; recorded `fuzz: example.test/m/p: unexpected failure: ... --- FAIL: FuzzSafe (0.01s)     pipe: too many open files ...`, **no** `guard.fuzz.crash` finding |
| `./scripts/candidate-fingerprint candidate` | `5fb3c895eceb9e3a37c42b94a3cf6aabf0ebab41e6ab3a2720dea0d0fba8943e` |

`git diff --cached --name-only` is empty; nothing is staged. `.agent/work/FJ-060/state.json`
and `.agent/reports/` were not modified by this stage.

## Residual risks

1. **The `ulimit` command still exits non-zero (`rc=1`), now as an operational
   error rather than a fabricated crash.** This matches the specifier's
   observation 8 and the coder's recorded conflict with the task packet's
   "(must pass now)". The failure is the test environment genuinely being unable
   to host a fuzz run, not a regression: the fabricated `guard.fuzz.crash` is
   gone (verified) and no crash finding is emitted. The only in-scope way to make
   that command green would be to loosen `TestRunFuzzPass`, which the specifier
   forbids as suppression of a failure; the coder escalated and did not change
   it. Not changed here either, because it is a product/test-behaviour decision
   outside a behaviour-preserving cleanup.
2. **The unit tests pin `classifyFuzzFailure`, not `runOneFuzz`'s dispatch.**
   As shown above, a call-site-only revert passes the package tests. The
   operational fall-through is integration-covered by `TestRunFuzzBuildFailure`,
   and there is no seam to make a bare `--- FAIL:` deterministic through
   `runOneFuzz`; recorded, not fixed.
3. **Detection of a genuine target hang depends on the configured window
   exceeding Go's per-exec worker deadline.** Verified that both a seed hang and
   a live hang still produce `guard.fuzz.crash` at `fuzztime 20s`; this is the
   unchanged bounded-budget semantics targeted by the non-goal.
4. **Comment version drift**: `go1.26` (pattern comment) vs `go1.26.3`
   (classifier comment). Cosmetic; left untouched to keep the candidate
   fingerprint stable.
