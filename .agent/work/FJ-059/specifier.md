---
stage: specifier
task: FJ-059
inputFingerprint: b108a8dd28dfd3b53b39add62b2ccc81d1fa0942c1b1cc27400dd98e805ddf55
outputFingerprint: b108a8dd28dfd3b53b39add62b2ccc81d1fa0942c1b1cc27400dd98e805ddf55
taskFingerprint: 879b10dee885044f7a81eed718e8b111e040775d016d6fa1c2bb8575fdbcd179
gitHead: 26c104f
generatedAt: 2026-10-02T20:15:37Z
author: worker/FJ-059-specifier
---

# Specifier — FJ-059 (static jev/v1 fixture validation for `fake-jev validate`)

Objective restated as observables: `fake-jev validate <path>` reports every
statically provable jev/v1 fixture-data inconsistency that the configuration
file alone determines, namely (G2) the `answers` object key set equalling the
stub's `when.questions` key set, including inside each `then.sequence`
element, and (G1) the per-answer payload validity for `noul`, `choice`, and
`score` computed from the fixture alone, and the answer helper type agreeing
with the configured question type. The command keeps the already-shipped
`internal/config` load surface and adds the compat-layer checks only on the
`validate` path. Serve-time generation and the `config.Load` strictness that
`serve` uses stay as they are. No verdict is expressed here; results come only
from `./scripts/verify-candidate FJ-059`.

## Traces

- C-CLI-005
- C-CLI-001

## Baseline at 26c104f — what already exists and what the delta is

- `internal/cli/validate.go` (`runValidate`) is a thin wrapper: it parses one
  path, calls `config.LoadFile`, prints
  `fake-jev: <path>: configuration is valid` on stdout, and returns
  `exitOK`; a load error prints `fake-jev: <path>: <err>` on stderr and returns
  `exitFailure` (2). It validates nothing itself.
- `internal/config` (provider-neutral, `validateDocument`/`validateValues`)
  already enforces the outer contract: top-level schema, unknown fields,
  duplicate stub ids, `when.questions` name/type shape (`noul|choice|score`),
  `expect` rules, exactly-one response form (`answers`/`sequence`/`raw`), and
  `requireJSONObject(then.Answers)`. It checks that `then.answers` **is an
  object** but never inspects its members' payloads, and never compares the
  `answers` keys to `when.questions`.
- `internal/compat/jev/v1/fixture.go` already enforces the inner payload rules
  at request time inside `GenerateAnswers` → `generateAnswer` →
  `generateNoulAnswer`/`generateChoiceAnswer`/`generateScoreAnswer`, via
  `validateHelperFields`, `probabilitiesField`, `confidenceField`,
  `validateProbabilityMap`, `finiteProbability`, `finiteNumber`, `jsonNumber`.
  Those functions take a validated request `Question` (with request-supplied
  `criteria`), so they run only when a request arrives; nothing calls them from
  `validate`.
- Therefore the two gaps are exactly: (G1) `then.answers` members are
  unvalidated at load time, and (G2) the `answers`↔`when.questions` key-set
  relation is unchecked — both of which are functions of the configuration
  file alone.
- `internal/cli/validate_test.go` already pins: exit 0 with the success line
  on stdout for a valid YAML/JSON file; exit 2 with a stderr-only diagnostic on
  a violation; no listener opened (`TestValidateOpensNoListener`); and that the
  import closure of `validate.go` links no `net`/`net/...` package
  (`TestValidatePathLinksNoNetworkingPackage`). `.agent/guards.json` forbids
  `net`, `net/http`, `os/exec`, templates, and `plugin` in
  `internal/compat/jev/v1`; `verify-candidate` runs `cmd/guard arch`.
- Design constraint (PM, recorded in FJ-059 `state.json`): the static
  validator belongs in `internal/compat/jev/v1` and is invoked from the CLI
  composition root; `internal/config` must not import the compat layer, and
  `config.Load` strictness must not change for `serve`.

## G2 — `answers` key set equals `when.questions` key set

### A1 — a top-level `then.answers` key-set mismatch is a `validate` violation

- Observable: for an enabled `jev/v1` stub whose `when.questions` is present,
  the key set of `then.answers` must equal the key set of `when.questions`.
  When it does, the command reports validity; when it does not, the command
  reports the configuration-invalid outcome and identifies the offending stub
  on stderr.
