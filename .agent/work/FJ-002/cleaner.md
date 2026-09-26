---
stage: cleaner
task: FJ-002
inputFingerprint: 71e7458cdb2181ae63b22be2d881a8fa128cd1cf2ef5fc9072649feadb684652
outputFingerprint: 71e7458cdb2181ae63b22be2d881a8fa128cd1cf2ef5fc9072649feadb684652
taskFingerprint: 97f85d62e553a71b7ad6766e5c74c996b012f9bb65b018bdbf5776b61111cda0
gitHead: c4192d0
generatedAt: 2026-09-26T07:51:36Z
---

# Cleaner — FJ-002

## Review

Artifacts only, no executable code. Reviewed for the two failure modes that
matter for a contract artifact: **claiming a guarantee it does not carry**, and
**duplicating a rule in a place that can drift**. One change was made; the rest
is recorded.

## Change made: none

The first thing reviewed was the control document's `Stub` schema, on the
expectation that it duplicated the config schema normatively and would drift.
It does not: the coder had already pointed it at
`spec/config.schema.json` as the normative form and stated that the two must not
drift. The duplication risk the review was looking for is already handled, by a
prose pointer rather than a second copy — which is the smaller failure surface
either way. Nothing was changed.

Recorded explicitly because the temptation in a later pass will be to *add* a
`$ref` to the config schema's `$defs` from the OpenAPI document. That must not
be done: the two files use different `$id` bases, and a cross-document `$ref`
would make the control API unresolvable for any consumer that loads it on its
own.

## Findings, no change

- **The three enumerations stay separate.** Tempting to hoist
  `unmatched`/`unknown_route`/… into one shared component, since seven of the
  eight fake failure codes are their `fake_jev_`-prefixed twins. The asymmetry
  is the signal: `journal_full` exists only in the verification list, and the
  prefix-and-suffix transformation is *not* uniform. Sharing a definition would
  make the uniform part look authoritative and hide the two exceptions.
- **§12.2 defaults appear as JSON Schema `default` annotations.** They are
  annotations, not constraints: JSON Schema `default` does not apply values. That
  is correct for a contract artifact — the *loader* applies defaults, and the
  specification says so — but a reader who assumes the schema fills in omitted
  fields would be wrong, so `compatibility`, `models`, and `stubs` say
  "Default […]" in prose next to the annotation.
- **`$id` uses an `invalid` TLD.** `https://fake-jev.invalid/...` is reserved by
  RFC 2606 precisely so it can never resolve. A schema that claims a real URL it
  does not control would invite a fetch that cannot succeed and might be
  redirected.
- **The superseded `spec/fake-jev-technical-spec.md` was left alone.** The README
  already marks it history-only. Deleting it is not this item's business, and
  `allowedFiles` does not include it.
- **The snapshot keeps its awkward name.** `openapi.snapshot.json` is a
  reconstruction, not a capture. The name is fixed by §45, so the mismatch is
  paid for with a prominent `$comment` and an `x-fake-jev.provenance` field
  rather than by renaming the file, which would break the §45 path and every
  reference to it. Renaming would also be worse: the honest name for a derived
  artifact is not the one the specification mandates, so the file has to carry
  the correction in its contents.

## Complexity

Not applicable: no functions, no branching, no duplication across the three
artifacts beyond the one resolved above. The configured analyzer is absent under
a recorded bootstrap exemption, and there is no executable product code in this
item for it to analyze.
