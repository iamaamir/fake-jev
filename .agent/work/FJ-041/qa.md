---
stage: qa
task: FJ-041
inputFingerprint: 3d896a0a7d7ae6bd6f347b4166a74c6d23bbf59a384869c2cc3c2697e96e9dcc
outputFingerprint: 3d896a0a7d7ae6bd6f347b4166a74c6d23bbf59a384869c2cc3c2697e96e9dcc
taskFingerprint: a39d771c0c282b258d73a682eeabd71d7ad9022cd4f0ae5a8c87b6911d2cef61
gitHead: e9be073
generatedAt: 2026-10-01T08:00:31Z
author: worker/FJ-041-qa
---

# QA — FJ-041 (operational envelope hardening)

This stage exercises the candidate revision `e9be073` with the FJ-041 change set
present as unstaged/untracked working-tree content. The artifact records what was
observed and the exact commands that produced the observations, per the QA role
pack. It declares no outcome; only `./scripts/verify-candidate` emits a result.

The author of this artifact (`worker/FJ-041-qa`) differs from the coder author
(`worker/FJ-041-coder`).

## Independent surface (not the in-package tests)

- The real binary was built once, outside the repository:

  ```text
  go build -o /private/tmp/fj041-qa/fake-jev ./cmd/fake-jev
  ```

- The probe is a self-contained Python 3 program at
  `/private/tmp/fj041-qa/probe.py` (standard library only, plus the
  environment's `requests` is not used — raw `http.client`, `socket`, and
  `subprocess`). It starts `/private/tmp/fj041-qa/fake-jev` as a subprocess for
  every case, reads the ready file it publishes, and drives the real HTTP
  surface over loopback. It never calls `internal/cli` or any in-package test in
  process.
- Exact probe invocation (all checks) and single-check re-runs:

  ```text
  python3 /private/tmp/fj041-qa/probe.py
  python3 /private/tmp/fj041-qa/probe.py <meta|dp|cp|max|preview|logging|jev017|bind|arch005|arch004|forgery|stdoutchan|grace>
  ```

  Full transcript of the final all-check run: `/private/tmp/fj041-qa/probe-output.txt`
  (each fixture config, ready file, and captured stdout/stderr live under
  `/private/tmp/fj041-qa/run-*`).
- No repository file was added or edited by this stage. `git status --short`
  shows only the pre-existing FJ-041 change set, the earlier stage artifacts,
  this artifact, and the report `./scripts/verify-candidate FJ-041` writes for
  itself. `git diff --cached --name-only` is empty (nothing staged).
- Candidate fingerprint recomputed after all probing:
  `./scripts/candidate-fingerprint candidate` →
  `3d896a0a7d7ae6bd6f347b4166a74c6d23bbf59a384869c2cc3c2697e96e9dcc`, identical
  to the declared input fingerprint; task fingerprint
  `a39d771c0c282b258d73a682eeabd71d7ad9022cd4f0ae5a8c87b6911d2cef61`. Probing
  staled nothing.

## C-HOST-008 — enforced §12.2 limits (§12.2, §19.4)

Server started with no configuration file (`fake-jev serve --port 0`, ready file
observed). `GET /__fake/v1/meta` returned HTTP `200` and:

```json
{"controlPlaneBodyBytes":2097152,"dataPlaneBodyBytes":8388608,"gracefulShutdownSeconds":5,"logBodyBytes":4096,"maxInteractions":10000}
```

All five values are identical to §12.2 / §41.3. This is the identity half; the
enforcement half was observed behaviourally, independently of the in-package
identity test.

### 8 MiB data-plane body limit

- A raw TCP request to `POST /v1/systemone` declaring
  `Content-Length: 8388609` and **sending zero body bytes** received:

  ```text
  HTTP/1.1 413 Request Entity Too Large
  {"error":"fake_jev_payload_too_large","message":"Request body exceeds the configured limit."}
  ```

  The response body is byte-identical to the §43.7 payload-too-large body. The
  response arrived without any body byte being sent, so the declared-length
  check fires before the body is read.
