---
stage: coder
task: FJ-065
inputFingerprint: f6dfb54c6debeda144bf07f521f013c381ef84fa812e6bc63756863113ec56d1
outputFingerprint: f6419cffc219fc46451b98d04986daedfa732d1f754fbe3c4dbbc116591cf587
taskFingerprint: ee28135d80573ff713fda4993a1571871f25f3ac6424a2b7e361c32336ad06ea
gitHead: c70d55b
generatedAt: 2026-10-03T18:10:07Z
author: worker/FJ-065-coder
---

# FJ-065 coder — the duplicate-key amplification on the paths we own

`outputFingerprint: f6419cffc219fc46451b98d04986daedfa732d1f754fbe3c4dbbc116591cf587`

(identical to the candidate fingerprint recorded in
`.agent/reports/FJ-065/report-20261003T180747Z.json`)

## 1. The change

`internal/config/load.go` is the only product file touched; `internal/control/`
gained one test and no production change. Nothing else changed.

### 1.1 The JSON path no longer delegates a known duplicate key to yaml.v3

`toJSON` previously discarded every `rejectDuplicateJSON` failure and fell
through to `decodeYAML`, whose key-uniqueness check emits one diagnostic per
repeated-key pair. A repeated key is now a distinct error type,
`duplicateKeyError`, returned by `scanJSONValue`, and `toJSON` reports it
directly:

```go
err := rejectDuplicateJSON(trimmed)
if err == nil {
    return trimmed, nil
}
var duplicate *duplicateKeyError
if errors.As(err, &duplicate) {
    return nil, err // "decode JSON configuration: duplicate object key \"id\""
}
```

This keeps the §12 YAML-flow fallback: any **other** `rejectDuplicateJSON`
failure (JSON syntax that may be YAML flow syntax, trailing or multiple
documents) still falls through to `decodeYAML` unchanged. The duplicate branch
is reachable only when the JSON token scan has already parsed the document
successfully up to the repeated key, so the message is complete for that
failure and the work stops at the first repeat — it does not even read the rest
of the body.

### 1.2 The YAML path keeps exactly one yaml.v3 duplicate diagnostic

Block YAML never passes through the JSON scan, so it still reaches yaml.v3, which
still builds `N*(N-1)/2` duplicate-key diagnostics for one repeated key.
`boundDuplicateKeyDiagnostics` filters the `*yaml.TypeError` that `decodeYAML`
returns: the **first** diagnostic matching "`line ` … ` already defined at line `"
is kept, later ones are dropped, and every other diagnostic is kept in its
original order. A `yaml.TypeError` with no duplicate-key line is returned
unchanged. The kept line for the block form is yaml.v3's own text, e.g.
`line 2: mapping key "a" already defined at line 1`.

The filter inspects each diagnostic once. The first implementation used a
`regexp.MatchString` per diagnostic and that alone cost more than the rest of the
decode (measured, `validate dup-3200.yaml`: regexp filter 3.161 s versus 0.767 s
before the change); the two-marker `strings` predicate in the final code costs
0.689 s, i.e. loading the same document is not slower than at `c70d55b`. Both
numbers are medians of three, same machine, `/usr/bin/time`-style subprocess
timing.

### 1.3 Which key is named (the determinism rule)

**The first key that is reached for the second time by a walk of the document in
document order.**

- JSON (`duplicateKeyError`): the order in which `json.Decoder.Token()` yields
  tokens, which `scanJSONValue` walks depth-first. The key is held in a
  `map[string]struct{}` only for membership; the map is never used to select a
  key, so Go's randomized map iteration cannot influence the result.
- YAML (the kept yaml.v3 line): the first line of yaml.v3's document-order pair
  scan, i.e. the pair (first occurrence, second occurrence) of the first key
  repeated in that mapping.

Both orders are already total, so no sort is needed; this is the same property
FJ-064's `sortedKeys`/`orderYAMLKeys` established. `sortedKeys` and
`orderYAMLKeys` themselves are untouched.

Observable: `{"schemaVersion":1,"a":{"x":1,"x":2},"y":9,"b":{"z":3,"z":4}}` names
`x`; the same document with the two groups swapped names `z`
(`TestLoadNamesFirstRepeatedKeyInDocumentOrder`).

### 1.4 The bounds are implementation choices

