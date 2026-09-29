package http

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"fake-jev/internal/config"
	"fake-jev/internal/engine"
)

func TestServerDataPlaneFailures(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		target      string
		body        string
		limits      Limits
		stubs       []engine.Stub
		wantStatus  int
		wantBody    string
		wantOutcome string
		wantJournal int
	}{
		{
			name: "unknown route", method: http.MethodGet, target: "/v1/unknown", wantStatus: http.StatusNotFound,
			wantBody:    `{"error":"fake_jev_unknown_route","message":"No active compatibility profile handles this route."}`,
			wantOutcome: "unknown_route", wantJournal: 1,
		},
		{
			name: "trailing slash", method: http.MethodGet, target: "/v1/models/", wantStatus: http.StatusNotFound,
			wantBody:    `{"error":"fake_jev_unknown_route","message":"No active compatibility profile handles this route."}`,
			wantOutcome: "unknown_route", wantJournal: 1,
		},
		{
			name: "wrong method", method: http.MethodPost, target: "/v1/models", wantStatus: http.StatusNotFound,
			wantBody:    `{"error":"fake_jev_unknown_route","message":"No active compatibility profile handles this route."}`,
			wantOutcome: "unknown_route", wantJournal: 1,
		},
		{
			name: "unmatched operation", method: http.MethodPost, target: "/v1/systemone?ignored=1", body: validRequest,
			wantStatus:  http.StatusNotImplemented,
			wantBody:    `{"error":"fake_jev_unmatched_request","message":"No configured stub matched this request.","profile":"jev/v1","operation":"systemone"}`,
			wantOutcome: "unmatched", wantJournal: 1,
		},
		{
			name: "journal full", method: http.MethodGet, target: "/v1/models", limits: Limits{MaxInteractions: 1},
			wantStatus:  http.StatusInsufficientStorage,
			wantBody:    `{"error":"fake_jev_journal_full","message":"Interaction journal limit reached. Reset or increase maxInteractions."}`,
			wantOutcome: "", wantJournal: 1,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			capacity := test.limits.MaxInteractions
			if capacity == 0 {
				capacity = DefaultMaxInteractions
			}
			state, err := engine.NewEngineWithJournal(test.stubs, capacity)
			if err != nil {
				t.Fatal(err)
			}
			server := NewServer(NewDefaultRouter(state), test.limits)
			if test.name == "journal full" {
				first := httptest.NewRecorder()
				server.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/v1/models", nil))
				if first.Code != http.StatusOK {
					t.Fatalf("first response status = %d", first.Code)
				}
			}
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(test.method, test.target, strings.NewReader(test.body))
			server.ServeHTTP(recorder, request)
			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, test.wantStatus)
			}
			if recorder.Body.String() != test.wantBody {
				t.Fatalf("body = %s, want %s", recorder.Body.String(), test.wantBody)
			}
			if test.wantJournal == 1 {
				journal := state.Interactions()
				if len(journal) != 1 {
					t.Fatalf("journal length = %d, want 1", len(journal))
				}
				if test.wantOutcome != "" && journal[0].Outcome != test.wantOutcome {
					t.Fatalf("outcome = %q, want %q", journal[0].Outcome, test.wantOutcome)
				}
			}
			if test.name == "journal full" {
				second := httptest.NewRecorder()
				server.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/v1/models", nil))
				if second.Code != http.StatusInsufficientStorage || !strings.Contains(second.Body.String(), "fake_jev_journal_full") {
					t.Fatalf("second response = %d %s", second.Code, second.Body.String())
				}
				if len(state.Interactions()) != 1 || len(state.VerificationFailures()) != 1 || state.VerificationFailures()[0].RequestSequence != nil {
					t.Fatalf("journal-full state = interactions %d failures %+v", len(state.Interactions()), state.VerificationFailures())
				}
			}
		})
	}
}

func TestServerPayloadLimitDoesNotInvokeStub(t *testing.T) {
	stub := engine.Stub{
		ID: "answer", Profile: "jev/v1", Matcher: engine.Matcher{Operation: "systemone"},
		Response: config.ResponseConfig{Answers: json.RawMessage(`{"q":{"noul":true}}`)},
	}
	state, err := engine.NewEngineWithJournal([]engine.Stub{stub}, 2)
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(NewDefaultRouter(state), Limits{DataPlaneBodyBytes: 4, MaxInteractions: 2})
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/systemone", strings.NewReader("12345")))
	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", recorder.Code)
	}
	if recorder.Body.String() != `{"error":"fake_jev_payload_too_large","message":"Request body exceeds the configured limit."}` {
		t.Fatalf("body = %s", recorder.Body.String())
	}
	journal := state.Interactions()
	if len(journal) != 1 || journal[0].Outcome != "payload_too_large" || journal[0].RawBody != nil {
		t.Fatalf("journal = %+v", journal)
	}
	if got := state.Registry.Stubs()[0].InvocationCount; got != 0 {
		t.Fatalf("invocations = %d, want 0", got)
	}
}

