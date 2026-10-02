---
stage: qa
task: FJ-059
inputFingerprint: 80852a3b20015001679e3171fe0271619845cf02203b70f2cb59470b976af21c
outputFingerprint: 80852a3b20015001679e3171fe0271619845cf02203b70f2cb59470b976af21c
taskFingerprint: 879b10dee885044f7a81eed718e8b111e040775d016d6fa1c2bb8575fdbcd179
gitHead: 26c104f
generatedAt: 2026-10-02T21:05:45Z
author: worker/FJ-059-qa
---

# QA — FJ-059 (static jev/v1 fixture validation for `fake-jev validate`)

Independent QA stage. Author of this artifact is `worker/FJ-059-qa`, distinct
from the coder author `worker/FJ-059-coder`. No repository file was edited:
every fixture was created under `/private/tmp/fj059-qa`, and the only file this
stage wrote in the repository is `.agent/work/FJ-059/qa.md` (plus the
verification report that `./scripts/verify-candidate` itself writes to
`.agent/reports/FJ-059/`). `state.json`, production code, and repo tests were
not touched (`git diff --cached --name-only` empty; `state.json` was already
modified by the PM before this stage).

Revision under test: working tree at `26c104f` with the FJ-059 diff applied.
Candidate fingerprint `./scripts/candidate-fingerprint candidate` =
`80852a3b20015001679e3171fe0271619845cf02203b70f2cb59470b976af21c`, equal to
the hardener `outputFingerprint`, which is used verbatim as this artifact's
input/output fingerprint. Binary under test built with
`go build -o /private/tmp/fj059-qa/fake-jev ./cmd/fake-jev`.

## Method

Each fixture is a standalone configuration in `/private/tmp/fj059-qa/fx`.
`validate` was run as `fake-jev validate <fixture>` with stdout, stderr, and the
exit code captured separately (`fx/run.sh`). For the cmd-level criteria, every
rejected fixture below was observed to exit 2 with an empty stdout and a
non-empty stderr; every accepted fixture was observed to exit 0 with the
`fake-jev: <path>: configuration is valid` line on stdout and an empty stderr.
Where a fixture violates more than one static rule, the observed diagnostic
names the first rule reached (length check, then per-name, then per-payload), so
the outcome confirms the rule named.

## Statically decidable rules — one violating fixture per rule

