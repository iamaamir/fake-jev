---
stage: hardener
task: FJ-058
inputFingerprint: 824082323515413a3dae3e1db663cb399e00af2891fefa137ad63b7b936cff55
outputFingerprint: 20531f37f909385810f6a49c231229f57bb43a864d5438e64b93905b8e8e1c58
taskFingerprint: 0aebbe106e52df5a069737102f0395e084481d4f0ab9c2b6f3aac15711a1db22
gitHead: cfe0e4563009ced4e01690773efbf17399ff4858
generatedAt: 2026-09-28T14:22:46Z
author: subagent/worker-FJ-058-hardener-artifact-correction
---

## Hardening actions

- Bound fuzz subprocess diagnostics to 1 MiB; oversized untrusted target output cannot grow guard memory without limit.
- Validate crash-input paths stay inside declaring package, reject symlinked path components and non-regular files, validate destination type, and preserve only canonical corpus files under `testdata/fuzz/<Target>/`.
- Validate every guard finding and `stats` entry in `verify-candidate` JSON contract, not only first finding.
- Added traversal regression test for crash-input preservation.

## Contract evidence

- Fuzztime positivity and overflow remain checked before discovery or subprocess execution.
- Target discovery remains offline, deterministic, package-scoped, and zero-target vacuous behavior unchanged.
- Fuzz subprocess failures remain classified as `guard.fuzz.crash` only for recognized Go crash output; unknown/build/tool failures remain tooling errors.
- Race row retains product/class and hardener-due gating, bootstrap skips, and task-less visibility/requiredness from interpretation note 3.
- All subprocesses retain `GOPROXY=off` and `GOWORK=off` environment boundaries.

## Validation evidence

- `go test ./cmd/guard` — 79 tests.
- `go test -race ./cmd/guard` — 79 tests.
- `./scripts/selftest` — 75 scenarios, 0 failures.
- `bash -n scripts/verify-candidate scripts/selftest` — completed.
- `./scripts/verify-candidate` — 20 checks, 0 failures.

## Residual risks

- Go fuzz process startup/compilation is governed by Go tooling outside configured fuzz execution window; no separate watchdog was introduced.
- Crash-output classification depends on Go toolchain diagnostic markers documented in interpretation note 9.
