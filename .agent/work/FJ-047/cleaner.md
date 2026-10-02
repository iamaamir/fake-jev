---
stage: cleaner
task: FJ-047
inputFingerprint: 21a64a50bf2ef95509f639bef1bdb0c944680121c8a9ae720998bec119f504dc
outputFingerprint: 21a64a50bf2ef95509f639bef1bdb0c944680121c8a9ae720998bec119f504dc
taskFingerprint: 1f155d31629426f20bd4ea8a1d488c44f6657d6cd9a733b01bf4c73c9639fb7e
gitHead: 7eb1252
generatedAt: 2026-10-02T15:02:59Z
author: worker/FJ-047-cleaner
---

# Cleaner — FJ-047

## Structural change: none

No file was edited. Every candidate-visible file (`scripts/`, `docs/`,
`.agent/README.md`, `.agent/gate-policy.json`, `.github/workflows/ci.yml`) is
inside the candidate digest (`EXCLUDED_PREFIXES` is only
`.agent/work|reports|logs` and `.scratch/`; `docs/`, `.agent/`, `scripts/`,
`.github/` are `METADATA_PREFIXES` in `scripts/candidate-fingerprint`), so any
"tidy" edit would move the fingerprint off the pinned
`21a64a50bf2ef95509f639bef1bdb0c944680121c8a9ae720998bec119f504dc`. The review
found no defect that would justify that, so the simplification pass is
deliberately empty. Re-verification was still run on the unchanged tree (below).

## Findings against the review questions

**Fail-closed path is unwaivable.** `scripts/verify-candidate:684-697` records a
literal `fail` row (`check_role_packs`, gate `G-C`, `required` 1, exit 1, code
`toolchain.pyyaml_missing`) on every non-green branch, including
`VERIFY_PYTHON3` empty. The only transform that can attach an exemption is
`scripts/verify-candidate:1762` — `if row["result"] != "skipped" or not
row["required"]: continue` — so a `fail` row is structurally out of reach of the
exemption engine, whatever a policy entry's `gate`/`blocks`/`scope`/`code`
values are. `GAUNTLET_SELFTEST_NA` (`:1786-1791`) only downgrades `pass`/
`skipped` to `not_applicable`, which for a required row is then rewritten to
`fail`/`gate.invalid_result`; the malformed-row path (`:1800-1818`) only ever
writes `fail`. The row is therefore blocking at `:1826` -> status `failed` ->
exit 1.

Probed directly (outside the repository, in a throwaway copy at
`/tmp/fj047-cleaner-probe`): with the pre-change `G-C` exemption entry restored
verbatim (`blocks: ["toolchain.pyyaml_missing"]`, scope both classes,
`trackedBy: FJ-047` non-complete), a metadata item under
`PATH=/opt/homebrew/bin:$PATH` produced
`policy.exemptions.unknown pass — "4 exemption(s) checked"`, and
`check_role_packs` was still `result: fail, code: toolchain.pyyaml_missing,
exempted: false`, report `status: failed`, process exit 1. No entry shape in the
matching loop can turn that row into a skip or a pass.

