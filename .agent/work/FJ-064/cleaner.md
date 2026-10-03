---
stage: cleaner
task: FJ-064
inputFingerprint: 74245107247629f9c366043ed7aada0dfbe29f4b63157bd6218afc3ee84a83d0
outputFingerprint: 74245107247629f9c366043ed7aada0dfbe29f4b63157bd6218afc3ee84a83d0
taskFingerprint: 71aa852bd544dfae210d4e52273e9036c9347ba7f7d92dacc4586a9c9da6e42a
gitHead: ab45ac5
generatedAt: 2026-10-03T13:17:54Z
author: worker/FJ-064-cleaner
---

# Cleaner — FJ-064

The cleaner made **no edits**; the candidate is byte-identical to the coder's
`74245107247629f9`. `outputFingerprint` therefore equals `inputFingerprint`.
Everything below is measurement and analysis; the scanned/reproduced facts are
recorded for the PM, and this artifact carries no verdict.

Two findings concern the same code path (`rejectCaseVariantKeys` / `foldKey` in
`internal/config/load.go`, added by the coder). Both are behaviour changes if
corrected, so per the stage boundary they are recorded, not fixed.

## F-A (headline) — the folded scan is document-wide and rejects content the specification leaves open

The PM's reading is confirmed, independently, on the post-fix binary `/tmp/fj064`
and contrasted with the pre-fix binary `/tmp/fj064-prefix` (both pre-built;
`/tmp/fj064` matches candidate `74245107247629f9`, and the repository-root
`fake-jev` binary reproduces the same outcomes).

Reproduction (files written under `/tmp/fj064-cleaner/`):

| File | Body (abridged) | Pre-fix | Post-fix |
|---|---|---|---|
| `raw.yaml` | `then.raw.body: {a: 1, A: 2}` | exit 0, `configuration is valid` | exit 2, `decode configuration: object keys "A" and "a" differ only in case` |
| `body_json.json` | `"body":{"Ok":1,"ok":2}` (JSON input) | exit 0, valid | exit 2, `... object keys "Ok" and "ok" differ only in case` |
| `state_ok.yaml` | `when.state: {Key: 1, KEY: 2}` | exit 0, valid | exit 2, `... object keys "KEY" and "Key" differ only in case` |
| `questions_case.yaml` | `when.questions: {Kind: noul, KIND: noul}` + matching `then.answers` | exit 0, valid | exit 2, `... object keys "KIND" and "Kind" differ only in case` |
| `answers_case.yaml` | `then.answers: {Kind: ..., KIND: ...}` | exit 0, valid | exit 2, `object keys "KIND" and "Kind" differ only in case` |
| `seqbody_case.yaml` | `then.sequence[0].raw.body: {providerPayload: {Ok, ok}}` | exit 0, valid | exit 2, `object keys "Ok" and "ok" differ only in case` |
| `headers_case.yaml` | `then.raw.headers: {X-Rate-Limit, x-rate-limit}` | exit 0, valid | exit 2, `object keys "X-Rate-Limit" and "x-rate-limit" differ only in case` |

Command form: `timeout 30 /tmp/fj064 validate <file>` vs
`timeout 30 /tmp/fj064-prefix validate <file>`.

Why the rejections are outside the ticket's target:

- In JSON and YAML, `"a"` and `"A"` are distinct object keys. Nothing is
  overwritten and nothing is lost when the decode target is a map
  (`map[string]string`, `map[string]json.RawMessage`) or `any`/`json.RawMessage`.
- The silent overwrite the ticket targets exists only where `encoding/json`
  matches an object key against a **struct field** name case-insensitively
  (`encoding/json` `fields.byFoldedName[string(foldName(key))]`). Scanning every
  object in the document, including objects that are never decoded into a
  project-owned struct, rejects documents in which no key resolution is
  ambiguous.
