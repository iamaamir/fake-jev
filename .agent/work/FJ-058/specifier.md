---
stage: specifier
task: FJ-058
inputFingerprint: 824082323515413a3dae3e1db663cb399e00af2891fefa137ad63b7b936cff55
outputFingerprint: 824082323515413a3dae3e1db663cb399e00af2891fefa137ad63b7b936cff55
taskFingerprint: 0aebbe106e52df5a069737102f0395e084481d4f0ab9c2b6f3aac15711a1db22
gitHead: 74df501e06370b7a2e4bbab87379196b864c1205
generatedAt: 2026-09-28T13:51:47Z
author: subagent/worker-FJ-058
---
Scope: design §3.3, §4, and §5 G-H, implementation plan interpretation note 3,
and spec §22.2/§23. Existing WIP production changes are input context; this
stage changes no production files.

## Observable acceptance

- `verify-candidate` emits G-H race row using existing `na_check`/`run_check`
  behavior and runs repository `go test -race ./...`. Missing `go.mod` and a
  repository with no Go packages use closed, non-required bootstrap skips;
  present Go packages with missing or invalid tooling fail closed.
- `guard fuzz` discovers only offline `Fuzz*` functions with exactly one
  syntactic `*testing.F` parameter, rejects non-positive configured
  `fuzz.fuzztime`, passes without invoking `go test` when target count is zero,
  and runs each discovered target within configured bound.
- A fuzz crash produces `guard.fuzz.crash` and preserves failing input at
  `testdata/fuzz/<Target>/`; malformed or unclassifiable guard execution fails
  closed as a tooling failure rather than passing.
- Task-ful fuzz and race rows are emitted only for product work when hardener is
  due, and are `not_applicable` for non-product classes; before hardener is due
  they emit no row. Task-less runs require fuzz and expose race as visible but
  non-required, with CI's §23 race job enforcing race.
- Selftest covers due pass/fail, metadata non-applicability, zero targets, stage
  non-applicability, crash preservation, and bounded live pass. Acceptance
  counts full selftest battery as **75 green**, and task-less
  `./scripts/verify-candidate` exits 0.

## Traces

- C-QUAL-002 — repository race execution required by spec §23.
- C-QUAL-004 — fuzz robustness invariant required by spec §22.2.

## Explicit non-goals

- Do not create fuzz targets; FJ-040 owns target creation.
- Do not implement mutation or complexity floors; FJ-044/FJ-045 own them.
- Do not weaken or delete existing selftest scenarios.
- Do not edit specification or alter acceptance semantics outside FJ-058.
- Do not add network, model, provider, third-party, mutation, or complexity
  behavior.
