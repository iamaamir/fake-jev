---
stage: cleaner
task: FJ-031
inputFingerprint: 969f6a5c976687fa98b89b4616a4622420c81a50470f4116d15ab1d3e71769b4
outputFingerprint: 969f6a5c976687fa98b89b4616a4622420c81a50470f4116d15ab1d3e71769b4
taskFingerprint: 703164b4cd18ba33de0272b4065a7861e596afa85da13bcc16fd751fda500e6a
gitHead: 6094965
generatedAt: 2026-09-30T15:48:44Z
author: worker/FJ-031-cleaner
---

# Cleaner — FJ-031: `fake-jev serve`

## Changes applied

**None.** No file outside this artifact was created, edited, or deleted by this
stage, and no staged file exists. The candidate fingerprint was re-read after
the review and after every command below, and is still
`969f6a5c976687fa98b89b4616a4622420c81a50470f4116d15ab1d3e71769b4`; the pinned
`inputFingerprint`/`outputFingerprint` above are that value.

Reason: the parent pinned the candidate fingerprint for this stage, and every
structure touched by the findings below is inside the fingerprint
(`internal/cli/*.go`, `internal/host/http/server.go`). A cleaner edit is only
made when it is justified *and* re-verified against a new candidate (stage
pack); with the fingerprint frozen, the review is recorded instead of applied.
Nothing found is a defect that falsifies an FJ-031 acceptance criterion, so no
finding needed to be forced through as a change.

Side effects of running the required commands, disclosed: `./scripts/verify-candidate
FJ-031` writes its report artifact, so `.agent/reports/FJ-031/report-20260930T154553Z.json`
now exists alongside the coder's `…T154312Z.json`. `.agent/work/FJ-031/state.json`
was **not** touched by this stage; its `M` status in `git status` is inherited
from the specifier/coder stages (`status: implementing`, `stage: specifier`).

## Files inspected

- `internal/cli/serve.go` (new, 373 lines) — full read.
- `internal/cli/serve_test.go` (new, 1315 lines) — full read.
- `internal/cli/dispatch.go` (diff: one `case "serve"` arm, usage text).
- `internal/host/http/server.go` (diff: `control` field, `AttachControl`,
  `serveControl`; full read for the surrounding `ServeHTTP`/`serveData`/`readBody`).
- `internal/cli/validate.go`, `internal/cli/version.go` (sibling-subcommand
  patterns), `internal/config/load.go` (error text of `Load`/`LoadFile`),
  `internal/host/http/server_test.go:133–152` (existing control-request test).
- `spec/fake-jev-technical-spec-v2.md` §15.1 (serve, ready file), §15.4 (`run`
  lifecycle), §20 (observability), §42.1/§42.3/§42.4/§42.5; `.agent/guards.json`,
  `.agent/gate-policy.json`, `.agent/work/FJ-030/cleaner.md` (F2 pattern),
  `.agent/work/FJ-032/state.json`, `.agent/work/FJ-041/state.json`.

## F1 — configuration diagnostics do not name the config file (error-message consistency)

