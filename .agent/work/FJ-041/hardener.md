---
stage: hardener
task: FJ-041
inputFingerprint: 66f95e12d85d159221cc39115bd577418fade40a183d160b33b495b5ec84f816
outputFingerprint: 3d896a0a7d7ae6bd6f347b4166a74c6d23bbf59a384869c2cc3c2697e96e9dcc
taskFingerprint: a39d771c0c282b258d73a682eeabd71d7ad9022cd4f0ae5a8c87b6911d2cef61
gitHead: e9be073
generatedAt: 2026-10-01T07:46:40Z
author: worker/FJ-041-hardener
---

# Hardener — FJ-041 (operational envelope hardening)

## FINAL CANDIDATE FINGERPRINT

```text
3d896a0a7d7ae6bd6f347b4166a74c6d23bbf59a384869c2cc3c2697e96e9dcc
```

Produced by `./scripts/candidate-fingerprint candidate` after every edit in this
stage. `task` remains
`a39d771c0c282b258d73a682eeabd71d7ad9022cd4f0ae5a8c87b6911d2cef61` and the
revision is `e9be073`.

## Files touched by this stage

- `internal/cli/logging.go` — preview mechanism replaced; dead member removed;
  comments corrected; log-safe rendering added.
- `internal/cli/logging_test.go` — four new tests (no-read-ahead, marker
  boundary, forgery escaping, split-rune validity) plus the oversized-request
  regression test.

`internal/cli/serve.go`, `internal/host/http/server.go`,
`internal/host/http/limits_test.go`, and `.agent/guards.json` were read and
audited but not modified by this stage; they carry the coder's changes.
`run.go`, `serve_test.go`, `server_test.go`, and `state.json` are untouched.
`.agent/reports/` was not written; `state.json` was not edited.

## 1. Wire-visible regression — finding and resolution

**What changed.** The coder's `logHandler.ServeHTTP` called
`captureBodyPreview(request, handler.logger.bodyLimit)` *before* delegating to
the host. That helper did
`io.ReadAll(io.LimitReader(request.Body, int64(bodyLimit)+1))` and replayed the
drained prefix through an `io.MultiReader`. Every served request therefore read
up to `logBodyBytes+1` bytes before the host ran. The host's §19.4/§43.7 early
rejection keys off `request.ContentLength > planeLimit` and returns without
reading: after the change, an oversized request consumed and buffered
`logBodyBytes+1` (default 4097) bytes before that rejection, changing read
timing and, for a configured `logBodyBytes` above the plane limit, buffering up
to `logBodyBytes+1` bytes per in-flight request. Status, journal outcome, and
exit code were unchanged (the cleaner confirmed this); the wire-visible part is
the pre-rejection read.

**Resolution (inside `allowedFiles`, no `run.go` edit needed).** `logging.go`
now captures the preview *passively* instead of reading ahead:

- `captureBodyPreview` replaces `request.Body` with a `bodyPreview` observer that
  wraps the real body and records only the bytes the host consumes while it
  reads (`Read` → `record`). It performs **zero** reads itself.
- The host therefore still sees an unread body for a declared oversize and
  rejects with the exact §43.7 body before any byte moves. For a streamed
  oversize the host reads its usual `planeLimit+1` bytes to detect the overflow;
  the observer simply records the first `logBodyBytes` of them.
- Because the buffer only advances as the host reads, the preview is bounded by
  `min(logBodyBytes, bytes the host read)` and can never exceed the plane limit's
  read — the configurable memory amplification the cleaner flagged is gone.

`internal/cli/run.go` did **not** need a change: `run` reaches the same
`startServer`/`logHandler` wiring and inherits the fix.

**Evidence.**
`TestOversizedDataPlaneRequestIsRejectedWithoutReadingBody` runs a declared
oversize (`DataPlaneBodyBytes+1`) `POST /v1/systemone` through the real
`logHandler` around a `hosthttp.Server` built from the minimal config, using a
body that counts every `Read`. It asserts:

- status `413` and the §43.7 JSON body
  (`{"error":"fake_jev_payload_too_large","message":"Request body exceeds the
  configured limit."}`);
- the body was never read (`reads == 0`);
- the journal holds exactly one record: `sequence == 1`, `outcome ==
  payload_too_large`, `RawBody == nil` (§43.7, §44.15);
- no configured stub's `InvocationCount` moved.

`TestCaptureBodyPreviewDoesNotReadAhead` asserts the observer reads nothing at
install time and that the host still receives every byte through it.

**Falsifiability (run in this workspace, then reverted).** Re-adding a single
read-ahead byte inside `captureBodyPreview` made both tests fail
(`captureBodyPreview read the body 1 times before the host` and `oversize body
was read 1 times before the 413 rejection`); the file was restored and
`gofmt -l`/`git status` confirmed no residue.

## 2. Preview robustness (`captureBodyPreview`)

