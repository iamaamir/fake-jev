---
stage: qa
task: FJ-017
inputFingerprint: 35e197abcaf1232556d96fbea5a9fc08ead097834fc3063daa85935685723864
outputFingerprint: 35e197abcaf1232556d96fbea5a9fc08ead097834fc3063daa85935685723864
taskFingerprint: e56486e8ee7c2c403e987715b5cd6c1cb6aff7bce6387ecbd053b7a256bea81e
gitHead: 49263a222fc59e85c21cc4cff6bb017f42432294
generatedAt: 2026-09-29T09:09:03Z
author: subagent/worker-FJ-017-qa
---

# QA observations — FJ-017

## Candidate and predecessor identity

- Candidate fingerprint: `35e197abcaf1232556d96fbea5a9fc08ead097834fc3063daa85935685723864`.
- Task fingerprint: `e56486e8ee7c2c403e987715b5cd6c1cb6aff7bce6387ecbd053b7a256bea81e`.
- Hardener input (cleaner output): `6929bb38a12d3f0a5137180715368be877a0d74d12ce5b5f71cd84b793aa66f8`.
- Hardener output/current candidate: `35e197abcaf1232556d96fbea5a9fc08ead097834fc3063daa85935685723864`.
- Hardener artifact author: `subagent/worker-FJ-017-hardener`.
- QA artifact author: `subagent/worker-FJ-017-qa` (distinct from coder `subagent/worker-FJ-017-coder`).
- The temporary listener probe was removed after execution; no production or persistent test file was modified.

## Real-listener observations

An independent temporary Go probe used `httptest.NewServer` and `http.Client` requests, rather than `httptest.ResponseRecorder` or direct handler calls.

- **C-HOST-004 / C-GOLD-015:** Known-length and forced chunked oversized data bodies both returned status 413 with the exact `fake_jev_payload_too_large` body. Both journal entries had outcome `payload_too_large` and `RawBody == nil`; control-plane oversized input did not enter the journal. No stub state was available to mutate in this probe.
- **C-HOST-006:** An oversized `/__fake/v1/unknown` control request returned 413 with the same exact payload body and left the data journal unchanged. Control traffic did not consume the one available data interaction slot.
- **C-HOST-007:** `/v1/nope`, `/v1/models/`, and `POST /v1/models` each returned status 404 with the exact `fake_jev_unknown_route` body and were journaled as `unknown_route` interactions.
- **C-MATCH-010:** `POST /v1/systemone?ignored=1` without a configured stub returned status 501 with the exact `fake_jev_unmatched_request` body, including `profile: jev/v1` and `operation: systemone`.
- **C-GOLD-014:** With capacity one, the first models request returned 200 and the second returned status 507 with the exact `fake_jev_journal_full` body. The journal remained at one entry/sequence, and the sticky `journal_full` failure had `requestSequence == nil`.
- **C-HOST-008:** A zero-value `Limits` passed to `NewServer` produced exactly 8 MiB data body, 2 MiB control body, 10,000 interactions, 4 KiB log preview, and 5 second graceful-shutdown defaults.
- **§17.4 concurrency:** Thirty-two concurrent requests through the real listener all returned 404; 32 journal records were present with `unknown_route` outcomes and engine sequences 1 through 32. The targeted probe also ran under the race detector.

## Commands and results

- Temporary real-listener probe: `go test ./internal/host/http -count=1 -run 'TestQAFJ017RealListener' -v` — all four probe tests passed; the probe was then removed.
- Temporary real-listener probe under race detection: `go test -race ./internal/host/http -count=1 -run 'TestQAFJ017RealListener' -v` — all four probe tests passed.
- `go test ./... -count=1` — passed.
- `go test -race ./... -count=1` — passed.
- `go vet ./...` — passed with no diagnostics.
- `./scripts/selftest` — `summary: 75 passed, 0 failed`.
- `./scripts/verify-candidate FJ-017` — completed with `48 passed, 0 failed, 3 skipped, 0 not_applicable`; report `.agent/reports/FJ-017/report-20260929T091008Z.json`.
- Candidate fingerprint after probe removal: `35e197abcaf1232556d96fbea5a9fc08ead097834fc3063daa85935685723864`.

## Residual risks and observed gaps

- Data-plane `payload_too_large`, `unknown_route`, and unmatched outcomes were journaled, but `Engine.VerificationFailures()` remained empty for those interactions. The normative criteria require each to fail verification; only journal-full produced the observed sticky failure. This also means the probe could not observe the required payload/route/unmatched verification items.
- `InteractionRecord` exposes outcome and raw-body state but no error-code field. The exact wire error was observed, but the required journal error `fake_jev_payload_too_large` cannot currently be inspected through the existing record API.
- The independent probe used `httptest.NewServer` (a real listener) and did not exercise graceful shutdown or the deferred control API; those are outside this slice.
- The final task verifier is run after this artifact is written; its report is the authoritative binding check for this QA artifact.

## Acceptance observations

These statuses describe the observed criterion coverage only; this artifact does not provide an overall QA verdict.

- C-HOST-004: `not-satisfied` — HTTP 413, null body, and journal outcome observed; required verification failure was absent.
- C-HOST-006: `satisfied` — control 413 stayed outside the journal and did not create an interaction failure.
- C-HOST-007: `not-satisfied` — exact 404 and journaling observed; required verification failure was absent.
- C-HOST-008: `satisfied` — all five defaults matched the specification.
- C-MATCH-010: `not-satisfied` — exact 501 envelope observed; required verification failure was absent.
- C-GOLD-014: `satisfied` — exact 507 vector, no second sequence/journal entry, and sticky nil-sequence `journal_full` failure observed.
- C-GOLD-015: `not-satisfied` — exact 413 vector and unchanged body/stub state observed; required verification failure and journal error field were not observable.
