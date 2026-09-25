# Agent development workflow

The operating model is:

```text
                  authoritative spec
                         |
                         v
                    repository
                         |
                 work-item state
                         |
                         v
                  agent-context
                         |
                         v
               bounded task packet
                         |
                         v
                  implementation
                         |
                         v
                verify-candidate
                         |
                         v
             structured evidence
                         |
                 +-------+-------+
                 |               |
               PASS             FAIL
                 |               |
               review        continue/fix
```

Two principles drive it:

> Context is RAM, not disk. The repository carries the project; agents do not.

> Generation may be probabilistic. Acceptance must be deterministic whenever possible.

## The loop

1. **Create a work item** — `.agent/work/<task-id>/state.json` with a short id such
   as `FJ-017`.
2. **Record the bounds** — objective, spec references, relevant files, allowed files,
   acceptance criteria, non-goals, verification commands.
3. **Run `./scripts/agent-context <task-id>`** — produces a concise task packet.
4. **Hand the packet to an implementation model** — the packet plus the repository is
   the whole context; no chat history is required.
5. **Implement only the bounded task** — edit only `allowedFiles`.
6. **Run `./scripts/verify-candidate <task-id>`** — deterministic checks, evidence
   written to `.agent/reports/<task-id>/`.
7. **Persist state** — update `status`, `lastVerification`, `nextAction`, `blockers`.
8. **Review** against the original specification.
9. **Fix or escalate** failures (see retry policy below).
10. **Mark complete** only when verification passes *and* review passes.

## Review

The reviewer must compare three things:

```text
original specification
        +
actual diff
        +
test/verification evidence
```

The reviewer must not rely solely on the implementer's explanation. A green verifier
means the checks ran and passed for that revision; it does not mean the specification
was satisfied.

## Model roles

Roles, not hard dependencies. The repository does not call models and is not coupled
to any model vendor.

```text
Orchestrator        typically Terra-class model   — decomposes work, writes work items
Implementor         typically Luna-class model    — executes one bounded task
Reviewer            typically Sol-class model     — checks diff vs spec vs evidence
Escalation          typically Astra-class model   — architecture/spec ambiguity,
                                                    repeated unclear failures
Acceptance authority  deterministic repository tooling / tests
```

The separation that matters is:

```text
orchestrate -> implement -> review -> accept
```

Acceptance belongs to deterministic tooling, never to a model's self-assessment.

## Task sizing

A work item should be a vertical slice small enough that an implementor can reason
about it without loading the entire repository.

A packet includes:

```text
objective
relevant spec sections
relevant files
allowed modifications
acceptance criteria
non-goals
verification commands
```

Bad task:

```text
Implement Phase 1 of fake-jev.
```

Better:

```text
Implement deterministic stub ordering.

Spec:     spec/fake-jev-technical-spec-v2.md § 40
Files:    internal/engine/stub.go, internal/engine/matcher.go
Acceptance: C-MATCH-001, C-MATCH-002, C-MATCH-003
Rule:     priority DESC, registration order ASC
Non-goals: no matcher specificity scoring, no Jev-specific behavior
```

Packets reduce reasoning scope; they do not duplicate the specification.

## Retry and escalation

```text
attempt 1:  implement + verify
attempt 2:  fix using concrete verification evidence
still unclear -> return to orchestrator / stronger model
architecture or specification ambiguity -> escalate, do not invent
```

The invariant: repeated failure must increase reasoning capability, not merely
increase retry count. There is no indefinite cheap-retry loop.

## Specification gaps

If implementation requires externally observable behavior that the specification does
not define, the agent MUST NOT silently choose a public behavior. Record it:

```json
{
  "blockers": [
    { "type": "spec-gap", "description": "§ 41 reset does not define ..." }
  ]
}
```

Internal implementation details that do not affect observable contracts do not need
specification escalation.

## Related files

- `AGENTS.md` — universal rules injected into tasks.
- `.agent/README.md` — work-item and report schemas.
- `docs/adr/` — architecture decision records (rare, meaningful decisions only).