- **Negative / zero limit.** `captureBodyPreview` clamps a negative limit to
  zero; `bodyPreview.record` computes `room := limit - len(buffer)` and flags
  truncation when `room <= 0` *before* any slice, so a zero or negative limit can
  never panic. `TestBodyPreviewMarkerBoundary` exercises limits `4`, `0`, and
  `-1`.
- **Byte-versus-rune semantics (chosen, spec-silent — recorded as an
  observation).** The cut is **by bytes**, because §19.4's `logBodyBytes` is a
  byte count and a byte cut is deterministic. A cut that lands inside a UTF-8
  sequence is therefore possible; it is made safe at render time, not by
  backing the cut up to a rune boundary. `TestBodyPreviewCutSplittingRuneStaysValid`
  pins this (`"ab€x"` cut at 3 bytes renders `"ab\xe2"`).
- **No invalid output / no log forgery.** `renderBodyPreview` renders the
  captured bytes with `strconv.Quote`, so newlines, CR, TAB, NUL, and other
  control characters become Go escapes and any dangling partial UTF-8 byte
  becomes `\xNN`. One request therefore still produces exactly one `fake-jev: `
  line and cannot inject a forged one. `TestBodyPreviewEscapesLogForgeryBytes`
  feeds `ok\nfake-jev: forged line\r\x00\tend` and asserts none of `\n`, `\r`,
  `\x00`, `\t` appears raw and the output is valid UTF-8.

## 3. Dead member and comment corrections

- `operationalLogger.Printf` (dead, never called) was removed.
- The type comment's "single sink for fake-jev operational output" and
  "serve and run construct it over stderr" were replaced with what the code
  guarantees: one served-request line per request, redaction and truncation on
  that path, lifecycle and `net/http` diagnostics on the raw `*log.Logger`, and
  nothing written to stdout.
- `logRequest`'s "Every path through the host … reaches this method" was
  replaced with the truthful scope: the wrapper reaches it once per request the
  host handler sees (matched, unmatched, rejected, control, unknown route), and a
  request `net/http` rejects before the handler, a nil logger, or a nil request
  produces no line.
- The `redactedHeaderValue` comment no longer claims a broader "wherever
  diagnostics would record it" coverage than its single call site.

## 4. Truncation-marker exactness

The marker is appended by `logRequest` iff the observer's `truncated` flag is
set, which happens iff the host read at least one byte past `logBodyBytes`.
`TestBodyPreviewMarkerBoundary` pins the exact boundary: shorter than limit →
no truncation; **exactly at limit** → no truncation; **limit+1** → truncation
with the exact first-`limit` bytes; **empty** → no preview and no truncation.
Falsifiability: suppressing the `truncated = true` assignment on overflow made
the `one over limit` case fail; reverted.

## 5. `.agent/guards.json` re-check (C-ARCH-004 / C-ARCH-005)

- `go run ./cmd/guard arch` → `{"check":"arch","findings":[],"stats":{"files":59,"packages":9}}`,
  exit 0, on the whole repository.
- `cmd/guard/arch.go` keys each rule by module-relative package directory
  (`rule.Pkg != rel`), so only `internal/engine`, `internal/compat/jev/v1`, and
  `internal/config` are covered; `internal/host/http` and `internal/cli` have no
  rule and are unaffected. Their required imports (`internal/host/http`:
  `net`, `net/http`; `internal/cli`: `net`, `net/http`, `net/url`, `os/exec`)
  are not matched by any rule.
- Import set vs. the rule (`go list -f '{{join .Imports "\n"}}'`): none of the
  three ruled packages imports any banned path today, so no rule fails.
- Minimality: `net` and `net/http` are the two egress entry points (exact match
  is per-path, so `net` does not cover `net/http`); `os/exec`, `text/template`,
  `html/template`, and `plugin` are the structural mechanisms for "shell
  commands", "Go templates", and "dynamic eval / arbitrary code callbacks"
  (§19.3). `html/template` must be listed separately from `text/template`
  because the guard matches direct imports. No rule was added, removed, or
  weakened in this stage; `forbidPrefix` and the `trace`/`lint`/`complexity`/
  `mutation`/`fuzz` sections are unchanged from the coder revision.

## 6. Audit of the allowed files

- **Authorization redaction on every path.** `logRequest` is the only production
  read of the header and always passes it through `redactAuthorization`. The
  truncation marker is a constant appended with no header interpolation. Error
  paths (413, 501, 507, validation, unknown route) all reach the wrapper, which
  logs after the response, so redaction holds on failure paths too. The observer
  records body bytes only; the header never enters the preview. `serve`'s
  lifecycle lines format only errors and the bound URL.
- **stdout/stderr discipline.** No new writer was introduced; the operational
  logger writes to the `*log.Logger` the command already built over stderr, and
  nothing in `logging.go` touches stdout. The unmodified child-process
  `TestServeLogsGoToStderrAndRedactAuthorization` still pins `stdout == ""` and
  the token's absence for both `--log-format` values.
