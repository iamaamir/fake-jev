---
stage: hardener
task: FJ-017
inputFingerprint: 6929bb38a12d3f0a5137180715368be877a0d74d12ce5b5f71cd84b793aa66f8
outputFingerprint: 35e197abcaf1232556d96fbea5a9fc08ead097834fc3063daa85935685723864
taskFingerprint: e56486e8ee7c2c403e987715b5cd6c1cb6aff7bce6387ecbd053b7a256bea81e
gitHead: 49263a222fc59e85c21cc4cff6bb017f42432294
generatedAt: 2026-09-29T09:01:52Z
author: subagent/worker-FJ-017-hardener
---

# Hardener — FJ-017

## Chain and scope

The hardener audit consumed the cleaner output
`6929bb38a12d3f0a5137180715368be877a0d74d12ce5b5f71cd84b793aa66f8` and
retains the previously applied HTTP-host hardening at candidate output
`35e197abcaf1232556d96fbea5a9fc08ead097834fc3063daa85935685723864`. The task
fingerprint is
`e56486e8ee7c2c403e987715b5cd6c1cb6aff7bce6387ecbd053b7a256bea81e`.

| Stage | Input fingerprint | Output fingerprint | Task fingerprint |
|---|---|---|---|
| cleaner | e7bf3ac7918e394e8334d7d0a08f9f20c3039486bc6d0183e6ec76d9d26f8cfc | 6929bb38a12d3f0a5137180715368be877a0d74d12ce5b5f71cd84b793aa66f8 | e56486e8ee7c2c403e987715b5cd6c1cb6aff7bce6387ecbd053b7a256bea81e |
| hardener | 6929bb38a12d3f0a5137180715368be877a0d74d12ce5b5f71cd84b793aa66f8 | 35e197abcaf1232556d96fbea5a9fc08ead097834fc3063daa85935685723864 | e56486e8ee7c2c403e987715b5cd6c1cb6aff7bce6387ecbd053b7a256bea81e |

The audit was limited to `internal/host/http/server.go`,
`internal/host/http/router.go`, and `internal/host/http/server_test.go`, with
normative behavior from §§17.4, 19.4, 43, and the selected acceptance contracts
C-HOST-004, C-HOST-006 through C-HOST-008, C-MATCH-010, and C-GOLD-014/015.

## Hardening audit and contract evidence

- `applyLimitDefaults` gives direct server construction and configuration-derived
  limits one defaulting path. Separate data/control limits remain enforced before
  body allocation; `ContentLength` rejects an obviously oversized body and a
  `LimitReader(limit+1)` detects oversized chunked/unknown-length input.
- Request bodies are closed on every server path. Data-plane oversize requests
  enter the serialized engine transition with a null raw body, a normal sequence,
  `payload_too_large` outcome, and no route, matcher, invocation, or response
  sequence mutation. Control-plane oversize requests remain outside the journal.
- Journal admission remains the engine serialization boundary. A full journal
  returns the exact 507 fake failure without assigning a sequence or selecting a
  stub; unknown routes remain journaled as exact 404 fake failures, while a
  recognized unmatched operation remains the exact 501 envelope.
- The consolidated `failureBody` preserves the specified field ordering and
  optional profile, operation, and stub fields for the §43 error classes.
  JSON-writing paths emit the required application/json content type; nil
  server, listener, request, and URL boundaries return controlled errors rather
  than entering compatibility dispatch unsafely.
- Route recognition remains exact for method/path, ignores only the query string,
  rejects trailing slashes, and keeps compatibility routing separate from the
  provider-neutral engine. No provider, network, or executable fixture boundary
  is introduced.

## Validation evidence

- `gofmt -d internal/host/http/server.go internal/host/http/router.go internal/host/http/server_test.go` — clean.
- `go test -count=1 ./...` — completed successfully; all repository packages, including `internal/host/http`, ran without cached results.
- `go test -count=1 -race ./...` — completed successfully; all repository packages ran under the race detector.
- `go vet ./...` — completed with no diagnostics.
- `go run ./cmd/guard arch` — no findings.
- `go run ./cmd/guard lint` — no findings.
- `go run ./cmd/guard trace` — no findings (`active: 0`, `covered: 17`).
- `go run ./cmd/guard fuzz` — no findings (`targets: 0`).
- `./scripts/candidate-fingerprint candidate .` — `35e197abcaf1232556d96fbea5a9fc08ead097834fc3063daa85935685723864`.
- `./scripts/candidate-fingerprint task .agent/work/FJ-017/state.json` — task fingerprint `e56486e8ee7c2c403e987715b5cd6c1cb6aff7bce6387ecbd053b7a256bea81e`.
- `./scripts/verify-candidate FJ-017` — report `.agent/reports/FJ-017/report-20260929T090144Z.json` recorded the hardener artifact, header, author, task fingerprint, and chain-link evidence; downstream QA remains an unproduced stage artifact.

## Residual risks

- Mutation/hardening analysis tooling remains under the repository bootstrap
  exemption; deterministic tests, race testing, vet, and repository guards are
  the available executable evidence.
- `NewServer` accepts a direct `Limits` value and only fills zero fields;
  configuration validation remains the boundary for rejecting invalid non-zero
  ranges. The host does not duplicate the configuration contract.
- Control endpoint implementation, graceful shutdown policy, and full journal
  bookkeeping remain outside this work item's allowed host slice or stated
  non-goals.
