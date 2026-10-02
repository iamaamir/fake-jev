// Go example: point an HTTP client at fake-jev and check the answer.
//
// Run it against a started server (see examples/README.md):
//
//	./fake-jev run --config examples/fake-jev.yaml -- go run ./examples/go
//
// Standard library only, and no go.mod of its own: the file compiles as part
// of the root module, so `go build ./cmd/fake-jev`, `go test ./...`, and
// `go vet ./...` already compile-check it. No provider SDK, no credential,
// no model, and no network beyond the loopback server.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const defaultBaseURL = "http://127.0.0.1:8787"

// requestTimeout bounds one call; a hung server must fail the example instead
// of blocking forever.
const requestTimeout = 10 * time.Second

type modelsDocument struct {
	Models []modelEntry `json:"models"`
}

type modelEntry struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ReleaseDate string `json:"release_date"`
}

type systemoneRequest struct {
	State     map[string]any      `json:"state"`
	Model     string              `json:"model"`
	Questions map[string]question `json:"questions"`
}

type question struct {
	Type     string         `json:"type"`
	Criteria map[string]any `json:"criteria,omitempty"`
}

type answerDocument struct {
	Model   string                     `json:"model"`
	Answers map[string]json.RawMessage `json:"answers"`
	Usage   usage                      `json:"usage"`
}

type usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type choiceAnswer struct {
	Type          string             `json:"type"`
	Choice        string             `json:"choice"`
	Confidence    float64            `json:"confidence"`
	Probabilities map[string]float64 `json:"probabilities"`
}

type noulAnswer struct {
	Type string  `json:"type"`
	Noul float64 `json:"noul"`
}

type verifyDocument struct {
	Passed   bool            `json:"passed"`
	Failures []verifyFailure `json:"failures"`
}

type verifyFailure struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func main() {
	baseURL := os.Getenv("FAKE_JEV_URL")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	failures, err := run(baseURL)
	if err != nil {
		emit(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
	for _, failure := range failures {
		emit(os.Stderr, "FAIL: %s\n", failure)
	}
	if len(failures) > 0 {
		os.Exit(1)
	}
	emit(os.Stdout, "OK: the fixture answers matched the expected values\n")
}

// emit writes formatted output and consumes the write error in one checked
// place: a failed write already lost the message, and the repository's lint
// rule forbids discarding an error-returning call at the call site.
func emit(w io.Writer, format string, a ...any) {
	if _, err := fmt.Fprintf(w, format, a...); err != nil {
		return
	}
}

func run(baseURL string) ([]string, error) {
	client := &http.Client{Timeout: requestTimeout}
	var failures []string
	check := func(ok bool, message string) {
		if !ok {
			failures = append(failures, message)
		}
	}
	emit(os.Stdout, "fake-jev base URL: %s\n", baseURL)

	// 1. Model listing. §39.1 returns the configured (here: default) list.
	var models modelsDocument
	raw, err := call(client, http.MethodGet, baseURL+"/v1/models", nil, &models)
	if err != nil {
		return failures, err
	}
	show("GET /v1/models", raw)
	check(len(models.Models) == 1, "expected exactly one listed model")
	if len(models.Models) == 1 {
		check(models.Models[0].Name == "jev-latest", "expected the model jev-latest")
	}

	// 2. One System One call with a choice and a noul question.
	// §39.5: any bearer value is accepted and never logged, so a real key is
	// neither needed nor read.
	request := systemoneRequest{
		State: map[string]any{"task": "triage"},
		Model: "jev-latest",
		Questions: map[string]question{
			"route": {
				Type: "choice",
				Criteria: map[string]any{
					"frontend": "the view layer",
					"backend":  "the api layer",
					"infra":    "the deployment",
				},
			},
			"urgent": {Type: "noul", Criteria: map[string]any{"deadline": "today"}},
		},
	}
	body, err := json.Marshal(request)
	if err != nil {
		return failures, err
	}
	var answers answerDocument
	raw, err = call(client, http.MethodPost, baseURL+"/v1/systemone", body, &answers)
	if err != nil {
		return failures, err
	}
	show("POST /v1/systemone", raw)
	check(answers.Model == "jev-latest", "response model should echo the request model")
	check(answers.Usage == usage{}, "usage should be the zero default")

	var route choiceAnswer
	if routeRaw, ok := answers.Answers["route"]; ok {
		if err := json.Unmarshal(routeRaw, &route); err != nil {
			return failures, err
		}
	} else {
		check(false, "route answer should be present")
	}
	check(route.Type == "choice", "route answer should be a choice")
	check(route.Choice == "backend", "route choice should be backend")
	check(route.Confidence == 0.91, "route confidence should be 0.91")
	check(route.Probabilities["backend"] == 0.91, "backend probability should be 0.91")

	urgentRaw, ok := answers.Answers["urgent"]
	if !ok {
		check(false, "urgent answer should be present")
	}
	if ok {
		var urgent noulAnswer
		if err := json.Unmarshal(urgentRaw, &urgent); err != nil {
			return failures, err
		}
		check(urgent.Type == "noul", "urgent answer should be a noul")
		check(urgent.Noul == 0.94, "urgent noul should be 0.94")
	}

	// 3. Control-plane verification: the server decides whether the exchanges
	// above matched the fixture. §41.9.
	var verification verifyDocument
	raw, err = call(client, http.MethodGet, baseURL+"/__fake/v1/verify", nil, &verification)
	if err != nil {
		return failures, err
	}
	show("GET /__fake/v1/verify", raw)
	check(verification.Passed, "server verification should pass")
	for _, failure := range verification.Failures {
		check(false, fmt.Sprintf("server failure %s: %s", failure.Code, failure.Message))
	}
	return failures, nil
}

// call performs one request and decodes the JSON reply into out. It reports
// the raw reply body so the example can print exactly what the server sent.
func call(client *http.Client, method, url string, body []byte, out any) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	request, err := http.NewRequest(method, url, reader)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer not-a-real-key")
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		return raw, fmt.Errorf("%s %s returned %d: %s", method, url, response.StatusCode, raw)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return raw, fmt.Errorf("%s %s: %w", method, url, err)
	}
	return raw, nil
}

func show(label string, raw []byte) {
	emit(os.Stdout, "%s: %s\n", label, raw)
}
