---
stage: qa
task: FJ-040
inputFingerprint: 8961ce46838962ae48d5bf900aa2013b9008df3c595b84cfa6e72c614d7cc434
outputFingerprint: 8961ce46838962ae48d5bf900aa2013b9008df3c595b84cfa6e72c614d7cc434
taskFingerprint: ae3ccc43bc6a790f3329233e8585379262d6e5b49e2751da583353742d06c6e4
gitHead: 433fc97
generatedAt: 2026-10-03T19:33:00Z
author: worker/FJ-040-qa-2
---

# FJ-040 QA (second independent pass): control-target seed, config guard, non-vacuity

Author differs from the coder (`worker/FJ-040-coder`), the first hardener
(`worker/FJ-040-hardener`) and the second hardener (`worker/FJ-040-hardener-2`).
This artifact states observations, measurements, and limits. It declares no verdict,
no acceptable/unacceptable judgement, and no pass/fail wording about the item.
Nothing in the repository was modified by this stage except this file.

## Fingerprints and scope

| observation | value |
| --- | --- |
| `git rev-parse HEAD` | `433fc974544cbf912a93b3f50cd2011bdd9511fa` (`433fc97`) |
| hardener.md `outputFingerprint` (read from the artifact front matter and body) | `8961ce46838962ae48d5bf900aa2013b9008df3c595b84cfa6e72c614d7cc434` |
| `./scripts/candidate-fingerprint candidate` | `8961ce46838962ae48d5bf900aa2013b9008df3c595b84cfa6e72c614d7cc434` |
| `candidateFingerprint` in the verifier report produced by this stage | `8961ce46838962ae48d5bf900aa2013b9008df3c595b84cfa6e72c614d7cc434` |

The artifact value and the workspace digest agree, so this stage takes `8961ce46…`
as both `inputFingerprint` and `outputFingerprint` (this stage changed no file that
the fingerprint covers).

Scope, observed directly:

- `git diff --stat 433fc97 -- .` → one path, `.agent/work/FJ-040/state.json`
  (35 insertions, 10 deletions). That file is work-item metadata, was already
  modified before this stage, and is not product code, a script, a guard config, or
  a test.
- `git status --short` → modified `state.json`, untracked `.agent/work/FJ-040/*.md`
  and `.agent/reports/FJ-040/report-*.json`, and exactly four untracked test files:
  `internal/engine/race_test.go`, `internal/control/fuzz_test.go`,
  `internal/compat/jev/v1/fuzz_test.go`, `internal/config/fuzz_test.go`.
  `git diff --cached --stat` is empty (zero staged paths).
- `find . -name fuzz_test.go` → `./cmd/guard/fuzz_test.go` (pre-existing, tracked,
  unchanged), plus the three new ones under `internal/control`, `internal/config`,
  `internal/compat/jev/v1`. `internal/engine` declares no fuzz target.
- No symbol collision: the new file in `internal/config` defines only
  `FuzzConfigMalformedInput`, `configFuzzMaxBytes`, `configFuzzMaxMappingKeys`,
  `configFuzzMappingKeys`, `configFuzzYAMLPath`, `configFuzzBeyondKeyBound`,
  `assertLoadedConfigDefaults`; none of those appear in `collision_test.go`
  (FJ-064: `exactSpellingDocument`, `mustJSONString`, `loadAccepted`,
  `assertRawPayload`, `foldSample`) or `duplicate_key_diagnostic_test.go`
  (FJ-065: `repeatedKeyObject`, `repeatedKeyConfiguration`, `repeatedKeyYAML`,
  `allocatedBytes`, `loadYAMLDiagnostic`, `maxDuplicateKeyDiagnosticBytes`).
  `internal/control` defines `newFuzzedAPI`, `controlRawRequest`,
  `readOnlyControlRoute`, `doControl`, `controlCall`, `FuzzControlMalformedInput`;
  `internal/compat/jev/v1` adds `assertGeneratedAnswers` and
  `FuzzProfileUntrustedInput` beside FJ-059's `FuzzValidateFixtureAnswers`;
  `internal/engine` adds `concurrentDynamicIDs` and `engineReaderSnapshot`.
  A package-level duplicate would be a compile error, and the package compiles.

