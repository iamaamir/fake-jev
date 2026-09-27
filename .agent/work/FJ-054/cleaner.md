---
stage: cleaner
task: FJ-054
inputFingerprint: 2c8379c31c3539f27c20e1e475e20802239c617e269e4928ae166c0ee1c50df0
outputFingerprint: 0495abf349c5eaa9c0ca1a59a6c6a9f68e139bc9cbcf6a3a8c22581f94357147
taskFingerprint: ea17d10b719f94a62eb9a5c68f66391a466e7cf325961bf5cd5f8082ea8db881
gitHead: befff1a
generatedAt: 2026-09-27T21:39:14Z
author: subagent/cleaner-1
---

# Cleaner — FJ-054

## Gate G-L

No complexity/CRAP analyzer exists in `scripts/`, so G-L reports
`stage.tooling_bootstrap_exempt` per `.agent/gate-policy.json` (pre-existing
exemption, tracked by FJ-044). The review below was therefore performed by
reading, not by tooling.

## What was reviewed

Read: `docs/agents/roles/cleaner.md`, `docs/agents/expertise/staff-architect.md`,
`.agent/work/FJ-054/specifier.md` (C1–C6, non-goals), `.agent/work/FJ-054/coder.md`,
`git show befff1a -- scripts/verify-candidate` (the whole change), the surrounding
script (`record_row` / `run_check` / `skip_check` / `na_check` / `check_code`,
the header banners at lines 177–190, the `check_candidate_fingerprint` precedent
at 197–216, the product-check dispatch at 1316–1380), `docs/development/acceptance-catalog.md`
rows C-QUAL-007..009, `scripts/selftest` (fixture mutation of the script source,
Go-requiredness scenarios), `scripts/candidate-fingerprint` (go-invocation audit).

Diff-quality pass covered: dead/duplicated code, stale or misleading comments,
quoting, globbing, `set -e` interaction, command-substitution failure modes,
style consistency with the surrounding script, leftover debug noise.

## Claim-by-claim verification of `coder.md`

| # | coder.md claim | verified against |
|---|---|---|
| 1 | change is +70/−7, `scripts/verify-candidate` only | `git show --stat befff1a`: `1 file changed, 70 insertions(+), 7 deletions(-)` — exact |
| 2 | section inserted after the gauntlet engine, before the first Go invocation in the product path | new block starts line 1204, after `rm -f "$ENGINE_ROWS" "$ENGINE_ERR"` (1202), before `# --- product checks` (1263) and `ALL_PKGS=…` (1325); `grep` confirms no `go` command anywhere before line 1241 |
| 3 | `AMBIENT_GOWORK` captured before anything overrides `GOWORK` | line 1215 precedes the seven `export`s at 1216–1222; no earlier `export`/`GOWORK` assignment in the file |
| 4 | the seven pinned values match the table | 1216–1222: `-mod=readonly`, `off`, `off`, `""`, `""`, `sum.golang.org`, `local` — table matches; `printf` arg order matches the format string |
| 5 | informational line prints, carries no check row | 1223–1225 is a bare `printf`, no `record_row` — clean battery stays at 15 rows |
| 6 | `-mod=readonly` is the C2 mechanism | exported at 1216, inherited by `repo_packages`/`check_go_*` (1288/1301/1311/1314); `GOTOOLCHAIN=local`/`GOPROXY=off` corroborated by `go env` values printed in the run log |
| 7 | unknown-class `run_check … go.*` 0→1, only those three | 1363/1364/1366 now `1`; diff shows exactly three `0`→`1` flips in the `*` branch |
| 8 | skip rows / metadata NA rows / product branch untouched | `*`-branch `skip_check` still `0` (1368/1372/1376); `na_check` defaults `required=0` (170–175); product branch has no diff hunks |
| 9 | comment above the branch rewritten and cites C-QUAL-007 | 1316–1324 — present; design §8.3 lead sentence retained with the carve-out spelled out (see observation O4) |
| 10 | C3 detection is existence-gated ambient first, then `env -u GOWORK go env GOWORK`, `!= "off"`, `-e` | 1238–1246 — exactly as described; violation-only row with the path in `reason` (1247–1254) |
| 11 | C4 is `[ -d vendor ]` relative to `REPO_ROOT` (`cd` at line 36) | 1255–1261 and line 36 — as described; root-only |
| 12 | no other file touched, no new abstraction (`record_row` reused) | diff is one file; both new rows call the existing `record_row` in the `check_candidate_fingerprint`/`check_engine` style |

Probe claims re-run independently by this stage (see outputs below): the clean
battery (1), task verify (2), selftest (3), one C1 mutation (4, `GOTOOLCHAIN`),
C3(a) (6a) and C4 (7). Coder probes 4 (remaining six variables), 5, 6b and 8
are reproducible and left to qa, which re-runs them against the final revision.

## Edits made

One, comment-only, in `scripts/verify-candidate` (behavior-preserving: no
command, variable, row, condition or output line changed):

