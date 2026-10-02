---
stage: qa
task: FJ-061
inputFingerprint: 050f8a89923dd23b42190e976b6db7fa5b1327118ac6fcca2eb1ab4247e0a4a2
outputFingerprint: 050f8a89923dd23b42190e976b6db7fa5b1327118ac6fcca2eb1ab4247e0a4a2
taskFingerprint: fba7216e76c708718baf8fca335a13ca9459e0a78927de751c9f07183d538a9b
gitHead: 9aef065
generatedAt: 2026-10-02T16:36:34Z
author: worker/FJ-061-qa
---

# QA — FJ-061

Independent stage. Author `worker/FJ-061-qa` differs from the coder author
(`worker/FJ-061-coder`). All verification ran in throwaway copies under
`/private/tmp/fj061-qa` (`base`, `fixed`, `neg`, `herm`, `corrupt`) and in
scenario-owned temp directories. No repository file was edited by this stage
other than this artifact.

## Environment and construction of the throwaway copies

| copy | how built | identity |
|---|---|---|
| `base` | `git archive 9aef065 \| tar -x -C .../base`, then `git init` + commit | HEAD `8a8bb4a1`; `sha256(scripts/selftest)=125979d9c10985689ba440b05d45b84763e2bb09ca5ff4d701edf27a6c3caf41` |
| `fixed` | `rsync -a --exclude .git --exclude fake-jev <repo>/ .../fixed/`, then `git init` + commit | HEAD `392e866d`; `sha256(scripts/selftest)=db3ebf5a0bb7ffeb9b860c933aab86772b63100efc2f3f3580211b619c621290` — equal to the real repository's working-tree file |
| `neg` | `rsync` of `fixed`, then `git init` + commit, then the negative mutation below | HEAD `db31c4d` |
| `herm` | `rsync` of `fixed`, then untracked files planted | HEAD + 2 untracked paths |
| `corrupt` | `rsync` of `fixed`, then the tracked report replaced by rowless JSON and committed | — |

The real repository at this session's start: `HEAD 9aef065`,
`git status --porcelain` = ` M .agent/work/FJ-061/state.json`, ` M scripts/selftest`,
untracked `specifier.md`, `coder.md`, `cleaner.md`. Nothing staged.

## C1 — embedding closure (referenced report travels and is byte-identical)

Probe built a pristine fixture through the module's own `build_pristine` and
compared sha256 per embedded item:

```text
$ cd /private/tmp/fj061-qa/fixed && python3 <closure probe using mod.build_pristine>
FJ-044  embedded, no lastVerification.report (status=planned)
FJ-045  embedded, no lastVerification.report (status=planned)
FJ-046  embedded, no lastVerification.report (status=planned)
FJ-047  .agent/reports/FJ-047/report-20261002T152756Z.json sha256-equal=True (status=complete)
fixture worktree clean after build: True
```

Same probe against `base` (9aef065):

```text
FJ-047  .agent/reports/FJ-047/report-20261002T152756Z.json present_in_fixture=False
```

The fixture's own `git ls-files --error-unmatch` resolves the embedded report:
`.agent/reports/FJ-047/report-20261002T152756Z.json`. In the real repository
`git ls-files --error-unmatch .agent/reports/FJ-047/report-20261002T152756Z.json`
exits 0 (tracked input, so the fixture cannot depend on worktree-only bytes).

The copy set is derived per embedded item, not from a second hardcoded list:
grep of `scripts/selftest` shows one call site (`copy_referenced_report(root, source)`
inside the existing `FJ-044…FJ-047` loop) and the path comes from each
`state.json`.

## C2 — suite outcome on the fixed tree

```text
$ cd /private/tmp/fj061-qa/fixed && ./scripts/selftest
exit=0
summary: 75 passed, 0 failed
FAIL lines: 0   stderr bytes: 0
```

Run twice; stdout and stderr byte-identical between runs (`diff` clean), so 75
of 75 registered scenarios ran and none was filtered.

Base revision, same suite, same machine:

```text
$ cd /private/tmp/fj061-qa/base && ./scripts/selftest
exit=1
summary: 49 passed, 26 failed
FAIL lines (stdout): 26
stderr tracebacks: 26   # one per failing scenario
```

