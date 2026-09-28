---
stage: cleaner
task: FJ-010
inputFingerprint: dfbe154ac2a066ce29e4e356e521685ab5e42f972a5bda3b1f1c64cf1e7d7775
outputFingerprint: dfbe154ac2a066ce29e4e356e521685ab5e42f972a5bda3b1f1c64cf1e7d7775
taskFingerprint: 3d07518a90863c5c849f81709c5ec1d04e4a3a37d6e754a76cdee53f7700df3a
gitHead: 50fc652cb7b6b30b16b9d4e9e92b015ebd88facf
generatedAt: 2026-09-28T18:24:08Z
author: subagent/worker-FJ-010-cleaner-refresh
---

# Cleaner — FJ-010

## Scope reviewed

Reviewed the coder artifact and the existing configuration loader, schema,
and validator under `internal/config` against §§12, 17, and 38. This is a
scope-only evidence refresh: no production files were edited and no structural
cleanup was performed. C-CFG-011 is excluded from this loader review; its
CLI/file/default precedence responsibility moved to FJ-031.

## Observations

- **C-CFG-001:** The document-presence check requires `schemaVersion` and
  accepts only integer `1`.
- **C-CFG-002:** Strict JSON decoding and the document checks reject unknown
  keys in the project-owned configuration envelopes.
- **C-CFG-003:** `state`, question content, and `then.raw.body` remain open
  JSON containers, while project-owned envelopes retain fail-closed checks.
- **C-CFG-004:** `jev` is normalized to `jev/v1` before duplicate detection,
  and unknown profiles are rejected.
- **C-CFG-005:** Static stub IDs are required to be non-empty and unique.
- **C-CFG-006:** A stub response requires exactly one of `answers`,
  `sequence`, or `raw`.
- **C-CFG-007:** Omitted server, mode, compatibility, limits, models, and
  stubs receive the §12.2 defaults.
- **C-CFG-008:** Validation permits only `strict` mode.
- **C-CFG-009:** JSON and YAML are normalized to one JSON decoding boundary;
  YAML document boundaries and flow documents are handled at load time.
- **C-CFG-010:** Validation covers server and limit ranges, model field shape,
  unique model names, and the required non-empty model set.
- **C-SEQ-007:** Expectation validation permits one exact count or a bounded
  range, with non-negative counts and ordered bounds.

The candidate fingerprint remains unchanged because this refresh changes only
this evidence artifact; the cleaner output therefore equals the coder output.
