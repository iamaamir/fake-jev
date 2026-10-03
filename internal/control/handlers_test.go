package control

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"fake-jev/internal/config"
	"fake-jev/internal/engine"
	jevhttp "fake-jev/internal/host/http"
)

func testAPI(t *testing.T, static ...engine.Stub) *API {
	t.Helper()
	cfg, err := config.Load([]byte(`{"schemaVersion":1,"stubs":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	state, err := engine.NewEngine(static)
	if err != nil {
		t.Fatal(err)
	}
	return NewAPI(state, cfg)
}

func request(t *testing.T, api *API, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	recorder := httptest.NewRecorder()
	api.ServeHTTP(recorder, req)
	return recorder
}

func TestControlHealthMetaAndUnsupportedRequests(t *testing.T) {
	api := testAPI(t)
	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		status     int
		want       string
		content    string
		journalLen int
	}{
		{"health", http.MethodGet, "/__fake/v1/health", "", http.StatusOK, `{"status":"ok","serverVersion":"0.0.0-dev","controlApiVersion":"v1"}`, "application/json", 0},
		{"meta", http.MethodGet, "/__fake/v1/meta", "", http.StatusOK, `{"serverVersion":"0.0.0-dev","controlApiVersion":"v1","configSchemaVersion":1,"mode":"strict","activeProfiles":["jev/v1"],"limits":{"dataPlaneBodyBytes":8388608,"controlPlaneBodyBytes":2097152,"maxInteractions":10000,"logBodyBytes":4096,"gracefulShutdownSeconds":5}}`, "application/json", 0},
		{"unknown", http.MethodGet, "/__fake/v1/nope", "", http.StatusNotFound, `{"error":"fake_jev_control_not_found","message":"Unknown control endpoint."}`, "application/json", 0},
		{"wrong method", http.MethodPost, "/__fake/v1/health", "{}", http.StatusMethodNotAllowed, `{"error":"fake_jev_control_method_not_allowed","message":"Method is not allowed."}`, "application/json", 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := request(t, api, test.method, test.path, test.body)
			if got.Code != test.status {
				t.Fatalf("status = %d, want %d", got.Code, test.status)
			}
			if got.Header().Get("Content-Type") != test.content {
				t.Fatalf("content type = %q, want %q", got.Header().Get("Content-Type"), test.content)
			}
			var gotJSON, wantJSON any
			if err := json.Unmarshal(got.Body.Bytes(), &gotJSON); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(test.want), &wantJSON); err != nil {
				t.Fatal(err)
			}
			gotBytes, err := json.Marshal(gotJSON)
			if err != nil {
				t.Fatal(err)
			}
			wantBytes, err := json.Marshal(wantJSON)
			if err != nil {
				t.Fatal(err)
			}
			if string(gotBytes) != string(wantBytes) {
				t.Fatalf("body = %s, want %s", gotBytes, wantBytes)
			}
			if len(api.engine.Interactions()) != test.journalLen {
				t.Fatalf("control request entered journal")
			}
		})
	}
}

func TestControlDynamicStubsLifecycleAndErrors(t *testing.T) {
	static := engine.Stub{ID: "static", Profile: "jev/v1", Matcher: engine.NewMatcher()}
	api := testAPI(t, static)
	body := `{"id":"dynamic","profile":"jev/v1","when":{"questions":{"x":"noul"}},"then":{"answers":{"x":{"noul":1}}}}`

	created := request(t, api, http.MethodPost, "/__fake/v1/stubs", body)
	if created.Code != http.StatusCreated || created.Body.String() != `{"id":"dynamic","registrationIndex":2}` {
		t.Fatalf("create = %d %s", created.Code, created.Body)
	}
	duplicate := request(t, api, http.MethodPost, "/__fake/v1/stubs", body)
	if duplicate.Code != http.StatusConflict || !strings.Contains(duplicate.Body.String(), `"fake_jev_duplicate_stub_id"`) || !strings.Contains(duplicate.Body.String(), `"stubId":"dynamic"`) {
		t.Fatalf("duplicate = %d %s", duplicate.Code, duplicate.Body)
	}
	invalid := request(t, api, http.MethodPost, "/__fake/v1/stubs", `{`)
	if invalid.Code != http.StatusBadRequest || invalid.Body.String() != `{"error":"fake_jev_bad_control_request","message":"Invalid control request."}` {
		t.Fatalf("invalid = %d %s", invalid.Code, invalid.Body)
	}
	badProfile := request(t, api, http.MethodPost, "/__fake/v1/stubs", `{"id":"bad","profile":"other","when":{},"then":{"answers":{}}}`)
	if badProfile.Code != http.StatusBadRequest || !strings.Contains(badProfile.Body.String(), `"fake_jev_bad_control_request"`) {
		t.Fatalf("profile = %d %s", badProfile.Code, badProfile.Body)
	}

	listed := request(t, api, http.MethodGet, "/__fake/v1/stubs", "")
	var payload struct {
		Stubs []stubItem `json:"stubs"`
	}
	if err := json.Unmarshal(listed.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if listed.Code != http.StatusOK || len(payload.Stubs) != 2 || payload.Stubs[0].Source != engine.SourceStatic || payload.Stubs[1].Source != engine.SourceDynamic || payload.Stubs[1].Invocations != 0 {
		t.Fatalf("list = %d %#v", listed.Code, payload.Stubs)
	}

	cleared := request(t, api, http.MethodDelete, "/__fake/v1/stubs", "")
	if cleared.Code != http.StatusNoContent || cleared.Body.Len() != 0 {
		t.Fatalf("clear = %d %q", cleared.Code, cleared.Body.String())
	}
	remaining := api.engine.Registry.Stubs()
	if len(remaining) != 1 || remaining[0].ID != "static" || remaining[0].InvocationCount != 0 {
		t.Fatalf("static state changed: %#v", remaining)
	}
}

func TestControlDynamicPrecedencePreservesStaticTie(t *testing.T) {
	matcher := engine.NewMatcher()
	matcher.Operation = "systemone"
	static := engine.Stub{ID: "static", Profile: "jev/v1", Matcher: matcher, Response: "static"}
	api := testAPI(t, static)
	body := `{"id":"dynamic","profile":"jev/v1","when":{"operation":"systemone"},"then":{"answers":{}}}`
	if response := request(t, api, http.MethodPost, "/__fake/v1/stubs", body); response.Code != http.StatusCreated {
		t.Fatalf("create = %d %s", response.Code, response.Body)
	}
	exchange := engine.Exchange{Profile: "jev/v1", Operation: "systemone"}
	selection := api.engine.Select(exchange)
	if selection == nil || selection.ID != "static" {
		t.Fatalf("equal-priority dynamic displaced static: %#v", selection)
	}
}

// The original document must reach config validation without zero values or
// omitempty changing required fields, nulls, or optional response properties.
func TestControlStubSchemaPreservation(t *testing.T) {
	tests := []struct {
		name, body string
		valid      bool
	}{
		{"missing when", `{"id":"x","profile":"jev/v1","then":{"answers":{}}}`, false},
		{"null priority", `{"id":"x","profile":"jev/v1","priority":null,"when":{},"then":{"answers":{}}}`, false},
		{"null expect", `{"id":"x","profile":"jev/v1","expect":null,"when":{},"then":{"answers":{}}}`, false},
		{"null questions", `{"id":"x","profile":"jev/v1","when":{"questions":null},"then":{"answers":{}}}`, false},
		{"missing usage fields", `{"id":"x","profile":"jev/v1","when":{},"then":{"answers":{},"usage":{}}}`, false},
		{"raw without headers", `{"id":"x","profile":"jev/v1","when":{},"then":{"raw":{"status":200}}}`, true},
		{"unknown field", `{"id":"x","profile":"jev/v1","when":{},"then":{"answers":{}},"extra":1}`, false},
		{"duplicate id", `{"id":"a","id":"x","profile":"jev/v1","when":{},"then":{"answers":{}}}`, false},
		{"escaped duplicate id", `{"id":"a","\u0069d":"x","profile":"jev/v1","when":{},"then":{"answers":{}}}`, false},
		{"nested duplicate", `{"id":"x","profile":"jev/v1","when":{"state":{"key":1,"key":2}},"then":{"answers":{}}}`, false},
		{"sequence duplicate", `{"id":"x","profile":"jev/v1","when":{},"then":{"sequence":[{"raw":{"status":200,"status":201}}]}}`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := config.Load([]byte(`{"schemaVersion":1,"stubs":[` + tt.body + `]}`))
			if (err == nil) != tt.valid {
				t.Fatalf("config validity = %v, want %t", err, tt.valid)
			}
			api := testAPI(t)
			got := request(t, api, http.MethodPost, "/__fake/v1/stubs", tt.body)
			if tt.valid {
				if got.Code != http.StatusCreated || got.Body.String() != `{"id":"x","registrationIndex":1}` {
					t.Fatalf("create = %d %s", got.Code, got.Body)
				}
			} else {
				var payload map[string]string
				if err := json.Unmarshal(got.Body.Bytes(), &payload); err != nil {
					t.Fatal(err)
				}
				if got.Code != http.StatusBadRequest || len(payload) != 2 || payload["error"] != "fake_jev_bad_control_request" || payload["message"] == "" {
					t.Fatalf("error = %d %s", got.Code, got.Body)
				}
				if len(api.engine.Registry.Stubs()) != 0 {
					t.Fatal("invalid request registered a stub")
				}
			}
			if got.Header().Get("Content-Type") != "application/json" || len(api.engine.Interactions()) != 0 {
				t.Fatal("control response contract or journal isolation violated")
			}
		})
	}
}

func TestControlStrictJSON(t *testing.T) {
	for _, body := range []string{
		`{`, `{"id":"x",}`, `{id: "x"}`, `{} {}`, `{} trailing`, "", `/*comment*/{}`, `{"id":"` + string([]byte{0xff}) + `"}`,
	} {
		t.Run(body, func(t *testing.T) {
			api := testAPI(t)
			got := request(t, api, http.MethodPost, "/__fake/v1/stubs", body)
			if got.Code != http.StatusBadRequest || got.Body.String() != `{"error":"fake_jev_bad_control_request","message":"Invalid control request."}` {
				t.Fatalf("response = %d %s", got.Code, got.Body)
			}
			if len(api.engine.Registry.Stubs()) != 0 || len(api.engine.Interactions()) != 0 {
				t.Fatal("invalid JSON mutated engine")
			}
		})
	}
}

func TestControlVersionMetadata(t *testing.T) {
	for _, version := range []string{"", "1.2.3-rc.1+build.2"} {
		api := NewAPIWithMetadata(nil, Metadata{ServerVersion: version})
		want := version
		if want == "" {
			want = "0.0.0-dev"
		}
		for _, endpoint := range []string{"health", "meta"} {
			got := request(t, api, http.MethodGet, "/__fake/v1/"+endpoint, "")
			var payload struct {
				ServerVersion string `json:"serverVersion"`
			}
			if err := json.Unmarshal(got.Body.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			if got.Code != http.StatusOK || payload.ServerVersion != want {
				t.Fatalf("%s version = %q, want %q", endpoint, payload.ServerVersion, want)
			}
		}
	}
}

func TestControlBodyLimit(t *testing.T) {
	const limit = 1024
	stub := `{"id":"bounded","profile":"jev/v1","when":{},"then":{"answers":{}}}`
	padding := strings.Repeat(" ", limit-len(stub))
	cases := []struct {
		name   string
		body   string
		status int
	}{
		{"below limit", stub, http.StatusCreated},
		{"exact limit", stub + padding, http.StatusCreated},
		{"over limit trailing whitespace", stub + padding + " ", http.StatusRequestEntityTooLarge},
		{"over limit leading whitespace", padding + " " + stub, http.StatusRequestEntityTooLarge},
		{"large body", strings.Repeat(" ", limit*4) + stub, http.StatusRequestEntityTooLarge},
		{"malformed", `{`, http.StatusBadRequest},
		{"extra JSON", stub + `{}`, http.StatusBadRequest},
	}
	for _, tc := range cases {
		for _, unknownLength := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/unknown-length=%t", tc.name, unknownLength), func(t *testing.T) {
				api := testAPI(t)
				api.metadata.Limits.ControlPlaneBodyBytes = limit
				body := &countedControlBody{Reader: strings.NewReader(tc.body)}
				req := httptest.NewRequest(http.MethodPost, "/__fake/v1/stubs", body)
				req.ContentLength = int64(len(tc.body))
				if unknownLength {
					req.ContentLength = -1
				}
				got := httptest.NewRecorder()
				api.Handler().ServeHTTP(got, req)
				want := `{"id":"bounded","registrationIndex":1}`
				if tc.status == http.StatusBadRequest {
					want = `{"error":"fake_jev_bad_control_request","message":"Invalid control request."}`
				}
				if tc.status == http.StatusRequestEntityTooLarge {
					want = `{"error":"fake_jev_payload_too_large","message":"Request body exceeds the configured limit."}`
				}
				if tc.status != http.StatusCreated {
					if len(api.engine.Registry.Stubs()) != 0 {
						t.Fatal("rejected body registered a stub")
					}
				}
				if got.Code != tc.status || got.Body.String() != want || got.Header().Get("Content-Type") != "application/json" {
					t.Fatalf("response = %d %s (%v)", got.Code, got.Body, got.Header())
				}
				if body.read > limit+1 {
					t.Fatalf("read %d bytes, limit+1 = %d", body.read, limit+1)
				}
				if len(api.engine.Interactions()) != 0 || len(api.engine.VerificationFailures()) != 0 {
					t.Fatal("control body changed journal or verification state")
				}
			})
		}
	}
}

type countedControlBody struct {
	io.Reader
	read int
}

func (b *countedControlBody) Read(p []byte) (int, error) {
	n, err := b.Reader.Read(p)
	b.read += n
	return n, err
}

func (b *countedControlBody) Close() error { return nil }

func TestControlConcurrentCreateClear(t *testing.T) {
	static := engine.Stub{ID: "static", Profile: "jev/v1", Matcher: engine.NewMatcher()}
	api := testAPI(t, static)
	// A second API must coordinate with the first when sharing its engine.
	other := NewAPIWithMetadata(api.engine, api.metadata)
	body := `{"id":"dynamic","profile":"jev/v1","when":{},"then":{"answers":{}}}`
	for i := 0; i < 100; i++ {
		start := make(chan struct{})
		created := make(chan *httptest.ResponseRecorder, 1)
		cleared := make(chan *httptest.ResponseRecorder, 1)
		go func() {
			<-start
			created <- request(t, api, http.MethodPost, "/__fake/v1/stubs", body)
		}()
		go func() {
			<-start
			cleared <- request(t, other, http.MethodDelete, "/__fake/v1/stubs", "")
		}()
		close(start)
		post, del := <-created, <-cleared
		want := fmt.Sprintf(`{"id":"dynamic","registrationIndex":%d}`, i+2)
		if post.Code != http.StatusCreated || post.Body.String() != want {
			t.Fatalf("iteration %d create = %d %s, want %s", i, post.Code, post.Body, want)
		}
		if del.Code != http.StatusNoContent || del.Body.Len() != 0 {
			t.Fatalf("clear = %d %s", del.Code, del.Body)
		}
		// Either ordering is legal. Clear before reusing the ID next iteration.
		request(t, api, http.MethodDelete, "/__fake/v1/stubs", "")
		stubs := api.engine.Registry.Stubs()
		if len(stubs) != 1 || stubs[0].ID != "static" || stubs[0].RegistrationIndex != 1 {
			t.Fatalf("static stub changed: %#v", stubs)
		}
	}
	if len(api.engine.Interactions()) != 0 || len(api.engine.VerificationFailures()) != 0 {
		t.Fatal("control mutations changed journal or verification state")
	}
}

// controlHarness binds the real HTTP host and the control API to one engine so
// request history and verification observe genuine data-plane recording.
type controlHarness struct {
	server *jevhttp.Server
	api    *API
}

func newControlHarness(t *testing.T, document string) *controlHarness {
	t.Helper()
	cfg, err := config.Load([]byte(document))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	server, err := jevhttp.NewServerFromConfig(cfg)
	if err != nil {
		t.Fatalf("construct server: %v", err)
	}
	return &controlHarness{server: server, api: NewAPI(server.Engine(), cfg)}
}

func (h *controlHarness) data(method, target, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	h.server.ServeHTTP(recorder, httptest.NewRequest(method, target, strings.NewReader(body)))
	return recorder
}

func (h *controlHarness) control(method, path, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	h.api.ServeHTTP(recorder, httptest.NewRequest(method, path, strings.NewReader(body)))
	return recorder
}

func (h *controlHarness) requestHistory(t *testing.T) []map[string]any {
	t.Helper()
	got := h.control(http.MethodGet, "/__fake/v1/requests", "")
	if got.Code != http.StatusOK {
		t.Fatalf("requests status = %d (%s)", got.Code, got.Body)
	}
	if got.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("requests content type = %q", got.Header().Get("Content-Type"))
	}
	var payload struct {
		Requests []map[string]any `json:"requests"`
	}
	if err := json.Unmarshal(got.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode requests %s: %v", got.Body.Bytes(), err)
	}
	return payload.Requests
}

func (h *controlHarness) verification(t *testing.T) (bool, []map[string]any) {
	t.Helper()
	got := h.control(http.MethodGet, "/__fake/v1/verify", "")
	if got.Code != http.StatusOK {
		t.Fatalf("verify status = %d (%s)", got.Code, got.Body)
	}
	var payload struct {
		Passed   bool             `json:"passed"`
		Failures []map[string]any `json:"failures"`
	}
	if err := json.Unmarshal(got.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode verify %s: %v", got.Body.Bytes(), err)
	}
	return payload.Passed, payload.Failures
}

var requestRecordKeys = []string{
	"sequence", "timestamp", "method", "path", "profile", "operation",
	"outcome", "matchedStubId", "status", "error", "requestBody",
}

func assertRecordShape(t *testing.T, record map[string]any) {
	t.Helper()
	if len(record) != len(requestRecordKeys) {
		t.Fatalf("record key set = %v", record)
	}
	for _, key := range requestRecordKeys {
		if _, ok := record[key]; !ok {
			t.Fatalf("record missing %q: %v", key, record)
		}
	}
	timestamp, ok := record["timestamp"].(string)
	if !ok {
		t.Fatalf("timestamp = %v", record["timestamp"])
	}
	if _, err := time.Parse(time.RFC3339Nano, timestamp); err != nil {
		t.Fatalf("timestamp %q is not RFC3339: %v", timestamp, err)
	}
}

func assertRecordFields(t *testing.T, record map[string]any, want map[string]any) {
	t.Helper()
	assertRecordShape(t, record)
	for key, wantValue := range want {
		gotJSON, err := json.Marshal(record[key])
		if err != nil {
			t.Fatal(err)
		}
		wantJSON, err := json.Marshal(wantValue)
		if err != nil {
			t.Fatal(err)
		}
		if string(gotJSON) != string(wantJSON) {
			t.Fatalf("record[%q] = %s, want %s (record %v)", key, gotJSON, wantJSON, record)
		}
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal %v: %v", value, err)
	}
	return encoded
}

func assertSemanticJSON(t *testing.T, got []byte, want string) {
	t.Helper()
	var gotValue, wantValue any
	if err := json.Unmarshal(got, &gotValue); err != nil {
		t.Fatalf("decode got %s: %v", got, err)
	}
	if err := json.Unmarshal([]byte(want), &wantValue); err != nil {
		t.Fatalf("decode want %s: %v", want, err)
	}
	if string(mustJSON(t, gotValue)) != string(mustJSON(t, wantValue)) {
		t.Fatalf("body = %s, want semantic JSON %s", got, want)
	}
}

func assertFailure(t *testing.T, failure map[string]any, code string, sequence any, stubID any) {
	t.Helper()
	for _, key := range []string{"code", "message", "requestSequence", "stubId"} {
		if _, ok := failure[key]; !ok {
			t.Fatalf("failure missing %q: %v", key, failure)
		}
	}
	if len(failure) != 4 {
		t.Fatalf("failure key set = %v", failure)
	}
	if failure["code"] != code {
		t.Fatalf("failure code = %v, want %s", failure["code"], code)
	}
	if message, ok := failure["message"].(string); !ok || message == "" {
		t.Fatalf("failure message = %v, want non-empty string", failure["message"])
	}
	gotSequence := mustJSON(t, failure["requestSequence"])
	wantSequence := mustJSON(t, sequence)
	if string(gotSequence) != string(wantSequence) {
		t.Fatalf("failure requestSequence = %s, want %s", gotSequence, wantSequence)
	}
	gotStub := mustJSON(t, failure["stubId"])
	wantStub := mustJSON(t, stubID)
	if string(gotStub) != string(wantStub) {
		t.Fatalf("failure stubId = %s, want %s", gotStub, wantStub)
	}
}

var (
	systemOneBody  = `{"state":"x","model":"jev-latest","questions":{"q":{"type":"noul"}}}`
	systemOneValue = map[string]any{"state": "x", "model": "jev-latest", "questions": map[string]any{"q": map[string]any{"type": "noul"}}}
	noulStub       = `{"id":"q-stub","profile":"jev/v1","when":{"questions":{"q":"noul"}},"then":{"answers":{"q":{"noul":1}}}}`
)

func TestControlRequestHistoryRecordRules(t *testing.T) {
	tests := []struct {
		name       string
		config     string
		method     string
		target     string
		body       string
		wantStatus int
		want       map[string]any
	}{
		{
			name: "models matched with query string", config: `{"schemaVersion":1}`,
			method: http.MethodGet, target: "/v1/models?x=1", body: "", wantStatus: http.StatusOK,
			want: map[string]any{"sequence": 1, "method": "GET", "path": "/v1/models", "profile": "jev/v1", "operation": "models", "outcome": "matched", "matchedStubId": nil, "status": 200, "error": nil, "requestBody": nil},
		},
		{
			name: "unmatched", config: `{"schemaVersion":1,"stubs":[` + noulStub + `]}`,
			method: http.MethodPost, target: "/v1/systemone", body: `{"state":"x","model":"jev-latest","questions":{"other":{"type":"noul"}}}`,
			wantStatus: http.StatusNotImplemented,
			want:       map[string]any{"sequence": 1, "method": "POST", "path": "/v1/systemone", "profile": "jev/v1", "operation": "systemone", "outcome": "unmatched", "matchedStubId": nil, "status": 501, "error": "fake_jev_unmatched_request", "requestBody": map[string]any{"state": "x", "model": "jev-latest", "questions": map[string]any{"other": map[string]any{"type": "noul"}}}},
		},
		{
			name: "unknown route", config: `{"schemaVersion":1}`,
			method: http.MethodGet, target: "/v1/nope", body: "", wantStatus: http.StatusNotFound,
			want: map[string]any{"sequence": 1, "method": "GET", "path": "/v1/nope", "profile": "jev/v1", "operation": nil, "outcome": "unknown_route", "matchedStubId": nil, "status": 404, "error": "fake_jev_unknown_route", "requestBody": nil},
		},
		{
			name: "malformed body", config: `{"schemaVersion":1,"stubs":[` + noulStub + `]}`,
			method: http.MethodPost, target: "/v1/systemone", body: `{`, wantStatus: http.StatusUnprocessableEntity,
			want: map[string]any{"sequence": 1, "method": "POST", "path": "/v1/systemone", "profile": "jev/v1", "operation": "systemone", "outcome": "validation_error", "matchedStubId": nil, "status": 422, "error": "fake_jev_validation_error", "requestBody": "{"},
		},
		{
			// §41.7 keeps the parsed value for a valid JSON body even when profile validation rejects it.
			name: "valid body rejected by validation", config: `{"schemaVersion":1,"stubs":[` + noulStub + `]}`,
			method: http.MethodPost, target: "/v1/systemone", body: `{"model":"jev-latest","questions":{"q":{"type":"noul"}}}`, wantStatus: http.StatusUnprocessableEntity,
			want: map[string]any{"sequence": 1, "method": "POST", "path": "/v1/systemone", "profile": "jev/v1", "operation": "systemone", "outcome": "validation_error", "matchedStubId": nil, "status": 422, "error": "fake_jev_validation_error", "requestBody": map[string]any{"model": "jev-latest", "questions": map[string]any{"q": map[string]any{"type": "noul"}}}},
		},
		{
			name: "invalid utf-8 body decodes with replacement", config: `{"schemaVersion":1,"stubs":[` + noulStub + `]}`,
			method: http.MethodPost, target: "/v1/systemone", body: "\xff", wantStatus: http.StatusUnprocessableEntity,
			want: map[string]any{"sequence": 1, "method": "POST", "path": "/v1/systemone", "profile": "jev/v1", "operation": "systemone", "outcome": "validation_error", "matchedStubId": nil, "status": 422, "error": "fake_jev_validation_error", "requestBody": "\ufffd"},
		},
		{
			name: "payload too large retains no body", config: `{"schemaVersion":1,"limits":{"dataPlaneBodyBytes":1024},"stubs":[` + noulStub + `]}`,
			method: http.MethodPost, target: "/v1/systemone", body: strings.Repeat("x", 1025), wantStatus: http.StatusRequestEntityTooLarge,
			want: map[string]any{"sequence": 1, "method": "POST", "path": "/v1/systemone", "profile": "jev/v1", "operation": "systemone", "outcome": "payload_too_large", "matchedStubId": nil, "status": 413, "error": "fake_jev_payload_too_large", "requestBody": nil},
		},
		{
			name: "invalid stub response", config: `{"schemaVersion":1,"stubs":[{"id":"partial","profile":"jev/v1","when":{"questions":{"route":"choice","urgent":"noul"}},"then":{"answers":{"route":{"choice":"backend"}}}}]}`,
			method: http.MethodPost, target: "/v1/systemone", body: `{"state":"x","model":"jev-latest","questions":{"route":{"type":"choice","criteria":{"backend":"API"}},"urgent":{"type":"noul"}}}`, wantStatus: http.StatusInternalServerError,
			want: map[string]any{"sequence": 1, "method": "POST", "path": "/v1/systemone", "profile": "jev/v1", "operation": "systemone", "outcome": "invalid_stub_response", "matchedStubId": "partial", "status": 500, "error": "fake_jev_invalid_stub_response", "requestBody": map[string]any{"state": "x", "model": "jev-latest", "questions": map[string]any{"route": map[string]any{"type": "choice", "criteria": map[string]any{"backend": "API"}}, "urgent": map[string]any{"type": "noul"}}}},
		},
		{
			name: "configured raw error stays matched", config: `{"schemaVersion":1,"stubs":[{"id":"rate-limited","profile":"jev/v1","when":{"questions":{"q":"noul"}},"then":{"raw":{"status":429,"headers":{"content-type":"application/json"},"body":{"detail":"rate limited"}}}}]}`,
			method: http.MethodPost, target: "/v1/systemone", body: systemOneBody, wantStatus: http.StatusTooManyRequests,
			want: map[string]any{"sequence": 1, "method": "POST", "path": "/v1/systemone", "profile": "jev/v1", "operation": "systemone", "outcome": "matched", "matchedStubId": "rate-limited", "status": 429, "error": nil, "requestBody": systemOneValue},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			h := newControlHarness(t, test.config)
			got := h.data(test.method, test.target, test.body)
			if got.Code != test.wantStatus {
				t.Fatalf("data status = %d, want %d (%s)", got.Code, test.wantStatus, got.Body)
			}
			records := h.requestHistory(t)
			if len(records) != 1 {
				t.Fatalf("records = %v", records)
			}
			assertRecordFields(t, records[0], test.want)
			if interactions := h.server.Engine().Interactions(); len(interactions) != 1 {
				t.Fatalf("control requests entered the journal: %+v", interactions)
			}
		})
	}
}

func TestControlRequestHistorySequenceAscending(t *testing.T) {
	h := newControlHarness(t, `{"schemaVersion":1,"stubs":[{"id":"retry","profile":"jev/v1","when":{"questions":{"retry":"noul"}},"then":{"sequence":[{"answers":{"retry":{"noul":1.0}}},{"answers":{"retry":{"noul":0.0}}}]}}]}`)
	body := `{"state":"x","model":"jev-latest","questions":{"retry":{"type":"noul"}}}`
	for i := 0; i < 3; i++ {
		h.data(http.MethodPost, "/v1/systemone", body)
	}
	if got := h.data(http.MethodGet, "/v1/systemone", ""); got.Code != http.StatusNotFound {
		t.Fatalf("wrong-method status = %d", got.Code)
	}
	records := h.requestHistory(t)
	if len(records) != 4 {
		t.Fatalf("records = %v", records)
	}
	for index, record := range records {
		assertRecordShape(t, record)
		if got := jsonNumberOf(t, record["sequence"]); got != float64(index+1) {
			t.Fatalf("record %d sequence = %v", index, record["sequence"])
		}
	}
	assertRecordFields(t, records[2], map[string]any{"outcome": "sequence_exhausted", "matchedStubId": "retry", "status": 409, "error": "fake_jev_sequence_exhausted"})
	assertRecordFields(t, records[3], map[string]any{"outcome": "unknown_route", "operation": nil, "status": 404, "error": "fake_jev_unknown_route"})
}

func jsonNumberOf(t *testing.T, value any) float64 {
	t.Helper()
	number, ok := value.(float64)
	if !ok {
		t.Fatalf("expected JSON number, got %T (%v)", value, value)
	}
	return number
}

func TestControlRequestHistoryEmptyShape(t *testing.T) {
	api := testAPI(t)
	got := request(t, api, http.MethodGet, "/__fake/v1/requests", "")
	if got.Code != http.StatusOK || got.Body.String() != `{"requests":[]}` {
		t.Fatalf("requests = %d %s", got.Code, got.Body)
	}
	if got.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("content type = %q", got.Header().Get("Content-Type"))
	}
}

func TestControlRequestHistoryWithoutHostMetadata(t *testing.T) {
	engineState, err := engine.NewEngine(nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := engineState.Transition(engine.InteractionRequest{}, func(uint64) engine.InteractionDecision {
		return engine.InteractionDecision{Exchange: engine.NewExchange("jev/v1", "models"), Valid: true, Outcome: "matched", ResponseStatus: 200}
	}); err != nil {
		t.Fatal(err)
	}
	api := NewAPIWithMetadata(engineState, Metadata{})
	got := request(t, api, http.MethodGet, "/__fake/v1/requests", "")
	var payload struct {
		Requests []map[string]any `json:"requests"`
	}
	if err := json.Unmarshal(got.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Requests) != 1 {
		t.Fatalf("requests = %v", payload.Requests)
	}
	assertRecordFields(t, payload.Requests[0], map[string]any{"sequence": 1, "method": "", "path": "", "profile": "jev/v1", "operation": "models", "outcome": "matched", "matchedStubId": nil, "status": 200, "error": nil, "requestBody": nil})
}

func TestControlVerifyShapes(t *testing.T) {
	api := testAPI(t)
	got := request(t, api, http.MethodGet, "/__fake/v1/verify", "")
	if got.Code != http.StatusOK || got.Body.String() != `{"passed":true,"failures":[]}` {
		t.Fatalf("verify = %d %s", got.Code, got.Body)
	}
	if got.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("verify content type = %q", got.Header().Get("Content-Type"))
	}
	again := request(t, api, http.MethodGet, "/__fake/v1/verify", "")
	if again.Body.String() != got.Body.String() {
		t.Fatalf("verify is not stable: %s then %s", got.Body, again.Body)
	}
	if len(api.engine.Interactions()) != 0 || len(api.engine.VerificationFailures()) != 0 {
		t.Fatal("verify changed journal or verification state")
	}
}

func TestControlVerifyFailureOrdering(t *testing.T) {
	h := newControlHarness(t, `{"schemaVersion":1,"stubs":[
		{"id":"A","profile":"jev/v1","when":{"questions":{"q":"noul"}},"then":{"answers":{"q":{"noul":1}}},"expect":{"exactly":5}},
		{"id":"B","profile":"jev/v1","when":{"questions":{"other":"noul"}},"then":{"answers":{"other":{"noul":1}}},"expect":{"atLeast":5}}
	]}`)
	if got := h.data(http.MethodGet, "/v1/unknown", ""); got.Code != http.StatusNotFound {
		t.Fatalf("unknown route status = %d", got.Code)
	}
	if got := h.data(http.MethodPost, "/v1/systemone", `{"state":"x","model":"jev-latest","questions":{"zzz":{"type":"noul"}}}`); got.Code != http.StatusNotImplemented {
		t.Fatalf("unmatched status = %d", got.Code)
	}
	passed, failures := h.verification(t)
	if passed {
		t.Fatal("verification passed with failures present")
	}
	if len(failures) != 4 {
		t.Fatalf("failures = %v", failures)
	}
	assertFailure(t, failures[0], "unknown_route", 1, nil)
	assertFailure(t, failures[1], "unmatched_request", 2, nil)
	assertFailure(t, failures[2], "expect_exactly", nil, "A")
	assertFailure(t, failures[3], "expect_at_least", nil, "B")
	// Repeated calls between lifecycle boundaries return the same result.
	if _, second := h.verification(t); len(second) != 4 {
		t.Fatalf("second verify = %v", second)
	}
}

// §41.9 orders interaction-derived failures by request sequence ascending and
// then expectation failures by registration index, but does not place a failure
// whose requestSequence was never assigned. The documented order is sequenced
// interaction failures, then null-sequence sticky failures (journal_full), then
// expectation failures.
func TestControlVerifyNullSequenceFailureOrdering(t *testing.T) {
	stubs := []engine.Stub{
		{ID: "first", Profile: "jev/v1", Matcher: engine.NewMatcher(), Expect: engine.NewExactExpectation(1)},
		{ID: "second", Profile: "jev/v1", Matcher: engine.NewMatcher(), Expect: engine.NewExactExpectation(1)},
	}
	// A zero-capacity journal refuses every interaction while still accepting
	// post-transition failures, so sequenced failures can be injected in an
	// order the engine never produces on its own.
	engineState, err := engine.NewEngineWithJournal(stubs, 0)
	if err != nil {
		t.Fatal(err)
	}
	engineState.RecordFailure(7, "fake_jev_unknown_route", "Request used an unknown route.", "", http.StatusNotFound)
	engineState.RecordFailure(3, "fake_jev_validation_error", "Request failed validation.", "", http.StatusUnprocessableEntity)
	if _, _, err := engineState.Transition(engine.InteractionRequest{}, func(uint64) engine.InteractionDecision {
		return engine.InteractionDecision{Exchange: engine.NewExchange("jev/v1", "models"), Valid: true, Outcome: "matched", ResponseStatus: http.StatusOK}
	}); !errors.Is(err, engine.ErrJournalFull) {
		t.Fatalf("journal-full transition error = %v", err)
	}

	api := NewAPIWithMetadata(engineState, Metadata{})
	got := request(t, api, http.MethodGet, "/__fake/v1/verify", "")
	if got.Code != http.StatusOK {
		t.Fatalf("verify status = %d (%s)", got.Code, got.Body)
	}
	var payload struct {
		Passed   bool             `json:"passed"`
		Failures []map[string]any `json:"failures"`
	}
	if err := json.Unmarshal(got.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Passed {
		t.Fatalf("verification passed: %s", got.Body)
	}
	if len(payload.Failures) != 5 {
		t.Fatalf("failures = %s", got.Body)
	}
	assertFailure(t, payload.Failures[0], "validation_error", 3, nil)
	assertFailure(t, payload.Failures[1], "unknown_route", 7, nil)
	assertFailure(t, payload.Failures[2], "journal_full", nil, nil)
	assertFailure(t, payload.Failures[3], "expect_exactly", nil, "first")
	assertFailure(t, payload.Failures[4], "expect_exactly", nil, "second")

	codes := make([]string, 0, len(payload.Failures))
	for _, failure := range payload.Failures {
		code, _ := failure["code"].(string)
		codes = append(codes, code)
	}
	if joined := strings.Join(codes, ","); joined != "validation_error,unknown_route,journal_full,expect_exactly,expect_exactly" {
		t.Fatalf("failure codes = %s", joined)
	}

	// Every sequenced failure precedes the null-sequence failure, and the
	// sequenced failures are ascending.
	var sequenced []float64
	nullSequenceAt := -1
	for index, failure := range payload.Failures {
		if failure["requestSequence"] == nil {
			if nullSequenceAt < 0 {
				nullSequenceAt = index
			}
			continue
		}
		if nullSequenceAt >= 0 {
			t.Fatalf("sequenced failure %d follows null-sequence failure %d: %s", index, nullSequenceAt, got.Body)
		}
		sequenced = append(sequenced, jsonNumberOf(t, failure["requestSequence"]))
	}
	if len(sequenced) != 2 || sequenced[0] >= sequenced[1] {
		t.Fatalf("sequenced failures = %v, want ascending", sequenced)
	}
}

func TestControlVerifyExpectationCodes(t *testing.T) {
	tests := []struct {
		name       string
		expect     string
		calls      int
		wantCode   string
		wantPassed bool
	}{
		{"exactly unsatisfied", `"exactly":1`, 0, "expect_exactly", false},
		{"exactly satisfied", `"exactly":1`, 1, "", true},
		{"atLeast unsatisfied", `"atLeast":2`, 1, "expect_at_least", false},
		{"atMost exceeded", `"atMost":0`, 1, "expect_at_most", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			document := `{"schemaVersion":1,"stubs":[{"id":"e","profile":"jev/v1","when":{"questions":{"q":"noul"}},"then":{"answers":{"q":{"noul":1}}},"expect":{` + test.expect + `}}]}`
			h := newControlHarness(t, document)
			for i := 0; i < test.calls; i++ {
				if got := h.data(http.MethodPost, "/v1/systemone", systemOneBody); got.Code != http.StatusOK {
					t.Fatalf("call %d status = %d", i, got.Code)
				}
			}
			passed, failures := h.verification(t)
			if passed != test.wantPassed {
				t.Fatalf("passed = %v, failures %v", passed, failures)
			}
			if test.wantPassed {
				if len(failures) != 0 {
					t.Fatalf("failures = %v", failures)
				}
				return
			}
			if len(failures) != 1 {
				t.Fatalf("failures = %v", failures)
			}
			assertFailure(t, failures[0], test.wantCode, nil, "e")
		})
	}
}

func TestControlVerifyRawErrorExemption(t *testing.T) {
	h := newControlHarness(t, `{"schemaVersion":1,"stubs":[{"id":"rate-limited","profile":"jev/v1","when":{"questions":{"q":"noul"}},"then":{"raw":{"status":429,"headers":{"content-type":"application/json"},"body":{"detail":"rate limited"}}}}]}`)
	if got := h.data(http.MethodPost, "/v1/systemone", systemOneBody); got.Code != http.StatusTooManyRequests {
		t.Fatalf("raw status = %d", got.Code)
	}
	passed, failures := h.verification(t)
	if !passed || len(failures) != 0 {
		t.Fatalf("raw error failed verification: %v %v", passed, failures)
	}
}

func TestControlClearRequestsEffects(t *testing.T) {
	h := newControlHarness(t, `{"schemaVersion":1,"stubs":[
		{"id":"flow","profile":"jev/v1","when":{"questions":{"retry":"noul"}},"then":{"sequence":[{"answers":{"retry":{"noul":1.0}}},{"answers":{"retry":{"noul":0.0}}}]},"expect":{"exactly":1}},
		{"id":"wide","profile":"jev/v1","when":{"questions":{"other":"noul"}},"then":{"answers":{"other":{"noul":0.2}}},"expect":{"exactly":5}}
	]}`)
	retryBody := `{"state":"x","model":"jev-latest","questions":{"retry":{"type":"noul"}}}`
	if got := h.data(http.MethodPost, "/v1/systemone", retryBody); answerNoul(t, got.Body.Bytes(), "retry") != 1 {
		t.Fatalf("first retry = %s", got.Body)
	}
	h.data(http.MethodPost, "/v1/systemone", `{"state":"x","model":"jev-latest","questions":{"other":{"type":"noul"}}}`)
	h.data(http.MethodGet, "/v1/missing", "")

	passed, failures := h.verification(t)
	if passed || len(failures) != 2 {
		t.Fatalf("pre-clear verify = %v %v", passed, failures)
	}
	assertFailure(t, failures[0], "unknown_route", 3, nil)
	assertFailure(t, failures[1], "expect_exactly", nil, "wide")

	cleared := h.control(http.MethodDelete, "/__fake/v1/requests", "")
	if cleared.Code != http.StatusNoContent || cleared.Body.Len() != 0 {
		t.Fatalf("clear = %d %q", cleared.Code, cleared.Body)
	}
	if contentType := cleared.Header().Get("Content-Type"); contentType != "" {
		t.Fatalf("clear content type = %q", contentType)
	}
	if got := h.control(http.MethodGet, "/__fake/v1/requests", "").Body.String(); got != `{"requests":[]}` {
		t.Fatalf("history after clear = %s", got)
	}

	// Interaction-derived failures are gone; the unsatisfied expectation is preserved.
	passed, failures = h.verification(t)
	if passed || len(failures) != 1 {
		t.Fatalf("post-clear verify = %v %v", passed, failures)
	}
	assertFailure(t, failures[0], "expect_exactly", nil, "wide")

	// Stub counters and sequence positions are untouched.
	stubs := h.server.Engine().Registry.Stubs()
	if len(stubs) != 2 || stubs[0].InvocationCount != 1 || stubs[0].SequencePosition != 1 || stubs[1].InvocationCount != 1 {
		t.Fatalf("stub state after clear = %+v", stubs)
	}

	// The next admitted request receives sequence 1 and continues the stub's
	// sequence position rather than rewinding it.
	got := h.data(http.MethodPost, "/v1/systemone", retryBody)
	if value := answerNoul(t, got.Body.Bytes(), "retry"); value != 0 {
		t.Fatalf("post-clear retry = %v (%s)", value, got.Body)
	}
	records := h.requestHistory(t)
	if len(records) != 1 {
		t.Fatalf("records = %v", records)
	}
	assertRecordFields(t, records[0], map[string]any{"sequence": 1, "outcome": "matched", "matchedStubId": "flow"})
}

func TestControlClearRequestsEndsJournalFull(t *testing.T) {
	h := newControlHarness(t, `{"schemaVersion":1,"limits":{"maxInteractions":1},"stubs":[`+noulStub+`]}`)
	if got := h.data(http.MethodPost, "/v1/systemone", systemOneBody); got.Code != http.StatusOK {
		t.Fatalf("first status = %d", got.Code)
	}
	if got := h.data(http.MethodPost, "/v1/systemone", systemOneBody); got.Code != http.StatusInsufficientStorage {
		t.Fatalf("second status = %d", got.Code)
	}
	passed, failures := h.verification(t)
	if passed || len(failures) != 1 {
		t.Fatalf("verify = %v %v", passed, failures)
	}
	assertFailure(t, failures[0], "journal_full", nil, nil)
	if records := h.requestHistory(t); len(records) != 1 {
		t.Fatalf("records = %v", records)
	}

	if got := h.control(http.MethodDelete, "/__fake/v1/requests", ""); got.Code != http.StatusNoContent {
		t.Fatalf("clear = %d", got.Code)
	}
	passed, failures = h.verification(t)
	if !passed || len(failures) != 0 {
		t.Fatalf("verify after clear = %v %v", passed, failures)
	}
	if got := h.data(http.MethodPost, "/v1/systemone", systemOneBody); got.Code != http.StatusOK {
		t.Fatalf("post-clear status = %d", got.Code)
	}
	records := h.requestHistory(t)
	if len(records) != 1 {
		t.Fatalf("records = %v", records)
	}
	assertRecordFields(t, records[0], map[string]any{"sequence": 1})
}

func TestControlFullResetVector(t *testing.T) {
	h := newControlHarness(t, `{"schemaVersion":1,"stubs":[
		{"id":"static-a","profile":"jev/v1","when":{"questions":{"q":"noul"}},"then":{"answers":{"q":{"noul":0.1}}},"expect":{"exactly":2}},
		{"id":"static-b","profile":"jev/v1","when":{"questions":{"other":"noul"}},"then":{"answers":{"other":{"noul":0.2}}}}
	]}`)
	body := systemOneBody
	if got := h.data(http.MethodPost, "/v1/systemone", body); got.Code != http.StatusOK {
		t.Fatalf("step 1 status = %d", got.Code)
	}
	created := h.control(http.MethodPost, "/__fake/v1/stubs", `{"id":"temp","profile":"jev/v1","when":{"questions":{"q":"noul"}},"then":{"answers":{"q":{"noul":0.9}}}}`)
	if created.Code != http.StatusCreated || created.Body.String() != `{"id":"temp","registrationIndex":3}` {
		t.Fatalf("step 2 = %d %s", created.Code, created.Body)
	}

	reset := h.control(http.MethodPost, "/__fake/v1/reset", "")
	if reset.Code != http.StatusNoContent || reset.Body.Len() != 0 {
		t.Fatalf("reset = %d %q", reset.Code, reset.Body)
	}
	if contentType := reset.Header().Get("Content-Type"); contentType != "" {
		t.Fatalf("reset content type = %q", contentType)
	}

	// Effects 1 and 2: dynamic stub gone, static counters zero.
	stubs := h.control(http.MethodGet, "/__fake/v1/stubs", "")
	assertSemanticJSON(t, stubs.Body.Bytes(), `{"stubs":[
		{"id":"static-a","profile":"jev/v1","priority":0,"source":"static","registrationIndex":1,"invocations":0},
		{"id":"static-b","profile":"jev/v1","priority":0,"source":"static","registrationIndex":2,"invocations":0}
	]}`)

	// Effect 4: request history empty.
	if records := h.requestHistory(t); len(records) != 0 {
		t.Fatalf("history after reset = %v", records)
	}

	// Effect 5: the next admitted request is recorded with sequence 1.
	if got := h.data(http.MethodPost, "/v1/systemone", body); got.Code != http.StatusOK {
		t.Fatalf("step 6 status = %d", got.Code)
	}
	records := h.requestHistory(t)
	if len(records) != 1 {
		t.Fatalf("records = %v", records)
	}
	assertRecordFields(t, records[0], map[string]any{
		"sequence": 1, "method": "POST", "path": "/v1/systemone", "profile": "jev/v1",
		"operation": "systemone", "outcome": "matched", "matchedStubId": "static-a",
		"status": 200, "error": nil, "requestBody": systemOneValue,
	})

	// Effect 6: next dynamic registration index is N+1 = 3.
	again := h.control(http.MethodPost, "/__fake/v1/stubs", `{"id":"temp-again","profile":"jev/v1","when":{"questions":{"q":"noul"}},"then":{"answers":{"q":{"noul":0.9}}}}`)
	if again.Code != http.StatusCreated || again.Body.String() != `{"id":"temp-again","registrationIndex":3}` {
		t.Fatalf("step 7 = %d %s", again.Code, again.Body)
	}

	// §44.8: expectations are re-evaluated against the zeroed counters.
	passed, failures := h.verification(t)
	if passed || len(failures) != 1 {
		t.Fatalf("step 8 = %v %v", passed, failures)
	}
	assertFailure(t, failures[0], "expect_exactly", nil, "static-a")

	// Control-plane traffic never entered the data-plane journal.
	if interactions := h.server.Engine().Interactions(); len(interactions) != 1 {
		t.Fatalf("journal = %+v", interactions)
	}
}

func TestControlResetRewindsSequenceAndClearsFailures(t *testing.T) {
	h := newControlHarness(t, `{"schemaVersion":1,"stubs":[{"id":"flow","profile":"jev/v1","when":{"questions":{"retry":"noul"}},"then":{"sequence":[{"answers":{"retry":{"noul":1.0}}},{"answers":{"retry":{"noul":0.0}}}]}}]}`)
	retryBody := `{"state":"x","model":"jev-latest","questions":{"retry":{"type":"noul"}}}`
	if got := h.data(http.MethodPost, "/v1/systemone", retryBody); answerNoul(t, got.Body.Bytes(), "retry") != 1 {
		t.Fatalf("first retry = %s", got.Body)
	}
	h.data(http.MethodGet, "/v1/missing", "")
	passed, failures := h.verification(t)
	if passed || len(failures) != 1 {
		t.Fatalf("pre-reset verify = %v %v", passed, failures)
	}
	assertFailure(t, failures[0], "unknown_route", 2, nil)

	if got := h.control(http.MethodPost, "/__fake/v1/reset", ""); got.Code != http.StatusNoContent {
		t.Fatalf("reset = %d", got.Code)
	}
	passed, failures = h.verification(t)
	if !passed || len(failures) != 0 {
		t.Fatalf("post-reset verify = %v %v", passed, failures)
	}
	stubs := h.server.Engine().Registry.Stubs()
	if len(stubs) != 1 || stubs[0].InvocationCount != 0 || stubs[0].SequencePosition != 0 {
		t.Fatalf("reset stub state = %+v", stubs)
	}
	got := h.data(http.MethodPost, "/v1/systemone", retryBody)
	if value := answerNoul(t, got.Body.Bytes(), "retry"); value != 1 {
		t.Fatalf("post-reset retry = %v (%s)", value, got.Body)
	}
	records := h.requestHistory(t)
	if len(records) != 1 {
		t.Fatalf("records = %v", records)
	}
	assertRecordFields(t, records[0], map[string]any{"sequence": 1, "outcome": "matched"})

	// One static stub: the next dynamic registration index is N+1 = 2.
	created := h.control(http.MethodPost, "/__fake/v1/stubs", `{"id":"dyn","profile":"jev/v1","when":{},"then":{"answers":{}}}`)
	if created.Code != http.StatusCreated || created.Body.String() != `{"id":"dyn","registrationIndex":2}` {
		t.Fatalf("dynamic = %d %s", created.Code, created.Body)
	}
}

func TestControlNewEndpointMethods(t *testing.T) {
	tests := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/__fake/v1/requests"},
		{http.MethodPut, "/__fake/v1/requests"},
		{http.MethodPost, "/__fake/v1/verify"},
		{http.MethodDelete, "/__fake/v1/verify"},
		{http.MethodGet, "/__fake/v1/reset"},
		{http.MethodPut, "/__fake/v1/reset"},
	}
	for _, test := range tests {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			api := testAPI(t)
			got := request(t, api, test.method, test.path, "")
			if got.Code != http.StatusMethodNotAllowed || got.Body.String() != `{"error":"fake_jev_control_method_not_allowed","message":"Method is not allowed."}` {
				t.Fatalf("response = %d %s", got.Code, got.Body)
			}
			if len(api.engine.Interactions()) != 0 || len(api.engine.VerificationFailures()) != 0 {
				t.Fatal("method rejection mutated engine state")
			}
		})
	}
}

func TestControlResetRacesDataPlane(t *testing.T) {
	document := `{"schemaVersion":1,"stubs":[{"id":"a","profile":"jev/v1","when":{"questions":{"q":"noul"}},"then":{"answers":{"q":{"noul":1}}}}]}`
	for i := 0; i < 100; i++ {
		h := newControlHarness(t, document)
		start := make(chan struct{})
		dataDone := make(chan struct{})
		resetDone := make(chan struct{})
		go func() {
			<-start
			h.server.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/v1/systemone", strings.NewReader(systemOneBody)))
			close(dataDone)
		}()
		go func() {
			<-start
			h.api.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/__fake/v1/reset", nil))
			close(resetDone)
		}()
		close(start)
		<-dataDone
		<-resetDone

		records := h.server.Engine().Interactions()
		stubs := h.server.Engine().Registry.Stubs()
		if len(stubs) != 1 || stubs[0].ID != "a" {
			t.Fatalf("iteration %d stubs = %+v", i, stubs)
		}
		switch len(records) {
		case 0:
			if stubs[0].InvocationCount != 0 {
				t.Fatalf("iteration %d reset state kept invocation %d", i, stubs[0].InvocationCount)
			}
		case 1:
			if records[0].Sequence != 1 || !records[0].Matched || records[0].MatchedStubID != "a" || stubs[0].InvocationCount != 1 {
				t.Fatalf("iteration %d partial state: record %+v stub %+v", i, records[0], stubs[0])
			}
		default:
			t.Fatalf("iteration %d records = %+v", i, records)
		}
	}
}

func TestControlClearRacesDataPlane(t *testing.T) {
	for i := 0; i < 100; i++ {
		h := newControlHarness(t, `{"schemaVersion":1}`)
		start := make(chan struct{})
		dataDone := make(chan struct{})
		clearDone := make(chan struct{})
		go func() {
			<-start
			h.server.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/v1/models", nil))
			close(dataDone)
		}()
		go func() {
			<-start
			h.api.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodDelete, "/__fake/v1/requests", nil))
			close(clearDone)
		}()
		close(start)
		<-dataDone
		<-clearDone

		records := h.server.Engine().Interactions()
		if len(records) > 1 {
			t.Fatalf("iteration %d records = %+v", i, records)
		}
		if len(records) == 1 && records[0].Sequence != 1 {
			t.Fatalf("iteration %d sequence = %d", i, records[0].Sequence)
		}
	}
}

func answerNoul(t *testing.T, body []byte, name string) float64 {
	t.Helper()
	var payload struct {
		Answers map[string]map[string]any `json:"answers"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode answer %s: %v", body, err)
	}
	value, ok := payload.Answers[name]["noul"].(float64)
	if !ok {
		t.Fatalf("answer %q = %v (%s)", name, payload.Answers, body)
	}
	return value
}

// TestEngineExchangeMetadataSurvivesTransition pins the seam the request
// history endpoint depends on: Exchange.Metadata is deep-copied into the stored
// InteractionRecord and the stored copy is isolated from later caller mutation.
func TestEngineExchangeMetadataSurvivesTransition(t *testing.T) {
	engineState, err := engine.NewEngine(nil)
	if err != nil {
		t.Fatal(err)
	}
	exchange := engine.NewExchange("jev/v1", "models")
	exchange.Metadata["method"] = "GET"
	exchange.Metadata["target"] = "/v1/models"
	caller := exchange.Metadata
	if _, _, err := engineState.Transition(engine.InteractionRequest{}, func(uint64) engine.InteractionDecision {
		return engine.InteractionDecision{Exchange: exchange, Valid: true, Outcome: "matched", ResponseStatus: 200}
	}); err != nil {
		t.Fatal(err)
	}
	records := engineState.Interactions()
	if len(records) != 1 {
		t.Fatalf("records = %+v", records)
	}
	metadata := records[0].Request.Metadata
	if method, ok := metadata["method"].(string); !ok || method != "GET" {
		t.Fatalf("stored metadata method = %#v", metadata["method"])
	}
	if target, ok := metadata["target"].(string); !ok || target != "/v1/models" {
		t.Fatalf("stored metadata target = %#v", metadata["target"])
	}

	// Later mutation of the caller's map must not reach the stored record.
	caller["method"] = "MUTATED"
	caller["target"] = "/mutated"
	metadata = engineState.Interactions()[0].Request.Metadata
	if method, _ := metadata["method"].(string); method != "GET" {
		t.Fatalf("caller mutation leaked into stored record: %#v", metadata)
	}
	if target, _ := metadata["target"].(string); target != "/v1/models" {
		t.Fatalf("caller mutation leaked into stored record: %#v", metadata)
	}
}

// TestControlRepeatedJSONKeyResponseStaysBounded pins that a control request of
// at most 16 KiB that repeats one JSON key is answered with one bounded §41.1
// error naming the repeated key. The pre-FJ-065 behaviour was one diagnostic per
// repeated-key pair, so this response grew quadratically with the repetition
// count.
func TestControlRepeatedJSONKeyResponseStaysBounded(t *testing.T) {
	const wantBody = `{"error":"fake_jev_bad_control_request","message":"decode JSON configuration: duplicate object key \"id\""}`
	for _, count := range []int{2, 16, 1600} {
		keys := make([]string, count)
		for i := range keys {
			keys[i] = fmt.Sprintf(`"id":%d`, i)
		}
		body := "{" + strings.Join(keys, ",") + "}"
		if len(body) > 16*1024 {
			t.Fatalf("test request for %d repeated keys is %d bytes, above the 16 KiB acceptance bound", count, len(body))
		}
		api := testAPI(t)
		got := request(t, api, http.MethodPost, "/__fake/v1/stubs", body)
		if got.Code != http.StatusBadRequest || got.Body.String() != wantBody {
			t.Fatalf("repeated keys=%d: response = %d %s, want 400 %s", count, got.Code, got.Body, wantBody)
		}
		if got.Header().Get("Content-Type") != "application/json" {
			t.Fatalf("repeated keys=%d: content type = %q", count, got.Header().Get("Content-Type"))
		}
		if len(api.engine.Registry.Stubs()) != 0 || len(api.engine.Interactions()) != 0 {
			t.Fatal("rejected control request mutated engine")
		}
	}
}
