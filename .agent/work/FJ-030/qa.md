---
stage: qa
task: FJ-030
inputFingerprint: bad76f7917d526d0978fb613aee212f2345ff06a717259788d4a04366f01ab19
outputFingerprint: bad76f7917d526d0978fb613aee212f2345ff06a717259788d4a04366f01ab19
taskFingerprint: d37327aa23e2afc94bc9b8bbc7d4c9cd3a87d61e03dc2cc2605c258b8b6c34a7
gitHead: abba008
generatedAt: 2026-09-30T15:17:50Z
author: worker/FJ-030-qa
---

# QA — FJ-030: `fake-jev validate <path>`

## Scope, author separation, and method

- Fresh QA session; author `worker/FJ-030-qa` differs from the coder author
  `worker/FJ-030-coder` and from the hardener author `worker/FJ-030-hardener`.
- Candidate under observation: fingerprint
  `bad76f7917d526d0978fb613aee212f2345ff06a717259788d4a04366f01ab19`
  (`./scripts/candidate-fingerprint candidate`, run this session), revision
  `abba008`. That digest equals the QA `inputFingerprint` above, so the
  revision observed here is the hardener revision.
- The evidence below comes from a **binary built from this revision and driven
  as a subprocess**, not from `internal/cli/validate_test.go`. The repository's
  own tests are not used as the basis for any observation; they were only run
  (and recorded) as the requested command battery.
- Built binary: `/private/tmp/fj030-qa/fake-jev`
  (`go build -o /private/tmp/fj030-qa/fake-jev ./cmd/fake-jev`, Mach-O arm64).
- Probe: `/private/tmp/fj030-qa/probe.py`, sha256
  `c8948af93dc6e723ba555d41dc52904e7040734105b8908ac3e59b760db5dee5`.
  Every fixture is created by the probe under a fresh
  `/private/tmp/fj030-qa/fixtures-*` directory; no file was added to the
  repository by this stage.
- Probe output: `/private/tmp/fj030-qa/probe-output.json`, sha256
  `5f555481a846402e9daca339fb0303ed9e1644dc700a41543cd315488c3dd98e`.
  The probe exited 0 with `asserted_total: 34, failed: 0`, `observations: 4`.
- Static closure helper: `/private/tmp/fj030-qa/closure/main.go`, sha256
  `7a9899bf930f3ccee1be9fdfbddef1cd3cd2461fab5adf1de1967d72023d2391`.

## C-CLI-005 — §15.2 static checks, no server, no network request

### Formats and success output (§15.2 "MUST load YAML or JSON", "success output MAY be human-readable")

| Input | exit observed | stdout observed | stderr observed |
|---|---|---|---|
| valid YAML | 0 | `fake-jev: <path>: configuration is valid` | empty |
| valid JSON (same logical document, §12 convergence) | 0 | `fake-jev: <path>: configuration is valid` | empty |
| `schemaVersion: 1` only (defaults applied, §12.2) | 0 | success line | empty |

The success line was non-empty and contained `valid`; stderr carried nothing on
each of these three runs.

### Inputs that produce exit 2 with a stderr diagnostic (stdout empty in every case)

All rows below were asserted for `exit == 2`, `stdout == ""`, and non-empty
stderr; the stdout-empty assertion was explicit per case.

| Case | exit | first stderr line observed |
|---|---|---|
| missing path argument | 2 | `fake-jev: validate requires a configuration path` (+ usage on stderr) |
| extra arguments (`validate <valid> <valid>`) | 2 | `fake-jev: validate takes exactly one configuration path` (+ usage) |
| unknown flag before path (`--strict`) | 2 | `fake-jev: unknown flag "--strict"` (+ usage) |
| unknown flag after path (`<valid> --strict`) | 2 | `fake-jev: unknown flag "--strict"` (+ usage) |
| missing file | 2 | `fake-jev: <path>: read configuration "<path>": open …: no such file or directory` |
| directory path | 2 | `fake-jev: <dir>: read configuration "<dir>": read <dir>: is a directory` |
| malformed YAML (`port: [unclosed`) | 2 | `fake-jev: <path>: decode YAML configuration: yaml: …` |
| malformed JSON (`{"schemaVersion": 1, "server": }`) | 2 | `fake-jev: <path>: server must be an object` |
| empty file | 2 | `fake-jev: <path>: configuration is empty` |
| unknown top-level key | 2 | `fake-jev: <path>: decode configuration: json: unknown field "unexpectedKey"` |
| unknown nested key (`server.bogus`) | 2 | `fake-jev: <path>: decode configuration: json: unknown field "bogus"` |
| duplicate stub id (§11.2) | 2 | `fake-jev: <path>: duplicate static stub id "stub-1"` |
| invalid `expect` (`exactly` + `atLeast`, §12.7) | 2 | `fake-jev: <path>: stub "stub-1" expect: exactly one of exactly or a range is required` |
| invalid response form (`answers` together with `raw`, §12.6/§38.7) | 2 | `fake-jev: <path>: stub "stub-1" then: exactly one of answers, sequence, or raw is required` |
| §38 required schema (missing `schemaVersion`) | 2 | `fake-jev: <path>: schemaVersion is required` |

