---
stage: cleaner
task: FJ-012
inputFingerprint: 51091883561b6bf71f2238f41dca36138f47df3a1ceeddd451d63c4565c7dbb1
outputFingerprint: 7b407c298c739a0b006157b0e1a3a835500cfc3efa87327ece7a386f9a3a5e60
taskFingerprint: ab72443cf7b377355d104172139b3a0b07decb20252a42b978316e820ed86f2c
gitHead: f866a06
generatedAt: 2026-09-28T20:53:03Z
author: subagent/worker-FJ-012-cleaner
---

# Cleaner — FJ-012

## Structure reviewed

Reviewed the refreshed specifier and coder artifacts, the amended allowed engine
files, the FJ-011 selection seam, and the provider-neutral boundary against
§17, §40.6, §40.7, §12.7, and §44.7. The only justified cleanup was structural:

- Added the private `Selection.asStub` adapter in `internal/engine/sequence.go`
and used it from both `Registry.Select` and `Engine.Select`. The legacy stub
return path now has one implementation for copying the selected action into
`Response`; `SelectInvocation` remains the explicit path for exhaustion state.
- Simplified the registry transition in `selectInvocation`: invocation count is
still committed under the registry lock before action selection, while the
selection result is cloned once after sequence state is finalized. This removes
the pre-transition clone and branch-only replacement without changing ownership
or mutation timing.

No matcher predicate, priority/registration ordering, registry locking boundary,
public response action type, expectation rule, or test interface was changed.
No compatibility, HTTP, provider, or reset behavior was introduced.

## Behavior evidence

- `Registry.SelectInvocation` remains the sole state transition: matching is
performed against the registry, selected state is mutated there, and unmatched
exchanges return without changing counters or sequence positions.
- Selection count, zero-based sequence consumption, exhausted-call counting, and
expectation evaluation remain covered by the existing sequence tests.
- `Registry.Select` and `Engine.Select` now share the same selected-action
adaptation, preserving deterministic legacy results while leaving exhaustion
distinguishable through `Selection.SequenceExhausted`.
- Existing matcher tests continue to exercise profile filtering, total ordering,
registration ownership, and detached returned stubs.

## Validation evidence

- `gofmt -w internal/engine/sequence.go internal/engine/stub.go internal/engine/engine.go internal/engine/matcher.go internal/engine/sequence_test.go` — completed.
- `go test ./...` — passed.
- `go test -race ./...` — passed.
- `go vet ./...` — passed.
- `./scripts/verify-candidate FJ-012` — repository, product Go, build, architecture,
lint, trace, fuzz, and race checks passed. The report also records the expected
bootstrap skips for absent cleaner/hardener/QA tooling. The pre-existing
specifier/coder artifacts declare a 63-character candidate fingerprint, so their
header checks remain invalid; hardener and QA artifacts are not present.

The candidate fingerprint after cleanup is
`7b407c298c739a0b006157b0e1a3a835500cfc3efa87327ece7a386f9a3a5e60`; the task
fingerprint is
`ab72443cf7b377355d104172139b3a0b07decb20252a42b978316e820ed86f2c`.
