---
stage: hardener
task: FJ-065
inputFingerprint: f6419cffc219fc46451b98d04986daedfa732d1f754fbe3c4dbbc116591cf587
outputFingerprint: 645887fd92010e90db08b55ca75104a56a43c67d918038b463184a968a75d5f4
taskFingerprint: ee28135d80573ff713fda4993a1571871f25f3ac6424a2b7e361c32336ad06ea
gitHead: c70d55b
generatedAt: 2026-10-03T18:28:51Z
author: worker/FJ-065-hardener
---

# FJ-065 hardener — the bound has to fail loudly when the library rewords itself

`outputFingerprint: 645887fd92010e90db08b55ca75104a56a43c67d918038b463184a968a75d5f4`
(exact output of `./scripts/candidate-fingerprint candidate` after this stage's
changes; the coder's candidate was `f6419cff…`, which this stage deliberately
moves by adding tests).

Stage input was the coder's candidate `f6419cff…`. This stage changes two files:
`internal/config/load.go` (one doc comment) and
`internal/config/duplicate_key_diagnostic_test.go` (three tests, one helper, two
imports). No production behaviour changed; the bound itself is the coder's.

Contract clauses served: §19 resource-bound posture and §19.4 (the control-plane
request body limit that the recorded residual now leans on), §41.1 (control
failure shape and its human-readable `message`), §22.2 / C-QUAL-004 (malformed
untrusted input yields a controlled error), §12.3 (unknown-key rejection, whose
sibling the repeated-key rejection is), spec line 143 (determinism), and §31 /
C-QUAL-001 (the test suite is the conformance carrier).

## 1. Residual 1 — the predicate is textually coupled to yaml.v3's wording: hardened

**What was at risk.** `yamlDuplicateKeyDiagnostic` matches two substrings of
yaml.v3's duplicate-key line. If the dependency is upgraded and that line is
reworded, the predicate returns false for every diagnostic, the filter keeps all
`C(N,2)` of them, and the amplification the item exists to remove comes back. The
coder's tests do fail in that case (measured below), so the honest statement of
the risk is not "no test": it is that the guard was entangled with exact text,
that no test asserted the *library wording itself* or the *product-promise
failure mode* independently of it, and that the coupling was not stated where a
reader of `load.go` would find it.

**Can the match be made robust rather than loud?** No, not through this
dependency, and I checked rather than assumed. `gopkg.in/yaml.v3 v3.0.1` exposes
duplicate-key diagnostics only as text: `yaml.TypeError` is
`struct { Errors []string }` (`yaml.go:315`) and the producer is a
`fmt.Sprintf` at `decode.go:776` inside `decoder.mapping`. There is no error kind,
sentinel, or node reference to match structurally. The only alternatives were
(a) a looser pattern that accepts several plausible wordings — speculative
future-proofing that AGENTS.md forbids and that risks swallowing unrelated
diagnostics, or (b) a bound that does not depend on identifying the diagnostic at
all, i.e. truncating or capping the message, which is the unapproved behaviour
change analysed under residual 2. So the honest hardening is a guard that fails
loudly and a marker whose coupling is stated where it lives.

**Actions (in `internal/config/duplicate_key_diagnostic_test.go` and a comment).**

1. `TestLoadFileBoundsRepeatedYAMLKeyDiagnostic` — the end-to-end guard. It writes
   a real block-YAML file with one key repeated 64 and 512 times and loads it
   through `config.LoadFile`, which is the entry point `internal/cli/validate.go`
   and `internal/cli/serve --config` (`internal/cli/serve.go:132`) call. It
   asserts only product promises, never the library's phrasing: the document is
   rejected; the message is inside `maxDuplicateKeyDiagnosticBytes`; the repeated
   key name appears at most twice (one diagnostic names both occurrences, while an
   unclamped list repeats the line `C(count,2)` times); and the message is
   byte-identical at both repetition counts.
2. `TestYAMLDuplicateKeyDiagnosticMarkerPinsDecoderWording` — the wording pin. It
   asserts the exact diagnostic text of the pinned dependency
   (`line 2: mapping key "a" already defined at line 1` for the block form,
   `line 1: …` for the flow form), asserts that `yamlDuplicateKeyDiagnostic`
   accepts it, and asserts that the predicate rejects four lookalikes (a
   `cannot unmarshal` diagnostic that also starts with `line `, the bare marker
   without the `line ` prefix, a `field … already set` diagnostic, and a
   `must be unique` wording). A reworded dependency fails the first assertion with
   a message that says what happened; a filter bug that over-matches fails the
   last.
