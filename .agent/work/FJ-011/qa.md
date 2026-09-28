---
stage: qa
task: FJ-011
inputFingerprint: 5253a8d6677849329dd077262fd8385b6130864268f26d4d2928c5ebd624953f
outputFingerprint: 5253a8d6677849329dd077262fd8385b6130864268f26d4d2928c5ebd624953f
taskFingerprint: b788644f51a266d22e3a0d6efc84fe854e22b1fba53fbc99cc44c97f32ab94bb
gitHead: 4d6dff04174428fe185635373cd1a8e1520396b7
generatedAt: 2026-09-28T20:01:12Z
author: subagent/worker-FJ-011-qa
---

# QA — FJ-011

## Scope and method

Read the predecessor artifacts, acceptance catalog, and specification §§8,
17.3, and 40. Exercised the complete assigned acceptance set without changing
production files or tests. The checks covered matcher behavior, registry
provenance and index assignment, golden matching/order vectors, provider
neutrality, and the engine import boundary.

## Criteria exercised and observations

- **C-MATCH-001:** The golden ordering vector registers priority 0 and priority
  10 candidates and selects the priority-10 stub.
- **C-MATCH-002:** Equal-priority candidates are returned in registration-index
  order (`[B C A]`); no specificity ordering is used.
- **C-MATCH-003:** The dynamic equal-priority candidate leaves the earlier static
  candidate selected; a dynamic candidate with greater priority is selected.
- **C-MATCH-004:** A matching stub from another profile is excluded even when its
  priority is higher than the active-profile candidates.
- **C-MATCH-005:** Present operation, model, and question fields are combined;
  omitted matcher fields match as wildcards. Mismatch cases reject.
- **C-MATCH-006:** Question matching rejects extra names, missing names, and
  differing types, while accepting the exact name/type set.
- **C-MATCH-007:** Model matching is exact; a different model/alias does not
  match.
- **C-MATCH-008:** State equality exercises null, ordered arrays, key-order-
  independent objects, `1` versus `1.0`/`1e0`, and a large integer beyond
  binary-float exact precision. Cyclic and overflowing-exponent inputs fail
  closed without a panic.
- **C-MATCH-009:** Instructions and criteria placed in the exchange payload do
  not affect a matching question/type set.
- **C-ARCH-001:** The engine surface is generic (`Exchange`, `Matcher`, `Stub`,
  opaque `ResponseAction`, and registry selection); source inspection found no
  provider response primitive, route, or schema implementation. The golden
  tests exercise only generic values and actions.
- **C-HOST-010:** The architecture guard checks `internal/engine` for forbidden
  `net/http`, `os/exec`, CLI, control, and `compat/jev` imports. The engine
  source has no filesystem access or non-standard provider/host dependency.

Golden-vector coverage included the priority/registration ordering behavior
corresponding to §44.6 and the dynamic registration behavior corresponding to
§44.12. The exact question-set behavior corresponding to §44.5 was exercised
alongside wildcard omission and provenance/index checks.

## Commands and results

- `go test ./internal/engine -count=1 -run 'TestMatcherAcceptanceCriteria|TestRegistryRegistrationProvenanceAndIndices|TestRegistryRejectsInvalidAndExhaustedRegistrations|TestRegistryDetachesMatcherInputsAndOutputs|TestMatcherRejectsOverflowingJSONExponent|TestMatcherRejectsCyclicStateWithoutPanic|TestGoldenMatchingAndOrderingVectors' -v` — passed; all focused matcher, provenance, hardening, and golden tests ran.
- `go test ./internal/engine -count=1 -run TestGoldenMatchingAndOrderingVectors -v` — passed.
- `go test ./...` — passed.
- `go test -race ./...` — passed.
- `go vet ./...` — passed.
- `./scripts/verify-candidate FJ-011` — passed with 48 checks passed, 0 failed, 3 skipped, and 0 not applicable; the public-surface/system QA tooling row is skipped because the harness is not implemented.
- `./scripts/candidate-fingerprint candidate .` — `5253a8d6677849329dd077262fd8385b6130864268f26d4d2928c5ebd624953f`.
- `./scripts/candidate-fingerprint task .agent/work/FJ-011/state.json` — `b788644f51a266d22e3a0d6efc84fe854e22b1fba53fbc99cc44c97f32ab94bb`.

## Residual risks and limits

- The repository has no public-surface/system QA harness; the verifier reports
  that tooling row as skipped. This review therefore exercises the available
  engine tests and static architecture guard, not HTTP integration behavior.
- The engine's opaque response action and exchange payload ownership remain
  caller responsibilities; concurrent mutation by callers is outside these
  deterministic matcher probes.
- The verifier's public-surface/system QA tooling row remains skipped because
  that harness is not implemented in the repository.

No acceptance or stage verdict is recorded here.
