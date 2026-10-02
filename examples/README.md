# Examples

Four runnable examples that talk to a locally running `fake-jev` server over
plain HTTP: TypeScript, Python, Go, and raw `curl`. Each one

1. calls `GET /v1/models`,
2. calls `POST /v1/systemone` with one `choice` question and one `noul`
   question, and
3. calls `GET /__fake/v1/verify` (the same control call `fake-jev verify`
   performs) and prints the result.

They all share the fixture in `fake-jev.yaml`, which answers `route` with
`backend` (confidence `0.91`, probabilities `frontend 0.06 / backend 0.91 /
infra 0.03`) and `urgent` with `noul 0.94`.

## Nothing to install

The examples add no dependency to this repository and add no dependency of
their own:

- TypeScript: Node's built-in `fetch` and its built-in type stripping; no
  `npm install`, no `node_modules`, no `package.json`, no `tsc`.
- Python: standard library only (`urllib`, `json`); no `pip`, no virtualenv.
- Go: standard library only, compiled as part of the root module; no `go.mod`
  of its own and no third-party import.
- Raw HTTP: `curl` plus `jq` from the environment; no language package.

None of them needs an API key, a credential, a model, a provider SDK, or any
network access beyond the loopback server. The `Authorization: Bearer
not-a-real-key` header they send exists only to show that fake-jev accepts any
bearer value and never logs it (spec §39.5).

## Start the server

`fake-jev run` starts the server, exports `FAKE_JEV_URL` to the child, runs
the child, and verifies the exchanges afterwards — no port juggling:

```bash
go build -o fake-jev ./cmd/fake-jev
./fake-jev run --config examples/fake-jev.yaml -- node examples/typescript/systemone.ts
```

The exit status follows the child: `0` when the child succeeded and
verification passed, `3` when the child succeeded and verification failed, and
the child's own nonzero code when the child failed.

Alternatively, start a server yourself and point the examples at it. The
examples read `FAKE_JEV_URL` and default to `http://127.0.0.1:8787`:

```bash
./fake-jev serve --config examples/fake-jev.yaml --port 8787 --ready-file /tmp/fake-jev-ready.json &
export FAKE_JEV_URL=http://127.0.0.1:8787
cat /tmp/fake-jev-ready.json              # written only once the server is ready
./fake-jev verify --url "$FAKE_JEV_URL"   # the CLI form of the third call
```

Stop the server with `kill %1` when you are done.

## Run one example

```bash
# TypeScript — Node strips the type annotations, nothing is compiled first.
# Node 23.6+ runs .ts files directly (verified on v24.14.0); older Node
# needs --experimental-strip-types.
node examples/typescript/systemone.ts

# Python (3.10+; verified on 3.11)
python3 examples/python/systemone.py

# Go (the toolchain go.mod requires)
go run ./examples/go

# Raw HTTP
bash examples/raw-http/systemone.sh
```

Each example prints the three responses it received and then either
`OK: the fixture answers matched the expected values` (exit `0`) or one
`FAIL: ...` line per mismatch (exit `1`).

## What the examples deliberately do not do

They are thin HTTP clients. They do not match requests, generate responses,
manage sequences, or reimplement the server, and they add no public surface:
a usage example is not a wrapper package. The matching, the answer generation,
and the verification decision all stay in the server.
