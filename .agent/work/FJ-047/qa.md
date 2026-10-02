---
stage: qa
task: FJ-047
inputFingerprint: 21a64a50bf2ef95509f639bef1bdb0c944680121c8a9ae720998bec119f504dc
outputFingerprint: 21a64a50bf2ef95509f639bef1bdb0c944680121c8a9ae720998bec119f504dc
taskFingerprint: 1f155d31629426f20bd4ea8a1d488c44f6657d6cd9a733b01bf4c73c9639fb7e
gitHead: 7eb1252
generatedAt: 2026-10-02T15:25:30Z
author: worker/FJ-047-qa
---

# QA — FJ-047

Fresh session; the coder author is `worker/FJ-047-coder`, so this run is
independent of the authoring session. The candidate tree was inspected and run
as-is. Nothing in the repository was edited: the only in-repo writes this stage
performed are the report files the mandated `./scripts/verify-candidate` runs
emit into `.agent/reports/FJ-047/` (`report-20261002T150746Z.json` from this
stage). `.agent/work/FJ-047/state.json` was already modified at pickup
(`mtime 2026-10-02 13:48:51Z`, before this session) and was not touched here.

All negative work was done outside the checkout:

- `/private/tmp/fj047-qa` — rsync copy of the candidate worktree (`.git`
  excluded, re-`git init` + commit), used for the fail-closed reproductions,
  the exemption re-add, the interpreter trace, the no-install shim run, the CI
  venv recipe simulation and the first mutation test.
- `/private/tmp/fj047-qa-baseline` — same copy with
  `scripts/verify-candidate` replaced by `git show 7eb1252:scripts/verify-candidate`
  and `gate-policy.json` at its pre-change (4-exemption) content, used only as
  the before/after contrast.
- `/private/tmp/fj047-qa-mutate`, `/private/tmp/fj047-qa-mutate2` — copies
  with deliberate regressions injected into `scripts/verify-candidate`, used to
  test whether the reworked `scripts/selftest` scenarios actually detect them.
- `/private/tmp/fj047-qa-shim` (PYTHONPATH `yaml.py` that raises `ImportError`),
  `/private/tmp/fj047-nopy-bin` (PATH with every tool except `python*`),
  `/private/tmp/fj047-noinstall-bin` (`pip`/`pip3`/`curl`/`wget`/`nc` shims that
  exit 127), `/private/tmp/fj047-qa-wheelhouse`, `/private/tmp/fj047-qa-target*`,
  `/private/tmp/fj047-qa-venv`, `/private/tmp/fj047-qa-old`.

## Mandated command battery (repository root, unmodified tree)

| # | command | exit | observed |
|---|---------|------|----------|
| 1 | `bash -n scripts/verify-candidate scripts/selftest` | 0 | no output |
| 2 | `./scripts/selftest` | 0 | `summary: 75 passed, 0 failed` (75 `@scenario` definitions) |
| 3 | `./scripts/verify-candidate` | 0 | `summary: 20 passed, 0 failed, 0 skipped, 0 not_applicable`; final line `RESULT: PASS` |
| 4 | `./scripts/verify-candidate FJ-047` | 0 | `summary: 14 passed, 0 failed, 0 skipped, 9 not_applicable`; report `.agent/reports/FJ-047/report-20261002T150746Z.json`, bound to `candidateFingerprint 21a64a50…` |
| 5 | `PATH=/opt/homebrew/bin:$PATH ./scripts/verify-candidate` | 1 | `summary: 19 passed, 1 failed, 0 skipped, 0 not_applicable`; role-pack block is a `FAIL` row, not a `SKIPPED` row |
| 6 | `go test ./... -count=1` | 0 | `cmd/guard`, `internal/cli`, `internal/compat/jev/v1`, `internal/config`, `internal/control`, `internal/engine`, `internal/host/http`, `test/integration` all `ok` |
| 7 | `./scripts/candidate-fingerprint candidate` | 0 | `21a64a50bf2ef95509f639bef1bdb0c944680121c8a9ae720998bec119f504dc` — equals the header `inputFingerprint`/`outputFingerprint` |

`scripts/verify-candidate` task mode row read from the report of command 4:

```json
{"name":"check_role_packs","gate":"G-C","required":true,"result":"pass",
 "exitCode":0,"code":"agent.role_pack_invalid","exempted":false,
 "command":"role packs match the role-pack contract","reason":null}
```

