---
stage: cleaner
task: FJ-065
inputFingerprint: f6419cffc219fc46451b98d04986daedfa732d1f754fbe3c4dbbc116591cf587
outputFingerprint: f6419cffc219fc46451b98d04986daedfa732d1f754fbe3c4dbbc116591cf587
taskFingerprint: ee28135d80573ff713fda4993a1571871f25f3ac6424a2b7e361c32336ad06ea
gitHead: c70d55b
generatedAt: 2026-10-03T18:16:40Z
author: worker/FJ-065-cleaner
---

# FJ-065 cleaner — routing boundary, message bound, vacuity and predicate audit

Nothing in the repository was changed by this stage: `./scripts/candidate-fingerprint
candidate` still reports `f6419cffc219fc46451b98d04986daedfa732d1f754fbe3c4dbbc116591cf587`,
the same value as the front matter, and `git status --porcelain` shows only the
coder's `internal/config/load.go`, `internal/control/handlers_test.go`,
`internal/config/duplicate_key_diagnostic_test.go` plus the stage artifacts.
All probes below were run in throwaway copies (`/tmp/fjc/pre` = working tree with
`git checkout c70d55b -- internal/config/load.go`, `/tmp/fjc/post` = working
tree as-is); the repository itself was never used as a scratch area.

## 1. Headline: the JSON→YAML fallback is preserved for every non-duplicate failure

### 1.1 Routing, enumerated from the code

`toJSON` (`internal/config/load.go:68`) calls `rejectDuplicateJSON` (`:123`), which wraps
what `scanJSONValue` or `requireEOF` returns with `"decode JSON configuration: %w"`.
It then classifies that single error value:

| error value produced by `rejectDuplicateJSON` | site | route after FJ-065 |
|---|---|---|
| `*duplicateKeyError` (wrapped, unwrapped by `errors.As`) | `scanJSONValue`, `name` seen twice in one object | returned directly: `decode JSON configuration: duplicate object key "<key>"` |
| `*json.SyntaxError` from `decoder.Token()` at the document start | `scanJSONValue` first `Token()` | `decodeYAML` (unchanged) |
| `*json.SyntaxError` from `decoder.Token()` at an object key / value / element / closer | `scanJSONValue` | `decodeYAML` (unchanged) |
| `*json.SyntaxError` from `decoder.More()` | `scanJSONValue` | `decodeYAML` (unchanged) |
| `io.EOF` | `scanJSONValue` on a truncated document | `decodeYAML` (unchanged) |
| `object key is not a string` | `scanJSONValue` | `decodeYAML` (unchanged) |
| `malformed object` / `malformed array` / `unexpected delimiter` | `scanJSONValue` | `decodeYAML` (unchanged) |
| `multiple JSON documents` (second JSON value decodes cleanly) | `requireEOF` | `decodeYAML` (unchanged) |
| `*json.SyntaxError` from trailing content after a complete JSON value | `requireEOF` | `decodeYAML` (unchanged) |
| `nil` | — | document returned as JSON, unchanged |
| any other value | — | `decodeYAML` (unchanged) |

Reading of the code: the only branch that does not fall through is
`errors.As(err, &duplicate)`, so no other class can change route. The three
`scanJSONValue` strings `object key is not a string`, `malformed object/array`,
`unexpected delimiter` are unreachable through `json.Decoder` (the decoder
rejects such input itself) and are pre-existing, unchanged code — they are listed
here because they are part of the fallback enumeration, not because FJ-065
touched them.

Observed route for one document per class, post-fix (`bnd` rows of
`/tmp/fjc/probe_post.clean`, run confined to the scratch copies):

```
bnd flow yaml accepted           other bytes=45  "decode configuration: json: unknown field \"a\""      ({a: 1} — YAML route, then strict field check)
bnd flow yaml int set            other bytes=56  "convert YAML configuration: object key 2 is not a string"  ({07,2} — non-string key)
bnd json key then unquoted       other bytes=45  "decode configuration: json: unknown field \"a\""      ({"a":1, a:2} — YAML route, no JSON-scan failure reported)
bnd json trailing content        YAML  bytes=71  "decode YAML configuration: yaml: did not find expected <document start>"
bnd json multiple documents      YAML  bytes=71  "decode YAML configuration: yaml: did not find expected <document start>"
bnd json syntax error            other bytes=45  "decode configuration: json: unknown field \"a\""      ({"a": } — YAML route)
bnd json non-string key          other bytes=47  "decode configuration: json: unknown field \"1:2\""    ({1:2} — YAML route)
bnd json truncated key           YAML  bytes=75  "decode YAML configuration: yaml: line 1: did not find expected node content"
bnd flow yaml dup 3              YAML  bytes=102 1 duplicate line  ({a: 1, a: 2, a: 3})
```

