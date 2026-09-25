# Gauntlet Roles and Expertise Packs — Design

Date: 2026-09-25
Status: frozen design (behavior and invariants only — not an implementation plan)
Source decisions: Sections A, B, C agreed in the brainstorming session of 2026-09-25

## 1. Purpose

Give this repository a portable team of expert workers: deterministic **stage
packs** (evidence boundaries that decide acceptance) plus advisory **expertise
packs** (domain knowledge that helps reasoning), implemented entirely as repo
convention so any harness, agent, or model can follow them.

The design obeys two constraints that outrank convenience:

1. **Generation is probabilistic, acceptance is not.** Only `verify-candidate`
   may produce `pass` / `fail` / `skipped` / `not_applicable`. Stage artifacts
   record what a worker did; reports record whether it was acceptable.
2. **No harness-specific files.** Everything is plain files under `docs/`,
   `.agent/`, and `scripts/`, discoverable through `AGENTS.md`.

## 2. Authority hierarchy (frozen)

```text
1. specification + executable contracts
2. work-item acceptance criteria
3. role pack
4. expertise pack
5. general model knowledge
```

Recorded once in `docs/agents/README.md` with a short pointer from `AGENTS.md`.

**Advisory rule (normative).** Expertise packs are advisory. They may explain
how to reason, what to inspect, and what questions to ask. They MUST NOT
introduce product requirements, acceptance conditions, architecture rules, or
behavioral constraints not traceable to the specification or executable
contracts. This prevents `golang.md` or `staff-architect.md` from becoming a
shadow specification.

Escalation to a human follows the same hierarchy: if resolution would change or
establish strategic policy, architecture boundaries, specification semantics, or
public contracts not already determined by higher-authority artifacts, the
worker records a blocker and stops (`needs-human` or `spec-gap`). Decisions
fully constrained by the specification are resolved with the `staff-architect`
expertise pack and never escalate.

## 3. Layout (Section A)

```text
docs/agents/
    README.md                 authority hierarchy + index (new)
    roles/                    stage packs — workflow authority
        specifier.md
        coder.md
        cleaner.md
        hardener.md
        qa.md
    expertise/                advisory domain knowledge
        product-manager.md    required for specifier, optional for qa
        staff-architect.md    required for cleaner
        golang.md             required for coder and hardener, optional for cleaner
        ui-ux.md              optional for qa
        security.md           optional for hardener
    issue-tracker.md          (exists)
    triage-labels.md          (exists)
    domain.md                 (exists)
.agent/
    schema/
        role-pack.schema.json
        stage-artifact.schema.json
        work-item.schema.json
    gate-policy.json          bootstrap exemptions (new, committed)
    work/<task-id>/           state.json + stage artifacts (exists)
scripts/
    candidate-fingerprint      workspace and task digests (new)
    selftest                   harness scenario suite (new)
```

Pack roles (settled):

| Kind | Authority | May contain |
|------|-----------|-------------|
| role pack | workflow authority — defines a stage's inputs, output artifact, gate, escalation | responsibilities, contract, pointers |
| expertise pack | advisory only | judgment, heuristics, inspection guidance, pointers |
| specification | source of truth | everything normative |

Packs are short (target 30–50 lines). Packs reference; the specification rules.

## 4. Role pack contract

Each role pack begins with YAML front matter, then prose. Example
(`docs/agents/roles/specifier.md`):

```yaml
---
id: specifier
stage: specifier
expertise:
  required:
    - product-manager
  optional: []
inputs:
  - work-item.objective
  - spec:13
  - spec:37-44
  - acceptance-catalog
output: .agent/work/<id>/specifier.md
gate: G-S
escalate:
  - condition: spec-change-required
    blocker: spec-gap
---
```

`check_role_packs` validates every role pack and fails closed on any violation:

- `id == stage`, both in the stage vocabulary `specifier|coder|cleaner|hardener|qa`;
- `gate` ∈ `G-S, G-C, G-L, G-H, G-Q`;
- `output` equals the canonical stage path `.agent/work/<id>/<stage>.md`;
- `expertise.required` and `expertise.optional` are arrays, even when empty
  (a bare string fails; nothing is coerced);