- Spec 12.3: "Profile-specific response bodies inside `then.raw.body` are
  arbitrary JSON and are exempt from unknown-key validation." Spec 12.6: "Raw
  responses are intentional emulated-provider behavior and MUST NOT be validated
  against `jev/v1` answer schemas." Spec 38.8 lists `state` values,
  question-instructions/criteria content, `then.raw.body`, and emulated provider
  JSON bodies as intentionally open containers.
- The item's own acceptance non-goal is "no change to the public configuration
  schema or to any accepted input that is currently unambiguous". A case-variant
  pair inside one of the containers below is unambiguous, and the seven documents
  above were accepted inputs before this change.
- Question names are not decoder-folded anywhere: the wire contract matches
  "exact question-name set + exact type per name" (§12.5, §38.6), so `Kind` and
  `KIND` are two distinct questions, not one ambiguous key.

## F-B (new, discovered in this stage) — `foldKey` does not terminate for non-ASCII runes without a case mapping; `config.Load` hangs

`foldKey`'s non-ASCII branch is

```go
for next := unicode.SimpleFold(r); next <= r; next = unicode.SimpleFold(r) {
    r = next
}
```

`unicode.SimpleFold` returns `r` itself for any code point with no case mapping
(`$GOROOT/src/unicode/letter.go`: falls through `caseOrbit`, then
`lookupCaseRange`, then `return r`). `next == r` satisfies `next <= r`, `r` never
changes, and the loop never exits.

Observed (post-fix binary; the same command completed normally on the pre-fix
binary):

| File | Body | Pre-fix | Post-fix |
|---|---|---|---|
| `uncased_top.json` | `{"schemaVersion":1,"中":1}` | exit 2, `json: unknown field "中"` | **exit 124** (killed by `timeout 8`, no output) |
| `cjk.json` | stub with `when.questions` key `質問` | exit 0, valid | **exit 124** |
| `emoji_body.json` | `then.raw.body` key `emoji🙂` | exit 0, valid | **exit 124** |
| `cased_top.json` | `{"schemaVersion":1,"é":1}` (cased) | exit 2, unknown field | exit 2, unknown field (no hang) |

So the hang needs a non-ASCII object key with no case mapping (CJK, emoji,
symbols, …) at any level of any object, and it is reachable from untrusted input
in both exempt and non-exempt positions. Any JSON object key is folded, so this
is not limited to the exempt containers. Reachability beyond `config.Load`: the
control API's `createStub` builds a document from the caller's stub and calls
`config.Load` on it (`internal/control/handlers.go:281`), so a crafted
`POST /__fake/v1/stubs` body reaches the same loop (by inspection; not exercised
here, no server run in this stage).

The pre-existing unit tests do not cover non-ASCII keys, which is why the
candidate's `go test ./internal/config` is green while the loop is reachable.

Divergence in the same helper (bounded): the loop's termination rule
("stop as soon as the next hop is not smaller") is not `encoding/json`'s
`foldRune`, which keeps ascending until the fold cycle wraps to its minimum
(`for { r2 := unicode.SimpleFold(r); if r2 <= r { return r2 }; r = r2 }`,
`$GOROOT/src/encoding/json/fold.go`). For a rune that is a middle member of a
fold cycle, the coder's helper returns the rune itself where `foldRune` (and
`strings.EqualFold` equivalence) gives the cycle minimum. Observed:
`dz_body.json` (`then.raw.body` keys `\u01c4` and `\u01c5`, equal under
`strings.EqualFold`) is still accepted after the fix — the pair is not detected.
The two fold cycles that contain an ASCII code point (`{K,k,U+212A}`,
`{S,s,U+017F}`, derived from `$GOROOT/src/unicode/tables.go caseOrbit`) have
their non-ASCII member as the cycle maximum, which the loop canonicalises
correctly — verified by `kelvin_body.json` (`\u212a` vs `k` is rejected,
`object keys "K" and "k" differ only in case`). Consequently the divergence
cannot hide a collision against the project's ASCII field names and has no
fail-open consequence; its observable effect is confined to the over-broad
rejection in F-A.

Minimal corrections (for the coder/PM, not applied here):

