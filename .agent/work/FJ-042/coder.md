---
stage: coder
task: FJ-042
inputFingerprint: 325768fd9665e78ee4264d86eb09e1dc651a4918c6ad1e140276a5b89208de31
outputFingerprint: 1c16ef3295b52c175f6cec8a9275faee3ec3d24f9910f36a0d6018be1b2f1ff0
taskFingerprint: 0a0388c558ac2868f0d8ee6d2dee106a087a88fa819e8ce3ca1e002bb199851e
gitHead: c2d671e
generatedAt: 2026-10-02T19:45:43Z
author: worker/FJ-042-coder
---

# Coder — FJ-042 (release matrix, container image, examples, README)

**Final candidate fingerprint: `1c16ef3295b52c175f6cec8a9275faee3ec3d24f9910f36a0d6018be1b2f1ff0`**

`./scripts/candidate-fingerprint candidate` printed the same digest twice (two
runs, idempotent). It was taken after every edit below, with no staged file and
no file outside `allowedFiles` touched by this stage.

## Files written (all inside `allowedFiles`)

| File | Lines | What it is |
|---|---|---|
| `Dockerfile` | 56 | two-stage: `golang:1.26-bookworm` builder, `scratch` runtime |
| `.github/workflows/release.yml` | 185 | tag-triggered binaries + checksums + image + release |
| `examples/fake-jev.yaml` | 34 | shared `jev/v1` fixture for the four examples |
| `examples/README.md` | 86 | how to start the server and run each example |
| `examples/typescript/systemone.ts` | 145 | TS example, Node built-in fetch, no npm |
| `examples/python/systemone.py` | 111 | Python example, stdlib only |
| `examples/go/main.go` | 238 | Go example, stdlib only |
| `examples/raw-http/systemone.sh` | 97 | curl + jq example |
| `README.md` | 137 | rewritten status/install/usage/limits |

Nothing else was modified by this stage except the verifier's own report files
under `.agent/reports/FJ-042/` (written by `verify-candidate`, not by hand).
`.agent/work/FJ-042/state.json` was already modified before this stage started
and was not touched here.

### Go example shape (C3 choice)

Shape **(a): root module**. `examples/go/main.go` is `package main` with no
`go.mod` of its own, importing only `bytes`, `encoding/json`, `fmt`, `io`,
`net/http`, `os`, `time`. Evidence: `go list ./...` lists
`fake-jev/examples/go`, and the mandatory rows compile and vet it —
`go test ./... -count=1` prints `? fake-jev/examples/go [no test files]` and
`go vet ./examples/...` exits 0. No `require` was added to the root `go.mod`
(unchanged: `go 1.26`, one direct requirement `gopkg.in/yaml.v3`).

## A. Build matrix and per-artifact checksums (§23.1, §23.2, §24.1)

`release.yml` has `binaries` (matrix of exactly `linux/amd64`, `linux/arm64`,
`darwin/amd64`, `darwin/arm64`, `windows/amd64` — no `windows/arm64`),
`checksums` (`needs: binaries`), `image`, and `release`
(`needs: [checksums, image]`).

Local rehearsal of the exact workflow commands (the workflow's build line and
its checksum step extracted from the YAML and executed verbatim):

```text
build {'goos': 'linux',   'goarch': 'amd64', 'ext': ''}      exit 0
build {'goos': 'linux',   'goarch': 'arm64', 'ext': ''}      exit 0
build {'goos': 'darwin',  'goarch': 'amd64', 'ext': ''}      exit 0
build {'goos': 'darwin',  'goarch': 'arm64', 'ext': ''}      exit 0
build {'goos': 'windows', 'goarch': 'amd64', 'ext': '.exe'}  exit 0
```

producing `fake-jev-linux-amd64`, `fake-jev-linux-arm64`,
`fake-jev-darwin-amd64`, `fake-jev-darwin-arm64`,
`fake-jev-windows-amd64.exe`. The checksum step, run unmodified
(`bash <extracted run: block>`) printed five lines and then five `OK` lines:

```text
e1da5b9698d775dd7c74ddcea2c1792b005ec660764bc9d5fffdf44528cd8d97  fake-jev-linux-amd64
d79e0380d10cca31e8be846217df1069e4b615502dd3c23f96003e6c5355b0ed  fake-jev-linux-arm64
a2cfd0f64ab85d5ca0ede713795854ca445a6dd80cc54b4c97a75773f87fb710  fake-jev-darwin-amd64
432a3eb8111eb014f08ae24ecd0fef0313150dcf915b04dd42439a9db8737f5e  fake-jev-darwin-arm64
0f685d33aa89aaf552c09de9fabab33bfea4b8dc2090d042fe9166809590610d  fake-jev-windows-amd64.exe
fake-jev-linux-amd64: OK ... (all five OK)
```

Line-shape assertion over that file: `awk 'NF!=2 || $2 ~ /[^[:alnum:]._-]/ ||
length($1)!=64 {bad=1} END {print (bad?"shape-mismatch":"shape-ok")}'` printed
`shape-ok`. Fail-closed check: after `rm dist/fake-jev-windows-amd64.exe` the
same step exits `1` (the `test -f` guard), so a missing leg cannot publish a
short checksum file.

## B. Container image (§24.2, §19.1, §30.8)

`Dockerfile` stage structure (grep of the leading directives):

```text
12:FROM golang:1.26-bookworm AS build
17:ARG TARGETOS=linux
18:ARG TARGETARCH=amd64
20:WORKDIR /src
23:COPY go.mod go.sum ./
24:RUN go mod download
26:COPY . .
31:RUN CGO_ENABLED=0 GOOS="${TARGETOS}" GOARCH="${TARGETARCH}" \
34:FROM scratch
39:COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
40:COPY --from=build /out/fake-jev /fake-jev
43:USER 65532:65532
46:EXPOSE 8787
48:ENTRYPOINT ["/fake-jev"]
56:CMD ["serve", "--host", "0.0.0.0", "--port", "8787"]
```

- The runtime stage is `scratch`, not a Go image; the builder is not the
  runtime base, and the runtime stage copies exactly two files (binary + CA
  bundle). No shell, no package manager, no Go tree.
- Toolchain pin agrees with the module: `FROM golang:1.26-bookworm` and
  `go 1.26` in `go.mod`.
- The builder's CA path is real, not assumed. `golang:1.26-bookworm` exists
  (`https://hub.docker.com/v2/repositories/library/golang/tags/1.26-bookworm`
  → HTTP 200) and its base chain
  (`buildpack-deps:bookworm-scm` → `buildpack-deps:bookworm-curl` →
  `debian:bookworm`) installs `ca-certificates` in
  `debian/bookworm/curl/Dockerfile`, so `/etc/ssl/certs/ca-certificates.crt`
  exists in the builder filesystem.
- The Dockerfile's own build line was executed locally on both linux and
  darwin targets:

```text
CGO_ENABLED=0 GOOS=linux  GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o .../linux-amd64 ./cmd/fake-jev
  -> ELF 64-bit LSB executable, x86-64, statically linked, stripped
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o .../darwin-arm64 ./cmd/fake-jev
  -> prompt: `version` exit 0, `--help` exit 0
```

  So `-s -w -trimpath` still yields a runnable binary, and the linux artifact
  the image copies is statically linked (no dynamic loader).
- Container-friendly binding: the image's default command is an explicit
  `--host 0.0.0.0`, and the same flag combination was exercised locally:
  `serve --config examples/fake-jev.yaml --host 0.0.0.0 --port 8787
  --ready-file ...` wrote `{"url":"http://0.0.0.0:8787","host":"0.0.0.0",
  "port":8787,...}` and `curl http://127.0.0.1:8787/v1/models` returned the
  model list; `SIGTERM` then removed the ready file. The binary's own default
  bind is untouched (`127.0.0.1` in `internal/config`), i.e. no code or
  default changed.

**The image itself was not built or run here.** `docker build -t
fake-jev:local .` exits `1` with:

```text
ERROR: failed to connect to the docker API at unix:///Users/mak/.docker/run/docker.sock;
check if the path is correct and if the daemon is running: dial unix
/Users/mak/.docker/run/docker.sock: connect: no such file or directory
```

No verified image is claimed; the evidence above is inspection plus the
equivalent local compile and the local bind/REST checks.

## C. Examples (§31 Phase 4)

