---
stage: hardener
task: FJ-040
inputFingerprint: 525ee08845fbb9098ba37a5146877529863cddd57dcb263e9050628d63b9a84a
outputFingerprint: 4d9e1ad756b11f6771e4e2d24cb495181300a120125791cad14551ca17ff5f5e
taskFingerprint: ae3ccc43bc6a790f3329233e8585379262d6e5b49e2751da583353742d06c6e4
gitHead: 433fc97
generatedAt: 2026-10-03T20:12:05Z
author: worker/FJ-040-hardener-4
---

# FJ-040 hardener (fourth pass): finalize the repeated-key-bomb seed repair

Fourth hardener pass. The third pass (`worker/FJ-040-hardener-3`) applied the code
repair to `internal/control/fuzz_test.go` and then ran out of time before writing its
artifact, leaving a skeleton with `outputFingerprint: PENDING`. This pass confirms the
repair is present, records the accounting, and adds the per-seed audit.

## **`outputFingerprint: 4d9e1ad756b11f6771e4e2d24cb495181300a120125791cad14551ca17ff5f5e`**

This pass wrote nothing except this file. No test file was edited by this pass: the
only real defect found by the audit was the `repeated-key-bomb` seed, and that repair
was already in place when this pass started.

## Fingerprints and chain

| observation | value |
| --- | --- |
| `git rev-parse HEAD` | `433fc974544cbf912a93b3f50cd2011bdd9511fa` (`433fc97`) |
| `qa.md` `outputFingerprint` (read from the artifact, not from the packet) | `8961ce46838962ae48d5bf900aa2013b9008df3c595b84cfa6e72c614d7cc434` |
| skeleton `hardener.md` `inputFingerprint` (read from the artifact) | `8961ce46838962ae48d5bf900aa2013b9008df3c595b84cfa6e72c614d7cc434` |
| this pass's `inputFingerprint` | `8961ce46838962ae48d5bf900aa2013b9008df3c595b84cfa6e72c614d7cc434` |
| `./scripts/candidate-fingerprint candidate` (this pass) | `4d9e1ad756b11f6771e4e2d24cb495181300a120125791cad14551ca17ff5f5e` |
| this pass's `outputFingerprint` | `4d9e1ad756b11f6771e4e2d24cb495181300a120125791cad14551ca17ff5f5e` |

`qa.md`'s front matter records `inputFingerprint` and `outputFingerprint` as the same
value `8961ce46…`, and that value is also the skeleton `hardener.md`'s
`inputFingerprint`, so the QA→hardener chain is consistent at `8961ce46…`. The packet's
claim was checked against the artifacts rather than taken on trust, and it holds.

**Discrepancy, measured.** The current workspace does not hash to `8961ce46…` any
more: `./scripts/candidate-fingerprint candidate` now returns `4d9e1ad7…`. The cause is
the third hardener pass itself. `qa.md` § *Determinism, hygiene, and command outcomes*
pins the four scoped files at hashes; three of the four still hash to those values and
exactly one moved:

| file | sha256 per `qa.md` | sha256 now |
| --- | --- | --- |
| `internal/control/fuzz_test.go` | `534141def7a50ab65ee4f3428e7f6d5996ab6a958c59d1524b06416fc7d45233` | `afb63fea7ea86572745d1d94f0c75c7c22e29da323ff06ae964a324eea3e48b2` |
| `internal/config/fuzz_test.go` | `932a8eec55e382a5b2adbf450e78c159f5c7f60296008a4db6c7a28071dec3c8` | unchanged |
| `internal/engine/race_test.go` | `c757a234551693962e1673b718ac584e2fae3eee672e32d7193022e213486911` | unchanged |
| `internal/compat/jev/v1/fuzz_test.go` | `c72cf7e9619e9fdfe30ec3ee3e9dfacd5a2a5112a42ce839e7f1eb40edf854e1` | unchanged |

