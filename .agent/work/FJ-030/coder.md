---
stage: coder
task: FJ-030
inputFingerprint: 4acd6dd6c828af008d216a52f543aa571806355cf45bac0e71937bf5588d49ab
outputFingerprint: 5fd7ac44377d157f72a1433afb3f7cffd14a622f092c3aef91e72c9d7ccb0b48
taskFingerprint: d37327aa23e2afc94bc9b8bbc7d4c9cd3a87d61e03dc2cc2605c258b8b6c34a7
gitHead: abba008
generatedAt: 2026-09-30T14:52:21Z
author: worker/FJ-030-coder
---

# Coder — FJ-030

## Final candidate fingerprint

```text
5fd7ac44377d157f72a1433afb3f7cffd14a622f092c3aef91e72c9d7ccb0b48
```

(`./scripts/candidate-fingerprint candidate`, run after the last edit; the
verify report under `.agent/reports/` and every `.agent/work/` file are
excluded from the digest, so the value is stable against both the specifier
artifact and the verification report.)

## What changed

Three files, all inside `allowedFiles`:

- `internal/cli/validate.go` (new) — `runValidate` plus the `validatePath`
  argument check.
- `internal/cli/dispatch.go` — one `case "validate"` arm and the usage text.
- `internal/cli/validate_test.go` (new) — the tests below.

`internal/config` is untouched: the loader is the single validation surface.

### Criterion by criterion

**`fake-jev validate <path>` loads YAML or JSON and runs the full statically
checkable configuration validation (no duplication).**
`runValidate` calls `config.LoadFile(path)` and nothing else. `LoadFile` →
`Load` is the same entry point the data plane's configuration path and the
control API's dynamic-stub path (`internal/control/handlers.go`) already use,
so YAML/JSON convergence (§12), defaults (§12.2), `DisallowUnknownFields`
(§12.3, §38.8), `validateDocument` (§38.1–§38.5), `validateValues`
(§12.2/§38.1–§38.4, duplicate static stub ids §11.2/§12.4), `validateWhen`
(§12.5/§38.6), `validateThen`/`validateResponse` (§12.6/§38.7),
`validateRaw`, `validateUsage`, and `validateExpect` (§12.7) all run exactly
once, in one place. No rule is restated in the CLI package.

**No server, no listener, no network request.** The command path contains no
server construction, no `net` import, and no I/O other than `os.ReadFile`
inside `config.LoadFile`. `internal/cli` imports only `fmt`, `io`,
`fake-jev/internal/config`, `runtime`, `runtime/debug`, `strconv`, `strings`;
`internal/config` imports only `bytes`, `encoding/json`, `errors`, `fmt`,
`io`, `os`, `regexp`, `time`, `gopkg.in/yaml.v3`. Asserted by
`TestValidatePathLinksNoNetworkingPackage` (structural) and
`TestValidateOpensNoListener` (behavioral).

**Exit 0 with a human-readable success line on stdout.** Success writes
`fake-jev: <path>: configuration is valid\n` to stdout and returns
`exitOK`. A failed stdout write is an internal operational failure → 2
(§42.1), mirroring `runVersion`.

**Exit 2 on usage failure (missing path, extra argument, unknown flag) and
on any configuration/schema violation.** Usage failures route through the
existing `usageError` helper, so they reuse the `fake-jev: ...` diagnostic
plus the usage text on stderr and return `exitFailure`. Loader failures are
reported as `fake-jev: <path>: <error>` on stderr and return
`exitFailure`. Nothing diagnostic is written to stdout: the CLI never touches
stdout on a failure path, and each test case asserts `stdout == ""`.

**Dispatch and usage text.** `case "validate": return runValidate(rest,
stdout, stderr)` is inserted next to `version`; the usage block now lists
`validate` as a real command and the trailing note reads "Further
subcommands (serve, run, verify) land in later work items."

**Conventions.** The package comment already states that subcommands receive
their streams explicitly; `runValidate(args, stdout, stderr)` follows
`runVersion`'s shape, reuses `exitOK`/`exitFailure` from `dispatch.go`,
reuses `usageError` and `writef`, adds no module (stdlib + the already
approved `gopkg.in/yaml.v3` through `internal/config`), and does not read
stdin (the `Run` signature exposes no stdin at all).

### Stream and exit table (as implemented)

