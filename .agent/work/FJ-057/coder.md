---
stage: coder
task: FJ-057
inputFingerprint: d94f4df07a2cb28ec7dbfd79d3d11099aafe1556e42c8ffbe534b7bde16b4dc7
outputFingerprint: d94f4df07a2cb28ec7dbfd79d3d11099aafe1556e42c8ffbe534b7bde16b4dc7
taskFingerprint: 5c1f809537db9d74755d771cc94844fac139f0a8ab2b524c031338b14ccf7986
gitHead: 8d34f6b
generatedAt: 2026-09-28T11:53:09Z
author: subagent/guard-1
---

Implemented per plan Task 5: cmd/guard/trace.go + trace_test.go (13 trace
tests; package total 54 top-level / 71 leaf after the review fix round), the
trace handlers entry, three verify rows (na metadata, bootstrap skip,
required product/CI) and 3 selftest scenarios. The fix round (system_one
exclude-dead-dirs, 0.78) excluded testdata/ (contracts excepted) and
_-prefixed files/dirs from marker sources and fixed escaped-newline line
numbers; both adjudications are recorded in the feat commit 8d34f6b body and
the state.json spec-gap blocker. No C-ID claims are made.
