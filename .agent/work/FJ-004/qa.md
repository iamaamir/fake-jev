---
stage: qa
task: FJ-004
inputFingerprint: c4e916423dd448457485d308552f3c4570ace87a995a9fff2b02c2e2e8c68cbe
outputFingerprint: c4e916423dd448457485d308552f3c4570ace87a995a9fff2b02c2e2e8c68cbe
taskFingerprint: 2ea92092d10f8694889c3b2ce13e29be73b029f0d14ff9cc9f8388d7bf688874
gitHead: 8295626
generatedAt: 2026-09-26T11:12:58Z
---

# QA — FJ-004

Observations only. No verdict is recorded here; `complete` is reachable only when
`./scripts/verify-candidate` exits 0, and that script was not run at this stage.
No product file was edited by this stage, and `.agent/work/FJ-004/state.json`
was not touched.

## Tooling exemption

G-Q's public-surface/system test harness does not exist. `.agent/gate-policy.json`
records the exemption, and its `scope` includes `metadata` — which is the class
`scripts/candidate-fingerprint` assigns to this work item
(`allowedFiles: [".github/workflows/ci.yml"]`):

```json
{ "gate": "G-Q", "blocks": ["stage.tooling_absent"],
  "scope": ["product", "metadata"], "code": "stage.tooling_bootstrap_exempt",
  "reason": "public-surface/system test harness not implemented yet",
  "trackedBy": "FJ-046" }
```

The exemption is **not discharged** by this artifact; it remains owned by FJ-046.
Because of it, the acceptance criteria below were exercised **directly** —
commands run by hand, with their real output recorded — rather than through a
public-surface or system harness. Where a harness would normally supply the
evidence, that absence is named in the gaps section rather than papered over.

## Environment the observations were taken in