That `internal/control/fuzz_test.go` is the only covered path whose content moved is
the whole of the explanation: `candidate-fingerprint` excludes `.agent/work/` and
`.agent/reports/` (`EXCLUDED_PREFIXES` in `scripts/candidate-fingerprint`) and
`.agent/work/FJ-040/state.json` therefore contributes nothing to the digest, so the
`8961ce46…` → `4d9e1ad7…` shift is attributable to the seed repair and nothing else.
`git diff --stat 433fc97 -- .` reports one path, `.agent/work/FJ-040/state.json` (35
insertions, 10 deletions); `git diff --cached --stat` is empty (zero staged paths).
This artifact's own edit cannot move the digest, because this file lives under the
excluded `.agent/work/` prefix — the value above was measured before and after the
body was written and did not change.

One bounded side-check on that attribution: a copy of the workspace with only the seed
literal reverted to `{"`+… (its current comment text kept) hashes to
`6389aad6b68d957f702144a85b2157111bdb13589da66a6c76415aa7b15015ea`, a third value.
The digest is therefore sensitive to this line, and the qa-era file differed from the
current one in its surrounding comment as well as in the literal, which is why the
reverted-literal copy does not reproduce `8961ce46…`.

Scope is otherwise as `qa.md` described: the only untracked test files are the four in
scope, none of the four is tracked, and no other product path changed.

## The repair and its byte accounting

`internal/control/fuzz_test.go` registers the seed as (lines 124–125):

```go
{"repeated-key-bomb", http.MethodPost, "/__fake/v1/stubs", "application/json",
	`{` + strings.Repeat(`"a":1,`, 1600) + `"a":1}`},
```

Body accounting, derived from the literal as written:

| shape | construction | length |
| --- | --- | --- |
| corrected (current) | `{` + 1600 × `"a":1,` + `"a":1}` | 1 + 6×1600 + 6 = **9,607 bytes** |
| malformed (previous) | `{"` + 1600 × `"a":1,` + `"a":1}` | 2 + 6×1600 + 6 = **9,608 bytes** |

The one-byte difference is the stray `"` after the opening `{`. With the corrected form
the body is `{"a":1,"a":1,…,"a":1}` — valid JSON, one key name (`a`) occurring 1,601
times, i.e. 1,600 duplicate keys. With the malformed form the body begins `{""a":1,…`:
an empty key string followed by an unexpected `a`, which is invalid JSON, so the
request is rejected by the generic invalid-request path before `config.Load` ever runs
and the duplicate-key handling FJ-065 added is never reached.

## Evidence that the corrected seed is meaningful (QA A/B)

`qa.md` records an A/B run in a throwaway copy with the FJ-065 behaviour reverted (the
`errors.As(err, &duplicate)` short-circuit removed from `toJSON`, and
`boundDuplicateKeyDiagnostics` made a no-op):

```text
A) malformed seed (9,608 B), pre-FJ-065 product:
   go test -run '^FuzzControlMalformedInput$' ./internal/control -count=1
   ok  	fake-jev/internal/control	0.200s

B) corrected seed (9,607 B), pre-FJ-065 product:
   go test -run '^FuzzControlMalformedInput$' ./internal/control -count=1
   --- FAIL: FuzzControlMalformedInput (0.43s)
       --- FAIL: FuzzControlMalformedInput/seed#10 (0.43s)
           fuzz_test.go:158: control request POST "/__fake/v1/stubs" answered 70444103 bytes
```

Reading: the malformed seed stayed green on a product whose duplicate-key handling is
broken, i.e. it was inert with respect to the regression it documents. The same seed
with the stray quote removed fails on that product with a 70,444,103-byte response, so
the body content is what decides whether the 64 KiB response bound is exercised. The
prior artifact's 70,356,103-byte figure for its own 9,607-byte construction differs
from QA's 70,444,103 by 88,000 bytes, consistent with two slightly different literals at
the same repetition count; either way the assertion fires and the corrected seed is
live. `seed#10` is `repeated-key-bomb`: the 16 seeds are added in source order from
index 0, and `repeated-key-bomb` is the eleventh entry.

## This pass's verification that the seed reaches the duplicate-key path

