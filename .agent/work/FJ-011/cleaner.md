---
stage: cleaner
task: FJ-011
inputFingerprint: 7d24ad5a06f13ec9404686209d64ad43704aa9c8383e2c21738e876c15aa0e5a
outputFingerprint: d6258777f129a21d33ee48b49d17604f2bc4e4fa260c8321ec8c5adf7301ca39
taskFingerprint: b788644f51a266d22e3a0d6efc84fe854e22b1fba53fbc99cc44c97f32ab94bb
gitHead: 4d6dff0
generatedAt: 2026-09-28T19:48:40Z
author: subagent/worker-FJ-011-cleaner
---

# Cleaner — FJ-011

## Structure

Reviewed the coder artifact and the five allowed engine files against §17.3,
§40, and the provider-neutral boundary. The only justified cleanup was in
`internal/engine/matcher.go`: `Registry.Select` and `Registry.Matching` had
identical candidate collection and total-order sorting. Their shared work is
now in the private `Registry.matching` helper; the exported methods retain
separate nil checks and read-lock ownership, and `Select` still returns only
the first ordered candidate.

No changes were made to the exchange, stub, registration, JSON equality, or
test interfaces. The engine still has no forbidden HTTP, process, CLI,
control-API, or compatibility imports, and the cleanup introduces no provider
knowledge or new public surface.

## Behavior evidence

- Candidate filtering remains profile equality followed by matcher evaluation.
- Candidate ordering remains priority descending, then registration index
  ascending; the helper sorts the copied candidate slice while the registry
  read lock is held.
- `Select` and `Matching` continue to return the same empty, selected, and
  ordered results; the registration slice is not exposed for mutation.
- The coder test suite remains unchanged and covers matching, provenance,
  registration indices, duplicate rejection, and ordering vectors.

## Validation evidence

- `gofmt -w internal/engine/*.go` — completed.
- `go test ./...` — all repository packages passed.
- `go test -race ./...` — all repository packages passed.
- `go vet ./...` — passed.
- `go test -run=^$ ./internal/engine` — passed.
- Forbidden-import guard over `internal/engine` — no matches.
- `./scripts/verify-candidate FJ-011` — product tests, vet, build, architecture,
  lint, trace, fuzz, and race checks passed; the run also reported the expected
  not-yet-present hardener and QA artifacts. Its candidate fingerprint was
  `d6258777f129a21d33ee48b49d17604f2bc4e4fa260c8321ec8c5adf7301ca39`.

## Observations

- The cleanup changes one private structure seam only; no behavior-facing
  method signature or data ownership contract moved.
- `sort.SliceStable` remains intentional at the shared ordering seam: it
  preserves the existing deterministic comparator behavior while registration
  indices provide the required tie order.
- `internal/engine` remains filesystem-free and provider-neutral under §17.3.
