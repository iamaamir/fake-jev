---
stage: qa
task: FJ-054
inputFingerprint: 0495abf349c5eaa9c0ca1a59a6c6a9f68e139bc9cbcf6a3a8c22581f94357147
outputFingerprint: 0495abf349c5eaa9c0ca1a59a6c6a9f68e139bc9cbcf6a3a8c22581f94357147
taskFingerprint: ea17d10b719f94a62eb9a5c68f66391a466e7cf325961bf5cd5f8082ea8db881
gitHead: e576407
generatedAt: 2026-09-27T22:01:17Z
author: subagent/qa-1
---

# QA — FJ-054

Stage author `subagent/qa-1` (distinct from coder `subagent/coder-1`); this
artifact is the only file this stage wrote. Nothing was staged or committed.

Read before probing: `docs/agents/roles/qa.md` (role contract; the packet's
`docs/development/roles/qa.md` path does not exist — the role pack lives under
`docs/agents/roles/`), `./scripts/agent-context FJ-054`,
`.agent/work/FJ-054/{specifier,coder,cleaner}.md`, `git log --oneline -5`,
`git diff 4962c85..HEAD -- scripts/verify-candidate
docs/development/acceptance-catalog.md`, `docs/development/acceptance-catalog.md`
rows C-QUAL-005..009, `docs/superpowers/specs/2026-09-25-gauntlet-roles-design.md`
§5 and §8.3, `scripts/{verify-candidate,selftest,candidate-fingerprint}`.

Revision under test: `e576407` (tree DIRTY = orchestrator's `M state.json`,
foreign `M AGENTS.md`, untracked `.agent/reports/` — identical before and after
every probe below).

## 1. Verification evidence

### 1.1 `./scripts/verify-candidate` (task-less battery)

```text
verify-candidate: repository=/Users/mak/git/fake-jev revision=e576407
verify-candidate: tree DIRTY (0 staged, 2 unstaged, 3 untracked) — evidence is bound to e576407 plus these changes

==> candidate fingerprint computed
    PASS
==> required repository files exist
    PASS
==> scripts are executable
    PASS
==> shell syntax: bash -n scripts/*
    PASS
==> toolchain: python3 available
    PASS
==> authoritative spec present and intact
    PASS
==> all .agent JSON files parse
    PASS
==> work items match the state.json schema
    PASS            (37 items, all "ok")
==> complete items have recorded-report coverage
    PASS
==> role packs match the role-pack contract
    PASS            (cleaner, coder, hardener, qa, specifier: ok)
==> trackedBy references resolve to existing work items
    PASS
==> trackedBy items are not complete
    PASS
verify-candidate: module environment pinned (GOFLAGS=-mod=readonly GOPROXY=off GOWORK=off GOPRIVATE= GONOSUMDB= GOSUMDB=sum.golang.org GOTOOLCHAIN=local)
==> go test (repository packages)
    PASS
==> go vet (repository packages)
    PASS
==> go build ./cmd/fake-jev
    PASS

summary: 15 passed, 0 failed, 0 skipped, 0 not_applicable

RESULT: PASS
EXIT=0
```

### 1.2 `./scripts/verify-candidate FJ-054` (status implementing / stage qa)

```text
verify-candidate: task=FJ-054
…
==> allowedFiles classify to product or metadata          PASS
==> requiredStages is a valid additive-only ordered set  PASS
==> stage specifier artifact / header / task fingerprint / author   PASS x4
==> stage coder artifact / header / task fingerprint / author       PASS x4
==> stage cleaner artifact / header / task fingerprint / author     PASS x4
==> stage coder chain link                                 PASS
==> stage cleaner chain link                               PASS
==> stage specifier traces                                 PASS
==> stage cleaner tooling        SKIPPED: complexity/CRAP analyzer not implemented yet
==> stage hardener requiredness  NOT APPLICABLE: metadata
verify-candidate: module environment pinned (GOFLAGS=-mod=readonly GOPROXY=off GOWORK=off GOPRIVATE= GONOSUMDB= GOSUMDB=sum.golang.org GOTOOLCHAIN=local)
==> go test (repository packages)   NOT APPLICABLE: class metadata (design §9.3)
==> go vet (repository packages)    NOT APPLICABLE: class metadata (design §9.3)
==> go build ./cmd/fake-jev         NOT APPLICABLE: class metadata (design §9.3)

report: .agent/reports/FJ-054/report-20260927T214636Z.json
summary: 29 passed, 0 failed, 1 skipped, 4 not_applicable

RESULT: PASS
EXIT=0
```

