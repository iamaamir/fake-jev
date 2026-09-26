# spec/ — normative product behavior

## Layout

Follows specification §17 and §45:

```text
spec/
    normative product behavior (prose, schemas, golden vectors)
    config.schema.json                          (§38)
    control-api.openapi.yaml                    (§41)
    compat/jev-v1/openapi.snapshot.json         (§6, §39)

testdata/contracts/
    golden vectors from §44 as executable inputs (§45, reserved)

test/
    conformance/     enforcement of those contracts
    integration/
```

## Files

| File | Status |
|------|--------|
| `spec/fake-jev-technical-spec-v2.md` | **Authoritative.** Normative v1 product specification, version 2.0. |
| `spec/fake-jev-technical-spec.md` | Superseded v1 document, retained only for history. Do not implement from it. |
| `spec/config.schema.json` | **Committed** (FJ-002). Machine-readable configuration contract from §38, with §12 defaults and the `jev` alias. |
| `spec/control-api.openapi.yaml` | **Committed** (FJ-002). Control API from §41, namespace `/__fake/v1`. |
| `spec/compat/jev-v1/openapi.snapshot.json` | **Committed** (FJ-002). The frozen `jev/v1` data-plane surface from §6 and §39. |
| `testdata/contracts/` | Required by §45. Reserved and empty until the golden vectors land. |

Both specification files were originally at the repository root and were moved here
unchanged during repository setup.

## Rules

- The specification is the source of truth for public product behavior.
- `AGENTS.md` holds only universal repository rules. It does **not** replace the
  specification and must not be treated as a summary of it.
- Contract artifacts under `spec/` and `testdata/contracts/` are derived from the
  specification, never a substitute for it. Do not invent contract files before the
  specification defines them.
- Tests enforce contracts; they do not define them.

## Conflict rule

If prose and executable contract artifacts appear to disagree, implementation MUST
stop. The inconsistency is resolved in the specification (or its derived artifacts)
before work continues — implementation must not choose whichever reading is convenient.
