---
stage: qa
task: FJ-020
inputFingerprint: 8dacde273d16cf959da358666963b3afdbec066ac5dd4ebea125c094c4796740
outputFingerprint: 8dacde273d16cf959da358666963b3afdbec066ac5dd4ebea125c094c4796740
taskFingerprint: bf0432ef36830af414bf83f372280b27549851174ee9ee40d083af3e9fb25d42
gitHead: b198957ebbd5beebffa3cf118d89fb55f3c8efc0
generatedAt: 2026-09-29T15:30:32Z
author: worker/FJ-020-qa-post-413-current
---

# Independent QA — control API health, meta, and stub management

## Scope and method

Read `docs/agents/roles/qa.md`, `docs/agents/README.md`, the current specifier,
coder, cleaner and hardener artifacts, specification §§11.2, 19.3, 19.4,
41.1–41.6, 43.7 and 44.12, and the nine acceptance-catalog rows C-CTRL-001,
C-CTRL-002, C-CTRL-003, C-CTRL-004, C-CTRL-005, C-CTRL-010, C-CTRL-011,
C-MATCH-003 and C-GOLD-012. Read `internal/control/api.go`,
`internal/control/handlers.go`, `internal/control/handlers_test.go`,
`internal/engine/stub.go`, `internal/engine/sequence.go`,
`internal/engine/matcher.go`, `internal/engine/engine.go`,
`internal/config/load.go`, `internal/config/validate.go` and the control-path
branch of `internal/host/http/server.go`.

`internal/control` is untracked in this worktree, so it was read from disk. The
candidate and task fingerprints above come from
`./scripts/candidate-fingerprint candidate` and
`./scripts/candidate-fingerprint task .agent/work/FJ-020/state.json`; the full
revision hash from `git rev-parse HEAD`.

Independent probes live outside the repository at
`/private/tmp/fj020-qa-final/probe_test.go` and are injected with
`go test -overlay /private/tmp/fj020-qa-final/overlay.json`; the overlay maps
the absent repository path
`internal/control/zz_qa_final_probe_test.go` to that file. The probes use only
the exported `control`, `engine` and `config` surfaces and share no helper with
`internal/control/handlers_test.go`. 11 probe subtests execute; the run reports
exit 0 with log `/private/tmp/fj020-qa-final/qa-probe.log`. No repository file
was added or modified by the probes, and `internal/control` production sources
and permanent tests were not touched.

## Per-criterion observations

### C-CTRL-001 — health exact shape (§41.2)

`TestQAHealthExactShape` issues `GET /__fake/v1/health` through the real
`API.ServeHTTP` handler. Observed HTTP 200, `Content-Type: application/json`,
and the byte-exact body
`{"status":"ok","serverVersion":"0.0.0-dev","controlApiVersion":"v1"}`. The
journal length is unchanged (0) after the request. `TestQAMetaPreservesConfiguredProfileOrder`
also exercises a host-supplied `serverVersion` `9.9.9-qa.1` and observes it
propagated on both `/health` and `/meta`.

### C-CTRL-002 — meta exact shape, profile order, five limits (§41.3)

`TestQAMetaDefaultExactShape` observes the byte-exact default body
`{"serverVersion":"0.0.0-dev","controlApiVersion":"v1","configSchemaVersion":1,"mode":"strict","activeProfiles":["jev/v1"],"limits":{"dataPlaneBodyBytes":8388608,"controlPlaneBodyBytes":2097152,"maxInteractions":10000,"logBodyBytes":4096,"gracefulShutdownSeconds":5}}`.

`TestQAMetaConfiguredLimitsAreAllForwarded` loads a validated configuration
with `dataPlaneBodyBytes:1111`, `controlPlaneBodyBytes:2222`,
`maxInteractions:33`, `logBodyBytes:44`, `gracefulShutdownSeconds:6` through
`config.Load` and observes all five values echoed in the `limits` object of
`/meta` with no substitution.

