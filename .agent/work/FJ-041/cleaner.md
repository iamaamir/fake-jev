---
stage: cleaner
task: FJ-041
inputFingerprint: 66f95e12d85d159221cc39115bd577418fade40a183d160b33b495b5ec84f816
outputFingerprint: 66f95e12d85d159221cc39115bd577418fade40a183d160b33b495b5ec84f816
taskFingerprint: a39d771c0c282b258d73a682eeabd71d7ad9022cd4f0ae5a8c87b6911d2cef61
gitHead: e9be073
generatedAt: 2026-10-01T07:33:33Z
author: worker/FJ-041-cleaner
---

# Cleaner — FJ-041 (operational envelope hardening)

Scope: review-only pass over the coder's diff (`internal/cli/logging.go`,
`internal/cli/logging_test.go`, `internal/cli/serve.go`,
`internal/host/http/server.go`, `internal/host/http/limits_test.go`,
`.agent/guards.json`), with a security audit of the redaction and truncation
paths and an assessment of the reported `cmd/guard` fuzz flake.

## Structure changes applied

**None.** No product, test, guard, or artifact file other than this one was
modified by this stage. `./scripts/candidate-fingerprint candidate` reports
`66f95e12d85d159221cc39115bd577418fade40a183d160b33b495b5ec84f816` both before
and after the pass, and `git status` shows the same five modified/new product
files plus `.agent/guards.json` that the coder left. The fingerprint is a
declared contract for this stage, so every item below is recorded rather than
edited; none of them is a correctness defect that forces an edit, and each one
is either a comment/over-claim with no behavior, a defensive guard that is
unreachable through the only caller, or a test-coverage gap.

Re-verification of the unchanged candidate (see "Command outcomes"): all suites
green, `go vet` clean, arch guard clean, fingerprint unchanged.

## Redaction audit (`Authorization`)

Audited every stream the process can write: the serve/run stderr
`*log.Logger`, `net/http`'s `Server.ErrorLog`, stdout, the ready file, the
control API's `/__fake/v1/requests` document, and the request-body preview.

Observations:

1. The only production read of the header in the whole module is
   `internal/cli/logging.go:63` (`logRequest`); it is always passed through
   `redactAuthorization`, which returns `redactedHeaderValue` for any non-empty
   value. `grep -rn "Authorization" --include=*.go internal/ cmd/` matches only
   this file and two comments in `internal/compat/jev/v1/profile.go` that state
   headers are deliberately not inspected.
2. No other logging site records request data: the four diagnostics converted
   in `serve.go` (`startServer` failure, ready-file failure, serve error,
   remove-ready-file failure) format an `error` only; `handle.stop()` and
   `net/http`'s `ErrorLog` use the raw logger, which `net/http` writes to only
   for panics/TLS/listener noise and never with header material.
3. No response path echoes a request header: `serveControl`, `writeJSON`, and
   `writeEncoded` write only the host's own documents, and
   `stampRequestMetadata` records `method`/`path` only, so §41.7's request
   history cannot carry the header.
4. The error, debug, and rejection paths all funnel through the same
   `logRequest` call that applies redaction, and the truncation-marker path
   appends a constant (no interpolation of header data) — so the marker path is
   not an additional leak surface.
5. No panic path carries header material.

Body-preview caveat (not a header leak): the preview writes raw request-body
bytes to stderr up to `logBodyBytes`. A client that places a secret in the body
therefore sees it echoed, bounded and marked. §42.5's guarantee is scoped to the
header value and holds; §20's looser "do not log secrets by default" is not
enforced against payload content, and the specifier already recorded the preview
shape as spec-undefined. Restated under residual risks.

Log-injection observation: the preview (and, before this change, the method and
path) is written verbatim, so a body or percent-decoded path containing newlines
can forge additional `fake-jev: ` lines in the log stream. The single
`logger.Print` call per request keeps one request's bytes from interleaving with
another line, but a forged line is indistinguishable to a reader. The new
`awaitLogLines` helper counts occurrences of the literal `fake-jev: ` prefix,
so a fixture whose preview contains that string would inflate the count (not
the case in the new fixtures).

