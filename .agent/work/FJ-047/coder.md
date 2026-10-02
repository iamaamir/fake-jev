---
stage: coder
task: FJ-047
inputFingerprint: 3d896a0a7d7ae6bd6f347b4166a74c6d23bbf59a384869c2cc3c2697e96e9dcc
outputFingerprint: 21a64a50bf2ef95509f639bef1bdb0c944680121c8a9ae720998bec119f504dc
taskFingerprint: 1f155d31629426f20bd4ea8a1d488c44f6657d6cd9a733b01bf4c73c9639fb7e
gitHead: 7eb1252
generatedAt: 2026-10-02T14:52:00Z
author: worker/FJ-047-coder
---

# Coder — FJ-047

## Final candidate fingerprint

`21a64a50bf2ef95509f639bef1bdb0c944680121c8a9ae720998bec119f504dc`
(`./scripts/candidate-fingerprint candidate` after all edits; `.agent/work` is
excluded from candidate identity, so writing this artifact did not move it —
recomputed after the write and identical).

## What changed, per criterion

**P1 — pinned PyYAML, with rationale and an offline path.**
`scripts/requirements-verify.txt` (new) carries exactly one requirement line,
`PyYAML==6.0.3` at column 0, plus comment lines recording the rationale (6.0.3
is the release the verification interpreter imports today and the current PyPI
release, so the environment role packs are already parsed in is reproducible on
a fresh machine; an exact pin is deliberate because a range would let a
different importable version satisfy verification invisibly) and both install
recipes. `scripts/README.md` gained a "Verification toolchain" section under the
existing `python3` paragraph: the pin, the same rationale, the ordinary install
(`python3 -m pip install -r scripts/requirements-verify.txt`), the offline recipe
(`pip download --only-binary=:all: --no-deps -d <wheel-dir>` then
`pip install --no-index --find-links <wheel-dir> -r scripts/requirements-verify.txt`),
the note that wheels are platform/interpreter specific, and the statement that
`verify-candidate` never installs anything.

**S2 — `check_role_packs` is unconditional and required.** The
`if python3 -c 'import yaml' … else skip_check … "toolchain.pyyaml_missing"` block
is deleted. The positive path is a plain
`run_check "role packs match the role-pack contract" check_role_packs`, whose row
is `{gate: G-C, required: true, result: pass, code: agent.role_pack_invalid,
exempted: false}` (verified in the task-mode report). No code path can turn the
row into `skipped` or `not_applicable`.

**S3 — exact fail-closed behaviour.** The new failure row, read from
`.agent/reports/FJ-047/report-20261002T145107Z.json`:

```json
{
  "name": "check_role_packs",
  "gate": "G-C",
  "required": true,
  "result": "fail",
  "exitCode": 1,
  "code": "toolchain.pyyaml_missing",
  "exempted": false,
  "command": "role packs match the role-pack contract",
  "reason": "PyYAML not importable by /opt/homebrew/bin/python3 (pinned in scripts/requirements-verify.txt; install with: /opt/homebrew/bin/python3 -m pip install -r scripts/requirements-verify.txt)"
}
```

Process exit is `1` in taskless and task mode alike (observed `19 passed, 1
failed, 0 skipped` taskless; `13 passed, 1 failed, 0 skipped, 9 not_applicable`
task-mode with a metadata item — the run the removed exemption used to turn
green). Console shape under the header
`==> role packs match the role-pack contract`:

```text
    PyYAML not importable by /opt/homebrew/bin/python3 (pinned in scripts/requirements-verify.txt; install with: /opt/homebrew/bin/python3 -m pip install -r scripts/requirements-verify.txt)
    FAIL (exit 1, code toolchain.pyyaml_missing)
```

The diagnostic names the absolute interpreter path probed and points at
`scripts/requirements-verify.txt`; the row's `reason` carries the same two facts.
No Python is imported for the probe beyond `yaml` itself, and the interpreter
being probed is never guessed. When no `python3` exists at all the reason is
`no python3 on PATH; role-pack parsing needs the PyYAML pinned in
scripts/requirements-verify.txt` — also a `fail`, never a skip, so the row is
never green while the toolchain is absent.

