---
stage: cleaner
task: FJ-004
inputFingerprint: c4e916423dd448457485d308552f3c4570ace87a995a9fff2b02c2e2e8c68cbe
outputFingerprint: c4e916423dd448457485d308552f3c4570ace87a995a9fff2b02c2e2e8c68cbe
taskFingerprint: 2ea92092d10f8694889c3b2ce13e29be73b029f0d14ff9cc9f8388d7bf688874
gitHead: 8295626
generatedAt: 2026-09-26T11:12:40Z
---

# Cleaner — FJ-004 (re-review of the current candidate)

Review scope is simplicity and altitude: one job, one concern; duplication;
speculative generality; readability and YAML idiom; credential surface;
fail-closed posture. Behavior correctness is the QA stage's business, not this
one.

This stage made **no edits**. `.github/workflows/ci.yml` was read and parsed,
not modified; no product file and no `state.json` was touched. The candidate
is therefore unchanged and the input and output fingerprints are equal.

## Tooling exemption

G-L's complexity/CRAP analyzer does not exist. `.agent/gate-policy.json`
records the exemption for exactly this case:

```json
{ "gate": "G-L", "blocks": ["stage.tooling_absent"], "scope": ["product", "metadata"],
  "code": "stage.tooling_bootstrap_exempt",
  "reason": "complexity/CRAP analyzer not implemented yet", "trackedBy": "FJ-044" }
```

FJ-004 classifies as `metadata` (`scripts/candidate-fingerprint` reports
`"class":"metadata"`), which is inside the exemption's `scope`, so the block
does not fire for this candidate. The analysis below was performed **by
reading** the whole 64-line file, line by line, and by parsing it — not by any
tool. The exemption is tracked by FJ-044; this artifact does not discharge it
and makes no claim about what the analyzer would report.

## What was inspected

