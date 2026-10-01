---
stage: specifier
task: FJ-041
inputFingerprint: e520fc826889999e1e361d655d9a5f9fd80b6abf0e368376ab8390805481ca4b
outputFingerprint: e520fc826889999e1e361d655d9a5f9fd80b6abf0e368376ab8390805481ca4b
taskFingerprint: a39d771c0c282b258d73a682eeabd71d7ad9022cd4f0ae5a8c87b6911d2cef61
gitHead: e9be073
generatedAt: 2026-10-01T06:48:35Z
author: worker/FJ-041-specifier
---

# Specifier — FJ-041 (operational envelope hardening)

Objective, restated as observables: with no configuration beyond
`schemaVersion: 1`, the server (a) resolves the §12.2 limits to their exact
values **and** routes every one of them into the code path that enforces it,
(b) keeps operational logs on stderr with body previews bounded by the
configured log-preview limit while the untruncated body stays retrievable
through the control plane, (c) never emits an `Authorization` value, (d) makes
no external network, model, or provider call, (e) never evaluates fixture
text as template, expression, shell, or code, and (f) binds loopback.

No verdict is expressed anywhere in this artifact. Only
`./scripts/verify-candidate FJ-041` emits results.

## Traces

- C-HOST-008
- C-CLI-010
- C-ARCH-004
- C-ARCH-005
- C-ARCH-006
- C-JEV-017

## Baseline: what the workspace already pins

Read from `e9be073` plus the one unstaged `state.json` edit. This is the delta
the coder should reason about, not a re-statement of the acceptance criteria.

- §12.2 value identity for all five limits is already asserted through
  `GET /__fake/v1/meta` in `internal/cli/serve_test.go`
  (`TestServeControlAPIReachableThroughListener`): `8388608 / 2097152 / 10000 /
  4096 / 5`. So the *values* are pinned; what is not pinned is that the
  host's resolved limits reach the enforcement paths.
- Enforced behaviour for *configured* (non-default) limits exists:
  `TestServePerPlaneBodyLimits` (1024-byte planes), `TestServeForceCloseAfterGracefulTimeout`
  (`gracefulShutdownSeconds: 10`), `TestServerPayloadLimitDoesNotInvokeStub`,
  `TestServerClosesRequestBodiesAndHandlesChunkedLimits`, and the 507
  journal-full path is covered by the engine/control suites.
- `internal/host/http/server.go` reads `limits.LogBodyBytes` only to carry it
  in `Limits`; no code path anywhere in the module truncates or emits a body
  preview (`grep -n "LogBodyBytes"` finds only `Limits`, the §12.2 constants,
  `/meta`, and the config validator). §19.4's "normal-log body preview: default
  4 KiB" therefore has no enforcement site yet.
- `--log-format` is parsed and validated in `internal/cli/serve.go:106-108` and
  is read nowhere else; emitted lines are the text form built by `logHandler`.
- `Authorization` redaction exists (`redactAuthorization`), and
  `TestServeLogsGoToStderrAndRedactAuthorization` already pins "stderr carries
  the logs, stdout carries none, the value never appears" for both accepted
  `--log-format` values. What is not pinned is *identical acceptance* of the
  three §39.5 header cases at the HTTP host, and that the value is also absent
  from explicitly requested interaction data.
- §19.1/§37 default bind is already pinned by
  `TestServeDefaultBindIsLoopback` (literal host, ready URL, and a
  non-loopback dial probe that reports itself as not exercised on a host with
  no non-loopback IPv4 interface).
- `.agent/guards.json` carries one arch rule (`internal/engine`) and no rule
  for the other core packages; `trace.active` is empty.

## C-HOST-008 — enforced §12.2 defaults

