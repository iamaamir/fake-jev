---
stage: hardener
task: FJ-030
inputFingerprint: 5fd7ac44377d157f72a1433afb3f7cffd14a622f092c3aef91e72c9d7ccb0b48
outputFingerprint: bad76f7917d526d0978fb613aee212f2345ff06a717259788d4a04366f01ab19
taskFingerprint: d37327aa23e2afc94bc9b8bbc7d4c9cd3a87d61e03dc2cc2605c258b8b6c34a7
gitHead: abba008
generatedAt: 2026-09-30T15:11:03Z
author: worker/FJ-030-hardener
---

# Hardener — FJ-030: `fake-jev validate <path>`

## Final candidate fingerprint

```text
bad76f7917d526d0978fb613aee212f2345ff06a717259788d4a04366f01ab19
```

Changed from the stage input `5fd7ac44…0b48` by the three edits below, all
inside `allowedFiles` (`internal/cli/dispatch.go`,
`internal/cli/validate_test.go`). `internal/cli/validate.go` and
`internal/config` are byte-identical to the coder revision; `.agent/work/` and
`.agent/reports/` are excluded from the digest by
`scripts/candidate-fingerprint`. Nothing was staged (`git status --short`:
one modified tracked file plus untracked work-item files).

## Hardening actions taken

### F2 — narrowed the no-network assertion to the validate path (applied)

`TestValidatePathLinksNoNetworkingPackage` previously walked every non-test
`.go` file in `internal/cli` and `internal/config` and failed on any `net` or
`net/...` import. That is wider than the clause it serves: §15.2 requires that
`validate` make no network request and open no listener, and §19.2 requires the
fake to run with outbound networking disabled — neither says anything about the
imports of unrelated CLI files. FJ-032 (`verify --url`) is expected to bring
`net/http` into `internal/cli`, at which point the old assertion would have
reported a failure for a file that has nothing to do with `validate`.

The assertion now walks the *import closure of `validate.go`* — the only
production file implementing the validate path — and fails if any file on that
path imports `net` or a `net/...` package. Concretely: seed `validate.go`;
follow imports under the module prefix `fake-jev/` into the corresponding
package directory (all non-test files of that package, since a package links as
a unit); record an error for any networking import. Unrelated files in
`internal/cli` are no longer constrained, because they are only reached if
`validate.go` imports them.

