---
stage: specifier
task: FJ-031
inputFingerprint: bad76f7917d526d0978fb613aee212f2345ff06a717259788d4a04366f01ab19
outputFingerprint: bad76f7917d526d0978fb613aee212f2345ff06a717259788d4a04366f01ab19
taskFingerprint: 703164b4cd18ba33de0272b4065a7861e596afa85da13bcc16fd751fda500e6a
gitHead: 6094965
generatedAt: 2026-09-30T15:28:30Z
author: worker/FJ-031-specifier
---

# Specifier — FJ-031

## Scope as bounded here

Make `fake-jev serve` observable as a long-running local process: effective
configuration precedence, loopback default bind, ephemeral-port support with
the chosen port reported, an atomic ready file, graceful shutdown on SIGINT
and SIGTERM, stream/redaction discipline for operational logs, the §42.1 exit
codes, and a control API that is reachable through the real listener.

State of the tree at this revision (verified by reading the code, not
assumed):

- `internal/cli/dispatch.go` routes only `version`, `validate`, and `help`;
  the usage text still says `serve` lands in a later work item. `cli.Run` is
  the whole process entry point (`cmd/fake-jev/main.go` calls it and exits
  with its return value), and it receives `stdout`/`stderr` as writers.
- `internal/cli/serve.go` does not exist.
- `internal/host/http/server.go` classifies `/__fake/` traffic
  (`Router.isControl`), bounds its body with the control-plane limit, and then
  answers every control request with `404 fake_jev_control_not_found`. The
  comment on `Server` states the control API is "implemented by a later host
  slice": the listener currently carries no control plane.
- `internal/control` is complete for §41 (`API` implements `http.Handler`,
  `NewAPI(engineState, cfg)` derives metadata from the configuration,
  `NewAPIWithMetadata` exists for hosts with precompiled metadata). Nothing in
  the repository constructs a control API and attaches it to a listener.
- `internal/config` (`Load`, `LoadFile`) applies the §12.2 defaults and
  validates; `Config.Server.Host/Port`, `Config.Compatibility`,
  `Config.Mode`, and `Config.Limits` are the fields a flag layer must be able
  to override. `internal/cli/version.go` holds `ProductVersion`,
  `ControlAPIVersion = "v1"`, and `ConfigSchemaVersion = 1`.

Files this item may carry: `internal/cli/dispatch.go` (route `serve`, refresh
usage), `internal/cli/serve.go` (new: flags, config selection, listener
lifecycle, signals, logging, exit codes), `internal/cli/serve_test.go` (new),
`internal/host/http/server.go` (delegate `/__fake/` to the control API).

Boundaries with neighbouring items (respect these, do not widen):

- FJ-032 owns `verify` and `run`; the §15.4 half of C-CLI-004 and the §42.2
  half of C-CLI-007 are therefore *not* in scope here.
- FJ-041 owns log truncation/limit machinery, `internal/cli/logging.go`, and
  C-HOST-008 hardening; this item owns only the stream destination and the
  redaction invariant as they are observable through `serve`.
- FJ-020/FJ-021 own the control API's contracts and handlers; this item owns
  only whether that API is reachable through the real listener.

## Observable acceptance criteria

Each criterion states what an observer sees, the specification clause it
traces to, and how it is exercised. "Falsified by" names the concrete event
that would show the criterion unmet.

### S1 — Command shape and usage failures

**Observable.** `fake-jev serve` is a recognized command. `serve` accepts
exactly the §15.1 flags: `--host <string>`, `--port <0..65535>`,
`--config <path>`, `--compat <profile>` (repeatable), `--mode <value>`,
`--log-format <value>`, `--ready-file <path>`. A usage failure — an unknown
flag, a missing flag value, a flag value outside its domain
(`--port 70000`, `--port abc`, `--mode loose`, `--log-format xml`,
`--compat unknown/v1`), or a stray positional argument — exits `2`, writes a
diagnostic plus the usage text to **stderr**, writes no success data to
stdout, binds no listener, and creates no ready file. `fake-jev help` and the
usage text no longer claim `serve` is unimplemented.

**Spec.** §15, §15.1, §42.1, §12.2, §38.1, §38.2, §4.6 (fail closed).

**Exercise.** In-process: call `cli.Run` (or the serve entry point) with each
malformed argument list, two buffers for stdout/stderr, and a ready-file path
inside a fresh `t.TempDir()`; assert the exit code, the stderr content, the
empty stdout, and that the ready-file path does not exist. No listener is
involved, so nothing is bound and nothing is racy. The usage-failure paths
must return before any bind attempt, which the absence of the ready file and
the immediate return observable confirm. A test-only flag (`go test`'s
`-test.*` prefix caveat applies only if the parser reuses `flag`; §15.6
leaves the parser choice open) must not turn a usage failure into a silent
success: assert each of the listed inputs.

