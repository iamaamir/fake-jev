---
stage: cleaner
task: FJ-061
inputFingerprint: 050f8a89923dd23b42190e976b6db7fa5b1327118ac6fcca2eb1ab4247e0a4a2
outputFingerprint: 050f8a89923dd23b42190e976b6db7fa5b1327118ac6fcca2eb1ab4247e0a4a2
taskFingerprint: fba7216e76c708718baf8fca335a13ca9459e0a78927de751c9f07183d538a9b
gitHead: 9aef065
generatedAt: 2026-10-02T16:19:50Z
author: worker/FJ-061-cleaner
---

# Cleaner — FJ-061

Candidate reviewed: `050f8a89923dd23b42190e976b6db7fa5b1327118ac6fcca2eb1ab4247e0a4a2`
(`./scripts/candidate-fingerprint candidate` prints this value; it matches the
coder artifact's `outputFingerprint`, used verbatim as this artifact's
`inputFingerprint` and `outputFingerprint`).

## Cleanup applied: none

No structural change was made. The diff is `scripts/selftest` only, `+30 -0`
(`1 file changed, 30 insertions(+)`): one new function `copy_referenced_report`
plus one call site inside `build_pristine`'s existing embedded-item loop. There
is no duplication to fold, no dead code, no layering violation, and no
naming drift that the specification constrains, so no edit met the "clearly
justified and strictly behaviour-preserving" bar. Editing for the two nits
below would change the candidate fingerprint and stale the coder artifact after
the fact, for no behavioural gain; both are recorded instead.

## Structure read

- `copy_referenced_report(root, state_source)` is defined once, called once
  (`scripts/selftest:146`), and reuses the module's existing `sh()`,
  `REAL`, `json`, and `shutil` — no new import, helper, or indirection. It
  matches the surrounding fixture-build style (`source.is_file()` guards,
  `shutil.copy2`, `dest.parent.mkdir(parents=True, exist_ok=True)` as in
  `build_pristine`'s `COPY_FILES` loop).
- No second source of truth was introduced: the report set is derived from the
  `state.json` files already being copied, so the embedded id list
  (`FJ-044…FJ-047`) stays the only hardcoded list, per specifier O7.
- The call sits inside the existing loop, before `git_init()`, so `git add -A`
  commits the report. A probe against the module's own `build_pristine` shows
  the fixture worktree clean (`git status --porcelain` in the pristine tree is
  empty) and the embedded report byte-equal to the tracked file.
- `.agent/reports/` is in `candidate-fingerprint`'s `EXCLUDED_PREFIXES`
  (`scripts/candidate-fingerprint:32`); measured directly, the pristine
  fixture's candidate digest is `7fb67bbbdf9d33c4b21f0f223b633cddbb800b3ddad877aa7bd797305917bb2c`
  both with and without the copied report, i.e. embedding it perturbs no
  fixture digest.

## Absent referenced report: cannot become a false pass

Three independent observations, all run here on the candidate tree:

1. `copy_referenced_report` never writes a substitute. Probes with a temp state
   file whose `lastVerification.report` names a tracked file, an
   untracked-but-present worktree file (`.agent/work/FJ-061/coder.md`), an
   absent path, an absolute path (`/etc/hosts`), `null`, `"  "`, a missing
   `lastVerification`, a non-dict `lastVerification`, and a non-dict state:
   only the tracked case copied anything; every other case wrote nothing.
2. If the real repository's report is genuinely missing, the fixture simply
   does not contain it, and the same rule that caught this item then fires
   inside the fixture: a pristine fixture with `.agent/reports/FJ-047` removed
   exits `1` from `./scripts/verify-candidate` with
   `policy.last_verification_coverage` and `... does not exist`. Independently,
   a fresh `complete` item whose report file is deleted exits `1` with the same
   code and message. Absence is loud, never silent.
3. `scripts/verify-candidate` is byte-identical to `9aef065`, so repository-level
   enforcement of a real missing report is untouched and fail-closed.

The only skip direction that can pass is the tracked case, and in that case the
bytes copied are the tracked bytes.

## Hermeticity read

- Independence from untracked local files: the copy is gated on
  `git ls-files --error-unmatch -- <path>` in the real repository, so a report
  that exists only in someone's worktree is not embedded (probe: untracked
  `.agent/work/FJ-061/coder.md` present in the worktree and absent from the
  index → not copied). The referenced report
  `.agent/reports/FJ-047/report-20261002T152756Z.json` is tracked
  (`git ls-files --error-unmatch` exits 0) and byte-equal after copying.
- No new environment dependency beyond the existing `sh()`/`GIT_HERMETIC_ENV`
  path; no wall-clock read, no network, no sleep
  (`grep -nE 'time\.sleep|socket\.|urllib|requests\.' scripts/selftest` exits 1,
  as at baseline). `git status --porcelain` in the real repository is identical
  before and after a full suite run.
