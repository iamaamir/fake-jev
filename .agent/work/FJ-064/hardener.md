---
stage: hardener
task: FJ-064
inputFingerprint: 74245107247629f9c366043ed7aada0dfbe29f4b63157bd6218afc3ee84a83d0
outputFingerprint: f6dfb54c6debeda144bf07f521f013c381ef84fa812e6bc63756863113ec56d1
taskFingerprint: 71aa852bd544dfae210d4e52273e9036c9347ba7f7d92dacc4586a9c9da6e42a
gitHead: ab45ac5
generatedAt: 2026-10-03T14:20:07Z
author: worker/FJ-064-hardener
---

# FJ-064 hardener report

## outputFingerprint

```
f6dfb54c6debeda144bf07f521f013c381ef84fa812e6bc63756863113ec56d1
```

This is the exact output of `./scripts/candidate-fingerprint candidate` after the
finalization pass. The finalization pass changed no file content beyond what was
already in the working tree, so the fingerprint covers the candidate that was
verified.

## Scope of this pass

Finalization only. The fix was already present in the working tree when this
stage started; this pass verified it, reproduced the pre-fix failures against
`ab45ac5`, checked the guard matrix against spec 12.3 and 38.8, confirmed the
fuzz target and out-of-package files were untouched, ran the command set once
each, and recorded the result. No source file was edited.

## The two defects and how they are fixed

### Defect F1 — `Load` fail-open on a case-variant duplicate key (spec 12.3, 38.8)

`encoding/json` resolves an object key to a struct field case-insensitively (and,
for multi-rune simple-fold sets, also by `unicode.SimpleFold`). A raw-document
check keyed on the exact spelling therefore never sees `sChemAVErsion`: the
decoder writes `Config.SchemaVersion` from the second spelling while
`validateDocument` reads `root["schemaVersion"]` from the first, so `Load`
returned a non-nil `*Config` with a nil error that its own `Validate` then
rejected. 12.3 requires unknown top-level keys to fail validation, and 38.8 says
project-owned schema objects use fail-closed unknown-field validation.

The working tree abandons document-wide case-folding entirely and enforces
**exact key spelling** at each project-owned object via
`rejectUnexpectedFields(value, name, fields...)` (`internal/config/validate.go`),
applied on top of the pre-existing `decoder.DisallowUnknownFields()`. A key that
does not exactly equal a known field name is an unknown key and fails with
`"<name>.<key> is not a known field"`. Because the guard runs on the raw document
at every project-owned level, there is no folding, no fold helper, and therefore
no non-terminating fold walk.

### Defect F2 — non-deterministic diagnostics (spec line 143)

`normalizeYAMLValue` ranged over `map[string]any` / `map[any]any` and
`validateWhenDocument`/`validateWhen` ranged over question maps. Go randomises
map iteration order, so the first failing entry — and thus the message — varied
between identical calls. `sortedKeys` and `orderYAMLKeys` (`internal/config/load.go`)
now drive every map walk whose first failure is reported, giving a diagnostic that
depends only on the document. Spec line 143 makes determinism a project principle.

## Pre-fix per-test evidence (revision `ab45ac5`)

Method: `git archive ab45ac5 | tar -x -C /tmp/fj064prefix`, then the two new test
files copied in, then one `go test -run '^<Test>$' -count=1` per test. Single
module build, no full-suite rebuild.

| # | test | pre-fix result | evidence |
|---|------|----------------|----------|
| 1 | `TestLoadNeverReturnsConfigThatValidateRejects` | FAIL | subtests `case variant with a different value`, `upper case variant`, `yaml block case variant`: `Load returned &Config{SchemaVersion:0,...} with a nil error, but Validate rejects it: schemaVersion must equal 1` |
| 2 | `TestLoadRejectsCaseVariantDuplicateKeys` | FAIL | 13 subtests; e.g. `lower case variant` and `yaml block case variant` loaded successfully; `server` loaded `Host:"0.0.0.0", Port:80`; `limits` loaded `MaxInteractions:2`; `models` loaded `Name:"b"`; `nested when field` loaded both `a` and `b`; `nested raw response field` loaded a non-nil `Raw` |
| 3 | `TestLoadRejectsKeysThatAreNotExactlySpelled` | FAIL | subtests `schemaVersion`, `server`, `compatibility`, `limits`, `models`, `stubs`; e.g. `Load({"schemaVersion":1,"ſchemaVersion":1}) = &Config{...} with a nil error, want an error: "ſchemaVersion" differs from "schemaVersion"`; also `schemAVersion`, `schemaVerſion`, `ſerver`, `compAtibility`, `limitſ`, `modelſ`, `ſtubs`, `stubſ` |
| 4 | `TestLoadTerminatesOnUncasedKeys` | PASS pre-fix | `ok fake-jev/internal/config 0.160s`. Regression guard, not a pre-fix failure: at `ab45ac5` there is no fold helper, so an uncased key simply hits `DisallowUnknownFields` and returns an ordinary unknown-field error. The non-termination lived only in the coder's intermediate fold implementation, never committed, and is absent from both the base and the candidate. |
| 5 | `TestLoadAcceptsCaseVariantKeysInOpenContainers` | PASS pre-fix | `ok fake-jev/internal/config 0.140s`. Non-regression guard: the open containers were already open. It confirms the fix does not close them. |
| 6 | `TestLoadDiagnosticsAreDeterministic` | FAIL | 4 subtests; e.g. `yaml non string keys`: `call 7 returned {... object key 2 is not a string}, call 1 returned {... object key 7 is not a string}`; `yaml invalid question types`: call 2 reported question `"b"`, call 1 reported question `"a"`; `json null questions`: call 18 reported `questions.b`, call 1 reported `questions.a` |

