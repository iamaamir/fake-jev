---
stage: hardener
task: FJ-060
inputFingerprint: 5fb3c895eceb9e3a37c42b94a3cf6aabf0ebab41e6ab3a2720dea0d0fba8943e
outputFingerprint: 325768fd9665e78ee4264d86eb09e1dc651a4918c6ad1e140276a5b89208de31
taskFingerprint: 41ba2cd97f56c930d6ac57fdd4e662aebc1be3c12067c6cc2bd7dac7194f93a1
gitHead: 9c43594
generatedAt: 2026-10-02T19:04:08Z
author: worker/FJ-060-hardener
---

# Hardener — FJ-060 (fuzz worker failures are not target crashes)

**Final candidate fingerprint: `325768fd9665e78ee4264d86eb09e1dc651a4918c6ad1e140276a5b89208de31`**

`inputFingerprint` is the cleaner's unchanged candidate
(`5fb3c895eceb9e3a37c42b94a3cf6aabf0ebab41e6ab3a2720dea0d0fba8943e`). This
stage edited `cmd/guard/fuzz.go` and `cmd/guard/fuzz_test.go`, so the workspace
fingerprint moved and is carried as `outputFingerprint`, per the frozen
semantics in design §6.1/§7 (`B.outputFingerprint` is the workspace digest
*after* the stage; a no-op hardener would have kept the cleaner's value).
The task packet's "use exactly that value for both" line would place a
pre-change digest in `outputFingerprint` while the workspace already differs,
which the §7 terminal-freshness check reads as `stage.evidence_stale`; the
template line ("after your changes") and the frozen design are followed instead.
Inputs read: `./scripts/agent-context FJ-060`, `docs/agents/roles/hardener.md`,
`docs/agents/expertise/golang.md`, the specifier/coder/cleaner artifacts,
spec §23, design §3.3, `cmd/guard/fuzz.go`, `cmd/guard/fuzz_test.go`,
`cmd/guard/pkgscan.go`, `cmd/guard/main.go`, `cmd/guard/guards.go`, the guard
rows in `scripts/verify-candidate`, and the installed toolchain
(`go1.26.3`, GOROOT `/usr/local/go`).

## Hardening actions and the clauses they serve

### H1 — line-anchor the seed-corpus marker (real defect, fixed)

`cmd/guard/fuzz.go`: replaced the bare substring
`const seedFailureMarker = "failure while testing seed corpus entry"` +
`strings.Contains` with

```go
var seedFailurePattern = regexp.MustCompile(
	`(?m)^[ \t]*failure while testing seed corpus entry: `)
```

and switched `classifyFuzzFailure` to `seedFailurePattern.MatchString(text)`.

Defect: the seed marker was matched anywhere in the merged subprocess output.
Go's compiler echoes the offending source line in a build diagnostic, so a
package whose build fails and whose source contains that string was classified
as a seed-corpus failure and reported as a fabricated `guard.fuzz.crash`
finding — a package that never ran an input. Reproduced on the pre-fix binary
in a scratch module:

```text
$ go build -o /tmp/fj060-guard ./cmd/guard
# scratch module, p/fuzz_p_test.go line 6:
#   "failure while testing seed corpus entry: FuzzBroken/seed#0"
# go test -fuzz error: ./fuzz_p_test.go:6:2: "…" (untyped string constant) is not used
$ /tmp/fj060-guard fuzz
{"check":"fuzz","findings":[{"code":"guard.fuzz.crash","path":"p/fuzz_p_test.go","line":5,
 "message":"fuzz target FuzzBroken failed"}],"stats":{"targets":1}}
rc=1
```

After the fix the same scratch run is an exit-2 operational error (no finding).
Clauses served: design §3.3 ("a crash fails with `guard.fuzz.crash`" — only for
a real target failure); spec §23 baseline (a failing build must not be
attributable as a target crash); the specifier's own AC-10 ("an uncompilable
package is still an operational error naming the import path"), which the
bare-substring marker violated whenever the diagnostic echoed the marker. The
genuine seed line is still matched: go1.26.3 writes
`failure while testing seed corpus entry: <Target>/<entry>` at the start of a
line (verified with the raw subprocess, no indentation), and the genuine
written-input line is unaffected. The written-input marker was left as-is
because `preserveCrashInput` already rejects an echoed string (the capture
carries the diagnostic's trailing text, so the path does not exist / escapes
the package) and tightening it would risk losing genuine live-crash detection.

### H2 — end-to-end regression at the production dispatch site

`cmd/guard/fuzz_test.go`: added `TestRunFuzzWorkerStartFailureIsNotCrash`. A
scratch module's `TestMain` emits the verbatim fd-pressure shape
(`--- FAIL: FuzzSafe (0.01s)` + `fork/exec …: too many open files`, no marker)
and exits non-zero, so the whole `runFuzz → runOneFuzz → classifyFuzzFailure`
path is exercised deterministically without exhausting file descriptors in the
test process. This closes the cleaner's recorded gap ("the unit tests pin the
classifier, not the call site"): re-injecting the old
`strings.Contains(text, "--- FAIL: "+tgt.Name)` clause at the `runOneFuzz`
dispatch made this test fail (verified in a throwaway edit),
`fuzz_test.go:381: worker-start failure must be an operational error, not a
clean run`. Serves the work-item objective and design §3.3.

### H3 — end-to-end regression for the H1 defect

`cmd/guard/fuzz_test.go`: added
`TestRunFuzzBuildFailureEchoingSeedMarkerIsNotCrash` (a scratch package whose
unused string literal is the seed marker, failing to compile) and the table row
`build failure echoing the seed marker` in `TestClassifyFuzzFailure`. Reverting
H1 (bare `strings.Contains`) made all three bite (verified: 3 failures,
`TestClassifyFuzzFailure = (1, ""), want (2, "")` and
`fuzz_test.go:354: build failure must be an operational error, not a crash
finding`). Serves AC-10.

### H4 — writer bound / truncation fail-closed pinned

`cmd/guard/fuzz_test.go`: added `TestFuzzOutputBoundsRetainedBytes`. It writes
2× the bound through the shared writer and asserts (i) every `Write` returns
`len(p), nil` — a short write would stall the child's pipe and deadlock
`cmd.Run`; (ii) retained bytes equal `maxFuzzOutput` (1 MiB) so target output
cannot grow guard memory; (iii) `truncated` is set; (iv) the truncated output
still classifies as `fuzzFailureOperational`. Serves the resource-handling
requirement and the specifier observation 5 (truncated output fails closed).

## False-pass audit result

No path was found in which a genuine crash, seed failure, race, or timeout
becomes a guard exit-0 / empty-findings result. Route by route:

- Non-zero `go test -fuzz` exit, no corpus marker → `fuzzFailureOperational`
  → `runOneFuzz` returns an error → `runFuzz` returns `nil, nil, err` →
  `run()` writes to stderr and returns 2 → `scripts/verify-candidate`
  `guard_row` maps `rc not in (0,1)` to `fail guard.tool.failed`. The fuzz row
  is recorded `required=1` unconditionally (`scripts/verify-candidate:1627`),
  so this is never a skipped or ignored row.
- Non-zero exit, written input → `preserveCrashInput` on the reported path;
  a failure inside it is an error (exit 2), never a silent pass; success emits
  `guard.fuzz.crash` (exit 1).
- Non-zero exit, genuine seed line → `guard.fuzz.crash` at the declaring source.
- `cmd.Start` failure (not an `*exec.ExitError`) → wrapped error → exit 2.
- Clean exit (`err == nil`) → `nil, nil` → empty findings, exit 0.

Consequences: the change can move a classification between `guard.fuzz.crash`
(exit 1) and `guard.tool.failed` (exit 2) — both non-zero, both failing rows —
but cannot turn any failing run into exit 0. Race is not in this code path:
`runFuzz` never runs `go test -race`; the design §3.3 race row is a separate
`verify-candidate` check with its own command. The H1 tightening moves
*more* output into the operational branch; it removes a fabricated finding and
does not weaken any genuine positive.

## Marker-matching and resource audit

- **Case / whitespace.** Both markers are go's verbatim toolchain strings
  (`testing/fuzz.go:374`, `internal/fuzz/fuzz.go:248` in GOROOT); matching is
  case-sensitive by design. The seed marker now requires line start with
  optional leading blanks, which also tolerates an indented framing.
- **Interleaving of stdout / stderr.** `cmd.Stdout` and `cmd.Stderr` hold the
  same `*fuzzOutput` pointer, so `os/exec`'s interface comparison treats them as
  one writer: one pipe and one copy goroutine, hence no concurrent writes to the
  `bytes.Buffer` and no mid-line interleaving between the two streams. The
  `-race` run is clean. The "comparable writers" constraint stays documented at
  the call site; a future change to a non-comparable writer value would split
  the streams into two goroutines, which the race detector would surface.
- **Very large output.** Retained bytes are bounded at `maxFuzzOutput` (1 MiB);
  `Write` always consumes the full chunk; the diagnostic tail is trimmed to 400
  runes. `H4` pins the bound.
- **Process handling.** `cmd.Run()` waits for the subprocess, so the guard does
  not return while `go test -fuzz` is running. The fd-pressure reproduction was
  run and `ps` showed no residual `fz.test` / `go test -fuzz` / guard process.
  No `CommandContext`/`WaitDelay` was added: none of the audited runs hung, and
  a subprocess wall-clock bound is not part of this item's objective (recorded
  as a residual risk, not fixed).