- The gate is not a second enforcement point: it only selects what travels into
  a fixture; it never decides whether a missing report is tolerated, because the
  fixture-side coverage row requires the file's presence (observation 2 above).

## Assertion and scenario integrity

- Diff is additive: `git diff 9aef065 -- scripts/selftest | grep -E '^-[^-]'`
  exits 1 with no output, and none of the diff matches `assert |@scenario`.
- `grep -c '^@scenario' scripts/selftest` prints `75` (baseline count, no
  scenario added, removed, renamed, or filtered).
- The coverage rule's negative control still fires:
  `./scripts/selftest coverage_missing_terminal_row` exits 0 with
  `summary: 1 passed, 0 failed`.

## Comment accuracy

The docstring's two claims line up with observed behaviour: design §7 is
"Evidence chain and invalidation" and is the section the
`policy.last_verification_coverage` row cites (design §12, line 687), and the
trackedness sentence describes exactly the gate probed above. Two wording
notes, neither changed: the sentence "the fixture is built from committed
content" is scoped to the reports this function copies — `build_pristine` has
always read the embedded `state.json` files from the worktree, which this diff
does not alter — and `report.strip()` is evaluated twice (the second call stores
what the first checked). Both are cosmetic; no behaviour hinges on either.

## Command outcomes (recorded, as required)

| command | outcome |
|---|---|
| `bash -n scripts/selftest` | exit 2: `scripts/selftest: line 23: syntax error near unexpected token '('` / `REAL = pathlib.Path(__file__).resolve().parent.parent`. Not a regression: `git show 9aef065:scripts/selftest \| bash -n /dev/stdin` reproduces the identical diagnostic with exit 2. The file is Python (`#!/usr/bin/env python3`), so the shell parser is the wrong tool; the equivalent check `python3 -c "compile(open('scripts/selftest').read(),'scripts/selftest','exec')"` prints `syntax ok`. |
| `./scripts/selftest` | exit 0, `summary: 75 passed, 0 failed`, zero `FAIL` lines (baseline `9aef065`: exit 1, `49 passed, 26 failed`) |
| `./scripts/selftest packs_frontmatter_valid` | exit 0, `summary: 1 passed, 0 failed` |
| `./scripts/selftest coverage_missing_terminal_row` | exit 0, `summary: 1 passed, 0 failed` |
| `./scripts/verify-candidate` | exit 0, `summary: 20 passed, 0 failed, 0 skipped, 0 not_applicable`, `RESULT: PASS` |
| `./scripts/candidate-fingerprint candidate` | exit 0, `050f8a89923dd23b42190e976b6db7fa5b1327118ac6fcca2eb1ab4247e0a4a2` (equals the front-matter value) |
| `git diff --exit-code 9aef065 -- scripts/verify-candidate` | exit 0, no output |
| `git diff 9aef065 -- scripts/selftest \| grep -E '^-[^-]'` | exit 1, no output (additive only) |
| `grep -c '^@scenario' scripts/selftest` | `75` |
| `git ls-files --error-unmatch .agent/reports/FJ-047/report-20261002T152756Z.json` | exit 0, prints the path |
| `git status --porcelain` before/after a full suite run | identical (` M .agent/work/FJ-061/state.json`, ` M scripts/selftest`, untracked `specifier.md`/`coder.md`) |
| probes (`/tmp`, not in the repository) | tracked report copied + byte-equal; untracked/absent/absolute/null/empty/malformed cases copied nothing; pristine worktree clean; fixture digest unchanged with or without the report; pristine-minus-report and deleted-report fixtures both exit 1 with `policy.last_verification_coverage` |

## Residual risks and observations

- Trackedness gate direction: a future item that completes while its report is
  still untracked makes its embedding fixtures red (`policy.last_verification_coverage`)
  while the real repository's own run may still pass, because the real check
  tests readability, not index membership. Failure direction is spurious-red, not
  false-pass.
- A checkout whose git metadata is unusable skips every report copy, so the
  embedding fixtures go red; `build_pristine` already runs `git init`/`commit`
  for every fixture, so this adds no new environment requirement.
- The report string is interpolated into a `shell=True` command
  (`git ls-files --error-unmatch -- %s`). A path containing shell
  metacharacters would be misparsed or executed; the value comes from a work-item
  `state.json`, i.e. repository content an author already controls, and no such
  path exists today. Recorded, not changed (out of this item's scope, and any fix
  would have to touch `sh()`'s shell-string interface).
- No assertion lives inside the builder itself, so a fixture-level closure
  failure still surfaces as the coverage row inside the affected scenarios
  rather than as an explicit "fixture incomplete" message; the scenarios that
  caught this defect are the ones that would catch its return.
- The copied evidence is stale by construction (its fingerprints and revision are
  bound to the real revision); nothing recomputes them (specifier O5).
- Cost grows with each completing embedded item: today exactly one report
  (14188 bytes) per fixture (specifier O4).
- `.agent/work/FJ-061/state.json` and `scripts/selftest` are unstaged working-tree
  modifications; nothing is staged.
