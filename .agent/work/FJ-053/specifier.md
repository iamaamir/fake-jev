---
stage: specifier
task: FJ-053
inputFingerprint: c4e916423dd448457485d308552f3c4570ace87a995a9fff2b02c2e2e8c68cbe
outputFingerprint: c4e916423dd448457485d308552f3c4570ace87a995a9fff2b02c2e2e8c68cbe
taskFingerprint: 83d86d767a7234b1ae2b171b0373044013e2c8821aa922631af2baeb80be6ba4
gitHead: 20a2c96
generatedAt: 2026-09-26T11:41:09Z
---

# Specifier — FJ-053

## What is delivered

One file changed: `go.mod`. The four third-party modules no product source
imports — `github.com/spf13/cobra`, `github.com/spf13/viper`,
`github.com/fatih/color`, `github.com/fsnotify/fsnotify` — plus the twelve
`// indirect` entries they drag in are removed, leaving the standard-library-only
posture §17.2 prefers and §15.6 calls for. No Go source, no test, no product
behavior changes. No `go.sum` is created.

This stage changes no product file, so input and output candidate fingerprints
are identical. Both were re-derived with `./scripts/candidate-fingerprint
candidate .` and `./scripts/candidate-fingerprint task
.agent/work/FJ-053/state.json`; they match the values in this front matter.

The change is deliberately sequenced *before* the first real import. The PM
note in `state.json` records why: FJ-004's CI runs `GOPROXY=off`, so an
imported-but-unhashed module would fail closed. This ticket removes that trap
rather than arming it.

## Class

`allowedFiles` is `go.mod` and `go.sum`. `scripts/candidate-fingerprint`
lists both in `PRODUCT_FILES`, so the class is **`product`**.

`derive_required_stages("product")` returns the full canonical ladder, so the
required stage set is **`specifier, coder, cleaner, hardener, qa`** — all five,
including `hardener`. This is stated explicitly because it is the one respect in
which FJ-053 differs from the FJ-004 precedent: FJ-004's single allowed file was
matched by `METADATA_PREFIXES` and dropped the hardener; FJ-053's cannot.

Consequence for the coder: this is a `product`-class change, so the hardener
stage is expected. There is no new runtime surface to harden, but the hardener
must still record a pass — most usefully by confirming the module graph is
closed and offline-resolvable (C5), which is the real risk surface here rather
than concurrency or input handling.

## Traces

C-QUAL-001, C-QUAL-003

Both ids exist in `docs/development/acceptance-catalog.md`, in section
`C-QUAL — repository quality gates (§22, §23, §45)`:

```text
| C-QUAL-001 | `go test ./...` passes. | §31, §23 |
| C-QUAL-003 | `go vet ./...` passes. | §31, §23 |
```

No other catalog id is claimed. C-QUAL-002 (`go test -race`) and C-QUAL-004
(fuzz targets) are not acceptance lines in `state.json` for this item and are
not claimed, though the cleaner and QA stages may note them if they are cheap to
run. The four plain-text acceptance lines in `state.json` have no catalog id;
they are specified as C3–C6 below and trace to §17.2, §15.6 and §31 directly.

## Criteria

Restated from `state.json.acceptance` in order, each as an observable check the
coder can satisfy and QA can falsify. Ids are stable — later stages refer to
these.

**C1 — `go test ./...` passes** (acceptance: `C-QUAL-001`; §31 Phase 0
acceptance block). The module builds and its packages test green at the
candidate revision. Falsifiable by running `go test ./...` and observing exit 0.

**C2 — `go vet ./...` passes** (acceptance: `C-QUAL-003`; §31 Phase 0
acceptance block). Note that `go vet` is the check that notices a malformed or
unsatisfiable module graph, so C2 is not redundant with C1 here. Falsifiable by
running `go vet ./...` and observing exit 0.

**C3 — `go.mod` declares no module that no product source imports** (plain-text
acceptance line; §17.2 "Prefer the Go standard library"). After the change, the
`require` blocks of `go.mod` account only for modules actually reachable from
the packages the Go tool lists. Falsifiable by
`go list -deps ./... | grep -v '^fake-jev'` (which must print nothing) and by
reading `go.mod`.

  *Scope directive the coder must follow — leave ZERO requires, not one.*
  `go.mod` after this change declares **no `require` directive at all**; the
  file is `module fake-jev`, a `go 1.26` directive, and nothing else. Do not
  pre-declare the YAML module that FJ-010 will need, even though §17.2 permits
  "One mature YAML dependency". Reasoning, so the decision is reviewable
  rather than asserted:

  - A pre-declared, unimported `require` is exactly the condition C3 forbids. It
    is a module no product source imports, so allowing it here would make the
    criterion untrue as written, and it would contradict the `state.json`
    non-goal "do not introduce any new dependency, including the YAML library
    FJ-010 will need".
  - It does not survive tooling. Verified in a scratch copy of the tree: with
    `require go.yaml.in/yaml/v3 v3.0.4` declared and no importer, `go mod tidy`
    removes it, leaving `module fake-jev` / `go 1.26` and no `go.sum`. A
    hand-pre-declared require is therefore not a stable state — the next
    `go mod tidy`, by a developer or by a future tidy check, silently reverts
    this ticket. A dependency posture that a routine command erases is not
    posture.
  - It would arm the exact trap this ticket exists to remove. Committing a
    `require` without its `go.sum` hashes leaves the module unhashed; FJ-004's CI
    runs `GOPROXY=off`, so the failure lands later and on a stranger's change
    than on this one. FJ-010 can introduce the YAML module deliberately, with
    `go get` and a committed `go.sum`, which is what the objective asks for
    ("so the first real dependency can be introduced deliberately with a
    committed go.sum").
  - FJ-010 is the item that owns the choice, and the specification is silent on
    *which* YAML module, so this stage must not prejudge it.

  Consequently no `go.sum` is created by this ticket: with zero requires there
  are no module hashes to record, and the non-goal "do not add go.sum entries
  for modules that are not imported" forbids inventing any. (Absence of
  `go.sum` is normal and tolerated by the toolchain for a dependency-free
  module; C5 confirms.)