| Input | stdout | stderr | exit |
|---|---|---|---|
| valid YAML or JSON file | `fake-jev: <path>: configuration is valid` | empty | 0 |
| no path / two paths / any flag | empty | diagnostic + usage | 2 |
| unreadable, empty, malformed, multi-document file | empty | `fake-jev: <path>: <error>` | 2 |
| schema or fixture violation | empty | same | 2 |

## G1 / G2 / G3 assessment

**G1 — unvalidated `jev/v1` helper payloads: recorded gap, not implemented.**
Confirmed by running the new command against deliberately illegal fixtures:
`answers: {urgent: {noul: 5}}` (outside `[0,1]`), a helper object carrying an
unknown key, `route: {choice: backend, legend: {"0": nope}}` (the `legend` key
is prohibited in v1 score fixtures by §13.3), and
`{"urgent":{"score":"high"}}` all exit 0 today. The rules are statically
decidable, so §15.2's "all enabled profile-specific fixture data" mandate is
not fully met. It is nonetheless **not implementable inside `allowedFiles`**:
the only existing helper checks live in `internal/compat/jev/v1/fixture.go`
(`generateAnswer` and `validateHelperFields`) and are unexported, request-typed
(`Question`/`Request`), and reached only through response generation. Composing
them from the CLI would require either exporting a new static entry point in the
compat package (outside `allowedFiles`) or restating §13.1–§13.3 rules in
`internal/cli` — a forked contract that would leave the control API's
dynamic-stub path and the data plane on the lax behaviour. **Recommendation:** a
follow-up work item that adds a static fixture-validation entry point to
`internal/config` (or an exported `v1.ValidateStaticAnswers(json.RawMessage)`
called from `config.Load`), then asserts the payloads above exit 2 through both
`validate` and the control API. Because that item changes strictness observed
through `config.Load` — a shared surface — it needs its own acceptance
criteria, not a silent change here.

**G2 — `when.questions` vs `then.answers` cross-check: recorded gap, not
implemented.** Confirmed: a stub whose `when.questions` is `{urgent: noul}`
while `then.answers` is `{something_else: {choice: nope}}` exits 0, although
§13.4/§13.5 require an `answers` map to cover exactly the matched request's
question set and the question matcher is exact (§12.5/§40.5). The comparison is
statically decidable whenever `when.questions` is present (skip when absent),
and the helper key would additionally have to match the declared question type.
Same blocker as G1: it needs new logic in `internal/config` and it changes
strictness on the shared `config.Load` path used by
`POST /__fake/v1/stubs`. **Recommendation:** fold it into the same follow-up as
G1, with an explicit decision on whether a mismatch is a configuration error
(§15.2 "all statically possible consistency checks") or stays a runtime
`fake_jev_invalid_stub_response` failure (§13.5 names the latter for coverage
failures). Do not implement before that decision: it is an interpretation of
the specification, not something this stage may settle.

**G3 — `then.model`/`then.usage` as siblings of `then.sequence`: no gap, no
change.** §38.7 allows optional `model`/`usage` siblings on an `answers`
response and defines a sequence element as "the same response shape except it
MUST NOT contain another `sequence`". The current validator matches that
reading: a sequence element with `answers` + `model` + `usage` is accepted
(verified by running the built binary), and a nested `sequence` is rejected with
exit 2 via `DisallowUnknownFields` (`json: unknown field "sequence"`). The
specifier's own artifact already records G3 as a prose ambiguity rather than a
gap; nothing to implement, and inventing a rejection would contradict §38.7.

No other §15.2 static check was added because the current validator cannot
honestly support one from `internal/cli` without duplicating
`internal/config` logic. This is recorded as gaps G1/G2 above; the orchestrator
may want them mirrored into `state.json` `blockers` (`type: "spec-gap"`), which
this stage is not allowed to edit.

## Tests added (`internal/cli/validate_test.go`)

All tests drive `Run(...)` (not `runValidate` directly), so the dispatch
wiring, the usage text, and the stream split are covered, not just the helper.
Every file lives in `t.TempDir()`; no test reads the environment, the clock,
stdin, or the network beyond an ephemeral loopback listener.

- `TestValidateAcceptsValidConfigurationFiles` — valid YAML, valid JSON, and a
  defaults-only document: exit 0, exact success line on stdout, empty stderr.
