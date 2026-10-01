package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	nethttp "net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"fake-jev/internal/config"
	hosthttp "fake-jev/internal/host/http"
)

// answerServerConfig is a minimal server whose only stub answers any valid
// /v1/systemone request, so body-preview tests can drive an arbitrarily large
// state without depending on matcher state.
const answerServerConfig = `schemaVersion: 1
server:
  port: 0
stubs:
  - id: answer
    profile: jev/v1
    when:
      operation: systemone
    then:
      answers:
        q:
          noul: true
`

// truncatedAnswerServerConfig is answerServerConfig with an explicit, small
// logBodyBytes, so the preview bound can be shown to follow configuration
// rather than the §12.2 default.
const truncatedAnswerServerConfig = `schemaVersion: 1
server:
  port: 0
limits:
  logBodyBytes: 64
stubs:
  - id: answer
    profile: jev/v1
    when:
      operation: systemone
    then:
      answers:
        q:
          noul: true
`

// matchingServerConfig answers only a literally-matched state, so the same
// configuration yields both a matched (200) and an unmatched (501) request.
const matchingServerConfig = `schemaVersion: 1
server:
  port: 0
stubs:
  - id: answer
    profile: jev/v1
    when:
      operation: systemone
      state: "matched"
    then:
      answers:
        q:
          noul: true
`

// startLoggingServer runs the serve lifecycle in-process with the operational
// log stream captured. The handle is stopped when the test ends, so the
// listener never outlives the test.
func startLoggingServer(t *testing.T, source string) (*serverHandle, *syncBuffer) {
	t.Helper()
	cfg, err := config.Load([]byte(source))
	if err != nil {
		t.Fatalf("load configuration: %v", err)
	}
	logs := &syncBuffer{}
	handle, err := startServer(cfg, log.New(logs, "fake-jev: ", 0))
	if err != nil {
		t.Fatalf("start server: %v", err)
	}
	t.Cleanup(handle.stop)
	return handle, logs
}

// awaitLogLines polls until at least want prefixed log lines have been captured.
// The server logs after it has written a response, so the log stream is
// observed through a bounded poll rather than a fixed sleep (no sleeps-as-sync).
func awaitLogLines(t *testing.T, logs *syncBuffer, want int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for strings.Count(logs.String(), "fake-jev: ") < want {
		if time.Now().After(deadline) {
			t.Fatalf("observed %d log lines, want at least %d: %q",
				strings.Count(logs.String(), "fake-jev: "), want, logs.String())
		}
		time.Sleep(time.Millisecond)
	}
}

// sentinelState builds a ~20,000-byte state string with a distinct sentinel near
// the start, at offset 100, at offset 8000, and at the end, so a preview bound
// is observable by which sentinels reach the logs.
func sentinelState() string {
	return "HEAD" + strings.Repeat(".", 96) + // HEAD at offset 0, pad to 100
		"BODY-100" + strings.Repeat(".", 7892) + // BODY-100 at offset 100, pad to 8000
		"MID-8000" + strings.Repeat(".", 11992) + // MID-8000 at offset 8000, pad to 20000
		"TAIL-END" // at offset 20000
}

func requestWithState(state string) string {
	encoded, err := json.Marshal(state)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf(`{"state":%s,"model":"m","questions":{"q":{"type":"noul"}}}`, encoded)
}

