---
stage: cleaner
task: FJ-030
inputFingerprint: 5fd7ac44377d157f72a1433afb3f7cffd14a622f092c3aef91e72c9d7ccb0b48
outputFingerprint: 5fd7ac44377d157f72a1433afb3f7cffd14a622f092c3aef91e72c9d7ccb0b48
taskFingerprint: d37327aa23e2afc94bc9b8bbc7d4c9cd3a87d61e03dc2cc2605c258b8b6c34a7
gitHead: abba008
generatedAt: 2026-09-30T15:05:35Z
author: worker/FJ-030-cleaner
---

# Cleaner — FJ-030: `fake-jev validate <path>`

## Changes applied

**None.** No file outside `.agent/work/FJ-030/cleaner.md` was created, edited,
or deleted by this stage, and no staged file exists. The candidate fingerprint
was re-read after the review and is still
`5fd7ac44377d157f72a1433afb3f7cffd14a622f092c3aef91e72c9d7ccb0b48`.

Reason: the parent pinned the candidate fingerprint for this stage, and every
cleanup considered below lands in an in-fingerprint file (`internal/**`). Any
edit — including a comment-only one — moves the candidate, invalidates the
pinned `inputFingerprint`, and stales the coder artifact plus the verify report
bound to this revision (`.agent/reports/FJ-030/report-20260930T145216Z.json`).
Per the cleaner stage pack, a structural move is only made when it is justified
*and* re-verified against a new candidate; with the fingerprint frozen the
review is reported instead of applied. The one finding that would otherwise be
applied is F1 and it is a single comment line.

## Files inspected

- `internal/cli/validate.go` (new, 50 lines)
- `internal/cli/validate_test.go` (new, 262 lines)
- `internal/cli/dispatch.go` (diff: one `case "validate"` arm, usage text)
- `internal/cli/version.go` (unchanged; read as the sibling-subcommand pattern
  the new code follows)
- `internal/config/load.go`, `internal/config/validate.go` (read-only; the
  shared validation surface the command reuses)
- `spec/fake-jev-technical-spec-v2.md` §15, §15.2, §15.3, §15.4, §15.6, §19.2,
  §42.1, §42.5; `docs/development/acceptance-catalog.md` C-CLI-001, C-CLI-005,
  C-CLI-010.

## Findings

### F1 — stale test name in a doc comment (comment accuracy)

`internal/cli/validate_test.go:130` opens with
`// TestValidateSuccessGoesToStdoutAndFailuresToStderr pins the stream split`
while the function it documents is `TestValidateStreamsAreDisjoint`
(`validate_test.go:133`). It is a rename artifact: `grep -rn` over the whole
repository finds the old name only on that comment line and nowhere else, and
every other doc comment in the file names its own function. Coverage and
behavior are unaffected; the mismatch is documentation drift inside the new
file. Left in place only because of the frozen fingerprint (see above) — the
repair is a one-line comment edit for the first stage allowed to move the
candidate.

### F2 — the no-network guard duplicates, and over-states, the central arch rule

`TestValidatePathLinksNoNetworkingPackage` (`validate_test.go:225`–`262`)
hand-rolls an import scan over every non-test `.go` file in `internal/cli` and
`internal/config` and fails on `net` or `net/...`. The repository already
centralizes import policy in `.agent/guards.json` (`arch.forbidden`, enforced by
`cmd/guard arch`, which `./scripts/verify-candidate` runs — it passed with
`findings: []` here).

Two consequences worth recording:

- The assertion is strictly wider than the specification. §15.2 requires that
  *validate* makes no network request and §19.2 that a test can run with egress
  disabled. The test instead forbids the two packages from ever importing `net`,
  so the §15.3 `verify` client or the §15.4 in-process loopback server, if they
  land inside `internal/cli`, will fail this test even though the validate path
  stays network-free. Related constraints that matter repository-wide belong in
  the guard registry that documents them.
- It reads imports of the *package*, not of the reachable validate path, so it
  is a proxy for the property rather than the property itself.

Not applied (fingerprint frozen; the assertion is currently true, so nothing is
being papered over). If a later stage trims it, the property that still holds is
the one in F3's second paragraph plus the validate-path import closure.

### F3 — one test depends on loopback rather than on in-memory state

