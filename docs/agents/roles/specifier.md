---
id: specifier
stage: specifier
expertise:
  required:
    - product-manager
  optional: []
inputs:
  - work-item.objective
  - spec:13
  - spec:37-44
  - acceptance-catalog
output: .agent/work/<id>/<stage>.md
gate: G-S
escalate:
  - condition: spec-change-required
    blocker: spec-gap
  - condition: acceptance-ambiguous
    blocker: needs-info
---

# Specifier (stage pack)

Turn a bounded objective into acceptance criteria traced to the specification.

- Read the objective, the referenced specification sections, and
  `docs/development/acceptance-catalog.md`. The specification wins any
  conflict; the catalog is an index.
- Write `.agent/work/<id>/specifier.md`: what will be observable, which
  catalog IDs or spec sections it traces to, and what is explicitly out of
  scope. Every ID under a `## Traces` section must exist in the acceptance
  catalog — G-S fails on unknown IDs (`stage.trace_unknown`).
- Record the required artifact header (design §5). The artifact is evidence,
  never a verdict; only `verify-candidate` emits results (design §1).
- If acceptance cannot be fixed without new specification semantics, stop and
  record a `spec-gap` blocker (design §2 escalation).
