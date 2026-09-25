# spec/ — normative product behavior

## Layout

```text
spec/
    normative product behavior (prose, schemas, golden vectors)

spec/contracts/
    executable/golden representations of normative behavior

tests/
    enforcement of those contracts
```

## Files

| File | Status |
|------|--------|
| `spec/fake-jev-technical-spec-v2.md` | **Authoritative.** Normative v1 product specification, version 2.0. |
| `spec/fake-jev-technical-spec.md` | Superseded v1 document, retained only for history. Do not implement from it. |
| `spec/contracts/` | Reserved for machine-readable/golden contract artifacts once the authoritative spec defines them. Empty on purpose. |

Both specification files were originally at the repository root and were moved here
unchanged during repository setup.

## Rules

- The specification is the source of truth for public product behavior.
- `AGENTS.md` holds only universal repository rules. It does **not** replace the
  specification and must not be treated as a summary of it.
- Contract artifacts in `spec/contracts/` are derived from the specification, never
  a substitute for it. Do not invent contract files before the specification defines them.
- Tests enforce contracts; they do not define them.

## Conflict rule

If prose and executable contract artifacts appear to disagree, implementation MUST
stop. The inconsistency is resolved in the specification (or its derived artifacts)
before work continues — implementation must not choose whichever reading is convenient.