Clauses served: §15.2 ("MUST NOT start a server", "MUST make no network
request"), §19.2. The property is preserved, not deleted: `validate` still has
to prove that it — and the loader it calls — cannot dial or listen.

Two-directional check of the narrowed assertion (temporary edits, reverted
immediately; `validate.go` and `internal/config/load.go` are back to their
committed/unedited bytes, confirmed by diff and `git status`):

| Injected change | `go test ./internal/cli -run TestValidatePathLinksNoNetworkingPackage` |
|---|---|
| `net/http` added to `internal/cli/dispatch.go` (the FJ-032 shape: unrelated file in the same package) | `ok` — no longer a false alarm |
| `net` added to `internal/cli/validate.go` | `FAIL`: `validate.go imports "net": the validate path must not be able to dial or listen` |
| `net` added to `internal/config/load.go` | `FAIL`: `../../internal/config/load.go imports "net": …` — the closure really does reach the shared loader |

Fail-closed behavior kept: a missing `validate.go` or a package directory with
no production files is a `t.Fatal`, not a silent pass.

### F1 — stale doc comment (applied)

The doc comment above `TestValidateStreamsAreDisjoint` named
`TestValidateSuccessGoesToStdoutAndFailuresToStderr`. The comment now names the
function it documents and keeps the §42.5 reference:

```go
// TestValidateStreamsAreDisjoint pins the stream split (§42.5): a run writes
// a success line only on stdout, and a rejection writes only on stderr.
```

### F4 — usage text now shows the required positional argument (applied)

`dispatch.go` usage block:

```text
commands:
  version          print version, build, profile, and contract information
  validate <path>  validate a configuration file without starting a server
  help             print this message
```

The two-column block was re-padded so the descriptions stay aligned (the
`validate <path>` entry is the widest). The listed commands remain exactly the
arms of `Run`. The test that reads the command block now asserts the
`validate <path>` spelling, so the usage and the command cannot drift apart
again. Serves §15.2's documented invocation form
(`fake-jev validate ./fake-jev.yaml`) and §42.5 (a usage failure is still
diagnostic text on stderr). Output wording is not a contract (§15.2: "Success
output MAY be human-readable, but exit semantics are normative"), so this is
presentation only.

### Boundary coverage added inside `validate_test.go` (applied)

Every case below is an exit-code / stream-discipline property, not new
behavior: the implementation already produced these outcomes, and the audit
found no defect to fix, so the audit is banked as tests.

| Added | Input | Asserted |
|---|---|---|
| table case `directory path` | `validate <tempdir>/adir` (a directory) | exit 2, stdout empty, stderr diagnostic, no usage text (a read failure is not a usage failure) |
| table case `utf-8 bom with unknown key` | UTF-8 BOM then `schemaVersion: 1` + `extra: true` | exit 2, stdout empty, stderr diagnostic — the BOM cannot make an invalid document pass |
| table case `help flag after command` | `validate --help` | exit 2 + usage on stderr: §15.2 defines no flags for `validate`, so §42.1 makes a flag a usage failure |
| `TestValidateRejectsUnreadableFile` | file created `0o000` | exit 2, stdout empty, stderr carries a `read configuration` diagnostic; skipped when `os.Geteuid() == 0`, where permission bits do not deny reads (CI runs `ubuntu-latest` as a non-root user, so the case executes there) |
| `TestUsageListsValidateAsACommand` extended to `help`, `-h`, `--help` | each help entry point | exit 0, usage text on stdout, stderr empty, and the `validate <path>` command line present |

## Audit of the rest of the candidate (no code change required)

Measured on the built binary (`go build -o /tmp/fjv ./cmd/fake-jev`) and by the
suite. Exit codes are the §42.1 vocabulary; `exitVerify` (3) is unreachable
from `validate`.

| Input | stdout | stderr (first line) | exit |
|---|---|---|---|
| valid YAML / JSON | `fake-jev: <path>: configuration is valid` | empty | 0 |
| valid YAML with a leading BOM | same success line | empty | 0 |
| `schemaVersion: 2` | empty | `fake-jev: <path>: schemaVersion must equal integer 1` | 2 |
| missing file | empty | `fake-jev: <path>: read configuration "<path>": open …: no such file or directory` | 2 |
| directory path | empty | `… read <path>: is a directory` | 2 |
| unreadable file (`0o000`) | empty | `… open <path>: permission denied` | 2 |
| empty file | empty | `fake-jev: <path>: configuration is empty` | 2 |
| BOM + unknown key | empty | `fake-jev: <path>: decode configuration: json: unknown field "extra"` | 2 |
| extra positional argument | empty | `fake-jev: validate takes exactly one configuration path` + usage | 2 |
| unknown flag (before or after the path) | empty | `fake-jev: unknown flag "--strict"` + usage | 2 |
| `validate --help` | empty | `fake-jev: unknown flag "--help"` + usage | 2 |
| `validate -` | empty | `fake-jev: -: read configuration "-": open -: no such file or directory` | 2 |
| `validate ""` | empty | `fake-jev: : read configuration "": open : no such file or directory` | 2 |
| `help` / `-h` / `--help` | usage text | empty | 0 |
| no arguments | empty | usage text | 2 |
| unknown command | empty | `fake-jev: unknown command "frobnicate"` + usage | 2 |

- **Exit-code correctness**: only 0 and 2 are produced; no failure path returns
  3, and no failure path returns 0. Confirmed by the table test's
  `code != exitFailure` assertion for all 23 rejection cases and by the matrix
  above.
- **Stream discipline**: the success line is the only thing ever written to
  stdout, and it is written only after `config.LoadFile` succeeded. Every
  diagnostic, every usage text, and the "no arguments" usage all go to stderr.
  The table test asserts `stdout == ""` on every rejection.
- **No listener, no dial**: structural (the closure test above, which also
  covers the shared `internal/config` loader) plus behavioral
  (`TestValidateOpensNoListener` holds an ephemeral loopback port equal to the
  configured `server.port` and observes success for a valid file and the
  ordinary configuration diagnostic for an invalid one). No code path on the
  validate route constructs a client or a listener.

## Decisions recorded, not applied

- **BOM-prefixed configuration is accepted when the document is otherwise
  valid** (verified: exit 0 for `\xef\xbb\xbfschemaVersion: 1`). YAML 1.2 permits
  a leading BOM and RFC 8259 says a JSON parser MAY ignore one, so accepting is
  spec-compatible; §15.2 is silent on encoding marks. Because the spec does not
  define this either way, the test added above pins only the fail-closed
  direction (BOM + invalid document → 2) rather than freezing acceptance as a
  contract.
- **`validate --help` is a usage failure (exit 2), not help.** §15.2 defines no
  flags for `validate`, and §15 defines help only as the `help` entry point, so
  an unknown flag is the fail-closed reading of §42.1. Pinned by the new table
  case; if a future item wants per-command help, that is a specification change.
- **`-` is a filename, not stdin**, and `--` is an unknown flag. §15.2 defines
  no stdin mode and §15 forbids prompts, so both read as ordinary input errors
  (exit 2) rather than new behavior. Consequence: a file whose name begins with
  `-` cannot be validated without renaming it; no spec clause requires an end of
  options marker.
- **No cap on the configuration file read.** `config.LoadFile` is
  `os.ReadFile`, so `validate /dev/zero` (and `serve --config /dev/zero`) reads
  until the process is killed — verified: `timeout 5` returns 124. Nothing on
  the validate path streams or bounds the input. Closing this needs a size
  limit, which the specification does not define and which would change
  strictness on the shared `config.LoadFile` surface used by the data plane and
  the control API; it cannot be done inside `allowedFiles` without forking the
  loader. Recorded here rather than patched.
- **A FIFO path blocks** for the same reason (a read that waits for a writer).
  Same class as the previous item, same handling.
- **`writef`'s `if err != nil { return }` in `dispatch.go` is a no-op branch.**
  It is present in `HEAD`, unchanged by this candidate, and is the checked-and-
  consumed shape that lint rule L1 asks for; not touched.
- **The success line is not a contract** (§15.2) and remains
  `fake-jev: <path>: configuration is valid`; a failed stdout write is treated
  as an internal operational failure (exit 2), matching `runVersion`.

## §15.2 gap ownership

The two statically decidable §15.2 checks that the shared validator does not
perform — **G1** (`jev/v1` helper payloads: `noul` range/finiteness, `choice`
and `score` probability rules, prohibited `legend` in v1 score fixtures) and
**G2** (`when.questions` versus `then.answers` key-set/type cross-check) — are
untouched by this stage. They require new logic in `internal/config` or an
exported static entry point in the compat layer, which is outside
`allowedFiles`, and they change strictness on the shared `config.Load` surface
used by `POST /__fake/v1/stubs`. Per the recorded ownership split in
`state.json`, both are owned by **FJ-059**.

## Commands run and outcomes

| Command | Outcome |
|---|---|
| `gofmt -w internal/cli/validate.go internal/cli/validate_test.go internal/cli/dispatch.go`; `gofmt -l internal/cli` | no output (formatted) |
| `go test ./internal/cli -count=1` | `ok fake-jev/internal/cli 0.279s` |
| `go test -race ./... -count=1` | ok for all 9 packages |
| `go test ./... -count=1` | ok for all 9 packages |
| `go test ./internal/cli -run 'Validate\|Usage' -v -count=1` | 8 test functions, 26 subtests; all PASS, no SKIP in this environment (`euid != 0`) |
| `go vet ./...` | exit 0, no output |
| `go run ./cmd/guard arch` | exit 0, `findings: []`, 50 files / 9 packages |
| `go run ./cmd/guard lint` | exit 0, `findings: []`, 50 files / 9 packages |
| `go run ./cmd/guard trace` | exit 0, `findings: []`, 0 active / 17 covered / 17 test files |
| `go run ./cmd/guard fuzz` | exit 0, `findings: []`, 0 targets |
| `go build ./cmd/fake-jev` | exit 0 |
| `./scripts/candidate-fingerprint candidate` | `bad76f7917d526d0978fb613aee212f2345ff06a717259788d4a04366f01ab19` |

`./scripts/verify-candidate FJ-030` was not run by this stage: it writes a new
report under `.agent/reports/FJ-030/` and this stage is not permitted to write
reports.

## Residual risks

- **§15.2 remains only partly satisfied** through no change of this stage: G1
  and G2 are still open and owned by FJ-059. `validate` reports exactly what
  `config.LoadFile` checks.
- **The no-network evidence is structural plus one behavioral port test**, not
  a sandboxed run with egress blocked. The closure test proves no file on the
  validate path can reach `net`; the port test proves the configured port stays
  unbound. That is the strongest hermetic evidence available in this
  environment.
- **Unbounded input read** (`/dev/zero`, FIFOs) as recorded above; no
  specification-defined limit exists, so no fix was invented.
- **`TestValidateRejectsUnreadableFile` skips as root.** In an environment
  where the suite runs as root, the unreadable-path case is asserted only
  manually (matrix above), not by the test.
- **The success-line text and the usage wording are presentation**, not
  contract; tests assert the exact strings, so a future wording change is a test
  change, not a behavior change.
