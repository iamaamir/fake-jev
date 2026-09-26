---
stage: qa
task: FJ-002
inputFingerprint: 71e7458cdb2181ae63b22be2d881a8fa128cd1cf2ef5fc9072649feadb684652
outputFingerprint: 71e7458cdb2181ae63b22be2d881a8fa128cd1cf2ef5fc9072649feadb684652
taskFingerprint: 97f85d62e553a71b7ad6766e5c74c996b012f9bb65b018bdbf5776b61111cda0
gitHead: c4192d0
generatedAt: 2026-09-26T07:52:04Z
---

# QA — FJ-002

## Class and scope

`metadata` class: the four `allowedFiles` are all under `spec/`, so the Go
checks are `not_applicable` rather than skipped, and Hardener is not required.

## What was exercised

There is no harness for contract artifacts, so the checks were run directly.
**Limitation stated plainly: no JSON Schema instance validation was performed.**
No `jsonschema` implementation is available in this environment, so the schemas
were checked structurally, not executed. Structural checking cannot catch a
constraint that is well-formed but wrong — for example a `oneOf` that is
satisfiable in two arms at once. Executing these schemas against fixtures is
FJ-010's acceptance work and FJ-046's harness; this item does not claim it.

| Check | Result |
|---|---|
| `spec/config.schema.json` parses as JSON | pass |
| `spec/control-api.openapi.yaml` parses as YAML | pass |
| `spec/compat/jev-v1/openapi.snapshot.json` parses as JSON | pass |
| every `$ref` resolves to a defined target (all three artifacts) | pass |
| no unused `$defs` / component schemas | pass |
| every `required` name exists in sibling `properties` | pass |
| all 9 §41.1 endpoints present with correct methods | pass |
| all 8 §21.2 fake failure codes appear in the snapshot | pass |
| all 4 §39.3 422 envelope types appear | pass |
| `journal_full` in verification codes, absent from interaction outcomes | pass |
| question types identical across config schema, control API, snapshot | pass |
| profile IDs identical across config schema and control API | pass |
| every limit: range and default identical in config schema and control API | pass |
| network access during authoring | none |
| `spec/README.md` no longer marks the artifacts reserved | pass |

## Acceptance criteria

- **C-QUAL-006** — the three required artifacts are committed, parse, and are
  internally consistent; `spec/README.md` indexes them. Pass.
- **config schema encodes C-CFG-001 … C-CFG-011** — machine-checked for
  C-CFG-001, -002, -003, -006, -007, -008, -010. The remaining five
  (-004 alias normalization, -005 duplicate IDs, -009 YAML/JSON equivalence,
  -011 precedence) are not expressible in JSON Schema; each is named in the
  artifact at the field it governs, with the reason, rather than being implied
  by silence. Criteria-dependent numeric rules in C-JEV-013/-015 are named the
  same way. Pass, with that boundary recorded.
- **control OpenAPI encodes C-CTRL-001 … C-CTRL-011** — all eleven are present:
  seven as operation/status/schema shapes, C-CTRL-006 as the journaled record
  with its `requestBody` rules, C-CTRL-008 as the verify result with fixed
  failure item shape and ordering, C-CTRL-010 as the shared `400` response,
  C-CTRL-011 as the content-type header reused across every JSON response and
  absent from the three `204`s. Pass.
- **snapshot records the frozen §6 surface without contacting the network** —
  pass, and the file says so itself: `provenance:
  "reconstructed-from-specification"`, `networkFetch: false`, plus a leading
  `$comment` stating it is not a capture of the upstream document.

## Spot checks against the specification, read back

Five values were compared to the specification text by eye rather than
mechanically, because they are the ones a schema author transcribes wrong:

| Value | Artifact | Spec |
|---|---|---|
| control API version `v1` | `ControlApiVersion` const | §25.2, §41.2 |
| config schema version `1` | `configSchemaVersion` const | §25.3, §38.1 |
| default model `jev-latest` / `1970-01-01` | `ModelList` example, `models` default | §12.2, §39.1, §44.1 |
| default limits 8388608 / 2097152 / 10000 / 4096 / 5 | `Limits` defaults | §12.2 |
| `gracefulShutdownSeconds` upper bound 60 | both artifacts | §38.3 |

## What a reviewer should press on

- **The snapshot is derived, not captured.** This is the artifact most likely to
  be misread later. It is the correct call — the work item forbids a network
  fetch — but it means the repository has no verbatim record of what upstream
  actually said on 2026-09-25. If a future profile needs real drift detection,
  that capture has to be taken deliberately, with a date and a reviewer, not
  inferred from this file.
- **The five non-expressible criteria have no mechanical enforcement yet.**
  Nothing in this item or the next can catch a duplicate stub ID. That is FJ-010.
- **No instance validation ran.** Stated above, repeated here, because a green
  row in the table above is easy to read as more than it is.