The usage failures additionally carried `usage: fake-jev <command> [flags]` on
stderr; the schema failures did not.

### Additional checks observed as performed (characterization of the shared surface)

Each fixture below was observed to produce `exit == 2`, empty stdout, non-empty
stderr:

- nested `sequence` inside a sequence element (§38.7) →
  `decode configuration: json: unknown field "sequence"`;
- `raw` with a `model` sibling (§38.7) →
  `raw cannot have model or usage siblings`;
- `raw.status: 999` → `raw.status must be between 100 and 599`;
- negative `usage.input_tokens` → `usage token counts must be non-negative`;
- unknown key inside `when` → `json: unknown field "boguswhen"`;
- unknown key inside `expect` → `json: unknown field "bogus"`;
- `compatibility: [jev, jev/v1]` → `duplicate compatibility profile "jev/v1"`
  (alias normalized before duplicate detection, §38.1);
- `server.port: 70000` → `server.port must be between 0 and 65535`;
- `limits.maxInteractions: 0` → `limits.maxInteractions must be at least 1`;
- `models: []` → `models must contain at least one model`;
- `release_date: not-a-date` → `models[0].release_date must match YYYY-MM-DD`;
- stub without an enabled `profile` → `stub "s": unknown profile ""`;
- a sequence element carrying both `answers` and `raw` →
  `sequence[0]: exactly one of answers or raw is required`.

### §15.2 checks NOT performed (observed, and owned by FJ-059)

Four deliberately-illegal fixtures were observed to produce `exit == 0` with
the success line, i.e. the following rules are **not** performed by
`config.LoadFile` and therefore not by `validate`:

| Fixture | §-rule it violates | exit observed |
|---|---|---|
| `then.answers.{q}.noul: 5` | §13.1 `noul` finite in `[0,1]` | 0 |
| `then.answers.{q}.bogus: 1` | helper object carries an unknown key | 0 |
| `then.answers.{q}: {score: 0.5, legend: …}` | §13.3 prohibits `legend` in v1 score fixtures | 0 |
| `when.questions: {urgent: noul}` vs `then.answers: {something_else: {choice: nope}}` | §13.4/§13.5 answer/coverage cross-check | 0 |

Stated plainly: the **inner `jev/v1` `then.answers` payload checks** and the
**`when.questions` → `then.answers` key-set/type cross-check are NOT performed
by this candidate**; per the owner-approved split recorded in
`.agent/work/FJ-030/state.json` (`notes`, ownership split 2026-09-29) they are
owned by **FJ-059**. This artifact records that observation; it does not decide
whether that split is acceptable.

### No listener and no network request (§15.2, §19.2)

Method used, stated explicitly: **static evidence of the import closure of
`internal/cli/validate.go`** (primary), corroborated by Go's own dependency
resolver and by two behavioral observations. No sandboxed egress-denied run was
performed.

1. Static closure (helper above, `go/parser`): seeded at
   `internal/cli/validate.go`, following module-local imports into every
   non-test file of each reached package. Closure observed:
   `internal/cli/{validate,dispatch,version}.go`,
   `internal/config/{load,schema,validate}.go`. Result line:
   `net imports on validate path: 0 []`.
2. `go list -deps ./internal/cli ./internal/config`: no `net` or `net/...`
   package in either dependency set (`net_deps: []`).
3. Behavioral, held port: a loopback listener was bound to an ephemeral port and
   `server.port` in the fixture was set to that held port; `validate` was
   observed to exit 0 with the success line (a command that tried to bind the
   configured port would instead have produced a bind error).