| Fixture | Violated rule | Observed |
| --- | --- | --- |
| `inv-keys-missing.yaml` (`when.questions: {urgent}`, `then.answers: {route}`) | answers key set ≠ configured set | exit 2, stdout empty, stderr `stub "mismatch": missing answer for question "urgent"` |
| `inv-keys-extra.yaml` (extra answer `route`) | same | exit 2, stdout empty, stderr `stub "extra": answers must exactly cover the configured questions` |
| `inv-same-count-swap.yaml` (questions `{urgent, route}`, answers `{urgent, other}`) | same, equal cardinality | exit 2, stdout empty, stderr `stub "swap": missing answer for question "route"` |
| `inv-sequence.yaml` (second sequence element answers `{route}`) | same for `then.sequence[i]` | exit 2, stdout empty, stderr `stub "g2-sequence" sequence[1]: missing answer for question "urgent"` |
| `inv-type-noul-choice.yaml` (noul question, `choice` payload) | helper type = configured type | exit 2, stdout empty, stderr `stub "type-noul-choice": answer "urgent": noul helper is required` |
| `inv-type-choice-score.yaml` (choice question, `score` payload) | same | exit 2, stdout empty, stderr `stub "type-choice-score": answer "route": choice helper is required` |
| `inv-type-score-noul.yaml` (score question, `noul` payload) | same | exit 2, stdout empty, stderr `stub "type-score-noul": answer "severity": score helper is required` |
| `inv-noul-range-high.yaml` (`noul: 1.5`) | noul finite `[0,1]` / bool | exit 2, stdout empty, stderr `stub "noul-high": answer "urgent": noul must be a finite number in [0,1] or a boolean` |
| `inv-noul-range-neg.yaml` (`noul: -0.1`) | same | exit 2, stdout empty, same message |
| `inv-noul-string.yaml` (`noul: "0.5"`) | same | exit 2, stdout empty, same message |
| `inv-noul-extra.yaml` (`noul: 0.5, confidence: 0.5`) | noul exactly one field | exit 2, stdout empty, stderr `answer "urgent": noul helper has unknown fields` |
| `inv-choice-nonstring.yaml` (`choice: 7`) | choice must be a string | exit 2, stdout empty, stderr `answer "route": choice must be a string` |
| `inv-choice-unknownfield.yaml` (`bogus: 1`) | allowed choice fields | exit 2, stdout empty, stderr `choice helper has unknown field "bogus"` |
| `inv-choice-confidence.yaml` (`confidence: 1.5`) | confidence finite `[0,1]` | exit 2, stdout empty, stderr `confidence must be a finite number in [0,1]` |
| `inv-choice-prob-range.yaml` (`{a: 1.1, b: -0.1}`) | probability `[0,1]` | exit 2, stdout empty, stderr `probability "a" must be a finite number in [0,1]` |
| `inv-choice-prob-sum.yaml` (`{a: 0.2, b: 0.2, c: 0.2}`) | probability sum within `1e-6` | exit 2, stdout empty, stderr `probabilities must sum to 1` |
| `inv-choice-prob-nonobj.yaml` (`probabilities: []`) | probabilities object shape | exit 2, stdout empty, stderr `probabilities must be an object` |
| `inv-score-legend.yaml` (`legend` sibling) | score helper must not provide `legend` (§13.3) | exit 2, stdout empty, stderr `score helper has unknown field "legend"` |
| `inv-score-nonnum.yaml` (`score: "1"`) | score finite number | exit 2, stdout empty, stderr `score must be a finite number` |
| `inv-score-confidence.yaml` (`confidence: 1.1`) | confidence finite `[0,1]` | exit 2, stdout empty, stderr `confidence must be a finite number in [0,1]` |
| `inv-score-prob-sum.yaml` (`{"0":0.5,"1":0.7}`) | probability sum within `1e-6` | exit 2, stdout empty, stderr `probabilities must sum to 1` |
| `inv-score-prob-range.yaml` (`{"0":1.2,"1":-0.2}`) | probability `[0,1]` | exit 2, stdout empty, stderr `probability "0" must be a finite number in [0,1]` |
| `inv-member-null.yaml` (`{urgent: null}`) | member must be an object | exit 2, stdout empty, stderr `answer "urgent": helper must be an object` |
| `inv-member-number.yaml` (`{urgent: 3}`) | same | exit 2, stdout empty, same message |
| `inv-wildcard-empty.yaml` (wildcard stub, `answers: {}`) | hardener fix: wildcard answers must not be empty | exit 2, stdout empty, stderr `stub "wildcard-empty": answers must not be empty` |
| `inv-answers-null.yaml` (`then.answers: null`) | config-layer (not the compat validator) | exit 2, stdout empty, stderr `stubs[0].then.answers must not be null` |
| `inv-nested-sequence.yaml` (sequence element containing `sequence`) | config-layer §38.7 | exit 2, stdout empty, stderr `decode configuration: json: unknown field "sequence"` |

## Accepted configurations — no over-rejection observed

| Fixture | Content | Observed |
| --- | --- | --- |
| `valid-everything.yaml` | one stub, six questions covering every helper: `noul` number, `noul` boolean, `choice` minimal, `choice` explicit probabilities + confidence, `score` integer, `score` fractional with explicit probabilities `{"0":0.0,"1":0.5,"2":0.5}` | exit 0, stdout `... configuration is valid`, stderr empty |
| `valid-sequence.yaml` | one stub with `then.sequence` of two `answers` elements, both valid, second with explicit probabilities summing to 1.0 | exit 0, stdout valid line, stderr empty |
| `valid-empty-questions-wildcard.yaml` | wildcard stub (`when: {}`) with a well-formed `noul` answer | exit 0, stdout valid line, stderr empty |
| `fp-tolerance-ok.yaml` | choice probabilities `{a: 0.6000005, b: 0.4}` (sum − 1.0 = 5e-7 ≤ 1e-6) | exit 0 (tolerance boundary is inclusive at `1e-6`) |

## Every-stub / every-element reachability

- `inv-last-stub.yaml`: stub `first` valid, stub `last` with an answers key-set
  mismatch → exit 2, stdout empty, stderr `stub "last": missing answer for
  question "urgent"`. The violation in the final stub is reached, so the loop
  does not stop at the first stub.
- `fp-last-seq.yaml`: stub `seqfirst` valid; stub `seqlast` with a valid first
  sequence element and `noul: 2.5` in the second → exit 2, stdout empty, stderr
  `stub "seqlast" sequence[1]: answer "urgent": noul must be a finite number in
  [0,1] or a boolean`. The final sequence element of the final stub is reached.

