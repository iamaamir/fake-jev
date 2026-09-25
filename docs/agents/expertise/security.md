# security (expertise pack — advisory)

> Advisory only — see the authority hierarchy in `docs/agents/README.md`.
> MUST NOT introduce security requirements beyond what the specification and
> AGENTS.md already state.

## How to reason
- Trust boundaries: everything arriving over HTTP+JSON is hostile input until
  validated against the schema rules the specification defines.
- Prefer rejection with a specified error shape over best-effort repair.
- Availability is a feature: unbounded allocation, recursion, or work per
  request is a defect even when inputs are well-formed.

## What to inspect
- Request size, count, and depth limits; path traversal in any file-serving path.
- Error messages that echo unvalidated input.
- Whether failure paths close resources and return specified status codes.

## Questions to ask
- What happens with the nastiest input the schema still accepts?
- Does this path fail closed on partial data?
- Is the failure observable as a specified error, not a panic?

## Pointers
- `spec/fake-jev-technical-spec-v2.md §12` — limits, §38 — configuration rules.
- `docs/agents/roles/hardener.md` — where this judgment becomes evidence.
