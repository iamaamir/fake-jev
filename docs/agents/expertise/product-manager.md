# product-manager (expertise pack — advisory)

> Advisory only — see the authority hierarchy in `docs/agents/README.md`.
> MUST NOT introduce product requirements, acceptance conditions, or
> behavioral constraints not traceable to the specification.

## How to reason
- Start from observable behavior: what does a user of the HTTP+JSON surface
  do, and what would they see if this changed?
- Separate scope deliberately: an objective names a vertical slice; `nonGoals`
  exist to be respected, not re-litigated.
- Treat acceptance criteria as the contract of the task; if a criterion
  cannot be met without contradicting the specification, that is a spec gap,
  not a negotiation.

## What to inspect
- `.agent/work/<id>/state.json` — objective, acceptance, nonGoals.
- `docs/development/acceptance-catalog.md` — which criteria already exist and
  what they trace to.
- Specification §33 deferred decisions — features deliberately out of scope.

## Questions to ask
- Is the observable outcome stated without implementation detail?
- Does anything here belong in a nonGoal instead?
- Would a reviewer be able to falsify each acceptance line?

## Pointers
- `spec/fake-jev-technical-spec-v2.md §31–§33` — phases, done, deferred.
- `docs/agents/roles/specifier.md` — where these judgments become evidence.
