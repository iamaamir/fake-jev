---
stage: qa
task: FJ-031
inputFingerprint: c8ee98ed15771f58cb8cecedd751633ac6c7bc4414a294da0db14807fda43338
outputFingerprint: c8ee98ed15771f58cb8cecedd751633ac6c7bc4414a294da0db14807fda43338
taskFingerprint: 703164b4cd18ba33de0272b4065a7861e596afa85da13bcc16fd751fda500e6a
gitHead: 6094965
generatedAt: 2026-09-30T16:10:15Z
author: worker/FJ-031-qa
---

# QA — FJ-031: `fake-jev serve`

Fresh session. Coder author `worker/FJ-031-coder`, hardener author
`worker/FJ-031-hardener`; this author (`worker/FJ-031-qa`) is neither, so the
independent-QA author differs from the coder's. Candidate fingerprint
`c8ee98ed15771f58cb8cecedd751633ac6c7bc4414a294da0db14807fda43338` was
re-read before and after every probe run and every repository command; it did
not change. This stage modified no production code, no repository test, and
not `.agent/work/FJ-031/state.json`.

Everything below is observed, not asserted from the code or from the earlier
stage artifacts; where a check is not externally exercisable it is recorded as
an evidence limit instead of a claim.

## Probe

- Source: `/private/tmp/fj031-qa/main.go`, module `/private/tmp/fj031-qa/go.mod`
  (standard library only). Nothing was written inside the repository.
- Binary: built from the candidate revision with
  `go build -o /private/tmp/fj031-qa/fake-jev ./cmd/fake-jev`, then driven
  only as a subprocess by the probe.
- Run: `cd /private/tmp/fj031-qa && go run .` (whole run: go1.26.3, darwin/arm64).
  Full transcript kept at `/private/tmp/fj031-qa/probe.log`.
- The probe builds a fresh `serve` child per observation, passes `--port 0` and
  an absolute `--ready-file` in a fresh directory, reads stdout/stderr into
  separate capture files, and inspects them after `Wait`.
- Guard rails: no fixed port is chosen except the normative §12.2 default
  `8787` leg, which is first probed with `net.Listen` and skipped if held (it
  was free in this run); the "port not bindable" startup case uses a listener
  the probe itself opened on `127.0.0.1:0`. No external host, no DNS, no
  network egress.

### Timing / synchronization method

No sleep is used as a barrier. Synchronization points:

- readiness = the appearance of the ready file (readiness barrier), polled with
  a 10 s failure bound; the sampler also validates each observed sample;
- "a request is actively being read" = the interim `HTTP/1.1 100 Continue`
  written by the host when it first reads the request body (the probe sends
  only the first body byte and waits for the interim response);
- "new accepts stopped" = a dial to the bound authority is refused, polled
  with a 10 s failure bound;
- shutdown complete = the child's exit status through `Wait`, with a
  `gracefulShutdownSeconds + 10 s` failure bound;
- the configured-grace lower bound = a bounded negative observation of the
  wall time to exit (not a sleep used to sequence anything).

The readiness sampler polls by re-reading the whole file and calling
`runtime.Gosched()`, with 10 s failure bounds — no wall-clock gap is a
synchronization primitive.

## C-CFG-011 (§12.1) — precedence flag > file > default, no env override

All rows driven through the built binary; the effective value is read back from
the ready file (`host`, `port`, `url`) and/or `GET /__fake/v1/meta`.

| Direction | Input | Observed |
|---|---|---|
| neither (built-in default) | `serve --ready-file …` | ready `host=127.0.0.1 port=8787`; a dial to `127.0.0.1:8787` succeeded |
| file over default (port) | config `{"server":{"port":0}}` | ready `host=127.0.0.1 port=61493`, i.e. the file value, not the default `8787` |
| file over default (host) | config `{"server":{"host":"::1","port":0}}` | ready `host="::1" url="http://[::1]:61496"`; dial `[::1]:61496` succeeded, dial `127.0.0.1:61496` refused |
| file over default (limits) | config `limits.maxInteractions: 7` | `/meta` → `200` with `limits.maxInteractions=7` |
| flag over file (port) | config `{"server":{"port":8788}}` + `--port 0` | ready `port=61503` (neither `8788` nor default `8787`) |
| flag over file (host) | config `host:"::1"` + `--host 127.0.0.1` | ready `host="127.0.0.1"`; dial `[::1]:61504` refused |
| `--compat` replaces file list | config `compatibility:["jev/v1"]` + `--compat jev` | `/meta` `"activeProfiles":["jev/v1"]` (alias normalised) |
| environment ignored | `FAKE_JEV_HOST=0.0.0.0 FAKE_JEV_PORT=9 FAKE_JEV_CONFIG=… FAKE_JEV_READY_FILE=… FAKE_JEV_LOG_FORMAT=json HOST=0.0.0.0 PORT=9` | ready `host=127.0.0.1 port=61508`; the env-named ready path did not appear |
| no implicit discovery | cwd containing `fake-jev.yaml` with `host:"::1" port:8788`, `serve --port 0` without `--config` | ready `host=127.0.0.1` (the cwd file was not consulted) |

