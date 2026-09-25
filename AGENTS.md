# AGENTS.md — universal repository rules

Rules for every agent working in this repository. Keep this file short: it is injected
into most task packets. It does **not** replace the technical specification.

## Source of truth

Primary product specification (normative, authoritative for public behavior):

```text
spec/fake-jev-technical-spec-v2.md
```

- `spec/README.md` describes how spec, contracts, and tests relate.
- If a required behavior is not defined there, it is a **spec gap**. Record it in
  `.agent/work/<task-id>/state.json` under `blockers` with `type: "spec-gap"`.
  Never invent public behavior.
- `AGENTS.md` never overrides the specification.

## What this project is

`fake-jev` is a deterministic, zero-model fake server implementing a strict `jev/v1`
compatibility profile over HTTP + JSON, driven by configuration, with a control API
and golden contract vectors. Details live in the specification, not here.

## Universal architectural constraints

- The implementation is Go-first.
- The core engine must remain provider-neutral.
- Jev-specific behavior belongs only in the compatibility layer.
- HTTP + JSON is the primary interoperability boundary.
- Normal operation must never require a model or provider network call.
- Tests must be deterministic (no sleeps-as-sync, no network, no randomness without seeds).
- Strict behavior must fail closed.
- Do not implement deferred features unless the specification is intentionally changed.

## Change discipline

- Do not weaken tests or contracts to make an implementation pass.
- Do not redesign architecture without an explicit specification change.
- Do not introduce abstractions for hypothetical future requirements.
- Prefer the smallest implementation satisfying the current contract.
- Do not invent public behavior when the specification is silent.
- Edit only the files listed in the task's `allowedFiles`.

## Completion rule

No implementation task is complete until:

```bash
./scripts/verify-candidate
```

exits `0` for the candidate revision. During the foundation phase the verifier runs
repository/setup checks; product checks (e.g. `go test ./...`, `go vet ./...`,
`go build ./cmd/fake-jev`) are added centrally as phases land. Never claim success
without running it.

## Agent handoff rule

- Task context comes from `./scripts/agent-context <task-id>`, not from chat history.
- Persist progress to `.agent/work/<task-id>/state.json` (status, blockers,
  nextAction, verification result).
- Verification evidence is written by `./scripts/verify-candidate <task-id>` to
  `.agent/reports/<task-id>/` and is bound to the revision it verified.
- A fresh agent must be able to continue from the repository alone.

## Before you edit

1. Read the task packet from `./scripts/agent-context <task-id>`.
2. Read the referenced specification sections.
3. Confirm the change stays inside `allowedFiles` and `acceptance`.

## After you edit

1. Run `./scripts/verify-candidate <task-id>`.
2. Update `state.json` (`status`, `lastVerification`, `nextAction`, `blockers`).
3. Stop on failure or on any spec gap; do not improvise a public behavior.

## Agent skills

### Issue tracker

Work items live as `.agent/work/<task-id>/state.json` (durable, committed). See
`docs/agents/issue-tracker.md`.

### Team packs

Stage packs (workflow authority) and advisory expertise packs live in
`docs/agents/` — start at `docs/agents/README.md`, which records the authority
hierarchy (specification > acceptance > role pack > expertise > model
knowledge) and the advisory MUST-NOT rule.

### Triage labels

Mapped onto the `state.json` status vocabulary (`planned`, `ready`, `blocked`,
`complete`, ...). See `docs/agents/triage-labels.md`.

### Domain docs

Single-context: `CONTEXT.md` + `docs/adr/` at the repo root. See `docs/agents/domain.md`.