## Serve-path split — observed behaviour

Fixture `inv-choice-prob-sum.yaml` (probability sum rule violation) was used on
both paths.

- `fake-jev validate inv-choice-prob-sum.yaml` → exit 2, stdout empty, stderr
  `stub "choice-sum": answer "route": probabilities must sum to 1`.
- `fake-jev serve --config inv-choice-prob-sum.yaml --port 0 --ready-file
  ready.json` → process starts normally, stderr `fake-jev: listening on
  http://127.0.0.1:52864`, ready file
  `{"url":"http://127.0.0.1:52864","host":"127.0.0.1","port":52864,"pid":73980,"controlApiVersion":"v1"}`.
  A `POST /v1/systemone` for `questions.route` of type `choice` with criteria
  `{a,b,c}` returned HTTP 500
  `{"error":"fake_jev_invalid_stub_response","message":"Configured stub cannot
  produce a valid response for this request.","stubId":"choice-sum"}`.

What this shows and why it is the intended split: the provider-neutral
`internal/config` load path accepts the file (its strictness is unchanged), so
`serve` starts; the invalid fixture is caught only when a request selects the
stub, as §13.5/§43 require for response generation. `validate` is deliberately
stricter than the serve load path (§15.2 "all statically possible consistency
checks"), so rejecting at validate time without changing what `serve` accepts
is the recorded design constraint. `TestValidateIsStricterThanTheServeLoadPath`
asserts the same split for the non-wildcard and wildcard-empty fixture cases.

## False-pass attempts

| Attempt | Expected if the check were sound | Observed |
| --- | --- | --- |
| duplicate key in an answers payload, JSON input (`fp-dup-keys.json`, `"noul": 0.5, "noul": 1.5`) | rejected | exit 2 at the load boundary; stderr `decode YAML configuration: yaml: unmarshal errors: line 9: mapping key "noul" already defined at line 9`. Duplicate keys never reach the compat validator. |
| duplicate question name, JSON input (`fp-dup-keys-map.json`) | rejected | exit 2; same YAML duplicate-key message for `"urgent"`. |
| duplicate key inside a probabilities map, JSON input (`fp-dup-prob-key.json`) | rejected | exit 2; `mapping key "a" already defined`. |
| `then.answers: null` | rejected | exit 2 by `internal/config` (`stubs[0].then.answers must not be null`). |
| `{urgent: null}` member | rejected | exit 2 by the compat validator (`helper must be an object`). |
| `{urgent: []}` member | rejected | exit 2 (`helper must be an object`). |
| `confidence: null` | rejected | exit 2 (`confidence must be a finite number in [0,1]`). |
| `score: null` | rejected | exit 2 (`score must be a finite number`). |
| `probabilities: null` | rejected | exit 2 (`probabilities must be an object`). |
| expression / wrong type where a number is required: `noul: "0.5"`, `noul: 1 + 1` (YAML string), `noul: yes` (YAML string, not a v3 bool), `score: "1"` | rejected | each exit 2 with the noul/score finite-number message (`fp-noul-yaml-expr.yaml`, `fp-noul-bool-word.yaml`). |
| number beyond float64: `confidence: 1e999`, `score: 1e999` | rejected | exit 2 (`confidence must be a finite number in [0,1]`; `score must be a finite number`). |
| probability value that is a boolean (`{a: true, b: 0.0}`) | rejected | exit 2 (`probability "a" must be a finite number in [0,1]`) |
| wrong helper for the declared question type (all three directions) | rejected | exit 2, three fixtures as tabulated above. |
| probability map not summing to 1.0 within `1e-6` (`fp-tolerance-bad.yaml`, sum = 1.000002) | rejected | exit 2 (`probabilities must sum to 1`). |
| probability map at the tolerance boundary (`fp-tolerance-ok.yaml`, sum = 1.0000005) | accepted, because within `1e-6` | exit 0. Confirms the tolerance is `1e-6` inclusive and not a stricter value. |
| wildcard stub with no question type and a mixed helper (`{noul: 0.5, choice: backend}`) | rejected | exit 2 (`noul helper has unknown fields`). |
| answer member with no helper at all (`{urgent: {}}`) | rejected under a configured type | exit 2 (`answer "urgent": noul helper is required`). |
| diagnostic determinism for several unknown sibling fields (`fp-multi-unknown.yaml`, `choice` + `z1..z7`) | identical stderr every run | 40 consecutive runs all produced the identical line `choice helper has unknown field "z1"`; no variation across runs. |

Accepted-but-unservable shape found during this stage (not a slip-through of an
in-scope rule; see residual risks): `oos-score-keyset.yaml`
(`when.questions: {severity: score}`, `then.answers: {severity: {score: 1,
probabilities: {"0": 0.2, "1": 0.8}}}`) is accepted by `validate` (exit 0), yet
`serve` returned HTTP 500 `fake_jev_invalid_stub_response` for the same stub at
criteria lengths 2 and 3. This is an instance of a rule the specifier recorded
as request-relative (§13.3 key set / expected value with `N` from the request),
so it is outside the frozen subset; it is recorded here because the request's
"try to construct a false pass" instruction asks for honest reporting.

## Explicitly out-of-scope rules — not claimed anywhere

- Grep of the stage artifacts and production code for the request-relative
  criterion rules
  (`grep -rniE "criteria key set|criteria key-set|request criteria"
  .agent/work/FJ-059/*.md internal/compat/jev/v1/fixture.go internal/cli/validate.go`)
  returns only: the specifier/coder/cleaner/hardener statements that these rules
  are *not* statically decidable; the `ValidateFixtureAnswers` doc comment
  listing them as deliberately not attempted; and the pre-existing
  `validateProbabilityMap` message `probabilities must exactly cover request
  criteria`, whose static call site passes the map's own keys. No artifact or
  code claims that `validate` enforces criteria key-set equality.
- Behavioural confirmation on built-binary fixtures: `oos-choice-keyset.yaml`
  (`choice: backend`, probabilities keys `{a,b}` that cannot be the configured
  key set), `oos-choice-max.yaml` (`choice: backend` with `frontend: 0.9` the
  maximum), and `oos-score-keyset.yaml` (score keys `{"0","1"}`) all exit 0 with
  the valid line. So criterion key-set equality, choice membership/maximum, and
  score key-set/range/expected-value are indeed not enforced by `validate`.
- `state.json` acceptance lists only `C-CLI-005`; its notes state
  "Not statically decidable: choice/score criterion key-set equality, because
  criteria come from the incoming client request". The specifier's out-of-scope
  list (its §354-§395) matches the observed behaviour.

## Duplicate-restatement re-check

- The static path *calls* the shared rule bodies rather than copying them:
  `validateConfiguredAnswer` dispatches to `generateNoulAnswer` for noul, and
  `validateChoiceAnswerFields`/`validateScoreAnswerFields` call
  `validateHelperFields`, `probabilitiesField`, `confidenceField`,
  `validateProbabilityMap` (`grep -n` on `fixture.go` confirms the call sites at
  lines 333, 368, 371, 374, 381, 385, 395, 400, 404).
- Residual restatements that *could* drift, independently re-confirmed by
  reading the source, all pre-documented by the cleaner/hardener:
  1. the answers↔questions key-set loop is spelled three times —
     `GenerateAnswers` (`fixture.go:41-53`), `ValidateFixtureAnswers`
     (`fixture.go:303-312`), and `validateResponseAnswers`
     (`response.go:126-139`); message texts currently agree character for
     character (`grep` across `internal/compat/jev/v1/*.go`).
  2. the `score` finite-number expression is duplicated verbatim:
     `fixture.go:182-183` (`generateScoreAnswer`) and `fixture.go:399-400`
     (`validateScoreAnswerFields`) are the same `!jsonNumber(...) ||
     json.Unmarshal(...) != nil || !finiteNumber(...)` guard with the same
     message. This is the one answer-rule restatement this stage confirms
     mechanically; it is bounded in one direction by
     `TestValidateFixtureAnswersNeverAcceptsWhatGenerationRejects`.
  3. helper dispatch is spelled twice (`generateAnswer`'s switch vs
     `validateConfiguredAnswer`'s switch) plus once in `validateDeclaredAnswer`
     and once in `internal/config`'s question-type switch.
  4. the choice-string check uses `isString` statically versus
     `json.Unmarshal` into a `string` in generation; the two agree for every
     value `config.Load` can deliver because the document is decoded before
     validation.
- No *new* rule body was copied in full; the noul payload rules, the allowed
  field sets, the probability value/shape rules, and the sum rule are shared.

## Command outcomes

| Command | Observed |
| --- | --- |
| `gofmt -l internal/compat/jev/v1/fixture.go internal/cli/validate.go internal/compat/jev/v1/fixture_test.go internal/cli/validate_test.go` | no output, exit 0 |
| `go test ./internal/compat/jev/v1 -count=1` | exit 0, 158 passed |
| `go test ./internal/cli -count=1` | exit 0, 204 passed |
| `go test ./... -count=1` | exit 0, 648 passed in 10 packages |
| `go test -race ./... -count=1` | exit 0, 648 passed in 10 packages |
| `go vet ./...` | no output, exit 0 |
| `go run ./cmd/guard arch` | exit 0, `"check": "arch"`, `"findings": []` |
| `go run ./cmd/guard lint` | exit 0, `"check": "lint"`, `"findings": []` |
| `go run ./cmd/guard trace` | exit 0, `"check": "trace"`, `"findings": []` |
| `go run ./cmd/guard fuzz` | exit 0, `"check": "fuzz"`, `"findings": []`, `"targets": 1` |
| `go test ./internal/cli -run 'TestValidateOpensNoListener\|TestValidatePathLinksNoNetworkingPackage\|TestValidateRejectsStaticallyInvalidFixtureData\|TestValidateIsStricterThanTheServeLoadPath\|TestValidateAcceptsValidConfigurationFiles' -count=1 -v` | exit 0; 10 fixture-violation subtests, 3 strictness subtests, 4 valid-file subtests, both no-listener/no-network tests PASS |
| `./scripts/candidate-fingerprint candidate` | exit 0, `80852a3b20015001679e3171fe0271619845cf02203b70f2cb59470b976af21c` |
| `./scripts/verify-candidate FJ-059` | exit 0, `RESULT: PASS`, 20 passed / 0 failed / 0 skipped / 0 not_applicable; report `.agent/reports/FJ-059/report-20261002T210448Z.json` |

Usage-path spot check (exit/stream discipline, §42.1/§42.5): `validate` with no
path, with two paths, and with `--flag` each exited 2 with empty stdout and the
diagnostic plus usage text on stderr.

## Residual risks and evidence limits

1. **Accepted-but-unservable score fixtures.** `validate` accepts a `score`
   answer whose explicit probabilities cannot be satisfied by any request
   (`oos-score-keyset.yaml`: keys `{"0","1"}` with `score: 1`; `serve` returns
   500 for criteria lengths 2 and 3). The hardener recorded this as a
   scope-decision item; this stage independently reproduced it. It is the
   §13.3 key-set/expected-value rule, which the specifier recorded as not
   statically decidable because `N` is request-derived. Honest statement: this
   violation of §13.3 is accepted by `validate` on the configurations tested.
2. **Underflow accepted.** `fp-underflow.yaml` (`noul: 1e-999`) exits 0. The
   value decodes to `0` without error and generation uses the same primitives,
   so `validate` and `serve` agree; only the literal author intent is lost.
   Matches the hardener's residual risk 4; no spec clause is violated on the
   observed evidence.
3. **Duplication surface remains** (three key-set spellings, one verbatim
   `score must be a finite number` guard, two helper dispatches). None was
   exercised as a divergent behaviour in this stage; the risk is future drift,
   not an observed defect. `assert`ing equality between the spellings is not
   mechanically guaranteed by a single test today.
4. **Wildcard type mismatch not statically detectable.** A wildcard stub's
   payload is validated by the helper it declares, so a wildcard fixture that
   can never satisfy a request's question type is accepted by `validate`. This
   follows from the request-decided type rule and is in the specifier's
   out-of-scope list.
5. **Diagnostic wording is not normative.** §42.1 fixes exit codes and §42.5
   the stream split; no clause fixes the failure sentence. This stage asserts
   exit code, stream split, the stub id, and (for sequences) the element index.
   The duplicate-key message for JSON input is YAML-worded
   (`decode YAML configuration: ...`) because `rejectDuplicateJSON` falls
   through to the YAML decoder; behaviour is fail-closed, wording is not
   contractual.
6. **Evidence boundary.** Observations are for the working tree at `26c104f`
   plus the FJ-059 diff, candidate
   `80852a3b20015001679e3171fe0271619845cf02203b70f2cb59470b976af21c`. No
   fixture, script, or temporary file from this stage exists in the repository;
   `/private/tmp/fj059-qa` is a throwaway directory. The full candidate
   verification was run by `./scripts/verify-candidate FJ-059` in this session
   and its report is bound to that revision.
