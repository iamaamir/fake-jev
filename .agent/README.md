# .agent/ — durable work state and evidence

Plain JSON files. No database, no service. This directory is how agents hand work to
each other without chat history.

```text
.agent/
    work/<task-id>/state.json     durable work item (commit as needed)
    reports/<task-id>/*.json      verification evidence (commit when it matters)
    logs/                         transient scratch logs (gitignored)
```

## Work item: `.agent/work/<task-id>/state.json`

```json
{
  "id": "FJ-001",
  "title": "Example work item",
  "phase": "foundation",
  "objective": "Short concrete objective",
  "status": "planned",
  "baseRevision": null,
  "candidateRevision": null,
  "assignedRole": null,
  "leaseOwner": null,
  "attempts": 0,
  "specReferences": ["spec/fake-jev-technical-spec-v2.md § 40"],
  "relevantFiles": ["internal/engine/stub.go"],
  "allowedFiles": ["internal/engine/stub.go", "internal/engine/stub_test.go"],
  "acceptance": ["C-MATCH-001: priority DESC then registration order ASC"],
  "nonGoals": ["no matcher specificity scoring"],
  "verificationCommands": ["./scripts/verify-candidate FJ-001"],
  "lastVerification": null,
  "blockers": [],
  "nextAction": "implement stub ordering",
  "notes": []
}
```

Field meanings:

| Field | Meaning |
|-------|---------|
| `id` | Stable task id; matches the directory name. |
| `phase` | Which specification phase the item belongs to (e.g. `foundation`, `phase-1`). |
| `status` | One of the vocabulary values below. |
| `assignedRole` | Optional role label for the worker (e.g. `implementer`); informational only. |
| `leaseOwner` | Exclusive claim on this item by a worker identifier (e.g. `agent-7` or a session id). Required while `status` is `implementing`/`verifying`; must be unique across work items. `null` when unclaimed. |
| `baseRevision` / `candidateRevision` | Git revisions before/after the attempt; `null` when Git metadata is unavailable. |
| `specReferences` | Pointers into the specification. Never paste spec content here. |
| `relevantFiles` | Files an implementor should read. |
| `allowedFiles` | Files the implementor may modify. Entries are paths or `dir/*.go`-style globs; anything else is out of bounds. |
| `acceptance` | Observable criteria that define done. |
| `nonGoals` | Explicit exclusions that prevent scope creep. |
| `lastVerification` | Copy of the most recent verification result (see below). |
| `blockers` | Obstacles, including `{"type": "spec-gap", "description": "..."}`. |
| `nextAction` | The single next step for whoever picks this up. |

### Status vocabulary

```text
planned -> ready -> implementing -> verifying -> review -> complete
                                   \-> blocked (any state)
```

Keep it small. Do not invent a workflow engine.

### Lease

When several workers share one checkout, task state is the only thing that says who
owns what:

```text
claim:   set leaseOwner to your own identifier before your first edit
rule:    status implementing | verifying requires a non-empty leaseOwner
contention: two items claiming the same leaseOwner fails verification
release: set leaseOwner back to null when you hand the item off (review/complete/blocked)
```

`verify-candidate` enforces the two mechanical rules; the rest is discipline. A lease
is not a lock — it is a visible claim, so a second agent can see the collision instead
of discovering it in the diff. `./scripts/agent-context <task-id>` prints the current
lease alongside the orientation packet.

### Revision binding

`verify-candidate` records the revision it actually verified. Evidence for revision A
must never be presented as evidence for revision B. If Git metadata is unavailable,
the revision field is `null`/`unknown` and the evidence is correspondingly weaker.

## Verification evidence: `.agent/reports/<task-id>/`

One JSON file per verification run, written by `./scripts/verify-candidate <task-id>`:

```json
{
  "taskId": "FJ-001",
  "candidateFingerprint": "0000000000000000000000000000000000000000000000000000000000000000",
  "taskFingerprint": "0000000000000000000000000000000000000000000000000000000000000000",
  "policy": { "class": "product", "requiredStages": ["specifier", "coder", "cleaner", "hardener", "qa"], "source": "derived" },
  "revision": "abc123",
  "treeState": { "gitAvailable": true, "clean": true, "staged": 0, "unstaged": 0, "untracked": 0 },
  "startedAt": "2026-09-25T10:00:00Z",
  "finishedAt": "2026-09-25T10:00:02Z",
  "status": "failed",
  "checks": [
    { "name": "check_go_test", "gate": "G-C", "required": false, "result": "skipped",
      "exitCode": null, "code": "go.test_failed", "exempted": false,
      "command": "go test (repository packages)", "reason": "no go.mod yet (foundation phase)" },
    { "name": "check_work_item_schema", "gate": "G-C", "required": true, "result": "fail",
      "exitCode": 1, "code": "agent.work_item_schema", "exempted": false,
      "command": "work items match the state.json schema", "reason": null }
  ]
}
```

Every check carries a stable `code` and a `result` in
`pass | fail | skipped | not_applicable`. The run is acceptable (`status:
"passed"`) iff no check with `"required": true` is `fail` or a non-exempt
`skipped` (design §9.1) — `SKIPPED ≠ PASS`. `gate` names the check's gate
(`G-S`/`G-C`/`G-L`/`G-H`/`G-Q`); `exempted` marks a policy waiver
(`.agent/gate-policy.json`). `candidateFingerprint` is the §6.1 content digest
of the workspace; `taskFingerprint` and `policy` are `null` without a task id
and bound to the item when one is given. An agent fingerprints repeated
failures from `code` + `name` instead of scraping stderr.

`treeState` makes the evidence's coverage explicit. A report with
`"clean": false` describes revision `abc123` **plus** the listed dirty entries — the
reviewer must not treat it as evidence for the committed tree alone. `clean` is
`null` when no git metadata is available.

Reports are durable evidence another agent or CI can inspect. They are small; keep
them. If report volume ever becomes noisy, commit reports only for accepted
revisions and document that policy here rather than guessing.

## Logs: `.agent/logs/`

Transient. Ignored by Git. Never put state here that another agent needs.