- **No unbounded allocation before the limit check.** The observer allocates only
  as the host reads and only up to `logBodyBytes`; the host's `readBody` still
  checks `ContentLength` before reading, so an oversize declared body allocates
  nothing. `renderBodyPreview` allocates proportional to the bounded buffer.
- **No goroutine or file leaks.** No goroutine or file handle was added. The
  observer's `Close` forwards to the wrapped body, and the host's deferred
  `request.Body.Close()` captures the observer at defer-statement time (verified
  with a standalone probe), so the real request body is still closed exactly
  once. `-race` is clean.

## Command outcomes

| command | outcome |
|---|---|
| `gofmt -w` on `internal/cli/logging.go`, `internal/cli/logging_test.go`; `gofmt -l internal/cli internal/host/http` | no output |
| `go test ./internal/cli -count=1` | `ok` 22.4s |
| `go test ./internal/host/http -count=1` | `ok` 0.6s |
| `go test ./... -count=1` | all packages `ok` (guard 7.8s, cli 22.4s, compat 1.2s, config 1.7s, control 2.2s, engine 3.1s, host/http 3.4s, integration 2.6s) |
| `go test -race ./... -count=1` | all packages `ok` |
| `go vet ./...` | exit 0, no output |
| `go run ./cmd/guard arch` | `findings: []`, `files: 59`, `packages: 9`, exit 0 |
| `go run ./cmd/guard lint` | `findings: []`, `files: 59`, `packages: 9`, exit 0 |
| `go run ./cmd/guard trace` | `findings: []`, `active: 0`, `covered: 17`, `test_files: 22`, exit 0 |
| `go run ./cmd/guard fuzz` | `findings: []`, `targets: 0`, exit 0 |
| `go build ./cmd/fake-jev` | exit 0 |
| `./scripts/candidate-fingerprint candidate` | `3d896a0a7d7ae6bd6f347b4166a74c6d23bbf59a384869c2cc3c2697e96e9dcc` |
| falsification probes (read-ahead, truncation flag) | each made the new tests fail as expected; files restored |

`./scripts/verify-candidate FJ-041` was not run: it writes to `.agent/reports/`,
which this stage was instructed not to touch.

## Residual risks / observations

1. **`cmd/guard TestRunFuzzPass` flake — owned by a separate ticket.**
   Deterministic reproduction recorded by the cleaner:
   `bash -c 'ulimit -n 64; go test ./cmd/guard -run TestRunFuzzPass -count=1 -v'`
   yields `findings=[{guard.fuzz.crash p/fuzz_p_test.go 5 fuzz target FuzzSafe
   failed}]` for an empty target. Classifier analysis: `runOneFuzz`
   (`cmd/guard/fuzz.go:~280-300`) classifies any non-zero `go test -fuzz` exit
   whose output contains `--- FAIL: <TargetName>` (or the seed-corpus marker) as
   `guard.fuzz.crash`; Go reports fuzzing-worker start/communication/termination
   failures with the same `--- FAIL: <TargetName>` shape, so a harness failure is
   indistinguishable from a genuine crashing input. Adjacent weakness:
   `fuzzOutput.truncated` is set but never consulted, and there is no retry when
   no input was preserved. `cmd/guard` is outside `allowedFiles` and was not
   touched; `go test ./... -count=1`, `-race`, and `go run ./cmd/guard fuzz` all
   exited 0 in this session.
2. **Preview content is payload.** A client secret placed in the request body is
   still echoed (bounded, quoted, marked) to stderr. §20's hard rule is the
   Authorization header value, which is redacted; `"do not log secrets by
   default"` is not enforced against payload content, and the preview shape is
   spec-undefined.
3. **Method/path are still logged verbatim.** A percent-decoded path containing
   a newline could in principle forge a line; this is pre-existing (method/path
   were logged before FJ-041), is not part of the enumerated preview fix, and was
   left unchanged to avoid widening scope. It is recorded here as an observation.
4. **`logBodyBytes: 0` remains spec-undefined.** The host substitutes 4096
   (`applyLimitDefaults`), the logger uses that resolved value, and
   `/__fake/v1/meta` still reports the configured `0` — the pre-existing
   `/meta`-vs-enforcement divergence, unchanged.
5. **`--log-format json` remains a validated no-op** (pre-existing; §20 calls
   structured logging optional).
6. **C-ARCH-004 remains structural only.** `internal/host/http` and
   `internal/cli` sit outside the guard by design (the server and the `run`
   child-process spawner); the call-site audit from the specifier/cleaner is
   unchanged.
7. **Semantics decision recorded as an observation, not a spec finding:** the
   byte-cut plus Go-quoted rendering in item 2 is a chosen rendering because the
   specification does not define the preview's shape, and no test asserts log
   wording (§42.5).