**C4 — `go test ./...` and `go vet ./...` still pass after the change** (plain-text
acceptance line; §31). The conjunction, re-run at the candidate revision rather
than inherited from the base. This is the "no regression" half of the ticket and
is deliberately stated in `state.json` separately from C1/C2: C1 and C2 assert
the check exists and is green, C4 asserts the trimming did not break it. QA
must run both commands against the candidate, not against a remembered prior
result.

**C5 — `go build ./cmd/fake-jev` still succeeds with `GOPROXY=off`** (plain-text
acceptance line; §31 Phase 0 acceptance block, and the CI posture from FJ-004).
This is the criterion that proves the trimmed module graph is closed and
offline-resolvable without any `go.sum`. Falsifiable by
`GOPROXY=off go build ./cmd/fake-jev` and observing exit 0. Run it with a clean
module cache expectation in mind: with no external requires there is nothing to
download, so the build must not touch the network at all.

**C6 — the resulting `go.mod` contains no `cobra`, `viper`, `fatih/color`, or
`fsnotify` entry** (plain-text acceptance line; §17.2, §15.6). The four named
direct requires are gone, and — the part that is easy to miss — so are their
twelve transitive `// indirect` entries (`mapstructure/v2`,
`go-colorable`, `go-isatty`, `go-toml/v2`, `locafero`, `conc`, `afero`, `cast`,
`pflag`, `gotenv`, `go.yaml.in/yaml/v3`, `golang.org/x/sys`,
`golang.org/x/text`). A partial trim that drops the four direct requires but
leaves the indirect block would satisfy a naive reading of the objective while
still violating §17.2. Falsifiable by reading the whole file, not a grep for
four names:

```text
grep -E 'cobra|viper|fatih|fsnotify|mapstructure|colorable|isatty|go-toml|locafero|conc|afero|cast|pflag|gotenv|yaml|golang.org/x/' go.mod   # must print nothing
```

Note the name collision this regex also covers: `go.yaml.in/yaml/v3` and
`golang.org/x/sys` were present only as transitive dependencies of the four
removed modules. Their disappearance is required by C3, not by the non-goal
about FJ-010's future YAML library — they are a *different* module and must not
be re-added as a way of "keeping the YAML dependency".

Preservation, which the coder must not lose while trimming: the `module
fake-jev` path and the `go 1.26` directive stay exactly as they are. §31 Phase 0
requires "Go module with minimum Go 1.26"; a trim that also edits the language
directive is out of scope and a review finding. The Go directive is a *minimum*,
so no `toolchain` line is to be added either.

## Non-goals

Echoing `state.json.nonGoals`, with the supporting text:

- **No new dependency, including the YAML library FJ-010 will need.** §17.2
  permits "One mature YAML dependency"; permission is not instruction, and §15.6
  says a heavy CLI framework "is not required" and the implementation "SHOULD
  use the standard library or a very small parsing layer". The spec also says "Do
  not add frameworks for routing, dependency injection, logging, configuration,
  or testing unless a real need appears" — cobra, viper, fatih/color and
  fsnotify are exactly a CLI framework, a configuration framework, a terminal
  color library, and a file watcher, none of which has a real need in Phase 0.
  The golang-cli skill recommending Cobra+Viper is authority level 5 and loses
  to §15.6/§17.2 at level 1; the CLI uses the standard library `flag` package.
- **No `go.sum` entries for modules that are not imported.** With zero requires
  this resolves to: no `go.sum` file is added.
- **No CI workflow change.** `.github/workflows/ci.yml` is out of `allowedFiles`
  and is not touched. The `GOPROXY=off` environment it already sets is what C5
  exercises.
- **No product source change.** `cmd/` and `internal/` are out of
  `allowedFiles`. The only in-scope product edits are `go.mod` (and, not needed,
  `go.sum`). A coder who must touch Go source has left scope.

## Open questions

None. The specification constrains this item completely enough to be specified
without invention:

- §17.2 fixes the policy (standard library preferred, no frameworks, one YAML
  dependency *permitted*), and standard-library-only is strictly inside it.
- §15.6 fixes the CLI parser posture and confirms no CLI framework is required.
- §31 Phase 0 fixes the module layout and the three acceptance commands that
  C1, C2 and C5 restate.

The one question that looked open — whether this ticket may pre-declare FJ-010's
YAML module — is not a spec gap. It is answered from higher-authority artifacts
already in the work item: C3 forbids any module no product source imports, and
`state.json.nonGoals` forbids introducing the YAML dependency. The recommendation
is therefore recorded as a directive under C3 rather than escalated. The
specification is genuinely silent on *which* mature YAML module FJ-010 will pick,
but that choice belongs to FJ-010, not to FJ-053, and stating it here would be
inventing public behavior for a future item.
