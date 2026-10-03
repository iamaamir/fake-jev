---
stage: specifier
task: FJ-064
inputFingerprint: 8c80ab298a252de531dbf9f61a0f67e866fdf5e091cbf8fb80d272d988305c29
outputFingerprint: 8c80ab298a252de531dbf9f61a0f67e866fdf5e091cbf8fb80d272d988305c29
taskFingerprint: 71aa852bd544dfae210d4e52273e9036c9347ba7f7d92dacc4586a9c9da6e42a
gitHead: ab45ac5
generatedAt: 2026-10-03T11:50:27Z
author: worker/FJ-064-specifier
---

# Specifier — FJ-064

## What is delivered

Two confirmed product defects in `internal/config` are turned into observable,
independently checkable acceptance criteria:

- **F1** — `config.Load` is fail-open on JSON object keys the decoder resolves
  ambiguously. A key that differs only in case from a known top-level key is
  matched case-insensitively by `encoding/json` and silently overwrites the
  decoded field, so `Load` returns `(cfg, nil)` for a document that
  `config.Validate(cfg)` rejects.
- **F2** — `config.Load`'s diagnostics are non-deterministic when they depend on
  Go map iteration order. The recorded repro is a YAML flow mapping with
  non-string keys, but the same defect class exists in every diagnostic that
  reports "the first" element of a Go map.

`allowedFiles` is exactly `internal/config/`. No product code is written by this
stage; this artifact fixes the observable criteria and the exhaustive checks.
Mechanism is *not* prescribed by the specification (see "Mechanism"), except for
the outcome the specification already fixes: §12.3 requires unknown keys to fail
validation, §38.1 requires `schemaVersion` to equal `1`, and the repository rule
"strict behavior must fail closed" (`AGENTS.md`) requires the load/validate gap to
close. F2 violates the determinism principle recorded at spec line 143
— "tests must be deterministic and reproducible" — which `AGENTS.md` generalises
into the project rule that behaviour must be deterministic; a product that emits
different diagnostics for identical input contradicts it.

## Grounding (re-derived from the code at `ab45ac5`)

- `internal/config/load.go` `toJSON`: when the first non-space byte is `{`/`[`,
  `rejectDuplicateJSON` is tried first; **if it succeeds** the *original* bytes are
  returned unchanged (input key order preserved), otherwise control falls through
  to the YAML decoder (needed so YAML flow syntax is accepted). For YAML input the
  document is decoded to `any`, normalised to `map[string]any`, and re-marshalled
  by `json.Marshal` (which sorts object keys). So one config object can reach
  `encoding/json` in two different key orders depending on the input format.
- `rejectDuplicateJSON`/`scanJSONValue` compare object keys **exactly**
  (`seen[name]`), so a case-variant pair passes.
- `encoding/json` matches object keys to struct fields case-insensitively, so a
  later case-variant key overwrites the field set by the exact key.
- `validateDocument` re-reads the raw document by exact key
  (`root["schemaVersion"] == 1`), and `validateValues` never re-checks
  `cfg.SchemaVersion`; `config.Validate` does. That is the load/validate gap.
- Confirmed empirically: `{"schemaVersion":1,"sChemAVErsion":0}` (raw JSON,
  order preserved) and `schemaVersion: 1\nschemaversion: 0\n` (YAML block,
  marshalled to `{"schemaVersion":1,"schemaversion":0}`) both decode
  `Config.SchemaVersion == 0` with no error. `sChemAVErsion`, `SCHEMAVERSION`
  and `schemaversion` all fold to `schemaVersion`.
- F2: `normalizeYAMLValue`'s `map[any]any` case reports whichever non-string key
  the `range` yields first; Go randomises that. `{07,2}` decodes to
  `map[any]any{7: nil, 2: nil}` and alternates between
  `convert YAML configuration: object key 7 is not a string` and
  `... object key 2 is not a string`.
