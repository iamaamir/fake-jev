---
stage: coder
task: FJ-052
inputFingerprint: 8b1b0a271df495b0be18092c3321f19c671d7ef8b957193eb4ce16d69e698e43
outputFingerprint: 7122486b0b0f93a063e01a58d8ef96276b4f917e9fe93f2614e0ae234fa85b69
taskFingerprint: f688e36458c7a2ba1571ca1f4371b29f3a3771234496b113c557bc0c2f73c9ad
gitHead: 5ad5e1c
generatedAt: 2026-09-27T11:48:50Z
author: subagent/coder-1
---

# Coder — FJ-052

## What was changed, and why

**Schemas (acceptance C2).** `.agent/schema/work-item.schema.json` gained the
optional `statusHistory` property (non-empty array of status-vocabulary
strings); `.agent/schema/stage-artifact.schema.json` gained the optional
`author` (`minLength: 1`). Both files keep `additionalProperties: false`
and neither field was added to `required`, so grandfathered items (design
§8.6) and pre-revision artifacts stay valid (§11: the schema admits, the
verifier imposes).

**`scripts/verify-candidate` (acceptance C3).** Four additive changes:

1. *Author in artifact parsing*: module-level `OPTIONAL_ARTIFACT_KEYS =
   ("author",)` beside `ARTIFACT_KEYS`; unknown-key rejection now excludes it,
   and a present-but-empty value fails the header (`author must be non-empty`).
   Parsing still never *requires* it.
2. *Transition order (§8.4)*: inside `check_work_item_schema`'s per-item loop,
   with module-level `GRANDFATHERED` (the eight explicit §8.6 ids) and
   `ALLOWED_EDGES` (the §8.4 edge table). Every non-grandfathered item must
   carry a non-empty vocabulary `statusHistory` whose first entry is
   `planned`/`ready`, whose consecutive pairs are allowed edges (which also
   rejects consecutive duplicates), and whose last entry equals the current
   `status`. Violations print as per-item problems, e.g.
   `statusHistory pair 'implementing->complete' not an allowed transition`.
3. *Recorded-report coverage (§7)*: a new repo-scoped
   `check_recorded_report_coverage` (python3 heredoc, wired right after the
   schema check, code `policy.last_verification_coverage` added to
   `check_code`). For each non-grandfathered `complete` item it requires a
   `lastVerification.report` that exists and parses, whose own
   `policy.requiredStages` is a non-empty list, and which carries `pass` rows
   for every stage's `.artifact`/`.header`/`.task_fingerprint`, `.chain` after
   the first, and the last stage's `.terminal`. Nothing is recomputed — the
   report's recorded policy is the source of truth. No qualifying item yields
   a pass with an explanatory line (the current repository prints
   `no non-grandfathered complete items (8 complete item(s) grandfathered)`).
4. *Attribution (§8.5)*: in the task-scoped `evidence_rows()`, a due stage of
   a non-grandfathered item now emits `stage.<s>.author` (stage gate, required)
   — fail with `stage.evidence_invalid` when the parsed header has no non-empty
   `author`; and once due parsing is done, `policy.author_overlap` (G-C,
   required) fails when the parsed `qa` author equals the parsed `coder` author
   and names both values.

**`scripts/agent-context` (design §5 conformance, same change).** Its artifact
parser required an exact seven-key front-matter set, which would render every
post-revision (author-bearing) artifact — including this task's own specifier
artifact — `invalid` in the GAUNTLET view. It now accepts the same optional
`author` key as the engine (`OPTIONAL_ARTIFACT_KEYS`), still requiring every
mandatory key and rejecting unknown ones.

**Seeding (acceptance C5).** All 28 non-grandfathered, non-FJ-052 work items
(every one of them `status: planned`) gained `statusHistory: ["planned"]` —
their true first entry — inserted textually so each file's existing formatting
is otherwise byte-identical. FJ-052 (already carrying
`[planned, ready, implementing]`) and the eight grandfathered ids were left
untouched. Seeding and the repo-scoped history check landed in the same
change, so no intermediate battery run could go red.

