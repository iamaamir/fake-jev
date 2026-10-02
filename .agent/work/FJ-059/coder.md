---
stage: coder
task: FJ-059
inputFingerprint: b108a8dd28dfd3b53b39add62b2ccc81d1fa0942c1b1cc27400dd98e805ddf55
outputFingerprint: bb445e590122427e20db13937225cf7a5b4eda3d9f223d907b07463fbc6e7248
taskFingerprint: 879b10dee885044f7a81eed718e8b111e040775d016d6fa1c2bb8575fdbcd179
gitHead: 26c104f
generatedAt: 2026-10-02T20:22:56Z
author: worker/FJ-059-coder
---

# Coder — FJ-059 (static jev/v1 fixture validation for `fake-jev validate`)

**Final candidate fingerprint: `bb445e590122427e20db13937225cf7a5b4eda3d9f223d907b07463fbc6e7248`**
(previously `b108a8dd28dfd3b53b39add62b2ccc81d1fa0942c1b1cc27400dd98e805ddf55`).

## What was implemented, per criterion

### A1 / A2 / A3 — `answers` key set equals the configured `when.questions` key set

- New exported validator `v1.ValidateFixtureAnswers(questions map[string]string, answers json.RawMessage) error`
  in `internal/compat/jev/v1/fixture.go`. It mirrors `GenerateAnswers`'s coverage
  loop over the **configured** question set instead of the request set:
  length equality first, then `missing answer for question %q`, then
  `answer provided for absent question %q`. Empty `answers` (`len == 0`) means
  the stub uses another response form (raw) and is accepted.
- A1/A3: `internal/cli/validate.go` calls it once per stub in configuration
  order via the new `validateFixtureData(cfg)`, so the first violating stub is
  reported regardless of position.
- A2: the same call is made for every `then.sequence[i]`, with the element index
  in the diagnostic (`stub %q sequence[%d]: ...`).

### B1 — answer helper matches the configured question type

- The static path dispatches on the configured type exactly as
  `generateAnswer` does; a `noul` question validated with the noul rules rejects
  a `choice`/`score` payload as `noul helper is required`, and so on.

### B2 — `noul` payload rules

- The static path calls the **existing** `generateNoulAnswer(fields)` and keeps
  only its error, so the boolean convenience form, the `[0,1]` finite-number
  rule, and the exact-one-field rule are not restated.

### B3 / B4 — `choice` / `score` payload rules

- New `validateChoiceAnswerFields` / `validateScoreAnswerFields` reuse
  `validateHelperFields` (allowed fields — this is what rejects a `legend` on
  `score`), `probabilitiesField` (object shape, `jsonNumber`, finite `[0,1]`),
  `confidenceField` (finite `[0,1]`), `finiteNumber`/`jsonNumber`, and
  `validateProbabilityMap` for the sum-to-`1.0`-within-`1e-6` rule, passed the
  map's **own** key list because only the fixture's values and sum are
  statically known. `choice` must be a JSON string (`isString`); `score` must be
  a finite JSON number. `{member: null}` / `{member: 3}` /
  non-object `answers` are rejected as `helper must be an object` /
  `answers must be an object`.

### B5 — document vs. member errors

- Non-object `then.answers` is still rejected by `config.Load` first (unchanged);
  a non-object member is rejected by the compat validator.

### Wildcard stubs (§38.6 `questions` omitted)

- `questions == nil` is the wildcard: key-set equality and helper/type agreement
  are skipped (the request name/type set is unknown), but every payload is still
  validated by dispatching on the helper the answer declares. A payload that
  declares no helper, or a helper conflicting with a sibling field, is rejected
  by the selected helper's own field rules.

### C1 — exit/stream discipline

- `runValidate` now: load → `validateFixtureData` → success line. A fixture
  violation writes `fake-jev: <path>: <err>` on stderr, nothing on stdout, and
  returns `exitFailure` (2), the same shape as an existing load failure. Usage
  failures are untouched.

### C2 — compat layer + composition root

- Validator lives in `internal/compat/jev/v1/fixture.go`; `internal/cli/validate.go`
  imports `fake-jev/internal/compat/jev/v1` and drives the per-stub loop.
  `internal/config` was not touched, so it still does not import the compat
  layer.

### C3 — serve load path unchanged

- No file under `internal/config`, `internal/host`, or `cmd/` is modified
  (`git diff --name-only -- internal/config internal/host cmd` is empty), and
  the diff to `fixture.go` is **additions only** (152 insertions, 0 deletions,
  confirmed by `git diff -- numstat` and by grepping removed lines).

### C4 — no restated rules

- The new code calls `generateNoulAnswer`, `validateHelperFields`,
  `probabilitiesField`, `confidenceField`, `validateProbabilityMap`,
  `finiteProbability`/`finiteNumber`, `jsonNumber`, `objectFields`, `isString`.
  No existing function body was edited. Tests include
  `TestValidateFixtureAnswersNeverAcceptsWhatGenerationRejects`, which asserts
  the shared subset cannot diverge: every fixture `GenerateAnswers` accepts for
  a question set must also be statically valid.

## Rules explicitly NOT enforced, and why (not statically decidable)

The request `criteria` never appear in the configuration, so the static path
deliberately does not attempt:

1. §13.2 choice `probabilities` keys equal the request criteria key set.
2. §13.2 the selected `choice` exists as a request criterion.
3. §13.2 the selected `choice` holds the maximum probability.
4. §13.2 `confidence` defaulting to the selected choice's probability.
5. §13.3 score `probabilities` keys are exactly `"0"`…`"N-1"` (`N` is
   request-derived).
