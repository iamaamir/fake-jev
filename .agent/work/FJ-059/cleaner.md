---
stage: cleaner
task: FJ-059
inputFingerprint: bb445e590122427e20db13937225cf7a5b4eda3d9f223d907b07463fbc6e7248
outputFingerprint: bb445e590122427e20db13937225cf7a5b4eda3d9f223d907b07463fbc6e7248
taskFingerprint: 879b10dee885044f7a81eed718e8b111e040775d016d6fa1c2bb8575fdbcd179
gitHead: 26c104f
generatedAt: 2026-10-02T20:32:37Z
author: worker/FJ-059-cleaner
---

# Cleaner — FJ-059 (static jev/v1 fixture validation for `fake-jev validate`)

Reviewed revision: candidate `bb445e59…c6e7248`, i.e. the coder revision at
`26c104f`. Diff under review: `internal/compat/jev/v1/fixture.go`
(+152/-0), `internal/compat/jev/v1/fixture_test.go` (+133/-0),
`internal/cli/validate.go` (+31/-4), `internal/cli/validate_test.go` (+146/-0).

## Cleanup applied: none — coder revision kept unchanged

`outputFingerprint` equals `inputFingerprint` because no file content changed.
This is deliberate, not an omission: the two edits the review produced were
tried, measured, and then reverted.

During the review I applied two local edits — deleting the unreachable
`answer provided for absent question` loop in `ValidateFixtureAnswers`
(`fixture.go:302-306`) and renaming the test row `absent question answered`
(`fixture_test.go:290`). The whole suite was green on that revision
(`gofmt` clean, compat 142, cli 192, `./...` 620, `-race` 620, `vet` clean,
guard arch/lint `"findings": []`), and the candidate fingerprint moved to
`b3df1984…4dba201b`. Both edits were then reverted; the fingerprint returned to
`bb445e59…c6e7248`, which is byte-for-byte equality with the coder revision and
is the evidence that the revert is exact.

Reason for reverting: every candidate edit is cosmetic (defect classes are
recorded below with proofs), while this stage's artifact is pinned to the coder
fingerprint, so an edit would move the chain to a new candidate for no
behavioural or coverage gain. The findings stay available to the hardener/QA
stages and to any follow-up item without re-pinning this stage.

## Duplication and drift analysis

Shared, not restated (verified by reading the call graph, not just the
comments):

- `noul` payload rules — the static path calls the existing
  `generateNoulAnswer(fields)` (`fixture.go:324-326`), so the boolean form, the
  `[0,1]` finite rule, and the exactly-one-field rule exist once.
- allowed-field sets — `validateHelperFields` is called from both paths
  (`fixture.go:360`, `:387`), which is also what makes `legend` on `score` fail.
- probability values/shape/`[0,1]`/finiteness — `probabilitiesField`
  (`fixture.go:366`, `:394`); confidence — `confidenceField`; numbers —
  `jsonNumber`/`finiteNumber`/`finiteProbability`.

Duplications that exist and could drift, with the reason each is unavoidable at
this revision:

1. **Answers↔question key-set rule is now spelled three times.** The block also
   exists in `GenerateAnswers` (`fixture.go:37-54`) and in
   `validateResponseAnswers` (`response.go:125-142`); all three use the same
   message texts (`answers must exactly cover …`, `missing answer for
   question %q`, `answer provided for absent question %q`). The static copy
   cannot share generation's code: generation compares against
   `request.Questions` and returns `*InvalidStubResponseError`, and the static
   path compares against the configured set with a plain `error` because no
   request exists. The pre-existing pair already carried this drift surface;
   FJ-059 adds the third spelling rather than removing one. Wordings currently
   agree character-for-character (checked by grep across
   `internal/compat/jev/v1/*.go`).
2. **`score` finite-number rule is duplicated verbatim.** `fixture.go:391` is
   the same expression and the same message as `fixture.go:182`
   (`!jsonNumber(…) || json.Unmarshal(…) != nil || !finiteNumber(score)` →
   `score must be a finite number`). Removing it would require editing
   `generateScoreAnswer`, whose body this item freezes (C3/C4 and the
   "no change to response generation at serve time" non-goal), or adding a new
   shared `scoreField` helper, which is a larger change than the duplication it
   removes. Drift in the accepting direction is bounded by
   `TestValidateFixtureAnswersNeverAcceptsWhatGenerationRejects`.
