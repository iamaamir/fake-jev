package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	nethttp "net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// recordedRequest is one request a canned control server received.
type recordedRequest struct {
	Method string
	Path   string
	Body   string
}

// cannedVerificationServer answers every request with one fixed status and body
// while recording what arrived, so the verify command's wire behavior and its
// exit codes are observable without a running fake-jev server. It binds an
// ephemeral loopback port (H3/H4).
type cannedVerificationServer struct {
	*httptest.Server
	status   int
	body     string
	mu       sync.Mutex
	requests []recordedRequest
}

func newCannedVerificationServer(t *testing.T, status int, body string) *cannedVerificationServer {
	t.Helper()
	canned := &cannedVerificationServer{status: status, body: body}
	canned.Server = httptest.NewServer(nethttp.HandlerFunc(func(writer nethttp.ResponseWriter, request *nethttp.Request) {
		data, err := io.ReadAll(request.Body)
		if err != nil {
			t.Errorf("read canned request body: %v", err)
		}
		canned.mu.Lock()
		canned.requests = append(canned.requests, recordedRequest{Method: request.Method, Path: request.URL.Path, Body: string(data)})
		canned.mu.Unlock()
		writer.WriteHeader(canned.status)
		if canned.body != "" {
			if _, err := io.WriteString(writer, canned.body); err != nil {
				t.Errorf("write canned response: %v", err)
			}
		}
	}))
	t.Cleanup(canned.Close)
	return canned
}

func (canned *cannedVerificationServer) recorded() []recordedRequest {
	canned.mu.Lock()
	defer canned.mu.Unlock()
	return append([]recordedRequest(nil), canned.requests...)
}

// TestVerifyUsageFailures pins C-CLI-001 and the §15.3 usage gate: a malformed
// command line exits 2, writes the diagnostic and usage on stderr, claims no
// result on stdout, and never reaches the network.
func TestVerifyUsageFailures(t *testing.T) {
	server := newCannedVerificationServer(t, nethttp.StatusOK, `{"passed":true,"failures":[]}`)
	cases := []struct {
		name string
		args []string
	}{
		{"missing url", []string{"verify"}},
		{"empty url", []string{"verify", "--url", ""}},
		{"unknown flag", []string{"verify", "--url", server.URL, "--nope"}},
		{"stray positional", []string{"verify", "--url", server.URL, "extra"}},
		{"unparseable url", []string{"verify", "--url", "://"}},
		{"url without scheme", []string{"verify", "--url", "127.0.0.1:8787"}},
		{"url with path", []string{"verify", "--url", server.URL + "/prefix"}},
		{"url with query", []string{"verify", "--url", server.URL + "/?x=1"}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Run(test.args, &stdout, &stderr)
			if code != exitFailure {
				t.Fatalf("args %q: exit = %d, want %d (stderr: %s)", test.args, code, exitFailure, stderr.String())
			}
			if stdout.Len() != 0 {
				t.Fatalf("stdout = %q, want empty: a usage failure claims no result", stdout.String())
			}
			if !strings.Contains(stderr.String(), "usage: fake-jev") {
				t.Fatalf("stderr does not carry usage: %q", stderr.String())
			}
		})
	}
	if requests := server.recorded(); len(requests) != 0 {
		t.Fatalf("a usage failure reached the network: %v", requests)
	}
}

// TestVerifyMakesOneControlCallAndConsultsNoConfig pins C-CLI-008 (§15.3): the
// command issues exactly one GET <base>/__fake/v1/verify with no body, and it
// loads no configuration file even when an invalid one sits in the working
// directory.
func TestVerifyMakesOneControlCallAndConsultsNoConfig(t *testing.T) {
	server := newCannedVerificationServer(t, nethttp.StatusOK, `{"passed":true,"failures":[]}`)
	workDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(workDir, "fake-jev.yaml"), []byte("schemaVersion: 9\nextra: true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(workDir)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"verify", "--url", server.URL}, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit = %d, want %d (stderr: %s)", code, exitOK, stderr.String())
	}
	requests := server.recorded()
	if len(requests) != 1 {
		t.Fatalf("requests = %v, want exactly one", requests)
	}
	if requests[0].Method != nethttp.MethodGet {
		t.Fatalf("method = %q, want GET", requests[0].Method)
	}
	if requests[0].Path != "/__fake/v1/verify" {
		t.Fatalf("path = %q, want /__fake/v1/verify", requests[0].Path)
	}
	if requests[0].Body != "" {
		t.Fatalf("request body = %q, want empty", requests[0].Body)
	}
}

