package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"fake-jev/internal/config"
)

// section122Limits is the §12.2 built-in limits table, restated here so a
// change to any constant fails the identity chain below instead of tracking it
// silently. The same five values are pinned through /__fake/v1/meta by the CLI
// suite; this file binds them to the enforcement paths.
func section122Limits() Limits {
	return Limits{
		DataPlaneBodyBytes:    8388608,
		ControlPlaneBodyBytes: 2097152,
		MaxInteractions:       10000,
		LogBodyBytes:          4096,
		GracefulShutdown:      5,
	}
}

func testDefaultConfig(t *testing.T) *config.Config {
	t.Helper()
	cfg, err := config.Load([]byte("schemaVersion: 1\n"))
	if err != nil {
		t.Fatalf("load minimal configuration: %v", err)
	}
	return cfg
}

// TestLimitsFromDefaultConfigMatchSection122 is the identity half of
// C-HOST-008: the minimal configuration resolves to the exact §12.2 numbers,
// and the host NewServerFromConfig builds carries those same resolved values to
// its enforcement paths.
func TestLimitsFromDefaultConfigMatchSection122(t *testing.T) {
	cfg := testDefaultConfig(t)
	if got := defaultLimits(); got != section122Limits() {
		t.Fatalf("defaultLimits() = %+v, want %+v", got, section122Limits())
	}
	if got := limitsFromConfig(cfg.Limits); got != section122Limits() {
		t.Fatalf("limitsFromConfig(defaults) = %+v, want %+v", got, section122Limits())
	}
	server, err := NewServerFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewServerFromConfig: %v", err)
	}
	if got := server.Limits(); got != section122Limits() {
		t.Fatalf("NewServerFromConfig limits = %+v, want %+v", got, section122Limits())
	}
}

// repeatingReader fills every supplied buffer, so a multi-megabyte limit is
// exercised without allocating a multi-megabyte fixture. io.LimitReader bounds
// how many bytes it will ever produce.
type repeatingReader struct{}

func (repeatingReader) Read(buffer []byte) (int, error) {
	for index := range buffer {
		buffer[index] = 'x'
	}
	return len(buffer), nil
}

// unreadableBody fails if it is read at all. It makes the "rejected before
// unbounded allocation" clause of §19.4 observable: an oversize declared
// Content-Length must be rejected without consuming a single body byte.
type unreadableBody struct{ read bool }

func (body *unreadableBody) Read([]byte) (int, error) {
	body.read = true
	return 0, errors.New("request body must not be read")
}

func (body *unreadableBody) Close() error { return nil }

// TestReadBodyEnforcesDefaultPlaneLimits is the enforcement half of C-HOST-008
// for dataPlaneBodyBytes and controlPlaneBodyBytes: with the §12.2 defaults, a
// body at the limit is admitted and a body one byte over is rejected, whether
// the length is declared or streamed, and a declared oversize is rejected
// before the body is read.
func TestReadBodyEnforcesDefaultPlaneLimits(t *testing.T) {
	for _, test := range []struct {
		name  string
		limit int
	}{
		{"data plane", DefaultDataPlaneBodyBytes},
		{"control plane", DefaultControlPlaneBodyBytes},
	} {
		t.Run(test.name, func(t *testing.T) {
			accepted := httptest.NewRequest(http.MethodPost, "/x", io.LimitReader(repeatingReader{}, int64(test.limit)))
			accepted.ContentLength = int64(test.limit)
			body, tooLarge, err := readBody(accepted, test.limit)
			if err != nil || tooLarge || len(body) != test.limit {
				t.Fatalf("limit-size body: len=%d tooLarge=%t err=%v", len(body), tooLarge, err)
			}

			declared := httptest.NewRequest(http.MethodPost, "/x", &unreadableBody{})
			declared.ContentLength = int64(test.limit) + 1
			if _, tooLarge, err := readBody(declared, test.limit); err != nil || !tooLarge {
				t.Fatalf("declared oversize: tooLarge=%t err=%v", tooLarge, err)
			}
			unread, ok := declared.Body.(*unreadableBody)
			if !ok {
				t.Fatalf("declared body type = %T, want *unreadableBody", declared.Body)
			}
			if unread.read {
				t.Fatal("declared oversize body was read before rejection")
			}

			streamed := httptest.NewRequest(http.MethodPost, "/x", io.LimitReader(repeatingReader{}, int64(test.limit)+1))
			streamed.ContentLength = -1
			if _, tooLarge, err := readBody(streamed, test.limit); err != nil || !tooLarge {
				t.Fatalf("streamed oversize: tooLarge=%t err=%v", tooLarge, err)
			}
		})
	}
}

