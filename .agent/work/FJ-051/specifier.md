---
stage: specifier
task: FJ-051
inputFingerprint: 80852a3b20015001679e3171fe0271619845cf02203b70f2cb59470b976af21c
outputFingerprint: 80852a3b20015001679e3171fe0271619845cf02203b70f2cb59470b976af21c
taskFingerprint: 7b1a0bba7ef9fad95e466829ca9beb13feda21a06662a1b0b4512f68272be131
gitHead: 53463e2
generatedAt: 2026-10-03T07:22:35Z
author: worker/FJ-051-specifier
---

# Specifier — FJ-051 (Phase 4: LICENSE repository artifact)

Objective restated as observables: a `LICENSE` file exists at the repository
root completing the §17 v1 layout entry; its bytes are the canonical SPDX MIT
text with the pinned copyright line `Copyright (c) 2026 iamaamir`; the file is
valid UTF-8, LF-terminated, carries no trailing whitespace, and has exactly the
pinned line count; no source file gains a license header; `package.json`
continues to declare `MIT`; and `./scripts/verify-candidate FJ-051` still exits
`0`. No verdict is expressed here; results come only from
`./scripts/verify-candidate FJ-051`.

## Traces

- C-QUAL-007
- C-QUAL-001
- C-QUAL-003

Interpretation of the traces: C-QUAL-007 is the gate that must keep passing
(`./scripts/verify-candidate` runs the product Go checks under a pinned module
environment); C-QUAL-001 (`go test ./...`) and C-QUAL-003 (`go vet ./...`) are
the product rows the "no source file gains a license header" criterion is
intended to leave untouched. The §17 repository-layout tree is SHOULD-level and
has no acceptance-catalog ID of its own, so the layout-completion criterion
traces to §17 directly (see criterion L1). No catalog ID is claimed that does
not exist in `docs/development/acceptance-catalog.md`.

## Fixed inputs (owner-fixed, not re-litigated here)

- **License identity is MIT** — owner decision recorded 2026-10-03 in
  `.agent/work/FJ-051/state.json` and in the task packet. The second-opinion
  record (system_one, MIT 0.98) is in the same notes.
- **The artifact body is fixed verbatim.** The canonical SPDX MIT text with the
  copyright line `Copyright (c) 2026 iamaamir` is the pinned contract. The
  coder must not reflow, re-wrap, add a title-header variant, add a trademark or
  no-endorsement sentence, append a warranty paragraph, or otherwise author a
  variant: any deviation produces a custom license rather than SPDX MIT. The
  byte digest in L2 is the arbiter; it is derived from exactly that fixed text.
- **Holder string** `iamaamir` is the repository owner (matches the git author
  and the `github.com/iamaamir/fake-jev` remote). A different legal name is a
  one-line change to `LICENSE` plus a re-run of the stage chain; it is not a
  question this stage reopens.

## Verification environment

All commands run from the repository root at candidate time. Digest commands
use `sha256sum`; on macOS without GNU coreutils the identical digest is
`shasum -a 256 LICENSE` (compare only the first field). Expected outputs below
are literal.

## L1 — LICENSE exists at the repository root matching the §17 layout entry

- Observable: a regular file named exactly `LICENSE` sits at the repository
  root, alongside the other §17 root entries `Dockerfile`, `go.mod`, and
  `README.md`; it is not a directory, symlink-to-directory, or nested file.
- Spec clause: §17 v1 repository layout (the tree lists a root `LICENSE`
  entry as SHOULD-level).
- Deterministic verification:

  ```bash
  test -f LICENSE && test ! -d LICENSE && echo license-file-present
  ls -1 Dockerfile go.mod LICENSE README.md
  ```

- Expected output: `license-file-present`, then the four entries in sorted
  order:

  ```text
  Dockerfile
  LICENSE
  go.mod
  README.md
  ```

## L2 — LICENSE bytes equal the canonical SPDX MIT text with the pinned copyright line

- Observable: `sha256sum LICENSE` equals the pinned digest of the canonical
  SPDX MIT text (fixed body above, trailing newline included); line 1 is
  `MIT License`; line 3 is the pinned copyright line.
- Spec clause: §17 (layout entry completed with an explicit license); the
  license identity and text are owner-fixed (Spec-silent observations).
- Deterministic verification:

  ```bash
  sha256sum LICENSE
  sed -n '1p'  LICENSE
  sed -n '3p'  LICENSE
  ```

- Expected output:

  ```text
  ea8b2b5f03b197b3b67aaf8af8a4dd663ab0384e71bba8d8a28751af931dadaa  LICENSE
  MIT License
  Copyright (c) 2026 iamaamir
  ```

  On macOS use `shasum -a 256 LICENSE` and compare the first field only.

## L3 — encoding and line discipline: UTF-8, LF only, single trailing newline, no trailing whitespace, pinned line count

- Observable: the file decodes as UTF-8; contains no carriage returns; the last
  byte is a single `\n` (not `\n\n`); no line ends in a space or tab; the file
  has exactly 21 lines and 1065 bytes.
- Spec clause: §17 layout entry; UTF-8/LF discipline is the repository's text
  convention (Spec-silent observations).
- Deterministic verification:

  ```bash
  python3 - <<'PY'
  b = open('LICENSE', 'rb').read()
  b.decode('utf-8')                      # raises on invalid UTF-8
  assert b.endswith(b'\n') and not b.endswith(b'\n\n'), 'bad terminator'
  assert b'\r' not in b, 'CR present'
  print(len(b), b.count(b'\n'))
  PY
  grep -nE '[[:space:]]+$' LICENSE       # trailing-whitespace lines; expect none
  wc -l < LICENSE
  ```

