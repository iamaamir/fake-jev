---
stage: specifier
task: FJ-052
inputFingerprint: 8b1b0a271df495b0be18092c3321f19c671d7ef8b957193eb4ce16d69e698e43
outputFingerprint: 8b1b0a271df495b0be18092c3321f19c671d7ef8b957193eb4ce16d69e698e43
taskFingerprint: f688e36458c7a2ba1571ca1f4371b29f3a3771234496b113c557bc0c2f73c9ad
gitHead: 5ad5e1c
generatedAt: 2026-09-27T11:22:32Z
author: session/opencode-main
---

# Specifier — FJ-052

## What is delivered

The full hardening package approved by the repository owner, in the order the
acceptance demands:

1. **Design revision, already committed** (`5ad5e1c`, `docs:`): the gauntlet
   design gained §5 `author` header field, §7 recorded-report coverage,
   §8.4 status transition history, §8.5 author attribution, §8.6
   grandfathering, schema notes in §11, four §12 self-test matrix rows, the
   §6.2 non-semantic list fix (`reviews` removed, `statusHistory` added), and
   the §14 non-goal clarification. `docs/development/agent-workflow.md` lost
   the single-worker sentence; it now mandates distinct workers per stage with
   the `qa`/`coder` author floor. This satisfies acceptance item 1.

   Remaining deliverables, in execution order:

2. **Schemas** (`.agent/schema/work-item.schema.json`,
   `.agent/schema/stage-artifact.schema.json`): add optional `statusHistory`
   (array of status strings) and optional `author` (non-empty string), keeping
   `additionalProperties: false`.
3. **`scripts/verify-candidate`**: (a) recorded-report coverage inside the
   repo-scoped `check_work_item_schema`; (b) transition-order rules from
   design §8.4 in the same check; (c) `author` presence on due artifacts and
   `qa != coder` inequality in the task-scoped evidence path; `author` joins
   `ARTIFACT_KEYS`; explicit `GRANDFATHERED` set (design §8.6 — all eight
   completes) exempts those items from all three.
4. **Seeding**: every non-grandfathered item gains
   `statusHistory: ["planned"]` in the same change as the check (all are
   `planned`; the first-entry rule admits `planned`). FJ-052's own history is
   seeded with this stage and appended on every subsequent flip.
5. **`scripts/selftest`**: one fail-closed scenario per new check, per the
   four new §12 matrix rows (history, coverage, author presence,
   `qa == coder`).

This stage changes no file outside `.agent/work/FJ-052/`, which the candidate
fingerprint excludes, so input and output candidate fingerprints are
identical (both re-derived with `./scripts/candidate-fingerprint candidate .`).

## Class

`allowedFiles` is `.agent/`, `docs/`, `scripts/` — all `METADATA_PREFIXES`
(§8.2), so the class is **`metadata`** and `derive_required_stages` returns
**`specifier, coder, cleaner, qa`** — no hardener (this item ships no product
code; non-goal 1). Verified live: `candidate-fingerprint task` reports
`class=metadata`, `source=derived`.

## Preconditions (acceptance item 6)

Both hold as of this stage:

- **FJ-003 and FJ-004 are complete** — closed by `8295626` (15:54) and
  `ddaeae6` (16:45) on 2026-09-26, before the design revision `5ad5e1c`.
- **The in-flight selftest reformat has landed** — no reformat is pending:
  the last change to `scripts/selftest` is `eb2241c` (10:03), before FJ-052
  was even opened (13:44), and no open work item claims one.

## Traces

None. The acceptance catalog (`docs/development/acceptance-catalog.md`,
108 ids across C-MATCH…C-QUAL) enumerates fake-jev product behavior; every
line of this item's acceptance is process enforcement over `state.json`,
artifact headers, and the verifier itself. Claiming an unrelated `C-QUAL`
id would assert product behavior this item does not touch. The verifier's
empty-Traces rule (no ids = no claims) applies.

## Criteria

Restated from `state.json.acceptance` in order, each observable. Ids are
stable — later stages refer to these.

**C1 — design revision committed first, covering all five named topics.**
Satisfied by `5ad5e1c`: status transition history (§8.4), per-stage
attribution (§5, §8.5), `qa != coder` rule (§8.5), workflow norm change
(`agent-workflow.md`: distinct worker per stage, single-worker multi-role
coverage ended), `reviews` resolution (§6.2 list). Falsifiable by
`git show 5ad5e1c`.

**C2 — schemas updated, additionalProperties rules preserved.** Work-item
schema gains optional `statusHistory`; stage-artifact schema gains optional
`author`; both remain `additionalProperties: false` with the new fields in
`properties`. Optional-in-schema, verification-required where design §7/§8.4/
§8.5 say (design §11). Falsifiable by reading both schema files.

**C3 — verify-candidate enforces (a), (b), (c).**

- (a) *lastVerification coverage*: for a `complete`, non-grandfathered item,
  the recorded report must contain pass rows for `.artifact`, `.header`,
  `.task_fingerprint` of every stage in its own `policy.requiredStages`,
  `.chain` for each such stage after the first, and `.terminal` for the last
  (design §7). Missing/failing rows, or missing/unparseable
  `lastVerification`, fail with `policy.last_verification_coverage`.
  Acceptance names the canary: FJ-002's pre-flip
  `report-20260926T075318Z` lacks qa and terminal rows and must fail this
  check (it stays green only via grandfathering).
