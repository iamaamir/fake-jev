---
stage: specifier
task: FJ-047
inputFingerprint: 3d896a0a7d7ae6bd6f347b4166a74c6d23bbf59a384869c2cc3c2697e96e9dcc
outputFingerprint: 3d896a0a7d7ae6bd6f347b4166a74c6d23bbf59a384869c2cc3c2697e96e9dcc
taskFingerprint: 1f155d31629426f20bd4ea8a1d488c44f6657d6cd9a733b01bf4c73c9639fb7e
gitHead: 7eb1252
generatedAt: 2026-10-02T13:49:53Z
author: worker/FJ-047-specifier-refresh
---

# Specifier — FJ-047

## Scope as bounded here

Make PyYAML an explicit, pinned dependency of repository verification, make
role-pack validation run as a required check that fails closed when the
dependency is unavailable, remove the `toolchain.pyyaml_missing` exemption from
`.agent/gate-policy.json`, and make CI consume the pin. The only file that may
change public behaviour is `scripts/verify-candidate`; everything else is the
pin artifact, its documentation, the CI workflow, and the design §4 sentence
that the change falsifies.

State of the tree at `7eb1252` (read from the files, not assumed):

- `scripts/verify-candidate:667` gates the check:
  `if python3 -c 'import yaml' ... then run_check ... else skip_check
  "PyYAML not importable (design §4 conditional)" "toolchain.pyyaml_missing"`.
  So today the check is required in name but conditional in fact.
- `check_code` maps `check_role_packs` to `agent.role_pack_invalid`;
  `run_check` derives a row's code from `check_code`, while `skip_check` and
  `na_check` accept an explicit code override. A row can therefore carry
  `toolchain.pyyaml_missing` only while it is `skipped` — and only `skipped`
  required rows are waivable.
- The exemption engine (`scripts/verify-candidate`, rows→policy application)
  waives a row only when `result == "skipped"`, `required` is true, the gate
  matches, `blocks` contains the row's code, and the item's class is in
  `scope`. A `fail` row can never be exempted by any `gate-policy.json` entry.
  Taskless runs apply no exemptions at all (there is no class).
- `check_toolchain_python3` (`scripts/verify-candidate:279`) only asserts that
  some `python3` exists on `PATH` and prints `python3: <command -v python3>`.
  It does not assert anything about PyYAML, and nothing pins the interpreter
  path for later use.
- `.agent/gate-policy.json` carries four exemptions, the fourth being
  `{gate: G-C, blocks: ["toolchain.pyyaml_missing"], scope: [product, metadata],
  code: toolchain.pyyaml_exempt, trackedBy: FJ-047}`.
- Observed baseline behaviour on this machine (commands run, output read):
  - `./scripts/verify-candidate` → exit 0, `summary: 20 passed, 0 failed,
    0 skipped, 0 not_applicable`.
  - `PATH=/opt/homebrew/bin:$PATH ./scripts/verify-candidate` → exit 1; the
    `role packs match the role-pack contract` row prints
    `SKIPPED: PyYAML not importable (design §4 conditional)`, summary
    `19 passed, 0 failed, 1 skipped`. `/opt/homebrew/bin/python3` exists and
    `import yaml` fails there; the pyenv `python3` on the default `PATH`
    (`/Users/mak/.pyenv/shims/python3` → 3.11.14) imports PyYAML 6.0.3.
  - In task mode with a metadata item the G-C exemption rewrites that skipped
    row to `toolchain.pyyaml_exempt` (`exempted: true`), which is
    non-blocking: the selftest scenario `pyyaml_exempt_with_task` asserts exit
    0 for exactly that setup, and `./scripts/selftest pyyaml` exits 0.
  - `./scripts/selftest` → exit 0, `summary: 75 passed, 0 failed`.
  - `python3 -m pip index versions PyYAML` lists `6.0.3` as latest; the
    installed interpreter reports 6.0.3.
