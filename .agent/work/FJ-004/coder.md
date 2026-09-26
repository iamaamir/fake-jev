---
stage: coder
task: FJ-004
inputFingerprint: 5fe4ba9be1430d49a566a75e1ca5e78d9b638b45cba746e9acfd54bd68bed9fc
outputFingerprint: c4e916423dd448457485d308552f3c4570ace87a995a9fff2b02c2e2e8c68cbe
taskFingerprint: 2ea92092d10f8694889c3b2ce13e29be73b029f0d14ff9cc9f8388d7bf688874
gitHead: 8295626
generatedAt: 2026-09-26T11:06:45Z
---

# Coder — FJ-004 (revision pass)

This is a revision of the coder artifact, not a fresh implementation. The
only product file in the work item is `.github/workflows/ci.yml`; the
revision changed that file and rewrote this artifact. No Go source, no test
code, no product behavior. `state.json` was not touched.

## What the file is

One workflow, `.github/workflows/ci.yml`, two jobs, both on `ubuntu-latest`,
triggers `push` to `main` and `pull_request`, `permissions: contents: read`.

**`test` — §23 required baseline checks.** Three named steps so the failing
command is named in the run summary: `go test ./...`, `go test -race ./...`,
`go vet ./...`. The job carries a single job-level `env: GOPROXY: "off"`; no
step declares an `env` of its own. Nothing in the job depends on a secret, a
model, a provider, or a funded account.

**`build` — §23.1 matrix.** `fail-fast: false` and five `include` entries, in
§23.1 order: `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`,
`windows/amd64`. One runner cross-compiles all five. The job-level
`env: GOPROXY: "off"` matches the `test` job; the build step keeps its own
step-level `env` for `CGO_ENABLED: "0"`, `GOOS` and `GOARCH`, which is
additive with the job-level map. Output goes to `dist/` (already git-ignored).
`windows/arm64` is absent because §23.1 permits it "when release tooling and
demand justify it" and neither condition is met. The binaries prove
compilation only — no upload, tag, checksum or signature, since §23.2 and §24
are not this item.

The Go version is not written into the workflow: `actions/setup-go@v5` reads
`go-version-file: go.mod`, so the pin is the `go 1.26` directive in `go.mod`
itself and cannot drift away from it. There is no cache configuration, no
concurrency group, no `timeout-minutes`, no conditional job and no scheduled
job.

## What `GOPROXY: "off"` does and does not do

`GOPROXY=off` disables the Go **module proxy**. Any attempt to resolve a module
that is not already in the local module store fails with a module-lookup error.
It does **not** disable outbound sockets, and it is not a sandbox: a test that
opened a TCP connection to a live host would still be able to. The comment
above the `test` job now says exactly that, and defers the runtime-path
offline guarantee to the §22.3 and §22.4 test suites once they exist.

## Review findings applied

Three changes, each answering one accepted G-L finding:

1. **Finding 1 — overclaiming comment.** The old comment asserted that the
   tests "read the committed golden contracts in `testdata/` and never reach
   the network". `GOPROXY: "off"` cannot support the second clause. The
   comment now states the module-download refusal, says plainly that it is
   not a socket sandbox, and points the runtime-path guarantee at §22.3/§22.4.
2. **Finding 2 — env inconsistency.** `GOPROXY: "off"` applied to the two test
   steps but not to `go vet ./...` and not to the `build` job at all. It is
   now a job-level `env` on `test` (covering the vet step) and on `build`.
3. **Finding 4 — dead flag.** `GOFLAGS: -mod=readonly` was removed from both
   test steps. It restated the Go default, which has been `-mod=readonly`
   since Go 1.16, and `go.mod` declares `go 1.26`. `GOPROXY: "off"` is the
   load-bearing line.

Finding 3 (the two-line per-step `env` block duplicated across the two test
steps) is resolved by change 2: the block now exists once per job, and no
step-level copy of `GOPROXY` remains.

Findings 5, 9, 14 and 15 were consciously **not** actioned — accepted cost, or
a missing-hardener-stage question reserved for the PM. This pass leaves them
alone. No caching, `timeout-minutes`, `concurrency`, composite actions,
reusable workflows, extra matrix targets, SHA-pinned actions, or
scheduled/drift job were added.

## Criterion mapping

| Specifier criterion | Where it is satisfied |
|---|---|
| 1 — C-QUAL-005, golden contracts run offline with no credentials, model or network | `test` job, job-level `GOPROXY: "off"`; no `secrets.*`, no key, no host anywhere in the file |
| 2 — C-QUAL-002, `go test -race ./...` is required | its own step in the `test` job, whose failure fails the workflow; not merged into or replaced by the non-race run |
| 3 — build matrix covers exactly the five §23.1 targets | `build` matrix `include` list, in §23.1 order |
| 4 — no secret or paid service needed for a normal change | no `secrets.*` reference, no environment variable other than `GOPROXY`/`CGO_ENABLED`/`GOOS`/`GOARCH`, and `permissions` reduced to read-only contents |

