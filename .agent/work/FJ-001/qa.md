---
stage: qa
task: FJ-001
inputFingerprint: 7a30b543583fc4e4849e2b615c5256f499e0d4b92d555bc81dd0166e02860745
outputFingerprint: 7a30b543583fc4e4849e2b615c5256f499e0d4b92d555bc81dd0166e02860745
taskFingerprint: cc94ff6af6bccd0fa2ce59f0742e09743b6bf854d5c48af4097e7194f000ee93
gitHead: 13172c0
generatedAt: 2026-09-26T07:13:55Z
---

# QA — FJ-001

## What was exercised

The public-surface harness (FJ-046) is not implemented; this item is under the
recorded bootstrap exemption. Because the item adds a command-line surface, it
was exercised by running the built binary rather than by leaving it unverified.

| Check | Result |
|---|---|
| `go test` over repository packages | exit 0 (no test files yet) |
| `go vet` over repository packages | exit 0 |
| `go build ./cmd/fake-jev` | exit 0 |
| `gofmt -l cmd internal` | clean |
| `fake-jev version` | all five §15.5 items present |
| `fake-jev version` with `-ldflags -X` | product version overridden, other four lines unchanged |
| `fake-jev help` | exit 0, usage on stdout |
| `fake-jev bogus` | exit 2, error + usage on stderr |
| `fake-jev` (no args) | exit 2, usage on stderr |
| `fake-jev version extra` | exit 2, usage on stderr |
| `go.mod` contents | `go 1.26`, no `require`; no `go.sum` exists |

## Actual output

```
$ fake-jev version
fake-jev dev
go: go1.26.3
build: 13172c0f73c46ed476b7e0d0b8454d8212668ea2 (modified)
profiles: jev/v1
control-api: v1
config-schema: 1
```

## Traces confirmed

- **C-CLI-009** — product version (line 1), Go build/runtime version (lines 2–3),
  built-in profile IDs (`jev/v1`, line 4), control API version (`v1`, line 5),
  config schema version (`1`, line 6). Every item §15.5 names is present.
- **C-QUAL-001** — `go test` exits 0. No test files exist; that satisfies the
  criterion for this item but is not evidence of behavior, which is why the
  table above runs the binary directly.
- **C-QUAL-003** — `go vet` exits 0.
- **§17.1** — `go.mod` declares `go 1.26`; the source uses no language feature
  newer than 1.26. The CI matrix rows for 1.26.x and 1.27.x are FJ-004's scope
  and are not delivered here.
- **§17.2** — zero dependencies; no router, DI, logging, configuration, or test
  framework, and no CLI framework.
- **§42.1** — usage failures exit 2.

## Not covered here, and why it is not a gap

No unit or system test file is added, because `allowedFiles` does not contain a
test path. A `version_test.go` is worth adding and belongs to whichever item
first introduces a test convention — FJ-046 owns the harness, and FJ-030/031
own the remaining subcommands. Recorded so the absence is a decision with an
owner rather than an oversight.

## Residual risk

- The product version string is build metadata, so a release build that forgets
  the `-ldflags -X` override ships as `dev`. FJ-042 owns the release build
  matrix and is the right place to assert the injected value.
- The bare `go test ./...` written in the §31 phase-0 acceptance block also
  walks the gitignored tool-install directories `agent/` and `.agents/`, which
  contain an unrelated Go example. The verifier excludes them, so the recorded
  acceptance is green; a human running the block verbatim will see the
  contamination. Tracked in the coder artifact; it needs a decision from the
  orchestrator, not a code change here.

## Re-evidenced 2026-09-26T07:13:55Z

FJ-048 added `/fake-jev` to `.gitignore`, which changes the workspace candidate
fingerprint and therefore invalidates this item's earlier evidence. The stage was
re-run: the four Go files are byte-identical to the original pass, so the findings
above stand unchanged, and `go test`, `go vet`, and `go build ./cmd/fake-jev` were
re-executed against them. Only the recorded fingerprints differ.
