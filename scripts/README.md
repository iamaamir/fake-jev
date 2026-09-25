# scripts/

Two deterministic, offline entry points. No network, no model calls, no services.

| Script | Purpose |
|--------|---------|
| `agent-context <task-id>` | Compile `.agent/work/<task-id>/state.json` into a concise bounded task packet for an implementation agent. |
| `verify-candidate [task-id]` | Run all mandatory repository checks; write JSON evidence to `.agent/reports/<task-id>/` when a task id is given. |

Both exit non-zero on failure. Product verification checks are added centrally to
`verify-candidate` as implementation phases land; do not fork per-task verification
scripts.

`agent-context` requires `python3` (used only to parse and format JSON, so malformed
state fails clearly instead of producing a misleading packet).
