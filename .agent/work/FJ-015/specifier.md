---
stage: specifier
task: FJ-015
inputFingerprint: e4dcbb3aa2b0fc4eb88b39ce09fd65c349311014a1ba7afa29d2a26371c2fba7
outputFingerprint: e4dcbb3aa2b0fc4eb88b39ce09fd65c349311014a1ba7afa29d2a26371c2fba7
taskFingerprint: c4409d7a1fbba4b732c5b1d9cc70c922f2a2f717e6b47adce19907a99d646528
gitHead: 45156682a651b5d6278239e21266d0bd36efce30
generatedAt: 2026-09-29T07:29:08Z
author: subagent/worker-FJ-015-specifier
---

# Specifier — FJ-015

## Observable criteria

- A matched strict `jev/v1` `answers` fixture must have an answer key for every request question name and no key for any absent request question name. Missing or extra names, including a partial map in a mixed noul/choice/score request, are an invalid stub response (`fake_jev_invalid_stub_response`, HTTP 500) and do not trigger implicit answer generation or partial fallback.
- A `noul` helper emits `{ "type": "noul", "noul": value }`. Numeric values and all confidence/probability values are finite and in the inclusive range `[0,1]`; the boolean convenience form converts exactly `true` to `1.0` and `false` to `0.0`.
- A `noul` helper is legal only for a request question whose type is exactly `noul`. A helper/request type mismatch is a fake-server error; no question type is coerced.
- A `choice` helper's selected choice must be a key in the request criteria object. With omitted probabilities it emits a one-hot map covering every request criteria key exactly: the selected key is `1.0` and every other key is `0.0`; omitted confidence is `1.0`.
- Explicit choice probabilities must have exactly the request criteria key set, finite values in `[0,1]`, and a sum within `1e-6` of `1.0`. The selected choice must have a maximum probability (ties are permitted). If explicit probabilities are supplied without confidence, confidence equals the selected choice probability. Any unknown/missing key, out-of-range/non-finite value, invalid sum, unknown selected choice, or non-maximum selected choice makes response generation invalid rather than being normalized or repaired.
- A `score` helper's score is within `[0,N-1]` for a request criteria array of length `N`. Its response legend is always derived by indexing the request criteria (`"0"` through `"N-1"`) and cannot be supplied by the fixture. An integer score emits a one-hot probability map at that level; an omitted-probability fractional score `s` emits deterministic linear interpolation with `lo=floor(s)`, `hi=ceil(s)`, `p(lo)=hi-s`, `p(hi)=s-lo`, and zero elsewhere. Omitted confidence is `1.0`.
- Explicit score probabilities must use exactly the keys `"0"` through `"N-1"`, have finite values in `[0,1]`, sum within `1e-6` of `1.0`, and have weighted expected value within `1e-6` of the configured score. A key-set, range, sum, or expected-value mismatch is an invalid stub response; probabilities are not silently adjusted. Explicit confidence, when present, remains a probability/confidence value subject to the finite `[0,1]` constraint.
- The §44 golden vectors remain observable: the noul vector emits `0.9` with model echo and zero usage; choice one-hot emits backend `1.0`, all criteria keys, and confidence `1.0`; fractional score `1.5` over three criteria emits the criteria-derived legend and probabilities `{ "0": 0.0, "1": 0.5, "2": 0.5 }` with confidence `1.0`.

## Fixture-only boundary

This slice is limited to deterministic `jev/v1` fixture-helper answer generation after a request has been validated and a fixture selected. Jev-specific answer shapes, type matching, criteria-key/level interpretation, legend derivation, and probability checks remain in the compatibility profile. The provider-neutral engine does not gain knowledge of `noul`, `choice`, `score`, criteria, legends, or probability semantics. Fixtures remain data-only and answer generation performs no model/provider call or network access.

## Traces

- C-JEV-008 — §§13.4–13.5 require exact answer-map coverage and prohibit partial answers; §43.4 defines the invalid-stub-response HTTP 500 outcome.
- C-JEV-011 — §13.1 requires exact boolean conversion for noul and finite probability/confidence values in `[0,1]`.
- C-JEV-012 — §13.2 and §44.3 define complete choice one-hot generation and the default confidence.
- C-JEV-013 — §13.2 defines explicit choice key-set, range, sum tolerance, selected-maximum, and omitted-confidence behavior.
- C-JEV-014 — §13.3 and §44.4 define score one-hot/interpolation behavior and request-derived legends.
- C-JEV-015 — §13.3 defines explicit score key-set, range, sum tolerance, and expected-value tolerance.
- C-JEV-016 — §§13.1 and 13.4 require exact helper/request type matching and prohibit coercion.
- C-GOLD-002 — §44.2 is the noul answer vector.
- C-GOLD-003 — §44.3 is the choice one-hot vector.
- C-GOLD-004 — §44.4 is the fractional score vector.

## Explicit non-goals

- No legend supplied by fixtures; score legends are derived only from request criteria.
- No type coercion between helper and request question types, and no conversion of malformed values into valid answer types.
- No partial-answer fallback, implicit answer generation, or silent reuse of answers for missing or extra question names.
- No changes to request routing/validation, stub matching or selection, response sequences, model/usage defaults, control APIs, journal/verification state, other profiles, or the provider-neutral engine.
- No probabilistic/random/fuzz generation, model/provider invocation, network access, generic mock behavior, or fixture code execution.

This artifact records observable behavior and traceability only; it does not declare an implementation or acceptance verdict.
