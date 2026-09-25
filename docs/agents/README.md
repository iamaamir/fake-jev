# docs/agents/ — team packs (stage authority + advisory expertise)

The repository's portable team: deterministic **stage packs** (evidence
boundaries that decide acceptance) plus advisory **expertise packs** (domain
knowledge that helps reasoning). Plain files, discoverable through `AGENTS.md`;
no harness-specific adapters (design §1, §14).

## Authority hierarchy (frozen)

```text
1. specification + executable contracts   spec/fake-jev-technical-spec-v2.md
2. work-item acceptance criteria          .agent/work/<id>/state.json
3. role pack                              docs/agents/roles/*.md
4. expertise pack                         docs/agents/expertise/*.md
5. general model knowledge
```

Higher authority wins. If resolving a question would change or establish
strategic policy, architecture boundaries, specification semantics, or public
contracts not already determined by higher-authority artifacts, record a
blocker and stop (`needs-human` or `spec-gap`) — never improvise (design §2).
Decisions fully constrained by the specification are resolved with the
`staff-architect` pack and never escalate.

**Advisory rule (normative).** Expertise packs are advisory. They may explain
how to reason, what to inspect, and what questions to ask. They MUST NOT
introduce product requirements, acceptance conditions, architecture rules, or
behavioral constraints not traceable to the specification or executable
contracts. They are never a shadow specification.

## Stage packs — `roles/` (workflow authority)

Each pack defines one stage's inputs, output artifact, gate and escalation.

| Pack | Stage | Gate | Output |
|------|-------|------|--------|
| `roles/specifier.md` | specifier | G-S | `.agent/work/<id>/specifier.md` |
| `roles/coder.md` | coder | G-C | `.agent/work/<id>/coder.md` |
| `roles/cleaner.md` | cleaner | G-L | `.agent/work/<id>/cleaner.md` |
| `roles/hardener.md` | hardener | G-H | `.agent/work/<id>/hardener.md` |
| `roles/qa.md` | qa | G-Q | `.agent/work/<id>/qa.md` |

Artifacts are evidence, never verdicts: only `./scripts/verify-candidate`
emits `pass|fail|skipped|not_applicable` (design §5). The required header and
the fingerprint chain are specified in design §5–§7 and validated by the G-*
gates.

## Expertise packs — `expertise/` (advisory)

| Pack | Used by |
|------|---------|
| `expertise/product-manager.md` | required: specifier · optional: qa |
| `expertise/staff-architect.md` | required: cleaner |
| `expertise/golang.md` | required: coder, hardener · optional: cleaner |
| `expertise/security.md` | optional: hardener |
| `expertise/ui-ux.md` | optional: qa |

ROLE decides what evidence is required; EXPERTISE helps reason about the
particular task. Required packs are read for the current stage
(`./scripts/agent-context` lists them); optional packs are pointers, read on
demand — never injected automatically.

## Related

- `issue-tracker.md`, `triage-labels.md`, `domain.md` — tracker and domain docs.
- `.agent/schema/` — canonical contracts for packs, artifacts, work items (§11).
- `.agent/gate-policy.json` — bootstrap exemptions (design §9.2).
