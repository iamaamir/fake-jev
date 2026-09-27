---
stage: specifier
task: FJ-054
inputFingerprint: 297b6528321db380641e79c8d702dbf4782c9e71b9fa2a667e4b9c5f97359033
outputFingerprint: 297b6528321db380641e79c8d702dbf4782c9e71b9fa2a667e4b9c5f97359033
taskFingerprint: ea17d10b719f94a62eb9a5c68f66391a466e7cf325961bf5cd5f8082ea8db881
gitHead: f325b98
generatedAt: 2026-09-27T20:46:09Z
author: session/opencode-main
---

# Specifier — FJ-054

## What is delivered

Turn the FJ-053 hardener findings F3 (four verified bypass conditions), F5
(ambient, unpinned posture in `verify-candidate`) and S1 (module-graph
directives unconstrained) into fail-closed verifier behavior plus the recorded
repository policy, in this order:

1. **Spec-gap record + catalog amendment, already committed** (`f325b98`,
   `docs:`): the `blockers` entry in `.agent/work/FJ-054/state.json` records
   the gap adjudication below, and `docs/development/acceptance-catalog.md`
   gains `C-QUAL-007` (pinned module environment), `C-QUAL-008`
   (`go.work`/`vendor/` fail-closed) and `C-QUAL-009` (the
   `replace`/`exclude`/`retract` review policy — this row *is* acceptance
   item 3's repository policy statement, so item 3 is satisfied at `f325b98`).
2. **`scripts/verify-candidate`** (coder stage): the product Go checks run
   under an explicitly pinned module environment, and the verifier fails
   closed on an ancestor `go.work` or a committed `vendor/` — criteria C1–C4.
3. **Cleaning + independent qa review** (cleaner and qa stages): a distinct
   qa subagent reviews the final diff and writes `qa.md` under its own author
   label (`qa.author != coder.author`).

No file outside `scripts/verify-candidate`, `docs/development/acceptance-catalog.md`
and `.agent/work/FJ-054/` is touched. The catalog change is part of this
stage's input (committed before this artifact), so input and output candidate
fingerprints are identical.

## Spec resolution — the go.work/vendor question (the packet's gap check)

The packet's nextAction asked whether the spec's silence on `go.work` and
`vendor/` is a spec-gap (record blocker, stop) or fully constrained by §17.2.
**Ruling: it is a spec-gap in the strict sense, resolved by amendment owned by
this item — never invented, never stalled.** Evidence:

- `spec/fake-jev-technical-spec-v2.md` never mentions `verify-candidate`,
  `go.work`, `vendor/`, `GOFLAGS`, `GOPROXY`, or `replace`/`exclude`/`retract`
  (grep: 0 hits for the verifier; `§17.2` covers dependency *substance*
  — stdlib preference, one YAML dependency, no frameworks — not build-graph
  verification). `spec/README.md` scopes the specification as "the source of
  truth for public product behavior", and none of the required behavior here
  is public product behavior; it is verifier and repository policy.
- Precedent: FJ-050 recorded exactly this class of `spec-gap` (per-stage
  attribution defined neither in spec nor design) with resolution owned by
  the executing item; FJ-052's specifier then wrote the amendment before
  executing. FJ-053's hardener deferred F5/S1 closure to "a future item to
  own" — this item, whose `allowedFiles` already include the amendment
  vehicle (`docs/development/acceptance-catalog.md`, the C-QUAL authority
  the acceptance itself cites).
- Enforcement anchors in existing spec text: §17.2 (only the declared
  dependency set may be relied on), §31 (`go.mod` MUST declare the minimum
  Go version — the file ambient `-mod=mod` rewrites silently), §2 (tests
  offline, deterministic, reproducible), §23 (integration tests with
  outbound network disabled). The new rows index that intent; they create no
  product behavior and cannot disagree with the specification (catalog rule 1).

## Class

`allowedFiles` = `scripts/` + `docs/` — all `METADATA_PREFIXES`, class
**`metadata`**, `requiredStages` = **specifier, coder, cleaner, qa**, no
hardener. Verified live: `./scripts/verify-candidate FJ-054` reports
`policy: {'class': 'metadata', 'requiredStages': ['specifier', 'coder',
'cleaner', 'qa'], 'source': 'derived'}`.

## Preconditions

All hold as of this stage:

- **FJ-053 is complete** — its hardener findings (F3/F5/S1) are the frozen
  source material.
- **FJ-052 is complete** — the evidence-chain, history, author and overlap
  checks this pipeline must satisfy are landed (closed by `0dd0096`).
- **No in-flight writer on `scripts/verify-candidate`** — FJ-047/FJ-050 are
  `planned`, not executing; the serialization precondition (FJ-052 complete)
  holds, and sequencing decision A makes FJ-054 the sole in-flight item.
- **The spec-gap blocker is recorded** (`f325b98`) per the ruling above.

## Criteria

Each criterion is observable and qa-executable. Probes that mutate a tree run
in throwaway copies under `/tmp` — never in the repository (no in-tree
`go build`).

**C1 — Pinned module environment (acceptance item 2; F5).**
`check_go_test`, `check_go_vet` and `check_go_build` execute under an
explicit module environment owned by the verifier, not the operator's
ambient shell. Probe: for each of `GOFLAGS=-mod=mod`,
`GOPROXY=https://proxy.golang.org`, `GOWORK=<some.work>`, `GOPRIVATE=*`,
`GONOSUMDB=*`, `GOSUMDB=off`, `GOTOOLCHAIN=go1.99.0` injected one at a time
into the ambient environment, `./scripts/verify-candidate` (task-less, on a
clean tree) yields the same `RESULT: PASS` and the same check counts as a
clean environment; exit code unchanged. The pinned posture must cover the
full F3/F5 variable set — a mutation of any one variable must not move the
outcome.

**C2 — Warm cache cannot silent-pass an undeclared import (acceptance item 2;
F3 case 1/2).**
Under the pinned environment, a source file in a throwaway copy of the tree
that imports a package absent from `go.mod` makes the Go checks fail
(nonzero exit, `RESULT: FAIL`), even with ambient `GOFLAGS=-mod=mod` and
regardless of whether the module cache already holds that package. This is
the exact F3 bypass (`-mod=mod` + warm cache → exit 0 + rewritten `go.mod`)
and it must now fail closed in `verify-candidate` itself, not only in CI.

**C3 — Ancestor `go.work` fails closed (acceptance item 3; F3 case 3).**
Workspace mode changes which modules resolve, leaving `go.mod` untouched —
F3's only silent-success shape that no `go.mod` diff detects. Probes:
(a) `GOWORK=<existing workspace file> ./scripts/verify-candidate` fails
closed with an indication naming the workspace, before any Go check can run
under workspace mode; (b) a copy of the repository placed under a temp
directory that contains a parent `go.work` (auto-discovery, no env var)
fails closed the same way; (c) the clean repository (no ancestor `go.work`,
`GOWORK` unset) stays green.

**C4 — Committed `vendor/` fails closed (acceptance item 3; F3 case 4).**
`vendor/` at the repository root changes module resolution (Go switches to
`-mod=vendor` automatically when a vendor directory is present). Probe:
`mkdir vendor` at the repo root → `./scripts/verify-candidate` fails with an
indication naming `vendor/`; `rmdir vendor` → battery green again. A
`vendor/` anywhere else is out of scope (only the root changes resolution).

**C5 — Recorded policy statement (acceptance item 4; S1).**
`docs/development/acceptance-catalog.md` row `C-QUAL-009` records that
`replace`, `exclude` and `retract` directives are not introduced into
`go.mod` without explicit review. Delivered at `f325b98`; qa confirms the
row is present and the battery stays green. Automated *rejection* of such
directives is explicitly not part of this item (see Non-goals).

**C6 — Non-regression envelope.**
`scripts/selftest` is untouched (outside `allowedFiles`) and all 56
scenarios still pass — the pinned environment must not disturb scenario
fixtures. The task-less battery stays `15 passed, 0 failed` with no network
use, no sleeps and no randomness introduced. Product files are untouched
(class `metadata`). After each stage commit,
`./scripts/verify-candidate <task>` passes at that status, and evidence
fingerprints bind per the chain rules (this stage's input == output == the
fingerprint above).

## Execution protocol

Same pipeline as FJ-052, all four stages with distinct artifacts:

- Orchestrator (this author, `session/opencode-main`) owns every state flip
  and `statusHistory` append: `planned → ready` commits with this artifact;
  then `ready → implementing` (stage `coder`, lease `subagent/coder-1`)
  before dispatching the coder; `→ verifying` for cleaner; `→ review`
  (stage `qa`, lease released) after all artifacts exist; `→ complete` with
  persisted `lastVerification` — matching FJ-052's history and coverage
  checks.
- Coder executes C1–C4 in `scripts/verify-candidate` only, runs the full
  evidence suite (battery, task verify, selftest 56/56, the C1–C4 probes
  above in `/tmp` copies), and writes `coder.md` with
  `author: subagent/coder-1`.
- Cleaner (`subagent/closer-1`) reviews diff quality and writes `cleaner.md`.
- qa is a **distinct subagent** (`subagent/qa-1`, must differ from the coder
  label) that re-runs the probes against the final revision and writes
  `qa.md`.
- Reports persist through the FJ-052 close recipe; never stage
  `.agent/reports/` before the close commit; never stage `AGENTS.md`
  (foreign edit present).

## Non-goals

- No `toolchain`/`go` directive pinning beyond today's `go` minimum (FJ-053
  deferred R3; not in acceptance).
- No `go.sum` — the repository still has no dependencies (FJ-010 introduces
  the YAML module later).
- No changes to `scripts/selftest`, `.github/workflows/ci.yml`, schemas, or
  any product file (all outside `allowedFiles`).
- No automated verifier rejection of `replace`/`exclude`/`retract` — this
  item records the policy (C5); enforcement would exceed acceptance.
- No seed/goal/task-system redesign (carve-out discipline from the guard
  stack).