## Truncation audit

Determinism, unit, and boundary behavior probed through the real
`logHandler`/`captureBodyPreview` path (in a scratch copy outside the
repository; the candidate tree was not touched):

| case | observed preview | marker |
|---|---|---|
| body shorter than limit | full body | absent |
| body exactly at limit | full body | absent |
| body at limit+1 | first `limit` bytes | present |
| empty body | no `body=` field at all | absent |
| 2-byte rune straddling the boundary (limit 64) | 63 bytes + first byte of the rune (`\xc3`) | present |

- The cut is a byte slice (`buffered[:bodyLimit]`) with the boundary at
  `len(buffered) > bodyLimit`, so it is deterministic (repeated identical
  probes) and it **can** split a UTF-8 sequence; the emitted line was
  `utf8.ValidString == false` in the straddle probe. No panic, no data loss to
  the host: the drained prefix is replayed through `io.MultiReader` ahead of the
  remainder, so `Server.ServeHTTP` still sees every byte.
- No panic on short bodies: the slice only executes when `len(buffered) > bodyLimit`.
- Reachable panic: `captureBodyPreview(request, limit)` with a negative `limit`
  panics (`slice bounds out of range [:-1]`), because the clamp lives in
  `newOperationalLogger`, not in the helper. Unreachable today: the only
  constructor clamps to 0, `config.Validate` rejects negative `logBodyBytes`
  (§38.3), and `applyLimitDefaults` substitutes 4096 for 0, so
  `Server.Limits().LogBodyBytes >= 0` on every path. Hardening candidate only.
- A zero limit (reachable only via a hand-built logger) emits
  `body=...[truncated]` — an empty preview plus marker. Unreachable through
  `startServer`, where an effective 0 is substituted by 4096.
- The preview read is bounded by `logBodyBytes` alone, not by
  `min(logBodyBytes, plane limit)`: a configured `logBodyBytes` above a plane
  limit buffers up to `logBodyBytes+1` bytes per in-flight request before the
  §19.4 plane rejection. Bounded per configuration, but configurable memory
  amplification; and the helper comment's "bounded without allocating from an
  unbounded body" is not enforced by any test (see test-quality gap 3).

## Behavior preservation of the serve extraction

- Converted diagnostics are byte-identical: with prefix `fake-jev: ` and flags
  `0`, `logger.Printf("%v", err)` emits exactly `writef(stderr, "fake-jev:
  %v\n", err)`; the ready-file failure and `serve: %v` sites are equivalent too.
  Same writer (`stderr`), same branches, same return values (`exitFailure`), so
  messages, destinations, and exit codes are unchanged for both `--log-format`
  values.
- The extracted `logHandler` differs from the in-`serve.go` version in exactly
  one respect: it captures a bounded preview before delegating and appends
  `body=` to the line. That is the new requirement (C-CLI-010/H8d), not a
  preservation break, but it has one wire-visible consequence worth recording:
  for a request whose declared `Content-Length` exceeds its plane limit, the
  wrapper now reads up to `logBodyBytes+1` bytes before the host's early `413`,
  where the host previously rejected without reading. Status, journal outcome
  (`payload_too_large`, `requestBody: null`), and exit code are unchanged;
  reject-path timing is not, and with no `ReadTimeout` configured a client that
  declares an oversize body and stalls now blocks the wrapper in that read. The
  in-package `unreadableBody` tests exercise `Server.ServeHTTP` directly and so
  bypass the wrapper; no test pins "no read before rejection" through the serve
  path.
- Existing tests unmodified: `git status` shows `serve_test.go`, `run.go`, and
  `server_test.go` untouched; `internal/cli` and `internal/host/http` suites
  pass, including the child-process
  `TestServeLogsGoToStderrAndRedactAuthorization` for both formats.
