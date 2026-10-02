---
stage: qa
task: FJ-042
inputFingerprint: b108a8dd28dfd3b53b39add62b2ccc81d1fa0942c1b1cc27400dd98e805ddf55
outputFingerprint: b108a8dd28dfd3b53b39add62b2ccc81d1fa0942c1b1cc27400dd98e805ddf55
taskFingerprint: 0a0388c558ac2868f0d8ee6d2dee106a087a88fa819e8ce3ca1e002bb199851e
gitHead: c2d671e
generatedAt: 2026-10-02T20:08:54Z
author: worker/FJ-042-qa-revision
---

# QA — FJ-042 (release matrix, container image, examples, README)

Independent stage. Author differs from the coder (`worker/FJ-042-coder`). The
repository was not edited; the mandated mutating work (compiled binaries,
temp configs, extracted workflow steps, sandbox runs) happened in the
throwaway copy `/private/tmp/fj042-qa`, populated from the candidate working
tree at `c2d671e` (README modified; `Dockerfile`, `examples/**`,
`.github/workflows/release.yml` untracked — the candidate surface, matching
`cleaner.md`). All four examples were executed against the real binary built
from that copy, never against a mock.

`inputFingerprint` and `outputFingerprint` are the `cleaner.md` front-matter
`outputFingerprint`
(`1c16ef3295b52c175f6cec8a9275faee3ec3d24f9910f36a0d6018be1b2f1ff0`). The
command `./scripts/candidate-fingerprint candidate` printed that exact string
twice in this session.

## Environment

| Item | Observed |
|---|---|
| Host | darwin/arm64 |
| Go | go1.26.3 |
| Node | v24.14.0 |
| Python | 3.11.14 |
| curl | 8.7.1 |
| jq | 1.8.2 |
| shellcheck | 0.11.0 |
| sha256sum | GNU coreutils 9.12 |
| shasum | 6.02 |
| docker client | 29.7.2 (daemon unreachable) |
| container runtimes | podman/nerdctl/buildah/lima/colima/finch/`container`: absent |
| actionlint | absent |
| `git tag` | 0 tags |

## 1. Build the real binary and run every example end to end

```text
cd /private/tmp/fj042-qa
go build -o fake-jev ./cmd/fake-jev            # exit 0
./fake-jev version                             # exit 0
  fake-jev dev
  go: go1.26.3
  build: unknown
  profiles: jev/v1
  control-api: v1
  config-schema: 1
./fake-jev validate examples/fake-jev.yaml     # exit 0
  fake-jev: examples/fake-jev.yaml: configuration is valid
```

Server start on an ephemeral port from the real fixture:

```text
./fake-jev serve --config examples/fake-jev.yaml --port 0 \
  --ready-file /private/tmp/fj042-qa/ready.json
# ready file, written before any example ran:
{"url":"http://127.0.0.1:63296","host":"127.0.0.1","port":63296,"pid":77678,"controlApiVersion":"v1"}
# stdout/stderr:
fake-jev: listening on http://127.0.0.1:63296
```

TypeScript:

```text
$ FAKE_JEV_URL=http://127.0.0.1:63296 node examples/typescript/systemone.ts
fake-jev base URL: http://127.0.0.1:63296
GET /v1/models: {"models":[{"name":"jev-latest","description":"Local deterministic fake model provided by fake-jev.","release_date":"1970-01-01"}]}
POST /v1/systemone: {"model":"jev-latest","answers":{"route":{"type":"choice","choice":"backend","confidence":0.91,"probabilities":{"backend":0.91,"frontend":0.06,"infra":0.03}},"urgent":{"type":"noul","noul":0.94}},"usage":{"input_tokens":0,"output_tokens":0}}
GET /__fake/v1/verify: {"passed":true,"failures":[]}
OK: the fixture answers matched the expected values
exit=0
```

Python:

```text
$ FAKE_JEV_URL=http://127.0.0.1:63296 python3 examples/python/systemone.py
fake-jev base URL: http://127.0.0.1:63296
GET /v1/models: {"models": [{"description": "Local deterministic fake model provided by fake-jev.", "name": "jev-latest", "release_date": "1970-01-01"}]}
POST /v1/systemone: {"answers": {"route": {"choice": "backend", "confidence": 0.91, "probabilities": {"backend": 0.91, "frontend": 0.06, "infra": 0.03}, "type": "choice"}, "urgent": {"noul": 0.94, "type": "noul"}}, "model": "jev-latest", "usage": {"input_tokens": 0, "output_tokens": 0}}
GET /__fake/v1/verify: {"failures": [], "passed": true}
OK: the fixture answers matched the expected values
exit=0
```

Go:

```text
$ FAKE_JEV_URL=http://127.0.0.1:63296 go run ./examples/go
fake-jev base URL: http://127.0.0.1:63296
GET /v1/models: {"models":[{"name":"jev-latest","description":"Local deterministic fake model provided by fake-jev.","release_date":"1970-01-01"}]}
POST /v1/systemone: {"model":"jev-latest","answers":{"route":{"type":"choice","choice":"backend","confidence":0.91,"probabilities":{"backend":0.91,"frontend":0.06,"infra":0.03}},"urgent":{"type":"noul","noul":0.94}},"usage":{"input_tokens":0,"output_tokens":0}}
GET /__fake/v1/verify: {"passed":true,"failures":[]}
OK: the fixture answers matched the expected values
exit=0
```

