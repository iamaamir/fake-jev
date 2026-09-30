---
stage: coder
task: FJ-032
inputFingerprint: c8ee98ed15771f58cb8cecedd751633ac6c7bc4414a294da0db14807fda43338
outputFingerprint: ae99fe0fc8cbe134424e38857e8d693afeea99c9cbc2039f71605b76dc99a813
taskFingerprint: 387ddaf6229fbab020186f256ec6fac2558180842ed9dbb9cac9f97728080a5d
gitHead: 22c0704
generatedAt: 2026-09-30T17:09:39Z
author: worker/FJ-032-coder
---

# Coder — FJ-032

## Final candidate fingerprint

```text
ae99fe0fc8cbe134424e38857e8d693afeea99c9cbc2039f71605b76dc99a813
```

(`./scripts/candidate-fingerprint candidate`, computed after every edit, with no
stray build artifact in the tree; the earlier `fake-jev.exe` left by an ad-hoc
`GOOS=windows go build ./cmd/fake-jev` cross-compile check was removed.)

## What changed, per deliverable

Changed files (5 of the 7 allowed):

```text
internal/cli/dispatch.go    routes verify and run, refreshes the usage text
internal/cli/serve.go       extracts the shared start/ready/stop lifecycle seam
internal/cli/verify.go      new: thin GET /__fake/v1/verify client + exit mapping
internal/cli/run.go         new: §15.4 orchestration and §42.2 exit precedence
internal/cli/verify_test.go new: verify tests
internal/cli/run_test.go    new: run lifecycle tests
```

`internal/host/http/server.go` was **not** modified: the lifecycle seam fits
entirely inside `internal/cli/serve.go` on top of the existing host surface
(`hosthttp.NewServerFromConfig`, `AttachControl`, `Engine()`), and `run` needs no
host capability that `serve` did not already use. No new host seam was invented,
so the allowed-but-unneeded file was left untouched.

### 1. `verify --url <url>` (§15.3, C-CLI-008, C-CLI-001)

- `runVerify` parses `--url` (REQUIRED), rejects any other flag, a stray
  positional, and a value that is not a plain `http`/`https` base URL (empty,
  unparseable, host-less, or carrying a path, query, or fragment) as a usage
  failure: exit 2, diagnostic plus usage text on stderr, nothing on stdout.
- `callVerification` is the only client: exactly one `GET <base>/__fake/v1/verify`
  with no body, no retry, no second call. It rejects a non-200 status
  regardless of body, a body that is not a JSON object, and a body without a
  boolean `passed` (decoded through `*bool` so `passed: false` is distinguishable
  from absent). Decoding is all the CLI does: it never matches stubs, evaluates
  expectations, orders failures, or loads configuration — the returned failure
  items are printed in the server's order.
- Exit mapping: `passed: true` → 0 (confirmation line on stdout);
  `passed: false` → 3 with every §41.9 item (code, message,
  `requestSequence`, `stubId`) written to stderr; anything the command cannot
  obtain or understand (dial failure, reset, non-200, undecodable body, bounded
  10 s timeout) → 2 with a diagnostic on stderr.
- `usage` lists `verify --url <url>` and the stale "run and verify land in later
  work items" note is gone.

### 2. `run` (§15.4, C-CLI-002/003/004/007, C-GOLD-013)

Execution order is exactly §15.4:

1. `parseRunFlags` splits at `--` (the child command is REQUIRED) and parses
   only `--config`, `--port`, `--url-env`; `validateRunFlags` rejects an empty
   `--url-env` name and out-of-range ports using the same §38 validator a file
   passes through.
2. `resolveRunConfig` reuses the serve §12.1 chain (`resolveServeConfig`) and
   then applies the §15.4 ephemeral rule: unless `--port` was explicitly passed,
   `server.port` is replaced with `0`, so a file value never reaches the
   listener. Configuration is validated before anything is started (an invalid
   config binds nothing and runs nothing).
3. `startServer` binds the loopback authority and attaches the control API
   before it returns, so the listener and the control API are ready before the
   child exists.
4. `childEnvironment` always sets `FAKE_JEV_URL` to the live base URL, dropping
   every inherited entry of that name, and additionally sets `--url-env NAME` to
   the same URL, dropping any inherited `NAME`.
5. The child inherits run's stdout/stderr (CI visibility, §28/§42.5) and reads
   the null device (run never prompts or consumes stdin, §15).
6. SIGINT/SIGTERM are subscribed **before** the child starts, so a signal sent
   the moment the child becomes observable is handled gracefully rather than
   killing run through the default disposition.
7. `awaitChild` waits for the child, forwarding the signal to the child process,
   allowing it `limits.gracefulShutdownSeconds`, then force terminating it.
