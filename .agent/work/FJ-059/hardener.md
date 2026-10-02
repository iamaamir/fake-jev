---
stage: hardener
task: FJ-059
inputFingerprint: bb445e590122427e20db13937225cf7a5b4eda3d9f223d907b07463fbc6e7248
outputFingerprint: 80852a3b20015001679e3171fe0271619845cf02203b70f2cb59470b976af21c
taskFingerprint: 879b10dee885044f7a81eed718e8b111e040775d016d6fa1c2bb8575fdbcd179
gitHead: 26c104f
generatedAt: 2026-10-02T20:52:24Z
author: worker/FJ-059-hardener
---

# Hardener — FJ-059 (static jev/v1 fixture validation for `fake-jev validate`)

**Final candidate fingerprint: `80852a3b20015001679e3171fe0271619845cf02203b70f2cb59470b976af21c`**

Input (cleaner) revision: candidate `bb445e590122427e20db13937225cf7a5b4eda3d9f223d907b07463fbc6e7248`.
This stage changed `internal/compat/jev/v1/fixture.go` (production),
`internal/compat/jev/v1/fixture_test.go`, and `internal/cli/validate_test.go`
(tests). `internal/cli/validate.go` was not touched by this stage.

## Hardening actions, with the clauses they serve

### 1. Fail-open closed: a wildcard stub whose `answers` object has no members

