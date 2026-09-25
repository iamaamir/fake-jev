# golang (expertise pack — advisory)

> Advisory only — see the authority hierarchy in `docs/agents/README.md`.
> MUST NOT add requirements; determinism, offline operation and fail-closed
> rules come from the specification and AGENTS.md.

## How to reason
- Deterministic by construction: no map-iteration order in outputs, no wall
  clock in decisions, no unseeded randomness, no sleeps-as-sync.
- Errors are values: propagate with context, fail closed, never
  log-and-continue on paths the specification calls strict.
- Table-driven tests mirror the golden vectors; a test that disagrees with
  `testdata/` is wrong until proven otherwise.

## What to inspect
- Goroutine/channel ownership and lifecycle — races surface under
  `go test -race`.
- Boundary parsing: who validates unknown keys, ranges, and duplicates.
- Whether new code could ever trigger a network call at runtime (it must not).

## Questions to ask
- What is the exact failure mode for malformed input here?
- Does this depend on timing, ordering, or locale?
- Would `go vet` and the golden harness both be satisfied?

## Pointers
- `spec/fake-jev-technical-spec-v2.md §17` — architecture, §22 — test strategy.
- `docs/agents/roles/coder.md`, `docs/agents/roles/hardener.md` — where this
  judgment becomes evidence.