Every disagreeing row makes the losing source's value an address that is
observably not bound, so the winner is shown by the socket, not only by a
reported string. Observation: every leg moved in the expected direction in all
21 observed ready-file publishes; no leg reported the losing source's value.

## C-ARCH-006 (§19.1, §37) — default bind is loopback

- Default run: ready `host="127.0.0.1"`, `net.ParseIP(host).IsLoopback()` true,
  `url="http://127.0.0.1:61487"`.
- Non-loopback sub-check **exercised**: this host has non-loopback IPv4
  `192.168.1.46`; dial `192.168.1.46:61487` was refused while the default host
  was loopback, so the listener was not on the wildcard/any address.
- Explicit host is honoured: `--config host:"::1"` (and separately file host
  overridden by `--host ::1` in the flag-vs-file leg) bound only `::1`: dial
  `[::1]:<port>` succeeded, dial `127.0.0.1:<port>` refused.

## C-CLI-004 (§15.1) — ephemeral port reported

- `--port 0`: ready `port=61487`, `url` authority port `= 61487`, and one
  `GET http://127.0.0.1:61487/__fake/v1/health` → `200` on the first attempt
  after the file appeared (no retry loop).
- A second `--port 0` run reported `61490 ≠ 61487`; each reported port accepted
  a connection at exactly the reported authority.
- A configured `server.port: 0` behaves the same (the "file over default"
  row reported port `61493`). No run reported port `0`.

## C-CLI-006 (§15.1, §42.4) — atomic ready file, exact fields, pid guard

21 readiness observes across the run; **every** observation ended in exactly one
complete document, with zero empty, truncated, or unparseable samples.

- Exact document: key set exactly `{url, host, port, pid, controlApiVersion}`
  (the probe rejects any sample whose JSON object does not have exactly five
  keys). `pid` equalled the child's `cmd.Process.Pid`. `controlApiVersion`
  `"v1"`, and equal to `/health`'s `controlApiVersion`. `host`/`port`/`url`
  described the live socket (a health request through `url` answered `200`).
- Never partial: across all runs the sample sequence was
  `absent…absent, fresh=1` (e.g. `absent=12621 fresh=1`) with no partial sample.
- Overwrite only after readiness: with a pre-existing foreign document
  (`{"pid":999999,"sentinel":true}`) the sampler observed `foreign=287 fresh=1`
  — only the complete foreign document or the complete new document, never a
  mixture; after readiness the path held the new five-key document.
- No temp residue: after readiness the destination directory held no
  `.fake-jev-ready-*` file. An independent manual run confirmed the directory
  held only `ready.json` while serving, and only `.`/`..` after shutdown.
- Removal with matching pid: after clean `SIGTERM` the ready file was absent
  (`os.Stat` → `ErrNotExist`) in every clean-shutdown run.
- Foreign pid survives: the probe overwrote the file with a five-key document
  whose `pid` was `childPid+424242`; after shutdown the bytes were byte-identical
  to the replacement.
- Unparseable survives: overwriting with `not json at all`; after shutdown the
  bytes were unchanged.
- Startup failure leaves a pre-existing ready file untouched: with a reserved
  port (`net.Listen("tcp","127.0.0.1:0")` held by the probe) the child exited
  `2` and the pre-existing foreign document was byte-identical afterwards.

## C-CLI-007 (§42.3) — SIGINT/SIGTERM graceful shutdown with in-flight drain

For each of `SIGINT` and `SIGTERM`, with the interim `100 Continue` proving the
handler was reading the request body at signal time:

| Step | SIGINT | SIGTERM |
|---|---|---|
| interim response seen | `HTTP/1.1 100 Continue` | `HTTP/1.1 100 Continue` |
| new accepts refused after signal | 50 ms | 0 ms |
| in-flight request response | `501 Not Implemented` + §43.1 body | `501 Not Implemented` + §43.1 body |
| exit status / elapsed after signal | `0` / 71 ms | `0` / 2 ms |
| ready file afterwards | removed | removed |

