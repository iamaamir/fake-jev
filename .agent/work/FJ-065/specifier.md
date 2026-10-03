---
stage: specifier
task: FJ-065
inputFingerprint: f6dfb54c6debeda144bf07f521f013c381ef84fa812e6bc63756863113ec56d1
outputFingerprint: f6dfb54c6debeda144bf07f521f013c381ef84fa812e6bc63756863113ec56d1
taskFingerprint: ee28135d80573ff713fda4993a1571871f25f3ac6424a2b7e361c32336ad06ea
gitHead: c70d55b
generatedAt: 2026-10-03T17:35:36Z
author: worker/FJ-065-specifier
---

# FJ-065 specifier — bound the duplicate-key diagnostic

Objective under specification: make a configuration document that repeats one
object key produce an error whose message and whose parsing cost do not grow
quadratically with the repetition count, on every path where the document is
parsed, while leaving the documented control failure shape, the acceptance
decision, and FJ-064's determinism guarantees untouched.

This artifact states what will be observable, the exact deterministic commands
that make each observation, and the expected output. It is evidence, not a
result.

## 1. Authority, and what this item is not

The specification sets **no control-plane response size limit**. §19.4
("Resource limits") constrains *request* body sizes only (data plane 8 MiB,
control plane 2 MiB, interaction count, log preview, shutdown timeout); §41.1
fixes the control failure *shape* but not any message or response bound; and no
clause anywhere bounds a control-plane response or a diagnostic message. The
64 KiB response bound that FJ-040's control fuzz target asserts is **FJ-040's own
robustness criterion, not a spec clause** (recorded in `.agent/work/FJ-065/state.json`).

