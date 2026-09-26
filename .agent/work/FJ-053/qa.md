---
stage: qa
task: FJ-053
inputFingerprint: 95cbb205f47378b98e9835d0c3807004ceb3281a6dc81f75de87f581974171e4
outputFingerprint: 95cbb205f47378b98e9835d0c3807004ceb3281a6dc81f75de87f581974171e4
taskFingerprint: 83d86d767a7234b1ae2b171b0373044013e2c8821aa922631af2baeb80be6ba4
gitHead: 20a2c96
generatedAt: 2026-09-26T11:54:53Z
---

# QA — FJ-053

Observations only. No product file was edited by this stage. `go.mod` is the
only changed product file in the candidate and it was **not** touched here;
`state.json` is PM-owned and was not touched. `./scripts/verify-candidate` was
**not** run (stage brief and `docs/agents/roles/qa.md`). This artifact records
criteria exercised, real command output, and gaps. It contains **no verdict**;
per `docs/agents/roles/qa.md` only `verify-candidate` may emit one, and
`complete` requires that script to exit 0.

Environment actually observed, recorded because two criteria depend on it:

```text
$ go version
go version go1.26.3 darwin/arm64
$ go env GOMOD GOFLAGS GOTOOLCHAIN GOPROXY
/Users/mak/git/fake-jev/go.mod
                                    # GOFLAGS empty -> default -mod=readonly
auto
https://proxy.golang.org,direct    # ambient; every command below overrode with GOPROXY=off
```

Every Go command in this artifact was run with `GOPROXY=off` as an inline
environment prefix, as instructed. Ambient `GOPROXY` is the public proxy; the
ambient `GOPROXY=off` result therefore cannot be attributed to the machine's
proxy setting.

## Header correction

The `outputFingerprint` in this artifact's front matter was **malformed on
first write**: 57 characters instead of the required 64-hex digest, i.e. a
truncation of the candidate fingerprint, not a different value. It was a
**transcription error** by the original session, not an observation about the
candidate. Correct value, re-derived here rather than copied:

```text
$ ./scripts/candidate-fingerprint candidate .
95cbb205f47378b98e9835d0c3807004ceb3281a6dc81f75de87f581974171e4   rc=0
$ ./scripts/candidate-fingerprint task .agent/work/FJ-053/state.json
… "taskFingerprint":"83d86d767a7234b1ae2b171b0373044013e2c8821aa922631af2baeb80be6ba4"}   rc=0
```

The other six header fields were checked key-by-key against
`.agent/schema/stage-artifact.schema.json` and are correct as originally
written, and were left untouched: `stage: qa` (enum), `task: FJ-053`
(pattern), `inputFingerprint` and `taskFingerprint` (both 64-hex, and both
equal to the live values above), `gitHead: 20a2c96` (non-empty; the schema
types it as diagnostic metadata only, and every upstream stage artifact for
this item records the same value), and `generatedAt`, which is re-stamped to
the time of this rewrite. Exactly the seven required keys are present, with no
extras.

To confirm the rest of the artifact still stands, the recorded observations
were **re-run** at the same candidate fingerprint, module-scoped
(`./cmd/... ./internal/...`) because the repo root contains git-ignored tool
directories outside the module:

| Re-run observation | Scope | Result |
|---|---|---|
| `GOPROXY=off go test ./cmd/... ./internal/...` | module | **exit 0**, `?  fake-jev/cmd/fake-jev` / `?  fake-jev/internal/cli` `[no test files]` |
| `GOPROXY=off go vet ./cmd/... ./internal/...` | module | **exit 0**, no output |
| `GOPROXY=off go build ./cmd/fake-jev` | module | **exit 0**, no output, 2563026-byte binary (removed again) |
| `GOPROXY=off go list -m all` | module | **exit 0**, single line `fake-jev` |
| `ls go.sum` (and `go.work`, `go.work.sum`, `vendor`) | repo | **does not exist** |
| `GOPROXY=off go test ./...` / `go vet ./...` | repo root, verbatim | **exit 1**, failure confined to `fake-jev/agent/skills/golang-cli/assets/examples` |
| same three commands in a `git archive HEAD` copy without ignored dirs | module, no ignored dirs | **exit 0**; `go mod tidy -diff` exit 0, no diff, still no `go.sum` |
| `grep -c 'require' go.mod`; `grep -nE 'cobra\|viper\|fatih\|fsnotify' go.mod` | file | `0`; no match |