and `policy.exemptions.unknown` `result: pass`, `reason: "3 exemption(s) checked"`;
`policy.exemptions.stale` `result: pass`. No row in that report carries
`exempted: true`.

Console of command 5 (verbatim, relevant slice):

```text
==> toolchain: python3 available
python3: /opt/homebrew/bin/python3
    PASS
...
==> role packs match the role-pack contract
    PyYAML not importable by /opt/homebrew/bin/python3 (pinned in scripts/requirements-verify.txt; install with: /opt/homebrew/bin/python3 -m pip install -r scripts/requirements-verify.txt)
    FAIL (exit 1, code toolchain.pyyaml_missing)
```

## Per-criterion observations

### C1 — the pin is exact and singular; offline recipe works; documented install matches

- `scripts/requirements-verify.txt` is the only `requirements*.txt` in the
  repository (`find . -name 'requirements*.txt'` → 1 hit). Its only
  non-comment, non-blank line is line 20, byte sequence
  `P y Y A M L = = 6 . 0 . 3 \n` (`sed -n 20p | od -c`) — column 0, no trailing
  space, no CR, no range operator, no second package. `grep -c '^PyYAML==6\.0\.3$'`
  → `1`.
- Observed offline path (networked populate, then index-free install), run from
  the repository root exactly as documented in `scripts/README.md`:
  - `python3 -m pip download --only-binary=:all: --no-deps -d /private/tmp/fj047-qa-wheelhouse -r scripts/requirements-verify.txt`
    → exit 0; `pyyaml-6.0.3-cp311-cp311-macosx_11_0_arm64.whl` (175577 B) saved.
  - `python3 -m pip install --no-index --find-links /private/tmp/fj047-qa-wheelhouse --target /private/tmp/fj047-qa-target -r scripts/requirements-verify.txt`
    → exit 0, `Looking in links: …` only, no index access; then
    `PYTHONPATH=/private/tmp/fj047-qa-target python3 -c "import yaml;print(yaml.__version__)"`
    → `6.0.3`.
  - Repeating that install with a dead proxy
    (`HTTP_PROXY=HTTPS_PROXY=PIP_PROXY=http://127.0.0.1:9`) still exits 0 and
    yields `6.0.3`, i.e. the recipe does not reach an index.