The 26 failing names, in suite order: `packs_frontmatter_valid`, `report_v2_shape`,
`required_pass_success`, `optional_skipped_success`,
`policy_additive_with_justification`, `go_required_for_product`,
`metadata_go_not_required`, `exemption_stale_trackedby`,
`go_product_zero_packages_skip`, `evidence_future_stage_pending`,
`evidence_terminal_freshness`, `optional_na_success`, `metadata_hardener_na`,
`exemption_tooling_bootstrap`, `gauntlet_block_renders`,
`gauntlet_report_binding`, `history_missing_or_edge_violation`,
`guard_clean_pass`, `guard_metadata_na`, `guard_bootstrap_skip`,
`trace_empty_active_pass`, `race_product_due_pass`, `race_metadata_na`,
`fuzz_zero_targets`, `fuzz_stage_na`, `fuzz_live_pass` — identical to the set
recorded by the specifier and coder artifacts.

Cause of each base failure, classified from the 26 stderr tracebacks (one per
failure): 25 quote

```text
    - recorded report .agent/reports/FJ-047/report-20261002T152756Z.json does not exist
      FAIL (exit 1, code policy.last_verification_coverage)
```

and the 26th (`exemption_stale_trackedby`, which asserts on report JSON rather
than fixture stdout) fails with

```text
AssertionError: check_recorded_report_coverage: result='fail', want 'pass'
(row: {'name': 'check_recorded_report_coverage', 'gate': 'G-C', 'required': True,
 'result': 'fail', 'exitCode': 1, 'code': 'policy.last_verification_coverage',
 'exempted': False, 'command': 'complete items have recorded-report coverage',
 'reason': None})
```

Single-scenario isolation at base:

```text
$ cd /private/tmp/fj061-qa/base && ./scripts/selftest packs_frontmatter_valid
exit=1
==> complete items have recorded-report coverage
.agent/work/FJ-047/state.json:
  - recorded report .agent/reports/FJ-047/report-20261002T152756Z.json does not exist
    FAIL (exit 1, code policy.last_verification_coverage)
summary: 0 passed, 1 failed
```

Same scenario on the fixed tree: `exit=0`, `summary: 1 passed, 0 failed`.

## C3 — completion-coverage rule unchanged and still enforced

```text
$ git diff --exit-code 9aef065 -- scripts/verify-candidate
exit=0, no output
sha256(scripts/verify-candidate) = 3d00923dc2d2a852bb3af7528672f0859c50587f88f09710a833fcb111558526
sha256(git show 9aef065:scripts/verify-candidate) = 3d00923dc2d2a852bb3af7528672f0859c50587f88f09710a833fcb111558526
```

Negative control inside the suite on the fixed tree:

```text
$ ./scripts/selftest coverage_missing_terminal_row   → exit 0, summary: 1 passed, 0 failed
$ ./scripts/selftest exemption_stale_trackedby       → exit 0, summary: 1 passed, 0 failed
```

Task mode is still exercised with a meaningful row. A pristine fixture run as
`./scripts/verify-candidate FJ-047` produced a task-mode report whose coverage
row is

```json
{"name": "check_recorded_report_coverage", "gate": "G-C", "required": true,
 "result": "pass", "exitCode": 0, "code": "policy.last_verification_coverage",
 "exempted": false, "command": "complete items have recorded-report coverage",
 "reason": null}
```

The same task-mode probe against the base copy produced the same row with
`result: "fail"`, `code: "policy.last_verification_coverage"`. The rest of the
task-mode verdict is identical between the two revisions: exit 1 in both, with
the same four failing rows `stage.{specifier,coder,cleaner,qa}.artifact`
(`stage.artifact_missing`). The fix therefore changes exactly the coverage
dimension; it does not add a suppression path.

## C4 — no assertion or scenario removed or loosened

```text
$ grep -oE '@scenario\("[^"]+"' <script> | sort   (base 9aef065 vs fixed tree)
base count=75   fixed count=75
diff → SCENARIO LISTS IDENTICAL (no add/remove/rename)

$ git diff 9aef065 -- scripts/selftest | grep -E '^-[^-]' | grep -E 'assert |@scenario'
(no output)

$ git diff 9aef065 -- scripts/selftest --numstat
1 file changed, 30 insertions(+)  # scripts/selftest only
```

`grep -c '^@scenario' scripts/selftest` prints `75`.

## C5 — fixture hermeticity

*Tracked, not machine-local.* The one copied report is tracked in the real
repository (`git ls-files --error-unmatch` exit 0) and byte-equal after copying
(C1). The trackedness gate was probed directly, importing the module from the
`herm` copy: an untracked-but-present worktree file
(`.agent/reports/FJ-061-scratch/untracked-report.json`, `git ls-files` exit 1,
`error: pathspec ... did not match any file(s) known to git`) referenced by a
state file was **not** copied (`copied into fixture root: False`), while a
tracked path was copied and sha256-equal (`tracked control copied: True`,
`byte-identical: True`).

