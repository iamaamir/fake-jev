---
stage: specifier
task: FJ-032
inputFingerprint: c8ee98ed15771f58cb8cecedd751633ac6c7bc4414a294da0db14807fda43338
outputFingerprint: c8ee98ed15771f58cb8cecedd751633ac6c7bc4414a294da0db14807fda43338
taskFingerprint: 387ddaf6229fbab020186f256ec6fac2558180842ed9dbb9cac9f97728080a5d
gitHead: 22c0704
generatedAt: 2026-09-30T16:48:32Z
author: worker/FJ-032-specifier
---

# Specifier — FJ-032

## Scope as bounded here

Make two command surfaces observable:

1. `fake-jev verify --url <url>` — a thin control-API call to
   `GET /__fake/v1/verify`, with exit `0`/`3`/`2` semantics and a stream
   contract, and with no verification logic in the CLI (§15.3, §42.1, §41.9).
2. `fake-jev run -- <cmd>` — orchestration: validate, start an ephemeral
   loopback server, wait for readiness, inject the server URL into the child
   environment, run the child, verify after the child exits, shut down
   gracefully, and return the §42.2/§44.13 exit precedence result — including
   signal forwarding per §42.3 (§15.4, §42.1, §42.2, §42.3, §44.13).

State of the tree at this revision (verified by reading the code, not
assumed):

- `internal/cli/dispatch.go` routes `version`, `serve`, `validate`, `help`;
  `verify` and `run` are unrouted and the usage text still says "The run and
  verify subcommands land in later work items." `cli.Run(args, stdout, stderr)`
  is the whole process entry point (`cmd/fake-jev/main.go` calls it and exits
  with the return value) and receives both streams as writers. `exitOK = 0`,
  `exitFailure = 2`, `exitVerify = 3` already exist.
