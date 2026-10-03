---
stage: qa
task: FJ-051
inputFingerprint: 8c80ab298a252de531dbf9f61a0f67e866fdf5e091cbf8fb80d272d988305c29
outputFingerprint: 8c80ab298a252de531dbf9f61a0f67e866fdf5e091cbf8fb80d272d988305c29
taskFingerprint: 7b1a0bba7ef9fad95e466829ca9beb13feda21a06662a1b0b4512f68272be131
gitHead: 53463e2
generatedAt: 2026-10-03T07:35:31Z
author: worker/FJ-051-qa
---

# QA — FJ-051 (LICENSE repository artifact)

Fresh-session, read-only inspection of revision `53463e2` with the candidate in
place. No file outside this artifact was written by this stage: not `LICENSE`,
not production code, not repo tests, not `state.json`. Candidate fingerprint
measured at this stage: `8c80ab29…d988305c29`, equal to the coder
`outputFingerprint` and to the cleaner `inputFingerprint`/`outputFingerprint`.
No verdict or pass/fail language is expressed here; where the tooling itself
printed a label (`PASS`, `NOT APPLICABLE`, `RESULT: PASS`) it is quoted as tool
output, not adopted as an assessment.

## Per-criterion observations

### A — LICENSE exists at the repository root (§17 layout entry)

```text
$ ls -a
.agent/ .github/ cmd/ docs/ examples/ internal/ scripts/ spec/ test/ testdata/
AGENTS.md Dockerfile LICENSE README.md go.mod go.sum package.json
```

`LICENSE` is a regular tracked-path candidate at the root, 1.0K on disk.
`git ls-files --others --exclude-standard` lists it as an untracked file
(`?? LICENSE`), i.e. it is not yet committed. §17 lists a root `LICENSE` entry
as SHOULD-level; the specification names no SPDX identifier and no license
text.

### B — independent text comparison of the license body

Method used: **offline authoritative rendering, no network**. An independent
copy of the MIT text already present on this machine was used as the comparison
source:

```text
/Applications/DBeaver.app/Contents/Eclipse/licenses/external/mit.txt
```

It is a vendored third-party license file shipped by an unrelated application
(Eclipse/DBeaver), i.e. authored by neither this repository nor any stage of
this chain. Comparison performed on non-blank lines with the repository's
copyright line removed:

```text
non-blank repo(no copyright): 16   non-blank offline source: 16
byte-identical line lists: True
max repo line len: 78
```

Line-level structure of the offline file vs `LICENSE` differs only by two blank
lines (the placeholder slot where the copyright line sits, and the trailing
separator): the offline file has 18 newline-terminated lines / 1035 bytes; the
repository file has 20 + copyright line / 1065 bytes. Every non-blank text line
is byte-for-byte equal, including the `MIT License` heading line and the
`SOFTWARE.` closing line.

Cross-check against a second offline rendering
(`/Users/mak/go/pkg/mod/honnef.co/go/tools@v0.7.0/LICENSE`, a Go module-cache
copy): after removing copyright lines and the title, the token sequence is
identical (162 tokens vs 162), differing only in wrap width (that file wraps at
70, repository at 78).

Third cross-check, weaker source: the from-scratch rendering
`/private/tmp/fj051-license-independent.txt` written by the cleaner stage is
token-identical (162 vs 162 tokens). This is noted as a chain artifact, not an
independent authority.

Limitation stated plainly: the offline sources are third-party vendored copies
of the MIT text, not an SPDX-published digest, so no byte-level digest could be
compared against an external canonical file. Equality here is established at
the non-blank-line byte level and at token-sequence level against two
independent offline copies; the repository's 78-column wrap is a formatting
choice not represented in either offline copy (the Eclipse copy wraps at 78 on
those lines but lacks the copyright line and the surrounding blank-line
layout). No network fetch was used in this stage.

### C — encoding and line discipline

```text
$ python3 (byte inspection)
bytes 1065 lines 21
utf8_ok=True
bom False
cr_count 0
ends_single_lf True
line1 'MIT License'
line3 'Copyright (c) 2026 iamaamir'
last_non_empty 'SOFTWARE.'
lines_with_trailing_ws []

$ sha256sum LICENSE
ea8b2b5f03b197b3b67aaf8af8a4dd663ab0384e71bba8d8a28751af931dadaa  LICENSE

$ grep -n "[[:space:]]$" LICENSE
(no output, exit 1)

$ wc -l -c LICENSE
21 1065
```

Observations: valid UTF-8 (all-ASCII), no BOM (`4d 49 54` = `MIT` at offset 0),
zero `\r` bytes, exactly one terminating `\n`, no line ends in space or tab,
21 lines / 1065 bytes. The digest equals the pin recorded in
`.agent/work/FJ-051/specifier.md` L2 (`ea8b2b5f…1af931dadaa`).

### D — copyright line and package.json agreement