Raw HTTP:

```text
$ FAKE_JEV_URL=http://127.0.0.1:63296 bash examples/raw-http/systemone.sh
fake-jev base URL: http://127.0.0.1:63296
GET /v1/models: {"models":[{"name":"jev-latest","description":"Local deterministic fake model provided by fake-jev.","release_date":"1970-01-01"}]}
POST /v1/systemone: {"model":"jev-latest","answers":{"route":{"type":"choice","choice":"backend","confidence":0.91,"probabilities":{"backend":0.91,"frontend":0.06,"infra":0.03}},"urgent":{"type":"noul","noul":0.94}},"usage":{"input_tokens":0,"output_tokens":0}}
GET /__fake/v1/verify: {"passed":true,"failures":[]}
OK: the fixture answers matched the expected values
exit=0
```

Wire evidence, one fresh server, one example at a time (server-side log tail
per run; each example left exactly three lines, all to the one loopback
server):

```text
startup log: fake-jev: listening on http://127.0.0.1:64004
=== typescript (exit=0) ===
fake-jev: GET /v1/models
fake-jev: POST /v1/systemone authorization=[redacted] body="<request body present>"
fake-jev: GET /__fake/v1/verify
=== python (exit=0) ===
fake-jev: GET /v1/models authorization=[redacted]
fake-jev: POST /v1/systemone authorization=[redacted] body="<request body present>"
fake-jev: GET /__fake/v1/verify authorization=[redacted]
=== go (exit=0) ===
fake-jev: GET /v1/models authorization=[redacted]
fake-jev: POST /v1/systemone authorization=[redacted] body="<request body present>"
fake-jev: GET /__fake/v1/verify authorization=[redacted]
=== raw-http (exit=0) ===
fake-jev: GET /v1/models
fake-jev: POST /v1/systemone authorization=[redacted] body="<request body present>"
fake-jev: GET /__fake/v1/verify
```

Non-vacuity control. A fixture that answers `frontend` validly
(`choice: frontend`, probabilities `frontend 0.91 / backend 0.06 / infra 0.03`)
made each example print its own mismatch lines and exit non-zero:

```text
node    exit=1  FAIL: route choice should be backend / FAIL: backend probability should be 0.91
python  exit=1  same two lines
go      exit=1  same two lines
bash    exit=1  same two lines
server /__fake/v1/verify -> {"passed":true,"failures":[]}   # stub has no `expect`, so the examples' own checks are what fail
```

A second control with `choice: frontend` left at `0.06` while the maximum is
`backend 0.91` made the server reject the stub
(`fake_jev_invalid_stub_response`, `stubId route-and-urgency`, HTTP 500) — the
§13.2 rule "selected choice MUST have a probability equal to the maximum" is
enforced, and all four examples still exited non-zero.

## 2. No credential, no model, no provider SDK, no egress beyond loopback

Wire evidence:

- The server log records only `authorization=[redacted]`; the literal bearer
  value sent by the examples never appears. A deliberately bogus
  `Authorization: Bearer sk-this-is-not-a-real-key-12345` produced
  `0` matches for that string in the server log, and the POST returned HTTP
  200.
- `POST /v1/systemone` with **no** `Authorization` header at all returned HTTP
  200, so no credential is required (§39.5).
- Control calls with no header: `GET /__fake/v1/verify` → 200,
  `POST /__fake/v1/reset` → 204.
- Per-example server log above: exactly three calls, each to the single
  loopback listener; no other target exists to call.

Egress denied at the OS level. macOS Seatbelt profile `loopback-only.sb`
(`(deny network*)`, then allow `remote ip "localhost:*"` / `local ip
"localhost:*"`) was applied with `sandbox-exec`:

```text
sanity, non-loopback denied:
$ sandbox-exec -f loopback-only.sb curl --max-time 8 https://example.com
curl: (6) Could not resolve host: example.com        exit=6
sanity, loopback allowed:
$ sandbox-exec -f loopback-only.sb curl ... "$BASE/v1/models"
HTTP 200                                             exit=0
```

Under that profile all four examples ran to their `OK:` line with `exit=0`
(TypeScript, Python, a pre-built Go binary, raw HTTP). A **server** started
under the same profile also came up and served a client that made all three
calls (`exit=0`). `lsof -p <server under sandbox>` listed only
`TCP 127.0.0.1:63325 (LISTEN)`, stdin/stdout/stderr, a kqueue and two pipes —
no outbound socket.

Dependencies.

```text
$ grep -RInE '^(import|from|require|use )' examples/...   # TS via fetch (global), Python json/os/sys/urllib.{error,request}
$ find examples \( -name package.json -o -name node_modules -o -name 'requirements*.txt' \
      -o -name vendor -o -name go.mod -o -name '*.lock' \)    # (no output)
$ grep -RInE 'API_KEY|APIKEY|SECRET|ACCESS_TOKEN|api[_-]?key' examples   # (no output)
$ grep -RInE 'https?://' examples | grep -vE '127\.0\.0\.1|localhost|FAKE_JEV_URL|0\.0\.0\.0'   # (no output)
$ go list -m all
fake-jev
gopkg.in/check.v1 v0.0.0-20161208181325-20d25e280405   # indirect, test-only, of yaml.v3
gopkg.in/yaml.v3 v3.0.1
$ git diff --stat c2d671e -- go.mod go.sum             # (empty)
```