*Untracked files in the fixture source do not perturb the suite.* With two
untracked paths present in the copy that builds the fixture
(`.agent/reports/FJ-061-scratch/`, `probe-state.json`), the full suite gave
`exit=0`, `summary: 75 passed, 0 failed`, and the per-scenario verdict sequence
was byte-identical to the run without those files (`diff` of the `PASS`/`FAIL`
lines clean).

*Fixture is deterministic and self-contained.* Two independent `build_pristine`
calls produced the same content hash over 45 non-`.git` entries
(`sha256=2b086d516254755c67acc1a8e394a703b7061afc11faec9abe04177c70acffc8`
both times), and the fixture worktree is clean (`git status --porcelain` empty).

*Embedded evidence does not perturb digests.* Fixture candidate fingerprint with
the copied report: `7fb67bbbdf9d33c4b21f0f223b633cddbb800b3ddad877aa7bd797305917bb2c`;
after `rm -rf .agent/reports` in the fixture: the same value (`.agent/reports/`
is in `candidate-fingerprint`'s `EXCLUDED_PREFIXES`). Fixture `verify-candidate`
exits 0 and prints `tree clean` as its preflight.

*No network.* `grep -nE 'socket|urllib|requests|http|curl|wget|net/http|https?://' scripts/selftest`
matches only a Go fixture source string at line 1562
(`'import _ "net/http"'`); the module imports are `hashlib, json, os, pathlib,
re, shutil, subprocess, sys, tempfile, traceback`. Every process the suite
launches goes through `sh()`, whose command strings are `git …`,
`./scripts/verify-candidate`, `./scripts/candidate-fingerprint`,
`./scripts/agent-context` and `command -v python3`. Empirically, the full suite
run under a network-denying sandbox still returned `exit=0`,
`summary: 75 passed, 0 failed`, empty stderr:

```text
$ sandbox-exec -p '(version 1)(allow default)(deny network*)' ./scripts/selftest
```

*No sleeps.* `grep -nE 'time\.sleep|socket\.|urllib|requests\.' scripts/selftest`
has no match (exit 1).

*The real repository is not modified.* `git status --porcelain` in the real
repository was identical before and after a full suite run
(` M .agent/work/FJ-061/state.json`, ` M scripts/selftest`, untracked
`specifier.md`/`coder.md`/`cleaner.md`).

## Negative evidence — the fixture does not hide a genuine missing report

Throwaway copy `neg` = the fixed tree with the real, tracked referenced report
removed while the item stays `complete` and bound to it.

*Artifact removed (exact path, in the throwaway copy's real repository):*

```text
/private/tmp/fj061-qa/neg/.agent/reports/FJ-047/report-20261002T152756Z.json
```

```text
$ git ls-files --error-unmatch .agent/reports/FJ-047/report-20261002T152756Z.json   # before removal
.agent/reports/FJ-047/report-20261002T152756Z.json
$ rm -f .agent/reports/FJ-047/report-20261002T152756Z.json
removed: ... report-20261002T152756Z.json (gone)
```

`neg/.agent/work/FJ-047/state.json` is untouched: `status: "complete"`,
`lastVerification.report = .agent/reports/FJ-047/report-20261002T152756Z.json`.

Real-repository run in `neg`:

```text
$ ./scripts/verify-candidate
verify-candidate: repository=/private/tmp/fj061-qa/neg revision=db31c4d
verify-candidate: tree DIRTY (0 staged, 1 unstaged, 0 untracked) …

==> complete items have recorded-report coverage
.agent/work/FJ-047/state.json:
  - recorded report .agent/reports/FJ-047/report-20261002T152756Z.json does not exist
    FAIL (exit 1, code policy.last_verification_coverage)

summary: 16 passed, 3 failed, 1 skipped, 0 not_applicable
RESULT: FAIL   (exit 1; the two other failing rows are Go test/race rows unrelated to coverage)
```

Rebuilt fixture, via the fixed suite's own builder:

```text
$ ./scripts/selftest packs_frontmatter_valid        # in neg
exit=1
verify-candidate: tree clean
==> complete items have recorded-report coverage
.agent/work/FJ-047/state.json:
  - recorded report .agent/reports/FJ-047/report-20261002T152756Z.json does not exist
    FAIL (exit 1, code policy.last_verification_coverage)
summary: 0 passed, 1 failed
```

Whole suite in `neg`:

```text
$ ./scripts/selftest
exit=1
summary: 49 passed, 26 failed      # identical to the base revision's red set
FAIL lines: 26   stderr missing-report mentions: 26
```

So with the real report genuinely absent, the fixture still fails
completion-coverage for that item, on a clean fixture worktree
(`verify-candidate: tree clean`), and the suite returns to its red set.

Second, independent vector — the report exists but carries no rows. In `corrupt`
the tracked file was replaced (and committed) with
`{"taskId":"FJ-047","checks":[]}`, 32 bytes:

```text
$ ./scripts/selftest packs_frontmatter_valid
exit=1
==> complete items have recorded-report coverage
.agent/work/FJ-047/state.json:
  - recorded report policy.requiredStages missing or empty
    FAIL (exit 1, code policy.last_verification_coverage)
```

Presence alone is not accepted by the fixture-side rule; the recorded rows are
read.

## Required commands (recorded)

| command | exit | observed |
|---|---|---|
| `bash -n scripts/selftest` | 2 | `scripts/selftest: line 23: syntax error near unexpected token '('` / `REAL = pathlib.Path(__file__).resolve().parent.parent`. Not a property of this change: `bash -n /private/tmp/fj061-qa/base/scripts/selftest` (9aef065) prints the identical diagnostic with exit 2. The file is Python (`#!/usr/bin/env python3`); its equivalent syntax check, `python3 -c "compile(open('scripts/selftest').read(),'scripts/selftest','exec')"`, prints `syntax ok`, exit 0, on both revisions |
| `./scripts/selftest` (real repo) | 0 | `summary: 75 passed, 0 failed`, 75 `PASS` lines, 0 `FAIL` lines, empty stderr |
| `./scripts/verify-candidate` (real repo, taskless) | 0 | `summary: 20 passed, 0 failed, 0 skipped, 0 not_applicable`, `RESULT: PASS` |
| `./scripts/candidate-fingerprint candidate` | 0 | `050f8a89923dd23b42190e976b6db7fa5b1327118ac6fcca2eb1ab4247e0a4a2` — equals the coder's and cleaner's `outputFingerprint`, i.e. the fingerprint this artifact carries is still current after all QA runs |

## Residual risks and evidence limits

- `neg` carried a dirty worktree (the deletion is unstaged), so its real-repo
  `verify-candidate` preflight printed `tree DIRTY`; the fixture-side observation
  that matters (`verify-candidate: tree clean` plus the failing coverage row) was
  reproduced on a clean fixture, and the base-revision failure reproduces the
  same way, so the dirty preflight is not the cause of the row.
- The `neg` full-suite counts equal the base counts (49/26), but the two runs are
  not byte-identical: `neg` has Go test/race rows failing for the `neg`-specific
  Go toolchain state. The coverage-caused set (26 missing-report mentions in
  stderr, one per failing scenario) is the aspect compared.
- The trackedness gate copies the *worktree bytes* of a tracked report. A future
  report edited but not committed would be embedded in its dirty form; nothing
  in this change detects that, and `verify-candidate` reads rows, not bytes.
- Fixture fidelity is report-level only. A task-mode run against the embedded
  `FJ-047` still fails with four `stage.artifact_missing` rows because the
  fixture embeds `state.json` and its report but not the item's stage artifacts;
  that behaviour is identical at the base revision and is outside this item.
- The in-suite regression for closure is indirect: no scenario is named for it,
  so a future break is observed through the ~25 fixture scenarios that run
  `verify-candidate` on fixtures embedding `FJ-047` (their coverage row and exit
  code), not through a dedicated assertion. `copy_referenced_report` has no
  self-assertion.
- Hermeticity evidence is behavioural, not exhaustive: no network attempt was
  intercepted at syscall level beyond the `sandbox-exec` deny-network run; host
  git configuration is neutralized by the pre-existing `GIT_HERMETIC_ENV`.
- Determinism was sampled twice per suite variant; it is not a long-horizon
  stability claim, and the suite has no random or time input that a longer run
  would expose.
- The candidate fingerprint is a workspace digest of the current revision, so it
  changes if any tracked file changes; the value above was re-read after all QA
  commands and matched the artifact's `inputFingerprint`/`outputFingerprint`.
- Nothing was staged by this stage; `git status --porcelain` after all runs shows
  only the pre-existing modifications plus this artifact.