```text
line 1: MIT License
line 3: Copyright (c) 2026 iamaamir
$ python3 -c "import json; print(json.load(open('package.json'))['license'])"
MIT
```

The holder string `iamaamir` matches `git log -1 --format='%an <%ae>'` at
`53463e2` (`iamaamir <8420386+iamaamir@users.noreply.github.com>`). The two
artifacts agree on MIT. `package.json` carries no `author` field, so the holder
*name* has no npm-side counterpart; `package.json` is byte-unchanged relative
to `53463e2`.

### E — what changed relative to 53463e2

```text
$ git status --short
 M .agent/work/FJ-051/state.json
?? .agent/reports/FJ-051/
?? .agent/work/FJ-051/cleaner.md
?? .agent/work/FJ-051/coder.md
?? .agent/work/FJ-051/specifier.md
?? LICENSE

$ git diff --stat 53463e2 -- .
 .agent/work/FJ-051/state.json | 36 +++++++++++++++++++++++-------------
 1 file changed, 23 insertions(+), 13 deletions(-)

$ git diff --cached --name-only
(empty — 0 staged files)

$ git diff --name-only 53463e2 -- ':(exclude).agent'
(empty)

$ git diff --name-only 53463e2 -- '*.go'
(empty)

$ grep -rIn --include='*.go' 'SPDX-License-Identifier' .
(no output, exit 1)

$ git ls-files --others --exclude-standard
.agent/reports/FJ-051/report-20261003T072548Z.json
.agent/reports/FJ-051/report-20261003T073504Z.json
.agent/work/FJ-051/cleaner.md
.agent/work/FJ-051/coder.md
.agent/work/FJ-051/specifier.md
LICENSE
```

Exactly one non-`.agent` change exists: the new untracked `LICENSE`. The only
tracked-file diff is `.agent/work/FJ-051/state.json` (+23/-13), which is
evidence bookkeeping (status/stage/notes), not product content. Two verifier
reports under `.agent/reports/FJ-051/` are tool-generated side effects of the
mandated verifier runs (`report-20261003T072548Z.json` from the coder stage,
`report-20261003T073504Z.json` from this stage). No staged files. No
`internal/**`, `cmd/**`, `scripts/**`, `spec/**`, `test/**`, `testdata/**`,
`package.json`, `README.md`, or `Dockerfile` change. `scripts/selftest` carries
no `LICENSE` reference (`grep -n "LICENSE" scripts/selftest` → exit 1); the only
`LICENSE` mention in `scripts/` is the `METADATA_FILES` entry in
`scripts/candidate-fingerprint` (line 37), which predates this candidate diff.

### F — what the license compels of a distributor

The sentence that creates the distributor attribution obligation, quoted
verbatim from `LICENSE`:

> The above copyright notice and this permission notice shall be included in all
> copies or substantial portions of the Software.

The grant sentence, also verbatim:

> Permission is hereby granted, free of charge, to any person obtaining a copy
> of this software and associated documentation files (the "Software"), to deal
> in the Software without restriction, including without limitation the rights
> to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
> copies of the Software, and to permit persons to whom the Software is
> furnished to do so, subject to the following conditions:

Stated plainly: the grant permits commercial use, modification and forking
(`use`, `copy`, `modify`, `merge`, `publish`, `distribute`, `sublicense`,
`sell`, "without restriction"), and for private/non-distributed use the notice
condition does not trigger because it binds only copies or substantial portions
that are distributed. What it compels of a distributor is exactly the retained
notice: the copyright and permission notice must travel with copies or
substantial portions. It does **not** compel user-visible credit (no UI notice,
license plate, or marketing attribution), and it does not impose copyleft
obligations on derivative source. The artifact contains no trademark or
no-endorsement clause and no warranty beyond the all-caps disclaimer paragraph.

### G — FJ-063 selftest-scenario criterion: any claim of satisfaction in the tree?

```text
$ grep -rn "FJ-063" . --exclude-dir=.git
.agent/work/FJ-051/state.json:38: ... acceptance criterion 4 ... split into a dedicated follow-up ticket (FJ-063) ...
.agent/work/FJ-051/cleaner.md:141,206: ... owned by FJ-063 ...
.agent/work/FJ-051/specifier.md:194: ... owned by the follow-up ticket FJ-063 ...
.agent/work/FJ-051/coder.md:207: ... it was split to FJ-063 and was not implemented here ...

$ ls -d .agent/work/FJ-063
ls: .agent/work/FJ-063: No such file or directory   (exit 1)
```

Nothing in the tree claims the FJ-063 selftest-scenario criterion is already
satisfied. Every mention states the opposite: the criterion lives in
`scripts/selftest`, is outside `allowedFiles [LICENSE]`, and is deferred to a
follow-up ticket. `scripts/selftest` exists (88416 bytes) and contains no
`LICENSE` scenario. No `.agent/work/FJ-063/state.json` exists yet, so the
follow-up ticket has an audit-trail mention but no work item on disk.

### H — commands run

