---
stage: coder
task: FJ-010
inputFingerprint: dfbe154ac2a066ce29e4e356e521685ab5e42f972a5bda3b1f1c64cf1e7d7775
outputFingerprint: dfbe154ac2a066ce29e4e356e521685ab5e42f972a5bda3b1f1c64cf1e7d7775
taskFingerprint: 3d07518a90863c5c849f81709c5ec1d04e4a3a37d6e754a76cdee53f7700df3a
gitHead: 50fc652cb7b6b30b16b9d4e9e92b015ebd88facf
generatedAt: 2026-09-28T18:23:08Z
author: subagent/worker-FJ-010-coder
---

## Changes

This corrective evidence refresh makes no production-file changes. The existing
`internal/config` implementation remains the configuration loader and validator
for the scoped criteria.

## Evidence

- **C-CFG-001:** Requires integer `schemaVersion: 1` and rejects missing,
  non-integer, or other values.
- **C-CFG-002:** Strict decoding rejects unknown keys in project-owned
  configuration envelopes, including nested server, limits, stub, response,
  and expectation objects.
- **C-CFG-003:** `state`, question content, and `then.raw.body` remain open JSON
  containers while project-owned envelopes stay fail-closed.
- **C-CFG-004:** Normalizes `jev` to `jev/v1` before duplicate/profile checks
  and rejects unknown profiles.
- **C-CFG-005:** Rejects duplicate static stub IDs.
- **C-CFG-006:** Requires exactly one of `then.answers`, `then.sequence`, or
  `then.raw`.
- **C-CFG-007:** Applies the §12.2 defaults for omitted server, mode,
  compatibility, limits, models, and stubs.
- **C-CFG-008:** Accepts only `strict` mode.
- **C-CFG-009:** Loads equivalent JSON and YAML documents into the same logical
  configuration model, including YAML document-boundary checks.
- **C-CFG-010:** Enforces limit ranges, exact model fields, unique model names,
  and at least one model after defaults.
- **C-SEQ-007:** Accepts valid exact and bounded expectation forms and rejects
  mixed, negative, missing, or inverted forms.

## Validation evidence

- `go test ./internal/config` — passed.
- `go test -race ./internal/config` — passed.
- `go vet ./internal/config` — passed.
- `./scripts/candidate-fingerprint candidate .` —
  `dfbe154ac2a066ce29e4e356e521685ab5e42f972a5bda3b1f1c64cf1e7d7775`.
- `go test ./...` was also attempted; the repository-wide command is blocked by
  pre-existing missing dependencies in `agent/skills/golang-cli/assets/examples`,
  while the configuration package checks above pass.