Therefore this item is **not** specified as enforcing a spec limit, and it must
not introduce a public or configurable limit setting (non-goal: "no new public
configuration key or user-visible limit setting"). It is justified as removing a
**quadratic resource amplification reachable from untrusted input**:

- §19's resource-bound posture requires the host to bound cost before unbounded
  allocation, and the amplification here is a small untrusted request producing
  an unbounded response;
- C-QUAL-004 requires malformed untrusted input to yield a **controlled error**
  rather than an uncontrolled one; a response that grows with the square of the
  repetition count is not controlled;
- the message must stay a **useful diagnostic** rather than collapsing into the
  generic "Invalid control request.", because §41.1 requires a *human-readable*
  `message` and the acceptance requires the offending key to be named.

Any numeric bound stated below is therefore an **implementation choice**, chosen
for headroom, and is stated as such at each criterion.

## 2. Defect, as re-measured at `c70d55b`

Mechanism (confirmed by reading the code and by reproduction): for input whose
first byte is `{` or `[`, `toJSON` (`internal/config/load.go`) returns the
original bytes only when `rejectDuplicateJSON` **succeeds**. A repeated key makes
`rejectDuplicateJSON` fail with its own linear `duplicate object key %q` error,
which is discarded and treated as "maybe YAML flow syntax"; the document then
reaches `decodeYAML`, and yaml.v3's key-uniqueness check emits one diagnostic per
repeated-key pair plus an internal O(n²) comparison loop, so both the echoed
message and the decoder cost are quadratic.

Reproduced end to end against a binary built from `c70d55b` (`go build -o
/tmp/fj065/fake-jev ./cmd/fake-jev`, `serve --port 0`), body =
`{"id":0,"id":1,…}` POSTed to `/__fake/v1/stubs`, HTTP status 400 in every row:

| N repeated keys | request bytes | response bytes | ratio |
|---|---|---|---|
| 2 | 15 | 159 | 10.6× |
| 4 | 29 | 439 | 15.1× |
| 16 | 119 | 6 823 | 57× |
| 256 | 2 195 | 1 827 943 | 833× |
| 1 600 (14.5 KiB) | 14 891 | 71 635 303 | 4 811× |
| 3 200 | 30 891 | 286 630 503 | 9 279× |
| 6 400 (61.4 KiB) | 62 891 | 1 146 700 903 | 18 233× |

The PM's independent reproduction against the merged candidate reports the same
shape from a differently sized body: 40 → 274; 640 → 111 424 (174×); 4 000 →
4 525 960 (1 131×); 16 000 → 72 823 360 (4 551×). Both sets are quoted because
the ratio depends on the padding of the request, not on the mechanism.

The same mechanism, same measurement protocol, on the file load paths:

| command | N | input bytes | emitted diagnostic bytes |
|---|---|---|---|
| `validate dup.json` | 1 600 | 14 922 | 67 797 716 |
| `validate dup.json` | 3 200 | 30 922 | 271 275 316 |
| `serve --config dup.json` | 1 600 | 14 922 | 67 797 716 |
| `validate dup.yaml` (block YAML `a: <i>` × N) | 3 200 | 24 490 | 293 326 022 |

Elapsed-time doubling steps (median-of-1 probe; the median-of-5 protocol of C3
is normative): control route N=1 600 → 3 200 = 255 ms → 1 175 ms (**4.6×**);
`validate dup.json` = 0.178 s → 0.656 s (**3.7×**); `validate dup.yaml` =
0.190 s → 0.842 s (**4.4×**). Response/diagnostic bytes over the same steps grow
**4.0×** in every case. Doubling the repetition count therefore multiplies cost
by ≈4, not ≈2: the quadratic shape.

## 3. Acceptance criteria

### C1 — control route: bounded response, independent of the repetition count

**Observable.** `POST /__fake/v1/stubs` with a body that is one stub object
repeating the key `id` N times answers HTTP 400 in every case, and the response
body length is **the same value for every N**, not a value that grows with N.

- Before (measured above): 159 / 439 / 6 823 / 1 827 943 / 71 635 303 /
  286 630 503 / 1 146 700 903 bytes at N = 2 / 4 / 16 / 256 / 1 600 / 3 200 /
  6 400. `size(3200)/size(1600) = 4.00`.
- After (expected, the smallest change satisfying C4): **107 bytes** at every N,
  the body
  `{"error":"fake_jev_bad_control_request","message":"decode JSON configuration: duplicate object key \"id\""}`.
  `size(3200)/size(1600) = 1.00`.

**Bound and its justification (implementation choice).** The control-plane 400
body must stay **≤ 512 bytes for any repeated key name of at most 400 bytes**
(measured after: `105 + len(key)` bytes, i.e. 107 for `id`, 361 for a 256-byte
key). 512 is chosen because it contains the §41.1 envelope, the fixed
`fake_jev_bad_control_request` code, and one diagnostic naming a key, with room
to spare; it sits 128 times below FJ-040's own 64 KiB fuzz assertion, so that
assertion stops being corpus-dependent; and it is **constant, not a limit on N**.
It is an implementation choice and not a specification limit (§1).

**Passing shape.** body length is a function of the repeated key name and the
error code alone; `size(N)` identical for all N ≥ 2.
**Failing shape.** any growth with N: the pre-fix value 1 146 700 903 bytes at
N = 6 400 is 7.2 × 10⁶ times the pre-fix value at N = 2.

### C2 — every parse path: bounded diagnostic

**Observable.** The bytes the process emits for a repeated-key configuration
document are bounded and do not grow with N on each of the three parse paths.

- `validate dup.json` (JSON config file): exit status 2, stdout empty, **stderr
  ≤ 512 bytes** for every N. Before: 67 797 716 bytes at N=1 600, 271 275 316
  bytes at N=3 200. After (expected): a single line,
  `fake-jev: <path>: decode JSON configuration: duplicate object key "id"`.
- `serve --config dup.json` (host load path, §15.1/§12.1): exit status 2,
  **stderr ≤ 512 bytes** for every N. Before: 67 797 716 bytes at N=1 600.
  After (expected): the same single line.
- `validate dup.yaml` (block-YAML document `a: <i>` repeated N times): exit
  status 2, **stderr ≤ 512 bytes**, and **exactly one** occurrence of the
  yaml.v3 duplicate diagnostic, naming the repeated key. Before: 293 326 022
  bytes at N=3 200 (24 490-byte input). After (expected), one line of the form
  `fake-jev: <path>: decode YAML configuration: yaml: unmarshal errors:` plus
  one `mapping key "a" already defined at line 1` line.

**Bound and its justification (implementation choice).** At most **one** echoed
yaml.v3 diagnostic line and at most **512 bytes** of emitted diagnostic per
path. This is an implementation choice, not a specification limit; it is the
same clamp as C1 so that one rule covers every parse path. The emitted size is a
function of a fixed path prefix plus the single named key and nothing else —
≈107 bytes for the JSON rows with key `id`, and a comparable single-figure value
for the YAML row with key `a` — so a limit of 512 covers repeated key names up to
roughly 350–400 bytes on every row. A document containing **more than one
distinct duplicated key** reports only the first; that information reduction is
accepted (recorded in §6).

**Passing shape.** diagnostic bytes at N=1 600 and N=3 200 identical, and each
≤ 512.
**Failing shape.** 271 275 316 bytes at N=3 200 (≈ 8 800 × the 30 922-byte
input).

### C3 — at most linear, not quadratic, in the repetition count

**Observable.** For the three paths, doubling the repetition count must not
multiply cost by ≈4.

**Protocol (median of five).** For each of N = 1 600 and N = 3 200, run the same
command five times and take the median elapsed wall time (`date +%s.%N` before
and after, or `/usr/bin/time -p`). Compare `median(3200) / median(1600)`. All
five runs of a step must first be internally consistent (max/min ≤ 1.5), which
the pre-fix quadratic behaviour satisfies trivially; a 6400-vs-3200 step is
available as a confirmation on the control route (61.4 KiB request, well inside
the 2 MiB control-plane limit of §19.4).

- **Passing shape:** ratio ≤ 2.5 on every path. A linear step costs ≈2×; fixed
  overhead makes the measured ratio smaller than 2 for small steps, never
  larger.
- **Failing shape:** ratio ≥ 3.5. Measured before: 4.6× (control route), 3.7×
  (`validate dup.json`), 4.4× (`validate dup.yaml`).
- **Inconclusive band** (2.5 < ratio < 3.5): re-measure at the next doubling
  step, 3 200 → 6 400, where a quadratic step is ≈4.0× and a linear step ≈2.0×.

**Deterministic companion (no timing).** Two byte-count facts already fix the
shape without clocks: (a) the emitted diagnostic byte count is identical across
N (C1, C2), and (b) the number of echoed `mapping key` / `already defined`
diagnostic occurrences is **at most 1** for every N, where the mechanism
produces `C(N,2)` of them before the fix. A Go-level test may additionally pin
total allocation for one `config.Load` of the JSON repeated-key document at
N = 3 200 to **at most 256 ×** the input length (30 922 bytes → ≤ 7.9 MB); the
pre-fix path allocates at least the 271 MB diagnostic, i.e. ≥ 8 700 × the input.

### C4 — the bounded message stays useful, and names a specific offending key

**Observable.** The 400 body's `message` value identifies the failure and names
the offending key:

- `message` contains the offending key name in quotes — `"id"` for the C1 body;
- `message` does **not** equal the generic `Invalid control request.` (bounding
  must not degrade the diagnostic into the generic malformed-body message);
- the same holds on the file paths: `validate`/`serve` stderr names the repeated
  key (`duplicate object key "id"` / `mapping key "a" already defined`).

**Ordering rule (prescribed).** The named key is **the first key that is
repeated when the document is walked depth-first in document order** — for a
JSON document, the order in which `json.Decoder.Token()` yields the tokens,
which is what `scanJSONValue` already produces; for a YAML document, the first
diagnostic yaml.v3 emits, which is itself in document order. This rule is
consistent with FJ-064's `sortedKeys`/`orderYAMLKeys` approach in the property
that matters: the named key is a function of the document alone and can never
depend on Go's randomized map iteration order. A sort is not required here
because the JSON token stream and yaml.v3's diagnostic list are already totally
ordered; where an implementation does report from an unordered collection of
duplicated keys, it must name the smallest key name in ascending byte order,
exactly as `sortedKeys` does.

**Observable for the rule.** `Load` on
`{"schemaVersion":1,"a":{"x":1,"x":2},"y":9,"b":{"z":3,"z":4}}` names `x`
(the first repeat reached in document order, inside `a`), while the same document
with the two groups swapped —
`{"schemaVersion":1,"b":{"z":3,"z":4},"y":9,"a":{"x":1,"x":2}}` — names `z`.
Neither answer may change between runs.

### C5 — determinism: byte-identical output for identical input

**Observable** (spec line 143, "tests must be deterministic and reproducible").
1000 identical POSTs produce byte-identical response bodies; 1000 `config.Load`
calls on the same repeated-key document return the same failure decision and the
same `err.Error()` string. This extends the existing protocol of
`internal/config/determinism_test.go::TestLoadDiagnosticsAreDeterministic`,
whose cases must keep holding, with at least these new cases:

- `{"schemaVersion":1,"a":{"x":1,"x":2},"b":{"y":1,"y":2}}`
- `{"schemaVersion":1,"a":{"x":1,"x":2},"y":9,"b":{"z":3,"z":4}}`
- `{"schemaVersion":1,"id":1,"id":2,"id":3}`
- the block-YAML form `a: 1\na: 2\nb: 1\nb: 2\n`

**Passing shape.** identical outcome and identical message on all 1000 calls for
each input.
**Failing shape.** any call differing from the first.

### C6 — the documented failure shape is unchanged

**Observable.** For the C1 request: HTTP status **400**; body is a JSON object
with exactly the members `error` and `message` (no `stubId`, no new member);
`error == "fake_jev_bad_control_request"`; `Content-Type: application/json`; and
no engine mutation — no stub registered, no interaction journaled, no
verification failure added (§41.1, §11; C-CTRL-010, C-CTRL-011; the machine-
readable shape is `spec/control-api.openapi.yaml` `BadControlRequestError`, whose
`message` is an unconstrained string, so a bounded message stays conformant).
For the file paths: exit status **2** (§42.1; C-CLI-001) and, for `validate`,
§15.2's guarantee of no server start and no network request (C-CLI-005).

**Passing shape.** `400` and `{"error":"fake_jev_bad_control_request","message":"…"}`
with `Content-Type: application/json`; `validate` exit 2 with empty stdout.
**Note (recorded, §6).** For JSON input the *wording* of an exact-duplicate
diagnostic changes from the yaml.v3 list to the single JSON-scan line. The
decision, the status, the `error` code, the envelope fields, and the exit code
are unchanged; §41.1 requires only a human-readable `message`. No committed test
pins the previous wording (checked: `grep -rn "mapping key" --include=*_test.go`
matches nothing).

### C7 — repeated keys still fail (the bound must not silently accept them)

**Observable.** A document with an exact repeated key is still **rejected**:
`config.Load` returns a non-nil error, `fake-jev validate` exits 2, and
`POST /__fake/v1/stubs` with the C1 body answers 400, never 201.

**Data and tests that already establish it and must keep doing so:**

- `internal/config/collision_test.go` — `TestLoadRejectsCaseVariantDuplicateKeys`,
  subtest `exact duplicate keys stay rejected` (a repeated `a` inside the exempt
  `then.raw.body` container), and `TestLoadNeverReturnsConfigThatValidateRejects`.
- `internal/control/handlers_test.go::TestControlStrictJSON` (rejected control
  bodies mutate no engine state).
- Behaviour recorded by FJ-064 (`.agent/work/FJ-064/qa.md`) that the duplicate
  rejection is **document-wide**, including inside exempt open containers
  (`then.raw.body`, `then.answers`, question names, raw header names), is a
  decision that does not change.

**Passing shape.** rejection with the same code/status as before; the *message*
may be the bounded one.

### C8 — no regression: accepted stays accepted, rejected stays rejected

**Observable.** `go test ./...` passes and every configuration accepted by
`c70d55b` is still accepted with the same decoded value, and every rejection
still rejects with the same code/status/exit code (the diagnostic wording may
change per §6).

**Existing tests and data that establish it (must keep passing unchanged):**

- `internal/config/validate_test.go`: `TestLoadDefaults`,
  `TestRejectsMultipleYAMLDocuments`, `TestAcceptsYAMLFlowDocument`,
  `TestJSONAndYAMLLoadSameModel` (§12 / C-CFG-009: YAML and JSON inputs
  deserialize to the same logical schema — the JSON fast path and the YAML
  fallback must not diverge), `TestRejectsUnknownAndInvalidConfiguration`
  (C-CFG-002), `TestRawBodyMayBeOmitted`, `TestRejectsNullAndEmptyStructuredValues`,
  `TestExpectForms`.
- `internal/config/collision_test.go` (all five tests, including
  `TestLoadAcceptsCaseVariantKeysInOpenContainers` and
  `TestLoadTerminatesOnUncasedKeys`) and `internal/config/determinism_test.go`.
- `internal/control/handlers_test.go`: `TestControlStrictJSON`,
  `TestControlBodyLimit` (the 2 MiB control limit of §19.4 and the 413
  `fake_jev_payload_too_large` path are untouched; C-HOST-006),
  `TestControlDynamicStubsLifecycleAndErrors`, `TestControlStubSchemaPreservation`,
  `TestControlVersionMetadata`.
- `internal/cli/validate_test.go`, `internal/cli/serve_test.go`,
  `internal/cli/run_test.go` (CLI exit codes and startup diagnostics, C-CLI-001).
- Data: `testdata/contracts/**` (24 golden contract vectors, including
  `testdata/contracts/validation/01-malformed-json.json`), `examples/fake-jev.yaml`,
  `spec/config.schema.json`.

**Specific non-regressions to keep visible** (each already covered by a named
test above, and each easy to break while touching `toJSON`): the YAML flow
document `{schemaVersion: 1}` is still accepted; `{07,2}` is still rejected with
`object key 2 is not a string` (the `map[any]any` / `orderYAMLKeys` path);
multiple YAML documents are still rejected; JSON `null` and `[]` still fail with
`configuration must be an object`; a valid JSON document is still returned
byte-for-byte unchanged; YAML `time.Time` still formats as `2006-01-02`.

### C9 — repository verification

**Observable.** `./scripts/verify-candidate FJ-065` exits 0 for the candidate,
under the pinned module environment of C-QUAL-007, with the Go checks of
C-QUAL-001 in place.

## 4. Deterministic commands and expected output

Build once:

```bash
cd "$(git rev-parse --show-toplevel)"
mkdir -p /tmp/fj065
go build -o /tmp/fj065/fake-jev ./cmd/fake-jev
```

Generate the fixtures (`N` is the repetition count):

```bash
N=1600
python3 -c "n=$N;print('{'+','.join('\"id\":%d'%i for i in range(n))+'}')" \
  > /tmp/fj065/dup-stub.json
python3 -c "n=$N;print('{\"schemaVersion\":1,\"stubs\":[{'+','.join('\"id\":%d'%i for i in range(n))+'}]}')" \
  > /tmp/fj065/dup.json
python3 -c "n=$N;print('\n'.join('a: %d'%i for i in range(n)))" \
  > /tmp/fj065/dup.yaml
```

**C1 (control route).**

```bash
/tmp/fj065/fake-jev serve --port 0 --ready-file /tmp/fj065/ready.json &
sleep 1
PORT=$(python3 -c "import json;print(json.load(open('/tmp/fj065/ready.json'))['port'])")
curl -s -o /tmp/fj065/resp.json \
     -w 'status=%{http_code} bytes=%{size_download}\n' \
     -H 'Content-Type: application/json' \
     --data-binary @/tmp/fj065/dup-stub.json \
     "http://127.0.0.1:$PORT/__fake/v1/stubs"
cat /tmp/fj065/resp.json
```

- Expected before (`c70d55b`, N=1 600):
  `status=400 bytes=71635303` and a body consisting of
  `{"error":"fake_jev_bad_control_request","message":"decode YAML configuration: yaml: unmarshal errors:\n  line 1: mapping key \"id\" already defined at line 1\n …"}`.
- Expected after (N=1 600): `status=400 bytes=107` and the body
  `{"error":"fake_jev_bad_control_request","message":"decode JSON configuration: duplicate object key \"id\""}`.
- Expected after for every N in {2, 16, 256, 1 600, 3 200, 6 400}: `bytes=107`.
- Killing the server afterwards keeps the fixture reusable.

**C2 / C3 (`validate`, `serve --config`, block YAML).**

```bash
for N in 1600 3200; do
  # regenerate dup.json and dup.yaml for this N as above
  /tmp/fj065/fake-jev validate /tmp/fj065/dup.json >/tmp/fj065/v.out 2>/tmp/fj065/v.err
  echo "N=$N exit=$? stdout=$(wc -c </tmp/fj065/v.out) stderr=$(wc -c </tmp/fj065/v.err)"
  /tmp/fj065/fake-jev serve --config /tmp/fj065/dup.json >/dev/null 2>/tmp/fj065/s.err
  echo "N=$N serve exit=$? stderr=$(wc -c </tmp/fj065/s.err)"
  /tmp/fj065/fake-jev validate /tmp/fj065/dup.yaml >/dev/null 2>/tmp/fj065/y.err
  echo "N=$N yaml exit=$? stderr=$(wc -c </tmp/fj065/y.err) dup-lines=$(grep -c 'already defined' /tmp/fj065/y.err)"
done
```

- Expected before (measured at `c70d55b`): `N=1600 exit=2 stdout=0
  stderr=67797716`; `N=1600 serve exit=2 stderr=67797716`; `N=3200 exit=2
  stdout=0 stderr=271275316`; `N=1600 yaml exit=2 stderr=72423591
  dup-lines=1279200`; `N=3200 yaml exit=2 stderr=293326022 dup-lines=5118400`
  (note the 4.05× step in diagnostic bytes for a 1.10× step in input bytes).
- Expected after: `exit=2 stdout=0` for every row; every `stderr` ≤ 512 and
  identical between N=1 600 and N=3 200; `dup-lines=1` for the YAML row.

Timing for C3 uses the same three commands with `date +%s.%N` around five
repetitions per N and the median ratio test stated in C3.

## 5. Out of scope

- **The unbounded configuration-file read recorded after FJ-030.** No
  configuration file size limit is added here; that item stays open and separate
  (non-goal of the work item). This artifact bounds only the *diagnostic* and the
  *duplicate-key parse work*, not file reading in general.
- **Any change to FJ-064's fail-closed exact-key behaviour or its determinism
  guarantee.** Whether a document is accepted or rejected, and the
  `sortedKeys`/`orderYAMLKeys` determinism of the existing walks, must not change.
  Only the human-readable wording of the exact-duplicate diagnostic changes for
  JSON input, which is a message change and not a decision change.
- **No new public configuration key, limit setting, or `/meta` field**; the
  numeric bounds of C1–C3 are internal implementation choices.
- **No change to FJ-040's test-side guard** (`configFuzzMaxMappingKeys` in
  `internal/config/fuzz_test.go`, on the parked FJ-040 branch). Whether it can be
  relaxed once the cause is fixed is FJ-040's decision; FJ-065 does not touch it.