- The detection paragraph claimed both posture checks were "pure
  environment/filesystem inspection", while the workspace branch actually runs
  `go env GOWORK` (line 1241). Reworded to "environment/filesystem inspection
  plus one offline `go env GOWORK` probe" so an auditor looking for subprocesses
  in that section finds the probe named in the comment. Paragraph re-wrapped;
  9 insertions / 8 deletions, all inside one comment block.

Nothing else was touched: no check was weakened, renamed, reordered or
de-required; the pinned environment, `check_go_workspace`, `check_go_vendor` and
the unknown-class requiredness are byte-identical to the coder's revision.

## Observations deferred to qa

None is a defect this stage may repair (each would change observable behavior or
lie outside `allowedFiles`); recorded for judgment.

- **O1 — pinning starts after the gauntlet engine.** The seven `export`s run at
  line 1216, i.e. after `verification_rows()` has already executed any
  `verificationCommands` in `state.json` (engine, cwd = repo root). Those rows
  therefore run with the ambient environment. Acceptance C-QUAL-007 scopes the
  pin to "the product Go checks", so this is not a criterion miss; qa may still
  want it noted that an unpinned `go` in a verify command remains possible.
  (`scripts/candidate-fingerprint` and the task-context parse never invoke `go`,
  so nothing else before the pin can be affected.)
- **O2 — explicit `GOWORK=off` is discarded by the discovery probe.** Because
  C3(b) requires detection with no environment variable at all, the probe clears
  `GOWORK`; a run with an explicit ambient `GOWORK=off` and an ancestor
  `go.work` therefore fails, although `go` itself would not enter workspace mode
  (process `off` beats auto-discovery). Fail-closed direction, stricter than
  C-QUAL-008's "would change module resolution"; left as-is.
- **O3 — `!= "off"` guard is asymmetric.** The discovery branch excludes the
  literal value `off`; the ambient branch (1238) does not, so `GOWORK=off` plus
  a file literally named `off` in the repo root would be reported as an active
  workspace. Purely theoretical; not edited because removing that fail would be
  a relaxation.
- **O4 — design §8.3's "required iff class == product" is now contradicted by
  the code**, with the carve-out documented only in the script comment and
  `coder.md`. Updating the design/roles text is outside this item's
  `allowedFiles`; qa should decide whether an amendment row is warranted.
- **O5 — violation-only rows.** `check_go_workspace`/`check_go_vendor` exist in
  reports only when they fire; a clean run's only evidence of the posture is the
  unrowed banner line. This is what preserves the `15 passed` invariant and the
  coder documented it; qa confirms it against the criteria text.
- **O6 — row names are not functions.** `check_go_workspace`/`check_go_vendor`
  echo the `check_go_*` scheme that `run_check` uses for function names, but
  they are not functions and are absent from `check_code()`'s dispatch table
  (they would fall through to `verify.check_failed` if ever routed through
  `run_check`). Both rows pass their explicit codes at `record_row` time, so
  behavior is correct today; latent/cosmetic only.
- **O7 — banner style.** The pinned-environment line uses `printf` while the
  header's `verify-candidate:` banners use `echo`; the script already mixes both
  and this line's immediate neighbours are `printf`, so it was left alone.

## Verification outputs

`./scripts/verify-candidate` (exit 0)

```text
verify-candidate: module environment pinned (GOFLAGS=-mod=readonly GOPROXY=off GOWORK=off GOPRIVATE= GONOSUMDB= GOSUMDB=sum.golang.org GOTOOLCHAIN=local)
...
summary: 15 passed, 0 failed, 0 skipped, 0 not_applicable

RESULT: PASS
```

`./scripts/verify-candidate FJ-054` (exit 0)

```text
report: .agent/reports/FJ-054/report-20260927T213830Z.json
summary: 24 passed, 0 failed, 0 skipped, 4 not_applicable

RESULT: PASS
```

`./scripts/selftest` (exit 0)

```text
summary: 56 passed, 0 failed
```

`bash -n scripts/verify-candidate` → OK (no output, exit 0).

Independent probe re-runs by this stage:

```text
GOTOOLCHAIN=go1.99.0 ./scripts/verify-candidate   → exit 0, 15 passed, 0 failed, RESULT: PASS
GOWORK=/tmp/fj054-cl-a.work ./scripts/verify-candidate
                                                   → exit 1, "workspace file in effect: /tmp/fj054-cl-a.work",
                                                     summary: 15 passed, 1 failed, RESULT: FAIL
mkdir vendor → ./scripts/verify-candidate          → exit 1, code go.vendor_present,
                                                     summary: 15 passed, 1 failed, RESULT: FAIL
rmdir vendor  → (cleaned; git status shows no vendor residue)
```

`./scripts/candidate-fingerprint candidate .` after the edit:

```text
0495abf349c5eaa9c0ca1a59a6c6a9f68e139bc9cbcf6a3a8c22581f94357147
```

(probe temp files `/tmp/fj054-cl-*` removed; `git status --short` shows only the
pre-existing foreign `M AGENTS.md`, my `M scripts/verify-candidate`, and the
untracked `.agent/reports/` files — none staged).
