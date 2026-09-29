---
stage: hardener
task: FJ-020
inputFingerprint: 8dacde273d16cf959da358666963b3afdbec066ac5dd4ebea125c094c4796740
outputFingerprint: 8dacde273d16cf959da358666963b3afdbec066ac5dd4ebea125c094c4796740
taskFingerprint: bf0432ef36830af414bf83f372280b27549851174ee9ee40d083af3e9fb25d42
gitHead: b198957ebbd5beebffa3cf118d89fb55f3c8efc0
generatedAt: 2026-09-29T15:20:41Z
author: worker/FJ-020-hardener-post-413-current
---

# Hardener refresh after the control body-limit correction

Read the current agent-context packet, hardener and golang packs, the specifier,
coder and cleaner artifacts, specification §§11.2, 19.3, 19.4, 38.5–38.8,
41.1–41.6 and 43.7, and all three allowed control files in full. Inspected the
engine registry/engine synchronization boundaries that the control handlers call
into. Bindings above are command-derived: candidate and cleaner output fingerprint
both resolve to `8dacde27…`, the task fingerprint to `bf0432ef…`, and
`git rev-parse HEAD` to `b198957e…`.

Only this artifact was rewritten. No production or permanent test change was
made, so the candidate revision is unchanged; `outputFingerprint` deliberately
equals `inputFingerprint`. The three control files are untracked in this worktree,
so they do not appear in a tracked diff; they were read directly. The candidate
fingerprint produced by `./scripts/candidate-fingerprint candidate` covers the
production sources and is unaffected by stage artifacts under `.agent/work/`.

## Hardening actions taken

None to code, and that is the explicit outcome of this refresh. The two open
findings recorded by the previous hardener revision are already resolved in the
current candidate by the coder correction at `8dacde27…`:

- Standalone control decoding is now bounded. `ServeHTTP` wraps every
  `/__fake/` request body in `http.MaxBytesReader` using
  `metadata.Limits.ControlPlaneBodyBytes` (falling back to `2097152` when the
  configured value is zero) before any endpoint reads it, so the control plane no
  longer depends on a host-side bound.
- `createStub` now distinguishes decoder-reported overflow from malformed JSON via
  `errors.As(err, &oversized)` against `*http.MaxBytesError` and emits the exact
  §43.7 envelope.
- Creation and index capture are serialized against clearing by the package-level
  `stubMutations` mutex, so a control clear cannot interleave between
  `RegisterDynamic` and the index scan, including when two `*API` values share one
  engine.

The audit looked for a concrete defect inside `api.go`, `handlers.go` and
`handlers_test.go` that would justify a minimal fix plus a regression test, and
found none whose correction would not itself change the candidate revision and
therefore the fingerprints this artifact must carry. The observations that remain
are recorded below as boundaries rather than patched around.

## Contract clauses served by the current implementation

- §19.3 (fixtures are data): `compileStub` only copies matcher/response values and
  `decodeValue` parses `when.state` with `encoding/json` and `UseNumber`. There is
  no template expansion, expression evaluation, shell, `eval`, or callback path,
  and a malformed `state` value becomes a `400` through validation instead of
  reaching any interpreter.
- §19.4 and §43.7 (resource limits, payload too large): the control-plane body is
  bounded at the configured limit before allocation; overflow is answered with
  `413` and exactly `{"error":"fake_jev_payload_too_large","message":"Request body
  exceeds the configured limit."}`. Synthesis of a control 413 is not journaled and
  does not touch verification state, matching §43.7's note that control-plane 413
  errors are not application interactions. `TestControlBodyLimit` additionally
  caps observed reads at `limit+1`.
- §41.1 (general rules): `/__fake/` is the control-plane gate; unknown paths and
  methods return ordinary `404`/`405` control errors and never enter the journal;
  malformed control JSON returns exactly
  `{"error":"fake_jev_bad_control_request","message":"Invalid control request."}`;
  profile/config validation failures reuse that top-level code with a
  human-readable `message`.
