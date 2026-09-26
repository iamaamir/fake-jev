---
stage: cleaner
task: FJ-053
inputFingerprint: 95cbb205f47378b98e9835d0c3807004ceb3281a6dc81f75de87f581974171e4
outputFingerprint: 95cbb205f47378b98e9835d0c3807004ceb3281a6dc81f75de87f581974171e4
taskFingerprint: 83d86d767a7234b1ae2b171b0373044013e2c8821aa922631af2baeb80be6ba4
gitHead: 20a2c96
generatedAt: 2026-09-26T11:44:31Z
---

# Cleaner — FJ-053

## Gate G-L: bootstrap exemption

`docs/agents/roles/cleaner.md` states that G-L "runs complexity/CRAP analysis
when the tooling exists; until then the gate reports
`stage.tooling_bootstrap_exempt`". No such analyzer exists in
`scripts/`, so the G-L tool requirement is waived by
`.agent/gate-policy.json`:

```json
{ "gate": "G-L", "blocks": ["stage.tooling_absent"],
  "scope": ["product", "metadata"],
  "code": "stage.tooling_bootstrap_exempt",
  "reason": "complexity/CRAP analyzer not implemented yet",
  "trackedBy": "FJ-044" }
```

The exemption is pre-existing and tracked by FJ-044; this stage neither creates
nor consumes it. The analysis below was therefore performed **by reading** —
of `go.mod`, the module graph, the build/vet/test surfaces, the CI workflow and
the tracked tree — not by an automated tool.

## What was inspected

Read: `go.mod` (3 lines, whole file), `git diff go.mod`,
`.github/workflows/ci.yml` (the two Go jobs), `scripts/verify-candidate`
(package discovery, Go check dispatch), `.agent/gate-policy.json`,
`.gitignore`, `spec/fake-jev-technical-spec-v2.md` §15.6 / §17.2 / §31 Phase 0,
`.agent/work/FJ-053/specifier.md`, `.agent/work/FJ-053/coder.md`, `docs/agents/README.md`,
`docs/agents/roles/cleaner.md`, `docs/agents/expertise/staff-architect.md`.

Commands run from the repository root:

```bash
git diff go.mod                          # the whole change
git status --short                       # working-tree surface
git diff --name-only                     # files touched by the stage
ls go.sum                                # stale-artifact check
GOPROXY=off go list -m all
GOPROXY=off go list -deps ./cmd/... ./internal/... | grep '\.'
GOPROXY=off go build ./cmd/fake-jev      # exit 0
GOPROXY=off go test ./...                # exit 1 (see finding 3)
GOPROXY=off go vet ./...                 # exit 1 (see finding 3)
git grep -nE 'spf13|fsnotify|fatih/color|mapstructure|go-colorable|go-isatty|pelletier|sagikazarmark|sourcegraph|afero|spf13/cast|pflag|gotenv|go\.yaml\.in|golang\.org/x/sys|golang\.org/x/text' --
git ls-files '*.go'
go env GOTOOLCHAIN GOFLAGS GOMOD
od -c go.mod
./scripts/candidate-fingerprint candidate .
./scripts/candidate-fingerprint task .agent/work/FJ-053/state.json
```

## Resulting artifact

```text
module fake-jev

go 1.26
```

Three lines, 31 bytes, trailing newline present (`od -c` confirms), no
`require`, no `replace`, no `exclude`, no `retract`, no `go.sum`.

## Findings

Observations only. Nothing here is repaired by this stage; `go.mod` and every
other product file are out of the cleaner's edit authority for this item.

### 1. Nothing genuinely needed was removed (correctness)

`GOPROXY=off go list -m all` prints exactly one line, `fake-jev`. The module
graph is the main module and nothing else, so every removed `require` was
unreachable. `GOPROXY=off go list -deps ./cmd/... ./internal/...` yields only
the two product packages and their standard-library closure — the
`grep '\.'` filter (which selects any dotted, i.e. non-stdlib, import path)
prints nothing. The three tracked Go files are `cmd/fake-jev/main.go`,
`internal/cli/dispatch.go`, `internal/cli/version.go`; the only non-stdlib
import in the tree is `fake-jev/internal/cli` in `main.go:8`, which is the
module's own path. So no removed entry backed a live import.

### 2. Preservation: module path and go directive are byte-identical, and no `toolchain` line was owed (minimality)

`git diff go.mod` shows only deletions: `@@ -1,26 +1,3 @@`, every removed line
inside the two `require` blocks. The `module fake-jev` line and the `go 1.26`
line are untouched context, not rewrites. §31 Phase 0 requires "Go module with
minimum Go 1.26"; `go 1.26` is the modern bare-minor form and means 1.26.0 as a
floor, so it is correct as written. The base `go.mod` carried **no** `toolchain`
line, so none was dropped and none is owed — adding one would be scope creep
(the local `GOTOOLCHAIN=auto` already resolves against the declared floor the
same way it did before). The `// indirect` comment convention disappeared with
the entries it annotated; that is the correct, not accidental, coupling, and
with no `require` there is nothing for the convention to apply to.