`TestQAMetaPreservesConfiguredProfileOrder` constructs an API with
`NewAPIWithMetadata` and `ActiveProfiles:["second","first","third"]` and
observes that exact order in the `activeProfiles` array. Evidence limit:
`config.Load`/`normalizeProfile` accept only `jev`/`jev/v1`, so a
multi-element *configured* profile list is not constructible through the
configuration contract; the ordering observation therefore exercises the
constructor-supplied metadata path, and the only value reachable from
configuration is `["jev/v1"]`.

### C-CTRL-003 — create 201 and duplicate 409 (§41.4)

`TestQACreateDuplicateListClearLifecycle` posts
`{"id":"dynamic","profile":"jev/v1","priority":0,"when":{},"then":{"answers":{}}}`
and observes HTTP 201 with the byte-exact body
`{"id":"dynamic","registrationIndex":2}` after one static registration.
A second post of the same id observes HTTP 409 with the byte-exact envelope
`{"error":"fake_jev_duplicate_stub_id","message":"A stub with id 'dynamic' already exists.","stubId":"dynamic"}`.
A post reusing the static id `static` observes HTTP 409 with
`{"error":"fake_jev_duplicate_stub_id","message":"A stub with id 'static' already exists.","stubId":"static"}`.
Both duplicate responses carry `Content-Type: application/json`. Source
inspection of `registerStub` shows the index is captured by scanning
`Registry.Stubs()` for the just-registered id under the shared `stubMutations`
mutex.

### C-CTRL-004 — list exact item shape and registration order (§41.5)

After creating one dynamic stub and driving two data-plane selections of the
static stub, `TestQACreateDuplicateListClearLifecycle` observes the byte-exact
list body
`{"stubs":[{"id":"static","profile":"jev/v1","priority":0,"source":"static","registrationIndex":1,"invocations":2},{"id":"dynamic","profile":"jev/v1","priority":0,"source":"dynamic","registrationIndex":2,"invocations":0}]}`.
Each item carries exactly the six defined fields; `source` distinguishes
`static` from `dynamic`; order is registration index ascending.

### C-CTRL-005 — clear returns 204, dynamic-only, static counters intact (§41.6)

The same probe snapshots `Registry.Stubs()` before `DELETE /__fake/v1/stubs`,
observes HTTP 204 with a zero-length body, and then observes exactly one
remaining stub whose full struct (`reflect.DeepEqual`, including
`InvocationCount == 2` and `SequencePosition == 2`) equals the pre-delete static
record. The journal and verification-failure counts are unchanged. A subsequent
data-plane selection resumes the static sequence at its third element, showing
the sequence position was not reset.

### C-CTRL-010 — invalid control JSON and validation failures (§41.1)

`TestQAInvalidJSONExactEnvelope` posts `{`, an empty body, `{} {}` and
`{"id":"x",}` and observes HTTP 400 with the byte-exact envelope
`{"error":"fake_jev_bad_control_request","message":"Invalid control request."}`
for each, with no stub registered and the journal unchanged.
`TestQAValidationFailureExactTopLevelCode` posts a schema-valid JSON object
whose `profile` is not enabled and observes HTTP 400 with top-level
`"error":"fake_jev_bad_control_request"` and a non-empty human-readable
`message`, with no stub registered. A valid-JSON-but-not-an-object body (for
example `[1,2]`) reaches profile/config validation and likewise yields HTTP 400
with the same top-level code and a diagnostic message, consistent with the
§41.4 "one stub object" body contract.

### C-CTRL-011 — JSON content type on every control JSON response (§11)

Every probe assertion for a non-204 control response checks
`Content-Type: application/json`: health, default meta, configured-limit meta,
constructor-order meta, list, create 201, duplicate 409, invalid 400,
oversized 413, unknown-path 404 and wrong-method 405. `DELETE /__fake/v1/stubs`
is asserted as HTTP 204 with a zero-length body and no content-type assertion,
matching the §11 exception for `204 No Content`.

### Unsupported path and method (§41.1)

