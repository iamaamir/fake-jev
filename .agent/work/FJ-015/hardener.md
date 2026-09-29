---
stage: hardener
task: FJ-015
inputFingerprint: 66f35765afb2948730b0bac0c54bc1b08852900e356f16423d15d9ad2690710e
outputFingerprint: 66f35765afb2948730b0bac0c54bc1b08852900e356f16423d15d9ad2690710e
taskFingerprint: c4409d7a1fbba4b732c5b1d9cc70c922f2a2f717e6b47adce19907a99d646528
gitHead: 45156682a651b5d6278239e21266d0bd36efce30
generatedAt: 2026-09-29T07:50:01Z
author: subagent/worker-FJ-015-hardener
---

# Hardener — FJ-015

## Chain and scope

The hardener input fingerprint is
`66f35765afb2948730b0bac0c54bc1b08852900e356f16423d15d9ad2690710e`; the
output fingerprint remains
`66f35765afb2948730b0bac0c54bc1b08852900e356f16423d15d9ad2690710e`. The task
fingerprint is `c4409d7a1fbba4b732c5b1d9cc70c922f2a2f717e6b47adce19907a99d646528`.

| Stage | Input fingerprint | Output fingerprint | Task fingerprint |
|---|---|---|---|
| specifier | e4dcbb3aa2b0fc4eb88b39ce09fd65c349311014a1ba7afa29d2a26371c2fba7 | e4dcbb3aa2b0fc4eb88b39ce09fd65c349311014a1ba7afa29d2a26371c2fba7 | c4409d7a1fbba4b732c5b1d9cc70c922f2a2f717e6b47adce19907a99d646528 |
| coder | e4dcbb3aa2b0fc4eb88b39ce09fd65c349311014a1ba7afa29d2a26371c2fba7 | d2c3dbdd16b4a19a61aba2150ec84e5fa188c7bb3a4dc5edf1f851111e5d9849 | c4409d7a1fbba4b732c5b1d9cc70c922f2a2f717e6b47adce19907a99d646528 |
| cleaner | d2c3dbdd16b4a19a61aba2150ec84e5fa188c7bb3a4dc5edf1f851111e5d9849 | 66f35765afb2948730b0bac0c54bc1b08852900e356f16423d15d9ad2690710e | c4409d7a1fbba4b732c5b1d9cc70c922f2a2f717e6b47adce19907a99d646528 |
| hardener | 66f35765afb2948730b0bac0c54bc1b08852900e356f16423d15d9ad2690710e | 66f35765afb2948730b0bac0c54bc1b08852900e356f16423d15d9ad2690710e | c4409d7a1fbba4b732c5b1d9cc70c922f2a2f717e6b47adce19907a99d646528 |

The audit was limited to `internal/compat/jev/v1/fixture.go` and
`internal/compat/jev/v1/fixture_test.go`, with normative behavior from §13,
including §§13.1–13.5. No product-file edit was needed after the audit.

## Hardening audit and contract evidence

- Probability parsing rejects non-numbers, malformed JSON numbers, NaN/Inf,
  and values outside inclusive `[0,1]`; confidence uses the same finite range.
  Probability maps enforce exact key sets and `abs(sum-1) <= 1e-6`. This covers
  §13 and the explicit choice/score probability constraints in §§13.2–13.3.
- Choice explicit probabilities reject a selected value below the maximum,
  while equal maxima remain accepted. Omitted probabilities are generated over
  every criteria key, with one selected value and zeroes elsewhere. Sorted key
  traversal makes validation and error selection deterministic.
- Score bounds are checked against `[0,N-1]` before floor/ceiling interpolation.
  Generated maps cover every level, explicit maps cover exactly `"0"` through
  `"N-1"`, and the weighted expected value is checked within `1e-6`. Legends
  are copied from request criteria only; a fixture-supplied legend is rejected.
- Nil, absent, malformed, empty, and wrong-kind criteria fail closed for choice
  and score generation. Nil/empty/non-object answer documents, malformed
  helpers, unknown helper fields, helper/request type mismatches, invalid
  numbers, and invalid maps return `InvalidStubResponseError` through the
  top-level generator rather than being coerced or partially emitted.
- Exact answer coverage is checked before generation in both directions. Mixed
  requests therefore cannot receive partial fallback answers, and sorted
  request-name processing keeps generation deterministic. The implementation
  does not invoke providers, network code, or executable fixture content.

## Validation evidence

- `gofmt -w internal/compat/jev/v1/fixture.go internal/compat/jev/v1/fixture_test.go` — completed.
- `go test -count=1 ./...` — completed successfully.
- `go test -count=1 -race ./...` — completed successfully.
- `go vet ./...` — completed successfully.
- `go run ./cmd/guard arch` — completed successfully.
- `go run ./cmd/guard lint` — completed successfully.
- `go run ./cmd/guard trace` — completed successfully.
- `go run ./cmd/guard fuzz` — completed successfully.
- `./scripts/candidate-fingerprint candidate` —
  `66f35765afb2948730b0bac0c54bc1b08852900e356f16423d15d9ad2690710e`.
- `./scripts/candidate-fingerprint task .agent/work/FJ-015/state.json` —
  `c4409d7a1fbba4b732c5b1d9cc70c922f2a2f717e6b47adce19907a99d646528`.
- `./scripts/verify-candidate FJ-015` — hardener artifact, chain, and all
  repository/tool checks were recorded; only the downstream QA artifact remains
  absent (`42 passed, 1 failed, 3 skipped, 0 not_applicable`).

## Residual risks

- The hardening/mutation analyzer is unavailable under the repository bootstrap
  exemption; deterministic Go tests, race testing, vet, and repository guards
  are the available executable evidence.
- JSON object duplicate-name rejection is not defined by §13; the standard
  decoder's map semantics remain outside this fixture contract.
- Request validation and HTTP body-size limits are outside the two allowed
  fixture files; this audit covers the generator's validated-request boundary.
