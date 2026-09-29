---
stage: qa
task: FJ-015
inputFingerprint: 66f35765afb2948730b0bac0c54bc1b08852900e356f16423d15d9ad2690710e
outputFingerprint: 66f35765afb2948730b0bac0c54bc1b08852900e356f16423d15d9ad2690710e
taskFingerprint: c4409d7a1fbba4b732c5b1d9cc70c922f2a2f717e6b47adce19907a99d646528
gitHead: 45156682a651b5d6278239e21266d0bd36efce30
generatedAt: 2026-09-29T07:56:35Z
author: subagent/worker-FJ-015-qa
---

# QA — FJ-015

## Scope and chain identity

QA independently exercised the current `jev/v1` fixture-helper candidate for
C-JEV-008/011/012/013/014/015/016 and C-GOLD-002/003/004 against §13 and
§13.5 of the authoritative specification. The hardener handoff is preserved
exactly:

- hardener input: `66f35765afb2948730b0bac0c54bc1b08852900e356f16423d15d9ad2690710e`
- hardener output: `66f35765afb2948730b0bac0c54bc1b08852900e356f16423d15d9ad2690710e`
- QA input/current candidate fingerprint: `66f35765afb2948730b0bac0c54bc1b08852900e356f16423d15d9ad2690710e`
- task fingerprint: `c4409d7a1fbba4b732c5b1d9cc70c922f2a2f717e6b47adce19907a99d646528`
- repository HEAD observed: `45156682a651b5d6278239e21266d0bd36efce30`
- coder author: `subagent/worker-FJ-015-coder`; QA author is distinct.

No production files were modified by QA. This artifact is the only QA file
written by this stage; no tests were added or updated.

## Criteria exercised and observations

- **C-GOLD-002 / C-JEV-011:** The noul golden vector emitted exactly
  `{"type":"noul","noul":0.9}`. Numeric `0.94`, `true`, and `false` produced
  `0.94`, `1.0`, and `0.0`; negative and greater-than-one values and a quoted
  numeric string were rejected. Inclusive confidence/probability boundaries
  `0` and `1` were accepted.
- **C-GOLD-003 / C-JEV-012:** The choice golden vector emitted the complete
  frontend/backend/infra map with backend `1.0`, all other entries `0.0`, and
  omitted confidence `1.0`. The selected criterion was required to exist and
  omitted probabilities covered every request criterion key.
- **C-JEV-013:** Explicit choice probabilities exercised selected maximum,
  tied maximum, sum within tolerance, unknown key, missing key, bad sum,
  out-of-range probability, non-maximum selection, unknown selected choice,
  and out-of-range confidence. Valid explicit confidence omission became the
  selected probability; invalid maps were rejected without normalization.
- **C-GOLD-004 / C-JEV-014:** The fractional score golden vector emitted
  score `1.5`, probabilities `{"0":0,"1":0.5,"2":0.5}`, confidence `1.0`,
  and the legend derived exactly from the three request criteria. Integer
  one-hot and fractional `0.25` interpolation were also exercised.
- **C-JEV-015:** Explicit score probabilities exercised exact level keys,
  expected-value-valid `1.5`, unknown and missing keys, bad sum,
  out-of-range values, and expected-value mismatch. Supplied legends were
  rejected; generated legends came only from request criteria.
- **C-JEV-008 / C-JEV-016:** Mixed noul/choice/score requests exercised
  missing answer, extra answer, mixed partial answer, noul/choice helper
  mismatch, and choice/score helper mismatch. Every failure surfaced as
  `InvalidStubResponseError`; no partial or implicit answer was generated.
- Malformed answer documents (`null`, array, null helper, and null noul),
  malformed helper values, unknown helper fields, malformed probability maps,
  and invalid confidence/probability values were covered by the focused
  fixture tests and rejected.

## Commands and results

- `go test ./internal/compat/jev/v1 -run 'TestGenerateAnswersGoldenVectors|TestGenerateAnswersNoulFormsAndValidation|TestGenerateAnswersChoiceProbabilityRules|TestGenerateAnswersScoreRules|TestGenerateAnswersExactCoverageAndTypes|TestGenerateAnswersRejectsMalformedAnswerDocument|TestFiniteProbabilityBoundaries' -count=1 -v` — passed; all golden, coverage, mismatch, malformed, interpolation, legend, and probability subtests passed.
- `go test ./...` — passed.
- `go test -race ./...` — passed.
- `go vet ./...` — passed.
- `./scripts/selftest` — passed; 75 repository guard scenarios.
- `./scripts/candidate-fingerprint candidate .` — returned
  `66f35765afb2948730b0bac0c54bc1b08852900e356f16423d15d9ad2690710e`.
- `./scripts/candidate-fingerprint task .agent/work/FJ-015/state.json` —
  returned task fingerprint
  `c4409d7a1fbba4b732c5b1d9cc70c922f2a2f717e6b47adce19907a99d646528`.
- `./scripts/verify-candidate FJ-015` — run after this artifact was written;
  repository, product Go, architecture, lint, trace, fuzz, race, evidence
  chain, and task checks were exercised. The resulting report is recorded in
  `.agent/reports/FJ-015/`.

## Residual risks and gaps

- The fixture package tests exercise `GenerateAnswers` at the validated-request
  boundary, not a live HTTP server. HTTP status/header integration, request
  decoding, route handling, and journal/verification effects remain outside
  this slice and the allowed fixture files.
- Duplicate JSON object-name behavior remains delegated to `encoding/json`;
  §13 does not define duplicate-name semantics.
- The repository's mutation/hardening analyzer is unavailable under the
  bootstrap exemption; deterministic unit tests, race testing, vet, selftest,
  and task verification are the available executable evidence.
- Probability tolerance cases exercise an accepted within-tolerance sum and
  invalid sums, but the focused existing tests do not independently pin both
  exact `1e-6` acceptance and just-over-tolerance rejection.