Every recorded result reproduced exactly; **nothing in the observations was
wrong and no body correction was needed**. Three incidental strings in the body
are now stale rather than inaccurate, because the candidate commit landed after
the original session ran: `git status` no longer lists `M go.mod` (it is
committed as `d3234a6`), `git rev-parse --short HEAD` is now `d3234a6` rather
than the `gitHead: 20a2c96` recorded in the header, and
`./scripts/candidate-fingerprint task …` now reports `"status":"review"`
instead of `"status":"implementing"`. None of these is a change to the
candidate fingerprint, which is unchanged. `./scripts/verify-candidate` was
still **not** run and no verdict is stated.

## Gate G-Q: bootstrap exemption

`docs/agents/roles/qa.md`: "G-Q runs public-surface/system tests when the
tooling exists; until then it reports `stage.tooling_bootstrap_exempt`". No
public-surface or system-test harness exists under `scripts/` (which holds only
`agent-context`, `candidate-fingerprint`, `selftest`, `verify-candidate`). The
exemption is recorded in `.agent/gate-policy.json`:

```json
{ "gate": "G-Q", "blocks": ["stage.tooling_absent"],
  "scope": ["product", "metadata"],
  "code": "stage.tooling_bootstrap_exempt",
  "reason": "public-surface/system test harness not implemented yet",
  "trackedBy": "FJ-046" }
```

It is pre-existing and tracked by FJ-046; this stage neither creates nor
consumes it. **The criteria below were therefore exercised directly by running
the acceptance commands, not by a G-Q harness.** Two consequences recorded for
the PM:

- C-QUAL-001 and C-QUAL-003 assert `go test ./...` and `go vet ./...` as
  catalogued. Both were run verbatim and both exit 1 in this working tree (see
  criterion 1/2). The failure is entirely inside a git-ignored tool directory
  that is not part of the module, and the module-scoped re-runs exit 0. This
  artifact does not adjudicate which reading the catalog intends; it records
  both results.
- C-QUAL-002 (`go test -race`) and C-QUAL-004 (fuzz targets) are not in this
  item's `acceptance` list and were not exercised.

## Module scope tested

The working tree contains git-ignored tool directories that are **not part of
the module** but *are* matched by a `./...` pattern at the repo root, because
the repo root is the module root:

```text
$ git check-ignore -v agent .agents .claude .pi .scratch
.gitignore:11:/agent/     agent
.gitignore:9:/.agents/     .agents
.gitignore:10:/.claude/     .claude
.gitignore:6:/.pi/          .pi
.gitignore:5:/.scratch/     .scratch
rc=0

$ git ls-files '*.go'
cmd/fake-jev/main.go
internal/cli/dispatch.go
internal/cli/version.go
```

Tracked product packages: `fake-jev/cmd/fake-jev` and
`fake-jev/internal/cli` (two packages, no `_test.go` files, so "test" below
means *resolve + compile + report no test files*).

`scripts/verify-candidate` already excludes exactly these directories, so the
verifier's Go checks are module-scoped even though its flag is not:

```text
$ grep -n 'NON_REPO_DIRS' scripts/verify-candidate
1039:NON_REPO_DIRS="agent .agents .claude .pi .scratch"
1051:    for skip in $NON_REPO_DIRS; do
1061:    echo "verify-candidate: excluded $excluded package(s) under gitignored tool dirs ($NON_REPO_DIRS)" >&2
```

**Scope of the observations below:** each acceptance command was run three
ways — (a) verbatim at the repo root, (b) module-scoped as
`./cmd/... ./internal/...`, and (c) in a tracked-files-only copy of the module
(`/tmp/fjq`, built with `git archive HEAD` plus the candidate `go.mod`, so the
repository tree was never mutated and no ignored directory was present). Only
the repo-root form sees the ignored directories. Results are labelled with
which scope produced them; nothing below silently substitutes one for another.