`TestQAUnsupportedPathAndMethod` observes `GET /__fake/v1/nope` and
`GET /__fake/v2/health` as HTTP 404 with the byte-exact body
`{"error":"fake_jev_control_not_found","message":"Unknown control endpoint."}`,
and `POST /__fake/v1/health` and `PUT /__fake/v1/stubs` as HTTP 405 with the
byte-exact body
`{"error":"fake_jev_control_method_not_allowed","message":"Method is not allowed."}`.
The engine journal remains empty after these requests.

### Control requests never enter the data-plane journal (§11, §41.1, §43.7)

`TestQAControlRequestsNeverEnterJournal` primes the journal with one real
data-plane selection, then issues health, meta, list, create, invalid create,
an oversized body, unknown path, wrong method and clear. The journal length
stays at 1 and `VerificationFailures()` does not change. `TestQAHealthExactShape`
and `TestQAUnsupportedPathAndMethod` additionally assert a zero journal on fresh
engines.

### C-MATCH-003 / C-GOLD-012 — dynamic stubs do not implicitly override static (§40.9, §44.12)

`TestQADynamicDoesNotDisplaceStaticTie` registers a static stub matching
`operation:systemone` at index 1, then creates a priority-0 dynamic stub that
also matches. A data-plane selection returns the static stub. Creating a
priority-1 dynamic stub for the same matcher, then selecting again, returns the
priority-1 dynamic stub. Both halves of §44.12 are observed through handler
creation and engine selection.

### Oversized control body (§19.4, §43.7)

`TestQAOversizedControlBodyExact413` builds an API whose control-plane limit is
1024 bytes, posts a valid stub preceded by 4096 spaces and a valid stub preceded
by limit+1 leading whitespace, and observes HTTP 413 with the byte-exact body
`{"error":"fake_jev_payload_too_large","message":"Request body exceeds the configured limit."}`
for both. A body at the limit still returns HTTP 201 with
`{"id":"x","registrationIndex":1}`. Recorded observation (not asserted as a
contract outcome): a 4096-byte body whose leading byte is already invalid JSON
(`x` repeated) returns HTTP 400 with the bad-control envelope, because the
decoder reports the syntax error before the byte limit is reached; the limit is
only observable as decoding advances. This is the decoding-time framing noted in
the hardener artifact.

## Commands and outcomes

All Go commands ran with `GOCACHE=/private/tmp/fj020-qa-final/gocache` from the
repository root. Logs are under `/private/tmp/fj020-qa-final/`.

- `go test ./internal/control -count=1` — exit 0; `ok fake-jev/internal/control 0.191s` (`check-1-control.log`).
- `go test ./... -count=1` — exit 0; `config`, `control`, `engine`, `host/http` and `test/integration` all report `ok`. The integration package bound loopback listeners successfully in this session (`check-2-all.log`).
- `go test -race ./... -count=1` — exit 0; all packages including `control`, `host/http` and `test/integration` report `ok` with no race reports (`check-3-race.log`).
- `go vet ./...` — exit 0; no diagnostics (`check-4-vet.log`).
- `go run ./cmd/guard arch` — exit 0; `"findings": []`, 47 files / 9 packages (`check-5-arch.log`).
- `go run ./cmd/guard lint` — exit 0; `"findings": []`, 47 files / 9 packages (`check-6-lint.log`).
- `go run ./cmd/guard trace` — exit 0; `"findings": []`, 0 active / 17 covered ids, 16 test files (`check-7-trace.log`).
- `go run ./cmd/guard fuzz` — exit 0; `"findings": []`, 0 targets (`check-8-fuzz.log`).
- `go test -overlay /private/tmp/fj020-qa-final/overlay.json ./internal/control -run TestQA -count=1 -v` — exit 0; 11 subtests executed (`qa-probe.log`).
- `./scripts/verify-candidate FJ-020` — first run (before this artifact was refreshed) exited 1. Report `.agent/reports/FJ-020/report-20260929T152721Z.json`; candidate `8dacde27…`, task `bf0432ef…`, revision `b198957`. The report's summary line reported 47 rows in the default success state, 1 row in the non-success state, 3 `skipped`, 0 `not_applicable`. The row not in the default success state was `stage.qa.chain` with machine code `stage.evidence_stale` (the then-current `qa.md` carried the earlier `26df167d…` chain input). The three `skipped` rows were the required `stage.tooling_bootstrap_exempt` entries for cleaner, hardener and qa (`check-9-verify-FJ-020.log`).
- `./scripts/verify-candidate FJ-020` — runs after this artifact was refreshed with `inputFingerprint 8dacde27…` so the hardener→qa chain link matches: exit 0. Reports `.agent/reports/FJ-020/report-20260929T152955Z.json`, `.agent/reports/FJ-020/report-20260929T153107Z.json`, `.agent/reports/FJ-020/report-20260929T153247Z.json` and `.agent/reports/FJ-020/report-20260929T153345Z.json`; candidate `8dacde27…`, task `bf0432ef…`, revision `b198957`. Every summary line reported every row in the default success state except 3 `skipped` and 0 `not_applicable`; the three `skipped` rows are the required-but-exempt `stage.tooling_bootstrap_exempt` entries for cleaner, hardener and qa (`check-10-verify-FJ-020.log` … `check-13-verify-FJ-020.log`). The reports were emitted by the verification script, not written by this stage.