- Same defect class found during this stage (within `allowedFiles`, so amendable
  here): `internal/config/validate.go` has two additional map ranges whose first
  error is reported, so identical input can yield different messages when two
  invalid entries exist:
  - `validateWhenDocument` (line ~334) `for question, kind := range questionObject`
    calls `rejectNullFields` with the question name embedded in the message;
  - `validateWhen` (line ~434) `for name, kind := range when.Questions` returns
    `question %q has invalid type %q`.
  (`validateRaw`'s `for name := range raw.Headers` reports a name-independent
  message today, so it is deterministic; sort it only for uniformity.)
  These are reachable from untrusted configuration and are the same
  "identical input, different diagnostic" violation accepted by FJ-064's
  acceptance ("identical input always produces an identical diagnostic message").
  Covering them is inside acceptance, not new scope.

## Acceptance criteria

Every command below is deterministic, offline, and free of sleeps/randomness.
`N` for determinism is fixed at **1000** identical calls per input.

### C1 — Inverted invariant: `Load` can never return a config its own `Validate` rejects

**Observable.** For every input `in`:

```text
cfg, err := Load(in)
err == nil  ⇒  cfg != nil  ∧  Validate(cfg) == nil
```

Equivalently `Validate(cfg) != nil ⇒ err != nil`. After the fix, the seed
`{"schemaVersion":1,"sChemAVErsion":0}` must return `(nil, non-nil error)`.

**Exhaustive, not case by case.** The invariant is a property of *arbitrary*
bytes, so it is checked by a fuzz target over `[]byte` input, not by one case per
key: for every generated input, if `Load` returns a nil error then `Validate` on
the returned `*Config` must also be nil. FJ-040's config fuzz target
(`internal/config/fuzz_test.go`, which does not exist on this branch) is that
exhaustive check; FJ-064 additionally pins a finite adversarial corpus (table
test) so the property is observable now, without FJ-040. The corpus must include
at least: the F1 seeds, `{"schemaVersion":1,"sChemAVErsion":1}`,
`{"schemaVersion":1,"SCHEMAVERSION":0}`, `schemaVersion: 1\nschemaversion: 0\n`,
a valid defaults document (`schemaVersion: 1`), and a handful of structurally
malformed documents (truncated JSON, `[]`, `null`, empty input).

**Command.**

```bash
go test ./internal/config/ -run 'TestLoadNeverReturnsConfigThatValidateRejects' -count=1 -v
```

**Expected.** `--- PASS: TestLoadNeverReturnsConfigThatValidateRejects` and
`ok  fake-jev/internal/config`.

**FJ-040-side command (once it resumes).**

```bash
go test ./internal/config/ -run '^$' -fuzz '^Fuzz' -fuzztime=30s
```

**Expected.** `ok fake-jev/internal/config` with no failing input.

### C2 — Case-variant collision on any known top-level key is rejected

**Observable.** `Load` returns `(nil, non-nil error)` for every document that
contains, inside a single object that the decoder resolves to the same schema
struct, two keys that are equal under the decoder's case-insensitive matching.
At minimum the top-level `Config` object:

| Input | Colliding keys |
|---|---|
| `{"schemaVersion":1,"sChemAVErsion":0}` | `schemaVersion` / `sChemAVErsion` |
| `{"schemaVersion":1,"SCHEMAVERSION":0}` | `schemaVersion` / `SCHEMAVERSION` |
| `{"schemaVersion":1,"schemaversion":1}` | `schemaVersion` / `schemaversion` (equal values — collision alone must reject) |
| `schemaVersion: 1\nschemaversion: 0\n` | YAML block form of the same |
| `{"schemaVersion":1,"mode":"strict","MODE":"strict"}` | `mode` / `MODE` |
| `{"schemaVersion":1,"server":{"host":"a","port":1},"SERVER":{"host":"b","port":2}}` | `server` / `SERVER` |
| `{"schemaVersion":1,"limits":{...},"Limits":{...}}` | `limits` / `Limits` |
| `{"schemaVersion":1,"compatibility":["jev/v1"],"Compatibility":["jev/v1"]}` | `compatibility` / `Compatibility` |
| `{"schemaVersion":1,"models":[...],"MODELS":[...]}` | `models` / `MODELS` |
| `{"schemaVersion":1,"stubs":[],"STUBS":[]}` | `stubs` / `STUBS` |

