---
stage: hardener
task: FJ-001
inputFingerprint: 4d87f2d06abe8969121119a69c0bc8f7c292249e97ece321d6dd43524da17524
outputFingerprint: 4d87f2d06abe8969121119a69c0bc8f7c292249e97ece321d6dd43524da17524
taskFingerprint: cc94ff6af6bccd0fa2ce59f0742e09743b6bf854d5c48af4097e7194f000ee93
gitHead: 13172c0
generatedAt: 2026-09-26T07:02:57Z
---

# Hardener — FJ-001

## Scope

The mutation/hardening analyzer is not implemented; this item is under the
recorded bootstrap exemption. Manual review of the four files against the
untrusted-input and fail-closed expectations of §42.1 and §17.2 was performed
instead. No code was changed.

## Findings

- **Untrusted input.** The only externally supplied values in this item are
  `args[0]`, which selects a command, and its length. Every unrecognized value
  reaches exactly one `default:` arm that formats the argument with `%q` and
  writes usage to stderr. `%q` escapes control characters, so a command name
  containing newlines, quotes, or terminal escapes cannot forge a second line of
  output. No value is ever passed to a shell, a filesystem path, or a formatter
  that would treat it as a verb.
- **Fail closed.** `Run` has no path that returns 0 for a request it did not
  understand. The empty-argument case and the unknown-command case both return
  `exitFailure` (2) and both print the usage block, so a caller scripting
  against exit status cannot mistake a mistake for success.
- **Write errors.** `runVersion` returns `exitFailure` when stdout rejects a
  write, so a closed pipe surfaces as exit 2 rather than a silent success. The
  usage and error writes in `Run` deliberately do not check their own errors:
  they are the failure path, and there is nowhere left to report to.
- **Panics.** No panic-capable construct is used: no type assertion, no map
  write, no slice write, no nil dereference on a value produced by this package.
  `debug.ReadBuildInfo` is called behind an `ok` check at both call sites.
- **No secrets, no logging, no network.** The binary opens no file, no socket,
  and no environment variable. §15.5's report is derived from build metadata
  compiled into the binary, so it cannot leak the environment of whoever ran
  it. FJ-041 owns Authorization redaction once a real log exists; there is
  nothing to redact yet.
- **Egress.** `go.mod` has no `require` block, so the compiled program links
  nothing that could open a connection.

## What remains

The analyzer-backed checks (FJ-045) and the fuzz targets named in the hardening
phase of §31 stay outstanding. This item introduces no untrusted parsing — the
first real parser is FJ-010 (configuration) and FJ-014 (request validation),
and those are where fuzzing and malformed-input handling actually apply.
