# Issue tracker: `.agent/work/` work items

Issues, PRDs, and implementation tasks for this repo are work items in this repo:

```text
.agent/work/<task-id>/
    state.json     the work item (durable, committed)
    PRD.md         optional planning document attached to the item
```

The work item is the unit of work. It is plain JSON — no database, no service.

## Conventions

- Task IDs are short and stable, e.g. `FJ-001`, `FJ-PRD-001`. The directory name
  must equal the `id` field; `scripts/agent-context` and `scripts/verify-candidate`
  both enforce this.
- One bounded vertical slice per work item (see `docs/development/agent-workflow.md`,
  "Task sizing"). A PRD or plan is attached as `PRD.md` beside `state.json` and is
  usually decomposed into child work items.
- Triage state is the `status` field (see `triage-labels.md` for the label mapping).
- Progress, obstacles, and handoffs go in `blockers`, `nextAction`, and `notes` —
  never in chat history alone.
- Verification evidence lands in `.agent/reports/<task-id>/`, written by
  `./scripts/verify-candidate <task-id>`.

## When a skill says "publish to the issue tracker"

Create `.agent/work/<new-task-id>/state.json` (and any attached document such as
`PRD.md` in the same directory). Fill every required field:

```text
id, title, phase, objective, status, acceptance,
allowedFiles, verificationCommands, nextAction
```

Validate it by running `./scripts/verify-candidate <new-task-id>`.

## When a skill says "fetch the relevant ticket"

Run `./scripts/agent-context <task-id>`, or read `.agent/work/<task-id>/state.json`
directly. The user will normally pass the task ID.

## What does not belong here

- Transient logs → `.agent/logs/` (gitignored).
- Scratch notes and tool state → `/.scratch/` (gitignored).