- `GET /__fake/v1/requests` then held exactly one record:
  `{"sequence":1,"outcome":"payload_too_large","status":413,"error":"fake_jev_payload_too_large","requestBody":null,"matchedStubId":null}`
  — the §43.7 journal shape (normal sequence, `payload_too_large`,
  `requestBody: null`, no stub selected).
- The same server carried one matching `systemone` stub; after the oversize
  request `GET /__fake/v1/stubs` reported `invocations: 0`, so the oversize
  request mutated no stub.
- A body of **exactly** 8388608 bytes (`Content-Length: 8388608`, real body)
  returned `422` (validation), not `413`: a body at the limit is admitted.

### 2 MiB control-plane body limit

- A raw `POST /__fake/v1/stubs` declaring `Content-Length: 2097153` and sending
  zero body bytes received `413` with the same JSON body above. `GET
  /__fake/v1/requests` stayed empty (`0` records) — control traffic is not
  journaled (§41.1). The response arrived without a body byte being sent.
- A body of **exactly** 2097152 bytes reached the control API (`400`
  `{"error":"fake_jev_bad_control_request","message":"Invalid control request."}`),
  not `413`.

Observation on the §41 half: §41 defines no distinct control-plane `413` body.
§41.1 specifies the `400` bad-control-request envelope and the control-plane
`payload_too_large` verification-failure label (§41.9); the observed
control-plane `413` body is the same `{error,message}` shape as §43.7. The `400`
row above is the exact §41.1 body.

### 10,000 interaction journal

Server configured with the §12.2 default `maxInteractions` (`10000`) and one
matching `systemone` stub. Exactly 10,000 admitted `POST /v1/systemone`
requests each returned `200` (last status `200`); `GET /__fake/v1/requests`
then held `10000` records with sequences `1..10000`. The 10,001st request
returned:

```text
status=507 {"error":"fake_jev_journal_full","message":"Interaction journal limit reached. Reset or increase maxInteractions."}
```

The body is byte-identical to §43.5. After the `507`, `GET /__fake/v1/stubs`
still reported `invocations: 10000` — the 10,001st request did not mutate the
stub — and the journal length stayed `10000` (no eviction).

### `logBodyBytes` normal-log body preview

Default server (`logBodyBytes: 4096`), one request per row, stderr captured:

| Request body | Observed |
| --- | --- |
| exactly 4096 bytes (`ATLIMIT-` prefix) | log line has the body preview, no `...[truncated]` marker |
| 4097 bytes (`OVERLIMIT-` prefix) | log line has the body preview and the `...[truncated]` marker |
| ~10 000 bytes with `HEAD` at the start and `TAILEND` at the end | `HEAD` present in the line, `TAILEND` absent |

Second server with an explicit configured `logBodyBytes: 64`:

| Request body | Observed |
| --- | --- |
| 207 bytes (`CFGHEAD` … `CFGDEEP`) | `CFGHEAD` present, `CFGDEEP` absent, marker present |
| exactly 64 bytes (`CFGAT-` prefix) | body preview present, no marker |

The preview follows the configured value: the 64-byte server drops content the
4096-byte server keeps, so the value is read from configuration rather than
hard-coded to the default.

### `gracefulShutdownSeconds`

Server configured with the non-default `gracefulShutdownSeconds: 8`. A raw
connection opened an in-flight `POST /v1/systemone` with
`Content-Length: 65536` and `Expect: 100-continue`; the interim `HTTP/1.1 100
Continue` was read (the host was actively reading the body), then `SIGTERM` was
delivered.

- still alive at 6.0 s after `SIGTERM` (the §12.2 default of 5 s would have
  exited);
- exited `0` after `8.0 s` total drain;
- the held connection was closed after exit.

