package control

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"fake-jev/internal/config"
	"fake-jev/internal/engine"
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