`runServe` prints `writef(stderr, "fake-jev: %v\n", err)` for the
configuration-resolution failure, and `config.Load` messages carry no path
(only `LoadFile`'s read error does, `internal/config/load.go:47–53`). Observed
on the built binary:

```text
$ fake-jev validate bad.yaml
fake-jev: bad.yaml: mode "loose" is not supported; only strict is valid
$ fake-jev serve --config bad.yaml
fake-jev: mode "loose" is not supported; only strict is valid
```

`validate` names the file; `serve` does not, although it knows the path
(`flags.config`). Exit code, stream, and usage-vs-diagnostic split are correct
and unchanged by the wording, and §42.5 makes log wording non-contractual, so
this is consistency polish only. Not applied (fingerprint frozen); the repair
is one wrap at the `config.LoadFile` call in `resolveServeConfig`, or one
`"fake-jev: %s: %v"` in `runServe` when `flags.set["config"]`.

## F2 — a fatal in-serve error exits 2 and leaves the owned ready file behind (latent defect, later tickets)

`serve()`'s select has two arms. The signal arm runs `Shutdown`/`Close`, then
`removeReadyFile` (§42.3 steps 4–5). The `serveErr` arm returns `exitFailure`
immediately:

```go
case err := <-serveErr:
    writef(stderr, "fake-jev: serve: %v\n", err)
    return exitFailure
```

`net/http.Server.Serve` closes the listener on return, but nothing removes the
ready file, so a `serve` that dies from a fatal accept error leaves a
byte-valid ready document naming a dead PID. The startup-failure arm three
lines above does clean up (`httpServer.Close()`, and the ready file was never
published), so the two failure exits are inconsistent. §42.4 only mandates
removal on *clean* shutdown and is silent about the fatal-error exit, so the
behavior is undefined-by-spec rather than a criterion breach — recorded, not
changed.

Concrete later bite: §15.4 steps 3 and 9 have `run` (FJ-032) wait for readiness
on exactly this file and then stop the server; a stale document from a crashed
`serve` is the same class of hazard the pid guard in §42.4 exists to avoid.
Also on this arm the message shape `"fake-jev: serve: %v"` differs from the
`"fake-jev: %v"` used everywhere else in the file.

## F3 — the force-close test cannot distinguish "reads the limit" from "hardcodes the default" (test-strength proxy, FJ-030 F2 shape)

`TestServeForceCloseAfterGracefulTimeout` writes `limits.gracefulShutdownSeconds: 5`,
which is exactly the §12.2 default (`DefaultGracefulShutdown = 5`,
`internal/host/http/server.go`). The two assertions (no exit within 1 s; exit
within 12 s) therefore hold for an implementation that ignores the configured
value and uses the constant, and for one that honors it. This is the F2 pattern
recorded in `.agent/work/FJ-030/cleaner.md`: the assertion is a proxy for the
property (§38.3 `1..60`, "the timeout is the configured value") rather than the
property. A value such as `3` keeps both observations valid (3 > the 1 s lower
bound, well inside the bounds) and falsifies a hardcoded 5. Not applied
(fingerprint frozen); it is a one-line change in `serve_test.go`.

## F4 — "stdout is empty" over-states §42.5 in three tests (F2 shape)

`TestServeLogsGoToStderrAndRedactAuthorization`, `TestServeUsageFailures` and
`TestServeConfigFailuresAreNotUsageFailures` require `stdout` to be exactly
empty. §42.5 says command result data intended for shell consumption MAY use
stdout and only forbids operational logs there; the specifier recorded the same
reading as U9 ("S10 asserts only that no operational log record appears there,
not that stdout is empty"). The assertions are true today, but they pin a
stricter property than the contract, so the first ticket that gives `serve` a
legitimate stdout line (FJ-032 reusing this lifecycle, a future `--json`
status) sees a red test with no contract violation behind it. Recorded, not
changed.

## F5 — the FJ-041 and FJ-032 boundaries need a seam this item does not yet provide

Recorded because both are already-open work items and the shape is structural
(the F2 pattern's "will bite a later ticket" form):

- FJ-041 (log truncation, §20/§12.2 `logBodyBytes`) has `allowedFiles`
  `internal/host/http/limits_test.go`, `internal/cli/logging.go`,
  `internal/cli/logging_test.go`. The request log wrapper lives in
  `internal/cli/serve.go` and reads only method/path/header, while the body it
  would have to truncate is read one layer down in `internal/host/http/server.go`.
  Neither `serve.go` nor `server.go` is in FJ-041's allowed set, so it needs a
  scope amendment (the FJ-031 precedent) or a host-owned logger seam.
- FJ-032 (`run`, §15.4 steps 2/3/9: in-process server, ephemeral port, wait for
  readiness, graceful stop) has `allowedFiles` `internal/cli/run.go`,
  `internal/cli/verify.go`, `internal/cli/run_test.go`. The reusable pieces of
  this item are package-level and therefore reachable
  (`resolveServeConfig`, `writeReadyFile`, `removeReadyFile`, the flag
  validator), but the listener + signals + shutdown block is inline in
  `serve()` and takes its streams from `Run`, so `run.go` either copies the
  lifecycle or needs `serve.go` added. Neither option is a defect here; it is a
  boundary the stage-3 pair was split across.

## F6 — the nil-control fallback is claimed but not pinned by a test

`serveControl`'s comment says "A host without a control API answers exactly as
a listener without a control plane did." The fallback is the same
`writeJSON(…, StatusNotFound, failureBody{Error: "fake_jev_control_not_found", …})`
call the deleted code made, so the claim holds by construction (verified by
reading the pre/post diff). No test asserts it:
`TestServerControlRequestsAreExcludedFromJournal`
(`internal/host/http/server_test.go:133–152`) sends the small control request
without attaching a control handler but asserts only the oversize status and
the untouched journal. `internal/host/http/server_test.go` is not in this
item's allowed files, so pinning it is a later edit (or unnecessary if the
fallback is considered dead once every host attaches a control API).

## F7 — test lifecycle hygiene: residue is cleared, not asserted

Specifier H6 asks the child cleanup to assert that the ready-file path and its
parent directory hold no residue. `serveChild.stop` `SIGKILL`s if needed and
reaps within 10 s, but asserts nothing about files; residue is only removed by
`t.TempDir()` and is explicitly asserted in the tests that care
(`assertDirectoryEntries`). Because each test gets its own `t.TempDir()`,
leakage cannot reach the next test — the missing assertion costs only
diagnostic sharpness on an already-failing run. `startServe` registers its
`t.Cleanup` after the argument's `t.TempDir()` in every call site, so LIFO order
kills the child before the directory is removed. Recorded, not changed.

## F8 — small control-flow and dedup observations (none worth a fingerprint move)

- `logHandler.ServeHTTP` calls `handler.next.ServeHTTP` before the
  `request == nil` guard, so the guard reads as misplaced. It is load-bearing
  only for the logging deref below it; `net/http` never hands a handler a nil
  request, and reordering it would drop the `500` the wrapped `Server` writes
  for nil, so the current order is the behavior-preserving one. A comment
  stating that would remove the double-take; not applied.
- `redactAuthorization`'s empty-string branch is unreachable from its only call
  site (the caller tests `!= ""` first), which makes the function a
  constant-returning wrapper around `redactedHeaderValue`. It reads as a
  deliberate single redaction point; the branch is dead, not wrong.
- The "on startup failure, close the listener and join the close error" shape
  appears twice with different wording (non-TCP `listener.Addr()` path,
  `writeReadyFile` failure path). A small helper would dedupe it; the two sites
  differ in which error is primary, so the extraction is not obviously a win.
- On the graceful-timeout path the `Shutdown` error is discarded while only a
  `Close` error is logged, so the ordinary force-close (context deadline met,
  `Close` succeeds) is silent. §42.5 makes the wording non-contractual; it is
  an operator-visibility gap, not a behavior gap.
- `flags.set` is a `map[string]bool` keyed by flag-name literals read in three
  functions. A typo compiles and silently drops the flag; all seven keys are
  exercised by `TestServeConfigPrecedenceSeam` plus the usage-failure cases, so
  the risk is bounded. `FlagSet.Visit` is the idiomatic source of the same
  information, already used at the writer side.
- A second `SIGINT`/`SIGTERM` during the drain is ignored (channel capacity 1,
  no reader after the select that consumed the first). §42.3 requires only the
  first-signal behavior; recorded as a possible future hardening item.

## Delegation review — data plane and body limits (task's explicit question)

Read as: the only behavior change in `internal/host/http/server.go` is the
control branch of `ServeHTTP`, and it preserves everything the data plane and
the limits did before.

- Data plane: `s.serveData(writer, request, body, false)` is textually
  unchanged, and the oversized-data path
  (`s.serveData(writer, request, nil, true)`) is unchanged, so journaling,
  routing, validation, matching, failure bodies, and `Outcome`
  (`payload_too_large`) are unaffected.
- Limits: `readBody` is unchanged and still bounded per plane; oversized
  control requests are still answered `413 fake_jev_payload_too_large` *before*
  delegation and are still not journalled (`TestServePerPlaneBodyLimits`
  asserts both directions through the real listener).
- Fallback: with no attached handler, the status, error code, and message are
  the same `writeJSON` call the previous revision made (F6).
- `serveControl` mutates the caller's `*nethttp.Request` (`Body`,
  `ContentLength`) to hand the already-read bytes to the API. This does not
  change connection keep-alive/drain handling: the post-handler drain in
  `net/http` (`server.go:1392` in go1.26.3) reads from the server's internal
  `w.reqBody`, which `readBody` has already driven to EOF, and its
  concrete-type switch on `w.req.Body` takes the `default: discard` branch
  whether the body is the server's `*body` or the injected `io.NopCloser`.
  The mutation is visible only to the control handler, which is the point of
  the restore. A later control endpoint that inspects `Transfer-Encoding` or
  the original `-1` length would see a normalized request; §41's endpoints
  decode JSON only.
- `AttachControl` writes `s.control` without synchronization. It is safe for
  the shipped wiring (attached once in `serve` before the listener serves) but
  the doc comment does not state that "attach before serving" is the contract;
  a host that attached mid-flight would race (`-race` would catch it in a
  test). One sentence in the existing comment would close that.

## Command outcomes

All commands run from `/Users/mak/git/fake-jev-FJ-031`, Go 1.26.3 (darwin/arm64),
candidate fingerprint unchanged before and after.

```text
gofmt -l internal/cli/serve.go internal/cli/serve_test.go internal/cli/dispatch.go internal/host/http/server.go
    (no output)
go test ./internal/cli -count=1                              ok   6.073s
go test ./internal/host/http -count=1                        ok   0.422s
go test ./... -count=1                                       ok   all 9 packages
go test -race ./... -count=1                                 ok   all 9 packages
go vet ./...                                                 (no output)
go test ./internal/cli -count=2 -run 'TestServe|TestUsage|TestRemoveReadyFile'   ok  11.277s
./scripts/verify-candidate FJ-031                            PASS 20 passed, 0 failed, 0 skipped, 0 not_applicable
./scripts/candidate-fingerprint candidate                    969f6a5c976687fa98b89b4616a4622420c81a50470f4116d15ab1d3e71769b4
```

`go test -v -run 'TestServePrecedenceEndToEnd|TestServeDefaultBind'` shows all
eight precedence subtests (including `default_port`, i.e. 8787 was bindable
here) and the loopback test passing; no subtest reported "not exercised".

Determinism/hermeticity spot-checks against the task's list, from reading:

- No fixed port is bound by a test except the normative §12.2 `8787` leg, which
  is probed and skipped if unavailable, and the "port not bindable" case, which
  uses a test-owned `net.Listen("tcp", "127.0.0.1:0")`. No other package in the
  repository mentions 8787 (grep over `*.go`/`*.yaml`/`*.json`).
- `time.Sleep` appears only as a 2 ms poll interval inside deadline-bounded
  loops (`awaitReadyShape`, `awaitDialRefused`) and inside the deliberate 1 s
  lower-bound observation in the force-close test; no sleep is used as the
  synchronization point itself (readiness = ready file, shutdown = exit
  channel, body active = interim `100 Continue`, closure = refused dial).
- Per-request `http.Client`s are not `CloseIdleConnections`-ed, but each test's
  child is shut down (which closes idle connections) and killed in cleanup, and
  the child's exit channel is buffered (capacity 1) so the `Wait` goroutine
  cannot leak.
- `go test -race ./... -count=1` reports nothing; `syncBuffer` covers the
  stdout/stderr capture races and all `serveChild` state is touched only by the
  test goroutine.

## Residual risks

- F2 is the only finding with a plausible operational consequence (stale ready
  file after an in-serve fatal error); it is undefined-by-spec, so it belongs to
  a spec/scope decision rather than to a cleaner edit.
- The `default_port` subtest probes 8787, closes the probe, then starts the
  child; a concurrent process taking 8787 in that window makes the child exit 2
  and surfaces as "ready file … did not reach the ready document", not as a
  skip. Low probability (no in-repo test binds 8787), poor message if it fires.
- `TestUsageListsServeAsACommand` pins the exact usage fragment
  `"\n  serve [flags] "` and the absence of `"serve, run, verify"`; the usage
  text is not a contract (§42.5), and FJ-032 will add lines next to it.
- Everything in F8 is cosmetic; none of it changes an observable the
  specification or the acceptance catalog names.