1. Replace `foldKey`'s non-ASCII loop with `encoding/json`'s `foldRune`
   algorithm verbatim (it terminates on fixed points by construction and gives
   the same canonical form as the decoder). Without this, even a narrowed scan
   still hangs on a non-ASCII uncased key in a struct-decoded object.
2. Restrict the folded comparison to the objects `encoding/json` resolves into
   the project-owned structs (the table below), i.e. a schema-directed walk in
   `Load` that recurses only into struct-mapped fields and stops at
   `json.RawMessage`/map values. C1/C2 as written still hold, because every
   fail-open case is a struct-level collision. The document-wide *exact*
   duplicate scan stays as it is (see "Pre-existing exact-duplicate behaviour").

## Which levels are genuinely ambiguous, and which containers are open

Derived from `internal/config/schema.go` and the `Load` decode path
(`json.Decoder.Decode(&cfg)` with `DisallowUnknownFields`).

Fold-ambiguous (a case-variant pair can silently overwrite a field, so rejection
is the fail-closed behaviour the ticket wants):

| JSON path | Go target |
|---|---|
| root object | `Config` |
| `server` | `ServerConfig` |
| `limits` | `LimitsConfig` |
| `models[i]` | `ModelConfig` |
| `stubs[i]` | `StubConfig` |
| `stubs[i].when` | `WhenConfig` |
| `stubs[i].then` | `ThenConfig` |
| `stubs[i].then.sequence[j]` | `ResponseConfig` |
| `stubs[i].then.raw`, `stubs[i].then.sequence[j].raw` | `RawResponse` |
| `stubs[i].then.usage`, `stubs[i].then.sequence[j].usage` | `UsageConfig` |
| `stubs[i].expect` | `ExpectConfig` |

Open containers (no decoder folding; both keys are kept separately, so a
case-variant pair is not a collision and the spec exempts the content from
project-owned key rules):

| JSON path | Go target | Basis |
|---|---|---|
| everything beneath `stubs[i].when.state` | `json.RawMessage` | §12.5 `state` exact-value match; §38.8 "state values" |
| everything beneath `stubs[i].then.answers` and `...then.sequence[j].answers` | `json.RawMessage` | answer payloads keyed by exact question name (§12.5/§12.6) |
| everything beneath `stubs[i].then.raw.body` and `...sequence[j].raw.body` | `json.RawMessage` | §12.3 arbitrary JSON; §12.6 raw is emulated-provider behaviour; §38.8 |
| the names in `stubs[i].when.questions` and in answer objects | `map[string]string` / raw object keys | exact question-name matching (§12.5, §38.6) |

Boundary case, deliberately not classed as open: `stubs[i].then.raw.headers` is
`map[string]string`, so the decoder does not fold its keys, but HTTP header names
are case-insensitive and `writeEncoded` writes them with
`writer.Header().Set(...)` (`internal/host/http/server.go:400`), which
canonicalises to one key and makes the surviving value depend on Go map iteration
order. Rejecting a case-variant pair there is defensible for a reason unrelated
to §38.8; whether the narrowed scan keeps it is a coder/PM call, not a
spec-mandated one.

## Pre-existing exact-duplicate behaviour (empirical)

Answer: **already document-wide before this change, and unchanged by it.** The
new scan's `foldedKeys` mode preserves the exact-duplicate diagnostic
(byte-identical repeated key → `duplicate object key %q`).

Evidence (`/tmp/fj064-cleaner/`):

- `exactdup_body.yaml` and `exactdup_body.json` (exact duplicate inside
  `then.raw.body`) are rejected by **both** binaries, byte-identical:
  `decode YAML configuration: yaml: unmarshal errors: line 1: mapping key "a" already defined at line 1`.
- `exactdup_top.json` (`{"schemaVersion":1,"schemaVersion":1}`) is likewise
  rejected identically pre- and post-fix.