- **`fuzzOutput.truncated`.** It is written and never read; classification is
  driven only by the retained prefix. That is the specifier's observation 5 and
  it fails closed: a truncated run with no marker is an operational error, never
  a silent pass. Left unconsulted deliberately (consulting it to force an
  operational disposition would suppress a genuine crash whose marker is in the
  retained prefix).

## Command outcomes

All commands run in `/Users/mak/git/fake-jev-FJ-060` at the candidate revision
(`go1.26.3 darwin/arm64`); rows are observed output, not verdicts.

| command | observed outcome |
|---|---|
| `gofmt -w cmd/guard/fuzz.go cmd/guard/fuzz_test.go`; `gofmt -l cmd/guard/` | no output |
| `go test ./cmd/guard -count=1` | `ok`, 97 passed, 1 package |
| `go test ./... -count=1` | `ok`, 552 passed, 9 packages, rc=0 |
| `go test -race ./... -count=1` | `ok`, 552 passed, 9 packages, rc=0 |
| `go vet ./...` | no output, rc=0 |
| `go run ./cmd/guard arch` | `{"check":"arch","findings":[],"stats":{"files":59,"packages":9}}`, rc=0 |
| `go run ./cmd/guard lint` | `{"check":"lint","findings":[],"stats":{"files":59,"packages":9}}`, rc=0 |
| `go run ./cmd/guard trace` | `{"check":"trace","findings":[],"stats":{"active":0,"covered":17,"test_files":22}}`, rc=0 |
| `go run ./cmd/guard fuzz` | `{"check":"fuzz","findings":[],"stats":{"targets":0}}`, rc=0 |
| `bash -c 'ulimit -n 64; go test ./cmd/guard -run TestRunFuzzPass -count=1 -v'` | rc=1; `fuzz: example.test/m/p: unexpected failure: … --- FAIL: FuzzSafe (0.01s)     pipe: too many open files …`, ready, no `guard.fuzz.crash` finding |
| revert probes (throwaway edits, then restored) | re-injecting the `--- FAIL:` clause fails `TestRunFuzzWorkerStartFailureIsNotCrash`; reverting the seed anchor fails `TestRunFuzzBuildFailureEchoingSeedMarkerIsNotCrash` and the echoing table row (3 failures) |
| `./scripts/candidate-fingerprint candidate` | `325768fd9665e78ee4264d86eb09e1dc651a4918c6ad1e140276a5b89208de31` |