3. `internal/config/load.go`: the comment on `yamlDuplicateKeyDiagnostic` now names
   the pinned version and the producing line, states that the match is necessarily
   textual, and names both guard tests. That is the constraint, not a restatement
   of the code — this is the one place a reader hits the coupling.

**Red proof (both simulations in throwaway copies; the repository was not used as
a scratch area).**

*Simulation A — the predicate stops matching (proxy for a reword).* Copy of the
working tree at `/tmp/fj065h/redA` with only `internal/config/load.go`'s marker
changed from `" already defined at line "` to `" already declared at line "`;
whole `internal/config` package, `go test ./internal/config -count=1`:

```
--- FAIL: TestLoadBoundsRepeatedYAMLKeyDiagnostic/{block,flow}_mapping
--- FAIL: TestYAMLDuplicateKeyDiagnosticMarkerPinsDecoderWording/{block,flow}_mapping
    yamlDuplicateKeyDiagnostic("line 2: mapping key \"a\" already defined at line 1") = false, want true
--- FAIL: TestLoadFileBoundsRepeatedYAMLKeyDiagnostic
    repeated keys=64: diagnostic is 108347 bytes, above the 512-byte bound; the
    duplicate-key diagnostic is no longer bounded: "… mapping key \"a\" already defined at line 1\n …"
--- FAIL: TestLoadStillRejectsRepeatedKeys/{block_yaml_mapping,flow_yaml_mapping}
--- PASS: TestLoadRepeatedKeyNamingIsDeterministicPerSyntax
```

*Simulation B — the dependency's wording actually changes.* Copy of `gopkg.in/yaml.v3
@v3.0.1` at `/tmp/fj065h/yamlv3` with `decode.go:776` reworded to
`"line %d: mapping key %#v already declared at line %d"`, wired into a copy of the
tree at `/tmp/fj065h/redB` with `replace gopkg.in/yaml.v3 => /tmp/fj065h/yamlv3`:

```
--- FAIL: TestYAMLDuplicateKeyDiagnosticMarkerPinsDecoderWording/{block,flow}_mapping
    yaml.v3 diagnostic = "line 2: mapping key \"a\" already declared at line 1", want
      "line 2: mapping key \"a\" already defined at line 1": the dependency reworded
      its duplicate-key line, so yamlDuplicateKeyDiagnostic no longer recognises it
      and the bound stops applying
--- FAIL: TestLoadFileBoundsRepeatedYAMLKeyDiagnostic
    repeated keys=64: diagnostic is 110363 bytes, above the 512-byte bound …