- every named expertise pack resolves to an existing file in
  `docs/agents/expertise/` — dangling references fail, including optional ones;
- `expertise.required ∩ expertise.optional = ∅`; no duplicate entries;
- `escalate[].blocker` ∈ `spec-gap, needs-info, needs-human, wontfix`;
- **unknown front-matter keys fail** (a typo such as `expertice:` is an error,
  never ignored).

Parsing: PyYAML when importable. When PyYAML is absent the check reports
`skipped` with code `toolchain.pyyaml_missing` — visible, never silently green —
and blocks completion only while that code is *not* waived by `gate-policy.json`
(§9.2, where a matched exemption reports it as `toolchain.pyyaml_exempt`).

### Stage → expertise mapping (frozen)

| Stage | required | optional |
|-------|----------|----------|
| specifier | `product-manager` | — |
| coder | `golang` | — |
| cleaner | `staff-architect` | `golang` |
| hardener | `golang` | `security` |
| qa | — | `ui-ux`, `product-manager` |

ROLE decides what evidence is required. EXPERTISE helps reason about the
particular task. Optional packs are pointers, read on demand — never injected
automatically.

## 5. Stage artifacts

Every executed stage writes one artifact:
`.agent/work/<id>/{specifier,coder,cleaner,hardener,qa}.md`.

Artifacts are evidence, not diaries, and **they never declare a verdict**. Only
`verify-candidate` may emit `pass|fail|skipped|not_applicable`. Required header:

```yaml
---
stage: hardener
task: FJ-017
inputFingerprint: 9f2c…
outputFingerprint: 4ab1…
taskFingerprint: 7c2e…
gitHead: a5a7846
generatedAt: 2026-09-25T11:00:00Z
---
```

- `inputFingerprint` / `outputFingerprint`: workspace content digest before and
  after the stage (§6.1).
- `taskFingerprint`: the task-semantic digest (§6.2) computed when the stage
  ran; verification requires it to equal the current task fingerprint (§7).
- `gitHead`: diagnostic metadata only. It is never evidence identity.
- No `status`/verdict field exists in an artifact. Verdicts
  (`pass|fail|skipped|not_applicable`) are produced solely by
  `verify-candidate`, in reports.

Below the header: concise prose — what was done, what was found, what remains.

## 6. Fingerprints (frozen)

Two independent identities, two staleness dimensions:

- **candidate fingerprint** (§6.1) — what the workspace contains; a content
  change invalidates downstream evidence;
- **task fingerprint** (§6.2) — what work is being accepted; a task-semantics
  change invalidates the recorded evidence itself.

Both are sha256 over canonical forms, computed offline with no timestamps.

### 6.1 Candidate fingerprint

A revision means **workspace content**, not `HEAD`. Two workers on uncommitted
trees must be distinguishable.

```text
fingerprint = sha256( canonical JSON array, sorted by path, of:
    { path, mode, contentSha256 }   for every file that is
        tracked (any working-tree state)
        OR untracked and not gitignored
    minus EXCLUDED prefixes )
```

Frozen mode/path semantics, so independent implementations agree:

- `path` is repository-relative with `/` separators (no `./` prefix, no
  absolute paths), byte-exact otherwise.
- Tracked files use the effective worktree mode: `120000` for symlinks
  (content hash = bytes of the link target), else `100755` when the
  executable bit is set, `100644` otherwise.
- Untracked regular files: `100755` iff the executable bit is set, else
  `100644`; content hash = file bytes.
- Untracked symlinks: `120000`, content hash = bytes of the link target.
- `EXCLUDED = [".agent/work/", ".agent/reports/", ".agent/logs/", ".scratch/"]`.
  Evidence and runtime files must not perturb the fingerprint they record —
  writing `hardener.md` cannot change `outputFingerprint`.
- Gitignored paths are excluded automatically (tool installs, build output).
- Content only: a commit with identical content yields the identical
  fingerprint, so chains survive commits. `HEAD` is never part of identity.

### 6.2 Task fingerprint

`taskFingerprint = sha256(canonical JSON of the task object)`, where the task
object contains exactly these fields (frozen set), taken from `state.json`
except where noted:

- `id`, `objective`, `nonGoals`, `acceptance`, `specReferences`,
  `allowedFiles`, `verificationCommands` — values as stored;
- `requiredStages` — the **effective** value: the explicit list when present,
  otherwise the value derived by policy (§8.2), so an override that restates
  the default hashes identically to no override;
- `requiredStagesJustification` — only when present (§8 fields).

`verificationCommands` is semantic because it records work-item-specific
mandatory verification beyond the repository-wide gauntlet; it MUST NOT
contain optional debugging, exploratory, or convenience commands.

```text
verificationCommands are requirements, not suggestions.
```

Enforcement: when invoked with a task id, `verify-candidate` executes each
entry from the repository root (offline, exit 0 = pass) and reports it as a
required check — `task.verify_command_failed` on failure — attributed to
G-C (§8.3, §9.3); an empty list contributes no checks. An entry that would
invoke `scripts/verify-candidate` itself fails `task.verify_command_recursive`
— the gauntlet does not recurse. A field the fingerprint treats as acceptance
but verification never enforces would break the repository's central
principle, so execution is part of this design, not an optional follow-up.

Migration note: every current work item still stores
`./scripts/verify-candidate <id>` in this field — a display pointer, not an
acceptance command. Those entries must be re-seeded to real acceptance
commands before enforcement lands; the implementation plan sequences this.

Canonical JSON: keys sorted lexicographically, no insignificant whitespace,
UTF-8, arrays in stored order; a field that is absent (or `null`) is omitted
from the object entirely. Classification failure (`policy.unknown_scope`)
prevents derivation and fails verification first (§8.2).

Every field not listed above is non-semantic by default and excluded —
`title`, `phase`, `assignedRole`, `relevantFiles`, `baseRevision`,
`candidateRevision`, `status`, `stage`, `leaseOwner`, `attempts`,
`lastVerification`, `reviews`, `blockers`, `nextAction`, `notes`. The
include-set is frozen: making another field semantic requires a design
revision. Workflow-only changes (status transitions, lease moves) therefore
leave `taskFingerprint` unchanged by construction.

Consequence, fail-closed by design: a task-semantics change stales **every**
existing stage artifact at once — each recorded the prior task fingerprint —
so evidence must be re-recorded against the new acceptance.

- One implementation, `scripts/candidate-fingerprint`, shared by
  `verify-candidate` and `agent-context`, computes both fingerprints (the task
  fingerprint from a `state.json` path), so verification and display cannot
  disagree. Deterministic, offline, no timestamps.

## 7. Evidence chain and invalidation (B8/B9)

Invariant: **stage evidence is fingerprint-bound. Candidate content changes
invalidate downstream evidence; task-semantics changes invalidate the recorded
evidence itself.**

For a required stage B whose artifact is due (§8.1), verification requires the
chain link to the immediately preceding required stage A:

```text
B.inputFingerprint == A.outputFingerprint
```

| Condition | Result | Code |
|-----------|--------|------|
| chain mismatch | `fail` | `stage.evidence_stale` |
| artifact `taskFingerprint` ≠ current task fingerprint (§6.2) | `fail` | `stage.evidence_stale` |
| missing/unparseable header, unknown `stage`/`task` | `fail` | `stage.evidence_invalid` |
| artifact absent for a stage due under §8.1 | `fail` | `stage.artifact_missing` |

The check belongs to stage B's gate (each stage gate validates its own link),
so no new gate id is introduced. The task-fingerprint check is a separate
check from the chain check — distinct `name`, same code — so reports tell
candidate staleness from task staleness. A cleaner that edits code after
hardening produces a new fingerprint; `hardener` and `qa` no longer chain and
cannot complete.

`agent-context` recomputes the current candidate and task fingerprints itself
(never trusts stored prose) and renders each stage as `fresh`, `stale`,
`missing` (due and absent), `invalid` (present but unparseable), `pending`
(not yet due under §8.1), or `not required` (§8.3).