- **No new dependency**: the whole change stays inside the standard library plus
  the already-required `gopkg.in/yaml.v3`.

## 6. Spec-silent observations to record

1. **No response bound exists in the specification.** §19.4 lists request-side
   limits only; §41.1 fixes the envelope and code but not the message. Every
   numeric bound in this artifact is an implementation choice.
2. **The message wording for JSON exact duplicates changes** from
   `decode YAML configuration: yaml: unmarshal errors: line 1: mapping key "x"
   already defined` to a single JSON-scan line. §41.1 requires only a
   human-readable `message`; the decision, status, code, and exit code are
   unchanged. Evidence artifacts that quoted the old wording become stale:
   `.agent/work/FJ-064/qa.md` (rows for `{"schemaVersion":1,"schemaVersion":1}`
   and the alias row) and `.agent/work/FJ-059/qa.md` rows `fp-dup-keys.json` /
   `fp-dup-prob-key.json`. No committed Go test pins that wording.
3. **Only the first offending key is reported.** A document with several
   distinct duplicated keys loses the remaining diagnostics; the acceptance
   requires a bounded, useful message naming *a* specific key, and the operator
   fixes that key and re-runs. Recorded as an accepted information reduction.
4. **Why the quadratic work exists at all:** yaml.v3's key-uniqueness check is a
   nested loop over the keys of each mapping (`gopkg.in/yaml.v3 decode.go`,
   `decoder.mapping`) that appends one `terrors` entry per repeated pair.
   Clamping only the *echoed* message therefore bounds the response but leaves
   the decoder cost quadratic; C3 requires the cost itself to be linear, which is
   why C3 is stated per path rather than only for the response bytes.