// TestVerifyExitCodesFromControlResponses pins C-CLI-001 (§42.1): the exit code
// tracks the server's answer, an unusable answer fails closed with 2, and only
// a reported verification failure produces 3 on stderr.
func TestVerifyExitCodesFromControlResponses(t *testing.T) {
	cases := []struct {
		name         string
		status       int
		body         string
		wantCode     int
		wantFailure  string
		wantErrorOut bool
	}{
		{name: "passed true", status: nethttp.StatusOK, body: `{"passed":true,"failures":[]}`, wantCode: exitOK},
		{
			name: "passed false", status: nethttp.StatusOK,
			body:     `{"passed":false,"failures":[{"code":"unmatched_request","message":"Request #2 did not match any configured stub.","requestSequence":2,"stubId":null}]}`,
			wantCode: exitVerify, wantFailure: "unmatched_request", wantErrorOut: true,
		},
		{name: "passed false without items", status: nethttp.StatusOK, body: `{"passed":false,"failures":[]}`, wantCode: exitVerify, wantErrorOut: true},
		{name: "server error", status: nethttp.StatusInternalServerError, body: `{"error":"fake_jev_internal_error"}`, wantCode: exitFailure, wantErrorOut: true},
		{name: "control not found", status: nethttp.StatusNotFound, body: `{"error":"fake_jev_control_not_found"}`, wantCode: exitFailure, wantErrorOut: true},
		{name: "no content", status: nethttp.StatusNoContent, body: "", wantCode: exitFailure, wantErrorOut: true},
		{name: "body is not json", status: nethttp.StatusOK, body: "not json", wantCode: exitFailure, wantErrorOut: true},
		{name: "json array", status: nethttp.StatusOK, body: "[]", wantCode: exitFailure, wantErrorOut: true},
		{name: "json null", status: nethttp.StatusOK, body: "null", wantCode: exitFailure, wantErrorOut: true},
		{name: "missing passed", status: nethttp.StatusOK, body: `{"failures":[]}`, wantCode: exitFailure, wantErrorOut: true},
		{name: "passed is not boolean", status: nethttp.StatusOK, body: `{"passed":"no"}`, wantCode: exitFailure, wantErrorOut: true},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			server := newCannedVerificationServer(t, test.status, test.body)
			var stdout, stderr bytes.Buffer
			code := Run([]string{"verify", "--url", server.URL}, &stdout, &stderr)
			if code != test.wantCode {
				t.Fatalf("exit = %d, want %d (stdout: %q stderr: %q)", code, test.wantCode, stdout.String(), stderr.String())
			}
			if got := strings.Contains(stdout.String(), "verification failed"); got {
				t.Fatalf("stdout claims a verification failure: %q", stdout.String())
			}
			if got := stderr.String() != ""; got != test.wantErrorOut {
				t.Fatalf("stderr = %q, want diagnostic = %v", stderr.String(), test.wantErrorOut)
			}
			if test.wantFailure != "" && !strings.Contains(stderr.String(), test.wantFailure) {
				t.Fatalf("stderr does not report %q: %q", test.wantFailure, stderr.String())
			}
		})
	}
}

