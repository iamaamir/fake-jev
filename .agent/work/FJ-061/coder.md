---
stage: coder
task: FJ-061
inputFingerprint: 21a64a50bf2ef95509f639bef1bdb0c944680121c8a9ae720998bec119f504dc
outputFingerprint: 050f8a89923dd23b42190e976b6db7fa5b1327118ac6fcca2eb1ab4247e0a4a2
taskFingerprint: fba7216e76c708718baf8fca335a13ca9459e0a78927de751c9f07183d538a9b
gitHead: 9aef065
generatedAt: 2026-10-02T16:13:44Z
author: worker/FJ-061-coder
---

# Coder — FJ-061

**Final candidate fingerprint: `050f8a89923dd23b42190e976b6db7fa5b1327118ac6fcca2eb1ab4247e0a4a2`**

## What changed

`scripts/selftest` only, additive: `+30 -0` (`git diff --stat` → `1 file changed,
30 insertions(+)`). No assertion, scenario, or other file touched.

1. New helper `copy_referenced_report(root, state_source)` (inserted above
   `build_pristine`): reads a copied work item's `state.json`, takes
   `lastVerification.report` when it is a non-empty string, and copies that one
   path from the real repository into the fixture. It returns without copying
   when the report is absent/empty, when the file does not exist, or when
   `git ls-files --error-unmatch -- <path>` in the real repository says the
   path is untracked.
2. `build_pristine()` calls it immediately after copying each embedded
   `state.json`, inside the existing id loop, before `git_init()` — so the
   report is committed by `git add -A` and the fixture worktree stays clean.

Nothing else changed: no new import, no dependency, no network, no sleep, no
wall-clock read, no change to `verify-candidate`, `candidate-fingerprint`,
`gate-policy.json`, or any real `.agent` content.

## Why this is the right layer

The defect is in fixture construction, not in the check. `verify-candidate`'s
`check_recorded_report_coverage` is correct and repository-scoped: a complete
item's own recorded report must show the evidence the completion rests on. The
fixture was building a `.agent` view that cannot exist in any checkout — a real
`state.json` with its bound report missing — so every scenario embedding
`FJ-047` failed on a repository-integrity row unrelated to what the scenario
tests. Fixing the fixture restores a real invariant; post-processing the report
row, grandfathering the id, or pre-empting the copy set (specifier option B)
would instead have hidden a class of future fixture/state drift.

The report set is derived from the `state.json` files already being copied, so
a fifth embedded item — or another item completing with a bound report — cannot
reintroduce the gap; this is the property the hardcoded path list version would
have lost. Referenced-only copying (not whole report directories) was chosen per
the specifier's recommendation O3: whole-directory copying imports unreferenced
evidence and re-creates a second list to keep in sync.

**Deviation from the specifier artifact, recorded:** the specifier's O2
recommends plain copy-if-present (mirroring the existing `source.is_file()`
guard). The task packet additionally requires the fixture not to depend on
machine state, so the copy is gated on trackedness in the real index: only
committed reports travel into the fixture, never bytes that happen to sit in
someone's worktree. Verified by probe (below): an untracked-but-present report
is not copied, a tracked one is, both while the suite stays deterministic. Cost
is one offline `git ls-files` per embedded item, inside a builder that already
runs repository git commands.

## Before / after

Baseline at `9aef065` (reproduced before editing):

```text
$ ./scripts/selftest
...
FAIL fuzz_zero_targets
FAIL fuzz_stage_na
PASS fuzz_crash
FAIL fuzz_live_pass
summary: 49 passed, 26 failed      # exit 1
```

Failing fixtures printed, under `==> complete items have recorded-report coverage`:

```text
.agent/work/FJ-047/state.json:
  - recorded report .agent/reports/FJ-047/report-20261002T152756Z.json does not exist
    FAIL (exit 1, code policy.last_verification_coverage)
```

After the change:

```text
$ ./scripts/selftest
...
PASS race_metadata_na
PASS fuzz_zero_targets
PASS fuzz_stage_na
PASS fuzz_crash
PASS fuzz_live_pass
summary: 75 passed, 0 failed      # exit 0
```

All 75 scenarios still run; none renamed, deleted, or weakened.

## Exact commands and results