The default value of `5` itself was observed only via `/meta` (identity); its
enforcement is exercised through the configured-value run above plus the
unmodified in-package `TestServeForceCloseAfterGracefulTimeout`. A direct
wall-clock bound on the 5-second default was not run (the specifier recorded
this choice).

## C-CLI-010 / §20 / §42.5 — logging destinations, redaction, bounded previews

One server received four requests carrying `Authorization: Bearer
FAKEJEV-TOKEN-SENTINEL-0123456789abc`: an accepted `GET /v1/models` (`200`), an
unmatched `POST /v1/systemone` (`501`), an invalid-JSON `POST /v1/systemone`
(`422`), and a raw declared-oversize `POST /v1/systemone` (`413`).

- `serve` stdout was `0` bytes (`''`) — no operational log and no result
  document on stdout (§15.1); the ready file is the result channel.
- The sentinel appeared in neither stdout nor stderr.
- The header's presence was rendered as `authorization=[redacted]` (4 lines
  carried an `authorization=` field, 4 carried `[redacted]`); the value itself
  never appeared. All four request paths (accepted, rejected, error, oversized)
  were redacted.
- The body preview appeared on the same per-request line; a request body whose
  bytes include `\n`, `\r`, NUL, and TAB was rendered as escaped text (see
  "Log forgery" below).

Result data on stdout is still permitted and used: `fake-jev version` exited `0`
with profile/config lines on stdout and empty stderr.

## C-JEV-017 / §10.1 / §39.5 — identical auth acceptance, value never logged

One server with a literal `state: "matched"` stub. For each header case —
absent, `Authorization: Bearer fake`, `Authorization: Bearer
FAKEJEV-TOKEN-SENTINEL-0123456789abc` — the three exchanges
(`GET /v1/models` → `200`, matched `POST /v1/systemone` → `200`, unmatched
`POST /v1/systemone` → `501`) returned status **and body** byte-identical to the
no-header baseline (the probe compared the full `(status, body)` tuples).

The sentinel appeared in neither stdout, stderr, nor the marshalled
`GET /__fake/v1/requests` document. All nine data-plane requests were journaled
(`journal-length: 9`), consistent with §39.1/§41.7 exposing no request headers.

## C-ARCH-006 / §19.1 / §37 — default bind is 127.0.0.1

Server started with no `--host` and no config file. The actual OS listener was
observed independently of the ready file and of the config default:

```text
lsof -nP -a -p <pid> -iTCP -sTCP:LISTEN
  fake-jev <pid> mak 3u IPv4 ... TCP 127.0.0.1:57695 (LISTEN)
```

- a data-plane `GET /v1/models` over `127.0.0.1:<port>` returned `200`;
- the ready file reported `host=127.0.0.1`, `url=http://127.0.0.1:57695`;
- the host has a non-loopback IPv4 interface (`192.168.1.46`); a TCP connect to
  `192.168.1.46:<port>` was refused (`ConnectionRefusedError`), so the listener
  is not reachable off loopback. Had no non-loopback interface existed, this
  sub-check would have been recorded as not exercised, per the repository
  convention.
- `--host 0.0.0.0` was not asserted against (explicit public binds remain legal
  per §19.1).

## C-ARCH-004 / §19.2 — no network egress (structural)

This is a **structural** observation, not an egress-blocked runtime proof; the
specifier recorded the same honest limit.

- `go run ./cmd/guard arch` exited `0` with
  `{"check":"arch","findings":[],"stats":{"files":59,"packages":9}}`.
- Import closures (`go list -deps`) of `internal/engine`, `internal/config`, and
  `internal/compat/jev/v1` contain none of `net`, `net/http`, `net/url`,
  `os/exec`, `text/template`, `html/template`, `plugin`; their direct import
  lists (`go list -f '{{join .Imports "\n"}}'`) contain none of them either.