**S4 — one interpreter.** `VERIFY_PYTHON3="$(command -v python3 2>/dev/null || true)"`
is resolved once, immediately before `check_toolchain_python3`, which now prints
`python3: $VERIFY_PYTHON3`; the same variable is used by the PyYAML probe and by
`check_role_packs`'s own heredoc (`"$VERIFY_PYTHON3" - <<'PY'`). Observed with
`/opt/homebrew/bin` first on `PATH`: the toolchain row printed
`python3: /opt/homebrew/bin/python3` and the failing role-pack reason contained
the identical string (asserted by the selftest scenario). Nothing hardcodes an
interpreter path and the check resolves `python3` no second time.

**S5 — exemption gone, verification still green.** `.agent/gate-policy.json` now
carries three exemptions (`G-L`, `G-H`, `G-Q`, all `stage.tooling_absent`,
tracked by FJ-044/045/046); the `G-C` entry is deleted. `./scripts/verify-candidate FJ-047`
exits 0; `policy.exemptions.unknown` passes with reason `3 exemption(s) checked`
and `policy.exemptions.stale` passes. `toolchain.pyyaml_exempt` no longer appears
in the file.

**S6 — CI installs the pin.** `.github/workflows/ci.yml`'s `gauntlet` job gains
an `install pinned verification dependencies` step ordered before
`./scripts/verify-candidate` (line 98 vs line 104), which consumes
`scripts/requirements-verify.txt` — one `pip install -r` of the artifact, no
free-standing `pip install PyYAML`. See "CI install approach" below.

**S7 — design §4 corrected.** The `PyYAML when importable` clause and the
`skipped`/`toolchain.pyyaml_exempt` sentence are replaced by a paragraph stating
the pin (`PyYAML==6.0.3` in `scripts/requirements-verify.txt`), that the check is
always required with no conditional form, that an import failure fails closed
with `toolchain.pyyaml_missing`, and that `gate-policy.json` can never waive it
(§9.2 matches `skipped` rows only). Per the task's widened scope, §9.2's
"four exemptions" sentence now counts three and records that the `G-C` entry is
gone, and §13's "PyYAML is conditional" limitation was replaced by the limitation
that remains true (the interpreter is whichever `python3` `PATH` resolves first;
a missing import is fail-closed, not exempted). `docs/superpowers/specs/2026-09-26-deterministic-guard-stack-design.md`
§4 now says the former `G-C` entry blocking `toolchain.pyyaml_missing` is gone.

**S8 — selftest covers the new contract, still green.**
`pyyaml_exempt_with_task` (which asserted `code=toolchain.pyyaml_exempt`,
`result=skipped`, `exempted=true`, exit 0) is deleted and replaced by
`pyyaml_missing_fails_closed`: the existing `yaml_shim` plus a task-mode metadata
item, asserting exit non-zero, `report["status"] == "failed"`, and a
`check_role_packs` row of `required: true`, `result: fail`,
`code: toolchain.pyyaml_missing`, `exempted: false`, plus the console `FAIL`
line, the agreement of the reason with `command -v python3`, and the pointer to
`scripts/requirements-verify.txt`. `required_skipped_failure` no longer depends
on PyYAML: it now asserts §12's "required + skipped → blocked" row with a due
Cleaner tooling row (`stage.cleaner.tooling`, `stage.tooling_absent`, gate
`G-L`, `exempted: false`) under an exemption-free policy, and its stale comment
about the committed policy waiving `toolchain.pyyaml_missing` was rewritten with
it. `exemption_tooling_bootstrap` still asserts required + skipped + exemption →
success. `./scripts/selftest` → `summary: 75 passed, 0 failed`.