No provider SDK is linked or imported; the only module requirement is
`gopkg.in/yaml.v3`.

## 3. Checksums story

The five matrix targets were built with the workflow's own command line, then
the workflow's checksum `run:` block was extracted from `release.yml` with
PyYAML and executed verbatim.

```text
$ for t in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64; do
    CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -trimpath -o dist/fake-jev-$os-$arch$ext ./cmd/fake-jev
  done                                                          # 5/5 exit 0
$ bash checksum-step.sh          # the extracted `write and check SHA256SUMS` block
e1da5b9698d775dd7c74ddcea2c1792b005ec660764bc9d5fffdf44528cd8d97  fake-jev-linux-amd64
d79e0380d10cca31e8be846217df1069e4b615502dd3c23f96003e6c5355b0ed  fake-jev-linux-arm64
a2cfd0f64ab85d5ca0ede713795854ca445a6dd80cc54b4c97a75773f87fb710  fake-jev-darwin-amd64
432a3eb8111eb014f08ae24ecd0fef0313150dcf915b04dd42439a9db8737f5e  fake-jev-darwin-arm64
0f685d33aa89aaf552c09de9fabab33bfea4b8dc2090d042fe9166809590610d  fake-jev-windows-amd64.exe
fake-jev-linux-amd64: OK
fake-jev-linux-arm64: OK
fake-jev-darwin-amd64: OK
fake-jev-darwin-arm64: OK
fake-jev-windows-amd64.exe: OK
exit=0
```

The digests reproduce the ones recorded in `coder.md` exactly. User-side
verification with a standard tool present on this host:

```text
$ awk 'NF!=2 || $2 ~ /[^[:alnum:]._-]/ || length($1)!=64 {bad=1} END {print (bad?"shape-mismatch":"shape-ok")}' SHA256SUMS
shape-ok
$ shasum -a 256 --check SHA256SUMS
... five lines of ": OK", exit=0
$ # tamper: flip one byte in fake-jev-linux-amd64, re-check
sha256sum: WARNING: 1 computed checksum did NOT match
tampered-check exit=1
$ # remove fake-jev-windows-amd64.exe (the `test -f` guard), re-run the step
exit=1
```

Independently computed hash of a locally built artifact (not from the
workflow):

```text
d0da2c716449c7defd04463712750004dff97acacc07a1d18572e5d8f709e8ce  fake-jev   # host darwin-arm64, go build -o fake-jev
```

Workflow naming and attachment, by inspection and YAML parse:

```text
triggers: ['push', 'workflow_dispatch']      # push is tags: ["v*"]; no pull_request
permissions: {'contents': 'write'}
jobs: ['binaries', 'checksums', 'image', 'release']
matrix: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64 (+".exe")
checksums needs: binaries       release needs: ['checksums', 'image']
image platforms: linux/amd64,linux/arm64   image tags: ghcr.io/${{ github.repository }}:${{ env.RELEASE_TAG }}
publish step: gh release create --verify-tag ... dist/fake-jev-* dist/SHA256SUMS  (or upload --clobber on re-run)
```

**No real GitHub release run was observed.** `git tag` lists 0 tags at this
revision and no Actions runner was available in this session; the checksum
file name a user would verify (`SHA256SUMS`, attached to the release body's
`sha256sum --check SHA256SUMS` recipe) is evidenced by inspection plus the
local verbatim execution above, not by a published release.

## 4. Dockerfile against §24.2 (inspection; no container runtime)

```text
$ grep -nE '^(FROM|ARG|WORKDIR|COPY|RUN|USER|EXPOSE|ENTRYPOINT|CMD)' Dockerfile
12:FROM golang:1.26-bookworm AS build
17:ARG TARGETOS=linux        18:ARG TARGETARCH=amd64
20:WORKDIR /src              23:COPY go.mod go.sum ./
24:RUN go mod download      26:COPY . .
31:RUN CGO_ENABLED=0 GOOS="${TARGETOS}" GOARCH="${TARGETARCH}" ... -o /out/fake-jev ./cmd/fake-jev
34:FROM scratch
39:COPY --from=build /etc/ssl/certs/ca-certificates.crt ...
40:COPY --from=build /out/fake-jev /fake-jev
43:USER 65532:65532          46:EXPOSE 8787
48:ENTRYPOINT ["/fake-jev"]  56:CMD ["serve", "--host", "0.0.0.0", "--port", "8787"]
```

- Multi-stage: `FROM` count 2; the final stage's base is `scratch`, not a Go
  image; the builder is referenced only by the two `COPY --from=build` lines.
- Final stage contents: binary plus CA bundle only. No `RUN` after
  `FROM scratch` (so no shell, package manager, or toolchain can be present),
  no `/usr/local/go` or `go/bin` copy. `USER 65532:65532` is numeric, as
  required on a base with no `/etc/passwd`.