`TestValidateOpensNoListener` (`validate_test.go:183`) opens
`net.Listen("tcp", "127.0.0.1:0")`, holds it for the run, closes it via a
`defer` with a checked error, and asserts that `validate` still exits 0 for a
valid file and 2 for an invalid one. It is loopback-only, port 0, and matches
the existing repository pattern (`test/integration/dataplane_test.go:168`); it
uses no external network, no sleeps, and no randomness. Recorded rather than
changed for two reasons: AGENTS.md's "no network" test rule is met here only on
a loopback reading, and the guard it provides is partial — it detects a validate
that binds the *configured* port, not one that bound some other port, since an
ephemeral bind would still let this test pass. The stronger evidence for the
same claim is structural (F2's scan) and the fact that the whole validate path
does a single `os.ReadFile` and nothing else.

### F4 — usage text does not show the required positional argument

`dispatch.go:65` reads
`validate   validate a configuration file without starting a server`. It is
accurate about the command's scope and consistent with the dispatcher in every
checkable way: the listed commands (`version`, `validate`, `help`) are exactly
the arms of `Run`; `-h`/`--help` are handled by the `help` arm and are covered
by the `[flags]` placeholder on the usage line; and the trailing
"Further subcommands (serve, run, verify) land in later work items." note now
matches §15's remaining required commands. It does not mention `<path>`, so a
user who runs a bare `fake-jev validate` (verified on the built binary) receives
diagnostic text plus usage that never names the missing argument, while the
`version` entry legitimately needs none. Optional polish; not applied, since the
success/usage wording is not a contract (§15.2, §42.5) and the edit moves the
fingerprint.

### F5 — control flow and naming: nothing to change

- `validatePath` (`validate.go:34`–`46`) rejects flag-shaped arguments before
  the "already have a path" arm, which is exactly what makes
  `validate <path> --strict` a usage failure rather than a silent extra
  argument. All four argument classes (none, one, two, flag) were confirmed
  against the built binary; the redundant `path := ""` initializer is harmless.
- `runValidate` (`validate.go:15`–`28`) is a straight four-step sequence —
  parse, load, report, return — with no dead branch and no hidden exit path. It
  reuses `usageError`, `exitOK`, and `exitFailure` and adds no new constant.
- The success line goes through a checked `fmt.Fprintf`, exactly as
  `runVersion` (`version.go:35`–`41`) does, rather than through `writef`: the
  write failure must change the exit code, and both result-writing sites follow
  the same pattern. Diagnostics use `writef`, matching the rest of `dispatch.go`.
- Naming is one spelling per concept and mirrors the existing sibling:
  `runValidate`/`validatePath`/`validateSuccessLine` beside
  `runVersion`/`versionLines`; `validYAML`/`validJSON` are test-only.
- No configuration rule is restated in `internal/cli`: the command calls
  `config.LoadFile` and nothing else, and `internal/config` is untouched by the
  candidate (`git status` shows only `dispatch.go` modified plus the two new
  CLI files).
- Pre-existing and outside the new-code diff, therefore not touched: the
  `if err != nil { return }` branch in `writef` (`dispatch.go:55`–`58`) is a
  no-op; it predates this candidate (present in `HEAD`).

### F6 — stream and exit discipline: nothing to change

Verified against the built binary (matrix below, see commands): stdout carries
only the success line; every failure path writes only to stderr and leaves
stdout empty; usage failures add the usage text on stderr; the only exits are 0
and 2, both taken from the §42.1 constants (`exitVerify` is unreachable from
`validate`). The table test asserts `stdout == ""` on every rejection case, so a
future diagnostic written to the wrong stream fails a test instead of passing
silently.

### F7 — test quality: nothing to change

- Table-driven where it matters: 19 rejection cases in one table with a per-case
  `usage bool`, so a usage failure and a configuration failure cannot be
  confused.
- Hermetic: every case writes into `t.TempDir()`; no clock, environment, stdin,
  randomness, sleeps, or subprocesses; no assertion depends on map iteration
  order.
- `validYAML`/`validJSON` encode the same logical document in both accepted
  encodings, which is the §12 convergence check (C-CFG-009) rather than two
  unrelated fixtures.
- Tests drive `Run` rather than `runValidate`, so dispatch wiring, usage text,
  and stream split are covered.
- The stream-split assertions in `TestValidateStreamsAreDisjoint`
  (`validate_test.go:133`) restate checks the table and the success test already
  imply. That is deliberate explicitness stated in its doc comment, not logic
  duplication; the duplicated text is three lines of test assertion.

## Commands run and outcomes

| Command | Outcome |
|---|---|
| `gofmt -l internal/cli/validate.go internal/cli/validate_test.go internal/cli/dispatch.go` | no output (formatted) |
| `go test ./internal/cli -count=1` | `ok fake-jev/internal/cli 0.601s` |
| `go test -race ./internal/cli -count=1` | `ok fake-jev/internal/cli 4.822s` |
| `go test ./... -count=1` | ok for all 9 packages (`cmd/guard`, `internal/cli`, `internal/compat/jev/v1`, `internal/config`, `internal/control`, `internal/engine`, `internal/host/http`, `test/integration`; `cmd/fake-jev` has no test files) |
| `go vet ./...` | exit 0, no output |
| `go run ./cmd/guard arch` | exit 0, `findings: []`, 50 files / 9 packages |
| `go run ./cmd/guard lint` | exit 0, `findings: []`, 50 files / 9 packages |
| `go run ./cmd/guard trace` | exit 0, `findings: []`, 0 active / 17 covered / 17 test files |
| `go build -o /tmp/fj-validate ./cmd/fake-jev` + manual matrix | see below |
| `./scripts/candidate-fingerprint candidate` | `5fd7ac44377d157f72a1433afb3f7cffd14a622f092c3aef91e72c9d7ccb0b48` |

Manual matrix on the built binary (stdout / stderr-first-line / exit):

| Invocation | stdout | stderr (first line) | exit |
|---|---|---|---|
| `validate ok.yaml` | `fake-jev: ok.yaml: configuration is valid` | (empty) | 0 |
| `validate bad.yaml` (unknown top-level key) | empty | `fake-jev: bad.yaml: decode configuration: json: unknown field "extra"` | 2 |
| `validate` | empty | `fake-jev: validate requires a configuration path` + usage | 2 |
| `validate ok.yaml ok.yaml` | empty | `fake-jev: validate takes exactly one configuration path` + usage | 2 |
| `validate --strict` | empty | `fake-jev: unknown flag "--strict"` + usage | 2 |
| `validate <dir>` | empty | `fake-jev: <dir>: read configuration "<dir>": read <dir>: is a directory` | 2 |
| `validate -` | empty | `fake-jev: -: read configuration "-": open -: no such file or directory` | 2 |
| `validate ''` | empty | `fake-jev: : read configuration "": open : no such file or directory` | 2 |

`./scripts/verify-candidate FJ-030` was **not** re-run by this stage: it writes a
new report under `.agent/reports/FJ-030/`, and this stage is instructed not to
modify reports. The report already bound to this exact fingerprint
(`report-20260930T145216Z.json`, revision `abba008`,
`check_candidate_fingerprint pass`) records 20 `pass` / 0 `fail` / 0 `skipped`,
including `check_go_test`, `check_go_vet`, `check_go_build`,
`check_guard_arch`, `check_guard_lint`, and `check_guard_trace`. No file in the
workspace changed during this review, so that binding still holds.

## Residual risks and notes for later stages

- F1 (stale doc-comment name at `validate_test.go:130`) is still in the tree and
  needs one comment line changed by the first stage permitted to move the
  candidate.
- F2/F3/F4 are recorded observations, not defects in effect: the structural
  test's scope (whole-package import policy), the one loopback listener in the
  test suite, and the usage text's omission of `<path>`.
- The §15.2 gaps G1 (`jev/v1` helper payloads unvalidated) and G2
  (`when.questions` versus `then.answers` cross-check) that the coder artifact
  records are untouched by this review. `validate` honestly reports only what
  `config.LoadFile` checks; no structural cleanup in `internal/cli` can close
  them, since they require new logic inside `internal/config` or an exported
  static entry point in the compat layer.
- `internal/cli` still imports no `net` package, so F2's assertion is true as of
  this candidate; it is an assertion about the package's future as much as its
  present.