Shared fixture `examples/fake-jev.yaml`: `schemaVersion: 1`, `mode: strict`,
`compatibility: [jev/v1]`, one stub `route-and-urgency` (`operation:
systemone`, questions `route: choice`, `urgent: noul`) answering
`route = backend / confidence 0.91 / {frontend 0.06, backend 0.91, infra 0.03}`
and `urgent = noul 0.94`. No `expect` block, so four runs against one server
instance all match (documented in the fixture comment).

`./fake-jev validate examples/fake-jev.yaml` → `configuration is valid`, exit 0.

Every example makes exactly three calls — `GET /v1/models`, `POST
/v1/systemone`, `GET /__fake/v1/verify`. `grep -RIohE
'/v1/models|/v1/systemone|/__fake/v1/[a-z]+' examples | sort -u` prints only
those three paths.

### Executed locally, against a real server (documented command form)

Server started by the CLI itself, so `FAKE_JEV_URL` came from the server:
`/tmp/fj-fj042/fake-jev run --config examples/fake-jev.yaml -- <example>`.

```text
node examples/typescript/systemone.ts   -> exit 0
python3 examples/python/systemone.py    -> exit 0
go run ./examples/go                    -> exit 0
bash examples/raw-http/systemone.sh     -> exit 0
```

Printed by the TypeScript run (Node v24.14.0, no npm install, no
`node_modules`, no `package.json` involvement, type annotations stripped by
Node):

```text
fake-jev base URL: http://127.0.0.1:62056
GET /v1/models: {"models":[{"name":"jev-latest","description":"Local deterministic fake model provided by fake-jev.","release_date":"1970-01-01"}]}
POST /v1/systemone: {"model":"jev-latest","answers":{"route":{"type":"choice","choice":"backend","confidence":0.91,"probabilities":{"backend":0.91,"frontend":0.06,"infra":0.03}},"urgent":{"type":"noul","noul":0.94}},"usage":{"input_tokens":0,"output_tokens":0}}
GET /__fake/v1/verify: {"passed":true,"failures":[]}
OK: the fixture answers matched the expected values
```