**Terminal freshness.** When `status` is `verifying`, the current stage's
`outputFingerprint` must equal the current workspace fingerprint; when `status`
is `review` or `complete`, the last required stage's `outputFingerprint` must
equal it. A mismatch fails `stage.evidence_stale`. `implementing` and `blocked`
have no terminal check (work in progress; blocked preserves evidence as-is),
and `planned`/`ready` have no stage to check. The check is evaluated only when
the relevant artifact exists — a missing artifact already fails the table
above — and belongs to that stage's gate, so no new gate id is introduced.

**Display rule.** A broken link is attributed to its target stage: the stage
whose `inputFingerprint` no longer matches its predecessor's `outputFingerprint`
renders `stale`, while the predecessor stays `fresh`. The terminal freshness
check attributes `stale` to the stage it covers (current stage under
`verifying`; last required stage under `review`/`complete`). A recorded
`taskFingerprint` mismatch marks that artifact `stale` too — when task
semantics change, every existing artifact renders `stale` at once. The first
required stage has no predecessor link — its `inputFingerprint` is recorded for
diagnostics only. A stage superseded by later, chain-intact work still renders
`fresh`.

## 8. Stage state machine and `requiredStages`

`state.json` gains three optional fields:

- `stage` — current stage or `null`;
- `requiredStages` — explicit override; when absent it is derived by policy
  (§8.2). Overrides are **additive-only** in v1: the explicit list may add
  stages the policy derives, but must still contain every derived stage in
  canonical order. Removing a derived stage fails
  `policy.stage_reduction_forbidden`;
- `requiredStagesJustification` — a non-empty string is required whenever the
  explicit list differs from the derived value (i.e. whenever stages were
  added), and the field must be absent when the list equals it; either
  violation fails the schema check.

### 8.1 State rules (frozen)

| `status` | `stage` | Required artifacts |
|----------|---------|--------------------|
| `planned`, `ready` | `null` | none |
| `implementing` | current stage | required predecessors |
| `verifying` | current stage | required predecessors + current artifact |
| `review` | last required stage | all required stage artifacts |
| `complete` | last required stage | all required artifacts **and** all required gates satisfied |
| `blocked` | preserved | evidence preserved, no new requirements |

`requiredStages` MUST be a unique ordered subsequence of
`specifier → coder → cleaner → hardener → qa`. Out-of-order or duplicate
entries fail (`policy.invalid_stages`). An explicit list that omits a
policy-derived stage fails `policy.stage_reduction_forbidden` (§8 fields,
additive-only rule).

### 8.2 Path-derived policy (frozen)

| Class | Matches | Default `requiredStages` |
|-------|---------|--------------------------|
| `product` | `cmd/**`, `internal/**`, `test/**`, `testdata/**`, `spec/contracts/**`, `go.mod`, `go.sum` | `S, C, L, H, Q` |
| `metadata` | `docs/**`, `.agent/**`, `scripts/**`, `.github/**`, `spec/**` (except `spec/contracts/**`), `examples/**`, `Dockerfile`, and the root files `README.md`, `AGENTS.md`, `CONTEXT.md`, `.gitignore` | `S, C, L, Q` |
| `unknown` | anything else | **fail** `policy.unknown_scope` |

Classification is prefix-ordered: `spec/contracts/**` is checked before the
generic `spec/**` rule.

The classes above cover every `allowedFiles` path in the current work items
(validated 2026-09-25); new repository areas fail closed until classified.

- Mixed classes take the **strongest** policy: `product > metadata`.
- Executable contract fixtures are product work: `testdata/**` (the golden
  corpus) and `spec/contracts/**` (currently absent, classified if
  reintroduced) are `product`, so changing a fixture requires
  `go test/vet/build` and cannot slip through under the `metadata` policy.
  `spec/contracts/**` is absent today because contracts live under
  `testdata/contracts/` (spec §45).
- `scripts/` and CI are `metadata`: the default excludes Hardener because G-H
  mutates executable product code. Hardener for a script/CI item must be added
  explicitly with `requiredStagesJustification` — and because the G-H
  bootstrap exemption is scoped to `product` (§9.2), such an opt-in blocks
  until mutation tooling ships.
- Unknown or empty classification fails closed so new repository areas are
  classified deliberately.
- Policy is evaluated from `allowedFiles` at verification time; the report
  records `{class, requiredStages, source: derived|explicit}`.

### 8.3 Gate scheduling (frozen)