--- FAIL: TestLoadBoundsRepeatedYAMLKeyDiagnostic/{block,flow}_mapping
--- FAIL: TestLoadStillRejectsRepeatedKeys/{block_yaml_mapping,flow_yaml_mapping}
--- FAIL: TestLoadRepeatedKeyNamingIsDeterministicPerSyntax/{yaml_block,yaml_flow}
```

Simulation A reddens four tests (the marker pin, the new file guard, and the
coder's two YAML bound tests); simulation B additionally reddens the naming test,
whose `want` strings quote the library's wording. No test in the package passes
while the bound is broken: the two independent byte-bound assertions
(`TestLoadFileBoundsRepeatedYAMLKeyDiagnostic` at 64 and 512 repeats,
`TestLoadBoundsRepeatedYAMLKeyDiagnostic` up to 800) fail on *size*, which is the
product promise, not on a string comparison. Simulation B also produced an
incidental confirmation of yaml.v3's pair order (`line 4: mapping key "a" already
declared at line 1` before `line 3: mapping key "b" …`), which is the ordering the
coder's filter relies on.

**Green proof.** In the working tree all three tests pass:
`go test ./internal/config -count=1 -run 'TestYAMLDuplicateKeyDiagnosticMarkerPinsDecoderWording|TestLoadFileBoundsRepeatedYAMLKeyDiagnostic|TestLoadRepeatedKeyNamingIsDeterministicPerSyntax' -v`
→ `ok fake-jev/internal/config 0.594s`, all subtests `PASS`.

## 2. Residual 2 — no absolute cap, size affine in the key text: recorded, not bounded

**The relation, measured at the control route** (body `{"<key>":1,"<key>":2}` POSTed
to `/__fake/v1/stubs`, status 400 in every row; the body is the fixed envelope
(105 bytes) plus the key name as it appears in the message, so
`response = 105 + len(key)` for a plain ASCII key and more only by the escaping
the message needs):

| key text | request bytes | response bytes | ratio |
|---|---|---|---|
| `id` | 15 | 107 | 7.13 (fixed overhead dominates) |
| `k`×400 | 811 | 505 | 0.62 |
| `k`×407 / ×408 | 825 / 827 | **512 / 513** | 0.62 |
| `k`×1 048 490 | 2 096 991 | 1 048 595 | 0.50 |
| `\u0000`×100 000 (escaped) | 1 200 011 | 500 105 | 0.417 |
| `"`×500 000 (escaped) | 2 000 011 | 2 000 105 | 1.000 |
| `U+0080`×1 000 (raw UTF-8) | 4 011 | 7 105 | **1.771** |
| `U+0080`×500 000 (raw UTF-8) | 2 000 011 | 3 500 105 | **1.750** |
| `U+00A0`×1 000 (raw UTF-8) | 4 011 | 7 105 | 1.771 |
| `U+2028`×1 000 (raw UTF-8) | 6 011 | 7 105 | 1.182 |
| raw `0xff`×1 000 (invalid UTF-8) | 2 011 | 77 | 0.038 (`Invalid control request.`: the control decoder rejects the body before the loader sees it) |

The worst row is the non-printable two-byte rune, and it is the honest correction
to my own first reading of this residual. An attacker cannot beat it: the message
spends at most 7 bytes per decoded key character (`%q` writes `\uXXXX` and the
envelope doubles the backslash), while the request must spell each such character
as at least 2 raw bytes **twice** to repeat the key — `7 / (2·2) = 1.75`, which is
exactly what the large row measures. So the response is `≤ 105 + 1.75 × request`
bytes, i.e. at most about 3.5 MB for the largest legal control request, since
§19.4 caps that body at 2 MiB — the response stays bounded *even though the
specification sets no response bound*. The amplification the item exists to
remove is gone: 14 891 request bytes → 107 response bytes (0.0072×) where `c70d55b`
answered 71 635 303 bytes (4 811×). The one row that reads like an amplification
(1.75×) is a constant factor on key *content*, not a growth with the repetition
count, and it needs no repetition at all (`{"<key>":1,"<key>":2}` is the whole
request).

**Decision: record, do not bound.** Reasons, in order of weight.

- The property the acceptance asks for is "a small fixed bound rather than growing
  with the repetition count", and it holds, on every parse path. The remaining
  size is a linear function of caller-supplied key *content*, not of structure or
  of the repeat count.
- It is not amplification in the sense that matters: the worst constant measured
  is 1.75× and it needs no repetition at all; the remaining size is a linear
  function of caller-supplied key *content*, not of structure or of the repeat
  count, and the fixed control-plane request limit caps it at ≈3.5 MB.
- Any absolute cap means one of two unapproved behaviour changes: truncating the
  message, which would break acceptance C4's "names the offending key" for long
  keys and degrade a diagnostic §41.1 requires to be human-readable, or clamping
  the key text inside the message, which silently changes the documented failure
  message. The approved direction says only "leave it as a recorded residual or
  bound it" and forbids a new configuration key or a document size limit; a
  truncation rule would be a third thing the specifier did not sanction.
- Capping the message would also mask residual 1 instead of guarding it: with an
  absolute cap the reworded-dependency failure would become a silently truncated
  message again, which is exactly the fail-open the guard tests now reject.

