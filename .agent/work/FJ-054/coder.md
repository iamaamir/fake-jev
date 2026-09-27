---
stage: coder
task: FJ-054
inputFingerprint: 297b6528321db380641e79c8d702dbf4782c9e71b9fa2a667e4b9c5f97359033
outputFingerprint: 2c8379c31c3539f27c20e1e475e20802239c617e269e4928ae166c0ee1c50df0
taskFingerprint: ea17d10b719f94a62eb9a5c68f66391a466e7cf325961bf5cd5f8082ea8db881
gitHead: 4962c85
generatedAt: 2026-09-27T21:19:53Z
author: subagent/coder-1
---

# Coder — FJ-054

## What changed (all in `scripts/verify-candidate`, +70/-7)

One new section, `module posture: pinned environment + fail-closed resolution
checks`, inserted after the gauntlet engine and before the product-checks
section — i.e. after the last non-Go check and before the first Go invocation
in the product path (`ALL_PKGS="$(repo_packages …)"` runs `go list`). Plus a
requiredness fix in the unknown-class branch. No other file touched; no new
abstraction (existing `record_row` is reused directly, following the
`check_candidate_fingerprint` precedent of a hand-recorded row).

### C1 — pinned module environment

The section captures `AMBIENT_GOWORK="${GOWORK:-}"` (before overriding
anything), then exports all seven variables explicitly:

| variable | pinned value | why this value |
|---|---|---|
| `GOFLAGS` | `-mod=readonly` | the C2 mechanism: forces read-only module mode regardless of ambient `GOFLAGS` or a `go env -w` file (process env beats both — verified live) |
| `GOPROXY` | `off` | aligns with `.github/workflows/ci.yml` (`GOPROXY: "off"`); no download can occur |
| `GOWORK` | `off` | workspace mode can never engage for any `go` the verifier runs; the fail-closed *signal* is the separate detection row below |
| `GOPRIVATE` | `""` (empty) | explicit default: ambient `GOPRIVATE=*` cannot exempt anything |
| `GONOSUMDB` | `""` (empty) | explicit default: ambient `GONOSUMDB=*` cannot bypass the sumdb |
| `GOSUMDB` | `sum.golang.org` | explicit default, set even though it equals the default |
| `GOTOOLCHAIN` | `local` | keeps every `go` invocation offline: a `go` floor the local toolchain cannot satisfy fails loudly instead of downloading (feasible today: `go.mod` says `go 1.26`, local is `go1.26.3`, and FJ-053 R4 verified `GOTOOLCHAIN=local` passes at this directive) |

Because these are exported by the script itself (process environment), they
beat the operator's ambient shell and beat any `GOENV` file for every child
process: `check_go_test`, `check_go_vet`, `check_go_build`, the `go list`
discovery in `repo_packages`, and the `go env GOWORK` probe. An informational
line (`verify-candidate: module environment pinned (…)`) prints so every run
evidences the posture; it carries no check row.

### C2 — warm cache / `-mod=mod` cannot silent-pass

`-mod=readonly` is the pinning mechanism: `go` refuses to resolve anything not
already declared in `go.mod`, so a warm `GOMODCACHE`, a reachable proxy, or an
ambient `GOFLAGS=-mod=mod` cannot turn an undeclared import into exit 0 or
rewrite `go.mod`. One requiredness change was needed for the criterion's
"(nonzero exit, `RESULT: FAIL`)" on a **task-less** run: in the unknown-class
branch (`case "$CLASS"` → `*`), the three `run_check … go.*` calls were
`required=0`, so a failing `go test` printed `FAIL (code go.test_failed)` yet
left `RESULT: PASS`/exit 0 — the run itself silent-passed. They are now
`required=1` (lines for `check_go_test`, `check_go_vet`, `check_go_build`).
Unchanged: `skip_check` rows in that branch stay `required=0` (missing `go.mod`
or missing `cmd/` still never deadlocks foundation work, per interpretation
note 1/1a), metadata `not_applicable` rows stay `required=0` (§9.3 — a required
NA would be rewritten to `gate.invalid_result`), and the product branch is
untouched (already required). The comment above the branch was rewritten to
say so and to cite C-QUAL-007.

### C3 — ancestor `go.work` fails closed, explicitly

Detection runs before any Go check and uses only env inspection plus one
offline, instant `go env GOWORK` (sanctioned by the packet):

1. **Ambient**: `AMBIENT_GOWORK` non-empty **and the path exists** → finding.
2. **Auto-discovery**: otherwise `env -u GOWORK go env GOWORK` — the pinned
   environment with `GOWORK` cleared, exactly the "clean pinned env" probe the
   criteria name. This surfaces a parent-directory `go.work` found with no
   environment variable at all, and also a `GOWORK` written into the `GOENV`
   file. Existence-gated; the literal value `off` is excluded.