4. Behavioral, live socket sampling: `validate` was run against a ~17 MB valid
   JSON configuration and `lsof -nP -a -p <pid> -i` was sampled in a loop while
   the process was alive — **56 samples, 0 socket lines**; exit observed 0 with
   the success line.

The closure in (1) covers `internal/cli` as a package (a package links as a
unit), so it is a superset of any file-level path from `validate.go`; (2) is an
independent resolver cross-check. Limitation: (3) only rules out binding the
*configured* port, and (4) is a sampled window, not a kernel-level guarantee.

## C-CLI-001 — §42.1 exit codes

- `0` was observed only on the success path (valid YAML, valid JSON,
  defaults-only, help, `--help`).
- `2` was observed on every usage failure, every loader/undecodable-input
  failure, and every schema violation exercised (24 rows above plus the
  3 usage rows).
- `3` was never observed. The probe asserts `exit != 3` on every one of its 34
  asserted cases; no validate invocation produced 3.
- On every non-zero exit the probe asserted `stdout == ""` explicitly; no
  failure case wrote anything to stdout.

## Usage text versus dispatcher

- `fake-jev help` and `fake-jev --help` were observed to exit 0 and print, on
  stdout, a command block containing `validate <path>  validate a configuration
  file without starting a server`; stderr was empty.
- A bare `fake-jev validate` was observed to exit 2 with the usage text on
  stderr, and that usage text also contains `validate <path>`.
- The listed commands observed in the usage block (`version`, `validate`,
  `help`) correspond one-to-one with the `Run` switch arms in
  `internal/cli/dispatch.go` (`version`, `validate`, `help`/`-h`/`--help`).
  Reaching `validate` through the dispatcher is what the whole matrix above
  exercises, so the usage listing and the dispatcher were observed consistent.

## Exact commands run this session and outcomes

All commands were run from `/Users/mak/git/fake-jev-FJ-030`.

| Command | Outcome observed |
|---|---|
| `go build -o /private/tmp/fj030-qa/fake-jev ./cmd/fake-jev` | exit 0; binary produced |
| `python3 /private/tmp/fj030-qa/probe.py` | exit 0; 34 asserted cases with no failed assertion; 4 gap observations |
| `go test ./internal/cli -count=1` | exit 0; `ok fake-jev/internal/cli 0.595s` |
| `go test ./... -count=1` | exit 0; all 9 packages `ok` (8 with tests, `cmd/fake-jev` no test files) |
| `go test -race ./... -count=1` | exit 0; all packages `ok` under `-race` |
| `go vet ./...` | exit 0; no output |
| `go run ./cmd/guard arch` | exit 0; `findings: []`, `files: 50`, `packages: 9` |
| `go run ./cmd/guard lint` | exit 0; `findings: []`, `files: 50`, `packages: 9` |
| `go run ./cmd/guard trace` | exit 0; `findings: []`, `active: 0`, `covered: 17`, `test_files: 17` |
| `go run ./cmd/guard fuzz` | exit 0; `findings: []`, `targets: 0` |
| `./scripts/verify-candidate FJ-030` | exit 0; tool summary `20 passed, 0 failed, 0 skipped, 0 not_applicable`, `RESULT: PASS`; report written to `.agent/reports/FJ-030/report-20260930T151727Z.json` |

`verify-candidate` reported the tree as DIRTY (`0 staged, 2 unstaged, 8
untracked`) and bound its evidence to `abba008` plus those changes. The two
unstaged files are `.agent/work/FJ-030/state.json` and
`internal/cli/dispatch.go`; the untracked files are the work-item artifacts and
the new CLI sources plus the reports under `.agent/reports/FJ-030/`. Nothing
was staged (`git diff --cached --stat` empty). The report file named above is
the script's own output from the mandated run; no other repository file was
written by this stage.

## §15.2 check surface, from the configuration interface

`internal/cli/validate.go` performs only argument parsing, `config.LoadFile`,
stream routing, and the exit code; every rule is `internal/config`. Observed as
performed (through the built binary): YAML/JSON decode convergence, top-level
`DisallowUnknownFields`, `schemaVersion` required and equal to integer 1,
`mode == strict`, non-empty/unique/known `compatibility` after alias
normalization, `server`/`limits` types and ranges, model required fields /
uniqueness / `YYYY-MM-DD` / non-empty list, `when`/`then` presence, unknown-key
rejection across `server`, `limits`, stub envelope, `when`, `then`, `expect`,
`then`/sequence response-form exactness, `raw` status range and sibling rules,
`usage` presence and non-negativity, `expect` form and ordering, and duplicate
static stub `id`.