Mechanism, from the code at `ab45ac5`: for input beginning with `{`/`[`,
`toJSON` keeps the original bytes only when `rejectDuplicateJSON` **succeeds**;
any error — including its own `duplicate object key` — falls through to the YAML
decoder. yaml.v3's duplicate-key check (`uniqueKeys`, which is on by default)
then rejects the document, document-wide, for both JSON and YAML text. The
`decode JSON configuration: ...` messages that `rejectDuplicateJSON` builds are
therefore not observable through `Load` on either revision (pre-existing, not
touched here). Reachable from exempt containers: yes — an exact repeat inside
`then.raw.body` is rejected today.

Judgement recorded for the PM: this is **not** the same class as F-A. A repeated
exact key has no well-defined decoded value for any JSON consumer (last-wins is
implementation-defined), and YAML requires mapping keys to be unique, so
rejecting it is consistent with "strict behavior must fail closed" rather than
validating exempt content. Reported as a separate, pre-existing observation; the
cleaner changed nothing about it.

## Cleanups applied: none

- The over-reach (F-A) and the hang (F-B) are behaviour; fixing either exceeds
  the stage boundary, and the instruction for this stage is to leave the code as
  the coder left it. Both corrections sit in the same few lines, so a partial
  cleanup now would only add churn to a revision that must be revised anyway.
