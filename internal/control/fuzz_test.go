package control

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"fake-jev/internal/config"
	"fake-jev/internal/engine"
)

// The tests in this file establish that malformed control-plane input is
// answered with a controlled error envelope instead of a panic, a 5xx, or a
// corrupted response, and that control-plane state survives concurrent reads,
// dynamic registration, history clearing, and resets. Traces C-QUAL-004,
// C-HOST-009, and C-QUAL-002.

const (
	controlFuzzMaxPath        = 1024
	controlFuzzMaxContentType = 256
	controlFuzzMaxBody        = 16 << 10
	controlFuzzMaxResponse    = 64 << 10
)

// controlStatuses is the complete set of statuses a control-plane request may
// produce. A status at or above 500 would mean a malformed request was reported
// as an internal fake-jev failure rather than as a controlled rejection.
var controlStatuses = map[int]bool{
	http.StatusOK:               true,
	http.StatusCreated:          true,
	http.StatusNoContent:        true,
	http.StatusBadRequest:       true,
	http.StatusNotFound:         true,
	http.StatusMethodNotAllowed: true,
	http.StatusConflict:         true,
}

func newFuzzedAPI(t *testing.T) *API {
	t.Helper()
	cfg, err := config.Load([]byte(`{"schemaVersion":1,"stubs":[]}`))
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	state, err := engine.NewEngineWithJournal(nil, 64)
	if err != nil {
		t.Fatalf("engine.NewEngineWithJournal: %v", err)
	}
	return NewAPI(state, cfg)
}

// controlRawRequest builds the request by hand instead of using
// httptest.NewRequest: the product surface under test is API.ServeHTTP, which
// reads only Method, URL.Path, headers, and Body, and a hand-built request keeps
// harness parsing panics out of the fuzzer's crash signal.
func controlRawRequest(method, path, contentType string, body []byte) *http.Request {
	request := &http.Request{
		Method: method,
		URL:    &url.URL{Path: path},
		Header: make(http.Header),
		Body:   io.NopCloser(bytes.NewReader(body)),
	}
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	return request
}

// readOnlyControlRoute reports whether the route reads control state without
// mutating it, which makes two identical requests comparable.
func readOnlyControlRoute(method, path string) bool {
	if method != http.MethodGet {
		return false
	}
	switch path {
	case "/__fake/v1/health", "/__fake/v1/meta", "/__fake/v1/stubs", "/__fake/v1/requests", "/__fake/v1/verify":
		return true
	default:
		return false
	}
}

