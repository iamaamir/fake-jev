---
stage: specifier
task: FJ-030
inputFingerprint: 4acd6dd6c828af008d216a52f543aa571806355cf45bac0e71937bf5588d49ab
outputFingerprint: 4acd6dd6c828af008d216a52f543aa571806355cf45bac0e71937bf5588d49ab
taskFingerprint: d37327aa23e2afc94bc9b8bbc7d4c9cd3a87d61e03dc2cc2605c258b8b6c34a7
gitHead: abba008
generatedAt: 2026-09-30T14:45:01Z
author: worker/FJ-030-specifier
---

# Specifier — FJ-030

## Objective as bounded here

Make `fake-jev validate <path>` observable: one positional path, the file
loaded through the shared configuration surface, every statically checkable
rule run, nothing started, nothing contacted, and the §42.1 exit codes
emitted. The command currently does not exist: `internal/cli/dispatch.go`
routes only `version` and `help`, and the usage text still lists
`validate` as a later work item. The packet's scope amendment (state.json
`notes`) admits `internal/cli/dispatch.go` so the command is reachable;
`internal/cli/validate.go` and `internal/cli/validate_test.go` are the new
implementation and test files.

## Observable acceptance criteria

Each line is stated so a reviewer can falsify it by running the binary and
reading its streams and exit code.

**V1 — Command shape, usage failures, no interaction.**
`fake-jev validate <path>` is a recognized command taking exactly one
positional path. No path argument, more than one path argument, and an
unrecognized flag or unknown option all produce exit `2`, a diagnostic plus
the usage text on stderr, and no success line on stdout. The command never
prompts, never reads stdin, and terminates on its own when stdin is closed.
(§15, §15.2, §15.6, §42.1)

