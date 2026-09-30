---
stage: qa
task: FJ-032
inputFingerprint: e520fc826889999e1e361d655d9a5f9fd80b6abf0e368376ab8390805481ca4b
outputFingerprint: e520fc826889999e1e361d655d9a5f9fd80b6abf0e368376ab8390805481ca4b
taskFingerprint: 387ddaf6229fbab020186f256ec6fac2558180842ed9dbb9cac9f97728080a5d
gitHead: 22c0704
generatedAt: 2026-09-30T17:39:50Z
author: worker/FJ-032-qa
---

# QA — FJ-032

This stage exercises the candidate revision `22c0704` with the FJ-032 change set
present as unstaged/untracked working-tree content. The artifact is evidence: it
records what was observed and the exact commands that produced the observations.
It declares no outcome; only `./scripts/verify-candidate` emits a result.

## Independent surface (not the in-package tests)

- `./cmd/fake-jev` was built once to a path outside the repository:

  ```text
  go build -o /private/tmp/fj032-qa/fake-jev ./cmd/fake-jev
  ```

- The probe is a self-contained Go program at `/private/tmp/fj032-qa/probe/main.go`
  (module `/private/tmp/fj032-qa/go.mod`, stdlib only, built with
  `GOWORK=off GOPROXY=off GOFLAGS=-mod=mod go build -o /private/tmp/fj032-qa/probe-bin ./probe`).
  It drives `/private/tmp/fj032-qa/fake-jev` as a subprocess for every case; it
  never calls `internal/cli` in-process.
- The probe hosts its own canned control endpoint (`net/http/httptest` on
  `127.0.0.1`) and re-invokes itself as the `run` child (`probe child dump|request|hold|trap`),
  so child environment, child exit codes, and signal handling are observed
  through real processes.
- Exact probe invocation:

  ```text
  FJ_REPO=/Users/mak/git/fake-jev-FJ-032 \
    /private/tmp/fj032-qa/probe-bin all \
    /private/tmp/fj032-qa/fake-jev /private/tmp/fj032-qa/work
  ```

  Full transcript: `/private/tmp/fj032-qa/probe-output.txt`. The run reported
  192 observations and 0 mismatches.
- No repository file was added or edited by this stage. `git status --short`
  shows only the pre-existing FJ-032 coder/hardener change set (plus
  `.agent/work/FJ-032/state.json`, already modified before this session) and this
  artifact. `./scripts/verify-candidate FJ-032` wrote its own report at
  `.agent/reports/FJ-032/report-20260930T173920Z.json`; nothing was staged
  (`git diff --cached --name-only` empty).
- Candidate fingerprint recomputed after all probing:
  `./scripts/candidate-fingerprint candidate` →
  `e520fc826889999e1e361d655d9a5f9fd80b6abf0e368376ab8390805481ca4b`, identical
  to the input fingerprint, so probing staled nothing.

## C-CLI-008 / §15.3 — `verify` is one control call and duplicates no verification logic

- One `verify --url <canned-base>` produced exactly one recorded request,
  `GET /__fake/v1/verify` with `bodylen=0`.
- Run from a working directory that contains a deliberately invalid
  `fake-jev.yaml` (`schemaVersion: 1\nmode: loose\n`), a canned
  `{"passed":true,"failures":[]}` still yielded exit `0` with empty stderr: no
  configuration file was consulted by the verdict.
- Divergence leg A — body
  `{"passed":true,"failures":[{"code":"unmatched_request","message":"would be a failure if computed","requestSequence":3,"stubId":null}]}`
  → exit `0` (`fake-jev: <url>: verification passed` on stdout). A CLI that
  computed failures from the `failures` array would have exited `3`.
- Divergence leg B — body `{"passed":false,"failures":[]}` → exit `3` with
  `stderr` showing `verification failed`. A CLI that required the failures array
  to be non-empty would not have exited `3`.
