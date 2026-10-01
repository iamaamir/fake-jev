---
stage: coder
task: FJ-041
inputFingerprint: e520fc826889999e1e361d655d9a5f9fd80b6abf0e368376ab8390805481ca4b
outputFingerprint: 66f95e12d85d159221cc39115bd577418fade40a183d160b33b495b5ec84f816
taskFingerprint: a39d771c0c282b258d73a682eeabd71d7ad9022cd4f0ae5a8c87b6911d2cef61
gitHead: e9be073
generatedAt: 2026-10-01T07:05:20Z
author: worker/FJ-041-coder
---

# Coder — FJ-041 (operational envelope hardening)

## FINAL CANDIDATE FINGERPRINT

```text
66f95e12d85d159221cc39115bd577418fade40a183d160b33b495b5ec84f816
```

Produced by `./scripts/candidate-fingerprint candidate` after all edits, with
`task` still `a39d771c0c282b258d73a682eeabd71d7ad9022cd4f0ae5a8c87b6911d2cef61`.
The PM ruling was applied: the §19.4 normal-log body-preview limit is binding,
so the §20 "should be truncated" sentence is implemented.

## Files changed

- `internal/cli/logging.go` (new) — operational logger, redaction, body-preview
  capture and truncation.
- `internal/cli/logging_test.go` (new) — C-CLI-010/C-JEV-017/C-ARCH-006 and the
  preview half of C-HOST-008, in-process through `startServer`.
- `internal/cli/serve.go` — logger extraction; request logging and lifecycle
  diagnostics routed through the logger; the extracted `logHandler` lives in
  `logging.go`.
- `internal/host/http/server.go` — new read-only `Server.Limits()` accessor that
  reports the resolved limits; used to feed the logger.
- `internal/host/http/limits_test.go` (new) — C-HOST-008 identity/binding and
  C-ARCH-005 behavioural evidence.
- `.agent/guards.json` — C-ARCH-004/C-ARCH-005 structural rules.

No other file was edited. `run.go`, `serve_test.go`, `server_test.go`, and
`state.json` are untouched; `startServer(cfg, logger)` keeps its signature and
all existing tests pass unmodified.

## What was implemented, per criterion

### C-HOST-008 — enforced §12.2 defaults (both halves)

The criterion has an identity half and an enforcement half; both are evidenced.

**Identity** (`internal/host/http/limits_test.go`):
`TestLimitsFromDefaultConfigMatchSection122` asserts
`defaultLimits() == limitsFromConfig(Load("schemaVersion: 1")) ==
NewServerFromConfig(default).Limits() == {8388608, 2097152, 10000, 4096, 5}`.
The test restates the five numbers locally so changing a constant fails the
chain instead of silently tracking it.

**Enforcement** — which limits were already enforced, and the one binding added:

| §12.2 limit | enforcement path before FJ-041 | status |
|---|---|---|
| `dataPlaneBodyBytes` 8388608 | `Server.ServeHTTP` → `readBody(request, s.limits.DataPlaneBodyBytes)` | already enforced |
| `controlPlaneBodyBytes` 2097152 | `Server.ServeHTTP` → `readBody(request, s.limits.ControlPlaneBodyBytes)` | already enforced |
| `maxInteractions` 10000 | `NewServerFromConfig` → `engine.NewEngineWithJournal(..., limits.MaxInteractions)` | already enforced |
| `gracefulShutdownSeconds` 5 | `serverHandle.stop()` → `cfg.Limits.GracefulShutdownSeconds` deadline | already enforced |
| `logBodyBytes` 4096 | **none — no code path read the value** | **binding added here** |

The specifier's finding is confirmed: before this change `LogBodyBytes` flowed
only into `Limits`, the `/meta` document, and the config validator, and was read
nowhere that truncated or emitted anything. The added binding is:

```text
Server.Limits().LogBodyBytes          (host resolves the value, §12.2 defaults)
   -> newOperationalLogger(logger, limit).bodyLimit
   -> logHandler captureBodyPreview(request, bodyLimit)
   -> operationalLogger.logRequest(... "body=" + preview + logTruncationMarker)
```

`startServer` reads the accessor rather than re-deriving the default, so the host
stays the single authority on the resolved limit (an explicit `logBodyBytes: 0`
is substituted to 4096 by the host, exactly as `applyLimitDefaults` already did).

Evidence for the four pre-existing bindings:

- `TestReadBodyEnforcesDefaultPlaneLimits` — at the default plane limits a
  limit-sized body is admitted, a declared oversize is rejected **without a
  single body byte being read** (`unreadableBody`), and a streamed oversize
  (unknown length) is rejected. A `repeatingReader` keeps the fixture at a few
  KiB instead of 8 MiB.
- `TestServerEnforcesSection122DefaultPlaneThresholds` — end-to-end through
  `ServeHTTP` with the minimal default configuration: `Content-Length` one byte
  over each plane default is answered `413` and the body is not read, so the
  resolved default is the server's threshold.
- `TestDefaultMaxInteractionsIsEnforced` — 10,000 in-process `GET /v1/models`
  requests are journalled, the journal reports itself full at exactly 10,000, the
  10,001st returns `507` `fake_jev_journal_full`, and no eviction occurs.
- `gracefulShutdownSeconds`: identity above, plus the existing
  `TestServeForceCloseAfterGracefulTimeout`, which proves the configured (10 s)
  value is read, not a hard-coded 5 s constant. A direct wall-clock observation
  of the 5 s default was deliberately not added (≈6 s of wall clock, and the
  configured-value test already excludes a hard-coded constant), as recorded by
  the specifier.

### C-CLI-010 / §20 / §42.5 — logging and bounded previews

- Operational output is centralised in `internal/cli/logging.go`. `serve` and
  `run` continue to build the sink as `log.New(stderr, "fake-jev: ", 0)`;
  `operationalLogger` wraps it and is the only writer of served-request and
  lifecycle lines. The `serve` failure/`ready-file`/`serve` diagnostics that were
  direct `writef(stderr, ...)` calls now go through `logger.Printf`, so nothing
  in the served-request or server-lifecycle path bypasses the logger.
- `logHandler` captures a bounded preview before delegating and logs after the
  response; every request path (matched, unmatched, rejected, control, unknown
  route) reaches `logRequest`, so redaction and truncation hold on every path
  including error paths.
- Previews are cut to the configured `logBodyBytes` and terminated with the
  explicit `logTruncationMarker = "...[truncated]"`.
- `TestOperationalLogsTruncateBodyPreview` drives a ~20 KiB `state` with
  sentinels at offset 0, 100, 8000, and the end. Default 4 KiB preview keeps
  `HEAD` and `BODY-100`, drops `MID-8000`/`TAIL-END`; configured 64-byte preview
  keeps only `HEAD`. Both runs also assert the marker is present, the
  Authorization sentinel is absent, and `GET /__fake/v1/requests`
  `requestBody.state` still returns the full 20,008-byte string.
- Destination (stderr, never stdout) is unchanged and still pinned by the
  existing child-process `TestServeLogsGoToStderrAndRedactAuthorization` for both
  `--log-format` values; it passes untouched.

### C-JEV-017 — identical acceptance, value never logged

`TestAuthorizationValuesAreAcceptedIdenticallyAndNeverLogged` runs three header
cases (absent, `Bearer fake`, arbitrary `Bearer FAKEJEV-TOKEN-SENTINEL-…`) over a
matched systemone (`200`), `/v1/models` (`200`), and an unmatched systemone
(`501`). The three response sets are byte-identical, the token appears in neither
the logs nor the marshalled `/__fake/v1/requests` document, and all nine
data-plane requests are journalled.

### C-ARCH-004 / C-ARCH-005 — structural guard rules