## Independent probe source

Path: `/private/tmp/fj020-qa-final/probe_test.go` (injected via
`/private/tmp/fj020-qa-final/overlay.json`). Reproduced verbatim below.

```go
package control_test

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"fake-jev/internal/config"
	"fake-jev/internal/control"
	"fake-jev/internal/engine"
)

// Independent QA probes for FJ-020. This file lives outside the repository and
// is injected through `go test -overlay`; it uses only exported control, engine
// and config surfaces so it does not share the repository's test helpers.

func qaAPI(t *testing.T, cfgJSON string, static ...engine.Stub) (*control.API, *engine.Engine) {
	t.Helper()
	cfg, err := config.Load([]byte(cfgJSON))
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	eng, err := engine.NewEngine(static)
	if err != nil {
		t.Fatalf("engine.NewEngine: %v", err)
	}
	return control.NewAPI(eng, cfg), eng
}

func qaDo(t *testing.T, a *control.API, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	a.ServeHTTP(rec, req)
	return rec
}

func qaWantJSON(t *testing.T, rec *httptest.ResponseRecorder, status int, body string) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d, want %d (body %s)", rec.Code, status, rec.Body)
	}
	if rec.Body.String() != body {
		t.Fatalf("body = %s, want %s", rec.Body, body)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content-type = %q, want application/json", ct)
	}
}

const qaDefaultConfig = `{"schemaVersion":1,"stubs":[]}`

func TestQAHealthExactShape(t *testing.T) {
	api, eng := qaAPI(t, qaDefaultConfig)
	rec := qaDo(t, api, http.MethodGet, "/__fake/v1/health", "")
	qaWantJSON(t, rec, http.StatusOK, `{"status":"ok","serverVersion":"0.0.0-dev","controlApiVersion":"v1"}`)
	if n := len(eng.Interactions()); n != 0 {
		t.Fatalf("health entered journal: %d", n)
	}
}

func TestQAMetaDefaultExactShape(t *testing.T) {
	api, _ := qaAPI(t, qaDefaultConfig)
	rec := qaDo(t, api, http.MethodGet, "/__fake/v1/meta", "")
	qaWantJSON(t, rec, http.StatusOK,
		`{"serverVersion":"0.0.0-dev","controlApiVersion":"v1","configSchemaVersion":1,"mode":"strict",`+
			`"activeProfiles":["jev/v1"],`+
			`"limits":{"dataPlaneBodyBytes":8388608,"controlPlaneBodyBytes":2097152,"maxInteractions":10000,"logBodyBytes":4096,"gracefulShutdownSeconds":5}}`)
}

func TestQAMetaConfiguredLimitsAreAllForwarded(t *testing.T) {
	cfg := `{"schemaVersion":1,"mode":"strict","compatibility":["jev/v1"],` +
		`"limits":{"dataPlaneBodyBytes":1111,"controlPlaneBodyBytes":2222,"maxInteractions":33,"logBodyBytes":44,"gracefulShutdownSeconds":6},` +
		`"stubs":[]}`
	api, _ := qaAPI(t, cfg)
	rec := qaDo(t, api, http.MethodGet, "/__fake/v1/meta", "")
	qaWantJSON(t, rec, http.StatusOK,
		`{"serverVersion":"0.0.0-dev","controlApiVersion":"v1","configSchemaVersion":1,"mode":"strict",`+
			`"activeProfiles":["jev/v1"],`+
			`"limits":{"dataPlaneBodyBytes":1111,"controlPlaneBodyBytes":2222,"maxInteractions":33,"logBodyBytes":44,"gracefulShutdownSeconds":6}}`)
}

func TestQAMetaPreservesConfiguredProfileOrder(t *testing.T) {
	api := control.NewAPIWithMetadata(nil, control.Metadata{
		ServerVersion:       "9.9.9-qa.1",
		ConfigSchemaVersion: 1,
		Mode:                "strict",
		ActiveProfiles:      []string{"second", "first", "third"},
		Limits:              control.Limits{DataPlaneBodyBytes: 1, ControlPlaneBodyBytes: 2, MaxInteractions: 3, LogBodyBytes: 4, GracefulShutdown: 5},
	})
	rec := qaDo(t, api, http.MethodGet, "/__fake/v1/meta", "")
	qaWantJSON(t, rec, http.StatusOK,
		`{"serverVersion":"9.9.9-qa.1","controlApiVersion":"v1","configSchemaVersion":1,"mode":"strict",`+
			`"activeProfiles":["second","first","third"],`+
			`"limits":{"dataPlaneBodyBytes":1,"controlPlaneBodyBytes":2,"maxInteractions":3,"logBodyBytes":4,"gracefulShutdownSeconds":5}}`)
}

func qaStubJSON(id string, priority int) string {
	return `{"id":"` + id + `","profile":"jev/v1","priority":` + itoa(priority) + `,"when":{},"then":{"answers":{}}}`
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	if neg {
		return "-" + string(b)
	}
	return string(b)
}

func TestQACreateDuplicateListClearLifecycle(t *testing.T) {
	staticMatcher := engine.NewMatcher()
	static := engine.Stub{
		ID:       "static",
		Profile:  "jev/v1",
		Matcher:  staticMatcher,
		Sequence: []engine.ResponseAction{"first", "second", "third"},
	}
	api, eng := qaAPI(t, qaDefaultConfig, static)

	// C-CTRL-003: 201 with exactly id and registrationIndex.
	created := qaDo(t, api, http.MethodPost, "/__fake/v1/stubs", qaStubJSON("dynamic", 0))
	qaWantJSON(t, created, http.StatusCreated, `{"id":"dynamic","registrationIndex":2}`)

	// C-CTRL-003: duplicate dynamic and duplicate static IDs use the exact 409 envelope.
	dupDynamic := qaDo(t, api, http.MethodPost, "/__fake/v1/stubs", qaStubJSON("dynamic", 0))
	qaWantJSON(t, dupDynamic, http.StatusConflict,
		`{"error":"fake_jev_duplicate_stub_id","message":"A stub with id 'dynamic' already exists.","stubId":"dynamic"}`)
	dupStatic := qaDo(t, api, http.MethodPost, "/__fake/v1/stubs", qaStubJSON("static", 0))
	qaWantJSON(t, dupStatic, http.StatusConflict,
		`{"error":"fake_jev_duplicate_stub_id","message":"A stub with id 'static' already exists.","stubId":"static"}`)

	// Advance the static stub's invocation and sequence counters via the data plane.
	exchange := engine.Exchange{Profile: "jev/v1"}
	for i := 0; i < 2; i++ {
		if sel := eng.Select(exchange); sel == nil || sel.ID != "static" {
			t.Fatalf("select %d = %#v", i, sel)
		}
	}

	// C-CTRL-004: exact item shape, provenance and registration-index ascending order.
	listed := qaDo(t, api, http.MethodGet, "/__fake/v1/stubs", "")
	qaWantJSON(t, listed, http.StatusOK,
		`{"stubs":[`+
			`{"id":"static","profile":"jev/v1","priority":0,"source":"static","registrationIndex":1,"invocations":2},`+
			`{"id":"dynamic","profile":"jev/v1","priority":0,"source":"dynamic","registrationIndex":2,"invocations":0}]}`)

	// C-CTRL-005: 204 empty body clears dynamic stubs only and preserves static state.
	before := eng.Registry.Stubs()
	journalBefore := len(eng.Interactions())
	failuresBefore := len(eng.VerificationFailures())
	cleared := qaDo(t, api, http.MethodDelete, "/__fake/v1/stubs", "")
	if cleared.Code != http.StatusNoContent || cleared.Body.Len() != 0 {
		t.Fatalf("clear = %d %q", cleared.Code, cleared.Body)
	}
	after := eng.Registry.Stubs()
	if len(after) != 1 || after[0].ID != "static" {
		t.Fatalf("remaining stubs = %#v", after)
	}
	if !reflect.DeepEqual(before[0], after[0]) {
		t.Fatalf("static stub state changed:\n before %#v\n after  %#v", before[0], after[0])
	}
	if after[0].InvocationCount != 2 || after[0].SequencePosition != 2 {
		t.Fatalf("static counters changed: %#v", after[0])
	}
	if len(eng.Interactions()) != journalBefore || len(eng.VerificationFailures()) != failuresBefore {
		t.Fatalf("control mutations changed journal/verification state")
	}
	// The next selection resumes the static sequence where it left off.
	if sel := eng.Select(exchange); sel == nil || sel.ID != "static" || sel.Response != "third" {
		t.Fatalf("sequence after clear = %#v", sel)
	}
}

func TestQAInvalidJSONExactEnvelope(t *testing.T) {
	for _, body := range []string{`{`, ``, `{} {}`, `{"id":"x",}`} {
		api, eng := qaAPI(t, qaDefaultConfig)
		rec := qaDo(t, api, http.MethodPost, "/__fake/v1/stubs", body)
		qaWantJSON(t, rec, http.StatusBadRequest, `{"error":"fake_jev_bad_control_request","message":"Invalid control request."}`)
		if len(eng.Registry.Stubs()) != 0 || len(eng.Interactions()) != 0 {
			t.Fatalf("invalid JSON %q mutated engine state", body)
		}
	}
}

func TestQAValidationFailureExactTopLevelCode(t *testing.T) {
	api, eng := qaAPI(t, qaDefaultConfig)
	rec := qaDo(t, api, http.MethodPost, "/__fake/v1/stubs",
		`{"id":"x","profile":"other","when":{},"then":{"answers":{}}}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (%s)", rec.Code, rec.Body)
	}
	if !strings.Contains(rec.Body.String(), `"error":"fake_jev_bad_control_request"`) || !strings.Contains(rec.Body.String(), `"message":"`) {
		t.Fatalf("envelope = %s", rec.Body)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content-type = %q", ct)
	}
	if len(eng.Registry.Stubs()) != 0 {
		t.Fatal("invalid profile registered a stub")
	}
}