- A real `serve --port 0 --ready-file` child was driven: `GET <url>/nope` → 404
  `fake_jev_unknown_route`; `verify --url <url>` → exit `3`, stderr
  `unknown_route: Request used an unknown route. (requestSequence 1)`;
  `DELETE <url>/__fake/v1/requests` → 204; the next `verify` → exit `0`; the
  journal from `GET <url>/__fake/v1/requests` was `{"requests":[]}` — neither
  `verify` call appeared in the data-plane journal (control read, §11.4).

## C-CLI-001 / §42.1 — `verify` exit codes and stream split

All against the built binary as a subprocess:

| Input | Observed exit | stdout | stderr |
|---|---|---|---|
| canned `{"passed":true,"failures":[]}` | 0 | `fake-jev: <url>: verification passed` | empty |
| canned `{"passed":false, …}` (two §41.9 items) | 3 | empty | `… verification failed`; then `unmatched_request: Request #2 did not match any configured stub. (requestSequence 2)` and `expect_exactly: stub s1 expected exactly 1 request (stubId s1)` |
| canned `{"passed":false,"failures":[]}` | 3 | empty | `… verification failed` |
| canned 500 / 404 / 204 / `not json` / `{}` / `{"passed":"no"}` / `null` / `[]` | 2 (each) | empty (each) | a diagnostic (each); e.g. `unexpected status 500`, `invalid character 'o' in literal null`, `response carries no boolean "passed" field` |
| unreachable loopback port (listener opened then closed) | 2 | empty | `dial tcp 127.0.0.1:<port>: connect: connection refused` |
| usage: no `--url`; `--url` without value; `notaurl`; `http://example.invalid/x`; stray positional; unknown flag | 2 (each) | empty (each) | diagnostic + usage text (each); the canned endpoint recorded **0** requests for every usage row |

So the exit is keyed on the body's `passed` boolean: `true` → 0, `false` → 3,
anything the CLI cannot obtain or decode → 2. The §41.9 failure items on stdout
were absent in every row; stream split matches §42.5.

## C-CLI-002 / §44.13 / C-GOLD-013 — `run` exit precedence (every row, real processes)

Each row ran `run --config <config> -- <probe> child <mode> …`; "verify pass" is
a child that makes no failing interaction, "verify fail" is a child that performs
one `GET <url>/nope` (→ 404 `fake_jev_unknown_route`).

| Row | Child | Observed exit | stderr observations |
|---|---|---|---|
| child 0 + verify pass | `dump … 0` | 0 | contains no `verification`; `fake-jev: GET /__fake/v1/verify` (serve request log only) |
| child 0 + verify fail | `request … /nope 0` | 3 | contains `verification` and `unknown_route` |
| child 7 + verify pass | `dump … 7` | 7 | contains `child exited with code 7`; no `verification` |
| child 7 + verify fail | `request … /nope 7` | 7 | contains **both** `verification` (and `unknown_route`) and `child exited with code 7` |
| startup failure (missing `--config`) | — | 2 | `read configuration "…/absent.yaml": … no such file or directory` |
| child cannot be started (missing binary) | — | 2 | `start …: fork/exec …: no such file or directory` |

After the child 7 + fail row the server recorded in the child snapshot was
already refused (`dial … connect: connection refused`), so the listener is
closed even for a nonzero child. The C-GOLD-013 token `verification` for the
`child 7 + verify fail` row appeared literally in stderr, and the child's code
was named separately (`child exited with code 7`).

## C-CLI-003 / §15.4 — `FAKE_JEV_URL` and `--url-env`

Parent environment carried sentinels `FAKE_JEV_URL=http://127.0.0.1:1` and
`FJ032_URL_ALT=stale-value`; the child wrote a snapshot of its environment.

- `run --config … --url-env FJ032_URL_ALT -- <child>` → exit 0; child's
  `FAKE_JEV_URL` = `http://127.0.0.1:<os-port>` (loopback, port > 0), differing
  from the inherited sentinel; `FJ032_URL_ALT` equalled the same URL and differed
  from `stale-value`. Both entries were replaced.
