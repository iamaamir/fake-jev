---
stage: coder
task: FJ-060
inputFingerprint: 050f8a89923dd23b42190e976b6db7fa5b1327118ac6fcca2eb1ab4247e0a4a2
outputFingerprint: 5fb3c895eceb9e3a37c42b94a3cf6aabf0ebab41e6ab3a2720dea0d0fba8943e
taskFingerprint: 41ba2cd97f56c930d6ac57fdd4e662aebc1be3c12067c6cc2bd7dac7194f93a1
gitHead: 9c43594
generatedAt: 2026-10-02T18:47:00Z
author: worker/FJ-060-coder
---

# Coder — FJ-060 (fuzz worker failures are not target crashes)

**Final candidate fingerprint: `5fb3c895eceb9e3a37c42b94a3cf6aabf0ebab41e6ab3a2720dea0d0fba8943e`**

`inputFingerprint` is the specifier's unchanged candidate
(`050f8a89923dd23b42190e976b6db7fa5b1327118ac6fcca2eb1ab4247e0a4a2`); the
specifier edited no product code.

## Change summary

Only `cmd/guard/fuzz.go` and `cmd/guard/fuzz_test.go` were edited.

`cmd/guard/fuzz.go` (+49/-9):

- Added a pure classifier seam
  `classifyFuzzFailure(text string) (fuzzFailureKind, string)` with the three
  dispositions `fuzzFailureWrittenInput`, `fuzzFailureSeedCorpus`, and
  `fuzzFailureOperational`.
- `runOneFuzz` now dispatches on that kind instead of testing
  `writtenInputPattern` and `strings.Contains(text, "--- FAIL: "+tgt.Name)`
  inline.
- Removed the `--- FAIL: <Name>` clause from the target-level branch.

`cmd/guard/fuzz_test.go` (+111):

- `fdPressureOutput` const: the verbatim fd-pressure shape (`=== RUN`,
  `--- FAIL: FuzzSafe (0.01s)`, `    fork/exec ...: too many open files`,
  `FAIL`, `exit status 1`).
- `TestClassifyFuzzFailureFDPressureIsNotCrash` — the required regression: the
  fd-pressure shape classifies as `fuzzFailureOperational` with no input.
- `TestClassifyFuzzFailure` — deterministic table over the marker catalogue:
  worker start (`fork/exec`, `pipe`, `open /dev/null` variants), worker
  communication, worker termination by signal, termination without fuzzing,
  internal worker failure, bare `--- FAIL:` with no marker, build failure,
  unrecognized output → operational; written input → `fuzzFailureWrittenInput`
  with the reported path; seed-corpus entry → `fuzzFailureSeedCorpus`.
- Existing integration tests (`TestRunFuzzLiveCrash`, `TestRunFuzzSeedCrash`,
  `TestRunFuzzBuildFailure`, `TestRunFuzzPass`, `TestPreserveCrashInputRejectsEscape`)
  were left unchanged and still pass.

## Classification rule as built (Reading A)

A non-zero `go test -fuzz` exit is target-level only when go has attributed the
failure to concrete corpus data:

1. `Failing input written to <path>` → `guard.fuzz.crash`, path = the preserved
   input under `<pkg>/testdata/fuzz/<Name>/<hash>` (existing behaviour, kept).
2. `failure while testing seed corpus entry` → `guard.fuzz.crash`, path = the
   declaring `_test.go` at the `Fuzz*` line (existing behaviour, kept).
3. Everything else — including a bare `--- FAIL: <Name>`, all
   `internal/fuzz` worker start/communication/termination diagnostics,
   `[build failed]`, and any unrecognized output — is an operational error
   (guard exit 2), never a `guard.fuzz.crash` finding.

The written-input marker is evaluated first, the seed marker second. This matches
the specifier's recommended Reading A verbatim.

### PM steer reconciliation (recorded, as required)

