# Triage Labels

The skills speak in terms of five canonical triage roles. This repo's tracker uses
the `status` vocabulary defined in `.agent/README.md`, so the roles map onto it.
When a skill mentions a role, use the corresponding value below.

| Label in mattpocock/skills | Label in our tracker                       | Meaning                                 |
| -------------------------- | ------------------------------------------ | --------------------------------------- |
| `needs-triage`             | `status: "planned"`                        | Maintainer needs to evaluate this issue |
| `needs-info`               | `status: "blocked"` + blocker `needs-info` | Waiting on reporter for more information |
| `ready-for-agent`          | `status: "ready"`                          | Fully specified, ready for an AFK agent |
| `ready-for-human`          | `status: "ready"` + `assignedRole` set     | Requires human implementation           |
| `wontfix`                  | `status: "blocked"` + blocker `wontfix`    | Will not be actioned                    |

The full status vocabulary (do not invent new values):

```text
planned -> ready -> implementing -> verifying -> review -> complete
                                   \-> blocked (from any state)
```

For `needs-info` and `wontfix`, record the reason in `blockers`, e.g.
`{"type": "needs-info", "description": "..."}`.

Edit this table if the team adopts different vocabulary — but keep it in sync with
the `STATUSES` set enforced by `scripts/verify-candidate`.