On a finding: a required row `check_go_workspace` (code
`go.workspace_active`, gate G-C) is recorded — printed as
`==> module posture: no go.work in effect`, the offending path
(`workspace file in effect: <path>`), `FAIL (exit 1, code go.workspace_active)`
— and the path is also stored in the row's `reason` for report consumers. On
absence: **no row is emitted**, which is what keeps the clean task-less battery
at exactly `15 passed`.

### C4 — root `vendor/` fails closed, explicitly

`[ -d vendor ]` relative to `REPO_ROOT` (the script `cd`s there at line 36) →
required row `check_go_vendor` (code `go.vendor_present`) naming `vendor/`, with
the explanation that a root `vendor/` switches Go to `-mod=vendor`. Detection is
pure filesystem inspection; `vendor/` deeper in the tree is out of scope (only
the root changes resolution). Absence emits no row (same clean-battery
invariant).

## Probe results (1–8, run in this order)

**1. Clean battery** — `./scripts/verify-candidate`

```text
summary: 15 passed, 0 failed, 0 skipped, 0 not_applicable
RESULT: PASS        (exit 0)
```

**2. Task verify** — `./scripts/verify-candidate FJ-054` (status
implementing / stage coder, specifier evidence live; no red row)

```text
summary: 19 passed, 0 failed, 0 skipped, 4 not_applicable
RESULT: PASS        (exit 0)
```

**3. Selftest** — `./scripts/selftest`

```text
summary: 56 passed, 0 failed
exit 0
```

(run after all three edits; scenarios assert substring/row facts, none asserts
the clean row count, so the violation-only rows are invisible to it)

**4. C1 probe — each ambient variable, one at a time** (`env <var>
./scripts/verify-candidate`, `x.work` never created — `ls x.work` →
`No such file or directory`):

```text
GOFLAGS=-mod=mod             rc=0 | summary: 15 passed, 0 failed, 0 skipped, 0 not_applicable | RESULT: PASS
GOPROXY=https://proxy.golang.org rc=0 | summary: 15 passed, 0 failed, 0 skipped, 0 not_applicable | RESULT: PASS
GOWORK=/Users/mak/git/fake-jev/x.work rc=0 | summary: 15 passed, 0 failed, 0 skipped, 0 not_applicable | RESULT: PASS
GOPRIVATE=*                  rc=0 | summary: 15 passed, 0 failed, 0 skipped, 0 not_applicable | RESULT: PASS
GONOSUMDB=*                  rc=0 | summary: 15 passed, 0 failed, 0 skipped, 0 not_applicable | RESULT: PASS
GOSUMDB=off                  rc=0 | summary: 15 passed, 0 failed, 0 skipped, 0 not_applicable | RESULT: PASS
GOTOOLCHAIN=go1.99.0         rc=0 | summary: 15 passed, 0 failed, 0 skipped, 0 not_applicable | RESULT: PASS
```

All seven identical to clean (counts, RESULT, exit code).

**5. C2 probe** — full `cp -R` copy at `/tmp/fj054-c2-$$`, added
`internal/bypass/bypass.go` importing `github.com/fatih/color` (absent from
`go.mod`), ran the copy's `./scripts/verify-candidate` with ambient
`GOFLAGS=-mod=mod`:

```text
EXIT=1
==> go test (repository packages)
internal/bypass/bypass.go:3:8: cannot find module providing package github.com/fatih/color: import lookup disabled by -mod=readonly
FAIL	fake-jev/internal/bypass [setup failed]
    FAIL (exit 1, code go.test_failed)
    FAIL (exit 1, code go.vet_failed)
summary: 13 passed, 2 failed, 0 skipped, 0 not_applicable
RESULT: FAIL
go.mod unchanged: YES   (sha256 before == after)
go.sum exists: NO
```

The message names the missing declaration and the pin that refused the lookup;
`go.mod` byte-unchanged, no `go.sum` created. Copy cleaned up.

**6. C3 probes**

```text
(a) GOWORK=/tmp/fj054-a.work ./scripts/verify-candidate     EXIT=1
    workspace file in effect: /tmp/fj054-a.work
    summary: 15 passed, 1 failed, 0 skipped, 0 not_applicable
    RESULT: FAIL
(b) copy nested at /tmp/fj054-c3b-$$/repo, parent go.work at
    /tmp/fj054-c3b-$$/go.work, `env -u GOWORK` (auto-discovery, no env)  EXIT=1
    workspace file in effect: /tmp/fj054-c3b-34653/go.work
    summary: 15 passed, 1 failed, 0 skipped, 0 not_applicable
    RESULT: FAIL
(c) clean env, repo root                                                  EXIT=0
    summary: 15 passed, 0 failed, 0 skipped, 0 not_applicable
    RESULT: PASS
```