`maxDuplicateKeyDiagnosticBytes = 512` and
`maxDuplicateKeyDiagnosticAllocationFactor = 256` are declared in the new test
file and are **implementation choices, not specification limits**. The
specification bounds neither a control-plane response nor a diagnostic message
(§19.4 bounds request bodies; §41.1 fixes the failure *shape*). 512 bytes is the
§41.1 envelope, the fixed `fake_jev_bad_control_request` code, and one diagnostic
naming a key; it constrains the repeated **key name** (a key of up to roughly
350 bytes stays inside it) and never the repetition count. No configuration key,
no limit setting, and no configuration-file size limit were added.

## 2. Files

| file | change |
|---|---|
| `internal/config/load.go` | `duplicateKeyError`; `toJSON` reports it directly; `boundDuplicateKeyDiagnostics` + `yamlDuplicateKeyDiagnostic` |
| `internal/config/duplicate_key_diagnostic_test.go` | new file, 7 tests + 2 helpers + 2 bound constants |
| `internal/control/handlers_test.go` | appended `TestControlRepeatedJSONKeyResponseStaysBounded`, 30 lines; no existing test touched |

`git diff --stat` for tracked files: `internal/config/load.go | 61 ++++`,
`internal/control/handlers_test.go | 30 +`, plus the pre-existing
`.agent/work/FJ-065/state.json` edit made by the PM/specifier stage (not mine;
`state.json` and the reports were not edited by this stage).

## 3. Pre-fix failure evidence per new test

Throwaway copy: a full copy of the tree at `HEAD` with
`git checkout -- internal/config/load.go` (pre-fix source; `grep -c
duplicateKeyError internal/config/load.go` → `0`) and only the two new test files
copied in: `/tmp/fj065/throwaway/repo`. Commands:
`go test ./internal/config -count=1 -run '…the seven new tests…' -v` and
`go test ./internal/control -count=1 -run TestControlRepeatedJSONKeyResponseStaysBounded -v`.

| new test | verdict on pre-fix code | pre-fix observation |
|---|---|---|
| `TestLoadBoundsRepeatedJSONKeyDiagnostic` | FAIL | `diagnostic for 2 repeated keys = "decode YAML configuration: yaml: unmarshal errors:\n  line 1: mapping key \"id\" already defined at line 1", want "decode JSON configuration: duplicate object key \"id\""` |
| `TestLoadBoundsRepeatedYAMLKeyDiagnostic` (block + flow) | FAIL both subtests | `block mapping: repeated keys=2 …` then `diagnostic for 3 repeated keys contains 3 duplicate-key lines, want 1`; flow likewise |
| `TestLoadRepeatedKeyDiagnosticAllocationStaysBounded` | FAIL | step 1 allocates 8 877 968 bytes for a 1 721-byte document, above the 256× bound of 440 576 |
| `TestLoadStillRejectsRepeatedKeys` | FAIL (4 of 6 subtests; both 64-repeat forms) | `diagnostic is 104 882 bytes, above the 512-byte bound` (block YAML 64 keys: `C(64,2)=2 016` lines) |
| `TestLoadDuplicateKeyDiagnosticIsDeterministic` | FAIL (1 of 3 subtests: `json sixteen repeated ids`) | the pre-fix message is deterministic but 6 864 bytes, above the 512-byte bound |
| `TestLoadNamesFirstRepeatedKeyInDocumentOrder` | FAIL all 3 subtests | the pre-fix names are yaml.v3 wording, not `duplicate object key "x"` |
| `TestControlRepeatedJSONKeyResponseStaysBounded` (`internal/control`) | FAIL | `repeated keys=2: response = 400 {"error":"fake_jev_bad_control_request","message":"decode YAML configuration: yaml: unmarshal errors:\n  line 1: …"}, want 400 {…"decode JSON configuration: duplicate object key \"id\""}` |
| `TestLoadAcceptsDocumentsWithoutRepeatedKeys` | PASS | see below |

**The one exception, stated plainly.** `TestLoadAcceptsDocumentsWithoutRepeatedKeys`
passes on the pre-fix code by construction: it is the mandated no-regression test
(accepted documents still accepted, unrelated rejections still rejected), and a
no-regression assertion is by definition one that holds before and after the
change. Every other new test fails on the pre-fix code.

`TestLoadDiagnosticsAreDeterministic` in `internal/config/determinism_test.go`
was deliberately **not** extended. Its protocol compares messages only, so the
specifier's C5 inputs would pass on the pre-fix code (a quadratic message is
still deterministic) and would add new cases that pass unbounded. Those inputs
are covered by `TestLoadDuplicateKeyDiagnosticIsDeterministic`, which runs the
same 1000-call protocol **and** the bound, so it fails pre-fix. No committed test
pins the previous JSON wording (`grep -rn 'mapping key' --include=*_test.go` in
the pre-fix tree matches only the new files).