Focused sources read in full: the four files under test, `.agent/work/FJ-040/`
`specifier.md`, `coder.md`, `cleaner.md`, `hardener.md`, `state.json` notes,
`docs/agents/roles/qa.md`, and the product seams the guard mirrors
(`internal/config/load.go` `toJSON`/`rejectDuplicateJSON`, `internal/control/handlers.go`
`createStub`/`decodeJSON`, `internal/engine/engine.go`, `journal.go`).

Artifact availability: `.agent/work/FJ-040/hardener.md` in the worktree holds only the
second pass (its heading is "FJ-040 hardener (second pass)", sections 1–7); the first
pass's sections 3 and 4 are not in the file, so the first-pass guard and its
measurements could only be read through the second pass's §2–§4 quotations of them.
The reconstruction of the old guard used below is therefore inferred from that prose
(colon-only count above 256, with any `rejectDuplicateJSON` failure treated as a YAML
document), not from the first pass's source.

## Headline observation: the new control regression seed does not carry the shape it names

`internal/control/fuzz_test.go` registers the seed as

```go
{"repeated-key-bomb", http.MethodPost, "/__fake/v1/stubs", "application/json",
    `{"` + strings.Repeat(`"a":1,`, 1600) + `"a":1}`},
```

`"{"` is two bytes (`{` and `"`), and the repeated literal already starts with `"`,
so the body begins `{""a":1,"a":1,…`: JSON with an empty object key followed by an
unexpected `a`. Measured in a throwaway copy of the repository
(`/private/tmp/fj040-qa2/probe`), driving `API.ServeHTTP` exactly as the target does
with the target's own `controlRawRequest`/`newFuzzedAPI` helpers:

```text
QA-PROBE4 seed-exact-as-committed    len=9608 jsonValid=false firstBytes="{\"\"a\":1,\"a\":1,..." status=400 responseBytes=77 response={"error":"fake_jev_bad_control_request","message":"Invalid control request."}
QA-PROBE4 corrected-duplicate-key-a  len=9607 jsonValid=true  firstBytes="{\"a\":1,\"a\":1,\"a\":1,..."  status=400 responseBytes=106 response={"error":"fake_jev_bad_control_request","message":"decode JSON configuration: duplicate object key \"a\""}
QA-PROBE4 id-key-duplicate           len=11208 jsonValid=true                                        status=400 responseBytes=107 response={"error":"fake_jev_bad_control_request","message":"decode JSON configuration: duplicate object key \"id\""}
```

- The committed seed's 9,608 bytes are rejected by `decodeJSON(r.Body, &source)` in
  `createStub`, before `validationDocument` or `config.Load` runs, so they take the
  generic `Invalid control request.` path (77 bytes) and never reach the
  duplicate-key handling FJ-065 changed.
- The intended shape (one `"` fewer) is 9,607 bytes — exactly the body size the
  hardener artifact reports in its JOB 1 table — and answers the constant 106-byte
  duplicate-key diagnostic. The hardener's measurement therefore used a different
  literal than the one committed.
- The 9,608-byte committed shape is answered with a constant 77 bytes at every
  repetition count tried (16, 64, 128, 256, 512, 1600, 2730 repeats; body 104 →
  16,388 bytes; elapsed 0.9 µs–6.3 µs), so the observation "the previously-failing
  shape is now bounded" holds for the *intended* shape but not for the seed as
  written.

Decisive A/B in a second throwaway copy (`/private/tmp/fj040-qa2/mut`) with the
FJ-065 behaviour reverted (the `errors.As(err, &duplicate)` short-circuit removed from
`toJSON`, and `boundDuplicateKeyDiagnostics` made a no-op):

```text
A) committed seed, pre-FJ-065 product:
   go test -run '^FuzzControlMalformedInput$' ./internal/control -count=1
   ok  	fake-jev/internal/control	0.200s

B) same seed with the stray quote removed, pre-FJ-065 product:
   go test -run '^FuzzControlMalformedInput$' ./internal/control -count=1
   --- FAIL: FuzzControlMalformedInput (0.43s)
       --- FAIL: FuzzControlMalformedInput/seed#10 (0.43s)
           fuzz_test.go:158: control request POST "/__fake/v1/stubs" answered 70444103 bytes
```

Reading of the A/B: the committed seed executes under plain `go test` and reports
green, but it is inert with respect to the regression it is documented to guard —
it stayed green under a product that reproduces the pre-FJ-065 behaviour. The same
seed with the quote fixed fails on that product with a 70,444,103-byte response; the
hardener artifact records 70,356,103 bytes for its 9,607-byte construction (the two
figures differ by 88,000 bytes, consistent with slightly different body literals at
the same repetition count). The response-bound assertion itself is therefore live;
the seed content is the defect.

Reproduction of the committed shape, independent of the test file, is possible with
`json.Valid([]byte(`{"`+strings.Repeat(`"a":1,`,1600)+`"a":1}`)) == false`, and the
minimal repair is to drop the leading `"` from the literal (a 9,607-byte body).

### Seed execution, stability, and bounded response (as requested)

- Seed execution under plain `go test`:
  `go test -run '^FuzzControlMalformedInput$' -v ./internal/control -count=1`
  reports `--- PASS: FuzzControlMalformedInput/seed#10 (0.00s)` (`ok … 0.581s`), and
  the prebuilt binary
  (`go test -c -o /private/tmp/fj040-qa2/bin/control.test ./internal/control`, built
  once and reused) with
  `-test.v -test.run '^FuzzControlMalformedInput$'` reports `PASS` for
  `seed#0` … `seed#15`; the seeds are registered in source order, so `seed#10` is
  `repeated-key-bomb`. The cache baseline line for the same corpus is
  `gathering baseline coverage: 0/548 completed` → `… 548/548 completed`. The file
  hashes of the four files at the end of this stage are recorded below so a later
  reader can tell whether they moved.
- Response bound for the exact committed shape: 77 bytes, constant, status 400,
  under the current revision (probe above). For the intended shape: 106 bytes,
  constant.
- Repeated 30-second runs of the control target (`go test -run '^$' -fuzz
  '^FuzzControlMalformedInput$' -fuzztime 30s ./internal/control`), four runs:

| run | exit | wall | final execs | new interesting (total) |
| --- | --- | --- | --- | --- |
| 1 | 0 | 31.66 s | 1,366,168 | 2 (550) |
| 2 | 0 | 31.58 s | 2,328,424 | 5 (555) |
| 3 | 0 | 31.57 s | 2,347,118 | 7 (562) |
| 4 | 0 | 31.63 s | 1,535,495 | 1 (563) |

  Each run printed every 3-second checkpoint with the exec counter advancing (rates
  15k–130k/sec); the only `0/sec` line is the terminal 31 s line in each run, where
  the reporter's last interval is zero. No run wrote a failing input:
  `internal/control/testdata` does not exist after the runs. The control target's
  coordinator counter did not freeze.
- Shape sweep near and beyond the target's 16 KiB cap (2,730 repeats, 16,388 body
  bytes):
  no status outside {400}, no response above 107 bytes, no 5xx, at any repetition
  count.

## Guard-routing falsification in `internal/config/fuzz_test.go`

The reshaped guard was checked against the product's own routing, then against
documents of each shape class. `toJSON` (read in `internal/config/load.go`) routes a
trimmed document that starts with `{` or `[` to `rejectDuplicateJSON`; on a clean
accept it returns the bytes for `encoding/json`, on a `*duplicateKeyError` it returns
that error, and on any other scan failure it falls through to `decodeYAML`. Every
other document goes to `decodeYAML`. `configFuzzYAMLPath` reproduces exactly that
(including `errors.As` on the duplicate type), so the routing mirror reproduces the
product's branch decision on every shape measured below:
documents that take the JSON path — accepted or duplicate-key-rejected — report
`yamlPath=false` and are never skipped, and the FJ-065 duplicate-key class stays
fuzzable at any size up to `configFuzzMaxBytes`. That was checked in both directions:
`rejectDuplicateJSON` returns nil for 49,781 bytes of valid JSON with 4,000 distinct
keys and for an 18,891-byte JSON array, and both are `beyond=false` despite carrying
7,999 / 3,999 separators, so the fuzzer sees large well-formed JSON documents as well
as repeated-key ones.

Measured with in-package probes in the throwaway copy (`QA-GUARD` lines; `seps` is
`configFuzzMappingKeys` = colons + commas):

| shape | bytes | colons | commas | seps | yamlPath | beyond (skipped) |
| --- | --- | --- | --- | --- | --- | --- |
| JSON repeated key ×1600 | 9,607 | 1,601 | 1,600 | 3,201 | false | false |
| JSON repeated key ×6400 | 38,407 | 6,401 | 6,400 | 12,801 | false | false |
| JSON repeated key ×8000 | 48,007 | 8,001 | 8,000 | 16,001 | false | false |
| valid JSON, 4,000 distinct keys | 49,781 | 7,999 | 4,000 | 11,999 | false | false |
| valid JSON array, 4,000 items | 18,891 | 0 | 3,999 | 3,999 | false | false |
| colon-free flow mapping ×260 | 523 | 0 | 260 | 260 | true | true |
| colon-free flow mapping ×1000 | 2,003 | 0 | 1,000 | 1,000 | true | true |
| colon-free flow mapping ×13100 | 26,203 | 0 | 13,100 | 13,100 | true | true |
| flow mapping with colons ×260 | 1,045 | 261 | 260 | 521 | true | true |
| flow sequence ×13100 | 26,203 | 0 | 13,100 | 13,100 | true | true |
| block repeated key ×256 | 1,280 | 256 | 0 | 256 | true | false |
| block repeated key ×257 | 1,285 | 257 | 0 | 257 | true | true |
| distinct block keys ×256 / ×257 | 2,340 / 2,350 | 256 / 257 | 0 | 256 / 257 | true | false / true |
| block sequence items ×8000 | 32,000 | 0 | 0 | 0 | true | false |
| deep flow nesting 20,000 | 20,000 | 0 | 0 | 0 | true | false |
| block-style alias escalation 5×10 / 6×10 / 7×10 | 390 / 458 / 526 | 6 / 7 / 8 | 0 | 6 / 7 / 8 | true | false |
| explicit-key YAML (`? a` / `: 1`) ×300 | 2,400 | 300 | 0 | 300 | true | true |
| mixed 200 block + 200 flow entries | 1,403 | 200 | 200 | 400 | true | true |
| single-key `a: 1` / empty / colon-free scalar | 5 / 0 / 14 | 1 / 0 / 0 | 0 | 1 / 0 / 0 | true / false / true | false |

Two claims from the second hardener artifact, re-established independently:

1. The first pass's guard was unsound for the class FJ-065 fixed.
   `QA-OLDGUARD json-repeated-key x8000 bytes=48007 oldRoutingWouldSayYAML=true
   duplicateReject=true newBeyond=false`: under the reconstructed old routing any
   `rejectDuplicateJSON` failure counted as a YAML document, and 8,001 colons is above
   the bound, so the old guard skipped a 48,007-byte JSON repeated-key document. The
   shake-out is visible in cost: at this revision the same document costs 18 µs and
   ~3 KB of allocation per `Load` because it is answered by the JSON scan, while the
   pre-FJ-065 path for this class was the 3-second/8.5 GB shape the cleaner and the
   first pass measured. The new guard fuzzes it (`beyond=false`).
2. The colon-only proxy missed a colon-free flow mapping.
   `QA-OLDGUARD colon-free-flow x13100 bytes=26203 colons=0 oldBeyond=false
   newBeyond=true`: 13,100 flow entries with no colon at all, which the hardener
   measured at 17.4 s / 13.9 GiB in one `Load`. The `':'`+`','` proxy now skips it.

Boundary and escape findings:

- Threshold behaviour is as designed except for a one-entry asymmetry: a document
  with exactly 256 separators is fuzzed, 257 is skipped (`block repeated key ×256` vs
  `×257`, `distinct block keys ×256` vs `×257`). The cost difference at that boundary
  is milliseconds, so the asymmetry is not itself a hazard.
- Cheap documents are skipped, i.e. coverage is lost, not correctness:
  `colon-free flow mapping ×260` (523 bytes) costs 9.4 ms / 2.98 MB per `Load` and is
  skipped; `explicit-key YAML ×300` (2,400 bytes) is skipped; `distinct block keys
  ×257` is skipped. Conversely `yaml-distinct-block ×8000` (93,780 bytes) is above
  `configFuzzMaxBytes` and unreachable by the byte cap regardless of the guard.
- Cost that remains reachable inside the guard, measured per `Load`: block repeated
  key ×256 = 3.6 ms / 3.8 MB (and ×257, which is skipped, is the same 3.7 ms);
  `json-repeated-key ×8000` (JSON path, fuzzed) = 18 µs; the target calls `Load` twice
  per iteration, so an iteration inside the bound is a few milliseconds. Just above
  the bound, `colon-free flow mapping ×260` costs 9.4 ms / 2.98 MB and
  `×1000` costs 74.6 ms / 57 MB; the ×13,100 case was not re-`Load`ed here, so its
  17.4 s / 13.9 GiB figure remains the hardener's measurement.
- Shapes with few or no separators that are *not* skipped were measured for cost and
  none was expensive: `block sequence items ×8000` 2.2 ms / 2.6 MB; `deep flow
  nesting 20,000` 3.8 ms / 3.0 MB (`yaml: exceeded max depth of 10000`);
  `alias escalation 5×10, 6×10, 7×10` 0.28 ms / 0.25 MB each, answered
  `yaml: document contains excessive aliasing`. The last row matters as a limit: the
  guard does not cover anchor/alias expansion at all (such documents can have almost
  no separators), and the effective bound there is yaml.v3's own alias-ratio check
  (`gopkg.in/yaml.v3@v3.0.1/decode.go:468-490`, ratio 0.99 for small documents,
  collapsing to 0.10 near 4M decoded nodes), which caps alias-driven decodes at
  roughly 400k per document. A document that stays just inside that ratio can still
  expand to tens of MB per `Load` (the cleaner measured 51.7 MB from 62,837 bytes),
  and no shape I tried escaped the guard while remaining expensive.

Corpus cost spot check (the `1,742 entries / 0 newly skipped` claim):

- Walking the shared fuzz cache read-only across all three cached config targets:
  `QA-CORPUS perTarget=map[FuzzConfigMalformedInput:789 FuzzZZTempLoadInvariant:496
  FuzzZZTimedConfig:496]`, `total=1781 oldSkipped=2 newSkipped=2 newlySkipped=0
  newlyFuzzed=0`. Two entries are skipped under both the old and the new guard, both
  290–296 bytes with 257 separators and `yamlPath=true`; none is newly skipped, and
  none that the old guard skipped became fuzzable. The `0 newly skipped` claim holds.
  Two qualifications: the total depends on which directories are counted (the cache
  also holds `FuzzZZTempLoadInvariant` and `FuzzZZTimedConfig` from probe targets that
  no longer exist in the source tree, and `FuzzConfigMalformedInput` grew from 789 to
  807 entries during this session), so `1,742` is a snapshot of the same cache rather
  than a property of the target; and the count is dominated by cache contents, so it
  is not reproducible from the repository alone.

## Non-vacuity experiments (throwaway copies only; repository untouched)

Method: the repository was copied to `/private/tmp/fj040-qa2/mut`; one product fault
was injected per row with a single-match text replacement; the targeted test was run
there; the file was restored afterwards. Nothing under
`/Users/mak/git/fake-jev-FJ-040` was mutated by any row.

| id | injected fault | command (in the copy) | observation |
| --- | --- | --- | --- |
| M1 | `e.mu.Lock()`/`defer e.mu.Unlock()` removed from `Engine.transition` | `go test -race -count=1 -run '^TestConcurrentTransitionsRetainExactlyTheAdmittedSequences$|^TestConcurrentRegistrationSelectionJournalAndResetAreRaceFree$' ./internal/engine/` | 35 `WARNING: DATA RACE` blocks; `race_test.go:317: admitted 256 of 256 calls, want exactly 64` |
| M2 | `interactionJournal.full()` returns only `j == nil` (capacity bound gone) | `go test -count=1 -run '^TestConcurrentTransitionsRetainExactlyTheAdmittedSequences$'…` (no `-race`) | `race_test.go:317: admitted 256 of 256 calls, want exactly 64` |
| M3 | FJ-065 reverted in the product (`toJSON` duplicate-key short-circuit removed, `boundDuplicateKeyDiagnostics` no-op) with the intended 9,607-byte duplicate-key seed | `go test -run '^FuzzControlMalformedInput$' ./internal/control -count=1` | `fuzz_test.go:158: … answered 70444103 bytes` |

M1 shows the engine-side race coverage and the conservation assertion are both live
(the same fault produces race reports and an exactly-64 admission failure). M2 shows
the conservation assertion fires without any race, i.e. it is not merely a race
proxy. M3 shows the control target's 64 KiB response bound is live and that an
oversized response is caught — and, by contrast with row A of the A/B above, that
the committed seed cannot produce it.

## Determinism, hygiene, and command outcomes

File hashes of the four files under test at the end of this stage (this stage wrote
none of them):

```text
534141def7a50ab65ee4f3428e7f6d5996ab6a958c59d1524b06416fc7d45233  internal/control/fuzz_test.go
932a8eec55e382a5b2adbf450e78c159f5c7f60296008a4db6c7a28071dec3c8  internal/config/fuzz_test.go
c757a234551693962e1673b718ac584e2fae3eee672e32d7193022e213486911  internal/engine/race_test.go
c72cf7e9619e9fdfe30ec3ee3e9dfacd5a2a5112a42ce839e7f1eb40edf854e1  internal/compat/jev/v1/fuzz_test.go
```

| command | observation |
| --- | --- |
| `gofmt -l` on the four files | no output (exit 0) |
| `grep -nE "time\.Sleep\|time\.After\|time\.Tick\|NewTimer\|rand\.\|time\.Now\|select \{\|os\.Getenv\|GOMAXPROCS\|runtime\.NumCPU"` on the four files | no match (exit 1): no sleep-as-sync, no timer, no `select`, no unseeded randomness, no wall-clock read, no CPU-count dependence |
| `go test ./... -count=1` | exit 0, one `ok` line per package, 10 packages, `real 24.13` |
| `go test -race ./... -count=1` | exit 0, 10 packages, `real 28.09`, zero `WARNING: DATA RACE` lines |
| focused four packages, `-count=1` × 3 | exit 0 each time (engine/control/v1/config all `ok` on all three repeats) |
| `go vet ./...` | exit 0, no diagnostics |
| `go run ./cmd/guard arch` | exit 0, `findings: []`, 67 files / 10 packages |
| `go run ./cmd/guard lint` | exit 0, `findings: []`, 67 files / 10 packages |
| `go run ./cmd/guard trace` | exit 0, `findings: []`, 29 test files, 17 covered / 0 active |
| `go run ./cmd/guard fuzz` (once) | exit 0, `findings: []`, `stats.targets: 4`, `real 125.91`, `user 971.67` |
| `./scripts/verify-candidate FJ-040` | exit 0, `RESULT: PASS`, 36 rows = 35 `pass`, 1 `skipped`, 0 `fail`, 0 `not_applicable`; report `.agent/reports/FJ-040/report-20261003T193026Z.json` bound to fingerprint `8961ce46…` |
| `./scripts/candidate-fingerprint candidate` | `8961ce46838962ae48d5bf900aa2013b9008df3c595b84cfa6e72c614d7cc434` |

Verifier rows: the skipped row is `stage.cleaner.tooling`
("complexity/CRAP analyzer not implemented yet", gate G-L) — a tooling exemption, not
work of this item. No row is `not_applicable` and no row is `fail`, so there is no
per-row explanation to give. The run confirms, consistently with the second hardener
artifact, that this verifier revision has **no** `qa` or `hardener` stage row, **no**
`go test -race` row, and **no** guard-fuzz row; the race battery, the fuzz battery,
and this artifact's own evidence are therefore not bound by any verifier row
(36 rows total: 35 pass + 1 skip). The row set is otherwise identical to the one the
hardener listed.

Note on the guard-fuzz row: it is a single exit-0 observation at this revision
(targets 4, findings `[]`). Its stability across revisions is not something one run
establishes; the control target's four repeated runs above are the stability evidence
this stage adds.

Note on verifier runs: the verifier was invoked twice, once with output captured to
`.agent/reports/FJ-040/report-20261003T193026Z.json` (the summary above) and once more
to confirm the exit status after the last artifact edit; the second invocation exited
0 as well and wrote `report-20261003T193457Z.json` with the same row set and the same
candidate fingerprint. Both are verifier-owned outputs, not artifact content.

## Residual risks and evidence limits

- **The YAML duplicate-key quadratic remains in the product and is guarded around,
  not asserted against.** The guard skips YAML-path documents above 256 separators;
  inside the bound the fuzzer still reaches 3.6 ms / 3.8 MB per `Load` (block
  repeated key ×256, measured), and just above it a 1,000-entry colon-free flow
  mapping already cost 74.6 ms / 57 MB in my probe (the hardener measured 17.4 s /
  13.9 GiB at 13,100 entries). Nothing in the
  four files asserts a bound on that decoding cost; the item's protection is a
  skip. I did not re-`Load` the 13,100-entry case (14 GiB on a 24 GiB host already
  under load), so that figure remains the hardener's measurement, not mine.
- **Anchor/alias expansion is outside the guard entirely.** Documents that use
  anchors and aliases can have almost no `:`/`,` separators, so the guard cannot
  bound them; the effective bound is yaml.v3's own alias-ratio check (~400k
  alias-driven decodes), which still permits tens of MB per `Load` from a few KB of
  input. Measured escalation shapes were stopped cheaply by yaml.v3, but that is the
  dependency's limit, not the test's.
- **The committed control seed is inert for the regression it documents** (headline
  section). Consequence to weigh: the FJ-065 duplicate-key class is covered on the
  control route only by fuzzer-generated inputs, not deterministically by the seed
  corpus, which is the property the second hardener artifact claims for it.
- **The Go-fuzzer coordinator counter freeze is harness-side and reproduces.** Two
  config-target runs at this revision both exited 0 after 31.6–31.9 s, and both
  reported the counter stopping after ~82k/119k execs at 3 s and staying at
  `0/sec` through the 31 s line (baseline corpus 807 and 816 entries). Four
  control-target runs on the same host showed no freeze, so the behaviour is
  target-specific; the mechanism was not identified, and this host was running
  several heavy jobs concurrently, so host load is not excluded. It makes the
  config target's reported throughput unrepresentative and would mask a real
  slowdown in that target.
- **The control-level `stubMutations` lock is not falsifiable through
  `TestControlStateRaceFree`.** Removing it does not make any assertion in the four
  files fail as far as the previous QA and hardener passes found, and no test here can
  observe it; the engine-level mutex is falsifiable (M1/M2), the control-level one is
  not proven by this item.
- **FJ-030 (unbounded configuration read via `LoadFile`/`os.ReadFile`) and FJ-021
  (post-reset `invalid_stub_response` recording race) remain open** and were not
  touched. The config target calls `Load([]byte)` only, and the engine test takes its
  "no failures after reset" assertion after all workers join and a final `Reset`, so
  neither is asserted away or fixed here.
- **§22.2's matcher-input fuzzing area remains uncovered** (`allowedFiles` contains no
  `internal/engine` fuzz file).
- **Fuzz evidence is a 30-second, coverage-guided sample per target.** Four
  consecutive control runs and the single guard-fuzz row bound the risk at this
  revision; they do not eliminate it. Corpus-backed claims (baseline counts, skip
  counts, `0 newly skipped`) read a shared, mutable warm cache and are snapshots:
  `FuzzConfigMalformedInput` grew 789 → 807 entries and the control corpus 548 → 563
  during this session.
- **The 9,607 vs 9,608 byte discrepancy** means the hardener artifact's JOB 1
  measurements describe a literal that is not the literal in the file. Any future
  pass reading either number should re-derive the body from the source line rather
  than from the artifact.
- Evidence was produced by one host under concurrent load; timings (wall times, the
  freeze observation, per-`Load` costs) are single-sample and load-sensitive, though
  every assertion-level observation above (races, assertion messages, response
  sizes, guard decisions) is deterministic in form.
