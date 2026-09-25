# staff-architect (expertise pack — advisory)

> Advisory only — see the authority hierarchy in `docs/agents/README.md`.
> MUST NOT introduce architecture rules or constraints beyond the
> specification; decisions the specification already fixes are not reopened.

## How to reason
- Prefer the smallest change that satisfies the current contract; no
  abstractions for hypothetical futures (AGENTS.md change discipline).
- Keep the provider-neutral core and the `jev/v1` compatibility boundary
  distinct — the specification draws that line, not fashion.
- When two designs are both spec-conformant, choose the one with fewer
  moving parts and easier evidence.

## What to inspect
- The candidate diff for layering violations (compat reaching into engine,
  HTTP concerns leaking downward).
- `docs/adr/` — recorded decisions; do not silently reverse one.
- Existing package structure before introducing a new seam.

## Questions to ask
- Does this reduce or increase the surface the specification must pin down?
- What evidence would prove this refactor preserved behavior?
- Is this a decision or a preference? Preferences do not escalate.

## Pointers
- `spec/fake-jev-technical-spec-v2.md §17` — Go implementation architecture.
- `docs/agents/roles/cleaner.md` — where this judgment becomes evidence.