Observed as **not** performed: the inner `jev/v1` `then.answers` payload rules
(§13.1–13.3: `noul` range/finiteness, `choice`/`score` probability rules,
prohibited `legend`, unknown helper keys) and the `when.questions` ↔
`then.answers` key-set/type cross-check. These are statically decidable but
owned by FJ-059 (see the table above and the ownership split recorded in
`state.json`). Separately, request-dependent §13 rules (criteria-derived
probability keys, score ranges against the request, answer coverage) are not
decidable from the file alone.

## Residual risks and evidence limits

- The no-network observation is structural (import closure + `go list -deps`)
  plus a held-port test plus a sampled `lsof` window; it is not a sandboxed run
  with egress disabled, and the live sampling is a window rather than a kernel
  guarantee.
- The probe is hermetic and offline but depends on macOS `lsof`; on a machine
  without it the live-socket observation would be unavailable (the static
  closure and the held-port test would still run).
- The `release_date` fixture used `not-a-date`. An unquoted
  `release_date: 1970-1-1` was separately observed to exit 0 because the YAML
  normalizer converts a YAML timestamp to `1970-01-01` before validation; this
  was not part of the required coverage and is recorded only as a boundary the
  probe did not assert.
- The malformed-JSON fixture is rejected with `server must be an object`
  (the JSON duplicate-key scan fails, then the YAML flow parse yields a null
  `server`); the outcome observed is exit 2 with a diagnostic, which is what the
  criterion asks, but the diagnostic text is not a JSON-syntax message.
- The probe asserts over its own fixtures and does not fuzz the argument parser;
  no adversarial inputs beyond the listed matrix were exercised.
- The candidate fingerprint was read through `./scripts/candidate-fingerprint`,
  which excludes `.agent/work/` and `.agent/reports/`; a change to production
  files after this observation would not be reflected in the digest recorded
  here.
- This artifact records observations only. It does not declare the candidate
  acceptable.

## Traces

- C-CLI-005 — observed `validate` states (formats accepted, static checks
  performed and not performed, no listener, no network request).
- C-CLI-001 — observed exit codes 0 and 2, and absence of 3, for `validate`.

## Probe source

Full source of `/private/tmp/fj030-qa/probe.py` (sha256
`c8948af93dc6e723ba555d41dc52904e7040734105b8908ac3e59b760db5dee5`):