**Named residual.** (a) The worst-case factor is ≈1.75× and it is reachable with a
2 MiB control body (measured 2 000 011 → 3 500 105 bytes with the body key being
500 000 repetitions of U+0080, i.e. `{"<key>":1,"<key>":2}` and no other
repetition); the response is bounded by `105 + 1.75 × request` but not by an
absolute constant. (b) On the file paths there is no request-size limit at all —
the unbounded configuration read recorded after FJ-030 and explicitly out of scope
— so a local YAML/JSON file can emit a stderr line proportional to its key text
(linear in the file, never in the repeat count); a config file whose repeated key
is raw invalid UTF-8 is also reachable there, where `config.Load` reports it (the
cleaner measured `\xff`×400 → a 1 250-byte `Load` message, ≈3 bytes per input
byte) while the control route answers that body with the generic 77-byte
`Invalid control request.` before the loader runs. Measured here for ordinary
configs: `validate` on a 62 921-byte JSON config emits 90 bytes; on a 50 090-byte
block-YAML config emits 140 bytes. (c) The 512-byte value asserted by the tests is
an implementation choice, as the coder recorded; nothing in the product clamps at
it, and the 407/408-byte boundary above shows exactly where it stops holding for
plain keys.

## 3. Residual 3 — which key is named: recorded, with the naming rules stated per route

**Measured divergence** (`fake-jev validate`, and the same shape at `Load` level):

| document | route | message contains |
|---|---|---|
| `{"a":1,"b":1,"b":2,"a":2}` | JSON scan | `duplicate object key "b"` |
| `a: 1\nb: 1\nb: 2\na: 2\n` | yaml.v3 | `line 4: mapping key "a" already defined at line 1` |
| `{a: 1, b: 1, b: 2, a: 2}` | yaml.v3 | `line 1: mapping key "a" already defined at line 1` |

The JSON scan reports the key whose **second** occurrence it reaches first,
because it stops at the first repeat. yaml.v3's pair loop reports the pair with
the earliest **first** occurrence, because its list is produced by an
`i`-then-`j` scan (`decode.go:769-780`). Both are deterministic functions of the
document alone, and the cleaner's sweeps found no verdict movement from the
difference; the questions the task asked are answered as follows.

- **Acceptable?** Yes, recorded. The two routes partition the inputs — a document
  with a repeat that the JSON scan can read goes to the JSON scan, everything else
  goes to yaml.v3 — so this is a naming difference *between two document syntaxes*,
  not an inconsistency for one input. Each route names the first offending key by
  its own document-order walk, which is what the acceptance requires.
- **Unify?** Not without adding cost and complexity. Unifying on the yaml.v3 rule
  needs the JSON scan to stop early-exiting and instead walk the whole document
  collecting repeats and their first positions: more work on the untrusted path
  (the scan currently allocates a constant ≈3.5 KB for documents from 1 721 to
  30 921 bytes) plus new bookkeeping, for a message difference on rejected input.
  Unifying on the JSON rule is not possible — yaml.v3's diagnostic list is built
  inside the dependency. So the divergence is recorded rather than unified.
- **Determinism for identical input, confirmed.** Each of the three forms above was
  loaded 200 times and produced one distinct message
  (`TestLoadRepeatedKeyNamingIsDeterministicPerSyntax`, with the two rules named in
  the test names). Independently, at process level: 200 `validate` runs on a
  repeated-key JSON config produced one distinct stderr (90 bytes), 200 runs on a
  repeated-key block-YAML config produced one distinct stderr (140 bytes), and 200
  identical control POSTs produced one distinct `(status, body)` pair (400, 107
  bytes).
- **Correction of the coder's artifact rule.** `coder.md` §1.3 states a single
  rule for both routes — "the first key that is reached for the second time by a
  walk of the document in document order" — which, as the cleaner showed, holds
  only for the JSON route; the YAML route names the pair with the earliest first
  occurrence. `coder.md` is outside my write set (`internal/config`,
  `internal/control`, and this artifact), so the corrected rule is recorded here:
  **JSON route — the key whose second occurrence comes first in token order; YAML
  route — the pair with the earliest first occurrence, as yaml.v3 reports it.** The
  code comments are already route-specific and true: `duplicateKeyError` says "the
  first object key that the JSON scan found repeated while walking a document in
  token order", and `boundDuplicateKeyDiagnostics` says "the first one yaml.v3
  produced, which its document-order pair scan makes a function of the document
  alone". I added no comment in either file that generalises beyond its route.

## 4. The duplicate-key error is still an error