`.agent/guards.json` now carries three `arch.forbidden` rules:

- `internal/engine` — `forbidExact` extended from `["net/http","os/exec"]` to
  `["net","net/http","os/exec","text/template","html/template","plugin"]`
  (`net` is the new egress-specific entry; the template/plugin entries are the
  fixture-interpretation bans); `forbidPrefix` unchanged.
- `internal/compat/jev/v1` — new rule with the same `forbidExact` set.
- `internal/config` — new rule with the same `forbidExact` set.

These cover the fixture-and-behaviour core (config load → profile decode →
engine) and count test imports too (`cmd/guard/pkgscan.go` unions
`Imports`/`TestImports`/`XTestImports`), so a future socket or template import
from any file trips the rule. `internal/host/http` (it *is* the server) and
`internal/cli` (`run` needs `os/exec`, §15.4) are deliberately excluded.

**Proof the repository satisfies the rules:** `go run ./cmd/guard arch` →
`{"check":"arch","findings":[],"stats":{"files":59,"packages":9}}`, exit `0`.
No rule was added that the current code fails.

**Falsifiability proof:** a temporary `internal/config/zz_falsify_tmp.go` with
`import _ "net"` and `internal/engine/zz_falsify_tmp.go` with
`import _ "text/template"` made the guard report both
`guard.arch.forbidden_import` findings and exit `1`; both temp files were removed
and `git status` confirmed no residue.

Behavioural C-ARCH-005 evidence (`internal/host/http/limits_test.go`):

- `TestFixtureRawBodyIsDataNotTemplate` — a `then.raw` body containing
  `{{ .Env.HOME }}`, `{{ 7*7 }}`, `${process.env.SECRET}`, `$(id)`, and `1+1` is
  returned byte-for-byte; `{{ 7*7 }}` stays a string and never becomes `49`.
- `TestFixtureStateMatchesLiterally` — a `when.state` of `"{{ .Env.HOME }}"`
  matches only the literal request state (`200`); a rendered-looking value is
  `501`, so matching is exact JSON equality, not evaluation.
- `TestFixtureExpressionWhereNumberRequiredIsRejected` — `noul: "{{ 7*7 }}"`
  loads as data and is refused when selected as `500`
  `fake_jev_invalid_stub_response`; `49` never appears.

### C-ARCH-006 — default bind is 127.0.0.1

`TestDefaultBindHostIsLoopback` adds the config-layer evidence the specifier
identified as the delta: `config.DefaultHost == "127.0.0.1"`, the minimal
document resolves `server.host`/`port` to `127.0.0.1:8787`, and an in-process
`startServer` with no host override binds `127.0.0.1`. The process-level bind,
ready-document, and not-bound-publicly checks already exist in the unmodified
`TestServeDefaultBindIsLoopback`. `--host 0.0.0.0` remains a legal explicit
choice and is not asserted against.

## Guard rules — before/after summary

- Added `net` to `internal/engine.forbidExact`; added two new rules
  (`internal/compat/jev/v1`, `internal/config`) with `net`, `net/http`,
  `os/exec`, `text/template`, `html/template`, `plugin`.
- `trace.active` left empty (no criterion requires C-ID string markers).
- `lint`, `complexity`, `mutation`, `fuzz` sections unchanged.

## Exact commands and outcomes

| command | result |
|---|---|
| `gofmt -w` on the five changed/new `.go` files; `gofmt -l internal/cli internal/host/http` | no output (clean) |
| `go test ./internal/cli -count=1` | `ok` (≈20.7 s) |
| `go test ./internal/host/http -count=1` | `ok` |
| `go test ./... -count=1` | all packages `ok` (guard, cli, compat, config, control, engine, host/http, integration) |
| `go test -race ./... -count=1` | all packages `ok` |
| `go vet ./...` | exit 0 |
| `go run ./cmd/guard arch` | `findings: []`, exit 0 |
| `go run ./cmd/guard lint` | `findings: []`, exit 0 |
| `go run ./cmd/guard trace` | `findings: []`, exit 0 |
| `go run ./cmd/guard fuzz` | `findings: []`, exit 0 |
| `go build ./cmd/fake-jev` | exit 0 |
| `./scripts/candidate-fingerprint candidate` | `66f95e12d85d159221cc39115bd577418fade40a183d160b33b495b5ec84f816` |