§23's `config/schema validation` and `integration tests` lines are not separate
steps, because neither exists as a command in the tree today: `spec/config.schema.json`
and `testdata/` are data, and phase 0 has no schema validator and no test
packages. Once FJ-046 and the schema-validation tests land, `go test ./...`
executes them — the §22.4 HTTP integration tests, §22.5 control API tests and
§22.3 golden tests are all plain `go test` packages, so a separate step would
duplicate the same run.

## What was verified locally

Commands were run in a `git clone` of the current tree at `/tmp/fj004-rehearsal`
(removed afterwards). A clean clone is the honest rehearsal of what CI sees: the
working tree additionally carries the git-ignored `agent/` and `.agents/`
skill directories, whose example programs do not build and which CI will not
have. Local Go is `go1.26.3 darwin/arm64`.

| Command | Result |
|---|---|
| `GOPROXY=off go test ./...` | exit 0 — `cmd/fake-jev` and `internal/cli` both `[no test files]` |
| `GOPROXY=off go test -race ./...` | exit 0 — same two packages, no test files |
| `GOPROXY=off go vet ./...` | exit 0, no output |
| `GOPROXY=off CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o dist/fake-jev-linux-amd64 ./cmd/fake-jev` | exit 0, ELF 64-bit x86-64 |
| same for `linux/arm64` | exit 0, ELF 64-bit ARM aarch64 |
| same for `darwin/amd64` | exit 0, Mach-O 64-bit x86_64 |
| same for `darwin/arm64` | exit 0, Mach-O 64-bit arm64 |
| same for `windows/amd64` | exit 0, PE32+ console x86-64 |
| `python3 -c "yaml.safe_load(...)"` on the workflow | parses; jobs are exactly `test` and `build`; `test` step runs are exactly `go test ./...`, `go test -race ./...`, `go vet ./...`; no step declares an `env`; `test` job env is `{'GOPROXY': 'off'}` and `build` job env is `{'GOPROXY': 'off'}`; the matrix asserts to exactly the five §23.1 pairs in §23.1 order |
| `grep -nE "secrets\.\|API_KEY\|api_key\|token\|https?://"` | no match (exit 1) |
| `grep -nE "timeout-minutes\|concurrency\|cache\|workflow_call\|schedule:"` | no match (exit 1) |
| `grep -n GOFLAGS` | no match (exit 1) |
| `./scripts/candidate-fingerprint candidate .` | `c4e916423dd448457485d308552f3c4570ace87a995a9fff2b02c2e2e8c68cbe` (recorded above) |
| `./scripts/candidate-fingerprint task .agent/work/FJ-004/state.json` | `taskFingerprint` `2ea92092d10f8694889c3b2ce13e29be73b029f0d14ff9cc9f8388d7bf688874`, unchanged |

For contrast, `go test ./...` in the **working tree** exits 1 with
`missing go.sum entry` for `github.com/fatih/color`, `fsnotify`, `cobra` and
`viper` — all from `agent/skills/golang-cli/assets/examples`, a git-ignored
skill directory that is not part of the repository. The clean-clone runs
above are the ones that describe CI.

## What could not be verified locally

`actionlint` and `yamllint` are not installed here and the workflow could not
be installed without network access, so no Actions expression or schema lint
was run. The only `${{ }}` expressions are `matrix.goos`/`matrix.goarch` in a
step name, a `run` command and an `env` value, and the `include`-only matrix
needs no cross-product expansion. Job-level `env` merging with step-level
`env` is documented Actions behavior but was not exercised on a real runner.
Whether the jobs actually pass on `ubuntu-latest` with Go 1.26 provisioned is
only knowable by running Actions. No verdict is claimed here.

## Deviations and open questions

- `GOPROXY: "off"` is the strongest form of "offline" that is honest without a
  network namespace, and the comment now says so. It is also a mild forward
  commitment: the first time a real dependency lands, its `go.sum` entries must
  be committed or the steps will fail with a module-lookup error rather than
  downloading. `go.mod` already lists five direct requirements and there is no
  committed `go.sum`, so that moment is close. Dropping the two job-level
  `env:` blocks is a two-line change if a reviewer prefers.
- The repository has no committed `go.sum` and no test files yet, so the `test`
  job proves "the checks run and are required", not "behavior is covered".
  That is expected for phase 0 and matches the specifier's reading that the
  evidence value here is CI present and green.
- No deviation from the specifier's criteria; no spec gap found.