## 4. Measurements

Driver: `/tmp/fj065/measure.py` — a real server per binary
(`go build -o … ./cmd/fake-jev`, `serve --port 0 --ready-file`), `http.client`
POST to `/__fake/v1/stubs`, `validate`/`serve --config` as subprocesses with
stderr captured to a pipe. Binaries: `/tmp/fj065/pre-fix` (built at `c70d55b`)
and `/tmp/fj065/post-fix` (final code). Medians over five runs where stated,
otherwise single-run probes.

Bodies: control route and JSON file `{"id":0,…,"id":N-1}` (the file form wrapped
as `{"schemaVersion":1,"stubs":[…]}`, as `validationDocument` wraps a control
stub); block YAML file `a: 0\n…\na: N-1`.

### 4.1 Control route `POST /__fake/v1/stubs`

| N | request bytes | pre-fix response bytes | pre-fix ratio | pre-fix median | post-fix response bytes | post-fix median |
|---|---|---|---|---|---|---|
| 200 | 1 691 | 1 114 503 | 659× | (probe) | 107 | <1 ms |
| 400 | 3 491 | 4 468 903 | 1 280× | (probe) | 107 | <1 ms |
| 800 | 7 091 | 17 897 703 | 2 524× | (probe) | 107 | <1 ms |
| 1 600 | 14 891 | 71 635 303 | 4 811× | 0.197 s | 107 | 0.001 s |
| 3 200 | 30 891 | 286 630 503 | 9 279× | 0.744 s | 107 | 0.001 s |
| 6 400 | 62 891 | (not run; PM measured 4 551× class) | — | — | 107 | 0.001 s |

Pre-fix response bytes step 4.00× per doubling of N; status is 400 in every row
in both builds. Post-fix the response is **107 bytes at every N**, byte-identical
across N:
`{"error":"fake_jev_bad_control_request","message":"decode JSON configuration: duplicate object key \"id\""}`.
The 16 KiB acceptance row is N=1 600 (14 891 request bytes): 71 635 303 → 107 bytes.
Post-fix step ratios `size(1600)/size(800)` … are 1.00, and the 1 600→3 200 latency
step is ≤ 2× (0.0007 s → 0.0011 s, both dominated by HTTP round-trip).

### 4.2 `validate <json>` and `serve --config <json>` (stderr bytes, exit code)

| N | input bytes | pre-fix stderr | pre-fix median | post-fix stderr | post-fix median | exit code (both) |
|---|---|---|---|---|---|---|
| 200 | 1 721 | 1 054 839 | (probe) | 141 | 0.004 s | 2 |
| 400 | 3 521 | 4 229 539 | (probe) | 141 | 0.003 s | 2 |
| 800 | 7 121 | 16 938 939 | (probe) | 141 | 0.003 s | 2 |
| 1 600 | 14 921 | 67 797 740 | 0.201 s | 142 | 0.003 s | 2 |
| 3 200 | 30 921 | 271 275 340 | 0.739 s | 142 | 0.003 s | 2 |
| 6 400 | 62 921 | (not run) | — | 142 | 0.003 s | 2 |

`serve --config` rows are the same numbers pre-fix (67 797 740 / 271 275 340 at
1 600 / 3 200, medians 0.215 s / 0.802 s) and the same 141/142 bytes post-fix at
0.003–0.004 s; exit code 2, stdout empty in both builds.

The post-fix 141-versus-142 difference is the digit count of N inside the printed
path (the fixture name carries N), not the diagnostic: with the path normalized
away the two lines are byte-identical, and the message itself is 60/61 bytes plus
the path. Message:
`fake-jev: <path>: decode JSON configuration: duplicate object key "id"`.

### 4.3 `validate <block yaml>` — diagnostic bounded, decode cost still quadratic

| N | input bytes | pre-fix stderr | pre-fix median | post-fix stderr | post-fix median | step ratio (post) |
|---|---|---|---|---|---|---|
| 200 | 1 290 | 1 093 047 | (probe) | 191 | 0.006 s | — |
| 400 | 2 690 | 4 425 847 | (probe) | 191 | 0.014 s | 2.3× |
| 800 | 5 490 | 17 811 447 | (probe) | 191 | 0.047 s | 3.4× |
| 1 600 | 11 690 | 72 423 594 | 0.217 s | 192 | 0.171 s | 3.6× |
| 3 200 | 24 490 | 293 325 994 | 0.878 s | 192 | 0.689 s | 4.0× |
| 6 400 | 50 090 | (not run) | — | 192 | 2.768 s | 4.0× |

