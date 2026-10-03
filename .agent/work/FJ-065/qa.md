---
stage: qa
task: FJ-065
inputFingerprint: 645887fd92010e90db08b55ca75104a56a43c67d918038b463184a968a75d5f4
outputFingerprint: 645887fd92010e90db08b55ca75104a56a43c67d918038b463184a968a75d5f4
taskFingerprint: ee28135d80573ff713fda4993a1571871f25f3ac6424a2b7e361c32336ad06ea
gitHead: c70d55b
generatedAt: 2026-10-03T18:47:56Z
author: worker/FJ-065-qa
---

# FJ-065 QA — the bound under a real server, the real CLI, and adversarial keys

Independent QA of candidate `645887fd…` (the hardener's output). Every number
below was produced by this stage from the candidate working tree; the
hardener's residual section was used as the agenda, not as evidence.

Scratch area: `/private/tmp/fj065-qa` (binary `/private/tmp/fj065-qa/fake-jev`,
built once from the candidate tree; pre-fix binary `/private/tmp/fj065-qa/fake-jev-prefix`,
built once from `git archive c70d55b` in `/private/tmp/fj065-qa/prefix` for
verdict comparison). Drivers: `control.py`, `filepaths.py`, `compare.py`,
`compare2.py`, transcripts `control.out`, `filepaths.out`, `compare.out`,
`tests.out`, `race.out`, `guard.out`, `verify.out`, `redA.out`. No repository
file was modified by this stage; the only write is this artifact. Nothing is
staged (`git diff --cached --name-only` empty).

## 1. Criterion 1 — is the bound real end to end, on all reachable paths?

All rows are against the real binary. "median" is the median of five runs.
Response/diagnostic byte counts include the fixture path (whose digit count
changes by 2 across the table), so the constant is the message plus path.

### 1.1 Control route — real server, `POST /__fake/v1/stubs`

Body `{"id":0,…,"id":N-1}`; one server from the candidate binary; status 400 in
every row. Command: `python3 control.py` (see `control.out`).

| N | request B | status | response B | median s |
|---|---|---|---|---|
| 50 | 391 | 400 | **107** | 0.000210 |
| 100 | 791 | 400 | **107** | 0.000213 |
| 200 | 1 691 | 400 | **107** | 0.000223 |
| 400 | 3 491 | 400 | **107** | 0.000191 |
| 800 | 7 091 | 400 | **107** | 0.000252 |
| 1 600 | 14 891 | 400 | **107** | 0.000286 |
| 3 200 | 30 891 | 400 | **107** | 0.000350 |
| 6 400 | 62 891 | 400 | **107** | 0.000578 |
| 12 800 | 129 691 | 400 | **107** | 0.001012 |

Eight doublings (50→12 800, a 332× request growth). Every response is
byte-identical (`distinct response bodies across all N: 1`);
`size(N+1)/size(N) = 1.0000` at every step, so **size is constant in N**.
`Content-Type: application/json`. Body:

```
{"error":"fake_jev_bad_control_request","message":"decode JSON configuration: duplicate object key \"id\""}
```

**Time shape.** 0.00021 s → 0.00101 s over the 332× request growth, i.e. the
cost is the linear cost of receiving and JSON-scanning the body (the scan stops
at the second `id`), with no visible quadratic term: the last doubling costs
0.00058 → 0.00101 s (1.75×, and the absolute values are at the noise floor).

### 1.2 `validate` on a JSON config file

Body `{"schemaVersion":1,"stubs":[{"id":0,…,"id":N-1}]}`. Command:
`python3 filepaths.py` (see `filepaths.out`).

| N | input B | exit | stdout B | stderr B | median s |
|---|---|---|---|---|---|
| 100 | 821 | 2 | 0 | 102 | 0.0033 |
| 200 | 1 721 | 2 | 0 | 102 | 0.0032 |
| 400 | 3 521 | 2 | 0 | 102 | 0.0032 |
| 800 | 7 121 | 2 | 0 | 102 | 0.0031 |
| 1 600 | 14 921 | 2 | 0 | 103 | 0.0031 |
| 3 200 | 30 921 | 2 | 0 | 103 | 0.0031 |
| 6 400 | 62 921 | 2 | 0 | 103 | 0.0032 |
| 12 800 | 129 721 | 2 | 0 | 104 | 0.0031 |

The 102→104 change is the two extra path digits (`dup-100.json` →
`dup-12800.json`); the message part is byte-identical. **Size is constant in
N**; time is flat at ≈3.1 ms from N=100 to N=12 800 (process start dominates).

### 1.3 `validate` on a block-YAML config file

Body `a: 0\na: 1\n…\na: N-1`. `dup_lines` counts `already defined` occurrences.

| N | input B | exit | stdout B | stderr B | dup lines | median s | step |
|---|---|---|---|---|---|---|---|
| 100 | 590 | 2 | 0 | 152 | **1** | 0.0038 | — |
| 200 | 1 290 | 2 | 0 | 152 | **1** | 0.0057 | 1.50× |
| 400 | 2 690 | 2 | 0 | 152 | **1** | 0.0141 | 2.47× |
| 800 | 5 490 | 2 | 0 | 152 | **1** | 0.0483 | 3.43× |
| 1 600 | 11 690 | 2 | 0 | 153 | **1** | 0.1803 | 3.73× |
| 3 200 | 24 490 | 2 | 0 | 153 | **1** | 0.6950 | 3.86× |
| 6 400 | 50 090 | 2 | 0 | 153 | **1** | 2.8815 | 4.15× |
| 12 800 | 104 090 | 2 | 0 | 154 | **1** | **16.9398** | **5.88×** |

**Size is constant in N with exactly one duplicate-key line at every N** (the
pre-fix mechanism emits `C(N,2)` of them). **Time is not constant and not
linear**: the doubling steps are 1.5 → 2.5 → 3.4 → 3.7 → 3.9 → 4.2 → 5.9, so
from N=400 the cost multiplier per doubling is ≥3.4 and rises above the
quadratic 4× at the last step. This is the residual the PM ruled out of scope
(see §8); it is measured, not inherited from any prior artifact.

Peak resident set size for the same command (`/usr/bin/time -l`, single run):

| N | input B | max RSS candidate | max RSS pre-fix (`c70d55b`) |
|---|---|---|---|
| 1 600 | 11 690 | 169 492 480 | 604 258 304 |
| 3 200 | 24 490 | 603 734 016 | 2 368 733 184 |
| 6 400 | 50 090 | 2 595 176 448 | (not measured) |
| 12 800 | 104 090 | **7 355 351 040** | (not measured) |

The fix reduces both time and peak memory at a given N (the huge message is no
longer materialised), but a 104 KB YAML file still allocates ≈7.4 GB and takes
≈17 s.

Flow YAML `{a: i, …}` (the third parse syntax on this route, not JSON and not
block YAML):

| N | input B | exit | stderr B | dup lines | median s |
|---|---|---|---|---|---|
| 2 | 13 | 2 | 151 | 1 | 0.0034 |
| 32 | 215 | 2 | 152 | 1 | 0.0034 |
| 256 | 1 939 | 2 | 153 | 1 | 0.0078 |
| 1 024 | 8 107 | 2 | 154 | 1 | 0.0692 |

### 1.4 `serve --config` on a bad file

| file | N | input B | exit | stderr B | median s |
|---|---|---|---|---|---|
| JSON | 100 | 821 | 2 | 102 | 0.0035 |
| JSON | 1 600 | 14 921 | 2 | 103 | 0.0035 |
| JSON | 3 200 | 30 921 | 2 | 103 | 0.0033 |
| JSON | 6 400 | 62 921 | 2 | 103 | 0.0036 |
| block YAML | 1 600 | 11 690 | 2 | 160 | (single run) |
| block YAML | 3 200 | 24 490 | 2 | 160 | (single run) |
| block YAML | 6 400 | 50 090 | 2 | 160 | (single run) |

Same messages and same exit code 2 as `validate`; the YAML stderr bytes are 160
because the path is shorter than the one in the `validate` table.

**Plain statement.** Response/diagnostic size is constant in the repetition
count on all four reachable paths (control route, `validate` JSON,
`validate` YAML block/flow, `serve --config`), modulo the fixture path's digit
count. The time shape is linear in the request for the control route, flat for
the JSON file paths, and **quadratic-to-worse for the YAML file paths** — the
bound covers the message, not yaml.v3's decode cost.

## 2. Criterion 2 — attacking the bound

Bodies `{"<key>":1,"<key>":2}` (two occurrences is all it takes), real server,
`control.py`. Keys are given as the raw bytes that sit between the JSON quotes,
so `escaped` rows are the literal backslash sequences a JSON body carries.
Ratio = response B / request B.

| key text | request B | status | response B | ratio |
|---|---|---|---|---|
| `id` | 15 | 400 | 107 | 7.13 (fixed overhead dominates) |
| plain `k` ×400 | 811 | 400 | 505 | 0.62 |
| plain `k` ×407 | 825 | 400 | **512** | 0.62 |
| plain `k` ×408 | 827 | 400 | **513** | 0.62 |
| plain `k` ×4 096 | 8 203 | 400 | 4 201 | 0.51 |
| plain `k` ×100 000 | 200 011 | 400 | 100 105 | 0.50 |
| `\"` escaped ×500 (key = `"`) | 2 011 | 400 | 2 105 | 1.047 |
| `\n` escaped ×500 (key = newline) | 2 011 | 400 | 1 605 | 0.798 |
| `\u0000` escaped ×500 (key = NUL) | 6 011 | 400 | 2 605 | 0.433 |
| `\u007f` escaped ×500 (key = DEL) | 6 011 | 400 | 2 605 | 0.433 |
| U+0080 raw ×1 000 (2-byte non-printable) | 4 011 | 400 | 7 105 | **1.771** |
| U+0080 raw ×500 000 (2-byte non-printable) | 2 000 011 | 400 | **3 500 105** | **1.750** |
| U+00A0 raw ×1 000 (2-byte non-printable) | 4 011 | 400 | 7 105 | 1.771 |
| U+200B raw ×1 000 (3-byte non-printable) | 6 011 | 400 | 7 105 | 1.182 |
| U+1D173 raw ×1 000 (4-byte non-printable) | 8 011 | 400 | 11 105 | 1.386 |
| U+FFFD raw ×1 000 (3-byte printable) | 6 011 | 400 | 3 105 | 0.517 |
| U+00E9 raw ×1 000 (2-byte printable) | 4 011 | 400 | 2 105 | 0.525 |
| U+1F600 raw ×1 000 (4-byte printable) | 8 011 | 400 | 4 105 | 0.512 |
| U+2028 raw ×1 000 | 6 011 | 400 | 7 105 | 1.182 |
| raw `0xff` ×1 000 (invalid UTF-8) | 2 011 | 400 | **77** | 0.038 (generic `Invalid control request.`; the control decoder rejects the body before the loader) |

**Reproduced.** The hardener's worst case reproduces exactly:
2 000 011 B request → **3 500 105 B** response (1.750×) with a 500 000-character
U+0080 key. I found no key worse than that; the ranking by ratio is
non-printable 2-byte rune (1.77 at small scale, 1.75 at the 2 MiB scale) >
non-printable 4-byte rune (1.39) > non-printable 3-byte rune (1.18) = U+2028 >
escaped printable `"` (1.05) > everything else < 1. The arithmetic is the
`%q`/\\-escaping envelope: a non-printable rune costs 7 response bytes per
character (`\uXXXX` = 6, doubled backslash = 7) while the request must spell it
as ≥2 raw bytes twice (4), giving the 7/4 = 1.75 ceiling; no rune has a worse
bytes-per-character ratio (1-byte non-printables must be escaped to `\u00XX` in
JSON, 5 response B per 6 request B).

**Does this count as amplification?** The response is bounded by
`105 + 1.75 × request` (≈3.50 MB for the largest legal control body, since
§19.4 caps control bodies at 2 MiB) — bounded, but not by an absolute constant.
It is a constant factor on key *content*: `{"<key>":1,"<key>":2}` amplifies with
**no repetition at all**, and the ratio does not move when the key is repeated
more times. The defect in this item — response growing with the repetition
count — is gone (`size(N)=107` for every N); this residual is a linear echo of
the caller's key text and is the item the hardener recorded rather than bounded.

**Repeated/distinct matrix** (all status 400):

| body | request B | response B | named key |
|---|---|---|---|
| `id` ×1 000, distinct values | 8 891 | 107 | `id` |
| `id` ×1 000, identical value | 7 001 | 107 | `id` |
| 1 000 **distinct** keys each repeated twice | 17 781 | 107 | `k0` |
| 3 distinct keys each repeated 500× | 11 671 | 106 | `a` |

**Response constant in N for a fixed adversarial key** (key = U+0080 ×10, i.e.
6 message characters): repeats 2/4/16/64/256/1 024 → request 51/101/407/1 655/
6 803/27 563 B → response **175 B** in every row. The residual is a function of
key content, not of the repeat count, which is the property under test.

## 3. Criterion 3 — the duplicate key is still reported

Every path rejects; none silently accepts.

| path | observation |
|---|---|
| control route | 400, body exactly `{"error":"fake_jev_bad_control_request","message":"decode JSON configuration: duplicate object key \"id\""}` (107 B) at N=50…12 800; `Content-Type: application/json`; engine unchanged (`GET /__fake/v1/stubs` = `{"stubs":[]}` immediately after the rejected POST) |
| `validate <json>` | exit 2, stdout empty, stderr one line naming `"id"`, at N=100…12 800 |
| `validate <block yaml>` | exit 2, stdout empty, one `mapping key "a" already defined at line 1` line, at N=100…12 800 |
| `validate <flow yaml>` | exit 2, stdout empty, one line, at N=2…1 024 |
| `serve --config <json>` | exit 2, same one line |
| `serve --config <yaml>` | exit 2, same one line |
| 2 MiB+1 control body | 413 `fake_jev_payload_too_large` (93 B) — the separate path is intact |

No committed test was needed to observe this: it was verified end to end from
the binary.

## 4. Criterion 4 — determinism

200 identical calls per syntax. Control route (`control.py`):

| input | distinct (status, body) pairs | counts |
|---|---|---|
| JSON `{"id":0,…,"id":15}` | 1 | 200 |
| JSON `{"schemaVersion":1,"a":{"x":1,"x":2},"y":9,"b":{"z":3,"z":4}}` | 1 | 200 |
| JSON same document, groups swapped | 1 | 200 |
| block YAML `a: 1\na: 2\nb: 1\nb: 2\n` | 1 | 200 |
| flow YAML `{a: 1, a: 2, a: 3}` | 1 | 200 |

**Naming is deterministic per route, and the two routes name different keys for
a multi-duplicate document.** 200 `validate` process runs each on
`{"a":1,"b":1,"b":2,"a":2}` (and its block/flow forms):

| document | route | distinct stderr | 200-run count | named |
|---|---|---|---|---|
| JSON `{"a":1,"b":1,"b":2,"a":2}` | JSON scan | 1 | 200 | `duplicate object key "b"` |
| block `a: 1\nb: 1\nb: 2\na: 2` | yaml.v3 | 1 | 200 | `line 4: mapping key "a" already defined at line 1` |
| flow `{a: 1, b: 1, b: 2, a: 2}` | yaml.v3 | 1 | 200 | `line 1: mapping key "a" already defined at line 1` |

So the JSON route names the key whose **second** occurrence comes first in token
order (`b`), while the YAML route names the pair with the earliest **first**
occurrence (`a`). Both are single-valued per route over 200 runs, and the
divergence is stable and a function of the document alone — it is recorded, not
a nondeterminism. This matches the hardener's corrected rule and contradicts
`coder.md` §1.3's single-rule wording for the YAML route.

## 5. Criterion 5 — falsifying the wording guard

Throwaway copy of the candidate tree at `/private/tmp/fj065-qa/redA` with exactly
one change — `internal/config/load.go:149`, the predicate's second marker:
`" already defined at line "` → `" already declared at line "`:
`grep -n 'already declared at line' internal/config/load.go` →
`149:	return strings.HasPrefix(diagnostic, "line ") && strings.Contains(diagnostic, " already declared at line ")`.

`go test ./internal/config -count=1` → **exit 1, 10 `--- FAIL` lines**
(transcript `redA.out`):

- `TestLoadFileBoundsRepeatedYAMLKeyDiagnostic` — fails on **size**:
  `repeated keys=64: diagnostic is 108347 bytes, above the 512-byte bound; the duplicate-key diagnostic is no longer bounded: "decode YAML configuration: yaml: unmarshal errors:\n line 2: mapping key \"a\" already defined at line 1\n line 3: …"`
- `TestLoadStillRejectsRepeatedKeys/{block,flow}_yaml_mapping` — fails on
  **size**: `diagnostic is 108347 bytes, above the 512-byte bound`.
- `TestLoadBoundsRepeatedYAMLKeyDiagnostic/{block,flow}_mapping` — fails on
  **count**: `diagnostic for 3 repeated keys contains 3 duplicate-key lines,
  want 1`.
- `TestYAMLDuplicateKeyDiagnosticMarkerPinsDecoderWording/{block,flow}_mapping`
  — the wording pin itself: `yamlDuplicateKeyDiagnostic("line 2: mapping key \"a\"
  already defined at line 1") = false, want true`.

So the guard goes red, and the dominant failure text is the byte bound
(`108347 bytes, above the 512-byte bound`, `104882 bytes, above the 512-byte
bound`), not an incidental string-comparison failure. The predicate rewording is
the same falsification the hardener performed; I did not additionally run the
dependency-`replace` variant (the predicate is the released code path and the
stronger claim — that a *reworded dependency* reddens the tests — was already
demonstrated by the hardener and is consistent with the 10 failures above).

## 6. Criterion 6 — no acceptance flips

Verdict comparison between the pre-fix binary (`c70d55b`, `fake-jev-prefix`) and
the candidate (`compare.py`, `compare2.py`): exit status and stdout must match;
only wording may differ. **`total flips in boundary cases: 0`.**

| document | pre | candidate | verdict |
|---|---|---|---|
| `{a: 1, a: 2, a: 3}` (flow YAML, repeated key) | exit 2 | exit 2 | same (message list reduced to one line) |
| `{07,2}` | exit 2 `object key 2 is not a string` | exit 2, same text | same |
| `{"a":1, a:2}` (invalid JSON **and** repeated key) | exit 2 `json: unknown field "a"` | exit 2, same text | same (the JSON scan fails on syntax, so yaml.v3 handles it exactly as before) |
| `{"a":1, "a":2,}` (trailing comma + repeat) | exit 2 yaml.v3 duplicate | exit 2 JSON duplicate | same decision |
| `{a:1,a:2,}` | exit 2 `json: unknown field "a:1"` | exit 2, same text | same |
| `{"a":1,"a":2}` | exit 2 | exit 2 | same decision |
| `{"schemaVersion":1,"id":1,"id":2,"id":3}` | exit 2 | exit 2 | same decision |
| `{schemaVersion: 1}` (YAML flow) | exit 0 | exit 0 | accepted both |
| `schemaVersion: 1` (block) / `{"schemaVersion":1}` | exit 0 | exit 0 | accepted both |
| `null` / `[]` / `{}` | exit 2 | exit 2, same text | same |
| two YAML docs / two JSON docs / malformed JSON / empty / scalar | exit 2 | exit 2, same text | same |
| exact repeat inside open `then.raw.body` (`{"a":1,"a":2}`), valid stub otherwise | exit 2 | exit 2 | same decision (document-wide rejection preserved) |
| accepted baseline: same stub **without** the repeat | exit 0 | exit 0 | accepted both |
| case-variant `then.raw.headers` names / case-variant question names | exit 0 | exit 0 | accepted both (FJ-064 behaviour intact) |
| duplicate `a`+`b` groups `{"schemaVersion":1,"a":{"x":1,"x":2},"y":9,"b":{"z":3,"z":4}}` | exit 2 (names `x`) | exit 2 (names `x`) | same decision |

Whole-corpus sweep (`compare.py`): all committed `testdata/**/*.json|yaml|yml`,
`examples/**/*.yaml|json`, plus `spec/config.schema.json` — **25 documents
compared, 0 decision mismatches**, accepted count 1 before and 1 after.

## 7. Criterion 7 — scope

`git diff --stat c70d55b -- .`:

```
 .agent/work/FJ-065/state.json     | 19 +++++++-----
 internal/config/load.go           | 65 +++++++++++++++++++++++++++++++++++++--
 internal/control/handlers_test.go | 30 ++++++++++++++++++
 3 files changed, 103 insertions(+), 11 deletions(-)
```

`git status --short`:

```
 M .agent/work/FJ-065/state.json
 M internal/config/load.go
 M internal/control/handlers_test.go
?? .agent/reports/FJ-065/
?? .agent/work/FJ-065/cleaner.md
?? .agent/work/FJ-065/coder.md
?? .agent/work/FJ-065/hardener.md
?? .agent/work/FJ-065/specifier.md
?? internal/config/duplicate_key_diagnostic_test.go
```

- Product files changed: `internal/config/load.go` (modified),
  `internal/config/duplicate_key_diagnostic_test.go` (new),
  `internal/control/handlers_test.go` (test appended). Nothing outside
  `internal/config/` and `internal/control/`. `.agent/work/FJ-065/state.json`
  is the earlier PM/specifier-stage edit, not a product change.
- `find . -name fuzz_test.go` → only `./cmd/guard/fuzz_test.go` (the guard
  tool's own test, pre-existing); **no `internal/config/fuzz_test.go`**.
- `git diff c70d55b -- . | grep '^+.*func Fuzz'` → **none**; no `Fuzz*` symbol
  was added (`internal/compat/jev/v1/fixture_test.go::FuzzValidateFixtureAnswers`
  is pre-existing).
- `git diff c70d55b -- internal/config/load.go | grep 'os.ReadFile|func LoadFile|io.ReadAll'`
  → no match: the unbounded configuration **read** (`os.ReadFile` in `LoadFile`)
  is untouched; only the duplicate-key diagnostic and the parse of a
  duplicate-key JSON document are bounded.
- `git diff c70d55b -- . | grep 'sortedKeys|orderYAMLKeys'` → no match: FJ-064's
  exact-key walk and its determinism are not touched. The corpus and
  case-variant rows of §6 independently confirm the FJ-064 decisions.
- Nothing staged: `git diff --cached --name-only` is empty.

## 8. Commands run (each once, uncached where Go caching allows)

| command | result |
|---|---|
| `gofmt -l internal/config/load.go internal/config/duplicate_key_diagnostic_test.go internal/control/handlers_test.go` | no output, exit 0 |
| `go test ./internal/config -count=1` | `ok fake-jev/internal/config 0.731s`, exit 0 |
| `go test ./... -count=1` | exit 0; `ok` for `cmd/guard` 8.970s, `internal/cli` 24.006s, `internal/compat/jev/v1` 1.796s, `internal/config` 0.932s, `internal/control` 2.969s, `internal/engine` 3.932s, `internal/host/http` 3.526s, `test/integration` 2.374s; `no test files` for `cmd/fake-jev`, `examples/go` |
| `go test -race ./... -count=1` (one run, `-count=1`, no `-count>1`) | exit 0; same packages `ok`, `internal/config` 5.503s, no race report |
| `go vet ./...` | exit 0, no output |
| `go run ./cmd/guard arch` | `findings: []`, 63 files / 10 packages, exit 0 |
| `go run ./cmd/guard lint` | `findings: []`, exit 0 |
| `go run ./cmd/guard trace` | `findings: []`, covered 17, test_files 25, exit 0 |
| `go run ./cmd/guard fuzz` | `findings: []`, `targets: 1`, 31.5 s wall (one target, ~30 s), exit 0 |
| `./scripts/verify-candidate FJ-065` | exit 0. Report `.agent/reports/FJ-065/report-20261003T184636Z.json`; `summary: 20 passed, 0 failed, 0 skipped, 0 not_applicable`; `RESULT: PASS`; report `candidateFingerprint 645887fd…`, `taskFingerprint ee28135d…` |
| `./scripts/candidate-fingerprint candidate` | `645887fd92010e90db08b55ca75104a56a43c67d918038b463184a968a75d5f4` (unchanged before and after this stage) |

`verify-candidate` rows, exactly as reported: all twenty are `result=pass`,
`required=true`, `exempted=false` — `check_candidate_fingerprint`,
`check_required_files`, `check_scripts_executable`, `check_shell_syntax`,
`check_toolchain_python3`, `check_spec_present`, `check_json_files`,
`check_work_item_schema`, `check_recorded_report_coverage`, `check_role_packs`,
`policy.exemptions.unknown`, `policy.exemptions.stale`,
`policy.classification`, `policy.required_stages`, `check_go_test`,
`check_go_vet`, `check_go_build`, `check_guard_arch`, `check_guard_lint`,
`check_guard_trace`. There is **no fail, no skip, and no not-applicable row**, so
no row needs a reason for not applying. The fuzz row is not part of
`verify-candidate` on this branch and was run separately (row above). This
artifact does not declare the item closed; closure is `verify-candidate`'s
exit code plus the owner's decision, and `state.json`/the reports were not
edited by this stage.

## 9. Residual risks and evidence limits

1. **yaml.v3's decode cost for YAML input remains quadratic, and is now
   quantified.** Independently measured here: `validate` on a block-YAML file
   takes 0.0038 / 0.0057 / 0.0141 / 0.0483 / 0.1803 / 0.6950 / 2.8815 /
   **16.9398 s** at N = 100 … 12 800 (doubling steps 1.5× → 5.9×) with peak RSS
   169 MB / 604 MB / 2.60 GB / **7.36 GB**, for inputs of 11.7 KB … 104 KB. The
   flow-YAML syntax behaves the same way. The emitted diagnostic is bounded
   (one line, 152–154 B), the decode cost is not. The PM ruled this out of scope
   for FJ-065 (no third-party surgery, no document-size cap); recorded, not
   chased. It is the one part of the work item's "parse cost grows at most
   linearly" statement that the candidate does not satisfy, and the failure is
   on the locally supplied file path, not on the control route.
2. **The worst-case response is ≈1.75× a legal 2 MiB control body** and is not
   capped by an absolute constant: 2 000 011 B request → 3 500 105 B response,
   responding with the caller's key text escaped. Reproduced independently in
   §2; no key found that beats it. It needs no repetition at all and does not
   grow with N, so it is a constant-factor echo of key content rather than the
   quadratic repetition amplification this item removed. §19.4 caps the request,
   which caps the response at ≈3.5 MB, but the specification sets no response
   bound. The tests' 512-byte figure is an implementation choice: it holds for
   plain keys up to 407 bytes (measured 512 at 407, 513 at 408) and is not a
   product limit.
3. **File paths have no request/size limit at all** (the FJ-030 read item), so a
   local file can emit a stderr line proportional to its key text, and (per the
   hardener) a raw-invalid-UTF-8 repeated key is reachable there while the
   control route rejects that body generically.
4. **The YAML wording predicate is textual.** §5 confirms that rewording it
   turns four guard tests red on size, but the coupling to `gopkg.in/yaml.v3`
   v3.0.1's diagnostic text remains. I did not re-run the dependency-`replace`
   simulation (the hardener did); my falsification used the released predicate
   path.
5. **Only the first repeated key is named**, and the two routes name different
   keys for a multi-duplicate document (§4). Each route is deterministic over
   200 runs; the divergence is a documented information reduction, and
   `coder.md` §1.3's single-rule statement is not the actual rule for the YAML
   route.
6. **Evidence limits.** All timing is this machine, wall-clock, single-process;
   the 12 800-repetition YAML row is memory-bound (7.4 GB peak RSS) and its
   5.9× step is partly allocator/GC pressure, so I report the step table rather
   than fitting an exponent. The 2 MiB adversarial rows were run once each.
   The pre-fix comparison covers verdicts (exit status/stdout), not message
   equality, because the message wording is an accepted change.
   `go test ./...` and `verify-candidate` used Go's build cache for some
   packages (the explicit `-count=1` rows were uncached).

## 10. Criteria exercised against the catalogs

C-CTRL-010, C-CTRL-011, C-CLI-001, C-CLI-005, C-HOST-006, C-CFG-002,
C-CFG-009, C-QUAL-001, C-QUAL-004, C-QUAL-007 as listed by the specifier, plus
the criterion set supplied for this stage. Observations are above; no verdict is
recorded here (design §5).