Summary: 4 of 6 tests fail against `ab45ac5` (1, 2, 3, 6). Tests 4 and 5 are
regression/non-regression guards that pass both before and after; they are not
evidence of a pre-fix failure and are reported as such rather than as failures.

## Struct-level versus open-container matrix

Spec 12.3 names the configuration root plus `server`, `limits`, the stub envelope,
`when`, `then`, and `expect`. Every one of those is guarded, and so is every
project-owned sub-object the schema recurses into. The intentionally open
containers of 38.8 are not guarded.

| level (raw-document name) | project-owned? | `rejectUnexpectedFields` | known field set |
|---|---|---|---|
| configuration root | yes | present | `schemaVersion, server, mode, compatibility, limits, models, stubs` |
| `server` | yes | present | `host, port` |
| `limits` | yes | present | `dataPlaneBodyBytes, controlPlaneBodyBytes, maxInteractions, logBodyBytes, gracefulShutdownSeconds` |
| `models[]` | yes | present | `name, description, release_date` |
| `stubs[]` (stub envelope) | yes | present | `id, profile, priority, when, then, expect` |
| `stubs[].when` | yes | present | `operation, model, state, questions` |
| `stubs[].then` | yes | present | `answers, sequence, raw, model, usage` |
| `stubs[].then.raw` | yes | present | `status, headers, body` |
| `stubs[].then.usage` | yes | present | `input_tokens, output_tokens` |
| `stubs[].then.sequence[]` | yes | present | `answers, raw, model, usage` |
| `stubs[].then.sequence[].raw` | yes | present | `status, headers, body` |
| `stubs[].then.sequence[].usage` | yes | present | `input_tokens, output_tokens` |
| `stubs[].expect` | yes | present | `exactly, atLeast, atMost` |
| `then.raw.body` | open (38.8) | absent | — |
| `when.state` | open (38.8) | absent | — |
| `then.answers` (and `sequence[].answers`) | open (38.8) | absent | — |
| `when.questions` names | open (38.8) | absent | — |
| `then.raw.headers` names | open (38.8) | absent | — |

No project-owned level named by 12.3 lacks the guard, and no open container was
given it. The guard is applied at every level the decoder struct-decodes from a
project-owned object, and stops exactly where 38.8 says content is arbitrary. No
level was missing, so no level was added in this pass.

## No-regression and no-scope-creep evidence

- `git status --porcelain` shows only `internal/config/load.go`,
  `internal/config/validate.go`, `internal/config/collision_test.go`,
  `internal/config/determinism_test.go` (plus `.agent/` state/report/stage files
  that this stage does not own). No file outside `internal/config/` changed.
- No `fuzz_test.go` exists in `internal/config/`, and no `func Fuzz` symbol exists
  in `internal/config/`. The only fuzz targets in the repository are
  `cmd/guard/fuzz_test.go` and `internal/compat/jev/v1/fixture_test.go`; FJ-040's
  config target is not present and was not added, as required.
- The pre-existing exact-duplicate-key rejection (`rejectDuplicateJSON` /
  `scanJSONValue`) is unchanged. `TestLoadAcceptsCaseVariantKeysInOpenContainers`
  includes an `exact duplicate keys stay rejected` case pinning that an exact
  repeat inside `then.raw.body` is still rejected.
- Open containers remain open: bodies, state, answers, question names, and header
  names with keys differing only in case still load with both payloads preserved.

## Command outcomes

