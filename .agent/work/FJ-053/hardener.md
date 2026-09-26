---
stage: hardener
task: FJ-053
inputFingerprint: 95cbb205f47378b98e9835d0c3807004ceb3281a6dc81f75de87f581974171e4
outputFingerprint: 95cbb205f47378b98e9835d0c3807004ceb3281a6dc81f75de87f581974171e4
taskFingerprint: 83d86d767a7234b1ae2b171b0373044013e2c8821aa922631af2baeb80be6ba4
gitHead: 20a2c96
generatedAt: 2026-09-26T11:50:09Z
---

# Hardener — FJ-053

Observations only. No product file was edited by this stage; `go.mod` and every
other product file are outside the hardener's edit authority for this item, and
nothing below is repaired here. No pass/fail verdict is recorded or implied.

## Gate G-H: bootstrap exemption

`docs/agents/roles/hardener.md` states that G-H "runs mutation/hardening analysis
when the tooling exists; until then it reports `stage.tooling_bootstrap_exempt`".
No mutation or hardening analyzer exists under `scripts/` (which contains only
`agent-context`, `candidate-fingerprint`, `selftest`, `verify-candidate`), so the
G-H tool requirement is waived by `.agent/gate-policy.json`:

```json
{ "gate": "G-H", "blocks": ["stage.tooling_absent"],
  "scope": ["product"],
  "code": "stage.tooling_bootstrap_exempt",
  "reason": "mutation/hardening tooling not implemented yet",
  "trackedBy": "FJ-045" }
```

The exemption is pre-existing and tracked by FJ-045; this stage neither creates
nor consumes it. The analysis below was therefore performed **by reading and by
running the toolchain against scratch copies of the tree** — never by an
automated mutation/hardening tool, because none exists.

