---
stage: specifier
task: FJ-048
inputFingerprint: 4d87f2d06abe8969121119a69c0bc8f7c292249e97ece321d6dd43524da17524
outputFingerprint: 4d87f2d06abe8969121119a69c0bc8f7c292249e97ece321d6dd43524da17524
taskFingerprint: 6e08706ef6fdac9c4cbbde26998da0980ce56477778ee5bc59bc24eb2fbe3e86
gitHead: 751185b
generatedAt: 2026-09-26T07:13:23Z
---

# Specifier — FJ-048

## Observable outcome

Running `./scripts/verify-candidate FJ-001` twice in a row, with the second run
starting from a clean tree, exits 0 both times. Today the second run exits
non-zero with `stage.qa.terminal` reporting `stage.evidence_stale`.

## Cause, stated as observation

`scripts/verify-candidate` runs `go build ./cmd/fake-jev`. With no `-o`, the Go
toolchain writes the binary as `./fake-jev` at the repository root. `.gitignore`
lists `/bin/` and `/dist/` but not that path, so the file is untracked *and* not
ignored. The candidate fingerprint (design §6.1) hashes every untracked,
non-gitignored file, so the fingerprint recorded by a stage artifact no longer
matches the workspace on the next run.

## Trace

This item traces to no section of `spec/fake-jev-technical-spec-v2.md`, and
that is deliberate: it changes no product behavior, no public contract, and no
command-line output. Its authority is the gauntlet design §6.1, which defines
what the candidate fingerprint covers. The specification's phase-0 acceptance
block does name `go build ./cmd/fake-jev` (§31), which is why the artifact has
to be cleaned up rather than tolerated.

No acceptance-catalog ID applies, so the `## Traces` section carries no catalog
IDs; G-S validates the IDs that exist under that heading and there are none.

## Scope decision

`allowedFiles` is `.gitignore` alone. `scripts/verify-candidate` is named in
`relevantFiles` but is deliberately not editable here. The alternative fix —
teaching `check_go_build` to write to a temporary path — would change the
central acceptance check, and weakening or rewriting a check to make a
symptom disappear is exactly what the repository's change discipline forbids.
The check is correct; the ignore coverage is missing.

## Out of scope

No change to the Go acceptance commands. No relocation of build output into
committed artifacts. No addition of a general `*.exe`/`bin/`-style blanket
pattern: the item ignores the one path the acceptance check actually writes.
