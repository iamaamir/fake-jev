---
stage: specifier
task: FJ-060
inputFingerprint: 050f8a89923dd23b42190e976b6db7fa5b1327118ac6fcca2eb1ab4247e0a4a2
outputFingerprint: 050f8a89923dd23b42190e976b6db7fa5b1327118ac6fcca2eb1ab4247e0a4a2
taskFingerprint: 41ba2cd97f56c930d6ac57fdd4e662aebc1be3c12067c6cc2bd7dac7194f93a1
gitHead: 9c43594
generatedAt: 2026-10-02T18:19:04Z
author: worker/FJ-060-specifier
---

# Specifier — FJ-060 (fuzz worker failures are not target crashes)

Objective restated as observables: a bounded `cmd/guard` fuzz invocation whose
`go test -fuzz` subprocess fails because a Go fuzzing worker could not start,
could not communicate, or terminated without Go attributing the failure to a
specific input, is reported as an operational error (guard exits 2, verify
records `guard.tool.failed`) and produces no `guard.fuzz.crash` finding. A
failing input that Go itself records, a failing seed-corpus entry, and a target
whose live run exceeds Go's per-exec worker deadline keep producing exactly one
`guard.fuzz.crash` finding. No verdict is expressed anywhere in this artifact;
only `./scripts/verify-candidate FJ-060` emits results.

Work item acceptance is `C-QUAL-005`. The observable criteria below stay inside
`cmd/guard/fuzz.go` and `cmd/guard/fuzz_test.go`; they change no HTTP + JSON
surface, no compatibility behavior, and no configuration contract.

## Traces

- C-QUAL-005 — the guard fuzz check is part of the offline, credential-free
  §23 CI baseline the item's acceptance names.
- C-QUAL-004 — §22.2's fuzz invariant is what the crash/timeout reporting
  preserves.
- C-QUAL-001 — `go test ./...` must keep reporting success for the repository.
- C-QUAL-002 — `go test -race ./...` must keep reporting success.
- C-QUAL-003 — `go vet ./...` must keep reporting success.

## Spec basis, and where the spec is silent

- §23 baseline (`go test ./...`, `go test -race ./...`, `go vet ./...`, no
  model / provider key / paid service) and §23.1 Build matrix, §23.2 Release
  integrity are the only §23 clauses. There is no §23.3. §23 fixes *that* the
  checks run offline; it does not define the guard's failure taxonomy.
- §22.2 is the fuzz invariant: untrusted input may produce a controlled error,
  never a panic or state corruption (indexed as C-QUAL-004).
- Design `docs/superpowers/specs/2026-09-26-deterministic-guard-stack-design.md`
  §3.3 defines the fuzz runner: AST discovery, a bounded per-target window, and
  "a crash fails with `guard.fuzz.crash` and the failing input is committed
  under `testdata/fuzz/<Target>/` so it becomes a permanent, deterministic
  regression". §3.3 does not enumerate which toolchain lines count as a crash.
- Plan interpretation note 9 (`docs/superpowers/plans/2026-09-26-guard-stack.md`,
  not in `allowedFiles`) supplies the marker list the current code implements,
  and is the frozen interpretation that FJ-060 narrows. See observation 6.
- Consequence: the criteria below are the item-level reading of the objective
  (`state.json`, authority level 2) applied to design §3.3 — not a new public
  contract and not a specification change.

## Evidence that separates a target-level failure from a worker/infrastructure failure

All strings below were read verbatim from the active toolchain
(`/usr/local/go`, go1.26) and, where marked observed, reproduced in a scratch
module.

| Observed line | Emitted by | Discriminates? |
|---|---|---|
| `Failing input written to <path>` | `testing/fuzz.go:374`, only when the coordinator holds a `fuzzCrashError` for an input, written to `testdata/fuzz/<Name>/<hash>` | **target-level** — Go has attributed the failure to a specific input |
| `failure while testing seed corpus entry: <Target>/<entry>` | `internal/fuzz/fuzz.go:248`, seed warmup only | **target-level** — a committed seed deterministically fails; no new input is written (it is already in the tree) |
| `--- FAIL: <Name>` | the testing framework, on any `f.Fail()` | **no** — emitted identically for target failures and for worker start/communication/termination failures |
| `pipe: too many open files` (observed), `open /dev/null: too many open files` (observed), `fork/exec <path>: too many open files` | `os.Pipe()` / `os.Open("/dev/null")` / `os/exec.Cmd.Start()` inside `internal/fuzz/worker.go` `startAndPing` | worker-start diagnostic |
| `communicating with fuzzing process: <err>` | `internal/fuzz/worker.go:168` | worker-communication diagnostic |
| `fuzzing process terminated by unexpected signal; no crash will be recorded: <sig>` (observed), `fuzzing process terminated without fuzzing: <err>`, `fuzzing process exited unexpectedly due to an internal failure: <err>` | `internal/fuzz/worker.go:176,322,141` | worker-termination diagnostic |
| `fuzzing process hung or terminated unexpectedly: <err>` (observed) | `internal/fuzz/worker.go:145,186` | **ambiguous on its own** — appears on a live target hang *and* on a worker kill |

