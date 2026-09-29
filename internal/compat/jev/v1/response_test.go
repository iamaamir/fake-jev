package v1

import (
	"encoding/json"
	"reflect"
	"testing"

	"fake-jev/internal/config"
)

func responseRequest(t *testing.T) Request {
	t.Helper()
	request, err := ValidateRequest([]byte(`{"state":{},"model":"request-model","questions":{"q":{"type":"noul"}}}`))
	if err != nil {
		t.Fatal(err)
	}
	return request
}

func TestEncodeSuccessDefaultsAndOverrides(t *testing.T) {
	request := responseRequest(t)
	answers := map[string]json.RawMessage{"q": json.RawMessage(`{"type":"noul","noul":1}`)}
	tests := []struct {
		name       string
		model      string
		usage      *config.UsageConfig
		wantModel  string
		wantUsage  Usage
		wantStatus int
	}{
		{name: "request model and zero usage", wantModel: "request-model", wantUsage: Usage{}, wantStatus: 200},
		{name: "configured model and usage", model: "configured-model", usage: &config.UsageConfig{InputTokens: 12, OutputTokens: 7}, wantModel: "configured-model", wantUsage: Usage{InputTokens: 12, OutputTokens: 7}, wantStatus: 200},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			encoded, err := EncodeSuccess(request, answers, test.model, test.usage)
			if err != nil {
				t.Fatal(err)
			}
			if encoded.Status != test.wantStatus || encoded.Headers["Content-Type"] != JSONContentType {
				t.Fatalf("response metadata = %#v, want status %d and JSON content type", encoded, test.wantStatus)
			}
			var got Response
			if err := json.Unmarshal(encoded.Body, &got); err != nil {
				t.Fatal(err)
			}
			if got.Model != test.wantModel || got.Usage != test.wantUsage || len(got.Answers) != 1 {
				t.Fatalf("response = %#v, want model %q usage %#v and one answer", got, test.wantModel, test.wantUsage)
			}
		})
	}
}

func TestEncodeResponseUsesFixtureAndExactWireShape(t *testing.T) {
	request := responseRequest(t)
	encoded, err := EncodeResponse(request, config.ResponseConfig{Answers: json.RawMessage(`{"q":{"noul":true}}`)})
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(encoded.Body, &got); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"model":   "request-model",
		"answers": map[string]any{"q": map[string]any{"type": "noul", "noul": float64(1)}},
		"usage":   map[string]any{"input_tokens": float64(0), "output_tokens": float64(0)},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestEncodeSuccessRejectsUnsafeResultValues(t *testing.T) {
	request := responseRequest(t)
	tests := []struct {
		name    string
		answers map[string]json.RawMessage
		usage   *config.UsageConfig
	}{
		{name: "missing answer", answers: map[string]json.RawMessage{}},
		{name: "unexpected answer", answers: map[string]json.RawMessage{"other": json.RawMessage(`{}`)}},
		{name: "invalid answer JSON", answers: map[string]json.RawMessage{"q": json.RawMessage(`{`)}},
		{name: "negative usage", answers: map[string]json.RawMessage{"q": json.RawMessage(`{}`)}, usage: &config.UsageConfig{InputTokens: -1}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := EncodeSuccess(request, test.answers, "", test.usage); err == nil {
				t.Fatal("unsafe response unexpectedly succeeded")
			}
		})
	}
}

func TestEncodeModelsGoldenAndConfiguredOrder(t *testing.T) {
	tests := []struct {
		name   string
		models []Model
		want   string
	}{
		{
			name: "default",
			want: `{"models":[{"name":"jev-latest","description":"Local deterministic fake model provided by fake-jev.","release_date":"1970-01-01"}]}`,
		},
		{
			name:   "configured order",
			models: []Model{{Name: "second", Description: "B", ReleaseDate: "2024-02-02"}, {Name: "first", Description: "A", ReleaseDate: "2024-01-01"}},
			want:   `{"models":[{"name":"second","description":"B","release_date":"2024-02-02"},{"name":"first","description":"A","release_date":"2024-01-01"}]}`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			encoded, err := EncodeModels(test.models)
			if err != nil {
				t.Fatal(err)
			}
			if encoded.Status != 200 || encoded.Headers["Content-Type"] != JSONContentType {
				t.Fatalf("metadata = %#v", encoded)
			}
			if string(encoded.Body) != test.want {
				t.Fatalf("body = %s, want %s", encoded.Body, test.want)
			}
		})
	}
}