- `.github/workflows/ci.yml` — the candidate, all 64 lines, with line numbers.
- `spec/fake-jev-technical-spec-v2.md` §22, §23, §23.1 (and §23.2 as the
  adjacent boundary the build job's comment names). No other spec section was
  used as authority.
- `go.mod` (module `fake-jev`, `go 1.26`, four direct requirements), absence of
  a committed `go.sum`, `.gitignore` (`/dist/` is ignored).
- The three committed Go files (`cmd/fake-jev/main.go`, `internal/cli/version.go`,
  `internal/cli/dispatch.go`) for actual third-party imports.
- `.agent/work/FJ-004/specifier.md` and `.agent/work/FJ-004/coder.md`.
- `docs/agents/roles/cleaner.md`, `docs/agents/expertise/staff-architect.md`,
  `docs/agents/README.md` (authority hierarchy), `.agent/gate-policy.json`,
  `.agent/schema/stage-artifact.schema.json`.

No `actionlint` or `yamllint` is available in this environment, so no Actions
schema lint or expression lint was run. The file was parsed with PyYAML
instead; note that PyYAML's YAML 1.1 resolver reads the `on:` key as boolean
`True` (top-level keys parse as `['name', True, 'permissions', 'jobs']`).
That is a PyYAML artifact, not a workflow defect — GitHub's parser treats
`on` as the trigger key — and it is recorded only so the next reader does not
mistake it for one.

## Findings

Observations, not verdicts. Severity is not a pass/fail judgment; it only
orders the reading.

### Simplification

1. **`ci.yml:59-60` — step name and command disagree, and the output is
   consumed by nothing.** The step is `name: go build ./cmd/fake-jev` but runs
   `go build -o dist/fake-jev-${{ matrix.goos }}-${{ matrix.goarch }}
   ./cmd/fake-jev`. Nothing downstream reads `dist/`: there is no upload
   action, no artifact step, no checksum, and §23.2 release integrity is
   explicitly out of scope. The `-o` therefore buys a template expression, an
   otherwise-unused directory, and a per-target filename that is never
   distinct in practice because the job is re-created per matrix entry.
   `go build ./cmd/fake-jev` proves the same compilation. Low cost either way;
   if the name is kept, at least make it match the command.

2. **`ci.yml:60` — the generated filename is not a usable Windows name.**
   `fake-jev-windows-amd64` carries no `.exe` extension. Harmless while the
   artifact is discarded, and it becomes a real defect the day upload or
   release is added. Mentioned only because it is the same line as finding 1
   and disappears with it.

3. **`ci.yml:21-22` — job id and display name do not match.** The id is
   `test`; the name is `test and vet`. The job carries two concerns (test
   execution and static analysis). Splitting `go vet` into its own job would
   satisfy "one job, one concern" literally, but it costs a third runner and
   a third copy of the four-line preamble (finding 6), and `go vet` operating
   on the same tree as `go test` is the cheaper structure at this size. The
   mismatch is the real smell, not the merge: either the name or the id is
   wrong, and picking the wrong one is what makes it look like an accident.

### Duplication

4. **`ci.yml:23,27-30` vs `ci.yml:42,55-58` — the four-line job preamble is
   byte-identical in both jobs.** `runs-on: ubuntu-latest`, the job-level
   `env: GOPROXY: "off"`, `actions/checkout@v4`, and `actions/setup-go@v5`
   with `go-version-file: go.mod` appear once per job. This is prior finding
   5, re-confirmed against the current file; it was consciously not actioned.
   The only native ways to share it are a composite action or a reusable
   workflow, and both are new moving parts that §23 does not ask for and that
   would need their own review surface later. At two jobs the duplication
   costs less than the abstraction. Recorded so the decision is visible, not
   to reopen it.

5. **`ci.yml:24-25` and `ci.yml:52-53` — `GOPROXY: "off"` is declared twice.**
   A workflow-level `env:` would be the one-line form, but it would then also
   apply to any job added later, including one that legitimately needs the
   proxy. Two explicit declarations is the more fail-closed shape. No action
   recommended; noted so the repetition reads as a choice.

### Speculative generality

6. **None found.** No composite action, no reusable workflow, no
   `workflow_call`, no `workflow_dispatch`, no `schedule:`, no conditional
   `if:`, no `continue-on-error`, no extra matrix target, no extra environment
   knob, no cache configuration. `windows/arm64` is absent, which is what
   §23.1's deferral sentence permits. The file is the size the specification
   asks for and no larger (AGENTS.md: no abstractions for hypothetical
   futures).

### Readability and YAML idiom

7. **`ci.yml:47-51` — flow-style mapping entries in a block-style file.**
   `- { goos: linux, goarch: amd64 }` is valid YAML, and it makes the five
   targets read like the §23.1 text block, which is a real readability gain
   worth the inconsistency. Noted only so that a future formatter pass
   (yamllint, prettier) is recognized as a style change, not a fix.

8. **`ci.yml:41` — `name: build ${{ matrix.goos }}/${{ matrix.goarch }}` on a
   matrix job.** Documented idiom; the auto-generated job name would be just
   as clear. Not load-bearing.

9. **`ci.yml:15-20` — the comment is six lines for a two-job workflow.** It is
   the densest prose in the file relative to the code it explains, and it now
   carries the same two facts twice: "GOPROXY=off is not a socket sandbox" and
   "the runtime offline guarantee belongs to §22.3/§22.4". The rewording is
   correct (see below); the observation is only that the surviving
   non-derivable content is two sentences, not six.

### Fail-closed

10. **`ci.yml:24-25,52-53` — the `GOPROXY=off` forward commitment, assessed.**
    The direction of failure is the safe one: a module that cannot be resolved
    from the local store is a hard error, so a check that cannot run fails
    rather than quietly downloading and passing. Measured on a **cold**
    module cache today (`GOMODCACHE=<empty tmp> GOPROXY=off go build|go
    test|go vet ./cmd/... ./internal/...`): all three exit 0, because no
    committed package imports a third-party module and Go's module-graph
    pruning never needs the four declared requirements for these builds. So
    the flag is not a trip-wire today. The first real dependency changes
    that: it must land with a committed `go.sum` in the same change, or the
    `test` and `build` jobs stop with a module-lookup error instead of
    downloading. `go.mod` declares four direct requirements and there is no
    `go.sum` in the tree, so that moment is close. This is a coupling the
    next author has to know about; the comment at `ci.yml:16-20` explains the
    mechanism but not this obligation. Observation, not a change I am making.

11. **`ci.yml:11-12` — `permissions: contents: read` at workflow level is the
    fail-closed posture**, and both jobs inherit it. No job widens it.

12. **No skip paths.** No `if:`, no `continue-on-error`, no `|| true`, no
    `exit 0` anywhere in the file (grepped). A failing command fails its step,
    fails its job, and fails the workflow. There is no construct in this file
    that can let a check silently pass.

13. **`ci.yml:28-30` — the module cache is implicit, not declared.**
    `actions/setup-go@v5` enables its built-in module/build cache by default
    when a `go.sum` is present and leaves it off when none is found. Today the
    cache is therefore silently inactive, and the day a `go.sum` lands it
    silently becomes active without the workflow changing by one character.
    Combined with finding 10, a cache miss then surfaces as a `GOPROXY=off`
    failure whose error text points at the proxy rather than at the missing
    cache entry. Declaring it (`cache: true` or `cache: false`) would make an
    environment-dependent behavior explicit. This is prior finding 9,
    re-confirmed against the current file.

14. **`ci.yml:27,55` — actions are tag-pinned, not SHA-pinned.** A retagged
    upstream tag changes what executes in CI with no change in this
    repository. The job-level `GOPROXY: "off"` does not constrain what the
    runner fetches for the action itself. This is prior finding 14. It is
    recorded rather than raised as a blocker because FJ-004 classifies as
    `metadata` and does not require the hardener stage, so no stage in this
    item's ladder currently owns the pinning decision. That ownership gap is
    the finding, not the tag.

15. **No `timeout-minutes` and no `concurrency` anywhere in the file**
    (grepped). A hung job holds a runner until the platform default; a
    superseded push on a long-lived branch runs both revisions to completion.
    Both are real costs and both are one line each, but neither is required by
    §22 or §23 and neither bites while the pipeline is three `go` commands.
    This is prior finding 15, re-confirmed.

### Credential surface

16. **None.** No `secrets.*`, no `API_KEY`, `api_key`, `token`, no
    `environment:`, no `pull_request_target`, no outbound host or URL anywhere
    in the file (all grepped, no match). Top-level keys are exactly `name`,
    `on`, `permissions`, `jobs`. The only credentials in play are the
    per-run `GITHUB_TOKEN` that `actions/checkout` mints from
    `permissions: contents: read`; no long-lived credential is introduced. A
    normal change reaches the same required checks with the secret store
    empty and no funded account.

## Applied changes 1-4: did they land, and are they correct?

Re-derived, not assumed.

1. **Comment reworded — landed, correct.** `ci.yml:15-20` now claims only what
   `GOPROXY=off` can support: module downloads are refused, the flag "is a
   module-proxy switch, not a sandbox: it does not close outbound sockets", and
   the runtime-path offline guarantee is deferred to §22.3/§22.4. The
   overclaim ("never reach the network") is gone. One follow-on: the file
   header at `ci.yml:4-5` says "The only network use here is fetching the Go
   toolchain named in go.mod", which is true on a cold cache but stops being
   true the day the implicit cache in finding 13 is active.
2. **`GOPROXY` hoisted to job level on both jobs — landed, correct.** Job env
   at `ci.yml:24-25` (`test`) and `ci.yml:52-53` (`build`). Parsed result: both
   job `env` maps are exactly `{'GOPROXY': 'off'}`, and the only step that
   declares an `env` is the build step at `ci.yml:61-64`
   (`CGO_ENABLED`, `GOOS`, `GOARCH`). Job-level `env` applies to every step in
   the job, and a step-level `env` is merged with the job map rather than
   replacing it, so `go vet` is now covered and the cross-compile variables
   survive. `go vet ./...` no longer runs with a different module-download
   posture from the tests it follows.
3. **Dead `GOFLAGS: -mod=readonly` removed — landed, correct.** `grep -n GOFLAGS`
   returns nothing. `-mod=readonly` has been the Go default since 1.16 and
   `go.mod` declares `go 1.26`, so the line restated the toolchain; the
   `GOPROXY` line is the load-bearing one.
4. **Prior finding 3 (per-step env block duplicated across the two test
   steps) — resolved as a consequence of 2.** The block now exists once per
   job and no step-level copy of `GOPROXY` remains, verified by parse.

## Open questions

1. §23's baseline list names `config/schema validation` and `integration
   tests` as lines distinct from `go test ./...`. The coder folded both into
   `go test ./...` on the argument that neither exists as a separate command
   in phase 0 and both will be ordinary `go test` packages when they land. The
   specifier explicitly declined to make either a falsifiable acceptance
   criterion. This is a spec-traceability call, not a simplification call, and
   the specifier's reading is defensible — raised here only so QA sees it, not
   because it blocks anything.
2. Who owns SHA pinning (finding 14) now that FJ-004 is classified `metadata`
   and therefore does not run the hardener stage. This is a work-item-planning
   question for the PM, not a change to this file.
