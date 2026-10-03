---
stage: qa
task: FJ-064
inputFingerprint: f6dfb54c6debeda144bf07f521f013c381ef84fa812e6bc63756863113ec56d1
outputFingerprint: f6dfb54c6debeda144bf07f521f013c381ef84fa812e6bc63756863113ec56d1
taskFingerprint: 71aa852bd544dfae210d4e52273e9036c9347ba7f7d92dacc4586a9c9da6e42a
gitHead: ab45ac5
generatedAt: 2026-10-03T14:32:48Z
author: worker/FJ-064-qa
---

# QA — FJ-064

Fresh-session QA of the frozen candidate. No product file, `state.json`, or report
was touched; nothing was written except this file. My author
(`worker/FJ-064-qa`) differs from the coder (`worker/FJ-064-coder`) and the
hardener (`worker/FJ-064-hardener`).

## Method and environment

- Candidate frozen: `./scripts/candidate-fingerprint candidate` returned
  `f6dfb54c6debeda144bf07f521f013c381ef84fa812e6bc63756863113ec56d1`, which equals
  the front-matter `inputFingerprint` and the hardener's output.
- Binary built once: `go build -o /private/tmp/fj064-qa/fake-jev ./cmd/fake-jev`
  (exit 0, 0.7 s). Reused for every CLI probe below.
- Direct `config.Load`/`config.Validate` probes used a scratch `main` package
  compiled with `go build -overlay=/private/tmp/fj064-qa/overlay.json -o
  /private/tmp/fj064-qa/qa_probe ./cmd/qa_probe`. The overlay maps a virtual
  `cmd/qa_probe/main.go` to `/private/tmp/fj064-qa/probe/qa_probe.go`; **nothing
  was written into the repository**, which is why `internal/config` could be
  exercised in-process while the build-once budget held.
- Scratch: `/private/tmp/fj064-qa`. Every hang probe ran under `timeout 10` (the
  control-API probe under `urllib` `timeout=10`).

Mechanism present in the candidate (read, not assumed): the coder's
document-wide case-variant fold was replaced. `internal/config/load.go` now has
`sortedKeys`/`orderYAMLKeys` only; `internal/config/validate.go` has
`rejectUnexpectedFields`, applied on the raw document at each project-owned
object. `grep -rn 'foldKey\|rejectCaseVariantKeys\|SimpleFold' internal/config`
matches only comments/one assertion in `collision_test.go`; there is no fold
helper.

## Criterion 1 — bounded hang probes (claim: the hang is gone)

`./fake-jev validate <file>` (which is `config.LoadFile` → `config.Load`) under
`timeout 10`:

| fixture | content | exit | output (truncated) |
|---|---|---|---|
| `h_top_cjk.json` | `{"schemaVersion":1,"中":1}` | 2 | `decode configuration: json: unknown field "中"` |
| `h_top_emoji.json` | `{"schemaVersion":1,"🙂":1}` | 2 | `unknown field "🙂"` |
| `h_top_symbol.json` | `{"schemaVersion":1,"§":1}` | 2 | `unknown field "§"` |
| `h_top_cased.json` | `{"schemaVersion":1,"é":1}` | 2 | `unknown field "é"` |
| `h_rawbody_cjk.json` | `then.raw.body` key `中` | 0 | `configuration is valid` |
| `h_state_cjk.json` | `when.state` key `中` | 0 | `configuration is valid` |
| `h_answers_cjk.json` | `then.answers` key `中` | 2 | `stub "x": missing answer for question "a"` (rejected later by the compat layer, not a hang) |
| `h_seq_raw_cjk.json` | sequence `raw.body` key `中` | 0 | `configuration is valid` |
| `h_yaml_cjk.yaml` | YAML block, key `中` | 2 | `unknown field "中"` |
| `h_invalid_utf8_top.bin` | bytes `{"schemaVersion":1,"\xff\xfe":1}` | 2 | `unknown field "��"` (`encoding/json` maps the bytes to U+FFFD) |
| `h_invalid_utf8_key_only.bin` | bytes `{"schemaVersion":1,"\x80":1}` | 2 | `unknown field "�"` |
| `h_invalid_utf8_body.bin` | `raw.body` key byte `\xff` | 0 | `configuration is valid` |