func TestQAOversizedControlBodyExact413(t *testing.T) {
	eng, err := engine.NewEngine(nil)
	if err != nil {
		t.Fatal(err)
	}
	api := control.NewAPIWithMetadata(eng, control.Metadata{ActiveProfiles: []string{"jev/v1"}, Limits: control.Limits{ControlPlaneBodyBytes: 1024}})
	stub := `{"id":"x","profile":"jev/v1","when":{},"then":{"answers":{}}}`
	for _, body := range []string{
		strings.Repeat(" ", 4096) + stub,
		strings.Repeat(" ", 1024) + " " + stub,
	} {
		rec := qaDo(t, api, http.MethodPost, "/__fake/v1/stubs", body)
		qaWantJSON(t, rec, http.StatusRequestEntityTooLarge, `{"error":"fake_jev_payload_too_large","message":"Request body exceeds the configured limit."}`)
	}
	// Observation: a body whose leading bytes are already invalid JSON is
	// rejected as malformed before the limit is reached. Recorded, not asserted.
	invalid := qaDo(t, api, http.MethodPost, "/__fake/v1/stubs", strings.Repeat("x", 4096))
	t.Logf("oversized invalid-leading-bytes status=%d body=%s", invalid.Code, invalid.Body)
	// A body at or below the configured limit still succeeds.
	rec := qaDo(t, api, http.MethodPost, "/__fake/v1/stubs", stub)
	qaWantJSON(t, rec, http.StatusCreated, `{"id":"x","registrationIndex":1}`)
}

