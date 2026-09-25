# Acceptance catalog

Traceability index from work-item `acceptance` entries back to the normative
specification.

## Rules

- The specification is normative. This catalog is an index; if the two disagree,
  the specification wins and this file is corrected.
- IDs are stable and never reused. Retired IDs stay listed as retired.
- A work item's `acceptance` references IDs from here and may add item-specific
  prose, but must not silently redefine a criterion.
- Golden vectors `C-GOLD-*` correspond one-to-one with specification §44 and must
  become executable tests.

## C-MATCH — stub selection (§40)

| ID | Criterion | Spec |
|----|-----------|------|
| C-MATCH-001 | Among matching stubs, larger `priority` is selected first. | §40.2 |
| C-MATCH-002 | Equal priority is broken by smaller `registrationIndex`; there is no specificity scoring. | §40.2, §44.6 |
| C-MATCH-003 | An equal-priority dynamic stub never displaces an earlier-registered static stub; override requires greater priority. | §40.9, §44.12 |
| C-MATCH-004 | Only stubs whose `profile` equals the active profile are candidates. | §40.1 |
| C-MATCH-005 | All present `when` fields combine with logical AND; an omitted field is a wildcard. | §40.1 |
| C-MATCH-006 | `when.questions` requires exact question-name set equality and exact per-name type equality. | §40.5, §44.5 |
| C-MATCH-007 | `when.model` matches by exact string equality; no aliases or regex. | §40.4 |
| C-MATCH-008 | `when.state` uses JSON value equality (`1`, `1.0`, `1e0` equal) without losing integer precision. | §40.3 |
| C-MATCH-009 | `instructions` and criteria contents never participate in matching. | §40.5 |
| C-MATCH-010 | An unmatched data-plane request returns HTTP 501 `fake_jev_unmatched_request` and fails verification. | §43.1 |

## C-SEQ — sequences and expectations (§40.6–40.7, §12.7)

| ID | Criterion | Spec |
|----|-----------|------|
| C-SEQ-001 | The selected stub's invocation count increments exactly once, before response selection. | §40.6 |
| C-SEQ-002 | The count stays incremented when response generation later fails. | §40.6, §44.7 |
| C-SEQ-003 | Sequence position starts at zero and advances once per selected invocation. | §40.7 |
| C-SEQ-004 | The invocation after the final element returns 409 `fake_jev_sequence_exhausted`, fails verification, and still counts. | §40.7, §43.3, §44.7 |
| C-SEQ-005 | There is no implicit repeat-last behavior. | §40.7 |
| C-SEQ-006 | Reset rewinds every static sequence to element zero and removes all dynamic stubs. | §41.10, §44.8 |
| C-SEQ-007 | `expect` accepts exactly-one-of `exactly` or (`atLeast`+`atMost`), with non-negative counts and `atLeast <= atMost`. | §12.7 |
| C-SEQ-008 | An unsatisfied invocation expectation fails verification with `expect_exactly` / `expect_at_least` / `expect_at_most`. | §41.9 |

## C-CFG — configuration (§12, §38)

| ID | Criterion | Spec |
|----|-----------|------|
| C-CFG-001 | `schemaVersion` is required and equals integer `1`. | §38.1 |
| C-CFG-002 | Unknown keys fail validation at top level, `server`, `limits`, stub envelope, `when`, `then`, and `expect`. | §12.3, §38.8 |
| C-CFG-003 | `state`, question content, and `then.raw.body` stay open JSON containers exempt from unknown-key rejection. | §38.8 |
| C-CFG-004 | The `jev` alias normalizes to `jev/v1` before duplicate detection; unknown profiles fail. | §38.1 |
| C-CFG-005 | Duplicate static stub IDs fail configuration validation. | §11.2, §15.2 |
| C-CFG-006 | A stub defines exactly one of `answers`, `sequence`, `raw`; two or more fail. | §12.6, §38.7 |
| C-CFG-007 | Omitted sections take the §12.2 built-in defaults. | §12.2 |
| C-CFG-008 | Any mode other than `strict` fails validation. | §12.2, §14 |
| C-CFG-009 | YAML and JSON inputs deserialize to the same logical schema. | §12 |
| C-CFG-010 | Limit ranges, model object keys, unique model names, and at-least-one-model rules hold. | §38.3, §38.4 |
| C-CFG-011 | Precedence is CLI flag > config file > default, with no environment override. | §12.1 |

## C-JEV — `jev/v1` wire behavior (§13, §39)