`git diff --cached --name-only` is empty; nothing is staged. `.agent/work/FJ-060/state.json`
and `.agent/reports/` were not modified by this stage.

## Residual risks

1. **The two markers are an interpretation of the toolchain's output.** They are
   matched from go1.26.3 verbatim strings. A toolchain change that rewrites,
   prefixes, or re-frames either line (`Failing input written to <path>` or the
   seed line) would send a genuine crash/seed failure to the operational branch
   — exit 2, fail-closed, never a silent pass, but with the crash attribution
   lost. The marker constants and their comments are the place to update; the
   classifier table is where the new shape is pinned.
2. **The `ulimit -n 64` reproduction still exits non-zero** (now as an
   operational error with no fabricated crash). `TestRunFuzzPass` asserts a
   clean bounded run, so an environment that genuinely cannot host a fuzz run
   still reports a test failure. This is specifier observation 8; making that
   command exit 0 would require either accepting a tool error in the test
   (suppression of a failure) or raising `RLIMIT_NOFILE` (unix-only helper
   outside `allowedFiles`).
3. **No subprocess wall-clock bound.** `runOneFuzz` waits on `cmd.Run()` with no
   `CommandContext`/`WaitDelay`. The audited runs did not hang and no leaked
   process was observed, but a fuzz worker grandchild that outlived `go test`
   while holding the shared pipe could delay the wait. Not fixed: no observed
   hang and outside this item's objective.
4. **`fuzzOutput.truncated` remains unconsulted** (specifier observation 5). A
   crash whose marker falls beyond the 1 MiB bound surfaces as an operational
   error rather than a crash finding. Deliberately left as-is so a truncated
   prefix containing a genuine marker is not discarded.
5. **Seed line anchoring depends on line-start framing.** The anchor tolerates
   leading blanks but not a non-blank prefix; go1.26.3 writes the line at line
   start on stdin/stdout of the subprocess (verified). A future toolchain that
   prefixes the line would classify it as operational (fail-closed).
6. **Fingerprint instruction conflict.** Recorded above: the packet's
   "use exactly that value for both" line conflicts with the post-change value
   the article header requires; the frozen design §7 semantics were followed.