3. **Helper dispatch is spelled twice** (`generateAnswer`'s switch,
   `fixture.go:68-84`, vs `validateConfiguredAnswer`'s switch,
   `fixture.go:318-339`) and the helper-name list a third time in
   `validateDeclaredAnswer` (`fixture.go:346`), plus a fourth time in
   `internal/config`'s `validateWhen` question-type switch. The split is forced:
   generation needs `request.Question` (criteria), and the static path must not
   fabricate criteria — doing so would impose request-relative rules on values
   the request alone can decide. The ordered probe list in
   `validateDeclaredAnswer` is behaviour-inert beyond selecting the declared
   helper: a sibling helper key is rejected either by
   `validateHelperFields` (choice/score) or by `len(fields) != 1` (noul), which
   the `wildcard mixed helpers` row pins.
4. **Choice string rule uses a different primitive on each side.** Static:
   `isString(fields["choice"])` (`fixture.go:363`, and `isString` is the
   package's existing request-validation primitive). Generation:
   `json.Unmarshal(choiceRaw, &choice) != nil` (`fixture.go:118`). These agree
   for every value `config.Load` can deliver, because the loader decodes the
   document (JSON, or YAML marshalled to JSON) before validation, so a member is
   always a well-formed JSON value; they diverge only on malformed raw JSON,
   which cannot reach either path.
5. **`validateProbabilityMap` is reused with the map's own keys**
   (`fixture.go:373`, `:401`), so four of its five branches are inert for this
   call (length, unknown key, missing key, and the finite re-check that
   `probabilitiesField` already performed). Only the sum-within-`1e-6` rule can
   fire, which is exactly the statically decidable part; the call-site comments
   state this and are accurate. No new message text was introduced for the sum
   rule.

Net: no rule that generation enforces on the shared subset is re-implemented
from scratch; the items above are the irreducible residue of reusing a
request-shaped generator without editing its bodies.

## `validate` strictness versus the serve load path

- Serve-path files untouched:
  `git diff --name-only 26c104f -- internal/config internal/host cmd internal/compat/jev/v1/response.go`
  prints nothing, and `fixture.go`'s diff is additions only, so no existing
  generation or helper body moved. `config.Load` strictness is unchanged.
- `ValidateFixtureAnswers` has exactly two production call sites, both in
  `validateFixtureData` (`internal/cli/validate.go:45`, `:49`); it is not
  reachable from `serve`. Grep for `ValidateFixtureAnswers|validateFixtureData`
  returns only `fixture.go`, `fixture_test.go`, `validate.go`.
- The asymmetry is pinned by test rather than by comment:
  `TestValidateIsStricterThanTheServeLoadPath` loads `fixtureViolationYAML` with
  `config.LoadFile` (must succeed) and runs `validate` on it (must exit 2 with a
  stderr-only diagnostic).
- Streams/exit codes are unchanged in shape: the new failure reuses the same
  `writef(stderr, "fake-jev: %s: %v\n", path, err)` + `exitFailure` path as a
  load failure; the usage path is untouched; the success line is untouched.
- No-listener/no-network properties still hold: `TestValidateOpensNoListener`
  and `TestValidatePathLinksNoNetworkingPackage` run green with the new
  `fake-jev/internal/compat/jev/v1` import in `validate.go`'s closure.

## Error messages versus the existing validate style

- Envelope is unchanged: `fake-jev: <path>: <err>` on stderr, nothing on
  stdout.
- The identity token added by the composition root (`stub %q: …`,
  `stub %q sequence[%d]: …`) matches the vocabulary `internal/config` already
  uses (`stub %q when: …`, `stub %q then: …`), so the two error families read as
  one style. Attribution is by stub id, and for sequences by 0-based element
  index.
- Compat-side texts mirror generation's (`missing answer for question %q`,
  `helper must be an object`, `noul helper is required`, `probabilities must sum
  to 1`) so a fixture failure reads the same pre-request and post-request.
- Returned type is a plain `error`, not `*InvalidStubResponseError`: the wire
  code `fake_jev_invalid_stub_response` is defined as a response to a validated
  request, and this path has no request, so leaking it would be wrong.
- No wording is normative (§42.1 fixes exit codes, §42.5 the stream split), and
  the tests assert exit code, stream split, and the `stub "<id>"` token rather
  than sentences — consistent with how the rest of `validate_test.go` asserts.

