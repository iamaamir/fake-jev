---
stage: specifier
task: FJ-013
inputFingerprint: 06d1e29f3bf779c3f69c2067ce6251da6491a25f787824aa019becac9c440e47
outputFingerprint: 06d1e29f3bf779c3f69c2067ce6251da6491a25f787824aa019becac9c440e47
taskFingerprint: 7d8bd46b210c54295204609b3375cf8457dbdd048d367c0cf830120a6e85ad8d
gitHead: 734ff6f
generatedAt: 2026-09-29T05:50:18Z
author: subagent/worker-FJ-013-specifier
---

# Specifier — FJ-013

## Observable criteria

- Each admitted data-plane interaction receives the next monotonically increasing sequence number, beginning at `1`, before route selection, profile decoding, validation, or stub selection. Unknown routes, malformed JSON, and profile validation failures follow the same admission rule. A request rejected because the bounded journal is already full receives no sequence number.
- The complete admitted-request transition is serialized against other data-plane transitions and engine state mutations: assign sequence, route/decode/validate, select a matching stub when valid, increment its invocation count, consume any applicable response-sequence position, and produce and record the outcome. A concurrent request may be ordered either way only at the point it reaches the engine; once admitted, its mutable state transition is indivisible and race-free.
- Journal entries are retained in sequence order up to the configured interaction capacity and contain the §8.6 minimum information: sequence number, profile, operation, diagnostic-only received timestamp, normalized request, raw body when available, matched stub or unmatched marker, response status/outcome, and applicable sequence position before/after. Timestamp data does not affect matching or response generation, and older interactions are not silently evicted.
- A request racing with reset observes one complete serialized ordering: either its transition completes before reset, or reset completes before the request sees reset state. No partially reset combination of journal, sequence, invocation, response-sequence, dynamic-stub, or failure state is observable.
- When capacity is exhausted, the engine exposes the journal-full outcome without assigning a sequence, executing a stub, or mutating invocation or response-sequence state. The host can map that outcome to the specified `507 Insufficient Storage` / `fake_jev_journal_full` response. The journal-full verification failure is sticky outside the bounded journal, uses `requestSequence: null`, and remains until request history is cleared or a full reset occurs.
- The request-history-clear primitive atomically clears retained interactions and interaction-derived verification failures, including the unrepresentable journal-full failure, and resets the next interaction sequence to `1`. It preserves stub invocation counts and response-sequence positions.
- The full-reset primitive is one atomic state transition: it removes dynamic stubs, zeros static invocation counters, resets static response sequences to element zero, clears interactions and interaction-derived failures, resets the next interaction sequence to `1`, and resets the next dynamic registration index to the first index after the static stubs.
- Engine state and its transition primitives remain race-free under concurrent requests and control mutations; verification with the race detector must not expose data races or nondeterministic partially applied state.

## Engine-only boundary

The implementation target is the provider-neutral engine journal and serialized state transition boundary. It records normalized engine exchanges and generic stub/sequence state; it does not own HTTP request/response translation, status mapping, routing framework behavior, or control endpoint paths. Host and control layers may adapt engine outcomes and invoke the clear/reset primitives, while the engine remains independent of `net/http`, CLI, control API, and `compat/jev` concerns.

## Traces

- C-HOST-001 — §17.4 requires an atomic next-interaction assignment before route/profile decoding, validation, or stub selection for every request admitted within journal capacity; §11.3 defines sequence numbering beginning at `1`.
- C-HOST-002 — §17.4 requires the assign → route/decode/validate → select → mutate → consume → produce/record transition to be serialized against other data-plane requests and control mutations.
- C-HOST-003 — §17.4 permits only request-then-reset or reset-then-request orderings and prohibits partially reset state; §41.10 requires full reset to be one atomic control operation.
- C-HOST-005 — §§19.4 and 43.5 require journal exhaustion to reject the request without a sequence or stub mutation and retain a sticky `journal_full` failure with `requestSequence: null` until history clear or full reset.
- C-HOST-009 — §§17.4 and 32 require race-free engine/control state and concurrent-safety coverage; serialized transitions and atomic reset/clear are the observable boundary.

## Explicit non-goals

- No HTTP status mapping or HTTP host implementation; the engine only exposes generic outcomes for host adaptation.
- No control API routes, wire schemas, request decoding, or endpoint response bodies.
- No provider-specific Jev matching, routing, compatibility behavior, model/provider calls, or network access.
- No journal eviction, unbounded history, timestamp-based matching, or timestamp-dependent response generation.
- No new verification semantics beyond the specified interaction-derived and sticky journal-full failure state.

This artifact records observable behavior and traceability only; it does not declare an implementation, gate, or acceptance verdict.