| ID | Criterion | Spec |
|----|-----------|------|
| C-JEV-001 | `GET /v1/models` returns the exact default body, or configured models in configuration order. | §39.1, §44.1 |
| C-JEV-002 | A successful models request is journaled as `profile: jev/v1`, `operation: models`, outcome `matched`, `matchedStubId: null`, `requestBody: null`. | §39.1 |
| C-JEV-003 | Validation order is `state`, `model`, `questions`, then question names lexicographically; only the first error is returned. | §39.2 |
| C-JEV-004 | Malformed JSON returns the exact 422 `json_invalid` envelope. | §39.3, §44.10 |
| C-JEV-005 | A missing required field returns the exact 422 `missing` envelope with its `loc`. | §39.3, §44.11 |
| C-JEV-006 | An unsupported question type returns `literal_error` at `body/questions/<name>/type` with message `Unsupported question type`. | §39.3 |
| C-JEV-007 | `choice` requires a non-empty criteria object and `score` a non-empty criteria array; `noul` criteria are optional. | §39.2 |
| C-JEV-008 | The answer map exactly covers the request question-name set; anything less is `fake_jev_invalid_stub_response` (HTTP 500). | §13.4, §13.5, §43.4 |
| C-JEV-009 | Response `model` is `then.model` when present, otherwise the request `model` echoed. | §13.6, §39.4 |
| C-JEV-010 | Usage defaults to exactly `{input_tokens: 0, output_tokens: 0}`; configured usage is non-negative integers. | §10.3, §13.6 |
| C-JEV-011 | `noul: true/false` converts exactly to `1.0`/`0.0`; probabilities stay in `[0,1]`. | §13.1 |
| C-JEV-012 | Omitted choice probabilities generate a one-hot over all request criteria keys; omitted confidence defaults to `1.0`. | §13.2, §44.3 |
| C-JEV-013 | Explicit choice probabilities cover exactly the criteria key set, lie in `[0,1]`, sum to `1 ± 1e-6`, and the selected choice is a maximum; omitted confidence equals the selected probability. | §13.2 |
| C-JEV-014 | Integer scores produce one-hot probabilities; fractional scores interpolate linearly between floor and ceiling; the legend is derived from request criteria. | §13.3, §44.4 |
| C-JEV-015 | Explicit score probabilities use keys `"0"`..`"N-1"`, sum to `1 ± 1e-6`, and have expected value equal to the score within `1e-6`. | §13.3 |
| C-JEV-016 | A helper whose type does not match the request question type fails as a fake-server error; types are never coerced. | §13.1, §13.4 |
| C-JEV-017 | Missing, `Bearer fake`, and arbitrary bearer tokens are accepted identically, and the value is never logged. | §10.1, §39.5, §42.5 |
| C-JEV-018 | Raw responses accept status `100..599`, string-to-string headers, JSON-serialized body, and default `Content-Type: application/json`. | §39.7 |
| C-JEV-019 | Data-plane routes are exact: query string ignored, trailing slash and wrong method are unknown routes. | §39.8, §44.16 |
| C-JEV-020 | Every non-raw JSON data-plane response carries a JSON `Content-Type`. | §39.9 |

## C-HOST — HTTP host and limits (§17.4, §19.4, §43)

| ID | Criterion | Spec |
|----|-----------|------|
| C-HOST-001 | Every admitted data-plane request is assigned the next interaction sequence before routing, decode, or validation. | §17.4 |
| C-HOST-002 | A request's assign → decode → select → mutate → record transition is serialized against other requests and control mutations. | §17.4 |
| C-HOST-003 | A reset racing a request yields only one of the two legal orderings; partially reset state is impossible. | §17.4 |
| C-HOST-004 | A data-plane 413 is journaled as `payload_too_large` with `requestBody: null`, mutates no stub state, and fails verification. | §19.4, §43.7, §44.15 |
| C-HOST-005 | Journal exhaustion returns 507, assigns no sequence, mutates no stub state, and sets a sticky `journal_full` failure with `requestSequence: null`. | §19.4, §43.5, §44.14 |
| C-HOST-006 | Control-plane requests never enter the journal; a control-plane 413 never affects verification. | §11, §41.1 |
| C-HOST-007 | An unknown data-plane route returns 404 `fake_jev_unknown_route`, is journaled, and fails verification. | §43.2 |
| C-HOST-008 | Default body, journal, log-preview, and shutdown limits match §12.2. | §12.2, §19.4 |
| C-HOST-009 | Engine and control state are race-free under `go test -race`. | §17.4, §32 |
| C-HOST-010 | Engine packages import neither `net/http`, `os/exec`, CLI, control, nor `compat/jev`. | §7.1, §17.3 |

## C-CTRL — control API (§41)

| ID | Criterion | Spec |
|----|-----------|------|
| C-CTRL-001 | `GET /__fake/v1/health` returns the exact health shape with status 200. | §41.2 |
| C-CTRL-002 | `GET /__fake/v1/meta` returns the exact meta shape; `activeProfiles` preserves configured order. | §41.3 |
| C-CTRL-003 | Creating a dynamic stub returns 201 with `id` and `registrationIndex`; a duplicate ID returns 409 `fake_jev_duplicate_stub_id`. | §41.4 |
| C-CTRL-004 | `GET /__fake/v1/stubs` returns the exact item shape ordered by registration index ascending. | §41.5 |
| C-CTRL-005 | `DELETE /__fake/v1/stubs` returns 204 and removes only dynamic stubs, leaving static counters untouched. | §41.6 |
| C-CTRL-006 | `GET /__fake/v1/requests` returns the exact record shape, valid `outcome` values, sequence-ascending order, and the defined `requestBody` rules. | §41.7 |
| C-CTRL-007 | `DELETE /__fake/v1/requests` returns 204, clears history and interaction-derived failures, resets the sequence to 1, and leaves stub counters alone. | §41.8 |
| C-CTRL-008 | `GET /__fake/v1/verify` always returns 200 with the exact pass/fail shapes, fixed failure item shape, required codes, and defined ordering. | §41.9 |
| C-CTRL-009 | `POST /__fake/v1/reset` returns 204 and performs all six reset effects atomically, restoring the next dynamic index to `N+1`. | §41.10, §44.8 |
| C-CTRL-010 | Invalid control JSON and profile/config validation failures return 400 with the defined error codes. | §41.1 |
| C-CTRL-011 | Every control JSON response uses `Content-Type: application/json` except 204 responses. | §11 |