- `scripts/selftest` copies the committed `.agent/gate-policy.json` and the
  `FJ-044/045/046/047` state files into every fixture (`build_pristine`), and
  its `yaml_shim` makes `import yaml` fail through `PYTHONPATH`. Removing the
  G-C entry and changing the check therefore changes fixture behaviour:
  `required_skipped_failure` and `pyyaml_exempt_with_task` depend on the
  removed path.
- `.agent/README.md` also lists four exemptions. It is now inside
  `allowedFiles` (scope amendment 2, 2026-10-01), so its gate-policy table and
  its "four exemptions" sentence must be corrected by this item; see S9 and
  observation O6.

Allowed files: `scripts/`, `docs/`, `.agent/gate-policy.json`,
`.github/workflows/ci.yml`, `.agent/README.md`. Class of the item is `metadata`;
`requiredStages` are `specifier, coder, cleaner, qa`.

## Observable acceptance criteria

Each criterion names what an observer sees and the event that would falsify it.
"Coder" means the implementer of this item; nothing here prescribes private
implementation shape beyond what is observable.

### P1 — Pinned PyYAML version recorded, with rationale and an offline path

**Observable.** `scripts/requirements-verify.txt` exists and its only
requirement line is `PyYAML==6.0.3` — an exact `==` pin, no range, no second
package. `scripts/README.md` documents, under a verification-toolchain heading,
the pinned version, the rationale, and the offline installation path.
`scripts/README.md` already states that `agent-context` requires `python3`; the
new text extends that section rather than creating a new document.

**Rationale to record (the version choice).**

1. 6.0.3 is the version importable by the `python3` that verification actually
   runs under today (`/Users/mak/.pyenv/versions/3.11.14/bin/python3`), so the
   pin describes the environment the role packs are already parsed in instead
   of forcing an environment change.
2. 6.0.3 is the current release on PyPI (`python3 -m pip index versions
   PyYAML` reports `LATEST: 6.0.3`), so a fresh CI runner can satisfy it.
3. Strict `==` is deliberate: a range would let a different importable version
   satisfy verification without being visible anywhere.

**Offline installation path (documented and executable).** Two steps, both
named in `scripts/README.md`: populate a wheel directory on a networked
machine, then install from it with `--no-index --find-links` (no network, no
index access). Exact commands and exit codes are listed below.

**Falsified by.** A missing pin file; a pin other than `6.0.3`; a range or
unpinned requirement; a documented install path that needs network access.

### S2 — `check_role_packs` is unconditional and required

**Observable.** With PyYAML importable, the check runs on the required G-C
path on every invocation, with no code path that can turn it into `skipped`.
In the report, its row is
`{name: check_role_packs, gate: G-C, required: true, result: pass,
code: agent.role_pack_invalid, exempted: false}`.

**Falsified by.** Any `skipped` or `not_applicable` row for `check_role_packs`
in any environment; a surviving construct of the form
`if python3 -c 'import yaml' ... else skip_check ...`; an invocation of the
check that does not happen at all.

### S3 — Exact failure behaviour when PyYAML is genuinely unavailable

When the resolved interpreter cannot `import yaml`, the run fails closed. This
is the behaviour the removed exemption used to soften, and it is the behaviour
the new selftest scenario asserts.

| field | value |
|-------|-------|
| row `name` | `check_role_packs` |
| row `gate` | `G-C` |
| row `required` | `true` |
| row `result` | `fail` |
| row `code` | `toolchain.pyyaml_missing` |
| row `exempted` | `false` |
| row `exitCode` | non-zero |
| process exit | `1` (taskless and task mode alike) |

**Console.** Under the header `==> role packs match the role-pack contract` the
next line contains `FAIL` and the substring `toolchain.pyyaml_missing`, in the
same shape the existing helpers use (`FAIL (exit <n>, code
toolchain.pyyaml_missing)`). The diagnostic names the absolute interpreter path
probed and points at `scripts/requirements-verify.txt`; if the row carries a
`reason`, it carries the same two facts.

