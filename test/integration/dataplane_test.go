package integration

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"

	"fake-jev/internal/config"
	jevhttp "fake-jev/internal/host/http"
)

func TestDataPlaneGoldenVectors(t *testing.T) {
	tests := []struct {
		name   string
		config string
		run    func(t *testing.T, client *http.Client, base string, server *jevhttp.Server)
	}{
		{"models", "schemaVersion: 1\n", func(t *testing.T, client *http.Client, base string, _ *jevhttp.Server) {
			response := request(t, client, http.MethodGet, base+"/v1/models?x=1", "")
			wantJSON(t, response, 200, `{"models":[{"name":"jev-latest","description":"Local deterministic fake model provided by fake-jev.","release_date":"1970-01-01"}]}`)
		}},
		{"noul", configWithStub(`{"id":"spam","profile":"jev/v1","when":{"questions":{"spam":"noul"}},"then":{"answers":{"spam":{"noul":0.9}}}}`), func(t *testing.T, client *http.Client, base string, _ *jevhttp.Server) {
			response := request(t, client, http.MethodPost, base+"/v1/systemone", `{"state":"buy now","model":"jev-latest","questions":{"spam":{"type":"noul","instructions":"Is this spam?"}}}`)
			wantJSON(t, response, 200, `{"model":"jev-latest","answers":{"spam":{"type":"noul","noul":0.9}},"usage":{"input_tokens":0,"output_tokens":0}}`)
		}},
		{"choice", configWithStub(`{"id":"route","profile":"jev/v1","when":{"questions":{"route":"choice"}},"then":{"answers":{"route":{"choice":"backend"}}}}`), func(t *testing.T, client *http.Client, base string, _ *jevhttp.Server) {
			response := request(t, client, http.MethodPost, base+"/v1/systemone", `{"state":{},"model":"jev-latest","questions":{"route":{"type":"choice","criteria":{"frontend":"UI","backend":"API","infra":"Infrastructure"}}}}`)
			wantJSON(t, response, 200, `{"model":"jev-latest","answers":{"route":{"type":"choice","choice":"backend","confidence":1,"probabilities":{"frontend":0,"backend":1,"infra":0}}},"usage":{"input_tokens":0,"output_tokens":0}}`)
		}},
		{"score", configWithStub(`{"id":"severity","profile":"jev/v1","when":{"questions":{"severity":"score"}},"then":{"answers":{"severity":{"score":1.5}}}}`), func(t *testing.T, client *http.Client, base string, _ *jevhttp.Server) {
			response := request(t, client, http.MethodPost, base+"/v1/systemone", `{"state":{},"model":"jev-latest","questions":{"severity":{"type":"score","criteria":["Can wait","Needs attention this week","Needs attention today"]}}}`)
			wantJSON(t, response, 200, `{"model":"jev-latest","answers":{"severity":{"type":"score","score":1.5,"confidence":1,"legend":{"0":"Can wait","1":"Needs attention this week","2":"Needs attention today"},"probabilities":{"0":0,"1":0.5,"2":0.5}}},"usage":{"input_tokens":0,"output_tokens":0}}`)
		}},
		{"exact-question-set", configWithStub(`{"id":"route","profile":"jev/v1","when":{"questions":{"route":"choice"}},"then":{"answers":{"route":{"choice":"backend"}}}}`), func(t *testing.T, client *http.Client, base string, server *jevhttp.Server) {
			response := request(t, client, http.MethodPost, base+"/v1/systemone", `{"state":{},"model":"jev-latest","questions":{"route":{"type":"choice","criteria":{"backend":"API"}},"urgent":{"type":"noul"}}}`)
			wantJSON(t, response, 501, `{"error":"fake_jev_unmatched_request","message":"No configured stub matched this request.","profile":"jev/v1","operation":"systemone"}`)
			if got := server.Engine().VerificationFailures()[0].Code; got != "unmatched_request" {
				t.Fatalf("failure code = %q, want unmatched_request", got)
			}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server, client, base, closeServer := startServer(t, test.config)
			defer closeServer()
			test.run(t, client, base, server)
		})
	}
}