## Criteria exercised

| # | Criterion | Exact command | Scope | Real result |
|---|---|---|---|---|
| 1 | C-QUAL-001 `go test ./...` passes | `GOPROXY=off go test ./...` | repo root, verbatim | **exit 1** — full output below |
| 1a | C-QUAL-001, module-scoped | `GOPROXY=off go test ./cmd/... ./internal/...` | module | **exit 0** |
| 1b | C-QUAL-001, tracked files only | `GOPROXY=off go test ./...` (in `/tmp/fjq`) | module, no ignored dirs | **exit 0** |
| 2 | C-QUAL-003 `go vet ./...` passes | `GOPROXY=off go vet ./...` | repo root, verbatim | **exit 1** — full output below |
| 2a | C-QUAL-003, module-scoped | `GOPROXY=off go vet ./cmd/... ./internal/...` | module | **exit 0**, no output |
| 2b | C-QUAL-003, tracked files only | `GOPROXY=off go vet ./...` (in `/tmp/fjq`) | module, no ignored dirs | **exit 0**, no output |
| 3 | `go.mod` declares no module that no product source imports | `GOPROXY=off go list -m all`; `GOPROXY=off go list -deps ./cmd/... ./internal/...`; `grep -c 'require' go.mod` | module | `go list -m all` → `fake-jev` (exit 0). `go list -deps` → exit 0, 63 packages, `grep -cE '\.'` → **0** non-stdlib paths. `require` count → **0** |
| 4 | `go test ./...` and `go vet ./...` still pass after the change | re-ran rows 1 and 2 at the candidate revision (not inherited from any prior artifact) | both | identical results, recorded above; no pre/post divergence found |
| 5 | `go build ./cmd/fake-jev` succeeds with `GOPROXY=off` | `GOPROXY=off go build ./cmd/fake-jev` | module | **exit 0**, no output, 2563026-byte binary written |
| 6 | resulting `go.mod` contains no cobra, viper, fatih/color, or fsnotify entry | `grep -nE 'cobra\|viper\|fatih\|fsnotify' go.mod` | file | **exit 1, no output** (no match) |
| 6a | same, specifier's wider regex incl. the 12 indirects | `grep -E 'cobra\|viper\|fatih\|fsnotify\|mapstructure\|colorable\|isatty\|go-toml\|locafero\|conc\|afero\|cast\|pflag\|gotenv\|yaml\|golang.org/x/' go.mod` | file | **exit 1, no output** (no match) |

Verbatim outputs for the two non-zero repo-root commands:

```text
$ GOPROXY=off go test ./...
# fake-jev/agent/skills/golang-cli/assets/examples
agent/skills/golang-cli/assets/examples/output.go:9:2: no required module provides package github.com/fatih/color; to add it:
	go get github.com/fatih/color
# fake-jev/agent/skills/golang-cli/assets/examples
agent/skills/golang-cli/assets/examples/config.go:9:2: no required module provides package github.com/fsnotify/fsnotify; to add it:
	go get github.com/fsnotify/fsnotify
# fake-jev/agent/skills/golang-cli/assets/examples
agent/skills/golang-cli/assets/examples/args.go:6:2: no required module provides package github.com/spf13/cobra; to add it:
	go get github.com/spf13/cobra
# fake-jev/agent/skills/golang-cli/assets/examples
agent/skills/golang-cli/assets/examples/config.go:10:2: no required module provides package github.com/spf13/viper; to add it:
	go get github.com/spf13/viper
# fake-jev/agent/skills/golang-cli/assets/examples
agent/skills/golang-cli/assets/examples/exit_codes.go:7:2: no required module provides package github.com/you/myapp/cmd; to add it:
	go get github.com/you/myapp/cmd
FAIL	fake-jev/agent/skills/golang-cli/assets/examples [setup failed]
?   	fake-jev/cmd/fake-jev	[no test files]
?   	fake-jev/internal/cli	[no test files]
FAIL
rc=1

$ GOPROXY=off go vet ./...
agent/skills/golang-cli/assets/examples/output.go:9:2: no required module provides package github.com/fatih/color; to add it:
	go get github.com/fatih/color
agent/skills/golang-cli/assets/examples/config.go:9:2: no required module provides package github.com/fsnotify/fsnotify; to add it:
	go get github.com/fsnotify/fsnotify
agent/skills/golang-cli/assets/examples/args.go:6:2: no required module provides package github.com/spf13/cobra; to add it:
	go get github.com/spf13/cobra
agent/skills/golang-cli/assets/examples/config.go:10:2: no required module provides package github.com/spf13/viper; to add it:
	go get github.com/spf13/viper
agent/skills/golang-cli/assets/examples/exit_codes.go:7:2: no required module provides package github.com/you/myapp/cmd; to add it:
	go get github.com/you/myapp/cmd
rc=1
```