- No Go at runtime: the runtime stage never contains a Go image; the built
  linux artifact is `statically linked` (`CGO_ENABLED=0`), so nothing external
  to the binary is needed.
- Container-friendly binding is explicit in `CMD` (`--host 0.0.0.0`), while
  the binary default stays `127.0.0.1` — independently observed:
  `serve --config examples/fake-jev.yaml --port 0 --ready-file ...` wrote
  `{"url":"http://127.0.0.1:...","host":"127.0.0.1",...}`, and
  `serve --host 0.0.0.0 ...` wrote
  `{"url":"http://0.0.0.0:63410","host":"0.0.0.0",...}` with
  `GET /v1/models` reachable over loopback.

Real build attempt:

```text
$ docker build -t fake-jev:local .
ERROR: failed to connect to the docker API at unix:///Users/mak/.docker/run/docker.sock; check if the path is correct and if the daemon is running: dial unix /Users/mak/.docker/run/docker.sock: connect: no such file or directory
exit=1
$ for rt in podman nerdctl buildah lima colima finch container; do command -v $rt; done   # (none)
```

No image was built, exported, or run, and `docker run -p 8787:8787` was not
observed. §24.2 evidence here is structural inspection plus the equivalent
local linux cross-compile and the local bind/REST checks.

## 5. C-ARCH-003 — no §33 deferred feature introduced

The candidate surface is exactly `README.md` (modified) plus the four new
paths (`Dockerfile`, `.github/workflows/release.yml`, `examples/**`); no Go
source, `go.mod`, `ci.yml`, `scripts/**`, `spec/**`, or `testdata/**` changed.

```text
$ grep -RInE 'auto[-_ ]?mode|scenario|plugin|dashboard|persist|latency|fault|record|replay|openapi|wasm|websocket|sqlite|database' \
    examples/ Dockerfile .github/workflows/release.yml
# only substring hits inside the word "default"/"defaultBaseURL" (and the word "record" in a workflow comment); no feature implementation
$ grep -RIohE '/v1/models|/v1/systemone|/__fake/v1/[a-z]+' examples | sort | uniq -c
   9 /__fake/v1/verify
   9 /v1/models
   9 /v1/systemone
```

The examples are thin HTTP clients over the three documented paths; the image
adds no sidecar, metrics port, or plugin loader (final stage is `scratch` with
two files); the workflow runs only build / checksum / image-push / publish
steps. No auto decision mode, scenario DSL, plugin ABI, model-backed
behaviour, persistence, or dashboard appears.

Adjacent observation, not part of the diff: a repository-root `package.json`
exists at `c2d671e` (`{name: fake-jev, version: 0.0.0, description: "Coming
soon."}`; `git cat-file -e HEAD:package.json` succeeds; it is unchanged by
this candidate). It is a placeholder npm manifest predating this item; FJ-042
neither adds nor changes it, and no npm package is published — the "no npm
package / no wrapper" non-goal is respected by the diff.

## 6. README accuracy against the real binary

Every actionable claim was exercised:

| README claim | Observation |
|---|---|
| `go build -o fake-jev ./cmd/fake-jev` | works, exit 0 |
| five artifact names `fake-jev-<os>-<arch>[.exe]` | match the matrix build output |
| `gh release download <tag> && sha256sum --check SHA256SUMS` | command form correct (`SHA256SUMS` shape verified above); **no tag exists — 0 in `git tag`** |
| `docker build -t fake-jev:local .` / `docker run --rm -p 8787:8787 fake-jev:local` | not executable here (no runtime); commands match the Dockerfile `CMD` |
| image default `serve --host 0.0.0.0 --port 8787` | equals the Dockerfile `CMD` |
| `./fake-jev validate examples/fake-jev.yaml` | `configuration is valid`, exit 0; missing file → exit 2 |
| `serve --config … --port 8787 --ready-file …` | works; ready file `{"url","host","port","pid","controlApiVersion"}` as documented; removed on SIGTERM |
| `curl .../v1/models` | returns the model document |
| `./fake-jev verify --url …` | exit 0; against an unreachable URL exit 2; the same command in `run` returns 3 when verification fails |
| CLI list `version serve validate verify run help` | matches the usage text |
| `serve` flags `--host --port --config --compat --mode --log-format --ready-file` | all accepted; `--port 0` ephemeral; `--mode auto` rejected exit 2 ("mode \"auto\" is not supported; only strict is valid"); `--compat jev/v1` accepted, unknown profile exit 2, duplicate profile exit 2 |
| `run` flags `--config --url-env --port --` | all accepted; child sees `FAKE_JEV_URL` / the `--url-env` name |
| exit codes `0 / 2 / 3` | 0 success; 2 usage/config/startup (bad command, missing file, `--mode auto`); 3 verification failed (child `true` with an unmet `expect: exactly 1`); child failure propagates (child `exit 7` → run exit 7) |
| default bind `127.0.0.1`, untrusted-network warning, unauthenticated control API | ready file host `127.0.0.1`; warning at README:123; `verify`/`reset` answer without any header |
| no-model-quality statement (§32) | present verbatim at README:22: `**fake-jev does not simulate model quality or intelligence.**` |
| stale status gone | `grep -c 'No product implementation has begun' README.md` → 0; no "foundation phase" |
| Verify section names the live product checks | `verify-candidate` runs `go test/vet/build`, guard rows, race (see §7) |