// TestServerEnforcesSection122DefaultPlaneThresholds binds the §12.2 default
// plane limits to the server's admission decision: a declared Content-Length one
// byte over each default is rejected with 413 and the body is never read, so the
// resolved default (not the configured override) is the threshold.
func TestServerEnforcesSection122DefaultPlaneThresholds(t *testing.T) {
	for _, test := range []struct {
		name   string
		target string
		limit  int
	}{
		{"data plane", "/v1/systemone", DefaultDataPlaneBodyBytes},
		{"control plane", "/__fake/v1/stubs", DefaultControlPlaneBodyBytes},
	} {
		t.Run(test.name, func(t *testing.T) {
			server, err := NewServerFromConfig(testDefaultConfig(t))
			if err != nil {
				t.Fatalf("NewServerFromConfig: %v", err)
			}
			declared := &unreadableBody{}
			request := httptest.NewRequest(http.MethodPost, test.target, declared)
			request.ContentLength = int64(test.limit) + 1
			recorder := httptest.NewRecorder()
			server.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusRequestEntityTooLarge {
				t.Fatalf("status = %d, want 413 (%s)", recorder.Code, recorder.Body.String())
			}
			if declared.read {
				t.Fatal("oversize request body was read before rejection")
			}
		})
	}
}

// TestDefaultMaxInteractionsIsEnforced binds the §12.2 maxInteractions default
// to the capacity the engine journal actually uses: exactly 10,000 admitted
// data-plane requests are journalled, the next returns 507 fake_jev_journal_full,
// and the journal stays at 10,000 with no eviction (§19.4).
func TestDefaultMaxInteractionsIsEnforced(t *testing.T) {
	server, err := NewServerFromConfig(testDefaultConfig(t))
	if err != nil {
		t.Fatalf("NewServerFromConfig: %v", err)
	}
	for index := 0; index < DefaultMaxInteractions; index++ {
		recorder := httptest.NewRecorder()
		server.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/models", nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("request %d status = %d", index+1, recorder.Code)
		}
	}
	if got := len(server.Engine().Interactions()); got != DefaultMaxInteractions {
		t.Fatalf("journal length = %d, want %d", got, DefaultMaxInteractions)
	}
	// The journal reports itself at capacity once it holds maxInteractions
	// records, which is the state from which the next request must fail.
	if !server.Engine().JournalFull() {
		t.Fatal("journal not reported full at maxInteractions")
	}

	overflow := httptest.NewRecorder()
	server.ServeHTTP(overflow, httptest.NewRequest(http.MethodGet, "/v1/models", nil))
	if overflow.Code != http.StatusInsufficientStorage || !strings.Contains(overflow.Body.String(), "fake_jev_journal_full") {
		t.Fatalf("overflow response = %d %s", overflow.Code, overflow.Body.String())
	}
	if !server.Engine().JournalFull() {
		t.Fatal("journal not reported full after the overflow request")
	}
	if got := len(server.Engine().Interactions()); got != DefaultMaxInteractions {
		t.Fatalf("journal length after overflow = %d, want %d (no eviction)", got, DefaultMaxInteractions)
	}
}