func TestDataPlanePriorityAndRegistrationOrder(t *testing.T) {
	config := `{"schemaVersion":1,"stubs":[
		{"id":"a","profile":"jev/v1","priority":0,"when":{"questions":{"q":"noul"}},"then":{"answers":{"q":{"noul":0}}}},
		{"id":"b","profile":"jev/v1","priority":10,"when":{"questions":{"q":"noul"}},"then":{"answers":{"q":{"noul":0.5}}}},
		{"id":"c","profile":"jev/v1","priority":10,"when":{"questions":{"q":"noul"}},"then":{"answers":{"q":{"noul":1}}}}
	]}`
	_, client, base, closeServer := startServer(t, config)
	defer closeServer()
	body := `{"state":{},"model":"jev-latest","questions":{"q":{"type":"noul"}}}`
	wantJSON(t, request(t, client, http.MethodPost, base+"/v1/systemone", body), 200, `{"model":"jev-latest","answers":{"q":{"type":"noul","noul":0.5}},"usage":{"input_tokens":0,"output_tokens":0}}`)

	withoutB := `{"schemaVersion":1,"stubs":[
		{"id":"a","profile":"jev/v1","priority":0,"when":{"questions":{"q":"noul"}},"then":{"answers":{"q":{"noul":0}}}},
		{"id":"c","profile":"jev/v1","priority":10,"when":{"questions":{"q":"noul"}},"then":{"answers":{"q":{"noul":1}}}}
	]}`
	_, client2, base2, closeServer2 := startServer(t, withoutB)
	defer closeServer2()
	wantJSON(t, request(t, client2, http.MethodPost, base2+"/v1/systemone", body), 200, `{"model":"jev-latest","answers":{"q":{"type":"noul","noul":1}},"usage":{"input_tokens":0,"output_tokens":0}}`)
}

func TestDataPlaneSequenceAndRawResponses(t *testing.T) {
	t.Run("sequence exhaustion", func(t *testing.T) {
		config := configWithStub(`{"id":"retry-flow","profile":"jev/v1","when":{"questions":{"retry":"noul"}},"then":{"sequence":[{"answers":{"retry":{"noul":1}}},{"answers":{"retry":{"noul":0}}}]}}`)
		server, client, base, closeServer := startServer(t, config)
		defer closeServer()
		body := `{"state":{},"model":"jev-latest","questions":{"retry":{"type":"noul"}}}`
		wantJSON(t, request(t, client, http.MethodPost, base+"/v1/systemone", body), 200, `{"model":"jev-latest","answers":{"retry":{"type":"noul","noul":1}},"usage":{"input_tokens":0,"output_tokens":0}}`)
		wantJSON(t, request(t, client, http.MethodPost, base+"/v1/systemone", body), 200, `{"model":"jev-latest","answers":{"retry":{"type":"noul","noul":0}},"usage":{"input_tokens":0,"output_tokens":0}}`)
		response := request(t, client, http.MethodPost, base+"/v1/systemone", body)
		wantJSON(t, response, 409, `{"error":"fake_jev_sequence_exhausted","message":"Configured response sequence is exhausted.","stubId":"retry-flow"}`)
		stub := server.Engine().Registry.Stubs()[0]
		if stub.InvocationCount != 3 || stub.SequencePosition != 2 {
			t.Fatalf("sequence state = invocations %d position %d", stub.InvocationCount, stub.SequencePosition)
		}
		if failures := server.Engine().VerificationFailures(); len(failures) != 1 || failures[0].Code != "sequence_exhausted" || failures[0].RequestSequence == nil || *failures[0].RequestSequence != 3 {
			t.Fatalf("verification failures = %+v", failures)
		}
	})

	t.Run("raw provider error remains matched", func(t *testing.T) {
		config := configWithStub(`{"id":"rate-limit","profile":"jev/v1","when":{"questions":{"q":"noul"}},"then":{"raw":{"status":429,"headers":{"content-type":"application/json"},"body":{"detail":"rate limited"}}}}`)
		server, client, base, closeServer := startServer(t, config)
		defer closeServer()
		body := `{"state":{},"model":"jev-latest","questions":{"q":{"type":"noul"}}}`
		response := request(t, client, http.MethodPost, base+"/v1/systemone", body)
		wantJSON(t, response, 429, `{"detail":"rate limited"}`)
		if len(server.Engine().VerificationFailures()) != 0 || server.Engine().Interactions()[0].Outcome != "matched" || server.Engine().Interactions()[0].ResponseStatus != 429 {
			t.Fatalf("raw state = interactions %+v failures %+v", server.Engine().Interactions(), server.Engine().VerificationFailures())
		}
	})
}

