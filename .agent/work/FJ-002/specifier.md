---
stage: specifier
task: FJ-002
inputFingerprint: 0afc05eb6de7cae7b203f94c226cd89bfa301b59f7162399001cf1a5cbf04556
outputFingerprint: 0afc05eb6de7cae7b203f94c226cd89bfa301b59f7162399001cf1a5cbf04556
taskFingerprint: 97f85d62e553a71b7ad6766e5c74c996b012f9bb65b018bdbf5776b61111cda0
gitHead: c4192d0
generatedAt: 2026-09-26T07:48:37Z
---

# Specifier — FJ-002

## What is observably delivered

Three committed, reviewable artifacts plus the index that describes them. No
runtime behavior changes, because nothing runs from them yet. "Observable" here
means a reviewer and a later implementation agent can read the frozen contracts
without reinterpreting prose.

| Artifact | Frozen by |
|---|---|
| `spec/config.schema.json` | §38, with defaults and aliases from §12 |
| `spec/control-api.openapi.yaml` | §41 |
| `spec/compat/jev-v1/openapi.snapshot.json` | §6, §39 |
| `spec/README.md` (update) | §45 index; the file already exists and currently marks all three artifacts "reserved" |

## The one judgment that needs stating up front

§6 records a *captured* upstream document: snapshot date 2026-09-25, source
`https://api.typesafe.ai/openapi.json`, observed API description version `0.2.0`,
OpenAPI `3.1.0`. This work item forbids contacting the network, so
`openapi.snapshot.json` cannot be that capture. It is a **reconstruction** of the
surface §6 and §39 freeze.

That distinction must survive into the file itself. A future reader who finds
`openapi.snapshot.json` next to a specification section citing a real capture
date and URL will otherwise assume a verbatim upstream capture and trust it as
evidence about live TypeSafe behavior. That is a false compatibility claim, and
§6.2 makes drift a versioned-profile problem rather than something a reader
should have to detect.

The artifact therefore records its own provenance explicitly: derived from
`spec/fake-jev-technical-spec-v2.md` §6 and §39, not fetched. This is an honesty
requirement on the artifact, not a new public behavior, and it is within this
item's `allowedFiles`.

## What the schema may and may not encode

JSON Schema can express C-CFG-001, -002, -003, -006, -007, -008, and -010
directly. It **cannot** express the rest, and pretending otherwise would make the
artifact lie:

| Criterion | Not expressible in JSON Schema | Why |
|---|---|---|
| C-CFG-004 | `jev` alias normalization, profile-enablement check | cross-reference between two fields; requires `if/then` over the whole document plus a known profile set |
| C-CFG-005 | duplicate stub IDs | uniqueness across array members |
| C-CFG-009 | YAML and JSON equivalence | a property of the loader, not of one encoding |
| C-CFG-011 | CLI > file > default precedence | involves a source outside the document |

Those five are recorded in the schema as `description` prose and remain the
responsibility of the loader (FJ-010). An artifact that silently appears to
enforce them is worse than one that says where enforcement lives, so each
non-expressible criterion is named in the schema next to the field it governs.

Numeric rules that JSON Schema *can* carry are carried rather than deferred:
`priority` as a 32-bit signed integer, `port` 0..65535, and every §38.3 limit
range. `additionalProperties: false` is what makes C-CFG-002 machine-checkable,
and the three open containers in §38.8 must deliberately *omit* it.

## Control API encoding notes

§41 is unusually precise about bodies, and two rules are easy to lose in a
schema:

- `204 No Content` responses carry no body and no content type. §11 exempts them
  explicitly; representing them with a JSON schema would be wrong.
- Failure ordering (§41.9) is a response contract but not a type constraint.
  Ordering is recorded in the endpoint description so it cannot be dropped when
  only the schema is read.

The eight failure codes, the eight interaction outcomes, and the eight
fake-server error codes are three separate enumerations in three separate
sections (§41.9, §41.7, §21.2). They are deliberately kept as three named
enumerations in the OpenAPI document. Collapsing them into one shared list would
be a convenience that destroys a distinction the specification maintains, and
would make `journal_full` — a verification state that has no interaction outcome
— look like an inconsistency rather than a deliberate exception.

## Scope kept out

No Go code, no YAML loader, no golden fixtures. The schemas are derived records
of prose; the implementation that enforces them is FJ-010 onward. Nothing in this
item may be used to justify a new configuration key, endpoint, or default: the
artifacts describe §38 and §41, they do not extend them.

## Traces

- C-CFG-001, C-CFG-002, C-CFG-003, C-CFG-006, C-CFG-007, C-CFG-008, C-CFG-010 —
  §38.1, §38.3, §38.4, §38.7, §38.8, §12.2, §12.3, §14. Encoded as machine
  rules in `spec/config.schema.json`.
- C-CFG-004, C-CFG-005, C-CFG-009, C-CFG-011 — §38.1, §11.2, §12, §12.1. Recorded
  as named non-expressible rules in the schema; enforcement belongs to FJ-010.
- C-CTRL-001 … C-CTRL-011 — §41.2 through §41.10 plus §11. Encoded as paths,
  operations, status codes, and schemas in `spec/control-api.openapi.yaml`.
- C-QUAL-006 — §45. The three artifacts plus an updated `spec/README.md` that no
  longer describes them as reserved.
- C-ARCH-006 — §19.1, §37. Recorded in the snapshot and control documents as
  the `127.0.0.1` default, so the default is visible in the artifacts rather
  than only in prose.