The fix must cover **all** keys `internal/config/schema.go` declares at the
top level (`schemaVersion`, `server`, `mode`, `compatibility`, `limits`,
`models`, `stubs`) and must not special-case `schemaVersion`. Rationale:
`encoding/json` performs case-insensitive field matching at *every* struct level,
so the silent-overwrite mechanism is identical for all of them; a
`schemaVersion`-only carve-out would leave e.g. `SERVER` silently replacing
`server` (decoded `cfg.Server` then carries the `SERVER` host/port while the raw
document keeps the `server` object, and `validateServer` passes). The mechanism
should generalise to the project-owned nested schema objects as well
(`server`, `limits`, `models[i]`, `stubs[i]`, `when`, `then`, `expect`), because
that is the same defect one level down (e.g. `{"server":{"port":1,"PORT":70000}}`
is caught only by the port range check today, not as an ambiguity).

**Command.**

```bash
go test ./internal/config/ -run 'TestLoadRejectsCaseVariantDuplicateKeys' -count=1 -v
```

**Expected.** `--- PASS` and `ok fake-jev/internal/config`.

### C3 — Every diagnostic is byte-identical for identical input

**Observable.** For each input in the determinism corpus, `N = 1000` successive
`Load` calls yield **exactly one** distinct outcome, where an outcome is the pair
`(err == nil, fmt.Sprintf("%v", err))` (nil error counts as one fixed outcome).
A byte-identical error message is required; whitespace or formatting changes
between calls are failures.

Determinism corpus (each input repeated `N` times):

1. `{07,2}` — the F2 repro (two non-string YAML keys).
2. `{a: {1: x}, b: {2: y}}` — nested maps whose recursive walk order is
   currently randomised across sibling string keys.
3. `schemaVersion: 1\nstubs:\n- id: x\n  profile: jev/v1\n  when:\n    questions:\n      a: bogus\n      b: bogus2\n  then:\n    answers: {}\n`
   — two invalid question types; `validateWhen` reports whichever the map yields
   first.
4. `{"schemaVersion":1,"stubs":[{"id":"x","profile":"jev/v1","when":{"questions":{"a":null,"b":null}},"then":{"answers":{}}}]}`
   — two null questions; `validateWhenDocument` reports whichever it yields first.
5. `{"schemaVersion":1,"stubs":[{"id":"r","profile":"jev/v1","when":{},"then":{"raw":{"status":200,"headers":{"":"x"}}}}]}` — exercised for its raw-header path (its diagnostic is already key-independent and therefore deterministic; included so the corpus covers the raw-header walk).
6. `{schemaVersion: 1}` and `{"schemaVersion":1}` — accepted inputs (one fixed
   nil-error outcome each).

The criterion is "every diagnostic", not only the recorded F2 input, because
FJ-040's fuzz target asserts determinism for arbitrary input and will otherwise
find inputs 3 and 4 (same map-iteration class).

**Command.**

```bash
go test ./internal/config/ -run 'TestLoadDiagnosticsAreDeterministic' -count=1 -v
```

**Expected.** `--- PASS` and `ok fake-jev/internal/config`. Pre-fix, this test
must fail on at least inputs 1–4 (multiple distinct messages). `N` is recorded
as **1000**; with a two-way randomised choice, `P(all 1000 equal) ≈ 2·2⁻¹⁰⁰⁰`,
so the pre-fix failure is not a flake.

### C4 — No regression on unambiguous input

**Observable.** Every configuration that loads today on unambiguous input still
loads with identical decoded values, and every configuration rejected today for a
non-collision reason is still rejected. The establishing tests and fixtures
(unchanged, run as-is):

- `internal/config/validate_test.go`: `TestLoadDefaults`,
  `TestAcceptsYAMLFlowDocument`, `TestJSONAndYAMLLoadSameModel`,
  `TestRejectsMultipleYAMLDocuments`, `TestRejectsUnknownAndInvalidConfiguration`,
  `TestRawBodyMayBeOmitted`, `TestRejectsNullAndEmptyStructuredValues`,
  `TestExpectForms`.
- `internal/cli/validate_test.go`: `TestValidateAcceptsValidConfigurationFiles`
  (`validYAML`, `validJSON`, `questionsYAML`, `"schemaVersion: 1\n"`),
  `TestValidateRejectsMalformedFixtureDocuments` (exact duplicate keys must stay
  rejected), `TestValidateHandlesLargeFixtureKeySets`,
  `TestValidateStreamsAreDisjoint`.
