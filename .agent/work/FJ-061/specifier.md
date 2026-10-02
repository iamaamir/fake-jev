---
stage: specifier
task: FJ-061
inputFingerprint: 21a64a50bf2ef95509f639bef1bdb0c944680121c8a9ae720998bec119f504dc
outputFingerprint: 21a64a50bf2ef95509f639bef1bdb0c944680121c8a9ae720998bec119f504dc
taskFingerprint: fba7216e76c708718baf8fca335a13ca9459e0a78927de751c9f07183d538a9b
gitHead: 9aef065
generatedAt: 2026-10-02T15:56:01Z
author: worker/FJ-061-specifier
---

# Specifier — FJ-061

## Scope as bounded here

Make `scripts/selftest`'s pristine fixture faithful: when a work item's real
`state.json` is embedded in a fixture, every report its `lastVerification`
references must be present in the fixture too. The only editable file is
`scripts/selftest`. `scripts/verify-candidate` must not change.

State of the tree at `9aef065`, read from the files and re-run here, not
assumed:

- `scripts/selftest:92 build_pristine()` copies `COPY_FILES`
  (`scripts/verify-candidate`, `scripts/agent-context`,
  `scripts/candidate-fingerprint`, `scripts/selftest`, `README.md`,
  `AGENTS.md`, `.gitignore`), `COPY_DIRS` (`docs`, `spec`),
  `.agent/README.md`, `.agent/schema`, `.agent/gate-policy.json`, and then the
  real `.agent/work/{FJ-044,FJ-045,FJ-046,FJ-047}/state.json` into a temp git
  repo. It never copies `.agent/reports/**`. `git_init()` then runs
  `git add -A` and commits, so every fixture input is committed and
  `verify-candidate` prints `tree clean`.
- Of those four items, FJ-044/FJ-045/FJ-046 are `planned` and have no
  `lastVerification`; FJ-047 is `complete` with
  `lastVerification.report = .agent/reports/FJ-047/report-20261002T152756Z.json`
  (14188 bytes, tracked: `git ls-files --error-unmatch` exits 0).
- `scripts/verify-candidate:432 check_recorded_report_coverage` iterates
  `.agent/work/*/state.json`, skips grandfathered ids
  (`FJ-001,002,003,004,043,048,053,FJ-PRD-001`) and non-`complete` items, and
  for each remaining item requires `lastVerification.report` to be a readable
  file whose checks cover every row implied by that report's own
  `policy.requiredStages` (`artifact`, `header`, `task_fingerprint`, `chain`
  after the first stage, `terminal` on the last). Its comment states that
  "nothing is recomputed here" — rows are read, not re-derived. It is a
  repository-scoped check: it runs taskless *and* inside task-mode runs
  (`scripts/verify-candidate:524`), so every fixture that embeds FJ-047's
  `state.json` currently fails it.
- Observed baseline on this machine (commands run here):
  - `./scripts/selftest` → exit 1, final line `summary: 49 passed, 26 failed`,
    with 26 `FAIL` lines.
  - `./scripts/selftest packs_frontmatter_valid` → exit 1; the fixture output
    under `==> complete items have recorded-report coverage` is
    `.agent/work/FJ-047/state.json:` /
    `- recorded report .agent/reports/FJ-047/report-20261002T152756Z.json does not exist` /
    `FAIL (exit 1, code policy.last_verification_coverage)`.
  - `./scripts/verify-candidate` (taskless, real repo) → exit 0,
    `summary: 20 passed, 0 failed, 0 skipped, 0 not_applicable`; it does not
    run `scripts/selftest`, so it does not reveal the red suite.
  - The 26 failing scenarios: `packs_frontmatter_valid`, `report_v2_shape`,
    `required_pass_success`, `optional_skipped_success`,
    `policy_additive_with_justification`, `go_required_for_product`,
    `metadata_go_not_required`, `exemption_stale_trackedby`,
    `go_product_zero_packages_skip`, `evidence_future_stage_pending`,
    `evidence_terminal_freshness`, `optional_na_success`,
    `metadata_hardener_na`, `exemption_tooling_bootstrap`,
    `gauntlet_block_renders`, `gauntlet_report_binding`,
    `history_missing_or_edge_violation`, `guard_clean_pass`,
    `guard_metadata_na`, `guard_bootstrap_skip`, `trace_empty_active_pass`,
    `race_product_due_pass`, `race_metadata_na`, `fuzz_zero_targets`,
    `fuzz_stage_na`, `fuzz_live_pass`.