// TestFixtureRawBodyIsDataNotTemplate is C-ARCH-005 behavioural evidence: a
// then.raw body whose text looks like a Go template, a shell command, or a
// JavaScript expression is returned verbatim and never evaluated (§12.8, §19.3).
func TestFixtureRawBodyIsDataNotTemplate(t *testing.T) {
	cfg, err := config.Load([]byte(`schemaVersion: 1
stubs:
  - id: raw
    profile: jev/v1
    when:
      operation: systemone
    then:
      raw:
        status: 200
        body:
          tpl: "{{ .Env.HOME }}"
          tmpl: "{{ 7*7 }}"
          js: "${process.env.SECRET}"
          shell: "$(id)"
          eval: "1+1"
`))
	if err != nil {
		t.Fatalf("fixture text is data and must load: %v", err)
	}
	server, err := NewServerFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewServerFromConfig: %v", err)
	}
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/systemone", strings.NewReader(validRequest)))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d (%s)", recorder.Code, recorder.Body.String())
	}
	var got, want map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("response body is not JSON: %v", err)
	}
	if err := json.Unmarshal([]byte(`{"tpl":"{{ .Env.HOME }}","tmpl":"{{ 7*7 }}","js":"${process.env.SECRET}","shell":"$(id)","eval":"1+1"}`), &want); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("raw body was altered: got %v, want %v", got, want)
	}
}

// TestFixtureStateMatchesLiterally is C-ARCH-005 behavioural evidence on the
// matching path: a state value that looks like a template is compared as an
// exact JSON string, so only the literal request state matches (§40.3).
func TestFixtureStateMatchesLiterally(t *testing.T) {
	cfg, err := config.Load([]byte(`schemaVersion: 1
stubs:
  - id: literal
    profile: jev/v1
    when:
      operation: systemone
      state: "{{ .Env.HOME }}"
    then:
      answers:
        q:
          noul: true
`))
	if err != nil {
		t.Fatalf("load configuration: %v", err)
	}
	server, err := NewServerFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewServerFromConfig: %v", err)
	}
	matched := httptest.NewRecorder()
	server.ServeHTTP(matched, httptest.NewRequest(http.MethodPost, "/v1/systemone",
		strings.NewReader(`{"state":"{{ .Env.HOME }}","model":"m","questions":{"q":{"type":"noul"}}}`)))
	if matched.Code != http.StatusOK {
		t.Fatalf("literal state status = %d (%s)", matched.Code, matched.Body.String())
	}
	rendered := httptest.NewRecorder()
	server.ServeHTTP(rendered, httptest.NewRequest(http.MethodPost, "/v1/systemone",
		strings.NewReader(`{"state":"/tmp/rendered-home","model":"m","questions":{"q":{"type":"noul"}}}`)))
	if rendered.Code != http.StatusNotImplemented {
		t.Fatalf("rendered state status = %d (%s), want 501", rendered.Code, rendered.Body.String())
	}
}

// TestFixtureExpressionWhereNumberRequiredIsRejected is C-ARCH-005 evidence
// that an expression is rejected as invalid rather than evaluated: the fixture
// loads as data, and when it is selected the host refuses it as an invalid stub
// response. The configured string never becomes the number 49.
func TestFixtureExpressionWhereNumberRequiredIsRejected(t *testing.T) {
	cfg, err := config.Load([]byte(`schemaVersion: 1
stubs:
  - id: expression
    profile: jev/v1
    when:
      operation: systemone
    then:
      answers:
        q:
          noul: "{{ 7*7 }}"
`))
	if err != nil {
		t.Fatalf("fixture text is data and must load: %v", err)
	}
	server, err := NewServerFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewServerFromConfig: %v", err)
	}
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/systemone", strings.NewReader(validRequest)))
	if recorder.Code != http.StatusInternalServerError ||
		!strings.Contains(recorder.Body.String(), "fake_jev_invalid_stub_response") {
		t.Fatalf("expression response = %d %s, want 500 fake_jev_invalid_stub_response", recorder.Code, recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), "49") {
		t.Fatalf("expression was evaluated: %s", recorder.Body.String())
	}
}