- `go list -m all` = `fake-jev`, `gopkg.in/check.v1`,
  `gopkg.in/yaml.v3 v3.0.1` — no model, provider, HTTP-client, or cloud SDK in
  the module graph, so "no model or provider call" holds at the dependency
  level.
- The guard's `arch.forbidden` rules match direct imports of module-relative
  package directories (`internal/engine`, `internal/compat/jev/v1`,
  `internal/config`); they do not cover `internal/host/http`, which legitimately
  imports `net` and `net/http` because it is the server, nor `internal/cli`,
  which needs `os/exec` for `run` (§15.4). Those packages' no-egress property is
  therefore not automatically enforced by the guard; it rests on the source
  audit recorded by the specifier/cleaner (only `net.Listen`,
  `net.JoinHostPort`, `net/http` server/handler types, and header plumbing; no
  `http.Client`, `net.Dial`, `net.Dialer`, or `os/exec` outside `run`).
- No behavioural "egress is blocked" assertion is available in this repository
  deterministically; the suite runs offline but nothing fails if a socket
  dial were added to a covered package at runtime — it would instead make the
  arch guard report a `forbidden_import` finding.

## C-ARCH-005 / §12.8 / §19.3 — fixtures are data

All cases driven through the real binary.

- `then.raw.body` carrying `{{ .Env.HOME }}`, `{{ 7*7 }}`,
  `${process.env.SECRET}`, `$(id)`, and `1+1` was returned verbatim
  (`content-type: application/json`, HTTP `200`): `tmpl` came back as the literal
  string `{{ 7*7 }}`, not `49`; `49` and the child environment value
  (`FJQA_SECRET=SECRET-ENV-VALUE-4242`) appeared in neither the response body
  nor the logs.
- A `when.state` of the literal string `"{{ .Env.HOME }}"` matched a request
  whose `state` is exactly that string (HTTP `200`) and did **not** match a
  request whose `state` is a rendered-looking value (`/Users/someone` → `501`),
  i.e. matching is exact JSON equality, not evaluation.
- An expression supplied where a number is required (`then.answers.q.noul:
  "{{ 7*7 }}"`) was accepted as data by `fake-jev validate <path>` (exit `0`,
  `configuration is valid`) and then refused at response generation with HTTP
  `500`:

  ```json
  {"error":"fake_jev_invalid_stub_response","message":"Configured stub cannot produce a valid response for this request.","stubId":"bad"}
  ```

  `"noul": 49` never appeared in the response — the string was neither
  evaluated nor repaired. (`validate`'s exit was `0`, so rejection is at the
  response seam, matching the coder's correction to the specifier's expectation.)

## Hardener wire-regression re-verification (independent)

The `logHandler` body-preview observer does not read ahead: the declared-oversize
data-plane request above returned the exact §43.7 `413` body with **zero** body
bytes sent, the journal recorded one `payload_too_large` record with
`requestBody: null`, the configured stub's invocation count stayed `0`, and a
body exactly at the plane limit was admitted (`422`, not `413`). No stub state
or journal outcome changed in the oversize path.

## Log forgery (independent)

A raw request body containing real `0x0A`, `0x0D`, `0x00`, and `0x09` bytes was
served. In stderr:

- no line began with the forged `fake-jev: FORGED-LINE` prefix (count `0`);
- the body preview line rendered the control bytes as Go escapes
  (`body="ok\nfake-jev: FORGED-LINE-1\r...\x00\tend"`), so a raw newline byte was
  not present in the stream (`raw-newline-byte-in-stderr: absent`, likewise CR
  and NUL);
- the total `fake-jev: ` line count was `3` (listen, request, shutdown), i.e. the
  forged body added no line.

## Exact commands and outcomes

