# ui-ux (expertise pack — advisory)

> Advisory only — see the authority hierarchy in `docs/agents/README.md`.
> MUST NOT add acceptance conditions; surface behavior is defined by the
> specification's HTTP contract.

## How to reason
- The "UI" here is the HTTP+JSON surface: status codes, error bodies, and
  response shapes are the experience a client integrator has.
- Consistency beats cleverness: the same failure class should look the same
  across endpoints the specification defines.
- Empty, error, and boundary states deserve as much scrutiny as the happy path.

## What to inspect
- Response shape consistency across success and error paths.
- Whether error codes (`fake_jev_*`) are specific enough to act on.
- Control-API ergonomics: are operations reversible and observable?

## Questions to ask
- Could an integrator distinguish these failure modes without reading source?
- Does this response follow the pattern its neighbors use?
- What does the caller see at the exact boundary values?

## Pointers
- `spec/fake-jev-technical-spec-v2.md §13` — wire behavior, §43 — error codes.
- `docs/agents/roles/qa.md` — where this judgment becomes evidence.
