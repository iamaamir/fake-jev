---
stage: qa
task: FJ-012
inputFingerprint: 06d1e29f3bf779c3f69c2067ce6251da6491a25f787824aa019becac9c440e47
outputFingerprint: 06d1e29f3bf779c3f69c2067ce6251da6491a25f787824aa019becac9c440e47
taskFingerprint: ab72443cf7b377355d104172139b3a0b07decb20252a42b978316e820ed86f2c
gitHead: f866a06d0726f64219d50ae05fc30c142909c7e1
generatedAt: 2026-09-28T21:04:04Z
author: subagent/worker-FJ-012-qa
---

# QA — FJ-012

## Scope and method

Read the current state and the specifier, coder, cleaner, and hardener
artifacts, then exercised the available provider-neutral engine surface for
C-SEQ-001..005, C-SEQ-007, C-SEQ-008 and the §44.7 sequence vector. No
production files or tests were modified by QA. The current hardener chain
values are recorded exactly below.

| hardener field | exact value |
|---|---|
| inputFingerprint | `7b407c298c739a0b006157b0e1a3a835500cfc3efa87327ece7a386f9a3a5e60` |
| outputFingerprint | `06d1e29f3bf779c3f69c2067ce6251da6491a25f787824aa019becac9c440e47` |
| taskFingerprint | `ab72443cf7b377355d104172139b3a0b07decb20252a42b978316e820ed86f2c` |

The current candidate fingerprint is
`06d1e29f3bf779c3f69c2067ce6251da6491a25f787824aa019becac9c440e47`; the
current task fingerprint is
`ab72443cf7b377355d104172139b3a0b07decb20252a42b978316e820ed86f2c`.

## Criteria exercised and observations

- **C-SEQ-001:** `TestSelectionCountsBeforeResponseFailure` observed one
  invocation committed at selection, before the opaque response action is
  consumed by any caller. The focused selection tests also observed one count
  per selected call, including exhausted calls.
- **C-SEQ-002:** The engine keeps the count after selecting an opaque action;
  `TestSelectionCountsBeforeResponseFailure` observed count `1`. There is no
  provider/compatibility response-generation failure hook in this engine
  package, so a later HTTP/provider failure could not be injected here.
- **C-SEQ-003:** The §44.7-shaped two-element sequence returned the first action
  at internal position `0` and the second at position `1`; after those calls
  the reported position was `2`. An unmatched selection did not advance the
  sequence.
- **C-SEQ-004:** The third selected call returned
  `SequenceExhausted=true`, retained invocation count `3`, and left the
  sequence position at `2`. The engine package does not provide the HTTP
  compatibility mapping or interaction verification accumulator, so HTTP `409`
  and the verification failure were not independently exercised here.
- **C-SEQ-005:** Distinct first/second actions were not repeated. An explicitly
  empty sequence produced nil actions and exhaustion on both first and second
  selections, with counts `1` and `2` and position `0` throughout.
- **C-SEQ-007:** Exactly-only, lower-only, upper-only, and bounded ranges were
  accepted. Empty, mixed exactly/range, and reversed ranges were rejected.
- **C-SEQ-008:** Exact, lower-bound, upper-bound, and bounded-range evaluation
  was exercised. Unsatisfied results used `expect_exactly`, `expect_at_least`,
  and `expect_at_most`; failures were emitted in registration order.

The §44.7 vector's two configured actions, exhausted third call, and invocation
count `3` were exercised by `TestSequenceConsumptionAndExhaustionGoldenVector`
(the test uses opaque actions `"yes"` and `"no"`; wire-level answer encoding is
outside this package).

## Commands and results

- `go test ./internal/engine -run 'Test(SequenceConsumptionAndExhaustionGoldenVector|EmptySequenceExhaustsWithoutRepeating|SelectionCountsBeforeResponseFailure|UnmatchedSelectionDoesNotAdvanceSequence|InvocationExpectation|ConcurrentSelectionSerializesState)' -count=1 -v` — passed; all focused sequence, empty-sequence, count, expectation, ordering, and concurrent-selection tests passed.
- `go test ./...` — passed.
- `go test -race ./...` — passed.
- `go vet ./...` — passed with no output.
- `git diff --check` — passed with no output.
- `./scripts/candidate-fingerprint candidate .` — `06d1e29f3bf779c3f69c2067ce6251da6491a25f787824aa019becac9c440e47`.
- `./scripts/candidate-fingerprint task .agent/work/FJ-012/state.json` — `{"class":"product","derivedRequiredStages":["specifier","coder","cleaner","hardener","qa"],"due":["specifier","coder","cleaner","hardener","qa"],"pending":[],"requiredStages":["specifier","coder","cleaner","hardener","qa"],"source":"derived","stage":null,"status":"implementing","taskFingerprint":"ab72443cf7b377355d104172139b3a0b07decb20252a42b978316e820ed86f2c"}`.
- `./scripts/verify-candidate FJ-012` — completed with `48 passed, 0 failed, 3 skipped, 0 not_applicable`; repository Go test, vet, build, architecture, lint, trace, bounded fuzz, and race checks passed. Cleaner complexity, hardener mutation, and QA public-surface tooling were skipped because those tools are not implemented.

## Limits

This is an engine-only QA exercise. HTTP status/body mapping for
`fake_jev_sequence_exhausted`, interaction-derived verification failure
aggregation, configuration decoding, and reset lifecycle behavior are not
implemented at this seam and were not exercised. The verifier's own report is
the source for any repository bootstrap or tooling skips.

No acceptance or stage verdict is recorded here.