```text
sha256sum LICENSE                     -> ea8b2b5f03b197b3b67aaf8af8a4dd663ab0384e71bba8d8a28751af931dadaa  LICENSE   (exit 0)
grep -n "[[:space:]]$" LICENSE        -> no output, exit 1
gofmt -l .                            -> no output, exit 0
go test ./... -count=1                -> "Go test: 648 passed in 10 packages", exit 0
go vet ./...                          -> no output, exit 0
./scripts/verify-candidate FJ-051      -> RESULT: PASS, exit 0
./scripts/candidate-fingerprint candidate -> 8c80ab298a252de531dbf9f61a0f67e866fdf5e091cbf8fb80d272d988305c29, exit 0
./scripts/candidate-fingerprint task .agent/work/FJ-051/state.json
  -> {"class":"metadata","derivedRequiredStages":["specifier","coder","cleaner","qa"],
      "pending":["coder","cleaner","qa"],"requiredStages":["specifier","coder","cleaner","qa"],
      "source":"derived","stage":"specifier","status":"implementing",
      "taskFingerprint":"7b1a0bba7ef9fad95e466829ca9beb13feda21a06662a1b0b4512f68272be131"}, exit 0
```

`verify-candidate FJ-051` report: `.agent/reports/FJ-051/report-20261003T073504Z.json`.
Its own summary line and row labels, quoted as emitted:

- Row labels emitted as `PASS` (14): `candidate fingerprint computed`; `required
  repository files exist`; `scripts are executable`; `shell syntax: bash -n
  scripts/*`; `toolchain: python3 available`; `authoritative spec present and
  intact` (99559 bytes); `all .agent JSON files parse`; `work items match the
  state.json schema` (including `.agent/work/FJ-051/state.json: ok`); `complete
  items have recorded-report coverage` (25 rows, all `complete` items, FJ-051
  not among them because its status is `implementing`); `role packs match the
  role-pack contract`; `trackedBy references resolve to existing work items`;
  `trackedBy items are not complete`; `allowedFiles classify to product or
  metadata`; `requiredStages is a valid additive-only ordered set`.
- Row labels emitted as `NOT APPLICABLE` (9): `stage hardener requiredness`
  (reason printed: `metadata`); and, each with reason printed `class metadata
  (design §9.3)`, the rows `go test (repository packages)`, `go vet (repository
  packages)`, `go build ./cmd/fake-jev`, `guard arch (forbidden imports)`,
  `guard lint (err_discarded, unsafe_type_assert)`, `guard trace (active C-ID
  coverage)`, `guard fuzz (bounded crash scan)`, `go test -race (repository
  packages)`.
- No row label emitted as `FAIL`; no row label emitted as `SKIP`.
- Header line printed: `tree DIRTY (0 staged, 1 unstaged, 5 untracked) — evidence
  is bound to 53463e2 plus these changes`.

Why the `NOT APPLICABLE` rows are inert for this item: FJ-051 is classified
`metadata` (its only `allowedFiles` entry, `LICENSE`, is a `METADATA_FILES`
entry of `candidate-fingerprint`), so the verifier deliberately does not run
the Go product rows for it. Consequently the tool's green result does not by
itself exercise the Go tree; the `gofmt -l .`, `go test ./... -count=1` and
`go vet ./...` results above were run manually against the same working tree to
cover that gap.

## Residual risks

- The offline independent sources are third-party vendored copies of the MIT
  text, not an SPDX attestation. No external byte digest exists in this
  comparison, so a shared textual error in those vendored copies could in
  principle go unnoticed. The word-level agreement of three separate copies
  (Eclipse, honnef.co Go module cache, cleaner from-scratch rendering) reduces
  this.
- Wrap width (78 columns) is a formatting choice; byte digests are therefore
  not comparable to a differently wrapped canonical file. Equality rests on
  non-blank-line bytes plus token sequence.
- The holder string `iamaamir` is unverified as a legal identity beyond its
  agreement with the git author and the `github.com/iamaamir/fake-jev` remote.
- `LICENSE` is untracked (`?? LICENSE`); commit state is outside this stage's
  authority.
- `verify-candidate` treats the item as `metadata`, so its green rows do not
  cover Go product behavior; the manual Go runs are not fingerprint-bound to a
  recorded report.
- Acceptance criterion 4 remains deferred to FJ-063; no such work item exists
  on disk yet, and the verifier has no row checking it.
- `state.json` still records `"stage": "specifier"` and
  `"pending":["coder","cleaner","qa"]` although `coder.md`, `cleaner.md` and
  this artifact exist. That is stage bookkeeping, observed here, not changed by
  this stage.

## Evidence limits

All observations were made on this machine at revision `53463e2` with the
candidate working tree dirty as reported. No network access was used. This
artifact does not close the item; closure requires `./scripts/verify-candidate
FJ-051` to exit 0 for the closing revision per AGENTS.md, and the recorded
report coverage obligation attaches when the item's status becomes `complete`.