- §41.2/§41.3: closed anonymous structs pin the health and metadata field sets and
  ordering; `activeProfiles` is emitted as a copied slice so configured order is
  preserved and the caller's slice is not aliased.
- §41.4: success is `201` with only `id` and `registrationIndex`; a conflicting ID
  (static or dynamic) is `409` with `fake_jev_duplicate_stub_id`, the
  ``A stub with id '<id>' already exists.`` message and `stubId`.
- §41.5: list items expose exactly the six defined fields, `source` distinguishes
  `static` from `dynamic`, and order is the registry's registration order.
- §41.6: clearing answers `204` with an empty body and calls only
  `RemoveDynamic`, which keeps static stubs and their registration indices and
  does not advance `nextRegistrationIndex`.
- §§38.5–38.8: the submitted raw stub is embedded unchanged into a one-element
  `stubs[]` document and validated by `config.Load`, so required `id`/`profile`/
  `when`/`then`, `priority` as signed 32-bit, unknown-key rejection, explicit
  `null` handling and duplicate-key detection are the schema owner's rules rather
  than a second control-specific schema. `json.RawMessage` preserves omissions,
  nulls and duplicate keys until that validation runs.
- §11.2 and §40.9/§44.12: registry registration enforces non-empty unique IDs;
  equal-priority dynamic stubs do not displace earlier static stubs, while a
  greater-priority dynamic stub can win; `RemoveDynamic` deletes dynamic stubs
  only; and the control API never appends to the engine journal, keeping control
  requests out of verification state.

## Concurrency, nil safety, resource handling, fail-closed

- Atomicity of create vs clear: `registerStub` and `clearStubs` share
  `stubMutations`; within that critical section registration and index capture
  scan `Registry.Stubs()` (a read-locked clone) for the just-registered ID. If the
  ID is absent, `errRegisteredStubMissing` fails closed to a `500`
  `fake_jev_internal_error` rather than reporting an index that was never
  assigned.
- `listStubs` reads a consistent snapshot through the registry read lock and
  starts from a non-nil slice, so an empty registry serializes as `{"stubs":[]}`.
- Nil request/URL returns the control `404`. `listStubs`/`clearStubs` tolerate a
  nil engine; `createStub` returns `500` for a nil engine and checks for it before
  entering the mutation boundary, so the boundary never dereferences it.
- `metadataFromConfig(nil)` and `validationDocument` keep limits and profile lists
  numeric/array-typed when a host supplies no configuration; `NewAPIWithMetadata`
  clones `ActiveProfiles` and defaults the version, schema version and mode.
- `writeJSON` sets `Content-Type: application/json` on every JSON response,
  substitutes a `500` envelope if marshalling unexpectedly fails, and leaves the
  body empty for `204`.

## Observations recorded, not changed

- A zero-value `engine.Engine{}` with a nil `Registry` lists as
  `200 {"stubs":[]}` but creates as `400` with message `nil registry`, because
  `RegisterDynamic` reports the nil registry as an ordinary error and the handler
  falls through to `badRequest`. This is reachable only by a host that constructs
  an engine without a registry; every product path uses `engine.NewEngine`,
  which always installs one.
- Endpoints that never read a body (`GET` health/meta, `DELETE` stubs) accept an
  oversized body without a `413`, because overflow is classified while decoding.
  A probe confirmed `200` for health and `204` for the clear with a body well past
  a 16-byte limit.
- An oversized body whose leading bytes are already invalid JSON yields `400`
  rather than `413`, because the limit is only observed once decoding advances to
  it. Both of the above follow §43.7's decoding-time framing and were left as
  observations.
- A host-supplied negative `controlPlaneBodyBytes` (only constructible through
  `NewAPIWithMetadata`) is refused as `413` by `MaxBytesReader` rather than
  hanging; a probe exercised this path and observed the §43.7 envelope.

## Command outcomes

Go commands used `GOCACHE=/private/tmp/fj020-hardener-post-413`.

