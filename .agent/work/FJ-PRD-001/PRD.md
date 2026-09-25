# PRD: fake-jev v1 — deterministic local test server for System One-style decision APIs

**Status:** ready-for-agent
**Source of truth:** `fake-jev-technical-spec-v2.md` (specification version 2.0, normative)
**Date:** 2026-09-25

---

## Problem Statement

As a developer whose application integrates with Jev (or any System One-style decision API), I cannot test my software without starting a model, downloading weights, hitting an external API, holding an API key, or spending money. In-process mocks only cover one language and skip the real integration path: request construction, serialization, base-URL configuration, HTTP client behavior, protocol validation, and response decoding. CI is either flaky, non-deterministic, networked, or expensive. I also have no reliable way to assert what my application actually asked the decision service, to force specific edge-case decisions, or to fail CI when the application makes an unexpected decision call.

## Solution

`fake-jev` is a small standalone Go HTTP server that emulates the Jev HTTP boundary at `localhost`, so an application can be tested by changing only its base URL. A developer writes strict, deterministic fixtures (YAML/JSON), starts the server, runs their existing test suite against it, and then asks the fake whether the interaction behaved as configured. Unmatched calls, unknown routes, validation failures, exhausted response sequences, and unsatisfied invocation expectations all fail verification and therefore fail CI. No model, API key, network egress, or payment is ever involved. Provider-specific behavior (`jev/v1`) sits behind a versioned compatibility profile so the provider-neutral engine never learns Jev's routes or primitives.

## User Stories

