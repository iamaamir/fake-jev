---
stage: specifier
task: FJ-020
inputFingerprint: 4f811c765e1cf47c069ff957b82979f307f7376776475789c31e8183d700b178
outputFingerprint: 4f811c765e1cf47c069ff957b82979f307f7376776475789c31e8183d700b178
taskFingerprint: bf0432ef36830af414bf83f372280b27549851174ee9ee40d083af3e9fb25d42
gitHead: b198957
generatedAt: 2026-09-29T10:22:30Z
author: subagent/worker-FJ-020-specifier
---

# Specifier — FJ-020

## Observable criteria

- Control requests are recognized under `/__fake/v1`, remain separate from emulated provider routes, and never enter the application interaction journal or data-plane verification state. Unsupported control paths and methods return ordinary control-plane errors without becoming journal-visible interactions. Every JSON control response uses `Content-Type: application/json`, except the empty `204 No Content` response.
- `GET /__fake/v1/health` returns HTTP `200` and exactly the health shape `{status: "ok", serverVersion: "<semver>", controlApiVersion: "v1"}`.
- `GET /__fake/v1/meta` returns exactly the metadata shape from §41.3: `serverVersion`, `controlApiVersion: "v1"`, `configSchemaVersion: 1`, `mode: "strict"`, configured `activeProfiles` in their configured order, and the five configured limits (`dataPlaneBodyBytes`, `controlPlaneBodyBytes`, `maxInteractions`, `logBodyBytes`, and `gracefulShutdownSeconds`).
- `POST /__fake/v1/stubs` accepts one stub object using the configured `stubs[]` entry schema. A successful creation returns HTTP `201` with exactly its `id` and assigned `registrationIndex`.
- A dynamic stub ID that conflicts with an active static or dynamic stub is rejected with HTTP `409` and the exact `fake_jev_duplicate_stub_id` response containing the human-readable duplicate message and `stubId`; every stub ID remains non-empty and unique among active stubs.
- Invalid control JSON submitted through the control API returns HTTP `400` with exactly `{"error":"fake_jev_bad_control_request","message":"Invalid control request."}`. Profile/config validation failures also return HTTP `400` with top-level error `fake_jev_bad_control_request` and a human-readable `message`.
- `GET /__fake/v1/stubs` returns the exact top-level `stubs` array. Each item exposes only the defined `id`, `profile`, `priority`, `source`, `registrationIndex`, and `invocations` fields; `source` visibly distinguishes `static` from `dynamic`, and items are ordered by ascending registration index.
- `DELETE /__fake/v1/stubs` returns HTTP `204` with an empty body, removes dynamic stubs only, keeps static stubs active, and leaves static invocation and sequence counters unchanged.
- Matching preserves provenance precedence: an equal-priority dynamic stub registered after a matching static stub does not displace that static stub, while a dynamic stub with greater priority can win. The dynamic/static behavior is observable through matching and the listed `source`/registration metadata.

## Traces

- C-CTRL-001 — §41.2 exact health response and status.
- C-CTRL-002 — §41.3 exact metadata shape and configured profile order.
- C-CTRL-003 — §41.4 dynamic creation success and duplicate-ID conflict.
- C-CTRL-004 — §41.5 stub item shape, provenance, and registration order.
- C-CTRL-005 — §41.6 dynamic-only clearing and unchanged static counters.
- C-CTRL-010 — §41.1 exact `400` control-request error contract.
- C-CTRL-011 — §§11 and 41.1 JSON content type and control-plane response rule.
- C-MATCH-003 — §40.9 equal-priority static-before-dynamic precedence and explicit higher-priority override.
- C-GOLD-012 — §44.12 dynamic stub does not implicitly override static.

## Explicit non-goals

- No request-history endpoint or history clearing: `GET /__fake/v1/requests` and `DELETE /__fake/v1/requests` are not implemented or made journal-visible here.
- No full reset: `POST /__fake/v1/reset` is out of scope.
- No verification endpoint: `GET /__fake/v1/verify` is out of scope.
- No control request is recorded in the application interaction journal or used as a data-plane verification interaction.
- No changes to data-plane request validation, response generation, other control endpoints, CLI/lifecycle behavior, persistence, authentication, or deferred §33 features.

This artifact records observable behavior and traceability only; it does not declare an implementation or acceptance verdict.