Due set at this stage, from
`./scripts/candidate-fingerprint task .agent/work/FJ-054/state.json`:

```text
{"class":"metadata","derivedRequiredStages":["specifier","coder","cleaner","qa"],"due":["specifier","coder","cleaner"],"pending":[],"requiredStages":["specifier","coder","cleaner","qa"],"source":"derived","stage":"qa","status":"implementing","taskFingerprint":"ea17d10b719f94a62eb9a5c68f66391a466e7cf325961bf5cd5f8082ea8db881"}
```

`due = specifier, coder, cleaner` — all three artifacts present and header/chain
checked green, exactly as the packet predicted.

### 1.3 `./scripts/selftest`

```text
summary: 56 passed, 0 failed
EXIT=0
```

All 56 scenario names PASS, including `go_required_for_product`,
`metadata_go_not_required`, `go_product_zero_packages_skip`,
`required_na_invalid_result`, `verify_command_failure`,
`author_overlap_qa_equals_coder`.

### 1.4 C1 probe — seven ambient variables, one at a time (task-less, clean tree)

`env <var> ./scripts/verify-candidate`:

```text
GOFLAGS=-mod=mod                rc=0 | summary: 15 passed, 0 failed, 0 skipped, 0 not_applicable | RESULT: PASS
GOPROXY=https://proxy.golang.org rc=0 | summary: 15 passed, 0 failed, 0 skipped, 0 not_applicable | RESULT: PASS
GOWORK=/tmp/fj054-qa-c1.work    rc=0 | summary: 15 passed, 0 failed, 0 skipped, 0 not_applicable | RESULT: PASS
GOPRIVATE=*                     rc=0 | summary: 15 passed, 0 failed, 0 skipped, 0 not_applicable | RESULT: PASS
GONOSUMDB=*                     rc=0 | summary: 15 passed, 0 failed, 0 skipped, 0 not_applicable | RESULT: PASS
GOSUMDB=off                     rc=0 | summary: 15 passed, 0 failed, 0 skipped, 0 not_applicable | RESULT: PASS
GOTOOLCHAIN=go1.99.0            rc=0 | summary: 15 passed, 0 failed, 0 skipped, 0 not_applicable | RESULT: PASS
```

Counts, `RESULT` and exit code identical to the clean run (1.1) for all seven;
`ls -l /tmp/fj054-qa-c1.work` → `No such file or directory` (the `GOWORK` probe
used a non-existent path, i.e. the value cannot put any build in workspace
mode).

**C1 read literally with an *existing* workspace file** (the packet's step-4
spelling `GOWORK=<existing /tmp workspace file>`) — run for completeness:

```text
GOWORK=/tmp/fj054-qa-a.work ./scripts/verify-candidate
==> module posture: no go.work in effect
    workspace file in effect: /tmp/fj054-qa-a.work
    FAIL (exit 1, code go.workspace_active)
==> go test (repository packages)     PASS
==> go vet (repository packages)      PASS
==> go build ./cmd/fake-jev           PASS
summary: 15 passed, 1 failed, 0 skipped, 0 not_applicable
RESULT: FAIL          rc=1
```

This is probe C3(a) verbatim (see 1.6): the run fails on the *posture* row
while all three product Go checks still PASS — so the ambient value never moved
the product Go checks' outcome (C-QUAL-007) and the failure is C-QUAL-008's
fail-closed signal. Step 4 (expects PASS) and step 6a (expects FAIL) of the
packet cannot both hold for the same input; resolved in §4 below.

