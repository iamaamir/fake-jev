---
stage: qa
task: FJ-003
inputFingerprint: 5fe4ba9be1430d49a566a75e1ca5e78d9b638b45cba746e9acfd54bd68bed9fc
outputFingerprint: 5fe4ba9be1430d49a566a75e1ca5e78d9b638b45cba746e9acfd54bd68bed9fc
taskFingerprint: f91008767970d165f99603537a03289a1e7a20598d44d203cf1151bb5781b8c2
gitHead: 7dbf861
generatedAt: 2026-09-26T10:21:58Z
---

# QA — FJ-003

## Class

`product` — `testdata/**` is classified as executable contract fixtures, so all
five stages are required and `go test` / `go vet` / `go build` are required
rather than exempt. That is deliberate: a golden vector defines product
behavior, so changing one must not slip through under the lighter policy.

## The central limitation, stated first

**No vector was executed.** This item delivers data; the runner is FJ-046. Every
check below is structural. A green row here means the corpus is well-formed and
internally consistent, not that fake-jev satisfies it. Nothing in this item is
evidence about the server's behavior, because there is no server yet.

| Check | Result |
|---|---|
| all 23 files parse as JSON | pass |
| required envelope keys present on every file | pass |
| every step has `request` and an `expect` with an exact `status` | pass |
| no duplicate vector ids | pass |
| all 16 `C-GOLD-001` through `C-GOLD-016` present | pass |
| no id outside the catalog uses the `C-GOLD-` prefix | pass (after rename) |
| every config top-level / stub / `when` / `then` key exists in `config.schema.json` | pass |
| every stub has exactly one of `answers` / `sequence` / `raw` | pass |
| every sequence element has exactly one of `answers` / `raw` | pass |
| every journaled `outcome` in the OpenAPI `InteractionOutcome` enum | pass |
| every `error` and failure-body code in `FakeFailureCode` | pass |
| every `/verify` failure `code` in `VerificationFailureCode` | pass |
| all eleven §45 concern directories present and non-empty | pass |
| no `timestamp` assertion | pass |
| no exact `/verify` message assertion | pass (sentinel used) |
| `go test`, `go vet`, `go build` | pass — no Go code added, so these confirm the corpus did not break the build |

## Trace coverage

- **C-GOLD-001 … C-GOLD-016** — one file per vector, each naming its own §44
  section plus the sections of §12, §13, §21, §39, §40, §41, and §43 it depends
  on.
- **C-QUAL-005** — every request is a relative path against an ephemeral
  loopback server. No vector needs a credential, a model, or a network, so the
  offline requirement holds by construction.
- **C-QUAL-006** — all eleven required directories exist with content. §45 lists
  `mixed/`; §44 has no mixed vector, which is the tension the specifier recorded.

## Spot checks read back against the specification

| Value | Vector | Spec |
|---|---|---|
| default model entry | `models/01` | §12.2, §39.1, §44.1 |
| fractional interpolation p(1)=0.5, p(2)=0.5 | `score/01` | §13.3 |
| derived legend keys `"0"`, `"1"`, `"2"` | `score/01` | §13.3 |
| one-hot over all criteria, confidence 1.0 | `choice/01` | §13.2 |
| invocation count 3 after exhaustion | `sequences/01` | §12.7, §44.7 |
| next dynamic index 3 for two static stubs | `reset/01` | §12.4, §44.8 |
| 507 assigns no sequence, mutates no counter | `verification/02` | §43.5, §44.14 |
| 413 receives sequence 1, `requestBody: null` | `verification/03` | §43.7, §44.15 |
| child exit 7 preserved with verify failing | `cli/01` | §42.2, §44.13 |

## What a reviewer should press on

- **Seven vectors are derived, not §44.** `VEC-MIXED-001/002`,
  `VEC-MATCH-004/006B`, `VEC-VALID-003/004`, `VEC-VERIF-004`. Each names a
  normative MUST outside §44 as its authority. If the orchestrator wants §44-only,
  these are the seven files to reject — but `mixed/` would then be empty and §45
  unmet.

- **The reset vector ends in `passed: false`.** That is §44.8's explicit
  consequence, not a defect in the fixture.

- **No message text is asserted for `/verify`.** §41.9 fixes the shape and the
  codes, not the prose. Data-plane failure bodies *do* have spec-fixed messages
  and those are asserted exactly, because §43.1 through §43.7 freeze them.

- **`matching/05` lives under `matching/`** even though it tests routing. §45
  fixes the directory list; the filename says what it is.
