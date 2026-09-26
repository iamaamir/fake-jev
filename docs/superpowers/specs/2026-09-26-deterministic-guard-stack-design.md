# Deterministic Guard Stack for Product Code — Design

Date: 2026-09-26
Status: design approved in the brainstorming session of 2026-09-26 (written-spec review pending)
Source decisions: five clarifying answers (scope, failure classes, tooling tolerance, quality bar,
sequencing) plus the architecture pick, all agreed in the brainstorming session of 2026-09-26;
section verdicts corroborated by `system_one` (see §7).

## 1. Goals and non-goals

Goals:

1. A **product-code-only** guard stack, landed as foundation items **before FJ-010** starts
   phase-1, so implementation code is born under constraint.
2. Guards all four model failure classes: **spec-conformance drift, structural rot, robustness
   holes, weak tests that lie**.
3. **Stdlib/in-repo tooling only** — every check runs offline (`GOPROXY=off`-safe): no
   third-party tools, no network, no paid services (§23 mandate).
4. **Hard floors, zero tolerance** — a violation fails `./scripts/verify-candidate` *and* CI
   identically, from one committed thresholds file.
5. **One in-repo binary** (`cmd/guard`, Go, stdlib) implements all guard logic as subcommands
   emitting stable JSON; **`verify-candidate` stays the single enforcer** (shells out, reads
   JSON, emits gate rows — precedent: `candidate-fingerprint` delegation); CI gains a job that
   runs `verify-candidate` itself.
6. **Satisfies the frozen FJ-044/045/046 acceptance unchanged** (deterministic analysis,
   stable machine codes, `gate-policy.json` exemption removal) — the guard binary is their
   vehicle; their acceptance text is never weakened.

Non-goals:

- Replacing the base layer: `go test` / `go vet` / `go build` / `go test -race` stay as they
  are today.
- Workflow or process hardening — FJ-052 owns that; this design touches no lease, status, or
  evidence semantics.
- Third-party tooling, network access, or live-upstream drift testing (§22.7 remains optional).
- Applying floors to metadata items: guard rows are **product-class checks**, mirroring the
  existing Go-check scoping of the frozen gauntlet design §8.3/§9.3 — required iff the work
  item's class is `product`, `not_applicable (stage.not_required)` for `metadata`. No new
  exemption mechanism is introduced; classification itself is unchanged.
- Coverage floors — subsumed by mutation + traceability (§3), and strictly weaker; adding a
  second score invites calibration drift.
- Running mutation or fuzz inside ordinary `go test ./...` (gate-scoped subcommands instead).

## 2. Architecture

```text
cmd/guard (Go, stdlib-only, offline)
  trace      C-ID -> test-marker traceability
  arch       import/layer rules (spec 17.3 first)
  lint       AST rules (L1 discarded errors, L2 unchecked type assertion)
  complexity complexity/CRAP             <- FJ-044 vehicle
  mutation   surviving-mutant report     <- FJ-045 vehicle
  each -> stable JSON {check, findings[], stats{}} + machine codes:
    guard.trace.untraced, guard.trace.unknown_id,
    guard.arch.forbidden_import,
    guard.lint.err_discarded, guard.lint.unsafe_type_assert,
    guard.complexity.threshold, guard.mutation.survivor,
    guard.fuzz.crash, guard.race.failed, guard.tool.failed

./scripts/verify-candidate            (single enforcer, unchanged role)
  shells out, validates JSON, maps findings to gate rows:
    G-L cleaner: guard.arch + guard.lint + guard.complexity
    G-H hardener: guard.mutation + guard.race + guard.fuzz
    G-Q qa:       guard.trace + FJ-046 system-suite command

.github/workflows/ci.yml  += "gauntlet gates: ./scripts/verify-candidate"
  (the 23-mandated test/race/vet/build jobs remain untouched)

.agent/guards.json   single committed thresholds file (sibling of gate-policy.json)
```

Key choices:

- **Guard binary = analysis checks only.** Test *execution* checks remain verify's existing
  `na_check` pattern (`go test`, `go test -race`, FJ-046's package-scoped system suite) — no
  re-architecture of what already works.
- **CI adds an additive job** running `verify-candidate`, so floors fail CI with zero logic
  duplication; §23's mandated raw commands keep their own jobs.
- **Fast vs slow split:** `trace`/`arch`/`lint`/`complexity` run on every product-class verify
  invocation (sub-second, AST-only); `mutation` runs only at the hardener gate and in CI;
  bounded `fuzz` runs at the hardener gate and in CI.
- **Activation, not blanket demand:** traceability enforces an explicit per-phase activation
  scope in `guards.json` (§3), starting empty and growing per work item.

## 3. The four guard families

### 3.1 Spec-conformance drift — `guard trace`

- **Marker convention:** a test covers a C-ID by carrying it in a subtest name:
  `t.Run("C-MATCH-007/choice-order", ...)`. `guard trace` walks `_test.go` files with
  `go/parser` and also scans `testdata/contracts` for C-ID tags (an empty result is fine —
  tags accumulate as vectors gain IDs).
- **Activation:** `guards.json` `trace.active` starts `[]`; a phase item's task packet appends
  its C-IDs **before that item's verification runs** (FJ-010 appends phase-1's during its own
  execution), and every appended ID persists for all later verify runs. Never demands tests
  for features that do not exist yet.
- **Floors (zero tolerance):** active ID with zero markers → `guard.trace.untraced`; marker
  naming an ID absent from `docs/development/acceptance-catalog.md` → `guard.trace.unknown_id`.
  Inactive-but-traced is allowed (early annotation is encouraged).
- The catalog stays **read-only** to guards — FJ-049 remains the sole catalog editor;
  activation lives in `guards.json`.

### 3.2 Structural rot — `guard arch` + `guard lint` + `guard complexity` (G-L)

- **arch — spec-driven first:** implements §17.3 verbatim: `internal/engine` must not import
  `net/http`, `os/exec`, `compat/jev`, `cli`, or control-API packages (filesystem avoidance in
  the engine is documented, not machine-checked). Additional one-way-dependency and
  transport-isolation rules live in `guards.json` and change only by design revision.
- **lint — closed initial set** (false positives kill adoption, so small and precise):
  - **L1** `err_discarded`: an error-returning call invoked as a statement, or assigned to `_`
    (via `go/types`, not name heuristics).
  - **L2** `unsafe_type_assert`: single-value type assertion `x.(T)` — the classic panic
    source.
  Any new lint rule is an explicit design revision, never a silent addition.
- **complexity (FJ-044):** cyclomatic ≤ 15 per function, CRAP ≤ 50 per function.

### 3.3 Robustness — `guard mutation` (+ race/fuzz) (G-H)

- **mutation (FJ-045):** AST mutators (operator negation, constant flip, condition deletion,
  branch swap) in **fixed mutant order**; each mutant runs `go test -count=1` under a
  `guards.json` per-mutant timeout. Floors: **overall ≥ 75%, per-package ≥ 60%**; survivors
  surface as `guard.mutation.survivor`. 0 mutants in scope = **pass** by defined convention.
- **race:** `go test -race ./...` becomes a required G-H verify row (CI already runs it per
  §23).
- **fuzz:** seed-corpus replay already rides ordinary `go test`; bounded live fuzz at G-H and
  CI: **30s per target** (`guards.json` `fuzz.fuzztime`); `guard fuzz` AST-discovers
  `func Fuzz*` targets; 0 targets = pass; a crash fails with `guard.fuzz.crash` and the
  failing input is committed under `testdata/fuzz/<Target>/` so it becomes a permanent,
  deterministic regression. This defends §22.2's invariant: untrusted input may produce a
  controlled error, never a panic or state corruption.

### 3.4 Weak tests — trace + mutation are the bite-measures

Mutation score proves assertions bite; traceability proves nothing is silently untested.
No coverage floor (non-goal in §1).