**S9 — `.agent/README.md`.** The `G-C` table row is deleted and the sentence now
reads "v1 commits exactly three exemptions". The three remaining rows are
unchanged and `toolchain.pyyaml_missing` / `toolchain.pyyaml_exempt` no longer
occur anywhere in the file.

## CI install approach and its justification

```yaml
      - name: install pinned verification dependencies
        run: |
          python3 -m venv "$RUNNER_TEMP/verify-venv"
          "$RUNNER_TEMP/verify-venv/bin/python" -m pip install \
            -r scripts/requirements-verify.txt
          echo "$RUNNER_TEMP/verify-venv/bin" >> "$GITHUB_PATH"
```

* PEP 668: `ubuntu-latest`'s `python3` is externally managed, so a system-wide
  `pip install` is refused. A venv is the narrow fix; it installs the pin
  without `--break-system-packages` and without touching the runner's managed
  interpreter. (`actions/setup-python` would also satisfy the criterion but adds
  a second toolchain and a second fetch; the venv is the smaller change.)
* It is the verifier's own interpreter that must have PyYAML, because
  `verify-candidate` resolves `python3` from `PATH` and never installs anything
  (non-goal). Appending the venv's `bin` to `$GITHUB_PATH` makes that resolution
  land on the pinned interpreter for the following step.
* The venv lives in `$RUNNER_TEMP`, outside the checkout, because
  `candidate-fingerprint` hashes untracked workspace content: a venv under the
  repository root would become thousands of untracked files inside the candidate
  digest. The candidate fingerprint in CI is therefore unaffected.
* The file header no longer claims the Go toolchain is the only network fetch;
  it now names the pinned PyYAML install as setup-time network use, and
  `scripts/README.md` records the `--no-index --find-links` path for runners
  without index access.

## Commands and outcomes

All from the repository root.

| # | command | exit | observation |
|---|---------|------|-------------|
| 1 | `./scripts/verify-candidate` | 0 | `summary: 20 passed, 0 failed, 0 skipped, 0 not_applicable` |
| 2 | `./scripts/verify-candidate FJ-047` | 0 | report written; `check_role_packs` `pass`/`agent.role_pack_invalid`/`exempted: false`; `policy.exemptions.unknown` reason `3 exemption(s) checked` |
| 3 | `./scripts/selftest` | 0 | `summary: 75 passed, 0 failed` (see residual risks for a one-off flake) |
| 4 | `bash -n scripts/verify-candidate` | 0 | no output |
| 5 | `grep -c '^PyYAML==6\.0\.3$' scripts/requirements-verify.txt` | 0 | `1` |
| 6 | json exemptions count | 0 | `3` |
| 7 | `grep -c 'toolchain.pyyaml_missing' .agent/gate-policy.json` | 1 | no match |
| 8 | `grep -c 'toolchain.pyyaml_exempt' .agent/gate-policy.json` | 1 | no match |
| 9 | `grep -n 'requirements-verify.txt' .github/workflows/ci.yml` | 0 | exactly one match, line 100, in the `gauntlet` job, above the verify step (line 104) |
| 10 | `python3 -c "import yaml;yaml.safe_load(open('.github/workflows/ci.yml'));print('yaml ok')"` | 0 | `yaml ok` |
| 11 | `grep -c 'PyYAML when importable' docs/superpowers/specs/2026-09-25-gauntlet-roles-design.md` | 1 | no match |
| 12 | `PATH=/opt/homebrew/bin:$PATH ./scripts/verify-candidate` | 1 | `19 passed, 1 failed, 0 skipped`; the role-pack block shows `FAIL (exit 1, code toolchain.pyyaml_missing)` (baseline: exit 1 with `SKIPPED`) |
| 13 | `PATH=/opt/homebrew/bin:$PATH ./scripts/verify-candidate FJ-047` | 1 | `13 passed, 1 failed, 0 skipped, 9 not_applicable` (baseline: exit 0 with the row rewritten to `toolchain.pyyaml_exempt`) |
| 14 | `python3 -m pip download --only-binary=:all: --no-deps -d /tmp/fj047-wheelhouse -r scripts/requirements-verify.txt` | 0 | `pyyaml-6.0.3-cp311-cp311-macosx_11_0_arm64.whl` saved |
| 15 | `python3 -m pip install --no-index --find-links /tmp/fj047-wheelhouse --target /tmp/fj047-target -r scripts/requirements-verify.txt` | 0 | installed with no index access |
| 16 | `PYTHONPATH=/tmp/fj047-target python3 -c "import yaml;print(yaml.__version__)"` | 0 | `6.0.3` |
| 17 | `grep -c 'toolchain.pyyaml' .agent/README.md` | 1 | no match |
| 18 | `go test ./...` | 0 | all packages ok (`cmd/guard`, `internal/*`, `test/integration`) |
| 19 | `python3 -m py_compile scripts/selftest` | 0 | compiles; the `__pycache__` it creates was removed |