| command | result |
|---|---|
| `./scripts/selftest` (before) | exit 1, `summary: 49 passed, 26 failed` |
| `./scripts/selftest` (after) | exit 0, `summary: 75 passed, 0 failed` |
| `./scripts/selftest packs_frontmatter_valid` (after) | exit 0, `summary: 1 passed, 0 failed` |
| `./scripts/selftest packs_frontmatter_valid` (before) | exit 1, `AssertionError`: `recorded report ... does not exist`, `FAIL (exit 1, code policy.last_verification_coverage)` |
| `./scripts/selftest coverage_missing_terminal_row` | exit 0, `summary: 1 passed, 0 failed` — the coverage rule's negative control still fires (`policy.last_verification_coverage`, `missing report row stage.qa.terminal`) |
| `./scripts/verify-candidate` | exit 0, `summary: 20 passed, 0 failed, 0 skipped, 0 not_applicable`, `RESULT: PASS` |
| `git diff --exit-code 9aef065 -- scripts/verify-candidate` | exit 0, no output (byte-identical) |
| `python3 -c "compile(open('scripts/selftest').read(),'scripts/selftest','exec')"` | `syntax ok` |
| `git diff 9aef065 -- scripts/selftest \| grep -E '^-[^-]' \| grep -E 'assert \|@scenario'` | no output (nothing removed or loosened) |
| `grep -c '^@scenario' scripts/selftest` | `75` |
| `grep -nE 'time\.sleep\|socket\.\|urllib\|requests\.' scripts/selftest` | no match (exit 1) |
| `./scripts/candidate-fingerprint candidate` | `050f8a89923dd23b42190e976b6db7fa5b1327118ac6fcca2eb1ab4247e0a4a2` |
| `git status --porcelain` before and after the suite | identical: ` M .agent/work/FJ-061/state.json`, ` M scripts/selftest`, `?? .agent/work/FJ-061/specifier.md` — the suite never touches the real repository |

Closure check (specifier command #5, run inline against the module's own
`build_pristine`):

```text
FJ-047 .agent/reports/FJ-047/report-20261002T152756Z.json True
git status --porcelain of the pristine fixture: ''
```

Hermeticity probes against `copy_referenced_report` (temp state files, temp
roots; probe file later removed and the tree re-checked clean):

```text
untracked (ignored) report -> copied: False
tracked report             -> copied: True
absent report              -> copied: False
empty report path          -> copied: False
```

## Observations

- `bash -n scripts/selftest` is not a meaningful check for this file and fails
  at baseline exactly as it does after the change: `scripts/selftest: line 23:
  syntax error near unexpected token '('` / `REAL = pathlib.Path(...)`. It is a
  Python program (`#!/usr/bin/env python3`), not shell; the same command against
  `git show 9aef065:scripts/selftest` exits 2 with the identical diagnostic. The
  equivalent check used here is the `compile(...)` line above (specifier command
  #11).
- The trackedness gate means that if a future item completes while its report is
  still untracked, the fixture skips that report and its embedding scenarios go
  red again — loudly, with `policy.last_verification_coverage`, and only for a
  repository that is itself inconsistent. Failing loudly was chosen over
  embedding machine-local bytes.
- Only 1 of the 4 embedded items (`FJ-047`) has a `lastVerification.report`, so
  the change copies exactly one file (14188 bytes) per fixture today; the copy
  cost grows with the number of completing embedded items (specifier O4).
- The copied report carries fingerprints and a revision bound to the real
  revision, not the fixture's (specifier O5). Nothing recomputes them: the
  coverage check reads rows only, and fixtures still report their own
  `revision=...`. The copied bytes are excluded from the candidate digest
  (`.agent/reports/` is in `candidate-fingerprint`'s `EXCLUDED_PREFIXES`), so the
  fixture's fingerprint is unaffected.
- The embedded id list (`FJ-044…FJ-047`) stays hardcoded; only the per-item
  report closure is now derived (specifier O7).

## Scope conflict left open for the parent

The specifier artifact's C4 "Exercise" recommends *adding* a scenario such as
`pristine_embeds_referenced_reports`; the task packet requires the suite to end
with exactly `75 passed, 0 failed`. Adding it would end at `76 passed, 0 failed`.
The supervisor was asked to choose and did not answer; the default stated in that
request was followed: **no scenario added**, keeping the packet's literal
criterion. Regression coverage of the closure is not lost — `packs_frontmatter_valid`
plus ~25 other fixture scenarios run `verify-candidate` on fixtures embedding
`FJ-047` and assert exit 0, so breaking the closure turns them red, which is how
this defect was found. If the parent prefers the named scenario, it is a
5-scenario-count change on top of this diff and nothing else.

## Residual risks

- Fixture fidelity now depends on the real repository's *index* (`git ls-files`).
  A checkout without a valid git metadata directory would skip report copies;
  `build_pristine` already depends on git for its own `git_init`, so this adds no
  new environment requirement.
- No assertion was added inside the fixture builder itself: a fixture-level
  closure failure still surfaces as the coverage row in the affected scenarios
  rather than as an explicit "fixture incomplete" message.
- `candidate-fingerprint candidate 9aef065` errors with `git metadata unavailable
  for 9aef065` (exit 4) on this machine; it is unrelated to this change, was not
  required by the task, and the pre/post fingerprints above are both workspace
  digests of the current revision.