**V2 — Formats and the success message.**
For a path naming a readable file whose content is a valid configuration in
YAML or in JSON, `validate` exits `0` and writes a non-empty human-readable
success line to stdout. A JSON encoding and a YAML encoding of the same
logical configuration produce the same outcome (§12 "both MUST deserialize
to the same logical schema"). No violation diagnostic appears on stderr.
The success text itself is not a contract; the exit code is (§15.2).

**V3 — Inputs the loader rejects.**
A nonexistent path, an unreadable file, an empty file, malformed YAML,
malformed JSON, and content that is not a single document all produce exit
`2` with a stderr diagnostic naming the problem and no success line on
stdout. These are configuration/startup failures under §42.1, not a distinct
code. (§15.2, §42.1)

**V4 — Top-level schema violations.**
Exit `2` with a stderr diagnostic for each of: missing `schemaVersion`;
`schemaVersion` not integer `1`; any unknown top-level key; `mode` other
than `strict`; `compatibility` present but empty, containing a duplicate
after alias normalization, or naming an unknown profile; a `server`,
`limits`, `models`, or `stubs` value of the wrong JSON type. A document that
omits an optional section remains valid and receives the §12.2 defaults.
(§12.2, §12.3, §38.1, §38.8)

**V5 — Nested project-owned schema violations.**
Exit `2` with a stderr diagnostic for each of: unknown keys inside `server`,
`limits`, the stub envelope, `when`, `then`, or `expect`; out-of-range
`server.port`; out-of-range `limits` values; a model object missing
`name`/`description`/`release_date`, a duplicate model name, a non-empty
model list requirement, or a `release_date` that is not `YYYY-MM-DD`; a
stub without a non-empty `id`, without an enabled `profile`, or without
`when`/`then`. (§12.4, §38.2, §38.3, §38.4, §38.5, §38.8)

**V6 — Open containers stay open.**
A configuration whose `when.state`, `then.raw.body`, or emulated-provider
body carries arbitrary JSON keys is not rejected: it can exit `0`. These
containers are exempt from project-owned unknown-key validation. (§38.8)

**V7 — Stub response-form exactness.**
Exit `2` with a stderr diagnostic for each of: `then` carrying two or more
of `answers`/`sequence`/`raw`, or none of them; a `sequence` element that
carries another `sequence`; a `sequence` element carrying both `answers`
and `raw` or neither; `raw` with `model` or `usage` siblings; `raw.status`
outside `100..599`; a raw header value that is not a string; a `usage`
block missing or negative `input_tokens`/`output_tokens`. (§12.6, §12.7,
§13.6, §38.7)

**V8 — Duplicate IDs and expectation rules.**
Exit `2` with a stderr diagnostic for: a duplicate static stub `id` (§11.2
"a duplicate static stub ID MUST make configuration validation fail"), and
for each malformed `expect`: `exactly` combined with `atLeast`/`atMost`,
neither `exactly` nor a range, a negative count, `atLeast > atMost`, or an
unknown key. An absent `expect` is valid and asserts nothing. (§11.2,
§12.7, §15.2)

**V9 — Enabled profile-specific fixture data.**
The command runs every statically decidable fixture rule that the shared
validator exposes for each enabled profile, and surfaces those violations on
stderr with exit `2`. In v1 the only enabled concrete profile is `jev/v1`,
so this is the `jev/v1` fixture surface: `answers` maps are JSON objects,
`usage` blocks carry non-negative integers, `model`/`usage` sibling
placement, sequence element shapes, and the raw escape hatch. The parts of
§13.1–13.3 that are statically decidable but that no current code checks are
listed under "§15.2 static-check gaps" below; this artifact records that
mapping rather than assuming the gap is closed.

**V10 — No server, no network, no listener.**
A `validate` run opens no TCP listener, creates no socket, performs no DNS
or HTTP request, contacts no model or provider, writes no ready file, and
leaves no goroutine running after it returns. Observable: with outbound
network blocked and with no listener permitted, a valid file still exits `0`
and an invalid file still exits `2` with the same stderr diagnostic as an
unrestricted run, and no port is bound during either. (§15.2, §19.2)

## Existing coverage map (`internal/config`)

The command's static surface already exists; the CLI must route through it
rather than re-implement validation. `internal/config/load.go` defines
`Load([]byte)` and `LoadFile(path)`; `internal/config/validate.go` defines
`Validate(*Config)`, `validateDocument`, `validateValues` and the per-object
validators; `internal/control/handlers.go:285` already reuses
`config.Load(document)` for dynamic stub registration.

| Criterion | Existing coverage |
|-----------|-------------------|
| V2 (YAML/JSON same schema) | `Load` → `toJSON` (JSON fast path with duplicate-key rejection, else YAML decode + normalize + marshal) |
| V3 (undecodable input) | `toJSON`/`decodeYAML`/`requireEOF` reject empty input, multiple documents, trailing content; `LoadFile` reports read errors |
| V4 (top-level schema) | `Load`'s `DisallowUnknownFields` decode; `validateDocument` (required `schemaVersion == 1`, object-ness of `server`/`limits`/`stubs`/`models`); `applyDefaults`; `validateValues` (`mode == "strict"`, non-empty/unique compatibility via `normalizeProfile`) |
| V5 (nested schema, ranges) | `DisallowUnknownFields` over `ServerConfig`, `LimitsConfig`, `ModelConfig`, `StubConfig`, `WhenConfig`, `ThenConfig`, `ResponseConfig`, `RawResponse`, `UsageConfig`, `ExpectConfig`; `validateDocument` (model required fields, `when`/`then` required); `validateServer`, `validateLimits`, `validateModels`, `validateWhen`, `validateThen`, `validateResponse`, `validateRaw`, `validateUsage` |
| V6 (open containers) | `state` held as `json.RawMessage` and only checked to be string/object/array; `then.raw.body` held as raw JSON; no unknown-key rejection inside them |
| V7 (response-form exactness) | `validateThen`/`validateResponse` form counting; `validateThenDocument`/`validateResponseDocument` reject nested/typed violations; `validateRaw`, `validateUsage`, `rejectNullFields` |
| V8 (duplicate IDs, expect rules) | `validateValues` (non-empty, unique static stub `id`) and `validateExpect` (exactly-one-of form, non-negativity, `atLeast <= atMost`) |
| V9 (profile fixture data) | Partial: `StubConfig.Profile` must normalize to an enabled profile; `WhenConfig.Questions` values limited to `noul`/`choice`/`score`; `when.questions` must be non-empty; `then.answers` only checked to be a JSON object. Helper payloads are not inspected (see gaps) |
| V10 (no server/network) | No server, listener, or HTTP client exists on the configuration path; `config` imports only `os` for file reading |

Consequences for the CLI layer: `validate` owns only argument parsing,
`config.LoadFile` invocation, stream routing (success line to stdout,
diagnostics to stderr), and the exit code. Duplicating configuration rules in
`internal/cli` would fork the contract and is out of scope.

## §15.2 static-check gaps (recorded, not closed here)

§15.2 requires "validate all enabled profile-specific fixture data" and "all
statically possible consistency checks". The current loader/validator leaves
the following statically decidable rules unperformed:

- **G1 — `jev/v1` helper payloads are unvalidated.** Nothing inspects the
  contents of `then.answers` or `then.sequence[*].answers` beyond "is a JSON
  object". Statically decidable and currently unchecked: exactly one of
  `noul`/`choice`/`score` per question entry, with no other keys; the
  `legend` key prohibited in v1 score fixtures (§13.3 "Fixtures MUST NOT
  provide a separate `legend` in v1"); `noul` numeric (or boolean) and
  finite in `[0,1]` (§13.1); `choice` non-empty, `confidence`/`probabilities`
  values finite in `[0,1]`, explicit probability keys summing to `1.0`
  within `1e-6`, and the selected choice present in the explicit map and
  holding the maximum probability (§13.2); `score` finite, explicit
  probability keys exactly `"0"`–`"N-1"`, values finite in `[0,1]`, sum
  within `1e-6` of `1.0`, and probability-weighted expected value within
  `1e-6` of `score` (§13.3). Request-dependent rules — choice probability
  keys equalling the request criteria key set, score within `[0, N-1]`,
  omitted-probability one-hot/interpolation derivation, exact answer coverage
  of the request question set — are not decidable from the file alone.
- **G2 — Stub-local consistency between `when.questions` and `then.answers`.**
  `when.questions` is an exact question-name/type set matcher (§12.5,
  §40), so for a stub that supplies it, the answer key set and each helper
  type are statically comparable to that matcher. The current validator
  performs neither comparison. Whether §15.2's "all statically possible
  consistency checks" requires this cross-check is an interpretation of the
  specification, not something this artifact decides.
- **G3 — `then.model`/`then.usage` as siblings of `then.sequence`.** The
  typed schema and `validateThen` accept them; §38.7 grants optional sibling
  `model`/`usage` to an `answers` response and describes a `sequence`
  element as having "the same response shape". Recorded as a prose ambiguity
  rather than a claimed gap.

Any move to close G1/G2 changes validation strictness observed through
`config.Load` (shared by the control API's dynamic-stub path) and is
therefore a decision outside this artifact's scope.

## Exit-code contract

For `validate` (§42.1):

- `0` — the file loaded and every static check the validator performs
  completed without violation.
- `2` — CLI usage failure (missing/extra positional argument, unknown flag),
  configuration or schema violation, unreadable/undecodable input, or an
  internal operational failure while loading or validating.
- `3` is reserved for `verify` reporting `passed: false` and is never
  produced by `validate`.

## Non-goals

- No server startup, no listener, no ready file, no control API, no
  background process.
- No runtime matching, request validation, answer generation, sequence
  advancement, invocation counting, or verification-state behavior; the
  request-dependent §13 rules are out of scope at validate time.
- No model or provider call and no network access of any kind.
- No new configuration semantics or stricter rules beyond what
  `internal/config` already enforces; no duplicated validation logic in the
  CLI package.
- No new flags for `validate`, no stdin interaction, no prompts.
- No changes to dynamic stub registration, the control API, or the engine.

## Traces

- C-CLI-005 — §15.2 `validate` performs all static checks without starting a server or making any network request (V1, V9, V10).
- C-CLI-001 — §42.1 exit codes: `0` on success, `2` on usage/configuration/internal failure (V1, V3–V8; exit-code contract).
- C-CFG-001 — §38.1 `schemaVersion` required and equal to integer `1` (V4).
- C-CFG-002 — §12.3/§38.8 unknown keys fail validation at top level, `server`, `limits`, stub envelope, `when`, `then`, `expect` (V4, V5, V7).
- C-CFG-003 — §38.8 `state`, question content, `then.raw.body` remain open JSON containers (V6).
- C-CFG-004 — §38.1 `jev` alias normalizes before duplicate detection; unknown profiles fail (V4, V5).
- C-CFG-005 — §11.2/§15.2 duplicate static stub IDs fail configuration validation (V8).
- C-CFG-006 — §12.6/§38.7 a stub defines exactly one of `answers`, `sequence`, `raw` (V7).
- C-CFG-007 — §12.2 omitted sections take the built-in defaults (V4).
- C-CFG-008 — §12.2/§14 any mode other than `strict` fails validation (V4).
- C-CFG-009 — §12 YAML and JSON inputs deserialize to the same logical schema (V2, V3).
- C-CFG-010 — §38.3/§38.4 limit ranges, model object keys, unique model names, at-least-one-model (V5).
- C-SEQ-007 — §12.7 `expect` accepts exactly one of `exactly` or (`atLeast`+`atMost`), non-negative, `atLeast <= atMost` (V8).

This artifact records observable behavior and traceability only; it does not
declare an implementation outcome.
