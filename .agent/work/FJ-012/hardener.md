---
stage: hardener
task: FJ-012
inputFingerprint: 7b407c298c739a0b006157b0e1a3a835500cfc3efa87327ece7a386f9a3a5e60
outputFingerprint: 06d1e29f3bf779c3f69c2067ce6251da6491a25f787824aa019becac9c440e47
taskFingerprint: ab72443cf7b377355d104172139b3a0b07decb20252a42b978316e820ed86f2c
gitHead: f866a06
generatedAt: 2026-09-28T21:02:14Z
author: subagent/worker-FJ-012-hardener
---

# Hardener — FJ-012

## Chain and scope

The refreshed chain is recorded exactly as follows:

| Stage | Input fingerprint | Output fingerprint | Task fingerprint |
|---|---|---|---|
| specifier | 51091883561b6bf71f2238f41dca36138f47df3a1ceeddd451d63c4565c7dbb1 | 51091883561b6bf71f2238f41dca36138f47df3a1ceeddd451d63c4565c7dbb1 | ab72443cf7b377355d104172139b3a0b07decb20252a42b978316e820ed86f2c |
| coder | 51091883561b6bf71f2238f41dca36138f47df3a1ceeddd451d63c4565c7dbb1 | 51091883561b6bf71f2238f41dca36138f47df3a1ceeddd451d63c4565c7dbb1 | ab72443cf7b377355d104172139b3a0b07decb20252a42b978316e820ed86f2c |
| cleaner | 51091883561b6bf71f2238f41dca36138f47df3a1ceeddd451d63c4565c7dbb1 | 7b407c298c739a0b006157b0e1a3a835500cfc3efa87327ece7a386f9a3a5e60 | ab72443cf7b377355d104172139b3a0b07decb20252a42b978316e820ed86f2c |
| hardener | 7b407c298c739a0b006157b0e1a3a835500cfc3efa87327ece7a386f9a3a5e60 | 06d1e29f3bf779c3f69c2067ce6251da6491a25f787824aa019becac9c440e47 | ab72443cf7b377355d104172139b3a0b07decb20252a42b978316e820ed86f2c |

The audit stayed within the amended engine scope. No compatibility, HTTP,
provider, journal, or reset implementation was added.

## Hardening actions and contract evidence

- `cloneStub` now preserves the distinction between a nil sequence and a
  configured empty sequence. The previous nil-backed append converted `[]` to
  `nil`, so an explicitly empty sequence could be treated as a regular response
  instead of exhausting on its first selected invocation. The replacement
  allocates a length-preserving copy. This serves §40.7 and C-SEQ-003–005.
- Selection remains integrated through `Registry.SelectInvocation`: matching,
  invocation increment, sequence transition, and exhaustion marking occur under
  the registry mutex. The increment precedes action selection and remains
  observable for exhausted calls, serving §40.6, §40.7, and C-SEQ-001–005.
- Matcher profile filtering, priority descending, registration-index tie
  breaking, and detached candidate data were audited without changing their
  total order. Unmatched exchanges return before mutation, serving the FJ-011
  seam and deterministic selection behavior.
- Expectation validation rejects empty, mixed exact/range, and reversed ranges;
  evaluation scans registration order and emits the specified exact, lower, or
  upper-bound failure code. Registered expectation pointers are copied, serving
  §12.7, §41.9, and C-SEQ-007–008.
- Added focused tests for empty-sequence exhaustion and non-repetition,
  concurrent selection state serialization, invalid expectation registration
  without index consumption, and nil engine/registry safety. Existing golden
  vector, exhaustion-count, unmatched-selection, expectation, matcher, and
  registration tests remain in the same allowed test file set.

## Validation evidence

- `gofmt -w internal/engine/matcher.go internal/engine/sequence_test.go`
- `go test ./...`
- `go test -race ./...`
- `go vet ./...`
- `./scripts/verify-candidate FJ-012` (repository checks, Go tests/build/vet,
  architecture, lint, trace, bounded fuzz, and race checks were exercised)
- Candidate fingerprint after the allowed-file hardening is
  `06d1e29f3bf779c3f69c2067ce6251da6491a25f787824aa019becac9c440e47`.

## Residual risks

- The provider-neutral engine returns `SequenceExhausted`; HTTP 409,
  `fake_jev_sequence_exhausted`, interaction journaling, and verification
  aggregation belong to the later compatibility/control boundary.
- `ResponseAction` remains intentionally opaque and is shallow-copied as an
  interface value; callers must not mutate referenced action data concurrently
  with selection.
- Resetting static counters and sequence positions is outside this task and
  remains assigned to the reset lifecycle work.
- Mutation/hardening tooling is still under the repository bootstrap exemption;
  the evidence here is the focused deterministic tests and Go toolchain checks.
