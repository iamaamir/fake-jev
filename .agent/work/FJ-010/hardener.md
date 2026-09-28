---
stage: hardener
task: FJ-010
inputFingerprint: 6c389bfe4202fbcd2b54d9092ec6bf4697ea4809bb10553a5a30bf56eebacf12
outputFingerprint: dfbe154ac2a066ce29e4e356e521685ab5e42f972a5bda3b1f1c64cf1e7d7775
taskFingerprint: ca4c08f959e903a8b93b1b11560b371a8079733d4442f2c030e42f0529ce2885
gitHead: 3a8aea7f914924e63c28346323bf68bc6ec7fc43
generatedAt: 2026-09-28T15:38:00Z
author: subagent/worker-FJ-010-hardener
---

## Hardening actions

- Changed YAML decoding to use a streaming decoder and reject trailing or multiple YAML documents. This closes the prior first-document-only behavior while retaining JSON duplicate-key rejection and accepting valid YAML flow mappings/sequences at the `{`/`[` boundary.
- Added fail-closed null/type-boundary checks for project-owned scalar, object, array, expectation, usage, matcher, response, and header fields. Open JSON containers remain open, including `then.raw.body`.
- Made raw response bodies optional as required by §39.7, while retaining status range and header validation.
- Rejected empty response sequences and enforced required usage token members when a usage object is supplied.
- Added focused regression tests for YAML document boundaries, YAML flow syntax, omitted raw bodies, null structured values, and empty sequences.

## Contract evidence

- `schemaVersion`, strict mode, profile normalization/unknown-profile handling, duplicate static IDs, response-form exclusivity, integer/range checks, and §12.2 defaults remain validated through the existing JSON boundary and model validation.
- `then.raw.body` is not inspected for project-owned keys; arbitrary JSON remains an explicitly open container under §38.8.
- YAML aliases and duplicate mapping keys remain parser-rejected by `gopkg.in/yaml.v3`; JSON object keys are scanned recursively before strict decoding.
- Configuration errors are returned before a configuration is returned; file read failures retain path context and wrapped causes.

## Validation evidence

- `gofmt -w internal/config/load.go internal/config/validate.go internal/config/validate_test.go`
- `go test ./...` — passed.
- `go test -race ./...` — passed.
- `go vet ./...` — passed.
- Focused configuration regression tests — passed.
- `git diff --check` — passed.
- `go list -m -json gopkg.in/yaml.v3` — dependency resolved as v3.0.1 with recorded module checksums.
- Candidate fingerprint after hardening: `dfbe154ac2a066ce29e4e356e521685ab5e42f972a5bda3b1f1c64cf1e7d7775`.

## Residual risks

- `Load` receives an already materialized byte slice and `LoadFile` uses `os.ReadFile`; no configuration-size cap is defined by §§12/38, so callers must bound file/input size before loading if hostile configuration sources are introduced.
- YAML timestamp scalar normalization remains dependent on `yaml.v3` scalar typing; date fields should be quoted when exact string semantics are required.
- Dependency vulnerability status requires the repository's dependency scanner or an externally maintained advisory database; no such scanner is present in the allowed task surface.