// TestOperationalLogsTruncateBodyPreview pins C-CLI-010 and the H8d evidence of
// C-HOST-008: the body preview in the operational logs is the configured
// logBodyBytes (default 4 KiB), the preview carries an explicit truncation
// marker, content past the bound never reaches the stream, and the untruncated
// body is still retrievable through GET /__fake/v1/requests (§19.4, §20).
func TestOperationalLogsTruncateBodyPreview(t *testing.T) {
	const (
		headSentinel = "HEAD"
		bodySentinel = "BODY-100"
		midSentinel  = "MID-8000"
		tailSentinel = "TAIL-END"
		token        = "FAKEJEV-TOKEN-SENTINEL-0123456789abc"
	)
	state := sentinelState()
	body := requestWithState(state)

	for _, test := range []struct {
		name    string
		config  string
		kept    []string
		dropped []string
	}{
		{
			name:    "default 4 KiB preview",
			config:  answerServerConfig,
			kept:    []string{headSentinel, bodySentinel},
			dropped: []string{midSentinel, tailSentinel},
		},
		{
			name:    "configured 64 byte preview",
			config:  truncatedAnswerServerConfig,
			kept:    []string{headSentinel},
			dropped: []string{bodySentinel, midSentinel, tailSentinel},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			handle, logs := startLoggingServer(t, test.config)
			status, response := serveRequest(t, nethttp.MethodPost, handle.URL()+"/v1/systemone", strings.NewReader(body),
				map[string]string{"Authorization": "Bearer " + token})
			if status != nethttp.StatusOK {
				t.Fatalf("status = %d (%s)", status, response)
			}
			awaitLogLines(t, logs, 1)
			stream := logs.String()
			for _, want := range test.kept {
				if !strings.Contains(stream, want) {
					t.Fatalf("logs missing %q: %q", want, stream)
				}
			}
			for _, unwanted := range test.dropped {
				if strings.Contains(stream, unwanted) {
					t.Fatalf("logs contain %q from beyond the preview bound: %q", unwanted, stream)
				}
			}
			if !strings.Contains(stream, logTruncationMarker) {
				t.Fatalf("logs missing the truncation marker %q: %q", logTruncationMarker, stream)
			}
			if strings.Contains(stream, token) {
				t.Fatalf("Authorization value leaked into logs: %q", stream)
			}

			records := listRequests(t, handle.URL())
			if len(records) != 1 {
				t.Fatalf("request history = %v, want one record", records)
			}
			requestBody, ok := records[0]["requestBody"].(map[string]any)
			if !ok {
				t.Fatalf("requestBody = %v", records[0]["requestBody"])
			}
			if got := requestBody["state"]; got != state {
				t.Fatalf("control-plane state length = %d, want %d", len(fmt.Sprint(got)), len(state))
			}
		})
	}
}

// TestAuthorizationValuesAreAcceptedIdenticallyAndNeverLogged pins C-JEV-017 at
// the host level: a missing header, Bearer fake, and an arbitrary bearer token
// produce byte-identical responses, and the token value appears in neither the
// logs nor the request history (§10.1, §39.5, §42.5).
func TestAuthorizationValuesAreAcceptedIdenticallyAndNeverLogged(t *testing.T) {
	const token = "FAKEJEV-TOKEN-SENTINEL-0123456789abc"
	handle, logs := startLoggingServer(t, matchingServerConfig)

	variants := []string{"", "Bearer fake", "Bearer " + token}
	matchedBody := `{"state":"matched","model":"m","questions":{"q":{"type":"noul"}}}`
	unmatchedBody := `{"state":"other","model":"m","questions":{"q":{"type":"noul"}}}`

	type result struct {
		status int
		body   string
	}
	exchange := func(auth string) []result {
		headers := map[string]string{}
		if auth != "" {
			headers["Authorization"] = auth
		}
		responses := make([]result, 0, 3)
		for _, request := range []struct{ method, path, body string }{
			{nethttp.MethodPost, "/v1/systemone", matchedBody},
			{nethttp.MethodGet, "/v1/models", ""},
			{nethttp.MethodPost, "/v1/systemone", unmatchedBody},
		} {
			status, body := serveRequest(t, request.method, handle.URL()+request.path, strings.NewReader(request.body), headers)
			responses = append(responses, result{status: status, body: string(body)})
		}
		return responses
	}

	baseline := exchange(variants[0])
	if baseline[0].status != nethttp.StatusOK || baseline[1].status != nethttp.StatusOK || baseline[2].status != nethttp.StatusNotImplemented {
		t.Fatalf("baseline responses = %+v", baseline)
	}
	for index := 1; index < len(variants); index++ {
		if got := exchange(variants[index]); !reflect.DeepEqual(got, baseline) {
			t.Fatalf("auth %q responses = %+v, want %+v", variants[index], got, baseline)
		}
	}

	awaitLogLines(t, logs, 9)
	if stream := logs.String(); strings.Contains(stream, token) {
		t.Fatalf("Authorization value leaked into logs: %q", stream)
	}

	records := listRequests(t, handle.URL())
	if len(records) != 9 {
		t.Fatalf("journal length = %d, want 9 data-plane records", len(records))
	}
	encoded, err := json.Marshal(records)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), token) {
		t.Fatalf("Authorization value leaked into the request history: %s", encoded)
	}
}

