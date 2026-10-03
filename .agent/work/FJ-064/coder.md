---
stage: coder
task: FJ-064
inputFingerprint: 8c80ab298a252de531dbf9f61a0f67e866fdf5e091cbf8fb80d272d988305c29
outputFingerprint: 74245107247629f9c366043ed7aada0dfbe29f4b63157bd6218afc3ee84a83d0
taskFingerprint: 71aa852bd544dfae210d4e52273e9036c9347ba7f7d92dacc4586a9c9da6e42a
gitHead: ab45ac5
generatedAt: 2026-10-03T12:17:48Z
author: worker/FJ-064-coder
---

# Coder — FJ-064

## outputFingerprint

```text
74245107247629f9c366043ed7aada0dfbe29f4b63157bd6218afc3ee84a83d0
```

`./scripts/candidate-fingerprint candidate` produced this value on the frozen
workspace, twice in a row (byte-identical). `inputFingerprint` is the specifier's
output, so the chain link is intact.

## Mechanism chosen: A (case-insensitive collision detection), as recommended

Both F1 and F2 are fixed with the specifier's recommended Mechanism A, and I did
not find a concrete reason to deviate.

Why A over the post-decode consistency check (B):

- B cannot satisfy C2. A collision whose values agree
  (`{"schemaVersion":1,"sChemAVErsion":1}`) leaves decoded and raw values equal,
  and a case-variant `server`/`limits`/`models`/`stubs` object overwrites a field
  that `validateValues` never compares against the raw document, so B would still
  load both. Re-running `Validate` inside `Load` would only re-check the decoded
  value and is redundant once the ambiguous document is rejected.
- A rejects the real cause (two keys the decoder cannot tell apart) with one
  local check on the JSON that is actually decoded, so no key is special-cased
  and no schema enumeration is needed. That is what the specifier asked for and
  what the F1 mechanism in the PM notes describes.

