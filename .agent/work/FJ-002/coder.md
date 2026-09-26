---
stage: coder
task: FJ-002
inputFingerprint: 0afc05eb6de7cae7b203f94c226cd89bfa301b59f7162399001cf1a5cbf04556
outputFingerprint: 71e7458cdb2181ae63b22be2d881a8fa128cd1cf2ef5fc9072649feadb684652
taskFingerprint: 97f85d62e553a71b7ad6766e5c74c996b012f9bb65b018bdbf5776b61111cda0
gitHead: c4192d0
generatedAt: 2026-09-26T07:51:24Z
---

# Coder — FJ-002

## What was written

Four files, all inside `allowedFiles`.

### `spec/config.schema.json` — §38

Draft 2020-12. Twelve `$defs`, all referenced, none unused. The load-bearing
decisions:

- `additionalProperties: false` at the top level and in `server`, `limits`, the
  stub envelope, `when`, `then`, and `expect` — this is what makes C-CFG-002
  machine-checkable rather than aspirational.
- The three open containers of §38.8 deliberately **omit** it: `state`,
  `probabilities`, and `raw.body`. Getting this backwards in either direction is
  the likely bug here — `state` rejecting an arbitrary JSON key would break
  ordinary fixtures, and `raw.body` rejecting provider JSON would defeat the
  escape hatch's purpose.
- `then` uses `oneOf` with three mutually exclusive arms rather than a single
  `oneOf` on required keys, so that "two or more of answers/sequence/raw" fails
  as well as "none of them". `minProperties: 1` catches the empty case that a
  `oneOf` alone would let through.
- `priority` is bounded to signed 32-bit because §38.5 says so, and an
  unbounded integer would silently accept a value the spec calls invalid.
- `expect` uses `oneOf` to enforce that `exactly` never co-occurs with
  `atLeast`/`atMost`, and requires at least one of `atLeast`/`atMost` in the
  range form.

### `spec/control-api.openapi.yaml` — §41

OpenAPI 3.1.0. All nine required endpoints from §41.1 across six paths, with
exact response bodies, the `400`/`409` error shapes, and `204` responses that
declare no body and no content type.

The three enumerations are kept deliberately separate, each with its own schema
and its own note:

- `InteractionOutcome` — 8 values, journaled `outcome`.
- `FakeFailureCode` — 8 values, journaled `error` and data-plane failure bodies.
- `VerificationFailureCode` — **11** values, `/verify` failure `code`.

The count difference is the point. `journal_full` is a verification failure with
no interaction outcome, because the request that discovers it is never journaled
(§43.5). `expect_exactly`, `expect_at_least`, and `expect_at_most` are
verification-only. Merging these lists into one would have been tidier and would
have hidden a distinction the specification maintains.

Failure ordering (§41.9) is not a type constraint, so it lives in the endpoint
description where it cannot be missed by a reader of the operation.

### `spec/compat/jev-v1/openapi.snapshot.json` — §6, §39

Two routes, the request and response schemas, all four 422 envelope variants,
every §21.2 failure code, and the exact-routing rules of §39.8 as an
`x-fake-jev-routing` block — including that `GET /v1/models?x=1` is handled
while `GET /v1/models/` is an unknown route, since that asymmetry is invisible in
a route table and is a §44.16 golden vector.

**Provenance is stated in the file.** `x-fake-jev.provenance` is
`reconstructed-from-specification` and `networkFetch` is `false`, with a leading
`$comment` saying so in prose. This was the specification-stage call: the §45
path name implies a capture, the file is not one, and a future reader must not
learn that from a git history.

Two facts are recorded explicitly because they look like omissions:

- the 422 body carries **no** `fake_jev_*` code, while
  `fake_jev_validation_error` still exists as a journal and verification
  classification (§21.2, §39.3);
- `fake_jev_internal_error` is a second 500 body shape distinct from
  `fake_jev_invalid_stub_response` (§43.4, §43.6).

Without those, a reader cross-checking the §21.2 list against this file finds
two codes unaccounted for and cannot tell whether that is intent or omission.

### `spec/README.md` (updated)

The three artifacts move from "reserved" to committed, and the layout block
drops its reservation markers. `testdata/contracts/` stays reserved — FJ-003
owns it.

## Validation performed

No `jsonschema` library is available in this environment, so the schemas were
checked structurally rather than executed against instances. That is a weaker
check and is recorded as such rather than described as validation:

- JSON parses; YAML parses; every `$ref` resolves to a defined target; no
  orphaned or unused `$defs`/schemas.
- Every `required` name exists in the sibling `properties`.
- Cross-artifact agreement: question types, profile IDs, and every limit's
  range and default are identical across `config.schema.json` and the control
  API's `Limits`.
- All 8 §21.2 codes appear in the snapshot; all 4 §39.3 envelope types appear;
  all 9 §41.1 endpoints are present.
- `journal_full` is present in the verification codes and absent from the
  interaction outcomes, as §41.7 requires.

## What remains

The five criteria JSON Schema cannot express — alias normalization, duplicate
IDs, YAML/JSON equivalence, precedence, and the criteria-dependent numeric
cross-checks — are named in the artifacts at the fields they govern, each with
the reason it is not mechanically enforced. A future agent reading
`config.schema.json` will not mistake a loader responsibility for a guarantee
the schema makes.

Executing these schemas against fixtures is FJ-010's job, and FJ-046's.
