---
stage: cleaner
task: FJ-042
inputFingerprint: 1c16ef3295b52c175f6cec8a9275faee3ec3d24f9910f36a0d6018be1b2f1ff0
outputFingerprint: b108a8dd28dfd3b53b39add62b2ccc81d1fa0942c1b1cc27400dd98e805ddf55
taskFingerprint: 0a0388c558ac2868f0d8ee6d2dee106a087a88fa819e8ce3ca1e002bb199851e
gitHead: c2d671e
generatedAt: 2026-10-02T20:03:36Z
author: worker/FJ-042-cleaner-revision
---

# Cleaner — FJ-042

Structure found already coherent; no structural edit was applied, so the
candidate fingerprint is the coder's:

```
./scripts/candidate-fingerprint candidate
1c16ef3295b52c175f6cec8a9275faee3ec3d24f9910f36a0d6018be1b2f1ff0
```

`coder.md` front matter carries `outputFingerprint:
1c16ef3295b52c175f6cec8a9275faee3ec3d24f9910f36a0d6018be1b2f1ff0`; the
command above prints that same string, before and after every command in this
stage.

## Cleanup applied

None. Each candidate change below was considered and left in place:

| Candidate | Left as-is because |
|---|---|
| Cross-language duplication of the three check steps in the four examples | Inherent to "one runnable example per language"; extracting a shared helper would add a dependency or a wrapper surface that §24.3/§31 Phase 5 make a non-goal |
| `examples/README.md` restating start-server commands from `README.md` | Scoped to running the examples; the top-level README points at it rather than inlining it |
| CA bundle copy into the `scratch` stage (Dockerfile:39) | Documented in the file as an intentional `§24.2` "required CA/system file"; removing it changes image content, so it is a product decision, not a structural one |
| `emit` helper in `examples/go/main.go` | It is the checked-write seam `guard lint` (`err_discarded`) requires; the comment names the rule |
| `version`/`help` argument handling unchanged | Outside this item's `allowedFiles` |

## Over-claim audit (README.md, examples/README.md)

Every claim that names a repo artifact or a command was checked against the
repository. The two items that do not correspond to an artifact that exists
today are listed under Findings.

- `README.md:7-11` status list — every named surface exists: `cmd/fake-jev`,
  `internal/` (`jev/v1` data plane, `/__fake/v1/*` control API),
  `testdata/contracts/`, `examples/`, `Dockerfile`,
  `.github/workflows/release.yml` (the release build matrix). The phrase "the
  container image" resolves to the `Dockerfile`, not to a built/pushed image
  (see Findings 1).
- `README.md:22-36` — contains the `§32` sentence verbatim
  (`grep -nF 'fake-jev does not simulate model quality or intelligence.'
  README.md` → line 22); the "normal execution performs no network egress"
  statement matches the runtime (`internal/` has no outbound HTTP client; the
  only `http.Client` uses are `internal/cli/verify.go` and `internal/cli/run.go`
  calling the loopback control API). The `§33` absent-feature list matches the
  code (`grep -rniE 'auto.?mode|plugin|dashboard|openapi' internal/ --include=*.go`
  excluding tests → no matches). "never logged" matches the observed log line
  `authorization=[redacted]`.
- `README.md:44-55` install block — `go 1.26` in `go.mod` matches the "Go 1.26+"
  sentence; the five artifact names equal `release.yml`'s matrix legs; the
  "static binary, non-root, no Go inside" parenthetical is what `Dockerfile`
  declares (`CGO_ENABLED=0`, `USER 65532:65532`, `FROM scratch`). The download
  and `docker build` lines are instructions, not statements that either already
  happened (Findings 1, 2).
