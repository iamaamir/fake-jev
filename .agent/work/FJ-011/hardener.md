---
stage: hardener
task: FJ-011
inputFingerprint: d6258777f129a21d33ee48b49d17604f2bc4e4fa260c8321ec8c5adf7301ca39
outputFingerprint: 5253a8d6677849329dd077262fd8385b6130864268f26d4d2928c5ebd624953f
taskFingerprint: b788644f51a266d22e3a0d6efc84fe854e22b1fba53fbc99cc44c97f32ab94bb
gitHead: 4d6dff04174428fe185635373cd1a8e1520396b7
generatedAt: 2026-09-28T19:57:40Z
author: subagent/worker-FJ-011-hardener
---

# Hardener — FJ-011

## Scope and hardening actions

The audit covered the five allowed engine files and the existing matcher tests
against §§8, 17.3, and 40. Changes are limited to invalid-input handling,
registry ownership, registration-index exhaustion, and defensive JSON
normalization; no response sequence, journal, HTTP, compatibility-profile, or
specificity behavior was added.

- `Registry` now rejects empty stub IDs and profiles, preserves one-based
  indices for a zero-value registry, and fails closed once the `uint64`
  registration index reaches its maximum rather than wrapping to zero.
- Registration overwrites caller-supplied provenance and index metadata as
  before, then stores detached matcher questions and state. `Stubs`, `Matching`,
  and `Select` return detached matcher data, so returned values and caller-owned
  matcher inputs cannot mutate registry provenance or matching state.
- Dynamic removal uses a fresh retained slice, avoiding stale references in the
  old backing array while preserving static entries and index monotonicity.
- JSON equality now normalizes JSON-compatible containers through
  `encoding/json` with `UseNumber`, preserving exact numeric comparison while
  rejecting unsupported, malformed, cyclic, or otherwise unmarshalable values
  without a panic. Exponent parsing checks its bound before arithmetic, avoiding
  integer overflow and unbounded exponent amplification.

## Contract evidence

- **§40.1 / C-MATCH-004–005:** profile filtering, AND matching, and wildcard
  omission behavior remain unchanged; defensive copies do not alter matcher
  inputs.
- **§40.2 / C-MATCH-001–003:** candidate sorting remains priority descending,
  then registration index ascending, with no source-specific precedence.
  Index exhaustion cannot create a wrapped index that would violate this total
  order.
- **§40.3 / C-MATCH-008:** JSON object/array/scalar semantics remain recursive,
  object key order remains irrelevant, and `json.Number` plus rational
  comparison avoids binary-float precision loss.
- **§40.4 / C-MATCH-007 and §40.5 / C-MATCH-006, C-MATCH-009:** exact model and
  question-set/type matching remains isolated from payload contents.
- **§8 and §17.3 / C-ARCH-001, C-HOST-010:** `ResponseAction` remains opaque,
  exchanges remain normalized provider-neutral values, and engine imports stay
  within the standard library with no HTTP, CLI, control, process, or profile
  dependencies.

## Hardening evidence

Added focused tests for zero-value registries, empty identity/profile inputs,
maximum-index exhaustion and non-wrapping behavior, matcher input/output
mutation isolation, overflowing JSON exponents, and cyclic JSON state.
Existing provenance,
failed-registration, ordering, profile, wildcard, question, model, payload,
and JSON-equality vectors continue to run.

Commands run:

- `gofmt -w internal/engine/engine.go internal/engine/exchange.go internal/engine/stub.go internal/engine/matcher.go internal/engine/matcher_test.go`
- `go test ./internal/engine`
- `go test ./...`
- `go test -race ./...`
- `go vet ./...`
- focused engine tests covering the new hardening cases and existing vectors
- `./scripts/verify-candidate FJ-011` (repository/product checks, architecture,
  lint, trace, fuzz, and race checks; the only stage artifact not yet present
  during that run was the downstream QA artifact)

## Residual risks

- `Exchange` maps and opaque `ResponseAction` values retain caller ownership by
  design; callers must not mutate them concurrently with selection. The engine
  does not inspect or clone response actions.
- The registry does not expose a reset operation; dynamic-removal index-reset
  semantics belong to the later control/reset lifecycle, outside this item.
- The hardening/mutation analyzer remains under the repository bootstrap
  exemption; evidence here is manual review plus deterministic Go toolchain
  checks.