Configured grace is read from configuration, not a constant: with
`limits.gracefulShutdownSeconds: 1` and a request whose body is never
completed, the process stayed alive for `1.004 s` and then exited `0`
(a hardcoded default of `5` would have produced ≈5 s). The held connection
ended without any response bytes (`readBytes=0`, read returned `nil` = EOF, not
a client timeout).

## C-CLI-010 (§42.5, §20) — stderr-only logs, Authorization never logged

Two runs, `--log-format text` and `--log-format json`, each issuing control and
data-plane requests carrying `Authorization: Bearer <64-char secret>`,
including error paths: control invalid JSON (`400`), control oversized body
(`413`), unsupported control path (`404`), data-plane validation error (`422`),
data-plane unknown route (`404`), plus matched `/v1/models` and control
`/health`.

| Observation | text | json |
|---|---|---|
| stdout bytes | 0 | 0 |
| stderr bytes | 469 | 469 |
| secret present in stdout | no | no |
| secret present in stderr | no | no |
| redacted marker present | `authorization=[redacted]` | `authorization=[redacted]` |

stderr carried the operational lines (`listening on …`, then one line per
request, method + path + `authorization=[redacted]`), including the lines for
the error-path requests above.

## Exit codes (§42.1) — 0, 2, never 3

Observed exit codes across the whole probe run: `{0: 20, 2: 13}`. Code `3` was
never observed.

- `0`: every clean signal shutdown (drain, force-close, default-bind,
  precedence, ready-file, control, and logging runs).
- `2`: usage failures — `--bogus`, `--port 70000`, `--port abc`,
  `--mode loose`, `--log-format xml`, `--compat unknown/v1`, a stray
  positional, `--host ""`, and `--ready-file ""`. Each wrote a diagnostic plus
  usage to stderr, wrote 0 bytes to stdout, and created no ready file.
- `2`: configuration failures — missing `--config` file and an invalid config
  (`mode: loose`). Both named the offending file in the stderr diagnostic and
  wrote 0 bytes to stdout.
- `2`: startup failures — port already held by the probe (`listen tcp
  127.0.0.1:…: bind: address already in use`) and a missing parent directory
  for `--ready-file`; the pre-existing ready file was untouched.
- Longer-lived runs (control, logging, drains) served requests and then stayed
  alive until the signal; none exited between requests.

## Control API reachability through the real listener (§19.5, §41, §43)

All through the ready file's `url` on one child process:

- `GET /__fake/v1/health` → `200`, `Content-Type: application/json`,
  `{"status":"ok","serverVersion":"0.0.0-dev","controlApiVersion":"v1"}`
  (exactly three keys; `controlApiVersion` equal to the ready file).
- `GET /__fake/v1/meta` → `200`, `mode:"strict"`, `activeProfiles:["jev/v1"]`,
  `limits` equal to the §12.2 values (8388608 / 2097152 / 10000 / 4096 / 5).
- `GET /__fake/v1/stubs` → `200 {"stubs":[]}`; `POST /__fake/v1/stubs` with a
  valid stub → `201 {"id":"qa-dynamic","registrationIndex":1}`; the next list
  contained the dynamic stub (`"source":"dynamic"`, `registrationIndex:1`,
  `invocations:0`). This shows methods and bodies are forwarded through the
  seam, not only GET.
- Unsupported control path `GET /__fake/v1/nope` → `404` with the control body
  `{"error":"fake_jev_control_not_found","message":"Unknown control endpoint."}`.
- Invalid control JSON → `400`; oversized control body (> 2 MiB) → `413`.
- Control-plane requests are not journaled: after the health/meta/stubs/404/
  400/413 calls, `GET /__fake/v1/requests` → `{"requests":[]}`.
- The data plane over the same listener: `GET /v1/models` → `200` with the
  §39.1 default body; `POST /v1/systemone` with no matching stub → `501` with
  the exact §43.1 body (`error/message/profile/operation`); `GET /v1/nope` →
  `404` with the §43.2 body. The journal then held exactly three records with
  sequences `1,2,3` (`matched`, `unmatched`, `unknown_route`).

## Repository commands run (from `/Users/mak/git/fake-jev-FJ-031`, go1.26.3)

