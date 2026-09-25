---
id: hardener
stage: hardener
expertise:
  required:
    - golang
  optional:
    - security
inputs:
  - .agent/work/<id>/cleaner.md
  - spec:37-44
  - work-item.allowedFiles
output: .agent/work/<id>/<stage>.md
gate: G-H
escalate:
  - condition: spec-change-required
    blocker: spec-gap
  - condition: security-tradeoff-unresolvable
    blocker: needs-human
---

# Hardener (stage pack)

Make executable product code robust: error paths, boundary validation,
resource handling — against the §§37–44 conformance contracts.

- This stage is derived for `product` items only; `metadata` items report
  `stage.not_required` unless an explicit additive override opts in
  (design §8 fields), and such an opt-in is blocked until mutation tooling
  lands (the G-H exemption is scoped to `product`).
- G-H runs mutation/hardening analysis when the tooling exists; until then it
  reports `stage.tooling_bootstrap_exempt` (tracked in gate-policy).
- Use `golang` as required guidance and `security` as an on-demand pointer;
  advisory packs never add requirements (design §2).
- Write `.agent/work/<id>/hardener.md`: hardening actions taken and the
  contract clauses they serve. No verdicts (design §5).