Two wordings that name an artifact that does not yet exist as an output, both
pre-existing concerns already recorded by the cleaner and neither asserting a
past event: README:47–51 "download a released binary … `gh release download`"
(0 tags), and README:10 listing "the container image" among present surfaces
(the `Dockerfile` is present; no image was built or published).

One actionable command that does not work verbatim in this repository:
README's illustrative `./fake-jev run --config examples/fake-jev.yaml
--url-env JEV_BASE_URL -- npm test`. Observed with npm 11.9.0:

```text
npm error ... To see a list of scripts, run: npm run
fake-jev: child exited with code 1
run exit=1
```

The fake-jev half behaves as documented (runs the command, propagates the
child's code); the failure is that the repository root `package.json` has no
`test` script, so the copied command cannot succeed here. It is presented as a
form illustration, not a claim that a test suite exists.

Documentation nuance: README lists `serve --log-format text|json`. With
`--log-format json` the observed output was identical to text
(`fake-jev: listening on …`, `fake-jev: GET /v1/models`); `logFormat` is
validated in `internal/cli/serve.go:106` but not consulted by the logger
(`internal/cli/logging.go`). §20 calls structured logging "optional", and
README only lists the flag name; the behavior lives in `internal/`, outside
this item's `allowedFiles`, and is unchanged by FJ-042.

## 7. C-QUAL-005 — golden contracts offline, no credentials/model/network

Golden-vector tests were run with egress denied and `GOPROXY=off` in a
credential-free environment:

```text
$ sandbox-exec -f loopback-only.sb env -i PATH=... HOME=... GOFLAGS=-mod=readonly \
    GOPROXY=off GOWORK=off GOTOOLCHAIN=local go test -count=1 ./internal/... ./test/...