**Why this code** (record for the cleaner/QA): `toolchain.pyyaml_missing` is
the code the design and the removed exemption already name, so an agent keying
on code plus check name sees the same condition as before with a now-blocking
verdict; the exemption removal is complete precisely because the engine matches
only `skipped` rows, so a `fail` with this code is unwaivable. The alternative
of reporting the generic `agent.role_pack_invalid` would make an environment
defect indistinguishable, by code alone, from a genuinely malformed role pack.

**Falsified by.** A `skipped` row; an exempted row; a different code; exit 0 in
a task-mode run of a metadata item; a diagnostic that does not say which
interpreter lacks PyYAML.

### S4 — The toolchain check and the PyYAML probe name the same interpreter

**Observable.** The interpreter the PyYAML probe interrogates is the same
absolute path that the `toolchain: python3 available` check reports, resolved
once from `PATH` after that check runs and reused by the probe. Nothing
hardcodes an interpreter path, and no `python3` is resolved a second time.

**Exercise.** With `/opt/homebrew/bin` first on `PATH`, the toolchain row
prints `python3: /opt/homebrew/bin/python3` and the failing role-pack
diagnostic contains the identical string (see the exact commands below). Under
the default `PATH` the pair prints `/Users/mak/.pyenv/shims/python3`. When
`python3` is absent entirely, `check_toolchain_python3` fails and
`check_role_packs` must not be green either — it may not become `skipped`.

**Falsified by.** The two checks naming different interpreters; a probe that
resolves an interpreter independent of the toolchain check; a green role-pack
row while no `python3` exists.

### S5 — The G-C exemption is gone and verification still exits 0

**Observable.** `.agent/gate-policy.json` contains no exemption whose `blocks`
includes `toolchain.pyyaml_missing`; the file stays valid JSON with the three
remaining exemptions (`G-L`, `G-H`, `G-Q`, all `stage.tooling_absent`) otherwise
unchanged, and each still resolves to an existing, non-complete tracker (FJ-044,
FJ-045, FJ-046 are `planned`). `./scripts/verify-candidate FJ-047` exits 0, with
`policy.exemptions.unknown` and `policy.exemptions.stale` still reporting
`result: pass` and the unknown row's reason reporting `3 exemption(s) checked`.
The string `toolchain.pyyaml_exempt` no longer appears anywhere in
`.agent/gate-policy.json`.

**Falsified by.** A surviving G-C entry; a malformed policy file; a
`policy.exemption_*` failure; a non-zero `./scripts/verify-candidate FJ-047`.

### S6 — CI installs the pin

**Observable.** In `.github/workflows/ci.yml`, the `gauntlet` job installs the
pinned artifact in a step ordered **before** the `./scripts/verify-candidate`
step, and that step consumes `scripts/requirements-verify.txt` (no free-standing
`pip install PyYAML` with an unpinned version). The workflow remains valid YAML
and the job still runs the verifier task-less.

**Falsified by.** A verifier step that runs before any install of the pin; a
CI install that does not consume the pinned artifact; invalid YAML; an install
step located in a job that does not run the verifier (the `test` and `build`
jobs do not run it).

### S7 — The design §4 sentence is corrected

**Observable.** In `docs/superpowers/specs/2026-09-25-gauntlet-roles-design.md`
§4, the parsing paragraph no longer claims PyYAML is conditional. It states
that PyYAML is a pinned verification dependency (naming
`scripts/requirements-verify.txt`), that the check is always required, and that
an import failure fails closed with code `toolchain.pyyaml_missing` and is not
waivable by `gate-policy.json`. The exact old clause `PyYAML when importable`
is gone from §4, and no sentence in §4 still describes a `skipped` result or a
`toolchain.pyyaml_exempt` waiver for this check.

**Falsified by.** The old sentence surviving; §4 still promising `skipped` or a
waiver.

### S8 — `scripts/selftest` stays green and covers the new failure

