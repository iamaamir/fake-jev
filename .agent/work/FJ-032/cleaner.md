---
stage: cleaner
task: FJ-032
inputFingerprint: ae99fe0fc8cbe134424e38857e8d693afeea99c9cbc2039f71605b76dc99a813
outputFingerprint: ae99fe0fc8cbe134424e38857e8d693afeea99c9cbc2039f71605b76dc99a813
taskFingerprint: 387ddaf6229fbab020186f256ec6fac2558180842ed9dbb9cac9f97728080a5d
gitHead: 22c0704
generatedAt: 2026-09-30T17:21:26Z
author: worker/FJ-032-cleaner
---

# Cleaner — FJ-032

Structure reviewed: `internal/cli/dispatch.go`, `internal/cli/serve.go` (the
extracted lifecycle seam), `internal/cli/verify.go`, `internal/cli/run.go`,
`internal/cli/verify_test.go`, `internal/cli/run_test.go` against §15.3, §15.4,
§42.1, §42.2, §42.3, §44.13, §41.9 and the acceptance catalog rows C-CLI-001,
C-CLI-002, C-CLI-003, C-CLI-004, C-CLI-007, C-CLI-008, C-GOLD-013.

**Behavior-preserving cleanup applied: none.** No file was edited. The candidate
fingerprint recomputed before and after this pass is identical to
`ae99fe0fc8cbe134424e38857e8d693afeea99c9cbc2039f71605b76dc99a813`, so nothing
upstream of this stage is staled. Every item below is recorded as a finding for
the hardener/QA rather than acted on: each of the four would change file content
and therefore the bound fingerprint, and two of them (F1, F4) are spec-silent
behavior decisions rather than cleanups. This artifact issues no gate verdict;
G-L's complexity/CRAP enforcement is still `stage.tooling_bootstrap_exempt`
(`.agent/gate-policy.json`, FJ-044).

## Serve/run lifecycle refactor (behavior preservation)

The `serve.go` diff moves "build host + attach control + listen + `go Serve`"
into `startServer`, "Shutdown → Close" into `(*serverHandle).stop`, the
ready-file failure close into `forceClose`, and keeps everything else in
`serve` (flag parsing, ready document, atomic write, pid-guarded removal,
signal select, 0/2 exits). Line-by-line, the observable serve surface is intact:

- flag set, §12.1 precedence, and `validateServeFlags` are untouched
  (`serve_test.go`, `validate_test.go` unmodified in the working tree);
- diagnostics keep their exact text: a bind failure is wrapped as
  `"listen on %s: %w"` in `startServer` (`serve.go:250`) and printed by
  `serve` with `"fake-jev: %v\n"`, reproducing the pre-refactor
  `fake-jev: listen on <authority>: <err>` byte-for-byte; the
  `"%v; close listener: %w"` combination reproduces the old
  `"…; close listener: …"` line; `logger.Printf("listening on %s", …)` and
  `"received %s; shutting down"` are unchanged in position and wording;
- ready-file content is built from `handle.URL()` / `handle.Port()` — the same
  `"http://" + net.JoinHostPort(host, boundPort)` and `bound.Port` values the
  old code used, so `readyDocument` is unchanged;
- exit codes are unchanged: `startServer` failure → 2, ready-file failure →
  `forceClose` then 2, `serveErr` → 2, signal → `stop()` then 0.

Ordering difference (measured, not latent): the `go handle.http.Serve(listener)`
goroutine is now started inside `startServer`, i.e. slightly *before*
`signal.Notify` rather than immediately after it. The window between
`net.Listen` and `signal.Notify` existed in the old code too (it spanned
`Addr()`/url construction), so no signal that was handled before is
unhandled now; the window only moved by a struct literal. Similarly, a bound
listener may begin accepting a few instructions earlier than before, which
§15.1 permits (the ready file is still published only after `startServer`
returns).

Empirical confirmation against the built binary: `serve --port <held>` → exit 2
with `fake-jev: listen on 127.0.0.1:<port>: listen tcp … bind: address already in
use` on stderr and empty stdout; `serve --port 0 --ready-file …` publishes the
exact five-key document, exits 0 on SIGTERM with
`listening on …` / `received terminated; shutting down` on stderr, and removes
the ready file. The existing serve tests are unmodified and all of them ran
(27 passed, 0 skipped, 0 failed in the `Serve|Ready|Signal|Usage|Shutdown`
selection; `go test ./internal/cli -count=1` → ok 18.785s).