```python
#!/usr/bin/env python3
"""FJ-030 independent QA probe.

Drives the *built* fake-jev binary as a subprocess (not the package tests) and
records exit codes, stdout, and stderr for the acceptance surface of
`fake-jev validate <path>` (spec §15.2, §42.1; catalog C-CLI-005, C-CLI-001).

Nothing here is imported by the repository; fixtures and outputs live under
/private/tmp/fj030-qa only. Exit code 0 means every asserted observation held.

Usage: python3 probe.py
"""

import json
import os
import shutil
import socket
import subprocess
import sys
import tempfile

BIN = "/private/tmp/fj030-qa/fake-jev"
REPO = "/Users/mak/git/fake-jev-FJ-030"
ROOT = "/private/tmp/fj030-qa"
CLOSURE = "/private/tmp/fj030-qa/closure/main.go"

results = {"asserted": [], "observations": [], "notes": []}


def run(argv):
    p = subprocess.run([BIN] + argv, capture_output=True, text=False)
    return {
        "argv": argv,
        "exit": p.returncode,
        "stdout": p.stdout.decode("utf-8", "replace"),
        "stderr": p.stderr.decode("utf-8", "replace"),
    }


def check(name, argv, want_exit, want_stdout_empty, want_stderr_nonempty,
          stdout_contains=None, stderr_contains=None):
    r = run(argv)
    problems = []
    if r["exit"] != want_exit:
        problems.append("exit=%d want %d" % (r["exit"], want_exit))
    if want_stdout_empty and r["stdout"] != "":
        problems.append("stdout not empty: %r" % r["stdout"])
    if not want_stdout_empty and r["stdout"].strip() == "":
        problems.append("stdout empty but expected output")
    if want_stderr_nonempty and r["stderr"].strip() == "":
        problems.append("stderr empty")
    if stdout_contains and stdout_contains not in r["stdout"]:
        problems.append("stdout lacks %r" % stdout_contains)
    if stderr_contains and stderr_contains not in r["stderr"]:
        problems.append("stderr lacks %r" % stderr_contains)
    if r["exit"] == 3:
        problems.append("exit 3 must never be produced by validate")
    entry = {
        "name": name,
        "argv": argv,
        "exit": r["exit"],
        "stdout": r["stdout"],
        "stderr_first_line": (r["stderr"].splitlines() or [""])[0],
        "pass": not problems,
        "problems": problems,
    }
    results["asserted"].append(entry)
    return entry


def observe(name, argv, note=""):
    r = run(argv)
    entry = {
        "name": name,
        "argv": argv,
        "exit": r["exit"],
        "stdout": r["stdout"],
        "stderr_first_line": (r["stderr"].splitlines() or [""])[0],
        "note": note,
    }
    results["observations"].append(entry)
    return entry


def main():
    fixtures = tempfile.mkdtemp(prefix="fixtures-", dir=ROOT)

    def fx(name, text):
        path = os.path.join(fixtures, name)
        with open(path, "w") as fh:
            fh.write(text)
        return path

    # --- valid documents (YAML and JSON of the same logical configuration) ---
    valid_yaml = fx("valid.yaml", """\
schemaVersion: 1
server:
  host: 127.0.0.1
  port: 8787
compatibility:
  - jev
limits:
  dataPlaneBodyBytes: 8388608
  controlPlaneBodyBytes: 2097152
  maxInteractions: 10000
  logBodyBytes: 4096
  gracefulShutdownSeconds: 5
models:
  - name: jev-latest
    description: Local deterministic fake model provided by fake-jev.
    release_date: 1970-01-01
stubs:
  - id: stub-1
    profile: jev
    priority: 0
    when:
      operation: systemone
      questions:
        urgent: noul
    then:
      answers:
        urgent:
          noul: 0.95
    expect:
      exactly: 1
""")
    valid_json = fx("valid.json", json.dumps({
        "schemaVersion": 1,
        "server": {"host": "127.0.0.1", "port": 8787},
        "compatibility": ["jev"],
        "limits": {"dataPlaneBodyBytes": 8388608,
                   "controlPlaneBodyBytes": 2097152,
                   "maxInteractions": 10000, "logBodyBytes": 4096,
                   "gracefulShutdownSeconds": 5},
        "models": [{"name": "jev-latest", "description": "x",
                    "release_date": "1970-01-01"}],
        "stubs": [{"id": "stub-1", "profile": "jev", "priority": 0,
                   "when": {"operation": "systemone",
                            "questions": {"urgent": "noul"}},
                   "then": {"answers": {"urgent": {"noul": 0.95}}},
                   "expect": {"exactly": 1}}],
    }))
    defaults_only = fx("defaults_only.yaml", "schemaVersion: 1\n")

    # --- failure documents ---
    malformed_yaml = fx("malformed.yaml", "schemaVersion: 1\nserver:\n  port: [unclosed\n")
    malformed_json = fx("malformed.json", '{"schemaVersion": 1, "server": }\n')
    unknown_toplevel = fx("unknown_toplevel.yaml", "schemaVersion: 1\nunexpectedKey: true\n")
    unknown_nested = fx("unknown_nested.yaml",
                        "schemaVersion: 1\nserver:\n  host: 127.0.0.1\n  port: 8787\n  bogus: 1\n")
    dup_stub = fx("dup_stub.yaml", """\
schemaVersion: 1
stubs:
  - id: stub-1
    profile: jev
    when: {}
    then:
      answers:
        urgent:
          noul: 0.95
  - id: stub-1
    profile: jev
    when: {}
    then:
      answers:
        urgent:
          noul: 0.5
""")
    bad_expect = fx("bad_expect.yaml", """\
schemaVersion: 1
stubs:
  - id: stub-1
    profile: jev
    when: {}
    then:
      answers:
        urgent:
          noul: 0.95
    expect:
      exactly: 1
      atLeast: 2
""")
    bad_form = fx("bad_form.yaml", """\
schemaVersion: 1
stubs:
  - id: stub-1
    profile: jev
    when: {}
    then:
      answers:
        urgent:
          noul: 0.95
      raw:
        status: 200
""")
    missing_schema_version = fx("missing_schema_version.yaml", "server:\n  host: 127.0.0.1\n")
    empty_file = fx("empty.yaml", "")

    # --- required assertions -------------------------------------------------
    # 1. valid YAML / JSON / defaults-only -> exit 0, success line on stdout.
    for label, path in (("valid YAML", valid_yaml),
                        ("valid JSON", valid_json),
                        ("defaults-only YAML", defaults_only)):
        check("%s exits 0 with human-readable stdout" % label, ["validate", path],
              0, want_stdout_empty=False, want_stderr_nonempty=False,
              stdout_contains="valid")

    # 2. the required failure matrix -> exit 2, empty stdout, stderr diagnostic.
    failure_cases = [
        ("missing path argument", ["validate"], None),
        ("extra arguments", ["validate", valid_yaml, valid_yaml], "usage"),
        ("unknown flag before path", ["validate", "--strict"], "usage"),
        ("unknown flag after path", ["validate", valid_yaml, "--strict"], "usage"),
        ("missing file", ["validate", os.path.join(fixtures, "does-not-exist.yaml")], None),
        ("directory path", ["validate", fixtures], None),
        ("malformed YAML", ["validate", malformed_yaml], None),
        ("malformed JSON", ["validate", malformed_json], None),
        ("empty file", ["validate", empty_file], None),
        ("unknown top-level key", ["validate", unknown_toplevel], None),
        ("unknown nested key", ["validate", unknown_nested], None),
        ("duplicate stub id", ["validate", dup_stub], None),
        ("invalid expect", ["validate", bad_expect], None),
        ("invalid response form answers+raw", ["validate", bad_form], None),
        ("missing schemaVersion (section 38 required schema)",
         ["validate", missing_schema_version], None),
    ]
    for name, argv, extra in failure_cases:
        stderr_contains = None
        if extra == "usage":
            stderr_contains = "usage: fake-jev <command>"
        check(name + " exits 2, empty stdout, stderr diagnostic", argv,
              2, want_stdout_empty=True, want_stderr_nonempty=True,
              stderr_contains=stderr_contains)

    # 3. usage text lists `validate <path>`; dispatcher reaches it.
    check("help lists validate <path>", ["help"], 0, want_stdout_empty=False,
          want_stderr_nonempty=False, stdout_contains="validate <path>")
    check("--help lists validate <path>", ["--help"], 0, want_stdout_empty=False,
          want_stderr_nonempty=False, stdout_contains="validate <path>")
    # A bare `validate` (no path) shows usage on stderr naming the command line.
    check("usage failure also prints usage with validate <path>", ["validate"],
          2, want_stdout_empty=True, want_stderr_nonempty=True,
          stderr_contains="validate <path>")

    # --- characterization of checks that ARE performed ----------------------
    performed = [
        ("nested sequence rejected", "seq_nested.yaml", """\
schemaVersion: 1
stubs:
  - id: s
    profile: jev
    when: {}
    then:
      sequence:
        - sequence:
            - answers:
                q:
                  noul: 0.5
"""),
        ("raw with model sibling rejected", "raw_model_sibling.yaml", """\
schemaVersion: 1
stubs:
  - id: s
    profile: jev
    when: {}
    then:
      raw:
        status: 200
      model: m
"""),
        ("raw.status out of range rejected", "raw_status_range.yaml", """\
schemaVersion: 1
stubs:
  - id: s
    profile: jev
    when: {}
    then:
      raw:
        status: 999
"""),
        ("negative usage tokens rejected", "usage_negative.yaml", """\
schemaVersion: 1
stubs:
  - id: s
    profile: jev
    when: {}
    then:
      answers:
        q:
          noul: 0.5
      usage:
        input_tokens: -1
        output_tokens: 2
"""),
        ("unknown key inside when rejected", "when_unknown.yaml", """\
schemaVersion: 1
stubs:
  - id: s
    profile: jev
    when:
      boguswhen: 1
    then:
      answers:
        q:
          noul: 0.5
"""),
        ("unknown key inside expect rejected", "expect_unknown.yaml", """\
schemaVersion: 1
stubs:
  - id: s
    profile: jev
    when: {}
    then:
      answers:
        q:
          noul: 0.5
    expect:
      bogus: 1
"""),
        ("duplicate compatibility after alias normalization rejected",
         "compat_dup.yaml", """\
schemaVersion: 1
compatibility:
  - jev
  - jev/v1
"""),
        ("server.port out of range rejected", "port_range.yaml", """\
schemaVersion: 1
server:
  port: 70000
"""),
        ("limits out of range rejected", "limits_range.yaml", """\
schemaVersion: 1
limits:
  maxInteractions: 0
"""),
        ("empty model list rejected", "models_empty.yaml", "schemaVersion: 1\nmodels: []\n"),
        ("bad release_date rejected", "release_bad.yaml", """\
schemaVersion: 1
models:
  - name: m
    description: d
    release_date: not-a-date
"""),
        ("stub without enabled profile rejected", "stub_no_profile.yaml", """\
schemaVersion: 1
stubs:
  - id: s
    when: {}
    then:
      answers:
        q:
          noul: 0.5
"""),
        ("sequence element with answers+raw rejected", "seq_elem_both.yaml", """\
schemaVersion: 1
stubs:
  - id: s
    profile: jev
    when: {}
    then:
      sequence:
        - answers:
            q:
              noul: 0.5
          raw:
            status: 200
"""),
    ]
    for label, fname, text in performed:
        path = fx(fname, text)
        check("performed: %s" % label, ["validate", path], 2,
              want_stdout_empty=True, want_stderr_nonempty=True)

    # --- §15.2 gaps: NOT performed (owned by FJ-059) ------------------------
    g1_noul = fx("g1_noul.yaml", """\
schemaVersion: 1
stubs:
  - id: g1
    profile: jev
    when:
      questions:
        urgent: noul
    then:
      answers:
        urgent:
          noul: 5
""")
    g1_key = fx("g1_helper_key.yaml", """\
schemaVersion: 1
stubs:
  - id: g1b
    profile: jev
    when:
      questions:
        urgent: noul
    then:
      answers:
        urgent:
          bogus: 1
""")
    g1_legend = fx("g1_legend.yaml", """\
schemaVersion: 1
stubs:
  - id: g1c
    profile: jev
    when:
      questions:
        route: score
    then:
      answers:
        route:
          score: 0.5
          legend:
            "0": low
""")
    g2_cross = fx("g2_cross.yaml", """\
schemaVersion: 1
stubs:
  - id: g2
    profile: jev
    when:
      questions:
        urgent: noul
    then:
      answers:
        something_else:
          choice: nope
""")
    for label, path in (
        ("G1 inner then.answers noul outside [0,1] accepted", g1_noul),
        ("G1 inner then.answers unknown helper key accepted", g1_key),
        ("G1 v1-prohibited legend key accepted", g1_legend),
        ("G2 when.questions vs then.answers mismatch accepted", g2_cross),
    ):
        observe("gap-%s" % label, ["validate", path],
                note="§15.2 inner jev/v1 payload rule not performed; owned by FJ-059")

    # --- no listener / no network -------------------------------------------
    # (a) static: import closure of internal/cli/validate.go
    static = subprocess.run(
        ["go", "run", CLOSURE, REPO, "internal/cli/validate.go"],
        capture_output=True, text=True, cwd=os.path.dirname(CLOSURE),
        env=dict(os.environ, GO111MODULE="off"))
    closure_net = [ln for ln in static.stdout.splitlines()
                   if "net imports on validate path:" in ln]
    results["no_listener_static"] = {
        "method": "go/parser import closure seeded at internal/cli/validate.go",
        "exit": static.returncode,
        "closure": [ln.strip() for ln in static.stdout.splitlines()
                    if ln.startswith("  ")],
        "net_line": closure_net[0] if closure_net else static.stdout.strip(),
    }
    # (b) static cross-check: Go's own package dependency resolver
    deps = subprocess.run(
        ["go", "list", "-deps", "-f", "{{.ImportPath}}",
         "./internal/cli", "./internal/config"],
        capture_output=True, text=True, cwd=REPO)
    net_deps = [ln for ln in deps.stdout.splitlines()
                if ln == "net" or ln.startswith("net/")]
    results["no_listener_deps"] = {
        "method": "go list -deps on internal/cli and internal/config",
        "exit": deps.returncode,
        "net_deps": net_deps,
    }
    # (c) behavioral: hold an ephemeral loopback port equal to server.port
    listener = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    listener.bind(("127.0.0.1", 0))
    listener.listen(1)
    held_port = listener.getsockname()[1]
    port_fixture = fx("held_port.yaml", """\
schemaVersion: 1
server:
  host: 127.0.0.1
  port: %d
""" % held_port)
    r = run(["validate", port_fixture])
    results["no_listener_port_hold"] = {
        "method": "config server.port set to a held ephemeral loopback port",
        "held_port": held_port,
        "exit": r["exit"],
        "stdout": r["stdout"],
        "stderr": r["stderr"],
    }
    listener.close()
    # (d) behavioral: sample the process sockets while it runs on a large file
    big = os.path.join(fixtures, "big.json")
    with open(big, "w") as fh:
        fh.write(json.dumps({
            "schemaVersion": 1,
            "compatibility": ["jev"],
            "limits": {"dataPlaneBodyBytes": 8388608,
                       "controlPlaneBodyBytes": 2097152,
                       "maxInteractions": 10000, "logBodyBytes": 4096,
                       "gracefulShutdownSeconds": 5},
            "models": [{"name": "jev-latest", "description": "x",
                        "release_date": "1970-01-01"}],
            "stubs": [{"id": "stub-%d" % i, "profile": "jev",
                       "when": {"operation": "systemone",
                                "questions": {"q": "noul"}},
                       "then": {"answers": {"q": {"noul": 0.5}}}}
                      for i in range(150000)],
        }))
    proc = subprocess.Popen([BIN, "validate", big],
                            stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    samples = 0
    socket_lines = []
    while proc.poll() is None:
        s = subprocess.run(["lsof", "-nP", "-a", "-p", str(proc.pid), "-i"],
                           capture_output=True, text=True)
        lines = [ln for ln in s.stdout.splitlines() if ln.strip()]
        samples += 1
        for ln in lines[1:]:
            socket_lines.append(ln)
    out, err = proc.communicate()
    results["no_listener_lsof_sampling"] = {
        "method": "lsof -nP -a -p PID -i sampled while validate parsed a "
                  "17 MB valid config",
        "samples": samples,
        "socket_lines": socket_lines,
        "exit": proc.returncode,
        "stdout": out.decode(),
        "stderr": err.decode(),
    }

    # --- summary -------------------------------------------------------------
    failures = [c for c in results["asserted"] if not c["pass"]]
    results["summary"] = {
        "asserted_total": len(results["asserted"]),
        "failed": len(failures),
        "failed_names": [c["name"] for c in failures],
        "observations": len(results["observations"]),
        "fixtures_dir": fixtures,
    }
    print(json.dumps(results, indent=2))
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main())
```