8. `runResult` always attempts the control-plane verify operation (including for
   a nonzero child and an interrupted run) and prints the failure report on
   stderr.
9. `defer handle.stop()` runs the shared graceful shutdown on every path,
   including a failed child start and an interrupt.
10. Exit precedence is exactly §42.2/§44.13: child 0 + pass → 0; child 0 + fail
    → 3; child nonzero → the child's code (with the verification failure also
    reported when both occurred); startup/child-start failure → 2.

### 3. `serve`/`run` lifecycle seam (no forked logic)

`serve.go` now exposes one seam used by both commands:

```text
type serverHandle   cfg, logger, listener, http.Server, url, port, serveErr
startServer(cfg, logger) (*serverHandle, error)  build host + attach control + listen + go Serve
(*serverHandle) URL() / Port()                   live base URL / bound port
(*serverHandle) forceClose()                     abandon (ready-file failure path)
(*serverHandle) stop()                           Shutdown bounded by gracefulShutdownSeconds, then Close
```

`serve` keeps its own pieces (flag parsing, ready file, signal select, ready-file
removal defer, exit codes) and now calls the seam; `run` calls the same seam and
owns the child instead of the ready file. Observable `serve` behavior is
unchanged: same flags and precedence, same ready-document content and atomic
write/pid-guarded removal, same stderr request logging with Authorization
redaction, same 0/2 exits, and every FJ-031 test still passes unmodified.

## Tests added

`internal/cli/verify_test.go`

- usage failures (missing/empty/unparseable/path/query URL, unknown flag, stray
  positional) → exit 2, usage on stderr, empty stdout, **zero** requests reaching
  a recording server;
- exactly one `GET /__fake/v1/verify`, empty body, run from a directory holding
  a deliberately invalid `fake-jev.yaml` (no config discovery);
- response table: pass → 0, §41.9 failure shape → 3 with the code on stderr,
  `passed:false` with no items → 3, and 500/404/204/not-JSON/`[]`/`null`/missing
  `passed`/non-boolean `passed` → 2 with a diagnostic;
- unreachable loopback port (a listener the test just closed) → 2;
- real `serve` child: unknown route fails → 3 (twice, read-only),
  `DELETE /__fake/v1/requests` → 3→0 flip, and no verify call in the journal;
  unsatisfied `expect: exactly 1` → 3 with `expect_exactly` on stderr;
- transport seam: server failure order preserved, nullable fields decoded,
  malformed bodies rejected, bounded timeout is an error;
- base URL with and without a trailing slash both target exactly
  `/__fake/v1/verify`; `--help` prints usage to stdout and exits 0.

`internal/cli/run_test.go` (helper child = a small Go program written to
`t.TempDir()`, compiled offline with `GOPROXY=off GOWORK=off`, whose lifetime is
the test)

- every §44.13 row: 0+pass → 0, 0+fail → 3, 7+pass → 7, 7+fail → 7 with the
  verification failure on stderr, plus child stdout visible through run and the
  listener refused after exit;
- `FAKE_JEV_URL` always set (an inherited `http://127.0.0.1:1` does not survive),
  `--url-env` adds an absent variable and overrides an inherited one, values are
  loopback with an OS port and answer 200 on `/v1/models`;
- ephemeral port: the test holds a listener and writes its port into
  `server.port`; run still exits 0 with a different port, and passing the held
  port via `--port` exits 2 proving the flag is the only path to a fixed port;
- usage failures (no `--`, empty command, unknown flag, empty `--url-env`,
  out-of-range port, stray positional) → 2 with usage on stderr and no child;
- startup failures: invalid config, missing config, unbindable `--port`,
  unstartable child → 2, no child, held listener untouched;
- signals: SIGINT and SIGTERM are forwarded (the helper records the signal it
  handled) with exit 128+signal and the listener closed; a helper that ignores
  the signal is force terminated after the configured
  `gracefulShutdownSeconds: 2`, observed still draining 1 s in, with its pid gone
  afterwards;
- a closed-stdin pipeline does not change the outcome.

`internal/cli/dispatch.go` tests are covered by the existing
`TestUsageListsServeAsACommand` / `TestUsageListsValidateAsACommand` plus the new
help checks.

## Commands run and outcomes