5. **The control route always presents JSON to the loader**
   (`internal/control/api.go`, `validationDocument` marshals the stub into a
   config JSON document), so the untrusted-input amplification of the defect is
   always on the JSON path; the block-YAML path of C2 is reachable only from a
   locally supplied file. This is the trust-boundary argument for treating the
   control route as the primary reachable amplification, and it is *not* a
   licence to leave the file paths unbounded: C2 and C3 are stated for all three
   paths.
6. **The 16 KiB acceptance figure.** The work item's acceptance says "at most
   16 KiB"; N = 1 600 produces a 14 891-byte stub body, so C1's headline row is
   inside that budget, and the larger N rows (3 200, 6 400) are extra evidence
   that the bound does not depend on N.
7. **The 413 path is separate and must stay separate.** A body over the 2 MiB
   control-plane limit answers 413 `fake_jev_payload_too_large` (§19.4) and is
   not affected by, and does not substitute for, the bounded 400 of C1
   (`TestControlBodyLimit`).

## 7. Traces

Acceptance-catalog IDs used by these criteria (every ID below exists in
`docs/development/acceptance-catalog.md`):

- **C-CTRL-010** — "Invalid control JSON and profile/config validation failures
  return 400 with the defined error codes." (§41.1) → C1, C6.
- **C-CTRL-011** — "Every control JSON response uses `Content-Type:
  application/json` except 204 responses." (§11) → C6.
- **C-CFG-002** — "Unknown keys fail validation at top level, …." (§12.3, §38.8)
  → C7, C8: the rejected/accepted boundary for configuration documents.
- **C-CFG-009** — "YAML and JSON inputs deserialize to the same logical schema."
  (§12) → C8: the JSON fast path and the YAML fallback must not diverge.
- **C-CLI-001** — "Exit codes are 0 success, 2 usage/config/startup/control/
  internal, …" (§42.1) → C2, C6.
- **C-CLI-005** — "`validate` performs all static checks without starting a
  server or making any network request." (§15.2) → C2, C6.
- **C-HOST-006** — "Control-plane requests never enter the journal; a
  control-plane 413 never affects verification." (§11, §41.1) → C6, C8.
- **C-QUAL-001** — "`go test ./...` passes." (§31, §23) → C8, C9.
- **C-QUAL-004** — "Fuzz targets show malformed untrusted input yields a
  controlled error and never a panic or corrupted state." (§22.2) → C1, C3, C4:
  the authority for calling this a *controlled* error at all.
- **C-QUAL-007** — "`./scripts/verify-candidate` runs the product Go checks under
  a pinned module environment …" (§2, §17.2, §31) → C9.

Specification clauses used directly (not catalog IDs): §19.4 (resource limits —
request-side only, so no response bound exists), §19.5 (the control API is local,
which is why C2/C3's file paths are stated explicitly rather than assumed),
§41.1 (control failure shape: 400, `fake_jev_bad_control_request`, a
human-readable `message`), §12.3 (configuration schema and unknown-key
rejection), spec line 143 ("tests must be deterministic and reproducible"),
§42.1 (exit codes).

## 8. Handoff notes for the coder stage

- The observable of C1 is achieved by not routing a document whose repeat the
  linear `scanJSONValue` already detected into the YAML decoder, and by keeping
  the YAML fallback for documents that are not JSON (the `TestAcceptsYAMLFlowDocument`
  case exists for exactly that reason).
- C2's block-YAML row is the one that cannot be satisfied by the JSON fast path
  alone: yaml.v3's diagnostic list has to be reduced to one line there.
- C3 is stated per path; the acceptance for it is the doubling-step ratio, with
  the byte-count facts of C1/C2 as the deterministic companion.
- C7 and C8 are the guard rails: the rejection must remain, and
  `TestAcceptsYAMLFlowDocument`, `TestJSONAndYAMLLoadSameModel`, and the
  `determinism_test.go` cases are the tests most likely to catch an over-broad
  early return.