| command | result |
|---|---|
| `gofmt -l` on the four changed files | empty output (exit 0) |
| `go test ./internal/config -count=1` | `ok fake-jev/internal/config 0.267s` |
| `go test ./... -count=1` | ok, 10 packages (8 test packages + 2 with no tests) |
| `go test -race ./... -count=1` | ok, 10 packages |
| `go vet ./...` | exit 0, no output |
| `go run ./cmd/guard arch` | `findings: []`, files 62, packages 10 |
| `go run ./cmd/guard lint` | `findings: []`, files 62, packages 10 |
| `go run ./cmd/guard trace` | `findings: []`, active 0, covered 17, test_files 24 |
| `go run ./cmd/guard fuzz` | `findings: []`, targets 1 |
| `go build ./cmd/fake-jev` | exit 0 |
| `./scripts/candidate-fingerprint candidate` | `f6dfb54c6debeda144bf07f521f013c381ef84fa812e6bc63756863113ec56d1` |

## PM-verified behaviour reproduced in the candidate

- `{"schemaVersion":1,"中":1}` -> error `json: unknown field "中"`; no hang.
- `then.raw.body` holding both `"a"` and `"A"` -> loads, both preserved.
- A sequence element's raw body holding both `"x"` and `"X"` -> loads.
- `when.state` holding both `"Key"` and `"KEY"` -> loads.
- `{"schemaVersion":1,"sChemAVErsion":0,...}` -> error
  `configuration.sChemAVErsion is not a known field`.
- `server: {Host: ...}` -> `server.Host is not a known field`;
  `when: {Operation: ...}` -> `stubs[0].when.Operation is not a known field`.
- `Load('{07,2}')` called 200 times -> 1 distinct diagnostic.
- `go test ./internal/config -count=1` and `go test ./... -count=1` pass.

## Residual risks

- The mechanism was replaced, not extended. The coder's document-wide
  case-variant folding approach was abandoned by the previous attempt and is not
  present in the candidate; the exact-spelling guard on raw documents replaced it.
  The consequence is a deliberate behaviour choice: a key that differs from a
  known field only in case (or by a simple-fold equivalent rune) is rejected as an
  unknown key rather than folded or matched. That is what 12.3 requires (unknown
  keys MUST fail), but any consumer that relied on `encoding/json`'s
  case-insensitive matching of a known field would see a new error. No such
  consumer is known, and no accepted unambiguous input changes.
- `TestLoadTerminatesOnUncasedKeys` does not reproduce a failure at `ab45ac5`; the
  non-termination it describes was in an uncommitted intermediate revision. It is
  kept as a regression guard for the exact-spelling design.
- The unbounded configuration-file read recorded after FJ-030, and yaml.v3
  anchor/alias parse-cost amplification, remain open and were deliberately not
  folded in.
- The `rejectUnexpectedFields` walk stops at `then.raw.body`, `state`, `answers`,
  `when.questions` names, and `raw.headers` names; if a future schema change makes
  any of those project-owned, the corresponding call site must gain a guard.

## Verifier

`./scripts/verify-candidate FJ-064` report:
`.agent/reports/FJ-064/report-20261003T142117Z.json` (revision `ab45ac5`, tree
staged 0 / unstaged 3 / untracked 7, `candidateFingerprint`
`f6dfb54c6debeda144bf07f521f013c381ef84fa812e6bc63756863113ec56d1`,
`taskFingerprint` `71aa852bd544dfae210d4e52273e9036c9347ba7f7d92dacc4586a9c9da6e42a`).

Summary: **20 passed, 0 failed, 0 skipped, 0 not applicable.** All 20 rows are
required rows under gates G-C, G-L, G-Q; none is exempted.

| row | gate | required | result |
|---|---|---|---|
| `check_candidate_fingerprint` | G-C | yes | pass |
| `check_required_files` | G-C | yes | pass |
| `check_scripts_executable` | G-C | yes | pass |
| `check_shell_syntax` | G-C | yes | pass |
| `check_toolchain_python3` | G-C | yes | pass |
| `check_spec_present` | G-C | yes | pass |
| `check_json_files` | G-C | yes | pass |
| `check_work_item_schema` | G-C | yes | pass |
| `check_recorded_report_coverage` | G-C | yes | pass |
| `check_role_packs` | G-C | yes | pass |
| `policy.exemptions.unknown` | G-C | yes | pass (3 exemptions checked) |
| `policy.exemptions.stale` | G-C | yes | pass |
| `policy.classification` | G-C | yes | pass (class=product) |
| `policy.required_stages` | G-C | yes | pass (stages=specifier,coder,cleaner,hardener,qa) |
| `check_go_test` | G-C | yes | pass |
| `check_go_vet` | G-C | yes | pass |
| `check_go_build` | G-C | yes | pass |
| `check_guard_arch` | G-L | yes | pass |
| `check_guard_lint` | G-L | yes | pass |
| `check_guard_trace` | G-Q | yes | pass |

No row is skipped or not-applicable, so there is nothing to explain as an
omission. `guard fuzz` is not a `verify-candidate` row; it was run separately and
reported `findings: []`, targets 1. Every row is bound to revision `ab45ac5` and
the candidate fingerprint above. This report records the row outcomes; it makes
no verdict about the work item.