- `scripts/selftest` holds 75 `@scenario` registrations; the suite is
  documented as offline ("no network, no sleeps"), builds fixtures in temp
  directories only, and neutralizes host git config via `GIT_HERMETIC_ENV`.
- Design §15.1 makes "`scripts/selftest` passes every case in §12" the way the
  design is known to work; §12's matrix lists rows such as "complete item whose
  recorded report lacks required stage rows → failure
  (`policy.last_verification_coverage`, §7)". Restoring a green suite serves
  §15.1; it does not add a §12 row.

## Observable acceptance criteria

Each criterion states what an observer sees and the observation that would
falsify it. Nothing here prescribes private implementation shape beyond the
observable behaviour.

### C1 — Embedding closure: referenced reports travel with the state.json

**Observable.** For every work item whose real `state.json` `build_pristine`
embeds, if that file's `lastVerification.report` names a path, the fixture
contains that path with byte-identical content (equal sha256). Today exactly
one pair must hold: `FJ-047` ↔
`.agent/reports/FJ-047/report-20261002T152756Z.json`.

**Exercise.** The closure script under "Exact commands" #4, which builds a
pristine fixture through the module's own `build_pristine` and compares hashes.
Expected: one line, `FJ-047 .agent/reports/FJ-047/report-20261002T152756Z.json True`.
Baseline at `9aef065`: the same line with `False`.

**Falsified by.** `False` in that line; a report present but with different
bytes; a fixture that embeds a state.json whose referenced report is absent;
the closure implemented as a second hardcoded path list that can drift from the
embedded ids.

### C2 — The scenario suite is green on a clean tree

