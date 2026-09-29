---
stage: qa
task: FJ-016
inputFingerprint: b36ecf5e1da6ff235dbfb96eab8ac7959c74e72db5e7034edc9f7056ff04c954
outputFingerprint: b36ecf5e1da6ff235dbfb96eab8ac7959c74e72db5e7034edc9f7056ff04c954
taskFingerprint: fe3ad8fed06766d931689414827eac15a4dfd515b391587f6f1a2bc1f7ffb72e
gitHead: c182855723ad6217e95b89eb7fb1baeaaf26fd11
generatedAt: 2026-09-29T08:30:00Z
author: subagent/worker-FJ-016-qa
---

# QA observations — FJ-016

## Candidate and chain identity

- Current candidate fingerprint: `b36ecf5e1da6ff235dbfb96eab8ac7959c74e72db5e7034edc9f7056ff04c954`.
- Current task fingerprint: `fe3ad8fed06766d931689414827eac15a4dfd515b391587f6f1a2bc1f7ffb72e`.
- Hardener input (cleaner output): `3cda3a05216d178ed35fb4e0cdbb2593c9c80033b23930cfdd78ffc237326afb`.
- Hardener output/current candidate: `b36ecf5e1da6ff235dbfb96eab8ac7959c74e72db5e7034edc9f7056ff04c954`.
- Hardener artifact author: `subagent/worker-FJ-016-hardener`; this QA artifact author is distinct: `subagent/worker-FJ-016-qa`.

## Criteria exercised and observations

- **C-JEV-001 / C-GOLD-001:** `TestEncodeModelsGoldenAndConfiguredOrder` compared the default body byte-for-byte with the §44.1 vector and exercised configured metadata order. The encoded response was status 200 with `Content-Type: application/json`.
- **C-JEV-002:** The profile-level models response was exercised, but no HTTP host/control integration is present in this candidate to observe a journal record. The required `models` journal fields (`profile`, `operation`, matched outcome, null stub, status, null request body) remain a host integration observation gap.
- **C-JEV-009:** `TestEncodeSuccessDefaultsAndOverrides` observed request-model echo and verbatim configured `then.model` override in the parsed success envelope.
- **C-JEV-010:** The same test observed exact zero usage defaults and configured non-negative integer usage values.
- **C-JEV-017:** `TestProfileAcceptsAuthorizationWithoutInspectingIt` exercised missing, `Bearer fake`, and arbitrary bearer values across absent, JSON, and unrelated content types. All decoded identically; the profile boundary does not inspect headers. There is no host log surface in this candidate for a runtime log-capture observation.
- **C-JEV-018 / C-GOLD-009:** `TestEncodeRawResponse` exercised status 429 with a JSON error body, default content type, explicit case-insensitive content type, configured headers, omitted body, explicit JSON `null`, invalid status rejection, and invalid JSON rejection. The profile encoder preserves the raw 429 response; the end-to-end journal/verification assertion that it is matched with no fake failure is not available without the HTTP host.
- **C-JEV-020:** Success and models encoders were observed to emit `Content-Type: application/json`; `EncodeJSONError` uses the same header path. Validation/error HTTP dispatch is host-owned and has no system harness here.
- **C-ARCH-002:** `go run ./cmd/guard arch` reported no findings. The Jev wire behavior exercised above is contained in the compile-time profile package; the provider-neutral engine has no Jev response-field or route implementation.

## Commands and results

- `go test ./internal/compat/jev/v1 -count=1 -run 'TestEncodeSuccessDefaultsAndOverrides|TestEncodeResponseUsesFixtureAndExactWireShape|TestEncodeModelsGoldenAndConfiguredOrder|TestEncodeRawResponse|TestProfileAcceptsAuthorizationWithoutInspectingIt|TestProfilePreservesLargeJSONNumbers'` — passed (`fake-jev/internal/compat/jev/v1`).
- `go test ./... -count=1` — passed.
- `go test -race ./... -count=1` — passed.
- `go vet ./...` — passed.
- `go run ./cmd/guard arch` — passed; JSON reported `findings: []`, 40 files, 6 packages.
- `go run ./cmd/guard trace` — passed; JSON reported 0 active IDs and 17 covered IDs.
- `./scripts/selftest` — passed; `summary: 75 passed, 0 failed`.
- `./scripts/verify-candidate FJ-016` before writing this artifact — failed only at `stage qa artifact` with `stage.artifact_missing`; repository, Go test, vet, build, guard, and race rows passed, and cleaner/hardener tooling rows were skipped under their documented bootstrap exemptions.
- `./scripts/verify-candidate FJ-016` after writing this artifact — passed; report `.agent/reports/FJ-016/report-20260929T082416Z.json` recorded `48 passed, 0 failed, 3 skipped, 0 not_applicable`, including the QA chain and author checks.

## Residual risks and scope limits

- The repository has no HTTP host/listener or public system-test harness yet. Consequently, C-JEV-002 and the host portions of C-JEV-017, C-JEV-018, C-JEV-020, and C-GOLD-009 could not be observed end-to-end.
- Journaling, verification mapping, HTTP header normalization, authentication log capture, and configured raw 429 matched-outcome behavior remain host responsibilities outside the four allowed compatibility files.
- The artifact records observations and gaps only; no overall QA verdict is recorded.