Closure helper source (`/private/tmp/fj030-qa/closure/main.go`, sha256
`7a9899bf930f3ccee1be9fdfbddef1cd3cd2461fab5adf1de1967d72023d2391`):

```go
package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func importsOf(path string) ([]string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, imp := range f.Imports {
		p, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func main() {
	repo := os.Args[1]
	seedRel := os.Args[2]
	module := "fake-jev"
	visitedPkg := map[string]bool{}
	visitedFile := map[string]bool{}
	var closure []string
	var netImports []string
	queue := []string{seedRel}
	for len(queue) > 0 {
		rel := queue[0]
		queue = queue[1:]
		pkgDir := filepath.Dir(rel)
		if visitedPkg[pkgDir] {
			continue
		}
		visitedPkg[pkgDir] = true
		matches, _ := filepath.Glob(filepath.Join(repo, pkgDir, "*.go"))
		for _, m := range matches {
			name := filepath.Base(m)
			if strings.HasSuffix(name, "_test.go") {
				continue
			}
			if visitedFile[m] {
				continue
			}
			visitedFile[m] = true
			closure = append(closure, filepath.ToSlash(relDir(repo, m)))
			imps, err := importsOf(m)
			if err != nil {
				fmt.Println("ERROR parsing", m, err)
				os.Exit(1)
			}
			for _, imp := range imps {
				if imp == "net" || strings.HasPrefix(imp, "net/") {
					netImports = append(netImports, filepath.ToSlash(relDir(repo, m))+" imports "+imp)
				}
				if strings.HasPrefix(imp, module+"/") {
					sub := strings.TrimPrefix(imp, module+"/")
					queue = append(queue, filepath.Join(sub, "x.go"))
				}
			}
		}
	}
	sort.Strings(closure)
	sort.Strings(netImports)
	fmt.Println("closure files:")
	for _, c := range closure {
		fmt.Println("  " + c)
	}
	fmt.Printf("net imports on validate path: %d %v\n", len(netImports), netImports)
}

func relDir(repo, p string) string {
	r, err := filepath.Rel(repo, p)
	if err != nil {
		return p
	}
	return r
}
```