The PM steer asked for the narrower rule: keep the `--- FAIL: <Name>` branch and
only demote it when a worker-start/management diagnostic
(`fork/exec`, `too many open files`, `failed to start fuzzing process`,
`cannot start fuzzing process`, fuzzing-process-failed-to-start family) is
present. The specifier artifact argues otherwise (Reading A, §"The rejected
alternative"): a denylist of `internal/fuzz` strings is not a stable contract
and would silently misclassify any future worker message. Per the task's
tiebreaker ("follow the specifier and say so"), the specifier wins: the bare
`--- FAIL: <Name>` clause is removed entirely rather than denylisted. The only
observable difference is a bare `--- FAIL: <Name>` with no recognized worker
diagnostic: PM steer → crash; Reading A → operational error (fail closed).
Reading A is fully covered by the deterministic classifier table.

## Before/after evidence for the `ulimit` reproduction

Before (this workspace, `9c43594`, `cmd/guard/fuzz.go` unmodified):

```text
$ bash -c 'ulimit -n 64; go test ./cmd/guard -run TestRunFuzzPass -count=1 -v'
=== RUN   TestRunFuzzPass
    fuzz_test.go:144: findings=[{guard.fuzz.crash p/fuzz_p_test.go 5 fuzz target FuzzSafe failed}] stats=map[targets:1], want a clean bounded run
--- FAIL: TestRunFuzzPass (0.65s)
FAIL	fake-jev/cmd/guard	1.215s
```

The spurious `guard.fuzz.crash` is present (the fabrication FJ-060 targets).

After (with the change; reproduced 1/1 in this session):

```text
$ bash -c 'ulimit -n 64; go test ./cmd/guard -run TestRunFuzzPass -count=1 -v'
=== RUN   TestRunFuzzPass
    fuzz_test.go:141: fuzz: example.test/m/p: unexpected failure: ... --- FAIL: FuzzSafe (0.01s)     pipe: too many open files FAIL exit status 1 ...
--- FAIL: TestRunFuzzPass (0.64s)
FAIL	cmd/guard	0.843s
```

The fabricated crash finding is gone; the fd-starved run now surfaces as an
operational error, which is the specifier's stated expected-after. `runFuzz`
under this limit is genuinely unable to host a fuzz run (the environment, not
the target, failed), so `TestRunFuzzPass` — which asserts a clean bounded run —
still exits non-zero. This is specifier observation 8 verbatim ("TestRunFuzzPass
... can still report a test failure — now as an operational error, with no
fabricated crash finding").

### Conflict with the task packet's "(must pass now)" — unresolved

The task run-list requires the `ulimit` command above to exit 0 after the
change. That is unsatisfiable within `allowedFiles` without contradicting the
specifier:

- (a) leave `TestRunFuzzPass` strict → command still exits non-zero (chosen);
- (b) make the test accept an operational error / skip → command exits 0, but
  specifier of:8 calls that "suppression of a failure (forbidden)" and it
  weakens the integration assertion (an unconstrained-environment tool error
  would be silently accepted);
- (c) raise `RLIMIT_NOFILE` from the test → needs `syscall.Setrlimit`, which is
  unix-only, so it needs either a new build-tagged helper file (outside
  `allowedFiles`) or a `//go:build unix` tag on the whole test file (drops all
  guard fuzz tests on Windows). Infeasible in scope.

I escalated this twice via the coordination channel (`need_decision`); both
attempts timed out, so I took the task's explicit tiebreaker default and
followed the specifier (option (a)). If the intended reading is (b), it is a
one-line change to `TestRunFuzzPass`; I did not make it.

## Exact commands and outcomes

| command | outcome |
|---|---|
| `bash -c 'ulimit -n 64; go test ./cmd/guard -run TestRunFuzzPass -count=1 -v'` (before) | FAIL, `guard.fuzz.crash p/fuzz_p_test.go 5 fuzz target FuzzSafe failed` (reproduced) |
| `gofmt -w cmd/guard/fuzz.go cmd/guard/fuzz_test.go`; `gofmt -l ...` | clean, no output |
| `go test ./cmd/guard -count=1` | ok — 93 passed, 1 package |
| `go test ./... -count=1` | ok — 548 passed, 9 packages |
| `go test -race ./... -count=1` | ok — 548 passed, 9 packages |
| `go vet ./...` | exit 0, no output |
| `go run ./cmd/guard fuzz` | `{"check":"fuzz","findings":[],"stats":{"targets":0}}`, exit 0 |
| `bash -c 'ulimit -n 64; go test ./cmd/guard -run TestRunFuzzPass -count=1 -v'` (after) | FAIL — operational error `unexpected failure: ... --- FAIL: FuzzSafe ... pipe: too many open files`, no crash finding (specifier obs 8; see conflict above) |
| `go build ./cmd/fake-jev` | exit 0 |
| `./scripts/candidate-fingerprint candidate` | `5fb3c895eceb9e3a37c42b94a3cf6aabf0ebab41e6ab3a2720dea0d0fba8943e` |

Targeted run of the fuzz suite (`go test ./cmd/guard -run 'TestRunFuzz|TestClassifyFuzzFailure|TestPreserveCrashInput' -count=1 -v`)
passes: `TestRunFuzzRejectsZeroFuzztime`, `TestRunFuzzZeroTargets`,
`TestRunFuzzSeedCrash`, `TestRunFuzzLiveCrash`, `TestRunFuzzPass`,
`TestClassifyFuzzFailureFDPressureIsNotCrash`, `TestClassifyFuzzFailure` (12
subtests), `TestPreserveCrashInputRejectsEscape`, `TestRunFuzzBuildFailure`.

## Observations

1. The classifier is now a pure function over retained output — the new tests
   feed it verbatim strings and never shell out; the fd-pressure regression is
   deterministic and hermetic (no fd exhaustion, no sleeps, no network).
2. Genuine crashes/timeouts are untouched: the live crash still yields
   `guard.fuzz.crash` with the preserved `testdata/fuzz/...` path
   (`TestRunFuzzLiveCrash`), and a hanging/seed failure still carries Go's
   retained marker. No detection was weakened.
3. `fuzzOutput.truncated` remains unconsulted (specifier obs 5, out of scope);
   classification still fails closed because a missing marker becomes an
   operational error.
4. No new dependency, no new failure code, no public-behaviour change;
   `fuzztime` default, discovery, the positive-window floor, and the zero-target
   vacuous pass are unchanged.

## Residual risks

1. **Task/specifier conflict on the `ulimit` command's exit code is unresolved.**
   The command still exits non-zero (as an operational error). This is the
   specifier's documented expected-after, but it contradicts the task packet's
   "(must pass now)". If QA/PM wants the command green, option (b) is a one-line
   `TestRunFuzzPass` change; I did not apply it because the specifier forbids it.
2. Reading A removes any crash detection for a bare `--- FAIL: <Name>` with no
   corpus-attribution marker. The specifier verified go1.26.3 always emits one
   of the two retained markers for genuine target failures, so no genuine
   signal is lost under the active toolchain; a future toolchain that attributes
   failures only via `--- FAIL:` would be reported as an operational error
   (fail closed, never a silent pass).
3. `state.json` and `.agent/reports/` were not modified (per instructions); the
   coder-stage gate and `./scripts/verify-candidate FJ-060` were not run from
   this stage.
