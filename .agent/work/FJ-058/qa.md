---
stage: qa
task: FJ-058
inputFingerprint: 5a1823c357476214dff1127b3f690202be37d0ab33c5b6c1ea28447c35966e94
outputFingerprint: 5a1823c357476214dff1127b3f690202be37d0ab33c5b6c1ea28447c35966e94
taskFingerprint: 0aebbe106e52df5a069737102f0395e084481d4f0ab9c2b6f3aac15711a1db22
gitHead: 74df501e06370b7a2e4bbab87379196b864c1205
generatedAt: 2026-09-28T14:20:11Z
author: subagent/qa-review-FJ-058
---

## Review scope

Fresh QA exercise against candidate fingerprint `5a1823c357476214dff1127b3f690202be37d0ab33c5b6c1ea28447c35966e94`.
Read FJ-058 specifier, coder, cleaner, and hardener artifacts; acceptance catalog; QA role pack; guard design §3.3/§4/§5; plan interpretation note 3 and fuzz note 9.

## Criteria exercised

- Race row: task-less verify output included `go test -race (repository packages)` at G-H with visible non-required row; repository package execution completed. Selftest exercised product due pass/fail, metadata `not_applicable`, task-less visibility, bootstrap behavior, and stage-not-due row absence.
- Fuzz runner: focused tests exercised offline `Fuzz*(*testing.F)` discovery and filtering, positive fuzztime rejection, zero-target pass, seed/live crash classification, crash-input preservation under `testdata/fuzz/<Target>/`, path traversal rejection, build-failure closure, and bounded live pass.
- Scheduling and class scope: selftest exercised hardener-due product rows, metadata rows, task-less requiredness, and product hardener-not-due omission.
- Selftest acceptance: full scenario battery reported 75 scenarios with 0 failures.

## Commands and observations

- `./scripts/selftest` — exit 0; `summary: 75 passed, 0 failed`.
- `go test ./cmd/guard` — exit 0; 79 tests passed in 1 package.
- `go test -race ./cmd/guard` — exit 0; 79 tests passed in 1 package.
- `./scripts/verify-candidate` — exit 0; `summary: 20 passed, 0 failed, 0 skipped, 0 not_applicable`; race and fuzz rows appeared in output.
- `go test ./cmd/guard -run 'Test(DiscoverFuzzTargets|RunFuzz|PreserveCrashInput)' -count=1 -v` — exit 0; 8 focused tests passed.
- `go test ./...` — exit 1 outside FJ-058 focused package scope; 79 tests passed and package `examples` failed because `agent/skills/golang-cli/assets/examples/output.go` imports unavailable `github.com/fatih/color`.
- `go test -race ./...` — same existing `examples` package dependency failure; FJ-058 package race tests remained green.
- `bash -n scripts/verify-candidate scripts/selftest && git diff --check` — exit 0.
- `./scripts/verify-candidate FJ-058` — exit 1 at evidence stage rows (`qa` artifact absent during command and hardener chain stale); later G-H execution rows completed with race and fuzz output. This task-ful run was used to observe scheduling, not to alter predecessor evidence.
- `./scripts/candidate-fingerprint candidate` — `5a1823c357476214dff1127b3f690202be37d0ab33c5b6c1ea28447c35966e94`.
- `./scripts/candidate-fingerprint task .agent/work/FJ-058/state.json` — task fingerprint `0aebbe106e52df5a069737102f0395e084481d4f0ab9c2b6f3aac15711a1db22`; class `product`; hardener due.

## Limits and follow-up inputs

- Full `go test ./...` and `go test -race ./...` are not clean because repository-wide discovery reaches existing `agent/skills/golang-cli/assets/examples` with missing third-party dependency; task-less verifier excludes that gitignored tool directory and uses its repository package set.
- Task-ful evidence chain needs workflow handling: command observed missing QA artifact and hardener input/output chain mismatch against cleaner output. No predecessor artifact or production file was changed during QA.
- Working tree had no staged files before QA artifact creation; unrelated dirty and untracked files remain outside QA scope.
