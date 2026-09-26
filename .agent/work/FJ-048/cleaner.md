---
stage: cleaner
task: FJ-048
inputFingerprint: 4d87f2d06abe8969121119a69c0bc8f7c292249e97ece321d6dd43524da17524
outputFingerprint: 7a30b543583fc4e4849e2b615c5256f499e0d4b92d555bc81dd0166e02860745
taskFingerprint: 6e08706ef6fdac9c4cbbde26998da0980ce56477778ee5bc59bc24eb2fbe3e86
gitHead: 751185b
generatedAt: 2026-09-26T07:13:44Z
---

# Cleaner — FJ-048

## Review

The change is a single anchored pattern in an existing block. Reviewed for
placement, specificity, and side effects on the gauntlet's own file discovery.
No further change was warranted.

- **Placement.** The pattern joins `/bin/` and `/dist/` under the comment
  "Build output (added when the Go implementation lands)", so the file stays
  readable as a whole: a reader looking for Go build output finds all three
  rules together rather than hunting.
- **Specificity.** `/fake-jev` is anchored, so it matches the root binary and
  not a hypothetical `nested/fake-jev` source directory. The gauntlet already
  treats directories as significant (`NON_REPO_DIRS` enumerates them by name), so
  an unanchored rule here would have been a real hazard, not a style point.
- **Side effects on discovery.** `scripts/candidate-fingerprint` excludes
  gitignored paths automatically, so this rule removes the binary from hashing
  without any change to the exclusion logic. `scripts/verify-candidate` discovers
  packages with `go list ./...` and then filters by `NON_REPO_DIRS`; the go
  toolchain does not consult `.gitignore`, so package discovery is unaffected and
  no package can be silently dropped by this rule.
- **What was not touched.** `scripts/verify-candidate` is byte-identical, which
  is the second acceptance line of this work item.

## Complexity

Not applicable in the usual sense: the class is `metadata`, no executable
product code was added, and the configured analyzer is absent under a recorded
bootstrap exemption. There is no function here to measure — a one-line ignore
rule has no branching, no duplication, and no abstraction surface.