Because FJ-053 changes a dependency-supply-chain artifact rather than executable
logic, the hardening surface is the module graph, not request handling,
concurrency or resource lifetime. `docs/agents/expertise/golang.md` is advisory
and contributes no requirement here; the fail-closed property comes from
`AGENTS.md` ("Strict behavior must fail closed", "Normal operation must never
require a model or provider network call") and from the acceptance line
"`go build ./cmd/fake-jev` still succeeds with `GOPROXY=off`". The security pack
was read on demand and is likewise advisory.

## What was inspected

Read: `go.mod` (whole file, 3 lines / 31 bytes), `git diff go.mod`,
`.github/workflows/ci.yml` (both Go jobs), `scripts/verify-candidate`
(`repo_packages`, `check_go_test`, `check_go_vet`, `check_go_build`,
`NON_REPO_DIRS`, lines 1035–1140), `scripts/candidate-fingerprint`
(`PRODUCT_FILES` line 34), `.agent/gate-policy.json`, `.gitignore`,
`docs/agents/README.md`, `docs/agents/roles/hardener.md`,
`docs/agents/expertise/golang.md`, `docs/agents/expertise/security.md`,
`spec/fake-jev-technical-spec-v2.md` §15.5, §15.6, §17.2, §31 Phase 0,
`.agent/work/FJ-053/{specifier,coder,cleaner}.md`,
`.agent/work/FJ-045/state.json`, `internal/cli/version.go`,
`testdata/contracts/` (searched for a `version` golden vector).

Commands run from the repository root (all read-only):

```bash
./scripts/agent-context FJ-053
./scripts/candidate-fingerprint candidate .
./scripts/candidate-fingerprint task .agent/work/FJ-053/state.json
git rev-parse --short HEAD
git status --short
git diff go.mod
od -c go.mod
ls go.sum
git log --oneline -- go.mod
git log --oneline --all -- go.sum          # never tracked
git ls-files '*.go'
grep -n 'GOFLAGS\|GOPROXY\|GOTOOLCHAIN\|go mod\|go\.sum\|GOSUMDB' scripts/verify-candidate
go env GOTOOLCHAIN GOFLAGS GOPROXY GOSUMDB GOPRIVATE GONOSUMDB GONOSUMCHECK GOMODCACHE GOWORK
GOPROXY=off go list -m all
GOPROXY=off go list -m -json all
GOPROXY=off go mod graph
GOPROXY=off go mod verify
GOPROXY=off go mod tidy -diff
GOPROXY=off go build ./cmd/fake-jev
GOPROXY=off go list -e -f '{{.ImportPath}} {{.Dir}}' ./...
go help environment | grep -i GONOSUMDB
```

Behavioural probes were run **only in scratch copies** under `/tmp/fjh/`, built
from `git archive HEAD` plus the candidate `go.mod`, so the repository tree was
never mutated. Each probe copied the clean tree, added a single
`internal/cli/probe.go` importing `github.com/fatih/color` (a module that is
absent from `go.mod` and from the module cache's *build* view), and varied one
environment axis at a time: `-mod` mode, `GOPROXY`, `GOMODCACHE` warmth,
`GOPRIVATE`/`GONOSUMDB`, `GOTOOLCHAIN`, `vendor/`, `go.work`, and `replace`.

## Resulting artifact

```text
module fake-jev

go 1.26
```

`od -c go.mod` confirms exactly 31 bytes ending in a newline. No `require`, no
`replace`, no `exclude`, no `retract`, no `exclude`, no `toolchain` line, and no
`go.sum` anywhere in the tree.

## Findings

Categorised observations with file/line references. None is repaired here.

### Fail-closed

**F1 — The canonical case fails closed, and it fails at module-graph
resolution, not at the network.** With a default `GOFLAGS` (empty;
`go help environment` shows the Go 1.16+ default is `-mod=readonly` for a
`go >= 1.16` module), `GOPROXY=off`, a *warm* module cache, the candidate
`go.mod`, and one undeclared import added, the exact failure is:

```text
internal/cli/probe.go:3:8: no required module provides package github.com/fatih/color; to add it:
	go get github.com/fatih/color
```

exit 1, `go.mod` byte-unchanged, no `go.sum` created. This is the right failure
for this posture: because no version is named anywhere, the go command cannot
even name a candidate to fetch, so a warm `GOMODCACHE` is irrelevant — the
`github.com/fatih/color@v1.19.0` zip *is* present in the local cache
(`$GOMODCACHE/cache/download/github.com/fatih/color/@v/v1.19.0.zip`) and is still
not used. The machine-local cache cannot rescue an undeclared import, because
resolution fails before the cache is consulted.

**F2 — The failure is identical with the proxy on.** Same command with the
default `GOPROXY=https://proxy.golang.org,direct` produces the same message and
the same exit 1. Nothing about this failure depends on the network being off;
it is a `-mod=readonly` refusal. This is stronger than a `GOPROXY=off`-only
guarantee.

**F3 — The exact conditions under which it would NOT fail.** Four, all verified,
all of which require the operator to opt in rather than something the tree does:

- **`GOFLAGS=-mod=mod`, warm `GOMODCACHE`, `GOPROXY=off`** →
  `go: finding module for package …` then the build **succeeds (exit 0)** and
  *writes* `go.mod` and a new `go.sum` from the local cache, with no network and
  no sumdb contact. This is the one real fail-open: a developer's ambient
  `GOFLAGS` or a wrapper script silently converts an undeclared import into a
  committed dependency. With a **cold** `GOMODCACHE` the same command fails
  (`module lookup disabled by GOPROXY=off`, exit 1), so the fail-open depends on
  the machine, not the tree — the worst kind of dependency for reproducibility.
- **`GOFLAGS=-mod=mod` with any reachable proxy** (the default, or
  `GOPROXY=file://$GOMODCACHE/cache/download`) → same silent success, and here
  the sumdb is bypassed for whatever `GOPRIVATE`/`GONOSUMDB` matches, so the
  written `go.sum` records whatever the proxy served. Observed: `GOPRIVATE=*`
  and `GONOSUMDB=*` both produced **byte-identical** `go.sum` files, confirming
  the private-prefix path skips the checksum database without changing the
  resulting hashes (the local cache copy was already the genuine article).
- **A `go.work` file.** `GOWORK` is empty in this repository and no `go.work`
  exists in any ancestor directory of `/Users/mak/git/fake-jev` (checked by
  walking upward to `/`). But a `go.work` in a *parent* directory is
  auto-discovered by the go command with no environment variable at all: a
  scratch tree with `go.work` containing `use (./child, …/stub/color)` built the
  undeclared `github.com/fatih/color` import at **exit 0 with `GOPROXY=off`,
  leaving `go.mod` at three lines and creating no `go.sum`**. The same is true of
  a `go.work` supplied via `GOWORK`. A workspace file is the cleanest bypass of
  the whole posture: it can supply arbitrary local code to a build that never
  touches `go.mod` and never produces a `go.sum`.
- **A committed `vendor/` directory.** With a matching `vendor/modules.txt` and a
  `require`, `go build ./...` succeeds with `GOPROXY=off` and **no `go.sum`** —
  vendoring bypasses the checksum mechanism entirely. It is not a bypass *today*
  only because the bare `go.mod` has nothing to be consistent with: adding a
  `vendor/` to the current tree fails loudly with
  `go: inconsistent vendoring … is marked as explicit in vendor/modules.txt, but
  not explicitly required in go.mod` (exit 1). That is fail-closed behaviour
  arriving as a side effect of the emptiness, not as a designed control.

**F4 — A `require` without `go.sum` fails closed on the checksum axis, which is
the trap this ticket removes.** Verified directly: `go.mod` with
`require github.com/fatih/color v1.19.0` and no `go.sum`, `GOPROXY=off`, warm
cache, default `-mod=readonly`:

```text
internal/cli/probe.go:3:8: missing go.sum entry for module providing package github.com/fatih/color (imported by fake-jev/internal/cli); to add:
	go get fake-jev/internal/cli
```

exit 1. This is the pre-change failure the coder artifact records for the
git-ignored skill directory. Post-change the same import yields the *earlier*,
resolution-level failure of F1 instead. The change moves the failure earlier in
the pipeline, which is strictly better: it fails before a version is chosen, so
there is no window in which an unhashed version is selected.

**F5 — Nothing in the repository or in `verify-candidate` pins the fail-closed
posture.** `grep` over `scripts/verify-candidate` for
`GOFLAGS|GOPROXY|GOTOOLCHAIN|go mod|go.sum|GOSUMDB` returns **zero matches**. The
three Go checks (`check_go_test` line 1072, `check_go_vet` line 1082,
`check_go_build` line 1085) run with whatever the operator's ambient
environment holds. `GOPROXY: "off"` is set only in `.github/workflows/ci.yml`
(the `test` and `build` job `env:` blocks), never in the verifier. So the
"fails closed" property of F1/F2 holds on a clean GitHub runner and is *not*
enforced locally. A developer with `GOFLAGS=-mod=mod` gets F3's silent success
under `verify-candidate` too. Recorded as an observation; `scripts/` is outside
this item's `allowedFiles`.

### Supply-chain attack surface

**S1 — What a future dependency adds, and what the current file forecloses vs
permits.** The bare `go.mod` is a *floor*, not a lockfile. It forecloses
nothing about graph manipulation because it contains no graph. Concretely, the
first real dependency (FJ-010's YAML module) would add: a `require` block, a
`go.sum` with per-module `h1:` hashes, a checksum-database contact
(`GOSUMDB=sum.golang.org` is the environment default here and is *not* pinned in
the tree), and transitive `// indirect` entries. The bare file permits all of
the risky forms:

- **`replace` — permitted, and the most dangerous form.** `go mod edit -json`
  on a copy shows `replace`, `exclude` and `retract` all parse and are all
  accepted by a file whose shape is `module` + `go` directive. `replace` is the
  directive that can defeat every other control: a filesystem `replace` to a
  local path supplies code with **no** `go.sum` entry at all, because there is
  nothing to hash. Verified: `replace github.com/fatih/color => /tmp/…` with
  `GOFLAGS=-mod=mod` and `GOPROXY=off` builds at exit 0 and the toolchain
  *itself* writes `require github.com/fatih/color v0.0.0-00010101000000-000000000000`
  into `go.mod` — a synthetic version, unresolvable, unhashed, committed. Under
  default `-mod=readonly` the same tree fails with `module … is replaced but not
  required` (exit 1), so the *default* posture holds; the exposure is the
  `-mod=mod` path again.
- **`exclude` / `retract` — permitted.** Both parse (verified via
  `go mod edit -json`). These are advisory-to-the-toolchain and low risk on a
  never-published module, but `retract` on a module path this project does not
  own is meaningless noise, and neither is currently forbidden.
- **Local policy.** There is no repository policy file constraining `go.mod`
  directives. `grep` over `docs/`, `.github/`, `scripts/`, `README.md`,
  `CONTEXT.md` for `go.sum|checksum|supply.chain|dependency` finds only
  `scripts/candidate-fingerprint` (which lists `go.sum` in `PRODUCT_FILES`,
  line 34, so its absence does not change the item's `product` class).
  `.agent/gate-policy.json` governs *agent stages*, not the module graph. A note
  forbidding `replace` (and requiring a committed `go.sum` for any future
  `require`) in `docs/development/` or `AGENTS.md` would be the natural home;
  neither is in this item's `allowedFiles`, and the specification is silent on
  module-graph directives, so recording the gap is the correct action here
  rather than writing the rule.

**S2 — The checksum database is not pinned by anything in the tree.**
`GOSUMDB=sum.golang.org` and `GOPRIVATE=`/`GONOSUMDB=` are the *environment*
defaults (`go env`), not repository state. Nothing in `go.mod`, CI, or the
verifier records or asserts them. When FJ-010 mints the first `go.sum`, whether
that hash was cross-checked against the public sumdb depends on the minting
machine's environment. The mitigation is procedural, not structural: mint the
`go.sum` in a clean environment and review it in the same commit as the
`require` — which is exactly what the FJ-053 objective anticipates
("so the first real dependency can be introduced deliberately with a committed
go.sum"). No action available to this stage.

**S3 — `GONOSUMDB` is a real Go 1.26 variable; `GONOSUMCHECK` is not.**
Recorded because the ambient-environment threat model above names both.
`go env -json` lists `GONOSUMDB` (empty) and does not list `GONOSUMCHECK` at
all; `go help environment` documents `GOPRIVATE, GONOPROXY, GONOSUMDB` together
as private-module prefix lists. Both `GOPRIVATE=*` and `GONOSUMDB=*` were
observed to suppress sumdb consultation while producing identical `go.sum`
content (F3). Anyone reasoning about this threat model should use
`GONOSUMDB`/`GOPRIVATE`; `GONOSUMCHECK` is a pre-modules fossil and setting it
has no effect.

### Reproducibility

**R1 — Two clean machines produce the same build from this tree, for a
narrower and stronger reason than before.** The module graph is closed at one
node: `go list -m all` prints exactly `fake-jev`; `go mod graph` prints
`fake-jev go@1.26` / `go@1.26 toolchain@go1.26` and nothing else; `go mod verify`
prints `all modules verified`; `go mod tidy -diff` exits 0 with no diff. With
zero external modules there is nothing to resolve, nothing to hash and nothing
to fetch, so the build inputs are exactly the tracked `.go` files, the standard
library, and the toolchain. Verified: two consecutive `GOPROXY=off go build
./cmd/fake-jev` in a `git archive` copy produced byte-identical binaries
(sha256 `a4e901394b76cf5e8be600e2a8080dc197595b0f47341d38c511608966e96a1d`
both times). The determinism is now a property of the toolchain, not of the
dependency lock — which is a *stronger* guarantee than a `go.sum` provides,
because there is no longer any third-party source that could differ.

**R2 — What stops being guaranteed: third-party source identity.** Before this
change, the (broken, non-building) `go.mod` named sixteen exact module
versions. Those names were a reproducibility anchor for *which* versions were
intended. They are now gone, replaced by "nothing". The anchor is not weaker in
practice — no product source imported any of them, so no build ever resolved
them — but the *declaration* of intent is gone, and the first future `go get`
will choose whatever version the proxy serves at that moment unless the author
pins it. That is a process obligation for FJ-010, not a defect in this tree.

**R3 — The `go` directive is a floor, not a pin, and CI reads it as one.**
`go 1.26` means "at least 1.26.0". `GOTOOLCHAIN=auto` is the environment
default. `.github/workflows/ci.yml` uses `actions/setup-go`'s
`go-version-file: go.mod`, which reads exactly that directive — so the CI
toolchain floats to the newest 1.26.x at run time. A developer on `go1.26.3`
and CI on `go1.26.9` produce binaries that differ in the `go:` line of
`fake-jev version` (`internal/cli/version.go:48`, `goVersion()` reading
`debug.ReadBuildInfo()`), which §15.5 requires to report "Go build/runtime
version or build metadata". Nothing in `testdata/contracts/` asserts a `version`
golden vector (the only CLI vector is `01-run-exit-precedence.json`, which
covers §15.4, not §15.5), so this is not currently a contract break — but it is
a real byte-level reproducibility limit that a `toolchain` line would close.
**Deferred, deliberately**: adding `toolchain go1.26.x` is outside this item's
objective and outside `allowedFiles` semantics, and §31 Phase 0 asks for a
*minimum*. Recorded so a future item can decide it consciously.

**R4 — Toolchain floor fails closed, and it is not a network dependency.**
Verified by raising a copy's directive to `go 1.99`:
- `GOTOOLCHAIN=auto` + `GOPROXY=off` → `go: download go1.99.0 for
  darwin/arm64: toolchain not available`, exit 1.
- `GOTOOLCHAIN=auto` + proxy on → same message, exit 1 (the version does not
  exist, so there is nothing to fetch).
- `GOTOOLCHAIN=local` → `go: go.mod requires go >= 1.99 (running go 1.26.3;
  GOTOOLCHAIN=local)`, exit 1.

So a machine too old for the declared floor fails closed under every
`GOTOOLCHAIN` setting, and never silently downgrades. At the real `go 1.26`
directive with `GOTOOLCHAIN=local` on `go1.26.3`, the build exits 0 — the floor
is satisfiable, as intended. The CI workflow's own comment
(`.github/workflows/ci.yml`, `test` job) states the only network use is
fetching the toolchain named in `go.mod`; with the bare `go.mod` that is
`actions/setup-go`, not the module graph.

### Deferred risk

**D1 — The `go.work` vector is the one worth a standing local note.** Of all the
bypasses in F3, only `go.work` produces a successful build that leaves `go.mod`
*and* `go.sum` completely untouched — a build that no diff of the repository
would reveal. `.gitignore` has no `go.work` or `go.work.sum` entry, and
`scripts/verify-candidate` does not check for either (zero grep matches for
`go mod`). A `go.work` landing in the repo root or a parent directory would
silently change what CI builds. Out of `allowedFiles`; recorded for the item
that owns `.gitignore`/verifier policy.

**D2 — `module fake-jev` is not a fetchable path.** The module path has no dot in
its first element, so it can never be resolved by a proxy and would be rejected
if another module ever tried to `require` it. This is correct for a
never-published main module and is unchanged from the base revision, so it is not
a consequence of this trim. It becomes a problem only if the module is ever
published or vendored into a consumer; no current work item calls for that.

**D3 — Nothing in the tree declares a `// indirect` convention any more.** With
zero requires the convention has no referent. When FJ-010 adds the YAML module
its transitive entries will reappear, and `go mod tidy` will author them. Not a
risk; noted so the next reader does not read the current emptiness as a policy.

**D4 — The pre-existing `./...` scope issue is unchanged and out of scope.** The
coder and cleaner both record that a bare `go test ./...` in this working tree
reaches the git-ignored `agent/skills/golang-cli/assets/examples/` directory and
fails on the now-undeclared `cobra`/`viper`/`fatih/color`/`fsnotify` imports
plus a placeholder `github.com/you/myapp/cmd`. Re-confirmed at the candidate
revision: the error text is now `no required module provides package …` rather
than `missing go.sum entry …`, and the file set is identical. The same
pre-existing trap applies to `go mod tidy`, which now fails on a network lookup
for the placeholder import before it can report anything useful. Both are
observations about the working tree, not about `go.mod`; the verifier already
excludes those directories via `NON_REPO_DIRS` (`scripts/verify-candidate:1040`,
used by `repo_packages` at lines 1043–1061). Not repaired by this stage.

## Ambiguity: close now, or defer

Nothing in the current shape needs closing *within this item*. Each candidate
was weighed against the constraint that this stage edits no product file and
invents no policy the specification does not contain:

- **Module path stability** — no ambiguity exists. `module fake-jev` is
  preserved byte-for-byte, matches the internal import path used at
  `cmd/fake-jev/main.go:8`, and the only fetchability question (D2) is unchanged
  from the base. **Close now: nothing to do.**
- **`go` directive vs the CI toolchain** — a real, measurable reproducibility
  limit (R3), but §31 Phase 0 specifies a *minimum* and a `toolchain` line is
  not in `allowedFiles`. **Defer**, and record it as a candidate for whichever
  item owns release/toolchain pinning.
- **`replace` / `exclude` / `retract` policy** — the shape permits all three
  (S1) and nothing in the repository forbids them. Writing that prohibition
  would be new policy the specification does not state, and the natural homes
  (`AGENTS.md`, `docs/development/`) are outside `allowedFiles`. **Defer**,
  recorded here as the gap it is.
- **Pinning the fail-closed posture in `verify-candidate`** — the checks run
  with ambient environment (F5). `scripts/` is `metadata` and outside
  `allowedFiles`. **Defer**.

## Open questions

None. The four questions this stage was asked to weigh — fail-closed trace,
reproducibility, `replace`/`exclude`/`retract` permissiveness, and the
toolchain/module-path ambiguity — are each answered above from observed
toolchain behaviour against scratch copies of this tree, not from assumption.
The single genuine gap found (no local policy constrains module-graph
directives, and the verifier does not pin `GOPROXY`/`GOFLAGS`) is recorded as
findings F5 and S1 rather than escalated: closing it requires authoring policy
the specification is silent on, which is a `spec-gap` for a future item to own,
not a decision this stage may make. No `spec-gap` blocker is raised against
FJ-053 itself, whose acceptance lines are fully decidable from §17.2, §15.6 and
§31.