// FuzzControlMalformedInput feeds arbitrary methods, paths, content types, and
// bodies to the control-plane handler. Every response must be a controlled
// envelope: a status from the documented set, JSON with the JSON content type
// whenever a body is present, a fake_jev_-prefixed error for every rejection,
// no interaction admitted into the engine journal, no mutation of the caller's
// body, and byte-identical bodies for repeated read-only requests.
func FuzzControlMalformedInput(f *testing.F) {
	createStub := `{"id":"dynamic","profile":"jev/v1","when":{},"then":{"answers":{}}}`
	seeds := []struct {
		name        string
		method      string
		path        string
		contentType string
		body        string
	}{
		{"health", http.MethodGet, "/__fake/v1/health", "", ""},
		{"meta", http.MethodGet, "/__fake/v1/meta", "", ""},
		{"stubs", http.MethodGet, "/__fake/v1/stubs", "", ""},
		{"requests", http.MethodGet, "/__fake/v1/requests", "", ""},
		{"verify", http.MethodGet, "/__fake/v1/verify", "", ""},
		{"reset", http.MethodPost, "/__fake/v1/reset", "", ""},
		{"empty-id", http.MethodPost, "/__fake/v1/stubs", "application/json", `{"id":""}`},
		{"truncated-json", http.MethodPost, "/__fake/v1/stubs", "application/json", `{`},
		{"json-null", http.MethodPost, "/__fake/v1/stubs", "application/json", `null`},
		{"two-values", http.MethodPost, "/__fake/v1/stubs", "application/json", createStub + createStub},
		// Regression seed for the shape that made this target fail: a JSON object
		// repeating one key. The body is valid JSON - one `{`, 1,600 `"a":1,`
		// entries, one final `"a":1}` - 9,607 bytes, so it reaches config.Load's
		// duplicate-key scan instead of being rejected as malformed JSON. Before
		// FJ-065 that scan delegated to yaml.v3, whose whole duplicate-key
		// diagnostic list was echoed back: this body was answered with 70,444,103
		// bytes against the controlFuzzMaxResponse cap of 64 KiB. The product now
		// answers it with a constant 106-byte duplicate-key error. Keeping it as
		// a seed means plain `go test` runs it, instead of the assertion being
		// reached only when a fuzzing window happens to construct the input.
		{"repeated-key-bomb", http.MethodPost, "/__fake/v1/stubs", "application/json",
			`{` + strings.Repeat(`"a":1,`, 1600) + `"a":1}`},
		{"unknown-path", http.MethodPost, "/__fake/v1/unknown", "application/json", "{}"},
		{"unknown-method", "BREW", "/__fake/v1/reset", "", ""},
		{"deep-brackets", http.MethodPost, "/__fake/v1/stubs", "application/json", strings.Repeat("[", 4<<10)},
		{"hostile-content-type", http.MethodGet, "/__fake/v1/health", "\r\nX-Injected: 1\r\n", ""},
		{"valid-create", http.MethodPost, "/__fake/v1/stubs", "application/json", createStub},
	}
	for _, seed := range seeds {
		f.Add(seed.method, seed.path, seed.contentType, []byte(seed.body))
	}

	f.Fuzz(func(t *testing.T, method, path, contentType string, body []byte) {
		if len(path) > controlFuzzMaxPath || len(contentType) > controlFuzzMaxContentType || len(body) > controlFuzzMaxBody {
			return
		}
		api := newFuzzedAPI(t)
		callerBody := append([]byte(nil), body...)

		recorder := httptest.NewRecorder()
		api.ServeHTTP(recorder, controlRawRequest(method, path, contentType, body))

		status := recorder.Code
		if !controlStatuses[status] {
			t.Fatalf("control request %s %q returned status %d: %s", method, path, status, recorder.Body.String())
		}
		payload := recorder.Body.Bytes()
		if status == http.StatusNoContent {
			if len(payload) != 0 {
				t.Fatalf("control request %s %q answered 204 with body %q", method, path, payload)
			}
		} else {
			if got := recorder.Header().Get("Content-Type"); got != "application/json" {
				t.Fatalf("control request %s %q answered %d with Content-Type %q", method, path, status, got)
			}
			if len(payload) > controlFuzzMaxResponse {
				t.Fatalf("control request %s %q answered %d bytes", method, path, len(payload))
			}
			var envelope map[string]any
			if err := json.Unmarshal(payload, &envelope); err != nil || envelope == nil {
				t.Fatalf("control request %s %q answered a non-object body %q", method, path, payload)
			}
			if status < 200 || status >= 300 {
				code, ok := envelope["error"].(string)
				if !ok || !strings.HasPrefix(code, "fake_jev_") {
					t.Fatalf("control request %s %q answered %d with error %q", method, path, status, code)
				}
			}
		}

		if interactions := api.engine.Interactions(); len(interactions) != 0 {
			t.Fatalf("control request %s %q admitted %d interactions", method, path, len(interactions))
		}
		if !bytes.Equal(callerBody, body) {
			t.Fatalf("control request %s %q mutated the caller body %q into %q", method, path, callerBody, body)
		}

		if readOnlyControlRoute(method, path) {
			repeated := httptest.NewRecorder()
			api.ServeHTTP(repeated, controlRawRequest(method, path, contentType, body))
			if repeated.Code != status || !bytes.Equal(repeated.Body.Bytes(), payload) {
				t.Fatalf("repeated control request %s %q answered %d %q then %d %q", method, path, status, payload, repeated.Code, repeated.Body.Bytes())
			}
		}
	})
}

// controlCall is one completed control-plane request recorded in its own slot.
type controlCall struct {
	method      string
	path        string
	status      int
	contentType string
	body        []byte
}

func doControl(api *API, method, path, body string) controlCall {
	recorder := httptest.NewRecorder()
	api.ServeHTTP(recorder, httptest.NewRequest(method, path, strings.NewReader(body)))
	return controlCall{
		method:      method,
		path:        path,
		status:      recorder.Code,
		contentType: recorder.Header().Get("Content-Type"),
		body:        recorder.Body.Bytes(),
	}
}

