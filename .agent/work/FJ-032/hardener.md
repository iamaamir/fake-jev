---
stage: hardener
task: FJ-032
inputFingerprint: ae99fe0fc8cbe134424e38857e8d693afeea99c9cbc2039f71605b76dc99a813
outputFingerprint: e520fc826889999e1e361d655d9a5f9fd80b6abf0e368376ab8390805481ca4b
taskFingerprint: 387ddaf6229fbab020186f256ec6fac2558180842ed9dbb9cac9f97728080a5d
gitHead: 22c0704
generatedAt: 2026-09-30T17:32:57Z
author: worker/FJ-032-hardener
---

# Hardener — FJ-032

## Final candidate fingerprint

```text
e520fc826889999e1e361d655d9a5f9fd80b6abf0e368376ab8390805481ca4b
```

`./scripts/candidate-fingerprint candidate`, computed after every edit
(`./fake-jev` is the gitignored binary `go build ./cmd/fake-jev` drops in the
repository root; it was removed after the build check and does not enter the
fingerprint, which ignores it via `.gitignore:/fake-jev`).

## Hardening actions, per clause

### F1 — `run` binds loopback regardless of the file's `server.host` (§15.4 step 2, §19.1, §30.8)

`resolveRunConfig` already replaced the configured `server.port` with `0` for
the §15.4 ephemeral rule but kept the file's `server.host`. A configuration with
`server.host: 0.0.0.0` therefore bound the wildcard address and injected `http://0.0.0.0:<port>` into the child, contradicting §15.4 step 2 ("start an
in-process fake server on an available loopback port") and the loopback host the
specifier's R2 asserts (§19.1, §30.8). The file's `server.host` is now replaced
with `config.DefaultHost` (`127.0.0.1`) in the same place, so the value drives
both the listen authority and the injected URL.

Ruling recorded as PM-directed: this is a conformance fix for `run` only.
`serve` intentionally still honours an explicit host (or `--host`), because
§15.1 documents a loopback **default**, not a forced value; `internal/cli/serve.go`
and the `--host` path are untouched, and no `serve` behavior changes.

Regression test `TestRunBindsLoopbackHostDespiteConfiguredWildcardHost` writes
`server.host: 0.0.0.0` and asserts (a) the child's `FAKE_JEV_URL` is loopback
with an OS port (`assertLoopbackURL`) and (b) the listener is not bound to the
wildcard address: the child, in helper mode `wildcard`, probes the host's
non-loopback local IPv4 address on the same port (`FAKE_JEV_HELPER_ALIAS`
supplied by `nonLoopbackLocalAddress`); a wildcard bind answers it, a
loopback-only bind refuses it immediately, so the test requires a refusal. The
alias-based check is skipped with `t.Log` when the host has no global-unicast
IPv4 address. With the host fix temporarily removed the test reports at the
loopback-host assertion, showing it fails against the defect.

### T2 — `run` reports both failures (§42.2, C-CLI-002, C-GOLD-013)

§42.2 requires that when the child exits nonzero **and** verification fails,
`run` report both failures to stderr while preserving the child's code, and the
C-GOLD-013 vector freezes `stderrContains: ["verification"]` for the
`child=7 + verify-fail` row. `runResult` now writes a
`fake-jev: child exited with code <n>` line whenever a non-interrupted run's
child exits nonzero, and `reportVerificationFailure` continues to write the
`…: verification failed` report; both lines reach stderr. The verification
report wording and the §42.2 exit precedence are unchanged.

`TestRunExitPrecedence` is tightened for every row: it asserts the exit code,
that stderr contains the frozen token `verification` exactly when verification
failed, and that the child's nonzero exit code is named exactly when the child
exited nonzero. The row set and the §44.13 startup-failure row
(`TestRunStartupFailuresExitTwo`) are otherwise pinned unchanged.

### T1 — the `--port` leg proves the flag, not the file (§15.4, C-CLI-004)

`TestRunUsesEphemeralPortDespiteConfiguredPort` previously passed the held port
both through `server.port` and `--port`, so the exit-2 bind failure could not be
attributed to the flag. The `--port` leg now uses a second configuration whose
`server.port` is a different OS-assigned free port (`freePort`); the held port
is supplied only via `--port`. A run that honoured the file value would bind the
free port and exit 0, so only an implementation that lets the flag win produces
the observed exit 2. The `file port is ignored` leg keeps its held-listener
configuration and its ephemeral-port assertion.

### F2 — signals are released once the child is reaped (§42.3 scope; §15 operation)

Signal notification was installed for the whole of `runCommand` and stopped only
in a defer, so a SIGINT/SIGTERM arriving during the post-child verification
(bounded by the 10 s `verificationTimeout`) or the graceful shutdown was
buffered and ignored. `runCommand` now calls `signal.Stop(signals)` immediately
after `awaitChild` returns, restoring the default disposition for the
verification and shutdown window while forwarding remains in force for the
child's whole lifetime. The deferred `signal.Stop` remains for the early-return
paths (startup/child-start failure); calling `Stop` twice is safe.

### F3 — dead `serverHandle.listener` removed (§4.6-adjacent hygiene; no behavior)

The `listener` field was written by `startServer` and read nowhere in product or
test code. It is removed; the local `listener` value is still used for
`net.Listen` and `http.Server.Serve`, and shutdown still reaches the socket
through `http.Server` (which owns the listener `Serve` was given). No observable
behavior changes.

### F4 — a mid-run `Serve` failure is surfaced (§42.5; §42.2 precedence preserved)

`runResult` never read `handle.serveErr`, so a `Serve` failure during the child's
run left the process with only a failed child request or a failed verification.
`serverHandle.serveFailure` performs a non-blocking receive on `serveErr`,
ignores the normal `http.ErrServerClosed`, and `runCommand` prints
`fake-jev: serve: <err>` to stderr when a real error has been recorded. No return
path or exit rule changes, so the §42.2 precedence for the normal paths is
unaffected.

## Audit of the allowed files

- **Child-process lifecycle.** Every path was re-read: config/generate/parse
  failures return before a child exists; `child.Start()` failure leaves the
  deferred `handle.stop()` to close the listener and returns 2; `awaitChild`
  reads `Wait` exactly once on every returning arm (normal exit, exit after
  forwarding, exit after a single `Kill`); the child is reaped before
  `runResult` runs; `handle.stop()` (Shutdown bounded by
  `limits.gracefulShutdownSeconds`, then Close) is deferred on all paths. No
  leaked, zombie, or double-killed child was found.
- **Bounded verify timeout.** `verificationTimeout` (10 s) bounds the client on
  both `verify` and `run`'s post-child call; a non-answer maps to exit 2.
- **stdout/stderr discipline.** `run` writes every diagnostic (start failure,
  serve failure, child exit, verification failure) to stderr and no result line
  of its own to stdout; the child inherits the caller's streams. `verify` keeps
  its success line on stdout and all failures/usage on stderr.
- **Authorization redaction.** The only log path is `logHandler`, which records
  method, path, and `[redacted]`; no other code reads the header. Unchanged.
- **Graceful shutdown timeout.** Both the child grace (`awaitChild`) and the
  server drain (`serverHandle.stop`) read
  `cfg.Limits.GracefulShutdownSeconds`, the configured §12.2/§38.3 value.
- **Goroutine/temp-file leaks.** The `Serve` writer is a capacity-1 buffered
  channel so the goroutine cannot block when nobody reads; `awaitChild`'s waiter
  is always consumed; `run` writes no temporary files. No leak found.

No other concrete defect inside the allowed files was identified, so no other
change was made.

## Commands run and outcomes

```text
gofmt -w internal/cli/{run,serve,run_test}.go        -> clean (gofmt -l empty)
go test ./internal/cli -count=1                      -> ok  18.765s
go test ./... -count=1                               -> ok  (cmd/guard, internal/cli,
                                                        compat, config, control, engine,
                                                        host/http, test/integration)
go test -race ./... -count=1                         -> ok  (same packages)
go vet ./...                                         -> no output, exit 0
go run ./cmd/guard arch                              -> exit 0, findings []
go run ./cmd/guard lint                              -> exit 0, findings []
go run ./cmd/guard trace                             -> exit 0, findings [] (active 0, covered 17)
go run ./cmd/guard fuzz                              -> exit 0, findings []
go build ./cmd/fake-jev                              -> ok (artifact ./fake-jev removed; gitignored)
./scripts/candidate-fingerprint candidate            -> e520fc826889999e1e361d655d9a5f9fd80b6abf0e368376ab8390805481ca4b
```

Behavior probes not covered by the committed suite were also run from a
temporary directory outside the repository for the wildcard-host fix and the
F1 regression, and the regression was confirmed to fail when the host fix is
reverted.

## Residual risks

- **No process-group forwarding.** §42.3 step 1 asks to forward to the child
  "process/process group where supported". Forwarding targets the child process
  only; `syscall.SysProcAttr{Setpgid: true}` does not compile on Windows and the
  allowed-file set has no place for a build-tagged file. A child that spawns
  grandchildren can still leave them behind when `run` is interrupted
  programmatically; an interactive Ctrl-C is unaffected because the shell
  delivers SIGINT to the whole foreground group. Unchanged by this stage.
- **Readiness is structural.** `run` has no `--ready-file` (R-U1); it publishes
  readiness by binding the listener and attaching the control API before
  `startServer` returns. A `Serve` failure after that point is observable only
  through a failed child/verify request (now also surfaced by F4, best-effort),
  not before the child starts.
- **The C-GOLD-013 `kind: cli` cases are not executed by a runner yet.** No test
  reads `testdata/contracts/cli/01-run-exit-precedence.json`; the vector's rows
  are pinned behaviorally by `TestRunExitPrecedence` (exit codes plus the frozen
  `verification` token and the child-exit line), and the remaining `kind: cli`
  cases await the runner that FJ-046 owns. The frozen `stderrContains` token is
  now asserted directly, so a rewording that dropped "verification" while
  keeping fault codes would be caught.
- **Verify response body has no byte cap.** `callVerification` reads the body
  with `io.ReadAll`; the 10 s client timeout bounds wall time, not bytes.
  §41.9 defines no response size limit and the endpoint is the in-process
  loopback control plane whose journal is itself bounded, so no cap was added;
  recorded rather than changed.
- **Signal handling assumes Unix semantics.** The forwarding/force-termination
  tests rely on Unix signal and `kill -0` behavior; Windows is only
  cross-compiled, as in the existing `serve` tests.

`state.json` and `.agent/reports/**` were not edited by this stage.