Every failing package is `fake-jev/agent/skills/golang-cli/assets/examples`,
which lives under `.gitignore:11:/agent/` and is returned by
`git ls-files '*.go'` as nothing. No product package appears in either failure.
`GOPROXY=off go build ./...` at the repo root fails on the same package
(exit 1) for the same reason; `GOPROXY=off go build ./cmd/... ./internal/...`
exits 0.

## Criteria — supporting observations

### C3 — the module graph is closed at the main module

```text
$ GOPROXY=off go list -m all
fake-jev
rc=0

$ GOPROXY=off go list -deps ./cmd/... ./internal/... > /tmp/deps.txt
rc=0 ; wc -l → 63 ; grep -cE '\.' → 0

$ GOPROXY=off go mod graph
fake-jev go@1.26
go@1.26 toolchain@go1.26
rc=0

$ GOPROXY=off go mod verify
all modules verified
rc=0

$ grep -c 'require' go.mod
0
```

`grep -E '\.'` is the crude "is this import path dotted, i.e. not standard
library" filter the specifier proposed; 0 of 63 dependency lines match, so no
third-party import path is reachable from either product package. Direct
confirmation by reading the tracked sources: `cmd/fake-jev/main.go` imports
`os` and `fake-jev/internal/cli`; `internal/cli/dispatch.go` imports `fmt`,
`io`; `internal/cli/version.go` imports `fmt`, `io`, `runtime`,
`runtime/debug`, `strconv`, `strings`. The only non-stdlib import in the tree
is the module's own path. `go.mod` therefore declares nothing that a product
source fails to import — and nothing at all.

### C5 — offline build, and what it rests on

```text
$ GOPROXY=off go build ./cmd/fake-jev
rc=0            # no output; ./fake-jev written, 2563026 bytes (removed again)
$ GOPROXY=off go build ./cmd/... ./internal/...
rc=0            # no output
$ GOPROXY=off go build ./cmd/fake-jev   (in /tmp/fjq)
rc=0
```

The build reached exit 0 with the proxy off and no `go.sum` because the graph
is closed; there is nothing to fetch or hash. This exercises the acceptance
line, but note what it does **not** exercise: no product test asserts anything
(§31 Phase 0 has no test files yet), and no `go.sum` hash path is reached,
because zero modules exist to hash. See gap G1.

### C6 — resulting `go.mod`

```text
$ od -c go.mod
0000000    m   o   d   u   l   e       f   a   k   e   -   j   e   v  \n
0000020  \n   g   o       1   .   2   6  \n
0000031
$ wc -lc go.mod
3 25 go.mod
$ git diff --stat -- go.mod
 go.mod | 23 -----------------------
 1 file changed, 23 deletions(-)
```

31 bytes, 3 lines, trailing newline, no `require` / `replace` / `exclude` /
`retract` / `toolchain` line. The diff is pure deletion: `module fake-jev` and
`go 1.26` are untouched context, which is the §31 Phase 0 "minimum Go 1.26"
floor preserved as the specifier required. Both grep forms exit 1 with no
output.

## Additional observations required by the stage brief

**Is there a `go.sum`?** No.