1. As a developer, I want to run a single binary that serves the Jev HTTP API on localhost, so that I can test without a model, API key, or network access.
2. As a developer, I want to point my existing Jev/SDK HTTP client at the fake by changing only the base URL, so that my production integration path is what actually gets tested.
3. As a developer, I want to declare a stub that answers a `noul` question with a fixed probability, so that I can exercise a specific binary decision branch.
4. As a developer, I want the `noul` boolean convenience form (`noul: true`) converted exactly to `1.0`/`0.0`, so that simple fixtures stay readable.
5. As a developer, I want to declare a `choice` fixture that selects one answer, so that I can route my application down a chosen path.
6. As a developer, I want a `choice` fixture without explicit probabilities to generate a one-hot distribution over all request criteria keys, so that I don't have to enumerate distractors.
7. As a developer, I want to supply explicit `choice` probabilities and confidence, so that I can test probability-sensitive application logic.
8. As a developer, I want a `score` fixture with an integer level, so that I can force a discrete rating.
9. As a developer, I want a fractional `score` fixture to interpolate deterministically between adjacent levels, so that I can test threshold behavior at boundaries.
10. As a developer, I want the score `legend` always derived from my request criteria, so that response legends stay consistent with what my application asked.
11. As a developer, I want to send a single request containing mixed `noul`, `choice`, and `score` questions, so that I can test real multi-question decisions.
12. As a developer, I want the response answer map to exactly cover the requested question names and nothing more, so that partial or stale fixtures are caught immediately.
13. As a developer, I want a stub matcher that requires an exact question-name set and exact per-name type, so that adding a new decision to my app cannot silently reuse an older fixture.
14. As a developer, I want matching ordered by priority descending then registration index ascending, so that stub selection is fully predictable with no hidden specificity scoring.
15. As a developer, I want a response sequence so that the same request returns different answers across successive calls, so that I can test retry and flapping workflows.
16. As a developer, I want sequence exhaustion to fail loudly with HTTP 409 and fail verification, so that extra calls never pass silently.
17. As a developer, I want a raw response escape hatch that returns any configured status/headers/JSON body, so that I can emulate provider errors like 429 before a first-class helper exists.
18. As a developer, I want a configured raw 429 to count as a successful match rather than a fake failure, so that testing my rate-limit handling doesn't fail CI.
19. As a developer, I want an unmatched request to return HTTP 501 with a fake error code rather than a plausible default answer, so that tests fail closed.
20. As a developer, I want an unknown data-plane route or wrong HTTP method to return 404 and fail verification, so that my application's drift in endpoints is visible.
21. As a developer, I want invalid request bodies to return the frozen deterministic 422 envelope, so that client-side validation bugs surface as wire behavior.
22. As a developer, I want the first validation error reported deterministically in lexicographic question order, so that failures are reproducible.
23. As a developer, I want `GET /v1/models` to return a deterministic default model list, so that application startup and model-selection code paths work offline.
24. As a developer, I want to configure my own model list that preserves config order, so that my app's model filtering logic sees the names it expects.
25. As a developer, I want the response `model` to echo the request model unless a stub overrides it, so that model-selection assertions work without extra fixture noise.
26. As a developer, I want default usage to be exactly zero input/output tokens, so that token-accounting assertions are deterministic.
27. As a developer, I want non-negative integer usage configured per stub, so that I can test usage-driven logic.
28. As a developer, I want bearer auth accepted but never validated or logged, so that clients sending `Authorization: Bearer …` work unchanged and secrets never leak into logs.
29. As a developer, I want to write fixtures in YAML or JSON that deserialize to the same logical schema, so that my team can pick the format it prefers.
30. As a developer, I want `schemaVersion: 1` required and validated, so that config drift is caught at startup rather than at request time.
31. As a developer, I want unknown configuration keys to fail validation, so that typos never silently disable a fixture.
32. As a developer, I want the `jev` alias normalized to `jev/v1`, so that short configs work while long-lived CI can pin the explicit identifier.
33. As a developer, I want duplicate static stub IDs rejected at config validation, so that ambiguous fixtures never load.
34. As a developer, I want `fake-jev validate config.yaml` to run all static checks without starting a server or touching the network, so that CI can lint fixtures cheaply.
35. As a developer, I want to register a dynamic stub at runtime via the control API, so that a test can reconfigure a running server without a restart.
36. As a developer, I want a dynamic stub with a duplicate ID rejected with 409, so that fixture identity stays unambiguous.
37. As a developer, I want an equal-priority dynamic stub NOT to implicitly override a static stub, so that precedence is explicit and never hidden.
38. As a developer, I want to list active stubs with their source, registration index, and invocation counts, so that I can debug matching behavior.
39. As a developer, I want to clear only dynamic stubs while static fixtures stay intact, so that per-test fixtures don't disturb the baseline.
40. As a developer, I want to inspect the full interaction history in deterministic sequence order, so that I can assert what my application actually sent.
41. As a developer, I want interaction records to include normalized request, raw body when available, matched stub ID, outcome, status, and error code, so that failures are diagnosable from the journal alone.
42. As a developer, I want interaction sequence numbers assigned atomically before routing/validation, so that ordering is stable even under concurrent requests.
43. As a developer, I want a single `reset` operation that atomically removes dynamic stubs, zeroes counters, rewinds sequences, clears history, and restores registration indices, so that each test starts from a known state.
44. As a developer, I want `GET /__fake/v1/verify` to return a structured pass/fail result with ordered failure items, so that CI can fail on any unexpected interaction.
45. As a developer, I want verification to fail on unmatched requests, unknown routes, validation failures, sequence exhaustion, journal overflow, internal errors, and unsatisfied `expect` counts, so that no category of surprise passes CI.
46. As a developer, I want `expect: {exactly: 1}` and `{atLeast, atMost}` forms, so that I can assert a fixture was exercised the right number of times.
47. As a developer, I want the invocation counter to increment even when response generation later fails, so that extra calls remain observable.
48. As a developer, I want a single command that starts the fake, injects its URL into a child process, runs my test suite, verifies interactions, and shuts down, so that CI needs no bespoke process orchestration.
49. As a developer, I want `run` to exit with the child's code when the child fails, exit 3 when verification fails, and exit 2 on startup failure, so that failure attribution in CI is unambiguous.
50. As a developer, I want `run` to use an ephemeral port by default, so that parallel CI jobs never collide on 8787.
51. As a developer, I want an atomically-written ready file carrying the resolved URL/host/port/pid, so that wrappers never scrape human logs for readiness.
52. As a developer, I want `serve` to handle SIGINT/SIGTERM with graceful shutdown within a configured timeout and to clean up its own ready file, so that test runs terminate cleanly.
53. As a developer, I want `fake-jev verify --url …` to be a thin HTTP call to the control endpoint, so that verification logic exists in exactly one place.
54. As a developer, I want the control API to answer health and meta (versions, mode, active profiles, limits), so that wrappers can do version-aware compatibility checks.
55. As a developer, I want configuration precedence fixed at CLI flag > config file > built-in default, so that behavior is explainable in one sentence.
56. As a developer, I want no environment variable to implicitly reconfigure the server (except `run`'s child URL injection), so that CI behavior doesn't depend on ambient env.
57. As a developer, I want the default bind address to be 127.0.0.1, so that a test server is never exposed to a trusted-network boundary by accident.
58. As a developer, I want enforced body-size, journal-size, and log-preview limits, so that a runaway test cannot exhaust memory.
59. As a developer, I want journal overflow to return 507 and fail verification rather than silently evicting old entries, so that history assertions stay sound.
60. As a developer, I want oversized data-plane requests recorded as `payload_too_large` with a null body and failing verification, so that limit breaches are visible in CI.
61. As a developer, I want logs optimized for test debugging (matched/unmatched per sequence number) with Authorization values redacted, so that failures are quick to triage.
62. As a developer, I want JSON log format available via flag, so that log pipelines can ingest fake-jev output.
63. As a developer, I want `fake-jev version` to print product version, Go build info, built-in profiles, control API version, and config schema version, so that bug reports carry reproducible context.
64. As a contributor, I want the fake engine isolated from `net/http`, the CLI, and Jev-specific code, so that engine behavior is unit-testable without a network and remains WASM-portable.
65. As a contributor, I want Jev routes, validation, and fixture helpers confined to the `jev/v1` compatibility profile, so that a future `jev/v2` or new provider is added without touching the engine.
66. As a contributor, I want committed machine-readable contract artifacts (config JSON Schema, control OpenAPI, upstream jev snapshot) and golden vectors as executable tests, so that conformance is enforced by the test suite rather than by re-reading prose.
67. As a contributor, I want phase-by-phase acceptance gates, so that implementation progress is verifiable and scope creep is blocked.
68. As a contributor, I want CI to run tests, race detector, vet, and builds with zero network egress and zero credentials, so that contributions are reviewable offline.
69. As a user, I want a native binary for linux/darwin/windows, so that I never need a Go toolchain installed.
70. As a user, I want a small Docker image binding through config, so that containerized test jobs work unchanged.
71. As a user, I want documentation stating plainly that fake-jev does not simulate model quality or intelligence, so that expectations stay correct.

## Implementation Decisions

### Modules to build

Seven modules, confirmed with the user. Dependencies point inward: CLI/hosts → compatibility profile → engine; control API → engine. The engine imports neither `net/http`, `os/exec`, CLI, control, nor provider packages.

1. **Engine** (provider-neutral, pure in-memory). Public surface: load compiled configuration, apply a data-plane transition for an `Exchange`, and perform control operations — register dynamic stub, list stubs, clear dynamic stubs, append/read/clear interaction journal, full reset, compute verification. Encapsulates: stub registry with static/dynamic provenance and registration indices; candidate selection restricted to the active profile; combined-AND matcher with omitted-field wildcards; total ordering by priority descending then registration index ascending; JSON state equality with precision-preserving numeric comparison; exact question-name-set and per-name type matching; sequence position consumption; invocation counters incremented at selection time; monotonically assigned interaction sequence; sticky verification failures for conditions the bounded journal cannot represent; expectation evaluation. No filesystem, no wall-clock influence on matching.
2. **`jev/v1` compatibility profile**. Public surface: route recognition (exact paths, query ignored, method-sensitivity, no trailing-slash alias), decode+validate a raw request into an `Exchange` or the frozen 422 envelope, and encode an engine `Result` into a wire response. Encapsulates: the frozen request schema and first-error-in-lexicographic-order rule; noul/choice/score fixture helper generation (boolean coercion, one-hot generation, explicit-probability validation, fractional score interpolation, legend derivation, probability sum and expected-value tolerances); answer-map exact coverage enforcement producing `fake_jev_invalid_stub_response`; model and usage defaults; raw response encoding including Content-Type defaulting; model-list construction. Registered at compile time via an internal registry; no dynamic plugins.
3. **Configuration loader and validator**. Public surface: bytes (YAML or JSON) → validated configuration model or a list of structured errors. Encapsulates: `schemaVersion: 1` enforcement; allowed-key sets with fail-closed unknown-key rejection at every project-owned level, while leaving `state`, question content, and `then.raw.body` as open JSON containers; profile alias normalization (`jev` → `jev/v1`) before duplicate detection; stub envelope, matcher, response-shape exclusivity (`answers` XOR `sequence` XOR `raw`), and `expect` form rules; duplicate static ID rejection; server/limits/model validation ranges; application of built-in defaults. Hosts and the engine consume the validated model only.
4. **HTTP host**. Public surface: listener lifecycle plus request dispatch. Encapsulates: atomic per-request serialized transition covering sequence assignment → route/decode/validate → stub selection → counter/sequence mutation → outcome recording; data-plane vs control-plane split on the `/__fake/` prefix (control requests never enter the journal); per-plane body limits producing 413 with correct journaling semantics; journal capacity producing 507 with no stub or sequence mutation and a sticky failure; concurrent request handling with race-free engine state; graceful shutdown within the configured timeout; atomic ready-file creation and PID-guarded cleanup; log emission to stderr with Authorization redaction and body truncation.
5. **Control API**. Public surface: the nine `__fake/v1` endpoints. Encapsulates: exact response contracts for health, meta, stub create (201/409), stub list, stub clear (204), request history, request clear (204), reset (204), verify (always 200 with pass/fail body); 400 handling for invalid control JSON and profile/config validation failures; failure-item shape and ordering (interaction-derived by sequence ascending, then expectation failures by registration index ascending). Delegates all state changes to the engine — no independent state.
6. **CLI**. Public surface: `serve`, `validate`, `verify`, `run`, `version`. Encapsulates: flag parsing with fixed precedence (flag > file > default); `--port 0` and `run`'s default-ephemeral-port behavior; ready-file contents; validate's no-server/no-network guarantee; verify's delegation to the control endpoint; run's ten-step lifecycle (validate → start on ephemeral port → wait ready → set `FAKE_JEV_URL` and optional `--url-env` → spawn child → wait → verify → graceful stop → exit); exit-code precedence (0 success, 2 usage/config/startup/operational, 3 verification, child code preserved on child failure with both failures reported to stderr); SIGINT/SIGTERM handling for serve and signal forwarding for run; logs to stderr, shell-consumable data to stdout.
7. **Contract artifacts and golden harness**. The spec-mandated artifact locations from specification §45: config JSON Schema, control API OpenAPI document, `jev/v1` upstream OpenAPI snapshot, and the golden vector corpus organized by concern (models, noul, choice, score, mixed, validation, matching, sequences, reset, verification, cli). Golden vectors from §44 become executable tests; changes under the contract artifact tree are high-scrutiny changes.

### Architectural and behavioral decisions

- The engine's exchange model carries a profile ID, operation, payload value tree, and metadata; it never requires a compile-time union of `choice | noul | score`.
- Compatibility profiles are compiled in and registered internally; no native plugins, no plugin ABI in v1.
- The only execution mode is `strict`; `auto` mode values fail validation.
- Config precedence is CLI flag > file > built-in default; environment configuration is absent except `run`'s child URL injection.
- Matching order is priority descending then registration index ascending, with no specificity scoring; an omitted matcher field is a wildcard; static and dynamic stubs share one ordering namespace so equal-priority statics win by earlier registration, and dynamic override requires an explicitly greater priority.
- Response shape exclusivity: a stub defines exactly one of `answers`, `sequence`, `raw`; sequence elements may be `answers` or `raw` but never nested sequences; no implicit repeat-last; exhaustion returns the defined 409 failure and fails verification.
- Invocation counters increment once at selection, before response generation, and survive response-generation failure.
- Interaction sequence numbers are assigned before routing/validation and reset to 1 on request-history clear and on full reset; the journal is bounded with no eviction in v1.
- Full reset is one atomic control operation over dynamic stubs, counters, sequence positions, history, sticky interaction-derived failures, the next interaction sequence, and the next dynamic registration index (`N+1` where `N` is the static stub count).
- Verification failure codes are the fixed set (unmatched, unknown route, validation, payload too large, sequence exhausted, invalid stub response, journal full, internal error, and the three expectation codes); intentionally configured raw non-2xx responses are successful matches and never fail verification by status.
- Wire contracts are frozen: data-plane routes and method exactness, the deterministic 422 detail envelope, the 501/404/409/500/507/500/413 fake-failure bodies, model list default and ordering, usage default of exactly 0/0, response-model echo, and auth acceptance without validation.
- Control API lives under a versioned namespace and is a project-owned public contract; breaking changes require a new namespace.
- Limits are the spec's normative defaults and are enforced by the host.
- Raw fixtures are data-only: no templates, expressions, shell, JavaScript, or code execution anywhere in configuration.
- No network egress in normal execution; default bind is loopback.
- Distribution target is a single native Go binary (module minimum Go 1.26, CI on 1.26.x and 1.27.x), with a Docker image; WASM remains a future target that the engine-purity rule preserves but v1 does not ship.

### Schema and API contracts

- Configuration: top-level `schemaVersion`, `server`, `mode`, `compatibility`, `limits`, `models`, `stubs`; stub envelope `id`/`profile`/`priority`/`when`/`then`/`expect`; `jev/v1` matcher keys `operation`/`model`/`state`/`questions`; response shapes and expect forms as above. Expressed normatively in specification §38 and materialized as the committed JSON Schema.
- Data plane: `POST /v1/systemone`, `GET /v1/models`, plus the exact unknown-route treatment.
- Control plane: health, meta, stub create/list/clear, request history/clear, reset, verify — with the exact status codes and JSON shapes in specification §41.

## Testing Decisions

### What makes a good test

- Assert only externally observable behavior: HTTP status, response headers/body, control-API JSON, exit codes, journal/verification output. Never assert internal types, lock strategy, package layout, or log wording (except where the spec freezes output).
- Compare parsed JSON values rather than bytes, except when the subject is explicitly raw bytes; ignore object-key order and insignificant number spelling.
- Every golden vector must be runnable offline with no credentials, no model, and no egress.
- Tests must be deterministic: no reliance on wall-clock timestamps, port numbers (except where the spec defines ready-file semantics), or network ordering of truly racy requests. Sequence-sensitive tests issue calls in controlled order.
- A test failing must point at a violated contract clause, not at an implementation detail.

### Modules under test

All seven modules, confirmed with the user:

1. **Engine** — deterministic matching, priority/registration ordering, wildcard omission, exact question-set semantics, state equality (including numeric precision cases), sequence consumption and exhaustion, invocation counters (including count-on-failed-generation), journal ordering and capacity, reset atomicity, expectation evaluation, concurrency under `go test -race`.
2. **`jev/v1` profile** — frozen 422 envelopes for every validation branch, first-error ordering, noul/choice/score helper generation including one-hot, fractional interpolation, probability sum/expected-value tolerances, legend derivation, answer-map coverage failures, model and usage defaults, raw encoding, route and method exactness.
3. **Configuration** — unknown-key rejection at each level, alias normalization before duplicate detection, response-shape exclusivity, expect-form rules, defaults application, YAML/JSON equivalence, duplicate static IDs.
4. **HTTP host** — sequence assignment before validation, data/control plane split, 413 and 507 semantics with no stub mutation, graceful shutdown, atomic ready file, concurrency safety.
5. **Control API** — exact contract for every endpoint including 400/409/204 paths, verify pass/fail shapes and failure ordering, reset's atomic multi-effect semantics.
6. **CLI** — exit-code matrix (including the run child/verify precedence table), validate's no-network guarantee, run lifecycle and env injection, signal handling, ready-file contents.
7. **Contract artifacts** — the §44 golden vector corpus as executable tests, plus schema validation of the committed artifacts themselves.

Cross-cutting: fuzz tests over JSON request parsing, malformed control payloads, matcher input, and state transitions, with the invariant that untrusted input yields a controlled error and never a panic or corrupted state. An integration suite that boots the real HTTP host on an ephemeral port and tests through HTTP rather than by calling handlers directly.

### Prior art

The repository is greenfield — there are no existing tests to mirror. The reference patterns are defined by the specification itself: §22 (engine unit tests, fuzz/property testing, offline golden fixtures, HTTP integration through a real listener, control-API tests) and §44 (the golden vector corpus that becomes the shared harness). The spec's phase acceptance gates are the template for how each module's tests are grouped and gated. A cross-repository smoke test pointing `system-one-core`'s HTTP provider at a running instance is required before a public v1 release but not in every change.

## Out of Scope

Everything in the specification's "Decisions intentionally deferred" (§33), which must not be pulled forward: auto-generated decision mode; named scenario/state-machine DSL beyond response sequences; browser-hosted WASM distribution; WASM component plugin ABI; third-party dynamic compatibility plugins; persistent state or databases; UI/dashboard; remote multi-user operation; control-plane authentication; latency/fault injection; record/replay against real providers; automatic provider OpenAPI ingestion; provider-quality evaluation; probabilistic/fuzz response generation; generic HTTP mocking outside the System One testing domain.

Also out of scope for this PRD: implementation phases 5 and 6 (TypeScript wrapper and WASM experiment), language wrappers in any language, npm/Homebrew/GitHub Action distribution channels, release build matrix and checksum automation beyond what phase 4 requires, live upstream drift testing, and any behavior not defined by the specification — per the no-invention rule, an unspecified observable behavior is a specification gap to be surfaced, not standardized by code.

## Further Notes

- Implementation proceeds as the specification's vertical slices (§31), and each phase's acceptance gate must pass before the next phase begins. Contract artifacts are committed and reviewable before protocol implementation starts.
- The test suite, not model intuition, is the final enforcement mechanism: specification → wire/config contracts → golden vectors → implementation → contract tests → conformance.
- If any two normative parts of the specification conflict, stop that implementation path and report the conflict rather than selecting the convenient interpretation or inventing behavior.
- Per-module test scope was confirmed with the user as all seven modules.
