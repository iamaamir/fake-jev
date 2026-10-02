---
stage: specifier
task: FJ-042
inputFingerprint: 325768fd9665e78ee4264d86eb09e1dc651a4918c6ad1e140276a5b89208de31
outputFingerprint: 325768fd9665e78ee4264d86eb09e1dc651a4918c6ad1e140276a5b89208de31
taskFingerprint: 0a0388c558ac2868f0d8ee6d2dee106a087a88fa819e8ce3ca1e002bb199851e
gitHead: c2d671e
generatedAt: 2026-10-02T19:27:04Z
author: worker/FJ-042-specifier
---

# Specifier — FJ-042 (Phase 4: release matrix, container, examples)

Objective restated as observables: (A) a documented release build matrix
produces the five §23.1 platform binaries plus per-artifact checksums;
(B) a minimal container image runs the server with no Go toolchain present
inside the image and contains only what running the binary needs; (C) four
usage examples (TypeScript, Python, Go, raw HTTP) drive the running fake
server offline, each adding no dependency and requiring no credential, model,
or provider SDK; (D) a release workflow produces the artifacts on a version
tag without changing normal CI; (E) `README.md` stops claiming the repository
is "foundation phase" and states that fake-jev does not simulate model
quality or intelligence. No verdict is expressed here; results come only from
`./scripts/verify-candidate FJ-042`.

## Traces

- C-QUAL-005
- C-ARCH-003
- C-ARCH-004

## Baseline at c2d671e — what already exists and what the delta is

- `.github/workflows/ci.yml` already has a `build` job that cross-compiles the
  five §23.1 targets into `dist/fake-jev-<goos>-<goarch>` with
  `CGO_ENABLED=0`, but the job uploads nothing and computes no checksums; its
  own comment says "§23.2 release publishing is out of scope". That job is a
  compile proof, not a release.
- There is no `Dockerfile`, no `examples/` directory, and no
  `.github/workflows/release.yml` at the base revision.
- `README.md` says "**Status: foundation phase.** No product implementation
  has begun." and its Verify section says product checks "are added centrally
  as implementation phases land". Both are false at `c2d671e`: the full v1
  implementation exists (`cmd/fake-jev`, `internal/…`), and
  `scripts/verify-candidate` already runs `go test ./...`, `go vet ./...`, and
  `go build ./cmd/fake-jev`. README contains no model-quality/intelligence
  statement and no untrusted-network warning.
- `go.mod` is module `fake-jev`, `go 1.26`, with one direct requirement
  (`gopkg.in/yaml.v3`). `go.mod` is **outside** this item's `allowedFiles`, so
  the examples structurally cannot add a Go module dependency; the only
  degrees of freedom are stdlib-only code under the root module, or a nested
  `examples/**/go.mod` that itself declares no `require`.
- `scripts/verify-candidate` (and `scripts/`) is outside `allowedFiles`. This
  item therefore cannot add a verifier row, and nothing below assumes one.
  Every criterion is stated as an observable plus a command or an exact
  inspection that a reviewer runs or reads directly. Where the command cannot
  run in this workspace it is called out explicitly.

## A. Platform build matrix and per-artifact checksums (§23.1, §23.2, §24.1)

### A1 — the release matrix names exactly the five §23.1 targets

- Observable: the release workflow's build matrix contains exactly
  `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`,
  `windows/amd64`, and each leg produces a platform-named executable artifact
  under `dist/` (`fake-jev-<goos>-<goarch>`, with the Windows leg carrying the
  `.exe` suffix). `windows/arm64` is absent because §23.1 makes it optional
  and no demand is recorded.