- `run.go` still reaches the new wrapper through `startServer(cfg, logger)`
  (signature unchanged), so `run` also emits previews; `run.go` is outside
  `allowedFiles` and needed no edit.

## Duplication, naming, comments, accessor scope

1. No logic duplication between `serve.go` and `logging.go`: the extraction
   moved `logHandler`, `redactAuthorization`, and `redactedHeaderValue` without
   restating anything. `redactedHeaderValue` is now referenced only by the one
   function that returns it — harmless, but the constant's "wherever
   diagnostics would otherwise record it" phrasing now reads as broader than
   its single call site.
2. Duplication across commands: `run.go:139` and `run.go:168` still format the
   same two diagnostics as `writef(stderr, "fake-jev: %v\n", ...)` /
   `"fake-jev: serve: %v\n"`, while `serve.go` now uses `logger.Printf`. Output
   is identical; the two spellings sit in different files and `run.go` is
   outside `allowedFiles`.
3. Dead member with an over-claiming comment: `operationalLogger.Printf`
   (`logging.go:43-48`) is never called — `grep` for `.Printf(` in
   `internal/cli` non-test files shows only `logger.Printf`/`handle.logger.Printf`
   on the raw `*log.Logger`, and the only `newOperationalLogger` call site feeds
   `logHandler` alone. Its comment ("Lifecycle diagnostics go through here")
   describes a path that does not exist.
4. Type comment over-claim: "operationalLogger is the single sink for fake-jev
   operational output" and "serve and run construct it over stderr". Lifecycle
   lines in `serve()` and `stop()` and `net/http`'s `ErrorLog` all use the raw
   logger, so it is not a single sink; the redaction/truncation guarantee does
   not depend on the claim because it only needs to cover the request path.
   `newOperationalLogger`'s "serve and run construct it over stderr" is also
   loose: the constructor takes a `*log.Logger`, not a stream.
5. `logRequest` comment "Every path through the host — matched, unmatched,
   rejected, control, or unknown route — reaches this method" is accurate for
   requests that reach the wrapper, but the nil-logger branch deliberately skips
   the line, and the wrapper (not the host) is what sees the paths. Minor
   over-claim; `handler.next == nil` would panic, which is the same shape the
   pre-extraction code had and unreachable via `startServer`.
6. `captureBodyPreview` comment says "The bool reports whether the body was
   longer than the preview limit" and omits the read-error case, where it
   returns `false` together with a partial preview (the inline comment covers
   it).
7. `Server.Limits()` exposes five fields where `serve` consumes one. Judged
   against the smallest-necessary rule: it returns a value copy of an existing
   exported type, adds no mutation surface, carries no secrets, follows the
   `Engine()` precedent, and the identity test in `limits_test.go` consumes all
   five fields — so it is not more than the surrounding code needs. A narrower
   `LogBodyBytes() int` would not serve that test.

## `.agent/guards.json` scope

- `forbidExact` is a true exact import-path match (`cmd/guard/arch.go` builds
  `exact[imp]` as a map lookup; prefix matching is the separate `forbidPrefix`
  list). `"net"` therefore forbids exactly package `net` and does **not** forbid
  `net/url`, `net/textproto`, `net/netip`, or `net/http/httptest`; `"net/http"`
  stays a distinct entry.