## Dead code, control flow, comment accuracy

- **Unreachable branch (found, not changed):** `fixture.go:302-306`,
  `answer provided for absent question %q`. Because
  `len(answerFields) != len(questions)` is rejected first, the branch cannot
  fire: a key-set mismatch with equal cardinality means some configured question
  is absent, which the preceding loop reports first. Evidence: a temporary
  reachability probe (exhaustive over all 7 non-empty question subsets × 8
  answer subsets of a 3-name space, 56 inputs) recorded
  `map[<nil>:7 cover:37 missing:12]` — zero occurrences of that message; the
  probe file was removed before this artifact was written. The same branch
  exists in `GenerateAnswers` (`fixture.go:51-55`) and
  `validateResponseAnswers` (`response.go:137-141`), so keeping it holds the new
  function parallel to two frozen twins, and the branch is inert for every
  input, so leaving it costs nothing at runtime.
- **Test row name overstates its assertion (found, not changed):**
  `fixture_test.go:290` is named `absent question answered` but asserts
  `missing answer for question "n"` — the same assertion as the neighbouring
  `missing answer` row with a different answer key. The row still fails if the
  key-set check regresses, so coverage is not lost; only the name is wrong,
  and it names a branch that cannot execute (previous bullet).
- **Comment broader than the code (found, not changed):**
  `validateConfiguredAnswer`'s doc says "with the same field rules"; the static
  path in fact enforces only the request-independent subset, and `score`'s range
  and `choice`'s criteria membership/maximum live only in generation. The
  over-claim is local and self-corrected: the sub-helpers below it say
  "applies the §13.2/§13.3 rules that do not read the request", and
  `ValidateFixtureAnswers`'s doc enumerates exactly which request-relative rules
  are not attempted.
- **Control flow:** straight-line and shallow; per-stub and per-element loops
  are the only nesting added. All map iteration goes through `sortedKeys`, so
  the first reported violation is deterministic and matches generation's
  sorted order. `validateFixtureData` allocates nothing beyond `fmt.Errorf`.
- **Fail-closed guards kept although unreachable from the CLI:** the
  `default: unsupported configured question type` arm (`fixture.go:336`) and
  `answers must be an object` (`fixture.go:282`). The first is reachable through
  the exported function with any caller-supplied map and removing it would force
  a silent accept; the second is reachable through the exported function with a
  non-object argument and is covered by the `answers is an array` /
  `answers is null` rows. Neither is dead in the removable sense.
- No TODO, placeholder, unused symbol, or new exported surface beyond
  `ValidateFixtureAnswers` (required by C2).

## Test quality: does a broken rule fail a row?

Checked row by row against the implementation. Internal-table rows that fail
when their rule is removed: the key-set count rows (`must exactly cover the
configured questions` would be replaced by the `missing answer` / `absent
question` texts if the length check went away), the `missing answer` row, the
three helper/type-agreement rows, every `noul`/`choice`/`score` payload row
(each names the rule-specific message), the document/member-shape rows, and the
wildcard rows (removing the `questions == nil` branch turns them into
`must exactly cover the configured questions`, so they fail). Valid rows are
non-vacuous: they fail if the static path over-rejects (e.g. `choice sum
tolerance` at 0.5000005 pins `1e-6`, `noul boundaries`/`noul upper boundary`
pin `[0,1]` inclusivity, `no answers form` pins the empty early return).

CLI-level rows pin the wiring, not the rule: `sequence element key set
mismatch`, `sequence element payload invalid` (the inner loop over
`Then.Sequence` would go untested if it were deleted), `every stub is checked`
(two stubs; the violation is in the last), and the single valid-file row that
carries `when.questions` end to end.

Gaps observed, none of them a coverage claim made by the artifact:

- `validateConfiguredAnswer`'s `default` arm has no unit row (unreachable from
  `config.Load`, which admits only `noul`/`choice`/`score`).
- `TestValidateFixtureAnswersNeverAcceptsWhatGenerationRejects` pins one
  direction only (static must not reject what generation accepts). The
  converse cannot be asserted, because fixtures generation rejects for
  request-relative reasons must be accepted statically; the test documents this
  by keeping such fixtures in the list while counting only the
  generation-accepting subset (asserted `>= 3`).
- The property test's fixture set is fixed rather than table-driven from the
  generation test corpus, so adding a helper to generation would not
  automatically extend it.