### S2 — Configuration source precedence: CLI flag > config file > built-in default

**Observable.** The effective configuration is the §12.2 default, overridden
by `--config <path>`, overridden by the explicit CLI flags that are present.
No environment variable changes any of it. The effective values are reported
back through the ready file (`host`, `port`, `url`) and through
`GET /__fake/v1/meta` (`activeProfiles`, `mode`, `limits`), so every leg of the
chain can be observed without reading logs.

| Leg | Invocation | Expected observation |
|---|---|---|
| default host | `serve --port 0`, no `--config`, no `--host` | ready-file `host` = `127.0.0.1`; `url` = `http://127.0.0.1:<port>` |
| default port | `serve`, no `--config`, no `--port` | ready-file `port` = `8787` and the socket answers on `127.0.0.1:8787` (environment probe, H7) |
| default mode/profile | `serve ... --port 0`, no `--config` | `GET /__fake/v1/meta` reports `"mode":"strict"`, `"activeProfiles":["jev/v1"]` |
| default limits | `serve ... --port 0`, no `--config` | `/meta` `limits` equal the §12.2 values (8 MiB / 2 MiB / 10000 / 4096 / 5) |
| file over default (port) | config `server: {port: 0}` + `--config` | ready-file `port` > 0 and != 8787 → the file value was used, not the default |
| file over default (host) | config `server: {host: 127.0.0.2, port: 0}` + `--config` | ready-file `host` = `127.0.0.2`; dialling `127.0.0.2:<port>` succeeds, dialling `127.0.0.1:<port>` is refused |
| file limits | config `limits: {maxInteractions: 7}` + `--config` | `/meta` reports `"maxInteractions":7` |
| flag over file (port) | config `server: {port: 8787}` + `--port 0` | ready-file `port` > 0 and != 8787 → the flag won |
| flag over file (host) | config `server: {host: 127.0.0.1, port: 0}` + `--host 127.0.0.2` | ready-file `host` = `127.0.0.2`; dial `127.0.0.2:<port>` succeeds, dial `127.0.0.1:<port>` refused |
| flag profiles replace file | config `compatibility: [jev/v1]` + `--compat jev` | `/meta` `"activeProfiles":["jev/v1"]` (alias normalized, §38.1) |
| no implicit file discovery | temp cwd containing `fake-jev.yaml` with `server: {host: 127.0.0.2, port: 0}`; `serve --port 0` with no `--config` | ready-file `host` = `127.0.0.1` → the cwd file was not consulted (no discovery rule exists in §15.1; interpretation I4) |
| no environment override | every row above re-run with `FAKE_JEV_HOST=0.0.0.0`, `FAKE_JEV_PORT=9`, `FAKE_JEV_CONFIG=<other file>`, `FAKE_JEV_READY_FILE=<other path>`, `FAKE_JEV_LOG_FORMAT=json`, `HOST=0.0.0.0`, `PORT=9` in the child environment | byte-identical observable to the env-free run of the same row |

**Spec.** §12.1 (precedence, no environment override), §12.2 (defaults), §12.3
(schema), §15.1 (`--config`, `--host`, `--port`, `--compat`), §37 (default
bind `127.0.0.1:8787`, config precedence), §38.1 (`jev` alias, unknown
profiles fail), §38.2 (`port` 0..65535), §38.3 (`limits` ranges).

A flag's *presence* must be distinguishable from its *value*: `--port 0` is
meaningful (ephemeral) and cannot be represented by a zero-value sentinel, and
a config file that omits `server` must not be distinguishable from one that
sets it to the default. The rows "file over default (port)" and "flag over
file (port)" are chosen so that the losing source's value is *never bound*:
the default/failing port `8787` is only ever bound by the row that actually
expects `8787`.

**Exercise.** Each row starts one child `serve` with `--port 0` (except the
default-port row, which is the only leg that needs a fixed port), waits on the
ready file (H2), reads it, optionally issues one `GET /__fake/v1/meta`, and
then terminates the child (H6). Nothing depends on log wording (§42.5), on
wall-clock sleeps, or on a port other than the two loopback test addresses
`127.0.0.1`/`127.0.0.2`. Falsified by: a row reporting the losing source's
value, by an env-set row differing from its env-free twin, or by any row
requiring `8787` to be free where the table does not say so.

### S3 — Default bind is loopback

**Observable.** With no host supplied anywhere, the listening socket is a
loopback address: the ready file reports `host` `127.0.0.1` and
`url` `http://127.0.0.1:<port>`, a dial to that URL succeeds, and the bound
address is not the wildcard address — a dial to any local non-loopback
interface address on the reported port is refused. `127.0.0.1` is the
complete default; the process never binds `0.0.0.0` unless a host is supplied
explicitly (§19.1 permits an explicit `0.0.0.0` for containers).