ok  fake-jev/internal/cli              19.049s
ok  fake-jev/internal/compat/jev/v1     0.533s
ok  fake-jev/internal/config            0.403s
ok  fake-jev/internal/control           0.688s
ok  fake-jev/internal/engine            0.800s
ok  fake-jev/internal/host/http         1.065s
ok  fake-jev/test/integration           1.105s
exit=0
```

The named golden/vector tests all reported `--- PASS` under that egress-denied
run: `TestGenerateAnswersGoldenVectors`, `TestEncodeModelsGoldenAndConfiguredOrder`,
`TestValidationGoldenErrors`, `TestControlFullResetVector`,
`TestGoldenMatchingAndOrderingVectors`, `TestSequenceConsumptionAndExhaustionGoldenVector`,
`TestDataPlaneGoldenVectors` (subtests models/noul/choice/score/exact-question-set).
The `testdata/contracts/` files are §44 vectors consumed as trace-coverage
data (counted by `guard trace`, `covered: 17`), not opened at test runtime.

## 8. Mandated command outcomes

| Command | Output | Exit |
|---|---|---|
| `go test ./... -count=1` | Go test: 552 passed in 10 packages | 0 |
| `go test -race ./... -count=1` | Go test: 552 passed in 10 packages | 0 |
| `go vet ./...` | (no output) | 0 |
| `go run ./cmd/guard arch` | `{"check":"arch","findings":[],"stats":{"files":60,"packages":10}}` | 0 |
| `go run ./cmd/guard lint` | `{"check":"lint","findings":[],"stats":{"files":60,"packages":10}}` | 0 |
| `go run ./cmd/guard trace` | `{"check":"trace","findings":[],"stats":{"active":0,"covered":17,"test_files":22}}` | 0 |
| `go run ./cmd/guard fuzz` | `{"check":"fuzz","findings":[],"stats":{"targets":0}}` | 0 |
| `./scripts/verify-candidate FJ-042` | `summary: 14 passed, 0 failed, 0 skipped, 9 not_applicable`; `RESULT: PASS`; repo c2d671e, tree DIRTY (0 staged, 2 unstaged, 9 untracked); report `.agent/reports/FJ-042/report-20261002T195836Z.json` | 0 |
| `./scripts/candidate-fingerprint candidate` | `1c16ef3295b52c175f6cec8a9275faee3ec3d24f9910f36a0d6018be1b2f1ff0` (twice, idempotent) | 0 |

`verify-candidate` classified FJ-042 as `metadata`, so the Go product rows
(`go test`, `go vet`, `go build`, guard arch/lint/trace/fuzz, race) reported
`not_applicable`; the same rows were run by hand above and all exited 0.
Running `verify-candidate` wrote one new untracked report file under
`.agent/reports/FJ-042/`; nothing was staged.

## 9. Fingerprint chain and diff surface

```text
$ git status --porcelain -uall | awk '{print $2}' | grep -vE '^\.agent/'
README.md
.github/workflows/release.yml
Dockerfile
examples/README.md
examples/fake-jev.yaml
examples/go/main.go
examples/python/systemone.py
examples/raw-http/systemone.sh
examples/typescript/systemone.ts
$ git diff --cached --name-only | wc -l      # 0 staged
$ git diff --name-only c2d671e -- .github/workflows/ci.yml   # (empty; ci.yml unchanged)
```

The surface equals `allowedFiles` (`Dockerfile`, `examples/`, the release
workflow, `README.md`); `cleaner.md` carries the same candidate fingerprint
this run reproduces.

## Residual risks

- No container runtime exists in this environment: the image was not built,
  exported, or run; §24.2 and the README container commands rest on
  inspection plus the local linux cross-compile and bind checks.
- No real GitHub release run: `git tag` is empty, no runner was available, and
  `actionlint` is absent. Tag resolution, `gh release create --verify-tag`,
  the GHCR login/tag/digest hand-off, and the cross-job artifact file names
  (`upload-artifact@v4` single file → `download-artifact@v4` `merge-multiple`)
  were parsed and rehearsed but not executed on a runner.
- The base image `golang:1.26-bookworm` is resolved at build time; a retag
  could change the CA bundle or toolchain patch level (not pinned by digest).
- Non-host release binaries were cross-compiled but not executed;
  `windows/arm64` is intentionally absent per §23.1.
- Windows behavior of the examples (raw-HTTP `bash` herestring) is untested;
  the example documents `bash`.
- `--log-format json` produces text-formatted output (pre-existing
  `internal/cli` behavior outside `allowedFiles`); README lists the flag
  without asserting a JSON rendering.
- README's illustrative `-- npm test` command cannot succeed verbatim here
  because the root `package.json` has no `test` script.
- README:10 and README:47–51 name a container image / downloadable release as
  capabilities; neither has been produced yet at this revision.
- `examples/raw-http/systemone.sh` needs `curl` and `jq` from the environment
  (both present here, both named by the example's own guard).
- The `guard fuzz` row reports `targets: 0` and `guard trace` reports
  `active: 0`; fuzz/active-coverage depth is a repository-wide concern outside
  this item's `allowedFiles`.

## Evidence limits

- Container image build/run: not attempted beyond the failed `docker build`
  above (daemon unreachable; no alternative runtime).
- Real release publication: not observed.
- `actionlint` / GitHub Actions schema validation: tool absent; only a PyYAML
  parse and structural dump were performed.
- Executing the five release binaries on their target platforms: only the host
  `darwin/arm64` binary was run.
- Egress denial was enforced with macOS Seatbelt (`sandbox-exec`), not a
  network namespace; it denies non-loopback name resolution and connections as
  shown, but is a host-specific mechanism.

---

# QA revision — README documentation-accuracy correction

Fresh independent QA session (`worker/FJ-042-qa-revision`), distinct from the
coder (`worker/FJ-042-coder`) and from the previous QA pass
(`worker/FJ-042-qa`). The section above is the previous pass and is preserved
unedited; everything below re-verifies the candidate at the *corrected*
fingerprint.

Front matter above now carries the cleaner-revision fingerprint. The
`1c16ef32…` strings quoted throughout the preserved section describe the prior
pass only. In this session
`./scripts/candidate-fingerprint candidate` printed
`b108a8dd28dfd3b53b39add62b2ccc81d1fa0942c1b1cc27400dd98e805ddf55` and that
value was written into both `inputFingerprint` and `outputFingerprint`, matching
`cleaner.md`'s `outputFingerprint`. Printed twice; identical.

Input under test: `README.md` only changed since the prior pass. Candidate copy
of the tree (`.git` and `.agent` excluded) at `/tmp/fj042-rev`; repository
itself not edited.

## R1. Every candidate file except README is byte-identical to the prior revision

The prior-pass working copy `/private/tmp/fj042-qa` (documented in the preserved
section as populated from the candidate tree at `c2d671e`) still exists, so the
non-README surface was compared file-by-file, not from memory.

```text
candidate-file comparison (current tree vs /private/tmp/fj042-qa):
  candidate files: 137
  differing files: ['README.md']            # exactly one
```

```text
sha256 Dockerfile
  current : 3a3503c4eb0741bc29f5b7ec311dacd07b9d730a8ecc8316c090a406851e8928
  earlier : 3a3503c4eb0741bc29f5b7ec311dacd07b9d730a8ecc8316c090a406851e8928
sha256 .github/workflows/release.yml
  current : 9d66c0e743564a676442dcc746d5062e6a6eb7a9b33fca92ef663e66d8ed4354
  earlier : 9d66c0e743564a676442dcc746d5062e6a6eb7a9b33fca92ef663e66d8ed4354
diff -rq earlier/examples examples                              -> no differences
diff -q earlier/go.mod go.mod  /  earlier/go.sum go.sum         -> identical
```

No new dependency: `go.mod`/`go.sum` unchanged and `git diff --stat c2d671e --
go.mod go.sum .github/workflows/ci.yml` is empty; `git diff --cached
--name-only` is empty (nothing staged).

## R2. README delta is exactly the five documented edits

`diff -u /private/tmp/fj042-qa/README.md README.md` yields five hunks and
nothing else, matching `cleaner.md` edits 1–5: (1) status list "the container
image" → "the container build (`Dockerfile`)"; (2)+(3) install comments
"download a released binary" → "when a release exists, download the published
binary" and "the container image" → "the image from the committed `Dockerfile`";
(4) the `--log-format` sentence; (5) the `run` child command `npm test` →
`go run ./examples/go`. No other README line changed, so no other claim was
touched or weakened.

## R3. Actionable README claims against the built binary

Binary built from the candidate copy: `go build -o fake-jev ./cmd/fake-jev`
(exit 0). Host darwin/arm64, go1.26.3, node v24.14.0, python 3.11.14, curl
8.7.1.

Subcommands — the README CLI block lists `version`, `serve`, `validate`,
`verify`, `run` (and the usage text adds `help`). All accepted; the verbatim
usage line printed by `./fake-jev` is:

```text
usage: fake-jev <command> [flags]
commands:
  version  serve [flags]  validate <path>  verify --url <url>  run [flags] -- <command>  help