The last four rows are the YAML-flow path the task called out: documents that are
not valid JSON still reach yaml.v3 and are still parsed there.

### 1.2 Boundary: a document that is both not-JSON and contains a repeated key

Both shapes were measured pre- and post-fix in the scratch copies
(`diff /tmp/fjc/probe_pre.clean /tmp/fjc/probe_post.clean`):

| document | pre-fix | post-fix |
|---|---|---|
| `{a: 1, a: 2, a: 3}` (not JSON; 3 repeats) | `decode YAML configuration: … 3 × mapping key "a" already defined at line 1` (206 B) | same route, `1 ×` the same line (102 B) |
| `{a: 1, a: 2, a: 3,}` | same, 206 B | same route, 102 B |
| `{schemaVersion: 1, schemaVersion: 1, …}` flow | YAML route, 1 line per pair | YAML route, 1 line |
| `{"a": 1, "a": 2}` (valid JSON, duplicates) | YAML route (dup discarded) | JSON route, 51 B |
| `{"a":1,"a":2,}` `{"a":1,"a"` `{"a":1,"a":2, ???}` `{"a":1,"a":2} {"b":1}` `{"a":1,"a":2} # c` | YAML route (syntax error or yaml dup) | JSON route, 51 B |
| `{"a":1, a:2}` (junk) | other (YAML route) | other (YAML route), byte-identical |

So the not-JSON-plus-repeat documents that YAML can read keep the YAML route and
gain the one-line bound, and the JSON-scan pair-detection only wins when the scan
reaches the repeat **before** it hits the offending token. `{"a":1, a:2}` is the
witness for the other order: the syntax failure at the second key happens before
any repeat, no `duplicateKeyError` is produced, and the document takes the YAML
route exactly as before.

### 1.3 No acceptance change: two document sweeps

Two deterministic sweeps were run in `/tmp/fjc/pre` and `/tmp/fjc/post` and the
`LOAD` verdicts (accepted vs rejected) compared key by key, ignoring wording:

- a fixed table of 57 documents (JSON duplicates, YAML flow, `{07,2}`, trailing
  content, multiple documents, merge keys `{"<<":1,"<<":2}`, quoted-vs-escaped
  duplicates `{"a":1,"\u0061":2}`, `{"a\/b":1,"a/b":2}`, non-ASCII and
  escaped-control keys, nested and multi-group duplicates, `null`/`[]`,
  `{1:2}`, truncations) plus a token-soup sweep of 4000 documents: 4057
  outcomes compared, **0 verdict flips**. The 88 changed lines are all `ERR`→`ERR`
  wording changes;
- a generated sweep of 3000 documents (seeded `math/rand` seed 65065, random
  nesting with duplicate keys injected, plus trailing junk and unquoted-key
  variants): 2999 rejected / 1 accepted in **both** builds, **0 verdict flips**;
  1177 of the rejections changed from `decode YAML configuration: … already
  defined` to `decode JSON configuration: duplicate object key …`, no verdict
  moved.

Class pairs across the generated sweep: `(YAML→JSON) 1177`,
`(decode configuration → decode configuration) 999`,
`(YAML→YAML) 662`, and 161 rejections whose message carries no colon-prefixed
class and was therefore identical. No case went `OK→ERR` or `ERR→OK`.

Cross-check of the direction "previously accepted must stay accepted": the
YAML route's key-uniqueness check in `gopkg.in/yaml.v3@v3.0.1/decode.go:769-780`
compares every pair in a mapping and is always enabled (`uniqueKeys: true`,
`decode.go:344`), so a document whose JSON-scan repeat is real can never have
been accepted through the YAML route; the sweeps above find no counterexample.
The library check does not exempt the merge key either — `{"<<":1,"<<":2}` is
rejected on both routes (row `case19`).

