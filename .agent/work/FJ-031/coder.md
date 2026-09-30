---
stage: coder
task: FJ-031
inputFingerprint: bad76f7917d526d0978fb613aee212f2345ff06a717259788d4a04366f01ab19
outputFingerprint: 969f6a5c976687fa98b89b4616a4622420c81a50470f4116d15ab1d3e71769b4
taskFingerprint: 703164b4cd18ba33de0272b4065a7861e596afa85da13bcc16fd751fda500e6a
gitHead: 6094965
generatedAt: 2026-09-30T15:42:16Z
author: worker/FJ-031-coder
---

# Coder — FJ-031

**Final candidate fingerprint: `969f6a5c976687fa98b89b4616a4622420c81a50470f4116d15ab1d3e71769b4`**

## Scope delivered

Implemented `fake-jev serve`, wired the control API through the real listener,
and made the §12.1 configuration precedence an explicit, directly testable
seam. Only the four allowed files changed.

## What changed, per acceptance criterion

### C-CFG-011 — precedence CLI flag > config file > built-in default (§12.1)

`internal/cli/serve.go` builds the chain as two functions:

- `defaultConfig()` returns the §12.2 defaults by running
  `config.Load([]byte("schemaVersion: 1\n"))`, so no default is re-spelled.
- `applyServeFlags(cfg, flags)` overlays only the flags that `flag.FlagSet.Visit`
  reported as *present*. Presence, not a zero sentinel, is the discriminator,
  so `--port 0` (ephemeral) is distinguishable from an unset `--port`, and a
  file that omits `server` is distinguishable from the default.
- `resolveServeConfig(flags)` = defaults (or `config.LoadFile(--config)`),
  then `applyServeFlags`, then `config.Validate` (the same §38 validator a
  file passes through; it also normalizes `jev` → `jev/v1`), then a fail-closed
  non-empty-host check.

No environment variable is read anywhere in the serve path. `TestServeConfigPrecedenceSeam`
exercises every leg directly (defaults; file over default host/port/limits;
flag over file port; flag over file host; flag-port-zero-is-not-default;
configured `port: 0`; flag compat replacing the file list; and the failure
legs: empty host, non-strict mode, out-of-range file port).
`TestServePrecedenceEndToEnd` re-checks the same directions against real child
processes, including "no implicit `./fake-jev.yaml` discovery" and "the
environment is ignored".

### C-ARCH-006 — default bind is loopback (§19.1, §30.8, §37)

The effective `server.host` defaults to `127.0.0.1`; the ready file reports it;
`TestServeDefaultBindIsLoopback` asserts loopback, the URL authority, and that
every non-loopback local IPv4 address refuses a dial on the bound port (that
sub-check is reported as not exercised when the host has no such interface,
per the specifier's H7).

### C-CLI-004 — ephemeral ports (§15.1)

`--port 0` (and a configured `port: 0`, interpretation I2) is passed straight
to `net.Listen`; the *bound* `*net.TCPAddr.Port` is what the ready file and URL
report. Two concurrent servers are asserted to receive different ports and to
be dialable at exactly the reported authority (`TestServeEphemeralPortsAreReported`).

### C-CLI-006 — atomic ready file (§15.1, §42.4)

`writeReadyFile` uses `os.CreateTemp(dir, ".fake-jev-ready-*")` in the
destination directory, writes, closes, then `os.Rename`s over the target; any
error path removes the temporary and joins the cleanup error. It runs only
after the listener is serving and the control API is attached, so it is never
published before readiness. `removeReadyFile` removes the file only when it
still parses and its `pid` equals `os.Getpid()`; a foreign-pid or unparseable
file is left byte-identical. Covered by `TestServeReadyFileIsAtomicAndClean`
(every observed sample equals the complete foreign or complete new document; no
temp residue; removal on shutdown), `TestServeStartupFailureLeavesReadyFileUntouched`,
`TestRemoveReadyFilePidGuard`, and `TestServeReadyFilePidGuardOnShutdown`.

### C-CLI-007 — graceful shutdown on SIGINT/SIGTERM (§42.3)

`serve` installs `signal.Notify` before publishing readiness, then
`http.Server.Shutdown` with a context bounded by the configured
`limits.gracefulShutdownSeconds`; on timeout it force-closes remaining work
with `http.Server.Close`, removes the owned ready file, and returns exit `0`.
Signals exercised: `TestServeSignalShutdown` (SIGINT + SIGTERM, exit 0, ready
file and directory clean, port refused), `TestServeGracefulShutdownDrainsInflight`
(a request actively reading its body finishes with its 501 while new accepts
are refused), `TestServeForceCloseAfterGracefulTimeout` (configured 5 s is
honoured; the held connection is force-closed and the process exits 0).

### C-CLI-010 — stderr-only logs with Authorization redaction (§42.5, §20)

All operational records go through a `*log.Logger` built on the explicit
`stderr` writer; nothing is written to stdout. A `logHandler` wraps the host
and logs method + path, plus `authorization=<redacted>` when the header is
present — `redactAuthorization` never returns the value. The `http.Server`
`ErrorLog` is the same logger. `TestServeLogsGoToStderrAndRedactAuthorization`
runs a data-plane and a control request carrying a 64-character secret under
both `--log-format text` and `--log-format json`, and asserts the secret is in
neither stream, stdout is empty, and stderr is non-empty.

### Control API reachability (§19.5, §41)

`internal/host/http/server.go` now holds a `nethttp.Handler` control plane and
`serveControl` delegates every classified `/__fake/` request to it after the
control-plane body limit has been applied, restoring the already-read body for
the API. `Server.AttachControl` is the seam; `internal/cli/serve.go` attaches
`control.NewAPI(server.Engine(), cfg)`.

`TestServeControlAPIReachableThroughListener` proves the §41 endpoints are
reachable end to end through the ready file's URL: `/health` (exact shape,
`controlApiVersion` agrees with the ready file), `/meta` (strict, `jev/v1`,
§12.2 limits), stateful `POST /stubs` → 201, an unsupported path → control 404,
and no control traffic in `/requests` while the next `GET /v1/models` is
journaled at sequence 1. `TestServePerPlaneBodyLimits` shows the per-plane
limits survive the seam: an oversized control body is 413 and never journaled,
an oversized data body is journaled as `payload_too_large`.