**Spec.** §19.1 (default bind `127.0.0.1`, do not bind publicly by default),
§30.8 (public bind by default is an anti-pattern), §37 (conformance matrix),
§12.2 (default `server.host`), §32 (definition of done).

**Exercise.** `serve --port 0`; assert the ready-file host parses to a
loopback address (`net.ParseIP(host).IsLoopback()`); dial `127.0.0.1:<port>`
and get an HTTP response. Then enumerate the process's non-loopback local
addresses: if the environment has one, dial it on the reported port and
observe refusal; if it has none (some containers), that sub-check is not
exercisable and is reported as not exercised (H7) rather than assumed.
Falsified by: a wildcard bind (the non-loopback dial would succeed), or a
ready-file host that is not loopback.

### S4 — Ephemeral port support with the chosen port reported

**Observable.** `--port 0` requests an OS-assigned ephemeral port; a config
file with `server.port: 0` does the same (interpretation I2). After
readiness, the ready file reports `port` > 0, and that value is the port the
socket actually listens on: `url` = `http://<host>:<port>`, a dial to exactly
that authority succeeds, and `port` equals the port component of `url`. Two
sequential `serve --port 0` runs report different ports. The chosen port is
never `0` in the ready file and is never a port the OS did not assign.

**Spec.** §15.1 (`--port 0` MUST request an ephemeral OS-assigned TCP port),
§12.3/§38.2 (`port` is `0..65535`), §15.1 (exact ready-file JSON with `port`
and `url`), §30.9 (wrappers must not need to scrape logs).

**Exercise.** `serve --port 0 --ready-file <tmp>`; readiness barrier (H2);
parse the ready file; dial the reported authority; compare `port` with the
`url` authority's port; then a second run and compare ports. All ports come
from the OS; the test never names one. Falsified by: a ready file reporting
`0`, or a reported port that does not accept a connection.

### S5 — Ready file: exact document, created only after readiness

