---
stage: cleaner
task: FJ-001
inputFingerprint: 7a30b543583fc4e4849e2b615c5256f499e0d4b92d555bc81dd0166e02860745
outputFingerprint: 7a30b543583fc4e4849e2b615c5256f499e0d4b92d555bc81dd0166e02860745
taskFingerprint: cc94ff6af6bccd0fa2ce59f0742e09743b6bf854d5c48af4097e7194f000ee93
gitHead: 13172c0
generatedAt: 2026-09-26T07:13:55Z
---

# Cleaner — FJ-001

## Review performed

Read the four new files against specification §17.2 (dependency policy), §17.3
(engine purity), §15.5 (`version` output), §15.6 (no heavy CLI framework) and
§42.1 (exit codes). No code was changed: nothing found warranted a rewrite.

## Findings

- **Complexity tooling.** The configured analyzer is absent; this item carries
  the recorded bootstrap exemption rather than a green result. Inspected by
  hand instead: the largest function is `buildRevision`, twelve statements and
  one loop over build settings. There is no branch worth naming and no
  duplication — `versionLines` is a literal list precisely so that adding a
  required §15.5 field is a one-line, reviewable change.
- **No speculative abstraction.** `BuiltInProfiles()` returns a fresh slice per
  call rather than a shared package-level slice, so a caller cannot mutate
  another caller's view. The alternative (a package var) would be shared
  mutable state for no benefit. `ControlAPIVersion` and
  `ConfigSchemaVersion` are constants and stay shared.
- **Duplication.** None. `Run` dispatches in one switch; `runVersion` is the
  only writer.
- **Dead code.** None. `exitVerify` is defined but unused until FJ-032 adds
  `verify`; it is kept deliberately so the §42.1 table is present in one place
  and the constant is not re-invented later. Noted here so a future reader does
  not read it as an oversight.
- **Naming and layout.** Package `cli` sits at `internal/cli` per §17; files are
  `dispatch.go` (argument routing) and `version.go` (the one subcommand), which
  is the split the §17 layout implies for `serve.go`/`run.go`/`verify.go` later.

## What remains

Nothing for this item. FJ-030/031/032 add the remaining subcommands into
`internal/cli` under the same split.

## Re-evidenced 2026-09-26T07:13:55Z

FJ-048 added `/fake-jev` to `.gitignore`, which changes the workspace candidate
fingerprint and therefore invalidates this item's earlier evidence. The stage was
re-run: the four Go files are byte-identical to the original pass, so the findings
above stand unchanged, and `go test`, `go vet`, and `go build ./cmd/fake-jev` were
re-executed against them. Only the recorded fingerprints differ.