Verified worked examples (scratch modules at the same limits command shape as
the reproduction, run through the current `cmd/guard` binary):

- A live `f.Fuzz` body that panics → guard emits one `guard.fuzz.crash` whose
  path is `testdata/fuzz/FuzzSignal/<hash>` and whose file exists on disk. Go's
  output carries `Failing input written to testdata/fuzz/FuzzSignal/<hash>`.
- A worker terminated by a non-crash signal (Go prints
  `fuzzing process terminated by unexpected signal; no crash will be recorded:
  signal: terminated`) → no `Failing input written`, no seed marker, and Go's
  output still carries `--- FAIL: <Name>`. The current guard emits a spurious
  `guard.fuzz.crash` against the declaring `_test.go:line`.
- A live `f.Fuzz` body that exceeds Go's 10 s per-exec worker deadline → Go
  prints `fuzzing process hung or terminated unexpectedly: exit status 2`
  **and** `Failing input written to testdata/fuzz/<Name>/<hash>`; the guard
  emits `guard.fuzz.crash` with the preserved input.
- A seed entry that exceeds the same deadline → Go prints
  `failure while testing seed corpus entry: <Target>/seed#0` and
  `--- FAIL: <Target>`; no input is written.
- Under `ulimit -n 64` the guard's own `go test -fuzz` subprocess reports
  `--- FAIL: FuzzSafe (0.01s)` plus a worker-start diagnostic and no written
  input (see the reproduction section).

## What evidence distinguishes them, and the recommended reading

The discriminating evidence is **whether Go recorded a failing input
(`Failing input written to <path>`) or named a failing seed-corpus entry
(`failure while testing seed corpus entry: ...`)**. Those two lines exist only
where the toolchain has attributed the failure to concrete corpus data. A bare
`--- FAIL: <Name>` carries no such attribution.

**Recommended reading (Reading A).** Treat exactly two marker classes as
target-level: the written input line, and the seed-corpus line. Every other
non-zero `go test -fuzz` exit — including any output containing only
`--- FAIL: <Name>`, and including the `internal/fuzz` worker diagnostics — is
an operational error (guard exit 2, verify `guard.tool.failed`). The written
input line is evaluated first; the seed marker second.

Why Reading A:

1. `--- FAIL: <Name>` has no discriminating power: the toolchain emits it for
   both target failures and worker failures (verified above for a worker start
   failure and for a non-crash worker termination).
2. The two retained markers are exactly the cases in which Go has already
   decided that corpus data caused the failure, which is the operational
   content of design §3.3's "a crash fails with `guard.fuzz.crash`".
3. Reading A is the only reading that satisfies the objective's requirement
   that worker start, communication, and termination failures be classifiable
   as operational errors rather than target crashes.
4. Reading A does not suppress genuine crashes or timeouts: a live crash, a
   live hang, and a seed failure all retain at least one of the two markers
   (verified). The written input is preserved, so §3.3's "permanent,
   deterministic regression" property is unchanged.
5. The rejected alternative (keep `--- FAIL: <Name>`, subtract a denylist of
   worker diagnostics) requires enumerating `internal/fuzz` error strings,
   which are not a stable contract, and would silently misclassify any future
   worker message; note 9 concedes that its markers are toolchain-specific.
   Reading A needs only the two markers note 9 already treats as primary.

## Observable acceptance criteria

Each line is falsifiable from guard JSON output, the process exit code, or the
filesystem; no implementation detail is asserted.

- **AC-1 — worker start failure is an operational error.** For a bounded run
  whose `go test -fuzz` subprocess fails because no worker could be started
  (output contains a worker-start diagnostic such as `open /dev/null: too many
  open files`, `pipe: too many open files`, or `fork/exec <path>: too many open
  files`, and contains neither `Failing input written to` nor `failure while
  testing seed corpus entry`), `guard fuzz` produces no finding, exits 2, and
  writes a diagnostic to stderr with stdout empty.