- `internal/cli/verify.go` and `internal/cli/run.go` do not exist.
- `internal/cli/serve.go` owns the lifecycle seam `run` must reuse:
  `parseServeFlags`/`validateServeFlags`/`resolveServeConfig`/`applyServeFlags`,
  `defaultConfig` (§12.2 defaults), `hosthttp.NewServerFromConfig` +
  `AttachControl(control.NewAPI(...))` + `net.Listen` (so `--port 0` yields an
  OS-assigned port), `writeReadyFile`/`removeReadyFile` (create-temp + atomic
  rename), the SIGINT/SIGTERM `select`, and the `Shutdown`-then-`Close` drain
  bounded by `cfg.Limits.GracefulShutdownSeconds`. `serve()` currently returns
  `0`/`2` and takes the ready path explicitly; it is a single function with an
  internal signal select, not a startable handle, so `run` cannot start a
  server and keep ownership of the listener, URL, and shutdown without a seam
  (this is the FJ-031 hardener's F5).
- `internal/cli/validate.go` shows the project's command style: `usageError`
  writes a diagnostic plus the usage text to **stderr** and returns `2`;
  success data goes to stdout; failures go to stderr.
- `internal/control` implements `GET /__fake/v1/verify` (§41.9): it derives the
  result from engine state, always answers HTTP `200` when it can be
  calculated, and never derives a failure from a configured raw provider
  error. `internal/control.API` is an `http.Handler`.
- `internal/host/http/server.go` classifies `/__fake/` traffic and delegates it
  to the attached control handler with the control-plane body limit; control
  traffic never enters the data-plane journal. `NewServerFromConfig(cfg)`
  builds the engine + router from a validated config.
- `internal/config.Validate` fills `limits.gracefulShutdownSeconds` default
  `5` (range `1..60`, §38.3).
- `internal/cli/serve_test.go` already builds `./cmd/fake-jev` once per test
  binary (`TestMain`), captures stdout/stderr separately, uses the ready file
  as the readiness barrier, and reaps children — the harness this item's
  tests should reuse rather than reinvent.

Files this item may carry: `internal/cli/dispatch.go` (route `verify` and
`run`, refresh usage), `internal/cli/verify.go` (new), `internal/cli/run.go`
(new), `internal/cli/verify_test.go` (new), `internal/cli/run_test.go` (new),
`internal/cli/serve.go` (extract a reusable start/stop lifecycle seam),
`internal/host/http/server.go` (only if the seam needs an explicit host hook).

Boundary with FJ-031 (do not re-open): `serve`'s public behavior, ready-file
contract, and signal handling are already delivered and are reused here, not
redefined. The `verify`/`run` halves of C-CLI-004, C-CLI-007, and C-CLI-001
(exit `3`), plus C-CLI-002, C-CLI-003, C-CLI-008, and C-GOLD-013, are this
item's.

## Observable acceptance criteria

Each criterion states what an observer sees, the specification clause it
traces to, and a deterministic exercise. "Falsified by" names the concrete
event that would show the criterion unmet. Readiness barriers and
synchronization are named per criterion; no exercise uses a wall-clock sleep
as a sync point, no exercise picks a port the OS did not hand out, and no
exercise leaves loopback.

### V1 — `verify` command shape and usage failures

**Observable.** `fake-jev verify --url <url>` is a recognized command.
`--url` is REQUIRED; an invocation missing `--url`, naming an unknown flag,
supplying an unparseable URL value, or passing a stray positional argument
exits `2`, writes a diagnostic plus the usage text to **stderr**, writes
nothing that claims a verification result to stdout, and makes **no** network
request. The usage text (`fake-jev help`) lists `verify`.

**Spec.** §15 (required command list, no interactive prompts, usable
non-interactively), §15.3 (`verify --url <url>`), §42.1 (`2` = usage failure),
§4.6 (fail closed).

**Exercise.** In-process: call `cli.Run` with each malformed argument list and
two buffers, pointed at an `httptest`-style loopback server that records
requests; assert exit `2`, the stderr diagnostic + usage, that stdout holds no
result, and that the recorder saw zero requests (the usage gate returns before
any dial). No listener of the subject is involved and nothing is bound.
Falsified by: any non-`2` code, a request reaching the recorder, or a
success-looking stdout line on a usage failure.

### V2 — `verify` is a single control call and duplicates no verification logic

**Observable.** `verify --url <base>` issues exactly **one** HTTP request,
`GET <base>/__fake/v1/verify` (method `GET`, path exactly `/__fake/v1/verify`,
no request body), and its exit code and report are a pure function of that
response. The CLI loads no configuration file, accepts no `--config`, consults
no engine/stub/expectation state, and computes no failures of its own. The
request is control-plane traffic: against a real server it does not appear in
`GET /__fake/v1/requests`, and repeating `verify` does not change the
verification result (§41.9 is a read).

**Spec.** §15.3 (`verify` MUST call `GET /__fake/v1/verify`; MUST NOT duplicate
verification logic in the CLI), §11.1/§41.9 (the verify operation is a
control-plane read whose result is calculated server-side), §11.4 (the control
plane, not the CLI, owns verification).

**Exercise.** Two legs.
(a) Canned server (`net/http/httptest` on `127.0.0.1:0`, or a bare
`net.Listen("tcp","127.0.0.1:0")`) that records every request and answers a
fixed §41.9 body. Drive `cli.Run` in-process (or as a child) and assert:
exactly one recorded request; method `GET`; path `/__fake/v1/verify`; empty
body. Run it with a directory that contains a deliberately invalid
`fake-jev.yaml` as cwd to show no config is consulted (the canned result still
decides the exit).
(b) Real server: start a `serve` child with `--port 0 --ready-file`, use the
ready file as the readiness barrier, issue one failing data-plane request
(e.g. `GET <url>/nope` → `404 fake_jev_unknown_route`, §44.16), run
`verify --url <url>` → exit `3`; then `DELETE /__fake/v1/requests` and run
`verify` again → exit `0`; assert `GET /__fake/v1/requests` contains no record
of either `verify` call. Wording is not asserted (§42.5).

Falsified by: more than one outbound request, a non-`GET`/off-path request, a
config file being required or read, a verify call appearing in the journal, or
an exit code that does not track the canned `passed` value.

### V3 — `verify` exit `0` when `passed` is true

**Observable.** With a `200` response whose body is `passed: true`, `verify`
exits `0`. Nothing that asserts a verification failure appears on either
stream. Any confirmation line the command chooses to print is shell-consumable
and, when present, must not be a failure report; success wording is not
contract (§42.5).

**Spec.** §41.9 (`passed: true` shape, always HTTP `200`), §42.1 (`0`
success; `verify` returns `3` only when `/verify` reports `passed: false`),
§42.5.

**Exercise.** Canned server returning `{"passed":true,"failures":[]}` → assert
exit `0` and `!bytes.Contains(stdout+stderr, "fail")`-style absence of a
failure report is **not** required (wording is free); assert only exit `0` and
that no verification-failure report is required to be present. Real-server
leg: a fresh `serve` with no data-plane requests, `verify` → `0`. Falsified
by: any non-zero exit with `passed: true`.

### V4 — `verify` exit `3` when `passed` is false, with a stderr report

**Observable.** With a `200` response whose body is `passed: false`, `verify`
exits `3`. The returned verification failure(s) are reported as a diagnostic on
**stderr**; the exit code is keyed on the `passed` boolean per §42.1, so a body
with `passed: false` yields `3` and the failure items (code, message,
`requestSequence`, `stubId` per §41.9) are the reportable content. Failure
wording/format is not contract (§42.5); the stream and the exit code are.

**Spec.** §42.1 ("`verify` MUST return `3` when `/verify` reports
`passed: false`"), §41.9 (failure shape and the required codes), §42.5
(operational logs to stderr), §11.4 (the conditions that make verification
fail).

**Exercise.** Canned server returning the §41.9 failure shape (e.g.
`unmatched_request` with `requestSequence: 2`, plus an `expect_exactly`
failure) → assert exit `3` and that stderr is non-empty. Real-server leg:
`serve --port 0`, then a data-plane request that fails verification
(`GET /nope` → `unknown_route`), `verify --url` → `3`; a second failing shape
via an unsatisfied `expect.exactly: 1` stub with no traffic → `3`. Repeat
`verify` twice in a row → same exit, same server-side result (read-only).
Falsified by: exit `0` with `passed: false`, or no failure report on stderr.

### V5 — `verify` exit `2` on operational failure

**Observable.** `verify` exits `2` (fake-jev-owned operational failure, not a
verification result) and reports a diagnostic on **stderr** — with no
verification-failure claim on stdout — when `/verify` cannot be obtained or
understood:

| Situation | Observable |
|---|---|
| URL unreachable (nothing listening on the loopback port) | exit `2`, stderr diagnostic |
| connection refused / reset mid-request | exit `2` |
| DNS/host resolution failure (e.g. a name that does not resolve) | exit `2` |
| server answers a non-`200` status for `/verify` | exit `2`, **regardless of body** |
| `200` with a body that is not a JSON object or lacks a boolean `passed` | exit `2` |
| request times out (bounded client timeout) | exit `2` |

§41.9 requires the endpoint to answer `200` whenever verification can be
calculated; a non-`200` therefore signals a failure to obtain the result, not
a `passed: false`. There is no code path that returns `3` from a non-`200`.

**Spec.** §42.1 (`2` = CLI usage, configuration, startup, control-API, or
internal operational failure), §41.9 (endpoint always `200` when calculable),
§19.5/§15.3 (loopback control call), §4.6 (fail closed).

**Exercise.** Canned servers on `127.0.0.1:0` returning each bad response
(`500`, `404`, `204`, `200` with `not json`, `200` with `{}`, `200` with
`{"passed":"no"}`) → assert exit `2` and stderr non-empty. For unreachable,
close the loopback listener first (or point at a port the OS just released)
and assert the dial is refused → exit `2`; for timeout, a canned handler that
never writes headers, with a bounded client timeout. No sleep is used as the
sync; `httptest`/listener lifetime is the barrier. Falsified by: exit `3` (or
`0`) for any row.

### R1 — `run` command shape

**Observable.** `fake-jev run [flags] -- <cmd> [args...]` is a recognized
command. The command after `--` is REQUIRED. A usage failure — no `--`, an
empty command after `--`, an unknown flag, a `--url-env` with an empty name, a
stray positional before `--` — exits `2`, writes a diagnostic plus the usage
text to **stderr**, starts no server, runs no child, and writes nothing that
claims a result to stdout. The usage text lists `run`. v1 adds no interactive
prompt (§15: commands MUST be usable non-interactively in CI).

**Spec.** §15 ("v1 MUST NOT add interactive prompts"), §15.4 ("The command
after `--` is REQUIRED"), §42.1 (`2`), §4.6 (fail closed).

**Exercise.** In-process `cli.Run` with each malformed argument list, two
buffers, and a child command that would write a sentinel file if ever run;
assert exit `2`, stderr diagnostic + usage, empty stdout result, and no
sentinel file. No listener, no child. Falsified by: a sentinel file, a bound
listener, or a non-`2` code.

### R2 — `run` uses an ephemeral loopback port unless `--port` is passed

**Observable.** `run` starts its in-process server on an OS-assigned ephemeral
port by default, **regardless of the configured `server.port`**, unless the
caller explicitly passes `--port` to `run`. The injected URL host is loopback
(`127.0.0.1`, §19.1/§30.8) and the port is > 0 and is the port the socket
actually listens on. This is the §15.4 half of C-CLI-004.

**Spec.** §15.4 ("`run` MUST use an ephemeral port by default regardless of the
`server.port` file value unless the caller explicitly passes a `--port` flag
to `run`"; "This avoids CI port collisions"), §15.1 (`--port 0` = ephemeral),
§19.1/§30.8 (loopback default), §12.2 (`server.port` default).

**Exercise.** Deterministic no-coincidence design: the test itself opens
`net.Listen("tcp","127.0.0.1:0")` and **holds** it; it writes that held port
into the config as `server.port`, then runs `run` with that config and no
`--port`. Assert `run` exits `0` and the child-observed `FAKE_JEV_URL` port is
different from the held port → the configured value was not bound (an
implementation that honored the file would have failed to bind and exited
`2`). Assert the URL host is loopback. Second leg: `run --port 0` still yields
a live ephemeral port. Third leg (only `--port` honored): pass the held port
via `--port` → assert exit `2` (bind failure, §42.1/R6), proving the flag is
the only way a fixed port reaches the server. No numeric port is chosen by the
test. Falsified by: a child URL whose port equals the configured/hold port, a
non-loopback host, or `run` succeeding while bound to a port the test holds.

### R3 — `FAKE_JEV_URL` injection and `--url-env` override

**Observable.**

- `run` **always** sets `FAKE_JEV_URL` in the child environment to the server
  base URL, including when the parent already supplied a different
  `FAKE_JEV_URL` (the child sees `run`'s value, not the inherited one).
- If `--url-env NAME` is supplied, the child additionally sees `NAME` set to
  the same URL, overriding any pre-existing value of `NAME` in the parent
  environment.
- The value is exactly the reachable base URL of the live server: `<scheme>`
  `http`, loopback host, chosen port; the child's HTTP request to it succeeds
  for the duration of the child's life and the port is refused after `run`
  exits (graceful stop).
- No other environment variable is required to be set or unset.

**Spec.** §15.4 steps 4–5 ("always set `FAKE_JEV_URL` in the child environment
to the server base URL"; "if `--url-env NAME` is supplied, also set `NAME` to
the same URL, overriding an existing child value"), §28 (`JEV_BASE_URL`),
§26, §19.1.

**Exercise.** The child is the test binary re-executed in a helper mode
(env-selected) or a tiny compiled helper written into `t.TempDir()`; it writes
a JSON snapshot of `FAKE_JEV_URL` and `NAME` (plus the HTTP status of one
`GET <FAKE_JEV_URL>/v1/models`) to a file whose path comes from the test, then
exits with the mode's code. The test launches `run` as a child with
`FAKE_JEV_URL=http://127.0.0.1:1` and `NAME=stale` pre-set in the environment,
uses `run`'s own exit as the barrier, then reads the snapshot and asserts:
`FAKE_JEV_URL` equals the value `NAME` received, host is loopback, port > 0,
the request status is `200` (the §44.1 default model list), and both differ
from the sentinels. After `run` exits, dial the recorded URL and assert it is
refused (server stopped). The same run with `--url-env` omitted asserts only
`FAKE_JEV_URL`. Falsified by: an inherited sentinel surviving into the child, a
`NAME` that differs from `FAKE_JEV_URL`, a URL the child cannot reach, or a
still-listening port after `run` exits.

### R4 — `run` lifecycle order and post-run verification/cleanup

**Observable.** The order in §15.4 is observable through side effects:

1. configuration is validated **before** anything is started — an invalid
   config exits `2` with no listener ever bound and no child ever started;
2. the server is listening and the control API ready before the child starts —
   the child's first call to `FAKE_JEV_URL` succeeds without any retry;
3. the child is waited for;
4. **verification is still attempted after the child exits, including when the
   child exits nonzero**, and the server is stopped gracefully afterward — so
   `run` always terminates with the listener closed, even for a nonzero child.

**Spec.** §15.4 steps 1–9, §42.2 ("After the child exits, `run` MUST still
attempt verification and clean shutdown"), §42.3 (graceful stop), §42.1.

**Exercise.** Invalid-config leg: `run --config <invalid> -- <helper>` with the
sentinel-file helper; assert exit `2`, no sentinel, and (by probing the child's
would-be behaviour) no child ran. Ordering leg: the R3 helper only records
after a successful request to the injected URL; a passing exit proves the
server was ready first. Nonzero-child leg: child exits `7` while a
verification failure is present (child performs `GET <url>/nope` then exits
`7`); after `run` exits, assert `7`, that the recorded URL is refused (server
stopped), and that stderr contains the verification failure (verification was
attempted). No sleep is used; `run`'s exit is the barrier. Falsified by: a
child started after a failed config, a child that cannot reach the server on
first try, a nonzero child leaving the listener open, or verification skipped
because the child was nonzero.

### R5 — `run` exit precedence (§42.2 / §44.13, incl. C-GOLD-013)

**Observable.** The exact table, with verify pass/fail produced by real server
state:

| child exit | verification | `run` exit | extra |
|---|---|---|---|
| `0` | pass | `0` | — |
| `0` | fail | `3` | verification failure reported on stderr |
| `7` | pass | `7` | child code preserved; no verification-failure report required |
| `7` | fail | `7` | child code preserved **and** the verification failure is reported on stderr (both failures reported, §42.2) |
| — | — | startup failure → `2` | diagnostic on stderr |

The table is exercised for the explicit §44.13 `7` row and for "nonzero"
generally: any nonzero child code is preserved verbatim (e.g. `2`, `7`), and a
nonzero child never becomes `3` merely because verification failed.

**Spec.** §42.2 ("child exit 0 + verify pass -> 0; child exit 0 + verify fail
-> 3; child exit nonzero -> preserve child exit code"; "If the child exit is
nonzero and verification also fails, `run` MUST report both failures to
stderr but MUST preserve the child's nonzero exit code"; "If fake-jev cannot
start the server or run the child at all, it exits `2`"), §44.13 (the golden
vector), §42.1.

**Exercise.** Deterministic pass/fail generation, no reliance on timing or
engine internals:

- **verify pass** = child makes no failing interaction (a fresh config, child
  does nothing, or the child does `GET <url>/v1/models` which is a matched,
  non-failing interaction, §44.1/§11.4).
- **verify fail** = child performs one deterministic failing data-plane
  interaction: `GET <url>/nope` → `404 fake_jev_unknown_route` (§44.16), which
  §11.4 makes a verification failure.

The helper is env-driven with mode `pass`/`fail` and an exit code; the test
runs `run --config <cfg> -- <helper>` for each of the four rows, using `run`'s
exit as the barrier, and asserts the exit code plus (for the failing rows) the
presence of a verification-failure report on stderr and its absence on the
passing rows. The combination `child=7 + verify fail` additionally asserts
stderr contains **both** the nonzero-child and verification indications
(§42.2 "report both failures"). Falsified by: `0` where the table says `3`/`7`,
`3` where the table says `7`, `2` for a child-derived nonzero, or a nonzero
child whose verification failure is silently dropped.

### R6 — `run` startup failure exits `2`

**Observable.** `run` exits `2`, with a diagnostic on stderr, when it cannot
start the server or cannot run the child at all, including:

- `--config` naming a missing/unreadable file or a config invalid under
  §12.2/§38;
- the requested listener cannot bind (`--port <held port>`);
- the child executable cannot be started (missing binary);
- any internal operational failure while serving.

No listener is left behind, no child runs, and no verification result is
reported.

**Spec.** §42.2 ("If fake-jev cannot start the server or run the child at all,
it exits `2`"), §42.1 (`2`), §15.4 step 1, §4.6.

**Exercise.** Three legs, all without a fixed test-chosen port: (1) invalid
config + sentinel helper → `2`, no sentinel; (2) `run --port <port held by a
test-owned listener>` → `2` (bind failure), and the held listener is untouched;
(3) `run -- definitely-not-a-real-binary-<random>` → `2` (exec `ENOENT`), with
stderr diagnostic. The held-listener port comes from the OS
(`net.Listen("tcp","127.0.0.1:0")`). Falsified by: a non-`2` code, a child
started on a config failure, or a leaked listener.

### R7 — `run` signal forwarding and graceful child shutdown (§42.3)

**Observable.**

- Sending `SIGINT` or `SIGTERM` to a running `run` process forwards the signal
  to the child (process/process group where supported): a child that handles
  the signal observes it while `run` is still alive.
- `run` allows the child up to the configured graceful shutdown timeout
  (`limits.gracefulShutdownSeconds`, default `5`, §12.2/§38.3) before force
  terminating it, then stops the server gracefully. A child that ignores the
  signal is force-terminated within the timeout plus a margin; a child that
  exits shortly after handling the signal is allowed to finish rather than
  being force-killed immediately.
- When `run` finally exits, its server listener is closed (a dial to the
  child-recorded URL is refused).
- The final exit code preserves conventional signal-derived semantics where
  the platform exposes them; the exact numeric value is platform-dependent
  (V-U3). The §42.1/§42.2 success/failure rules for normal non-signal
  execution are unchanged.

**Spec.** §42.3 (run steps 1–5), §42.1, §12.2/§38.3
(`gracefulShutdownSeconds` default/range), §32.

**Exercise.** The helper child is the test binary re-executed in a mode that
(a) writes a "signal received" file and exits `0`, or (b) writes the file and
blocks/ignores the signal. The test launches `run` as a child, waits for the
child to be live by the helper's "started + reached server" file (no ready
file for `run` is specified — V-U1), sends `SIGTERM` to `run`, then uses the
helper's "signal received" file as the forwarding barrier and `run`'s exit as
the termination barrier. Leg (a): assert the file appears (forwarded) and
`run` exits within `gracefulShutdownSeconds + margin`, and the recorded URL is
refused afterwards. Leg (b): assert the file appears, then that the helper
process is gone within `gracefulShutdownSeconds + margin` (force-terminate),
and `run` exits; assert `run` had not exited before the helper was killed by
using the bounded negative observation of FJ-031 S8c (a generous lower
window, not a sleep-as-sync). Repeat for `SIGINT`. Falsified by: the helper
never observing the signal, `run` exiting before force-termination, a child
outliving the timeout plus margin, a still-open listener, or a change to the
normal (non-signal) exit rules.

### R8 — `run` is non-interactive and hermetic

**Observable.** `run` requests no stdin input and prompts for nothing; it
performs no network egress beyond the child's loopback calls and the
in-process server's loopback listener; no model or provider call occurs. The
child's stdout/stderr are visible to `run`'s caller (CI must see `npm test`
output); `run`'s own operational output goes to stderr per §42.5.

**Spec.** §15 (no prompts, usable non-interactively), §4.9/§19.2 (no model
egress), §28 (CI workflow), §42.5 (stderr logs, stdout consumable data),
§4.6 (fail closed).

**Exercise.** Launch `run` with stdin wired to a pipe that is immediately
closed (EOF); assert the same observable outcome as with no stdin input — no
prompt, no hang. Capture stdout/stderr separately; assert the child's own
output appears and no provider/egress endpoint is contacted (the test provides
no such endpoint and observes loopback only). Falsified by: a hang on stdin
EOF, a prompt, or a non-loopback dial.

## Exercise strategy and non-flakiness rules

**H1 — Harness reuse.** Reuse `internal/cli/serve_test.go`'s `TestMain` build
of `./cmd/fake-jev`, the separate stdout/stderr capture buffers, and the
child-registration/cleanup helpers rather than adding a second harness. `run`
and signal cases must run the compiled binary as a child (a real child process,
real signals, real exit status); pure `verify` usage/response cases may run
`cli.Run` in-process against a canned server.

**H2 — Synchronization points (no sleeps-as-sync).** `verify`'s response legs
synchronize on a canned `httptest`/`net.Listen` server whose lifetime is the
barrier. `run`'s legs synchronize on `run`'s own exit (`cmd.Wait`) after the
child exits. The helper child's snapshot/signal files are written before it
exits, so reading them after `run` exits is race-free. No test proceeds on
elapsed time alone; every poll/wait carries a bounded deadline used only as a
failure bound.

**H3 — Ports.** Every subject server binds port `0`; ports reach the test only
through the child-observed `FAKE_JEV_URL`/`--url-env` value or the `serve`
ready file. A test may hold its own `net.Listen("tcp","127.0.0.1:0")` to
provoke bind failure (R2, R6); the test never invents a numeric port. The
golden exit-precedence rows use the ephemeral URL only.

**H4 — Network.** Loopback only (`127.0.0.1`/`127.0.0.0/8`); no external
host, no DNS, no egress.

**H5 — Timeouts.** Process-exit waits are bounded by
`gracefulShutdownSeconds + 10 s`; client HTTP timeouts are bounded by `10 s`;
the R7 force-terminate lower window is `1 s` against a configured `5 s` grace
(a bounded negative observation with a 5× margin). No bound is tightened to
make a timing bug observable, and no bound is a synchronization primitive.

**H6 — Lifecycle hygiene.** Every spawned `serve`/`run`/helper child is
registered for cleanup (signal, then `SIGKILL` if needed), reaped, and checked
for leaked listeners and sentinel files; the helper snapshot files live in
`t.TempDir()`. Leaked children or listener residue fail the enclosing test.

**H7 — Deterministic verify pass/fail.** Pass/fail is generated by
deterministic configuration and request choice (`GET /nope` →
`unknown_route`; a matched `GET /v1/models`; an `expect.exactly: 1` stub with
no traffic), never by timing, randomness, or engine internals.

## Undefined by §15.3/§15.4/§42 (observations, not requirements)

- **V-U1 — `verify`/`run` success and failure wording.** §42.5 says tests must
  not depend on human log wording except where the specification defines
  output. §15.3/§15.4 define no output text. The criteria above fix only the
  **stream** (failures/diagnostics → stderr) and the exit code, not the text,
  format, or the presence of any success line. Whether `verify` prints the
  §41.9 failure items verbatim, a summary, or nothing beyond a diagnostic is
  unconstrained.
- **V-U2 — `verify --url` schema.** §15.3 shows a base URL with no trailing
  slash and no path. Whether a trailing slash or a path prefix is accepted,
  how the path is joined, and whether query/fragment are rejected are not
  specified. This item requires the request target to be exactly
  `/__fake/v1/verify` for the shown base form and records the rest as
  undefined.
- **V-U3 — `verify` client timeout.** §15.3 defines no timeout. A bounded
  internal timeout that maps to exit `2` is required by §42.1's "operational
  failure" rule but its value is unspecified.
- **V-U4 — non-`200` verify response.** §41.9 says the endpoint always
  answers `200` when verification can be calculated and gives no semantics for
  any other status. This item maps every non-`200` to exit `2` (fail closed,
  §4.6) and records that an alternative reading (e.g. `3`) is not derived from
  the specification.
- **V-U5 — `passed`/`failures` inconsistency.** §42.1 keys the exit on
  `passed`; §41.9 fixes the item shape. A body with `passed: true` and
  non-empty `failures`, or the reverse, is not specified; the criteria key the
  exit on `passed` per §42.1 and do not define the report for the inconsistent
  case.
- **V-U6 — routing/usage text.** §15 lists the required commands; the exact
  usage string and the exit code for `verify`/`run` with `--help` are not
  specified (the existing CLI treats `flag.ErrHelp` as exit `0` and prints
  usage to stdout — an implementation choice, not a spec clause).
- **R-U1 — `run` readiness artifact.** §15.4 defines no `--ready-file` for
  `run`; readiness is internal. Tests therefore use the child's own successful
  access to `FAKE_JEV_URL` as the readiness barrier. Whether `run` accepts
  `--ready-file` (by reusing the serve seam) is undefined and is not required
  or exercised here.
- **R-U2 — `run` flag set beyond §15.4.** §15.4's example shows `--config`,
  `--url-env`, and (in prose) `--port`; it enumerates no complete flag list.
  Whether service-only flags (`--host`, `--log-format`, `--compat`, `--mode`)
  are accepted by `run` is unspecified; this item requires only the three
  named flags plus the `--` separator.
- **R-U3 — signal-derived exit code.** §42.3 step 5 says to "preserve
  conventional signal-derived exit semantics where the platform exposes them",
  but fixes no numeric value, and it is not stated whether a child that exits
  `0` in response to a forwarded signal yields `0` or a `128+signal` code.
  R7 asserts forwarding, force-termination, shutdown, and refusal afterward —
  not the code.
- **R-U4 — child stdio wiring.** §15.4 does not state that the child inherits
  `run`'s stdout/stderr. §28's CI workflow implies the child's output is
  visible, so R8 asserts the child's output reaches `run`'s caller, but the
  mechanism (inherit vs. relay) is unspecified and untested.
- **R-U5 — `run` interruption during startup.** What a signal delivered
  before the child starts does, and how `run` reports it, is not specified.
- **R-U6 — concurrent `run` servers.** §15.4's ephemeral-port rule avoids
  collisions; the behavior of two `run` invocations sharing configuration or
  output files is unspecified.
- **R-U7 — timeout value used for the child grace.** §42.3 run step 2 says
  "the graceful shutdown timeout" without restating that it is
  `limits.gracefulShutdownSeconds`; this item reads it as the same
  configured value used by `serve` (§12.2/§38.3) and records the alternative
  (a fixed constant) as not adopted.
- **U-ENV — environment mutation scope.** §15.4 says `run` sets variables
  "in the child environment"; it does not state whether `run` may mutate its
  own environment. The criteria assert the child's environment only.

## Non-goals

- **No duplicated verification logic in the CLI.** Neither `verify` nor `run`
  may recompute failures from configuration/engine state, load a config to
  verify, or interpret stub expectations; both obtain the result from
  `GET /__fake/v1/verify` (§15.3, §11.4). This is a non-goal to implement, not
  a behavior to add.
- **No interactive prompts** of any kind (no stdin confirmation, no
  yes/no gate), and no reliance on stdin for normal operation (§15).
- No changes to the public `serve` behavior, ready-file contract, or `serve`
  exit codes delivered by FJ-031; `serve.go` changes are limited to extracting
  a reusable start/stop seam.
- No new flags beyond §15.3/§15.4, no configuration discovery, no
  environment-variable configuration of `verify`, no `--json` result mode.
- No change to `internal/control`, `internal/engine`, `internal/compat`, or
  their contracts; the verify endpoint is only *called*, never modified.
- No model, provider, or network egress; no reimplementation of the engine in
  the child or by the CLI.
- No wrapper/library surface (JSON output for shell parsing is not required for
  v1, §15.5-style `--json` is explicitly optional and out of scope here).

## Traces

- **C-CLI-001** — §42.1: exit codes `0` success, `2`
  usage/config/startup/control/internal, `3` verification failed; `verify`
  returns `3` when `passed: false` (V1, V3, V4, V5, R5, R6).
- **C-CLI-002** — §42.2/§44.13: `run` follows the child/verify precedence
  table, preserves a nonzero child code, and reports both failures when both
  occur (R4, R5).
- **C-CLI-003** — §15.4: `run` always sets `FAKE_JEV_URL`; `--url-env NAME`
  additionally overrides that child variable (R3).
- **C-CLI-004** — §15.4 (the `run` half of the criterion): `run` uses an
  ephemeral port unless `--port` is passed (R2; the `serve --port 0` half is
  FJ-031).
- **C-CLI-007** — §42.3 (the `run` half of the criterion): `run` forwards
  SIGINT/SIGTERM to the child and applies the graceful shutdown timeout (R7;
  the `serve` half is FJ-031).
- **C-CLI-008** — §15.3: `verify` calls `GET /__fake/v1/verify` and duplicates
  no verification logic (V2; `run`'s post-run verification in R4/R5 uses the
  same control operation).
- **C-GOLD-013** — §44.13: `run` exit precedence (R5).

This artifact records observable behavior and traceability; it declares no
implementation outcome.
