---
stage: specifier
task: FJ-003
inputFingerprint: e1fe08909f2476a93dcde226ef3d41fcaa37bcf0e4b586ba8940974bc8c946f6
outputFingerprint: e1fe08909f2476a93dcde226ef3d41fcaa37bcf0e4b586ba8940974bc8c946f6
taskFingerprint: f91008767970d165f99603537a03289a1e7a20598d44d203cf1151bb5781b8c2
gitHead: 7dbf861
generatedAt: 2026-09-26T10:17:28Z
---

# Specifier — FJ-003

## What is delivered

Data files only. No assertions, no harness, no Go code. Each file states enough
for a later test to reproduce one frozen contract: the configuration in force,
the request issued, the exact status and body expected, the journal record
expected, and the verification result expected. §45 calls these
`testdata/contracts/` and §44 calls them golden vectors; FJ-046 builds the
runner.

## A tension in the item, resolved explicitly

`nonGoals` says "no behavior invented beyond §44". §45 requires eleven concern
directories, one of which is `mixed/`. §44 contains no mixed-questions vector.

Leaving the directory empty fails §45. Inventing a vector for it would fail the
nonGoal. The resolution used here: the three non-§44 vectors are not invented,
they are **derived from normative requirements that §44 does not exercise**, and
each one names that authority in its `traces`:

| Vector | Authority, not §44 |
|---|---|
| heterogeneous questions | §13.4 (mandatory full coverage, no extra answers), §13.1, §13.2 |
| structured state matching | §40.3 (JSON-value equality), §22.3 (structured state) |
| invalid criteria | §39.2 (choice needs a non-empty criteria object), §22.3 |
| unknown configured choice | §43.4, §13.2 (selected key must exist), §22.3 |

§22.3 is the strongest support: it lists exactly these situations as golden
fixtures the project must maintain, independently of §44. Every value in those
files is copied from a MUST in §13, §39, §40, or §43 — none of them is a design
choice. An orchestrator who wants §44-only is right to reject them, and that
rejection is a scope change, not a correctness fix.

## What the vectors deliberately do not encode

- **JSON key order.** §39.4 says it is not a protocol guarantee, so no vector
  asserts on serialized bytes.
- **Timestamps.** §8.6 and §41.7 make the journal timestamp diagnostic only. The
  two vectors that assert a journal record assert every field except
  `timestamp`, and say so in the file.
- **Body key order inside `probabilities`** — same reason, same rule.
- **Numeric spelling.** §44 says comparisons ignore insignificant number
  spelling, so `1.0` and `1` are not distinguished anywhere.

## The one invariant every vector carries

No vector may pass by being lenient. Each `expect` block states the exact
status and the exact expected body, and each vector that must fail verification
says so in `verify`. A vector that only asserted "status is 2xx" would let a
wrong answer through, and these files are the repository's only record of what
correct behavior is. The 501 vector and the 409 vector are the sharpest cases:
a 200 there is a silent test failure that would ship.

## Vector-to-directory map

| Directory | §44 vectors | Derived |
|---|---|---|
| `models/` | 44.1 | — |
| `noul/` | 44.2 | — |
| `choice/` | 44.3 | — |
| `score/` | 44.4 | — |
| `matching/` | 44.5, 44.6, 44.12, 44.16 | structured state |
| `sequences/` | 44.7 | — |
| `reset/` | 44.8 | — |
| `validation/` | 44.10, 44.11 | invalid criteria |
| `verification/` | 44.9, 44.14, 44.15 | unknown configured choice |
| `mixed/` | — | heterogeneous questions |
| `cli/` | 44.13 | — |

44.16 (exact routes) is filed under `matching/` because `matching/` is the
closest concern directory in the §45 list; the file name says
`routes-exact-paths` so the routing subject is not hidden by its location.
§45 fixes the directory list, so this item cannot add a `routing/` directory.

## Traces

- C-GOLD-001 … C-GOLD-016 — §44.1 … §44.16, one file per vector, each naming
  its own section.
- C-QUAL-005 — §23, golden contracts run offline with no credentials, model, or
  network. No vector requires any of those; the files are static data.
- C-QUAL-006 — §45, the required artifact tree exists with content rather than
  as empty directories.