// TestDefaultBindHostIsLoopback is the config-layer evidence of C-ARCH-006: the
// §12.2/§19.1 default host is the literal 127.0.0.1, so an unconfigured serve
// binds loopback. The process-level bind and the not-bound-publicly observation
// already exist in TestServeDefaultBindIsLoopback.
func TestDefaultBindHostIsLoopback(t *testing.T) {
	if config.DefaultHost != "127.0.0.1" {
		t.Fatalf("config.DefaultHost = %q, want 127.0.0.1", config.DefaultHost)
	}
	cfg, err := config.Load([]byte("schemaVersion: 1\n"))
	if err != nil {
		t.Fatalf("load minimal configuration: %v", err)
	}
	if cfg.Server.Host != "127.0.0.1" || cfg.Server.Port != 8787 {
		t.Fatalf("resolved default server = %s:%d, want 127.0.0.1:8787", cfg.Server.Host, cfg.Server.Port)
	}

	handle, _ := startLoggingServer(t, "schemaVersion: 1\nserver:\n  port: 0\n")
	host, _, err := net.SplitHostPort(strings.TrimPrefix(handle.URL(), "http://"))
	if err != nil {
		t.Fatalf("split bound url %q: %v", handle.URL(), err)
	}
	if host != config.DefaultHost {
		t.Fatalf("enforced default bind host = %q, want %q", host, config.DefaultHost)
	}
}

// countingBody records every Read so the "preview never reads the body ahead of
// the host" guarantee (§19.4) is observable, and so an oversize request can be
// shown to be rejected without consuming a body byte.
type countingBody struct {
	payload string
	offset  int
	reads   int
}

func (body *countingBody) Read(chunk []byte) (int, error) {
	body.reads++
	if body.offset >= len(body.payload) {
		return 0, io.EOF
	}
	read := copy(chunk, body.payload[body.offset:])
	body.offset += read
	return read, nil
}

func (body *countingBody) Close() error { return nil }

// TestCaptureBodyPreviewDoesNotReadAhead pins the wire-behavior fix: installing
// the preview observer must not read the request body itself, so the host's
// early §19.4/§43.7 rejection still sees an unread body. The host must still
// receive every byte through the observer.
func TestCaptureBodyPreviewDoesNotReadAhead(t *testing.T) {
	body := &countingBody{payload: "hello body"}
	request := httptest.NewRequest(nethttp.MethodPost, "/x", body)
	preview := captureBodyPreview(request, 4)
	if body.reads != 0 {
		t.Fatalf("captureBodyPreview read the body %d times before the host", body.reads)
	}
	served, err := io.ReadAll(request.Body)
	if err != nil {
		t.Fatalf("read body through the observer: %v", err)
	}
	if string(served) != body.payload {
		t.Fatalf("host body = %q, want %q", served, body.payload)
	}
	got, truncated := preview()
	if got != `"hell"` || !truncated {
		t.Fatalf("preview = (%q, %t), want (%q, true)", got, truncated, `"hell"`)
	}
}

// TestBodyPreviewMarkerBoundary pins that the truncation marker is applied
// exactly when the observed body exceeds the limit: not at the limit, at
// limit+1, and never for an empty body. Zero and negative limits are exercised
// because they must be safe rather than panic.
func TestBodyPreviewMarkerBoundary(t *testing.T) {
	for _, test := range []struct {
		name          string
		limit         int
		body          string
		wantPreview   string
		wantTruncated bool
	}{
		{"shorter than limit", 4, "abc", `"abc"`, false},
		{"exactly at limit", 4, "abcd", `"abcd"`, false},
		{"one over limit", 4, "abcde", `"abcd"`, true},
		{"empty body", 4, "", "", false},
		{"zero limit with body", 0, "abc", "", true},
		{"negative limit with body", -1, "abcd", "", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(nethttp.MethodPost, "/x", strings.NewReader(test.body))
			preview := captureBodyPreview(request, test.limit)
			served, err := io.ReadAll(request.Body)
			if err != nil {
				t.Fatalf("read restored body: %v", err)
			}
			if string(served) != test.body {
				t.Fatalf("host body = %q, want %q", served, test.body)
			}
			got, truncated := preview()
			if got != test.wantPreview || truncated != test.wantTruncated {
				t.Fatalf("preview = (%q, %t), want (%q, %t)", got, truncated, test.wantPreview, test.wantTruncated)
			}
		})
	}
}

