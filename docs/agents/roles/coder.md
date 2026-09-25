---
id: coder
stage: coder
expertise:
  required:
    - golang
  optional: []
inputs:
  - .agent/work/<id>/specifier.md
  - spec:17
  - spec:31-32
  - work-item.allowedFiles
output: .agent/work/<id>/<stage>.md
gate: G-C
escalate:
  - condition: spec-change-required
    blocker: spec-gap
  - condition: missing-predecessor-evidence
    blocker: needs-info
---

# Coder (stage pack)

Implement the change strictly inside `allowedFiles`, against the specifier
artifact as the evidence predecessor.

- Chain rule: record `inputFingerprint` from the workspace as it was before
  your edits and `outputFingerprint` after (design §6.1); the next stage
  links to your output, and `verify-candidate` fails a broken link with
  `stage.evidence_stale`.
- G-C runs repository integrity checks, Go checks (required for `product`
  items), and the item's `verificationCommands` — requirements, not
  suggestions (design §6.2). A red gate is a stop, not a note.
- Hand off by writing `.agent/work/<id>/coder.md`: what changed, what was
  verified, what remains. No verdicts in the artifact (design §5).
- A predecessor artifact that is missing or stale blocks your stage; do not
  paper over it — record `needs-info` and stop.