Diagnostic bytes are constant (191/192 = path digit count) and contain **exactly
one** duplicate-key line at every N, where the pre-fix message has `C(N,2)` of
them (1 279 200 at N=1 600, 5 118 400 at N=3 200). Normalizing the path away, the
stderr line is byte-identical at N = 1 600, 3 200 and 6 400:

```
fake-jev: <path>: decode YAML configuration: yaml: unmarshal errors:
  line 2: mapping key "a" already defined at line 1
```

**Honest scaling shape.** The message is bounded and constant; the *time* is not.
The post-fix time steps are 2.3×, 3.4×, 3.6×, 4.0×, 4.0× for doubling N — the
residual quadratic lives inside yaml.v3 (§6). Post-fix medians are slightly
*below* the pre-fix medians at the same N (0.171 vs 0.217 s; 0.689 vs 0.878 s)
because the run no longer writes a 293 MB message, which is larger than the cost
of the single-pass filter.

### 4.4 Allocation measured in-process (`go test -v` logs of the new tests)

| repeated keys | JSON document bytes | allocated bytes (`Load`) | block-YAML document bytes | allocated bytes |
|---|---|---|---|---|
| 2 | — | — | 10 | 11 208 |
| 16 | — | — | 86 | 68 280 |
| 200 | 1 721 | 3 688 | — | — |
| 256 | — | — | 1 682 | 5 417 792 |
| 400 | 3 521 | 3 688 | — | — |
| 800 | 7 121 | 3 688 | 5 490 | 57 891 512 |
| 1 600 | 14 921 | 3 408 | — | — |
| 3 200 | 30 921 | 3 688 | — | — |

The JSON path allocates a constant ≈3.5 KB from N=200 to N=3 200 (the scan stops
at the second `id`). The YAML path still allocates ≈10 500× the document at
N=800 — the `TypeError` list yaml.v3 builds internally, which the test logs but
does not assert on because it is not ours to bound here.

## 5. Message usefulness and determinism evidence

- The control 400 body names the key and is not the generic message: `message` =
  `decode JSON configuration: duplicate object key "id"`, not `Invalid control
  request.` (`TestControlRepeatedJSONKeyResponseStaysBounded` compares the whole
  body against that exact string at N = 2, 16 and 1 600, and asserts
  `Content-Type: application/json`, status 400, and no engine mutation).
- The file paths name the key too: `duplicate object key "id"` for JSON,
  `mapping key "a" already defined at line 1` for block YAML.
- Determinism: `TestLoadDuplicateKeyDiagnosticIsDeterministic` runs the
  1000-call protocol of `TestLoadDiagnosticsAreDeterministic` over
  `{"schemaVersion":1,"a":{"x":1,"x":2},"b":{"y":1,"y":2}}`,
  `{"schemaVersion":1,"id":1,"id":2,"id":3}`, a 16-repeat `id` document and the
  block-YAML form `a: 1\na: 2\nb: 1\nb: 2\n`, requiring an identical message on
  every call **and** the 512-byte bound. `TestLoadBoundsRepeatedJSONKeyDiagnostic`
  independently requires the byte-identical message at N = 2 … 3 200, and the YAML
  test at N = 2 … 800 for both block and flow mappings.
- The control-route body is identical at N = 200 … 6 400 in the driver run
  (107 bytes each), and the CLI stderr at N = 1 600 … 6 400 is identical after
  normalizing the path.

## 6. Residual risks

1. **yaml.v3 is still quadratic for YAML input; we did not touch it.** For block
   YAML the reported diagnostic is bounded, but the decoder still does its own
   quadratic work. Measured post-fix: `validate dup.yaml` 0.006 / 0.014 / 0.047 /
   0.171 / 0.689 / 2.768 s at N = 200 … 6 400 (steps 2.3×, 3.4×, 3.6×, 4.0×,
   4.0×) and 11 208 / 68 280 / 5 417 792 / 57 891 512 bytes allocated at
   N = 2 / 16 / 256 / 800. The cost is yaml.v3 formatting one diagnostic per
   repeated-key pair (a `fmt.Sprintf` per pair, `decode.go` `mapping`). A large
   YAML file that repeats one key can therefore still burn quadratic CPU inside
   the dependency; the *response* and the *process output* are bounded, the
   *decode cost* is not. This is the item the PM ruled out of scope for FJ-065
   (no third-party surgery, no document-size cap); it is recorded here for the
   PM rather than silently claimed as linear.
   Secondary observation: yaml.v3's uniqueness scan compares every key pair with
   no early exit, so the comparison loop is present even without duplicates, but
   its constant is small (block YAML with 1 600 / 3 200 / 6 400 *distinct* keys:
   0.008 / 0.015 / 0.053 s).