func TestQAUnsupportedPathAndMethod(t *testing.T) {
	api, eng := qaAPI(t, qaDefaultConfig)
	cases := []struct {
		method, path string
		status       int
		body         string
	}{
		{http.MethodGet, "/__fake/v1/nope", http.StatusNotFound, `{"error":"fake_jev_control_not_found","message":"Unknown control endpoint."}`},
		{http.MethodGet, "/__fake/v2/health", http.StatusNotFound, `{"error":"fake_jev_control_not_found","message":"Unknown control endpoint."}`},
		{http.MethodPost, "/__fake/v1/health", http.StatusMethodNotAllowed, `{"error":"fake_jev_control_method_not_allowed","message":"Method is not allowed."}`},
		{http.MethodPut, "/__fake/v1/stubs", http.StatusMethodNotAllowed, `{"error":"fake_jev_control_method_not_allowed","message":"Method is not allowed."}`},
	}
	for _, tc := range cases {
		rec := qaDo(t, api, tc.method, tc.path, "{}")
		qaWantJSON(t, rec, tc.status, tc.body)
	}
	if len(eng.Interactions()) != 0 {
		t.Fatal("unsupported control request entered journal")
	}
}

func TestQAControlRequestsNeverEnterJournal(t *testing.T) {
	matcher := engine.NewMatcher()
	static := engine.Stub{ID: "static", Profile: "jev/v1", Matcher: matcher, Response: "ok"}
	api, eng := qaAPI(t, qaDefaultConfig, static)

	// Prime the journal with a real data-plane interaction.
	if sel := eng.Select(engine.Exchange{Profile: "jev/v1"}); sel == nil {
		t.Fatal("expected a data-plane selection")
	}
	journalBefore := len(eng.Interactions())
	failuresBefore := len(eng.VerificationFailures())
	if journalBefore != 1 {
		t.Fatalf("journal = %d, want 1", journalBefore)
	}

	requests := []struct{ method, path, body string }{
		{http.MethodGet, "/__fake/v1/health", ""},
		{http.MethodGet, "/__fake/v1/meta", ""},
		{http.MethodGet, "/__fake/v1/stubs", ""},
		{http.MethodPost, "/__fake/v1/stubs", qaStubJSON("dynamic", 0)},
		{http.MethodPost, "/__fake/v1/stubs", "{"},
		{http.MethodPost, "/__fake/v1/stubs", strings.Repeat(" ", 4096)},
		{http.MethodGet, "/__fake/v1/nope", ""},
		{http.MethodPost, "/__fake/v1/health", "{}"},
		{http.MethodDelete, "/__fake/v1/stubs", ""},
	}
	for _, req := range requests {
		qaDo(t, api, req.method, req.path, req.body)
	}
	if len(eng.Interactions()) != journalBefore {
		t.Fatalf("journal grew from %d to %d", journalBefore, len(eng.Interactions()))
	}
	if len(eng.VerificationFailures()) != failuresBefore {
		t.Fatalf("verification state changed: %d -> %d", failuresBefore, len(eng.VerificationFailures()))
	}
}