### Usage failures / usage text

`serve` is routed in `internal/cli/dispatch.go`; the usage text now lists it
and no longer calls it unimplemented. `runServe` validates flag values against
the §38 rules (via a defaults probe) before touching the file or the network,
so `--port 70000`, `--mode loose`, `--log-format xml`, `--compat unknown/v1`,
duplicate/empty compat, stray positionals, and an empty `--ready-file` all exit
2 with diagnostic + usage on stderr, empty stdout, and no ready file
(`TestServeUsageFailures`).

## Precedence resolution as built

```text
effective = §12.2 built-in defaults
          → config.LoadFile(--config)        (only when --config is present)
          → applyServeFlags (present flags only, from FlagSet.Visit)
          → config.Validate                  (§38 ranges, profile normalization)
          → reject empty host                (fail closed; would bind wildcard)
```

`--log-format` is validated separately (text|json) and is not configuration
state. `--compat`, when supplied, replaces the file/default list entirely and
is normalized/duplicate-checked by the shared config validator (interpretation
I5). `--config` absent means no file is consulted at all (I4).

## Commands run and outcomes

```text
gofmt -w internal/cli/serve.go internal/cli/serve_test.go internal/cli/dispatch.go internal/host/http/server.go   clean
go test ./internal/cli -count=1                    ok (5.8s)
go test ./internal/cli -count=3                    ok (16.3s)
go test ./internal/host/http -count=1              ok
go test ./... -count=1                             ok (all packages)
go test -race ./... -count=1                       ok (all packages)
go vet ./...                                       ok
go run ./cmd/guard arch                            ok (no findings)
go run ./cmd/guard lint                            ok (no findings)
go run ./cmd/guard trace                           ok (no findings)
go run ./cmd/guard fuzz                            ok (no findings)
go build ./cmd/fake-jev                            ok
./scripts/candidate-fingerprint candidate          969f6a5c976687fa98b89b4616a4622420c81a50470f4116d15ab1d3e71769b4
```

Manual end-to-end smoke: `serve --port 0` reported
`{"url":"http://127.0.0.1:59789",...,"controlApiVersion":"v1"}`; `/health`,
`/meta`, `POST /stubs`, `GET /v1/models` and `/requests` all answered; SIGTERM
exited `0` and removed the ready file; stderr held only operational lines.

## Observations where the specification is silent

- **Import placement of the control API.** `internal/control`'s in-package test
  file imports `internal/host/http`; if `server.go` imported `internal/control`
  directly, the control test binary would have an import cycle. The host
  therefore delegates to an `nethttp.Handler` and `internal/cli/serve.go`
  attaches `*control.API` (import cycle avoided, behavior unchanged). This is
  the specifier's stated seam "the control API is only wired, not modified".
- **`serve --help`.** §15.1 defines no per-command help (specifier U1). The
  `flag` package's help request is treated as the general `help` command: usage
  on stdout, exit `0`.
- **Empty `--host ""` / an empty `server.host`.** §12.3 allows any string and
  does not forbid empty, but `:0` would bind the wildcard address, contradicting
  §19.1/§30.8. `serve` fails closed with exit `2` instead of binding publicly.
- **Empty `--ready-file ""`.** Treated as a usage failure (a path was requested
  but none is usable) rather than silently skipping readiness.
- **`--log-format json`.** Accepted and kept on stderr, but the record shape is
  unchanged; the JSON schema is a non-goal (specifier U2).
- **Ready-file `url` for a wildcard host.** The reported `host`/`url` echo the
  effective configured host (specifier U4); no criterion depends on a wildcard
  URL being dialable.

## Residual risks

- `serve` currently logs one line per request (method, path, redacted auth
  marker). The line wording is not a contract; log truncation/limit machinery
  belongs to FJ-041 and is deliberately absent.
- The drain tests synchronize on the `100 Continue` interim response, as the
  specifier recorded (fallback: a test-only seam). If a future host stops
  emitting it, those two tests need the fallback.
- The child-process tests build `./cmd/fake-jev` once per test binary and bind
  only ephemeral loopback ports; the default-port leg probes 8787 and skips if
  unavailable, and the `127.0.0.2` host legs fall back to `localhost` (this
  macOS cannot bind `127.0.0.2`), so host precedence is still asserted end to
  end plus directly in the seam test.
- `removeReadyFile` returning an error (e.g. an unexpected permission change)
  exits `2`; removal failure on a healthy filesystem is not expected.