- Spec clause: §23.1 (initial release targets), §24.1 ("GitHub releases should
  provide platform-specific binaries"), §32 ("a standalone native binary is
  released for the required platform matrix").
- Deterministic verification (local, no daemon needed): the exact commands the
  workflow runs, run by hand over a temp output directory —

  ```bash
  out="$(mktemp -d)"
  for t in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64; do
    os=${t%/*}; arch=${t#*/}
    CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" \
      go build -o "$out/fake-jev-$os-$arch" ./cmd/fake-jev
  done
  ls "$out"
  ```

  then confirm the five names are present and that the release workflow's
  matrix block matches them (inspection:
  `grep -n 'goos\|goarch\|matrix' .github/workflows/release.yml`).
- Local executability: the cross-compile runs locally. Executing the
  non-host binaries does not (see "Not executable locally").

### A2 — per-artifact checksums on the released artifacts

- Observable: the release produces one checksum file (`SHA256SUMS`)
  containing exactly one line per released binary artifact, each of the form
  `<64 lowercase hex sha256><two spaces><artifact filename>`; the workflow
  fails before publishing if a checksum cannot be produced, and the checksum
  file is attached to the GitHub Release together with the binaries.
- Spec clause: §23.2 ("Release artifacts should include checksums"), §31
  Phase 4 deliverables ("checksums").
- Deterministic verification (local): build the matrix as in A1, then run the
  same checksum command the workflow uses and assert the line shape and count:

  ```bash
  ( cd "$out" && sha256sum fake-jev-* > SHA256SUMS && cat SHA256SUMS )   # macOS: use `shasum -a 256`
  awk 'NF!=2 || $2 ~ /[^[:alnum:]._-]/ || length($1)!=64 { bad=1 } END { print (bad?"shape-mismatch":"shape-ok") }' "$out/SHA256SUMS"
  ```

  (On macOS `sha256sum` exists via GNU coreutils; otherwise `shasum -a 256`
  is the same digest.) Inspect the workflow for the checksum step and for the
  release upload step (inspection:
  `grep -n 'sha256\|SHA256SUMS\|softprops\|gh release\|upload' .github/workflows/release.yml`).
  That the file is actually attached and matches the uploaded bytes requires a
  real release run (see "Not executable locally").
- Local executability: the digest computation and format are local; the
  publish-and-verify half is not.

## B. Minimal container image (§24.2, §19.1, §30.8)

### B1 — the image builds the server without a Go toolchain in the final stage

- Observable: `Dockerfile` at the repository root is multi-stage. A builder
  stage carries the Go toolchain and compiles `./cmd/fake-jev` with
  `CGO_ENABLED=0` for the target platform (`ARG TARGETOS`/`ARG TARGETARCH`,
  defaulting to `linux/amd64`); the **final** stage contains no Go toolchain
  and no `golang` base image. The image's default command is the server
  (`serve`).
- Spec clause: §31 Phase 4 ("Docker image"; "A user can download a single
  binary/container and use it without installing Go, a model, or a provider
  SDK"), §24.1, §32.
- Deterministic verification: inspection of the `FROM` lines — the final
  stage's `FROM` is not a Go image and the builder stage is not referenced by
  the runtime stage; plus, when a daemon is available, a build:

  ```bash
  docker build -t fake-jev:local .
  ```

  A YAML/structural check that needs no daemon is the stage inspection
  (`grep -n '^FROM\|COPY --from\|ENTRYPOINT\|CMD\|EXPOSE' Dockerfile`).
- Local executability: **not executable in this workspace at spec time** — the
  `docker` client exists but the daemon is unreachable
  (`dial unix /Users/mak/.docker/run/docker.sock: no such file or directory`).
  The build executes in GitHub Actions (`ubuntu-latest` has a daemon) and on
  any machine with a running daemon. Achievable evidence without a daemon:
  the Dockerfile inspection above plus the A1 binary compile.

### B2 — the image runs the server with no Go toolchain present

- Observable: `docker run --rm -p 8787:8787 <image>` starts the server and
  `GET /v1/models` succeeds; the image filesystem contains no `go` executable
  and no `/usr/local/go` tree. The binary is statically linked (`CGO_ENABLED=0`,
  no dynamic loader), so it runs on a runtime base that provides no toolchain.
- Spec clause: §31 Phase 4 acceptance, §32 ("normal execution requires no
  model, … no outbound network"; "a standalone native binary"), §4.9.
- Deterministic verification (with a daemon): list the image filesystem
  without needing a shell in the image, and drive the server:

  ```bash
  docker create --name fjinspect fake-jev:local >/dev/null
  docker export fjinspect | tar -t        # expect no path containing 'go/bin' or '/usr/local/go'
  docker rm fjinspect
  docker run --rm -d --name fjrun -p 8787:8787 fake-jev:local
  curl --fail-with-body http://127.0.0.1:8787/v1/models
  docker stop fjrun
  ```

  Without a daemon, the same property is evidenced one level down on the
  locally built binary: `file <binary>` reports a statically linked
  executable and `ldd <binary>` reports "not a dynamic executable" (or the
  platform equivalent), so nothing external to the binary is needed to run it.
- Local executability: the binary-level static-link check is local; the
  image-level check needs a daemon.

### B3 — the final image contains only what running the binary needs

- Observable: the final stage copies only the server binary (and nothing else:
  no package manager cache, no source, no shell, no Go tree). If a runtime base
  other than `scratch` is chosen, the only additions are OS/CA/system files
  that the chosen base provides.
- Spec clause: §24.2 ("The final image should contain only what is required to
  run the binary and any required CA/system files").
- Deterministic verification: the `docker export | tar -t` listing in B2,
  asserted to contain the binary path plus only base-image system files that
  the Dockerfile visibly derives from its `FROM` (inspection of the final-stage
  `FROM` and `COPY` lines). No size threshold is asserted (see Spec-silent
  observations).
- Local executability: requires a daemon; otherwise Dockerfile inspection.

### B4 — container-friendly binding is an explicit choice; the built-in default stays loopback

- Observable: the image makes the server reachable from outside the container
  by an **explicit** `--host 0.0.0.0` in its `CMD`/`ENTRYPOINT` (or an explicit
  equivalent in a config the image ships), so `docker run -p 8787:8787` works;
  the binary's/config's built-in default bind remains `127.0.0.1`, i.e. no
  code or default changes to §19.1/§30.8.
- Spec clause: §19.1 ("Do not bind publicly by default. Users running in
  Docker may explicitly choose `0.0.0.0`"), §30.8 ("Do not listen on `0.0.0.0`
  by default"), §24.2 (container-friendly binding, example
  `docker run --rm -p 8787:8787 fake-jev …`), §37 ("Default bind |
  `127.0.0.1:8787` for `serve`").
- Deterministic verification: inspection of the Dockerfile `CMD`/`ENTRYPOINT`
  for the literal `0.0.0.0` flag or an explicit shipped config host, plus the
  existing default-bind observation staying true: `./fake-jev serve` with no
  flags still reports `127.0.0.1` in its ready file (existing coverage in
  `internal/cli/serve_test.go`; not modified here). The repository's config
  default is asserted by the unchanged `go test ./...`.
- Local executability: inspection is local; the `docker run -p` reachability
  check needs a daemon.

## C. Usage examples (§31 Phase 4, §27)

Proposed layout (the coder may rename; the exact paths must then be recorded
in `coder.md`, and the commands below use the recorded paths):
`examples/fake-jev.yaml` (shared fixture), `examples/typescript/systemone.ts`,
`examples/python/systemone.py`, `examples/go/main.go` (+ optional nested
`go.mod`), `examples/raw-http/systemone.sh`, `examples/README.md`.

### C0 — the shared fixture is a valid v1 configuration and fixes the answers

- Observable: one fixture config defines `schemaVersion: 1`, `mode: strict`,
  `compatibility: [jev/v1]`, and one `jev/v1` stub matching
  `operation: systemone` with a `choice` question `route` and a `noul`
  question `urgent`, returning the §13.1/§13.2 exact answer shapes
  (`{type: noul, noul: 0.94}` and
  `{type: choice, choice: backend, confidence: 0.91, probabilities: {…}}`).
  It uses no `expect` block (the four examples may each invoke it once, and
  §12.7 makes an omitted `expect` an absence of expectations).
- Spec clause: §12.3 (top-level schema), §12.5 (matcher shape), §12.6/§13
  (response shapes), §27 (end-to-end workflow).
- Deterministic verification: `./fake-jev validate examples/fake-jev.yaml`
  exits `0` and starts no server (§15.2). Mirrors the repository's existing
  golden fixtures; it does not need the server running.
- Local executability: local.

### C1 — TypeScript example, offline, no npm dependency

- Observable: a `.ts` example uses the runtime's built-in HTTP client
  (`fetch`) to call `GET /v1/models` and `POST /v1/systemone` and compares the
  parsed JSON answer to the fixture's fixed answer; it runs directly under
  Node with no `npm install`, no `node_modules`, and no `package.json`
  declaring dependencies. It uses only erasable TypeScript syntax (no `enum`,
  `namespace`, or parameter properties) so type stripping needs no transform.
- Spec clause: §31 Phase 4 ("examples for TypeScript …"), §24 ("The
  implementation language should be invisible to most users"), C-QUAL-005
  (offline), §30.2 ("Do not build one fake per language": the example talks
  HTTP, it does not fake anything).
- Deterministic verification: with a server running at the ready-file URL,

  ```bash
  FAKE_JEV_URL="http://127.0.0.1:8787" node examples/typescript/systemone.ts
  ```

  (Node ≥23 strips types by default for erasable syntax; Node 22.6–22.x uses
  `node --experimental-strip-types …`.) The command's exit code is the
  observable: `0` on match, non-zero on mismatch. The expected-answer values
  are literal in the example (not computed), so the outcome is deterministic.
- Local executability: local (Node v24.14.0 present).

### C2 — Python example, stdlib only, offline

- Observable: a `python3` example uses only the standard library
  (`urllib.request`/`http.client` + `json`) to make the same two calls and
  asserts the same answer; no `requirements.txt`, no virtualenv, no `pip`.
- Spec clause: §31 Phase 4, C-QUAL-005.
- Deterministic verification:
  `FAKE_JEV_URL="http://127.0.0.1:8787" python3 examples/python/systemone.py`
  — exit `0` on match, non-zero on mismatch.
- Local executability: local (Python 3.11.14 present).

### C3 — Go example, stdlib only, offline

- Observable: a Go example uses only `net/http`, `encoding/json`, and other
  standard-library packages to make the same two calls and assert the same
  answer. Two mutually exclusive acceptable shapes, and the coder records in
  `coder.md` which was chosen:
  (a) **root module** — `examples/go/main.go` carries no `go.mod`, so the
  existing mandatory `go vet ./...` / `go build ./cmd/fake-jev` /
  `go test ./...` rows already compile-check it;
  (b) **nested module** — `examples/go/go.mod` exists and declares **no**
  `require`; root-module `./...` does not traverse into it, so it needs its
  own explicit compile step.
- Spec clause: §31 Phase 4, §17.2 (dependency policy), C-QUAL-005. PM
  constraint: "a Go example must either compile under the root module with
  stdlib only or carry its own go.mod inside examples/".
- Deterministic verification:
  - shape (a): `go vet ./examples/...` (also covered by the mandatory
    `go vet ./...` row) and
    `FAKE_JEV_URL="http://127.0.0.1:8787" go run ./examples/go`;
  - shape (b): `go -C examples/go build ./...` plus
    `go -C examples/go mod graph` (the graph lists only the example module)
    and `FAKE_JEV_URL="http://127.0.0.1:8787" go -C examples/go run .`.
  Exit code `0` on match is the observable.
- Local executability: local.

### C4 — raw HTTP example, offline, curl only

- Observable: `examples/raw-http/systemone.sh` issues the same
  `GET /v1/models` and `POST /v1/systemone` with `curl`
  (`--fail-with-body`, explicit JSON body) and checks the response with a
  tool already present (`jq` or a language-free string check), exiting
  non-zero on mismatch. `curl` is an environment tool, not a language
  package; no SDK, no library install.
- Spec clause: §4.1 ("HTTP is the universal interoperability boundary"),
  §31 Phase 4 ("raw HTTP").
- Deterministic verification: `FAKE_JEV_URL="http://127.0.0.1:8787" bash
  examples/raw-http/systemone.sh` (and `shellcheck examples/raw-http/*.sh` as
  a static check — both available locally).
- Local executability: local (`curl`, `jq`, `shellcheck` present).

### C5 — examples are offline: no dependency, no credential, no model, no provider SDK

- Observable (C-QUAL-005 for this item): none of the four examples installs or
  declares a dependency (`node_modules`, `package.json` dependency block,
  `requirements*.txt`, `vendor/`, a `require` in a nested `go.mod`); none
  reads a real credential (any `Authorization` header is the literal arbitrary
  bearer of §39.5, and `POST /v1/systemone` is accepted with no header at
  all); all network targets are loopback (`127.0.0.1`/`localhost` or the
  `FAKE_JEV_URL` env override); no model, provider SDK, or API key is needed.
- Spec clause: C-QUAL-005, §23 (normal CI and tests require no model/key/paid
  service), §4.9 and §19.2 (zero-model, no network egress), §39.5 (auth
  ignored), §17.2 (dependency policy).
- Deterministic verification (inspection, no network):

  ```bash
  find examples -name package.json -o -name node_modules -o -name 'requirements*.txt' -o -name vendor
  grep -RInE 'https?://' examples        # expect only 127.0.0.1 / localhost / env-derived URL
  grep -RInE 'API_KEY|APIKEY|SECRET|ACCESS_TOKEN|api[_-]?key' examples
  ```

  expecting no dependency artifacts, no non-loopback URL, and no credential
  reads; plus the four example run commands from C1–C4, all offline.
- Local executability: local.

### C6 — examples are thin clients; they introduce no §33 deferred feature (C-ARCH-003)

- Observable: each example makes at most the documented v1 data-plane calls
  (`GET /v1/models`, `POST /v1/systemone`) and, if it verifies, the documented
  control call `GET /__fake/v1/verify`; none reimplements matching,
  response generation, sequence handling, or lifecycle management, and none
  adds auto decision mode, a named scenario DSL, a plugin ABI, persistence, a
  UI/dashboard, control-plane auth, latency/fault injection, record/replay,
  provider OpenAPI ingestion, provider-quality evaluation, probabilistic
  response generation, or generic non-System-One HTTP mocking. No example is
  an installable wrapper package (Phase 5 is a non-goal).
- Spec clause: §33 (deferred decisions), §31 Phase 4/Phase 5 boundary, §16.1
  ("Runtime strategy must remain replaceable"), §30.2, §30.7; PM non-goals.
- Deterministic verification: inspection of each example's imports/requires
  (stdlib only) and of the exact HTTP paths it calls:

  ```bash
  grep -RInE '/v1/models|/v1/systemone|/__fake/v1' examples
  grep -RIlnE 'scenario|plugin|dashboard|persist|latency|fault|record|replay|openapi|auto[- ]?mode' examples
  ```

  The first must show only the three documented paths; the second must show
  no deferred-feature implementation (a word appearing solely in a "not
  supported" README sentence is a documentation match, not a feature).
- Local executability: local (inspection).

## D. Release workflow (§23.1, §23.2, §24.1, §24.2)

### D1 — the release workflow does not interfere with `ci.yml`

- Observable: `.github/workflows/release.yml` exists as a separate workflow
  whose triggers are a version-tag push (`tags: ['v*']`) and, optionally,
  `workflow_dispatch`; it does **not** trigger on `pull_request` or on `push`
  to `main`. `.github/workflows/ci.yml` is byte-identical to the base revision
  (it is outside `allowedFiles`). Normal PR CI behaviour and its triggers are
  unchanged; the two workflows share no job.
- Spec clause: §23 ("Normal PR CI must require no model, no provider API key,
  and no paid external service"), §23.2; PM constraint ("release.yml must not
  break the normal CI workflow").
- Deterministic verification:

  ```bash
  grep -n '^on:' -A6 .github/workflows/release.yml
  git diff --name-only c2d671e -- .github/workflows/ci.yml   # expect no output
  python3 -c "import yaml; yaml.safe_load(open('.github/workflows/release.yml'))"
  ```

  (PyYAML 6.0.3 is importable here, the same pin the verifier uses.)
- Local executability: local. GitHub-side trigger behaviour itself requires a
  real run (see "Not executable locally").

### D2 — on a version tag, the workflow builds the matrix, checksums, and publishes

- Observable: the workflow has a build job over the five §23.1 targets that
  uploads each `dist/` binary as a workflow artifact, a checksum step
  producing `SHA256SUMS` (one line per binary, A2 shape), and a publish step
  that creates/updates the GitHub Release for the pushed tag and attaches the
  binaries plus `SHA256SUMS`. It declares `permissions: contents: write`
  (uploading a release needs it; `ci.yml` stays `contents: read`).
- Spec clause: §24.1, §23.1, §23.2, §31 Phase 4 ("release build matrix;
  checksums").
- Deterministic verification: inspection of the build/checksum/publish steps
  and of `permissions:`; plus the local A1/A2 rehearsal. The end-to-end
  publish is a real release run only.
- Local executability: inspection and A1/A2 rehearsal are local; the publish
  is not.

### D3 — the container image is built (and published) by the release workflow

- Observable: a release job builds the root `Dockerfile` and pushes the image
  to a registry (`ghcr.io/<owner>/fake-jev:<tag>`), for `linux/amd64` and
  `linux/arm64`, and records the resulting image digest in the release notes
  or body. The image build uses the same Dockerfile B1–B4 describe; no
  second container build definition exists.
- Spec clause: §24.2 ("Provide a small image suitable for CI and local use"),
  §23.2 (documented, automated build), §31 Phase 4 ("Docker image").
- Deterministic verification: inspection of the workflow's image job (buildx
  platforms, push target, digest capture). The build/push itself needs a
  daemon plus registry credentials.
- Local executability: **not executable locally at spec time** (daemon
  unreachable; registry push and token unavailable). Achievable evidence:
  Dockerfile inspection, and the workflow's own run on a tag.

### D4 — the build process is documented and automated

- Observable: `README.md` (the public documentation surface in
  `allowedFiles`) documents the release build and checksum commands, and those
  exact commands appear in `release.yml`; `go.mod`'s Go version is the
  toolchain the Dockerfile's builder stage pins, so the build is not
  implicitly toolchain-dependent.
- Spec clause: §23.2 ("Releases should be reproducible enough that the build
  process is documented and automated").
- Deterministic verification: inspection — the README command block and the
  workflow's run steps agree; `grep -n 'go-version\|FROM golang' Dockerfile
  .github/workflows/release.yml go.mod`.
- Local executability: local (inspection).

## E. README corrections (§32, §19.5)

### E1 — the stale foundation-phase status is removed and corrected

- Observable: `README.md` no longer contains "No product implementation has
  begun" or the "foundation phase" status claim; it instead states the actual
  status consistent with the repository (the v1 `jev/v1` server, control API,
  and CLI exist), and the Verify section no longer says the product Go checks
  "are added … as implementation phases land" (they run now: `go test ./...`,
  `go vet ./...`, `go build ./cmd/fake-jev`). The claimed build/run commands
  are the ones the repository actually provides.
- Spec clause: §32 (v1 definition, whose items are already implemented);
  §24.1 (user-facing usage), §15.1 (`serve`).
- Deterministic verification:

  ```bash
  grep -c 'No product implementation has begun' README.md   # expect 0
  grep -ni 'foundation phase' README.md                       # expect no stale-status hit
  ```

  plus inspection that the replacement status text is not itself false (it
  must not claim deferred features or unimplemented surfaces).
- Local executability: local.

### E2 — public documentation states fake-jev does not simulate model quality or intelligence

- Observable: `README.md` contains an explicit statement that fake-jev does
  not simulate model quality or intelligence (e.g. that it does not evaluate
  or emulate the provider's intelligence, is not a replacement model, and
  returns only the developer-configured answers). The wording is free; the
  denial must be present.
- Spec clause: §32 ("public documentation states that fake-jev does not
  simulate model quality or intelligence"), §3.2 ("What fake-jev is not":
  not a replacement model, not a simulator of Jev's intelligence, not a
  model-quality evaluation framework), §4.9.
- Deterministic verification: inspection, made reproducible by requiring the
  coder to record the exact sentence and its line in `coder.md`; then
  `grep -nF '<recorded sentence>' README.md` finds it. (No exact spec wording
  exists — see Spec-silent observations.)
- Local executability: local.

### E3 — documentation warns against exposing the server to untrusted networks (adjacent §19.5 MUST)

- Observable: `README.md` warns users not to expose the server (data plane or
  the unauthenticated `/__fake/v1/*` control plane, §19.5, §33) to untrusted
  networks, and notes that non-loopback binds are an explicit choice (§19.1).
- Spec clause: §19.5 ("Documentation must warn users not to expose the server
  to untrusted networks"), §19.1, §30.8, §33 ("authentication for the control
  plane" deferred).
- Deterministic verification: inspection for an explicit warning sentence
  naming untrusted/external exposure of the control API.
- Local executability: local.
- Boundary note: §19.5 is a documentation MUST that the edited README is the
  natural home for. If the parent wants this item held to the exact two
  README corrections named in the objective, E3 is the single sentence to
  drop; it adds no code and no new public surface.

## F. Phase-4 prose acceptance — "a user can run the binary or image with no Go toolchain, model, or provider SDK"

- Observable: (i) a statically linked, platform-named binary executes with no
  `go` on `PATH` and no provider SDK installed, binding loopback by default
  and answering `GET /v1/models` and a configured `POST /v1/systemone`; (ii)
  the container image does the same with no Go toolchain inside the image
  (B1–B3); (iii) no API key is required for either (`Authorization` is ignored,
  §39.5); (iv) the module graph still contains only the one YAML dependency, so
  no model/provider SDK is linked.
- Spec clause: §31 Phase 4 acceptance ("A user can download a single
  binary/container and use it without installing Go, a model, or a provider
  SDK"), §32 ("a standalone native binary is released…"; "normal execution
  requires no model, model weights, GPU, API key, paid service, or outbound
  network"), §4.9, §19.2, C-ARCH-004.
- Deterministic verification (local, binary half):

  ```bash
  file  "$out/fake-jev-darwin-arm64"     # statically linked, no dynamic loader
  env -i PATH=/usr/bin:/bin "$out/fake-jev-darwin-arm64" serve \
        --config examples/fake-jev.yaml --port 8787 --ready-file /tmp/fj.ready &
  curl --fail-with-body -H 'Content-Type: application/json' \
        -d @- http://127.0.0.1:8787/v1/systemone <<'JSON'
  {"state":{"task":"demo"},"model":"jev-latest","questions":{"route":{"type":"choice","criteria":{"frontend":true,"backend":true,"infra":true}},"urgent":{"type":"noul"}}}
  JSON
  kill %1
  ```

  The minimal `env -i` environment carries no `go`, no provider credentials,
  and no SDK; the response is the fixture answer. The image half uses B2.
  `go list -m all` (existing, unchanged) shows the dependency graph.
- Local executability: the binary half is local on the host platform; the
  image half needs a daemon.

## G. C-ARCH-003 — no §33 deferred feature is introduced by this item

- Observable: the union of `Dockerfile`, `examples/**`,
  `.github/workflows/release.yml`, and the README delta introduces no §33
  deferred feature and no §31 Phase-5/6 deliverable: no auto decision mode,
  no named scenario/state-machine DSL beyond response sequences, no browser
  WASM, no WASM/plugin ABI, no third-party compatibility plugin, no persistent
  state/database, no UI/dashboard, no remote multi-user operation, no
  control-plane authentication, no advanced latency/fault injection, no
  record/replay, no provider OpenAPI ingestion, no provider-quality
  evaluation, no probabilistic generation, no generic HTTP mocking. It also
  adds no npm/Homebrew package, no GitHub Action, no TypeScript wrapper
  package, and no WASM build.
- Spec clause: §33 (deferred decisions), §31 Phase 5/6 gates, §30.2, §30.7,
  §16.1; PM non-goals.
- Deterministic verification: (a) the file-level diff is confined to
  `allowedFiles`:

  ```bash
  git status --porcelain
  git diff --name-only c2d671e  # expect only Dockerfile, examples/**, .github/workflows/release.yml, README.md
  ```

  (b) the C6 path/feature greps; (c) inspection that the Dockerfile's final
  stage adds no sidecar service, metrics port, or plugin loader, and that
  `release.yml` runs only build/checksum/publish steps. Any deferred-feature
  name appearing only inside a README "not supported" sentence is
  documentation, not an implementation.
- Local executability: local.

## Not executable locally — and what evidence is achievable instead

- **A real GitHub release run** (tag push → workflow → Release creation,
  asset upload, checksum file attached): not executable here. Achievable
  evidence: static inspection of `release.yml` (D1–D3), a local rehearsal of
  the exact build/checksum commands (A1/A2), and a YAML parse with the pinned
  PyYAML.
- **Container build and run** (`docker build`, `docker run`, the image
  filesystem listing): not executable in this workspace at spec time — the
  `docker` CLI is present (29.7.2) but the daemon is unreachable
  (`failed to connect to the docker API at
  unix:///Users/mak/.docker/run/docker.sock: … no such file or directory`).
  Achievable evidence: Dockerfile inspection (B1, B3, B4) and the local
  static-binary check (B2, F). On any machine with a running daemon (or in
  the `ubuntu-latest` release job) B1–B3, B4's reachability check, and D3's
  build execute.
- **Registry push and multi-arch manifest** (`docker buildx … --push`, GHCR
  digest): not executable here (no daemon, no credentials). Achievable
  evidence: workflow inspection (D3).
- **Executing the non-host release binaries** (linux, windows, and
  darwin/amd64 on this arm64 host): only host-native `darwin/arm64` executes
  locally. Achievable evidence: cross-compile all five targets successfully
  (A1) and `file`-inspect them; the release job runs the produced binaries'
  consumers.
- **`actionlint` / GitHub Actions schema validation**: `actionlint` is not
  installed. Achievable evidence: YAML parse (D1) plus inspection; the
  workflow executes for real only in GitHub Actions.
- **Cross-platform shell/Python/Node behaviour of the examples on Windows**:
  not exercised locally; the examples target the POSIX shell examples and the
  host platforms. Achievable evidence: local execution on macOS/Linux and
  inspection that the examples use no platform-specific API beyond the HTTP
  client.

## Spec-silent observations

- **No image size threshold.** §24.2 says "small" and "only what is required";
  it fixes no bytes, base image, or layer count. No criterion asserts a size.
  "Scratch vs distroless vs alpine" is not decided by the spec: the observable
  is "no Go toolchain and nothing beyond what running the binary needs".
- **No checksum algorithm or filename.** §23.2 says artifacts "should include
  checksums" without naming SHA-256 or `SHA256SUMS`. SHA-256 and
  `SHA256SUMS` are chosen here because they are the de-facto Go-release
  convention and are locally verifiable; the observable is "one checksum per
  released artifact, in a file attached to the release".
- **No release trigger specified.** The spec does not name a tag pattern,
  workflow filename, or publisher. `v*` tags and `.github/workflows/release.yml`
  are chosen to keep release activity disjoint from `ci.yml`; the observable is
  "release activity happens on tags and does not alter PR/push CI".
- **No required README wording** for the §32 model-quality statement or the
  §19.5 warning. Verification is a recorded exact-sentence grep, not a fixed
  phrase.
- **Image platform set.** §23.1 lists the binary matrix; it does not say which
  platforms the image must cover. `linux/amd64`+`linux/arm64` mirrors the two
  Linux binary targets; a single-arch image would still satisfy §24.2.
- **"Dependency" for the raw-HTTP example.** `curl` (and `jq`) are environment
  tools, not language packages. §24/§31 say "raw HTTP" without naming a tool.
  C4 assumes `curl` availability (present on `ubuntu-latest` and this host) and
  the example is documented so it is usable with literal `curl` commands even
  where `jq` is absent.
- **Type-checking of the TypeScript example.** "No dependency" rules out
  `tsc`/`ts-node`; the criterion is offline runnability via Node's built-in
  type stripping, not static type-checking. The example therefore avoids
  non-erasable syntax (no `enum`/`namespace`/parameter properties).
- **Per-artifact checksum content.** §23.2 does not say whether the checksum
  file covers the container image digest too. Criterion A2 covers the binary
  artifacts; the image digest is recorded in the release body (D3) and is not
  required to be inside `SHA256SUMS`.
- **`windows/arm64`.** §23.1 permits it "when release tooling and demand
  justify it"; nothing here adds it.

## Non-goals (respected, not re-litigated)

- No npm convenience package (§24.3), no Homebrew formula (§24.4), no GitHub
  Action (§24.5), no TypeScript wrapper package (§31 Phase 5 — a *usage
  example* that calls HTTP is not a wrapper), no WASM (§31 Phase 6), and no
  convenience wrapper that duplicates server behaviour.
- No changes to `ci.yml`, `scripts/**`, `spec/**`, `testdata/**`, `go.mod`,
  or any Go source: those paths are outside `allowedFiles`. The examples are
  consumers of the HTTP surface, not new production code.
- No new public behavior, config key, route, flag, or CLI surface. §33
  deferred features stay absent.
- No new verifier row: adding one would require `scripts/verify-candidate`,
  which is outside `allowedFiles`. Verification here is command- and
  inspection-based.

## Out of scope for this item

- Publishing to any registry beyond the container image named in §24.2, and
  any signing/attestation of artifacts (§23.2 asks only for checksums).
- Multi-architecture image attestations, SBOMs, and provenance metadata; none
  are required by the specification and none are proposed.
- Bit-for-bit reproducibility guarantees; §23.2 asks that the build be
  "documented and automated", which D4 covers.
- Making the examples part of the golden-contract suite (`testdata/contracts/`
  is outside `allowedFiles`); the examples evidence C-QUAL-005's offline
  property but are not golden vectors.
