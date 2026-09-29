---
stage: specifier
task: FJ-014
inputFingerprint: a63a3d6fedef600dcdcdb8aed573eef907f8fea92e52605686e2133a0dea1b4c
outputFingerprint: a63a3d6fedef600dcdcdb8aed573eef907f8fea92e52605686e2133a0dea1b4c
taskFingerprint: 1df19a608667d96667c2059da5f0a35005334f81ca89b7ab6c1307c5440d2021
gitHead: 2944771
generatedAt: 2026-09-29T06:37:44Z
author: subagent/worker-FJ-014-specifier
---

# Specifier — FJ-014

## Observable criteria

- The Jev data-plane recognizes exactly `GET /v1/models` and `POST /v1/systemone`. Query strings are ignored for route matching, so `GET /v1/models?x=1` is handled as `/v1/models`; trailing-slash variants and a wrong method for a known path are not aliases and return the unknown-data-plane-route `404` response. Unknown-route requests remain data-plane interactions rather than being redirected or treated as a control request.
- A `POST /v1/systemone` body must be valid JSON whose top-level value is an object. Malformed JSON returns HTTP `422` with exactly one detail item at `loc: ["body"]`, message `Invalid JSON`, and type `json_invalid`.
- Schema validation checks only the frozen public schema and returns only the first discovered error. Its order is `state`, then `model`, then `questions`, then question entries by lexicographic question name. Unknown object fields are accepted; validation does not emulate additional upstream framework behavior.
- `state` is required and accepts a JSON string, object, or array. `model` is required and must be a JSON string; it is not required to appear in `/v1/models`. `questions` is required, must be an object, and must contain at least one property. A question value must be an object whose string `type` is one of `noul`, `choice`, or `score`.
- A `noul` question may omit `criteria`; when present, its criteria is null or an object, and known boolean criteria values may be string, object, array, or null. `instructions`, when present for any supported question type, may be string, object, array, or null. A `choice` question requires a non-empty object `criteria`, with string, object, array, or null values. A `score` question requires a non-empty array `criteria`, and each element is string, object, or array.
- A missing required field returns HTTP `422` with exactly one detail item whose location is `body/<field>`, message is `Field required`, and type is `missing`. Other schema violations return HTTP `422` with the deterministic detail shape: location at the invalid field, message `Invalid value`, and type `value_error`.
- An unsupported question type is reported as the first applicable validation error at `body/questions/<name>/type`, with message `Unsupported question type` and type `literal_error`. The response uses the same deterministic single-item `detail` envelope and HTTP `422`; later question errors are not examined.
- The malformed, missing-state, and exact-route golden vectors preserve these wire outcomes: `{` yields the §44.10 malformed envelope, a request containing `model` and a valid `x: {"type":"noul"}` question but no `state` yields the §44.11 missing envelope, and the four route/method examples in §44.16 produce the specified handled-versus-404 results.

## Compat-only boundary

The route recognition, Jev request schema, supported question types, validation order, and 422 wire envelope are owned by the `jev/v1` compatibility profile. The provider-neutral engine does not acquire Jev route or schema knowledge. The profile uses the frozen fake contract and does not attempt exact live-upstream validation prose or framework behavior.

## Traces

- C-JEV-003 — §39.2 freezes `state`, `model`, `questions`, then lexicographic question-name validation and first-error-only behavior.
- C-JEV-004 — §§39.3 and 44.10 define the exact malformed-JSON 422 `json_invalid` detail envelope.
- C-JEV-005 — §§39.3 and 44.11 define the exact missing-field 422 `missing` detail envelope and location.
- C-JEV-006 — §39.3 defines the unsupported-type `literal_error`, message, and `body/questions/<name>/type` location.
- C-JEV-007 — §39.2 defines the supported `noul`, `choice`, and `score` question criteria requirements and non-empty choice/score criteria.
- C-JEV-019 — §§39.8 and 44.16 define exact path/method recognition, query-string handling, trailing-slash rejection, and wrong-method unknown routes.
- C-GOLD-010 — §44.10 is the malformed request vector and its validation-failure consequence.
- C-GOLD-011 — §44.11 is the missing-state vector and exact envelope.
- C-GOLD-016 — §44.16 is the exact-routes vector covering query, trailing slash, and wrong methods.

## Explicit non-goals

- No answer generation, response construction, or stub selection/matching; those are outside this validation and route-recognition slice.
- No model/provider calls, network access, or provider-specific behavior beyond the `jev/v1` compatibility profile.
- No changes to the provider-neutral engine, control-plane routes, control API schemas, journal/state semantics, or unrelated compatibility profiles.
- No exact upstream framework validation prose emulation, automatic slash redirects, model-list membership validation, or rejection of unknown request fields.
- No new question types, criteria semantics, validation rules, or public error shapes beyond §§39.2, 39.3, and 39.8 and the traced golden vectors.