Bounding did not turn a rejection into an acceptance anywhere. Evidence:
`validate` exits 2 with empty stdout and `serve --config` exits 2 on the
repeated-key JSON and block-YAML fixtures at N = 2 … 6 400; the control route
answers 400 with the fixed envelope at N = 2 … 16 000; the new file guard asserts
a non-nil error at both counts; the coder's
`TestLoadStillRejectsRepeatedKeys` (including the open `then.raw.body` and
`then.raw.headers` containers) and `TestControlRepeatedJSONKeyResponseStaysBounded`
cover the same property at `Load` and at the handler; and the acceptance decision
itself is unchanged (no test, old or new, observes an accepted document that has
an exact repeated key).

## 5. End-to-end measurements after the change (this stage, own driver)

Control route `POST /__fake/v1/stubs`, body `{"id":0,…,"id":N-1}`:

| N | request bytes | status | response bytes |
|---|---|---|---|
| 2 | 15 | 400 | 107 |
| 16 | 119 | 400 | 107 |
| 256 | 2 195 | 400 | 107 |
| 1 600 (inside the 16 KiB acceptance row) | 14 891 | 400 | 107 |
| 6 400 | 62 891 | 400 | 107 |
| 16 000 | 164 891 | 400 | 107 |

Body at every N: `{"error":"fake_jev_bad_control_request","message":"decode JSON
configuration: duplicate object key \"id\""}`.

File paths, stderr bytes / exit code / duplicate-key lines:

| command | N | input bytes | exit | stderr bytes | `already defined` lines |
|---|---|---|---|---|---|
| `validate dup.json` | 2 / 1 600 / 3 200 / 6 400 | 45 / 14 921 / 30 921 / 62 921 | 2 | 87 / 90 / 90 / 90 | 0 (JSON wording) |
| `validate dup.yaml` (block) | 2 / 1 600 / 3 200 / 6 400 | 10 / 11 690 / 24 490 / 50 090 | 2 | 137 / 140 / 140 / 140 | **1** at every N |
| `validate flow-64.yaml` | 64 | 438 | 2 | 139 | **1** |
| `serve --config dup-3200.json` | 3 200 | 30 921 | 2 | 90 | 0 |

Exact texts (path removed by the fixture name only in the digit-count column):
`fake-jev: <path>: decode JSON configuration: duplicate object key "id"`;
`fake-jev: <path>: decode YAML configuration: yaml: unmarshal errors:\n  line 2:
mapping key "a" already defined at line 1`.

Cost shape (median of 5 `validate` runs, this stage): JSON config
0.0033 s at N=1 600 and 0.0031 s at N=3 200 (flat, bounded by the first repeat;
the per-run spread max/min was 1.05–1.10); block YAML 0.1893 s → 0.7056 s for the
same doubling (spread 1.18–1.26), i.e. the residual quadratic inside yaml.v3
recorded below.

## 6. Commands and results

Each command below was run in the working tree; `go test ./...`, `go test -race
./...`, `go vet ./...`, `go build ./cmd/fake-jev`, the four `guard` rows,
`verify-candidate` and `candidate-fingerprint` were each run exactly once,
uncached, in this order. (The targeted `-v` run came before the throwaway
simulations; the full `go test ./internal/config` run afterwards covers the same
shipped files and reported the same result.)

| command | result |
|---|---|
| `gofmt -w internal/config/load.go internal/config/duplicate_key_diagnostic_test.go`; `gofmt -l internal/config internal/control` | no output (formatted) |
| `go test ./internal/config -count=1` | `ok fake-jev/internal/config 0.456s` |
| `go test ./internal/config -count=1 -run '<the three new tests>' -v` | `ok 0.594s`, every subtest `PASS` |
| `go test ./... -count=1` | exit 0; `ok` for `cmd/guard` 10.8 s, `internal/cli` 25.8 s, `internal/compat/jev/v1` 1.2 s, `internal/config` 1.5 s, `internal/control` 1.8 s, `internal/engine` 2.3 s, `internal/host/http` 2.9 s, `test/integration` 3.4 s; `no test files` for `cmd/fake-jev`, `examples/go` |
| `go test -race ./... -count=1` | exit 0; same packages, `internal/config` 4.6 s, no race report |
| `go vet ./...` | exit 0, no output |
| `go build ./cmd/fake-jev` | exit 0 (writes the git-ignored `./fake-jev`, `.gitignore:23`) |
| `go run ./cmd/guard arch` | `findings: []`, 63 files / 10 packages |
| `go run ./cmd/guard lint` | `findings: []` |
| `go run ./cmd/guard trace` | `findings: []`, covered 17, test_files 25 |
| `go run ./cmd/guard fuzz` | `findings: []`, `targets: 1`, 31.1 s wall, exit 0 |
| `./scripts/verify-candidate FJ-065` | exit 0. Report `.agent/reports/FJ-065/report-20261003T182826Z.json`. `summary: 20 passed, 0 failed, 0 skipped, 0 not_applicable`; `RESULT: PASS`; report candidate fingerprint `645887fd…`, report task fingerprint `ee28135d…` |
| `./scripts/candidate-fingerprint candidate` | `645887fd92010e90db08b55ca75104a56a43c67d918038b463184a968a75d5f4` |