No fixture produced exit 124 and no probe exceeded the 10 s bound. The
control-API path (untrusted input → `internal/control/handlers.go:280`
`validationDocument` → `config.Load`) was also probed against a live server
(§ below): the two uncased-key stub bodies returned HTTP 400 in ≤6 ms.

## Criterion 2 — fail-closed is general, at every struct-decoded level

For each level a valid base document was loaded (`INVARIANT-OK`, i.e. `Load` and
`Validate` both succeed), then the same document with one case-variant key added
at that level was loaded. Each variant was rejected, and the diagnostic is the
guard's (`... is not a known field`), **not** `json: unknown field ...` — which
is independent proof that `encoding/json`'s fold-match accepted the key and the
guard, not `DisallowUnknownFields`, is what fails closed.

| level | case-variant key added | base | variant outcome |
|---|---|---|---|
| configuration root | `"SERVER"` | OK | `configuration.SERVER is not a known field` |
| `server` | `"HOST"` | OK | `server.HOST is not a known field` |
| `limits` | `"MAXINTERACTIONS"` | OK | `limits.MAXINTERACTIONS is not a known field` |
| stub envelope | `"PRIORITY"` | OK | `stubs[0].PRIORITY is not a known field` |
| `stubs[].expect` | `"EXACTLY"` | OK | `stubs[0].expect.EXACTLY is not a known field` |
| `stubs[].when` | `"OPERATION"` | OK | `stubs[0].when.OPERATION is not a known field` |
| `stubs[].then` | `"MODEL"` | OK | `stubs[0].then.MODEL is not a known field` |
| `stubs[].then.raw` | `"BODY"` | OK | `stubs[0].then.raw.BODY is not a known field` |
| `stubs[].then.usage` | `"INPUT_TOKENS"` | OK | `stubs[0].then.usage.INPUT_TOKENS is not a known field` |
| `stubs[].then.sequence[]` | `"MODEL"` | OK | `stubs[0].then.sequence[0].MODEL is not a known field` |
| `stubs[].then.sequence[].raw` | `"BODY"` | OK | `stubs[0].then.sequence[0].raw.BODY is not a known field` |
| `stubs[].then.sequence[].usage` | `"INPUT_TOKENS"` | OK | `stubs[0].then.sequence[0].usage.INPUT_TOKENS is not a known field` |
| `models[]` | `"DESCRIPTION"` | OK | `models[0].DESCRIPTION is not a known field` |

No level accepted a non-exact key. Two extra spellings were also checked:
`{"schemaVersion":1,"ſchemaVersion":1}` (U+017F long s, decoder-fold-equivalent
to `s`) → `configuration.ſchemaVersion is not a known field`; and a
non-ASCII-named key in a struct object (`{"server":{"host":"a","port":0,"中":1}}`)
→ `server.中 is not a known field`. Both rejected.

An intermediate run of this matrix printed one apparent hole
(`stubs[].then.sequence[].raw` accepted) and one spurious
`json: unknown field "PRIORITY"`; both were my own malformed fixtures (extra
brace / key inserted at the wrong level). After correcting the fixtures the
results are the table above. Recorded here rather than silently dropped.

## Criterion 3 — the core invariant, attacked adversarially

`qa_probe invariant <file>` runs `config.Load`; if the error is nil it asserts
`cfg != nil` and `config.Validate(cfg) == nil`. Any row that is not `LOADERR` or
`INVARIANT-OK` would be a headline finding. No such row occurred.