// TestControlStateRaceFree drives one control API from a fixed set of
// concurrent readers, dynamic stub registrations, clears, resets, and
// verification reads. Each goroutine owns its result slot, and every assertion
// holds for any interleaving: fixed statuses per route, well-formed success and
// error envelopes, every distinct valid stub admitted exactly once, and unique
// increasing registration indices in the listed stub set.
func TestControlStateRaceFree(t *testing.T) {
	api := newFuzzedAPI(t)

	const (
		healthReaders   = 16
		metaReaders     = 8
		stubReaders     = 8
		stubWriters     = 8
		stubClearers    = 8
		requestReaders  = 8
		requestClearers = 8
		verifiers       = 8
		resets          = 4
	)
	const total = healthReaders + metaReaders + stubReaders + stubWriters + stubClearers + requestReaders + requestClearers + verifiers + resets

	calls := make([]controlCall, total)
	start := make(chan struct{})
	var workers sync.WaitGroup
	workers.Add(total)

	slot := 0
	launch := func(method, path, body string) {
		index := slot
		slot++
		go func() {
			defer workers.Done()
			<-start
			calls[index] = doControl(api, method, path, body)
		}()
	}
	for reader := 0; reader < healthReaders; reader++ {
		launch(http.MethodGet, "/__fake/v1/health", "")
	}
	for reader := 0; reader < metaReaders; reader++ {
		launch(http.MethodGet, "/__fake/v1/meta", "")
	}
	for reader := 0; reader < stubReaders; reader++ {
		launch(http.MethodGet, "/__fake/v1/stubs", "")
	}
	for writer := 0; writer < stubWriters; writer++ {
		body := fmt.Sprintf(`{"id":"dynamic-%d","profile":"jev/v1","when":{},"then":{"answers":{}}}`, writer)
		launch(http.MethodPost, "/__fake/v1/stubs", body)
	}
	for clearer := 0; clearer < stubClearers; clearer++ {
		launch(http.MethodDelete, "/__fake/v1/stubs", "")
	}
	for reader := 0; reader < requestReaders; reader++ {
		launch(http.MethodGet, "/__fake/v1/requests", "")
	}
	for clearer := 0; clearer < requestClearers; clearer++ {
		launch(http.MethodDelete, "/__fake/v1/requests", "")
	}
	for verifier := 0; verifier < verifiers; verifier++ {
		launch(http.MethodGet, "/__fake/v1/verify", "")
	}
	for reset := 0; reset < resets; reset++ {
		launch(http.MethodPost, "/__fake/v1/reset", "")
	}

	close(start)
	workers.Wait()

	created := 0
	for index, call := range calls {
		switch {
		case call.method == http.MethodGet:
			if call.status != http.StatusOK {
				t.Fatalf("call %d: %s %s answered %d: %s", index, call.method, call.path, call.status, call.body)
			}
		case call.method == http.MethodPost && call.path == "/__fake/v1/stubs":
			switch call.status {
			case http.StatusCreated:
				created++
			case http.StatusBadRequest, http.StatusConflict:
				if call.contentType != "application/json" {
					t.Fatalf("call %d: rejected create answered Content-Type %q", index, call.contentType)
				}
			default:
				t.Fatalf("call %d: create answered %d: %s", index, call.status, call.body)
			}
		default:
			if call.status != http.StatusNoContent || len(call.body) != 0 {
				t.Fatalf("call %d: %s %s answered %d with body %q", index, call.method, call.path, call.status, call.body)
			}
		}
		if call.status >= 200 && call.status < 300 && call.status != http.StatusNoContent {
			var envelope map[string]any
			if err := json.Unmarshal(call.body, &envelope); err != nil || envelope == nil {
				t.Fatalf("call %d: success body %q is not a JSON object", index, call.body)
			}
		}
		if call.status < 200 || call.status >= 300 {
			var envelope map[string]any
			if err := json.Unmarshal(call.body, &envelope); err != nil {
				t.Fatalf("call %d: error body %q is not JSON", index, call.body)
			}
			code, ok := envelope["error"].(string)
			if !ok || !strings.HasPrefix(code, "fake_jev_") {
				t.Fatalf("call %d: error body %q does not carry a fake_jev_ code", index, call.body)
			}
		}
	}
	if created != stubWriters {
		t.Fatalf("admitted %d of %d distinct dynamic stubs", created, stubWriters)
	}

	listed := doControl(api, http.MethodGet, "/__fake/v1/stubs", "")
	if listed.status != http.StatusOK {
		t.Fatalf("final stub listing answered %d: %s", listed.status, listed.body)
	}
	var envelope struct {
		Stubs []struct {
			ID                string `json:"id"`
			RegistrationIndex uint64 `json:"registrationIndex"`
		} `json:"stubs"`
	}
	if err := json.Unmarshal(listed.body, &envelope); err != nil {
		t.Fatalf("final stub listing %q is not JSON: %v", listed.body, err)
	}
	var previousIndex uint64
	for _, item := range envelope.Stubs {
		if item.RegistrationIndex <= previousIndex {
			t.Fatalf("stub %q has registration index %d after %d, want strictly increasing", item.ID, item.RegistrationIndex, previousIndex)
		}
		previousIndex = item.RegistrationIndex
	}
	seen := make(map[string]bool)
	for _, stub := range api.engine.Registry.Stubs() {
		if seen[stub.ID] {
			t.Fatalf("duplicate stub id %q survived concurrent control traffic", stub.ID)
		}
		seen[stub.ID] = true
	}
}