func TestEncodeRawResponse(t *testing.T) {
	tests := []struct {
		name       string
		raw        config.RawResponse
		wantStatus int
		wantBody   string
		wantType   string
		wantHeader string
	}{
		{
			name:       "json body gets default content type",
			raw:        config.RawResponse{Status: 429, Body: json.RawMessage(`{"error":"busy"}`)},
			wantStatus: 429,
			wantBody:   `{"error":"busy"}`,
			wantType:   JSONContentType,
		},
		{
			name:       "explicit content type is case insensitive",
			raw:        config.RawResponse{Status: 201, Headers: map[string]string{"content-type": "text/plain"}, Body: json.RawMessage(`"ok"`)},
			wantStatus: 201,
			wantBody:   `"ok"`,
			wantHeader: "text/plain",
		},
		{
			name:       "omitted body stays empty",
			raw:        config.RawResponse{Status: 204, Headers: map[string]string{"X-Test": "yes"}},
			wantStatus: 204,
			wantHeader: "yes",
		},
		{
			name:       "explicit null body is serialized",
			raw:        config.RawResponse{Status: 200, Body: json.RawMessage(`null`)},
			wantStatus: 200,
			wantBody:   `null`,
			wantType:   JSONContentType,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			encoded, err := EncodeRawResponse(test.raw)
			if err != nil {
				t.Fatal(err)
			}
			if encoded.Status != test.wantStatus || string(encoded.Body) != test.wantBody {
				t.Fatalf("response = %#v, want status %d body %q", encoded, test.wantStatus, test.wantBody)
			}
			if test.wantType != "" && encoded.Headers["Content-Type"] != test.wantType {
				t.Fatalf("content type = %q, want %q", encoded.Headers["Content-Type"], test.wantType)
			}
			if test.wantHeader != "" && encoded.Headers["content-type"] != test.wantHeader && encoded.Headers["X-Test"] != test.wantHeader {
				t.Fatalf("headers = %#v, want configured header %q", encoded.Headers, test.wantHeader)
			}
		})
	}
	if _, err := EncodeRawResponse(config.RawResponse{Status: 99}); err == nil {
		t.Fatal("invalid raw status unexpectedly succeeded")
	}
	if _, err := EncodeRawResponse(config.RawResponse{Status: 200, Body: json.RawMessage(`{"unterminated"`)}); err == nil {
		t.Fatal("invalid raw JSON body unexpectedly succeeded")
	}
}

func TestProfileAcceptsAuthorizationWithoutInspectingIt(t *testing.T) {
	profile := DefaultProfile()
	for _, authorization := range []string{"", "Bearer fake", "Bearer arbitrary"} {
		for _, contentType := range []string{"", "application/json", "text/plain"} {
			headers := map[string]string{"Authorization": authorization}
			if contentType != "" {
				headers["Content-Type"] = contentType
			}
			exchange, err := profile.Decode(HostRequest{
				Meta:    RequestMeta{Method: "POST", Target: "/v1/systemone"},
				Headers: headers,
				Body:    []byte(`{"state":{},"model":"m","questions":{"q":{"type":"noul"}}}`),
			})
			if err != nil {
				t.Fatalf("authorization %q content type %q: Decode() error = %v", authorization, contentType, err)
			}
			if exchange.Model != "m" {
				t.Fatalf("authorization %q content type %q changed decoded model: %#v", authorization, contentType, exchange)
			}
		}
	}
	models, err := profile.Models()
	if err != nil {
		t.Fatal(err)
	}
	if models.Status != 200 {
		t.Fatalf("models status = %d, want 200", models.Status)
	}
}

func TestProfilePreservesLargeJSONNumbers(t *testing.T) {
	profile := DefaultProfile()
	exchange, err := profile.Decode(HostRequest{
		Meta: RequestMeta{Method: "POST", Target: "/v1/systemone"},
		Body: []byte(`{"state":{"large":90071992547409931234567890},"model":"m","questions":{"q":{"type":"noul"}}}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	state, ok := exchange.State.(map[string]any)
	if !ok {
		t.Fatalf("state = %#v, want object", exchange.State)
	}
	if got, ok := state["large"].(json.Number); !ok || got.String() != "90071992547409931234567890" {
		t.Fatalf("large state number = %#v, want exact json.Number", state["large"])
	}
}