// TestVerifyUnreachableURLExitsTwo pins the operational-failure row of §42.1:
// an address nothing listens on is a failure to obtain the result, not a
// verification result, so the command exits 2 with a diagnostic on stderr. The
// released port is the OS's own choice (H3), and the listener lifetime is the
// synchronization point (H2).
func TestVerifyUnreachableURLExitsTwo(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	address, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("listener address %v is not a TCP address", listener.Addr())
	}
	baseURL := "http://" + net.JoinHostPort("127.0.0.1", strconv.Itoa(address.Port))
	if err := listener.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"verify", "--url", baseURL}, &stdout, &stderr)
	if code != exitFailure {
		t.Fatalf("exit = %d, want %d (stdout: %q stderr: %q)", code, exitFailure, stdout.String(), stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if stderr.Len() == 0 {
		t.Fatal("stderr is empty, want a diagnostic")
	}
	if strings.Contains(stderr.String(), "usage: fake-jev") {
		t.Fatalf("an operational failure printed usage: %q", stderr.String())
	}
}

// TestVerifyAgainstServer pins C-CLI-008 and V4 end to end: the command reports
// the server's own failure items, repeating it is a read that changes nothing,
// and the verify call never enters the interaction journal (§11.1, §41.9).
func TestVerifyAgainstServer(t *testing.T) {
	t.Run("unknown route fails then clears", func(t *testing.T) {
		dir := t.TempDir()
		child := startServe(t, serveLaunch{readyPath: filepath.Join(dir, "ready.json"), args: []string{"--port", "0"}})
		document := child.waitReady(t)

		status, body := serveRequest(t, nethttp.MethodGet, document.URL+"/nope", nil, nil)
		if status != nethttp.StatusNotFound || !strings.Contains(string(body), "fake_jev_unknown_route") {
			t.Fatalf("unknown route = %d (%s)", status, body)
		}

		for attempt := 0; attempt < 2; attempt++ {
			var stdout, stderr bytes.Buffer
			if code := Run([]string{"verify", "--url", document.URL}, &stdout, &stderr); code != exitVerify {
				t.Fatalf("attempt %d: exit = %d, want %d (stdout: %q stderr: %q)", attempt, code, exitVerify, stdout.String(), stderr.String())
			}
			if !strings.Contains(stderr.String(), "unknown_route") {
				t.Fatalf("attempt %d: stderr does not report the server failure: %q", attempt, stderr.String())
			}
		}
		if records := listRequests(t, document.URL); len(records) != 1 {
			t.Fatalf("journal = %v, want only the data-plane request", records)
		}

		status, body = serveRequest(t, nethttp.MethodDelete, document.URL+"/__fake/v1/requests", nil, nil)
		if status != nethttp.StatusNoContent {
			t.Fatalf("clear requests = %d (%s)", status, body)
		}
		var stdout, stderr bytes.Buffer
		if code := Run([]string{"verify", "--url", document.URL}, &stdout, &stderr); code != exitOK {
			t.Fatalf("exit after clear = %d, want %d (stderr: %s)", code, exitOK, stderr.String())
		}
		if records := listRequests(t, document.URL); len(records) != 0 {
			t.Fatalf("verify calls were journalled: %v", records)
		}

		child.signal(t, syscall.SIGTERM)
		if code := child.wait(t, 10*time.Second); code != exitOK {
			t.Fatalf("serve exit = %d, want 0 (stderr: %s)", code, child.stderr.String())
		}
	})

	t.Run("unsatisfied expectation", func(t *testing.T) {
		dir := t.TempDir()
		configPath := writeConfigFile(t, dir, "schemaVersion: 1\nstubs:\n  - id: never\n    profile: jev/v1\n    when: {}\n    then: {answers: {urgent: {noul: 1}}}\n    expect: {exactly: 1}\n")
		child := startServe(t, serveLaunch{readyPath: filepath.Join(dir, "ready.json"), args: []string{"--port", "0", "--config", configPath}})
		document := child.waitReady(t)

		var stdout, stderr bytes.Buffer
		if code := Run([]string{"verify", "--url", document.URL}, &stdout, &stderr); code != exitVerify {
			t.Fatalf("exit = %d, want %d (stdout: %q stderr: %q)", code, exitVerify, stdout.String(), stderr.String())
		}
		if !strings.Contains(stderr.String(), "expect_exactly") {
			t.Fatalf("stderr does not report the expectation failure: %q", stderr.String())
		}
		if records := listRequests(t, document.URL); len(records) != 0 {
			t.Fatalf("verify calls were journalled: %v", records)
		}

		child.signal(t, syscall.SIGTERM)
		if code := child.wait(t, 10*time.Second); code != exitOK {
			t.Fatalf("serve exit = %d, want 0 (stderr: %s)", code, child.stderr.String())
		}
	})
}

// TestCallVerificationDecodesServerReport pins the transport seam: the CLI
// decodes the §41.9 document, keeps the server's failure order, and rejects a
// response it cannot understand rather than inventing a result.
func TestCallVerificationDecodesServerReport(t *testing.T) {
	body := `{"passed":false,"failures":[` +
		`{"code":"unknown_route","message":"Request used an unknown route.","requestSequence":1,"stubId":null},` +
		`{"code":"expect_at_least","message":"Stub \"a\" expected at least 2 invocations.","requestSequence":null,"stubId":"a"}]}`
	server := newCannedVerificationServer(t, nethttp.StatusOK, body)
	report, err := callVerification(&nethttp.Client{Timeout: verificationTimeout}, server.URL)
	if err != nil {
		t.Fatalf("callVerification: %v", err)
	}
	if report.Passed {
		t.Fatal("passed = true, want false")
	}
	if len(report.Failures) != 2 {
		t.Fatalf("failures = %+v, want two in server order", report.Failures)
	}
	if report.Failures[0].Code != "unknown_route" || report.Failures[1].Code != "expect_at_least" {
		t.Fatalf("failure order changed: %+v", report.Failures)
	}
	if report.Failures[0].RequestSequence == nil || *report.Failures[0].RequestSequence != 1 {
		t.Fatalf("first requestSequence = %v, want 1", report.Failures[0].RequestSequence)
	}
	if report.Failures[1].StubID == nil || *report.Failures[1].StubID != "a" {
		t.Fatalf("second stubId = %v, want a", report.Failures[1].StubID)
	}
	if report.Failures[1].RequestSequence != nil {
		t.Fatalf("second requestSequence = %v, want null", report.Failures[1].RequestSequence)
	}

	for _, malformed := range []string{`{"passed":true,"failures":"none"}`, `{"passed":true,"failures":[{"code":2}]}`} {
		bad := newCannedVerificationServer(t, nethttp.StatusOK, malformed)
		if _, err := callVerification(&nethttp.Client{Timeout: verificationTimeout}, bad.URL); err == nil {
			t.Fatalf("malformed body %s was accepted", malformed)
		}
	}
}

// TestCallVerificationTimeoutIsAnError pins the bounded-timeout row: an
// endpoint that never answers is an operational failure the command maps to
// exit 2, not a verification result. The client timeout is the
// synchronization point, so no sleep is used as sync (H2).
func TestCallVerificationTimeoutIsAnError(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(nethttp.HandlerFunc(func(writer nethttp.ResponseWriter, request *nethttp.Request) {
		<-release
	}))
	t.Cleanup(func() {
		close(release)
		server.Close()
	})

	if _, err := callVerification(&nethttp.Client{Timeout: 200 * time.Millisecond}, server.URL); err == nil {
		t.Fatal("callVerification succeeded against an endpoint that never answers")
	}
}

// TestVerifyURLCanonicalization pins the accepted base-URL forms: the plain form
// §15.3 shows and the same form with a trailing slash both target exactly
// /__fake/v1/verify (V-U2 records that everything else is undefined).
func TestVerifyURLCanonicalization(t *testing.T) {
	server := newCannedVerificationServer(t, nethttp.StatusOK, `{"passed":true,"failures":[]}`)
	for _, rawURL := range []string{server.URL, server.URL + "/"} {
		var stdout, stderr bytes.Buffer
		if code := Run([]string{"verify", "--url", rawURL}, &stdout, &stderr); code != exitOK {
			t.Fatalf("--url %q: exit = %d, want %d (stderr: %s)", rawURL, code, exitOK, stderr.String())
		}
	}
	if requests := server.recorded(); len(requests) != 2 {
		t.Fatalf("requests = %v, want two", requests)
	}
	for _, request := range server.recorded() {
		if request.Path != "/__fake/v1/verify" {
			t.Fatalf("path = %q, want /__fake/v1/verify", request.Path)
		}
	}
}

// TestVerificationReportRoundTripsThroughJSON pins that the CLI's failure type is
// the §41.9 item shape and not a private reinterpretation.
func TestVerificationReportRoundTripsThroughJSON(t *testing.T) {
	encoded, err := json.Marshal(verificationFailure{Code: "journal_full", Message: "full", RequestSequence: nil, StubID: nil})
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != `{"code":"journal_full","message":"full","requestSequence":null,"stubId":null}` {
		t.Fatalf("encoded = %s", encoded)
	}
	var decoded verificationFailure
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Code != "journal_full" || decoded.Message != "full" || decoded.RequestSequence != nil || decoded.StubID != nil {
		t.Fatalf("decoded = %+v", decoded)
	}
}

// TestVerifyHelpPrintsUsage pins the same V-U6 implementation choice for verify.
func TestVerifyHelpPrintsUsage(t *testing.T) {
	for _, args := range [][]string{{"verify", "--help"}, {"verify", "-h"}} {
		var stdout, stderr bytes.Buffer
		if code := Run(args, &stdout, &stderr); code != exitOK {
			t.Fatalf("args %q: exit = %d, want %d (stderr: %s)", args, code, exitOK, stderr.String())
		}
		if !strings.Contains(stdout.String(), "usage: fake-jev") {
			t.Fatalf("args %q: stdout = %q, want usage", args, stdout.String())
		}
		if stderr.Len() != 0 {
			t.Fatalf("args %q: stderr = %q, want empty", args, stderr.String())
		}
	}
}
