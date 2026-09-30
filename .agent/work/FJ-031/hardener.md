---
stage: hardener
task: FJ-031
inputFingerprint: 969f6a5c976687fa98b89b4616a4622420c81a50470f4116d15ab1d3e71769b4
outputFingerprint: c8ee98ed15771f58cb8cecedd751633ac6c7bc4414a294da0db14807fda43338
taskFingerprint: 703164b4cd18ba33de0272b4065a7861e596afa85da13bcc16fd751fda500e6a
gitHead: 6094965
generatedAt: 2026-09-30T15:57:25Z
author: worker/FJ-031-hardener
---

# Hardener — FJ-031

**Final candidate fingerprint: `c8ee98ed15771f58cb8cecedd751633ac6c7bc4414a294da0db14807fda43338`**

Input candidate for this stage: `969f6a5c976687fa98b89b4616a4622420c81a50470f4116d15ab1d3e71769b4`
(the coder's output, unchanged by the cleaner).

Changed files: `internal/cli/serve.go`, `internal/cli/serve_test.go`,
`internal/host/http/server.go`. `internal/cli/dispatch.go` was inspected and
not modified. `.agent/work/FJ-031/state.json` and `.agent/reports/` were not
written by this stage; the candidate fingerprint was re-read after every
command and is the value pinned above.

## Hardening actions

### H-F2 — ready-file ownership cleanup is now on every exit path after publish

`serve()` previously removed the owned ready file only on the signal path; the
`serveErr` arm returned `exitFailure` with the file still in place, so a fatal
in-serve failure could leave a byte-valid document naming a dead pid — exactly
what `run` (FJ-032, §15.4 steps 3/9) waits on.

`serve` now has a named result and registers one deferred cleanup immediately
after `writeReadyFile` succeeds:

```go
defer func() {
        if err := removeReadyFile(readyPath); err != nil {
                logger.Printf("remove ready file: %v", err)
                status = exitFailure
        }
}()
```

Servers: §42.4 (ownership-bound removal, atomic file), §15.1 (file removed
during clean shutdown), §42.3 step 4, §42.1 (`0` success / `2` operational
failure). Properties preserved:

- The defer is registered *after* a successful publish, so the startup-failure
  arm (which runs before it) still leaves a pre-existing ready file
  byte-identical — §42.4 "overwrite only after successful readiness".
- Removal still goes through `removeReadyFile`, so the parsed-`pid` guard is
  untouched: a file another process replaced, or one that no longer parses,
  survives (§42.4, specifier I3).
- The named result keeps the previous exit semantics: an unexpected removal
  failure still yields exit `2` on the clean path, and cannot upgrade the
  fatal-error arm's `2`.
- Exactly one removal per run: the earlier explicit call after `Shutdown` was
  deleted, so no double-remove and no second `os.Stat`/`os.Remove` race.

### H-F1 — configuration failure names the file on the serve path

`serve --config bad.yaml` printed `fake-jev: mode "loose" is not supported; …`
while `validate bad.yaml` printed `fake-jev: bad.yaml: mode "loose" …`.
`resolveServeConfig` now wraps the `config.LoadFile` error with the path
(`fmt.Errorf("%s: %w", flags.config, err)`), giving byte-identical text to
`validate` for both the invalid-file and the unreadable-file cases:

```text
fake-jev: /tmp/bad.yaml: mode "loose" is not supported; only strict is valid
fake-jev: /tmp/absent.yaml: read configuration "/tmp/absent.yaml": open /tmp/absent.yaml: no such file or directory
```

The wrap sits at the load site rather than at the print site on purpose: a
print-site prefix would also have prefixed the flag-caused `server.host must
not be empty` rejection with a config path the flag, not the file, is
responsible for. Clauses: §42.1 (diagnostics for configuration failures),
§15.2 (validate's existing diagnostic shape as the consistency reference),
§20 (operator-facing diagnostics). `TestServeConfigFailuresAreNotUsageFailures`
now asserts the path appears in the diagnostic for both rows.

### H-F3 — the force-close test now reads a non-default grace value

`TestServeForceCloseAfterGracefulTimeout` configured `gracefulShutdownSeconds: 5`,
which is the §12.2/§38.3 default (`hosthttp.DefaultGracefulShutdown`), so both
existing observations held for an implementation that ignored the configured
value. The test now configures `10` (within §38.3's `1..60`) and observes the
process still draining after `6 s > 5 s`, the default it would have exited at
had the constant been used:

```go
const configuredSeconds = 10
const drainObservation = 6 * time.Second
...
select {
case code := <-child.exited:
        t.Fatalf("serve exited with %d before the configured %d s grace period elapsed", code, configuredSeconds)
case <-time.After(drainObservation):
}
if code := child.wait(t, time.Duration(configuredSeconds+10)*time.Second); code != 0 { … }
```

The lower bound is a bounded negative observation, not a synchronization
primitive (specifier H5), the upper bound stays `configured + 10 s`, and the
held connection is still required to end in EOF/reset. Clauses: §42.3 steps
2–3, §38.3 / §12.2 (`gracefulShutdownSeconds` is configuration, not a
constant).

### H-F7 — child cleanup asserts ready-file residue

`serveChild.stop` reaped the child but asserted nothing about files
(specifier H6). It now ends with `assertReadyFileGone`, and is skipped only
when cleanup itself had to kill a live child (no shutdown happened) or when the
launch declared `keepReadyFile`:

```go
if _, err := os.Stat(child.readyPath); !errors.Is(err, os.ErrNotExist) {
        t.Errorf("ready file %s survives serve shutdown (stat error: %v)", child.readyPath, err)
}
```

Two tests deliberately leave the file behind and declare it:
`TestServeStartupFailureLeavesReadyFileUntouched` (a pre-existing foreign
document must not be disturbed, §42.4) and
`TestServeReadyFilePidGuardOnShutdown` (a foreign/unparseable replacement must
survive, §42.4/§15.1). Every other launch must leave no residue. Servers: H6,
§42.4, §15.1.

### H-A1 — `AttachControl` states its ordering contract (comment only)

`Server.control` is written without synchronization. The shipped wiring
attaches before the listener opens, which is safe, but the contract was not
stated. One sentence was added to the existing doc comment: `AttachControl`
must be called before the host starts serving. No behavior change; §19.5/§41
(the control API shares the listener) is unaffected.

## Decisions on the numbered findings

- **F2** — treated as the real defect it is (a stale document naming a dead pid
  is the hazard §42.4's pid guard exists to avoid, and FJ-032 waits on that
  file). Fixed structurally with one deferred path rather than by duplicating
  the removal into the `serveErr` arm.
- **F1** — fixed at the loader call site; the diagnostic now matches `validate`
  byte-for-byte. Wording remains non-contractual (§42.5); only the presence of
  the path is asserted.
- **F3** — fixed by changing the configured value and the observation window,
  not by tightening the exit bound below the configured grace (specifier H5
  forbids tightening a bound to make a timing bug observable). Both directions
  still use generous margins.
- **F4** — not relaxed. `serve` keeps stdout empty, and the three "stdout is
  empty" assertions are untouched, per the ruling that §42.5 puts logs on
  stderr and §15.1's ready file is the readiness channel.
- **F5** — recorded, not acted on: FJ-041 needs `internal/cli/serve.go` /
  `internal/host/http/server.go` (its allowed set is `logging.go`,
  `logging_test.go`, `limits_test.go`) and FJ-032 needs `serve.go` or a
  host-owned lifecycle seam; neither file is reachable from those items today.
- **F6** — recorded, not tested: the nil-control `404 fake_jev_control_not_found`
  fallback would need `internal/host/http/server_test.go`, which is outside
  this item's allowed files.
- **F7** — fixed as above, including a declared exemption instead of an
  unconditional assertion, because two existing tests require the file to
  survive.
- **F8 (cleaner's small observations)** — audited, no real defect found; see
  the audit notes below.

## Audit of the remaining task checklist

- **Signals / double close / double remove.** `signal.Notify` is installed
  before readiness with a buffered channel, and `signal.Stop` is deferred.
  `Shutdown` then `Close` on timeout is the documented pairing; `Serve` closes
  the listener itself, so no path closes it twice. Removal happens exactly once
  (only the new defer). A second signal during the drain is still absorbed and
  ignored — §42.3 specifies only first-signal behavior, so it was left as is
  and is recorded below.
- **Drain uses the configured grace.** `timeout := time.Duration(cfg.Limits.GracefulShutdownSeconds) * time.Second`
  reads the validated effective configuration (default 5 filled by
  `config.Load`, range enforced by §38 validation). Confirmed by mutation: with
  the timeout replaced by `hosthttp.DefaultGracefulShutdown`, the H-F3 test
  reported `serve exited with 0 before the configured 10 s grace period
  elapsed`.
- **Goroutines and temp files.** The single `Serve` goroutine sends into a
  capacity-1 channel and exits when `Serve` returns; the process exits without
  a reader on the fatal arm, so no goroutine is stranded. `writeReadyFile`
  removes its temp file on every failure path (write, close, rename) and
  renames in the destination directory, so no `.fake-jev-ready-*` residue
  survives; `TestServeReadyFileIsAtomicAndClean` and `assertDirectoryEntries`
  keep observing that.
- **Atomic write.** `os.CreateTemp(filepath.Dir(path), ".fake-jev-ready-*")` +
  `os.Rename` in the destination directory, only after the listener is serving
  and the control API is attached — §42.4, §15.1.
- **Loopback default / ephemeral port.** `resolveServeConfig` rejects an empty
  host (fail closed, would bind the wildcard), the reported `port` is the bound
  `*net.TCPAddr.Port`, and `url` is built from the effective host and the bound
  port. Covered by the existing loopback, ephemeral-port and precedence tests.
- **stdout/stderr.** `serve` writes operational data only through the logger
  built on the passed `stderr` writer; `http.Server.ErrorLog` is the same
  logger; nothing reaches stdout (H-F4 keeps the assertions).
- **Authorization redaction on error paths.** The only request-derived log line
  is produced by `logHandler` (method, `URL.Path` only — never the raw query or
  headers — plus a constant marker when the header is present);
  `redactAuthorization` never returns the value. Server-level `ErrorLog` output
  carries transport errors only. The 413/404/500 control and data paths write
  bodies to the client, not to the log.
- **Control delegation.** `serveControl` delegates only after the control-plane
  limit has been applied and returns the previous 404 body when no handler is
  attached; `serveData`, `readBody`, and the per-plane limits are unchanged, so
  data-plane routing, journaling, outcomes, and isolation are preserved
  (`TestServeControlAPIReachableThroughListener`,
  `TestServePerPlaneBodyLimits`).

## Mutation checks used as evidence

Both were applied to a temporary copy and reverted; the candidate fingerprint
was re-read afterwards and is unchanged from the value pinned above.

1. `timeout := time.Duration(cfg.Limits.GracefulShutdownSeconds) * time.Second`
   replaced by `hosthttp.DefaultGracefulShutdown` →
   `TestServeForceCloseAfterGracefulTimeout` failed with
   `serve exited with 0 before the configured 10 s grace period elapsed`
   (the H-F3 assertion distinguishes configured from constant).
2. The whole deferred removal block deleted (the pre-fix F2 shape on every
   path) → `TestServeForceCloseAfterGracefulTimeout` and
   `TestServePerPlaneBodyLimits` both failed with
   `ready file …/ready.json survives serve shutdown` (the H-F7 cleanup check
   catches residue that the tests' own assertions do not look at).

Mutation 2 also exposed that the first version of H-F7 returned early when the
test had already reaped the child, which left the check inert; `stop` was
restructured so the residue check runs for any child that exited on its own.

## Command outcomes

All from `/Users/mak/git/fake-jev-FJ-031`, Go 1.26.3 (darwin/arm64), on the
final candidate.

```text
gofmt -w internal/cli/serve.go internal/cli/serve_test.go internal/cli/dispatch.go internal/host/http/server.go
gofmt -l <same files>                            no output
go test ./internal/cli -count=1                  ok   10.856s
go test ./internal/cli -count=2 -run 'TestServe|TestRemoveReadyFile|TestUsage'   ok   21.481s
go test ./internal/host/http -count=1            ok   0.519s
go test ./... -count=1                           ok   all 9 packages
go test -race ./... -count=1                     ok   all 9 packages
go vet ./...                                     no output, exit 0
go run ./cmd/guard arch                          0 findings (52 files, 9 packages)
go run ./cmd/guard lint                          0 findings (52 files, 9 packages)
go run ./cmd/guard trace                         0 findings (17 covered, 18 test files)
go run ./cmd/guard fuzz                          0 findings (0 targets)
go build ./cmd/fake-jev                          wrote ./fake-jev, removed before fingerprinting
./scripts/candidate-fingerprint candidate        c8ee98ed15771f58cb8cecedd751633ac6c7bc4414a294da0db14807fda43338
```

The candidate fingerprint was printed three times across the run and did not
change. `./scripts/verify-candidate` was not run by this stage: the task pinned
the command list above and directed that reports and `state.json` not be
written.

## Residual risks

- **F5 (boundary, unowned by this item).** FJ-041 needs `internal/cli/serve.go`
  and/or `internal/host/http/server.go` to reach the request body for §20
  truncation, and FJ-032 needs `internal/cli/serve.go` (or a host-owned
  lifecycle seam) to reuse the listener/signal/shutdown block for `run`. Those
  files are outside both items' allowed sets as they stand.
- **F6 (untested fallback).** The nil-control `404 fake_jev_control_not_found`
  fallback in `serveControl` is not pinned by a test because
  `internal/host/http/server_test.go` is outside this item's allowed files.
- **Fatal-error arm is structural, not exercised.** No external test can force
  `http.Server.Serve` to return an error without a seam (the listener is local
  to `serve`), so the H-F2 guarantee rests on the deferred path being
  unconditional; the clean-shutdown path that shares the defer is exercised.
- **Drain tests depend on the interim `100 Continue`.** If a future host stops
  emitting it, `TestServeGracefulShutdownDrainsInflight` and the force-close
  test need the specifier's recorded fallback seam (test-only synchronization).
- **Force-close test duration.** The configured 10 s grace makes that one test
  run ~10 s; the `./internal/cli` package now takes ~11 s (`-count=1`).
- **Second signal ignored.** A second SIGINT/SIGTERM during the drain is
  absorbed by the full channel with no reader; §42.3 defines only first-signal
  behavior, so this was left unchanged.
- **Residue check is bounded by design.** `assertReadyFileGone` is skipped when
  cleanup had to kill a live child, so residue produced by a child that never
  shut down is not reported.
- **`Shutdown` error is not logged** when the force-close `Close` succeeds
  (§42.5 makes wording non-contractual); recorded, not changed.