- **AC-2 — worker communication failure is an operational error.** Same
  observable for output containing `communicating with fuzzing process: ...`
  with neither marker present.
- **AC-3 — worker termination without an attributable input is an operational
  error.** Same observable for output containing `fuzzing process terminated
  by unexpected signal; no crash will be recorded: ...`, `fuzzing process
  terminated without fuzzing: ...`, or `fuzzing process exited unexpectedly due
  to an internal failure: ...`, with neither marker present.
- **AC-4 — genuine target crash is still reported with the preserved input.**
  A target whose `f.Fuzz` body panics on a generated input yields exactly one
  finding with `code: guard.fuzz.crash`, a path under
  `<pkg>/testdata/fuzz/<Target>/`, and a message naming the target; the
  preserved file exists on disk.
- **AC-5 — seed-corpus failures are still reported.** A target whose committed
  seed fails during warmup yields exactly one `guard.fuzz.crash` finding whose
  path is the declaring `_test.go` at the `Fuzz*` line and whose message names
  the target.
- **AC-6 — genuine timeouts are still reported, never silently passing.** With
  the configured window exceeding Go's per-exec worker deadline, a target whose
  live `f.Fuzz` body hangs yields exactly one `guard.fuzz.crash` with the
  preserved input, and a target whose seed hangs yields exactly one
  `guard.fuzz.crash` at the declaring source. Neither is demoted to an
  operational error and neither is dropped.
- **AC-7 — a bare `--- FAIL: <Name>` is not a crash.** For a non-zero exit
  whose output contains `--- FAIL: <Name>` and a worker diagnostic but neither
  retained marker, the run produces no `guard.fuzz.crash` finding.
- **AC-8 — fail closed, never a silent pass.** Every non-zero `go test -fuzz`
  exit that yields neither retained marker is an operational error: guard exit
  2, stdout empty. No such run exits 0, and none is reported as a crash.
- **AC-9 — a clean bounded run is unchanged.** A target that neither crashes
  nor hangs within the configured window yields no findings, `stats.targets`
  equal to the discovered target count, and exit 0.
- **AC-10 — an uncompilable package is still an operational error naming the
  import path.** Output carrying `[build failed]` and neither marker yields
  exit 2 whose stderr names the failing import path.
- **AC-11 — scope is preserved.** The configured `fuzz.fuzztime` default stays
  `30s`; the run uses the configured value verbatim; the positive-window floor
  is unchanged; zero discovered targets still performs no `go test` invocation
  and reports `stats.targets: 0` with exit 0; the fuzz subprocess still runs
  offline; and classification consumes only the retained output of the existing
  subprocess, with no new dependency and no new failure code or report field.

## Reproduction, and the expected before/after

Confirmed reproduction (environment-triggered; 32 load-only invocations did not
trigger it):

```text
bash -c 'ulimit -n 64; go test ./cmd/guard -run TestRunFuzzPass -count=1 -v'
```

Observed before (this workspace, `9c43594`): the run reports failure, and
`runFuzz` returns `findings=[{guard.fuzz.crash p/fuzz_p_test.go 5 fuzz target
FuzzSafe failed}]` for a target whose `f.Fuzz` body is empty. The underlying
`go test -fuzz` output for the same limit is `--- FAIL: FuzzSafe (0.01s)`,
`    open /dev/null: too many open files` (also observed at neighbouring
limits: `pipe: too many open files`), then `FAIL` / `exit status 1`, with no
`Failing input written to` line.

Expected after: no `guard.fuzz.crash` finding exists for that run; the failure
is an operational error, so `go run ./cmd/guard fuzz` (or the built guard) exits
2 with a `guard: ...` diagnostic on stderr and empty stdout, and
`verify-candidate` would record `guard.tool.failed` rather than
`guard.fuzz.crash`. `TestRunFuzzPass` asserts a clean bounded run, so under the
same fd limit it can still report a test failure — now as an operational error,
with no fabricated crash finding (see observation 8). In an unconstrained
environment the same test observes `findings=[]`, `stats.targets=1`, exit 0.

Pressure-independent evidence for the same classification (deterministic, no fd
limit): a fuzz target whose `f.Fuzz` body sends the worker process SIGTERM to
itself. Its `go test -fuzz` output is `--- FAIL: FuzzSignal (0.04s)`,
`    fuzzing process terminated by unexpected signal; no crash will be
recorded: signal: terminated`, then `FAIL` / `exit status 1`, with no written
input. The current guard reports `guard.fuzz.crash fuzz_test.go 9 fuzz target
FuzzSignal failed`; Reading A requires an operational error instead. A
regression test built on this probe is unix-only (`syscall.Kill` is not defined
on Windows), so it would need a build tag or an equivalent trigger.