- `TestValidateRejectsUsageFailuresAndInvalidConfiguration` — 19 table cases:
  missing path, extra argument (with a *valid* first file, so a parser that
  ignored extras would be caught), unknown flag, unknown flag after the path,
  missing file, empty file, malformed YAML, malformed JSON, multiple documents,
  missing `schemaVersion`, unknown top-level key, unknown nested key, non-strict
  `mode`, unknown profile, out-of-range port, wrong type for `stubs`, duplicate
  stub id, invalid `expect`, invalid response form (two `then` forms), missing
  response form. Each asserts exit 2, `stdout == ""`, a non-empty stderr
  diagnostic, and — for the usage cases only — the usage text on stderr (so a
  usage failure and a configuration failure cannot be confused).
- `TestValidateStreamsAreDisjoint` — explicit stdout/stderr split for one valid
  and one invalid file.
- `TestUsageListsValidateAsACommand` — `help` lists `validate` in the command
  block and no longer lists it among the future subcommands.
- `TestValidateOpensNoListener` — holds an ephemeral loopback port open, points
  the configuration's `server.port` at it, and asserts a valid file still exits
  0 (a command that bound the port would fail) and an invalid file still exits 2
  with a configuration diagnostic rather than a bind error. Hermetic: loopback
  only, port 0, no egress.
- `TestValidatePathLinksNoNetworkingPackage` — parses the non-test sources of
  `internal/cli` and `internal/config` with `go/parser` and fails if any imports
  `net` or a `net/...` package, which is the structural reason the command
  cannot dial or listen. Fails closed if a scanned directory has no Go files.

## Commands run and outcomes

| Command | Result |
|---|---|
| `gofmt -w internal/cli/validate.go internal/cli/validate_test.go internal/cli/dispatch.go`; `gofmt -l internal/cli` | clean (no output) |
| `go test ./internal/cli -count=1` | ok |
| `go test ./... -count=1` | ok (all 8 packages) |
| `go test -race ./... -count=1` | ok (all packages) |
| `go vet ./...` | ok |
| `go run ./cmd/guard arch` | exit 0, `findings: []`, 50 files / 9 packages |
| `go run ./cmd/guard lint` | exit 0, `findings: []` |
| `go run ./cmd/guard trace` | exit 0, `findings: []`, 17 test files covered |
| `go run ./cmd/guard fuzz` | exit 0, `findings: []`, 0 targets |
| `go build ./cmd/fake-jev` | ok |
| `./scripts/verify-candidate FJ-030` | exit 0, 20 passed / 0 failed / 0 skipped; report `.agent/reports/FJ-030/report-20260930T145216Z.json` |
| `./scripts/candidate-fingerprint candidate` | `5fd7ac44377d157f72a1433afb3f7cffd14a622f092c3aef91e72c9d7ccb0b48` |

End-to-end smoke run of the built binary confirmed: valid YAML and JSON → exit
0 with the success line on stdout and empty stderr; unknown key → exit 2 with
`fake-jev: bad.yaml: decode configuration: json: unknown field "extra"` on
stderr and empty stdout; `validate` with no path, with two paths, and with
`--strict` → exit 2 with the diagnostic plus usage on stderr.

## Residual risks

- **§15.2 is only partly satisfied.** G1 and G2 remain open by design (see
  above); the command honestly reports only what the shared validator checks.
  A reader who takes "all enabled profile-specific fixture data" literally will
  see the gap, which is why it is recorded here rather than papered over.
- **The no-network proof is structural plus one behavioral port test**, not a
  sandboxed run with egress blocked. It rules out any code path in the two
  packages on the validate path reaching `net`, which is the strongest evidence
  available without a network-isolated harness.
- **`-` as a path argument** is treated as a filename, not as stdin; §15.2
  defines no stdin mode and §15 forbids prompts, so the read fails and exits 2.
  If a future item wants stdin support, that is a spec change, not this one.
- **No new flags** are accepted for `validate`; any flag is a usage failure.
  `--help` after the command name is therefore exit 2 rather than help; the
  specifier's V1 requires exactly that behavior, so it is recorded here as an
  intentional consequence rather than a defect.
- The success line is written with a checked `fmt.Fprintf`; a broken stdout
  pipe exits 2. This matches `runVersion` and §42.1 but means a closed stdout
  turns a valid file into a failure.