**Spec clause.** §12.2 (`dataPlaneBodyBytes: 8388608`, `controlPlaneBodyBytes:
2097152`, `maxInteractions: 10000`, `logBodyBytes: 4096`,
`gracefulShutdownSeconds: 5`, §12.1 "config file value overrides built-in
default"), §19.4 ("The host MUST enforce: … default 8 MiB … default 2 MiB …
default 10,000 … default 4 KiB … default 5 seconds"), §38.3 ranges.

The criterion has two halves that must be evidenced separately, because value
identity alone is satisfied today:

1. **identity** — the resolved default equals the §12.2 number;
2. **enforcement binding** — the same resolved value reaches the code that
   accepts, rejects, journals, truncates, or times out, so a change to the
   constant changes observable behaviour.

### H8a — data-plane body limit, 8 MiB

- Observable: with defaults, a data-plane request whose decoded body length is
  ≤ `8388608` is admitted (any status other than `413`), and a request of
  `8388609` bytes receives `413` with error `fake_jev_payload_too_large`, is
  journaled with outcome `payload_too_large` and `requestBody: null`, and
  reaches no stub. An oversize `Content-Length` is rejected **without reading
  the body** (§19.4 "before unbounded allocation").
- Exercise (deterministic, no network, no sleeps):
  1. In-package (`internal/host/http/limits_test.go`, package `http`): build an
     `*nethttp.Request` whose `Body` is an `io.LimitReader` over a repeating
     reader and whose `ContentLength` is declared, then assert `readBody` with
     `limit = DefaultDataPlaneBodyBytes` returns `tooLarge` for `limit+1`
     without consuming the body, and accepts exactly `limit` bytes. The
     repeating reader keeps the test's own allocation at a few KiB instead of
     8 MiB, and no wall clock is consulted.
  2. Identity chain, same file: `limitsFromConfig(defaultConfig().Limits)`
     equals the §12.2 `Limits` exactly (all five fields), which is the value
     `NewServerFromConfig` passes to the host and the engine.
  3. One end-to-end at the default: a `serve` with no config file and a
     `8388609`-byte `POST /v1/systemone` body over the ready-file URL yields
     `413`; `GET /__fake/v1/requests` then shows exactly one record with
     outcome `payload_too_large` and `requestBody: null`. This is the only
     multi-MiB step and it is a single request.

### H8b — control-plane body limit, 2 MiB

- Observable: with defaults, `2097153` bytes to a control path yields `413`
  with `fake_jev_payload_too_large` and the interaction journal stays empty
  (§11, §41.1); `2097152` bytes does not yield `413`.
- Exercise: `readBody` at `limit = DefaultControlPlaneBodyBytes` in the
  in-package test as in H8a, plus one end-to-end `2097153`-byte
  `POST /__fake/v1/stubs` over the default-config server asserting the `413`
  plus `len(requests) == 0`.

### H8c — interaction journal, 10000

- Observable: with defaults, the 1st through 10,000th admitted data-plane
  request are journaled; the 10,001st returns `507` with
  `fake_jev_journal_full`, assigns no sequence, and leaves the journal at
  exactly 10,000 records with no eviction (§19.4, §43.5).
- Exercise: in-package, `NewServerFromConfig(defaultConfig())` (equivalently
  `NewServerWithConfig`), driven by `httptest.NewRecorder` +
  `server.ServeHTTP` — no sockets, no sleeps. Issue 10,000 `GET /v1/models`
  requests, then assert `Engine().JournalFull()` is false and
  `len(Engine().Interactions()) == 10000`; issue one more and assert `507`,
  `JournalFull()` true, and still `10000` records. `GET /v1/models` is chosen
  because §39.1 journals it without needing a stub, and 10,000 in-process
  transitions are deterministic and fast. The engine-level 507 semantics are
  already covered elsewhere; this exercise exists to bind the *default* 10000
  to the capacity actually used.

### H8d — normal-log body preview, 4 KiB

- Observable, per §19.4 + §20: for a request body strictly larger than
  `limits.logBodyBytes`, normal operational logs on stderr contain the
  beginning of that body (or of its `state` value) and never contain body
  content from beyond the configured preview limit; the untruncated body
  remains available through `GET /__fake/v1/requests` `requestBody` (§41.7).
- Exercise (wording-independent, because §42.5 forbids tests depending on
  human log wording): a JSON body with a `state` string of ~20,000 bytes
  carrying four distinct sentinels at known offsets —
  `HEAD` at offset ~8, `BODY-100` at offset ~100, `MID` at ~8000, `TAIL` at
  the end — plus a matching static stub so the request is `matched` and the
  response is `200`.
  - default (`logBodyBytes: 4096`): stderr contains `HEAD`, does **not**
    contain `MID` or `TAIL`; `GET /__fake/v1/requests` `requestBody.state`
    compares equal to the full 20,000-byte string.
  - configured (`logBodyBytes: 64`): stderr contains `HEAD`, does **not**
    contain `BODY-100`, `MID`, or `TAIL`.
  - both runs: assert on both `--log-format text` and `--log-format json`, and
    assert the `Authorization` sentinel is absent (H8d shares its request
    shape with C-CLI-010/C-JEV-017).
  - The pair is what proves "configured, not hard-coded": the 64-byte run must
    drop content at offset ~100 that the 4096-byte run keeps, while both are
    genuinely large bodies so the "truncate large bodies" clause is exercised
    in both.
  - Do not assert the truncation marker text, the preview's placement, or
    whether the preview is the raw body or the parsed `state` value; all four
    are undefined (see Observations).

### H8e — graceful shutdown, 5 s

- Observable: `stop()` drains in-flight requests for the configured
  `gracefulShutdownSeconds` and force-closes whatever remains (§42.3). With
  defaults, that value is 5.
- Exercise and honest boundary: the default's *enforcement* is bound by
  (i) the identity chain (`limitsFromConfig(defaultConfig().Limits).GracefulShutdown == 5`
  and the `/meta` value already pinned) and (ii) the existing configured-value
  observation in `TestServeForceCloseAfterGracefulTimeout`, which holds a
  request open, sends `SIGTERM`, and requires that the process does **not**
  exit before a 10-second configured grace elapses — so a hard-coded 5-second
  constant cannot satisfy it.
  A direct wall-clock observation of the 5-second default (the same
  `holdRequest` shape with a ≥5 s lower bound) is available but is **not part
  of the criterion**: it costs ~6 s of wall clock and would be satisfied by
  the same hard-coded constant that the configured-value observation already
  excludes. Record the choice explicitly in the coder artifact rather than
  adding a slow timing test silently.

### C-HOST-008 observations

- `logBodyBytes: 0` is valid per §38.3 (`integer >= 0`), and §12.1 makes an
  explicit config-file value beat the built-in default. `config.applyDefaults`
  already writes 4096 for an *omitted* field, so a `0` reaching
  `limitsFromConfig` can only mean "explicitly 0" — yet
  `applyLimitDefaults` (`internal/host/http/server.go:74-79`) substitutes 4096.
  The meaning of 0 (no preview? unlimited?) is **undefined in the spec**, so no
  criterion asserts it. It is recorded here because `/__fake/v1/meta` reports
  the config value while enforcement would use the substituted one, and any
  change to it is a decision, not part of C-HOST-008.
- `/meta` is fed from `config.Config.Limits`, not from the host's resolved
  `Limits`, so `/meta` alone can never prove enforcement. The in-package
  identity chain plus the behavioural assertions above are the evidence for
  the enforcement half.
- The host's `Limits` zero-value substitution ("a zero value is filled with the
  specification defaults") is a Go-level convenience, not a §12.1 precedence
  rule; nothing in the criteria depends on it.

## C-CLI-010 / §42.5 / §20 — logging destinations and bounded previews

**Spec clause.** §42.5 ("Human/JSON operational logs go to stderr. Command
result data … MAY use stdout. … Authorization header values MUST never be
logged."), §20 ("Do not log secrets by default. Authorization values must be
redacted. Large state/request bodies should be truncated in normal logs while
remaining available through explicitly requested interaction data subject to
configured limits."), §19.4 (preview limit is a MUST-enforced host limit),
§15.1 (`serve` writes no result document to stdout; the ready file is the
result channel).

- Observable 1 (destination): every operational line `serve` emits appears on
  stderr; stdout carries no log record. Already pinned for `text` and `json`
  by `TestServeLogsGoToStderrAndRedactAuthorization`; the new evidence must not
  duplicate it but must keep it true, including for a request that carries a
  body preview.
- Observable 2 (result channel): command result data (`version` output, ready
  file, usage on `--help`) is allowed on stdout and stays there; the new logger
  must not write to stdout (§42.5 "MAY use stdout" is a permission, not an
  obligation).
- Observable 3 (redaction): no `Authorization` value ever appears on either
  stream, for both accepted log formats, for data-plane and control-plane
  requests. The spec names no redaction marker text, so the assertion is
  absence of the value, never presence of `[redacted]`.
- Observable 4 (bounded previews): as H8d — the preview bound follows the
  configured `limits.logBodyBytes` and body content beyond it never reaches
  stderr, while `GET /__fake/v1/requests` still returns the full
  `requestBody`.
- Exercise: one parameterised in-process/child-server test over the three
  axes already used by the existing test (`text`, `json` × data plane, control
  plane), asserting: `stdout == ""` for `serve`; sentinel absent from both
  streams; `HEAD` present and `MID`/`TAIL` absent on stderr; full
  `requestBody.state` length via the control plane. Full child-process variant
  (the existing pattern) is preferred for the destination assertion because it
  is what a wrapper observes; an in-process variant is acceptable for the
  preview bound (it needs the logger's stream, which `startServer` already
  receives as an `io.Writer`-backed `*log.Logger`).

### C-CLI-010 observations

- §20's truncation sentence uses "should"; §19.4 states the preview limit as a
  host MUST. This artifact reads the pair as: a normal-log body preview exists
  and is bounded by `limits.logBodyBytes`. The reading is recorded because
  "no body bytes in logs at all" would leave §19.4's preview limit with no
  enforcement site and would make H8d's `HEAD`-present assertion unsatisfiable.
  The specifier takes the §19.4 reading; if the parent prefers the vacuous
  reading, that is a scope decision and H8d's first assertion is the item to
  drop.
- The preview's shape is undefined by the spec: whether it is the raw request
  body, the parsed `state` value, or a re-encoded JSON object; whether it is on
  the request line or a separate line; whether a truncation marker or byte
  count is printed; whether one budget is shared per line or per field. Tests
  must therefore assert only (a) presence of head content, (b) absence of
  beyond-limit content, (c) absence of the Authorization value, (d) the byte
  bound. No criterion asserts a log line's wording.
- `--log-format json` is accepted and validated but no code path consumes it
  today; §20 labels structured logging "optional". The truncation bound must
  hold for whichever shape is emitted, so H8d runs both formats; the JSON
  record's schema is not asserted, and honouring `--log-format` is not part of
  any criterion here.
- `internal/cli/run.go:135-136` constructs `log.New(stderr, "fake-jev: ", 0)`
  and calls `startServer(cfg, logger)`, and `run.go` is outside `allowedFiles`.
  The `startServer(cfg *config.Config, logger *log.Logger)` signature and the
  `serverHandle`/`startServer` names must therefore stay source-compatible, and
  any preview wiring must live inside `startServer` (which already holds `cfg`)
  rather than in `serve()`.

## C-JEV-017 — identical auth acceptance, value never logged

**Spec clause.** §10.1 ("accept requests with no Authorization header; accept
arbitrary bearer tokens; never validate tokens externally; never call a
provider"), §39.5 ("The following are all accepted identically: no
Authorization header / `Authorization: Bearer fake` / `Authorization: Bearer
anything`. The value MUST NOT be logged."), §39.1 (auth headers ignored on
`GET /v1/models`), §42.5.

- Observable 1 (acceptance identity): for each of the three header cases, the
  data-plane and control-plane responses are byte-identical — same status,
  same body — for `GET /v1/models` and for a `POST /v1/systemone` that matches
  a configured stub (e.g. `200` with the configured answer), and for an
  unmatched `POST` (`501`).
- Observable 2 (no side effect): the three cases produce identical journaled
  records apart from the timestamp, which §41.7 declares diagnostic.
- Observable 3 (never logged, never exposed): the token value appears in
  neither stream, and not in `GET /__fake/v1/requests` either — §41.7 exposes
  no request headers, so an implementation that logged or journaled the header
  would be observable there.
- Exercise (deterministic, no network beyond loopback, no sleeps): start one
  default-config server; use a distinctive constant such as
  `Bearer FAKEJEV-TOKEN-SENTINEL-0123456789abc` rather than a plausible-looking
  token; send the six requests (3 header cases × 2 routes plus the unmatched
  case), collect status+body per case, and assert byte-equality across cases
  plus the absence assertions over `stderr`, `stdout`, and the
  `/__fake/v1/requests` JSON text. Profile-level acceptance already exists
  (`TestProfileAcceptsAuthorizationWithoutInspectingIt`); this exercise adds
  the host-level identity and the "never logged/journaled" half, which are the
  parts the criterion names.
- Observation (spec-undefined): the spec fixes no header-value normalization
  (`bearer` lowercase, multiple `Authorization` headers, `Bearer` with no
  token) beyond the three listed forms, and no error shape for an exotic value;
  no criterion asserts them. Also, since the value must never be logged, the
  presence-vs-absence of a header must not be inferable from log content in a
  way that leaks the value; only the value's absence is asserted.

## C-ARCH-004 — no network egress, no model or provider call

**Spec clause.** §19.2 ("Core fake behavior must not make external network
requests. A test should be able to run fake-jev in an environment with outbound
networking disabled"), §4.9 (zero-model execution: no model weights, GPU,
runtime, credentials, network, paid APIs), §7.1 and §17.3 (engine import
boundaries, engine must not import `net/http`, `os/exec`, `compat/jev`, CLI,
control), §17.2 (dependency policy: standard library plus one YAML
dependency), §37 ("Network egress | none in normal execution").

**Provenance of this criterion: structural, not behavioural — and the honest
limits of that proof.**

The evidence is a structural guard over the fixture-and-behaviour core, not a
behavioural observation. `.agent/guards.json` `arch.forbidden` already supports
per-package rules with `forbidExact` and `forbidPrefix`; `cmd/guard/arch.go`
keys each rule by module-relative directory (`rule.Pkg == moduleRel(...)`,
`arch.go:31-63`) and matches `allImports()`, which unions imports, test
imports, and external test imports (`cmd/guard/pkgscan.go:38-48`). No change to
`cmd/guard` is required — confirmed by reading `arch.go`, `pkgscan.go`,
`guards.go` (`validate()` accepts any non-empty pattern list per rule and
rejects unknown JSON fields), and `arch_test.go` (exact-string and
trailing-slash-prefix semantics are already covered by tests). The rules are
executed by `check_guard_arch` inside `./scripts/verify-candidate`, so they are
deterministic and offline.

- Observable (structural): the core packages import nothing that can dial a
  socket or spawn a process — i.e. a rule for each of `internal/engine`,
  `internal/compat/jev/v1`, and `internal/config` whose `forbidExact` contains
  `net`, `net/http`, and `os/exec`. Today `internal/engine` bans only
  `net/http` and `os/exec`; `internal/config` imports `os` (allowed — §7.1
  places filesystem configuration loading outside the engine) but no `net`; the
  profile imports no networking at all. Adding the `net` entry is the new
  egress-specific part; it holds for the current tree and would trip on any
  future socket import, including from a test file.
- Observable (dependency graph): `go.mod` declares exactly one direct
  requirement, `gopkg.in/yaml.v3 v3.0.1`, and `go list -m all` lists only that
  plus its test-only `gopkg.in/check.v1`; no model, provider, HTTP-client, or
  cloud SDK is in the module graph, so "no model or provider call" holds at the
  dependency level and cannot be reached without a deliberate `go.mod` change.
- Falsifiability: adding `import _ "net/http"` to `internal/config` (or any
  template/exec import, see C-ARCH-005) must make `verify-candidate` report the
  guard failure; that is what makes the rule evidence rather than decoration.
- **Honest limit 1**: `internal/host/http` legitimately imports `net` and
  `net/http` because it *is* the server, so a per-package import rule cannot
  cover it. Its no-egress property is not automatically enforced. The honest
  adjacent evidence is a call-site audit recorded in the coder/cleaner
  artifacts: the only `net`/`net/http` usage in the module is `net.Listen`,
  `net.JoinHostPort`, the `net/http` server and handler types, and header
  plumbing — no `http.Client`, `http.Get/Post/Do`, `net.Dial`, `net.Dialer`,
  `net/http/httptest`-client-in-production, or `os/exec` outside `internal/cli`'s
  documented `run` child process. A guard rule cannot express call-site
  absence, and `http.Client`-shaped assertions by source grep are not proposed
  as a test because a grep test in `_test.go` would be brittle and would not
  be a behavioural observation either.
- **Honest limit 2**: no behavioural proof of "no egress" is available in this
  repository deterministically. Running under network isolation (network
  namespace, firewall, or a sentinel DNS) is environmental and not something a
  passing/absent test outcome can encode; `verify-candidate`'s header already
  documents that the whole suite runs offline, and the integration tests
  exercise the data plane through loopback only. This artifact therefore does
  **not** claim a behavioural proof: C-ARCH-004 is proven structurally, plus
  the call-site audit, and the criterion's "test should be able to run with
  outbound networking disabled" (§19.2) is satisfied in the weaker sense that
  nothing in the suite requires egress.
- Observation: §19.2 says "Core fake behavior", not "the process". The
  `serve`/`verify` lifecycle does open a listener and (for `run`) spawns a
  child; neither is egress. `internal/cli` importing `os/exec` is required by
  §15.4 and is excluded from the rules for that reason.

## C-ARCH-005 — fixtures are data only

**Spec clause.** §19.3 ("Do not support JavaScript expressions, Go templates
with dangerous functions, shell commands, dynamic eval, or arbitrary code
callbacks in configuration. Fixtures are data."), §12.8 ("v1 fixtures MUST
remain data-only and MUST NOT execute code, templates, shell commands,
JavaScript, or expressions"), §12.6/§39.7 (raw body serialized once,
verbatim), §40.3 (state matching is exact JSON equality), §13 (answer values
are finite numbers, not expressions), §17.2.

**Behavioural evidence (primary).** Fixture values whose text looks like an
expression are returned byte-for-byte and are never evaluated:

1. **Verbatim response.** A stub with `then.raw` (status `200`, no configured
   `Content-Type`, `body` containing `{"tpl": "{{ .Env.HOME }}", "tmpl":
   "{{ 7*7 }}", "js": "${process.env.SECRET}", "shell": "$(id)",
   "eval": "1+1"}`) responds to a matched `POST /v1/systemone` with a body whose
   JSON compares equal to the configured value: `{{ 7*7 }}` is still a string,
   not `49`; no environment value, no command output, and no arithmetic appears.
   §12.3 exempts `then.raw.body` from unknown-key validation, so this fixture is
   accepted as data. With no configured `Content-Type`, §39.7 requires
   `Content-Type: application/json`, which also pins that the body took the
   normal JSON path rather than an arbitrary-bytes path.
2. **Verbatim matching.** A stub whose `when.state` is the string
   `"{{ .Env.HOME }}"` matches a request whose `state` is literally that string
   and does **not** match a request whose `state` is the value such a template
   would render to (e.g. a home-directory-looking string). §40.3 makes this
   exact JSON equality, so non-evaluation is directly observable through
   `200` vs `501`.
3. **No expression where a value is required.** `then.answers` with
   `q: {noul: "{{ 7*7 }}"}` is rejected by configuration validation (a string
   where a finite number in `[0,1]` is required, §13): the observable is
   `fake-jev validate` exiting non-zero with a configuration diagnostic and no
   server ever binding. Symmetrically, an answer map that sums outside `1e-6`
   of `1.0` is rejected, not repaired — "fail closed" rather than evaluated.
4. **No environment reflection.** Set a distinctive environment variable on
   the child, run the same verbatim fixture, and assert the variable's value
   appears nowhere in the response body, the journal, or the logs.

**Structural evidence (supporting).** Per-package `forbidExact` rules for
`internal/config`, `internal/compat/jev/v1`, and `internal/engine` naming
`text/template`, `html/template`, and `os/exec` — the packages on the fixture
interpretation path (config load → profile decode → engine match). None of the
three imports any of them today, and the guard counts test imports too. The
rules deliberately do **not** cover `internal/host/http` (the server) or
`internal/cli` (`run` requires `os/exec`, §15.4); neither package has a
fixture-evaluation seam — `compileStubs` copies `json.RawMessage` and answer
values verbatim into engine stubs (`internal/host/http/server.go`). A
JavaScript engine is excluded by §17.2 plus the fact that the module graph has
one requirement, which is the structural half of "no JavaScript".

**Observations (spec-undefined).** The spec does not define an error or a
warning for a fixture value that *looks* like a template/expression (there is
no such diagnostic; it is data), does not define a maximum fixture string
length or nesting depth beyond the body limits, and does not define behaviour
for a raw body that is not valid JSON (currently an encoding error surfaces as
§21.2/§43.4-style failure, not as an "evaluate the string" path). No criterion
asserts a diagnostic for these.

## C-ARCH-006 — default bind is 127.0.0.1

**Spec clause.** §19.1 ("Default bind address: `127.0.0.1`. Do not bind
publicly by default. Users running in Docker may explicitly choose `0.0.0.0`."),
§12.2/§12.3 (`server.host` default `127.0.0.1`), §37 ("Default bind |
`127.0.0.1:8787` for `serve`"), §15.1 (ready-file `host`/`url`), §30.8.

- Observable 1 (config layer): `config.DefaultHost == "127.0.0.1"` and loading
  the minimal document `schemaVersion: 1` resolves `Server.Host` to the literal
  `"127.0.0.1"` and `Server.Port` to `8787`.
- Observable 2 (process layer): `serve --port 0` with no config file and no
  `--host` writes a ready document whose `host` is exactly `"127.0.0.1"` and
  whose `url` is exactly `http://127.0.0.1:<bound-port>`; the first data-plane
  request over that URL is answered (`GET /v1/models` → `200` with the default
  model list).
- Observable 3 (negative space): the listener refuses connections on every
  non-loopback local IPv4 address for the bound port. This sub-check must
  report itself as **not exercised** when the machine has no such interface,
  following the existing `assertNotBoundPublicly` convention, rather than
  degrading into a silent success.
- Observable 4 (explicit override stays legal): `--host 0.0.0.0` is a
  permitted explicit choice per §19.1; the criterion covers the default only,
  and no assertion may require public binds to be impossible.
- Exercise: `TestServeDefaultBindIsLoopback` already covers observables 2-3 at
  the child-process level, so the delta is observable 1 (a `config`-level
  assertion, which cannot live in `internal/config`'s own test file because
  that package is outside `allowedFiles`) and, optionally, the `run` path's
  forced loopback (`internal/cli/run.go`), which is outside `allowedFiles` and
  is therefore recorded as existing coverage rather than added here.
- Observation: the port default `8787` is not part of this criterion, and
  `run`'s ephemeral-port default is C-CLI-004's territory.

## Non-goals (respected, not re-litigated)

- **No latency or fault injection.** §33 defers "advanced latency/fault
  injection"; no criterion here introduces delaying, failing, or
  corrupting behaviour. The `nice-to-have` fault knobs are out of scope even
  though they would sit near the limits/logger code.
- **No control-plane auth.** §19.5 keeps the control API on the same local
  server with no auth, and §33 defers "authentication for the control plane".
  C-JEV-017 covers the *data* plane only; the control plane remains
  unauthenticated by design, and `/__fake/v1/*` must not grow a token check.
- **No metrics backend.** §20 optimises diagnostics for test debugging rather
  than production telemetry, and §33 defers a UI/dashboard. No Prometheus,
  counters endpoint, or `/metrics` route is added; the observable surface stays
  logs, the ready file, the control API, and the journal.

## Out of scope for this item

- Products of the criteria that already have unmodifiable coverage:
  `internal/cli/serve_test.go`, `internal/host/http/server_test.go`, and
  `internal/config` are outside `allowedFiles`; existing tests in them must
  keep working unmodified, which is why the new evidence belongs in
  `internal/cli/logging_test.go` and `internal/host/http/limits_test.go`.
- JSON-shape production for `--log-format json` (§20 calls it optional).
- Honouring an explicit `logBodyBytes: 0` (undefined; see C-HOST-008
  observations) — recorded, not asserted.
- Any change to `cmd/guard` source: the per-package arch rules needed for
  C-ARCH-004/C-ARCH-005 are expressible in `.agent/guards.json` as it stands.
- Optional strengthening, not required by any criterion: adding these six
  catalog IDs to `trace.active` in `.agent/guards.json` would make the trace
  guard require a C-ID string marker in a test file for each one (the guard
  scans string literals in `_test.go`). It is supported today, but the
  objective does not ask for it, and it would oblige each new test to carry the
  ID in a string literal (e.g. a subtest name).
