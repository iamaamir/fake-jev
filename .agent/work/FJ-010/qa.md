---
stage: qa
task: FJ-010
inputFingerprint: dfbe154ac2a066ce29e4e356e521685ab5e42f972a5bda3b1f1c64cf1e7d7775
outputFingerprint: dfbe154ac2a066ce29e4e356e521685ab5e42f972a5bda3b1f1c64cf1e7d7775
taskFingerprint: ca4c08f959e903a8b93b1b11560b371a8079733d4442f2c030e42f0529ce2885
gitHead: 3a8aea7f914924e63c28346323bf68bc6ec7fc43
generatedAt: 2026-09-28T15:41:58Z
author: subagent/worker-FJ-010-qa
---

# QA — FJ-010

## Scope and method

Exercised the configuration loader and validator independently against every
listed configuration/expectation criterion. The focused probe was transient
(`internal/config/qa_probe_test.go`), removed after execution; no production
file or persistent test file was changed. Existing configuration tests and the
hardener predecessor artifact were also reviewed. The normative references were
§§12 and 38 and the acceptance catalog entries C-CFG-001..011 and C-SEQ-007.

## Criteria exercised and observations

- **C-CFG-001:** Missing, zero, and non-integer `schemaVersion` inputs were
  rejected; integer `1` was accepted.
- **C-CFG-002:** Unknown fields were rejected at top level, `server`,
  `limits`, stub envelope, `when`, `then`, and `expect` boundaries.
- **C-CFG-003:** Arbitrary nested state, answer content, and raw provider body
  values were accepted while response-form ownership remained validated.
- **C-CFG-004:** `jev` normalized to `jev/v1`; alias/concrete duplicates and
  unknown profiles were rejected.
- **C-CFG-005:** Duplicate static stub IDs were rejected.
- **C-CFG-006:** Single `raw`, `answers`, and `sequence` forms were accepted;
  multiple response forms were rejected.
- **C-CFG-007:** Omitted sections produced the §12.2 host, port, mode,
  compatibility, limits, default model, and empty-stub defaults.
- **C-CFG-008:** `strict` was accepted and `permissive` was rejected.
- **C-CFG-009:** Equivalent JSON and YAML documents loaded to deeply equal
  logical configurations; `LoadFile` was also exercised with YAML.
- **C-CFG-010:** Lower/upper limit violations, empty models, duplicate model
  names, and unknown model keys were rejected; a valid model object was
  accepted.
- **C-CFG-011:** An environment variable did not override the built-in strict
  default; file loading was exercised. No CLI/config merge seam exists in the
  current allowed configuration surface, so explicit CLI-over-file precedence
  could not be exercised here.
- **C-SEQ-007:** Valid `exactly`, `atLeast`, `atMost`, and bounded-range forms
  were accepted. Missing forms, mixed exact/range forms, negative counts, and
  inverted ranges were rejected.

## Commands and results

- `go test ./...` — passed.
- `go test -race ./...` — passed.
- `go vet ./...` — passed.
- `go test ./internal/config -run 'TestQA' -count=1 -v` — passed; all probe
  subtests for C-CFG-001..011 and C-SEQ-007 passed.
- `./scripts/candidate-fingerprint candidate .` — produced
  `dfbe154ac2a066ce29e4e356e521685ab5e42f972a5bda3b1f1c64cf1e7d7775`.
- `./scripts/verify-candidate FJ-010` — passed; 48 checks passed, 0
  failed, 3 skipped, 0 not applicable. Report:
  `.agent/reports/FJ-010/report-20260928T154315Z.json`.

## Residual risks and limits

- C-CFG-011's explicit CLI-flag-over-file precedence remains unobservable
  because CLI configuration merging is outside the current allowed files and
  no merge API is present.
- The loader has no input-size limit defined by §§12/38; hostile callers would
  need to bound input before calling it.
- YAML scalar date typing/normalization remains delegated to `yaml.v3`; exact
  date strings should be quoted by configuration authors.

No acceptance or stage verdict is recorded here.