`.github/workflows/ci.yml` is **untracked** in this working tree
(`git status --porcelain .github` → `?? .github/`;
`git ls-files --error-unmatch .github/workflows/ci.yml` → *"did not match any
file(s) known to git"*), so a clone of `HEAD` does not contain it. The working
tree also carries the git-ignored `agent/` and `.agents/` skill directories.

Both facts change what a faithful local rehearsal is, so every command below was
run in a **clean `git clone` of `8295626` at `/tmp/fj004-qa`** (removed
afterwards), which is the tree a real runner would check out. For contrast, the
same commands in the raw working tree are recorded in observation O-0.

- Local Go: `go1.26.3 darwin/arm64`
- PyYAML 6.0.3 available. `actionlint` and `yamllint` are **not installed**
  (`which actionlint yamllint` → exit 1, no output), so no Actions-schema or
  expression lint was possible. `GOC` exemption
  `toolchain.pyyaml_exempt` (FJ-047) is noted in the gate policy but the PyYAML
  parse below ran anyway.

## Criteria exercised

One observation per acceptance item from `state.json.acceptance`. Each row
records the exact command and its real result.

### O-0 — baseline: raw working tree (not a CI-equivalent tree)

```
cd /Users/mak/git/fake-jev && GOPROXY=off go test -race ./... 2>&1 | tail -30
```

Result: **exit 1**. Real output:

```
# fake-jev/agent/skills/golang-cli/assets/examples
agent/skills/golang-cli/assets/examples/output.go:9:2: missing go.sum entry for module providing package github.com/fatih/color ...
agent/skills/golang-cli/assets/examples/config.go:9:2: missing go.sum entry for module providing package github.com/fsnotify/fsnotify ...
agent/skills/golang-cli/assets/examples/args.go:6:2: missing go.sum entry for module providing package github.com/spf13/cobra ...
agent/skills/golang-cli/assets/examples/config.go:10:2: missing go.sum entry for module providing package github.com/spf13/viper ...
agent/skills/golang-cli/assets/examples/exit_codes.go:7:2: github.com/fatih/color@v1.19.0: missing go.sum entry for go.mod file; to add it:
	go mod download github.com/fatih/color
FAIL	fake-jev/agent/skills/golang-cli/assets/examples [setup failed]
?   	fake-jev/cmd/fake-jev	[no test files]
?   	fake-jev/internal/cli	[no test files]
FAIL
EXIT=1
```

Every failure is under `agent/skills/golang-cli/assets/examples/`, which
`.gitignore` excludes (`/agent/`, `/.agents/`, `/skills-lock.json`) and which
CI never sees. The two committed packages report `[no test files]`. This is
recorded because it is a property of the *developer* tree, not of CI, and
because a reader who runs the command in place will otherwise think the criterion
fails. All criteria below are therefore assessed against the clean clone.

### O-1 — C-QUAL-002: `go test -race ./...` passes on supported runners

```
cd /tmp/fj004-qa && GOPROXY=off go test -race ./... 2>&1
```

Real output:

```
?   	fake-jev/cmd/fake-jev	[no test files]
?   	fake-jev/internal/cli	[no test files]
EXIT=0
```

The command exits 0. It is also a **required** check, not an optional one: the
workflow gives it its own named step, and no construct in the file can let a
failure pass (O-4). It is **not** satisfied by the non-race run — `go test ./...`
is a separate step and both were run (O-3).

Two limits on this observation, both material:

- **The runner is not a supported-runner observation.** §23 annotates this line
  `# supported runner(s)`. The command was executed on `darwin/arm64`; the
  workflow pins `runs-on: ubuntu-latest`. Whether the step passes on
  `ubuntu-latest` is unobservable here.
- **The command passes vacuously.** The tree contains **zero** `*_test.go` files
  (`find . -name '*_test.go' -not -path './.git/*' | wc -l` → `0`). The race
  detector therefore has nothing to instrument. Exit 0 here means "the command
  runs and is wired as a required check", not "concurrent code is race-free".
  §22.1's "concurrent safety" test obligation is untouched by this item.

### O-2 — C-QUAL-005: golden contracts run offline with no credentials, model, or network

Three parts: a dependency scan, a secret/URL grep, and an offline execution.

**No credential, model, or paid-service dependency in the tree.**

```
cd /Users/mak/git/fake-jev && grep -rniE 'secrets\.|api[_-]?key|https?://' .github/
```

Result: **exit 1, no output** — no match anywhere under `.github/`.

```
cd /Users/mak/git/fake-jev && grep -nEi 'secrets\.|api[_-]?key|token|credential|password|https?://|\.com|\.io|\.ai|aws|gcp|azure|s3|registry|docker|npm |pip ' .github/workflows/ci.yml
```

Result: **exit 1, no output** — no match. The file contains no `secrets.*`
reference, no key/token/credential name, no URL, no hostname, and no reference
to any registry or paid provider. The only network-touching steps in the file
are `actions/checkout@v4` and `actions/setup-go@v5`.

The committed Go sources have no network or subprocess surface at all:

```
cd /tmp/fj004-qa && grep -rnE '"net|"os/exec"|"net/http"|http\.|net\.Dial|exec\.Command' cmd internal
```

Result: **exit 1, no output**. Their entire import set is `fmt`, `io`, `os`,
`runtime`, `runtime/debug`, `strconv`, `strings` and the internal
`fake-jev/internal/cli` — standard library only. `go.mod` declares five direct
third-party requirements (`fatih/color`, `fsnotify`, `cobra`, `viper`, plus
indirects) but **no committed package imports any of them**, so no build or test
path touches them.

**Offline execution.** The golden contract data is present and machine-readable
with no network and no model:

```
cd /Users/mak/git/fake-jev && python3 -c "import json,glob; [json.load(open(p)) for p in glob.glob('testdata/contracts/**/*.json',recursive=True)]; print('golden vectors parsed offline:',n)"
```

Real output: `golden vectors parsed offline: 23`. All 23 §22.3-shaped vectors
parse as JSON offline. **No test reads them yet** — there is no Go test code in
the tree, so nothing executes the vectors as contracts today.

The Go checks were additionally re-run with a **credential-scrubbed
environment** (`env -i PATH=... HOME=... GOPROXY=off`), because this developer
shell does export unrelated third-party credentials
(`TYPESAFE_API_KEY`, `SOFA_API_KEY`, `AMO_API_KEY`, `AMO_API_SECRET`,
`SYSTEM_ONE_API_KEY`, `HF_TOKEN` — none of them fake-jev's, none referenced by
the repository, but present enough to make an unscoped run unrepresentative of a
CI runner):

```
cd /tmp/fj004-qa && env -i PATH=/usr/local/go/bin:/usr/bin:/bin HOME=$HOME GOPROXY=off go test -race ./...
  -> ?  fake-jev/cmd/fake-jev [no test files]
     ?  fake-jev/internal/cli [no test files]
     EXIT=0
cd /tmp/fj004-qa && env -i PATH=... GOPROXY=off go vet ./...
  -> (no output)  EXIT=0
all five GOOS/GOARCH builds under env -i  ->  linux/amd64 EXIT=0, linux/arm64 EXIT=0,
     darwin/amd64 EXIT=0, darwin/arm64 EXIT=0, windows/amd64 EXIT=0
```

Removing every credential from the environment changes no result. That is the
honest extent of the offline claim: **the checks require no credential**, which
§23 asks for ("must *require* no model, no provider API key, and no paid
external service").

**The offline flag is load-bearing, proven negatively:**

```
cd /tmp/fj004-qa && env GOPROXY=off GOMODCACHE=$(mktemp -d) go get github.com/fatih/color@v1.19.0
```

Real output: `go: github.com/fatih/color@v1.19.0: module lookup disabled by
GOPROXY=off`, **exit 1**. So `GOPROXY: "off"` genuinely refuses module
resolution rather than silently downloading, and the failure direction is
fail-closed. It is **not** a socket sandbox (it does not close outbound
connections) — the workflow's own comment says so, and that claim is accurate
as written.

### O-3 — Build matrix covers exactly linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64

Parsed with PyYAML and asserted against the §23.1 list:

```
cd /Users/mak/git/fake-jev && python3 -c "<yaml.safe_load of .github/workflows/ci.yml; extract jobs.build.strategy.matrix.include>"
```

Real output:

```
top-level keys: ['name', True, 'permissions', 'jobs']
on (True==YAML1.1 'on'): {'push': {'branches': ['main']}, 'pull_request': None}
permissions: {'contents': 'read'}
jobs: ['test', 'build']
matrix got : [('linux','amd64'), ('linux','arm64'), ('darwin','amd64'), ('darwin','arm64'), ('windows','amd64')]
matrix want: [('linux','amd64'), ('linux','arm64'), ('darwin','amd64'), ('darwin','arm64'), ('windows','amd64')]
MATRIX_EQUALS_SPEC_SET: True
MATRIX_IS_SET_EQUAL: True | extra: set() | missing: set()
fail-fast: False
test job env: {'GOPROXY': 'off'}
build job env: {'GOPROXY': 'off'}
```

The matrix is **exactly** the five §23.1 targets, in §23.1 order: no extra
target, none missing, and `windows/arm64` correctly absent (§23.1 permits it only
"when release tooling and demand justify it"). Note `top-level keys` contains
`True` rather than `'on'`: that is PyYAML's YAML 1.1 boolean resolution of the
`on` key, not a workflow defect — GitHub's parser reads `on` as the trigger key.
Same for the `['push','pull_request']` output in the duplicate-key check below.

All five were then actually built, exactly as the workflow's step does
(`GOPROXY=off CGO_ENABLED=0 GOOS=<os> GOARCH=<arch> go build -o dist/fake-jev-<os>-<arch> ./cmd/fake-jev`):

```
cd /tmp/fj004-qa && for pair in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64; do ... done
```

Real output, one line per target:

```
=== linux/amd64   === EXIT=0  dist/fake-jev-linux-amd64:   ELF 64-bit LSB executable, x86-64, statically linked
=== linux/arm64   === EXIT=0  dist/fake-jev-linux-arm64:   ELF 64-bit LSB executable, ARM aarch64, statically linked
=== darwin/amd64  === EXIT=0  dist/fake-jev-darwin-amd64:  Mach-O 64-bit executable x86_64
=== darwin/arm64  === EXIT=0  dist/fake-jev-darwin-arm64:  Mach-O 64-bit executable arm64
=== windows/amd64 === EXIT=0  dist/fake-jev-windows-amd64: PE32+ executable (console) x86-64, for MS Windows
```

`file` confirms each artifact carries the intended target format and
architecture, not merely that the command returned 0. `dist/` does not pre-exist
in a clean clone; `go build -o` creates it, verified separately (exit 0, `dist`
populated), so the build step does not depend on a git-ignored directory that
is already there. These are cross-compiles from one `darwin/arm64` host, not
builds on the target platforms — which is what §23.1's matrix in this workflow
is designed to do, and the workflow runs on `ubuntu-latest`, a third host.

### O-4 — No CI secret or paid external service is required for a normal change

```
cd /Users/mak/git/fake-jev && grep -nE 'continue-on-error|if:|always\(\)|\|\| true|exit 0|workflow_dispatch|schedule:|environment:' .github/workflows/ci.yml
```

Result: **exit 1, no output**. There is no skip path, no soft-fail, no
conditional, no scheduled or manually-dispatched job, and no GitHub
`environment:` (which could gate on protection rules or secrets). A failing
command fails its step, fails its job, and fails the workflow.

The credential surface is nil (O-2), and `permissions: contents: read` is
workflow-level with no job widening it — the only token in play is the
per-run `GITHUB_TOKEN` that `actions/checkout` mints at read-only scope.

The §23 offline discipline is carried by a single job-level `env: GOPROXY: "off"`
on each of the two jobs (parsed above). §23 also says "Where practical, run
deterministic integration tests with outbound network access disabled." The
workflow disables the **module proxy**, not outbound sockets, and the file's own
comment states that limitation and defers the runtime-path guarantee to the
§22.3/§22.4 suites. That is an honest reading of "where practical" for phase 0,
not a hidden gap — but see G-3.

**Local rehearsal of a normal change.** A clean clone with the workflow file
dropped in, no secrets configured, no funded account, and an empty module cache
was used for every command above. The untracked `.github/` in this working tree
means the committed tree at `8295626` does not contain the workflow, so the
clone used was `git clone` of `HEAD` plus the workflow copied in — the closest
local analogue to what a runner checks out. The residual unobservable is G-1/G-2.

### O-5 — Additional local validation of the workflow file

```
cd /Users/mak/git/fake-jev && python3 -c "yaml.load with a duplicate-key-asserting constructor over .github/workflows/ci.yml"
```

Real output: `parsed with duplicate-key check: OK; jobs= ['push', 'pull_request']`
(the printed list is the `on:` block, reached through the YAML 1.1 `True` key).

The file is well-formed YAML with no duplicate mapping keys at any level. Its
full step inventory, as parsed:

```
job test  (runs-on: ubuntu-latest, env {'GOPROXY': 'off'}):
  - uses: actions/checkout@v4
  - uses: actions/setup-go@v5   with {go-version-file: go.mod}
  - name: go test ./...          run: go test ./...
  - name: go test -race ./...    run: go test -race ./...
  - name: go vet ./...           run: go vet ./...
job build (runs-on: ubuntu-latest, fail-fast: false, env {'GOPROXY': 'off'}):
  - uses: actions/checkout@v4
  - uses: actions/setup-go@v5   with {go-version-file: go.mod}
  - name: go build ./cmd/fake-jev
    run: go build -o dist/fake-jev-${{ matrix.goos }}-${{ matrix.goarch }} ./cmd/fake-jev
    env: {CGO_ENABLED: "0", GOOS: ${{ matrix.goos }}, GOARCH: ${{ matrix.goarch }}}
```

`go vet ./...` and `go test ./...` were both run in the clean clone and both
exit 0 (O-1, O-3; `go vet` → no output, exit 0).

**Not performed:** no `actionlint` and no `yamllint` (neither installed, no
network to install them). So there is **no schema validation** of the workflow
against the GitHub Actions workflow schema, and **no expression lint** of the
`${{ matrix.goos }}` / `${{ matrix.goarch }}` references. The only `${{ }}`
expressions in the file are those three (step name, `run` string, `env` value),
and the matrix is `include`-only so it needs no cross-product expansion — but
that is reasoning about the file, not a tool confirming it.

## Explicit gaps

Stated as limits of what was observed. None is a verdict, and none is fixed here.

**G-1 — The workflow has never executed on a real GitHub runner.** This is the
central limit. No run of `.github/workflows/ci.yml` exists. The properties below
are **unobservable locally** and no local command can substitute:

- a **fresh runner** — image contents, preinstalled Go, disk, and the
  `ubuntu-latest` image definition itself are all outside this repository;
- **`actions/checkout@v4` and `actions/setup-go@v5` resolution** — the tags are
  fetched from GitHub at run time. `GOPROXY: "off"` does not constrain them, and
  both are tag-pinned rather than SHA-pinned, so what executes is whatever the
  tag points at on the day of the run;
- **`actions/setup-go@v5`'s implicit module cache behavior.** The workflow
  declares no `cache:` key and the repository has **no committed `go.sum`**
  (`ls go.sum` → *No such file or directory*). `setup-go` enables its built-in
  cache by default when a `go.sum` is present and leaves it off when none is
  found, so the cache is silently inactive today and will silently activate the
  day a `go.sum` lands, with no change to the workflow. That is an
  environment-dependent behavior the file does not declare either way;
- **`go-version-file: go.mod` resolution** — `go.mod` declares `go 1.26`; whether
  `setup-go` provisions a toolchain satisfying it on `ubuntu-latest` is only
  knowable by running Actions;
- **job-level `env` merged with step-level `env`** — documented Actions behavior,
  but not exercised on a runner here. If merging ever replaced rather than
  merged, the `build` step would silently lose `GOPROXY: "off"`;
- **runner-side `GOPROXY` interaction** — the workflow sets `GOPROXY=off` but not
  `GOSUMDB`/`GONOSUMDB`; with no `go.sum` and no third-party imports this is
  inert today, and its first exercise would be on a real runner.

**G-2 — The workflow file is untracked.** `git status --porcelain .github` →
`?? .github/`, and `git ls-files --error-unmatch .github/workflows/ci.yml` fails.
The candidate is real on disk and the candidate fingerprint
(`c4e916…c68cbe`, which I re-derived) covers untracked files
(`scripts/candidate-fingerprint` enumerates `git ls-files` **plus**
`git ls-files --others --exclude-standard`), so the file is inside the verified
candidate. But a *fresh clone of `HEAD`* does not contain it, so criterion 4
("a normal change … reaches the same required checks") is satisfied only once
this file is actually committed and pushed. Until then no GitHub run can exist
at all, which is also the mechanical reason G-1 is unclosable today. Observed and
reported; not staged, not committed, not fixed — that is the PM's call and
outside this stage's file scope.

**G-3 — "Outbound network access disabled" is not implemented, only proxied.**
§23 says "Where practical, run deterministic integration tests with outbound
network access disabled." `GOPROXY: "off"` disables the module proxy; it does
not disable sockets. Today this is unobservable because no test exists and no
source imports `net` or `net/http`, but the moment the §22.3/§22.4 suites land
they will run with sockets open. The workflow's comment says this plainly, so
this is a disclosed scope boundary, not a concealed gap — recorded so the PM can
decide whether §23's sentence needs its own future item.

**G-4 — No runtime coverage exists.** Zero `*_test.go` files. Every `go test`
result in this artifact is a vacuous pass, and `go test -race ./...` has nothing
to race. C-QUAL-005's "golden contracts run **offline**" is only half-exercised
as a result: the 23 vectors parse offline (O-2), but no code consumes them. The
offline *runtime path* is deferred to the §22.3/§22.4 suites, as the workflow
comment states.

**G-5 — No `config/schema validation` or `integration tests` step.** §23 lists
both as baseline checks distinct from `go test ./...`; the workflow has neither
as a separate step. Consistent with O-4's step inventory. The specifier
declined to make either a falsifiable acceptance line, and `state.json.acceptance`
does not list them, so this is recorded as an observation rather than a failed
criterion. The specifier and the cleaner both flagged it; QA observed the
absence in the parse and neither can settle whether §23 requires a distinct
line.

**G-6 — The `GOPROXY: "off"` forward commitment.** Proven load-bearing in
O-2. `go.mod` declares five direct requirements and there is **no `go.sum`**, so
the first change that actually imports a third-party module must commit
`go.sum` in the same change or both jobs stop with a module-lookup error instead
of downloading. The direction of failure is safe (fail-closed) but the coupling
is undocumented in the file. Observed; not changed.

## Open questions

Empty. No acceptance item was ambiguous enough to warrant `needs-info`, and no
behavior in this candidate would require a specification change, so no
`spec-gap` is recorded. The items above are limits of local observability and
scope boundaries, not specification gaps; each is stated with its evidence so
the PM can route it (G-2 to a commit decision, G-3 and G-5 to a future item,
G-1 to the first real run).