**Observable.** `./scripts/selftest` exits 0 with `0 failed` (75 scenarios
today, count may grow). The scenario `pyyaml_exempt_with_task`, which asserts
`code=toolchain.pyyaml_exempt`, `result=skipped`, `exempted=true`, exit 0, no
longer exists. A scenario asserts the fail-closed behaviour with the existing
`yaml_shim` (`PYTHONPATH` shadow) plus a task-mode metadata item: exit
non-zero, and the `check_role_packs` row is `required: true`, `result: fail`,
`code: toolchain.pyyaml_missing`, `exempted: false`.

The §12 matrix row "required + skipped → verify failure" must remain asserted
by a scenario that does **not** depend on PyYAML (for example a due Cleaner
tooling row under an exemption-free policy), and "required + skipped +
exemption → success" must remain asserted (`exemption_tooling_bootstrap`). The
stale comment that says the committed policy waives `toolchain.pyyaml_missing`
is corrected with the scenario it explains.

**Falsified by.** Any scenario still asserting `toolchain.pyyaml_exempt`; a
red selftest; the §12 required+skipped row losing its scenario.

### S9 — `.agent/README.md` no longer asserts the removed G-C exemption

**Observable.** In `.agent/README.md`, the "Gate policy" section no longer
carries the G-C table row (the row formerly at line ~187:
`| G-C | toolchain.pyyaml_missing | product, metadata |
 toolchain.pyyaml_exempt | FJ-047 |`), and the sentence that precedes the table
says v1 commits **three** exemptions rather than four. The three remaining rows
(`G-L`, `G-H`, `G-Q`, all `stage.tooling_absent`) stay unchanged, and the
substrings `toolchain.pyyaml_missing` and `toolchain.pyyaml_exempt` no longer
appear anywhere in the file. This is the doc-consistency obligation added by
scope amendment 2; `.agent/README.md` is now inside `allowedFiles`, so the
correction lands in the same change as the policy edit.

**Falsified by.** A surviving G-C table row; the phrase "v1 commits exactly four
exemptions" surviving; any remaining occurrence of `toolchain.pyyaml_missing`
or `toolchain.pyyaml_exempt`.

## Exact commands and expected exit codes

Run from the repository root. "before" values are the observed baseline at
`7eb1252`; every other expectation is the acceptance target.

| # | command | expected exit | expected observation |
|---|---------|---------------|----------------------|
| 1 | `./scripts/verify-candidate` | 0 | `summary: … 0 failed` |
| 2 | `./scripts/verify-candidate FJ-047` | 0 | report written to `.agent/reports/FJ-047/`; no exempted row for `check_role_packs` |
| 3 | `./scripts/selftest` | 0 | `0 failed` |
| 4 | `bash -n scripts/verify-candidate` | 0 | no output |
| 5 | `grep -c '^PyYAML==6\.0\.3$' scripts/requirements-verify.txt` | 0 | prints `1` |
| 6 | `python3 -c "import json;print(len(json.load(open('.agent/gate-policy.json'))['exemptions']))"` | 0 | prints `3` |
| 7 | `grep -c 'toolchain.pyyaml_missing' .agent/gate-policy.json` | 1 | no match (grep exits 1 when nothing matches) |
| 8 | `grep -c 'toolchain.pyyaml_exempt' .agent/gate-policy.json` | 1 | no match |
| 9 | `grep -n 'requirements-verify.txt' .github/workflows/ci.yml` | 0 | one match, in the `gauntlet` job, above the verify step |
| 10 | `python3 -c "import yaml;yaml.safe_load(open('.github/workflows/ci.yml'));print('yaml ok')"` | 0 | prints `yaml ok` |
| 11 | `grep -c 'PyYAML when importable' docs/superpowers/specs/2026-09-25-gauntlet-roles-design.md` | 1 | no match |
| 12 | `PATH=/opt/homebrew/bin:$PATH ./scripts/verify-candidate` | 1 | `FAIL` line for `check_role_packs` carrying `toolchain.pyyaml_missing`; before: exit 1 with `SKIPPED` |
| 13 | `PATH=/opt/homebrew/bin:$PATH ./scripts/verify-candidate FJ-047` | 1 | before: exit 0 (row rewritten to `toolchain.pyyaml_exempt`) |
| 14 | `python3 -m pip download --only-binary=:all: --no-deps -d /tmp/fj047-wheelhouse -r scripts/requirements-verify.txt` | 0 | wheel for the runner's interpreter appears in `/tmp/fj047-wheelhouse` (networked step) |
| 15 | `python3 -m pip install --no-index --find-links /tmp/fj047-wheelhouse --target /tmp/fj047-target -r scripts/requirements-verify.txt` | 0 | resolves without index access (offline step) |
| 16 | `PYTHONPATH=/tmp/fj047-target python3 -c "import yaml;print(yaml.__version__)"` | 0 | prints `6.0.3` |
| 17 | `grep -c 'toolchain.pyyaml' .agent/README.md` | 1 | no match (the G-C row and its code are gone) |