**Observable.** `./scripts/selftest` exits 0 with a final line
`summary: 75 passed, 0 failed`. Every fixture's `==> complete items have
recorded-report coverage` row reports `PASS` (no fixture stdout contains
`policy.last_verification_coverage`). Baseline at `9aef065`: exit 1,
`summary: 49 passed, 26 failed`, 26 `FAIL` lines.

**Falsified by.** Any `FAIL` line; a non-zero exit; `0 failed` reached by
selecting fewer scenarios, by filters, or by deleting/renaming scenarios
(guarded by C4).

### C3 — The completion-coverage rule is not weakened

**Observable.**

1. `scripts/verify-candidate` is byte-identical to `9aef065`
   (`git diff --exit-code 9aef065 -- scripts/verify-candidate` exits 0 with no
   output). The allowed-files list contains only `scripts/selftest`, so this is
   a hard requirement, not a preference.
2. The rule's negative control still fires inside the suite:
   `./scripts/selftest coverage_missing_terminal_row` exits 0. That scenario
   builds a fixture whose complete item has a recorded report missing
   `stage.qa.terminal` and asserts the run exits non-zero with
   `policy.last_verification_coverage` and `missing report row stage.qa.terminal`.
3. The rule stays enforced in task mode as well as taskless: a task-mode run in
   a scratch fixture produced the row `{name: check_recorded_report_coverage,
   gate: G-C, required: true, result: pass, exitCode: 0, code:
   policy.last_verification_coverage, exempted: false}` (the check runs at
   `scripts/verify-candidate:524` in both modes, and FJ-047's embedded item
   keeps it meaningful).

**Falsified by.** Any diff in `scripts/verify-candidate`; a grandfathered-id
addition; a check that consults a fixture-local substitute for the report; the
negative-control scenario failing, being renamed away, or losing its
assertions; the coverage row disappearing from task-mode reports.

### C4 — No selftest assertion is removed or loosened

**Observable.** The change to `scripts/selftest` is additive where assertions
are concerned: `git diff 9aef065 -- scripts/selftest | grep -E '^-[^-]' | grep -E
'assert |@scenario'` prints nothing, and `grep -c '^@scenario' scripts/selftest`
is `>= 75`. Existing scenario bodies keep their `assert` statements and their
names; no scenario is filtered out, renamed, or turned into a no-op.

**Exercise (regression coverage for the closure).** A scenario (name contains
`pristine` and `report`, e.g. `pristine_embeds_referenced_reports`) builds a
pristine fixture and asserts closure for every embedded item with a referenced
report: the referenced path exists, its bytes equal the source file's bytes,
and `./scripts/verify-candidate` on that fixture exits 0 with
`check_recorded_report_coverage` `result: pass`. This keeps the C1 invariant
under regression in the same suite that caught the original defect.

**Falsified by.** A removed `assert`/`@scenario` line; a scenario count below
75; a loosened assertion (e.g. asserting only that the report path is a string);
the closure invariant asserted nowhere in the suite.

### C5 — The fixture stays hermetic

**Observable.**

1. Every file the fixture is built from is tracked at the current revision:
   `git ls-files --error-unmatch .agent/reports/FJ-047/report-20261002T152756Z.json`
   exits 0 and prints the path; the copied report's bytes equal the tracked
   file's bytes (C1). The fixture therefore does not depend on untracked or
   machine-specific state.
2. The fixture is a temp git repo whose worktree is clean after `build_pristine`
   (`git status --porcelain` empty), because `git_init()` commits the copied
   report along with the rest; `verify-candidate` on the fixture still prints
   `tree clean` (its preflight `revision=... tree clean` line).
3. The real repository is unmodified by a suite run: `git status --porcelain`
   is identical before and after `./scripts/selftest`.
4. No network and no sleeps are introduced:
   `grep -nE 'time\.sleep|socket\.|urllib|requests\.' scripts/selftest` finds
   nothing (exit 1). Expected residual matches for the broader pattern
   `sleep|socket|urllib|requests|http` are textual only: the module docstring
   line `Offline: no network, no sleeps. ...` and a Go guard fixture's source
   string `import _ "net/http"` — no call performs either.
5. Nothing in the change reads wall-clock time, the environment beyond the
   existing `GIT_HERMETIC_ENV`, or any path outside the real repository and
   temp directories; two consecutive closure runs print identical output.
6. The embedded evidence cannot perturb fixture digests: `.agent/reports/` is
   in `candidate-fingerprint`'s `EXCLUDED_PREFIXES`, so a fixture's candidate
   fingerprint is unchanged by the copied report (verified by building a
   pristine fixture, reading its digest, removing `.agent/reports`, and reading
   it again: equal).

**Falsified by.** A report copied from an untracked path; a dirty fixture
worktree or a `tree not clean` preflight; the real repository modified by the
suite; any added sleep/socket/HTTP call; a fixture digest that changes when the
report is added or removed.

## Recommended fix and reasoning

Two principled fixes exist. **Recommendation: embed the referenced reports
(option A).**

**Option A — copy the report each embedded item references.** `build_pristine`
reads each copied `state.json`, takes `lastVerification.report` when present and
non-empty, and copies that one path from the real repository into the fixture
(skipping when `lastVerification` is absent, as for three of the four items
today). Reasoning:

1. It restores a real invariant rather than masking a symptom: a fixture that
   embeds a `state.json` while omitting its bound evidence is a repository state
   that cannot exist in the real tree, and every scenario then fails on a defect
   unrelated to what it tests. The fixture becomes a consistent snapshot of the
   real `.agent` view again.
2. The closure is mechanical and self-maintaining: the copy set is derived from
   the state files already copied, so adding another real item (or another item
   completing with a bound report) cannot reintroduce the gap. Option B instead
   needs a per-item decision each time, which is the same drift class as the
   original bug.
3. It keeps the fixture faithful for the scenarios that depend on the real
   items' committed content — the exemption scenarios (`exemption_tooling_bootstrap`,
   `exemption_stale_trackedby`) require `FJ-044/045/046` to exist and not be
   complete, and the latter constructs a completion on the *real* `FJ-044`
   `state.json`. Embedding the real files keeps that observation anchored to
   the repository's own items instead of a paraphrase.
4. It is the smallest change: a handful of lines inside the existing copy loop,
   `8 insertions(+)` in a scratch run, touching no other file, no fingerprint
   rule, and no check.

**Option B — derive the embedded items so they cannot reference absent
reports.** `build_pristine` would synthesize stub items for `FJ-044…047`
instead of copying the real `state.json`. It also fixes the red suite, but:

1. The fixture would stop exercising the repository's real work-item content,
   so scenarios such as the exemption `trackedBy` checks would validate a
   paraphrase rather than the committed items — a real loss of what the fixture
   detects, in exchange for convenience.
2. The stub contents become a shadow copy that can silently diverge from the
   real items (missing fields, different statuses) — the same
   two-sources-of-truth failure mode, inverted.
3. `exemption_stale_trackedby` deliberately mutates the real `FJ-044`
   `state.json`; a synthetic stub makes that mutation a fixture-local ritual
   with no relation to the item the gate policy actually tracks.

A third variant — copying whole `.agent/reports/{FJ-044…047}` directories —
satisfies C1 but imports unreferenced evidence and grows the fixture for no
observable gain; not recommended, and not required (see O3).

## Exact commands and expected results

Run from the repository root. "Baseline" is observed at `9aef065`; the other
expectation column is the target this item asks for. Every command here has
been run at `9aef065`, or in a scratch copy of the repository with option A
applied (rows 2, 4, 5, 7, 8, 9 are from that scratch run; the scratch copy is
outside the repository and is not part of the deliverable).

| # | command | expected exit | expected observation |
|---|---------|---------------|----------------------|
| 1 | `./scripts/verify-candidate` | 0 | `summary: 20 passed, 0 failed, 0 skipped, 0 not_applicable` (identical before/after; no report written by a taskless run) |
| 2 | `git diff --exit-code 9aef065 -- scripts/verify-candidate` | 0 | no output — the rule is untouched |
| 3 | `./scripts/selftest` | 0 | final line `summary: 75 passed, 0 failed`; no `FAIL` line. Baseline: exit 1, `summary: 49 passed, 26 failed`, 26 `FAIL` lines |
| 4 | `./scripts/selftest packs_frontmatter_valid` | 0 | `summary: 1 passed, 0 failed`. Baseline: exit 1, `AssertionError` quoting `recorded report .agent/reports/FJ-047/report-20261002T152756Z.json does not exist` and `FAIL (exit 1, code policy.last_verification_coverage)` |
| 5 | closure script below | 0 | `FJ-047 .agent/reports/FJ-047/report-20261002T152756Z.json True`. Baseline: the same line with `False` |
| 6 | `./scripts/selftest coverage_missing_terminal_row` | 0 | `summary: 1 passed, 0 failed` — the negative control for the coverage rule still fires |
| 7 | `git ls-files --error-unmatch .agent/reports/FJ-047/report-20261002T152756Z.json` | 0 | prints the path (input is tracked) |
| 8 | `git diff 9aef065 -- scripts/selftest \| grep -E '^-[^-]' \| grep -E 'assert \|@scenario'` | 1 | no output (nothing removed) |
| 9 | `grep -c '^@scenario' scripts/selftest` | 0 | prints `75` or more, never fewer |
| 10 | `grep -nE 'time\.sleep\|socket\.\|urllib\|requests\.' scripts/selftest` | 1 | no output (no sleeps, no network) |
| 11 | `python3 -c "compile(open('scripts/selftest').read(),'scripts/selftest','exec'); print('syntax ok')"` | 0 | prints `syntax ok` |
| 12 | `git status --porcelain` before and after `./scripts/selftest` | 0 | identical output: the suite never modifies the real repository |

**Reproduction command (the defect this item closes).** At `9aef065`, on a
clean tree:

```bash
./scripts/selftest
echo $?
```

Observed: exit `1`, final line `summary: 49 passed, 26 failed`, and each failing
fixture's output contains

```text
==> complete items have recorded-report coverage
.agent/work/FJ-047/state.json:
  - recorded report .agent/reports/FJ-047/report-20261002T152756Z.json does not exist
    FAIL (exit 1, code policy.last_verification_coverage)
