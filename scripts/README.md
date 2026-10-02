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

## Verification toolchain: `scripts/requirements-verify.txt`

Every script here is offline at runtime, but `verify-candidate` needs one Python
library: `check_role_packs` parses the role-pack front matter with PyYAML. That
dependency is pinned exactly in `scripts/requirements-verify.txt`:

```text
PyYAML==6.0.3
```

Why this version, and why an exact pin:

- `6.0.3` is the release the verification interpreter imports today and the
  current release on PyPI, so a fresh machine can satisfy the artifact instead
  of verification forcing an environment change.
- the pin is `==` rather than a range on purpose: a range would let a different
  importable version satisfy verification without being visible in the
  repository.

Install it with the interpreter verification uses, i.e. whatever
`command -v python3` resolves to:

```bash
python3 -m pip install -r scripts/requirements-verify.txt
```

Offline (no index access) — populate a wheelhouse first on a networked machine
whose platform and Python version match the target, then install from it:

```bash
python3 -m pip download --only-binary=:all: --no-deps \
  -d <wheel-dir> -r scripts/requirements-verify.txt
python3 -m pip install --no-index --find-links <wheel-dir> \
  -r scripts/requirements-verify.txt
```

Wheels are platform- and interpreter-specific, so a wheelhouse is reusable only
on matching machines; the `==` pin itself is platform-neutral.

`verify-candidate` never installs anything: installing the pin is the
environment's job. `.github/workflows/ci.yml` installs it into a job-local venv
before running the verifier (a stock `ubuntu-latest` `python3` is externally
managed, so a system-wide `pip install` is refused under PEP 668). When the
resolved interpreter cannot `import yaml`, the role-pack check fails closed as
`toolchain.pyyaml_missing` — it is never skipped.