Commands 1–3 and 12–13 are the behavioural evidence; 5–11 and 17 are the
artifact evidence; 14–16 prove the documented offline path works rather than
merely existing. Command 13 is the regression this item closes: today the exemption
turns a missing dependency into a green metadata run.

## Non-goals

- No other new dependency. `jsonschema` stays out (design §11), and the Go
  module graph, `go.mod`/`go.sum`, and the product runtime are untouched.
  PyYAML remains the only Python dependency of verification.
- `./scripts/verify-candidate` must not install anything: verification stays
  offline and side-effect-free; installing the pin is CI's job (S6).
- No change to exemption matching, to the gate/status vocabularies, or to
  `verify-candidate`'s exit-code contract (still 0 or 1).
- No interpreter selection policy beyond S4 (no venv, no version-pinned
  interpreter, no search for whichever `python3` happens to have PyYAML).
- No new acceptance-catalog IDs (see O7).
- No edits outside `allowedFiles`. In particular `AGENTS.md` stays untouched
  (O6).

## Observations (what the artifacts leave undefined)

**O1 — Version-equality enforcement is undefined.** Nothing specifies whether
an importable PyYAML whose version differs from the pin must fail verification.
These criteria require the pin to be recorded and installed by CI, and require
failure only when PyYAML is genuinely unavailable (S3). A version-equality
failure at verification time would be a new requirement, not implied here;
recommend reporting the imported version in the check output for diagnosis and
leaving enforcement to CI plus the pin.

**O2 — Interpreter choice is undefined.** Design §4 says "PyYAML when
importable" and names no interpreter. This item requires only *agreement*
between the toolchain check and the PyYAML probe (S4), not a preferred
interpreter. Consequence to state honestly: which `python3` runs remains a
`PATH` matter, but *whether a missing dependency is silent* no longer does —
that is the fail-closed property the objective asks for.

**O3 — CI network policy is not contradicted, but the comment is.** Spec §23
requires golden contracts to run offline; the `ci.yml` header comment says the
Go toolchain is the only network fetch. Installing the pin adds a PyPI fetch at
job setup. §2's "CI must not depend on external services" is about runtime
provider services, and dependency installation at setup is not forbidden by the
specification; this item therefore corrects the comment to name the pinned
PyYAML install as well, and documents the wheelhouse path (P1) so a
network-restricted runner can satisfy the pin without index access.

**O4 — Externally managed Python on CI is undefined.** On `ubuntu-latest`,
`python3 -m pip install` can fail because the interpreter is externally managed
(PEP 668). The specification does not define a mechanism. The criterion is
only that the `gauntlet` job installs the pinned artifact before the verifier;
`actions/setup-python`, a virtualenv, or an explicitly justified pip flag are
all acceptable implementations of that criterion.