- `internal/cli/serve_test.go`, `internal/cli/run_test.go`,
  `internal/cli/logging_test.go` — all `config.Load`/`LoadFile` call sites.
- `internal/control/handlers_test.go` `TestControlStubSchemaPreservation`,
  including `nested duplicate` and `sequence duplicate`, which prove that
  duplicate-key rejection already applies document-wide (inside `when.state` and
  `then.raw.sequences`).
- `examples/fake-jev.yaml`; the contract fixtures under `testdata/contracts/`;
  `spec/config.schema.json`.

**Command.**

```bash
go test ./... -count=1
go vet ./...
./scripts/verify-candidate FJ-064
```

**Expected.** `ok` for every package; `go vet` silent; `verify-candidate` exits
`0` with `RESULT: PASS`.

### C5 — Interaction with FJ-040 (blocked dependency)

FJ-040 is `blocked` with `blocker: dependency`/`trackedBy: FJ-064`. Its config
fuzz target `internal/config/fuzz_test.go` **does not exist on this branch** and
asserts precisely the two properties above (fail-closed `Load` and deterministic
diagnostics). Therefore:

- the fix must make those two properties hold for arbitrary input (C1 + C3), so
  FJ-040's `guard fuzz` exits non-zero-free once it resumes and
  `./scripts/verify-candidate` goes green for FJ-040;
- FJ-064 must **not** create `internal/config/fuzz_test.go` and must not define a
  package-level `Fuzz*` function that FJ-040 will declare (duplicate symbol).
  FJ-064's regression tests live in the existing `internal/config/validate_test.go`
  or a new file in that package *not* named `fuzz_test.go`
  (for example `internal/config/collision_test.go`).

**Command (deferred, FJ-040-side).**

```bash
go test ./internal/config/ -run '^$' -fuzz '^Fuzz' -fuzztime=30s
```

**Expected.** `ok fake-jev/internal/config`; no failing input; no panic.

## Mechanism — spec-silent and therefore a coder choice

The specification does not prescribe *how* keys are compared or where the check
lives; it only fixes the outcomes (fail closed, deterministic diagnostics). The
PM recorded two candidate approaches. This stage recommends one.

**Recommended — Mechanism A: case-insensitive collision detection in the
duplicate-key scan, run on the JSON that the decoder actually consumes.**

- The document that reaches `encoding/json` is the one that can be ambiguous, so
  the scan must run on exactly those bytes. In `Load`, after `toJSON` returns
  `jsonData` (which is the original JSON when `rejectDuplicateJSON` succeeded, and
  the re-marshalled/sorted JSON for YAML), walk `jsonData` and, for each object,
  reject two keys that are equal under the decoder's case-insensitive matching
  (`strings.EqualFold`, the same folding `encoding/json` uses). Returning this
  error from `Load` (not from the JSON-detection fast path) is required because
  `toJSON` deliberately swallows `rejectDuplicateJSON`'s error to admit YAML flow
  syntax, and because YAML block input only becomes a decoder-visible key
  collision after normalisation and re-marshalling.
- It rejects the ambiguous document directly, is fail-closed, and covers every
  known top-level key (and nested schema objects) with one local change and no
  schema enumeration, so no key can be special-cased.
- It leaves exact-duplicate rejection at least as strong as today.

**Not recommended — Mechanism B: post-decode consistency check.** Re-decoding and
comparing against the raw document, or re-running `Validate`, cannot satisfy C2:
a collision whose two values agree (`sChemAVErsion: 1` beside `schemaVersion: 1`)
leaves decoded and raw values equal and would still load, and collisions on
fields `Validate` does not range-check (e.g. a case-variant `server`) still
silently overwrite. It also cannot make the message deterministic. B is at most a
belt-and-suspenders addition (for example also re-asserting
`cfg.SchemaVersion == 1` in `validateValues`, which is redundant once A holds);
that addition is a coder choice, not required by acceptance.

**Determinism mechanism (implementation choice, same reasoning).** Remove
map-iteration-order dependence from every diagnostic: impose a deterministic
total order on the keys before ranging (sort string-keyed maps; for
`map[any]any` in `normalizeYAMLValue`, sort by a total, deterministic
representation of each key, or emit a key-independent message), in
`normalizeYAMLValue` and in the `validate.go` walks listed under "Grounding".
The exact message text is spec-silent; only byte-identical repetition for
identical input is required (C3).