### 1.5 C2 probe — undeclared import with ambient `GOFLAGS=-mod=mod`

Fresh `cp -R` copy at `/tmp/fj054-qa-c2-56931/repo`; added
`internal/bypass/bypass.go` importing `github.com/fatih/color` (absent from
`go.mod`); ran the copy's task-less `./scripts/verify-candidate` with ambient
`GOFLAGS=-mod=mod`:

```text
go.mod sha256 before: a703b1650b6cbd9900fec4351cb25e5ad22a5f113dabaa07ae05b243d3b71542
C2 rc=1
go.mod sha256 after:  a703b1650b6cbd9900fec4351cb25e5ad22a5f113dabaa07ae05b243d3b71542
go.mod byte-unchanged: YES
ls: /tmp/fj054-qa-c2-56931/repo/go.sum: No such file or directory
```

FAIL lines, verbatim from the copy's log:

```text
verify-candidate: module environment pinned (GOFLAGS=-mod=readonly GOPROXY=off GOWORK=off GOPRIVATE= GONOSUMDB= GOSUMDB=sum.golang.org GOTOOLCHAIN=local)
==> go test (repository packages)
verify-candidate: excluded 1 package(s) under gitignored tool dirs (agent .agents .claude .pi .scratch)
# fake-jev/internal/bypass
internal/bypass/bypass.go:3:8: cannot find module providing package github.com/fatih/color: import lookup disabled by -mod=readonly
FAIL	fake-jev/internal/bypass [setup failed]
?   	fake-jev/cmd/fake-jev	[no test files]
?   	fake-jev/internal/cli	[no test files]
FAIL
    FAIL (exit 1, code go.test_failed)

==> go vet (repository packages)
internal/bypass/bypass.go:3:8: cannot find module providing package github.com/fatih/color: import lookup disabled by -mod=readonly
    FAIL (exit 1, code go.vet_failed)

==> go build ./cmd/fake-jev
    PASS

summary: 13 passed, 2 failed, 0 skipped, 0 not_applicable

RESULT: FAIL
```

Nonzero exit, missing declaration named, the refusing pin named, `go.mod`
byte-identical (sha256 before == after), no `go.sum` created. Copy removed.

### 1.6 C3 probes

(a) existing ambient workspace file:

```text
C3(a) rc=1
    workspace file in effect: /tmp/fj054-qa-a.work
    FAIL (exit 1, code go.workspace_active)
summary: 15 passed, 1 failed, 0 skipped, 0 not_applicable
RESULT: FAIL
```

(b) copy nested under a `/tmp` dir holding a parent `go.work`
(`/tmp/fj054-qa-c3b-64525/go.work`, `use ./repo`), run with `env -u GOWORK`
(auto-discovery, no environment variable):

```text
C3(b) rc=1
==> module posture: no go.work in effect
    workspace file in effect: /tmp/fj054-qa-c3b-64525/go.work
    FAIL (exit 1, code go.workspace_active)
summary: 15 passed, 1 failed, 0 skipped, 0 not_applicable
RESULT: FAIL
```

(c) strict clean environment, repository root (`env -i HOME="$HOME" PATH="$PATH"`):

```text
rc=0
summary: 15 passed, 0 failed, 0 skipped, 0 not_applicable
RESULT: PASS
```

Also run (O2 shape): `GOWORK=off ./scripts/verify-candidate` on the clean repo
→ `rc=0 | summary: 15 passed, 0 failed, 0 skipped, 0 not_applicable | RESULT: PASS`.
No ancestor `go.work` exists anywhere above the repository (otherwise 1.1 and
1.6c could not be green).

Both `/tmp` copies and `/tmp/fj054-qa-a.work` removed; no `go.work` was ever
created inside the repository.

### 1.7 C4 probe — root `vendor/`