**`scripts/selftest` (acceptance C4).** Four new fail-closed scenarios above
the `# --- runner ---` marker, one per §12 matrix row:

- `history_missing_or_edge_violation` — a non-grandfathered item with no
  `statusHistory`, and one recording `implementing -> complete` (with valid
  coverage attached so the history rule is the only defect): task-less verify
  exits non-zero, the pair message appears, and
  `policy.last_verification_coverage` does not.
- `coverage_missing_terminal_row` — a `complete` item whose recorded report
  omits `stage.qa.terminal`: fails with `policy.last_verification_coverage`
  naming the missing row, while `agent.work_item_schema` stays absent.
- `author_missing_on_due_artifact` — due specifier artifact stripped of
  `author`: `stage.specifier.author` fails (`stage.evidence_invalid`, G-S)
  while the header/task-fingerprint rows still pass.
- `author_overlap_qa_equals_coder` — coder and qa artifacts rewritten to the
  same author: `policy.author_overlap` fails (G-C, required) while both
  presence rows and the terminal row pass.

Fixture maintenance for the new rules (no assertion changed from fail to
pass, no scenario deleted or loosened): `base_item()` now seeds a compliant
`statusHistory` path ending at the item's status; `write_artifact()` headers
carry a per-stage `author` (so attribution rows are satisfied by ordinary
fixtures, and coder/qa authors never collide by accident); and the one
pre-existing `complete` fixture — `exemption_stale_trackedby`, which flips
FJ-044 to `complete` — now also records a compliant history plus a
`lastVerification`/report covering every required stage row, gaining two
`expect(... result="pass")` assertions (`check_work_item_schema`,
`check_recorded_report_coverage`) to pin that isolation. `write_chain()`
gained an `author` passthrough; new helpers `write_fixture_report()` and
`attach_last_verification()` build recorded reports for complete fixtures.

## Commands run and observed output

Run after every edit above, on this worktree (dirty by these edits):

- `./scripts/verify-candidate FJ-052` → `summary: 19 passed, 0 failed,
  0 skipped, 4 not_applicable`, `RESULT: PASS`, exit 0; the new row
  `stage specifier author` and the coverage check both reported PASS.
- `./scripts/verify-candidate` (battery, no task id) → `summary: 15 passed,
  0 failed, 0 skipped, 0 not_applicable`, `RESULT: PASS`, exit 0.
- `./scripts/selftest` → `summary: 56 passed, 0 failed`, exit 0 (the suite
  had 52 scenarios before this stage; the four new rows are included).

`bash -n` on `scripts/verify-candidate` and `scripts/agent-context` and
`py_compile` on `scripts/selftest` produced no diagnostics. The candidate
fingerprint after all workspace edits (schemas + scripts) is
`7122486b0b0f93a063e01a58d8ef96276b4f917e9fe93f2614e0ae234fa85b69`,
recorded as this artifact's `outputFingerprint`; evidence directories are
excluded from it (§6.1), so the runs above did not move it.

## Notes for later stages

- Sequencing hazard C7 (a) held: no task-scoped verify ran between the
  `implementing` flip and the first edit landing `author` acceptance — the
  first task-scoped run occurred after every verifier/schema change.
- The coverage check reads `policy.requiredStages` from the recorded report
  itself; the FJ-002 canary (`report-20260926T075318Z`, lacking qa/terminal
  rows) is not exercised by any current fixture — it stays green only via
  grandfathering, as designed.
- `scripts/agent-context` is an edit outside the literal deliverable list but
  inside `allowedFiles`; without it, design §5 artifacts render `invalid` in
  the display tool and two existing selftest scenarios (`gauntlet_*`) would
  break on author-bearing fixture headers.