func TestServerControlRequestsAreExcludedFromJournal(t *testing.T) {
	state, err := engine.NewEngineWithJournal(nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(NewDefaultRouter(state), Limits{ControlPlaneBodyBytes: 4, MaxInteractions: 1})
	for _, body := range []string{"12345", "{}"} {
		recorder := httptest.NewRecorder()
		server.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/__fake/v1/unknown", strings.NewReader(body)))
		if body == "12345" && recorder.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("oversize control status = %d", recorder.Code)
		}
	}
	if len(state.Interactions()) != 0 || len(state.VerificationFailures()) != 0 {
		t.Fatalf("control request mutated engine: journal=%v failures=%v", state.Interactions(), state.VerificationFailures())
	}
}

func TestServerMatchedResponseAndModels(t *testing.T) {
	stub := engine.Stub{
		ID: "answer", Profile: "jev/v1", Matcher: engine.Matcher{Operation: "systemone"},
		Response: config.ResponseConfig{Answers: json.RawMessage(`{"q":{"noul":true}}`)},
	}
	state, err := engine.NewEngineWithJournal([]engine.Stub{stub}, 4)
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(NewDefaultRouter(state), Limits{MaxInteractions: 4})
	for _, test := range []struct {
		method, target, body string
		status               int
		contains             string
	}{
		{http.MethodGet, "/v1/models?x=1", "", http.StatusOK, "jev-latest"},
		{http.MethodPost, "/v1/systemone", validRequest, http.StatusOK, `"noul":1`},
	} {
		recorder := httptest.NewRecorder()
		server.ServeHTTP(recorder, httptest.NewRequest(test.method, test.target, strings.NewReader(test.body)))
		if recorder.Code != test.status || !strings.Contains(recorder.Body.String(), test.contains) {
			t.Fatalf("%s %s = %d %s", test.method, test.target, recorder.Code, recorder.Body.String())
		}
		if !strings.HasPrefix(recorder.Header().Get("Content-Type"), "application/json") {
			t.Fatalf("content type = %q", recorder.Header().Get("Content-Type"))
		}
	}
}

func TestServerClosesRequestBodiesAndHandlesChunkedLimits(t *testing.T) {
	state, err := engine.NewEngineWithJournal(nil, 2)
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(NewDefaultRouter(state), Limits{DataPlaneBodyBytes: 4, MaxInteractions: 2})
	for _, oversized := range []bool{false, true} {
		body := &trackingBody{Reader: strings.NewReader("12345")}
		request := httptest.NewRequest(http.MethodPost, "/v1/systemone", body)
		if oversized {
			request.ContentLength = -1
		} else {
			request.ContentLength = int64(len("12345"))
		}
		recorder := httptest.NewRecorder()
		server.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("oversized=%t status = %d, want 413", oversized, recorder.Code)
		}
		if !body.closed {
			t.Fatalf("oversized=%t request body was not closed", oversized)
		}
	}
	if got := len(state.Interactions()); got != 2 {
		t.Fatalf("journal length = %d, want 2", got)
	}
}

func TestServerNilURLAndListenerAreSafe(t *testing.T) {
	if server := NewServer(nil, Limits{}); server != nil {
		recorder := httptest.NewRecorder()
		server.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/models", nil))
		if recorder.Code != http.StatusInternalServerError || recorder.Body.String() != `{"error":"fake_jev_internal_error","message":"Internal fake-jev error."}` {
			t.Fatalf("nil router response = %d %s", recorder.Code, recorder.Body.String())
		}
	}
	state, err := engine.NewEngineWithJournal(nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(NewDefaultRouter(state), Limits{MaxInteractions: 1})
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, &http.Request{Method: http.MethodGet})
	if recorder.Code != http.StatusNotFound || recorder.Body.String() != `{"error":"fake_jev_unknown_route","message":"No active compatibility profile handles this route."}` {
		t.Fatalf("nil URL response = %d %s", recorder.Code, recorder.Body.String())
	}
	if err := server.Serve(nil); err == nil || err.Error() != "nil HTTP listener" {
		t.Fatalf("nil listener error = %v", err)
	}
}

type trackingBody struct {
	io.Reader
	closed bool
}

func (body *trackingBody) Close() error {
	body.closed = true
	return nil
}

func TestServerConcurrentRequestsAreRaceSafe(t *testing.T) {
	state, err := engine.NewEngineWithJournal(nil, 64)
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(NewDefaultRouter(state), Limits{MaxInteractions: 64})
	var group sync.WaitGroup
	for i := 0; i < 32; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			recorder := httptest.NewRecorder()
			server.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/unknown", nil))
			if recorder.Code != http.StatusNotFound {
				t.Errorf("status = %d", recorder.Code)
			}
		}()
	}
	group.Wait()
	if len(state.Interactions()) != 32 {
		t.Fatalf("journal length = %d, want 32", len(state.Interactions()))
	}
}

const validRequest = `{"state":{},"model":"m","questions":{"q":{"type":"noul"}}}`