- `README.md:58-65` — `Dockerfile`'s `CMD ["serve", "--host", "0.0.0.0",
  "--port", "8787"]`; the binary's own default is `config.DefaultHost =
  "127.0.0.1"` (`internal/config/schema.go:7`). The mount example passes
  `serve --config … --host 0.0.0.0`, which `serve` accepts.
- `README.md:86-92` — flag list matches `serve.go` (`--host`, `--port`,
  `--config`, `--compat` repeatable, `--mode`, `--log-format`, `--ready-file`)
  and `internal/config/validate.go:207` rejects any `--mode` other than
  `strict`. The ready-file document matches `readyDocument`
  (`url, host, port, pid, controlApiVersion`); a live run with `--port 0
  --ready-file` wrote
  `{"url":"http://127.0.0.1:63216","host":"127.0.0.1","port":63216,"pid":75485,"controlApiVersion":"v1"}`
  and `SIGTERM` to the server removed the file.
- `README.md:94-102` — `run` help/flags (`--config`, `--url-env`, `--`) match
  `internal/cli/run.go`; exit-code line exercised (below).
- `README.md:105-125` — `127.0.0.1` default bind, the `§19.5` untrusted-network
  warning, the unauthenticated `/__fake/v1/*` (including `verify` and `reset`)
  claim, and "fixtures are data only" all match `internal/control/handlers.go`
  routes and the absence of any template/exec/callback path in `internal/`
  (`os/exec` appears only in `internal/cli/run.go`, which spawns the user's own
  child command).
- `README.md:128-137` — "the guard rows (`arch`, `lint`, `trace`, `fuzz`) and
  the Go race row" matches `scripts/verify-candidate` (`check_guard_arch`,
  `check_guard_lint`, `check_guard_trace`, `check_guard_fuzz`, `check_go_race`).
- `examples/README.md:19-30` — dependency claims match the tree: `find examples
  -name package.json -o -name node_modules -o -name 'requirements*.txt' -o -name
  vendor -o -name go.mod` → 0 hits; `grep -RInoE 'https?://…' examples` → only
  `127.0.0.1`; no credential is read (`FAKE_JEV_URL` is the only env read).
- `examples/README.md:15-17, 63, 67` — the `route`/`urgent` answers are what the
  four runs printed; the "verified on" versions equal the local environment
  (Node v24.14.0, Python 3.11.14).
- `examples/README.md:58-61` — exit-status sentence matches behavior: child
  failure propagates the child's code (`exit 7` case), child success with a
  failed verification returns `3`, success returns `0`.

### Over-claim findings

1. `README.md:47,51` — "Or download a released binary. … `gh release download
   <tag>`". `git tag` is empty at this revision, so no release (and therefore no
   downloadable binary) exists yet. The surrounding comment describes the
   workflow's behavior on a `v*` tag rather than asserting a past release, and no
   sentence claims a release ran. Recorded, not edited — the wording is a
   capability description of `release.yml`, and editing it changes the candidate
   fingerprint.
2. `README.md:10` — the status list names "the container image" among things
   "present". The `Dockerfile` is present; no image was built here (the daemon is
   unreachable: `docker build` exits 1 with `dial unix
   /Users/mak/.docker/run/docker.sock: no such file or directory`), and no image
   is published. Read as the §24.2 build definition. Recorded, not edited.

Neither finding is a claim that an image was built or that a release ran; no
README sentence was found asserting either.

## Command outcomes

Working copy and cleanliness:

```
git status --porcelain -uall   # README.md modified; Dockerfile, release.yml,
                               # examples/** untracked; .agent/work/FJ-042/{state.json,specifier.md,coder.md}
                               # plus .agent/reports/FJ-042/* from verify-candidate
git diff --cached --name-only | wc -l   -> 0     (nothing staged)
git diff --stat c2d671e -- go.mod go.sum        -> empty (no new dependency)
git diff --stat c2d671e -- .github/workflows/ci.yml -> empty (ci.yml untouched)
```

| Command | Outcome |
|---|---|
| `go test ./... -count=1` | exit 0, 552 tests in 10 packages |
| `go vet ./...` | exit 0, no output |
| `go run ./cmd/guard arch` | exit 0, `{"files":60,"packages":10}`, `findings: []` |
| `go run ./cmd/guard lint` | exit 0, `{"files":60,"packages":10}`, `findings: []` |
| `gofmt -l examples/` | no output |
| `go list ./...` | includes `fake-jev/examples/go` |
| `bash -n examples/raw-http/systemone.sh` | exit 0 (only `.sh` under `examples/`) |
| `shellcheck examples/raw-http/systemone.sh` | exit 0 |
| `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/release.yml'))"` | parses; triggers `['push','workflow_dispatch']`, jobs `['binaries','checksums','image','release']`, top-level `permissions {contents: write}` |
| `/tmp/fj-clean/fake-jev validate examples/fake-jev.yaml` | exit 0, `configuration is valid` |
| four examples via `/tmp/fj-clean/fake-jev run --config examples/fake-jev.yaml -- …` | node 0, python3 0, `go run ./examples/go` 0, `bash examples/raw-http/systemone.sh` 0; each printed `OK: the fixture answers matched the expected values` and `GET /__fake/v1/verify: {"passed":true,"failures":[]}` |
| matrix rehearsal (5 legs, `CGO_ENABLED=0 GOOS=… GOARCH=… go build -trimpath -o dist/fake-jev-<os>-<arch>`) | 5/5 exit 0; the parent dir is created by `go build -o dist/…` (checked from an empty `dist/`) |
| `release.yml` checksum `run:` block extracted with PyYAML and executed verbatim | exit 0; five lines, digests identical to `coder.md` (`e1da5b96…`, `d79e0380…`, `a2cfd0f6…`, `432a3eb8…`, `0f685d33…`), then five `OK` from `sha256sum --check` |
| same checksum block with `dist/fake-jev-windows-amd64.exe` removed | exit 1 (the `test -f` guard; `set -euo pipefail`) |
| `fake-jev run --config <fixture with expect: exactly 1> -- true` | exit 3 (child 0, verification failed) |
| `fake-jev run --config examples/fake-jev.yaml -- bash -c 'exit 7'` | exit 7 |
| `fake-jev serve --port 0 --ready-file` then `SIGTERM` | ready JSON as in the audit above; file removed after the server process received SIGTERM |
| `./scripts/verify-candidate FJ-042` | exit 0, `RESULT: PASS`, 14 passed / 0 failed / 9 not_applicable (`.agent/reports/FJ-042/report-20261002T195023Z.json`) |
| `./scripts/candidate-fingerprint candidate` | `1c16ef3295b52c175f6cec8a9275faee3ec3d24f9910f36a0d6018be1b2f1ff0` |

## Wire-contract observations (examples)

- All four examples call exactly `GET /v1/models`, `POST /v1/systemone`,
  `GET /__fake/v1/verify`; `internal/cli/verify.go` uses the same path
  (`verificationPath`), so the "same control call `fake-jev verify` performs"
  sentence holds.
- The POST body (`state`, `model`, `questions` with string `criteria`) is
  accepted by `§39.2`; criteria values are strings, which is the accepted form
  (`§39.2`: boolean criteria values are not in the allowed set), and the stub's
  `when.questions` / `then.answers` shape matches the observed response
  (`{"route":{"type":"choice","choice":"backend","confidence":0.91,…},"urgent":{"type":"noul","noul":0.94}}`).
- The failure item type declared in the examples (`{code, message}`) is a subset
  of `internal/control/verify.go`'s item (`code, message, requestSequence,
  stubId`); unknown fields are ignored by all four decoders.
- Request shapes are the upstream provider shape (`state`/`model`/`questions`),
  not a fake-jev-specific one.

## Residual risks

- `release.yml` has not executed on a runner: tag resolution
  (`github.event.inputs.tag || github.ref_name`), `gh release create
  --verify-tag`, the GHCR login/tag, the buildx multi-arch push, and the
  `needs.image.outputs.digest` hand-off are inspected plus locally rehearsed
  (build + checksum lines) only. `actionlint` is not installed here, so the
  workflow was checked by YAML parse, structural dump, and command rehearsal.
- The cross-job file names depend on `actions/upload-artifact@v4` placing a
  single-file upload at the artifact root and `download-artifact@v4` with
  `merge-multiple: true` merging it into `dist/`; not executable here (no
  runner) and not covered by any local check.
- Image build and run are unrunnable here (no daemon, no registry credentials);
  the base tag `golang:1.26-bookworm` and the builder's CA path were confirmed
  in `coder.md` by an HTTP check, not re-checked in this stage.
- `COPY . .` in the builder sends the whole context (no `.dockerignore`; the
  file would be outside `allowedFiles`). Final image content is unaffected.
- The examples were exercised on darwin/arm64 with Node v24.14.0, Python
  3.11.14, Go 1.26.3, curl 8.7.1, jq 1.8.2; Windows behavior of the raw-HTTP
  example (bash herestring) is untested and the example documents `bash`.
- An empty `git tag` set means the release path in `release.yml` has never been
  exercised end to end (see Findings 1).
- The `same-value` front matter requirement pins this stage to the coder's
  fingerprint; a structural edit would have required re-running the downstream
  gauntlet, which is why no edit was made.

---

# Cleaner revision — documentation-accuracy correction

Input was the previous cleaner candidate fingerprint
`1c16ef3295b52c175f6cec8a9275faee3ec3d24f9910f36a0d6018be1b2f1ff0`. QA and the
previous cleaner pass recorded README over-claims that FJ-042 now treats as in
scope (public documentation accuracy). This revision edits `README.md` only;
no Go source, `Dockerfile`, `.github/workflows/release.yml`, or `examples/**`
file changed.

## Exact edits (README.md)

### Edit 1 — status list (was lines 10-12)

Before:

```text
golden contract vectors, the release build matrix, the container image, and the
examples are all present. Behavior is defined by the specification — not by this
file.
```

After:

```text
golden contract vectors, the release build matrix, the container build
(`Dockerfile`), and the examples are all present. Behavior is defined by the
specification — not by this file.
```

Justification: `git tag` is empty and no image was built or published here
(`docker build` is unreachable — the daemon is absent), so "the container
image" named an artifact that does not exist. The `Dockerfile` does exist, so
the line now names the committed build definition, the capability, not a built
image. The paragraph was re-wrapped to keep the original line width.

### Edit 2 — install comment, released binary (was line 47)

Before:

```text
# Or download a released binary. `.github/workflows/release.yml` cross-compiles
# the platform matrix below on a `v*` tag and attaches SHA256SUMS:
```

After:

```text
# Or, when a release exists, download the published binary.
# `.github/workflows/release.yml` cross-compiles the platform matrix below on a
# `v*` tag and attaches SHA256SUMS:
```

Justification: `git tag` is empty, so no release and no downloadable binary
exists yet. "when a release exists" frames the `gh release download <tag>` line
below as available once the committed workflow has published a release; it
asserts no past release and no build that already ran.

### Edit 3 — install comment, container image (was line 53)

Before:

```text
# Or build and run the container image (static binary, non-root, no Go inside).
```

After:

```text
# Or build and run the image from the committed `Dockerfile` (static binary,
# non-root, no Go inside).
```

Justification: the line is an instruction, but "the container image" could read
as an existing artifact. It now names the committed `Dockerfile` as the source
of the image, so the capability is clearly a local `docker build`. The
`docker build` / `docker run` commands below are unchanged.

### Edit 4 — `--log-format` accuracy (was lines 97-100)

Before:

```text
`--compat` (repeatable), `--mode strict`, `--log-format text|json`,
`--ready-file`. With `--ready-file`, the file is written atomically once the
listener and control API are ready, and removed on clean shutdown:
```

After:

```text
`--compat` (repeatable), `--mode strict`, `--log-format text|json`,
`--ready-file`. `--log-format` accepts `text` or `json` and rejects any other
value; both accepted values currently produce the same text output, since
distinct JSON log output is not implemented yet. With `--ready-file`, the file
is written atomically once the listener and control API are ready, and removed
on clean shutdown:
```

Justification: `internal/cli/serve.go:106` accepts exactly `text` and `json`
and rejects other values (`internal/cli/serve.go:107`); `serve` never branches
on `flags.logFormat` after validation, and `serve` builds one
`log.New(stderr, "fake-jev: ", 0)` (`internal/cli/serve.go:283`) plus the shared
text operational logger. The README now states the accepted-and-validated
contract and that JSON output is not distinct. JSON logging is not implemented
here; FJ-042's non-goal is "no log-format json beyond the flag contract".

### Edit 5 — `run` example child command (was line 112)

Before:

```text
./fake-jev run --config examples/fake-jev.yaml --url-env JEV_BASE_URL -- npm test
```

After:

```text
./fake-jev run --config examples/fake-jev.yaml --url-env JEV_BASE_URL -- go run ./examples/go
```

Justification: this repository has no `package.json` and no `npm test` script,
so the printed command failed for a reader who copied it. `go run ./examples/go`
is a committed, runnable example that talks to the temporary server. The
surrounding sentence (server on an ephemeral loopback port, `FAKE_JEV_URL` plus
`--url-env NAME`, command after `--`, verification, shutdown) is unchanged.

## Verification of the touched claims against the built binary

Binary built from this revision: `go build -o /tmp/fj042/fake-jev ./cmd/fake-jev`.

`--log-format` contract (Edit 4):

```text
$ /tmp/fj042/fake-jev serve --port 0 --log-format xml
fake-jev: log-format "xml" is not supported; use text or json
... usage ...                                              (exit 2)

$ serve --port 0 --log-format text  (one GET /v1/models, then SIGTERM)
fake-jev: listening on http://127.0.0.1:64063
fake-jev: GET /v1/models
fake-jev: received terminated; shutting down

$ serve --port 0 --log-format json  (same request)
fake-jev: listening on http://127.0.0.1:64065
fake-jev: GET /v1/models
fake-jev: received terminated; shutting down
```

Both accepted values produced byte-identical text; no JSON log line appeared.
This matches the revised sentence.

`run` example child (Edit 5):

```text
$ /tmp/fj042/fake-jev run --config examples/fake-jev.yaml --url-env JEV_BASE_URL \
    -- go run ./examples/go
fake-jev base URL: http://127.0.0.1:64069
...
OK: the fixture answers matched the expected values
fake-jev: GET /__fake/v1/verify
                                                           (exit 0)
```

Edits 1-3 are rewordings of claims about artifacts that do not exist here
(`git tag` empty, no built/published image); no command was run for them beyond
confirming `git tag` is empty and the `Dockerfile` and `release.yml` exist.

## Command outcomes (this revision)

| Command | Outcome |
|---|---|
| `go test ./... -count=1` | exit 0, 552 tests in 10 packages |
| `go vet ./...` | exit 0, no output |
| `go run ./cmd/guard arch` | exit 0, `{"files":60,"packages":10}`, `findings: []` |
| `go run ./cmd/guard lint` | exit 0, `{"files":60,"packages":10}`, `findings: []` |
| `go build -o /tmp/fj042/fake-jev ./cmd/fake-jev` | exit 0 |
| `--log-format` text vs json vs invalid | text and json identical text output; invalid rejected exit 2 |
| `run … -- go run ./examples/go` | exit 0, three exchanges, `OK: the fixture answers matched the expected values` |
| `git diff --cached --name-only \| wc -l` | 0 (nothing staged) |
| `./scripts/candidate-fingerprint candidate` | `b108a8dd28dfd3b53b39add62b2ccc81d1fa0942c1b1cc27400dd98e805ddf55` |

## New candidate fingerprint

```
./scripts/candidate-fingerprint candidate
b108a8dd28dfd3b53b39add62b2ccc81d1fa0942c1b1cc27400dd98e805ddf55
```

Previous candidate fingerprint (input): `1c16ef3295b52c175f6cec8a9275faee3ec3d24f9910f36a0d6018be1b2f1ff0`.
`README.md` is a candidate-class metadata file, so this documentation edit
changes the candidate fingerprint; no Go, `Dockerfile`, `release.yml`, or
`examples/**` content changed.

## Residual risks

- Edits 1-3 remain prose: the artifact-availability claims cannot be executed
  here (`git tag` is empty, no container daemon, no registry credentials). They
  were rewritten to describe committed capabilities (`Dockerfile`,
  `release.yml`) rather than existing artifacts.
- The `--log-format` gap itself is unchanged and still owned by the follow-up
  ticket; the README now documents the gap instead of hiding it.
