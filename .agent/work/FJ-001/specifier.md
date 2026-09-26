---
stage: specifier
task: FJ-001
inputFingerprint: 260f6f151ace53875ecf51a4f702ed8e4f9e93643e25fec7fca37e13a3013feb
outputFingerprint: 260f6f151ace53875ecf51a4f702ed8e4f9e93643e25fec7fca37e13a3013feb
taskFingerprint: cc94ff6af6bccd0fa2ce59f0742e09743b6bf854d5c48af4097e7194f000ee93
gitHead: 13172c0
generatedAt: 2026-09-26T07:01:06Z
---

# Specifier — FJ-001

## Objective restated as observable behavior

Running `fake-jev version` on a locally built binary prints, on one command:

1. the product version,
2. Go build/runtime information,
3. the built-in compatibility profile IDs,
4. the control API version,
5. the config schema version.

Everything else about this item is repository structure, not user-visible: a
`go.mod` declaring a Go 1.26 minimum, a `cmd/fake-jev` entry point, and the
`internal/cli` package that owns command dispatch. No subcommand other than
`version` exists yet.

## Spec resolution (no ambiguity, no spec gap)

- §15.5 fixes the five required content items. The `--json` machine-readable
  form is explicitly MAY, so it is deferred to FJ-030/031 rather than invented here.
- §25.2 fixes the control API version as the URL-space version `v1`
  (path prefix `/__fake/v1`).
- §25.3 fixes the config schema version as `schemaVersion: 1`.
- §25.4 and the configuration section fix the v1 built-in profile set to exactly
  `jev/v1`; the input alias `jev` normalizes to it and is not a second profile.
- Product version: the specification names no release scheme, so the value is
  build metadata (ldflags-overridable, `dev` default) — private implementation
  detail, no public behavior invented.
- §42.1 exit codes apply to later subcommands; `version` is a success-only
  command and exits 0, and a usage error exits 2.

## Out of scope (restates nonGoals)

No HTTP server, no stub matching, no fixtures, no Go 1.27-only language
features, no CLI framework. No YAML dependency is added in this item: the
allowed files are the module, the entry point, and two `internal/cli` files, so
dependency policy is satisfied by adding nothing at all.

## Traces

- C-CLI-009 — §15.5 version output content; §25.2 control API version; §25.3
  config schema version; §25.4 built-in profile IDs.
- C-QUAL-001 — §31 phase 0 acceptance, `go test ./...`.
- C-QUAL-003 — §31 phase 0 acceptance, `go vet ./...`.
- §17.1 toolchain baseline — `go.mod` minimum Go 1.26; CI matrix rows for 1.26.x
  and 1.27.x belong to FJ-004, not here.
- §17.2 dependency policy — standard library only in this item.
- §17 layout — `cmd/fake-jev/main.go` and `internal/cli/` created; the
  remaining package directories are created by the work items that own their
  behavior (FJ-010 onward), so no empty package is committed.

## What remains

The coder implements the four allowed files. Package boundaries beyond
`internal/cli` are deliberately not scaffolded: an empty package has no
observable behavior and would only be deleted and rewritten by FJ-010+.