- Documented normal install: `scripts/README.md:40` is
  `python3 -m pip install -r scripts/requirements-verify.txt` (also recorded in
  the artifact's own header, `scripts/requirements-verify.txt:13`). Executing it
  (`--target /private/tmp/fj047-qa-target3`) exits 0 and imports `6.0.3`, so the
  documented install and the pin agree. There is no free-standing
  `pip install PyYAML` anywhere in `scripts/` or `.github/` (grep → none).
- The offline variant in the docs (`scripts/README.md:47-50`,
  `scripts/requirements-verify.txt:16`) is the same `--no-index --find-links`
  form that was executed.

### C2 — `check_role_packs` is required with no conditional skip path

Static: the pre-change `if python3 -c 'import yaml' … else skip_check …
"toolchain.pyyaml_missing"` construct is absent from `scripts/verify-candidate`.
The call site (`scripts/verify-candidate:683-697`) has exactly two outcomes:
`run_check "$ROLE_PACK_DISPLAY" check_role_packs` or a literal
`record_row "check_role_packs" "$ROLE_PACK_DISPLAY" "G-C" 1 "fail" 1
"toolchain.pyyaml_missing" "$ROLE_PACK_REASON"`. `check_code` still maps the
name to `agent.role_pack_invalid` (`:134`), and the row is on the required G-C
path. No `skip_check`/`na_check` call exists for this check name anywhere
(`grep -rn check_role_packs` across the tree: the only call sites are `:691`
and `:695`).

Exhaustive inventory of the two strings, every hit, with a staleness reading
(`.git` excluded). "Historical" = a dated record or evidence artifact that is
not read as current behaviour; "live" = read as current behaviour today.

`toolchain.pyyaml_missing`:

| file | hits | reading |
|------|------|---------|
| `scripts/verify-candidate` | 3 | live — the failure code of the new `fail` row (`:679` comment, `:693` console, `:696` row) |
| `scripts/selftest` | 2 | live — `pyyaml_missing_fails_closed` asserts the row code and the console line |
| `scripts/requirements-verify.txt` | 1 | live — names the failure code the pin prevents |
| `scripts/README.md` | 1 | live — documents the fail-closed code |
| `docs/superpowers/specs/2026-09-25-gauntlet-roles-design.md` | 3 | live, correct — §4/§13 describe the blocking `fail`; §9.2 past-tense record |
| `docs/superpowers/specs/2026-09-26-deterministic-guard-stack-design.md` | 1 | live, correct — §4 records the `G-C` entry as gone |
| `.agent/work/FJ-047/state.json` | 2 | live acceptance text (the code is the condition to remove/expose) |
| `.agent/work/FJ-047/{specifier,coder,cleaner}.md` | 41 | evidence artifacts of this item |
| `.agent/reports/FJ-047/report-{140515,144635,145107}Z.json` | 3 | recorded report rows (negative-run evidence) |
| `docs/superpowers/plans/2026-09-25-gauntlet.md` | 10 | stale relative to current behaviour, historical dated plan (deliberately not rewritten) |

`toolchain.pyyaml_exempt`:

| file | hits | reading |
|------|------|---------|
| `docs/superpowers/specs/2026-09-25-gauntlet-roles-design.md` | 1 (`:561`) | live document, explicit past tense: "existed until FJ-047 … the entry is gone" — not an assertion that a waiver exists |
| `.agent/work/FJ-004/qa.md` | 1 (`:55`) | **stale** — says the `toolchain.pyyaml_exempt` (FJ-047) entry "is noted in the gate policy"; the entry no longer exists. FJ-004 is `complete`; this is a frozen evidence artifact of an earlier item, not a live contract |
| `.agent/work/FJ-047/{specifier,coder,cleaner}.md` | 20 | evidence artifacts of this item (describing the removal) |
| `docs/superpowers/plans/2026-09-25-gauntlet.md` | 3 | stale, historical dated plan |

No hit remains in `.agent/gate-policy.json`, `.agent/README.md`,
`.agent/schema/`, `scripts/README.md`, `scripts/selftest`,
`scripts/requirements-verify.txt`, `.github/workflows/ci.yml`, or
`scripts/verify-candidate`. The only normalized live documents that mention the
old code are the two design docs, both in explicitly past-tense clauses.

### C3 — fail-closed is real and unwaivable (throwaway-copy negative evidence)

Shim used: `PYTHONPATH=/private/tmp/fj047-qa-shim`, where `yaml.py` contains
`raise ImportError(...)`. Sanity: `PYTHONPATH=… python3 -c 'import yaml'` raises
`ImportError`, while `command -v python3` still resolves
`/Users/mak/.pyenv/shims/python3` (the shim changes only importability, not
interpreter resolution).

**A — taskless, PyYAML unavailable (copy).** `PYTHONPATH=… ./scripts/verify-candidate`
→ exit 1; `summary: 19 passed, 1 failed, 0 skipped, 0 not_applicable`; the
toolchain row printed `python3: /Users/mak/.pyenv/shims/python3`; the role-pack
block printed the reason naming that same path plus
`FAIL (exit 1, code toolchain.pyyaml_missing)`.

**B — task mode, metadata item `FJ-047`, PyYAML unavailable (copy).**
`PYTHONPATH=… ./scripts/verify-candidate FJ-047` → exit 1,
`summary: 13 passed, 1 failed, 0 skipped, 9 not_applicable`. Report row
(`report-20261002T151213Z.json`):

```json
{"name":"check_role_packs","gate":"G-C","required":true,"result":"fail",
 "exitCode":1,"code":"toolchain.pyyaml_missing","exempted":false,
 "command":"role packs match the role-pack contract",
 "reason":"PyYAML not importable by /Users/mak/.pyenv/shims/python3 (pinned in scripts/requirements-verify.txt; install with: /Users/mak/.pyenv/shims/python3 -m pip install -r scripts/requirements-verify.txt)"}
```

Status in that report: `failed`; `policy.class: metadata`; the exemption rows
`policy.exemptions.unknown` (`3 exemption(s) checked`) and
`policy.exemptions.stale` both `result: pass`; zero rows carry `exempted: true`.

**C — the removed exemption re-added verbatim, then the same run (copy).**
`(cd /Users/mak/git/fake-jev-FJ-047 && git show 7eb1252:.agent/gate-policy.json)`
was written over the copy's `.agent/gate-policy.json`, restoring the exact old
entry `{gate: G-C, blocks: ["toolchain.pyyaml_missing"], scope: [product,
metadata], code: toolchain.pyyaml_exempt, reason: "PyYAML not pinned as a
verification dependency yet", trackedBy: FJ-047}` (file now parses with 4
exemptions). Re-running `PYTHONPATH=… ./scripts/verify-candidate FJ-047` →
**exit 1**, `summary: 13 passed, 1 failed, 0 skipped, 9 not_applicable`,
`policy.class: metadata`, `policy.exemptions.unknown: pass — "4 exemption(s)
checked"`, and the row is unchanged: `required true`, `result fail`, `code
toolchain.pyyaml_missing`, `exempted false`; still no `exempted` row in the
report. A policy entry whose gate, blocks and scope all match the condition
therefore cannot soften this row.

**Contrast that makes case C non-vacuous (baseline copy).** The same shim, the
same restored 4-exemption policy, and `scripts/verify-candidate` from `7eb1252`
produce exit **0**, `RESULT: PASS`, and the row
`{required: true, result: "skipped", code: "toolchain.pyyaml_exempt",
exempted: true, reason: "PyYAML not pinned as a verification dependency yet"}`
— i.e. the deleted entry was load-bearing, and its matching path (skipped rows
only) is exactly what the new `fail` row is outside of.

The mechanism is visible in the candidate source: the exemption loop
(`scripts/verify-candidate:1762`) begins
`if row["result"] != "skipped" or not row["required"]: continue`, and
`GAUNTLET_SELFTEST_NA` (`:1786`) only rewrites `pass`/`skipped` rows, after
which a required `not_applicable` becomes `fail`/`gate.invalid_result` (`:1796`).
No transform in the engine turns this row into a `skipped` row.

**Interpreter named in the message vs the toolchain row.** In every negative
observation the two strings are identical:
`/opt/homebrew/bin/python3` (repository run, command 5) and
`/Users/mak/.pyenv/shims/python3` (copy runs A/B). The row `reason` carries the
same path as the console message.

**No-`python3` clause (S4).** With `PATH=/private/tmp/fj047-nopy-bin` (every
`/bin`, `/usr/bin`, `/usr/sbin`, `/sbin` and Go tool except `python*`;
`command -v python3` empty) the copy's taskless run prints the toolchain row
`python3 not found (required by scripts/agent-context)` /
`FAIL (exit 1, code toolchain.python3_missing)` and the role-pack block
`no python3 on PATH; role-pack parsing needs the PyYAML pinned in
scripts/requirements-verify.txt` / `FAIL (exit 1, code toolchain.pyyaml_missing)`
— a `fail` row, not a skip, so the row cannot be green while the toolchain is
absent. (That run ends at exit 127 because the report writer at
`scripts/verify-candidate:1678` itself needs `python3`; the baseline script under
the same PATH shows the same exit-127 abort but its role-pack block reads
`SKIPPED: PyYAML not importable (design §4 conditional)`. The abort is
pre-existing; only the row verdict changed.)

### C4 — interpreter agreement (one resolved absolute python3 path)

`grep -n 'VERIFY_PYTHON3' scripts/verify-candidate` shows one assignment
(`:286`, `VERIFY_PYTHON3="$(command -v python3 2>/dev/null || true)"`) and three
consumers: `check_toolchain_python3` (`:293`, the reported row), the probe
(`:687`), and `check_role_packs`' heredoc (`:527`). `grep -c 'command -v python3'
scripts/verify-candidate` → 1.

Dynamic confirmation by `PS4='+L${LINENO}: ' bash -x ./scripts/verify-candidate`
inside the copy (one run, PyYAML importable). The expanded trace lines are:

```text
+L286: VERIFY_PYTHON3=/Users/mak/.pyenv/shims/python3
+L293: echo 'python3: /Users/mak/.pyenv/shims/python3'
+L687: /Users/mak/.pyenv/shims/python3 -c 'import yaml'
+L527: /Users/mak/.pyenv/shims/python3 -
```

All three uses expand to the identical absolute path, and no absolute
interpreter is hardcoded (the only absolute `python3` strings anywhere in the
trace come from that one resolution; unrelated pre-existing rows at `:314`,
`:327`, `:433`, `:706`, `:1678` invoke bare `python3` for JSON/report work and
are outside this seam).

### C5 — no self-installation, no network from `verify-candidate`

`grep -nE 'pip |pip3|install|curl|wget|https?://|git clone|git fetch|go get|go mod download|/dev/tcp' scripts/verify-candidate`
returns only: the file header comment, the diagnostic string at `:688`
(`… install with: $VERIFY_PYTHON3 -m pip install -r scripts/requirements-verify.txt`),
and an unrelated comment at `:1296`. The script never reads
`scripts/requirements-verify.txt` (grep for `requirements-verify` → only
comments and the two reason strings; no `cat`/`open`/`source` of it).

Dynamic: `PATH=/private/tmp/fj047-noinstall-bin:$PATH ./scripts/verify-candidate`
in the copy, with `pip`, `pip3`, `curl`, `wget`, `nc` replaced by shims that
print `BLOCKED: <tool> invoked by verify-candidate` to stderr and exit 127 →
exit 0, `summary: 20 passed, 0 failed, 0 skipped, 0 not_applicable`, and no
`BLOCKED` line in the output, i.e. none of those programs was executed. The Go
checks in the same run are offline by construction (`GOPROXY=off`,
`GOTOOLCHAIN=local`, `GOWORK=off`, `GOFLAGS=-mod=readonly` exported at
`scripts/verify-candidate:1246-1252`).

### C6 — documents no longer assert the removed exemption / old skip, counts match

- `.agent/gate-policy.json`: 3 exemption objects (`G-L`, `G-H`, `G-Q`, all
  `stage.tooling_absent`, tracked by FJ-044/045/046); no `G-C` entry, and no
  occurrence of either `toolchain.pyyaml_missing` or `toolchain.pyyaml_exempt`.
  Valid JSON (loaded by `json.load` and by the verifier's policy checks).
- `.agent/README.md`: the "Gate policy" section reads "v1 commits exactly three
  exemptions" and its table has exactly the three `stage.tooling_absent` rows;
  the former `G-C` row is gone. The section also states, matching the code,
  "Exemptions apply only to `skipped` rows: they never waive a `fail`". No
  `toolchain.pyyaml*` string remains in the file (grep → 0).
- `docs/superpowers/specs/2026-09-25-gauntlet-roles-design.md`: §4 no longer
  contains `PyYAML when importable` and no longer describes a `skipped` result
  or a waiver for this check; it states the pin artifact, that the check is
  always required, that the row fails closed with `toolchain.pyyaml_missing`, and
  that `gate-policy.json` can never waive it. §9.2 now opens "v1 commits three
  exemptions" and records the fourth (`G-C`) entry as removed. §13's
  "PyYAML is conditional" limitation is replaced by the remaining true
  limitation (which interpreter `PATH` resolves first). Grep for
  `PyYAML when importable`, `PyYAML is conditional`, `design §4 conditional`
  across the tree returns hits only in this item's own evidence artifacts and
  the dated plan — never in the design docs.
- `docs/superpowers/specs/2026-09-26-deterministic-guard-stack-design.md` §4:
  "The G-C entry that used to block `toolchain.pyyaml_missing` is gone (FJ-047)".
- Count agreement: the three documents that state a count
  (`.agent/README.md`, gauntlet-roles design §9.2, and the guard-stack design's
  G-L/G-H/G-Q list) all describe three exemptions, matching the three objects in
  `.agent/gate-policy.json` and the verifier's runtime reason
  `3 exemption(s) checked`.

### C7 — `scripts/selftest` is green, and the reworked scenarios exercise the new path

`./scripts/selftest` → exit 0, `summary: 75 passed, 0 failed`. The scenario
`pyyaml_exempt_with_task` (which asserted `code=toolchain.pyyaml_exempt`,
`result=skipped`, `exempted=true`, exit 0) no longer exists (`grep` → no match).
`yaml_shim` is used exactly once, at `scripts/selftest:1269`, inside
`pyyaml_missing_fails_closed`.

Non-vacuity was tested by mutation, not by reading:

- **Mutation 1** (`/private/tmp/fj047-qa-mutate`): the new block at
  `scripts/verify-candidate:683-697` was replaced by the pre-change
  `if "$VERIFY_PYTHON3" -c 'import yaml' … else skip_check … "PyYAML not
  importable (design §4 conditional)" …`. `bash -n` → 0. Running
  `./scripts/selftest pyyaml_missing_fails_closed` in the mutated copy → exit 1,
  `FAIL pyyaml_missing_fails_closed`, with the assertion diagnostic
  `check_role_packs: result='skipped', want 'fail'` (row:
  `{required: True, result: 'skipped', code: 'toolchain.pyyaml_missing',
  exempted: False, reason: 'PyYAML not importable (design §4 conditional)'}`).
  So the scenario distinguishes the new `fail` row from the old `skipped` row;
  it is not a restatement. The unmutated copy runs the same scenario → exit 0,
  `1 passed`.
- **Mutation 2** (`/private/tmp/fj047-qa-mutate2`): the engine's `blocking`
  predicate was narrowed to `required and result == "fail"` (required `skipped`
  no longer blocks). Running `./scripts/selftest required_skipped_failure
  exemption_tooling_bootstrap` → exit 1; `FAIL required_skipped_failure`,
  `PASS exemption_tooling_bootstrap`. The reworked §12 scenario therefore still
  observes a real required+skipped blocker.
- The reworked `required_skipped_failure` no longer depends on PyYAML: it builds
  a due Cleaner tooling row under `{"exemptions": []}` and asserts
  `stage.cleaner.tooling` `required/skipped/G-L/stage.tooling_absent` with
  `exempted: false`. Running it under the PyYAML shim
  (`PYTHONPATH=… ./scripts/selftest required_skipped_failure`) → exit 0,
  `1 passed`, i.e. the scenario does not require PyYAML to reach its row.
- `exemption_tooling_bootstrap` still asserts required + skipped + exemption →
  success (`exempted: True`), and it survived mutation 2 (as expected, since an
  exempted skip is not a blocker). The stale comment that claimed the committed
  policy waives `toolchain.pyyaml_missing` is gone
  (`grep 'waives\|committed policy' scripts/selftest` → no match).

### C8 — CI installs the pin, ahead of the runner default, outside the checkout

`.github/workflows/ci.yml` parses as YAML (`yaml.safe_load`). In job `gauntlet`
(`runs-on: ubuntu-latest`) the steps are, in order: `actions/checkout@v4`;
`actions/setup-go@v5`; `install pinned verification dependencies`; `./scripts/verify-candidate`.
So the install step (index 2) precedes the verifier step (index 3). Only the
`gauntlet` job runs the verifier, and only `gauntlet` installs the pin (`test`
and `build`: verifier=no, install=no).

The install step:

```yaml
python3 -m venv "$RUNNER_TEMP/verify-venv"
"$RUNNER_TEMP/verify-venv/bin/python" -m pip install \
  -r scripts/requirements-verify.txt
echo "$RUNNER_TEMP/verify-venv/bin" >> "$GITHUB_PATH"
```

- Consumes the pinned artifact (`-r scripts/requirements-verify.txt`); no
  free-standing unpinned `pip install PyYAML` exists anywhere in `scripts/` or
  `.github/` (grep → none).
- Writes only under `$RUNNER_TEMP`, which is the runner temp directory outside
  `$GITHUB_WORKSPACE`; nothing in the step targets a path inside the checkout.
- Places the pinned interpreter ahead of the runner default: GitHub's runner
  *prepends* each directory written to `$GITHUB_PATH` to `PATH` for subsequent
  steps, so the verifier's single `command -v python3` resolves to
  `"$RUNNER_TEMP/verify-venv/bin/python3"`, the interpreter that holds the pin.
- Local shape simulation (not ubuntu-latest): `python3 -m venv
  /private/tmp/fj047-qa-venv`; `/private/tmp/fj047-qa-venv/bin/python -m pip
  install -r scripts/requirements-verify.txt` → exit 0;
  `PATH=/private/tmp/fj047-qa-venv/bin:$PATH ./scripts/verify-candidate` in the
  copy → exit 0, toolchain row `python3: /private/tmp/fj047-qa-venv/bin/python3`,
  `summary: 20 passed, 0 failed, 0 skipped, 0 not_applicable`. The recipe's shape
  therefore resolves the verifier to the venv interpreter on this machine's
  Python; the PEP 668 behaviour it exists for is specific to `ubuntu-latest` and
  was not observable locally.
- The workflow header no longer claims the Go toolchain is the only network
  fetch; it now names the pinned PyYAML install as setup-time network use.

## Catalogue traces

- **C-QUAL-005** (golden contracts run offline with no credentials, model, or
  network; §23) — the verification dependency is pinned in a committed artifact
  and its offline install path was executed from a local wheelhouse with
  `--no-index` (exit 0, `6.0.3`) and again with a dead proxy (exit 0, `6.0.3`).
  `verify-candidate` itself executes no installer and no network client (C5).
  The one setup-time PyPI fetch is in CI, at job setup, which the workflow
  header states; spec §23 concerns runtime provider services.
- **C-QUAL-006** (required machine-readable contract artifacts are committed and
  reviewable; §45) — `scripts/requirements-verify.txt` and the updated
  `.agent/gate-policy.json` are committed, machine-readable, and unchanged by
  this stage. The policy file parses and lists exactly three exemptions.

## Residual risks and evidence limits

- **CI is not executable here.** The venv recipe was simulated on macOS with a
  local Python; `ubuntu-latest`, its PEP 668 externally-managed `python3`, and
  the `$GITHUB_PATH` prepend were verified by reading the workflow and the
  documented runner behaviour, not by running Actions.
- **Local version equality with the pin is not enforced.** Observed directly:
  with `PYTHONPATH=/private/tmp/fj047-qa-old` (PyYAML 6.0.2 installed from its
  own wheel) the copy's `./scripts/verify-candidate` exits 0 and the role-pack
  row is green — the check tests importability, not `==6.0.3`. The pin is
  enforced by CI installing the artifact and by the documented install path.
  This matches the specifier's O1; no acceptance criterion in this item requires
  a version-equality failure.
- **Wheelhouse portability.** The downloaded wheel is
  `cp311-cp311-macosx_11_0_arm64`; the offline recipe is only reproducible for a
  matching platform/Python. `scripts/README.md` states this. The pin itself is
  platform-neutral.
- **No-`python3` environment aborts at the report writer.** Exit 127 at
  `scripts/verify-candidate:1678`, identical to the `7eb1252` baseline; the
  fail-closed row verdict is observable before that point.
- **The old `toolchain.pyyaml_exempt` code is retained in one live design
  clause** (gauntlet-roles §9.2, `:561`) as an explicitly past-tense record of
  the deleted entry, and in the dated plan and prior evidence artifacts
  (including the stale `.agent/work/FJ-004/qa.md:55`). None of these is a live
  behaviour contract; `.agent/work` and the dated plan are outside this item's
  `allowedFiles` and excluded from candidate identity.
- **Report churn.** The mandated runs added
  `.agent/reports/FJ-047/report-20261002T150746Z.json` to the six untracked
  reports already present; `.agent/reports` is excluded from candidate identity
  and the candidate fingerprint is unchanged (`21a64a50…`).
- **Selftest flake (FJ-060, unrelated).** The coder reported one
  `fuzz_live_pass` failure across its session. This session's full
  `./scripts/selftest` run completed `75 passed, 0 failed`; nothing in this
  change touches the guard fuzz runner.

## Commands run (all recorded above)

Repository: `bash -n scripts/verify-candidate scripts/selftest`;
`./scripts/selftest`; `./scripts/verify-candidate`;
`./scripts/verify-candidate FJ-047`;
`PATH=/opt/homebrew/bin:$PATH ./scripts/verify-candidate`;
`go test ./... -count=1`; `./scripts/candidate-fingerprint candidate`;
`git diff --cached --stat`; the grep battery for
`toolchain.pyyaml_missing` / `toolchain.pyyaml_exempt` / `import yaml` /
`check_role_packs` / `pip` / `requirements-verify.txt`.

Outside the repository: the wheelhouse download, offline and strict-offline and
normal installs; the `bash -x` interpreter trace; the no-install shim run; the
venv recipe simulation; the no-`python3` PATH run; the copy negative cases A/B/C
and the baseline contrast; the two mutation tests; the PyYAML shim runs of
`required_skipped_failure` and `pyyaml_missing_fails_closed`; the 6.0.2
version-mismatch run.
