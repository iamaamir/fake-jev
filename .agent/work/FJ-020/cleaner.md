---
stage: cleaner
task: FJ-020
inputFingerprint: 8dacde273d16cf959da358666963b3afdbec066ac5dd4ebea125c094c4796740
outputFingerprint: 8dacde273d16cf959da358666963b3afdbec066ac5dd4ebea125c094c4796740
taskFingerprint: bf0432ef36830af414bf83f372280b27549851174ee9ee40d083af3e9fb25d42
gitHead: b198957ebbd5beebffa3cf118d89fb55f3c8efc0
generatedAt: 2026-09-29T15:03:59Z
author: worker/FJ-020-cleaner-post-413-independent-refresh
---

# Cleaner evidence after the 413 correction

Reviewed the current task packet, state, stage artifacts, all three allowed
control files (untracked, therefore absent from ordinary tracked diff), cleaner
role and staff-architect guidance, specification §§11.2, 17, 41.1–41.6 and 43.7,
and the nine acceptance catalog entries. Inspected host dispatch and engine
registration/removal boundaries. The coder output matches this review input.
Bindings above come from candidate-fingerprint and git rev-parse commands.

No structural cleanup is justified. HTTP routing and serialization remain in
control, config.Load owns schema validation, and engine owns mutable stub state.
RawMessage retains field presence, nulls and duplicate keys for schema validation.
Extracting compilation or changing constructors would widen this refresh without
a demonstrated behavior-preserving benefit. Formatting is already canonical.
Only this cleaner artifact was edited; no production, test, state, report,
other stage artifact or staging changes were made by this review.

## Current observations

- The narrow correction in createStub classifies decoder-reported
  *http.MaxBytesError with errors.As and emits the exact §43.7 two-field 413
  envelope. Valid-size malformed JSON retains its exact 400 envelope. Leading
  and trailing whitespace overflow, large bodies, known/unknown content lengths,
  below/exact-limit bodies, and malformed/extra JSON are exercised by existing
  tests, including bounded reads and journal/verification isolation.
- The preserved independent QA body_limit probe now succeeds. The prior cleaner
  observation of 400 and older QA observation of 201 are historical, not current
  behavior. All eight preserved QA subtests execute successfully against this
  candidate; the temporary probe matches the Go block in qa.md verbatim apart
  from surrounding whitespace.
- Health/meta exact shapes, semver fallback and supplied version, profile order,
  schema-preserving validation, duplicate rejection and dynamic-only clearing
  remain covered. The independent lifecycle probe checks exact static-ID 409,
  six-field list, empty 204, nonzero static counter/sequence preservation, static
  tie precedence and a higher-priority dynamic override.
- Shared stubMutations protects registration plus index capture against control
  clears across API instances. The old control-create/control-clear finding in
  hardener/QA predates this mutex. Direct engine/registry/reset mutation does
  not participate; embedding callers must coordinate. The global mutex also
  serializes unrelated engines. The concurrency regression runs 100 rounds
  with channel starts but does not force every possible interleaving.
- Host server.go still returns 404 for bounded control requests instead of
  delegating to this API; its oversized-control branch returns 413. Package
  evidence does not establish integrated endpoint availability. Decoder-based
  overflow classification does not establish body admission for endpoints that
  do not read their body. No host behavior was changed in this review.
- Hardener/QA headers still bind 26df167d..., and existing reports are from older
  candidates. Those artifacts require their own refresh; no current stage-chain
  acceptance follows from their historical results or freshness labels.

## Executed evidence

Go commands used GOCACHE=/private/tmp/fj020-revision-cache. No wider access or
additional writable roots were requested.

- ./scripts/agent-context FJ-020: exit 0; scope and acceptance loaded.
- gofmt -l internal/control/api.go internal/control/handlers.go internal/control/handlers_test.go: exit 0, empty output; no formatting write needed.
- go test ./internal/control -count=1: exit 0, ok (0.288s).
- go test -race ./internal/control -count=1: exit 0, ok (1.289s), no race reports.
- go test -overlay /private/tmp/fj020-final-qa/overlay.json ./internal/control -run TestIndependentQA -count=1 -v: exit 0, ok (0.535s); lifecycle, metadata_order, missing_when, null_priority, raw_without_headers, duplicate_key, body_limit and semver all succeed. No repository tests added or changed.
- ./scripts/verify-candidate: exit 1; 17 passed, 2 failed, 1 skipped, 0 not_applicable. Full test and race integration checks cannot bind 127.0.0.1:0 (operation not permitted). Role-pack validation skips because PyYAML is unavailable. Build, vet and guards succeed. Log: /private/tmp/fj020-cleaner-413-verify.log.
- ./scripts/verify-candidate FJ-020: not run because task mode writes prohibited reports. Taskless verification does not validate task-stage chaining. No implementation edits followed failed verification.
- ./scripts/candidate-fingerprint candidate: exact unchanged input/output binding above.
- ./scripts/candidate-fingerprint task .agent/work/FJ-020/state.json: exact taskFingerprint above.
- git rev-parse HEAD: exact full gitHead above.
- git diff --cached --exit-code: exit 0; no staged files.

The available tool catalog contains no contact_supervisor tool, so no supervisor
coordination or approval is claimed. This refresh yields repository-specific
evidence only; no transferable contribution was identified. Independent reviewer
assessment remains required. This artifact records observations and execution
evidence without a stage or product acceptance verdict.
