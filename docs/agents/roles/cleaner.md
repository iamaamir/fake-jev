---
id: cleaner
stage: cleaner
expertise:
  required:
    - staff-architect
  optional:
    - golang
inputs:
  - .agent/work/<id>/coder.md
  - spec:17
  - work-item.allowedFiles
output: .agent/work/<id>/<stage>.md
gate: G-L
escalate:
  - condition: architecture-boundary-undetermined
    blocker: needs-human
  - condition: spec-change-required
    blocker: spec-gap
---

# Cleaner (stage pack)

Simplify structure without changing behavior: naming, duplication, layering,
dead code — judged against the specification, not taste.

- Work from the coder artifact and re-verify after every structural move; a
  cleaner edit changes the candidate fingerprint, which stales every
  downstream artifact (design §7). Re-run `verify-candidate` before hand-off.
- Use the `staff-architect` expertise pack for judgment; decisions fully
  constrained by the specification are resolved with it and never escalate
  (design §2).
- G-L runs complexity/CRAP analysis when the tooling exists; until then the
  gate reports `stage.tooling_bootstrap_exempt` (tracked in
  `.agent/gate-policy.json`).
- Write `.agent/work/<id>/cleaner.md`: structure changed, behavior unchanged,
  evidence of re-verification. No verdicts (design §5).