```text
$ ls -la go.sum
ls: go.sum: No such file or directory
$ ls go.work go.work.sum vendor
ls: go.work: No such file or directory
ls: go.work.sum: No such file or directory
ls: vendor: No such file or directory
```

Also never tracked: `git log --oneline --all -- go.sum` returns nothing. With
zero requires there is nothing to hash, and the non-goal forbids inventing
entries. Note the shadowing risk recorded in gap G2: a machine-local warm
`GOMODCACHE` can substitute for a `go.sum` if the go command is ever run with
`-mod=mod`, so "no go.sum" is only the correct state while the requires are
empty.

**`go list -m all`** — see C3 above: exactly one line, `fake-jev`, exit 0.

**`go mod tidy -diff` — does `go.mod` change?** At the repo root, no, because
the command never gets far enough to touch it: it exits 1 on the ignored
directory.

```text
$ GOPROXY=off go mod tidy -diff        # repo root
go: finding module for package github.com/you/myapp/cmd
go: finding module for package github.com/spf13/cobra
go: finding module for package github.com/fatih/color
go: finding module for package github.com/spf13/viper
go: finding module for package github.com/fsnotify/fsnotify
go: fake-jev/agent/skills/golang-cli/assets/examples imports
	github.com/fatih/color: cannot find module providing package github.com/fatih/color: module lookup disabled by GOPROXY=off
	... (same for fsnotify/fsnotify, spf13/cobra, spf13/viper, github.com/you/myapp/cmd)
rc=1
```

In the tracked-files-only copy, `go mod tidy -diff` exits 0 with **no diff**,
and applying `go mod tidy` leaves `go.mod` byte-identical (`diff` reports no
difference) and still produces no `go.sum`:

```text
$ GOPROXY=off go mod tidy -diff      # in /tmp/fjq
rc=0                                 # no output
$ cp go.mod /tmp/fjq-go.mod.before; GOPROXY=off go mod tidy
rc=0
$ diff /tmp/fjq-go.mod.before go.mod   # → go.mod IDENTICAL after tidy
$ ls go.sum                            # → No such file or directory
```

So within the module's real scope the toolchain has nothing to add and nothing
to remove: `go.mod` is already in the state `go mod tidy` would write. The
repo-root failure is caused by the same out-of-scope directory as criteria 1/2
and is not evidence about `go.mod`.

**Fingerprints** (the verifier binds the terminal row to the live candidate
fingerprint; re-derived, not copied):

```text
$ ./scripts/candidate-fingerprint candidate .
95cbb205f47378b98e9835d0c3807004ceb3281a6dc81f75de87f581974171e4   rc=0
$ ./scripts/candidate-fingerprint task .agent/work/FJ-053/state.json
{"class":"product","derivedRequiredStages":["specifier","coder","cleaner","hardener","qa"],
 "due":["specifier","coder","cleaner","hardener"],"pending":[],"requiredStages":["specifier","coder","cleaner","hardener","qa"],
 "source":"derived","stage":"qa","status":"implementing",
 "taskFingerprint":"83d86d767a7234b1ae2b171b0373044013e2c8821aa922631af2baeb80be6ba4"}   rc=0
$ git rev-parse --short HEAD
20a2c96
```

`candidate` = `95cbb20…171e4`, matching the values assigned to this artifact's
`inputFingerprint`/`outputFingerprint` and the value in the task brief. **No
mismatch.** `taskFingerprint` = `83d86d7…6ba4`, also matching. `gitHead` = `20a2c96`,
matching HEAD. Because this stage edits no product file, input and output
candidate fingerprints are identical, as expected.

Re-run **after** this artifact was written, to confirm the write did not move
the bound value:

```text
$ ./scripts/candidate-fingerprint candidate .     # after writing qa.md
95cbb205f47378b98e9835d0c3807004ceb3281a6dc81f75de87f581974171e4   rc=0
```

Unchanged, as expected: `.agent/work/` is agent metadata for
`scripts/candidate-fingerprint`, not a `PRODUCT_FILES` path. `gitHead` was
re-checked against `git rev-parse --short HEAD` at the same time (unchanged).

## Gaps and things not locally observable