## Duplication

- `verify.go` ↔ `run.go`: shared as intended — `callVerification`,
  `verificationReport`/`verificationFailure`, `verificationPath`,
  `verificationTimeout`, and `writeVerificationFailure` are defined once in
  `verify.go` and reused by `runResult`. No verification logic is duplicated in
  the CLI (C-CLI-008), and there is no second HTTP call path.
- `verify.go`/`run.go` ↔ `internal/control/verify.go`: the CLI re-declares the
  §41.9 wire shape (`verificationFailure`). This is not removable duplication:
  the control package's types are unexported, and the CLI's `verifyDocument`
  deliberately differs (a `*bool` `passed` to distinguish absent from `false`,
  §41.9). Exporting the control type would widen the non-goal surface and would
  not fit the decoder.
- `run.go` ↔ `serve.go`: the seam is genuinely shared (`startServer`,
  `URL`, `Port`, `forceClose`, `stop`). What remains duplicated is two small
  call-site blocks, not logic: `logger := log.New(stderr, "fake-jev: ", 0)`
  (serve and `runCommand`) and the `signals := make(chan os.Signal, 1)` +
  `signal.Notify(…, os.Interrupt, syscall.SIGTERM)` + `defer signal.Stop(…)`
  triple. The two commands' signal *behaviours* differ (serve selects and
  drains; run forwards and force-terminates), so a shared helper would have to
  be parameterised by the handler — indirection without a second concrete need
  (AGENTS.md). Recorded, not extracted.
- test harness: `run_test.go` re-uses `serveBinary`, `TestMain`, `syncBuffer`,
  `writeConfigFile`, `serveRequest`, `listRequests`, and `startServe` from
  `serve_test.go` (specifier H1 satisfied), but re-implements the child
  process wrapper (`runChild`, `startRun`, `stop`, `wait`, `signal`) as a near
  copy of `serveChild`. The two differ only in the ready-file assertions absent
  from `run`, so one shared wrapper with an optional ready-file field would
  collapse ~60 lines. Left in place: it is a test-internal refactor that would
  change the bound fingerprint, and the copied code carries its own
  `runWaitTimeout` semantics.

## stdout / stderr discipline

`verify` writes only the success line to stdout
(`fake-jev: <url>: verification passed`) and everything else to stderr: usage
diagnostics through `usageError`, operational failures through
`writef(stderr, "fake-jev: %v\n", err)`, and the §41.9 failure report through
`writeVerificationFailure(stderr, …)`. The §41.9 detail (code, message,
`requestSequence`, `stubId`) therefore cannot reach stdout on any path, which
`TestVerifyExitCodesFromControlResponses` also asserts
(`!strings.Contains(stdout, "verification failed")`). `run` writes its own
operational lines and the child's inherited streams as the coder recorded;
`run` prints no result line of its own.

## Child-process lifecycle

Paths walked, all with a single `Wait` and no second kill:

- child start failure → `writef(stderr, "fake-jev: start …")`, exit 2, deferred
  `handle.stop()` closes the listener, no child exists (`run.go:150-158`);
- normal exit → `awaitChild` returns from the `<-exited` arm, child reaped;
- signal → forward (`child.Process.Signal`, `ErrProcessDone` tolerated), then
  `<-exited` within `gracefulShutdownSeconds`, else exactly one `Kill` and a
  blocking `<-exited` (`run.go:195-214`) — the child is reaped on every arm that
  returns, which is what the `awaitChild` doc comment claims;
- verification and `handle.stop()` both run after the child is reaped; the
  deferred stop is registered before `signal.Stop`, so signals are restored
  before the drain and a post-child SIGTERM cannot kill run through the default
  disposition mid-shutdown.

Verified by observation: `run -- sh -c 'exit 7'` → 7; `run -- false` → 1;
`run -- sh -c 'kill -TERM $$'` → 143; a `trap '' TERM INT` child signalled with
SIGTERM was force-terminated at the configured 2 s, run exited 143, and `ps`
showed no surviving child and no zombie; `run -- true` → 0 (child exiting with
no interaction still verifies clean).

"Child exits before readiness" cannot arise in `run`: `startServer` binds the
listener and attaches the control API before returning, so the child is created
only after the socket is accepting from the backlog; the child's first request
in every traffic-producing case returned the server's answer on the first
attempt (no retry exists in the helper). §15.4's ready-file-less readiness is
therefore structural; the caveat in the coder's residual risks (a `Serve` error
after `startServer` returns is only observable through a failed child/verify
request) is accurate.