`./scripts/verify-candidate FJ-041` was not run: it writes
`.agent/reports/`, and this stage was instructed to write only `coder.md` and not
to edit reports. `state.json` was not edited (its pre-existing unstaged
specifier update is untouched).

One `go test ./... -count=1` run failed `cmd/guard` `TestRunFuzzPass` with
`guard.fuzz.crash ... FuzzSafe failed`; that test spawns a live
`go test -fuzz -fuzztime=1s` and is load-sensitive. Three subsequent full-suite
runs were clean, and `cmd/guard` source is untouched by this change, so it is a
pre-existing flake, not a regression (recorded under residual risks).

## Spec-silent observations

- **Preview shape is undefined** (§20). The implementation previews the head of
  the raw request body. No test asserts wording, marker placement, or whether the
  preview is raw body versus the parsed `state`; the assertions are presence of
  head content, absence beyond the bound, absence of the Authorization value, and
  the byte bound.
- **`logBodyBytes: 0` is spec-undefined** (§38.3 permits it; §12.1 lets an
  explicit value win). The host substitutes 4096 (`applyLimitDefaults`), and the
  logger now uses that host-resolved value, so enforcement is 4096 while
  `/__fake/v1/meta` still reports the configured `0`. This is the pre-existing
  `/meta`-vs-enforcement divergence the specifier flagged; it is recorded, not
  changed.
- **`--log-format json` is still not consumed.** It is parsed and validated and
  the emitted record is text for both accepted values. §20 calls structured
  logging optional; no criterion here requires a JSON schema, so no behaviour
  was invented.
- **Truncation marker text is our choice** (`"...[truncated]"`). The spec
  requires no marker; the deliverable requires an explicit, documented one, and
  the constant's comment documents it.
- **Binding order.** The preview read is bounded by `logBodyBytes` and happens
  before the host's plane-limit rejection, so a configured `logBodyBytes` larger
  than a plane limit permits up to that many bytes to be read before a `413`. The
  read is always bounded (§19.4 "before unbounded allocation" holds) and with the
  §12.2 defaults it is ≤ 4 KiB; the spec does not relate the two limits.
- **Specifier corrections (actual behaviour pinned):**
  - C-ARCH-005 item 3: `fake-jev validate` does **not** reject
    `noul: "{{ 7*7 }}"`; `config.Load` accepts it as data (verified:
    `validate` exits 0). Rejection happens at response generation as `500`
    `fake_jev_invalid_stub_response`. The test pins the actual seam.
  - C-HOST-008/H8c: `Engine().JournalFull()` is `true` at exactly 10,000
    records (`journal.full()` is `len >= max`), not `false`; the journal is at
    capacity and the next request is the one that must `507`. The test pins the
    actual semantics.

## Residual risks

- `cmd/guard` `TestRunFuzzPass` is load-sensitive (live `go test -fuzz`); one
  flake was observed under a parallel full-suite run, then three clean runs.
- A configured `logBodyBytes` above a plane limit increases how many bytes are
  read before the plane rejection (bounded; spec-silent).
- `--log-format json` remains a validated no-op (pre-existing).
- The C-ARCH-004 guard is structural only; `internal/host/http` and
  `internal/cli` are outside it by design. The call-site audit (the only
  `net`/`net/http` uses are `net.Listen`, `net.JoinHostPort`, the `net/http`
  server/handler types, and header plumbing; no `http.Client`, `net.Dial`, or
  `os/exec` outside `run`) is recorded by the specifier and unchanged here.