```

| README claim | Observation |
|---|---|
| `fake-jev version` → version, build, profiles, API versions | exit 0; prints `fake-jev dev` / `go: go1.26.3` / `build: unknown` / `profiles: jev/v1` / `control-api: v1` / `config-schema: 1` |
| `validate <path>` starts no server; `configuration is valid` | exit 0; missing path → exit 2 |
| `serve` flags `--host --port --config --compat --mode strict --log-format text\|json --ready-file` | all accepted |
| `--port 0` ephemeral | ready file gave a fresh high port per run |
| `--compat` repeatable | flags parse; duplicate `--compat jev/v1` rejected exit 2 (`duplicate compatibility profile "jev/v1"`); unknown profile exit 2 |
| `--mode strict` only | `--mode auto` → exit 2 |
| `--log-format` accepts `text`/`json`, rejects others | `xml` and `JSON` → exit 2, `fake-jev: log-format "xml" is not supported; use text or json` |
| `--ready-file` written once listener+control ready, removed on clean shutdown | file `{"url":"http://127.0.0.1:64658","host":"127.0.0.1","port":64658,"pid":5067,"controlApiVersion":"v1"}`; removed after `SIGTERM`; `serve` then exits 0 |
| binary default bind stays `127.0.0.1` | ready-file `host` = `127.0.0.1`; `internal/config/schema.go:7` `DefaultHost = "127.0.0.1"` |
| `curl --fail-with-body .../v1/models` | returns the model document, exit 0 |
| `verify --url <url>` calls the verify operation | loopback → `verification passed`, exit 0; unreachable URL → exit 2 |
| exit codes `0` success, `2` usage/config/startup/operational, `3` verification failed | 0: version/validate/serve-clean/run-success/verify; 2: unknown command, `validate` missing path, `validate` no arg, `--mode auto`, unknown `--compat`, duplicate `--compat`, `--log-format xml`, `verify` unreachable; 3: `run` with an unmet `expect` and a child that exits 0 (`run --config <expect: exactly 99> -- true` → exit 3) |
| child command failure propagates | `run … -- bash -c 'exit 7'` → exit 7 |

## R4. `--log-format` behaviour, exactly as now described

README now states the flag accepts `text` or `json`, rejects any other value,
and that both accepted values currently produce the same text output. Observed:

```text
$ ./fake-jev serve --config examples/fake-jev.yaml --port 0 --log-format text --ready-file …
fake-jev: listening on http://127.0.0.1:64663
fake-jev: GET /v1/models
fake-jev: received terminated; shutting down
$ ./fake-jev serve --config examples/fake-jev.yaml --port 0 --log-format json --ready-file …
fake-jev: listening on http://127.0.0.1:64665
fake-jev: GET /v1/models
fake-jev: received terminated; shutting down
# after normalizing the port, the two logs are identical (diff -q clean)
```

Source: `internal/cli/serve.go:106-107` rejects any value other than `text`/
`json`; `logFormat` appears nowhere else in `internal/` except that declaration,
its flag binding, and tests, so no JSON rendering branch exists. `grep -rn
"logFormat" internal/ --include=*.go` confirms it is validated and then unused.

## R5. The documented `run` example command works verbatim from a clean copy

From the clean copy `/tmp/fj042-rev` (candidate minus `.git`/`.agent`, binary
built there), the README line run exactly as printed:

```text
$ ./fake-jev run --config examples/fake-jev.yaml --url-env JEV_BASE_URL -- go run ./examples/go
fake-jev base URL: http://127.0.0.1:64672
GET /v1/models: {"models":[{"name":"jev-latest",…}]}
POST /v1/systemone: {"model":"jev-latest","answers":{…}}
GET /__fake/v1/verify: {"passed":true,"failures":[]}
OK: the fixture answers matched the expected values
exit=0
```

This is the correction: the prior pass recorded that the printed `-- npm test`
command could not succeed (root `package.json` has no `test` script). The
`go run ./examples/go` form succeeds as printed.

## R6. All four examples still run end to end against one real server

One built binary, one `serve --config examples/fake-jev.yaml --port 0
--ready-file …` listener; each example run against it:

```text
TypeScript  FAKE_JEV_URL=… node examples/typescript/systemone.ts     exit=0  OK
Python      FAKE_JEV_URL=… python3 examples/python/systemone.py      exit=0  OK
Go          FAKE_JEV_URL=… go run ./examples/go                      exit=0  OK
raw HTTP    FAKE_JEV_URL=… bash examples/raw-http/systemone.sh       exit=0  OK
```

Every run printed `OK: the fixture answers matched the expected values` and the
server log recorded the three expected calls (`GET /v1/models`,
`POST /v1/systemone`, `GET /__fake/v1/verify`) with
`authorization=[redacted]`.

## R7. No release or container image implied as existing; no-model statement present

```text
$ grep -niE "releas|publish|container image|download|ghcr|docker|image" README.md
10: the release build matrix, the container build
11: (`Dockerfile`), and the examples are all present. …
47: # Or, when a release exists, download the published binary.
48: # `.github/workflows/release.yml` cross-compiles the platform matrix below on a
52: #   gh release download <tag> && sha256sum --check SHA256SUMS
54: # Or build and run the image from the committed `Dockerfile` (static binary,
56: docker build -t fake-jev:local .
57: docker run --rm -p 8787:8787 fake-jev:local
60: The image's default command is `serve --host 0.0.0.0 --port 8787`: …
65: docker run --rm -p 8787:8787 -v "$PWD/examples/fake-jev.yaml:/etc/fake-jev.yaml" \
$ git tag | wc -l        -> 0
```

- Line 10 names the `Dockerfile` (a committed artifact) and the workflow matrix
  (committed), not a built/published image.
- Lines 47–52 frame the download as available "when a release exists"; `git tag`
  is empty, so no release is claimed.
- Line 54 names the committed `Dockerfile` as the source of the image; `docker
  build`/`docker run` are instructions.
- Line 60 "The image's default command" follows the `docker build` block and
  matches `Dockerfile:56 CMD ["serve", "--host", "0.0.0.0", "--port", "8787"]`.

No-model-quality statement present verbatim:

```text
$ grep -nF "fake-jev does not simulate model quality or intelligence." README.md
22:**fake-jev does not simulate model quality or intelligence.** It does not
```

Status paragraph no longer claims an image: `grep -nc "No product implementation
has begun\|foundation phase" README.md` → 0.

