---
stage: specifier
task: FJ-016
inputFingerprint: 66f35765afb2948730b0bac0c54bc1b08852900e356f16423d15d9ad2690710e
outputFingerprint: 66f35765afb2948730b0bac0c54bc1b08852900e356f16423d15d9ad2690710e
taskFingerprint: fe3ad8fed06766d931689414827eac15a4dfd515b391587f6f1a2bc1f7ffb72e
gitHead: c182855723ad6217e95b89eb7fb1baeaaf26fd11
generatedAt: 2026-09-29T08:01:04Z
author: subagent/worker-FJ-016-specifier
---

# Specifier — FJ-016

## Observable criteria

- `GET /v1/models` returns HTTP `200` and the exact default JSON body containing the single `jev-latest` model with description `Local deterministic fake model provided by fake-jev.` and release date `1970-01-01`. When models are configured, it returns exactly those `name`, `description`, and `release_date` values in configuration order. Authorization headers are ignored for this request.
- A successful models request is a data-plane interaction journaled as `profile: "jev/v1"`, `operation: "models"`, outcome `matched`, `matchedStubId: null`, status `200`, `error: null`, and `requestBody: null`; it does not require or invoke a stub and does not create a verification failure.
- A successful non-raw `POST /v1/systemone` response has the exact top-level shape `model`, `answers`, and `usage`, with an answer-name set equal to the request question-name set. The response model is `then.model` verbatim when configured and otherwise echoes the request `model`.
- When `then.usage` is omitted, usage is exactly `{ "input_tokens": 0, "output_tokens": 0 }`. Configured usage, when supported by the response configuration, consists of non-negative integer `input_tokens` and `output_tokens` values; it is not populated from a model or provider.
- `then.raw` is encoded as a raw response without provider answer validation: `status` is required and may be `100..599`, headers default to empty, an absent body produces an empty response body, and a present body is serialized once as JSON. If a body is present and no case-insensitive `Content-Type` header was configured, `Content-Type: application/json` is added. Configured string-to-string headers are emitted subject to normal HTTP header normalization. A configured raw status such as `429` remains a matched response and is not by itself a fake-server verification failure.
- Every non-raw JSON data-plane response, including validation and fake-server JSON errors, carries a JSON-compatible `Content-Type: application/json` header (a charset parameter is permitted). The profile parses `POST /v1/systemone` as JSON even when `Content-Type` is absent or is `application/json`; strict media-type emulation is not added.
- No `Authorization` header, `Bearer fake`, or arbitrary bearer token is rejected or treated differently. Authorization values are never validated externally, used to select behavior, logged, or sent to a provider.

## Provider-neutral boundary

The compile-time registered `jev/v1` compatibility profile owns models-route behavior, response construction and encoding, model/usage defaults, raw-response rules, authentication pass-through, and Jev-specific JSON content-type behavior. The provider-neutral engine exchanges generic results and remains unaware of `/v1/models`, Jev response fields, raw encoding, model or usage semantics, and Authorization handling. Normal operation remains deterministic and requires no model, provider call, or network egress.

## Traces

- C-JEV-001 — §39.1 defines the exact default models response and configuration-order model metadata.
- C-JEV-002 — §39.1 defines the successful models interaction journal record, including no stub invocation and no verification failure.
- C-JEV-009 — §§13.6 and 39.4 define the configured response-model override and request-model echo default.
- C-JEV-010 — §§10.3 and 13.6 define exact zero usage defaults and non-negative integer configured usage.
- C-JEV-017 — §§10.1 and 39.5 require identical acceptance of missing and arbitrary bearer authentication without logging or external validation.
- C-JEV-018 — §39.7 defines raw status, headers, JSON body serialization, default content type, and raw response behavior; §44.9 defines that a configured raw `429` is matched without a fake failure.
- C-JEV-020 — §39.9 requires a JSON content type on every non-raw JSON data-plane response.
- C-ARCH-002 — §§9 and 9.1 require Jev behavior to live in a compile-time registered compatibility profile rather than the provider-neutral engine.
- C-GOLD-001 — §44.1 is the exact default `/v1/models` response vector.
- C-GOLD-009 — §44.9 is the configured raw error vector: status `429`, matched outcome, null error, invocation increment, and no verification failure solely from the raw status.

## Explicit non-goals

- No HTTP host/server, listener, routing multiplexer, request admission, journal implementation, or HTTP limit enforcement; this slice specifies profile behavior at the compatibility boundary.
- No authentication enforcement, token validation, credential lookup, Authorization logging, provider call, or network egress.
- No stub matching, selection, registration, invocation counting, response-sequence consumption, or verification implementation; a selected engine result/configured response is assumed at this boundary.
- No stub-selection logic or fallback model/provider behavior, and no model execution. The models list is metadata only.
- No arbitrary binary raw bodies, raw-byte passthrough, case-sensitive content-type requirement, or strict upstream media-type emulation; raw bodies remain JSON-serialized as specified.
- No changes to Jev request validation, answer-helper semantics, other compatibility profiles, control-plane APIs, or provider-neutral engine concepts.

This artifact records observable behavior and traceability only; it does not declare an implementation, gate, or acceptance verdict.