**verify-candidate rows, exactly.** All twenty rows are `result=pass`,
`required=true`, `exempted=false`; there is no `fail`, no `skip`, and no
`not_applicable` row, so none needs an explanation of why it does not apply. The
rows are: `check_candidate_fingerprint`, `check_required_files`,
`check_scripts_executable`, `check_shell_syntax`, `check_toolchain_python3`,
`check_spec_present`, `check_json_files`, `check_work_item_schema`,
`check_recorded_report_coverage`, `check_role_packs`, `policy.exemptions.unknown`,
`policy.exemptions.stale`, `policy.classification`, `policy.required_stages`,
`check_go_test`, `check_go_vet`, `check_go_build`, `check_guard_arch`,
`check_guard_lint`, `check_guard_trace`. The report notes the pinned module
environment (`GOFLAGS=-mod=readonly GOPROXY=off GOWORK=off GOTOOLCHAIN=local`,
C-QUAL-007). The fuzz row is **not** part of `verify-candidate` on this branch and
was run separately (row above); it is the FJ-040 target the acceptance mentions.
The work item is left open: `state.json` was not edited and the only report in
`.agent/reports/FJ-065/` is the one `verify-candidate` wrote itself.

`git status --porcelain` at the end: ` M .agent/work/FJ-065/state.json` (the
PM/specifier-stage edit, untouched by this stage), ` M internal/config/load.go`,
` M internal/control/handlers_test.go` (coder's),
`?? internal/config/duplicate_key_diagnostic_test.go`,
`?? .agent/work/FJ-065/{specifier,coder,cleaner,hardener}.md`,
`?? .agent/reports/FJ-065/`. `git diff --cached --name-only` is empty: nothing is
staged.

## 7. Residual risks recorded for the PM

1. **yaml.v3's decode cost stays quadratic for YAML input.** The PM ruled this out
   of scope for FJ-065 (no third-party surgery, no document-size cap), so it is
   recorded rather than chased. Only the echoed diagnostic is bounded. Measured
   here: `validate dup.yaml` 0.1893 s → 0.7056 s per doubling at N = 1 600 → 3 200
   (≈3.7×, not 2×), while the emitted message stays 140 bytes with exactly one
   duplicate line at N = 2, 1 600, 3 200, 6 400. The reachable untrusted path is
   the JSON one (the control route always presents a JSON document via
   `validationDocument`); the YAML path needs a local file or a non-JSON flow
   document.
2. **The predicate stays textual.** Made loud, not robust (section 1): yaml.v3
   offers no structural handle, and a fuzzier pattern would only trade one silent
   failure for another. Any future upgrade of `gopkg.in/yaml.v3` must run
   `TestYAMLDuplicateKeyDiagnosticMarkerPinsDecoderWording` and
   `TestLoadFileBoundsRepeatedYAMLKeyDiagnostic` first.
3. **The recorded size residual of section 2** (affine in the key text,
   `≤ 105 + 1.75 × request`, worst measured 3 500 105 bytes from a 2 000 011-byte
   control body, no absolute cap; the file paths have no request limit at all —
   FJ-030).
4. **The recorded naming divergence of section 3** and the corrected rule that
   supersedes `coder.md` §1.3.
5. **FJ-040's test-side bound** (`configFuzzMaxMappingKeys`) does not exist on this
   branch, so nothing about this item had to be reconciled with it here.
