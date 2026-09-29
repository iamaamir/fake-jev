---
stage: qa
task: FJ-014
inputFingerprint: e4dcbb3aa2b0fc4eb88b39ce09fd65c349311014a1ba7afa29d2a26371c2fba7
outputFingerprint: e4dcbb3aa2b0fc4eb88b39ce09fd65c349311014a1ba7afa29d2a26371c2fba7
taskFingerprint: 1df19a608667d96667c2059da5f0a35005334f81ca89b7ab6c1307c5440d2021
gitHead: 2944771
generatedAt: 2026-09-29T07:06:41Z
author: subagent/worker-FJ-014-qa
---

# QA — FJ-014

## Scope and chain identity

QA exercised the hardened candidate against C-JEV-003/004/005/006/007/019 and
C-GOLD-010/011/016, using §§39.2, 39.3, 39.8, and §44.10/44.11/44.16 as the
behavioral source. The hardener handoff is preserved exactly:

- hardener input: `ab4294d2021797bf919b58dbe3c2a61d068e4f998715ec42e28a82aac96b3db2`
- hardener output: `e4dcbb3aa2b0fc4eb88b39ce09fd65c349311014a1ba7afa29d2a26371c2fba7`
- QA input/current candidate fingerprint: `e4dcbb3aa2b0fc4eb88b39ce09fd65c349311014a1ba7afa29d2a26371c2fba7`
- task fingerprint: `1df19a608667d96667c2059da5f0a35005334f81ca89b7ab6c1307c5440d2021`
- repository HEAD observed: `294477136452e5ca85278a22ee3344ad0cc68c0f`
- coder author: `subagent/worker-FJ-014-coder`; QA author is intentionally distinct.

No production files were modified by QA. This artifact is the only QA file
written.

## Criteria exercised and observations

- **C-JEV-019 / C-GOLD-016:** `TestRecognizeRouteExactness` exercised
  `GET /v1/models`, query-bearing `/v1/models?x=1`, trailing-slash
  `/v1/models/`, wrong-method `POST /v1/models`, `POST /v1/systemone`,
  query-bearing `/v1/systemone?x=1`, trailing-slash `/v1/systemone/`, wrong
  method `GET /v1/systemone`, and an unknown path. Exact pairs were recognized,
  query strings were ignored, and trailing slash/wrong-method cases remained
  unknown routes.
- **C-JEV-004 / C-GOLD-010:** `{` produced one detail at `loc: ["body"]`
  with `msg: "Invalid JSON"`, `type: "json_invalid"`; JSON serialization
  retained the same single-detail envelope.
- **C-JEV-005 / C-GOLD-011:** the §44.11 request missing `state` produced one
  detail at `loc: ["body", "state"]` with `msg: "Field required"` and
  `type: "missing"`.
- **C-JEV-006:** an unsupported question type produced one detail at
  `body/questions/x/type` with `msg: "Unsupported question type"` and
  `type: "literal_error"`.
- **C-JEV-003:** focused order cases observed required-field precedence
  (`state`, then `model`, then `questions`), top-level object enforcement, and
  lexicographic question-name selection (`a` before `z`). Each returned only
  the first detail.
- **C-JEV-007:** focused criteria cases observed accepted `noul` optional/null
  criteria, non-empty `choice` object criteria, non-empty `score` array criteria
  with string/object/array elements, and rejection locations for empty choice,
  empty score, null score element, and invalid instructions. Unknown top-level
  and question fields remained accepted.

## Commands and results

- `go test ./internal/compat/jev/v1 -count=1` — passed (`ok`, package tests).
- `go test ./internal/compat/jev/v1 -run 'TestRecognizeRouteExactness|TestValidationGoldenErrors|TestValidationOrderAndFrozenSchema|TestSupportedQuestionsAndCriteriaRules' -count=1 -v` — passed; all route, golden envelope, ordering, and criteria subtests passed.
- `go test ./...` — passed.
- `go test -race ./...` — passed.
- `go vet ./...` — passed (no output).
- `./scripts/selftest` — passed; 75 repository guard scenarios.
- `./scripts/candidate-fingerprint candidate .` — returned
  `e4dcbb3aa2b0fc4eb88b39ce09fd65c349311014a1ba7afa29d2a26371c2fba7`.
- `./scripts/candidate-fingerprint task .agent/work/FJ-014/state.json` —
  returned task fingerprint
  `1df19a608667d96667c2059da5f0a35005334f81ca89b7ab6c1307c5440d2021`.
- `./scripts/verify-candidate FJ-014` — initial invocation exercised
  repository checks and product Go checks and reported the expected missing QA
  artifact (`42 passed, 1 failed, 3 skipped, 0 not_applicable`) before this
  artifact existed; the post-artifact invocation completed with
  `48 passed, 0 failed, 3 skipped, 0 not_applicable` and report
  `.agent/reports/FJ-014/report-20260929T070835Z.json`.

## Residual risks and gaps

- The repository has no HTTP host/public-surface test harness yet; route checks
  exercise `RecognizeRoute` directly, and envelope checks exercise the
  compatibility validator and JSON marshaling directly. HTTP status/header
  integration (including §39.9 `Content-Type`) is therefore not observed here.
- Duplicate JSON object-name behavior remains delegated to `encoding/json`
  (last value wins); §§39.2–39.3 do not define duplicate-name semantics.
- Body-size enforcement and interaction journaling are host responsibilities
  outside the four allowed compatibility files and were not exercised by this
  QA slice.
- Verifier tooling reports the public-surface/system test harness as skipped by
  the repository bootstrap exemption; no additional mutation/hardening tool is
  available.
