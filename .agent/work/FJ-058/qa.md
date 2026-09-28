---
stage: qa
task: FJ-058
inputFingerprint: 20531f37f909385810f6a49c231229f57bb43a864d5438e64b93905b8e8e1c58
outputFingerprint: 20531f37f909385810f6a49c231229f57bb43a864d5438e64b93905b8e8e1c58
taskFingerprint: 0aebbe106e52df5a069737102f0395e084481d4f0ab9c2b6f3aac15711a1db22
gitHead: cfe0e4563009ced4e01690773efbf17399ff4858
generatedAt: 2026-09-28T14:28:20Z
author: subagent/qa-review-FJ-058-rerun
---

## Review scope

Fresh QA exercise against the corrected hardener output and current candidate. Read
all FJ-058 predecessor artifacts, the QA role pack, the acceptance catalog rows
C-QUAL-002 and C-QUAL-004, guard design §3.3/§4/§5, and the plan's interpretation
note 3 and fuzz note 9.

## Criteria exercised

- Repository race execution: task-less verification emitted the G-H
  `go test -race (repository packages)` row as visible and non-required; the
  command completed successfully. The uncached guard-package race suite also
  completed successfully.
- Fuzz runner: focused tests exercised offline `Fuzz*(*testing.F)` discovery and
  filtering, positive fuzztime rejection, zero-target behavior, seed and live
  crash classification, crash-input preservation under
  `testdata/fuzz/<Target>/`, traversal rejection, build-failure closure, and a
  bounded live pass.
- Scheduling and class scope: the selftest scenarios exercised product due
  pass/fail, metadata non-applicability, task-less requiredness, hardener-stage
  non-due omission, crash handling, and bounded live execution. The taskful
  FJ-058 report was inspected directly: with product class and hardener due,
  `check_guard_fuzz` and `check_go_race` were both emitted at G-H with
  `required: true` and `exitCode: 0`.
- Selftest acceptance: the full battery reported 75 passed and 0 failed.

## Commands and observations

- `./scripts/selftest` — exit 0; `summary: 75 passed, 0 failed`.
- `go test ./cmd/guard` — exit 0; package test command completed.
- `go test -race ./cmd/guard` — exit 0; package race-test command completed.
- `go test -count=1 ./cmd/guard` — exit 0; uncached package tests completed.
- `go test -race -count=1 ./cmd/guard` — exit 0; uncached race tests completed.
- `go test ./cmd/guard -run 'Test(DiscoverFuzzTargets|RunFuzz|PreserveCrashInput)' -count=1 -v` — exit 0; 8 focused fuzz/discovery and preservation tests completed.
- `./scripts/verify-candidate` — exit 0; `summary: 20 passed, 0 failed, 0 skipped, 0 not_applicable`, including G-H fuzz and race rows.
- `./scripts/verify-candidate FJ-058` before this artifact correction — exit 1 only at the stale QA chain and terminal-freshness checks; its taskful G-H rows were emitted with both fuzz and race required and passing. After this artifact was corrected with the hardener output and current candidate fingerprints, the taskful command completed with exit 0; `summary: 49 passed, 0 failed, 3 skipped, 0 not_applicable`, and both G-H rows remained required with `exitCode: 0`.
- `bash -n scripts/verify-candidate scripts/selftest && git diff --check` — exit 0.
- `./scripts/candidate-fingerprint candidate` — `20531f37f909385810f6a49c231229f57bb43a864d5438e64b93905b8e8e1c58`.
- `git rev-parse HEAD` — `cfe0e4563009ced4e01690773efbf17399ff4858`.

## Limits and follow-up inputs

- The repository-wide `verify-candidate` package set excludes one package under a
  gitignored tool directory; the enforced repository package run completed. No
  production files were changed during QA.
- The working tree contains pre-existing unstaged hardener/state changes and
  untracked report artifacts outside this QA artifact; no files were staged by
  QA.