**O5 — The wheel is platform/interpreter specific.** Observed: `pip download`
produced `pyyaml-6.0.3-cp311-cp311-macosx_11_0_arm64.whl` here. A wheelhouse
meant for CI must be populated for the runner's platform and Python version,
or the offline install is used only for equally-shaped offline workstations.
The pin itself (`==6.0.3`) is platform-neutral.

**O6 — Statements that become false when the policy entry is removed.** The
design §4 sentence (§4, S7) and `.agent/README.md` (S9) are in this item's
acceptance and are corrected in the same change. The remaining committed
statements below are recorded here so the discrepancy is visible rather than
silently carried; they are **still tracked** (not corrected by this item):

- design §9.2 "v1 commits four exemptions: … and `G-C` with `blocks:
  ["toolchain.pyyaml_missing"]` (both classes, reported as
  `toolchain.pyyaml_exempt`)" — the count becomes three; still tracked.
- design §13 "**PyYAML is conditional.** Role-pack validation is `skipped`
  (visible, exempt while tracked) when PyYAML is missing; pin it when this
  becomes a required CI gate" — this item is that pin; the bullet is the
  limitation being removed. §13 is one of the item's own `specReferences`;
  still tracked.
- `docs/superpowers/specs/2026-09-26-deterministic-guard-stack-design.md`
  "Today's bootstrap entries: … the G-C entry blocks
  `toolchain.pyyaml_missing`" — the entry no longer exists; this guard-stack
  design wording is still tracked.
- `.agent/README.md` "v1 commits exactly four exemptions" plus its G-C table
  row — **now inside `allowedFiles`** (scope amendment 2, 2026-10-01) and
  corrected by S9 in this item.
- `docs/superpowers/plans/2026-09-25-gauntlet.md` — a dated implementation
  plan; historical record, deliberately not corrected.

Recommendation: `.agent/README.md` is handled by S9 in this change. Keep the
guard-stack design §4 wording and the gauntlet-roles design §9.2/§13 wording
tracked — correct them by widening this item's doc-consistency scope or by
opening a follow-up item — so no committed document contradicts the committed
policy.

**O7 — The acceptance catalog has no criterion for this dependency.** The
closest entries are C-QUAL-005 (offline) and C-QUAL-006 (committed
machine-readable artifacts), which are the two traces below. Adding a new
`C-QUAL-*` id for the Python verification dependency would extend the catalog
beyond §23/§45 and is out of scope.

**O8 — Selftest is not itself a `verify-candidate` check.** `verify-candidate`
requires `scripts/selftest` to exist and be executable, but does not run it, so
S8 must be exercised explicitly rather than inferred from command 1.

**O9 — The `toolchain.pyyaml_missing` code survives the exemption.** Removing
the exemption does not remove the code; the code moves from an exempted
`skipped` row to a blocking `fail` row. Recorded so the change is not read as
"delete the code".

## Traces

- C-QUAL-005 — §23, golden contracts run offline with no credentials, model, or
  network: the verification dependency is pinned in a repository artifact with
  a documented `--no-index --find-links` installation path, so verification
  stays offline after the dependency is present (P1, S6; CI setup fetch noted
  in O3).
- C-QUAL-006 — §45, required machine-readable contract artifacts are committed
  and reviewable: `scripts/requirements-verify.txt` and the updated
  `.agent/gate-policy.json` are the committed, reviewable artifacts that carry
  the pin and the exemption removal (P1, S5).

Design references (not catalog ids): `docs/superpowers/specs/2026-09-25-gauntlet-roles-design.md`
§4 (role-pack contract and the conditional-skip sentence corrected by S7), §9.2
(exemption matching, which is why a `fail` row cannot be waived — S3, S5),
§9.3 (G-C code inventory, `toolchain.*` — S3), §11 (`jsonschema` stays out;
PyYAML is the only Python dependency — non-goals), §12 (the required+skipped
and exemption rows S8 must keep asserted), §13 (the PyYAML limitation this item
removes — O6).

This artifact states observable behaviour and traces; it declares no
implementation outcome.