| Command | Observed outcome |
| --- | --- |
| `go build -o /private/tmp/fj041-qa/fake-jev ./cmd/fake-jev` | exit `0` |
| `python3 /private/tmp/fj041-qa/probe.py` | all `OBS` rows as above; transcript `/private/tmp/fj041-qa/probe-output.txt` |
| `go test ./internal/cli -count=1` | `ok  fake-jev/internal/cli  20.718s`, exit `0` |
| `go test ./internal/host/http -count=1` | `ok  fake-jev/internal/host/http  0.817s`, exit `0` |
| `go test ./... -count=1` | all 8 test packages `ok`, exit `0` |
| `go test -race ./... -count=1` | all 8 test packages `ok`, exit `0` |
| `go vet ./...` | no output, exit `0` |
| `go run ./cmd/guard arch` | `findings: []`, files 59, packages 9, exit `0` |
| `go run ./cmd/guard lint` | `findings: []`, files 59, packages 9, exit `0` |
| `go run ./cmd/guard trace` | `findings: []`, active 0, covered 17, test_files 22, exit `0` |
| `go run ./cmd/guard fuzz` | `findings: []`, targets 0, exit `0` |
| `./scripts/verify-candidate FJ-041` | exit `0`; tool summary `20 passed, 0 failed, 0 skipped, 0 not_applicable`; report `.agent/reports/FJ-041/report-20261001T075935Z.json` |
| `./scripts/candidate-fingerprint candidate` | `3d896a0a7d7ae6bd6f347b4166a74c6d23bbf59a384869c2cc3c2697e96e9dcc` |
| `git diff --cached --name-only` | empty (nothing staged) |

`go test ./... -count=1` completed with no `cmd/guard TestRunFuzzPass` flake in
this session (guard `8.570s`); the flake was not reproduced here.

## Traces

C-HOST-008, C-CLI-010, C-ARCH-004, C-ARCH-005, C-ARCH-006, C-JEV-017.

## Residual risks and evidence limits

- **C-ARCH-004 is structural only.** The guard proves forbidden imports are
  absent; it is not an egress-blocked runtime observation, and admits no
  behavioural "no dial happened" assertion. `internal/host/http` legitimately
  imports `net` and `net/http` by design, so its no-egress property is
  audit-based (source read, not enforced), as is `internal/cli`'s `os/exec` for
  `run`. A future socket call inside `internal/host/http` would not fail the
  arch guard.
- **`cmd/guard TestRunFuzzPass` flake is owned by a separate ticket.** It is
  load/resource-triggered (`ulimit -n 64` reproduces it) and lives in
  `cmd/guard` (`runOneFuzz` classifies a fuzzing-harness failure as a
  `guard.fuzz.crash` finding); it was not reproduced in this session and
  `cmd/guard` was not modified by FJ-041. Recorded, not re-opened.
- **Control-plane `413` body is not separately specified in §41.** The observed
  body equals §43.7's; if a distinct §41 body was intended, the spec is silent
  and that would be a spec question, not an implementation divergence.
- **`logBodyBytes: 0` remains spec-undefined.** `/__fake/v1/meta` reports the
  configured value while the host substitutes `4096` (`applyLimitDefaults`);
  the probe did not exercise `0`, and no criterion asserts it.
- **`--log-format json` remains a validated no-op** (the emitted record is text
  for both accepted values); §20 calls structured logging optional and no
  criterion here requires a JSON schema.
- **Preview content is payload.** A secret placed in the request body is still
  echoed (bounded, escaped, marked) to stderr; §42.5's hard rule covers the
  `Authorization` header value, which is redacted.
- **Method/path are logged verbatim** (pre-existing): a percent-decoded path
  containing a newline could in principle forge a line; the body preview is
  escaped, the path is not. This was not changed and is not part of the
  enumerated preview fix.
- **Graceful-shutdown default (5 s) enforcement** was exercised only through the
  configured non-default value plus the unmodified in-package test; no direct
  wall-clock bound on the default itself was run (recorded by the specifier as
  deliberate).
- **`go test ./... -count=1` in this session happened to run clean.** The guard
  fuzz flake is environment-triggered, so absence here is not a guarantee for
  other machines.