### 3.5 Thresholds (all in `.agent/guards.json`, zero tolerance)

| Floor | Value |
|---|---|
| Engine forbidden imports (§17.3) / arch violations | 0 |
| Lint findings (L1, L2) | 0 |
| Cyclomatic / CRAP per function | ≤ 15 / ≤ 50 |
| Active C-IDs traced | 100% |
| Mutation overall / per-package | ≥ 75% / ≥ 60% |
| Live fuzz per target | 30 s; any crash fails (repro input committed with the fix) |
| Race | 0 (existing check) |

Thresholds are committed with FJ-056 before measurement; changing any value requires an
explicit design revision — never a silent loosening.

## 4. Gate integration, exemptions, fail-closed semantics

**Exemptions are per-code, not per-gate.** Each entry in `.agent/gate-policy.json` lists
`blocks: [failure codes]`, a `scope` of item classes, and a `trackedBy` item that verify §9.2
requires to exist and be **non-complete**. Today's bootstrap entries block only
`stage.tooling_absent` — the row a gate reports while its planned tooling does not exist.

Consequences (normative):

1. **`guard.*` codes appear in no `blocks` list** — guard checks enforce the moment their
   tooling item lands, regardless of any remaining bootstrap exemption. There is no
   either-order choreography and no "≥1 check present" precondition.
2. FJ-044/045/046's frozen acceptance (drop the exemption entry; verify exits 0) is
   **mechanically enforced**: a leftover entry at completion fails verify's `trackedBy` rule,
   so each entry must be removed together with the `stage.tooling_absent` row its item
   replaces with real tool output.
3. An exempt gate's blocked code is skipped with a visible reason and machine code — never
   silently half-run.
4. **Class scoping mirrors the Go checks** (frozen design §8.3/§9.3): every guard row is
   required iff the work item's class is `product`; metadata items receive
   `not_applicable (stage.not_required, class metadata)` for guard rows exactly as they
   already do for `go test`/`go vet`/`go build`. Gate-policy `scope` matching is untouched.

**Vacuous-pass (restated):** each guard check *proves the absence of violations in what
exists*. Checks are required at tooling-item landing and pass red-free over an empty or young
codebase by defined convention: no findings, `trace.active: []` = 100%, 0 mutants = pass,
0 fuzz targets = pass. Floors bind automatically as code grows. There is no "has product
code yet?" heuristic — that would be a fail-open branch.

**Fail-closed (tool/config integrity):** guard binary fails to build → fail;
`guards.json` missing or unparsable → fail; guard JSON missing required fields or naming an
unknown `check` → fail; nonzero exit with no parseable findings → fail with
`guard.tool.failed`. Verify never crashes on guard output (existing error-mapping discipline).

**Selftest scenarios** (added with the corresponding items): missing/malformed thresholds
file; malformed guard JSON → fail row; each floor violation → correct code and red result;
vacuous-pass on empty repo; exempt-gate skip is visible; 0-mutant convention; unknown-ID and
untraced-active trace failures; empty activation passes.

## 5. Work items and sequencing

New items (intent; task packets formalize acceptance and `allowedFiles` at execution):

| Item | Delivers |
|---|---|
| **FJ-056 guard core** | `cmd/guard` skeleton, JSON contract, machine-code registry; `arch` subcommand (§17.3 + `guards.json` rules); `lint` subcommand (L1, L2); `.agent/guards.json` with **all** §3.5 thresholds; verify rows `guard.arch`/`guard.lint` at G-L with fail-closed invocation; CI `gauntlet gates` job; matching selftest scenarios |
| **FJ-057 guard trace** | `trace` subcommand (test-AST + testdata marker scan); `trace.active` starts `[]`; verify row at G-Q; selftest: untraced-active, unknown-id, empty-pass |
| **FJ-058 guard race+fuzz** | G-H rows: `go test -race ./...` (existing `na_check` pattern) and the `fuzz` runner (target discovery, `fuzztime`, 0-target pass, crash → committed repro input) |