## Spec-silent observations

1. **Mechanism and placement** are not specified — a coder choice (above).
2. **Diagnostic text** (wording, which colliding key is named, whether the key is
   named at all) is not specified. Only determinism is required (spec line 143 /
   repository rule).
3. **Case-variant keys in open containers** (`then.raw.body`, `when.state`,
   question content) — §38.8 exempts these from *unknown-key* rejection but is
   silent on case-variant duplicates. Duplicate rejection is already
   document-wide today (exact duplicates inside `when.state` and
   `then.raw.sequences` are rejected — `TestControlStubSchemaPreservation`,
   `TestValidateRejectsMalformedFixtureDocuments`), so a document-wide
   case-insensitive scan is consistent rather than a new principle. Residual:
   a provider-emulation body that legitimately contains two keys differing only
   in case would newly be rejected. If the coder scopes the scan to
   decoder-resolved objects instead, C2 still holds and this residual disappears;
   either is acceptable, and this is the one place where the two mechanisms
   differ observably beyond the top-level criteria. The recommended default is
   document-wide (matches existing exact-duplicate behaviour, maximally
   fail-closed).
4. **Folding semantics**: `encoding/json` matching is Unicode-case-insensitive;
   `strings.EqualFold` mirrors it. ASCII `strings.ToLower` happens to agree on
   the recorded cases; use the decoder's own folding to avoid a divergence.
5. **Broader F2 class** (the `validate.go` map walks) was not in the recorded
   repro but is the same acceptance property; covering it is in scope.
6. **No spec-gap**: the specification is silent on mechanism but the outcomes are
   fully determined by §12.3, §38.1, and the repository fail-closed rule, so no
   new public behaviour is invented and no `spec-gap` blocker is raised.

## Out of scope (must not be folded in)

- **Bounding configuration file size or parse cost.** The unbounded
  configuration-file read recorded after FJ-030 and the `yaml.v3`
  anchor/alias amplification measured by the FJ-040 cleaner (a 62,837-byte
  alias-using document allocating ~51.7 MB in 97 ms) are explicitly **not**
  fixed here. If FJ-040's fuzz target turns out to need a bound, that is a
  separate decision and a separate work item.
- No new dependency; no change to `spec/config.schema.json`; no change to any
  currently unambiguous accepted input; no behaviour change outside
  `internal/config/`.
- No case-sensitive decoder replacement and no change to
  `DisallowUnknownFields` behaviour.

## Traces

- C-CFG-001 — §38.1: `schemaVersion` required and equal to integer `1`; F1
  returns `SchemaVersion == 0` with a nil error today.
- C-CFG-002 — §12.3, §38.8: unknown keys fail validation; the fail-closed
  principle that a case-variant key collision must not load.
- C-CFG-007 — §12.2: omitted sections take built-in defaults; must be unchanged
  by the fix.
- C-CFG-009 — §12: YAML and JSON deserialize to the same logical schema; F1
  reaches the decoder through both formats (raw JSON preserves order, YAML is
  re-marshalled), so the check must cover both.
- C-QUAL-001 — §31: `go test ./...` passes.
- C-QUAL-002 — §23: `go test -race ./...` passes.
- C-QUAL-004 — §22.2: malformed untrusted input yields a controlled error, never
  a panic or corrupted state; the FJ-040 fuzz target encodes this.
- Spec line 143 — §2: tests (and the product) must be deterministic; F2's
  varying diagnostics violate it.
- §12.3 unknown-key rejection and `AGENTS.md` "strict behavior must fail closed"
  — the normative basis for closing the `Load`/`Validate` gap (C1).

## Test plan (coder stage)

Regression tests belong in `internal/config/` (allowed), in
`internal/config/validate_test.go` or a new file not named `fuzz_test.go`:

- `TestLoadNeverReturnsConfigThatValidateRejects` (C1, corpus).
- `TestLoadRejectsCaseVariantDuplicateKeys` (C2, the table above).
- `TestLoadDiagnosticsAreDeterministic` (C3, `N = 1000`, the corpus above).
- Existing tests unchanged (C4).

No product behaviour is specified beyond the criteria above; the coder owns the
implementation, the message text, and the exact scan placement within
`internal/config/`.