Nothing else the build or CI depends on was removed. `.github/workflows/ci.yml`
reads the toolchain from `go.mod` via `actions/setup-go`'s `go-version-file:
go.mod` (both the `test` and `build` jobs), which reads the `go` directive —
preserved. The `GOPROXY: "off"` env the jobs set needs no `go.sum` when the
module has no external requires; the `dist/` output path and the
`./cmd/fake-jev` package path the build job targets are both untouched.

### 3. `./...` at the repo root reaches a git-ignored tool directory (pre-existing; verifier already excludes it) — observation

In *this* working tree `GOPROXY=off go test ./...` and `go vet ./...` both exit
1. Every failure is in one package,
`fake-jev/agent/skills/golang-cli/assets/examples`, with messages
`no required module provides package github.com/{fatih/color,fsnotify/fsnotify,spf13/cobra,spf13/viper}`
plus a non-existent `github.com/you/myapp/cmd`. That directory is git-ignored
(`.gitignore:11:/agent/`, confirmed by `git check-ignore -v`; `git ls-files
'agent/'` returns nothing) and is not part of the module. This is **pre-existing
and unchanged by this stage** — the coder artifact records the same exit 1 at the
base revision, with the message differing only as `missing go.sum entry` instead
of `no required module provides package`. The failure class, the file set and
the count are identical. It matters for two reasons: a bare `go test ./...`
typed by hand in this tree is not a module-scope command, and the change
visibly *removed the requires* those ignored files were leaning on, so their
error text changed. `scripts/verify-candidate` is unaffected — it enumerates
packages through `repo_packages` and filters `NON_REPO_DIRS="agent .agents
.claude .pi .scratch"` (`scripts/verify-candidate:1039–1061`), so the verifier's
`go test`/`go vet` never see that package. Recorded as an observation about the
working tree, not a defect in this change; no product file and no script is
edited by this stage.

### 4. Deletion is minimal and fully reversible (reversibility)

`git diff --name-only` shows two modified files: `go.mod` and
`.agent/work/FJ-053/state.json` (PM-owned, not this stage's edit). The
coder touched no other product file — no `.go` source, no CI workflow, no
script, no `docs/`. The 23 deleted lines are recoverable in full from
`git show HEAD:go.mod`, and the change is a pure deletion with no reordering,
regrouping or reformatting of the surviving lines. Untracked additions in the
tree are the stage artifacts themselves (`.agent/work/FJ-053/specifier.md`,
`.agent/work/FJ-053/coder.md`, `.agent/reports/FJ-053/`) plus
`.agent/work/FJ-053/cleaner.md`; none is a product artifact.

### 5. No stale dependency artifact left behind (cleanliness)

`ls go.sum` → `No such file or directory`. No `vendor/` directory, no
`go.work`, no `go.work.sum` exists in the tree. A dependency-free module with no
`go.sum` is the canonical state, and the specifier's C3 directive and the
non-goal "do not add go.sum entries for modules that are not imported" both
call for exactly this. Nothing to tidy: the coder's tracked-files-only
`go mod tidy` run exited 0 and left `go.mod` byte-identical, so the file is
already in the state the toolchain would write.

### 6. No leftover or now-dead configuration references the removed modules (duplication)

`git grep` over the tracked tree for the sixteen removed module paths
(`spf13`, `fsnotify`, `fatih/color`, `mapstructure`, `go-colorable`,
`go-isatty`, `pelletier`, `sagikazarmark`, `sourcegraph`, `afero`, `cast`,
`pflag`, `gotenv`, `go.yaml.in`, `golang.org/x/sys`, `golang.org/x/text`)
returns matches in exactly three files, all durable historical record, none
live configuration: `.agent/work/FJ-004/coder.md:125`,
`.agent/work/FJ-004/qa.md:73–78,154,196,199`, and
`.agent/work/FJ-053/state.json:5,31`. Those are prior stage artifacts and the
PM-owned work item; per `docs/agents/issue-tracker.md` they are committed
durable history and are correctly left untouched. A second grep for
`cobra|viper|fsnotify` across the tracked tree hits the same three files and
nothing else — in particular `docs/adr/` (which holds only `README.md`),
`README.md` and `CONTEXT.md` mention no dependency or `go.mod` state, and no
ADR is silently contradicted. `.github/workflows/ci.yml` and every script are
clean. So the removal left zero dead configuration behind; the only remaining
references are narrative records of the removal itself.

### 7. Readability (altitude)

The file is at the floor of what the toolchain needs and is legible in one
glance: a module path that matches the internal import path `fake-jev/...`
used in `cmd/fake-jev/main.go:8`, and a single language-floor line. Nothing
about the remaining content is inferable only from history. The `module fake-jev`
path has no dot in its first element, which the Go toolchain permits for a main
module that is never fetched from a proxy — observed to work across
`go list -m all`, `go list -deps`, `go build`, `go vet` and `go test`. It is
also unchanged from the base revision, so it is an observation about the
module's existing identity, not a consequence of this trimming; it will need
attention only if the module is ever published to a proxy, which no current
work item calls for.

## Minimality and correctness of the applied change

The applied change is minimal: a single-file, pure-deletion diff confined to
the two `require` blocks, with both surviving lines preserved byte-for-byte
(finding 2) and nothing outside `go.mod` edited (finding 4). It is correct
against the criteria the specifier derived: the module graph is closed at one
module (finding 1), no `require` directive remains, no `go.sum` is required or
left stale (finding 5), the surviving directives are exactly what §31 Phase 0
asks for, and no committed configuration referenced the removed modules
(finding 6). `./scripts/verify-candidate` was **not** run by this stage, per the
stage brief; no pass/fail verdict is recorded here or implied by any statement
above.

## Open questions

None.
