---
stage: hardener
task: FJ-013
inputFingerprint: 2e72044c97a6734dfe0811e350ac33184312405c867061432cefc8b2e9502d7e
outputFingerprint: a63a3d6fedef600dcdcdb8aed573eef907f8fea92e52605686e2133a0dea1b4c
taskFingerprint: 7d8bd46b210c54295204609b3375cf8457dbdd048d367c0cf830120a6e85ad8d
gitHead: 734ff6f
generatedAt: 2026-09-29T06:10:06Z
author: subagent/worker-FJ-013-hardener
---

# Hardener — FJ-013

## Chain and scope

The current stage chain and fingerprints are:

| Stage | Input fingerprint | Output fingerprint | Task fingerprint |
|---|---|---|---|
| specifier | 06d1e29f3bf779c3f69c2067ce6251da6491a25f787824aa019becac9c440e47 | 06d1e29f3bf779c3f69c2067ce6251da6491a25f787824aa019becac9c440e47 | 7d8bd46b210c54295204609b3375cf8457dbdd048d367c0cf830120a6e85ad8d |
| coder | 06d1e29f3bf779c3f69c2067ce6251da6491a25f787824aa019becac9c440e47 | d84f77ff9b4f9d3bf8278a18d9e1f37b4e5a59b4ba11e40617ea8e65fe0cdb3e | 7d8bd46b210c54295204609b3375cf8457dbdd048d367c0cf830120a6e85ad8d |
| cleaner | d84f77ff9b4f9d3bf8278a18d9e1f37b4e5a59b4ba11e40617ea8e65fe0cdb3e | 2e72044c97a6734dfe0811e350ac33184312405c867061432cefc8b2e9502d7e | 7d8bd46b210c54295204609b3375cf8457dbdd048d367c0cf830120a6e85ad8d |
| hardener | 2e72044c97a6734dfe0811e350ac33184312405c867061432cefc8b2e9502d7e | a63a3d6fedef600dcdcdb8aed573eef907f8fea92e52605686e2133a0dea1b4c | 7d8bd46b210c54295204609b3375cf8457dbdd048d367c0cf830120a6e85ad8d |

The audit and edits stayed within `internal/engine/engine.go`,
`internal/engine/journal.go`, and `internal/engine/journal_test.go`.

## Hardening actions and contract evidence

- Nil transition callbacks and manually malformed engines with no journal now
  return errors without admitting a request or panicking. This protects the
  serialized transition boundary and invalid-input path in §17.4 and serves
  C-HOST-001, C-HOST-002, and C-HOST-009.
- Interaction sequence reservation now uses zero as an exhausted sentinel after
  issuing `math.MaxUint64`; it cannot wrap and issue a duplicate sequence.
  Subsequent admission is rejected before the callback or registry mutation.
  This strengthens monotonic assignment in §17.4 and the journal sequencing
  requirement in §8.6.
- Journal internals now fail closed for nil receivers and bounded append paths,
  while clear and snapshot operations remain detached and safe. Existing
  journal-full handling remains sticky, retains no out-of-capacity entry, and
  is cleared only by history clear or full reset, serving §§19.4 and 43.5 and
  C-HOST-005.
- Added focused tests for nil-handler non-admission and sequence-overflow
  non-wrapping. Existing tests continue to cover serialized transitions,
  concurrent reset, bounded capacity, sticky failures, detached request data,
  and reset/clear state preservation.

## Validation evidence

- `gofmt -w internal/engine/engine.go internal/engine/journal.go internal/engine/journal_test.go`
- `go test ./...`
- `go test -race ./...`
- `go vet ./...`
- `./scripts/verify-candidate FJ-013` exercised repository checks, product Go
  tests/build/vet, architecture, lint, trace, bounded fuzz, and race checks;
  all executed product checks completed, with only the downstream QA artifact
  still absent from the stage chain.
- Candidate fingerprint after the allowed-file hardening:
  `a63a3d6fedef600dcdcdb8aed573eef907f8fea92e52605686e2133a0dea1b4c`.

## Residual risks

- The engine cannot prevent callers from mutating the public `Registry` field
  or opaque `ResponseAction` values directly; those ownership boundaries remain
  caller responsibilities.
- Sequence exhaustion is an in-memory `uint64` boundary condition and has no
  HTTP mapping in this engine-only task; the host must continue to map normal
  journal admission errors according to its own contract.
- Mutation/hardening tooling is still under the repository bootstrap exemption;
  deterministic unit tests and Go toolchain checks provide the available
  hardening evidence.