- `run --config … -- <child>` (no `--url-env`) → exit 0; child's `FAKE_JEV_URL`
  was present and loopback and differed from the inherited sentinel.
- A child that performed one `GET <FAKE_JEV_URL>/v1/models` observed status
  `200` on the first attempt (no retry in the helper).

## C-CLI-004 / §15.4 + F1 host ruling

- The probe held its own `net.Listen("127.0.0.1:0")` and wrote that port into
  `server.port`; `run --config <that file>` → exit 0 and the child URL port
  (65210) differed from the held port (65209) — the file value never reached the
  listener.
- `run --port 0` → exit 0 with an OS-assigned child port.
- `run --config <file=held> --port <free>` → exit 0 and the child URL port
  equalled the `--port` value (65214), with the file naming a different
  (held) port: the flag won.
- `run --config … --port <held>` → exit 2, empty stdout, stderr
  `listen on 127.0.0.1:65209: … bind: address already in use`; no child
  snapshot was written, and the held listener accepted the probe's dial
  afterwards (untouched).
- F1 ruling, independent probe: config `server.host: 0.0.0.0` with
  `server.port: 0`; the child's `FAKE_JEV_URL` host was `127.0.0.1`; a dial to
  `127.0.0.1:<port>` connected; a dial to this host's non-loopback IPv4
  `192.168.1.46:<port>` was **refused** (`connect: connection refused`) — the
  listener was not wildcard-bound. The child (`hold` mode) was still alive and
  `run` still draining 500 ms after SIGTERM, then `run` exited 143 and the child
  pid was gone.

## C-CLI-007 / §42.3 — signal forwarding and child lifecycle

Config `limits.gracefulShutdownSeconds: 2`.

- `trap` child (handles the forwarded signal, exits 0): SIGTERM to `run` →
  child wrote `terminated`; `run` exited `143`; SIGINT to another `run` → child
  wrote `interrupt`; `run` exited `130`. In both cases the child pid was gone
  after `run` exited and the child-recorded URL was refused
  (`connect: connection refused`).
- Force-terminate (`hold` child that ignores SIGINT/SIGTERM): SIGTERM to `run` →
  child and `run` both still alive after 500 ms; `run` then exited after a
  further ~1.5 s (2.0 s total, the configured grace), and the child pid was gone.
  Total `run` termination stayed inside `gracefulShutdownSeconds + 5 s`.
- Process residue check after the whole probe: `ps -axo pid,ppid,stat,command |
  grep -E "fj032-qa|fake-jev"` returned nothing. The only `Z` (defunct) rows on
  the host are children of the terminal emulator (`ppid 51594` = `kitty`) and
  predate this run; no fake-jev/helper zombie remained.
- Process-group forwarding limit, independently observed: a shell child started
  `sleep 300` (a grandchild) then trapped TERM and exited. SIGTERM to `run`
  produced `received terminated; forwarding to …/child.sh`, `run` exited 143,
  and the grandchild was still alive afterwards (it was killed by the probe for
  cleanup). Forwarding therefore targets the direct child only.

## Serve unchanged by the lifecycle refactor

- `serve --port <held>` → exit 2; stdout empty; stderr
  `listen on 127.0.0.1:<port>: … bind: address already in use`.
- `serve --config <missing>` → exit 2; stdout empty; stderr names the file.
- `serve --port 0 --ready-file <path>` → ready document published with exactly
  five keys; `host=127.0.0.1`, `port` OS-assigned, `pid` equal to the serve
  process pid, `controlApiVersion=v1`; `GET <url>/__fake/v1/health` → 200; stdout
  empty throughout.
- SIGTERM → exit 0; stderr carried only
  `listening on <url>`, the health request line, and
  `received terminated; shutting down`; stdout empty; the ready file was absent
  after shutdown.

## Commands run and outcomes

