---
stage: qa
task: FJ-013
inputFingerprint: a63a3d6fedef600dcdcdb8aed573eef907f8fea92e52605686e2133a0dea1b4c
outputFingerprint: a63a3d6fedef600dcdcdb8aed573eef907f8fea92e52605686e2133a0dea1b4c
taskFingerprint: 7d8bd46b210c54295204609b3375cf8457dbdd048d367c0cf830120a6e85ad8d
gitHead: 734ff6f
generatedAt: 2026-09-29T06:14:58Z
author: subagent/worker-FJ-013-qa
---

# QA — FJ-013

## Fingerprint chain

The current hardener artifact records:

- hardener input: `2e72044c97a6734dfe0811e350ac33184312405c867061432cefc8b2e9502d7e`
- hardener output: `a63a3d6fedef600dcdcdb8aed573eef907f8fea92e52605686e2133a0dea1b4c`
- task fingerprint: `7d8bd46b210c54295204609b3375cf8457dbdd048d367c0cf830120a6e85ad8d`

The current candidate fingerprint is `a63a3d6fedef600dcdcdb8aed573eef907f8fea92e52605686e2133a0dea1b4c`; the current task fingerprint is `7d8bd46b210c54295204609b3375cf8457dbdd048d367c0cf830120a6e85ad8d`. The QA author is `subagent/worker-FJ-013-qa`, distinct from coder `subagent/worker-FJ-013-coder`.

## Acceptance criteria exercised

- **C-HOST-001 — sequence before routing:** `TestTransitionAssignsSequenceBeforeRouteAndRecordsRejectedRequests` observed callback sequence values `1, 2, 3` for unknown-route, malformed, and valid inputs. Each admitted transition produced a journal record; the handler observed its sequence before returning its route/decode/validation decision.
- **C-HOST-002 — serialized transition:** `TestConcurrentSelectionSerializesState` and `TestConcurrentTransitionsAndResetAreRaceFree` exercised concurrent engine transitions. The resulting journal entries were sequence ordered, and selection/mutation occurred through the serialized engine boundary. The focused run also exercised the nil-handler non-admission guard and sequence exhaustion guard.
- **C-HOST-003 — reset racing a request:** `TestConcurrentTransitionsAndResetAreRaceFree` ran 128 concurrent selections with a concurrent reset and observed one retained static stub, invocation count no greater than the number of calls, and sequence-ordered retained records. `TestResetAtomicallyRestoresStaticAndDynamicState` observed removal of the dynamic stub, static counter/sequence rewind, journal clearing, and registration index restoration after reset.
- **C-HOST-005 — bounded journal and sticky failure:** `TestJournalFullIsBoundedStickyAndDoesNotMutateState` exercised capacity one. The second selection did not run, did not advance invocation or response-sequence state, retained only sequence 1, and exposed one `journal_full` failure with nil request sequence. Clearing history removed the retained record and failure, reset the next sequence to 1, and preserved the stub counter for the next selection.
- **C-HOST-009 — race freedom:** `go test -race ./...` exercised the repository race suite, including the concurrent transition/reset engine coverage, without a reported race. The focused concurrent test also completed under the ordinary test runner.

## Direct observations for reset and clear

`TestResetAtomicallyRestoresStaticAndDynamicState` observed that reset leaves only the static stub, rewinds its invocation and response-sequence positions, empties the journal, and permits a new dynamic registration at index 2. The bounded-journal test observed that request-history clear removes interactions and interaction-derived journal-full failure while retaining the stub invocation count, then starts the next journal at sequence 1.

## Commands and results

- `go test ./internal/engine -run 'Test(Transition|Journal|Reset|Concurrent)' -count=1 -v` — completed; all selected tests and subtests reported `PASS`.
- `go test ./internal/engine -count=1` — completed; package tests reported `ok`.
- `go test ./...` — completed; all repository test packages reported `ok` or `[no test files]`.
- `go test -race ./...` — completed; all repository test packages reported `ok` or `[no test files]`; no race report was emitted.
- `go vet ./...` — completed with no output.
- `./scripts/candidate-fingerprint candidate .` — reported `a63a3d6fedef600dcdcdb8aed573eef907f8fea92e52605686e2133a0dea1b4c`.
- `./scripts/verify-candidate FJ-013` — emitted `RESULT: PASS`; summary `48 passed, 0 failed, 3 skipped, 0 not_applicable`; report `.agent/reports/FJ-013/report-20260929T061640Z.json`. The public-surface/system-test, complexity/CRAP, and mutation/hardening checks were reported as skipped because those tools are not implemented.

## Residual risks and gaps

- This work item is engine-only. These runs do not exercise HTTP 507 mapping, host body-limit admission, control endpoints, or compatibility-profile routing; those are explicitly outside the task boundary.
- The concurrent reset test observes the post-run invariants but does not expose a synchronization hook that enumerates every possible request/reset linearization. The test and race run provide coverage of the implemented mutex boundary, not a network-level ordering proof.
- `Engine.Registry` remains publicly reachable, so callers that mutate registry internals directly can bypass the serialized engine methods; this is the hardener-documented ownership risk.

No candidate verdict is recorded in this artifact.
