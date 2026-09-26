---
stage: coder
task: FJ-003
inputFingerprint: e1fe08909f2476a93dcde226ef3d41fcaa37bcf0e4b586ba8940974bc8c946f6
outputFingerprint: 5fe4ba9be1430d49a566a75e1ca5e78d9b638b45cba746e9acfd54bd68bed9fc
taskFingerprint: f91008767970d165f99603537a03289a1e7a20598d44d203cf1151bb5781b8c2
gitHead: 7dbf861
generatedAt: 2026-09-26T10:20:59Z
---

# Coder — FJ-003

## What was written

23 JSON vector files across the eleven §45 concern directories. All sixteen
`C-GOLD-*` ids are covered, plus seven derived vectors in a separate `VEC-*`
namespace. No code, no assertions — this is the data a runner will execute.

| Directory | Files |
|---|---|
| `models/` | 1 |
| `noul/` | 1 |
| `choice/` | 1 |
| `score/` | 1 |
| `mixed/` | 2 (derived) |
| `matching/` | 6 (4 derived incl. the §44.6 second assertion) |
| `sequences/` | 1 |
| `reset/` | 1 |
| `validation/` | 4 (2 derived) |
| `verification/` | 4 (1 derived) |
| `cli/` | 1 |

## Vector shape

```json
{
  "id": "C-GOLD-001",
  "title": "...",
  "kind": "data-plane | control-plane | cli",
  "traces": ["§44.1", "§39.1"],
  "note": "why this contract exists and what it is pinning",
  "config": { ...complete fake-jev configuration in force... },
  "steps": [ { "request": {...}, "expect": { "status", "body", "journal" } } ],
  "conventions": { "comparison", "messageSentinel" },
  "verify": { "expectPassed", "expectFailures" }
}
```

Every `config` is a complete document rather than a stub fragment, so a runner
never has to merge fragments and never has to guess a default. Every `expect`
states an exact status. Nothing is asserted as "2xx".

## Three things this corpus deliberately refuses to assert

**`timestamp`.** §8.6 and §41.7 make the journal timestamp diagnostic. Vectors
that assert a journal record assert every other field and carry
`_timestampOmitted` with the reason. A runner that tried to compare timestamps
would fail on every run.

**`/verify` failure `message`.** §41.9 freezes the failure item *shape* — `code`,
`message`, `requestSequence`, `stubId` — and the *codes*, but never the wording.
The four vectors that assert a verify failure assert `code`, `requestSequence`,
and `stubId`, and set `message` to the `<not asserted>` sentinel. An earlier draft
of this corpus asserted an exact sentence; that sentence would have been a public
string the specification never froze, so the assertion was removed rather than
kept as a convenience. The sentinel and the rule for it are declared once, in
`conventions`, on every file.

**Serialized byte order and number spelling.** §44 compares parsed JSON values;
§39.4 says key order is not a protocol guarantee.

## Cross-artifact check performed

The vectors are data, so nothing executes them yet. What could be checked
mechanically was, against the artifacts FJ-002 committed:

- every `config` top-level key, stub key, `when` key, and `then` key exists in
  `spec/config.schema.json` with no unknown keys;
- every stub defines exactly one of `answers` / `sequence` / `raw`, and every
  sequence element exactly one of `answers` / `raw`;
- every journaled `outcome` is in the OpenAPI `InteractionOutcome` enum;
- every `error` and every failure body code is in the `FakeFailureCode` enum;
- every `/verify` failure `code` is in the `VerificationFailureCode` enum.

Result: no cross-artifact errors. This is a real check — it is what catches a
vector that typos `unknown_route` as `unknwon_route` or invents a stub field the
schema forbids — but it is not execution, and it is not a substitute for FJ-046.

## The two inversions worth naming

These are the vectors most likely to be "fixed" by a later reader who assumes
they are backwards:

- `verification/01-raw-error-not-a-failure.json` asserts that a configured
  HTTP 429 produces `outcome: matched`, `error: null`, and `passed: true`.
- `verification/02-journal-full.json` and `verification/03-payload-too-large.json`
  look similar but differ deliberately: 507 is never admitted (no sequence
  number, no stub mutation, sticky failure with `requestSequence: null`), while
  413 was admitted (sequence 1, journaled, `requestBody: null`). Both assert the
  stub's invocation count, which is what makes the difference observable.

## What remains

A runner (FJ-046) that reads these files, boots a server on an ephemeral port,
executes the steps, and compares. Until then this corpus is a claim, not a
check — recorded as such in the qa artifact.