// TestBodyPreviewEscapesLogForgeryBytes pins that preview bytes can never forge
// additional log lines or emit control characters: newlines, CR, NUL, and TAB
// are rendered as Go escape sequences (§20).
func TestBodyPreviewEscapesLogForgeryBytes(t *testing.T) {
	forged := "ok\nfake-jev: forged line\r\x00\tend"
	request := httptest.NewRequest(nethttp.MethodPost, "/x", strings.NewReader(forged))
	preview := captureBodyPreview(request, len(forged))
	if _, err := io.ReadAll(request.Body); err != nil {
		t.Fatalf("read restored body: %v", err)
	}
	rendered, truncated := preview()
	if truncated {
		t.Fatalf("preview unexpectedly truncated: %q", rendered)
	}
	for _, forbidden := range []string{"\n", "\r", "\x00", "\t"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("preview %q contains the raw control byte %q", rendered, forbidden)
		}
	}
	if !utf8.ValidString(rendered) {
		t.Fatalf("preview is not valid UTF-8: %q", rendered)
	}
}

// TestBodyPreviewCutSplittingRuneStaysValid pins the documented byte-cut
// semantics: the cut is by bytes, and a cut that splits a UTF-8 sequence is
// rendered as \xNN so the emitted line is always valid (§20).
func TestBodyPreviewCutSplittingRuneStaysValid(t *testing.T) {
	body := "ab\u20acx" // '€' is three bytes, so a 3-byte cut lands inside it
	request := httptest.NewRequest(nethttp.MethodPost, "/x", strings.NewReader(body))
	preview := captureBodyPreview(request, 3)
	if _, err := io.ReadAll(request.Body); err != nil {
		t.Fatalf("read restored body: %v", err)
	}
	rendered, truncated := preview()
	if !truncated {
		t.Fatalf("want truncated, got %q", rendered)
	}
	if !utf8.ValidString(rendered) {
		t.Fatalf("preview is not valid UTF-8: %q", rendered)
	}
	if rendered != `"ab\xe2"` {
		t.Fatalf("preview = %q, want %q", rendered, `"ab\xe2"`)
	}
}

// TestOversizedDataPlaneRequestIsRejectedWithoutReadingBody pins the wire
// behavior the serve logging wrapper must preserve (the regression the cleaner
// found): a declared oversize data request still receives the exact §43.7 413
// body, the host never reads a body byte first (§19.4), the journal records
// sequence 1 with outcome payload_too_large and a null body, and no stub state
// changes (§43.7, §44.15).
func TestOversizedDataPlaneRequestIsRejectedWithoutReadingBody(t *testing.T) {
	cfg, err := config.Load([]byte(answerServerConfig))
	if err != nil {
		t.Fatalf("load configuration: %v", err)
	}
	host, err := hosthttp.NewServerFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewServerFromConfig: %v", err)
	}
	handler := logHandler{next: host, logger: newOperationalLogger(log.New(io.Discard, "", 0), host.Limits().LogBodyBytes)}

	body := &countingBody{payload: "declared oversize body that must not be read"}
	request := httptest.NewRequest(nethttp.MethodPost, "/v1/systemone", body)
	request.ContentLength = int64(host.Limits().DataPlaneBodyBytes) + 1
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != nethttp.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413 (%s)", recorder.Code, recorder.Body.String())
	}
	var failure struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &failure); err != nil {
		t.Fatalf("decode 413 body: %v", err)
	}
	if failure.Error != "fake_jev_payload_too_large" || failure.Message != "Request body exceeds the configured limit." {
		t.Fatalf("413 body = %s, want the §43.7 payload-too-large body", recorder.Body.String())
	}
	if body.reads != 0 {
		t.Fatalf("oversize body was read %d times before the 413 rejection", body.reads)
	}
	records := host.Engine().Interactions()
	if len(records) != 1 {
		t.Fatalf("journal = %d records, want 1", len(records))
	}
	if records[0].Sequence != 1 || records[0].Outcome != "payload_too_large" || records[0].RawBody != nil {
		t.Fatalf("journal record = %+v, want sequence 1, outcome payload_too_large, null body", records[0])
	}
	for _, stub := range host.Engine().Registry.Stubs() {
		if stub.InvocationCount != 0 {
			t.Fatalf("stub %q was invoked %d times by an oversize request", stub.ID, stub.InvocationCount)
		}
	}
}