- `answer provided for absent question` (see above) can never be asserted by
  any row on any input.

## Reachability: every stub, including sequence elements

- Entry: `Run(["validate", path])` → `runValidate` → `config.LoadFile` →
  `validateFixtureData`. The loop is `for i := range cfg.Stubs`, so it visits
  every parsed stub in configuration order; `StubConfig` has no enable/disable
  field, so no stub is skipped, and `internal/config` normalises or rejects
  every stub profile (`normalizeProfile` admits only `jev`/`jev/v1`), so "every
  enabled profile-specific fixture data" (§15.2) is exactly the stub list here
  and no profile gate is needed at the call site.
- For each stub, `stub.Then.Answers` is validated, and every
  `stub.Then.Sequence[i].Answers` is validated in index order. A sequence
  element is a `config.ResponseConfig`, the same value `serve` feeds to
  `EncodeResult` → `GenerateAnswers`, so the static check mirrors the serve
  path element by element rather than only for the direct form.
- Stubs/elements whose form is `raw` pass an empty `Answers` and are accepted by
  the `len(answers) == 0` early return. That is not a hole: `config.Load`
  counts forms by `len(Answers) > 0`, so an empty `answers` object cannot be the
  sole declared form (it is rejected as "exactly one of answers, sequence, or
  raw is required"), and a sequence element must declare `answers` or `raw`.
- Documented non-reachability, unchanged from the specifier contract: rules
  that read the request (`choice` criteria key set, membership, maximum,
  `confidence` default, `score` `0…N-1` key set, range, expected value,
  wildcard type agreement) and `raw` bodies.

## Command outcomes (revision `bb445e59…c6e7248`, repo root)

| Command | Outcome |
| --- | --- |
| `gofmt -l` on the four changed files | no output, exit 0 |
| `go test ./internal/compat/jev/v1 -count=1` | 142 passed, exit 0 |
| `go test ./internal/cli -count=1` | 192 passed, exit 0 |
| `go test ./... -count=1` | 620 passed in 10 packages, exit 0 |
| `go test -race ./... -count=1` | 620 passed in 10 packages, exit 0 |
| `go vet ./...` | no output, exit 0 |
| `go run ./cmd/guard arch` | exit 0, `"check": "arch"`, `"findings": []`, 60 files / 10 packages |
| `go run ./cmd/guard lint` | exit 0, `"check": "lint"`, `"findings": []`, 60 files / 10 packages |
| `./scripts/candidate-fingerprint candidate` | `bb445e590122427e20db13937225cf7a5b4eda3d9f223d907b07463fbc6e7248`, exit 0 |
| `git diff --name-only 26c104f -- internal/config internal/host cmd internal/compat/jev/v1/response.go` | no output (serve path unchanged) |
| `go test ./internal/cli -run 'TestValidateOpensNoListener\|TestValidatePathLinksNoNetworkingPackage\|TestValidateRejectsStaticallyInvalidFixtureData\|TestValidateIsStricterThanTheServeLoadPath\|TestValidateAcceptsValidConfigurationFiles' -count=1` | all PASS; 10 fixture-violation subtests, 4 valid-file subtests, 2 strictness subtests |

Working tree at review end: the four `allowedFiles` modified as reviewed plus
`state.json` (modified before this stage), and the untracked stage artifacts;
no probe, scratch, or stray file remains.

## Residual risks

- Three spellings of the answers↔questions key-set rule and one verbatim
  `score must be a finite number` duplicate remain (see duplication section);
  each is bounded by tests today but none is mechanically pinned against the
  other, so a future change to one spelling can diverge silently outside the
  covered fixtures.
- The unreachable `answer provided for absent question` branch and the test row
  named `absent question answered` remain in the revision. They are behaviour-
  and coverage-neutral, but the row name can mislead a later reader into
  believing that branch is exercised.
- `validateConfiguredAnswer`'s "with the same field rules" comment remains
  broader than the code; a reader who stops at that line could over-estimate
  what `validate` proves.
- `TestValidateFixtureAnswersNeverAcceptsWhatGenerationRejects` is a fixed
  fixture list; it will not automatically widen when generation grows.
- Not run by this stage: `./scripts/verify-candidate FJ-059` (it writes
  `.agent/reports/**`, outside this stage's write scope). It is the next gate
  for candidate `bb445e590122427e20db13937225cf7a5b4eda3d9f223d907b07463fbc6e7248`.