- (b) *transition order*: design §8.4 edge table, first entry `planned` or
  `ready`, last entry equals current `status`, vocabulary strings, no
  consecutive duplicates — enforced in `check_work_item_schema`, which runs
  repo-scoped on every battery invocation, for every non-grandfathered item.
- (c) *attribution*: due artifacts of non-grandfathered items must carry a
  non-empty `author` (presence row `stage.<s>.author`); with both `coder` and
  `qa` artifacts parsed, `qa.author != coder.author` or
  `policy.author_overlap` fails. `author` joins `ARTIFACT_KEYS` so unknown-key
  rejection covers it.

**C4 — selftest fail-closed scenario per new check.** Four new §12 rows:
missing/edge-violating `statusHistory` fails; complete item whose recorded
report lacks required stage rows fails; missing author on a due artifact
fails; `qa` author equal to `coder` author fails. Each demonstrated in
`scripts/selftest` with a fixture that exits non-zero — not merely asserted
in prose.

**C5 — grandfathering for pre-revision completions, battery stays green.**
Explicit set, design §8.6: `FJ-001, FJ-002, FJ-003, FJ-004, FJ-043, FJ-048,
FJ-053, FJ-PRD-001` — all eight items complete before `5ad5e1c` (the
acceptance enumerates five because it was written at 13:44, when only those
five had closed; FJ-003/004/053 closed at 15:54/16:45/17:26 the same day and
meet the same condition). Retrofitting their history from git is impossible
without fabrication: their recorded paths include `ready -> complete` and
`implementing -> complete`, exactly the skips (b) exists to reject. The list
is explicit, not timestamp-derived, so the check stays deterministic.

**C6 — preconditions hold.** See Preconditions above; both verified true at
this stage.

**C7 — `./scripts/verify-candidate` and `./scripts/selftest` exit 0 after
every stage.** Falsifiable by running both after each flip. Two sequencing
hazards the coder must respect:

- The specifier artifact (this file) already carries `author`. It is parsed
  only from `implementing(coder)` onward (due = `[specifier]`), and the
  coder's *first* change lands `author` in `ARTIFACT_KEYS` plus both schemas,
  so no verify run parses an artifact whose header the verifier rejects.
  Do not run a task-scoped verify between the `implementing` flip and the
  coder's first edit.
- Seeding other items' `statusHistory` and the new checks must land in one
  change: the history check is repo-scoped, so a check without seeding turns
  the battery red.

## Execution protocol (binding on every stage worker)

The design revision mandates distinct workers per stage (workflow norm).
Execution model: **this session (`session/opencode-main`) performs the
specifier stage as orchestrator; coder, cleaner, and qa stages are each
dispatched to a distinct subagent** with author labels `subagent/coder-1`,
`subagent/cleaner-1`, `subagent/qa-1`. `session/opencode-main` orchestrates
status flips, appends `statusHistory`, and manages `leaseOwner` — state edits
are workflow-only and never touch task-semantic fields (that would stale
every artifact's task fingerprint, §6.2).

Stage sequence with due-sets (design §8.1, `candidate-fingerprint` output):

```text
planned   (seeded history ["planned"])     due = []        specifier written here
ready                                  due = []
implementing  stage=coder  lease=subagent/coder-1          due = [specifier]
verifying     stage=coder                due = [specifier, coder]
implementing  stage=cleaner lease=subagent/cleaner-1       due = [specifier, coder]
verifying     stage=cleaner              due = [specifier, coder, cleaner]
implementing  stage=qa     lease=subagent/qa-1             due = [S, C, L]
verifying     stage=qa                   due = [S, C, L, qa]
review        stage=qa                   due = all  -> R1, persist lastVerification
complete      stage=qa                   -> R2, close commit (only .agent/work/FJ-052 + .agent/reports/FJ-052)
```

Chain discipline: `coder.inputFingerprint` equals this artifact's
`outputFingerprint` (`8b1b0a27…`); each later stage's input equals the prior
stage's output, computed after that stage's last workspace edit (excluding
`.agent/work/`, `.agent/reports/`, `.agent/logs/`, `.scratch/`). After the
cleaner, no workspace file outside the excluded prefixes may change — the qa
stage's terminal freshness requires the last required stage's
`outputFingerprint` to equal the then-current candidate fingerprint. Note
`.agent/schema/` is **not** excluded: schema edits count as workspace content.

## Non-goals

Echoing `state.json.nonGoals`:

- **No fake-jev product code.** `cmd/`, `internal/`, `spec/`, `testdata/`
  are outside `allowedFiles`; a coder reaching for them has left scope.
- **No soft self-attested markers.** `assignedRole` nudges and lease-handoff
  theater were explicitly rejected by the owner; only the artifact-header
  attribution above counts.
- **No engine or gate changes.** G-C…G-Q invent no new gate ids; every new
  check belongs to an existing gate (`check_work_item_schema` → G-C policy
  rows; evidence rows → each stage's own gate), exactly as §7's "no new gate
  id" rule requires.

## Open questions

None. The ambiguities that existed were resolved with evidence, not
invention:

- *Grandfather count* (enumeration of 5 vs condition of 8): timeline in C5 —
  the condition governs; a literal five-item list turns the battery red
  because the history check is repo-scoped.
- *Coverage emission* (per-stage terminal rows vs design-applicable rows):
  design-applicable rows — no change to frozen §7 emission; the check reads
  what a completed run must have produced, and FJ-002's pre-flip report still
  fails it.
- *History check location*: inside `check_work_item_schema`, where status,
  vocabulary, and lease rules already live and where the battery already
  reaches every item.
- *Selftest reformat referent*: resolved in Preconditions — nothing pending.