## R8. Re-confirmed security / no-credential claims cited by README

```text
GET /v1/models             no header            -> HTTP 200
GET /__fake/v1/verify      no header            -> HTTP 200
POST /__fake/v1/reset      no header            -> HTTP 204
POST /v1/systemone         no Authorization     -> HTTP 200 (exact fixture body)
POST /v1/systemone         Bearer sk-this-is-not-a-real-key-12345 -> HTTP 200
grep -c "sk-this-is-not-a-real-key-12345" <server log>  -> 0
# server log line: POST /v1/systemone authorization=[redacted] body="…"
```

Credential accepted-and-ignored, never logged; control API unauthenticated;
`serve --host 0.0.0.0` writes `{"host":"0.0.0.0",…}` (explicit opt-in). The only
`os/exec` in `internal/` is `internal/cli/run.go`, which spawns the user's own
child command — no fixture-executed shell/template/callback path.

## R9. Mandated command outcomes (this session)

| Command | Output | Exit |
|---|---|---|
| `go test ./... -count=1` | `Go test: 552 passed in 10 packages` | 0 |
| `go test -race ./... -count=1` | `Go test: 552 passed in 10 packages` | 0 |
| `go vet ./...` | (no output) | 0 |
| `go run ./cmd/guard arch` | `{"check":"arch","findings":[],"stats":{"files":60,"packages":10}}` | 0 |
| `go run ./cmd/guard lint` | `{"check":"lint","findings":[],"stats":{"files":60,"packages":10}}` | 0 |
| `go run ./cmd/guard trace` | `{"check":"trace","findings":[],"stats":{"active":0,"covered":17,"test_files":22}}` | 0 |
| `go run ./cmd/guard fuzz` | `{"check":"fuzz","findings":[],"stats":{"targets":0}}` | 0 |
| `./scripts/verify-candidate FJ-042` | `summary: 14 passed, 0 failed, 0 skipped, 9 not_applicable`; `RESULT: PASS`; repo c2d671e, tree DIRTY (0 staged, 2 unstaged, 11 untracked); report `.agent/reports/FJ-042/report-20261002T200619Z.json` | 0 |
| `./scripts/candidate-fingerprint candidate` | `b108a8dd28dfd3b53b39add62b2ccc81d1fa0942c1b1cc27400dd98e805ddf55` (printed twice) | 0 |

`verify-candidate` wrote one new untracked report under `.agent/reports/FJ-042/`
(excluded from the candidate fingerprint); nothing was staged.

## R10. Revision residual risks and limits

- No container runtime here (docker daemon unreachable; no podman/buildah/etc.):
  `docker build`/`docker run` and README line 60 were checked by Dockerfile
  inspection only, not executed.
- `git tag` is empty and no Actions runner is available: `gh release download
  <tag>` remains an instruction that cannot execute at this revision; the
  "when a release exists" wording is a statement about `release.yml`, not an
  observed release.
- README line 60 ("The image's default command") still uses "the image"
  contextually, immediately after the `docker build` block; it matches the
  Dockerfile `CMD` and does not assert that an image exists on any registry.
- `--compat` is described as repeatable; the flag parses when repeated, but the
  only profile value exists once, so a duplicate is rejected (exit 2). No
  distinct multi-profile use is documented.
- The correction verified here is `README.md` only; the `--log-format` code gap
  itself (json renders as text) is unchanged and still owned by the follow-up
  ticket, now documented rather than hidden.
