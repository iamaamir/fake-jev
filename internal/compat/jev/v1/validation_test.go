package v1

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestRecognizeRouteExactness(t *testing.T) {
	tests := []struct {
		name   string
		method string
		target string
		want   Route
	}{
		{"models", "GET", "/v1/models", ModelsRoute},
		{"models query", "GET", "/v1/models?x=1", ModelsRoute},
		{"models trailing slash", "GET", "/v1/models/", UnknownRoute},
		{"models wrong method", "POST", "/v1/models", UnknownRoute},
		{"systemone", "POST", "/v1/systemone", SystemOneRoute},
		{"systemone query", "POST", "/v1/systemone?x=1", SystemOneRoute},
		{"systemone trailing slash", "POST", "/v1/systemone/", UnknownRoute},
		{"systemone wrong method", "GET", "/v1/systemone", UnknownRoute},
		{"unknown path", "GET", "/v1/other", UnknownRoute},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := RecognizeRoute(test.method, test.target); got != test.want {
				t.Fatalf("RecognizeRoute() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestValidationGoldenErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
		want Detail
	}{
		{
			name: "malformed JSON",
			body: `{`,
			want: Detail{Loc: []string{"body"}, Msg: "Invalid JSON", Type: "json_invalid"},
		},
		{
			name: "missing state",
			body: `{"model":"jev-latest","questions":{"x":{"type":"noul"}}}`,
			want: Detail{Loc: []string{"body", "state"}, Msg: "Field required", Type: "missing"},
		},
		{
			name: "unsupported question type",
			body: `{"state":"x","model":"jev-latest","questions":{"x":{"type":"unsupported"}}}`,
			want: Detail{Loc: []string{"body", "questions", "x", "type"}, Msg: "Unsupported question type", Type: "literal_error"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ValidateRequest([]byte(test.body))
			if err == nil || len(err.Envelope.Detail) != 1 {
				t.Fatalf("ValidateRequest() error = %#v, want one detail", err)
			}
			if !reflect.DeepEqual(err.Envelope.Detail[0], test.want) {
				t.Fatalf("detail = %#v, want %#v", err.Envelope.Detail[0], test.want)
			}
			wire, marshalErr := json.Marshal(err)
			if marshalErr != nil {
				t.Fatal(marshalErr)
			}
			var envelope ErrorEnvelope
			if json.Unmarshal(wire, &envelope) != nil || !reflect.DeepEqual(envelope.Detail[0], test.want) {
				t.Fatalf("wire envelope = %s, want detail %#v", wire, test.want)
			}
		})
	}
}

func TestValidationOrderAndFrozenSchema(t *testing.T) {
	tests := []struct {
		name string
		body string
		want Detail
	}{
		{
			name: "required fields precede question errors",
			body: `{"questions":{},"model":3}`,
			want: Detail{Loc: []string{"body", "state"}, Msg: "Field required", Type: "missing"},
		},
		{
			name: "model precedes questions",
			body: `{"state":"x","questions":{}}`,
			want: Detail{Loc: []string{"body", "model"}, Msg: "Field required", Type: "missing"},
		},
		{
			name: "questions precedes entries",
			body: `{"state":"x","model":"m","questions":[]}`,
			want: Detail{Loc: []string{"body", "questions"}, Msg: "Invalid value", Type: "value_error"},
		},
		{
			name: "valid JSON non-object is a schema error",
			body: `[]`,
			want: Detail{Loc: []string{"body"}, Msg: "Invalid value", Type: "value_error"},
		},
		{
			name: "question names are lexicographic",
			body: `{"state":"x","model":"m","questions":{"z":{"type":"unsupported"},"a":{"type":"bad"}}}`,
			want: Detail{Loc: []string{"body", "questions", "a", "type"}, Msg: "Unsupported question type", Type: "literal_error"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ValidateRequest([]byte(test.body))
			if err == nil || !reflect.DeepEqual(err.Envelope.Detail[0], test.want) {
				t.Fatalf("error = %#v, want %#v", err, test.want)
			}
		})
	}
}

func TestSupportedQuestionsAndCriteriaRules(t *testing.T) {
	valid := []string{
		`{"state":[],"model":"arbitrary","questions":{"n":{"type":"noul"}}}`,
		`{"state":{},"model":"m","questions":{"n":{"type":"noul","criteria":null,"instructions":null}}}`,
		`{"state":"x","model":"m","questions":{"c":{"type":"choice","criteria":{"yes":[1],"no":null}}}}`,
		`{"state":"x","model":"m","questions":{"s":{"type":"score","criteria":["low",{},[]]}}}`,
		`{"state":"x","model":"m","unknown":{"kept":true},"questions":{"n":{"type":"noul","unknown":false}}}`,
	}
	for _, body := range valid {
		if request, err := ValidateRequest([]byte(body)); err != nil {
			t.Errorf("valid request rejected: %s: %v", body, err)
		} else if request.Model == "" || len(request.Questions) != 1 {
			t.Errorf("decoded request incomplete: %#v", request)
		}
	}

	invalid := []struct {
		name string
		body string
		loc  []string
	}{
		{"state null", `{"state":null,"model":"m","questions":{"n":{"type":"noul"}}}`, []string{"body", "state"}},
		{"model non-string", `{"state":"x","model":false,"questions":{"n":{"type":"noul"}}}`, []string{"body", "model"}},
		{"empty questions", `{"state":"x","model":"m","questions":{}}`, []string{"body", "questions"}},
		{"question non-object", `{"state":"x","model":"m","questions":{"n":null}}`, []string{"body", "questions", "n"}},
		{"choice empty criteria", `{"state":"x","model":"m","questions":{"c":{"type":"choice","criteria":{}}}}`, []string{"body", "questions", "c", "criteria"}},
		{"score empty criteria", `{"state":"x","model":"m","questions":{"s":{"type":"score","criteria":[]}}}`, []string{"body", "questions", "s", "criteria"}},
		{"score null element", `{"state":"x","model":"m","questions":{"s":{"type":"score","criteria":[null]}}}`, []string{"body", "questions", "s", "criteria"}},
		{"invalid instructions", `{"state":"x","model":"m","questions":{"n":{"type":"noul","instructions":true}}}`, []string{"body", "questions", "n", "instructions"}},
	}
	for _, test := range invalid {
		t.Run(test.name, func(t *testing.T) {
			_, err := ValidateRequest([]byte(test.body))
			if err == nil || !reflect.DeepEqual(err.Envelope.Detail[0].Loc, test.loc) {
				t.Fatalf("error = %#v, want location %v", err, test.loc)
			}
		})
	}
}