```

A single scenario isolates it: `./scripts/selftest packs_frontmatter_valid`.

**Closure script (command #5).** Saved nowhere; run inline, or replaced by the
equivalent scenario asserted in C4:

```python
import hashlib, importlib.machinery, importlib.util, json, pathlib, sys, tempfile

# The repo does not gitignore __pycache__, so loading the module must not write
# bytecode: a stray .pyc dirties the tree and perturbs the candidate digest
# (the same reason scripts/selftest sets sys.dont_write_bytecode itself).
sys.dont_write_bytecode = True

loader = importlib.machinery.SourceFileLoader("st", "scripts/selftest")
spec = importlib.util.spec_from_loader("st", loader)
mod = importlib.util.module_from_spec(spec); loader.exec_module(mod)

real = pathlib.Path(".").resolve()
root = mod.build_pristine(tempfile.mkdtemp(prefix="fj061-closure-"))
for item in sorted((real / ".agent/work").glob("*/state.json")):
    if not (root / ".agent/work" / item.parent.name / "state.json").is_file():
        continue
    rel = ((json.loads(item.read_text(encoding="utf-8")) or {}
            ).get("lastVerification") or {}).get("report")
    if not rel:
        continue
    src, dst = real / rel, root / rel
    same = dst.is_file() and (hashlib.sha256(dst.read_bytes()).hexdigest()
                              == hashlib.sha256(src.read_bytes()).hexdigest())
    print(item.parent.name, rel, same)
