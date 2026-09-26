---
stage: hardener
task: FJ-003
inputFingerprint: 5fe4ba9be1430d49a566a75e1ca5e78d9b638b45cba746e9acfd54bd68bed9fc
outputFingerprint: 5fe4ba9be1430d49a566a75e1ca5e78d9b638b45cba746e9acfd54bd68bed9fc
taskFingerprint: f91008767970d165f99603537a03289a1e7a20598d44d203cf1151bb5781b8c2
gitHead: 7dbf861
generatedAt: 2026-09-26T10:21:58Z
---

# Hardener — FJ-003

## Scope

The mutation/hardening analyzer is not implemented; this item is under the
recorded bootstrap exemption. Manual review was performed instead, against the
§19.3 rule that fixtures are data and the §12.8 statement that the raw escape
hatch MUST NOT turn fake-jev into a programmable mock server. No code changed.

## Findings

- **These files execute nothing.** They are JSON: no expression syntax, no
  template placeholders, no command field, no callback. A runner will read them
  and issue HTTP requests. The corpus cannot become the code-execution vector
  §19.3 forbids, because there is no interpreter that reads it as code. The
  enforcement of that property lives in FJ-046: the runner must treat every
  field as an opaque value except the ones it explicitly knows.

- **No secret material.** The snapshot declares bearer auth as a no-op; no vector
  sends an `Authorization` header, and none contains a token, a key, or a URL
  with credentials. The `Bearer fake` acceptance case belongs to C-JEV-017 and is
  FJ-016's to assert.

- **No network dependency.** Every request targets a relative path; the base URL
  is the runner's ephemeral loopback server. The corpus cannot cause egress, so
  C-QUAL-005 holds by construction rather than by convention.

- **Untrusted input flows one way only.** These are inputs *to* the system under
  test, and they include deliberately hostile ones: a raw `{` body, an unsupported
  question type, empty choice criteria, a body that exceeds the configured limit.
  That is the point. The corpus is where malformed input is a first-class
  expectation rather than an afterthought.

- **Bounded resource use.** The two limit vectors set
  `dataPlaneBodyBytes: 1024` and `maxInteractions: 1` rather than constructing a
  genuinely huge payload. The 413 vector would have been cheaper to write with a
  large body and much more expensive to execute.

- **The sentinel is a data value, not a control value.** The `<not asserted>`
  marker lives in a JSON string. A runner that mishandles it fails loudly by
  comparing a literal; it cannot be tricked into skipping a real assertion,
  because no expected value can legitimately be the string `<not asserted>`.

## What remains

The fuzz targets demonstrating that malformed input cannot panic the server
(FJ-040, C-QUAL-004) are the executable form of the same property. This corpus
pins specific hostile inputs; fuzzing explores the space around them.