```text
go test ./internal/cli -count=1     -> ok fake-jev/internal/cli 19.719s, exit 0
go test ./... -count=1              -> ok cmd/guard 8.280s; internal/cli 23.328s;
                                       internal/compat/jev/v1 1.247s; config 2.275s;
                                       control 2.817s; engine 1.782s; host/http 3.803s;
                                       test/integration 3.319s; cmd/fake-jev no test files; exit 0
go test -race ./... -count=1        -> ok cmd/guard 9.562s; internal/cli 24.184s;
                                       compat/jev/v1 3.006s; config 1.980s; control 3.599s;
                                       engine 4.570s; host/http 5.076s; integration 4.057s; exit 0
go vet ./...                        -> no output, exit 0
go run ./cmd/guard arch             -> {"findings":[],"stats":{"files":56,"packages":9}}, exit 0
go run ./cmd/guard lint             -> {"findings":[],"stats":{"files":56,"packages":9}}, exit 0
go run ./cmd/guard trace            -> {"findings":[],"stats":{"active":0,"covered":17,"test_files":20}}, exit 0
go run ./cmd/guard fuzz             -> {"findings":[],"stats":{"targets":0}}, exit 0
./scripts/verify-candidate FJ-032   -> exit 0; toolline "summary: 20 passed, 0 failed,
                                       0 skipped, 0 not_applicable", "RESULT: PASS";
                                       report .agent/reports/FJ-032/report-20260930T173920Z.json
./scripts/candidate-fingerprint candidate
                                    -> e520fc826889999e1e361d655d9a5f9fd80b6abf0e368376ab8390805481ca4b
```

`git diff --cached --name-only` is empty (nothing staged).

## Evidence limits and residual risks

- **Process-group forwarding is not implemented.** `run` forwards to the direct
  child process; no `SysProcAttr{Setpgid: true}` is present. The grandchild probe
  above shows a grandchild outliving `run` in that scenario. This stage's signal
  observations only cover a direct child.
- **Readiness is structural, not probed.** §15.4 defines no `--ready-file` for
  `run`; the listener is bound and the control API attached before the child is
  started. The evidence is that the helper's single first request to
  `FAKE_JEV_URL` returned 200 with no retry; a `Serve` goroutine error after
  `startServer` returns is not separately inducible from outside.
- **C-GOLD-013's `kind: "cli"` rows have no vector runner.** No Go test reads
  `testdata/contracts/cli/01-run-exit-precedence.json`
  (`grep -rn "testdata/contracts/cli\|01-run-exit-precedence" --include=*.go`
  returned nothing); the rows are pinned behaviourally by
  `TestRunExitPrecedence` and by this probe. The vector file's `kind` is `cli`
  (read by the probe).
- **`verify` timeout row exercised at the transport/command level only via an
  unreachable port**, not by a server that accepts and never answers within the
  10 s bound; the command-level timeout mapping (`verificationTimeout` → 2) was
  read from the code, not observed in this pass.
- **The unreachable-port case is a race** (a closed ephemeral port could be
  re-bound); on this host the dial was refused as expected.
- **Signal semantics assume Unix.** All signal observations used POSIX
  SIGINT/SIGTERM and `kill(pid, 0)`; Windows is only cross-compiled by other
  stages and was not exercised here.
- **`serve` behavior was re-observed only on the success and two failure paths
  above**; the ready-file pid-guard/overwrite paths and Authorization redaction
  were not re-driven in this pass (they are covered by the unmodified FJ-031
  tests, which ran inside `go test ./internal/cli`).
- **`verify` response body has no byte cap** (`io.ReadAll`); the 10 s client
  timeout bounds wall time, not bytes. Not probed for a large-body effect.
- **Tooling exemptions still apply**: G-Q's public-surface harness is
  `stage.tooling_bootstrap_exempt` (`.agent/gate-policy.json`, tracked by FJ-046),
  and G-L/G-H tooling is likewise exempt (FJ-044/FJ-045); this stage therefore
  hand-rolled its surface harness under `/private/tmp/fj032-qa`.
- Cleaner/hardener findings not re-adjudicated here (T2/T3 prose-vs-vector
  strength, F2 signal window, F4 `serveErr` surfacing) are recorded in their own
  artifacts; this stage only confirms the observable surfaces listed above.