2. **YAML flow documents with repeated unquoted keys take the same residual
   path.** `{a: 1, a: 2, …}` is not JSON (unquoted key), so the JSON scan fails on
   syntax and the document reaches yaml.v3; its message is bounded by §1.2 and
   covered by the flow-mapping subtest, its decode cost is the same residual as
   above.
3. **The JSON wording for a repeated key changed.** For JSON input the message is
   now `decode JSON configuration: duplicate object key "id"` instead of a
   yaml.v3 list. The status, the `error` code, the envelope fields, the exit code
   and the acceptance decision are unchanged; §41.1 leaves `message` free-form
   and no committed test pinned the old text.
4. **Only the first repeated key is named** when a document repeats several
   *different* keys, on both paths. That information reduction is what makes the
   message bounded; recorded here explicitly.
5. **The 512-byte bound assumes a key name shorter than roughly 350 bytes.** A
   repeated key of 400 bytes produces a 105 + len(key) control body (505 bytes,
   still inside 512); longer keys would exceed the test bound, which is an
   implementation choice rather than a documented limit, so no behaviour is
   incorrect — the property asserted is "a function of the key name and the error
   code, not of N".
6. **`validate` still writes its diagnostic to a caller-supplied stderr and
   reads the whole configuration file into memory** (the FJ-030 item). Neither is
   changed here.
7. **FJ-040's test-side bound** (`configFuzzMaxMappingKeys = 256`) does not exist
   on this branch (`internal/config/fuzz_test.go` is absent), so nothing had to
   be reconciled with it here; the guard fuzz row runs one target and reported no
   findings over its 30 s window.

## 7. Commands run and what they reported

| command | observation |
|---|---|
| `gofmt -w internal/config/load.go internal/config/duplicate_key_diagnostic_test.go internal/control/handlers_test.go`; `gofmt -l internal/config internal/control` | no output (formatted) |
| `go test ./internal/config -count=1` | `ok fake-jev/internal/config` |
| `go test ./... -count=1` | `ok` for `cmd/guard`, `internal/cli`, `internal/compat/jev/v1`, `internal/config`, `internal/control`, `internal/engine`, `internal/host/http`, `test/integration`; `no test files` for `cmd/fake-jev`, `examples/go` |
| `go test -race ./... -count=1` | same packages `ok` |
| `go vet ./...` | exit 0, no output |
| `go build ./cmd/fake-jev` | exit 0 |
| `go run ./cmd/guard arch` | `findings: []`, 63 files / 10 packages |
| `go run ./cmd/guard lint` | `findings: []` (the first draft of the test helpers discarded `strings.Builder`/`fmt.Fprintf` error results, and `guard lint` reported `guard.lint.err_discarded` findings against `internal/config/duplicate_key_diagnostic_test.go` and `internal/control/handlers_test.go`; the helpers were rewritten to build string slices and `strings.Join` instead) |
| `go run ./cmd/guard trace` | `findings: []`, covered 17, test_files 25 |
| `go run ./cmd/guard fuzz` | `findings: []`, `targets: 1`, 31.5 s wall |
| `./scripts/verify-candidate FJ-065` | exit 0; report `.agent/reports/FJ-065/report-20261003T180747Z.json`; `summary: 20 passed, 0 failed, 0 skipped, 0 not_applicable`, `RESULT: PASS`; its 20 rows are the repository/setup rows, `check_go_test`, `check_go_vet`, `check_go_build`, `check_guard_arch`, `check_guard_lint` and `check_guard_trace`, all `result=pass`, `required=True`, `exempted=False`; the report's candidate fingerprint is `f6419cff…`, the same value as the front matter above. No row was skipped or not-applicable, so no row needed an explanation of why it does not apply. The fuzz row is not part of verify-candidate on this branch and was run separately (previous row). |
| `./scripts/candidate-fingerprint candidate` | `f6419cffc219fc46451b98d04986daedfa732d1f754fbe3c4dbbc116591cf587` |

`git status --porcelain` at the end: ` M .agent/work/FJ-065/state.json` (from the
PM/specifier stage, not this one), ` M internal/config/load.go`,
` M internal/control/handlers_test.go`, `?? internal/config/duplicate_key_diagnostic_test.go`,
`?? .agent/work/FJ-065/{specifier,coder}.md`, `?? .agent/reports/FJ-065/`.
Nothing is staged.