- Expected output: `1065 21` from the Python snippet; the `grep` prints no
  lines and exits `1`; `wc -l` prints `21`. (The `grep` exit status `1` is the
  expected "no match" result, not a failure. `[[:space:]]+$` is used rather
  than a `\t` escape so the pattern is identical under GNU and BSD `grep`.)

## L4 — no source file gains a license header

- Observable: the candidate diff adds no license header or SPDX identifier to
  any Go (or other) source file. The only non-evidence change is the new root
  `LICENSE`; `.agent/work/FJ-051/**` is evidence and is excluded from the
  candidate fingerprint.
- Spec clause: work-item non-goal ("no license headers added to source files");
  §17/§5.
- Deterministic verification:

  ```bash
  git diff --name-only 53463e2 -- '*.go'                       # expect no output
  git diff --name-only 53463e2 -- ':(exclude).agent' ':(exclude)LICENSE'   # expect no output
  grep -rIn --include='*.go' 'SPDX-License-Identifier' .       # expect no output (exit 1)
  git status --porcelain
  ```

- Expected output: both `git diff --name-only` invocations print nothing; the
  `grep` prints no lines and exits `1`; `git status --porcelain` shows the new
  `LICENSE` plus only evidence paths under `.agent/work/FJ-051/`. In particular
  no `internal/**`, `cmd/**`, `scripts/**`, `spec/**`, `test/**`, or
  `testdata/**` path appears.

## L5 — package.json continues to declare MIT consistently

- Observable: `package.json`'s `license` field still reads `MIT`, and
  `package.json` is byte-unchanged relative to the base revision.
- Spec clause: consistency with the owner-fixed MIT choice; `package.json` is
  deliberately outside this item's `allowedFiles` and is not edited.
- Deterministic verification:

  ```bash
  python3 -c "import json; print(json.load(open('package.json'))['license'])"
  git diff --name-only 53463e2 -- package.json    # expect no output
  ```

- Expected output: `MIT`, then no output from `git diff`.

## L6 — the repository gates still pass ./scripts/verify-candidate

- Observable: the verifier exits `0` for the candidate revision, and its
  fingerprint-bound report lands under `.agent/reports/FJ-051/`.
- Spec clause: §31/§23 (quality gates); work-item acceptance
  ("`./scripts/verify-candidate` exits 0"); C-QUAL-007, C-QUAL-001, C-QUAL-003.
- Deterministic verification:

  ```bash
  ./scripts/verify-candidate FJ-051
  ```

- Expected output: exit status `0`. The verifier writes its report under
  `.agent/reports/FJ-051/`; that report is the verification evidence, not this
  artifact.

## Out of scope for this item

- **Recorded acceptance criterion 4 — the selftest scenario asserting `LICENSE`
  classifies as metadata — is OUT OF SCOPE here and is owned by the follow-up
  ticket FJ-063.** Reason: that scenario lives in `scripts/selftest`, which is
  outside this item's `allowedFiles` (exactly `[LICENSE]`); the split was
  recorded 2026-10-03 in `.agent/work/FJ-051/state.json` rather than widening
  `allowedFiles`. Criteria 1, 2, 3 and 5 of the recorded acceptance are covered
  here by L1, L2/L3, and L6 respectively. The criterion-4 split is stated here
  so it is visible in the stage record.
- No license headers in source files (non-goal; L4 observes their absence).
- No `README.md` changes (non-goal).
- No `package.json` changes (outside `allowedFiles`; L5 observes consistency).
- No `scripts/**` changes: the candidate-fingerprint `METADATA_FILES` entry for
  `LICENSE` was added centrally when this item was opened and is not part of the
  candidate diff.

## Spec-silent observations

- **The §17 layout tree is SHOULD-level.** §17 says the v1 layout "SHOULD start
  as given"; the `LICENSE` gap is layout completion plus release hygiene, not a
  MUST. Nothing in the specification requires a license file to exist.
- **The specification names no SPDX identifier and no license text.** The MIT
  identity is an owner decision (2026-10-03), not a spec clause; the SPDX-MIT
  body is therefore fixed by the owner/task packet, and L2's digest is the
  authoritative pin.
- **The `MIT License` title line.** SPDX's short identifier does not itself
  mandate a title line; the fixed artifact begins with `MIT License`, so the
  pin includes it. This is part of the fixed body, not a new decision.
- **No required byte encoding is stated in the spec.** UTF-8 with LF endings and
  a single trailing newline is the repository text convention; L3 asserts it
  because byte-for-byte equality with the canonical text (L2) already implies
  it.
- **No spec clause governs the copyright holder string.** `iamaamir` is the
  repository owner; changing it is a one-line `LICENSE` edit plus a stage-chain
  re-run, not a spec question.
- **Fingerprint behavior.** `.agent/**` is excluded from the candidate
  fingerprint, so writing only evidence leaves the fingerprint unchanged;
  `LICENSE` is not excluded, so the coder stage output fingerprint differs from
  the base `80852a3b…`. This stage (specifier) records
  `inputFingerprint == outputFingerprint == 80852a3b…`. Product/metadata
  classification governs task fingerprints and stage policy, not candidate
  hashing.
- **`package.json` `license` is not enforced by the spec.** Its `MIT` value
  predates this item (FJ-042 npm stub); L5 observes consistency rather than
  requiring an edit.

## Non-goals (respected, not re-litigated)

- No new public behavior, route, flag, config key, or CLI surface.
- No dependency, module, or build changes: `LICENSE` is inert data.
- No change to any file outside `allowedFiles` (exactly `[LICENSE]`) other than
  evidence under `.agent/work/FJ-051/**`.
- No license variant, no additional license file, no `NOTICE`/`COPYING`/
  `licenses/` tree.