func TestDataPlaneValidationLimitsAndRoutes(t *testing.T) {
	t.Run("validation", func(t *testing.T) {
		server, client, base, closeServer := startServer(t, configWithStub(`{"id":"q","profile":"jev/v1","when":{"questions":{"q":"noul"}},"then":{"answers":{"q":{"noul":1}}}}`))
		defer closeServer()
		wantJSON(t, request(t, client, http.MethodPost, base+"/v1/systemone", `{`), 422, `{"detail":[{"loc":["body"],"msg":"Invalid JSON","type":"json_invalid"}]}`)
		wantJSON(t, request(t, client, http.MethodPost, base+"/v1/systemone", `{"model":"jev-latest","questions":{"q":{"type":"noul"}}}`), 422, `{"detail":[{"loc":["body","state"],"msg":"Field required","type":"missing"}]}`)
		if got := server.Engine().Interactions(); len(got) != 2 || got[0].Outcome != "validation_error" || got[1].Outcome != "validation_error" {
			t.Fatalf("validation journal = %+v", got)
		}
		if failures := server.Engine().VerificationFailures(); len(failures) != 2 || failures[0].Code != "validation_error" || failures[1].Code != "validation_error" {
			t.Fatalf("validation failures = %+v", failures)
		}
	})

	t.Run("journal limit", func(t *testing.T) {
		server, client, base, closeServer := startServer(t, `{"schemaVersion":1,"limits":{"maxInteractions":1}}`)
		defer closeServer()
		wantJSON(t, request(t, client, http.MethodGet, base+"/v1/models", ""), 200, `{"models":[{"name":"jev-latest","description":"Local deterministic fake model provided by fake-jev.","release_date":"1970-01-01"}]}`)
		wantJSON(t, request(t, client, http.MethodGet, base+"/v1/models", ""), 507, `{"error":"fake_jev_journal_full","message":"Interaction journal limit reached. Reset or increase maxInteractions."}`)
		if failures := server.Engine().VerificationFailures(); len(failures) != 1 || failures[0].Code != "journal_full" || failures[0].RequestSequence != nil {
			t.Fatalf("journal limit failures = %+v", failures)
		}
	})

	t.Run("payload limit", func(t *testing.T) {
		server, client, base, closeServer := startServer(t, `{"schemaVersion":1,"limits":{"dataPlaneBodyBytes":1024},"stubs":[{"id":"q","profile":"jev/v1","when":{"questions":{"q":"noul"}},"then":{"answers":{"q":{"noul":1}}}}]}`)
		defer closeServer()
		wantJSON(t, request(t, client, http.MethodPost, base+"/v1/systemone", strings.Repeat("x", 1025)), 413, `{"error":"fake_jev_payload_too_large","message":"Request body exceeds the configured limit."}`)
		if got := server.Engine().Interactions(); len(got) != 1 || got[0].Outcome != "payload_too_large" || got[0].RawBody != nil || len(server.Engine().VerificationFailures()) != 1 {
			t.Fatalf("payload state = interactions %+v failures %+v", got, server.Engine().VerificationFailures())
		}
		if invocations := server.Engine().Registry.Stubs()[0].InvocationCount; invocations != 0 {
			t.Fatalf("payload invocation count = %d", invocations)
		}
	})

	t.Run("exact routes", func(t *testing.T) {
		server, client, base, closeServer := startServer(t, `{"schemaVersion":1}`)
		defer closeServer()
		wantJSON(t, request(t, client, http.MethodGet, base+"/v1/models?x=1", ""), 200, `{"models":[{"name":"jev-latest","description":"Local deterministic fake model provided by fake-jev.","release_date":"1970-01-01"}]}`)
		for _, target := range []string{"/v1/models/", "/v1/systemone"} {
			wantJSON(t, request(t, client, http.MethodGet, base+target, ""), 404, `{"error":"fake_jev_unknown_route","message":"No active compatibility profile handles this route."}`)
		}
		wantJSON(t, request(t, client, http.MethodPost, base+"/v1/models", ""), 404, `{"error":"fake_jev_unknown_route","message":"No active compatibility profile handles this route."}`)
		if got := server.Engine().Interactions(); len(got) != 4 || len(server.Engine().VerificationFailures()) != 3 {
			t.Fatalf("route state = interactions %+v failures %+v", got, server.Engine().VerificationFailures())
		}
	})
}

