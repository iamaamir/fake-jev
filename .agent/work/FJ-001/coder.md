---
stage: coder
task: FJ-001
inputFingerprint: 260f6f151ace53875ecf51a4f702ed8e4f9e93643e25fec7fca37e13a3013feb
outputFingerprint: 4d87f2d06abe8969121119a69c0bc8f7c292249e97ece321d6dd43524da17524
taskFingerprint: cc94ff6af6bccd0fa2ce59f0742e09743b6bf854d5c48af4097e7194f000ee93
gitHead: 13172c0
generatedAt: 2026-09-26T07:02:39Z
---

# Coder — FJ-001

## What was implemented

Four files, all inside `allowedFiles`:

- `go.mod` — `module fake-jev`, `go 1.26`. No `require` block: the item adds
  zero dependencies, which satisfies §17.2 by construction rather than by
  restraint. No `go.sum` exists.
- `cmd/fake-jev/main.go` — entry point only: forwards `os.Args[1:]` plus
  `os.Stdout`/`os.Stderr` to `cli.Run` and exits with its result.
- `internal/cli/dispatch.go` — `Run`, the usage text, and the §42.1 exit codes
  (`exitOK` 0, `exitFailure` 2, `exitVerify` 3 reserved for FJ-032). Today
  `version` and `help` are the only commands; anything else is a usage error
  that exits 2 and prints usage on stderr.
- `internal/cli/version.go` — the §15.5 report. `ProductVersion` defaults to
  `dev` and is overridable with `-ldflags -X`; `ControlAPIVersion` is `v1`
  (§25.2), `ConfigSchemaVersion` is `1` (§25.3), `BuiltInProfiles()` returns
  `jev/v1` (§25.4). Go version and VCS revision come from `runtime/debug`, with
  a `runtime.Version()` fallback when build metadata is absent.

## Decisions worth review

- **No `version --json`.** §15.5 marks it MAY; it is deferred rather than
  invented, and belongs to the CLI item (FJ-030/031) if anyone wants it.
- **No `internal/` skeleton packages.** The objective mentions §17 package
  boundaries, but an empty package has no observable behavior and each would be
  rewritten wholesale by FJ-010 onward. `internal/cli` is created because this
  item puts real code in it. The boundary rule that matters now — `internal/cli`
  is the only package touching the process environment — is honored: `Run`
  takes its streams as arguments and never reads `os` itself.
- **Module path `fake-jev`.** The specification fixes the `go.mod` minimum Go
  version but not the module path, and the repository has no remote. A
  placeholder that asserts no ownership was chosen over inventing a GitHub
  path. It is a one-line change plus import updates if a canonical path is
  decided later; flagging it so the orchestrator can decide before phase 0
  closes.

## Observed behavior

```
$ fake-jev version
fake-jev dev
go: go1.26.3
build: 13172c0f73c46ed476b7e0d0b8454d8212668ea2 (modified)
profiles: jev/v1
control-api: v1
config-schema: 1
```

All five §15.5 content items are present. `-ldflags -X` moves the first line to
`fake-jev 0.1.0`; the remaining four lines are unaffected. Unknown command and
empty argument list both exit 2 with usage on stderr; `help` exits 0.

## Toolchain note (not a defect in this item)

`go test ./...` typed by hand at the repository root also walks the gitignored
tool installs `agent/` and `.agents/`, which contain an unrelated Go example and
therefore fail to resolve. `scripts/verify-candidate` already excludes those
directories from package discovery (`NON_REPO_DIRS`), so the acceptance checks
run against the real repository packages: `go test`, `go vet`, and
`go build ./cmd/fake-jev` all exit 0. Recorded because the specification's
phase-0 acceptance block is written as bare `./...` commands and a reviewer
running them by hand will see the tool-install contamination.