```

## Non-goals

- No change to `check_recorded_report_coverage`, to `GRANDFATHERED`, to the
  coverage-row set, or to any other part of `scripts/verify-candidate`; it is
  outside `allowedFiles` and must stay byte-identical (C3).
- No weakening, renaming, skipping, or removal of any selftest assertion or
  scenario; only additions (C4).
- No new dependency, no change to interpreter selection, no CI change.
- No change to `candidate-fingerprint`'s exclusion rules, to
  `gate-policy.json`, or to the real `.agent/work` and `.agent/reports`
  contents (this item edits only `scripts/selftest`).
- No new §12 matrix row is claimed: §12 stays as frozen; the suite is brought
  back to satisfying §15.1.
- No fix to the real FJ-047 item (it cannot be reopened; its committed report
  exists and is tracked).

## Observations (what the artifacts leave undefined)

**O1 — The fixture's own consistency is not a design requirement.**
Design §12 enumerates scenarios and §11 scopes selftest guarantees to them;
neither says a fixture embedding a real `state.json` must also embed the
evidence that file references. The requirement here comes from §15.1 (the suite
must pass every §12 case) plus this item's objective, not from a §12 row. It is
recorded as an observation because a reader looking for the fixture rule in the
design will not find it.

**O2 — Behaviour when a referenced report is absent or untracked is
undefined.** The real repository's case is covered by `verify-candidate`
itself, which fails closed on a complete item whose report is missing. The
fixture rule is unspecified: copy-if-present (mirroring the existing
`source.is_file()` guard for `state.json`) versus failing the fixture build
loudly. Recommendation: copy-if-present; the fixture is not a second
enforcement point, and a fixture that refused to build would convert a
repository defect into an unrelated suite failure — the exact failure mode this
item removes. If a referenced report exists but is untracked in the real repo,
copying it makes the fixture depend on working-tree-only state (C5.1); today no
such case exists (all 326 files under `.agent/reports` are tracked), and the
real repository fails its own verification in that condition anyway.

**O3 — Copy granularity is unspecified.** Referenced reports only, or whole
report directories. Recommendation: referenced-only, derived from the copied
`state.json` files; whole-directory copying satisfies C1 but imports
unreferenced evidence, enlarges every fixture, and re-creates a second list to
maintain.

**O4 — Fixture cost is unspecified.** FJ-047's report is 14188 bytes, copied
into every fixture (fixtures are per-scenario temp trees). Nothing bounds this
growth as more items complete; it is recorded, not designed against.

**O5 — Embedded evidence is stale by construction and that is intended.**
A copied report carries `candidateFingerprint`/`taskFingerprint`/`revision`
values bound to the *real* revision, not the fixture's; the fixture prints
`revision=29a9c1d`-style hashes of its own. Nothing recomputes or compares
them, and `check_recorded_report_coverage` reads rows only ("nothing is
recomputed here"). A future reader might mistake this for inconsistency; it is
the documented behaviour of a fixture embedding committed evidence.

**O6 — `verify-candidate` does not run `selftest`.** Already noted for FJ-047;
restated because it is why C2 must be exercised explicitly and why a green
`verify-candidate` is not evidence for this item.

**O7 — The embedded id list stays hardcoded.** `build_pristine` names
`FJ-044…FJ-047` explicitly, and nothing in the design says the list must be
derived from the repository. Only the *report closure* per embedded item is
required here; whether the id list becomes data-driven is left to the coder and
is not observable from outside the fixture.

**O8 — Acceptance-catalog trace.** The work item's assigned criterion is
`C-QUAL-005`; the catalog's closest entries are `C-QUAL-005` (offline operation,
§23) and `C-QUAL-006` (committed, reviewable machine-readable artifacts, §45).
Both are used as traces below. Adding a `C-QUAL-*` id for fixture fidelity
would extend the catalog beyond §23/§45 and is out of scope.

## Traces

- C-QUAL-005 — §23, golden contracts run offline with no credentials, model, or
  network: the suite stays offline after the change (C5.4, command #10), and the
  embedded report is read from the local tracked tree only (C5.1, command #7),
  so fixture construction performs no network access in either the baseline or
  the fixed suite.
- C-QUAL-006 — §45, required machine-readable contract artifacts are committed
  and reviewable: the evidence a fixture embeds is the committed
  `state.json` and the committed report it references, byte-identical (C1,
  command #5); the fix keeps fixtures reproducible from tracked artifacts
  (C5.1) and keeps `scripts/verify-candidate`'s committed coverage contract
  untouched (C3).

Design references (not catalog ids):
`docs/superpowers/specs/2026-09-25-gauntlet-roles-design.md` §6.1 (evidence
directories excluded from the candidate fingerprint — C5.6), §7 (recorded-report
coverage, the rule C3 leaves intact), §11 (selftest guarantees are scenario
bound; "no network, no modification of the real repository" — C5), §12 (the
frozen scenario matrix restored to green — C2, and the recorded-report coverage
row exercised by C3), §15.1 (the suite passes every §12 case — the normative
basis for C2). `docs/development/acceptance-catalog.md` §C-QUAL for the two
traces above.

This artifact states observable criteria and traces; it declares no
implementation outcome.