The `/tmp` workspace file (`use /Users/mak/git/fake-jev`) and both copies were
removed afterwards; the repository never gained a `go.work`.

**7. C4 probe** — `mkdir vendor` at the repo root, then `rmdir vendor`:

```text
mkdir vendor →  EXIT=1
==> module posture: no vendor/ directory
    vendor/ present at repository root switches Go to -mod=vendor
    FAIL (exit 1, code go.vendor_present)
    summary: 15 passed, 1 failed, 0 skipped, 0 not_applicable
    RESULT: FAIL
rmdir vendor →  EXIT=0
    summary: 15 passed, 0 failed, 0 skipped, 0 not_applicable
    RESULT: PASS
```

`git status --short` after: only `M scripts/verify-candidate` (this change),
the orchestrator's `M .agent/work/FJ-054/state.json`, the foreign `M AGENTS.md`
(never staged), and untracked `.agent/reports/…`. No `vendor/`, no `go.work`.

**8. Syntax** — the file is a shell script; `bash -n scripts/verify-candidate`
→ OK (the battery's own `shell syntax: bash -n scripts/*` row also passes).
`python3 -m py_compile` not applicable.

## Decisions and deviations

- **GOWORK detection is existence-gated.** The criterion says "ambient `GOWORK`
  non-empty", but the two probes pin opposite outcomes for the two cases: C1's
  probe sets `GOWORK=$PWD/x.work` with `x.work` explicitly *not created* and
  must stay `PASS`, while C3(a) points `GOWORK` at an existing workspace file
  and must `FAIL`. Reconciliation: a nonexistent file cannot put a build into
  workspace mode (and the pinned `GOWORK=off` makes the dangling value inert
  for every `go` the verifier runs), so an **existing** ambient/discovered
  workspace file is the fail-closed trigger. Both probes hold as written; if
  probe 4 is instead run with an *existing* `/tmp` workspace file, it is
  identical to probe 6(a) and fails by design (that is C3, not C1).
- **Unknown-class Go checks are now required when `go.mod` exists.** Forced by
  C2's explicit "(nonzero exit, `RESULT: FAIL`)" on a task-less run — with
  `required=0` no other mechanism can make that run exit nonzero without adding
  a row (forbidden by the `15 passed` invariant). This narrows the old comment's
  "an unknown class *or* a missing module runs them non-required": the
  missing-module and zero-package skips remain non-required (interpretation
  note 1/1b untouched), metadata NA rows remain `required=0` (§9.3), and the
  product branch is untouched. The roles-design sentence "required iff class ==
  `product`" therefore now has one carve-out (unknown class **with** a module),
  recorded in the code comment and here; flagging it for cleaner/qa review —
  it is a semantics change to an existing row, but criteria C2 requires it and
  nothing is weakened (fail-closed strictly increases).
- **Violation-only rows.** `check_go_workspace`/`check_go_vendor` are recorded
  only when they fail; a clean run emits no posture row, which is the only way
  to satisfy "clean task-less battery still reports 15 passed" while also
  "failing with a check that names" the offender. The unconditional
  `module environment pinned (…)` line keeps the passing posture visible
  without a row.
- **`GOTOOLCHAIN=local` instead of the default `auto`.** Both satisfy the probe
  (ambient `go1.99.0` is overridden either way) and both give the same clean
  outcome; `local` was chosen because the verifier's own header promises
  offline operation and `local` turns a would-be toolchain download into a loud
  failure. If a future `go` floor outruns the local toolchain, the battery
  fails closed rather than reaching for the network.
- No deviation needed on any other criterion; no off-limits file was touched,
  no spec gap encountered, no criterion left unmet.

## Verification summary

| # | command | outcome |
|---|---|---|
| 1 | `./scripts/verify-candidate` | `15 passed, 0 failed`, `RESULT: PASS` |
| 2 | `./scripts/verify-candidate FJ-054` | `RESULT: PASS` |
| 3 | `./scripts/selftest` | `56 passed, 0 failed`, exit 0 |
| 4 | seven ambient C1 mutations | each `15 passed, 0 failed`, `RESULT: PASS` |
| 5 | C2 copy probe | `RESULT: FAIL` exit 1, `go.mod` unchanged |
| 6 | C3 (a)/(b)/(c) | FAIL naming workspace / FAIL naming workspace / PASS |
| 7 | C4 `mkdir vendor` / `rmdir vendor` | FAIL naming `vendor/` / PASS |
| 8 | `bash -n scripts/verify-candidate` | OK |

outputFingerprint (after edits, `./scripts/candidate-fingerprint candidate .`):
`2c8379c31c3539f27c20e1e475e20802239c617e269e4928ae166c0ee1c50df0`.
