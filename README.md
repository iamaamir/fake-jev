# fake-jev

Deterministic, zero-model fake server implementing a strict `jev/v1` compatibility
profile over HTTP + JSON, driven by configuration, with a control API and golden
contract vectors. It lets clients exercise provider-shaped request/response behavior
without a model, an API key, or outbound network access.

**Status: foundation phase.** No product implementation has begun. This repository
currently holds the specification, repository conventions, task state, and
verification tooling.

- Authoritative specification: `spec/fake-jev-technical-spec-v2.md`
- Repository rules for agents: `AGENTS.md`
- Agent workflow: `docs/development/agent-workflow.md`
- Work state and evidence: `.agent/`

## Verify

```bash
./scripts/verify-candidate            # repository foundation checks
./scripts/agent-context <task-id>     # print a bounded task packet
```

`verify-candidate` exits `0` only when every mandatory check passes. Product checks
(`go test ./...`, `go vet ./...`, `go build ./cmd/fake-jev`) are added centrally as
implementation phases land.