## 2. Message bound and key-content probes

Run at `Load` level (`key …` rows) and through the real control handler
(`ctl …` rows, `internal/control` scratch test reusing `testAPI`/`request`).

### 2.1 Does the message grow with the repetition count? No.

| input | response bytes |
|---|---|
| `{"id":0,…,}` N = 2 via `POST /__fake/v1/stubs` | 107 |
| same, N = 1600 (14 891 B request) | 107 |
| `Load` of `{"schemaVersion":1,"stubs":[… N times …]}` N = 2…3200 | 52 bytes at every N |

### 2.2 Key content and key length (JSON route, `msg` = `err.Error()` length, `resp` = control body)

| key | raw bytes | `Load` msg bytes | control resp bytes |
|---|---|---|---|
| `id` | 2 | 52 | 107 |
| `"k"×350` | 350 | 400 | — |
| `"k"×400` | 400 | 450 | 505 |
| `"k"×1000` | 1000 | 1050 | 1105 |
| `"k"×100000` | 100000 | 100050 | — |
| `"`×400 | 400 | 850 | 1705 |
| `\`×400 | 400 | 850 | 1705 |
| LF×400 | 400 | 850 | 1305 |
| `\x00`×400 | 400 | 1650 | 2105 |
| `\x00`×100000 | 100000 | — | 500105 (request 1 200 011 B, ratio 0.42) |
| `\x7f`×400 | 400 | 1650 | — |
| `\xff` (invalid UTF-8)×400 | 400 | 1250 | — |
| `é`×400 | 800 | 850 | — |
| `😀`×400 | 1600 | 1650 | 1705 |
| `:`×400 / `{`×400 | 400 | 450 | — |
| `k`×400 in a YAML block document | 400 | 501 (1 line) | — |

Observations that matter for the artifact's claim:

- The message length is `48 + len(%q(key))` and never depends on N; the control
  body is `53 + len(msg) + 2·(number of quote characters in msg)`, which for a plain ASCII
  key is `105 + len(key)`.
- The 512-byte bound is a function of the **key text**, not of N, and the
  boundary was located by measurement: a plain key of 406 / 407 / 408 bytes
  gives a 511 / 512 / 513-byte control body, so 512 holds for plain keys up to
  407 raw bytes (message-only, 462 raw bytes). The coder's "roughly 350 bytes or
  less" is conservative, and the specifier's `105 + len(key)` matches the
  measurement.
- Attacker-chosen content inflates the message by the `%q` escape factor, not
  beyond: measured worst case `\x00`/`\x7f` at ×4 per byte (1650 for 400 bytes)
  and `"`/`\` at ×2; invalid UTF-8 at ×3. There is **no absolute cap**, so with a
  near-2 MiB control body a ~1 MiB repeated key yields a ~0.5 MB response
  (measured: 1 200 011 B request → 500 105 B response, ratio 0.42). That is
  linear in the key length and strictly below the request size, and it does not
  grow with the repetition count. The coder's artifact states the 512-byte
  assumption only for the test bound and does not claim an absolute product cap;
  the observed numbers are recorded here so the assumption is not read as one.
- Many **distinct** repeated keys do not inflate the message: `n` distinct keys
  each repeated twice gives 52 bytes for n = 2, 4, 8, 64, 512, 4096 (only `k1`
  is named), and on the YAML route 103 bytes with exactly 1 duplicate line for
  n = 2, 8, 128 distinct repeated mapping keys.
- The duplicate-key **error is still reported**: every row above is a rejection
  (non-nil error, 400 at the control boundary). A repeated key with identical
  values (`{"id":0,"id":0}`) and with different values (`{"id":0,"id":1}`) both
  reject with 52 bytes; the generated sweep also contains both shapes and none
  was accepted. `TestLoadStillRejectsRepeatedKeys` and
  `TestControlRepeatedJSONKeyResponseStaysBounded` assert this post-fix, and the
  pre-fix run of the latter shows the same rejection with the old wording.

### 2.3 YAML route: long keys cannot reach the bound

`yamlkey long 4000` (a 4000-byte block-mapping key repeated three times) returns
`decode YAML configuration: yaml: mapping values are not allowed in this
context`, 79 bytes, because yaml.v3's scanner refuses simple keys longer than
1024 characters; the same document without repetition gives the same 79-byte
error. So on the YAML route the "long key" case is short-circuited by the
scanner before any duplicate diagnostic is built. `yamlkey long` (400 bytes)
gives 501 bytes with exactly 1 duplicate line pre- and post-bound.

## 3. Determinism

- Rule for the named key: the JSON route names the key whose second occurrence
  is reached first in token order (`seen` is used only for membership, never
  iterated, so Go map randomization cannot influence it); the YAML route keeps
  the first yaml.v3 diagnostic. Both are read from the code and confirmed by
  3000/3000 and 500/500 identical calls: `det doc0…doc3 distinct=1`,
  `det ydoc0…ydoc2 distinct=1` in `/tmp/fjc/probe_post.out`.
- Two routes, two orders — worth recording as a nuance: for
  `{"a":1,"b":1,"b":2,"a":2}` the JSON route names `b` (second occurrence
  earliest) while the same shape as block/flow YAML names `a` (yaml's pair scan
  orders by the first occurrence). Measured:
  `first json a b b a → duplicate object key "b"`,
  `first yaml a b b a block → line 4: mapping key "a" already defined at line 1`.
  The specifier's C4 prescribes exactly this split ("for a YAML document, the
  first diagnostic yaml.v3 emits"), so the implementation follows the
  prescription; the coder's §1.3 one-sentence rule ("the first key that is
  reached for the second time") describes only the JSON route and is an
  over-generalisation in the artifact prose (not in the code comments — see §5).
- No wall-clock or randomness introduced: the change contains no `time.Sleep`,
  no `time.Now`, no `rand`/map-iteration-dependent selection. A 200-iteration
  load of a 4000-repeat body (`TestZZSleepAndRandomAudit`, scratch copy) takes
  57.8 ms for the whole loop, i.e. no hidden per-call wait.

## 4. Vacuity: do the new tests fail pre-fix?

Run in `/tmp/fjc/pre` (pre-fix `load.go`, coder's test files copied in):

| new test | pre-fix verdict | pre-fix observation |
|---|---|---|
| `TestLoadBoundsRepeatedJSONKeyDiagnostic` | FAIL | message is the yaml.v3 list, not `decode JSON configuration: duplicate object key "id"` |
| `TestLoadBoundsRepeatedYAMLKeyDiagnostic` (block, flow) | FAIL both | "contains 3 duplicate-key lines, want 1" |
| `TestLoadRepeatedKeyDiagnosticAllocationStaysBounded` | FAIL | allocation bound exceeded |
| `TestLoadStillRejectsRepeatedKeys` | FAIL 4 of 6 subtests | the two 64-repeat subtests exceed 512 bytes; the two open-container subtests pass |
| `TestLoadDuplicateKeyDiagnosticIsDeterministic` | FAIL 1 of 4 subtests (`json sixteen repeated ids`) | 16 repeats → measured 6410-byte message over the bound; the other three sub-4-repeat cases pass |
| `TestLoadNamesFirstRepeatedKeyInDocumentOrder` | FAIL all 3 | pre-fix names are yaml.v3 wording |
| `TestControlRepeatedJSONKeyResponseStaysBounded` | FAIL | `repeated keys=2: response = 400 {… "decode YAML configuration: …"}` |
| `TestLoadAcceptsDocumentsWithoutRepeatedKeys` | PASS | mandated no-regression test; passes before and after by construction |
| `TestLoadDiagnostics...` extension | none | not extended, as the coder states |

The coder's claim "7 of 8 fail pre-fix, the eighth is the mandated no-regression
test" is reproduced exactly, including which subtests pass pre-fix. One numeric
detail differs from `coder.md` §3: that artifact records the pre-fix 16-repeat
message as 6 864 bytes, while the run above measures 6 410 bytes for
`{"schemaVersion":1,"id":0,…}` (`duplicate_key_diagnostic_test.go:235`).
Both figures are far above 512, so nothing depends on it. My own
non-vacuity check of the two tests the task named:

- bounded message: `TestLoadBoundsRepeatedJSONKeyDiagnostic` asserts both
  rejection and an exact 52-byte string, so the pre-fix 103-byte yaml.v3 list
  fails it (measured); it is not a restatement of the implementation because the
  expected string is fixed text, not `err.Error()` of the same code path.
- scaling: `TestLoadRepeatedKeyDiagnosticAllocationStaysBounded` fails pre-fix
  (step 1 allocates 8 877 968 B for a 1 721-byte document against a 440 576 B
  bound) and its per-step limit `256 × len(document)` is independent of the
  measured allocation.

## 5. Predicate and filter audit

`yamlDuplicateKeyDiagnostic(d) = HasPrefix(d, "line ") && Contains(d, " already
defined at line ")`. Checked against the pinned dependency
(`gopkg.in/yaml.v3@v3.0.1/decode.go`, `go.mod` pins v3.0.1):

- the only producer of `" already defined at line "` is `decode.go:776`,
  `fmt.Sprintf("line %d: mapping key %#v already defined at line %d", …)`, inside
  `decoder.mapping` — prefix `line ` present;
- the other four `terrors` producers are `cannot unmarshal %s%s into %s`
  (`decode.go:362`, whose scalar value appears as at most its first 7 characters
  inside a backticked wrapper — 13 characters, so the 24-character marker cannot
  fit — while collection nodes have an empty `Value`), `field %s already set in
  type %s` (`:924`), `field %s not found in type %s` (`:944`), and `e.Errors`
  re-appended from a nested `TypeError` (`:368` in `callUnmarshaler`, `:390` in
  `callObsoleteUnmarshaler` — neither runs on this path, where `decodeYAML`
  targets `any` and no type implements `Unmarshaler`);
- empirical check (`pred …` rows, `/tmp/fjc/probe_post.out`): of the real
  diagnostics produced by duplicate block, duplicate flow, nested duplicate,
  merge-key duplicate and scanner-level failures, only the uniqueness lines
  matched; scanner failures arrive as non-`TypeError` values and bypass the
  filter entirely.
- filter unit checks (scratch test, all held): exactly the first duplicate line
  is kept and the remaining `C(N,2)` lines dropped; unrelated diagnostics before,
  between and after duplicate lines are preserved in their original order; a
  line containing the marker but not the `line ` prefix is kept; a line with the
  prefix but without the marker is kept; a document with two *distinct*
  duplicated keys keeps only the first duplicate line (consistent with C2's
  "reports only the first"); a `TypeError` with no duplicate line is returned
  unchanged; a non-`TypeError` is returned unchanged; applying the filter twice
  is a no-op.
- only one YAML decode site exists on any load path (`internal/config/load.go:107`;
  `grep` for `yaml.NewDecoder|yaml.Unmarshal` finds no other), so wrapping the
  error there covers the `validate`, `serve --config` and control paths.

Comment audit (no changes made):

- `toJSON`'s new comment ("diagnostic is already complete, linear, and in
  document order … one diagnostic for every repeated-key pair") matches the code
  and the measurements.
- `boundDuplicateKeyDiagnostics`'s comment says "the first one yaml.v3 produced,
  which its document-order pair scan makes a function of the document alone" —
  accurate as written and confirmed by the determinism sweep; the over-claim is
  only in `coder.md` §1.3, which compresses both routes into one sentence (see
  §3 above).
- `yamlDuplicateKeyDiagnostic`'s comment ("no other yaml.v3 diagnostic contains
  the second one") is accurate for v3.0.1 as read above; it is a textual
  dependency on the pinned library's wording, see risks.
- Dead code: none added. `duplicateKeyError` and its `Error()` method, the
  predicate, the filter, both test constants, `repeatedKeysThatExceedTheBound`
  and the two test helpers are all referenced. The `strings` import added to
  `load.go` is used only by the predicate. The pre-existing unreachable strings
  in `scanJSONValue` (§1.1) were left untouched on purpose — removing them would
  change the candidate fingerprint for no behavioural reason.

## 6. Cleanups applied

None. Structure was reviewed against `staff-architect` judgement and every
candidate move was rejected for one of these reasons:

- folding `duplicateKeyError` into a sentinel would lose the key name that C4
  requires the message to carry;
- moving the filter onto `decodeYAML`'s return would change the wrapper text and
  touch shared code for no gain, and the single call site is already the only
  place the error is formatted;
- any re-grouping of the new test file or its helpers would invalidate the
  candidate fingerprint and the coder's measurements without changing behaviour.

Behaviour-preserving opportunity explicitly weighed and rejected: sorting the
kept YAML diagnostic by key name (as FJ-064's `sortedKeys` does) — the yaml
diagnostic list is already totally ordered, the specifier prescribes the
first-diagnostic rule for YAML, and a sort would change which key is named.

## 7. Commands and outcomes

| command | observation |
|---|---|
| `git status --porcelain`, `git diff --cached --name-only` | only coder files + stage artifacts; nothing staged |
| `./scripts/candidate-fingerprint candidate` | `f6419cff…` (unchanged) |
| `gofmt -l internal/config internal/control` | no output |
| `grep -rn "time.Sleep\|rand\." internal/config internal/control` (non-test) | no match |
| `find internal -name '*fuzz*'`, `grep -rn "func Fuzz" internal` | only the pre-existing `internal/compat/jev/v1/fixture_test.go:378`; no new fuzz target or `fuzz_test.go` |
| `go test ./... -count=1` (single run, repository tree) | see §8 |
| `go test ./internal/config -count=1 -run '<the 7 new config tests>' -v` in `/tmp/fjc/pre` | 6 FAIL, 1 PASS (table in §4) |
| `go test ./internal/control -count=1 -run TestControlRepeatedJSONKeyResponseStaysBounded -v` in `/tmp/fjc/pre` | FAIL with the old wording |
| `go test ./internal/config -count=1 -run '<scratch probes>' -v` in `/tmp/fjc/pre` and `/tmp/fjc/post` | verdicts compared; §1.3 |
| `go test ./internal/control -count=1 -run TestZZControlHostileKeySizes -v` in `/tmp/fjc/post` | §2.2 table |

`./scripts/verify-candidate`, the race suite and `go run ./cmd/guard …` were not
run in this stage (reserved for the hardener and QA stages by the task packet).

## 8. Full suite

`go test ./... -count=1` in the repository tree: **PASS** — `ok` for
`cmd/guard`, `internal/cli`, `internal/compat/jev/v1`, `internal/config`,
`internal/control`, `internal/engine`, `internal/host/http`,
`test/integration`; `no test files` for `cmd/fake-jev`, `examples/go`. Run once,
as the task packet required; `gofmt -l`, `git status` and the candidate
fingerprint were re-checked afterwards and are unchanged.

## 9. Residual risks

1. **The predicate is a text match on yaml.v3's wording.** v3.0.1 is pinned in
   `go.mod`, but an upgrade that reworded `decode.go:776` would make
   `yamlDuplicateKeyDiagnostic` return false, the filter would keep the full
   quadratic list, and the new tests (also pinned to that wording indirectly via
   `already defined at line`) would be the only signal. Not a defect now.
2. **No absolute message cap.** The bound is constant in N but linear in the key
   text and its `%q` escaping; a near-2 MiB control body can produce a ~0.5 MB
   response (measured 0.42×). Linear, strictly below request size, but larger
   than the 512-byte test bound.
3. **yaml.v3's decode cost is still quadratic for YAML input**, as the coder's
   artifact records; only the echoed diagnostic is bounded here. The yaml route
   requires a local file or a non-JSON flow document, so the reachable
   untrusted-input path is the JSON one.
4. **Wording and route changes for invalid-JSON documents that contain a
   repeat**: `{"a":1,"a":2, ???}`, `{"a":1,"a"}`, `{"a":1,"a":2,}` now report
   `decode JSON configuration: duplicate object key "a"` even though the document
   is not valid JSON. The decision (reject), status, code and exit code are
   unchanged and the message is human-readable and names the key, but the
   "JSON configuration" prefix is no longer strictly true for those inputs.
5. **Two routes name a different repeated key** for documents with several
   duplicated keys (§3). Consistent with the specifier's C4; it is a message
   difference between input syntaxes, not a decision difference.
6. **FJ-040's test-side bound** (`configFuzzMaxMappingKeys`) does not exist on
   this branch; nothing was reconciled with it here.
