---
stage: coder
task: FJ-020
inputFingerprint: 4f811c765e1cf47c069ff957b82979f307f7376776475789c31e8183d700b178
outputFingerprint: 8dacde273d16cf959da358666963b3afdbec066ac5dd4ebea125c094c4796740
taskFingerprint: bf0432ef36830af414bf83f372280b27549851174ee9ee40d083af3e9fb25d42
gitHead: b198957ebbd5beebffa3cf118d89fb55f3c8efc0
generatedAt: 2026-09-29T15:14:25Z
author: worker/FJ-020-coder-chain-correction
---

# Narrow control overflow correction

## Metadata correction (chain link)

`inputFingerprint` above was corrected from
`841886beb205c4e91a3e10e3fc5cf755366aa9cfdb90df64d8e4313b0e5b429d` to
`4f811c765e1cf47c069ff957b82979f307f7376776475789c31e8183d700b178`.
The former value had recorded an intermediate candidate from an earlier coder
revision; the latter is the specifier stage `outputFingerprint`, so the stage
chain link now matches. `outputFingerprint`, `taskFingerprint`, `gitHead` and the
evidence below are unchanged.

Read the current agent-context packet, cleaner finding, specification §§19.3,
19.4 and 43.7. Only handlers.go, handlers_test.go and this artifact changed in
this revision. State, reports, api.go, engine, host, other stage artifacts and
staging were not edited. Fingerprints and full gitHead above are command-derived.

createStub distinguishes *http.MaxBytesError using errors.As and returns HTTP
413 with exactly fake_jev_payload_too_large and message
"Request body exceeds the configured limit." Malformed valid-size JSON retains
the exact 400 fake_jev_bad_control_request envelope. Existing MaxBytesReader
bounds and the shared control mutation mutex/index capture remain unchanged.

Updated TestControlBodyLimit expects the exact 413 envelope for leading/trailing
whitespace overflow and large bodies, with both known and unknown content
lengths. Below/exact-limit success and malformed/extra-JSON 400 cases remain.
All rejected cases assert no registered stub; every case checks JSON content
type, at most limit+1 bytes read, journal and verification isolation.
TestControlConcurrentCreateClear remains unchanged and passes under race testing.

## Validation

Go commands used GOCACHE=/private/tmp/fj020-revision-cache.

- gofmt -w internal/control/handlers.go internal/control/handlers_test.go: exit 0.
- gofmt -l on those files: exit 0, empty output.
- go test ./internal/control -count=1: exit 0, ok (0.593s).
- go test -race ./internal/control -count=1: exit 0, ok (1.646s).
- go vet ./...: exit 0, no diagnostics.
- go run ./cmd/guard arch: exit 0, no findings, 47 files/9 packages.
- go run ./cmd/guard lint: exit 0, no findings, 47 files/9 packages.
- go run ./cmd/guard trace: exit 0, no findings, 0 active/17 covered IDs.
- go run ./cmd/guard fuzz: exit 0, no findings, 0 targets (no fuzz coverage claimed).
- ./scripts/verify-candidate: exit 1; 17 passed, 2 failed, 1 skipped.
  Full test and race integration checks cannot bind 127.0.0.1:0 (operation not
  permitted); role-pack validation skips because PyYAML is unavailable.
  Log: /private/tmp/fj020-413-verify.log. No implementation edits followed failure.
- ./scripts/verify-candidate FJ-020: not run; task mode writes prohibited reports.
- candidate-fingerprint candidate, candidate-fingerprint task and git rev-parse
  HEAD: exit 0; exact bindings above.
- git diff --cached --exit-code: exit 0; no staged files.

## Remaining boundaries

Full verification remains nonzero; independent reviewer approval and later-stage
artifact refresh are required. Host delegation and direct engine/registry/reset
mutation outside the control mutex remain unchanged and outside this correction.
The concurrency test does not force every possible interleaving. The correction
classifies decoder-reported overflow; this is not a redesign of host body
admission or endpoints that do not decode bodies.

No contact_supervisor tool was exposed in the available catalog, so no supervisor
coordination or approval is claimed. Contribution reflection found only a routine
repository-specific correction; no external contribution was warranted.