- `sortedKeys` duplicates the small helper of the same name in
  `internal/compat/jev/v1/fixture.go`. Per `AGENTS.md` ("do not extract shared
  code solely to eliminate small or incidental duplication") the duplicate is
  acceptable; the alternative would add a dependency from `internal/config` onto
  a compatibility profile, which is worse.
- No dead code introduced: `keyFolding`/`exactKeys` are still used by
  `rejectDuplicateJSON`, `orderedKey`/`orderYAMLKeys` by `normalizeYAMLValue`,
  and `sort`/`unicode`/`utf8` are all used. Control flow in `scanJSONValue` is
  one added branch; no restructuring is warranted.

## Comment audit

- `rejectCaseVariantKeys`' doc comment is accurate about the document-wide
  scope; no comment claims the scan is limited to ambiguous levels, so the
  specific over-claim the PM asked about does not occur.
- `foldedKeys`' comment ("so two keys the decoder cannot tell apart count as a
  repeat") over-claims: in map/`json.RawMessage` targets the decoder tells them
  apart. This is the comment form of F-A.
- `foldKey`'s comment ("two keys have the same folded form exactly when
  `strings.EqualFold` reports them equal") is false for middle members of fold
  cycles (the `\u01c4`/`\u01c5` case above), and the helper it documents does not
  terminate (F-B).

## Determinism and randomness audit of the change

- No sleep, no unseeded randomness, no wall-clock use, no new package-level
  state; `Load` remains a pure function of its input bytes.
- The change **removes** map-iteration-order dependence from
  `normalizeYAMLValue` (`map[string]any`, `map[any]any`) and from the two
  `validate.go` walks whose messages embed a key (`validateWhenDocument`,
  `validateWhen`).
- Remaining map walk in `internal/config`: `validate.go:520`
  `for name := range raw.Headers` reports a key-independent message
  (`raw header names must be non-empty`), so its outcome is byte-identical
  regardless of iteration order; it was deliberately left unsorted. No other
  order-dependent diagnostic remains in the package.
- `orderYAMLKeys` orders by `%T\x00%v`, which is a total order for all
  practically reachable YAML keys. Ties are possible only for two mutually
  unequal keys of the same type that format identically — in the current data,
  two `NaN` float keys (`.nan` vs `.NaN`, which yaml.v3's unique-key check does
  not merge because the spellings differ). On a tie the walk's first key can
  vary, but the reported message is `object key NaN is not a string` for either
  key and the walk returns before recursing, so the diagnostic a caller observes
  is still byte-identical. Recorded as a bounded, non-observable residual.
- The new scan itself walks the document in document order and never reports a
  map-derived first element, so its diagnostic is a function of the input alone.
- Empirically, the pre-fix binary returned two different messages for the F2
  seed in 60 runs (50× `object key 7 is not a string`, 10× `object key 2 is not
  a string`); the post-fix binary returned one message 60/60
  (`object key 2 is not a string`).

## Test vacuity (would the new tests fail on pre-fix code)

Checked against the pre-fix binary rather than by re-running the suite in a
second tree:

- `TestLoadNeverReturnsConfigThatValidateRejects`: the F1 seed
  `{"schemaVersion":1,"sChemAVErsion":0}` (`f1.json`) is accepted by the pre-fix
  binary (`configuration is valid`, exit 0), so `Validate(Load(seed))` is
  reachable with `err == nil` and the invariant assertion fails pre-fix.
- `TestLoadRejectsCaseVariantDuplicateKeys`: `c2_mode.json`
  (`mode`/`MODE`) and `c2_limits.json` (`limits`/`Limits`) are accepted by the
  pre-fix binary (exit 0), so at least those rows fail pre-fix; the coder's
  throwaway-tree run reported all 13 rows failing pre-fix.
- `TestLoadDiagnosticsAreDeterministic`: the 60-run measurement above shows the
  pre-fix binary producing two distinct outcomes for corpus item 1; the coder's
  run reported 4 of 7 corpus items varying pre-fix, matching the specifier's
  prediction.
- The tests are not restatements of the implementation: they assert observable
  outcomes (error presence, `Validate` agreement, byte-identical repetition),
  and three of them are corpus-driven.

Scope check: `git status --short` shows only
`internal/config/load.go`, `internal/config/validate.go` (modified) and
`internal/config/collision_test.go`, `internal/config/determinism_test.go`
(untracked) beside the expected `.agent/` artifacts. No
`internal/config/fuzz_test.go`, and `grep -rn "func Fuzz" internal/config`
returns nothing, so FJ-040's declared symbols remain free.

## Command outcomes

| Command | Outcome |
|---|---|
| `gofmt -l internal/config/load.go internal/config/validate.go internal/config/collision_test.go internal/config/determinism_test.go` | no output |
| `go test ./internal/config -count=1` | `ok fake-jev/internal/config 0.667s`, exit 0 |
| `./scripts/candidate-fingerprint candidate` | `74245107247629f9c366043ed7aada0dfbe29f4b63157bd6218afc3ee84a83d0` (unchanged) |
| `git status --short` | config source/tests plus `.agent/` artifacts, as listed above |
| `timeout N /tmp/fj064 validate <file>` / same with `/tmp/fj064-prefix` | outcomes in the tables above |

No full-suite, race, vet, build, guard, or `go run ./cmd/guard` command was run
in this stage, per the stage budget.

## Residual risks

1. F-B is an availability defect, not only a wording/scope question: with the
   candidate as it stands, a configuration file (or a control-API stub body)
   whose object keys include one non-ASCII uncased character never returns. The
   fix is two lines in `foldKey`, but it must land together with F-A's scope
   decision, since a narrowed scan still folds every struct-level key.
2. F-A rejects legitimate emulated-provider payloads and state values; the
   correction needs a schema-directed walk, which is the "more code" the coder
   deliberately avoided. The alternative — accepting a case-variant pair inside
   `then.raw.body` while reasoning that `encoding/json` never folds it there — is
   the spec-consistent outcome, but it makes the check schema-aware.
3. The `\u01c4`/`\u01c5` divergence (missed rejection) disappears if `foldKey`
   adopts `encoding/json`'s `foldRune`; if the scope is instead narrowed by a
   different mechanism (e.g. `strings.EqualFold` comparisons), F-B's hang goes
   away as a side effect but the divergence must be re-measured.
4. The pre-existing document-wide exact-duplicate rejection (yaml.v3
   `uniqueKeys`, reached through `toJSON`'s fall-through) still rejects repeated
   exact keys inside exempt containers. It is unchanged by this candidate and
   judged defensible, but the PM may want it recorded as its own observation.
5. `then.raw.headers` (case-insensitive at the HTTP layer, canonicalised on
   write) is the one container whose treatment is a genuine judgement call rather
   than a spec-mandated one; whichever way the narrowed scan goes, the
   `Header().Set` interaction at `internal/host/http/server.go:400` is a separate
   pre-existing order-dependence, out of this item's scope.