**No surviving live reference to the removed exemption.** Repository-wide
search (`grep -rI`, excluding `.agent/reports/`): `toolchain.pyyaml_exempt`
survives only in the explicitly past-tense clause of design §9.2
(`docs/superpowers/specs/2026-09-25-gauntlet-roles-design.md:561`,
"existed until FJ-047 … the entry is gone"), in the dated implementation plan
`docs/superpowers/plans/2026-09-25-gauntlet.md`, and in evidence artifacts
(`.agent/work/FJ-047/specifier.md`, `.agent/work/FJ-047/coder.md`,
`.agent/work/FJ-004/qa.md:55`). `.agent/gate-policy.json`,
`.agent/README.md`, `.agent/schema/` (no gate-policy schema exists; only
role-pack/stage-artifact/work-item), `scripts/README.md`, `scripts/selftest`,
`scripts/requirements-verify.txt` and `.github/workflows/ci.yml` are free of it.
`toolchain.pyyaml_missing` survives on purpose: it is the live code of the new
`fail` row (specifier O9) and is named in design §4/§13, `scripts/README.md`
and `scripts/requirements-verify.txt` as the failure code, not as a waivable
skip code. The dated plan and the prior evidence artifacts are historical
records (specifier O6 chose not to rewrite the plan; `.agent/work` is outside
this item's `allowedFiles` and is excluded from candidate identity), so they
were left untouched rather than silently rewritten.

**One interpreter, named in the diagnostic.** `VERIFY_PYTHON3` is resolved once
at `:286`, before `check_toolchain_python3`, which prints it at `:293`; the same
variable is the program for `check_role_packs`' heredoc at `:527` and for the
probe at `:687`. Observed on the same tree: with the default `PATH` the
toolchain row printed `python3: /Users/mak/.pyenv/shims/python3` and the check
passed; with `/opt/homebrew/bin` first the toolchain row printed
`python3: /opt/homebrew/bin/python3` and the failing reason contained the
identical string
(`PyYAML not importable by /opt/homebrew/bin/python3 (pinned in
scripts/requirements-verify.txt; install with: /opt/homebrew/bin/python3 -m pip
install -r scripts/requirements-verify.txt)`). The scenario
`pyyaml_missing_fails_closed` asserts that agreement independently
(`probed in reason`). No interpreter path is hardcoded and `python3` is not
resolved a second time inside `check_role_packs`.

**No other caller is broken.** The conditional `skip_check` was the only
producer of a role-pack `skipped` row; `grep` finds no other caller and no
`YAML`-shadowing scenario besides the new one (`yaml_shim` is used exactly once,
at `scripts/selftest:1269`). Taskless path: `./scripts/verify-candidate` exits 0
(20/0/0/0); with the PyYAML-less interpreter first it exits 1 with
`19 passed, 1 failed, 0 skipped`. Task path: the recorded passing report for
this item shows `check_role_packs pass / agent.role_pack_invalid / exempted
false` and `policy.exemptions.unknown pass — "3 exemption(s) checked"`; the
recorded negative report shows the same row as `fail / toolchain.pyyaml_missing
/ exempted false / exitCode 1`. `./scripts/selftest`: 75/0. The two §12 rows the
specifier required to stay asserted still are — `required_skipped_failure` now
drives the required+skipped+no-exemption case with a due Cleaner tooling row
(`stage.tooling_absent`, G-L, `exempted: false`) and no PyYAML dependency, and
`exemption_tooling_bootstrap` still asserts required+skipped+exemption ->
success. The three `role_pack_*` scenarios (dangling reference, unknown key,
non-array front matter) still pass, i.e. the check itself still detects
malformed packs.

**Docs.** The four edited documents state only what the code does: design §4
(pin artifact, no conditional form, `fail` with `toolchain.pyyaml_missing`,
never waivable because §9.2 matches `skipped` rows only), design §9.2 (three
exemptions, the `G-C` entry recorded as removed), design §13 (the remaining
limitation is `PATH` interpreter choice, not silent absence), guard-stack design
§4 (the `G-C` entry is gone), `scripts/README.md` (pin, rationale, ordinary and
`--no-index --find-links` install paths, wheel-platform caveat, "verify-candidate
never installs"), `.agent/README.md` (three exemptions; the G-C table row is
deleted and the same file's "never waive a `fail`" paragraph now matches
reality). The remaining occurrences of the old behaviour are the dated plans
and prior evidence artifacts listed above; no other document asserts the removed
exemption or the old conditional skip.

**CI recipe.** `python3 -m venv "$RUNNER_TEMP/verify-venv"` avoids PEP 668
without `--break-system-packages`; the pin is installed from the committed
artifact (`-r scripts/requirements-verify.txt`, no free-standing
`pip install PyYAML`) with the venv's own interpreter; `>> "$GITHUB_PATH"`
prepends the venv's `bin` for all subsequent steps, so the verifier's single
`command -v python3` resolves to the interpreter that holds the pin; the venv
lives in `$RUNNER_TEMP`, outside the checkout, so it never becomes untracked
workspace content in the candidate digest. The step precedes
`./scripts/verify-candidate` (lines 96-104), `gauntlet` is the only job that
runs the verifier, and the file parses as YAML (`yaml.safe_load` -> `yaml ok`).
The header comment now names the setup-time PyPI fetch instead of claiming the
Go toolchain is the only one.

**Shell quality.** Quoting is correct throughout the new block: the probe path
is quoted, the `printf` format is a literal with `%s` placeholders only
(no data in the format), and `record_row` arguments are quoted. `set -e` is not
tripped: `command -v ... || true` and the `elif ! ...` condition both keep the
non-zero probe inside a guard. Exit-code semantics match the surrounding
conventions (synthetic verdict rows elsewhere also record exit 1). The
duplication between the new block and `check_toolchain_python3` is a shared
`-z "$VERIFY_PYTHON3"` test with different messages for different gates
(`toolchain.python3_missing` vs `toolchain.pyyaml_missing`), both of which must
be emitted; the three-line console shape duplicates `run_check`'s
`FAIL (exit %s, code %s)` line, which is not worth a new helper for a single
site. No error is swallowed: the probe's stderr is discarded deliberately
because the reason string carries the interpreter and the install command.

**Script quality.** `verify-candidate` gains no dependency (the only `pip`
occurrence is inside the diagnostic text), never installs, and makes no network
call; `grep` for `pip|curl|wget|http|git clone` shows only the message strings.
The new `scripts/requirements-verify.txt` is not read by the verifier at all, so
it adds no runtime coupling.

## Commands run (all from the repository root)

| command | exit | observation |
|---------|------|-------------|
| `bash -n scripts/verify-candidate scripts/selftest` | 0 | no output |
| `./scripts/selftest` | 0 | `summary: 75 passed, 0 failed` |
| `./scripts/selftest pyyaml` | 0 | `1 passed` (`pyyaml_missing_fails_closed`) |
| `./scripts/selftest required_skipped_failure` | 0 | `1 passed` |
| `./scripts/selftest exemption_tooling_bootstrap` | 0 | `1 passed` |
| `./scripts/selftest role_pack` | 0 | `3 passed` |
| `./scripts/verify-candidate` | 0 | `20 passed, 0 failed, 0 skipped, 0 not_applicable`, `RESULT: PASS` |
| `PATH=/opt/homebrew/bin:$PATH ./scripts/verify-candidate` | 1 | `19 passed, 1 failed, 0 skipped`; `FAIL (exit 1, code toolchain.pyyaml_missing)` with `/opt/homebrew/bin/python3` named in the reason; `RESULT: FAIL` |
| `./scripts/candidate-fingerprint candidate` | 0 | `21a64a50bf2ef95509f639bef1bdb0c944680121c8a9ae720998bec119f504dc` (before and after this stage) |

Ad hoc probes (outside the checkout, no repository file touched):

- `/tmp/fj047-cleaner-probe` (rsync copy, own git init) with the old `G-C`
  exemption restored and a metadata `FJ-900` item:
  `PATH=/opt/homebrew/bin:$PATH ./scripts/verify-candidate FJ-900` -> exit 1,
  `check_role_packs` `fail`/`toolchain.pyyaml_missing`/`exempted: false`,
  `4 exemption(s) checked`, `status: failed`.
- PATH without any `python3` (`/tmp/nopy/bin`, symlinks to every needed tool
  except python): candidate `./scripts/verify-candidate` -> exit 127, toolchain
  row `FAIL (exit 1, code toolchain.python3_missing)`, role-pack row
  `FAIL (exit 1, code toolchain.pyyaml_missing)` with reason `no python3 on
  PATH; role-pack parsing needs the PyYAML pinned in
  scripts/requirements-verify.txt`. The baseline script (`git show
  7eb1252:scripts/verify-candidate`) under the same PATH also exits 127, but its
  role-pack row reads `SKIPPED: PyYAML not importable (design §4 conditional)`.
  The process exit is unchanged (the report writer needs `python3` in both
  revisions); only the row verdict changed, from skip to fail.

## Residual risks and observations (no changes made)

- The no-`python3`-at-all environment still aborts at the report writer
  (`scripts/verify-candidate:1678`) with exit 127 before a summary is printed.
  Pre-existing and identical in the 7eb1252 baseline; the new behaviour under
  review (role-pack row fails rather than skips) is observable before that
  point.
- `scripts/requirements-verify.txt` is not in `check_required_files`, so
  deleting the pin would not fail verification on a machine that still imports
  PyYAML; only the CI install step would notice. No acceptance criterion asks
  for pin-drift detection (the specifier's P1/S6 are about the artifact's
  presence and CI consuming it), and adding a new required file would be a new
  requirement, so this is recorded rather than implemented.
- `.github/workflows/ci.yml` cannot be executed here. The venv recipe relies on
  the runner image providing `venv`/`ensurepip` for its system `python3`
  (standard for `ubuntu-latest`, not observable offline); the step order,
  artifact path, PATH direction and YAML validity are verified by inspection
  and `yaml.safe_load`.
- Stale historical text remains, by decision, in
  `docs/superpowers/plans/2026-09-25-gauntlet.md` (old §4 sentence, the
  four-exemption table, the `pyyaml_exempt_with_task` listing) and in prior
  evidence artifacts (`.agent/work/FJ-004/qa.md`, `.agent/work/FJ-020/*`, this
  item's own specifier/coder artifacts). None of them is a live behaviour
  contract; editing them would rewrite frozen evidence.
- The design §9.2 clause keeps the string `toolchain.pyyaml_exempt` alive in a
  normative document as a past-tense record of the deleted entry (with the
  entry's removal stated). That is doc consistency with the earlier plan, not
  an assertion that a waiver exists.