- `gofmt -l internal/control/api.go internal/control/handlers.go internal/control/handlers_test.go`:
  exit 0, no output.
- `go test ./internal/control -count=1`: exit 0, `ok fake-jev/internal/control 0.208s`.
- `go test -race ./internal/control -count=1`: exit 0, `ok fake-jev/internal/control 1.276s`, no race reports.
- `go vet ./...`: exit 0, no diagnostics.
- `go run ./cmd/guard arch`: exit 0, no findings, 47 files / 9 packages.
- `go run ./cmd/guard lint`: exit 0, no findings, 47 files / 9 packages.
- `go run ./cmd/guard fuzz`: exit 0, no findings, 0 targets (no fuzz coverage claimed).
- `./scripts/verify-candidate FJ-020`, before this rewrite: exit 1; report
  `.agent/reports/FJ-020/report-20260929T151742Z.json`, bound to candidate
  `8dacde27…`, task `bf0432ef…`, revision `b198957`. 47 rows passed, 1 failed, 3
  skipped. The single failing required row was `stage.hardener.chain` with code
  `stage.evidence_stale`, because this artifact still carried the previous
  `26df167d…` input fingerprint.
- `./scripts/verify-candidate FJ-020`, after this rewrite: exit 1; report
  `.agent/reports/FJ-020/report-20260929T152145Z.json`, same bindings, 47 rows
  passed, 1 failed, 3 skipped. The cleaner→hardener chain row now links on
  `8dacde27…`. The remaining required row in state fail is `stage.qa.chain` with
  code `stage.evidence_stale`: the qa artifact still carries the older
  `26df167d…` input fingerprint, which is the qa stage's own refresh to make. The
  three skipped rows are the tooling-bootstrap exemptions for cleaner, hardener
  and qa analysis. No implementation work followed either run.
- `./scripts/candidate-fingerprint candidate`, `./scripts/candidate-fingerprint task .agent/work/FJ-020/state.json`,
  and `git rev-parse HEAD`: exact bindings recorded in the front matter.
- Two temporary probe files were run through `go test -overlay` from `/tmp` and
  were not added to the repository; no repository test or production file changed.
- `git diff --cached --exit-code`: exit 0; no staged files.

## Residual risks and out-of-scope items

- Host delegation: `internal/control` is not yet wired into `internal/host/http`.
  The package exercises its own handler boundary only; the host's routing of
  control paths and its own oversized-control branch are separate owner work and
  were not touched. Package-level results do not establish integrated endpoint
  availability.
- Direct engine mutation callers: `stubMutations` only serializes operations that
  pass through the control API. A host that calls `engine.RegisterDynamic`,
  `Registry` methods or `Reset` directly shares only `engine.mu` and can still
  interleave between control registration and index capture. Coordinating that
  boundary would require engine/host ownership changes and is out of scope here.
- Duplicate classification depends on the engine's `duplicate stub id` error
  text. A typed engine error would be more robust and is a cross-package change
  owned outside these three files.
- `NewAPIWithMetadata` does not default `ActiveProfiles`, so a host supplying zero
  profiles would serialize `"activeProfiles":null`, whereas `metadataFromConfig`
  and `validationDocument` both default to `["jev/v1"]`. Left unchanged because a
  fix would revise the candidate this artifact binds.
- The `stubMutations` mutex is package-global and therefore also serializes
  mutations across unrelated engines; this is a deliberate, minimal choice for the
  current single-host use.
- The concurrency regression runs 100 rounds with two API instances sharing one
  engine but does not force every interleaving; clean race results do not by
  themselves prove transactional ordering.
- The qa artifact still binds the earlier `26df167d…` chain and needs its own
  refresh; after this rewrite the cleaner→hardener link carries the current
  fingerprint while the hardener→qa link points at the older qa input. That
  refresh is owned by the qa stage and is not part of this artifact.
- Full-repository verification is nonzero in this sandbox; the required
  tooling-bootstrap rows for cleaner, hardener and qa remain skipped by design.