func TestQADynamicDoesNotDisplaceStaticTie(t *testing.T) {
	matcher := engine.NewMatcher()
	matcher.Operation = "systemone"
	static := engine.Stub{ID: "static", Profile: "jev/v1", Matcher: matcher, Response: "static"}
	api, eng := qaAPI(t, qaDefaultConfig, static)

	tie := `{"id":"dynamic","profile":"jev/v1","priority":0,"when":{"operation":"systemone"},"then":{"answers":{}}}`
	if rec := qaDo(t, api, http.MethodPost, "/__fake/v1/stubs", tie); rec.Code != http.StatusCreated {
		t.Fatalf("create tie = %d %s", rec.Code, rec.Body)
	}
	exchange := engine.Exchange{Profile: "jev/v1", Operation: "systemone"}
	if sel := eng.Select(exchange); sel == nil || sel.ID != "static" {
		t.Fatalf("equal-priority dynamic displaced static: %#v", sel)
	}

	higher := `{"id":"higher","profile":"jev/v1","priority":1,"when":{"operation":"systemone"},"then":{"answers":{}}}`
	if rec := qaDo(t, api, http.MethodPost, "/__fake/v1/stubs", higher); rec.Code != http.StatusCreated {
		t.Fatalf("create higher = %d %s", rec.Code, rec.Body)
	}
	if sel := eng.Select(exchange); sel == nil || sel.ID != "higher" {
		t.Fatalf("greater-priority dynamic did not win: %#v", sel)
	}
}
```

## Residual risks and evidence limits

- Host integration: `internal/host/http/server.go` classifies `/__fake/` paths
  and bounds their bodies but replies `404 fake_jev_control_not_found` for every
  bounded control request (`server.go:163–167`); it does not delegate to
  `control.API`. All observations above are at the `control.API` handler
  boundary through `net/http/httptest`; end-to-end availability through the
  `fake-jev` listener is not established by these probes. Wiring the host is
  outside FJ-020's `allowedFiles`.
- Configured profile order: `config.Load` accepts only `jev`/`jev/v1`, so the
  only configuration-reachable `activeProfiles` value is `["jev/v1"]`; the
  multi-element ordering observation relies on constructor-supplied metadata.
- Oversized-body framing: the control body limit is enforced by
  `http.MaxBytesReader` in `ServeHTTP` and classified in `createStub`; endpoints
  that never read a body (health, meta, clear) accept an oversized body, and an
  oversized body whose leading bytes are invalid JSON is classified as a 400
  malformed-control error before the limit is reached. Both follow
  decoding-time framing and were recorded, not asserted.
- Create-versus-clear atomicity: `registerStub` and `clearStubs` share the
  package-level `stubMutations` mutex, so control mutations against one engine
  are serialized. Direct engine mutation made by an embedding host (calling
  `engine.RegisterDynamic`/`RemoveDynamic`/`Reset` outside the control API) is
  not covered by that mutex; no probe exercised that path.
- The 409 classification in `createStub` derives from the engine's
  `duplicate stub id` error text rather than a typed error; a changed engine
  message would change the mapping. This is observed by source inspection only.
- Sandbox: the `test/integration` package bound loopback listeners successfully
  in this session; earlier stage artifacts recorded
  `bind 127.0.0.1:0: operation not permitted`, so listener availability is
  environment-dependent. Role-pack validation rows remain
  `stage.tooling_bootstrap_exempt`.
- `stage.cleaner.tooling`, `stage.hardener.tooling` and `stage.qa.tooling`
  remain required-but-exempt `skipped` rows; their exemption does not assert
  that a product check ran.
- `internal/control` is untracked in this worktree, so it is absent from a
  tracked `git diff`; this artifact's candidate fingerprint covers the workspace
  content by the `candidate-fingerprint` definition, not a staged diff.

## Traces

- C-CTRL-001 — §41.2 health exact shape and status.
- C-CTRL-002 — §41.3 metadata shape, configured limits, profile order.
- C-CTRL-003 — §41.4 create 201 and duplicate 409 envelope.
- C-CTRL-004 — §41.5 list item shape, provenance, registration order.
- C-CTRL-005 — §41.6 204 clear, dynamic-only, static counters unchanged.
- C-CTRL-010 — §41.1 invalid-JSON and validation-failure 400 contract.
- C-CTRL-011 — §11 JSON content type on non-204 control responses.
- C-MATCH-003 — §40.9, §44.12 equal-priority static-before-dynamic precedence.
- C-GOLD-012 — §44.12 dynamic stub does not implicitly override static.