The negative case #12/#13 is the regression this item closes: on this machine
the default `PATH` resolves `/Users/mak/.pyenv/shims/python3` (PyYAML 6.0.3
importable) while `/opt/homebrew/bin/python3` does not have it. Verification now
takes the verdict from whichever interpreter it actually resolved, so the same
tree can no longer be green in a task-mode metadata run while the dependency is
absent.

## Spec-silent observations

* **Version equality is not enforced at verification time (specifier O1).** An
  importable PyYAML other than 6.0.3 still passes locally; the pin is enforced by
  CI installing the artifact and by the documented install path. No equality
  check was added — that would be a new requirement, and the specifier's O1
  recommends leaving enforcement to CI plus the pin. The imported version is
  therefore not printed either (adding a second subprocess purely for a
  diagnostic was judged speculative).
* **Interpreter choice stays a `PATH` matter (O2).** Which `python3` runs is not
  specified; only the agreement requirement (S4) was implemented, not any
  preference for a venv or a version-pinned interpreter (explicit non-goal).
* **The `toolchain.pyyaml_missing` code survives (O9).** It moved from an
  exempted `skipped` row to a blocking `fail` row; design §9.3's `toolchain.*`
  inventory entry for `G-C` remains accurate.
* **Report churn.** The commanded verifier runs wrote six report files under
  `.agent/reports/FJ-047/` (timestamps 140117, 140515, 144635, 144657, 145107,
  145129). The last two are bound to the final candidate fingerprint
  `21a64a…` (145107 failed on the negative case, 145129 passed); the first four
  were written against the intermediate tree before the last CI comment edit.
  They are untracked; nothing in this change reads or writes report contents.
* **`.agent/work/FJ-047/state.json` is dirty but untouched by this stage** (it
  was already modified at pickup). No stage state was edited here.

## Residual risks

* **`scripts/selftest` flake, unrelated to this change.** One of three full
  suite runs in this session failed `fuzz_live_pass` (`guard.fuzz.crash`
  misclassified from a Go fuzz-worker start/finish failure) — the flake tracked
  by FJ-060. The scenario passed on re-run and both subsequent full suite runs
  were green (`75 passed, 0 failed`). Nothing in this change touches the guard
  fuzz runner.
* **CI is not executable here.** The workflow is valid YAML and the step order
  is verified by inspection and `grep`; the venv recipe follows the standard
  PEP 668-safe pattern but has not run on a real `ubuntu-latest` runner.
* **A developer machine with a different importable PyYAML version** (e.g.
  6.0.2) verifies green; only CI is guaranteed to use the pin.
* **Wheelhouse portability (O5).** Verified for this machine's interpreter
  (cp311, macOS arm64) only; the README says the wheelhouse must match the
  target platform/Python.
* **`.github/workflows/ci.yml` is now a network-fetching job at setup.** This is
  permitted by spec §23 (runtime needs no provider service) and is documented in
  the workflow header; a truly air-gapped runner would need the pre-populated
  wheelhouse path from `scripts/README.md`.
