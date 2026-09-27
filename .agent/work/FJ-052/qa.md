---
stage: qa
task: FJ-052
inputFingerprint: 7122486b0b0f93a063e01a58d8ef96276b4f917e9fe93f2614e0ae234fa85b69
outputFingerprint: 7122486b0b0f93a063e01a58d8ef96276b4f917e9fe93f2614e0ae234fa85b69
taskFingerprint: f688e36458c7a2ba1571ca1f4371b29f3a3771234496b113c557bc0c2f73c9ad
gitHead: 7954187
generatedAt: 2026-09-27T12:31:18Z
author: subagent/qa-1
---

# QA — FJ-052 (independent acceptance review)

## Method

Reviewed commits `5ad5e1c` (docs), `b74a9f3` (feat), `7954187` (chore) against
the review brief's criteria C1–C7, cross-referenced to the item's own criteria
(`state.json.acceptance` 1–7 = `specifier.md:84–145` C1–C7 — the numbering in
this report follows the brief's topical split, mapped below).

Beyond reading, all probes were run in throwaway copies outside the workspace
(`/var/folders/…/T/opencode/fj052-canary`, `…/fj052-hist`): mutation of the
verifier, mutation of work items, and mutation testing of the new selftest
scenarios. No workspace file was edited; the workspace diff is `qa.md` only
(plus the report file the mandated `./scripts/verify-candidate FJ-052` run
inherently writes, and the pre-existing orchestrator flip — see Finding 4).

## Per-criterion findings

### C1 — status history (brief C1) — **supported**

- Edge table: `scripts/verify-candidate:331–337` `ALLOWED_EDGES` is **13**
  edges; parsed programmatically against design §8.4
  (`docs/superpowers/specs/2026-09-25-gauntlet-roles-design.md:433–464`) —
  sets are **identical** (`EQUAL: True`, `only design: []`, `only code: []`).
  The brief's "exactly 17" matches neither (Finding 2).
- Rules, all inside repo-scoped `check_work_item_schema`
  (`scripts/verify-candidate:375–396`): non-grandfathered ⇒ `statusHistory`
  present (`:379`), non-empty list (`:381`), vocabulary strings (`:383`),
  first entry ∈ {`planned`,`ready`} (`:387`), every consecutive pair in
  `ALLOWED_EDGES` (`:390` — no `(x,x)` exists, so consecutive duplicates fail),
  last entry == `status` (`:394`). Grandfathered skip guard at `:377`.
- Grandfathered set: `scripts/verify-candidate:329–330` (schema heredoc),
  `:422–423` (coverage heredoc), `:696–697` (task engine) — all three
  `{"FJ-001","FJ-002","FJ-003","FJ-004","FJ-043","FJ-048","FJ-053","FJ-PRD-001"}`
  = exactly the 8 design §8.6 ids (`design:482–491`).
- Seeding: `git show b74a9f3 --numstat -- .agent/work/` = 28 state.json at
  exactly `2/1` (each just `"notes": []` → `"notes": [],` +
  `"statusHistory": ["planned"]`, verified on FJ-010/FJ-016-style diffs) plus
  `.agent/work/FJ-052/state.json` at `10/3` (orchestrator flip) and the two
  artifacts = 31 file rows, 29 of them state.json. Untouched set
  (`ls .agent/work` − touched) is exactly the 8 grandfathered ids. All 28
  seeded items are `status: planned` with `statusHistory == ["planned"]`
  (checked programmatically); FJ-010 was created as `planned`
  (`git show b1b435b:…FJ-010/state.json`), so the seed is its true first entry.
- Hand-traced FJ-052's own history (`state.json:48–56`): first `planned`,
  pairs `planned→ready→implementing→verifying→implementing→verifying→implementing`
  all in the table, last `implementing` == `status`.
- Adversarial probes (mutated FJ-010 in a temp copy, battery run each time):
  missing / `["planned","planned"]` / `["implementing"]` first / last≠status /
  `blocked→implementing` / non-string entry / string-instead-of-list / `[]`
  ⇒ **all 8 failed** with the expected message and code
  `agent.work_item_schema`; `["planned","blocked"]` with `status: blocked`
  ⇒ pass; `statusHistory` given garbage on grandfathered FJ-001 ⇒ still pass
  (skip works), baseline restored to `15 passed`.

### C2 — recorded-report coverage (brief C2) — **supported**

- `check_recorded_report_coverage` at `scripts/verify-candidate:418–508`;
  code arm `policy.last_verification_coverage` at `:130`; wired unconditionally
  at `:510` (top level, before the gauntlet engine) ⇒ runs repo-scoped on the
  battery as well as task-scoped — observed in both.
- Reads the item's own `lastVerification.report` (`:475–495`); missing /
  unparseable / absent file each append a problem (`:478–493`); the report's
  own `policy.requiredStages` must be a non-empty list (`:431–433`); per stage
  requires pass rows `stage.<s>.artifact/.header/.task_fingerprint` (`:441–443`),
  `.chain` for index > 0 (`:444–445`), `.terminal` for the last index
  (`:446–447`); a non-`pass` result is reported (`:452–454`). Grandfathered
  complete items counted and skipped (`:470–472`).
- Canary: `.agent/reports/FJ-002/report-20260926T075318Z.json` has
  `requiredStages ["specifier","coder","cleaner","qa"]`, **zero** `stage.qa.*`
  rows and **zero** terminal rows. Patched a temp copy to drop `FJ-002` from
  that function's grandfathered set and ran the battery: **exit 1**,
  `FAIL (exit 1, code policy.last_verification_coverage)`, listing
  `missing report row stage.qa.{artifact,header,task_fingerprint,chain,terminal}`
  (5 rows) — the check is real, not ceremonial. Unpatched battery prints
  `no non-grandfathered complete items (8 complete item(s) grandfathered)` and
  PASS.
- Brief expectation "FJ-003/FJ-004 reports pass": **FJ-004 passes; FJ-003 does
  not** (its report has `requiredStages […, hardener, qa]` but no qa/terminal
  rows); FJ-048 also fails, FJ-001/FJ-043/FJ-053/FJ-PRD-001 pass. 3 of 8
  grandfathered reports fail the check — which is precisely what §8.6 exempts
  (Finding 3).

### C3 — author presence (brief C3) — **supported**

- Row emission `scripts/verify-candidate:1029–1041`: for a due stage of a
  non-grandfathered item, `stage.<s>.author`, gate `STAGE_GATES[stage]`
  (`:687–689` → G-S/G-C/G-L/G-H/G-Q), `required=True`, pass when non-empty,
  fail with `stage.evidence_invalid` and reason
  `missing or empty author in <path>` otherwise. Presence rows confirmed in
  the live report: `stage.specifier.author` G-S, `stage.coder.author` G-C,
  `stage.cleaner.author` G-L, all `required=true`.
- Parsing: `author` in `OPTIONAL_ARTIFACT_KEYS` (`:924`), accepted by the
  unknown-key filter (`:949–951`), and a present-but-empty value fails the
  header with `author must be non-empty` (`:965–967`) — probes: deleting the
  `author:` line from specifier.md ⇒ `stage.specifier.author` FAIL, header/
  task_fingerprint rows still PASS, run exit 1; `author:` with empty value ⇒
  header row FAIL with `author must be non-empty`, exit 1.
- Schema: `author` optional (not in `required`, `stage-artifact.schema.json:7–8`),
  `minLength: 1` with an accurate description (`stage-artifact.schema.json:18`).
- Grandfathering: task-scoped run on FJ-004 in a temp copy emits **no**
  `stage.*.author` and no `policy.author_overlap` rows (guard `:1031`/`:1067`).
- Nuance: the author row is suppressed when the artifact is missing or its
  header fails to parse (the `continue`s at `:1007`/`:1015` precede `:1031`),
  and the docstring at `:989–992` enumerates the suppressed rows without
  mentioning author. Fail-closed overall (artifact/header rows still fail) —
  Finding 1.

### C4 — qa ≠ coder (brief C4) — **supported**

- `scripts/verify-candidate:1065–1078`: `policy.author_overlap`, gate `G-C`,
  `required=True`, code `policy.author_overlap`; fails iff the two parsed
  values are equal (`:1070`), naming both (`:1077–1078`), and passes with
  `coder=… qa=…` when different; guarded by `TASK_ID not in GRANDFATHERED`
  and both stages in `parsed`.
- Probes (fixture artifacts in a temp copy): coder and qa both
  `author: subagent/coder-1` ⇒ `policy.author_overlap` FAIL
  (`coder author 'subagent/coder-1' equals qa author 'subagent/coder-1'`),
  presence rows and `stage.qa.terminal` PASS, exit 1; control with
  `author: subagent/qa-1` ⇒ overlap PASS, exit 0.

### C5 — schemas (brief C5) — **supported**

- `additionalProperties: false` intact on both:
  `.agent/schema/work-item.schema.json:9`, `.agent/schema/stage-artifact.schema.json:9`.
- New keys optional: `statusHistory` not in `required`
  (`work-item.schema.json:7–8`, property `:36–42`, `minItems: 1`, items enum
  = the 7 statuses); `author` not in `required`, `minLength: 1`
  (`stage-artifact.schema.json:18`).
- Descriptions accurate (both name the design section and say the verifier
  imposes presence). `python3 -m json.tool` exits 0 on both files.
- Design §11 note updated (`design:621–626`); §6.2 list fixed — `reviews`
  removed, `statusHistory` added (`design:269`), and no state.json contains
  `"reviews"`.

### C6 — selftest (brief C6) — **supported**

- Four new scenarios at `scripts/selftest:1380` (`history_missing_or_edge_violation`),
  `:1410` (`coverage_missing_terminal_row`), `:1427` (`author_missing_on_due_artifact`),
  `:1449` (`author_overlap_qa_equals_coder`); helpers `HISTORY_PATHS` `:206`,
  `write_fixture_report` `:283`, `attach_last_verification` `:315`;
  `base_item` seeds a compliant history (`:240–244`), `write_artifact` injects
  a per-stage author (`:249–272`).
- Totals: `@scenario(` count 52 (pre-commit `b74a9f3^`) → **56** now.
- No weakening: the entire deletion set of `git show b74a9f3 -- scripts/selftest`
  is the `write_artifact`/`write_chain` header-construction refactor (13 lines);
  **no pre-existing `expect`/`assert` was removed or relaxed**. The one
  pre-existing scenario touched, `exemption_stale_trackedby`
  (`scripts/selftest:857–879`), gained fixture compliance (complete history +
  compliant recorded report) and **two added** `result="pass"` expectations —
  a strengthening, not a loosening.
- Assertions are real: `expect()` raises when a row is absent
  (`scripts/selftest:179–194`), `assert_exit` compares return codes (`:197`).
  **Mutation testing** (temp copy, each check neutered, then selftest filter
  run): disabling the author row ⇒ `FAIL author_missing_on_due_artifact`;
  disabling the overlap row ⇒ `FAIL author_overlap_qa_equals_coder`;
  neutering the edge test ⇒ `FAIL history_missing_or_edge_violation`;
  forcing the coverage check to exit 0 ⇒ `FAIL coverage_missing_terminal_row`;
  pristine script ⇒ `4 passed`. The scenarios detect their checks, so they are
  not self-fulfilling.

### C7 — sequencing / chain (brief C7) — **supported**

- Preconditions: FJ-003 closed `8295626` 2026-09-26 15:54, FJ-004 closed
  `ddaeae6` 2026-09-26 16:45 — both before the design revision `5ad5e1c`
  (2026-09-27 16:24). Last pre-FJ-052 change to `scripts/selftest` is
  `eb2241c` 2026-09-26 10:03, before FJ-052 opened (`7dbf861` 13:44) ⇒ no
  pending reformat.
- Chain discipline (artifact headers): specifier `in=out=8b1b0a27…`
  (output recorded as coder's input), coder `in=8b1b0a27… out=7122486b…`,
  cleaner `in=out=7122486b…`; every `taskFingerprint=f688e364…` ==
  `./scripts/candidate-fingerprint task` live output; authors distinct per
  stage (`session/opencode-main`, `subagent/coder-1`, `subagent/cleaner-1`).
  Current candidate fingerprint `7122486b…` == cleaner's output ⇒ terminal
  freshness premise holds for the later qa/review flip.
- `.agent/reports/FJ-054/` exists untracked (1 file,
  `report-20260926T115633Z.json`) and `git diff --cached` is empty ⇒ nothing
  staged, FJ-054 not swept in.

## Independent runs (workspace, exit codes)

| command | result | exit |
|---|---|---|
| `./scripts/verify-candidate FJ-052` | `summary: 29 passed, 0 failed, 1 skipped, 4 not_applicable` / `RESULT: PASS` | 0 |
| `./scripts/verify-candidate` (battery) | `summary: 15 passed, 0 failed, 0 skipped, 0 not_applicable` / `RESULT: PASS`; coverage row PASS with `no non-grandfathered complete items (8 complete item(s) grandfathered)` | 0 |
| `./scripts/selftest` | `summary: 56 passed, 0 failed` (incl. the 4 new scenarios) | 0 |
| `bash -n scripts/verify-candidate` | no diagnostics | 0 |
| `bash -n scripts/agent-context` | no diagnostics | 0 |
| `python3 -m py_compile scripts/selftest` | no diagnostics (`__pycache__` removed immediately; selftest is Python, so `bash -n` on it reports token errors by design — cleaner.md:26–29) | 0 |
| `python3 -m json.tool .agent/schema/work-item.schema.json` | parses | 0 |
| `python3 -m json.tool .agent/schema/stage-artifact.schema.json` | parses | 0 |
| `./scripts/agent-context FJ-052` / `FJ-010` | packet renders (author-bearing artifact + statusHistory accepted) | 0 / 0 |

Note: the brief's single `bash -n a b c` invocation only checks the first
operand (bash takes one script file; the rest become positional parameters),
so the three scripts were checked individually as above.

## Cross-checks

- `git status --porcelain`: ` M .agent/work/FJ-052/state.json` (orchestrator's
  qa flip only — `stage`/`leaseOwner`/`nextAction`/`statusHistory`, 5+/3−,
  workflow-only fields per `specifier.md:152–161`), 5 untracked
  `.agent/reports/FJ-052/report-*.json` (4 pre-existing from coder/cleaner runs
  + 1 from my mandated task-scoped run), `?? .agent/reports/FJ-054/`. Nothing
  else dirty; no staged changes (Finding 4).
- `git ls-files .agent/work` = 77 files = 37 state.json (one per directory,
  all tracked) + 40 artifacts; untracked under `.agent/work/`: none.
- `git show b74a9f3 --name-only`: only `.agent/`, `scripts/` paths — no
  `cmd/`, `internal/`, `spec/`, `testdata/`; no grandfathered state.json in the
  commit (verified by set difference, untouched = exactly the 8 ids).
- `git show b74a9f3 -- scripts/verify-candidate | grep -c '^+'` = **173**
  (172 insertions + the `+++ b/…` header) — consistent with the brief's ~172.
- Trailing whitespace in added lines: 0 (checked with `[[:blank:]]$` over all
  three commits).
- Battery canary statement confirmed verbatim (see C2).

## Findings

1. **Low — `scripts/verify-candidate:1001–1041` (with `:965–967`,
   `:989–992`)** — the `stage.<s>.author` row is not emitted when a due
   artifact is absent or its header fails to parse (the `continue`s at
   `:1007`/`:1015` run before the author emission at `:1031`), and the
   empty-author case surfaces on `stage.<s>.header` (`author must be
   non-empty`) instead of on `stage.<s>.author`. Design §8.5
   (`design:466–474`) states a missing/empty author fails
   `stage.<s>.author`, and the `evidence_rows` docstring lists the suppressed
   rows without author. The run still fails closed in both paths (probes C and
   D), and the designed case — valid header, no `author` — produces the
   correct row; the gap is row naming/coverage in the compound-failure paths.
2. **Informational — review brief vs `design:433–464` / `verify-candidate:331–337`**
   — the brief asks for "exactly 17 allowed edges"; design §8.4 and the code
   both contain **13**, and the two sets were verified identical
   programmatically. The artifacts are consistent with each other; the
   brief's expected count is wrong.
3. **Informational — review brief vs `.agent/reports/FJ-003/report-20260926T102346Z.json`
   (and FJ-002, FJ-048)** — the brief expects "FJ-003/FJ-004 reports pass":
   FJ-004's report passes coverage, but **FJ-003's fails** (no
   `stage.qa.*` / terminal rows; FJ-048 likewise, FJ-002 by design). Only
   grandfathering keeps the battery green, which is exactly what §8.6 is for;
   no artifact defect, but the brief's expectation is inaccurate.
4. **Informational — workspace vs brief cross-check** — brief expects only
   `.agent/reports/FJ-054/` untracked; actual state also has the
   orchestrator's uncommitted FJ-052 qa flip in `.agent/work/FJ-052/state.json`
   and 5 untracked FJ-052 reports. Both are legitimate mid-flight evidence
   (the protocol reserves state edits to the orchestrator; the close commit
   scope is `.agent/work/FJ-052` + `.agent/reports/FJ-052`,
   `specifier.md:175`), but the expectation as written does not hold right now.
5. **Low (contract-sanctioned residual gap) — `scripts/verify-candidate:418–508`**
   — coverage validates rows only; it never reads the report's own `status`
   or the item's `lastVerification.status`. A report from a run that failed on
   a non-stage check could satisfy coverage if every required stage row passed.
   Design §7 (`design:324–335`) and acceptance 3 (`state.json:29`) specify
   exactly the row set, so this is in-contract, not a violation — noted for the
   gate's awareness.
6. **Informational — `scripts/verify-candidate:329`, `:422`, `:696`** — the
   `GRANDFATHERED` literal is duplicated in three python heredocs/interpreters.
   Verified byte-identical today, and the file already duplicates
   `GATES`/`STAGE_GATES` across heredoc boundaries, but three independent
   copies can drift silently.
7. **Informational (pre-existing, out of scope) — `scripts/verify-candidate:344–409`
   vs `.agent/schema/work-item.schema.json:9`** — `check_work_item_schema`
   still never rejects *unknown* state.json keys, so `additionalProperties:
   false` is enforced only for the fields the validator checks. Design §13
   (`design:696–697`) explicitly allows unexercised schema fields to drift,
   and FJ-052 neither introduced nor widened this; recorded because acceptance
   1's parenthetical mentions the schema forbidding `reviews`.

No criterion of the brief was found unsupported: every check was exercised at
least once in a failing direction (history ×8 mutations, coverage canary,
author presence, author overlap, four selftest mutation tests), and all three
mandated runs exit 0. These are observations; the review gate, not this
artifact, decides acceptability.