| input | Load outcome |
|---|---|
| `{"schemaVersion":1,"sChemAVErsion":0}` | `LOADERR configuration.sChemAVErsion is not a known field` |
| `{"schemaVersion":1,"SCHEMAVERSION":0}` | `LOADERR configuration.SCHEMAVERSION is not a known field` |
| `{"schemaVersion":1,"sChemAVErsion":1}` (equal values) | `LOADERR configuration.sChemAVErsion is not a known field` |
| `{"schemaVersion":1,"mode":"strict","MODE":"permissive"}` | `LOADERR configuration.MODE is not a known field` |
| `{"schemaVersion":1,"server":{"host":"127.0.0.1","port":0,"PORT":70000}}` | `LOADERR server.PORT is not a known field` |
| `{"schemaVersion":1,"schemaVersion":1,"sChemAVErsion":1}` (exact + case dup) | `LOADERR decode YAML configuration: yaml: unmarshal errors: line 1: mapping key "schemaVersion" already defined` |
| `{"schemaVersion":1,"schemaVersion":1}` | `LOADERR` (same yaml duplicate-key path) |
| `{"schemaVersion":1,"limits":{"maxInteractions":1,"MaxInteractions":2}}` | `LOADERR limits.MaxInteractions is not a known field` |
| `{"schemaVersion":1,"server":5}` | `LOADERR decode configuration: json: cannot unmarshal number into ... config.ServerConfig` |
| `{"schemaVersion":1,"server":true}` | `LOADERR ... cannot unmarshal bool ...` |
| `{"schemaVersion":1,"server":[]}` | `LOADERR ... cannot unmarshal array ...` |
| `{"schemaVersion":1,"server":null}` | `LOADERR server must be an object` |
| `{"schemaVersion":1,"limits":3}` | `LOADERR ... cannot unmarshal number ... LimitsConfig` |
| `{"schemaVersion":1,"ſchemaVersion":1}` (long s) | `LOADERR configuration.ſchemaVersion is not a known field` |
| `{"schemaVersion":1,"limits":{},"limitsᴋ":1}` (Kelvin) | `LOADERR decode configuration: json: unknown field "limitsᴋ"` |
| YAML anchor `server: &srv {...}`, block stubs | `INVARIANT-OK` |
| YAML anchor + alias reuse across `models` | `LOADERR duplicate model name "a"` |
| YAML merge key `server: {<<: {host: 127.0.0.1, port: 0}}` | `INVARIANT-OK` |
| YAML flow `{schemaVersion: 1, sChemAVErsion: 0}` | `LOADERR configuration.sChemAVErsion is not a known field` |
| YAML block `schemaVersion: 1\nschemaversion: 0` | `LOADERR configuration.schemaversion is not a known field` |
| YAML `{07,2}` | `LOADERR convert YAML configuration: object key 2 is not a string` |
| YAML `{true: 1, false: 2}` | `LOADERR ... object key false is not a string` |
| sequence `raw.body` `{a:1}`, sibling `"Body"` at raw level | `LOADERR stubs[0].then.sequence[0].raw.Body is not a known field` |
| `{"schemaVersion":"1"}` | `LOADERR ... cannot unmarshal string ... type int` |
| `{"schemaVersion":1.0}` | `LOADERR ... cannot unmarshal number 1.0 ... type int` |
| `{"schemaVersion":1} {"schemaVersion":1}` | `LOADERR decode YAML configuration: yaml: did not find expected <document start>` |
| `[]` | `LOADERR ... unmarshal array into Go value of type config.Config` |
| `null` | `LOADERR configuration must be an object` |
| empty bytes | `LOADERR configuration is empty` |
| `{"schemaVersion":1,"models":[...],"MODELS":[...]}` | `LOADERR configuration.MODELS is not a known field` |
| `when.questions` case-variant sibling `"QUESTIONS"` | `LOADERR stubs[0].when.QUESTIONS is not a known field` |
| stub `"ID"` instead of `"id"` | `LOADERR stubs[0].ID is not a known field` |
| `then.usage` sibling `"INPUT_TOKENS"` | `LOADERR stubs[0].then.usage.INPUT_TOKENS is not a known field` |
| `{"SCHEMAVERSION":1}` (exact key absent) | `LOADERR configuration.SCHEMAVERSION is not a known field` |
| `{"schemaVersion":1,"Server":{}}` | `LOADERR configuration.Server is not a known field` |
| `{"schemaVersion":1,"schemaVersion ":1}` (trailing space) | `LOADERR decode configuration: json: unknown field "schemaVersion "` |
| YAML `server: &s` then `server: *s` (dup via alias) | `LOADERR decode YAML configuration: yaml: unmarshal errors: line 3: mapping key "server" already defined at line 2` |
| valid base stub document | `INVARIANT-OK` |

