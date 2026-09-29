---
stage: hardener
task: FJ-014
inputFingerprint: ab4294d2021797bf919b58dbe3c2a61d068e4f998715ec42e28a82aac96b3db2
outputFingerprint: e4dcbb3aa2b0fc4eb88b39ce09fd65c349311014a1ba7afa29d2a26371c2fba7
taskFingerprint: 1df19a608667d96667c2059da5f0a35005334f81ca89b7ab6c1307c5440d2021
gitHead: 2944771
generatedAt: 2026-09-29T07:02:37Z
author: subagent/worker-FJ-014-hardener
---

# Hardener — FJ-014

## Chain and scope

The hardener input fingerprint was
`ab4294d2021797bf919b58dbe3c2a61d068e4f998715ec42e28a82aac96b3db2`; the
hardened candidate output is
`e4dcbb3aa2b0fc4eb88b39ce09fd65c349311014a1ba7afa29d2a26371c2fba7`. The task fingerprint
is `1df19a608667d96667c2059da5f0a35005334f81ca89b7ab6c1307c5440d2021` and the
chain is:

| Stage | Input fingerprint | Output fingerprint | Task fingerprint |
|---|---|---|---|
| specifier | a63a3d6fedef600dcdcdb8aed573eef907f8fea92e52605686e2133a0dea1b4c | a63a3d6fedef600dcdcdb8aed573eef907f8fea92e52605686e2133a0dea1b4c | 1df19a608667d96667c2059da5f0a35005334f81ca89b7ab6c1307c5440d2021 |
| coder | a63a3d6fedef600dcdcdb8aed573eef907f8fea92e52605686e2133a0dea1b4c | ab4294d2021797bf919b58dbe3c2a61d068e4f998715ec42e28a82aac96b3db2 | 1df19a608667d96667c2059da5f0a35005334f81ca89b7ab6c1307c5440d2021 |
| cleaner | ab4294d2021797bf919b58dbe3c2a61d068e4f998715ec42e28a82aac96b3db2 | ab4294d2021797bf919b58dbe3c2a61d068e4f998715ec42e28a82aac96b3db2 | 1df19a608667d96667c2059da5f0a35005334f81ca89b7ab6c1307c5440d2021 |
| hardener | ab4294d2021797bf919b58dbe3c2a61d068e4f998715ec42e28a82aac96b3db2 | e4dcbb3aa2b0fc4eb88b39ce09fd65c349311014a1ba7afa29d2a26371c2fba7 | 1df19a608667d96667c2059da5f0a35005334f81ca89b7ab6c1307c5440d2021 |

The audit was
limited to `internal/compat/jev/v1/routes.go`, `request.go`, `validation.go`,
and `validation_test.go`, plus §§39.2, 39.3, 39.8, and the traced vectors.

## Hardening audit and contract evidence

- `RecognizeRoute` compares exact uppercase method/path pairs, removes only the
  query suffix, rejects trailing slashes and wrong methods, and does not invoke
  redirects or provider code. This serves C-JEV-019 and C-GOLD-016.
- `decodeObject` requires one complete JSON value, rejects malformed input,
  rejects a non-object top-level value as a schema error, and rejects trailing
  JSON values. The rejected second value is decoded as `json.RawMessage` rather
  than `any`, avoiding recursive map/slice allocation on untrusted trailing
  input. Nil and empty bodies therefore return the deterministic malformed JSON
  detail rather than panicking. This serves C-JEV-003, C-JEV-004, and
  C-GOLD-010.
- Unknown fields remain accepted and retained as raw JSON at the top level and
  question level, as required by §39.2. Duplicate object names retain the
  standard decoder's deterministic last-value behavior; the specification does
  not define duplicate-name rejection or a duplicate-specific envelope, so no
  new public behavior was introduced.
- Validation checks `state`, `model`, and `questions` before sorting question
  names lexicographically and returns immediately after the first failure.
  Type/kind checks are bounded to the frozen schema, and criteria/instruction
  raw values are copied before returning. This serves C-JEV-003 and C-JEV-007.
- Error constructors produce one copied location slice and one `Detail`; the
  malformed, missing, value, and unsupported-type forms carry fixed message and
  type strings. `ValidationError` nil/error serialization paths remain
  non-panicking and status is fixed at 422. This serves C-JEV-004 through
  C-JEV-006 and C-GOLD-010/C-GOLD-011.
- The compatibility package imports only `bytes`, `encoding/json`, `io`, `sort`,
  and `strings`; no provider, network, HTTP, engine, or execution boundary is
  reachable from this slice. This preserves C-ARCH-002 and C-ARCH-004.

## Validation evidence

- `gofmt -w internal/compat/jev/v1/routes.go internal/compat/jev/v1/request.go internal/compat/jev/v1/validation.go internal/compat/jev/v1/validation_test.go` — completed.
- `go test ./...` — completed successfully.
- `go test -race ./...` — completed successfully.
- `go vet ./...` — completed successfully.
- `./scripts/selftest` — 75 repository guard scenarios completed successfully.
- `./scripts/candidate-fingerprint candidate .` —
  `e4dcbb3aa2b0fc4eb88b39ce09fd65c349311014a1ba7afa29d2a26371c2fba7`.
- `./scripts/candidate-fingerprint task .agent/work/FJ-014/state.json` —
  `1df19a608667d96667c2059da5f0a35005334f81ca89b7ab6c1307c5440d2021`.
- `./scripts/verify-candidate FJ-014` — repository, stage-chain, Go test/vet/build,
  architecture, lint, trace, fuzz, and race checks were exercised; the report
  records the downstream QA artifact as not yet present (`42 passed, 1 failed,
  3 skipped, 0 not_applicable`).

## Residual risks

- Duplicate JSON object-name semantics are not specified by §§39.2–39.3;
  `encoding/json` uses the last occurrence, and raw interaction preservation
  does not expose duplicate occurrences through the map-shaped request type.
- Body-size enforcement belongs to the HTTP host limit in §19.4/§43.7 and is
  outside the four allowed compatibility files; this validator accepts the
  byte slice it is given and does not invent a second size policy.
- Mutation/hardening tooling remains under the repository bootstrap exemption;
  deterministic tests and Go toolchain checks are the available evidence.