## Test quality

Against the specifier's H1–H7 rules: builds come from `TestMain`
(`./cmd/fake-jev`) or a `t.TempDir()`-resident stdlib-only helper built offline
(`GOPROXY=off`, `GOWORK=off`); every listener is `127.0.0.1:0` or
`httptest.NewServer`; no test picks a numeric port (the only fixed value is
`http://127.0.0.1:1` as a poisoned inherited `FAKE_JEV_URL`, never dialled on a
succeeding path); synchronization is the ready file, the helper's `started`
file, `run`'s own exit, or the client timeout; the two `time.Sleep` loops are
bounded 2 ms polls of a real file/`httptest` barrier, matching
`serve_test.go`'s `awaitReadyShape`; waits are bounded by
`runWaitTimeout` (15 s), `gracefulShutdownSeconds + 10 s`, or 10 s HTTP
timeouts; cleanup kills and reaps every spawned process, and
`t.TempDir()`/`t.Chdir`/`os.Pipe` are used correctly. After the full suite `ps`
showed no surviving `run`, helper, or `serve` process from it. No test leaks a
goroutine on a succeeding path (`awaitChild`'s waiter always completes; the
`serveErr` channel is buffered, so the `Serve` goroutine cannot block when
nobody reads it). The only `serve` process visible before this pass was an
orphan whose cwd was `…/fake-jev-FJ-031` and whose binary and ready-file
arguments belonged to an earlier worktree's ad-hoc probe; it predates this
suite, and no `serve`/`run`/helper process from the FJ-032 suite survived it.
The live probes below used `/tmp/fj`, never a repository path.

Cases that can succeed without proving their stated requirement:

- **T1 — `TestRunUsesEphemeralPortDespiteConfiguredPort` / "explicit port is
  honoured" (`run_test.go:556-568`).** The config file already names the held port
  (`server.port: %d` with `heldAddress.Port`), and the leg passes the *same*
  value via `--port`, so exit 2 is reachable whether the flag or the file
  produced the bind attempt. The leg cannot attribute the failure to `--port`
  as its claim ("proving the flag is the only way a fixed port reaches the
  server") states. A config whose `server.port` differs from the `--port` value
  would isolate the flag. The first leg does establish that a file value never
  reaches the listener, and is not confounded.
- **T2 — C-CLI-002's "reports both failures" half.** `TestRunExitPrecedence`
  ("child 7 verify fail") asserts only that stderr carries the verification
  failure (`unknown_route`). `run` prints nothing that names the child's
  nonzero exit — measured: `run -- sh -c 'curl "$FAKE_JEV_URL/nope"; exit 7'`
  produces the child's output, the request logs, and
  `…: verification failed` / `…: unknown_route: …`, and no child-exit line — so
  the assertion cannot distinguish "both reported" from "only verification
  reported". The committed golden vector
  `testdata/contracts/cli/01-run-exit-precedence.json` freezes only
  `stderrContains: ["verification"]` for that row, so the implementation
  satisfies the materialized vector while the criterion's prose remains
  half-pinned.
- **T3 — the golden vector's `"verification"` substring is not asserted.**
  The same row looks for `unknown_route`, not `verification`. The current
  wording contains both, but a future rewording that dropped the word
  "verification" while keeping fault codes would leave this test green and any
  FJ-046-style runner that reads `stderrContains` red. No test reads
  `testdata/contracts/cli/01-run-exit-precedence.json` (§44 requires the vectors
  to become executable tests; that runner is FJ-046's scope).
- **T4 — `TestRunStartupFailuresExitTwo` doc comment (`run_test.go:621-623`)
  over-claims for two legs.**
  "run no child, and leave no listener behind" is asserted for the invalid-config
  leg (snapshot absence) but the "unbindable port" and "missing configuration"
  legs pass no helper environment and probe no listener, so `run no child` is
  only asserted where the child could record itself. The bind-failure leg does
  probe the held listener (it still accepts), which is the meaningful half.
- **T5 — timeout → exit 2 is pinned only at the transport layer.**
  `TestCallVerificationTimeoutIsAnError` calls `callVerification` directly with
  a 200 ms client; `runVerify`'s error → 2 mapping is pinned by the
  unreachable-port case instead. No false-green follows, but the timeout row is
  a unit assertion rather than a command-level one.
- **T6 — §41.9 item shape from the live server is not asserted.**
  `requestSequence`/`stubId` decoding is pinned against a canned body
  (`TestCallVerificationDecodesServerReport`); the real-server legs assert only
  failure codes. Server-side shape ownership is `internal/control`'s tests
  (FJ-021), so this is a boundary note rather than a gap.
- Minor redundancy: `TestRunUsageFailures` creates and exports
  `helperStartedEnv` but asserts only the snapshot file (sufficient — the helper
  writes the snapshot first and exits 2 if the started write fails), and
  `helperSnapshot.URLEnvName` is decoded but never asserted.

## Comments and doc accuracy

- Accurate as written: `verificationTimeout`/`verificationPath` rationale,
  `verifyDocument`'s pointer explanation, `callVerification`'s claim that it
  transports and decodes only, `runVerify`'s "loads no configuration",
  `parseVerifyFlags`'s fail-closed rationale, `runFlags`'s "absent vs zero
  value" note, `validateRunFlags`'s "no rule is restated here",
  `resolveRunConfig`'s §12.1 note, `runCommand`'s defer/signal ordering notes,
  `awaitChild`'s "always reaped before it returns" (true on every returning
  arm), `childExitCode`/`signalTerminationStatus`/`signalExitStatus`, and the
  `serverHandle`/`stop`/`forceClose` docs.
- `run.go:241-243` (`reportVerificationFailure`) attributes §42.2's "report
  both failures" clause to a function that writes only the verification
  diagnostic; the sentence reads as if the child's failure were also written
  here. Sharpen to "the verification half of the two failures §42.2 requires".
- `serve.go:237-238` (`startServer`): "a caller never holds a handle for a
  server that is not listening" is true for the bound listener but not for a
  `Serve` that fails afterwards; the following `serverHandle` doc already
  narrows the claim to the backlog, so only this sentence is loose.
- `run_test.go:517-520` doc comment: "an implementation that honoured the file
  value could not bind and would fail instead of exiting 0" is stated for the
  test as a whole but is only true of the first leg (see T1).
- `awaitFile`'s "so no sleep is used as synchronization" is a polling wait, not
  a sleep-as-sync; the wording matches `serve_test.go`'s convention.

## Latent defects / later-ticket risks

- **F1 — `run` binds and injects the configured host, not a loopback host.**
  Measured: `config` with `server.host: 0.0.0.0` makes
  `run --config … -- sh -c 'echo $FAKE_JEV_URL'` inject
  `FAKE_JEV_URL=http://0.0.0.0:<port>` and bind the wildcard address (the child's
  request still returned 200 on macOS/Linux). `run` forces `server.port` to 0
  for §15.4's ephemeral rule but has no corresponding rule for the host, while
  §15.4 step 2 says "start an in-process fake server on an available loopback
  port" and specifier R2 states the injected host is loopback (§19.1, §30.8).
  §19.1 explicitly allows `0.0.0.0` when the user chooses it, so this is a
  spec-silent decision, not a clear violation — but it is exactly the boundary
  FJ-041 ("loopback default bind") will touch, and `assertLoopbackURL` only
  inspects default-host runs, so no test would notice. Decide: force loopback in
  `run` (mirroring the port rule) or declare `server.host` honored and document
  the non-loopback URL.
- **F2 — signals stay trapped after the child exits.** `signal.Notify` is
  installed for the whole of `runCommand` and stops only at deferred
  `signal.Stop`; once `awaitChild` has returned nothing reads `signals`, so a
  SIGINT/SIGTERM arriving during the post-child verify (bounded by the 10 s
  `verificationTimeout`) or during `handle.stop()` is buffered and ignored, and
  a second one is dropped too. An interactive Ctrl-C in that window therefore
  cannot interrupt `run` for up to `verificationTimeout + gracefulShutdownSeconds`.
  §42.3 covers signals before/during the child, so this window is spec-silent
  (adjacent to the coder's R-U5 note).
- **F3 — `serverHandle.listener` is written and never read** (`serve.go:263`;
  no reader in product or test code). It is the natural in-package seam for
  forcing the `serveErr` arm that the FJ-031 hardener recorded as "structural,
  not exercised", but nothing uses it today, so as it stands it is unused state.
  Keep only with that justification; otherwise remove.
- **F4 — `run` silently discards `handle.serveErr`** (nothing reads the channel
  in `runCommand`; capacity 1 keeps the goroutine from blocking). A `Serve`
  failure mid-run surfaces only as a failed child request or a failed
  verification mapped to exit 2, never as a diagnostic naming the server error,
  unlike `serve`'s `fake-jev: serve: <err>`. Spec-silent; recorded by the coder.
- **F5 — activation of `cmd/guard trace` would show the CLI criteria as
  unmapped.** `.agent/guards.json` has `trace.active: []` today; the scanner
  collects markers from `_test.go` **string literals** and `testdata/contracts`
  tags only, so the new tests' `C-CLI-00x` / `C-GOLD-013` mentions in comments
  are invisible to it, and only `C-GOLD-013` (via
  `testdata/contracts/cli/01-run-exit-precedence.json`) is tagged anywhere.
  Populating `active` for the CLI rows will require string-literal markers
  (the repo's observed form is `"C-GOLD-001/choice"` in `cmd/guard/trace_test.go`).
- **F6 — test-only: a failing test can orphan the helper.** `runChild.stop`
  sends SIGKILL to `run` without first forwarding a signal, so if a test fails
  before the barrier (e.g. `awaitFile` in `TestRunForwardsSignalToChild`),
  `run`'s graceful path never runs and a `signal`- or `hold`-mode helper stays
  blocked forever with nobody to reap it. It holds no port, so the residue is a
  sleeping orphan, and this mirrors `serveChild.stop`'s existing pattern;
  recorded for the QA stage's process-residue check.
- **Spec-silent behaviour frozen by the tests (not defects, listed so a later
  ticket knows what is pinned):** interrupt status `128+signal` even when the
  child exited 0 or nonzero (§42.3 step 5 vs §42.2's preserve rule; R-U3,
  `runResult`'s first case), `passed:true` with non-empty `failures` → 0
  (V-U5), every non-200 `/verify` → 2 (V-U4), `https` and uppercase schemes
  accepted (Go's `url.Parse` lowercases the scheme; measured
  `--url HTTP://…` → exit 0 against a live server), and `run` printing no line
  of its own for a nonzero child (V-U1).

## Commands run in this pass

```text
gofmt -l .                                             -> no output (exit 0)
go test ./internal/cli -count=1                        -> ok fake-jev/internal/cli 18.785s
go test ./... -count=1                                 -> 8 ok packages (incl. cmd/guard),
                                                          1 package without test files
go test -race ./... -count=1                           -> ok (all packages)
go vet ./...                                           -> no output (exit 0)
./scripts/candidate-fingerprint candidate              -> ae99fe0fc8cbe134424e38857e8d693afeea99c9cbc2039f71605b76dc99a813
go run ./cmd/guard arch|lint|trace|fuzz                -> exit 0 each, findings [] (trace stats: active 0, covered 17)
GOOS=windows GOARCH=amd64 go build -o /tmp/… ./cmd/fake-jev  -> ok (artifact outside the repo)
GOOS=linux   GOARCH=arm64 go build -o /tmp/… ./cmd/fake-jev  -> ok
```

Live probes against `/tmp/fj` (built from this revision, outside the repo):
serve bind-failure message/exit, serve ready-file publish+remove on SIGTERM,
`run -- true|false|sh -c 'exit 7'|sh -c 'kill -TERM $$'`, a SIGTERM-ignoring
child force-terminated at a 2 s grace with no surviving process, `run` with a
`server.host: 0.0.0.0` config, and `verify` against a live `serve` for the plain,
trailing-slash, and uppercase-scheme URL forms.

`./scripts/verify-candidate FJ-032` was not re-run: this pass wrote no product
content, the candidate fingerprint is unchanged, and the existing report in
`.agent/reports/FJ-032/` is already bound to that fingerprint. `state.json` and
`.agent/reports/**` were not touched. Nothing is staged; the only unstaged
content is the coder's change set and this artifact, which the fingerprint
excludes.

## Residual risks of this pass

- The lifecycle-preservation argument rests on reading the diff plus the
  unmodified serve tests and live probes; the `serveErr` arm is still
  unreachable from outside `serve` without the FJ-031 seam (F3), so its
  equivalence is argued structurally.
- Two `time.Sleep` polls (2 ms) remain in the new tests, consistent with the
  existing harness; a future "no sleeps in tests" guard would have to exempt
  the whole package or be applied to all of `internal/cli`.
- T1/T2/T3 are test-strength observations. Acting on them (or on F1–F4) would
  change the candidate fingerprint and should be routed as a new stage or work
  item rather than folded into this pass.