func startServer(t *testing.T, document string) (*jevhttp.Server, *http.Client, string, func()) {
	t.Helper()
	cfg, err := config.Load([]byte(document))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	server, err := jevhttp.NewServerFromConfig(cfg)
	if err != nil {
		t.Fatalf("construct server: %v", err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	errorsCh := make(chan error, 1)
	client := &http.Client{}
	go func() { errorsCh <- server.Serve(listener) }()
	closeServer := func() {
		t.Helper()
		client.CloseIdleConnections()
		if err := listener.Close(); err != nil {
			t.Fatalf("close listener: %v", err)
		}
		if err := <-errorsCh; err != nil && !errors.Is(err, net.ErrClosed) {
			t.Fatalf("serve: %v", err)
		}
	}
	return server, client, "http://" + listener.Addr().String(), closeServer
}

func request(t *testing.T, client *http.Client, method, target, body string) *http.Response {
	t.Helper()
	response, err := client.Do(mustRequest(t, method, target, body))
	if err != nil {
		t.Fatalf("request %s %s: %v", method, target, err)
	}
	return response
}

func mustRequest(t *testing.T, method, target, body string) *http.Request {
	t.Helper()
	request, err := http.NewRequest(method, target, strings.NewReader(body))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	return request
}

func wantJSON(t *testing.T, response *http.Response, status int, expected string) {
	t.Helper()
	defer response.Body.Close()
	if response.StatusCode != status {
		t.Fatalf("status = %d, want %d", response.StatusCode, status)
	}
	actual, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	var actualValue, expectedValue any
	actualDecoder := json.NewDecoder(bytes.NewReader(actual))
	expectedDecoder := json.NewDecoder(strings.NewReader(expected))
	if err := actualDecoder.Decode(&actualValue); err != nil {
		t.Fatalf("decode actual %q: %v", actual, err)
	}
	if err := expectedDecoder.Decode(&expectedValue); err != nil {
		t.Fatalf("decode expected: %v", err)
	}
	if !jsonEqual(actualValue, expectedValue) {
		t.Fatalf("body = %s, want semantic JSON %s", actual, expected)
	}
}

func jsonEqual(left, right any) bool {
	leftJSON, leftErr := json.Marshal(left)
	rightJSON, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftJSON, rightJSON)
}

func configWithStub(stub string) string {
	return `{"schemaVersion":1,"stubs":[` + stub + `]}`
}
