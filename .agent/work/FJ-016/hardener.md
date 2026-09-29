---
stage: hardener
task: FJ-016
inputFingerprint: 3cda3a05216d178ed35fb4e0cdbb2593c9c80033b23930cfdd78ffc237326afb
outputFingerprint: b36ecf5e1da6ff235dbfb96eab8ac7959c74e72db5e7034edc9f7056ff04c954
taskFingerprint: fe3ad8fed06766d931689414827eac15a4dfd515b391587f6f1a2bc1f7ffb72e
gitHead: c182855723ad6217e95b89eb7fb1baeaaf26fd11
generatedAt: 2026-09-29T08:17:19Z
author: subagent/worker-FJ-016-hardener
---

# Hardener — FJ-016

## Chain and scope

The hardener input is the cleaner candidate
`3cda3a05216d178ed35fb4e0cdbb2593c9c80033b23930cfdd78ffc237326afb`; the
current hardened candidate is
`b36ecf5e1da6ff235dbfb96eab8ac7959c74e72db5e7034edc9f7056ff04c954`. The
task fingerprint is
`fe3ad8fed06766d931689414827eac15a4dfd515b391587f6f1a2bc1f7ffb72e`.

| Stage | Input fingerprint | Output fingerprint | Task fingerprint |
|---|---|---|---|
| specifier | 66f35765afb2948730b0bac0c54bc1b08852900e356f16423d15d9ad2690710e | 66f35765afb2948730b0bac0c54bc1b08852900e356f16423d15d9ad2690710e | fe3ad8fed06766d931689414827eac15a4dfd515b391587f6f1a2bc1f7ffb72e |
| coder | 66f35765afb2948730b0bac0c54bc1b08852900e356f16423d15d9ad2690710e | 9b41eb056a93671e363b2cd42cb0948511dec384fba10e9df96e8b7d693ad668 | fe3ad8fed06766d931689414827eac15a4dfd515b391587f6f1a2bc1f7ffb72e |
| cleaner | 9b41eb056a93671e363b2cd42cb0948511dec384fba10e9df96e8b7d693ad668 | 3cda3a05216d178ed35fb4e0cdbb2593c9c80033b23930cfdd78ffc237326afb | fe3ad8fed06766d931689414827eac15a4dfd515b391587f6f1a2bc1f7ffb72e |
| hardener | 3cda3a05216d178ed35fb4e0cdbb2593c9c80033b23930cfdd78ffc237326afb | b36ecf5e1da6ff235dbfb96eab8ac7959c74e72db5e7034edc9f7056ff04c954 | fe3ad8fed06766d931689414827eac15a4dfd515b391587f6f1a2bc1f7ffb72e |

The audit and edits stayed within the four allowed compatibility files:
`internal/compat/jev/v1/response.go`, `models.go`, `profile.go`, and
`response_test.go`.

## Hardening actions and contract evidence

- Successful response encoding now fails closed unless the supplied answer map
  has exactly the request question-name set and each answer is valid JSON. This
  protects §39.4's exact answer-map contract on both fixture-generated and
  engine-result paths.
- Direct response encoding now rejects negative configured token counts instead
  of emitting an invalid usage object. Omitted usage remains the exact pair of
  zero integers required by §§10.3 and 13.6; configured non-negative integer
  values remain unchanged.
- Raw response encoding continues to validate status `100..599`, preserve a
  missing body as empty, serialize a present JSON body once (including explicit
  `null`), clone configured headers, and add a case-insensitive default JSON
  content type only when a body is present (§39.7). Invalid raw JSON reaches an
  error path rather than being emitted.
- Models encoding retains deterministic default metadata, configured ordering,
  and copied slices. Non-raw success, models, and JSON-error paths emit
  `Content-Type: application/json` as required by §§39.1 and 39.9.
- Profile decoding does not inspect `HostRequest.Headers`, so absent,
  `Bearer fake`, arbitrary bearer values, and unrelated content types do not
  alter valid JSON decoding. Authorization is not validated or logged (§§39.5
  and 39.6). `UseNumber` preserves large JSON state integers without binary
  floating-point conversion.

## Validation evidence

- `gofmt -w internal/compat/jev/v1/response.go internal/compat/jev/v1/models.go internal/compat/jev/v1/profile.go internal/compat/jev/v1/response_test.go` — completed.
- `go test ./...` — completed successfully.
- `go test -race ./...` — completed successfully.
- `go vet ./...` — completed successfully.
- `go run ./cmd/guard arch` — completed with no findings.
- `go run ./cmd/guard lint` — completed with no findings.
- `go run ./cmd/guard trace` — completed with no findings.
- `go run ./cmd/guard fuzz` — completed with no findings.
- `./scripts/candidate-fingerprint candidate .` —
  `b36ecf5e1da6ff235dbfb96eab8ac7959c74e72db5e7034edc9f7056ff04c954`.
- `./scripts/candidate-fingerprint task .agent/work/FJ-016/state.json` —
  `fe3ad8fed06766d931689414827eac15a4dfd515b391587f6f1a2bc1f7ffb72e`.
- `./scripts/verify-candidate FJ-016` — report
  `.agent/reports/FJ-016/report-20260929T081821Z.json` recorded the hardener
  artifact and chain; repository, Go tests/build, vet, race, architecture,
  lint, trace, and bounded-fuzz checks were exercised. The QA artifact remains
  downstream work.

## Residual risks

- Mutation/hardening analysis and the public HTTP/system harness remain under
  repository bootstrap exemptions; deterministic tests, race testing, vet,
  and repository guards are the available evidence here.
- HTTP body-size enforcement, response-header normalization, journaling, and
  verification mapping remain host responsibilities outside the allowed files.
- `config` validation normally rejects negative usage and invalid raw status;
  the profile boundary independently rejects these values when called directly.