No input produced `BROKEN` (nil config with nil error, or a config that
`Validate` rejects).

## Criterion 4 — exempt containers are exempt, and survive to the wire

Acceptance (`./fake-jev validate`, exit 0) for case-variant pairs inside exempt
containers:

| config | exempt content | exit |
|---|---|---|
| `rt.json` | `then.raw.body` = `{"Ok":1,"ok":2,"中":3}` | 0 (`configuration is valid`) |
| `rt_seq.json` | `then.sequence[0].raw.body` = `{"Alpha":1,"alpha":2}` | 0 |
| `rt_state.json` | `when.state` = `{"Key":1,"KEY":2}` | 0 |

Decoded-payload preservation (`qa_probe load`): `rt.json` keeps
`{"Ok":1,"ok":2,"中":3}`, `rt_seq.json` keeps `{"Alpha":1,"alpha":2}`,
`rt_state.json` keeps `{"Key":1,"KEY":2}` (both spellings present).

Live server, ephemeral port (`serve --config <f> --port 0 --ready-file …`), then
`POST /v1/systemone` with `{"state":{},"model":"m","questions":{"q":{"type":"noul"}}}`:

- `rt.json` — response body bytes were byte-identical to the config's `body`
  value bytes, `{"Ok":1,"ok":2,"\u4e2d":3}` (the literal JSON escape of `中` is
  preserved). `verbatim-match=True`.
- `rt_seq.json` — response body bytes `{"Alpha":1,"alpha":2}`, byte-identical to
  the configured body. `verbatim-match=True`. Status 200.
- `rt2.json` — a JSON config whose body is written with whitespace
  `{"Ok": 1, "ok": 2, "中": 3}`. Response was
  `{"Ok":1,"ok":2,"中":3}`, exactly the `encoding/json`-compacted form of the
  configured body (`EQUAL=True`); both keys and both values preserved, order
  preserved, only insignificant whitespace removed. That whitespace compaction
  is `json.RawMessage` decoding via `decode`'s `raw()` and is pre-existing, not
  introduced by this change.

## Criterion 5 — determinism

`qa_probe det <file> 1000` ran 1000 successive `Load` calls per input and
collected the distinct `(failed, message)` outcomes. Every input produced
**exactly one** distinct outcome (24 of 24 rows would be needed to show a
violation; zero rows did):

| input | distinct outcomes | the single outcome |
|---|---|---|
| `{07,2}` (F2 repro) | 1 | `convert YAML configuration: object key 2 is not a string` |
| `{a: {1: x}, b: {2: y}}` | 1 | `... object key 1 is not a string` |
| `{true: 1, false: 2}` | 1 | `... object key false is not a string` |
| `{1: a, "2": b, true: c}` | 1 | `... object key true is not a string` |
| `{3: a, 1: b, 2: c, true: d, "z": e}` | 1 | `... object key true is not a string` |
| `{2.5: a, 1.5: b}` | 1 | `... object key 1.5 is not a string` |
| `{null: a, true: b}` | 1 | `... object key <nil> is not a string` |
| `{-3: a, 5: b, -1: c}` | 1 | `... object key -1 is not a string` |
| two invalid question types (YAML) | 1 | `stub "x" when: question "a" has invalid type "bogus"` |
| two null questions (JSON) | 1 | `stubs[0].when.questions.a.value must not be null` |
| raw header with empty name | 1 | `stub "r" then: raw header names must be non-empty` |
| accepted `{schemaVersion: 1}` | 1 | `OK` |