```text
gofmt -w internal/cli/{verify,verify_test,run,run_test,dispatch,serve}.go
gofmt -l internal/cli cmd                                    -> clean
go test ./internal/cli -count=1                              -> ok 18.4s
go test ./... -count=1                                       -> all packages ok
go test -race ./... -count=1                                 -> all packages ok
go vet ./...                                                 -> clean
go run ./cmd/guard arch                                      -> exit 0, 0 findings
go run ./cmd/guard lint                                      -> exit 0, 0 findings
go run ./cmd/guard trace                                     -> exit 0, 0 findings
go run ./cmd/guard fuzz                                      -> exit 0, 0 findings
go build ./cmd/fake-jev                                      -> ok
GOOS=windows go build -o /tmp/fake-jev.exe ./cmd/fake-jev     -> ok (cross-compile proof; artifact removed)
./scripts/candidate-fingerprint candidate                    -> ae99fe0fc8cbe134424e38857e8d693afeea99c9cbc2039f71605b76dc99a813
./scripts/verify-candidate FJ-032                            -> RESULT: PASS (20 passed, 0 failed, 0 skipped)
```

## Spec-silent choices recorded as observations

Specifier observations are referenced by ID; new ones are marked NEW.

- V-U1 wording: verify prints `fake-jev: <url>: verification passed` on stdout
  on success and `fake-jev: <url>: verification failed` plus one line per §41.9
  item (`code: message (requestSequence N) (stubId S)`) on stderr on failure.
  run prints no result line of its own; its stdout carries the child's output.
- V-U2 URL schema: the plain base form and the same form with a trailing slash
  are accepted and both target exactly `/__fake/v1/verify`; a path, query, or
  fragment is a usage failure rather than being silently dropped.
- V-U3 timeout: `verificationTimeout = 10s`, mapped to exit 2.
- V-U4 non-200: every non-200 (including 204) is exit 2 regardless of body.
- V-U5 `passed`/`failures` inconsistency: the exit is keyed on `passed` only, so
  `passed:false` with no items is still 3 and `passed:true` with items is 0.
- V-U6 help: `verify --help`, `run --help` print usage on stdout and exit 0 (the
  same choice `serve` already made; `validate` rejects `--help`, so the CLI is
  not uniform here).
- R-U1 readiness artifact: `run` has no `--ready-file`; readiness is structural —
  the listener is bound and the control API attached before `startServer`
  returns, so the child's first connection is accepted from the listen backlog
  with no retry.
- R-U2 flag set: `run` accepts exactly `--config`, `--port`, `--url-env` plus `--`.
- R-U3 signal-derived status: 128+signal of the received signal (143/130), and a
  child killed by a signal maps to 128+signal where the platform exposes the wait
  status; a status the platform does not expose falls back to 2 rather than
  being reported as success.
- R-U7 child grace: `limits.gracefulShutdownSeconds` (default 5) is the child's
  graceful timeout, the same value `serve` uses.
- R-U4/R-U5-adjacent NEW: the child inherits run's stdout/stderr and reads the
  null device; a signal arriving before the child starts is buffered (channel
  capacity 1) and forwarded once the child exists.
- NEW: when the child exits nonzero **and** verification could not be obtained at
  all (operational failure), run preserves the child's code and reports the
  operational failure on stderr; §42.2 fixes only the case where verification
  *fails*, and "preserve the child exit code" reads as the dominant rule.
- R-U6/ENV NEW: run does not mutate its own environment, only the child's.
- NEW: interrupt handling still attempts verification before the shutdown, then
  reports the signal-derived status regardless of the verification result.

## Residual risks

- **No process-group forwarding.** §42.3 asks for the child "process/process
  group where supported". Forwarding targets the child process only: process
  groups require `syscall.SysProcAttr{Setpgid: true}`, which does not compile on
  Windows, and the allowed-file set has no place for a build-tagged file. A child
  that spawns grandchildren (for example `npm test`) can therefore leave them
  behind when `run` is interrupted programmatically; an interactive Ctrl-C is
  unaffected because the shell delivers SIGINT to the whole foreground group.
- Ready-file-less readiness is structural, not probed: if the `Serve` goroutine
  failed after `startServer` returned, `run` would only notice through the
  child's/verify's failed requests (exit 2), not before the child starts.
- `run` does not surface a mid-run `Serve` error from `handle.serveErr`;
  verification of a dead server becomes an operational failure → exit 2 (or the
  child's nonzero code is preserved).
- The `run` suite compiles a helper program per test function (about 8 `go
  build` invocations, cache-warm) and adds ~12 s to `./internal/cli`; the
  force-terminate case intentionally spends 1 s + the configured 2 s grace.
- Signal tests assume Unix signal/kill-0 semantics, like the existing `serve`
  tests; Windows behavior is untested (only cross-compilation is checked).
- `internal/host/http/server.go` remains unchanged, so the FJ-031 hardener's F5
  concern is answered by a cli-level seam rather than a host-level one.

## Handoff

- No spec gap was hit; nothing blocked the stage.
- `state.json` and `.agent/reports/**` were not edited. `./scripts/verify-candidate
  FJ-032` was run as the repository completion check and wrote its own report
  (`RESULT: PASS`, bound to the final fingerprint above).