**Observable.** With `--ready-file <path>` the path exists, parses as JSON
with exactly the five keys `url`, `host`, `port`, `pid`, `controlApiVersion`,
and no others; `pid` equals the `serve` process's PID; `controlApiVersion` is
`"v1"` and equals the `controlApiVersion` that `GET
/__fake/v1/health` returns; `host` and `port` describe the actually listening
socket (§15.1's example URL has no trailing slash and no path). The file is
created only after the listener is active *and* the control API is ready:
immediately after the file is first observed, a single `GET
/__fake/v1/health` on the ready file's `url` returns `200` with the §41.2 body
and needs no retry.

**Spec.** §15.1 (ready-file trigger condition, exact JSON, "exact ready-file
JSON"), §41.2 (health shape and status), §30.9 (stable readiness mechanism),
§42.4 (atomic write).

**Exercise.** Readiness barrier: poll for `os.Stat(path)`/read with a bounded
deadline (H2). On the first successful read, parse the document and make one
health request with a bounded client timeout; compare its `serverVersion`
with `/meta`'s and its `controlApiVersion` with the ready file. The
single-shot health check is the observable form of "the readiness file is
written after the control API is ready": no retry loop, no sleep. Falsified
by: a missing/extra key, a `pid` that is not the child's, a health request
that is refused or timing out on the first attempt after the file appears, or
a `url`/`host`/`port` that does not describe the live socket.

### S6 — Ready file: atomic write, overwrite-on-readiness, untouched on failure

**Observable.**

- **Atomic.** While the process starts, a concurrent reader sampling
  `--ready-file` observes only "absent" or the complete valid document; it
  never observes an empty, truncated, or otherwise unparseable path. After
  readiness, the destination directory contains no leftover temporary file
  from the create-temp + rename (§42.4); after clean shutdown it contains
  neither the ready file nor residue.
- **Overwrite only after readiness.** If the destination already holds a
  different document at startup, every sample is either that complete
  pre-existing document or the new complete document — never a mixture — and
  only after startup succeeded does it hold the new document (§42.4: overwrite
  atomically *only after successful readiness*).
- **Untouched on startup failure.** With a pre-existing ready file, a startup
  failure leaves the file byte-identical to what it was. Startup failures in
  scope: `--config` naming a missing or invalid file, a config violating
  §12.2/§38 (`mode: loose`), and `--port <port already held by a test-owned
  listener>`. Each exits `2`, reports on stderr, and leaves the pre-existing
  ready file untouched.

**Spec.** §15.1 (atomic creation only after readiness), §42.4 (create-temp +
atomic rename in the destination directory; overwrite only after successful
readiness), §42.1 (`2` for startup failure), §4.6 (fail closed).

**Exercise.** The reader is a loop (bounded deadline, no sleep-as-barrier)
that reads the whole file at once and records every sample: each sample must
either be absent or parse to the five-key document; a zero-length or
partially-parsed sample is the falsification. The pre-existing document is
written by the test as a valid-but-foreign document (e.g.
`{"pid":999999,"sentinel":true}`) so the sample comparison is byte-exact.
Startup failure uses a listener the test opened on `127.0.0.1:0` and passes
its port to `--port`, so no fixed port is needed. Falsified by: any sample
that is non-empty and unparseable, by temp-file residue in the destination
directory after readiness, or by a modified ready file after a failed start.

### S7 — Ready file removal uses the pid guard

**Observable.** After a clean shutdown (signal-triggered) the ready file is
gone *when its parsed `pid` still equals the serving process's PID*. If an
observer replaces the file with a document carrying a different `pid` while
the server runs, that replacement survives the clean shutdown untouched. A
file that has become unparseable is likewise left untouched (interpretation
I3). Nothing else in the destination directory is removed.

**Spec.** §42.4 ("remove it only if its parsed `pid` still equals the current
process PID"), §15.1 (removal during clean shutdown), §42.3 (step 4).

**Exercise.** Run 1: `serve --port 0 --ready-file <tmp>/ready.json`; readiness
barrier; signal; wait for exit (H2/H6); assert `os.IsNotExist`. Run 2: same
start; readiness barrier; overwrite the file with
`{"pid":<not the child pid>,"url":"...","host":"...","port":0,"controlApiVersion":"v1"}`;
signal; wait for exit; assert the file still holds the exact replacement bytes.
Run 3: same start; overwrite with `not json`; signal; wait for exit; assert
the bytes are unchanged. Falsified by: a missing file after run 1, or a
deleted/replaced file after runs 2–3.

### S8 — Graceful shutdown on SIGINT and SIGTERM

**Observable.**

- **S8a — both signals, clean exit.** Sending `SIGINT`, and separately
  `SIGTERM`, to a ready `serve` process causes it to stop accepting, finish
  shutdown, remove its owned ready file, and exit with code `0`. After exit,
  a dial to the former port is refused. `serve` produces code `3` never.
- **S8b — new accepts stop while in-flight work is served.** With a request
  admitted and being read at the moment the signal arrives, all subsequent
  connection attempts to the port are refused while the process is still
  alive, and the admitted request still receives its complete HTTP response
  after the client supplies the rest of its body. This is the observable form
  of "stopping new accepts" followed by "allowing in-flight HTTP requests".
- **S8c — force close after `gracefulShutdownSeconds`.** With
  `limits.gracefulShutdownSeconds` configured (e.g. `5`) and a request whose
  body is never completed, the process stays alive well after the signal
  (more than 1 s, i.e. it does not abandon the held connection immediately),
  then exits `0` within the configured timeout plus a bound (H5), and the held
  connection is closed by the server (the client observes EOF/reset, not an
  eventual response). The timeout is the configured value, not a constant of
  the implementation (§38.3 range `1..60`).

**Spec.** §42.3 (the five ordered steps, Unix `SIGINT`/`SIGTERM`, "allowing
in-flight HTTP requests up to `gracefulShutdownSeconds`", "force-closing
remaining server work after the timeout", "exiting cleanly"), §42.1 (`0`
success), §12.2/§38.3 (`gracefulShutdownSeconds` default and range), §32
("signal handling ... satisfy Section 42").

**Exercise (S8a).** Child process, readiness barrier, then
`Process.Signal(syscall.SIGTERM)`; wait for exit with a bounded wait (H5);
assert exit status `0`, ready file absent, and a dial refused afterwards.
Repeat identically with `syscall.SIGINT`. No sleeps; the sync points are the
ready file before the signal and process exit after it.

**Exercise (S8b).** Deterministic synchronization without sleeps is available
from the HTTP transport itself: open a raw TCP connection, send request line +
headers for a data-plane request with `Content-Length: 65536` and
`Expect: 100-continue`, and one body byte. Go's `net/http` writes
`HTTP/1.1 100 Continue` on the first body read, which happens inside the
host's body read for the request under test — receiving that interim response
proves the handler is active and blocked on the request body. Then signal;
then poll (bounded) until a *new* dial to the port is refused; assert the held
connection is still usable; then send the remaining bytes (a valid
`POST /v1/systemone` document padded with trailing spaces inside the declared
length, so the request reaches stub selection and, with no stubs configured,
produces `501 fake_jev_unmatched_request` with the §43.1 body) and read the
complete response. Finally assert exit `0`. The interim-response trick is a
test synchronization device only; if a future host change removes it, the same
observable can be synchronized with a test-only seam inside the allowed files
(`internal/host/http/server.go`, `internal/cli/serve_test.go`) that reports
"request body read started" — recorded as a fallback, not as product
behavior.

**Exercise (S8c).** Configure `limits.gracefulShutdownSeconds: 5`; hold a
request the same way (interim response as the sync), signal, then:
(a) assert the process has not exited after a 1 s observation window — with a
5 s configured grace, an implementation that honors the timeout cannot
legitimately have exited; (b) assert it exits `0` within 5 s + margin;
(c) assert the client's read on the held connection ends in EOF/ECONNRESET.
The 1 s lower bound is a bounded negative observation with a 5× margin, not a
sleep used as synchronization.

**Falsified by:** a non-zero exit, a surviving ready file, a connection
accepted after the signal while draining, an in-flight request whose response
never arrives, an immediate exit while a request is still held, an exit past
the configured timeout plus margin, or any reliance on a constant other than
the configured `gracefulShutdownSeconds`.

### S9 — The control API is reachable through the real listener

**Observable.** The single `serve` listener serves both planes: a request
whose path begins with `/__fake/` reaches the control API, everything else
reaches the data plane. Through the ready file's `url`:

- `GET /__fake/v1/health` → `200` with the exact §41.2 body
  (`status`, `serverVersion`, `controlApiVersion`) and a JSON content type.
- `GET /__fake/v1/meta` → `200` with the exact §41.3 shape; `activeProfiles`
  preserves the effective configured order (§41.3) and `limits` echoes the
  effective values from S2.
- The remaining §11.1 endpoints are reachable at their defined paths and
  methods; exercised at minimum by one stateful call, e.g.
  `POST /__fake/v1/stubs` returning `201` with `id` and `registrationIndex`,
  which proves methods and bodies are forwarded through the seam rather than
  only `GET`.
- Control traffic does not touch the data plane: after several control
  requests (including one stateful mutation), `GET /__fake/v1/requests`
  contains no control record, and the next data-plane request is recorded with
  interaction sequence `1`. An unsupported control path
  (`GET /__fake/v1/nope`) returns the control-plane `404` with the
  control error body and likewise leaves the journal and verification state
  untouched.
- The control API is bound to the same host/port as the data plane and is
  therefore reachable only where the data plane is (§19.5).

**Spec.** §19.5 (the control API shares the same local server in v1), §11 and
§11.1 (control endpoints live under `/__fake/v1` and MUST NOT be journalled),
§41.1 (unsupported control paths return an ordinary control `404`, outside the
journal and verification state), §41.2, §41.3, §41.4, §11
(`Content-Type: application/json` for control JSON), §15.1 (ready file written
only after the control API is ready).

**Exercise.** All of these are ordinary HTTP requests against the ready
file's `url` using a loopback client; the journaling assertion reads
`GET /__fake/v1/requests` before and after the control calls and compares
counts, then issues `GET /v1/models` and asserts one recorded interaction with
sequence `1`. No listener is opened besides the child's, no fixed port is
used, and the readiness barrier is the ready file. Falsified by: a `404
fake_jev_control_not_found` (today's unwired behavior) or an unrouted control
path, by a control call appearing in the request history, or by `/meta`
reporting values that differ from the effective configuration.
`GET /__fake/v1/health` under S5's single-shot rule is the same observation
used as the readiness proof; the two criteria must agree.

### S10 — Operational logs: stderr destination and Authorization redaction

**Observable.**

- Operational log records for the serve process are written to **stderr**;
  stdout carries no operational log record (§42.5 allows shell-consumable
  result data on stdout for commands that have it; `serve` defines none, and
  the ready file is the readiness contract — §30.9). The wording of the lines
  is not a contract (§42.5: tests must not depend on human log wording except
  where the specification defines output); the examples in §20 are examples.
- With a data-plane request carrying `Authorization: Bearer <secret>` and with
  a control request carrying `Authorization: Bearer <secret>`, the secret
  string appears in **neither** stdout nor stderr at any point of the process
  lifetime, with `--log-format text` and with `--log-format json`. Redaction
  means the value must not appear (§20, §42.5); whether the header name is
  echoed with a placeholder is not specified.
- `--log-format` accepts `text` (default) and `json`; any other value is a
  usage failure (S1). Both accepted values keep logs on stderr and keep the
  secret out.
- For `--port 0` the chosen port is reported by the ready file (S4); the §20
  "listening on" line is an example, and the JSON record shape under
  `--log-format json` is deliberately **not** part of this item's contract
  (non-goal). Without `--ready-file`, §15.1/§20 define no machine-readable
  port report (observation U5).

**Spec.** §42.5 (human/JSON operational logs go to stderr; Authorization
header values MUST never be logged), §20 (observability; "Do not log secrets
by default"; "Authorization values must be redacted"), §15.1 (`--log-format`),
§30.9, §19.4/§20 (log-body truncation belongs to FJ-041).

**Exercise.** The child's stdout and stderr are captured into separate
buffers for the whole run. After readiness, the test issues (a) a data-plane
`POST /v1/systemone` and (b) a control request, both with the same
distinctive `Authorization` value (e.g. a 64-hex-character string generated by
the test); after the process has been signalled and reaped, assert
`!bytes.Contains(stdout, secret)`, `!bytes.Contains(stderr, secret)`, and
that stderr contains at least one non-empty line while stdout contains no log
record. The same run is repeated with `--log-format json`. Falsified by: any
occurrence of the secret in either stream, or operational records on stdout.

### S11 — Exit codes for `serve`

**Observable.**

| Situation | Code | Observable evidence |
|---|---|---|
| clean shutdown after `SIGINT`/`SIGTERM` | `0` | child exit status; ready file removed; port closed |
| CLI usage failure (unknown flag, bad flag value, stray positional argument) | `2` | stderr diagnostic + usage; nothing bound; no ready file |
| `--config` missing/unreadable, or configuration invalid under §12.2/§38 | `2` | stderr diagnostic; no listener left behind |
| startup failure: requested port not bindable, or `--ready-file` not creatable | `2` | stderr diagnostic; pre-existing ready file untouched; no listener left behind |
| internal operational failure while serving | `2` | stderr diagnostic |
| `verify`-style verification failure | `3` **is never produced by `serve`** | no `serve` input yields `3` |

`serve` with a valid configuration and a healthy listener does not exit on its
own: after readiness and after serving requests the process is still running
until a signal or a fatal error.

**Spec.** §42.1 (exit codes `0`, `2`, `3`), §42.3 (clean exit on signals),
§15.1/§15.2, §4.6 (fail closed), §14/§12.2 (`strict` only).

**Exercise.** Codes `2` are asserted from in-process runs for usage failures
and from child runs with separate stdout/stderr for config/startup failures;
`0` is asserted from the child exit status of the S8 runs; "never `3`" is
asserted by covering every input the table names and observing no `3`.
"Does not exit on its own" is asserted by polling the child's exit channel
with a bounded deadline after serving a request and observing no exit — a
bounded negative observation, not a sleep used as synchronization. Falsified
by: any code outside `{0,2}` observed, or a `serve` process that exits without
a signal or a fatal error.

### S12 — The server keeps serving while it runs

**Observable.** After readiness, repeated data-plane requests over separate
connections all receive complete HTTP responses (e.g. `GET /v1/models` →
`200` with the §44.1 default body), and the interaction journal records them
in ascending sequence order (§11.3), proving `serve` is a long-running server
rather than a one-shot that stops after the first request or after a fixed
number of connections. The listener's host/port stay those reported in the
ready file for the whole lifetime.

**Spec.** §3.1 (standalone local HTTP server), §11.3 (monotonic sequence
starting at `1`), §15.1, §22.4 (integration through ephemeral-port HTTP).

**Exercise.** Three sequential `GET /v1/models` requests, then
`GET /__fake/v1/requests`; assert three records with sequences `1,2,3` and
`outcome` `matched`. Falsified by: a dropped/refused request, an exit of the
child between requests, or a non-monotonic/duplicated sequence.

## Exercise strategy and non-flakiness rules

**H1 — Harness.** Anything that needs real signals, a real listener, real
stream separation, or a real exit status runs the compiled CLI as a child
process (build `./cmd/fake-jev` once per test binary into a `t.TempDir()`
path; `go test` caching may be used, no network). Pure usage-failure cases run
in-process against `cli.Run` with two buffers. No test binds a port for a
`serve` under test except the one default-port case, and no test scrapes log
wording.

**H2 — Synchronization points (no sleeps-as-sync).** Readiness is observed by
the appearance of the ready file; completion of shutdown by the child's exit
status through `cmd.Wait`; a request being actively read by the interim
`100 Continue` response; listener closure by a refused dial. Poll loops and
`Wait` calls carry bounded deadlines (they are failure bounds, not barriers),
and no test proceeds on the basis of elapsed time alone.

**H3 — Ports.** Every server under test binds `--port 0` (or config
`server.port: 0`) and is reached through the ready file. The two exceptions
are explicit and probed: the §12.2 default port `8787` for the default-port
leg, and a port already held by a test-owned `net.Listen("tcp","127.0.0.1:0")`
listener used to provoke the "port not bindable" startup failure. Neither case
picks a numeric port that the OS did not hand out (except the normative
default `8787`, which the test first probes).

**H4 — Network.** Loopback only: `127.0.0.1` and `127.0.0.2` (`127.0.0.0/8`).
No external host, no DNS, no egress (§19.2, §4.9).

**H5 — Timeouts.** Failure bounds are generous and independent of the
observable: process-exit waits bounded by `gracefulShutdownSeconds + 10 s`;
readiness/refusal polls bounded by 10 s; client HTTP timeouts bounded by 10 s.
No bound is tightened to make a timing bug observable, and no bound is used as
a synchronization primitive.

**H6 — Lifecycle hygiene.** Every spawned child is registered for cleanup that
signals `SIGKILL` if needed, reaps the process, and asserts the ready file
path and its parent directory hold no residue; leaked children or ready files
must fail the enclosing test rather than the next one.

**H7 — Explicit environmental preconditions.** Three sub-checks are
environment-dependent and are probed before use, then reported as exercised or
not exercised with the reason: (1) `127.0.0.1:8787` bindable by the test
itself (default-port leg); (2) `127.0.0.2` bindable (host-override legs);
(3) at least one non-loopback local interface address exists (S3's
public-bind sub-check). An unavailable probe never turns into a silent
non-assertion without a recorded reason, and never turns into a bound-port
assumption.

## Exit-code contract for `serve` (§42.1)

```text
0  clean shutdown after SIGINT/SIGTERM, with normal signal-driven state removal
2  CLI usage failure; configuration load/validation failure; startup failure
   (listener, ready file); control-API or internal operational failure
3  never produced by serve (reserved for verify reporting passed: false)
```

## Interpretations applied (with the alternative reading recorded)

- **I1 — signal exit code.** §42.3 ends with "exiting cleanly" and §42.1 maps
  success to `0`; §42.2 grants signal-derived exit codes (`130`/`143`-style)
  only to `run`, and `serve` has no such clause. S8 therefore expects `0`
  after a signal. Alternative reading (recorded, not adopted): `serve` could
  preserve signal-derived codes; §42 does not say so, and adopting it would
  contradict §42.1's `0 success` for a clean shutdown.
- **I2 — configured `port: 0`.** §15.1 defines `--port 0` as ephemeral and
  §38.2 permits `0` in the file. S2/S4 read a configured `0` as the same
  request, because the value is the listener's port. Alternative reading: a
  configured `0` is invalid for `serve`; §38.2 does not support that, since it
  permits the whole `0..65535` range.
- **I3 — ready-file pid guard, unparseable file.** "Remove it only if its
  parsed `pid` still equals the current process PID" (§42.4) means any file
  whose `pid` cannot be parsed or does not match is left untouched (S7).
  Alternative reading: an unparseable file is an error; §42.4 assigns no
  failure semantics, so no error is claimed.
- **I4 — no implicit configuration discovery.** §15.1 defines
  `--config <path>`; nothing in §12/§15 defines a default path or a
  cwd/DIG-style discovery rule, so with `--config` absent no file is
  consulted and the §12.2 defaults apply (S2 row "no implicit file
  discovery"). Alternative reading: an implicit `./fake-jev.yaml` when
  present; the specification is silent, so this artifact does not require it,
  and any such behavior would have to be a specification change.
- **I5 — flag-supplied profiles are validated like configured ones.** A
  `--compat` value is normalized (`jev` → `jev/v1`) and a duplicated value is
  a usage/configuration failure (`2`), because §12.1 makes the flag supply the
  effective compatibility list and §38.1 requires explicitly supplied
  compatibility lists to be non-empty and unique after normalization (S1/S2).
  Alternative reading: duplicates in flag input are tolerated; that would let
  an invalid effective configuration through, contrary to §4.6.
- **I6 — stray positional argument is a usage failure.** §15.1 lists only
  flags and §15 states v1 adds no interactive prompts; an unexpected positional
  argument therefore fails closed with `2` (S1). Alternative reading: ignore
  it; silently ignoring CLI input is the behavior §4.6 rules out.

## Recorded as undefined by §15.1/§42 (observations, not requirements)

- **U1 — no `serve --help` contract.** §15 defines `help` only as a general
  command; per-command help text is not specified.
- **U2 — log record content and JSON schema.** §20 gives human-log examples
  and `--log-format json`; it defines no record schema, field set, ordering,
  or timestamp format. Non-goal for this item.
- **U3 — startup-failure ordering.** Whether the ready file path is validated
  before or after binding is unspecified; both orderings yield exit `2`
  (S6/S11). Also unspecified: whether the parent directories of
  `--ready-file` are created (this artifact assumes they are not, so a
  missing parent is a startup failure under §42.1 rather than a silent
  success without a ready file).
- **U4 — ready-file `url` for wildcard hosts.** With `--host 0.0.0.0` the
  specification does not say whether `url`/`host` report `0.0.0.0` or a
  loopback substitute; §15.1's example only covers a concrete host. S2/S3
  therefore assert the reported host echoes the effective host, and no
  criterion depends on a dialable `url` for the wildcard case.
- **U5 — port reporting without `--ready-file`.** §15.1 defines the ready
  file as the machine-readable channel; with `--port 0` and no ready file the
  specification defines no machine-readable port report (the §20 "listening
  on" line is an example, and §30.9 warns against log scraping).
- **U6 — drain semantics at the boundary.** §42.3 says "allowing in-flight
  HTTP requests up to `gracefulShutdownSeconds`" and "force-closing remaining
  server work after the timeout"; whether a request whose bytes have not yet
  reached the handler counts as in-flight is not defined (S8b avoids the
  question by first proving the handler is reading, via the interim
  `100 Continue`).
- **U7 — signal exit timing relative to file removal.** §42.3 lists the
  removal as step 4 before "exiting cleanly"; no criterion asserts the instant
  of removal, only that the file is gone once the process has exited (S7).
- **U8 — Windows.** §42.3 requires the closest practical behavior on Windows
  without changing the public success/failure rules. All criteria here are
  exercised on the Unix targets of §23.1; the Windows path is not exercised by
  this item's tests and is not claimed.
- **U9 — `serve` stdout.** §42.5 allows shell-consumable result data on
  stdout but defines none for `serve`; S10 asserts only that no operational
  log record appears there, not that stdout is empty.
- **U10 — `serverVersion` value.** §41.2 requires `<semver>`; the
  development default in `internal/control` is not semver-shaped, and §41.2
  does not state that the value must equal the §15.5 product version. S5/S9
  require an exact shape and agreement between `/health`, `/meta`, and the
  ready file's `controlApiVersion`, and leave the version string's source to
  FJ-020/FJ-021 territory.

## Non-goals

- No `run` orchestration (FJ-032): no child processes, no `FAKE_JEV_URL`
  injection, no `--url-env`, no §42.2 child/verify precedence.
- No `verify` command (FJ-032) and no verification logic in the CLI.
- No JSON log format contract beyond accepting the flag (U2); no metrics,
  tracing, or telemetry backend.
- No control-plane authentication or authorization of any kind (§19.5 keeps a
  separate listener/auth mechanism deferred); no network exposure warning UI.
- No new flags beyond §15.1, no config-file discovery, no environment-variable
  configuration, no prompts, no stdin interaction.
- No changes to `internal/config`, `internal/engine`, `internal/compat`,
  `internal/control`, or their contracts; the control API is only *wired*,
  not modified.
- No log truncation/limit machinery or `internal/cli/logging.go` (FJ-041).
- No change to the data plane's routing, validation, matching, or failure
  bodies; the only host change is control-plane delegation.

## Traces

- C-CLI-004 — §15.1/§15.4: `serve --port 0` requests an ephemeral OS-assigned
  port with the chosen port reported (S2, S4; the §15.4 `run` half belongs to
  FJ-032).
- C-CLI-006 — §15.1/§42.4: the ready file is written atomically with the exact
  fields and removed on shutdown only when its `pid` still matches (S5, S6,
  S7).
- C-CLI-007 — §42.3: `serve` handles SIGINT/SIGTERM with graceful shutdown
  within `gracefulShutdownSeconds` (S8; the `run` forwarding half belongs to
  FJ-032).
- C-CLI-010 — §42.5/§20: operational logs go to stderr, consumable data may
  go to stdout, and Authorization values are never logged (S10).
- C-ARCH-006 — §19.1/§37: the default bind address is `127.0.0.1` (S2, S3).
- C-CFG-011 — §12.1: precedence is CLI flag > config file > default, with no
  environment override (S2, S11).
- C-CLI-001 — §42.1: exit codes `0` success, `2` usage/config/startup/
  control/internal, `3` verification failed, which `serve` never emits
  (S1, S11).
- C-CFG-004 — §38.1: the `jev` alias normalizes before duplicate detection and
  unknown profiles fail (S1, S2).
- C-CFG-007 — §12.2: omitted sections take the built-in defaults (S2, S9).
- C-CFG-008 — §12.2/§14: any mode other than `strict` fails (S1, S11).
- C-HOST-008 — §12.2/§19.4: default body, journal, log-preview, and shutdown
  limits match §12.2 (S2, S8c).
- C-CTRL-001 — §41.2: `GET /__fake/v1/health` returns the exact health shape
  with status 200, reached through the real listener (S5, S9).
- C-CTRL-002 — §41.3: `GET /__fake/v1/meta` returns the exact meta shape and
  `activeProfiles` preserves configured order (S2, S9).
- C-CTRL-011 — §11: every control JSON response uses
  `Content-Type: application/json` except 204 responses (S9).
- C-CTRL-006 — §41.7: `GET /__fake/v1/requests` returns the exact record shape
  in sequence-ascending order, used here to show control traffic is not
  journalled and data-plane sequences start at `1` (S9, S12).

This artifact records observable behavior and traceability; it declares no
implementation outcome.
