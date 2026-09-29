package v1

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

func fixtureRequest(t *testing.T, questions string) Request {
	t.Helper()
	body := `{"state":{},"model":"jev-latest","questions":` + questions + `}`
	request, err := ValidateRequest([]byte(body))
	if err != nil {
		t.Fatalf("ValidateRequest() error = %v", err)
	}
	return request
}

func decodeAnswers(t *testing.T, answers map[string]json.RawMessage) map[string]any {
	t.Helper()
	encoded, err := json.Marshal(answers)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	return decoded
}

func TestGenerateAnswersGoldenVectors(t *testing.T) {
	tests := []struct {
		name      string
		questions string
		fixture   string
		want      string
	}{
		{
			name:      "noul",
			questions: `{"urgent":{"type":"noul"}}`,
			fixture:   `{"urgent":{"noul":0.9}}`,
			want:      `{"urgent":{"type":"noul","noul":0.9}}`,
		},
		{
			name:      "choice one hot",
			questions: `{"route":{"type":"choice","criteria":{"frontend":null,"backend":null,"infra":null}}}`,
			fixture:   `{"route":{"choice":"backend"}}`,
			want:      `{"route":{"type":"choice","choice":"backend","confidence":1,"probabilities":{"frontend":0,"backend":1,"infra":0}}}`,
		},
		{
			name:      "fractional score",
			questions: `{"severity":{"type":"score","criteria":["Can wait","Needs attention this week","Needs attention today"]}}`,
			fixture:   `{"severity":{"score":1.5}}`,
			want:      `{"severity":{"type":"score","score":1.5,"confidence":1,"probabilities":{"0":0,"1":0.5,"2":0.5},"legend":{"0":"Can wait","1":"Needs attention this week","2":"Needs attention today"}}}`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := fixtureRequest(t, test.questions)
			answers, err := GenerateAnswers(request, json.RawMessage(test.fixture))
			if err != nil {
				t.Fatalf("GenerateAnswers() error = %v", err)
			}
			var got, want any
			gotJSON, err := json.Marshal(answers)
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(gotJSON, &got); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(test.want), &want); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("answers = %s, want %s", gotJSON, test.want)
			}
		})
	}
}

func TestGenerateAnswersNoulFormsAndValidation(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    float64
		invalid bool
	}{
		{name: "number", value: `0.94`, want: 0.94},
		{name: "true", value: `true`, want: 1},
		{name: "false", value: `false`, want: 0},
		{name: "negative", value: `-0.1`, invalid: true},
		{name: "greater than one", value: `1.1`, invalid: true},
		{name: "string coercion prohibited", value: `"0.5"`, invalid: true},
	}
	request := fixtureRequest(t, `{"n":{"type":"noul"}}`)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			answers, err := GenerateAnswers(request, json.RawMessage(`{"n":{"noul":`+test.value+`}}`))
			if test.invalid {
				if err == nil {
					t.Fatal("GenerateAnswers() succeeded for invalid noul")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var answer struct {
				Noul float64 `json:"noul"`
			}
			if err := json.Unmarshal(answers["n"], &answer); err != nil {
				t.Fatal(err)
			}
			if answer.Noul != test.want {
				t.Fatalf("noul = %v, want %v", answer.Noul, test.want)
			}
		})
	}
}