**Ownership pin:** FJ-058 owns the fuzz *runner*; **FJ-040 (phase-4) owns fuzz *targets***
and finds the machinery already in place — its frozen acceptance is untouched. Mutation needs
no target creation (it walks all product packages), so FJ-045's acceptance also stands.

**Existing items:** FJ-044 and FJ-045 implement their analyzers **as `cmd/guard`
subcommands** (vehicle chosen by this design; their acceptance stays byte-unchanged — their
non-semantic `nextAction` gains a reference to this design at pickup). FJ-046 remains a *test
package* (G-Q harness), not a subcommand; it waits for a real HTTP surface (after FJ-018),
while trace enforces from FJ-057 onward regardless — the per-code exemption model makes this
work. FJ-047 and FJ-049 are unchanged.

**Sequence:**

```text
FJ-056 -> FJ-057 -> FJ-058 -> FJ-044 -> FJ-045 -> FJ-010 (phase-1 begins)
   ... FJ-046 at data-plane maturity (~after FJ-018); FJ-040 stays phase-4
```

**Serialization precondition (out of scope of this design):** FJ-052 — an independent
process-hardening track — must complete before FJ-056 starts, because both edit
`scripts/verify-candidate` and `scripts/selftest`. One writer at a time on the shared files
(`scripts/verify-candidate`, `scripts/selftest`, `.agent/guards.json`) avoids rebase theater.

**Item class and completion:** all new items are class `product` (they ship Go code); the
completion rule is unchanged — `./scripts/verify-candidate <id>` exits 0 for the candidate
revision.

## 6. Risks and calibration

- **Uncalibrated thresholds:** ≤ 15/≤ 50 and ≥ 75%/≥ 60 are committed before any measurement.
  If first real measurements show the numbers misjudge this codebase, the fix is an explicit
  design revision with evidence — never an in-flight edit.
- **Mutation runtime:** per-mutant test runs scale with package count; bounded by the
  per-mutant timeout and by running only at the hardener gate and CI, never on every verify
  invocation.
- **Lint false positives:** the closed L1/L2 set exists precisely to keep trust; expansion is
  a design revision with its own review.
- **Fuzz nondeterminism:** discovery is bounded-random, but a crash saves its input into the
  committed corpus, so every fuzz failure is reproducible from then on — real failures, not
  flakes.

## 7. Decision log

| Decision | Verdict | Evidence |
|---|---|---|
| Scope = product code only | user pick | clarifying Q1 |
| All four failure classes guarded | user pick | clarifying Q2 |
| Stdlib/in-repo tooling only | user pick | clarifying Q3 |
| Hard floors, zero tolerance | user pick | clarifying Q4 |
| Guard foundation before FJ-010 | user pick | clarifying Q5 |
| Single `cmd/guard` binary architecture | user pick | approaches Q |
| §1–§3 approved | — | session ("lgtm", "ok") |
| §4 approved with mandate | system_one: `accept_with_mandate` 0.97, conf. 0.95 (initial soundness `noul` 0.63) | then amended to the per-code exemption model after reading `gate-policy.json`: `amend_section4_now` conf. 0.99; behavioral-vs-mechanical probe `noul` 0.44 |
| §4 amendment approved | user pick | — |
| §5 initial verdict | system_one: `approve_with_adjust` 0.53, conf. 0.29 (`approve` 0.19, `rework` 0.28) | — |
| §5 adjusted (FJ-052 reclassified as serialization precondition) | user delegated to designer; system_one re-run: `approve` 0.57, conf. 0.36 (`rework` 0.01) | direction shift accepted as confirmation |
| Guard items renumbered FJ-053/054/055 → FJ-056/057/058 | collision fix | concurrent agent allocated FJ-053 (go.mod requires cleanup, closed) and FJ-054 (verifier module posture, queued) between the first commit (`2186b66`) and this revision; FJ-055 left free for the other agent's next sequential allocation |