| Command | Result |
|---|---|
| `go test ./internal/cli -count=1` | exit 0 (`ok fake-jev/internal/cli 11.258s`) |
| `go test ./internal/host/http -count=1` | exit 0 (`ok … 0.516s`) |
| `go test ./... -count=1` | exit 0 (all 9 packages `ok`; 15.4s) |
| `go test -race ./... -count=1` | exit 0 (all packages `ok`; 17.7s) |
| `go vet ./...` | exit 0, no output |
| `go run ./cmd/guard arch` | exit 0, `findings: []` (52 files, 9 packages) |
| `go run ./cmd/guard lint` | exit 0, `findings: []` |
| `go run ./cmd/guard trace` | exit 0, `findings: []` (17 covered, 18 test files) |
| `go run ./cmd/guard fuzz` | exit 0, `findings: []` (0 targets) |
| `./scripts/candidate-fingerprint candidate` | `c8ee98ed…da43338` |
| `./scripts/verify-candidate FJ-031` | exit 0; report written to `.agent/reports/FJ-031/report-20260930T160850Z.json` |

Side effect disclosed: `verify-candidate FJ-031` wrote that report artifact.
The probe's own run `exit status` was 0 with `mismatches: 0`.

## Residual risks and evidence limits

- **Fatal `serveErr` arm is not externally reachable.** `http.Server.Serve`'s
  error cannot be provoked through the process boundary (the listener is local
  to `serve`), so the hardener's guarantee that the deferred ready-file removal
  runs on that arm was not exercised through the built binary; the probe
  exercised only the clean-shutdown path that shares the same defer. The
  guarantee rests on the defer being unconditional in the source.
- **nil-control fallback is not externally distinguishable.** When no control
  handler is attached, `serveControl` answers `404 fake_jev_control_not_found`
  with the same message as `internal/control`'s own unsupported-path `404`
  (`handlers.go:31`). The probe's `404` observation therefore cannot, by itself,
  show which path served it; the reachability conclusion rests on the
  endpoints only the delegated API can answer (`/health`, `/meta`, stateful
  `/stubs`). With `serve`, `AttachControl` is always called, so the fallback is
  not reachable from the shipped command.
- **Drain observations depend on the interim `100 Continue`.** The two drain
  observations and the configured-grace observation use the host's interim
  `100 Continue` as the "handler is reading the body" primitive. If a future
  host change removes that interim response, those observations lose their
  synchronization point (the specifier recorded a test-only fallback seam for
  the in-repo tests; this external probe has no such seam).
- **Default-port leg is environment-dependent.** The `8787` row asserted
  `host=127.0.0.1 port=8787` and a successful dial only after probing that the
  probe itself could hold `127.0.0.1:8787`; it was free in this run. A
  concurrent process taking `8787` would make that row unexercisable here.
- **`127.0.0.2` is not bindable on this macOS** (`bind: can't assign requested
  address`), so host precedence was exercised with `::1` vs `127.0.0.1` rather
  than the specifier's `127.0.0.2` example. Both directions remained
  socket-observable (`[::1]` refused / accepted as expected).
- **Single platform.** All observations are darwin/arm64, IPv4 loopback
  (`127.0.0.1`), IPv6 loopback (`::1`), and one non-loopback IPv4
  (`192.168.1.46`). The Windows path of §42.3 and any non-Linux/BSD behavior
  are not exercised.
- **"stdout is empty" is a stronger observation than §42.5 requires.** §42.5
  permits shell-consumable result data on stdout (specifier U9); the probe
  observed 0 stdout bytes, which does not by itself rule out a future
  legitimate stdout line.
- **`serverVersion` is `"0.0.0-dev"`.** The probe checked the §41.2 shape and
  the agreement between `/health`, `/meta`, and the ready file's
  `controlApiVersion`; it did not check semver shape (§41.2 `<semver>`, recorded
  as specifier U10).
- **"never exit code 3" is a negative claim** bounded by the inputs exercised
  (usage, config, startup, drain, force-close, control, logging); it is not a
  proof over all possible inputs.
- **Log wording is not a contract** (§42.5); the probe asserted stream
  destination, emptiness of stdout, absence of the secret, and the presence of a
  redaction marker, not any particular wording.
- **`--log-format json` record shape** is a non-goal (specifier U2); both
  formats were observed to keep logs on stderr with the secret absent, and no
  JSON schema was asserted.

## Files touched by this stage

Only `.agent/work/FJ-031/qa.md` (this file). No production code, repository
test, `state.json`, or other artifact was edited. `git status` shows no staged
files; the `M` on `.agent/work/FJ-031/state.json` and the untracked
`internal/cli/serve.go`, `internal/cli/serve_test.go`, `internal/cli/dispatch.go`
(modification) and `internal/host/http/server.go` (modification) predate this
stage.