**G1 — C-QUAL-001 and C-QUAL-003, taken literally, do not pass in this working
tree.** Both catalog entries name `go test ./...` / `go vet ./...`, and both
exit 1. The module-scoped and tracked-files-only forms of the same commands
exit 0. This is a scope property of the tree (git-ignored tool directories
inside the module root), not a property of `go.mod`; it is the same condition
the cleaner flagged and the coder recorded, and it predates the change. This
stage records it and does not resolve it, because resolving it would require
choosing between two readings of the catalog (catalog wording vs. module
scope), which is a work-item/PM decision and outside QA's authority. Raised as
an open question below, not as a `spec-gap` against the specification: §31
Phase 0's acceptance block is satisfied in module scope, and the specification
does not address ignored directories.

**G2 — the "no `go.sum`" state is enforced only by emptiness.** Locally
observable now: zero requires, no `go.sum`, `go mod tidy` idempotent. Not
locally observable, and worth the PM's attention: a warm `GOMODCACHE` on a
developer machine can stand in for a `go.sum` if the go command is run with
`-mod=mod`, in which case a `require` and a locally-minted `go.sum` appear with
no proxy contact. This is exactly the un-hashed-dependency trap the item's
objective targets, and it re-arms the moment FJ-010 adds a require. No
repository artifact (CI, verifier, `.gitignore`) pins `-mod=readonly` or
`GOPROXY=off` for local runs; the hardener records the same observation
(F5/D1). Not repairable within this item's `allowedFiles`.

**G3 — no product tests exist, so C1/C4 exercise resolution, not assertions.**
`?   fake-jev/cmd/fake-jev [no test files]` and `?   fake-jev/internal/cli
[no test files]`. `go test` exiting 0 here means "the module graph resolves and
the packages compile". No product behavior was validated by any command in this
artifact, and none is claimed.

**G4 — criteria 3 and 5 are green partly because the module is empty.** With no
external modules, `go list -m all`, `go mod verify`, `go mod graph` and
`GOPROXY=off go build` are close to tautologies. They do not demonstrate that
the toolchain *would* work with a real dependency plus a committed `go.sum`,
which is the forward-looking property in the objective. Not locally observable
without introducing a dependency, which the non-goals forbid.

**G5 — no G-Q harness.** `stage.tooling_bootstrap_exempt` (trackedBy FJ-046)
stands; no public-surface or system test was run. The golden contract vectors
under `testdata/contracts/` that the role pack names as a G-Q input are not
reachable through any test in this item's scope, because no test binary exists
yet.

**Not exercised, and why:** C-QUAL-002 (`go test -race`) and C-QUAL-004 (fuzz
targets) are not in this item's `acceptance`. Multi-platform build (FJ-004's CI
matrix) is CI-side and out of `allowedFiles`.

## Open questions

1. Should the PM confirm that C-QUAL-001 and C-QUAL-003 are to be read in module
   scope (or via `verify-candidate`, which applies `NON_REPO_DIRS`) rather than
   as literal repo-root `./...` invocations? A literal reading is exit 1 for
   both at the candidate revision. This is a scope decision, not a defect in
   `go.mod`; QA did not make it.
2. Should the un-hashed-dependency window (G2) be closed by a repository-owned
   constraint — a `-mod=readonly` / `GOPROXY=off` pin in `verify-candidate` or
   CI, or a documented FJ-010 obligation — or is it tracked elsewhere? Nothing
   in the tree currently pins it, and `scripts/` is outside this item's
   `allowedFiles`.

## Non-actions taken by this stage

- `go.mod` was not modified; `git status --porcelain` after this stage shows
  ` M .agent/work/FJ-053/state.json` (PM-owned) and ` M go.mod` (the coder's
  change) and untracked stage artifacts only.
- `.agent/work/FJ-053/state.json` was read, never written.
- `./scripts/verify-candidate` was not run.
- All scratch work (`/tmp/fjq`) was outside the repository and has been removed;
  the `fake-jev` binary produced by the criterion-5 build was deleted from the
  repository root.
- No verdict is stated or implied.
