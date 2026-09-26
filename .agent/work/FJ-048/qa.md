---
stage: qa
task: FJ-048
inputFingerprint: 7a30b543583fc4e4849e2b615c5256f499e0d4b92d555bc81dd0166e02860745
outputFingerprint: 7a30b543583fc4e4849e2b615c5256f499e0d4b92d555bc81dd0166e02860745
taskFingerprint: 6e08706ef6fdac9c4cbbde26998da0980ce56477778ee5bc59bc24eb2fbe3e86
gitHead: 1460b64
generatedAt: 2026-09-26T07:14:50Z
---

# QA — FJ-048

## Acceptance exercised

| Line | Check | Result |
|---|---|---|
| 1 | `git check-ignore -q fake-jev` | exit 0; `-v` reports `.gitignore:23:/fake-jev` |
| 2 | `scripts/verify-candidate` unchanged | byte-identical to the previous revision |
| 3 | two consecutive `verify-candidate FJ-001` runs | both exit 0 |

## The regression test, before and after

Before, on FJ-001:

```
run 1 -> PASS  37 passed, 0 failed
run 2 -> FAIL  36 passed, 1 failed
         stage.qa.terminal  stage.evidence_stale
         recorded 4d87f2d0… != current 1b4dafb2…
```

After, on the same item with the same code:

```
run 1 -> PASS  37 passed, 0 failed, 3 skipped
run 2 -> PASS  37 passed, 0 failed, 3 skipped
```

Between the two runs the binary is present on disk and `git status --porcelain`
stays empty, so the second run hashed an identical workspace and the evidence
chain held. That is the defect this item existed to remove, observed before and
after on the same evidence chain.

## Collateral that had to be handled

The `.gitignore` change is a real workspace change, so it invalidated FJ-001's
recorded evidence. That item was re-evidenced against the new fingerprint
before the runs above; its Go sources were not touched. Recorded here because it
is the general consequence of the design: any content change re-stales the
evidence of every completed item bound to the previous workspace, and re-running
the stages is the only honest way to satisfy the chain.

## Not covered

No automated regression test guards this, because a test that runs the
verifier twice and asserts on its own evidence chain would be a harness change
outside this item's `allowedFiles`. The guarantee is therefore procedural. If
that guarantee matters, it belongs in `scripts/selftest`, which already builds
fixture repositories for exactly this purpose.
