---
stage: hardener
task: FJ-056
inputFingerprint: 60b820379466889f42cafda4874a8364c046011aaa582672ae8265616c46a6c1
outputFingerprint: 60b820379466889f42cafda4874a8364c046011aaa582672ae8265616c46a6c1
taskFingerprint: d2e7e5ac44a0cb958d227131df2d53e589b64acaea19cb731bf7616cd12fadf0
gitHead: 53b51d3
generatedAt: 2026-09-28T09:59:44Z
author: subagent/guard-1
---
Floors are fail-closed at the tool: missing/malformed guards.json and build
failures record guard.tool.failed; the arch rule set is spec 17.3 verbatim;
the lint set is closed at L1/L2 with zero-tolerance L1 (no suppression list);
complexity/mutation/fuzz/trace floors are committed and inert until their
subcommands land (registry exit 2). No C-ID claims are made.
