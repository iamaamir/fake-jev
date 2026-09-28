---
stage: hardener
task: FJ-010
inputFingerprint: dfbe154ac2a066ce29e4e356e521685ab5e42f972a5bda3b1f1c64cf1e7d7775
outputFingerprint: dfbe154ac2a066ce29e4e356e521685ab5e42f972a5bda3b1f1c64cf1e7d7775
taskFingerprint: 3d07518a90863c5c849f81709c5ec1d04e4a3a37d6e754a76cdee53f7700df3a
gitHead: 50fc652cb7b6b30b16b9d4e9e92b015ebd88facf
generatedAt: 2026-09-28T18:27:04Z
author: subagent/worker-FJ-010-hardener-refresh
---

# Hardener — FJ-010

## Scope and hardening actions

This is a corrective evidence refresh only. No production files were changed;
the candidate fingerprint therefore remains unchanged from the cleaner artifact.
C-CFG-011 is out of scope for this loader review: CLI/file/default precedence
moved to FJ-031, which owns CLI flag handling.

The hardening review covered the loader boundaries in `internal/config`:

- YAML and JSON are normalized through a single JSON decoding boundary; empty,
  trailing, multiple-document, and duplicate-key inputs fail closed.
- Strict decoding rejects unknown keys in project-owned envelopes, while the
  specified open JSON containers (`state`, question content, and
  `then.raw.body`) are not recursively constrained.
- Document-shape checks reject null or wrong-type project-owned objects and
  arrays before model validation; raw bodies remain optional as specified.
- Validation keeps response-form exclusivity, non-empty sequences, status and
  limit ranges, non-negative usage and expectation counts, and ordered
  expectation bounds explicit.
- `LoadFile` wraps read failures with the path, and load/validation failures do
  not return a configuration.

## Contract evidence

- **C-CFG-001:** Requires `schemaVersion` and accepts only integer `1`.
- **C-CFG-002:** Strict decoding rejects unknown keys at the top level and in
  project-owned server, limits, stub, when, then, and expect envelopes.
- **C-CFG-003:** Keeps `state`, question content, and `then.raw.body` open JSON
  containers while retaining fail-closed checks for owned envelopes.
- **C-CFG-004:** Normalizes `jev` to `jev/v1` before duplicate/profile checks
  and rejects unknown profiles.
- **C-CFG-005:** Rejects empty and duplicate static stub IDs.
- **C-CFG-006:** Requires exactly one of `answers`, `sequence`, or `raw` for a
  stub response and rejects incompatible raw siblings.
- **C-CFG-007:** Applies the §12.2 defaults for omitted server, mode,
  compatibility, limits, models, and stubs.
- **C-CFG-008:** Accepts only `strict` mode.
- **C-CFG-009:** Loads JSON and YAML into the same model and rejects malformed,
  trailing, or multiple YAML documents.
- **C-CFG-010:** Enforces server and limit ranges, model field shape, unique
  model names, and a non-empty model set.
- **C-SEQ-007:** Accepts one exact count or one bounded range, with
  non-negative counts and `atLeast <= atMost`.

## Validation evidence

- `go test ./internal/config` — passed.
- `go test -race ./internal/config` — passed.
- `go vet ./internal/config` — passed.
- `git diff --check` — passed.
- Candidate fingerprint: `dfbe154ac2a066ce29e4e356e521685ab5e42f972a5bda3b1f1c64cf1e7d7775`.

## Residual risks

- `Load` accepts an already materialized byte slice and `LoadFile` uses
  `os.ReadFile`; the specification defines no configuration-size cap, so an
  untrusted caller must bound input size before loading.
- YAML timestamp scalar normalization remains dependent on `yaml.v3`; callers
  requiring exact string semantics for date fields should quote them.
- Dependency vulnerability status still requires a repository scanner or an
  external advisory database; none is available in this task surface.