- A stage's own checks — artifact presence, header/task-fingerprint validity,
  chain link, stage tools — are emitted and evaluated only when that stage is
  due under §8.1.
- A required stage that is not yet due is **pending**: it produces no check
  entry and no verdict — it is not `skipped`, which would be a verdict — so a
  future required stage can never block the current stage's verification.
- A stage outside `requiredStages` is not pending; it is evaluated as
  non-required exactly as §9.3 specifies (`not_applicable`,
  `stage.not_required`). In v1 this happens exactly once by policy:
  `product` items require all five stages; `metadata` items derive
  `S, C, L, Q`, so **Hardener is the one policy-derived stage-level
  `not_applicable` case** (an explicit additive override can bring it back,
  §8 fields). Further stage-level `not_applicable` cases are reserved for
  future policy revisions. `stage.not_required` is additionally reported by
  G-C's Go checks on `metadata` items.
- G-C's repository-integrity checks and its Go checks (required iff class ==
  `product`) are unconditional: they run on every verification, independent of
  `status` and `stage`. When a task id is given, G-C additionally executes the
  item's `verificationCommands` (§6.2), equally status-independent; without a
  task id they are unknowable and do not run.

## 9. Verification results and gate policy

### 9.1 Result semantics (frozen)

Every check carries `{name, code, required, result, exitCode, reason?}` with
`result ∈ {pass, fail, skipped, not_applicable}`.

| required | result | effect on completion |
|----------|--------|----------------------|
| yes | `pass` | allowed |
| yes | `fail` | blocked |
| yes | `skipped` (no exemption) | **blocked** |
| yes | `not_applicable` | **invalid state** — `gate.invalid_result` |
| yes | `skipped` (policy-exempt, §9.2) | visible, non-blocking, distinct code |
| no | `pass` | allowed |
| no | `fail` | visible, non-blocking (advisory checks only; none in v1 stage gates) |
| no | `skipped` | allowed |
| no | `not_applicable` | allowed (`stage.not_required`) |

`SKIPPED ≠ PASS`, `NOT_APPLICABLE ≠ SKIPPED`, and a required check can never be
`not_applicable`. Overall `verify-candidate` exits non-zero iff any required
check is `fail` or non-exempt `skipped`, or any combination is invalid.

### 9.2 Bootstrap exemptions (frozen mechanism)

Because required + skipped blocks completion, unavailable product tooling needs
an explicit, reviewable waiver — otherwise v1 deadlocks. `.agent/gate-policy.json`:

```json
{
  "exemptions": [
    { "gate": "G-L", "blocks": ["stage.tooling_absent"],
      "scope": ["product", "metadata"],
      "code": "stage.tooling_bootstrap_exempt",
      "reason": "complexity/CRAP analyzer not implemented yet",
      "trackedBy": "FJ-040" }
  ]
}
```

Shape only: `trackedBy` must be an existing work item id — the rules below
decide which id is acceptable.

An exemption matches a blocking check only when **all** of these hold:

- `gate` equals the check's gate;
- `blocks` contains the code the check would otherwise report;
- the item's path class is in `scope`.

Matching is deliberately narrow: exempting `G-L` cannot waive a missing
`python3`, and exempting `stage.tooling_absent` for `G-L` cannot waive the same
code under `G-H`. A matched check is reported `skipped` with the exemption's
`code` — visibly non-green, never a `pass`, distinguishable from an unmet
requirement by its code alone.

v1 commits four exemptions: `G-L` (`product`+`metadata`), `G-H` (`product`),
`G-Q` (`product`+`metadata`), and `G-C` with
`blocks: ["toolchain.pyyaml_missing"]` (both classes, reported as
`toolchain.pyyaml_exempt`). Their tracking work items are created before this
file is first committed.

- Every exemption requires `trackedBy`, the id of an existing work item.
  Verification fails with `policy.exemption_unknown` when no such item exists,
  and with `policy.exemption_stale` when that item has reached `complete` —
  the exemption must then be removed or the tooling shipped. Exemptions clean
  up after themselves instead of lingering.
- Exemptions are policy, not worker choice: a worker cannot switch one on for a
  single task.

