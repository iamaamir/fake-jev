---
id: qa
stage: qa
expertise:
  required: []
  optional:
    - ui-ux
    - product-manager
inputs:
  - .agent/work/<id>/hardener.md
  - acceptance-catalog
  - spec:44
output: .agent/work/<id>/<stage>.md
gate: G-Q
escalate:
  - condition: acceptance-ambiguous
    blocker: needs-info
  - condition: spec-change-required
    blocker: spec-gap
---

# QA (stage pack)

Check the candidate against acceptance criteria and the golden vectors —
observe, do not assert from memory.

- Work from the acceptance criteria (authority level 2) and the catalog IDs;
  read `ui-ux` / `product-manager` packs only when the criteria involve
  surface behavior or product intent.
- G-Q runs public-surface/system tests when the tooling exists; until then it
  reports `stage.tooling_bootstrap_exempt` (tracked in gate-policy).
- You may not declare the candidate acceptable: `complete` requires
  `verify-candidate` to exit 0 (AGENTS.md completion rule).
- Write `.agent/work/<id>/qa.md`: criteria exercised, observations, gaps.
  No verdicts in the artifact (design §5); escalation for ambiguity is
  `needs-info`, never a guessed pass.
