---
stage: specifier
task: FJ-010
inputFingerprint: dfbe154ac2a066ce29e4e356e521685ab5e42f972a5bda3b1f1c64cf1e7d7775
outputFingerprint: dfbe154ac2a066ce29e4e356e521685ab5e42f972a5bda3b1f1c64cf1e7d7775
taskFingerprint: 3d07518a90863c5c849f81709c5ec1d04e4a3a37d6e754a76cdee53f7700df3a
gitHead: 50fc652cb7b6b30b16b9d4e9e92b015ebd88facf
generatedAt: 2026-09-28T18:20:24Z
author: subagent/worker-FJ-010-specifier
---

# Specifier — FJ-010

## Observable behavior

The configuration boundary must accept JSON and YAML documents that deserialize to the same logical configuration model. It must require integer `schemaVersion: 1`, reject unknown project-owned keys, normalize `jev` to `jev/v1` before duplicate/profile checks, reject unknown or disabled profiles, reject duplicate static stub IDs, and require exactly one of `then.answers`, `then.sequence`, or `then.raw`. It must validate the §38 ranges and model rules, and accept only `strict` mode.

When omitted, §12.2 supplies the server, mode, compatibility, limits, default model, and empty-stub defaults. `expect` is either `exactly` or the `atLeast`/`atMost` range form, with non-negative integer counts and an ordered range when both bounds are present. `state`, question content, and `then.raw.body` remain open JSON containers as specified; the project-owned envelopes remain fail-closed on unknown keys.

## Feasibility boundary

JSON decoding can use the Go standard library from the allowed configuration files. Go's standard library has no YAML decoder, so one mature YAML dependency may be added through the now-allowed `go.mod` and `go.sum` files to implement C-CFG-009. No custom YAML parser or additional dependency is proposed.

## Traces

- C-CFG-001 — §38.1: `schemaVersion` is required and exactly integer `1`.
- C-CFG-002 — §12.3 and §38.8: unknown keys are rejected in each named project-owned envelope.
- C-CFG-003 — §38.8: `state`, question content, and `then.raw.body` are open JSON containers.
- C-CFG-004 — §38.1 and §38.5: normalize `jev` to `jev/v1` before duplicate detection and reject unknown profiles.
- C-CFG-005 — §11.2 and §15.2: duplicate static stub IDs are invalid configuration.
- C-CFG-006 — §12.6 and §38.7: a stub has exactly one response form among `answers`, `sequence`, and `raw`.
- C-CFG-007 — §12.2: omitted sections receive the built-in defaults.
- C-CFG-008 — §12.2 and §14: `strict` is the only valid mode.
- C-CFG-009 — §12 and §38: YAML and JSON deserialize to one logical schema; one mature YAML dependency may be added under §17.2 through the allowed module files.
- C-CFG-010 — §38.3 and §38.4: limit ranges, exact model keys, unique names, and at least one model after defaults are enforced.
- C-SEQ-007 — §12.7: `expect` accepts exactly one exact count or a valid bounded range with non-negative counts.

## Explicit non-goals

- Do not start a server.
- Do not wire the engine.
- Do not add profile-specific answer validation beyond envelope shape.
- Do not add more than one mature YAML dependency or implement a custom YAML parser.
- Do not declare a gate, implementation, or acceptance verdict in this artifact.