```text
C4 mkdir rc=1
==> module posture: no vendor/ directory
    vendor/ present at repository root switches Go to -mod=vendor
    FAIL (exit 1, code go.vendor_present)
summary: 15 passed, 1 failed, 0 skipped, 0 not_applicable
RESULT: FAIL

C4 rmdir rc=0
summary: 15 passed, 0 failed, 0 skipped, 0 not_applicable
RESULT: PASS
```

`[ -e vendor ]` after → absent; `git status --short` unchanged from the
pre-probe state.

### 1.8 Syntax

```text
bash -n scripts/verify-candidate → rc=0 (no output)
```

(The battery's own `shell syntax: bash -n scripts/*` row is green too.)

### 1.9 Fingerprints and chain

```text
./scripts/candidate-fingerprint candidate .
0495abf349c5eaa9c0ca1a59a6c6a9f68e139bc9cbcf6a3a8c22581f94357147
```

Equals the cleaner's `outputFingerprint` — used as this artifact's
`inputFingerprint` and `outputFingerprint` (no tracked file was edited by this
stage; `.agent/work/` is excluded from the candidate digest by
`EXCLUDED_PREFIXES`, so writing this file does not move it).

Chain, read from the artifacts' front matter and re-checked by
`verify-candidate`'s `stage coder chain link` / `stage cleaner chain link`
rows (both PASS):

```text
specifier.md  in 297b6528321db380641e79c8d702dbf4782c9e71b9fa2a667e4b9c5f97359033
              out 297b6528321db380641e79c8d702dbf4782c9e71b9fa2a667e4b9c5f97359033
coder.md      in 297b6528321db380641e79c8d702dbf4782c9e71b9fa2a667e4b9c5f97359033
              out 2c8379c31c3539f27c20e1e475e20802239c617e269e4928ae166c0ee1c50df0
cleaner.md    in 2c8379c31c3539f27c20e1e475e20802239c617e269e4928ae166c0ee1c50df0
              out 0495abf349c5eaa9c0ca1a59a6c6a9f68e139bc9cbcf6a3a8c22581f94357147
all three:    taskFingerprint ea17d10b719f94a62eb9a5c68f66391a466e7cf325961bf5cd5f8082ea8db881
authors:      specifier session/opencode-main | coder subagent/coder-1
              cleaner subagent/cleaner-1 | qa subagent/qa-1 (≠ coder)
```

### 1.10 Scope of the change under review

```text
$ git diff 4962c85..HEAD --stat
.agent/work/FJ-054/cleaner.md | 165 +++
.agent/work/FJ-054/coder.md   | 252 +++
.agent/work/FJ-054/state.json |  14 +-
scripts/verify-candidate      |  78 +-
4 files changed, 497 insertions(+), 12 deletions(-)

$ git diff 4962c85..HEAD -- docs/development/acceptance-catalog.md
(empty — catalog untouched by the coder and cleaner commits)
```

Only `scripts/verify-candidate` is a product-of-this-item file; no product file,
no dependency, no CI workflow, no `scripts/selftest` change (non-goals intact).

## 2. Acceptance bullet rulings

| # | bullet | ruling | evidence |
|---|---|---|---|
| 1 | C-QUAL-005 offline (untouched, still green) | **MET** | Catalog row present (`docs/development/acceptance-catalog.md:171`); `git diff 4962c85..HEAD -- docs/development/acceptance-catalog.md` is empty and `git show f325b98 --stat` is `1 file changed, 3 insertions(+)` — rows 007..009 appended after C-QUAL-006, so C-QUAL-005 is byte-untouched; no golden-contract/product file appears in the change set (1.10); every run above completed green with `GOPROXY=off`/`GOTOOLCHAIN=local` and `scripts/verify-candidate` contains no `curl`, `wget`, `go get`, `go mod download`, `git fetch/clone` or `http(s)://` invocation (grep: 0 hits), so nothing in the battery reaches the network. |
| 2 | pinning: ambient GOFLAGS / warm cache cannot silent-pass a missing declaration | **MET** | C1: all seven ambient mutations (1.4) give the clean run's `15 passed, 0 failed` / `RESULT: PASS` / rc=0 — counts, verdict and exit unchanged. C2 (1.5): undeclared import with ambient `GOFLAGS=-mod=mod` → rc=1, `RESULT: FAIL`, declaration and pin named, `go.mod` sha256 identical before/after, no `go.sum`. Product-Go-check outcome is also unaffected by an in-effect workspace (1.4 literal run: three Go checks PASS, failure comes from the posture row). Catalog `C-QUAL-007` present at line 173. |
| 3 | go.work / vendor detection | **MET** | C3(a) rc=1 naming `/tmp/fj054-qa-a.work`; C3(b) rc=1 naming the auto-discovered parent `go.work` with `env -u GOWORK`; C3(c) strict `env -i` rc=0 PASS (no false positive on the clean repo); C4 `mkdir vendor` rc=1 naming `vendor/` with code `go.vendor_present`, `rmdir vendor` rc=0 PASS and no residue. Catalog `C-QUAL-008` present at line 174 (added at `f325b98`). |
| 4 | repository policy statement for replace/exclude/retract | **MET** | Catalog `C-QUAL-009` present at `docs/development/acceptance-catalog.md:175`, added by `f325b98`: “`replace`, `exclude`, and `retract` directives are not introduced into `go.mod` without explicit review.” (spec §17.2). The row *is* the policy statement; automated rejection is explicitly out of scope (specifier non-goals). |

No bullet is NOT MET; no `needs-info` escalation is raised (nothing in the
acceptance text is ambiguous once C1's `GOWORK` spelling is read with C3(a),
see §4).

## 3. Criteria / envelope cross-check (C1–C6)

- **C1** proven by 1.4 (seven mutations) — MET.
- **C2** proven by 1.5 — MET.
- **C3** proven by 1.6a/b/c — MET.
- **C4** proven by 1.7 — MET.
- **C5** proven by 1.10 + catalog line 175 at `f325b98`; battery green — MET.
- **C6** proven by 1.1 (`15 passed, 0 failed`), 1.2 (`RESULT: PASS`),
  1.3 (`56 passed, 0 failed`), 1.9 (fingerprints/chain), 1.10 (`selftest` and
  product files untouched), and the unchanged `git status --short` — MET.

## 4. Criteria-text tension recorded (not a blocker)

The packet's step 4 spells the C1 `GOWORK` probe as `<existing /tmp workspace
file>` and expects `RESULT: PASS`, while step 6a spells the same input and
expects FAIL. Both cannot hold. Specifier C1 says `GOWORK=<some.work>` and
coder.md's “Decisions and deviations” already reconciles the two: a workspace
path that does not exist cannot engage workspace mode (probe 1.4 → PASS), while
an existing one is C3's fail-closed signal (probe 1.6a → FAIL). Both readings
were run at this revision and both behave as designed; the product Go checks'
own outcome is unchanged in either case (1.4 literal run: `go test/vet/build`
all PASS). This is a wording tension between the packet and criterion C3, not a
defect in the implementation and not an acceptance miss — recorded here as
follow-up material for the orchestrator's packet wording.

## 5. Cleaner observations O1–O7

- **O1 — pinning starts after the gauntlet engine (`verificationCommands` run
  ambient): PASS.** Acceptance bullet 2 says “runs **product Go checks** with
  the module environment pinned”, and catalog C-QUAL-007 scopes the pin to “the
  product Go checks” identically; `verificationCommands` rows are a separate
  G-C family (design §8.3) and are not product Go checks. Exposure today is
  zero: 0 of 37 `state.json` files define a non-empty `verificationCommands`
  (script-wide scan), and no `go` command exists before the pin — an
  `awk 'NR<1216'` scan of `scripts/verify-candidate` finds only comments, while
  `scripts/candidate-fingerprint` invokes `git` only. Residual worth knowing:
  a future item that adds a `verificationCommands` row invoking `go` would run
  it ambient — recorded, not a defect of this item.
- **O2 — explicit `GOWORK=off` still fails when an ancestor `go.work` exists:
  PASS.** Stricter than C-QUAL-008 requires, in the fail-closed direction, and
  it never false-positives on the clean repo: strict `env -i` run PASS (1.6c),
  `GOWORK=off` PASS (1.6), dangling `GOWORK` PASS (1.4), default ambient PASS
  (1.1) — four clean-path confirmations at this revision.
- **O3 — asymmetric `!= "off"` guard: NOTE.** The ambient branch would report a
  repo file literally named `off` as an active workspace if `GOWORK=off`. Not
  reachable today (`ls off` → `No such file or directory`) and strictly
  fail-closed if ever reached; leaving it is correct (removing it would relax a
  fail). Cosmetic asymmetry recorded only.
- **O4 — design §8.3 “required iff class == `product`” vs the unknown-class
  carve-out: NOTE.** The carve-out violates **no** acceptance bullet and **no**
  catalog row — it is *required* by bullet 2 / C-QUAL-007 (“a warm module cache
  cannot turn a missing declaration into a silent pass”) together with C2's
  “nonzero exit, `RESULT: FAIL`” on a task-less run, and it is strictly
  fail-closed: metadata NA rows stay `required=0` (§9.3), the product branch is
  unchanged, and the missing-module skip stays non-required so foundation work
  cannot deadlock. The design sentence lives in
  `docs/superpowers/specs/2026-09-25-gauntlet-roles-design.md:427,573`, outside
  this item's `allowedFiles`, so it cannot be amended from here; per the FJ-050
  precedent (design-layer gaps recorded without editing the design from a
  non-allowedFiles position) the tension is recorded as follow-up material for
  a later item that owns the design file — a NOTE, not a blocker.
- **O5 — violation-only rows: PASS.** Criteria and packet both pin the clean
  battery at `15 passed` and only require a naming check *on violation*, so
  emitting `check_go_workspace`/`check_go_vendor` only when they fire is the
  only shape that satisfies both; the clean run's posture evidence is the
  unrowed banner (`verify-candidate: module environment pinned (…)`), confirmed
  absent-of-rows in 1.1 and carrying no `record_row` call.
- **O6 — row names absent from `check_code()`'s table: NOTE (latent/cosmetic).**
  Confirmed: `check_code` (lines 121–136) maps only `check_required_files` …
  `check_go_build` with a `*` → `verify.check_failed` fallback, while both new
  rows call `record_row` directly with explicit codes `go.workspace_active`
  (line 1253) and `go.vendor_present` (line 1261) — never `run_check`/
  `skip_check`. Behavior and machine codes are correct today; only a future
  refactor that routed those names through `run_check` would degrade the code
  to `verify.check_failed` (still a failure — fail-closed preserved, precision
  lost). Impact check: `scripts/selftest` has no scenario referencing either
  row name or asserting the name↔function correspondence (it expects
  `check_go_test`/`check_go_build` and validates report *fields*, lines
  646–653), so 56/56 stands and the task-less battery is unaffected.
- **O7 — `printf` vs `echo` banner style: PASS.** Purely cosmetic; the script
  already mixes both, the line's immediate neighbours are `printf`, and no
  output contract depends on the builtin.

## 6. Verdict

**READY FOR REVIEW** — all four acceptance bullets MET (§2), criteria C1–C6
MET (§3), O1–O7 clear (no CONCERN), evidence reproduced independently at the
final revision `e576407` (§1): `15 passed, 0 failed` task-less, `RESULT: PASS`
for `verify-candidate FJ-054` with `due = specifier, coder, cleaner`,
`56 passed, 0 failed` selftest, fingerprint `0495abf3…` re-derived and chained.
No blocker, no `needs-info`, no spec gap beyond the already-recorded
`spec-gap` entry in `state.json` (owned and adjudicated by this item at
`f325b98`).

This is the qa stage's observation against acceptance, not a gate verdict:
per AGENTS.md the item may only go `complete` when `./scripts/verify-candidate`
exits 0, which it does at this revision.