Different non-string key types (bool, int, float, `null`, mixed in one mapping)
all yielded one stable diagnostic, and the observed "first key" is consistent
with the documented total order (`%T\x00%v`, so type name first): e.g.
`{1: a, "2": b, true: c}` always reports `true`, and `{-3,5,-1}` always reports
`-1`. N = 1000, far above the required 200.

## Criterion 6 — non-regression

- `go test ./... -count=1` (once): every package `ok` — `cmd/guard` 8.3 s,
  `internal/cli` 23.7 s, `internal/compat/jev/v1` 1.5 s, `internal/config` 0.65 s,
  `internal/control` 2.5 s, `internal/engine` 2.0 s, `internal/host/http` 3.0 s,
  `test/integration` 3.4 s. `internal/cli` (which contains the `validate` tests
  `TestValidateAcceptsValidConfigurationFiles`,
  `TestValidateRejectsMalformedFixtureDocuments`,
  `TestValidateHandlesLargeFixtureKeySets`, `TestValidateStreamsAreDisjoint`,
  `TestValidateIsStricterThanTheServeLoadPath`, …) passed; `test/integration`
  passed.
- `go test -race ./... -count=1` (once): every package `ok`.
- `go vet ./...`: exit 0, no output.
- Repository data files: `examples/fake-jev.yaml` still loads
  (`./fake-jev validate examples/fake-jev.yaml` returns `configuration is valid`
  — exercised through the CLI suite and directly in this stage's level probes).

## Criterion 7 — scope

- `git diff --stat ab45ac5 -- .` shows exactly:
  `.agent/work/FJ-064/state.json` (modified by an earlier stage, not this one),
  `internal/config/load.go` (47 lines), `internal/config/validate.go` (122
  lines).
- `git status --short`: modified `internal/config/load.go`,
  `internal/config/validate.go`, `.agent/work/FJ-064/state.json`; untracked
  `internal/config/collision_test.go`, `internal/config/determinism_test.go`, and
  the `.agent/` stage artifacts. No file outside `internal/config/` (product) and
  `.agent/` (metadata) changed. `git diff --cached` is empty (nothing staged).
- `internal/config/fuzz_test.go` does not exist (`ls` → "No such file or
  directory").
- No `func Fuzz` symbol exists in `internal/config` (`grep -rn "func Fuzz"
  internal/config` → NONE). `go run ./cmd/guard fuzz` reports `targets: 1`, the
  pre-existing `FuzzValidateFixtureAnswers` in `internal/compat/jev/v1`.
- The coder's original hanging revision: **not reachable anywhere in the shipped
  tree.** `grep -rn 'foldKey\|rejectCaseVariantKeys' internal/config` matches
  nothing; `git log --all -S'foldKey' --oneline` and
  `git log --all -S'rejectCaseVariantKeys' --oneline` are both empty across all
  160 ref-reachable commits; `git show ab45ac5:internal/config/load.go` contains
  no fold code. The intermediate revision was never committed, so no ref
  (working tree, `main`, `origin/main`) can reach it. The gitignored repo-root
  `fake-jev` build artifact also contains `is not a known field` and zero
  occurrences of the coder's `differ only in case` string.

## Commands run

| command | result |
|---|---|
| `gofmt -l internal/config/{load,validate,collision_test,determinism_test}.go` | no output, exit 0 |
| `go test ./internal/config -count=1` | `ok fake-jev/internal/config 0.652s` |
| `go test ./... -count=1` | all packages `ok` |
| `go test -race ./... -count=1` | all packages `ok` |
| `go vet ./...` | exit 0, silent |
| `go build -o /private/tmp/fj064-qa/fake-jev ./cmd/fake-jev` | exit 0 (built once) |
| `go run ./cmd/guard arch` | `findings: []`, files 62, packages 10, exit 0 |
| `go run ./cmd/guard lint` | `findings: []`, files 62, packages 10, exit 0 |
| `go run ./cmd/guard trace` | `findings: []`, active 0, covered 17, test_files 24, exit 0 |
| `go run ./cmd/guard fuzz` | `findings: []`, targets 1, exit 0 |
| `./scripts/verify-candidate FJ-064` | exit 0, `summary: 20 passed, 0 failed, 0 skipped, 0 not_applicable`, `RESULT: PASS`, report `.agent/reports/FJ-064/report-20261003T143136Z.json` |
| `./scripts/candidate-fingerprint candidate` | `f6dfb54c6debeda144bf07f521f013c381ef84fa812e6bc63756863113ec56d1` |

### verify-candidate row outcomes

All 20 rows passed; none failed, skipped, or was not applicable. Rows:
`check_candidate_fingerprint`, `check_required_files`,
`check_scripts_executable`, `check_shell_syntax`, `check_toolchain_python3`,
`check_spec_present`, `check_json_files`, `check_work_item_schema`,
`check_recorded_report_coverage`, `check_role_packs`,
`policy.exemptions.unknown`, `policy.exemptions.stale`,
`policy.classification`, `policy.required_stages`, `check_go_test`,
`check_go_vet`, `check_go_build`, `check_guard_arch`, `check_guard_lint`,
`check_guard_trace`. `guard fuzz` is not a `verify-candidate` row and was run
separately. This report records the row outcomes for the candidate revision; it
does not assert anything about the item's status.

## Residual risks and evidence limits

1. **No pre-fix binary was built.** The build-once budget was honoured, so the
   claim "these case-variant documents were accepted before the fix" is taken
   from the coder/cleaner/hardener evidence, not independently re-reproduced
   here. What *is* independently shown is the forward direction: for every
   case-variant at a struct level the decoder emitted no `json: unknown field` —
   so it accepted the key — and the guard then rejected it.
2. **Corpus, not exhaustive.** Criterion 3's invariant was attacked with 40
   documents and criterion 5 with 12 inputs × 1000 calls; neither is exhaustive
   over arbitrary bytes. FJ-040's fuzz target is the exhaustive check and is
   confirmed absent (no `fuzz_test.go`, no `Fuzz*` symbol); `guard fuzz` reports
   1 target.
3. **Hang probes are bounded.** All probes used a 10 s timeout and none was
   reached; no open-ended fuzzing was performed. Uncased keys were probed at
   root, `when`, `then.raw.body`, `when.state`, `then.answers`, a sequence raw
   body, and through the control API; a non-ASCII uncased key at an untested
   level is not excluded by construction, only by the mechanism (there is no
   fold helper left, so no level can enter a fold loop).
4. **Round-trip formatting.** `rt.json`/`rt_seq.json` round-trip byte-for-byte;
   `rt2.json` differs only by insignificant whitespace because `json.RawMessage`
   receives the decoder's compacted value. This compaction is pre-existing and
   unrelated to the case-variant change; it is recorded so "verbatim" is not
   over-read.
5. **`then.raw.headers` case-variant pairs are accepted** (`map[string]string`),
   and the HTTP writer sets them with `Header().Set`, which canonicalises names
   and can make the surviving value depend on map iteration (`internal/host/http/server.go:400`).
   This is a pre-existing order-dependence outside this item's scope; the
   hardener flagged it as a judgement call.
6. **Independent-appeal limitation.** The tests in `internal/config` that pin
   these properties were written by the change's author; this stage did not
   modify or re-derive them. Its independent checks are the CLI/`Load` probes
   above, using the built binary and an overlay-compiled probe.