## C-CLI — CLI, lifecycle, exit codes (§15, §42)

| ID | Criterion | Spec |
|----|-----------|------|
| C-CLI-001 | Exit codes are 0 success, 2 usage/config/startup/control/internal, 3 verification failed; `verify` returns 3 when `passed: false`. | §42.1 |
| C-CLI-002 | `run` follows the child/verify precedence table, preserves a nonzero child code, and reports both failures when both occur. | §42.2, §44.13 |
| C-CLI-003 | `run` always sets `FAKE_JEV_URL`; `--url-env NAME` additionally overrides that child variable. | §15.4 |
| C-CLI-004 | `run` uses an ephemeral port unless `--port` is passed; `serve --port 0` requests an ephemeral port. | §15.1, §15.4 |
| C-CLI-005 | `validate` performs all static checks without starting a server or making any network request. | §15.2 |
| C-CLI-006 | The ready file is written atomically with the exact fields and is removed on shutdown only when its `pid` still matches. | §15.1, §42.4 |
| C-CLI-007 | `serve` handles SIGINT/SIGTERM with graceful shutdown within `gracefulShutdownSeconds`; `run` forwards signals to the child. | §42.3 |
| C-CLI-008 | `verify` calls `GET /__fake/v1/verify` and duplicates no verification logic. | §15.3 |
| C-CLI-009 | `version` prints product version, Go build information, built-in profile IDs, control API version, and config schema version. | §15.5 |
| C-CLI-010 | Operational logs go to stderr, consumable data may go to stdout, and Authorization values are never logged. | §42.5, §20 |

## C-ARCH — architectural invariants (§4, §30, §33)

| ID | Criterion | Spec |
|----|-----------|------|
| C-ARCH-001 | The engine contains no `choice`, `noul`, `score`, route, or schema knowledge. | §4.2, §30.1 |
| C-ARCH-002 | All Jev behavior lives in the compatibility profile, which is registered at compile time. | §9, §9.1 |
| C-ARCH-003 | No feature listed in §33 deferred decisions exists in v1. | §33 |
| C-ARCH-004 | Normal execution performs no network egress and no model or provider call. | §4.9, §19.2 |
| C-ARCH-005 | Fixtures are data only: no templates, expressions, shell, JavaScript, or code execution. | §12.8, §19.3 |
| C-ARCH-006 | The default bind address is `127.0.0.1`. | §19.1, §37 |

## C-GOLD — golden contract vectors (§44)

| ID | Vector | Spec |
|----|--------|------|
| C-GOLD-001 | Default model list response | §44.1 |
| C-GOLD-002 | Noul answer | §44.2 |
| C-GOLD-003 | Choice one-hot generation | §44.3 |
| C-GOLD-004 | Fractional score generation | §44.4 |
| C-GOLD-005 | Exact question set prevents accidental reuse | §44.5 |
| C-GOLD-006 | Priority then registration order | §44.6 |
| C-GOLD-007 | Sequence exhaustion | §44.7 |
| C-GOLD-008 | Reset | §44.8 |
| C-GOLD-009 | Configured raw error is not a fake failure | §44.9 |
| C-GOLD-010 | Malformed request | §44.10 |
| C-GOLD-011 | Missing state | §44.11 |
| C-GOLD-012 | Dynamic stub does not implicitly override static | §44.12 |
| C-GOLD-013 | `run` exit precedence | §44.13 |
| C-GOLD-014 | Journal limit | §44.14 |
| C-GOLD-015 | Payload too large | §44.15 |
| C-GOLD-016 | Exact routes | §44.16 |

## C-QUAL — repository quality gates (§22, §23, §45)

| ID | Criterion | Spec |
|----|-----------|------|
| C-QUAL-001 | `go test ./...` passes. | §31, §23 |
| C-QUAL-002 | `go test -race ./...` passes on supported runners. | §23 |
| C-QUAL-003 | `go vet ./...` passes. | §31, §23 |
| C-QUAL-004 | Fuzz targets show malformed untrusted input yields a controlled error and never a panic or corrupted state. | §22.2 |
| C-QUAL-005 | Golden contracts run offline with no credentials, model, or network. | §23 |
| C-QUAL-006 | Required machine-readable contract artifacts are committed and reviewable. | §45 |
