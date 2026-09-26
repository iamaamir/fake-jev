---
stage: specifier
task: FJ-004
inputFingerprint: 5fe4ba9be1430d49a566a75e1ca5e78d9b638b45cba746e9acfd54bd68bed9fc
outputFingerprint: 5fe4ba9be1430d49a566a75e1ca5e78d9b638b45cba746e9acfd54bd68bed9fc
taskFingerprint: 2ea92092d10f8694889c3b2ce13e29be73b029f0d14ff9cc9f8388d7bf688874
gitHead: 8295626
generatedAt: 2026-09-26T10:59:30Z
---

# Specifier — FJ-004

## What is delivered

One file: `.github/workflows/ci.yml`. No Go source, no test code, no product
behavior. The work item makes the continuous-integration pipeline the
repository currently lacks, so that §23's baseline checks and §23.1's build
matrix are exercised on a normal change.

The specifier stage changes no product file, so the input and output candidate
fingerprints are identical. Both were re-derived with
`./scripts/candidate-fingerprint candidate .` and
`./scripts/candidate-fingerprint task .agent/work/FJ-004/state.json`; they
match the values recorded in this front matter.

## Class

`allowedFiles` is exactly `.github/workflows/ci.yml`. The fingerprint tool
(`scripts/candidate-fingerprint`, `classify_path`) matches it under
`METADATA_PREFIXES`, so the work item classifies as **`metadata`**, not
`product`.

Consequence for required stages, per `derive_required_stages`: the canonical
product ladder would otherwise be `specifier, coder, hardener, cleaner, qa`.
For `metadata` the derived set is **`specifier, coder, cleaner, qa`** — the
`hardener` stage is **not required** for FJ-004. This is why the gauntlet
output shows `Hardener: not required`.

Two further consequences follow from `metadata` and the coder must respect
them:

- The item has no product surface to harden, so there is no concurrency,
  auth, or secret-handling review to perform. The security question that does
  exist here is answered by C-QUAL-005 below, not by a hardener pass.
- A `metadata` change does not alter runtime behavior. The evidence value of
  this stage is that CI is *present and green*, not that the product changed.

## Traces

C-QUAL-002, C-QUAL-005

Both ids exist in `docs/development/acceptance-catalog.md` (section
`C-QUAL — repository quality gates (§22, §23, §45)`). No other catalog id is
claimed: C-QUAL-001 and C-QUAL-003 are exercised *by* the workflow rather than
being new observable outcomes of this item, and the remaining acceptance lines
in `state.json` are plain-text requirements, not catalog ids.

## Criteria

Restated from `state.json.acceptance`, each as an observable check.

1. **C-QUAL-005 — golden contracts run offline with no credentials, model, or
   network (§23, §22.3).** The workflow contains no step that requires a
   `secrets.*` reference, an API key, a model endpoint, or any paid external
   service. QA can falsify this by grepping the workflow for `secrets.`,
   `API_KEY`, and outbound hosts, and by reading the job list: the only
   network-touching operation is fetching the Go toolchain and module
   dependencies. Nothing in the workflow may be conditional on a secret being
   present. §23 states the requirement as "Normal PR CI must require no model,
   no provider API key, and no paid external service", so the check is about
   *requiring*, not about forbidding all network use.

2. **C-QUAL-002 — `go test -race ./...` passes on supported runners (§23).** The
   workflow runs a job whose command is `go test -race ./...` and whose exit
   status gates the workflow. §23 annotates this line `# supported runner(s)`,
   so running it on a Linux runner is sufficient; the criterion is that the
   check exists and is required, not that it runs on every matrix platform.
   The non-race `go test ./...` from the same §23 baseline is a separate
   invocation and must not be substituted for it.

3. **Build matrix covers five platforms (§23.1).** The matrix job enumerates
   exactly `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`,
   `windows/amd64`, cross-compiled with `GOOS`/`GOARCH` `env` settings so a
   single Linux runner covers all five. QA can falsify this by comparing the
   matrix entries against the §23.1 list. `windows/arm64` is deliberately
   absent: §23.1 permits it "when release tooling and demand justify it", and
   neither condition is met here. Adding it is out of scope, not a defect.
   Note the distinction: the *build* matrix is cross-compilation; §23.1 says
   nothing about *running* tests on all five platforms, so the matrix must
   build, and running the test suite natively on a non-Linux OS is not
   required.

4. **No CI secret or paid external service is required for a normal change.**
   Observable as: a push or pull request against this repository with a clean
   clone, no configured secrets, and no funded accounts reaches the same
   required checks. This is the criterion that a passing run is *evidence* for,
   and it overlaps C-QUAL-005 deliberately — QA should confirm both the grep
   and an actual run with the secret store empty.

Remaining §23 baseline items not claimed as criteria here: `go vet ./...`,
config/schema validation, and integration tests are named in §23 and are
expected to appear in the workflow, but `state.json.acceptance` does not list
them, so this stage does not turn them into falsifiable acceptance lines. The
coder should still include them — §23 says "Required baseline checks" — but
their absence is a review finding, not a failed criterion of this item.

## Non-goals

Echoing `state.json.nonGoals`, with the spec text that supports each:

- **No release publishing yet.** §23.2 mentions release checksums and
  reproducible, automated release builds; §24 covers distribution. FJ-004 does
  not publish, tag, upload artifacts to a registry, or sign anything. The
  build matrix produces binaries to prove they compile; it does not release
  them.
- **No live upstream drift job.** §22.7 describes an optional scheduled or
  manual job that calls the real Jev API. It is excluded here for the four
  reasons §22.7 itself lists: never required for normal PR CI, never required
  for contributors without credentials, never part of deterministic test
  correctness, and credentials stored only in CI secrets. §22.7's line that
  this "is the only area where real API cost is acceptable" is permission for
  a future item, not an instruction now. The system-one-core cross-repository
  smoke test (§22.6) is likewise out of scope: §22.6 says it is not required
  in every PR.

## Open questions

None. The specification constrains this item completely enough to be specified
without invention: §23 fixes the required check list and the no-credential
rule, §23.1 fixes the five build targets and permits deferring
`windows/arm64`, §22 fixes the test-suite behavior CI must exercise, and §22.7
fixes the boundary that the drift job is outside it. Remaining choices
(runner image, Go version pinning, job names, matrix mechanics) are
implementation detail, not public behavior, and the authority hierarchy
reserves them for the coder rather than for a blocker.