## Verification commands

```text
go test ./cmd/guard
go test ./...
go run ./cmd/guard fuzz
bash -c 'ulimit -n 64; go test ./cmd/guard -run TestRunFuzzPass -count=1 -v'
./scripts/verify-candidate FJ-060
```

`go run ./cmd/guard fuzz` in this repository discovers zero targets, so it is
the vacuous-pass path (`findings: []`, `stats.targets: 0`, exit 0). The
`ulimit` form is the reproduction: before/after differ in the classification
(operational error instead of `guard.fuzz.crash`), not in whether the fd-limited
environment can host a fuzz run. `./scripts/verify-candidate FJ-060` is the
repository completion rule (AGENTS.md) and is the only emitter of results.

## Spec-silent observations

1. **No §23.3 exists.** The task packet cites §23.1–§23.3; the specification
   has §23.1 Build matrix and §23.2 Release integrity only. Recorded, not a
   blocker: §23's baseline checks and offline requirement are sufficient
   anchors, and the guard classifier is internal tooling.
2. **A genuine target hang classifies as `guard.fuzz.crash`, not as an
   operational error.** Verified: a live `f.Fuzz` body exceeding Go's 10 s
   per-exec worker deadline makes Go print both
   `fuzzing process hung or terminated unexpectedly: exit status 2` and
   `Failing input written to testdata/fuzz/<Name>/<hash>`. The written-input
   marker decides, so the hang keeps its preserved, replayable input and
   §3.3's deterministic-regression property. The `hung or terminated` wording
   is deliberately *not* a demotion signal, because it also appears when a
   worker is killed with no attributable input. Demoting a live hang would
   violate the non-goal against suppressing genuine timeout findings.
3. **Hang detection is bounded by the configured window.** Because Go's
   per-exec worker deadline is ~10 s, a hanging target is reported only when
   the configured `fuzztime` exceeds it; the committed `30s` does. Under a
   shorter window a hanging target can complete its run undetected. This is the
   bounded-budget semantics targeted by the non-goal, and is unchanged.
4. **A hanging seed classifies as `guard.fuzz.crash` at the declaring source.**
   Verified: Go prints `failure while testing seed corpus entry:
   <Target>/seed#0` with no written input. The committed seed is the
   deterministic regression, so the seed marker is retained. Classifying it as
   an operational error would discard that signal.
5. **`fuzzOutput.truncated` stays unconsulted.** Not required by the objective.
   The retained-output classification already fails closed, so a truncated
   output with no matching marker becomes an operational error, never a silent
   pass. Residual limitation: a crash whose `Failing input written to` line
   falls beyond the 1 MiB bound would surface as an operational error rather
   than a crash finding. Recorded; out of scope.
6. **Interpretation note 9's `--- FAIL: <Name>` clause is superseded for
   worker-failure cases.** Its written-input and seed-marker clauses are kept
   verbatim. The plan file is not in `allowedFiles` and is not edited; this is a
   classification-precision change inside `cmd/guard`, not a public-behavior
   change.
7. **No retry is required.** The objective is classification, not recovery, and
   a retry would multiply the bounded fuzz budget (a non-goal). A worker that
   cannot start under fd pressure therefore yields an honest operational error.
8. **`TestRunFuzzPass` asserts the happy path and remains fd-sensitive.** Under
   the reproduction limit it can still report a test failure, because that
   environment genuinely cannot host a fuzz run; after this change the failure
   is an operational error instead of a fabricated `guard.fuzz.crash`. Making
   that specific test tolerate fd pressure is not required by the objective and
   would need either suppression of a failure (forbidden) or a retry (out of
   scope).
9. **The operational-error surface already exists.** Exit 2 maps to
   `guard.tool.failed` in `verify-candidate`; no new failure code, subcommand,
   or report field is introduced.
10. **No public contract moves.** Nothing in the data plane, control plane,
    configuration schema, or compatibility profile is touched, so no
    specification clause changes.

## Explicit non-goals

- No change to the bounded fuzz budget or to the `30s` `fuzztime` default.
- No suppression of genuine crash or timeout findings.
- No new dependency.
- No change to fuzz discovery, the positive-window floor, the offline
  subprocess environment, or the zero-target vacuous-pass convention.
- No new guard subcommand, failure code, or report field.
- No edit outside `cmd/guard/fuzz.go` and `cmd/guard/fuzz_test.go`.
