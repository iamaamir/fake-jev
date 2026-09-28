---
stage: hardener
task: FJ-057
inputFingerprint: d94f4df07a2cb28ec7dbfd79d3d11099aafe1556e42c8ffbe534b7bde16b4dc7
outputFingerprint: d94f4df07a2cb28ec7dbfd79d3d11099aafe1556e42c8ffbe534b7bde16b4dc7
taskFingerprint: 5c1f809537db9d74755d771cc94844fac139f0a8ab2b524c031338b14ccf7986
gitHead: 8d34f6b
generatedAt: 2026-09-28T11:53:09Z
author: subagent/guard-1
---

Floors: untraced is zero-tolerance with activation site as finding path;
unknown_id is enforced on _test.go markers only with absent/rowless catalog
as exit-2 tool errors; testdata tags never fail catalog validation; dot-dirs
and root-anchored skipDirs stay outside the walk; markers in files go test
never executes (testdata/ outside contracts, _-prefixed paths) can neither
cover an active ID nor raise unknown_id. No C-ID claims are made.