Re-demonstrated in a scratch copy under `/private/tmp/fj040-h4/probe` (a copy of the
current workspace with one extra in-package test file; no repository file was created
or modified). The probe re-derives the body from the same expression as the seed and
drives `API.ServeHTTP` through the target's own `newFuzzedAPI`/`controlRawRequest`
helpers:

```text
PROBE name=corrected bytes=9607 jsonValid=true  decodedDistinctKeys=1 decodedLen=1 rawKeyTokens=1601 status=400 respBytes=106 resp={"error":"fake_jev_bad_control_request","message":"decode JSON configuration: duplicate object key \"a\""}
PROBE name=malformed bytes=9608 jsonValid=false decodedDistinctKeys=0 decodedLen=0 rawKeyTokens=1601 status=400 respBytes=77  resp={"error":"fake_jev_bad_control_request","message":"Invalid control request."}
```

- `json.Valid` (from `encoding/json`, the same decoder the product uses) accepts the
  corrected body and rejects the malformed one.
- `rawKeyTokens` counts literal `"a":` occurrences: 1,601 in both bodies (1,600 with a
  trailing comma plus the final one), i.e. 1,600 duplicates of one key name. The
  corrected body decodes to a single-entry JSON object; the malformed body does not
  decode at all.
- The live route answers the corrected 9,607-byte body with the **constant 106-byte**
  duplicate-key diagnostic `decode JSON configuration: duplicate object key "a"`. The
  malformed 9,608-byte body is answered with the generic **77-byte**
  `Invalid control request.` envelope. The seed therefore does reach the duplicate-key
  path, and the 106-vs-77 byte contrast is the observable difference between reaching it
  and being short-circuited earlier.

## Per-seed inertness audit (the audit question carried in from the packet)

Class asked about: a seed that is inert because its body lands on a generic path or is
rejected earlier than the seed name claims. Each of the four allowed files was read in
full and each seed was driven through the same product entry point the target uses, in
the scratch copy, so the routing decision is observed rather than inferred.

### `internal/control/fuzz_test.go` — 16 seeds, all executed, one real defect (already repaired)

Bodies, caps, and observed response per seed (`skipped` = the target's own size guard
would drop it before any assertion; status/response are from the live route):

| # | seed | body bytes | skipped | status | resp bytes | response |
| --- | --- | --- | --- | --- | --- | --- |
| 0 | health | 0 | no | 200 | 68 | ok envelope |
| 1 | meta | 0 | no | 200 | 266 | ok envelope |
| 2 | stubs | 0 | no | 200 | 12 | `{"stubs":[]}` |
| 3 | requests | 0 | no | 200 | 15 | `{"requests":[]}` |
| 4 | verify | 0 | no | 200 | 29 | `{"passed":true,"failures":[]}` |
| 5 | reset | 0 | no | 204 | 0 | no body |
| 6 | empty-id | 9 | no | 400 | 78 | `stubs[0].when is required` |
| 7 | truncated-json | 1 | no | 400 | 77 | generic invalid |
| 8 | json-null | 4 | no | 400 | 79 | `stubs[0] must be an object` |
| 9 | two-values | 134 | no | 400 | 77 | generic invalid |
| 10 | repeated-key-bomb | 9,607 | no | 400 | **106** | `duplicate object key "a"` |
| 11 | unknown-path | 2 | no | 404 | 76 | unknown endpoint |
| 12 | unknown-method | 0 | no | 405 | 82 | method not allowed |
| 13 | deep-brackets | 4,096 | no | 400 | 77 | generic invalid |
| 14 | hostile-content-type | 0 | no | 200 | 68 | ok envelope |
| 15 | valid-create | 67 | no | 201 | 38 | `registrationIndex:1` |

Findings:

- **`repeated-key-bomb` (#10) was the one real defect of the class** — its body was
  rejected as invalid JSON before `config.Load`, so it never exercised the
  duplicate-key regression it is named for. Repaired (9,608 → 9,607 bytes, generic 77 →
  duplicate-key 106). This is the repair this pass finalizes; no further edit was made.
- Seeds #7, #9 and #13 answer the generic 77-byte envelope. Their names describe the
  input shape (`truncated-json`, `two-values`, `deep-brackets`), they carry no comment
  claiming a specific product path, and their purpose in the corpus is malformed-body
  coverage on the shared invalid-request path — which they exercise. They are not
  defects of this class; in particular `deep-brackets` never reaches a nesting-depth
  limit because the body is truncated JSON, but nothing claims that it does.
- Seeds #6 and #8 are *not* generic: `{"id":""}` and `null` both decode and are rejected
  by stub validation (`stubs[0].when is required`, `stubs[0] must be an object`), so
  they reach the validation layer rather than the decoder's error path.
- No seed is dropped by the size caps (`skipped=false` for all 16, which is what makes
  the file's own guard the only reason the 9,607-byte seed is inside the target's
  16 KiB body cap).

### `internal/config/fuzz_test.go` — 18 seeds, all fuzzed, no inert seed

Per seed: separator count (`configFuzzMappingKeys`), which decoder `Load` routes it to
(`configFuzzYAMLPath`, which reuses the product's own scan), whether the target's guard
would skip it, and the `Load` outcome.

| # | head | bytes | seps | YAML path | skipped | `Load` outcome |
| --- | --- | --- | --- | --- | --- | --- |
| 0 | `{"schemaVersion":1}` | 19 | 1 | no | no | accept |
| 1 | `{}` | 2 | 0 | no | no | reject: schemaVersion is required |
| 2 | `null` | 4 | 0 | yes | no | reject: configuration must be an object |
| 3 | `[]` | 2 | 0 | no | no | reject: cannot unmarshal array |
| 4 | `"x"` | 3 | 0 | yes | no | reject: cannot unmarshal string |
| 5 | `0` | 1 | 0 | yes | no | reject: cannot unmarshal number |
| 6 | `schemaVersion: 1` | 17 | 1 | yes | no | accept |
| 7 | two YAML documents | 38 | 2 | yes | no | reject: multiple YAML documents |
| 8 | duplicate `schemaVersion` | 37 | 3 | no | no | reject: duplicate object key "schemaVersion" |
| 9 | `"unknown":true` | 34 | 3 | no | no | reject: unknown field "unknown" |
| 10 | `schemaVersion: 2` | 19 | 1 | no | no | reject: must equal integer 1 |
| 11 | negative `maxInteractions` | 51 | 4 | no | no | reject: limits.maxInteractions must be at least 1 |
| 12 | stub with `raw` | 114 | 15 | no | no | reject: unknown field "raw" |
| 13 | `mode:"auto"` | 33 | 3 | no | no | reject: mode "auto" is not supported |
| 14 | 1,024 × `<` | 1,024 | 0 | yes | no | reject: cannot unmarshal string |
| 15 | `"müde"` model | 94 | 8 | no | no | accept |
| 16 | `{"schemaVersion":1}\x00` | 20 | 1 | yes | no | reject: YAML control characters not allowed |
| 17 | 256 × `[` + 256 × `]` | 512 | 0 | no | no | reject: cannot unmarshal array |

- **No seed is skipped** by `configFuzzBeyondKeyBound` (max separator count is 15, far
  below the 256 bound), so every seed is handed to `Load` under plain `go test`.
- The duplicate-key shape is covered by seed #8, and it genuinely reaches the JSON scan:
  `yamlPath=false`, rejection is `duplicate object key "schemaVersion"`, i.e. the
  FJ-065 path. That is the config-side analogue of the control defect, and it is sound.
- Seed #16 is the one seed whose routing is subtle — a trailing NUL makes the JSON scan
  fail, so `Load` falls through to the YAML decoder, which then rejects the document.
  That is the product's documented fallback, and the seed exercises it rather than
  being deflected from it.
- The comment on `configFuzzMappingKeys`/`configFuzzBeyondKeyBound` is accurate about
  its own limits: the guard is applied to seeds as well, and none of these small seeds
  trips it.

### `internal/compat/jev/v1/fuzz_test.go` — 13 seeds, all live, one neighbouring observation

| # | seed | skipped | route | decode | answers |
| --- | --- | --- | --- | --- | --- |
| 0 | models | no | models | accept | n/a (no questions) |
| 1 | systemone-valid | no | systemone | accept | ok, 1 question |
| 2 | systemone-invalid-answer | no | systemone | accept | reject: noul must be in [0,1] |
| 3 | configured-null | no | systemone | accept | reject: answers must be an object |
| 4 | configured-array | no | systemone | accept | reject: answers must be an object |
| 5 | configured-empty-object | no | systemone | accept | reject: answers must exactly cover request questions |
| 6 | body-truncated | no | systemone | reject: Invalid JSON | — |
| 7 | body-null | no | systemone | reject: Invalid value | — |
| 8 | body-array | no | systemone | reject: Invalid value | — |
| 9 | body-duplicate-questions | no | systemone | accept | ok, 1 question |
| 10 | body-many-questions | no | systemone | accept | generation skipped, 64 questions |
| 11 | body-invalid-utf8 | no | systemone | reject: Invalid JSON | — |
| 12 | unknown-target | no | (none) | reject: unknown jev/v1 route | — |

- No seed is dropped by the size caps and no seed is inert with respect to its name: the
  accept-side seeds reach `GenerateAnswers`, the reject-side seeds reach the validator,
  and the route seeds hit their routes.
- `body-many-questions` (#10) is the seed that exists to cover the
  `len(request.Questions) <= profileFuzzMaxQuestions` skip; the probe measures 64
  questions against the cap of 32, so the branch it names is the branch it takes.
- Neighbouring observation, not a defect: `body-duplicate-questions` (#9) carries two
  `"questions"` members, and `encoding/json` collapses them last-wins into a one-entry
  mapping, so the seed lands on the accept path. The profile decoder has no
  duplicate-key rejection to reach — `rejectDuplicateJSON` is a `Load`-only guard, and
  this file makes no claim that the seed trips it — so the seed is doing what a valid
  request seed should. Its name describes the input, and that is how it is covered.

### `internal/engine/race_test.go` — no fuzz corpus

The file declares no `f.Fuzz` target and no seed corpus, so there is nothing of this
class to audit. The concurrency tests are plain Go tests that start goroutines from one
closed channel and join them with one `sync.WaitGroup`. The one deliberately
non-firing assertion is annotated in the source: the `journalFullFailures > 1` count in
`TestConcurrentRegistrationSelectionJournalAndResetAreRaceFree` is envelope-only (128
capacity versus at most 72 admissions), and the comment names
`TestConcurrentTransitionsRetainExactlyTheAdmittedSequences` as the live journal-full
proof. That is a documented, intended limitation rather than an inert seed.

## Scope confirmation

- This pass edited exactly one path: `.agent/work/FJ-040/hardener.md` (this artifact).
- No file under `internal/` was created or modified by this pass. The four allowed test
  files are byte-identical to how this pass found them (`afb63fea…`, `932a8eec…`,
  `c757a234…`, `c72cf7e9…`).
- The audit results all came from scratch copies under `/private/tmp/fj040-h4/`; nothing
  was written inside the repository other than this artifact.
- Configuration/state files were not touched: `state.json` and `.agent/reports/` were
  left exactly as found.

## Command outcomes (this pass, all bounded)

| command | result |
| --- | --- |
| `go test ./internal/control -run FuzzControlMalformedInput -count=1` | exit 0, `ok fake-jev/internal/control 0.805s` — plain `go test` runs the seed corpus, so all 16 seeds including `repeated-key-bomb` execute |
| `gofmt -l internal/control/fuzz_test.go` | no output, exit 0 |
| `go vet ./internal/control` | exit 0, no diagnostics |
| live-route probe (scratch copy) | corrected 9,607 B valid JSON → 400 / 106 B duplicate-key error; malformed 9,608 B invalid JSON → 400 / 77 B generic |
| per-seed audit probe (scratch copy) | control 16/16 executed with distinct routing; config 18/18 handed to `Load`, none skipped; v1 13/13 executed, none skipped |
| `./scripts/candidate-fingerprint candidate` | `4d9e1ad756b11f6771e4e2d24cb495181300a120125791cad14551ca17ff5f5e` (unchanged before and after writing this file) |

Not run by this pass, per instruction: the full suite, the race suite, guard fuzz, and
`./scripts/verify-candidate` (owned by the PM stage). No file was staged
(`git diff --cached --stat` empty).

## Residual risks

- **The YAML duplicate-key quadratic remains in the product and is guarded around, not
  asserted against.** `configFuzzMaxMappingKeys` skips YAML-path documents above 256
  separators so the target does not ask for the multi-second, multi-GiB decodes that
  `qa.md` measured (17.4 s / 13.9 GiB at 13,100 entries). Inside the bound the cost per
  `Load` is milliseconds. The protection is a skip in a test, not a product bound, and
  nothing in the four files asserts a bound on YAML decoding cost.
- **Anchor/alias expansion is outside the guard.** Such documents can have almost no
  `:`/`,` separators, so the separator proxy cannot bound them; the effective limit is
  yaml.v3's own alias-ratio check (roughly 400k alias-driven decodes per document),
  which can still expand a few KB to tens of MB per `Load`. Unchanged by this pass.
- **The control-route duplicate-key class is now covered by the seed corpus again**, but
  the seed is the only deterministic corpus entry for it; fuzzer-generated inputs are
  the rest. If this literal is ever edited without re-measuring the body, it can regress
  to the inert malformed shape silently — the failure mode this whole item is about. The
  seed's own comment now states the body size and the `{`-count, which makes a future
  drift visible on inspection but is not enforced by any assertion on the literal.
- **`deep-brackets` is shape-named, not path-asserting.** It contributes generic
  invalid-body coverage only; if a future reader expected it to exercise a nesting-depth
  limit, it does not. No change made, because nothing claims otherwise.
- **The duplicate `"questions"` key in the v1 seed is silently collapsed** by
  `encoding/json` last-wins. That is the product's behaviour, not a test defect, but it
  means the seed is not a duplicate-key test in the FJ-065 sense and should not be read
  as one.
- **Fingerprint bookkeeping is now split across two stages.** `qa.md` pins `8961ce46…`,
  which no longer describes the workspace; the workspace is `4d9e1ad7…`. Any later
  stage that treats `qa.md`'s `outputFingerprint` as "the current revision" will be
  wrong. The value to use for the current revision is the one at the top of this file.
- **FJ-030 (unbounded `LoadFile`/`os.ReadFile`) and FJ-021 (post-reset
  `invalid_stub_response` recording race) remain open** and were not touched.
- This pass did not run the full suite, the race suite, guard fuzz, or
  `./scripts/verify-candidate`, by instruction; those are the PM stage's evidence, and
  nothing here should be read as substituting for them.

## Chain-link correction

Corrected field: `inputFingerprint` in this artifact's front matter.

- Before: `8961ce46838962ae48d5bf900aa2013b9008df3c595b84cfa6e72c614d7cc434`
- After: `525ee08845fbb9098ba37a5146877529863cddd57dcb263e9050628d63b9a84a`

The value previously present (`8961ce46…`) is the `outputFingerprint` of the earlier qa
pass (`.agent/work/FJ-040/qa.md`). It was recorded here because a later hardener pass was
instructed to source its input from that qa artifact rather than from `cleaner.md`. The
stage chain requires each stage after the first, in the canonical order
`specifier, coder, cleaner, hardener, qa`, to carry an `inputFingerprint` equal to the
immediately preceding stage's `outputFingerprint`; for the hardener stage the preceding
stage is the cleaner, and `cleaner.md` records
`inputFingerprint = outputFingerprint = 525ee08845fbb9098ba37a5146877529863cddd57dcb263e9050628d63b9a84a`.

This change is a front-matter correction only. No measurement, finding, table, claim,
command outcome or residual risk elsewhere in this artifact was altered; `outputFingerprint`,
`taskFingerprint`, `gitHead`, `generatedAt` and `author` are unchanged, and no existing body
text was modified.

This is the same class of artifact correction recorded for FJ-020.

Author of this correction: `worker/FJ-040-hardener-5` (a fresh hardener-role session),
2026-10-03.
