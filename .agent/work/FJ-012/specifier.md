---
stage: specifier
task: FJ-012
inputFingerprint: 51091883561b6bf71f2238f41dca36138f47df3a1ceeddd451d63c4565c7dbb1
outputFingerprint: 51091883561b6bf71f2238f41dca36138f47df3a1ceeddd451d63c4565c7dbb1
taskFingerprint: ab72443cf7b377355d104172139b3a0b07decb20252a42b978316e820ed86f2c
gitHead: f866a06
generatedAt: 2026-09-28T20:54:49Z
author: subagent/worker-FJ-012-specifier
---

# Specifier — FJ-012

## Observable criteria

- After the FJ-011 matcher selects a stub, its invocation count increments exactly once before any response or sequence element is selected. A later response-generation failure does not undo that increment.
- Each configured response sequence has an internal zero-based next position. The first selected invocation consumes element `0`, the next consumes element `1`, and each selected invocation consumes at most one sequence element; sequence state is not advanced by an unmatched request.
- Once the final sequence element has been consumed, the next selected invocation returns HTTP `409 Conflict` with the `fake_jev_sequence_exhausted` outcome, including the selected stub identity as specified by §43.3. That exhaustion is a verification failure, and the invocation count still includes the exhausted invocation. The last configured response is never implicitly repeated.
- `expect` is optional. A valid expectation is either `exactly` alone, or a range form containing at least one of `atLeast` and `atMost`; all supplied counts are non-negative integers, `exactly` cannot coexist with either range bound, and when both range bounds are supplied `atLeast <= atMost`. No expectation means no invocation-count assertion.
- Verification evaluates each stub expectation against its invocation count. An unsatisfied exact expectation contributes an `expect_exactly` failure; an unsatisfied lower or upper range bound contributes `expect_at_least` or `expect_at_most`, respectively. Expectation failures are reported after interaction-derived failures and in stub registration-index order, using the §41.9 failure shape.

## Integration boundary with FJ-011

FJ-011 owns provider-neutral candidate matching and total-order selection: profile filtering, matcher evaluation, priority, and registration-index tie breaking. FJ-012 begins only with the selected stub and owns post-selection invocation counting, sequence consumption/exhaustion, and expectation evaluation. It does not alter matching, registration order, specificity behavior, or Jev request/response knowledge. The sequence and counter state remains provider-neutral; the compatibility layer supplies the response actions and wire mapping at its existing boundary.

## Allowed scope

- `internal/engine/sequence.go`
- `internal/engine/stub.go`
- `internal/engine/engine.go`
- `internal/engine/matcher.go`
- `internal/engine/sequence_test.go`

Sequence consumption is committed at `Registry.Select` selection time, so matcher integration is within scope for this objective.

## Traces

- C-SEQ-001 — §40.6: once a stub is selected, increment its invocation count exactly once before response selection.
- C-SEQ-002 — §40.6: retain that increment when response generation later fails.
- C-SEQ-003 — §40.7: maintain a zero-based internal next position; selected calls consume elements `0`, `1`, and onward exactly once per selected invocation.
- C-SEQ-004 — §40.7 requires exhaustion after the final element; §43.3 maps it to HTTP `409` and `fake_jev_sequence_exhausted`; §44.7 requires verification failure and a count of `3` after the two-element vector's third selected call.
- C-SEQ-005 — §40.7 expressly forbids implicit repeat-last behavior.
- C-SEQ-007 — §12.7 defines the exact-count and bounded-range forms, non-negative integer counts, mutual exclusion of `exactly` and range bounds, the requirement for at least one range bound, and the ordering rule when both bounds are present.
- C-SEQ-008 — §12.7 defines the expectation assertion and §41.9 defines verification evaluation and the `expect_exactly`, `expect_at_least`, and `expect_at_most` failure codes and ordering.

## Explicit non-goals

- No named scenario or state-machine DSL; §40.8 keeps v1 multi-call workflows to response sequences and invocation expectations.
- No implicit repeat-last response behavior.
- No reset endpoint or reset lifecycle implementation; full reset belongs to FJ-021, despite §40.7 and §41.10 defining its eventual sequence-state effects.
- No changes to FJ-011 matching, candidate ordering, registration semantics, or provider-specific compatibility routing.

This artifact records observable behavior and traceability only; it does not declare an implementation, gate, or acceptance verdict.
