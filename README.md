# fake-jev

Deterministic, zero-model fake server implementing a strict `jev/v1` compatibility
profile over HTTP + JSON, driven by configuration, with a control API and golden
contract vectors. It lets clients exercise provider-shaped request/response behavior
without a model, an API key, or outbound network access.

**Status: v1 is implemented here.** The `jev/v1` data plane, the `/__fake/v1/*`
control API, the CLI (`serve`, `validate`, `verify`, `run`, `version`), the
golden contract vectors, the release build matrix, the container build
(`Dockerfile`), and the examples are all present. Behavior is defined by the
specification — not by this file.

- Authoritative specification: `spec/fake-jev-technical-spec-v2.md`
- Repository rules for agents: `AGENTS.md`
- Agent workflow: `docs/development/agent-workflow.md`
- Runnable usage examples: `examples/README.md`
- Work state and evidence: `.agent/`

## What fake-jev is not

**fake-jev does not simulate model quality or intelligence.** It does not
evaluate, rank, or emulate any provider model's reasoning or accuracy; it
returns exactly the answers a fixture configures, and it is not a replacement
model or a model-quality evaluation framework.

Normal execution also performs no network egress and no model or provider call:
the server binds a local port, reads a configuration file, and answers from
that file. No API key is required — authentication headers are accepted and
ignored (`Authorization: Bearer <anything>`), and they are never logged.

Several things are deliberately absent from v1: auto decision modes, named
scenario/state-machine DSLs beyond response sequences, plugin ABIs, persistent
state, dashboards, control-plane authentication, record/replay against real
providers, provider OpenAPI ingestion, provider-quality evaluation, and generic
HTTP mocking.

## Install

Requires no Go toolchain, model, or provider SDK to *run*. Go 1.26+ is needed
only to build from source.

```bash
# From source.
go build -o fake-jev ./cmd/fake-jev

# Or, when a release exists, download the published binary.
# `.github/workflows/release.yml` cross-compiles the platform matrix below on a
# `v*` tag and attaches SHA256SUMS:
#   fake-jev-linux-amd64, fake-jev-linux-arm64, fake-jev-darwin-amd64,
#   fake-jev-darwin-arm64, fake-jev-windows-amd64.exe
#   gh release download <tag> && sha256sum --check SHA256SUMS

# Or build and run the image from the committed `Dockerfile` (static binary,
# non-root, no Go inside).
docker build -t fake-jev:local .
docker run --rm -p 8787:8787 fake-jev:local
```

The image's default command is `serve --host 0.0.0.0 --port 8787`: the binary's
own default bind stays `127.0.0.1`, and only the container opts into a
container-reachable address explicitly. Mount a config to serve fixtures:

```bash
docker run --rm -p 8787:8787 -v "$PWD/examples/fake-jev.yaml:/etc/fake-jev.yaml" \
  fake-jev:local serve --config /etc/fake-jev.yaml --host 0.0.0.0
```

## Quick start

```bash
# 1. Validate a configuration without starting a server.
./fake-jev validate examples/fake-jev.yaml

# 2. Serve it. Without --config the built-in defaults are used.
./fake-jev serve --config examples/fake-jev.yaml --port 8787 \
  --ready-file /tmp/fake-jev-ready.json

# 3. Point a client at http://127.0.0.1:8787 and make normal calls.
curl --fail-with-body http://127.0.0.1:8787/v1/models

# 4. Ask the server whether the exchanges matched the fixture.
./fake-jev verify --url http://127.0.0.1:8787
```

The same four steps are worked through for TypeScript, Python, Go, and raw
HTTP in `examples/README.md`, each with its own response checks.

## CLI

```text
fake-jev version                          # version, build, profiles, API versions
fake-jev serve [flags]                    # start the local fake server
fake-jev validate <path>                  # validate a config, start no server
fake-jev verify --url <url>               # call the server verify operation
fake-jev run [flags] -- <command>         # run a command against a temporary server
```

`serve` flags: `--host`, `--port` (0 requests an ephemeral port), `--config`,
`--compat` (repeatable), `--mode strict`, `--log-format text|json`,
`--ready-file`. `--log-format` accepts `text` or `json` and rejects any other
value; both accepted values currently produce the same text output, since
distinct JSON log output is not implemented yet. With `--ready-file`, the file
is written atomically once the listener and control API are ready, and removed
on clean shutdown:

```json
{"url": "http://127.0.0.1:8787", "host": "127.0.0.1", "port": 8787,
 "pid": 1234, "controlApiVersion": "v1"}
```

`run` starts a server on an ephemeral loopback port, exports `FAKE_JEV_URL`
(plus `--url-env NAME` when given), runs the command after `--`, verifies the
exchanges, and shuts the server down:

```bash
./fake-jev run --config examples/fake-jev.yaml --url-env JEV_BASE_URL -- go run ./examples/go
```

Exit codes: `0` success, `2` usage/configuration/startup/operational failure,
`3` verification failed.

## Security

`fake-jev` is a local development and test utility. It binds `127.0.0.1` by
default, and the `/__fake/v1/*` control API (including `verify` and `reset`) is
unauthenticated. **Do not expose the server — data plane or control API — to
untrusted networks**; a non-loopback bind such as `0.0.0.0` is an explicit
choice for container or VM use, not a default. Fixtures are data only: no
templates, expressions, shell commands, or callbacks are executed.

## Verify

```bash
./scripts/verify-candidate            # repository and product checks
./scripts/agent-context <task-id>     # print a bounded task packet
```

`verify-candidate` exits `0` only when every mandatory check passes. The
product checks are live: `go test ./...`, `go vet ./...`, `go build
./cmd/fake-jev`, plus the guard rows (`arch`, `lint`, `trace`, `fuzz`) and the
Go race row.
