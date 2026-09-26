---
stage: coder
task: FJ-048
inputFingerprint: 4d87f2d06abe8969121119a69c0bc8f7c292249e97ece321d6dd43524da17524
outputFingerprint: 7a30b543583fc4e4849e2b615c5256f499e0d4b92d555bc81dd0166e02860745
taskFingerprint: 6e08706ef6fdac9c4cbbde26998da0980ce56477778ee5bc59bc24eb2fbe3e86
gitHead: 751185b
generatedAt: 2026-09-26T07:13:38Z
---

# Coder — FJ-048

## What was implemented

One line, in the existing "Build output" block of `.gitignore`:

```gitignore
/fake-jev
```

Anchored with a leading slash so it matches the repository-root binary that
`go build ./cmd/fake-jev` writes and nothing else. It sits beside `/bin/` and
`/dist/`, which were added for the same phase.

## What was deliberately not done

- **No blanket pattern.** `/bin/` is not a catch-all for Go output, and adding
  `*.exe` or a root `fake-jev*` wildcard would also hide a source directory
  someone might legitimately add later. The defect is one path; the fix is that
  path.
- **No change to `scripts/verify-candidate`.** The build check is correct
  behaviour, not the bug. It stays exactly as it was.

## Verification of the change itself

```
$ git check-ignore -v fake-jev
.gitignore:23:/fake-jev	fake-jev
$ git status --porcelain
 M .gitignore
```

The binary is now reported as ignored and no longer appears as untracked. The
behavioral proof — two consecutive verification runs — is the work item's own
acceptance line and is exercised after the remaining stages are recorded.

## Note on blast radius

Adding a line to `.gitignore` changes the candidate fingerprint, which is
correct by design: the workspace genuinely changed. It therefore stales the
stage evidence recorded for FJ-001, whose last-stage output fingerprint was
bound to the previous workspace. FJ-001 is re-evidenced against the new
fingerprint as part of this item's acceptance; the underlying FJ-001 code is
untouched.