Scope decision (document-wide scan), and the one observable residual: the scan
rejects a case-variant pair in **any** JSON object of the decoded document, not
only objects the decoder resolves into a struct. This matches existing behaviour,
which already enforces structural uniqueness document-wide: exact duplicates
inside open containers (`when.state`, `then.sequence[].raw`) are rejected today —
`TestControlStubSchemaPreservation/nested_duplicate` and `/sequence_duplicate`
prove it. Consequence: a document that puts two keys differing only in case
inside an opaque payload (`then.raw.body`, `when.state`, `then.raw.headers`) is
now rejected, where it was accepted before. This is the residual the specifier
recorded (spec-silent observations #3) and called acceptable either way; the
document-wide default is what it recommended, and it is the maximally fail-closed
reading of the repository rule "strict behavior must fail closed". The
alternative (mirroring the decoder's struct resolution with a reflective schema
walk) would be more code and more indirection for that one residual, and would
re-introduce the schema enumeration that the specifier's recommendation avoids.
Observed residual example (accepted pre-fix, rejected now):

```text
{"schemaVersion":1,"stubs":[{"id":"x","profile":"jev/v1","when":{},"then":{"raw":{"status":200,"headers":{},"body":{"Ok":1,"ok":2}}}}]}
=> decode configuration: object keys "Ok" and "ok" differ only in case
```

The check runs on the JSON the decoder consumes: the original bytes for JSON
input (key order preserved) and the re-marshalled, key-sorted document for YAML
input. It is placed in `Load` after `toJSON`, **not** inside the JSON fast path,
because `toJSON` deliberately swallows `rejectDuplicateJSON`'s error to admit
YAML flow syntax; a check inside that probe would route ambiguous documents
through the YAML decoder instead of rejecting them. Both formats are covered
(verified below).

## What changed

`internal/config/load.go`

1. `Load` now calls the new `rejectCaseVariantKeys(jsonData)` immediately after
   `toJSON`. Errors read
   `decode configuration: object keys %q and %q differ only in case`.
2. `scanJSONValue` gained a `keyFolding` parameter (`exactKeys` / `foldedKeys`).
   `rejectDuplicateJSON` (the JSON detection probe) keeps `exactKeys` and its
   behaviour is unchanged; `rejectCaseVariantKeys` uses `foldedKeys`. The object
   walk now remembers the first spelling of each key so the diagnostic can name
   both spellings; a byte-identical repeat still reports
   `duplicate object key %q`, so the exact-duplicate message is preserved.
3. `foldKey` reproduces the folding `encoding/json` applies when it looks up an
   object key in a struct type (`encoding/json.appendFoldedName`/`foldRune`:
   ASCII letters upper-cased, other runes reduced to the smallest rune of their
   Unicode simple fold set), so two keys are treated as one exactly when
   `strings.EqualFold` reports them equal — the same relation as
   `bytes.EqualFold`, which `encoding/json` documents for its name matching.
4. F2: map walks in `normalizeYAMLValue` no longer depend on Go map iteration
   order. `map[string]any` is walked through the new package-local
   `sortedKeys[T any](map[string]T) []string` helper (same shape as the existing
   helper in `internal/compat/jev/v1/fixture.go`); `map[any]any` is walked
   through `orderYAMLKeys`, which orders keys by `%T\x00%v`. The key type is part
   of the order because YAML keys are not necessarily strings and keys of
   different types can format identically (for example the string `"2"` and the
   integer `2`), which would otherwise leave the first reported failure
   order-dependent. Only the order of the walk changed; the returned
   `map[string]any` is identical for identical input.

`internal/config/validate.go`

5. F2: the two walks whose reported error embeds a map key are now driven by
   `sortedKeys`: `validateWhenDocument` (`when.questions` -> question name inside
   the message) and `validateWhen` (`when.Questions` -> `question %q has invalid
   type %q`). The `validateRaw` header walk was deliberately left alone: its
   message (`raw header names must be non-empty`) does not name the key, so it
   is already byte-identical for identical input; sorting it would add work with
   no observable effect.

Out of scope and untouched: the unbounded configuration-file read (recorded after
FJ-030) and the `yaml.v3` anchor/alias parse-cost amplification. No new
dependency, no schema change, no change outside `internal/config/`, no
`internal/config/fuzz_test.go` and no `Fuzz*` symbol (guard fuzz still reports
`targets: 1`, the pre-existing `FuzzValidateFixtureAnswers`).

Post-fix diagnostics for the recorded repros:

```text
{"schemaVersion":1,"sChemAVErsion":0}                 -> decode configuration: object keys "schemaVersion" and "sChemAVErsion" differ only in case
{"schemaVersion":1,"SCHEMAVERSION":0}                 -> decode configuration: object keys "schemaVersion" and "SCHEMAVERSION" differ only in case
{"schemaVersion":1,"schemaversion":1}                 -> decode configuration: object keys "schemaVersion" and "schemaversion" differ only in case
schemaVersion: 1\nschemaversion: 0\n                  -> decode configuration: object keys "schemaVersion" and "schemaversion" differ only in case
{"schemaVersion":1,"limits":{"maxInteractions":1},"Limits":{"maxInteractions":2}}
                                                      -> decode configuration: object keys "limits" and "Limits" differ only in case
{07,2} (1000x, always)                                -> convert YAML configuration: object key 2 is not a string
{a: {1: x}, b: {2: y}} (1000x, always)                -> convert YAML configuration: object key 1 is not a string
two null questions (1000x, always)                    -> stubs[0].when.questions.a.value must not be null
two invalid question types (1000x, always)            -> stub "x" when: question "a" has invalid type "bogus"
```

## Tests added (inside `internal/config`, no `fuzz_test.go`, no `Fuzz*`)

`internal/config/collision_test.go`

- `TestLoadNeverReturnsConfigThatValidateRejects` — C1 invariant
  (`err == nil` implies `cfg != nil` and `Validate(cfg) == nil`) over the
  specifier's corpus: both F1 seeds, the equal-value seed, the YAML block form, a
  case variant of `mode`, a nested case variant, valid YAML/JSON defaults and
  four malformed documents (truncated JSON, `[]`, `null`, empty input).
- `TestLoadRejectsCaseVariantDuplicateKeys` — C2 table: `schemaVersion`
  (`sChemAVErsion`, `SCHEMAVERSION`, `schemaversion`, YAML block form), and one
  colliding pair each for `mode`, `server`, `limits`, `compatibility`, `models`,
  `stubs`, plus nested pairs in `server`, `when` and `then.raw`. Each row asserts
  a non-nil error and a nil `*Config`.

`internal/config/determinism_test.go`

- `TestLoadDiagnosticsAreDeterministic` — C3, `N = 1000` calls per input, outcome
  = (failed?, exact message). Corpus: `{07,2}`, `{a: {1: x}, b: {2: y}}`, the
  two-bad-question-type YAML document, the two-null-question JSON document, the
  raw-header document, and two accepted documents.

No existing test was modified.

## Pre-fix failure evidence (throwaway copy)

Throwaway tree built from the unmodified revision, with only the two new test
files copied in:

```bash
git archive HEAD | tar -x -C /tmp/fj064-prefix          # product code at ab45ac5
cp internal/config/collision_test.go internal/config/determinism_test.go \
   /tmp/fj064-prefix/internal/config/
cd /tmp/fj064-prefix
go test ./internal/config -count=1 \
  -run 'TestLoadNeverReturnsConfigThatValidateRejects|TestLoadRejectsCaseVariantDuplicateKeys|TestLoadDiagnosticsAreDeterministic'
```

Outcome: exit `1`, `FAIL fake-jev/internal/config` — all three test functions
fail, 20 failing subtests in total:

- C1: 3 of 12 subtests fail, all with
  `Load returned &config.Config{SchemaVersion:0, ...} with a nil error, but
  Validate rejects it: schemaVersion must equal 1`.
- C2: all 13 subtests fail, e.g.
  `collision_test.go:70: case-variant duplicate keys loaded successfully:
  &config.Config{... Server:config.ServerConfig{Host:"0.0.0.0", Port:80} ...}`
  (the case-variant `SERVER` object overwrote the decoded server while
  `validateDocument` inspected the raw `server` key).
- C3: 4 of 7 subtests fail, exactly the specifier's predicted inputs 1–4:
  `call 9 returned ... object key 2 is not a string, call 1 returned ... object
  key 7 is not a string`; `call 6 returned ... object key 2 ... call 1 returned
  ... object key 1`; `call 5 returned ... question "b" has invalid type "bogus2",
  call 1 returned ... question "a" has invalid type "bogus"`; `call 11 returned
  ...questions.b.value must not be null, call 1 returned
  ...questions.a.value must not be null`. The two accepted inputs and the
  raw-header input were already deterministic, as the specifier expected.

Differential check of the fix, same corpus of unambiguous inputs, pre-fix vs
post-fix tree: `diff` of the two outcome dumps is empty
(`UNAMBIGUOUS-CASE OUTCOMES IDENTICAL`), including lone case-variant keys the
decoder resolves without a collision, e.g. `{"server":{"HOST":"127.0.0.1"}}` and
`{"...","when":{"QUESTIONS":{"a":"noul"}}...}`, which still load unchanged.
`{"SCHEMAVERSION":1}` (no exact `schemaVersion` key) still fails with
`schemaVersion is required`, unchanged.

## No-regression evidence on unambiguous documents

- `internal/config/validate_test.go` (unchanged, run explicitly): `TestLoadDefaults`,
  `TestRejectsMultipleYAMLDocuments`, `TestAcceptsYAMLFlowDocument`,
  `TestJSONAndYAMLLoadSameModel`, `TestRejectsUnknownAndInvalidConfiguration`
  (7 subtests), `TestRawBodyMayBeOmitted`, `TestRejectsNullAndEmptyStructuredValues`,
  `TestExpectForms` — all PASS.
- `internal/cli/validate_test.go`: `TestValidateAcceptsValidConfigurationFiles`
  (`yaml`, `json`, `questions and answers`, `defaults only`),
  `TestValidateRejectsMalformedFixtureDocuments` (including
  `duplicate json keys in an answer`), `TestValidateHandlesLargeFixtureKeySets`
  (4000 question keys, `accepted` and `one extra answer`),
  `TestValidateStreamsAreDisjoint`, `TestValidateIsStricterThanTheServeLoadPath`,
  `TestValidateRejectsStaticallyInvalidFixtureData` — all PASS.
- `internal/control/handlers_test.go` `TestControlStubSchemaPreservation`
  (12 subtests, including `nested duplicate` and `sequence duplicate`) — PASS.
- Fixture-level differential: on both trees every JSON/YAML data file in the
  repository (30 files: `examples/fake-jev.yaml`, `testdata/contracts/**`,
  `spec/**`, `package.json`, `.github/**`) was loaded and the outcome
  (`json.Marshal` of the accepted config, or the exact error text) dumped; the
  two dumps are byte-identical. `examples/fake-jev.yaml` is the one accepted
  document; the rest are rejected identically before and after.
- `testdata/contracts/**` are exercised through the CLI/control/integration
  suites above, which all pass.

## Commands run (final workspace) and outcomes

| Command | Outcome |
|---|---|
| `gofmt -w internal/config/*.go` then `gofmt -l internal/config` | no output (clean) |
| `go test ./internal/config -count=1` | `ok fake-jev/internal/config 0.238s` |
| `go test ./... -count=1` | all packages `ok` (cli 24.4s, guard 9.4s, integration 1.5s, config 1.7s) |
| `go test -race ./... -count=1` | all packages `ok` |
| `go vet ./...` | silent, exit 0 |
| `go build ./cmd/fake-jev` | exit 0 |
| `go run ./cmd/guard arch` | `findings: []`, files 62, packages 10, exit 0 |
| `go run ./cmd/guard lint` | `findings: []`, files 62, packages 10, exit 0 |
| `go run ./cmd/guard trace` | `findings: []`, covered 17, test_files 24, exit 0 |
| `go run ./cmd/guard fuzz` | `findings: []`, targets 1, exit 0 |
| `./scripts/verify-candidate FJ-064` | exit 0, `summary: 20 passed, 0 failed, 0 skipped, 0 not_applicable`, `RESULT: PASS`, report `.agent/reports/FJ-064/report-20261003T121740Z.json` |
| `./scripts/candidate-fingerprint candidate` | `74245107247629f9c366043ed7aada0dfbe29f4b63157bd6218afc3ee84a83d0` (twice, identical) |

## verify-candidate row analysis

`RESULT: PASS`; every row passed, so there is **no failing row to report**. In
particular the specifier/PM expectation of a red
`complete items have recorded-report coverage` row did not materialise: FJ-064 is
`status: implementing`, and the verifier only requires recorded-report coverage
for items whose status is `complete` (this run lists coverage for the 26 complete
items, not for FJ-064). Rows: candidate fingerprint, repository files, script
executability, shell syntax, python3 toolchain, authoritative spec intact,
`.agent` JSON parses, work-item schema, recorded-report coverage, role packs,
`trackedBy` resolution, `trackedBy` not complete, `allowedFiles` classification,
`requiredStages` shape, `go test`, `go vet`, `go build`, guard arch, guard lint,
guard trace — all PASS. The item was not closed while mid-chain; `state.json` was
not edited by this stage.

## Residual risks

1. Document-wide scope: case-variant keys inside opaque payloads
   (`then.raw.body`, `when.state`, `then.raw.headers`) are now rejected. This is
   the specifier's recorded, accepted residual; an HTTP-header-style payload that
   lists the same header twice with different casing will no longer load.
2. `foldKey` deliberately reproduces `encoding/json`'s folding instead of using
   `strings.EqualFold` per comparison (EqualFold has no canonical form usable as
   a map key). The helper is exercised only through ASCII cases in the tests; the
   equality relation it implements is pinned by construction to
   `appendFoldedName`/`foldRune` in the standard library.
3. `Load` now walks the document three times for JSON input (duplicate probe,
   case-variant scan, struct decode). This is linear in document size and does
   not impose the size bound that is explicitly out of scope.
4. FJ-040's config fuzz target (`internal/config/fuzz_test.go`) does not exist on
   this branch, so the two properties are demonstrated by the finite corpora
   above plus `guard fuzz` (1 target). FJ-040's own fuzz run remains the
   exhaustive check once it resumes; nothing in this change can define its
   symbols.