Defect (confirmed before the change). `when: {}` (no `questions`, the §38.6
wildcard stub) together with `then: {answers: {}}` was reported by `validate` as
`configuration is valid`, exit 0. The stub can never produce a valid response:
every request carries at least one question (§39.2 "`questions` MUST exist, be
an object, and contain at least one property"; §12), and an `answers` document
with no members provides an answer for none of them (§13.4 "MUST provide an
answer for every requested question"; §13.5). §15.2 requires `validate` to
"MUST perform all statically possible consistency checks", and this one is
decidable from the configuration alone.

Evidence before the fix (same file, same request):

```text
$ /tmp/fj059h/fake-jev validate wildcard-empty.yaml
fake-jev: wildcard-empty.yaml: configuration is valid      # exit=0
$ curl -s -X POST "$url/v1/systemone" -d '{"state":{},"model":"jev-latest","questions":{"urgent":{"type":"noul"}}}'
http=500 {"error":"fake_jev_invalid_stub_response","message":"Configured stub cannot produce a valid response for this request.","stubId":"wildcard-empty"}
```

Change (`fixture.go:286-294`): inside the existing wildcard branch, before
dispatching declared helpers, an `answers` object with zero members returns
`answers must not be empty`.

Evidence after the fix:

```text
$ /tmp/fj059h/fake-jev validate wildcard-empty.yaml
fake-jev: wildcard-empty.yaml: stub "wildcard-empty": answers must not be empty   # exit=2, stdout empty
```

Boundary kept deliberately: the non-wildcard case is still rejected by the
pre-existing length rule (`answers must exactly cover the configured questions`,
`fixture.go:303`), so a configured stub with `answers: {}` continues to name the
coverage requirement it misses, and no existing message or test row changed.
`len(answers) == 0` (absent answers document = `raw`/`sequence` form, §38.7)
still returns early, so no other response form became rejectable. Pinned by
`TestValidateFixtureAnswers` row `wildcard answers key set empty`,
`TestValidateRejectsStaticallyInvalidFixtureData` row
`wildcard answer document is empty`, and the strictness row
`wildcard answers document empty` (the loader still accepts the file, so
`config.Load` strictness is provably unchanged).

### 2. Non-deterministic diagnostic for an answer with several unknown fields

Defect (confirmed before the change). `validateHelperFields` iterated its field
map in map order, so a payload with more than one unknown sibling field
produced a different stderr line for byte-identical input. The diagnostic is
the only thing `validate` emits on failure (§42.5 stream split), and the
repository rule is one spelling and no map-iteration order in outputs.

Evidence before the fix — 40 runs of one fixture whose answer carried
`choice` plus seven unknown siblings:

```text
Counter({'"z1"': 10, '"z6"': 8, '"z2"': 8, '"z3"': 5, '"z7"': 4, '"z5"': 3, '"z4"': 2})
```

Change (`fixture.go:421-424`): the loop iterates `sortedKeys(fields)` instead of
the map, so the reported unknown field is the lexicographically first. The rule
body is unchanged — same rejection set, same "first rejection wins" semantics,
same messages — only the selection among several equally-unacceptable fields
became deterministic, matching the `sortedKeys` idiom the rest of the file
already uses. The serve path discards this internal reason
(`internal/host/http/server.go:333` sends the fixed
`Configured stub cannot produce a valid response for this request.`), so nothing
wire-visible changed.

Evidence after the fix (20 runs, same fixture): `20 "z1"`. Pinned by
`TestValidateFixtureAnswersReportsUnknownFieldDeterministically` (32 repeated
calls, message asserted) and by the determinism assertion inside the new fuzz
target.

### 3. Panic-freedom proof for arbitrary malformed input

New `FuzzValidateFixtureAnswers` (`fixture_test.go:378`) fuzzes
`(answers []byte, name, kind string)` across three question maps — `nil`
(wildcard), a two-name configured set, and `{name: kind}` so the
`unsupported configured question type` arm is reachable — and asserts, for
every input and every map:

- no panic (any panic fails the target);
- identical input gives an identical accept/reject decision and an identical
  diagnostic string (this is the property that surfaced defect 2);
- a rejection always carries a non-empty message, so a violation can never be
  reported as a silent success.

Seven seed entries keep the ordinary `go test` run deterministic. A manual
`-fuzztime 10s` run executed 3,631,518 inputs (≈331k/s) with no crash, and
`go run ./cmd/guard fuzz` now discovers **1** target and ran it for the
configured 30s budget with no findings. The table coverage that already existed
was not sufficient for this claim, hence the fuzz entry point.

### 4. Serve path and loader left alone

- `git diff --name-only 26c104f -- internal/config internal/host cmd` prints
  nothing (this stage and the two before it).
- `ValidateFixtureAnswers` has exactly two production call sites, both in
  `internal/cli/validate.go`'s `validateFixtureData`, which only `runValidate`
  calls; `serve` does not reach it.
- The one edit inside a function shared with generation (`validateHelperFields`)
  changes which of several candidate messages is chosen, never whether a
  fixture is accepted; the serve-time wire response for an invalid fixture is a
  fixed string, and no answer value produced by `GenerateAnswers` changed.

## Fail-open audit — probe classes and what actually rejects them

Method: every §13 rule that does not read the incoming request was mapped to a
place on the static path; every request-parameterised rule was re-derived as
"can *some* request satisfy the fixture?" — that re-derivation is what exposed
defect 1. Each class below was exercised through the built binary, not only in
unit tests.

| Probe class | Enforced by | Evidence |
| --- | --- | --- |
| duplicate JSON keys in an answers payload | `internal/config` load boundary: `rejectDuplicateJSON` for JSON input, `yaml.v3` duplicate-key error for YAML input. Both exit 2 before any compat code runs. | CLI rows `duplicate json keys in an answer`, `duplicate json keys in the answer map`. Also structural: member raws are handed to the compat layer only after an outer `json.Unmarshal` has validated the whole document, so a malformed or duplicate-collapsed member literal cannot reach it. |
| deeply nested answers document | `encoding/json` max-depth limit during `config.Load` (depth > 10000 → `invalid character '[' exceeded max depth`, exit 2). The compat validator itself recurses nowhere: one `objectFields` level per helper map plus one for `probabilities`. | CLI row `answers nested beyond the decoder depth limit` (16384 levels), plus 3.6M fuzz execs. |
| very large answers document | no spec-defined size limit; work is O(n log n) with allocation proportional to the document, and the document is bounded by what the operator supplies. | a 4000-question × 4000-answer JSON fixture loads and validates in ~0.02s with the built binary; CLI rows `accepted` / `one extra answer`. |
| wrong JSON type where an object is required | `objectFields` → `answers must be an object` / `helper must be an object` / `probabilities must be an object`. | existing rows `answers is an array`, `answers is null`, `answers member is not an object`, `answers member is a number`, `choice probabilities not an object`; new row `choice probability is not a number`. |
| wrong JSON type where a number is required | `jsonNumber` prefilter + `json.Unmarshal` into `float64` + `finiteNumber`/`finiteProbability`, the same primitives generation uses. | rows `noul string coercion prohibited`, `score not a number`, `confidence …`; new rows for `probability` values that are objects. |
| NaN / Inf-looking numbers | JSON cannot spell them (`NaN`/`Infinity` are not JSON tokens); YAML resolves `.nan`/`.inf`/`-.inf` to a `float64` that `json.Marshal` then refuses (`convert YAML configuration to JSON: json: unsupported value: NaN` / `+Inf`, exit 2). A JSON number outside `float64` (`1e999`) survives the loader as raw bytes and is rejected by the compat chain. | CLI rows `not a number from yaml`, `infinity from yaml`, `answer number beyond float64 range`; unit rows `noul beyond float64 range`, `score beyond float64 range`, `choice probability beyond float64 range`. |
| empty maps | `answers: {}` → rejected (configured: length rule; wildcard: new rule). `probabilities: {}` → sum rule (`probabilities must sum to 1`). `when.questions: {}` → `internal/config` (`questions must not be empty`, §38.6). | new unit rows `choice probabilities empty map`, `score probabilities empty map`, `wildcard answers key set empty`. |
| null payloads | `answers: null` → `answers must be an object`; a `null` member → `helper must be an object`; `confidence: null` / `score: null` / `probabilities: null` → the number/object rules; `then.answers: null` → `internal/config` (`must not be null`). | existing rows `answers is null`, `answers member is not an object`; CLI row for the document-level null; fuzz corpus seed `null`. |
| a sequence with a nested sequence | `internal/config` struct decode with `DisallowUnknownFields` (§38.7 "MUST NOT contain another `sequence`"): `decode configuration: json: unknown field "sequence"`, exit 2. | CLI row `nested sequence element`. |
| extremely large key set | same as "very large answers document": the key sets are compared in full via `sortedKeys`, never truncated; a single extra name in a 4000-name fixture is still reported with the stub id. | CLI rows `accepted` / `one extra answer`. |
| any path where a violation could be silently accepted | the two defects above were the paths found; both now fail closed. The remaining accepted-but-unservable class is recorded under residual risks, deliberately unchanged. | see residual risks |

Result: after actions 1 and 2 every class in the table above fails closed with a
message on stderr, and the answer documents the fuzz target accepts are the ones
the table explains. The one accepted-but-unservable shape that remained is
recorded under residual risks and was deliberately not changed.

## Command outcomes (all from the repository root, final revision)

| Command | Outcome |
| --- | --- |
| `gofmt -w` on the four `allowedFiles`, then `gofmt -l` on them | no output, exit 0 |
| `go test ./internal/compat/jev/v1 -count=1` | exit 0, 22 tests + 136 subtests, 0 failures |
| `go test ./internal/cli -count=1` | exit 0, 56 tests + 148 subtests, 0 failures |
| `go test ./... -count=1` | exit 0, 10 packages, 221 tests + 427 subtests, 0 failures |
| `go test -race ./... -count=1` | exit 0, same packages, 0 failures |
| `go vet ./...` | no output, exit 0 |
| `go run ./cmd/guard arch` | exit 0, `"check": "arch"`, `"findings": []`, 60 files / 10 packages |
| `go run ./cmd/guard lint` | exit 0, `"check": "lint"`, `"findings": []`, 60 files / 10 packages |
| `go run ./cmd/guard trace` | exit 0, `"check": "trace"`, `"findings": []` (active 0, covered 17, test files 22) |
| `go run ./cmd/guard fuzz` | exit 0, `"check": "fuzz"`, `"findings": []`, `"targets": 1` (the new target, 30s budget) |
| `go build ./cmd/fake-jev` | exit 0 |
| `./scripts/candidate-fingerprint candidate` | `80852a3b20015001679e3171fe0271619845cf02203b70f2cb59470b976af21c`, exit 0 |
| `git diff --name-only 26c104f -- internal/config internal/host cmd` | no output |
| `git diff --cached --name-only` | no output (nothing staged) |

Added coverage: 7 rows in `TestValidateFixtureAnswers`, 2 compat test
functions (one of them the fuzz target), 2 rows in
`TestValidateRejectsStaticallyInvalidFixtureData`, 1 row in
`TestValidateIsStricterThanTheServeLoadPath` (its table gained a `stub` field so
the row names its own stub), plus `TestValidateRejectsMalformedFixtureDocuments`
(6 rows) and `TestValidateHandlesLargeFixtureKeySets` (2 rows).

## Residual risks

1. **Accepted-but-unservable score fixtures (unchanged, needs a scope
   decision).** Explicit score probabilities can be accepted by `validate` even
   though no request can ever make response generation succeed:

   ```yaml
   - id: score-dead
     when: {questions: {severity: score}}
     then: {answers: {severity: {score: 1, probabilities: {"0": 0.2, "1": 0.8}}}}
   ```

   `validate` reports `configuration is valid` (exit 0), while `serve` answers
   500 `fake_jev_invalid_stub_response` for criteria lengths 2, 3 and 4
   (verified). A second shape, keys `{"0": 0.5, "2": 0.5}`, behaves the same
   way. The specifier recorded §13.3's score key set, range and
   probability-weighted expected value as "not statically decidable" because
   `N` is request-derived; the audit refines that: the fixture's own key set
   fixes the only candidate `N = len(keys)`, so the existence of a servable
   request is in fact decidable. This class is **not** changed here: adding
   those checks would be new `validate`-path rules outside the approved
   statically-provable subset, would restate generation's key-set/range/
   expected-value blocks a fourth time (C4, AGENTS "avoid duplicating
   non-trivial business logic"), and would introduce the new interpretation
   "a fixture must be servable by at least one request" that the specification
   does not state. It needs an explicit scope decision and its own work item.
   The wildcard-empty fix above is a different case: it is an instance of
   §13.4/§13.5 for every request, not an existence argument.
2. **Answer rules deliberately unchecked, because the request decides them.**
   Each is request-parameterised, so the configuration alone does not fix it:
   §13.2 `probabilities` keys equal to the request criteria key set; the
   selected `choice` existing as a request criterion; the selected `choice`
   holding the maximum probability; `confidence` defaulting to the selected
   choice's probability; §13.3 the score level range and expected value for the
   request's `N` (see risk 1 for the existence-based refinement); §13.4/§40.5
   helper-type agreement and exact cover for a wildcard stub, whose request
   question names and types are unknown; §12.6/§38.7 `raw` bodies, which are
   intentionally not validated against `jev/v1` answer schemas. All of these
   remain enforced at request time by `GenerateAnswers`.
3. **Diagnostic wording of a rejected configuration document.** Duplicate JSON
   keys are rejected on the JSON path by `rejectDuplicateJSON`, which makes
   `toJSON` fall through to the YAML decoder; the final message is therefore
   YAML-worded (`decode YAML configuration: yaml: unmarshal errors: … mapping
   key "noul" already defined …`) even for a `.json` input. Behaviour is
   fail-closed and the wording is not normative; the fix would live in
   `internal/config`, outside `allowedFiles`.
4. **Underflow is accepted silently.** A JSON `1e-999` reaches the compat layer
   as bytes, decodes to `0` without error, and is accepted as `noul: 0`
   (exit 0). Generation uses the same primitives and also yields `0`, so
   `validate` and `serve` agree; only literal intent is lost.
5. **Duplication risks carried over from the cleaner** (not introduced here):
   three spellings of the answers↔questions key-set rule and one verbatim
   `score must be a finite number` expression, each bounded by
   `TestValidateFixtureAnswersNeverAcceptsWhatGenerationRejects` on the shared
   subset but not mechanically pinned against the other spellings. This stage
   also did not remove the unreachable `answer provided for absent question`
   branch or rename the test row that asserts `missing answer for question "n"`;
   both remain behaviour- and coverage-neutral.
6. **Not run by this stage:** `./scripts/verify-candidate FJ-059` — it writes
   `.agent/reports/**`, outside this stage's write scope. It is the next gate
   for candidate `80852a3b20015001679e3171fe0271619845cf02203b70f2cb59470b976af21c`.