### 9.3 Gate inventory

| Gate | Stage | Checks | Codes |
|------|-------|--------|---------------------|
| `G-S` | specifier | artifact + task fingerprint + header validated; every ID under `## Traces` exists in `docs/development/acceptance-catalog.md` | `stage.evidence_stale`, `stage.evidence_invalid`, `stage.trace_unknown`, `stage.artifact_missing` |
| `G-C` | coder | artifact + chain link + task fingerprint, then repository integrity (JSON, work-item schema, role packs, shell syntax, required files) — the integrity and Go portions required on every verification, independent of stage — plus `go test/vet/build`, required iff class == `product`, plus the item's `verificationCommands` when a task id is given (status-independent, §6.2) | `stage.artifact_missing`, `stage.evidence_invalid`, `stage.evidence_stale`, `go.test_failed`, `task.verify_command_failed`, `task.verify_command_recursive`, `toolchain.*`; for `metadata` items the Go checks are `not_applicable` (`stage.not_required`) |
| `G-L` | cleaner | artifact + chain link + task fingerprint + complexity/CRAP analysis | `stage.artifact_missing`, `stage.evidence_invalid`, `stage.evidence_stale`, `stage.tooling_absent`, `stage.tooling_bootstrap_exempt` |
| `G-H` | hardener | artifact + chain link + task fingerprint + mutation/hardening of executable product code | `stage.artifact_missing`, `stage.evidence_invalid`, `stage.evidence_stale`, `stage.tooling_absent`, `stage.tooling_bootstrap_exempt` |
| `G-Q` | qa | artifact + chain link + task fingerprint + public-surface/system tests | `stage.artifact_missing`, `stage.evidence_invalid`, `stage.evidence_stale`, `stage.tooling_absent`, `stage.tooling_bootstrap_exempt` |

A gate belonging to a stage outside `requiredStages` reports
`required: false, result: not_applicable, code: stage.not_required,
reason: product|metadata`.

## 10. `agent-context` view

Additive, still deterministic (no wall-clock in output):

- header line: `stage`, `requiredStages`, lease, the current candidate and
  task fingerprints;
- `GAUNTLET` block — one row per required stage:

```text
Specifier   artifact: fresh    gate: pass
Coder       artifact: fresh    gate: pass
Cleaner     artifact: stale    gate: fail
Hardener    artifact: pending  gate: not run
QA          artifact: pending  gate: not run
```

  - `fresh`/`stale`/`missing`/`invalid` follow §7 and are recomputed from
    fingerprints, never read from prose;
  - `pending` means not yet due under §8.1: no check entry exists, hence
    `not run`. It is distinct from `missing`, which is due and absent and
    fails verification;
  - gate results come from the latest report bound to the current candidate
    **and** task fingerprints; a report bound to either older fingerprint is
    ignored here (it surfaces in the `EVIDENCE FRESHNESS` block instead) —
    this keeps an artifact that renders `stale` from pairing with a `pass`
    from before the change;
  - gate display collapses required checks as: `fail` if any required check
    failed, else `exempt` if any required check is bootstrap-exempt (§9.2),
    else `pass`; `not_applicable` for stages outside `requiredStages` (§9.3),
    `not run` when the report has no entry (pending).
- `EXPERTISE` block — required packs listed as paths to read; optional packs
  listed as pointers only (progressive disclosure, no inlining).

## 11. Schemas

`.agent/schema/{role-pack,stage-artifact,work-item}.schema.json` are the
canonical contracts; `verify-candidate`'s validator is an implementation of
those contracts. `role-pack.schema.json` is fully determined by §4;
`stage-artifact.schema.json` covers the artifact front matter (§5);
`work-item.schema.json` covers the whole `state.json` contract — including the
pre-existing `leaseOwner` rules — not only the fields introduced here.

`jsonschema` is not available in this environment and is not a dependency: the
validator enforces the schema rules directly. `scripts/selftest` guarantees
only what the §12 matrix exercises: for the fields those scenarios touch, a
schema edit the validator does not reflect (or the reverse) fails a selftest
case. It does **not** prove full equivalence between the JSON schemas and the
validator — divergence in unexercised fields remains possible (§13).