The Python run printed the same three JSON documents (re-serialized with sorted
keys by that example's `json.dumps`), the Go run printed the same three byte
for byte, and the raw-HTTP run printed the same with `cURL`-fetched bodies. All
four ended with `OK: the fixture answers matched the expected values`.

### Negative control (the checks are not vacuous)

With a fixture whose choice is `frontend` (`/tmp/fj-fj042/mismatch.yaml`,
validated first) each example exits non-zero:

```text
python  exit 1
node    exit 1
go      exit 1
bash    exit 1
```

Server-side verification still reported `passed: true` in that run (the stub
carries no `expect`), which is the intended fixture behaviour: the examples'
own assertions, not the server's verdict, are what fail.

### Offline / no-dependency inspection

```text
find examples -name package.json -o -name node_modules -o -name 'requirements*.txt' -o -name vendor -o -name go.mod   -> no output
grep -RInE 'https?://' examples | grep -v 127.0.0.1 | grep -v FAKE_JEV_URL                                            -> no output
grep -RInE 'API_KEY|APIKEY|SECRET|ACCESS_TOKEN|api[_-]?key' examples                                                  -> no output
grep -RIohE '[A-Za-z_-]*(scenario|plugin|dashboard|persist|latency|fault|record|replay|openapi)[A-Za-z_-]*' examples | sort | uniq -c
   9 default
   2 defaultBaseURL
```

i.e. no dependency artifact, no non-loopback URL, no credential read, and the
only "deferred-feature" shape matches are the word `default` (the `fault`
substring). `Authorization: Bearer not-a-real-key` appears in each example
purely to show §39.5 acceptance; the server log line for the POST reads
`authorization=[redacted]`.

## D. Release workflow (§23.1, §23.2, §24.1, §24.2)

- Triggers are `push: tags: ["v*"]` and `workflow_dispatch` (required `tag`
  input) only — no `pull_request`, no `push` to `main`, so nothing release
  related runs on an ordinary push.
- `git diff --name-only c2d671e -- .github/workflows/ci.yml` produces no
  output: `ci.yml` is byte-identical to the base revision.
- `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/release.yml'))"`
  parses; triggers `['push', 'workflow_dispatch']`, jobs `['binaries',
  'checksums', 'image', 'release']`, top-level `permissions: {contents:
  write}`, workflow env
  `RELEASE_TAG: ${{ github.event.inputs.tag || github.ref_name }}`.
- Artifact naming is explicit in both the build step
  (`dist/fake-jev-${GOOS}-${GOARCH}${EXT}`, `.exe` only on the Windows leg) and
  in the checksum step's artifact list, so file name, artifact name, and
  `SHA256SUMS` entry are the same string a user gets from
  `gh release download`.
- The publish step uses `gh release create --verify-tag` (aborts if the tag
  does not exist, instead of creating one from the default branch) or
  `gh release upload --clobber` on a re-run, and attaches
  `dist/fake-jev-*` plus `dist/SHA256SUMS`. The release body carries the
  download-and-`sha256sum --check` instructions and the GHCR image reference
  plus the digest captured from the image job
  (`needs.image.outputs.digest`).
- Deliberate difference from `ci.yml`'s compile-proof job: this workflow does
  not pin `GOPROXY=off`, so a release with a cold module cache can still
  resolve `gopkg.in/yaml.v3` from the proxy. It is commented in the file.

Not executable here: the GitHub-side run (tag push → release creation, asset
upload, trigger behaviour), the GHCR push, and `actionlint` (not installed).
Evidence is the YAML parse, the structural inspection above, and the local
rehearsal of the build and checksum commands.

## E. README (§32, §19.5)

- Stale status gone: `grep -c 'No product implementation has begun' README.md`
  → `0`; `grep -ni 'foundation phase' README.md` → no output. The replacement
  states that v1 is implemented here and lists the surfaces that exist.
- §32 statement present verbatim at line 22, recorded so it can be grepped
  with `grep -nF 'fake-jev does not simulate model quality or intelligence.'
  README.md`:

  > **fake-jev does not simulate model quality or intelligence.**

  followed by the denial of evaluation/ranking/emulation and of being a
  replacement model or a model-quality evaluation framework.
- No-egress statement at line 27: "Normal execution also performs no network
  egress and no model or provider call", plus "No API key is required —
  authentication headers are accepted and ignored".
- §19.5 warning at line 123: "**Do not expose the server — data plane or
  control API — to untrusted networks**", with the note that a non-loopback
  bind is an explicit choice.
- Installation and usage are the real ones: `go build -o fake-jev
  ./cmd/fake-jev`; the five commands and their flags (`serve --host --port
  --config --compat --mode --log-format --ready-file`, `validate <path>`,
  `verify --url <url>`, `run --config --url-env --port -- <command>`); the
  §15.1 ready-file JSON; exit codes 0/2/3; the release artifact names and
  `sha256sum --check SHA256SUMS`; the container commands; and a pointer to
  `examples/README.md`. The three flag claims that could drift were executed:
  ready-file contents, ready-file removal on SIGTERM, and `--host 0.0.0.0`.

## G. No §33 feature, no new public surface

The diff is confined to `Dockerfile`, `examples/**`,
`.github/workflows/release.yml`, and `README.md` (`git status --porcelain
--untracked-files=all` shows exactly those plus the pre-existing
`.agent/work/FJ-042/state.json` and `specifier.md` and the verifier's reports).
No Go source, `go.mod`, `ci.yml`, `scripts/**`, `spec/**`, or `testdata/**`
changed. No auto mode, scenario DSL, plugin ABI, persistence, dashboard,
control-plane auth, latency/fault injection, record/replay, OpenAPI ingestion,
provider-quality evaluation, probabilistic generation, generic HTTP mocking,
npm/Homebrew package, GitHub Action, TypeScript wrapper package, or WASM build
was added. The examples call only the three documented HTTP paths.

## Exact commands and outcomes

| Command | Outcome |
|---|---|
| `go build ./cmd/fake-jev` | exit 0 |
| `go test ./... -count=1` | exit 0, 552 tests in 10 packages |
| `go vet ./...` | exit 0 |
| `go run ./cmd/guard arch` | exit 0, `{"files":60,"packages":10}`, no findings |
| `go run ./cmd/guard lint` | exit 0, no findings (after fixing 5 `err_discarded` findings in `examples/go/main.go`) |
| `go run ./cmd/guard trace` | exit 0, `{"active":0,"covered":17,"test_files":22}` |
| `go run ./cmd/guard fuzz` | exit 0 |
| `gofmt -l examples/` | no output |
| `bash -n examples/raw-http/systemone.sh` | exit 0 |
| `shellcheck examples/raw-http/systemone.sh` | exit 0 |
| `python3 -c "import yaml; yaml.safe_load(...release.yml)"` | parses |
| `./fake-jev validate examples/fake-jev.yaml` | `configuration is valid`, exit 0 |
| four examples via `fake-jev run` | all exit 0 |
| four examples against a mismatching fixture | all exit 1 |
| matrix rehearsal (5 targets, workflow command lines) | 5/5 exit 0 |
| checksum step (extracted from the workflow, run verbatim) | exit 0, 5 lines, `sha256sum --check` all OK, `shape-ok` |
| checksum step with one artifact removed | exit 1 |
| `docker build -t fake-jev:local .` | exit 1, daemon unreachable (see B) |
| `./scripts/verify-candidate` (task-less) | RESULT: PASS, 20 passed / 0 failed |
| `./scripts/verify-candidate FJ-042` | RESULT: PASS, 14 passed / 0 failed / 9 not_applicable (item class is `metadata`, so the Go rows are N/A; they were run by hand above) |
| `./scripts/candidate-fingerprint candidate` | `1c16ef3295b52c175f6cec8a9275faee3ec3d24f9910f36a0d6018be1b2f1ff0` |

## Observations

- `guard lint` rejected the first version of the Go example (5 ×
  `err_discarded` on `fmt.Fprintf`/`fmt.Printf`/`fmt.Println`); output now goes
  through a checked `emit` helper, mirroring `internal/cli.writef`. Re-run of
  the lint row: exit 0.
- Adding `examples/go` to the root module means the mandatory
  `go test`/`go vet` rows compile it, which is why shape (a) was chosen; it
  shows up as `fake-jev/examples/go` in `go list ./...`.
- `CGO_ENABLED=0` produces a fully static linux binary but the darwin binary
  still lists `/usr/lib/libSystem.B.dylib` and macOS frameworks under `otool
  -L`; that is the normal Go-on-darwin linkage and does not affect the image,
  which builds linux.
- There is no `.dockerignore` (outside `allowedFiles`), so `COPY . .` sends
  the whole build context — including `.git` — to the builder. The final image
  is unaffected (only the binary and CA bundle are copied).
- `testdata/contracts/noul/01-basic-noul.json` served as the shape reference
  for the fixture; the example request uses criteria *strings* because §39.2
  does not allow boolean criteria values.
- `README.md` documents the release workflow as a capability ("cross-compiles
  … on a `v*` tag and attaches SHA256SUMS"): the workflow exists and parses,
  but no release has run, and no such claim of a past release is made.

## Residual risks

- The container image is unverified end to end: no daemon here, so no
  `docker build`, no image filesystem listing, no `docker run -p` reachability
  check. The base tag exists, the CA path is present in that base, and the
  build line and the bind flags were exercised locally, but the first real
  build will be in GitHub Actions or on a machine with a running daemon.
- `golang:1.26-bookworm` is resolved at build time; a future retag of the base
  could change the CA bundle or the toolchain patch level. The Dockerfile
  pins the major/minor the module declares but not a digest.
- `release.yml` has never executed. Its tag resolution
  (`github.event.inputs.tag || github.ref_name`), `gh release create
  --verify-tag`, the GHCR login/tag naming, and the artifact glob in the
  publish step are inspected and rehearsed locally but not run on a runner.
- The examples were executed on macOS/darwin-arm64 with Node v24.14.0, Python
  3.11.14, Go 1.26.3, curl 8.7.1, jq 1.8.2. Windows behaviour (path style, `sh`
  availability for the raw-HTTP example) was not exercised; the raw-HTTP
  example therefore documents `bash` rather than `sh`.
- The raw-HTTP example needs `curl` and `jq`; both are environment tools
  present on `ubuntu-latest` and here, but neither is a repository-owned
  dependency and a machine without `jq` will be told so explicitly.
