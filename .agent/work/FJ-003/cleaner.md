---
stage: cleaner
task: FJ-003
inputFingerprint: 5fe4ba9be1430d49a566a75e1ca5e78d9b638b45cba746e9acfd54bd68bed9fc
outputFingerprint: 5fe4ba9be1430d49a566a75e1ca5e78d9b638b45cba746e9acfd54bd68bed9fc
taskFingerprint: f91008767970d165f99603537a03289a1e7a20598d44d203cf1151bb5781b8c2
gitHead: 7dbf861
generatedAt: 2026-09-26T10:21:58Z
---

# Cleaner — FJ-003

## Review

23 data files. Reviewed for the two ways a fixture corpus rots: **duplicated
expectation that can drift**, and **assertions stronger than the specification
supports**. Both were found; the first was removed during the coder stage and is
recorded there. Two further changes were made in review.

## Changes made

**1. `C-GOLD-006b` renamed to `VEC-MATCH-006B`.** The second assertion of §44.6
had been given an id in the acceptance-catalog namespace by analogy with
`C-GOLD-006`. The catalog has no such entry, and inventing one is exactly what
that file's "IDs are stable and never reused" rule exists to prevent. It now
sits in the `VEC-*` namespace with the other derived vectors. `C-GOLD-006`
appears exactly once in the corpus.

**2. §44.6 split into two self-contained files.** The single `matching/02` file
asserted the B-wins case and then described the C-wins case in prose as "a
second execution with B absent". A runner cannot execute prose, and phrasing a
required step as a mutation the runner must invent moves a judgment call into the
harness. `matching/06-priority-equal-registration-order.json` now states that
precondition as a complete configuration of its own.

## Findings, no change

- **Every `config` is a full document.** Twenty-three files repeat
  `schemaVersion`, `mode`, and `compatibility`. That is duplication with a
  purpose: a vector embedding only the interesting fragment forces the runner to
  merge, and merging is where a default silently diverges from §12.2. The
  repetition is the cost of a file being independently executable, and it is
  small and static.

- **The `note` field earns its place.** Six vectors assert something a
  reasonable implementer would expect to be the other way round — raw 429 passing
  verification, 507 not journaling, 413 journaling, sequence exhaustion still
  incrementing the counter, reset leaving verification failing. Each says why in
  one sentence at the point of use. Without them these files invite
  "corrections" that break the contract.

- **No vector asserts an expectation it does not set up.** The one vector using
  an invocation expectation (`reset/01`) deliberately ends in `passed: false`,
  because §44.8 says a static `expect.exactly: 2` fails after a single call. That
  is the most counter-intuitive assertion in the corpus and it is intentional.

- **No complexity surface.** Data files, no functions, no branching. The
  configured analyzer is absent under a bootstrap exemption and there is no
  executable code in this item for it to analyze.
