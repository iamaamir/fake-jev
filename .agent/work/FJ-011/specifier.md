---
stage: specifier
task: FJ-011
inputFingerprint: dfbe154ac2a066ce29e4e356e521685ab5e42f972a5bda3b1f1c64cf1e7d7775
outputFingerprint: dfbe154ac2a066ce29e4e356e521685ab5e42f972a5bda3b1f1c64cf1e7d7775
taskFingerprint: b788644f51a266d22e3a0d6efc84fe854e22b1fba53fbc99cc44c97f32ab94bb
gitHead: 4d6dff0
generatedAt: 2026-09-28T19:33:29Z
author: subagent/worker-FJ-011-specifier
---

# Specifier — FJ-011

## Observable criteria

- The engine represents a stub as a provider-neutral deterministic rule with a matcher, response action, priority, and registration index. Static stubs receive registration order from configuration; dynamic stubs receive registration order from successful registration order. Matching and ordering do not depend on source location beyond that explicit index.
- Candidate selection considers only stubs whose profile equals the active profile. Every matcher field present in `when` is required, and all present fields combine with logical AND; each omitted field is a wildcard.
- A present `when.model` compares the request model by exact string equality only. No aliases or regular expressions participate.
- A present `when.questions` requires exact equality of the question-name sets and exact equality of each configured question type to the corresponding request type. Extra or missing questions prevent a match. Request instructions and criteria contents are ignored by matching.
- A present `when.state` compares JSON values recursively: strings, booleans, null, arrays, and objects use JSON-value semantics; object key order is irrelevant; numbers compare by numeric value, including `1`, `1.0`, and `1e0`, without a binary-floating-point-only conversion that loses integer precision.
- Among matching candidates, selection is a total order: larger priority first, then smaller registration index. Equal-priority matches never use specificity scoring. The first stub in that order is selected.
- A later equal-priority dynamic stub does not displace an earlier static stub. A dynamic stub overrides that static stub only when it has a greater priority.

## Provider-neutral engine boundary

The engine operates on normalized, generic exchanges, values, stubs, matchers, response actions, and selection state. Compatibility code compiles provider-specific configuration and wire requests into those primitives; it owns provider routes, schemas, question primitives, and response shapes. The engine therefore contains no Jev primitive or route knowledge and does not require a compile-time `choice | noul | score` union.

The `internal/engine` boundary excludes `net/http`, `os/exec`, CLI packages, control API packages, and `compat/jev`; configuration is loaded and compiled outside the engine. Matching is deterministic and provider-neutral, with no provider or model call required for normal operation.

## Traces

- C-MATCH-001 — §40.2 requires larger priority first; C-GOLD-006 (§44.6) exercises priority selection.
- C-MATCH-002 — §40.2 requires smaller registration index for equal priority and forbids specificity scoring; C-GOLD-006 (§44.6) exercises the tie order and no-specificity rule.
- C-MATCH-003 — §40.9 defines static configuration order, dynamic successful-registration order, and the greater-priority override rule; C-GOLD-012 (§44.12) exercises equal-priority preservation and greater-priority override.
- C-MATCH-004 — §40.1 limits candidates to stubs whose profile equals the active profile.
- C-MATCH-005 — §40.1 requires logical AND across present `when` fields and wildcard semantics for omitted fields; C-GOLD-005 (§44.5) exercises strict matcher-field behavior through the exact question-set case.
- C-MATCH-006 — §40.5 requires exact question-name sets and per-name types; C-GOLD-005 (§44.5) exercises rejection of an extra question.
- C-MATCH-007 — §40.4 requires exact model equality and excludes aliases and regular expressions.
- C-MATCH-008 — §40.3 defines JSON-value equality and precision-preserving numeric comparison.
- C-MATCH-009 — §40.5 explicitly excludes instructions and criteria contents from matching; C-GOLD-005 (§44.5) exercises question matching without reusing a stub for a different request shape.
- C-ARCH-001 — §4.2 and §30.1 require a provider-neutral engine with no `choice`, `noul`, `score`, route, or schema knowledge; the provider-specific behavior remains in a compatibility profile.
- C-HOST-010 — §7.1 and §17.3 prohibit HTTP, process, CLI, control, and Jev compatibility imports from engine packages and require configuration compilation outside the engine.
- C-GOLD-005 — §44.5 exact question-set vector: an extra request question must not match a matcher configured for only `route`.
- C-GOLD-006 — §44.6 priority-then-registration vector: the higher-priority stub wins, and the earlier equal-priority registration wins without specificity scoring.
- C-GOLD-012 — §44.12 dynamic-registration vector: an equal-priority dynamic stub does not implicitly override static, while a greater-priority dynamic stub does.

## Explicit non-goals

- No response sequences.
- No interaction journal.
- No Jev or HTTP knowledge in the engine.
- No specificity scoring.

This artifact defines observable behavior and traceability only; it does not declare an implementation, gate, or acceptance verdict.