- No rule covers `internal/host/http` (the server) or `internal/cli` (`run`
  needs `os/exec`, §15.4), so the required `net`/`net/http` imports there are
  unaffected; the question in the brief ("must keep importing net and
  net/http") has no constraint applied to it at all.
- Verified against the live import sets (`go list -f '{{.ImportPath}}: {{join
  .Imports " "}}'`): `internal/engine`, `internal/config`, and
  `internal/compat/jev/v1` intersect none of the banned paths;
  `go run ./cmd/guard arch` → `findings: []`, `files: 59`, `packages: 9`, exit
  0.
- Minimality: three rules, one shared `forbidExact` list, covering exactly the
  fixture/behaviour core (config load → profile decode → engine). Relative to
  the pre-change file the additions are `net`, `text/template`, `html/template`,
  `plugin` for `internal/engine` plus the two new package rules; `trace`,
  `lint`, `complexity`, `mutation`, and `fuzz` sections are untouched.
- One low-probability future conflict: the exact `net` ban on `internal/config`
  and `internal/compat/jev/v1` would also forbid a direct `net.ParseIP` /
  `net.SplitHostPort` host-validation helper in those packages. No current or
  specified requirement needs one (§38 gives `server.host` no format rule, and
  the non-empty host check lives in `internal/cli`), and subpackage imports such
  as `net/url` remain legal, so this is recorded as a scope note rather than a
  defect.

## Test quality

- Deterministic and hermetic: the new tests use an ephemeral loopback listener
  (stopped via `t.Cleanup`) or in-process `ServeHTTP`/`httptest`; no egress, no
  randomness, no clock-dependent assertions other than bounded deadlines, no
  shared mutable fixtures.
- `awaitLogLines` is a bounded poll (1 ms interval, 5 s deadline) that fails
  with the observed stream. The polling is necessary because the operational
  line is written after the response reaches the client, so it is not
  sleeps-as-sync — but it is the one wall-clock wait in the new tests and the
  only place they could flake on a pathologically loaded machine.
- Falsifiability verified by mutation in a scratch copy (repository untouched):
  1. removing the byte cut and the marker → `TestOperationalLogsTruncateBodyPreview`
     fails at the dropped-sentinel assertion (`logging_test.go:172`) and the
     marker assertion (`:176`);
  2. removing only the marker → same test fails at `:176`;
  3. returning the header value from `redactAuthorization` → both new tests
     fail at the token assertions (`logging_test.go:179` and `:242`);
  4. disabling the plane-limit early rejection and changing
     `DefaultLogBodyBytes` → `TestLimitsFromDefaultConfigMatchSection122`
     (`limits_test.go:46`), `TestReadBodyEnforcesDefaultPlaneLimits` (`:108`),
     and `TestServerEnforcesSection122DefaultPlaneThresholds` (`:151`) fail.
- Gaps recorded, none of which is a criterion miss:
  1. no test falsifies an unbounded preview *read*: a variant that reads the
     whole body and then cuts to `logBodyBytes` keeps
     `TestOperationalLogsTruncateBodyPreview` green, so the comment's
     "bounded without allocating from an unbounded body" is unverified even
     though the emitted bound is enforced;
  2. `TestDefaultMaxInteractionsIsEnforced` and the `Default*`-parameterised
     cases are self-consistent with the constants they read, so only
     `section122Limits()` (locally restated) catches constant drift — which is
     the intended design and does fail on drift;
  3. `TestDefaultBindHostIsLoopback` asserts the default port `8787`, which is
     outside C-ARCH-006 but already pinned by the `/meta` child-process test.

## `cmd/guard TestRunFuzzPass` flake — assessment (recorded, not fixed)

Determination: real and environment-triggered, in the guard's fuzz runner
(`cmd/guard`, outside `allowedFiles`), not in product code.

- Reproduced deterministically in this workspace by constraining resources
  rather than by load:
  `bash -c 'ulimit -n 64; go test ./cmd/guard -run TestRunFuzzPass -count=1 -v'`
  yields `findings=[{guard.fuzz.crash p/fuzz_p_test.go 5 fuzz target FuzzSafe
  failed}]` and `--- FAIL: TestRunFuzzPass` — the identical code/path/message
  shape the coder recorded, for a target whose body is empty
  (`f.Fuzz(func(t *testing.T, s string) {})`) and therefore cannot crash.
- Mechanism: the same `go test -fuzz=FuzzSafe -fuzztime=1s` under that limit
  prints `--- FAIL: FuzzSafe (0.01s)` / `pipe: too many open files` with no
  failing input written. `runOneFuzz` (`cmd/guard/fuzz.go:280-300`) classifies
  any non-zero `go test -fuzz` exit whose combined output contains
  `--- FAIL: <TargetName>` or the seed-corpus marker as a `guard.fuzz.crash`
  finding against the declaring source, so a fuzzing-harness failure is
  indistinguishable from a genuine crashing input. Go reports worker
  start/communication/termination failures as `--- FAIL: <TargetName>` too
  (`internal/fuzz/worker.go:145,186`; the worker liveness window is
  `workerTimeoutDuration = 1s`), which is the load-sensitive path.
- Adjacent weakness in the same classifier: `fuzzOutput.truncated` is set but
  never consulted, so a tool error whose explanation lies past the 1 MiB cap is
  classified from the retained prefix; and there is no retry for a failure that
  preserved no input.
- A load-only reproduction attempt did not trigger it: 32 `TestRunFuzzPass`
  invocations (up to 8 concurrent) plus 12 CPU burners all exited 0, which is
  consistent with "observed once" and with a resource-threshold trigger rather
  than a deterministic defect.
- Impact on this candidate: none observed. `go test ./... -count=1`,
  `go test -race ./... -count=1`, and `go run ./cmd/guard arch` all completed
  with exit 0 in this session.

## Command outcomes

| command | outcome |
|---|---|
| `gofmt -l` on the five changed/new Go files | no output |
| `go test ./internal/cli -count=1` | `ok` 19.885s |
| `go test ./internal/host/http -count=1` | `ok` 0.624s |
| `go test -race ./... -count=1` | all packages `ok` (guard 9.7s, cli 24.7s, host/http 5.4s, integration 5.2s) |
| `go test ./... -count=1` | all packages `ok` |
| `go vet ./...` | exit 0, no output |
| `go run ./cmd/guard arch` | `findings: []`, `files: 59`, `packages: 9`, exit 0 |
| `./scripts/candidate-fingerprint candidate` | `66f95e12d85d159221cc39115bd577418fade40a183d160b33b495b5ec84f816` (unchanged) |
| scratch-copy mutation probes (truncation, marker, redaction, plane limits) | each mutation produced the expected test failure; scratch copies deleted, repository untouched |
| `bash -c 'ulimit -n 64; go test ./cmd/guard -run TestRunFuzzPass -count=1 -v'` | fails with `guard.fuzz.crash … FuzzSafe failed` (flake reproduction) |

`.agent/reports/` and `.agent/work/FJ-041/state.json` were not modified;
`git diff --cached` is empty.

## Residual risks (recorded for the hardener)

1. Body-preview bytes reach stderr verbatim up to `logBodyBytes`, so payload
   secrets are echoed (bounded, marked) and newline-bearing bodies/paths can
   forge log lines.
2. The preview read is bounded by `logBodyBytes` only — no clamp to the plane
   limit — and no test falsifies an unbounded preview read.
3. `captureBodyPreview` panics on a negative limit; unreachable through the only
   constructor, which clamps, but not defended in the helper.
4. The wrapper now reads up to `logBodyBytes+1` bytes before the host's early
   `413`, with no `ReadTimeout` on the serve `net/http.Server`.
5. `operationalLogger.Printf` is dead and its comment, the type comment's
   "single sink", and `logRequest`'s "every path" wording over-claim.
6. `cmd/guard` `TestRunFuzzPass` misclassifies fuzzing-harness failures as
   `guard.fuzz.crash` (follow-up ticket; reproduce with `ulimit -n 64`).
7. The arch rules leave `internal/host/http` and `internal/cli` outside the
   structural no-egress proof (the specifier's documented honest limit), and the
   exact `net` ban in `internal/config` / `internal/compat/jev/v1` would block a
   future direct-`net` host-validation helper.