- Spec clause: §15.2 ("MUST validate all enabled profile-specific fixture
  data"; "MUST perform all statically possible consistency checks"), §38.6
  (`when.questions` is a non-empty map), §38.7 (`then.answers`), §13.4/§13.5
  (an `answers` response covers the exact request question-name set), §40.5
  (the request question-name set equals the configured set exactly).
- Deterministic commands (binary built once):
  - `go build -o /tmp/fj059/fake-jev ./cmd/fake-jev`
  - valid file `ok-keys.yaml` (`when.questions: {urgent: noul, route: choice}`
    and `then.answers: {urgent: …, route: …}`):
    `/tmp/fj059/fake-jev validate ok-keys.yaml; echo "exit=$?"`
  - mismatch file `g2-keys.yaml`:
    ```yaml
    schemaVersion: 1
    mode: strict
    stubs:
      - id: mismatch
        profile: jev/v1
        when:
          questions:
            urgent: noul
        then:
          answers:
            route:
              choice: backend
    ```
    `/tmp/fj059/fake-jev validate g2-keys.yaml; echo "exit=$?"`
- Exact expected output:
  - valid file: stdout `fake-jev: ok-keys.yaml: configuration is valid` then
    `exit=0`; stderr empty.
  - mismatch file: `exit=2`; stdout empty (0 bytes); stderr non-empty and
    containing the stub id `mismatch` (the diagnostic wording is not fixed by
    the specification; the stub id is the stable identifying token).
- Boundary: when `when.questions` is absent (wildcard), the request name set
  is unknown, so A1 does not apply (see "Not statically decidable").

### A2 — the same key-set relation holds for every `then.sequence[i].answers`

- Observable: each element of `then.sequence` whose response form is
  `answers` is checked against the same `when.questions` key set as the
  top-level form; an element that deviates makes the whole configuration
  invalid, and the diagnostic identifies the stub (and, at the implementation's
  option, the element index).
- Spec clause: §15.2 ("all enabled profile-specific fixture data"), §38.7
  ("For a `sequence`, each element has the same response shape"), §13.4/§13.5.
- Deterministic command: file `g2-sequence.yaml` with
  `when.questions: {urgent: noul}` and
  ```yaml
  then:
    sequence:
      - answers: {urgent: {noul: true}}
      - answers: {route: {choice: backend}}
  ```
  `/tmp/fj059/fake-jev validate g2-sequence.yaml; echo "exit=$?"`
- Exact expected output: `exit=2`; stdout empty; stderr non-empty containing
  `g2-sequence` (stub id). A file whose every element matches reports
  `exit=0` with the `configuration is valid` line.

### A3 — the check runs for every stub, not only the first

- Observable: a configuration whose last stub carries the violation is still
  reported invalid, so no stub is skipped; equivalently, adding a
  statically-invalid stub to an otherwise valid file changes the outcome.
- Spec clause: §15.2 ("all enabled profile-specific fixture data").
- Deterministic command: file `g2-last.yaml` with stub `first` (valid) and
  stub `last` (answers key set `{route}` against `when.questions: {urgent noul}`),
  `/tmp/fj059/fake-jev validate g2-last.yaml; echo "exit=$?"`
- Exact expected output: `exit=2`; stdout empty; stderr non-empty containing
  `last`.

## G1 — per-type payload validity of each `then.answers` member

Rules reused verbatim from `internal/compat/jev/v1/fixture.go`
(`validateHelperFields`, `probabilitiesField`, `confidenceField`,
`validateProbabilityMap`, `finiteProbability`, `jsonNumber`); no second copy of
these rules. For all of B1–B5 the cross-check of A1 is also assumed present, so
each answer name can be attributed to `when.questions[name]`.

### B1 — answer helper type equals the configured question type

- Observable: for `answers[name]`, the single recognized helper key
  (`noul`/`choice`/`score`) must equal `when.questions[name]`; a `noul` payload
  under a configured `choice` question (or any other combination) makes the
  configuration invalid. This mirrors `generateAnswer`'s switch on the request
  question type and §13.4's "fixture helper type MUST match each request
  question type".
- Spec clause: §13.1 (noul helper on a non-noul question is a failure), §13.4,
  §38.6 (`when.questions` names a type), §40.5 (configured type equals request
  type), §15.2.
- Deterministic command: `g1-type.yaml` with `when.questions: {urgent: noul}`
  and `then.answers: {urgent: {choice: backend}}`;
  `/tmp/fj059/fake-jev validate g1-type.yaml; echo "exit=$?"`
- Exact expected output: `exit=2`; stdout empty; stderr non-empty containing
  `g1-type` (stub id).

### B2 — `noul` payload value validity

- Observable: the `noul` payload is exactly one `noul` field whose value is a
  boolean or a finite JSON number in `[0,1]`; `noul: 1.5`, `noul: -0.1`,
  `noul: "0.5"`, or an extra sibling field makes the configuration invalid.
  `noul: 0.94`, `noul: 0`, `noul: 1`, `noul: true`, `noul: false` remain valid.
  The value rule is `jsonNumber` + `finiteProbability`, the same primitives
  `generateNoulAnswer` uses.
- Spec clause: §13 preamble (finite numbers in `[0,1]`), §13.1 (boolean
  convenience form, emitted answer), §15.2.
- Deterministic commands:
  `/tmp/fj059/fake-jev validate g1-noul-range.yaml; echo "exit=$?"`
  (`then.answers: {urgent: {noul: 1.5}}`, `when.questions: {urgent: noul}`) and
  `/tmp/fj059/fake-jev validate g1-noul-extra.yaml; echo "exit=$?"`
  (`{noul: 0.5, confidence: 0.5}`).
- Exact expected output: `exit=2`; stdout empty; stderr non-empty for each;
  and `exit=0` with the `configuration is valid` line for a file using
  `noul: 0.94`, `noul: true`, and `noul: false`.

### B3 — `choice` payload validity (configuration-only part)

- Observable: the payload is an object whose fields are a subset of
  `{choice, confidence, probabilities}` and contains `choice` (a JSON string);
  `confidence`, when present, is a finite number in `[0,1]`; `probabilities`,
  when present, is an object whose every value is a finite number in `[0,1]`
  and whose values sum to `1.0` within `1e-6`. A non-string `choice`, an unknown
  sibling field, `confidence: 1.5`, a probability of `1.1`/`-0.1`, or
  probabilities not summing to 1 (e.g. `{a: 0.2, b: 0.2, c: 0.2}`) makes the
  configuration invalid. The value/range/sum checks are exactly
  `validateHelperFields`, `probabilitiesField`, `confidenceField`,
  `validateProbabilityMap` (the map's own keys as the expected key list).
- Spec clause: §13 preamble (finite `[0,1]`, sum within `1e-6`), §13.2 (choice
  helper shape, explicit-probability value/range/sum rules, `confidence`
  default), §15.2.
- Deterministic commands:
  `/tmp/fj059/fake-jev validate g1-choice-sum.yaml; echo "exit=$?"`
  (`{choice: backend, probabilities: {a: 0.2, b: 0.2, c: 0.2}}`),
  `/tmp/fj059/fake-jev validate g1-choice-range.yaml; echo "exit=$?"`
  (`{choice: backend, probabilities: {a: 1.1, b: -0.1}}`),
  `/tmp/fj059/fake-jev validate g1-choice-conf.yaml; echo "exit=$?"`
  (`{choice: backend, confidence: 1.5}`).
- Exact expected output: `exit=2`; stdout empty; stderr non-empty for each;
  and `exit=0` with the `configuration is valid` line for
  `{choice: backend, confidence: 0.91, probabilities: {frontend: 0.06, backend: 0.91, infra: 0.03}}`.

### B4 — `score` payload validity (configuration-only part)

- Observable: the payload is an object whose fields are a subset of
  `{score, confidence, probabilities}` (in particular `legend` is rejected —
  §13.3 forbids a fixture-provided legend) and contains a finite JSON number
  `score`; `confidence`, when present, is a finite number in `[0,1]`;
  `probabilities`, when present, is an object whose every value is a finite
  number in `[0,1]` and whose values sum to `1.0` within `1e-6`.
  `{score: 1.5, legend: {...}}`, `{score: 1, probabilities: {0: 0.5, 1: 0.7}}`,
  and `{score: 1, confidence: 1.1}` make the configuration invalid; the same
  values used by `generateScoreAnswer`'s valid cases remain valid.
- Spec clause: §13 preamble, §13.3 ("Fixtures MUST NOT provide a separate
  `legend`"; explicit score probabilities "MUST be in `[0,1]`"; "MUST sum to
  `1.0` within `1e-6`"; `confidence` default), §15.2.
- Deterministic commands:
  `/tmp/fj059/fake-jev validate g1-score-legend.yaml; echo "exit=$?"`
  (`{score: 1, legend: {"0": "x"}}`),
  `/tmp/fj059/fake-jev validate g1-score-prob.yaml; echo "exit=$?"`
  (`{score: 1.5, probabilities: {"0": 0.0, "1": 0.5, "2": 0.5}}` sums to 1 and
  is valid; the invalid variant uses `{"0": 0.5, "1": 0.7}` summing to 1.2).
- Exact expected output: `exit=2`; stdout empty; stderr non-empty for the
  legend and non-summing variants; `exit=0` with the `configuration is valid`
  line for `{score: 1.5, confidence: 0.8, probabilities: {"0": 0.0, "1": 0.5, "2": 0.5}}`.

### B5 — unresolvable `answers` form stays a `config` error, not a compat error

- Observable: `then.answers` that is not a JSON object (e.g. `null`, `[]`, a
  string) is already rejected by `config.Load` before the compat validator is
  reached; the CLI still reports exit 2 with a stderr diagnostic and no
  stdout. A member value that is not an object (e.g. `{urgent: null}` or
  `{urgent: 3}`) is in the compat subset and is rejected there.
- Spec clause: §38.7, §15.2, §42.1.
- Deterministic commands:
  `/tmp/fj059/fake-jev validate g1-null-member.yaml; echo "exit=$?"`
  (`then.answers: {urgent: null}`).
- Exact expected output: `exit=2`; stdout empty; stderr non-empty.

## Command-level criteria

### C1 — exit 0 on a fully valid file, exit 2 with a stderr-only diagnostic on a violation

- Observable: `fake-jev validate <path>` returns `exitOK` (0) and writes the
  success line on stdout and nothing on stderr when every static check holds;
  it returns `exitFailure` (2), writes nothing on stdout and a non-empty
  diagnostic on stderr when any static check fails. Usage failures remain 2
  with the usage text on stderr (`validate` takes exactly one path).
- Spec clause: §15.2, §42.1 (0 success; 2 usage/config/startup/control/internal),
  §42.5 (stream split).
- Deterministic command: run the valid and invalid files from A1–B5 and
  inspect `exit`, stdout, stderr:
  `for f in ok-keys.yaml g2-keys.yaml g1-noul-range.yaml; do /tmp/fj059/fake-jev validate "$f"; echo "$f exit=$?"; done`
- Exact expected output: `ok-keys.yaml exit=0` (stdout is
  `fake-jev: ok-keys.yaml: configuration is valid`, stderr empty);
  `g2-keys.yaml exit=2` and `g1-noul-range.yaml exit=2` (stdout empty, stderr
  non-empty).

### C2 — the static validator lives in the compat layer and is called from the CLI composition root

- Observable: the new validator function is defined in
  `internal/compat/jev/v1/fixture.go` (package `v1`) and `internal/cli/validate.go`
  imports `fake-jev/internal/compat/jev/v1` and calls it once per stub; the
  compat package still contains no `net`, `net/http`, `os/exec`, template, or
  `plugin` import (`.agent/guards.json`).
- Spec clause: §15.2 ("all static checks"), C-ARCH-001/§4.2 and §30.1 (the
  compat layer, not the provider-neutral engine/config, owns v1 knowledge);
  PM design constraint recorded in FJ-059 `state.json`.
- Deterministic commands:
  `grep -n "compat/jev/v1" internal/cli/validate.go`,
  `grep -n "func .*Validate\|func .*Answers" internal/compat/jev/v1/fixture.go`,
  `go run ./cmd/guard arch; echo "exit=$?"`,
  `go test ./internal/cli/ -run TestValidatePathLinksNoNetworkingPackage -count=1`.
- Exact expected output: the first two greps print at least one matching line
  (the import and the new validator symbol); `go run ./cmd/guard arch` prints
  a JSON report with `"check": "arch"` and no findings and `exit=0`;
  the targeted test prints `ok` and exits 0 (the compat import closure of
  `validate.go` adds no `net`/`net/...` import).

### C3 — serve load path unchanged

- Observable: `internal/config` (its `Load`/`LoadFile`/`validateDocument`
  strictness) and the serve-time generation (`GenerateAnswers` and its
  helpers' rule bodies in `fixture.go`) are not modified by this item; a
  configuration whose only defect is a statically-decidable compat violation
  is still accepted by `config.LoadFile` and a server still starts from it,
  while `validate` rejects it. `serve` behavior is unchanged.
- Spec clause: §15.1 (`serve`), §15.2, PM constraint ("Do not change
  `config.Load` strictness or anything `serve` does"); §42.1.
- Deterministic commands:
  `git diff --name-only 26c104f -- internal/config internal/compat/jev/v1/response.go`
  (expect no output), plus `go test ./internal/config/... ./internal/cli/... -count=1`
  (existing suite, including `serve_test.go` and `TestValidateOpensNoListener`).
- Exact expected output: the `git diff` prints nothing; `go test` prints
  `ok` for the listed packages and exits 0.

### C4 — no duplicate or divergent restatement of the answer rules

- Observable: the static validator calls the existing
  `validateHelperFields`, `probabilitiesField`, `confidenceField`,
  `validateProbabilityMap`, `finiteProbability`, `jsonNumber`; the bodies of
  `GenerateAnswers`, `generateNoulAnswer`, `generateChoiceAnswer`,
  `generateScoreAnswer`, `validateHelperFields`, `probabilitiesField`,
  `confidenceField`, `validateProbabilityMap`, `finiteProbability`,
  `finiteNumber`, and `jsonNumber` are not rewritten (only additions of new
  static-validation code that calls them are present); the new tests mirror
  the existing generation-test cases so the two paths cannot silently diverge
  on the shared subset.
- Spec clause: §15.2 ("all statically possible consistency checks" must agree
  with runtime rules); AGENTS.md "Avoid duplicating non-trivial business
  logic"; §13 (the single set of helper rules).
- Deterministic commands:
  `git diff 26c104f -- internal/compat/jev/v1/fixture.go` (must show the
  existing generation/helper functions unchanged; the diff read as
  "additions only for the new static path"),
  `grep -nE "validateHelperFields|probabilitiesField|confidenceField|validateProbabilityMap|finiteProbability|jsonNumber" internal/compat/jev/v1/fixture.go`
  (the new static-validator function appears among the call sites),
  `go test ./internal/compat/jev/v1/ -count=1`.
- Exact expected output: the diff contains no changed/removed line inside the
  named generation/helper functions; the grep shows the named helpers; the
  package test prints `ok` and exits 0.

### C5 — the stage's own verifier is green

- Observable: the repository's central verifier reports success for the
  candidate revision of FJ-059.
- Spec clause: §15.2 via C-CLI-005; AGENTS.md completion rule.
- Deterministic command: `./scripts/verify-candidate FJ-059`
- Exact expected output: exit `0`, with the `go test ./...`,
  `go vet ./...`, `go build ./cmd/fake-jev`, and `guard arch` rows present in
  `.agent/reports/FJ-059/`.

## Not statically decidable — explicitly out of scope

These rules are stated relative to the incoming client request's `criteria`
and cannot be decided from the configuration file; they stay in
`GenerateAnswers`'s request-time path and are **not** part of this item's
acceptance.

1. **Choice explicit-probability key set = request criteria key set**
   (§13.2 "keys MUST exactly equal the request criteria key set"). Criteria
   come from the request, not the config.
2. **The selected `choice` exists as a request criterion** (§13.2). Same
   reason; also covered by the PM note in FJ-059 `state.json`.
3. **The selected `choice` probability is the maximum** (§13.2). The maximum
   is taken over request criteria. (When explicit probabilities are present
   and the choice key is in the map, a configuration-only shadow is
   computable, but the rule as specified is request-relative; no shared helper
   for it exists today, so including it would mean new, restated logic — see
   "No duplicate or divergent restatement". It is therefore not required.)
4. **`confidence` defaults to the selected choice probability** when explicit
   probabilities are present (§13.2). The default value is request-derived.
5. **Score key set = `"0"` … `"N-1"` where `N` is the request criteria
   length** (§13.3 "MUST use exactly the keys `"0"` through `"N-1"`"). `N` is
   request-derived, so the fixture's keys cannot be checked statically.
6. **Requested score within `[0, N-1]`** (§13.3). `N` is request-derived.
7. **Score probability-weighted expected value equals `score` within `1e-6`**
   (§13.3). Depends on the full `"0"`…`"N-1"` key set and `N`.
8. **Request-time exact-cover rule for partial/mixed answers** (§13.4/§13.5):
   when `when.questions` is omitted (wildcard), the request question-name set
   is unknown, so A1's key-set equality and B1's type equality are not applied.
   The per-type payload checks (B2–B5) still apply to every `answers` member
   independently of the request.
9. **Type coercion and any `raw` payload semantics** (§12.6, §38.7): `raw`
   responses are intentionally not validated against `jev/v1` answer schemas,
   and no answer helper applies to them.

## Spec-silent observations

- **Cited section mismatch.** FJ-059 `state.json` `specReferences` lists
  "§40.2-40.6 fixture answer rules". §40 is "Normative matching, ordering,
  sequences, and state" (§40.2 exact matching order, §40.3 JSON state equality,
  §40.4 model matching, §40.5 question matching, §40.6 invocation count); the
  fixture answer/helper rules are §13.1–13.6, with §12.6 for `raw`, §38.6/§38.7
  for the `when`/`then` object shapes, and §12.7 for `expect`. §40.5 supplies
  the request-time exact-set rule that makes A1/A2/B1 meaningful. The criteria
  above trace to the actual clauses; the work item's reference list is recorded
  here as a documentation discrepancy only, with no new requirement inferred.
- **No normative diagnostic text.** §42.1 fixes exit codes; §42.5 fixes the
  stream split. No clause fixes the wording of a `validate` failure message.
  The criteria therefore assert exit code, the stream split, and the presence
  of the stub id in the stderr diagnostic as the stable token, not a fixed
  sentence. The coder records the exact text in `coder.md`.
- **`when.questions` is the only static source of question types.** §40.5 pins
  the request question-name set and types to the configured ones, which is what
  makes B1 decidable. If a future profile allowed `questions` to be omitted as
  a wildcard while still requiring exact answer coverage, B1 would become
  request-dependent; the specification does not make that requirement.
- **Number-source distinction.** `config.Load`'s JSON decode and the
  compat-layer raw fields keep numbers as `json.Number`/`RawMessage`; §40.3
  warns against routing integer equality through binary floating point. The
  static checks compare probability/confidence values as `float64` exactly as
  `GenerateAnswers` does today, so no new precision policy is introduced; the
  existing §40.3 rule is about `when.state` equality, not answer payloads.
- **Empty `when.questions`.** §38.6 requires a supplied `questions` to be a
  non-empty map and `internal/config` already rejects empty; the static
  validator never sees an empty-vs-non-empty boundary decision of its own.
- **`answers` object with zero members.** `config.Load` treats
  `len(then.Answers) > 0` as the "answers form present" test, so a `then` with
  no form is already rejected. No new rule is needed for the empty case.
- **Sequence with `raw` elements.** §38.7 allows each sequence element to take
  the same response shape, including `raw`; the static answer checks apply only
  to `answers`-form elements and the raw schema is out of scope (§12.6).
- **Invocation.** The specification does not say whether the CLI or the profile
  should drive the per-stub loop; PM fixed the composition-root invocation as
  the design constraint, so C2 asserts only that the validator is in the compat
  layer and is called from `internal/cli/validate.go`.

## Non-goals (respected, not re-litigated)

- No change to runtime request validation (`ValidateRequest`) or to
  `GenerateAnswers` behavior; no change to serve-time response generation.
- No server startup and no network access on the `validate` path; the
  `TestValidateOpensNoListener` and `TestValidatePathLinksNoNetworkingPackage`
  properties stay true.
- No new wire-visible behavior, route, config key, or flag; `validate` still
  takes exactly one configuration path.
- No change to `internal/config` load strictness and no change to `serve`.
- No verification row added: `scripts/verify-candidate` is outside
  `allowedFiles`, so verification is the command- and inspection-based evidence
  above plus the central verifier's existing rows.

## Out of scope for this item

- All request-criteria-dependent rules listed under "Not statically
  decidable".
- Validating `then.raw` / `sequence[i].raw` bodies, statuses, and headers
  beyond what `internal/config` already does (§12.6/§38.7 make raw
  intentionally unvalidated against `jev/v1`).
- Any cross-stub consistency (duplicate answers, shared state) — the
  specification defines none.
- Any change to the `expect` rules or to the outer schema checks owned by
  FJ-030.