func TestGenerateAnswersChoiceProbabilityRules(t *testing.T) {
	request := fixtureRequest(t, `{"route":{"type":"choice","criteria":{"a":null,"b":null,"c":null}}}`)
	tests := []struct {
		name       string
		fixture    string
		wantError  bool
		confidence float64
	}{
		{name: "explicit selected maximum", fixture: `{"route":{"choice":"b","probabilities":{"a":0.2,"b":0.7,"c":0.1}}}`, confidence: 0.7},
		{name: "maximum tie permitted", fixture: `{"route":{"choice":"b","probabilities":{"a":0.5,"b":0.5,"c":0}}}`, confidence: 0.5},
		{name: "sum tolerance", fixture: `{"route":{"choice":"b","probabilities":{"a":0.2,"b":0.7,"c":0.1000005}}}`, confidence: 0.7},
		{name: "unknown key", fixture: `{"route":{"choice":"b","probabilities":{"a":0,"b":1,"extra":0}}}`, wantError: true},
		{name: "missing key", fixture: `{"route":{"choice":"b","probabilities":{"a":0,"b":1}}}`, wantError: true},
		{name: "bad sum", fixture: `{"route":{"choice":"b","probabilities":{"a":0.2,"b":0.2,"c":0.2}}}`, wantError: true},
		{name: "out of range probability", fixture: `{"route":{"choice":"b","probabilities":{"a":-0.1,"b":1.1,"c":0}}}`, wantError: true},
		{name: "not maximum", fixture: `{"route":{"choice":"b","probabilities":{"a":0.8,"b":0.1,"c":0.1}}}`, wantError: true},
		{name: "unknown selected choice", fixture: `{"route":{"choice":"d"}}`, wantError: true},
		{name: "out of range confidence", fixture: `{"route":{"choice":"b","confidence":2}}`, wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			answers, err := GenerateAnswers(request, json.RawMessage(test.fixture))
			if test.wantError {
				if err == nil {
					t.Fatal("GenerateAnswers() succeeded for invalid choice")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var answer struct {
				Confidence float64 `json:"confidence"`
			}
			if err := json.Unmarshal(answers["route"], &answer); err != nil {
				t.Fatal(err)
			}
			if answer.Confidence != test.confidence {
				t.Fatalf("confidence = %v, want %v", answer.Confidence, test.confidence)
			}
		})
	}
}

func TestGenerateAnswersScoreRules(t *testing.T) {
	request := fixtureRequest(t, `{"severity":{"type":"score","criteria":["low","medium","high"]}}`)
	tests := []struct {
		name      string
		fixture   string
		wantError bool
		want      map[string]float64
	}{
		{name: "integer one hot", fixture: `{"severity":{"score":2}}`, want: map[string]float64{"0": 0, "1": 0, "2": 1}},
		{name: "fractional interpolation", fixture: `{"severity":{"score":0.25}}`, want: map[string]float64{"0": 0.75, "1": 0.25, "2": 0}},
		{name: "explicit expected value", fixture: `{"severity":{"score":1.5,"probabilities":{"0":0,"1":0.5,"2":0.5}}}`, want: map[string]float64{"0": 0, "1": 0.5, "2": 0.5}},
		{name: "unknown key", fixture: `{"severity":{"score":1,"probabilities":{"0":0,"1":1,"2":0,"3":0}}}`, wantError: true},
		{name: "missing key", fixture: `{"severity":{"score":1,"probabilities":{"0":0,"1":1}}}`, wantError: true},
		{name: "bad sum", fixture: `{"severity":{"score":1,"probabilities":{"0":0,"1":0.2,"2":0}}}`, wantError: true},
		{name: "out of range probability", fixture: `{"severity":{"score":1,"probabilities":{"0":-0.1,"1":1.1,"2":0}}}`, wantError: true},
		{name: "bad expected value", fixture: `{"severity":{"score":1,"probabilities":{"0":0,"1":0,"2":1}}}`, wantError: true},
		{name: "out of range score", fixture: `{"severity":{"score":3}}`, wantError: true},
		{name: "supplied legend prohibited", fixture: `{"severity":{"score":1,"legend":{"0":"wrong","1":"wrong","2":"wrong"}}}`, wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			answers, err := GenerateAnswers(request, json.RawMessage(test.fixture))
			if test.wantError {
				if err == nil {
					t.Fatal("GenerateAnswers() succeeded for invalid score")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var answer struct {
				Probabilities map[string]float64 `json:"probabilities"`
				Legend        map[string]string  `json:"legend"`
			}
			if err := json.Unmarshal(answers["severity"], &answer); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(answer.Probabilities, test.want) {
				t.Fatalf("probabilities = %#v, want %#v", answer.Probabilities, test.want)
			}
			if !reflect.DeepEqual(answer.Legend, map[string]string{"0": "low", "1": "medium", "2": "high"}) {
				t.Fatalf("legend = %#v", answer.Legend)
			}
		})
	}
}

func TestGenerateAnswersExactCoverageAndTypes(t *testing.T) {
	request := fixtureRequest(t, `{"n":{"type":"noul"},"route":{"type":"choice","criteria":{"yes":null,"no":null}},"severity":{"type":"score","criteria":["low","high"]}}`)
	tests := []struct {
		name    string
		fixture string
	}{
		{name: "missing answer", fixture: `{"n":{"noul":true},"route":{"choice":"yes"}}`},
		{name: "extra answer", fixture: `{"n":{"noul":true},"route":{"choice":"yes"},"severity":{"score":0},"other":{"noul":false}}`},
		{name: "mixed partial answer", fixture: `{"n":{"noul":true},"route":{"choice":"yes"}}`},
		{name: "noul choice mismatch", fixture: `{"n":{"choice":"yes"},"route":{"choice":"yes"},"severity":{"score":0}}`},
		{name: "choice score mismatch", fixture: `{"n":{"noul":true},"route":{"score":0},"severity":{"score":0}}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := GenerateAnswers(request, json.RawMessage(test.fixture)); err == nil {
				t.Fatal("GenerateAnswers() succeeded for invalid answer map")
			} else {
				var invalid *InvalidStubResponseError
				if !errors.As(err, &invalid) {
					t.Fatalf("error = %T, want InvalidStubResponseError", err)
				}
			}
		})
	}
}

func TestGenerateAnswersRejectsMalformedAnswerDocument(t *testing.T) {
	request := fixtureRequest(t, `{"n":{"type":"noul"}}`)
	for _, fixture := range []string{`null`, `[]`, `{"n":null}`, `{"n":{"noul":null}}`} {
		if _, err := GenerateAnswers(request, json.RawMessage(fixture)); err == nil {
			t.Fatalf("GenerateAnswers(%s) succeeded", fixture)
		}
	}
}

func TestFiniteProbabilityBoundaries(t *testing.T) {
	request := fixtureRequest(t, `{"route":{"type":"choice","criteria":{"a":null,"b":null}}}`)
	for _, fixture := range []string{
		`{"route":{"choice":"a","confidence":0,"probabilities":{"a":1,"b":0}}}`,
		`{"route":{"choice":"a","confidence":1,"probabilities":{"a":1,"b":0}}}`,
	} {
		if _, err := GenerateAnswers(request, json.RawMessage(fixture)); err != nil {
			t.Fatal(err)
		}
	}
}