6. §13.3 score within `[0, N-1]`.
7. §13.3 score probability-weighted expected value equals `score`.
8. §13.4/§13.5 request-time exact-cover rule when `questions` is a wildcard.

These remain in `GenerateAnswers`/`generateAnswer` at request time. This is the
statically provable subset named in the specifier contract; criterion key-set
equality in particular cannot be checked because criteria come from the
incoming client request.

## Evidence that serve is unaffected

- `git diff --name-only -- internal/config internal/host cmd` → empty.
- Behavioral: the config `g2-keys.yaml` (answers key set ≠ `when.questions`) is
  rejected by `validate` with exit 2, yet `fake-jev serve --config g2-keys.yaml
  --port 0 --ready-file …` starts and publishes
  `{"url":"http://127.0.0.1:65435",…}` before being killed. Same claim is pinned
  in `TestValidateIsStricterThanTheServeLoadPath` (asserts `config.LoadFile`
  accepts, `validate` rejects; a fully valid file passes both).

## Commands and outcomes (all in `/Users/mak/git/fake-jev-FJ-059`)

| Command | Outcome |
| --- | --- |
| `gofmt -w` on the four changed files | reformatted `validate_test.go`; others already clean |
| `go test ./internal/compat/jev/v1 -count=1` | 142 passed |
| `go test ./internal/cli -count=1` | 192 passed |
| `go test ./... -count=1` | 620 passed (10 packages) |
| `go test -race ./... -count=1` | 620 passed (10 packages) |
| `go vet ./...` | clean |
| `go run ./cmd/guard arch` | exit 0, `"findings": []` |
| `go run ./cmd/guard lint` | exit 0, `"findings": []` |
| `go run ./cmd/guard trace` | exit 0, `"findings": []` (covered 17) |
| `go run ./cmd/guard fuzz` | exit 0, `"findings": []` (targets 0) |
| `go build ./cmd/fake-jev` | ok |
| `./scripts/candidate-fingerprint candidate` | `bb445e590122427e20db13937225cf7a5b4eda3d9f223d907b07463fbc6e7248` |

Additional behavioral runs against the built binary (specifier A1/B2/C1):

- `ok-keys.yaml` (`when.questions` = `{urgent, route}`, both answered, explicit
  choice probabilities summing to 1): exit 0, stdout
  `fake-jev: ok-keys.yaml: configuration is valid`, stderr empty.
- `g2-keys.yaml`: exit 2, stdout empty, stderr
  `fake-jev: g2-keys.yaml: stub "mismatch": missing answer for question "urgent"`.
- `g2-sequence.yaml`: exit 2, stdout empty, stderr
  `fake-jev: g2-sequence.yaml: stub "g2-sequence" sequence[1]: missing answer for question "urgent"`.
- `g1-noul-range.yaml` (`noul: 1.5`): exit 2, stdout empty, stderr
  `fake-jev: g1-noul-range.yaml: stub "g1-noul-range": answer "urgent": noul must be a finite number in [0,1] or a boolean`.

No normative wording is fixed by the specification; the coder records the exact
text above, and the tests assert exit code, stream split, and the
`stub "<id>"` token rather than a fixed sentence.

## Changed files

- `internal/compat/jev/v1/fixture.go` — added `ValidateFixtureAnswers`,
  `validateConfiguredAnswer`, `validateDeclaredAnswer`,
  `validateChoiceAnswerFields`, `validateScoreAnswerFields` (additions only).
- `internal/compat/jev/v1/fixture_test.go` — `TestValidateFixtureAnswers`
  (table, valid per helper + one failing case per rule, sequence-element cases,
  wildcard cases) and `TestValidateFixtureAnswersNeverAcceptsWhatGenerationRejects`.
- `internal/cli/validate.go` — `validateFixtureData` and the call from
  `runValidate`.
- `internal/cli/validate_test.go` — `questionsYAML`/`fixtureViolationYAML`
  fixtures, `TestValidateRejectsStaticallyInvalidFixtureData`,
  `TestValidateIsStricterThanTheServeLoadPath`, and a questions-bearing valid
  case added to the existing acceptance table.

`.agent/work/FJ-059/state.json` and `.agent/reports/**` were not modified by
this stage.

## Observations

- Error wording is a coder choice; only exit code, stream split, and the stub
  identifier are contractual. Diagnostics mirror `GenerateAnswers`'s phrasing
  (`missing answer for question`, `answer provided for absent question`,
  `helper must be an object`, `noul helper is required`) so the two paths read
  the same.
- `choice: null` is now rejected as `choice must be a string`, which is
  marginally stricter than `generateChoiceAnswer` (which would let a `null`
  through to the criteria-membership check). Every non-null case agrees with
  generation; the divergence only affects a value no §13.2 fixture can use.
- `when.questions` is the only static source of question types, so no profile
  check is needed at the call site: `internal/config` normalizes every stub's
  profile to `jev/v1` and rejects any other value.
- The specifier's "§40.2-40.6 fixture answer rules" reference is a documentation
  discrepancy (§40 is matching/ordering/state); the rules implemented trace to
  §13.1–13.5, §12.6, §15.2, §38.6/§38.7.

## Residual risks

- Low: the sequence-element diagnostic index is 0-based (`sequence[1]` for the
  second element), which the specifier allowed at implementation option.
- Low: a wildcard stub's payload is validated by the helper it declares, so a
  fixture that will never match the request type it is asked for cannot be
  caught statically (§13.4's type agreement needs the request).
- Not run by this stage: `./scripts/verify-candidate FJ-059` (it writes
  `.agent/reports/**`, which this stage was told not to edit); it is the next
  gate and should be run against candidate
  `bb445e590122427e20db13937225cf7a5b4eda3d9f223d907b07463fbc6e7248`.