## 12. Self-test matrix (frozen)

`scripts/selftest` builds fixtures in temp directories — no network, no
modification of the real repository — and asserts:

```text
required + pass                    → verify success
required + fail                    → verify failure
required + skipped                 → verify failure
required + not_applicable          → invalid state (gate.invalid_result)
required + skipped + exemption     → success, code stage.tooling_bootstrap_exempt
exemption whose trackedBy is complete → failure (policy.exemption_stale)
exemption with unknown trackedBy  → failure (policy.exemption_unknown)
exemption whose `blocks`/`gate` does not match the blocking code → failure (still blocked)
optional + skipped                 → success
optional + not_applicable          → success
stale evidence chain               → failure (stage.evidence_stale)
missing due artifact             → failure (stage.artifact_missing)
artifact header absent/invalid     → failure (stage.evidence_invalid)
dangling expertise reference       → failure
unknown front-matter key           → failure
required/optional not an array     → failure
invalid requiredStages order       → failure (policy.invalid_stages)
requiredStages omitting a derived stage → failure (policy.stage_reduction_forbidden)
additive requiredStages override   → accepted with requiredStagesJustification
additive override without justification → failure
future required stage              → pending: no check entry, current stage completes
source modified after the terminal artifact
  (verifying: current stage; review/complete: last required stage)
                                  → failure (stage.evidence_stale)
testdata/ or spec/contracts/ fixture change → class product, go test required
failing verificationCommands entry → failure (task.verify_command_failed)
verificationCommands entry invoking the verifier itself → failure (task.verify_command_recursive)
metadata item                    → Hardener gate not_applicable (stage.not_required), non-blocking
task semantics changed after the final artifact → failure (stage.evidence_stale)
workflow-only state change (status transition, lease move)
                                  → taskFingerprint unchanged
candidate source change           → candidate staleness while the task-fingerprint check passes (independent dimensions)
untracked regular file and symlink → canonical repo-relative `/` paths and 100644/100755/120000 mode rules reflected in the digest (§6.1), when creatable in the test environment
mixed path classes                 → strongest policy wins
unknown path classification        → failure (policy.unknown_scope)
unknown trace id in specifier.md   → failure (stage.trace_unknown)
fingerprint excludes .agent/work   → artifact write does not change digest
```

This applies the gauntlet to the gauntlet itself.

## 13. Known limitations (deferred, deliberately)

- **Trace validation is one-way in v1.** Specifier traces are validated
  `artifact → catalog`. Deferred: `selected acceptance → artifact
  completeness`, until the work-item `acceptance` representation is stable.
  Unknown trace IDs already fail.
- **Schema/validator parity is scenario-bound.** `scripts/selftest` proves
  parity only for the §12 scenarios; unexercised schema fields may drift
  without detection.
- **PyYAML is conditional.** Role-pack validation is `skipped` (visible,
  exempt while tracked) when PyYAML is missing; pin it when this becomes a
  required CI gate.
- **Fingerprint excludes evidence directories by design.** Edits confined to
  `.agent/work|reports|logs` are not part of candidate identity;
  `state.json` semantic changes are caught by the task fingerprint (§6.2),
  workflow-only field changes intentionally are not.
- **Evidence is workspace-wide.** The fingerprint covers the whole checkout:
  two work items interleaved in one worktree invalidate each other's chains and
  terminal freshness. Items that must both produce evidence run in separate
  worktrees or serially.
- **No harness adapters are produced.** By design: portability is achieved
  through repo convention. Harness-specific agent registries are out of scope.

## 14. Non-goals

- No workflow engine, no new status vocabulary, no stage history database.
- No generated persona prompts; packs stay short and pointer-based.
- No model-specific or harness-specific files in the repository.
- No self-certification: artifacts never carry verdicts.

## 15. How this design will be known to work

1. `scripts/selftest` passes every case in §12.
2. `./scripts/verify-candidate <task-id>` enforces §9 semantics on a real item,
   including deliberately staged candidate-stale and task-stale scenarios.
3. Continuity test: a fresh worker given only `./scripts/agent-context
   <task-id>` identifies the current stage, the required packs, fresh vs stale
   evidence, and the next action without reading any chat history.
