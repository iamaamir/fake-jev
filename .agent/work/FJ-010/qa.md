---
stage: qa
task: FJ-010
inputFingerprint: dfbe154ac2a066ce29e4e356e521685ab5e42f972a5bda3b1f1c64cf1e7d7775
outputFingerprint: dfbe154ac2a066ce29e4e356e521685ab5e42f972a5bda3b1f1c64cf1e7d7775
taskFingerprint: 3d07518a90863c5c849f81709c5ec1d04e4a3a37d6e754a76cdee53f7700df3a
gitHead: 50fc652cb7b6b30b16b9d4e9e92b015ebd88facf
generatedAt: 2026-09-28T18:29:59Z
author: subagent/worker-FJ-010-qa-refresh
---

# QA — FJ-010

## Scope and method

Reviewed the current state, refreshed specifier/coder/cleaner/hardener artifacts,
acceptance catalog, §§12 and 38, and the `internal/config` loader, schema,
validator, and tests. Exercised only this ticket's acceptance set:
C-CFG-001 through C-CFG-010 and C-SEQ-007. No production files or persistent
tests were changed.

C-CFG-011 (CLI flag > config file > default precedence, with no environment
override) is explicitly out of FJ-010. It moved to FJ-031, which owns CLI flag
handling; no CLI precedence claim is made in this artifact.

## Criteria exercised and observations

- **C-CFG-001:** Missing, non-integer, zero, and non-`1` schema versions were
  rejected; integer `schemaVersion: 1` was accepted.
- **C-CFG-002:** Unknown project-owned keys were rejected at the top level and
  in server, limits, stub, when, then, and expect envelopes. Open raw body and
  other specified JSON containers remained unconstrained.
- **C-CFG-003:** Arbitrary nested state, answer content, and raw provider body
  values remained loadable while owned response envelopes stayed validated.
- **C-CFG-004:** `jev` normalized to `jev/v1` before duplicate/profile checks;
  duplicate aliases/concrete profiles and unknown profiles were rejected.
- **C-CFG-005:** Empty and duplicate static stub IDs were rejected.
- **C-CFG-006:** Single answers, sequence, and raw response forms were accepted;
  missing, multiple, and raw-with-forbidden-siblings forms were rejected.
- **C-CFG-007:** Omitted server, mode, compatibility, limits, model, and stubs
  received the §12.2 defaults.
- **C-CFG-008:** `strict` was accepted and non-strict modes were rejected.
- **C-CFG-009:** Equivalent JSON and YAML documents loaded to deeply equal
  logical configurations; file loading and YAML document-boundary handling
  were exercised.
- **C-CFG-010:** Limit ranges, model field shape, release-date shape, unique
  model names, and the at-least-one-model rule were exercised.
- **C-SEQ-007:** Valid exactly-one and bounded range forms were accepted;
  missing forms, mixed forms, negative counts, and inverted ranges were
  rejected.

## Commands and results

- `go test ./internal/config -count=1` — passed.
- `go test -race ./...` — reached the configuration and product packages, but
  the repository-wide command stopped on pre-existing missing dependencies in
  `agent/skills/golang-cli/assets/examples` (`fatih/color`, `fsnotify`,
  `cobra`, `viper`, and example module imports).
- `go vet ./...` — stopped on the same pre-existing missing dependencies in
  `agent/skills/golang-cli/assets/examples`.
- `./scripts/candidate-fingerprint candidate .` —
  `dfbe154ac2a066ce29e4e356e521685ab5e42f972a5bda3b1f1c64cf1e7d7775`.
- `./scripts/candidate-fingerprint task .agent/work/FJ-010/state.json` —
  `3d07518a90863c5c849f81709c5ec1d04e4a3a37d6e754a76cdee53f7700df3a`.
- `./scripts/verify-candidate FJ-010` — repository/task checks completed with
  49 passed, 0 failed, 3 skipped, and 0 not applicable; the public-surface QA
  tooling row remains skipped because its harness is not implemented.

## Observations and limits

- The hardener input and output fingerprints are unchanged and equal to the
  current candidate fingerprint; this refresh is evidence-only.
- The configuration loader has no specification-defined input-size cap; an
  untrusted caller must bound input before calling it.
- YAML scalar timestamp normalization remains delegated to `yaml.v3`; callers
  needing exact string semantics for date-like values should quote them.

No acceptance or stage verdict is recorded here.
