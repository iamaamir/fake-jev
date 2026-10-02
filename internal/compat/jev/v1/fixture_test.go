package v1

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
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

// TestValidateFixtureAnswers covers the statically decidable §13 rules. Each
// failing case names one rule, and the "sequence element" cases call the
// validator the way the CLI does for one then.sequence element.
func TestValidateFixtureAnswers(t *testing.T) {
	noul := map[string]string{"n": "noul"}
	choice := map[string]string{"route": "choice"}
	score := map[string]string{"severity": "score"}
	all := map[string]string{"n": "noul", "route": "choice", "severity": "score"}
	tests := []struct {
		name      string
		questions map[string]string
		answers   string
		wantError string
	}{
		// Valid fixtures, one per helper and per payload shape.
		{name: "noul number", questions: noul, answers: `{"n":{"noul":0.94}}`},
		{name: "noul true", questions: noul, answers: `{"n":{"noul":true}}`},
		{name: "noul false", questions: noul, answers: `{"n":{"noul":false}}`},
		{name: "noul boundaries", questions: noul, answers: `{"n":{"noul":0}}`},
		{name: "noul upper boundary", questions: noul, answers: `{"n":{"noul":1}}`},
		{name: "choice minimal", questions: choice, answers: `{"route":{"choice":"backend"}}`},
		{name: "choice confidence", questions: choice, answers: `{"route":{"choice":"backend","confidence":0.91}}`},
		{name: "choice explicit probabilities", questions: choice, answers: `{"route":{"choice":"backend","confidence":0.91,"probabilities":{"frontend":0.06,"backend":0.91,"infra":0.03}}}`},
		{name: "choice sum tolerance", questions: choice, answers: `{"route":{"choice":"a","probabilities":{"a":0.5,"b":0.5000005}}}`},
		{name: "score minimal", questions: score, answers: `{"severity":{"score":2}}`},
		{name: "score fractional", questions: score, answers: `{"severity":{"score":1.5}}`},
		{name: "score explicit probabilities", questions: score, answers: `{"severity":{"score":1.5,"confidence":0.8,"probabilities":{"0":0,"1":0.5,"2":0.5}}}`},
		{name: "every helper", questions: all, answers: `{"n":{"noul":0.94},"route":{"choice":"a","probabilities":{"a":0.5,"b":0.5}},"severity":{"score":1,"probabilities":{"0":0,"1":1}}}`},
		{name: "sequence element", questions: noul, answers: `{"n":{"noul":true}}`},
		{name: "wildcard noul", answers: `{"a":{"noul":true}}`},
		{name: "wildcard choice", answers: `{"route":{"choice":"a","probabilities":{"a":0.5,"b":0.5}}}`},
		{name: "wildcard score", answers: `{"severity":{"score":1.5}}`},
		{name: "no answers form", answers: ``},

		// Name-set rules.
		{name: "answers key set too small", questions: noul, answers: `{}`, wantError: "must exactly cover the configured questions"},
		{name: "wildcard answers key set empty", answers: `{}`, wantError: "answers must not be empty"},
		{name: "answers key set too large", questions: noul, answers: `{"n":{"noul":true},"other":{"noul":false}}`, wantError: "must exactly cover the configured questions"},
		{name: "missing answer", questions: noul, answers: `{"other":{"noul":true}}`, wantError: `missing answer for question "n"`},
		{name: "absent question answered", questions: noul, answers: `{"route":{"noul":true}}`, wantError: `missing answer for question "n"`},
		{name: "sequence element key mismatch", questions: score, answers: `{"route":{"choice":"b"}}`, wantError: `missing answer for question "severity"`},

		// Helper/type agreement.
		{name: "choice helper on noul question", questions: noul, answers: `{"n":{"choice":"a"}}`, wantError: "noul helper is required"},
		{name: "score helper on choice question", questions: choice, answers: `{"route":{"score":1}}`, wantError: "choice helper is required"},
		{name: "noul helper on score question", questions: score, answers: `{"severity":{"noul":true}}`, wantError: "score helper is required"},

		// noul payload rules.
		{name: "noul above range", questions: noul, answers: `{"n":{"noul":1.5}}`, wantError: "noul must be a finite number in [0,1] or a boolean"},
		{name: "noul below range", questions: noul, answers: `{"n":{"noul":-0.1}}`, wantError: "noul must be a finite number in [0,1] or a boolean"},
		{name: "noul string coercion prohibited", questions: noul, answers: `{"n":{"noul":"0.5"}}`, wantError: "noul must be a finite number in [0,1] or a boolean"},
		{name: "noul beyond float64 range", questions: noul, answers: `{"n":{"noul":1e999}}`, wantError: "noul must be a finite number in [0,1] or a boolean"},
		{name: "noul extra field", questions: noul, answers: `{"n":{"noul":0.5,"confidence":0.5}}`, wantError: "noul helper has unknown fields"},
		{name: "wildcard noul above range", answers: `{"a":{"noul":2}}`, wantError: "noul must be a finite number in [0,1] or a boolean"},
		{name: "wildcard mixed helpers", answers: `{"a":{"noul":1,"choice":"x"}}`, wantError: "noul helper has unknown fields"},

		// choice payload rules.
		{name: "choice value not a string", questions: choice, answers: `{"route":{"choice":3}}`, wantError: "choice must be a string"},
		{name: "choice missing", questions: choice, answers: `{"route":{"confidence":0.5}}`, wantError: "choice helper is required"},
		{name: "choice unknown field", questions: choice, answers: `{"route":{"choice":"a","legend":{"a":"x"}}}`, wantError: `choice helper has unknown field "legend"`},
		{name: "choice confidence above range", questions: choice, answers: `{"route":{"choice":"a","confidence":1.5}}`, wantError: "confidence must be a finite number in [0,1]"},
		{name: "choice probability above range", questions: choice, answers: `{"route":{"choice":"a","probabilities":{"a":1.1,"b":-0.1}}}`, wantError: `probability "a" must be a finite number in [0,1]`},
		{name: "choice probabilities not an object", questions: choice, answers: `{"route":{"choice":"a","probabilities":1}}`, wantError: "probabilities must be an object"},
		{name: "choice probabilities empty map", questions: choice, answers: `{"route":{"choice":"a","probabilities":{}}}`, wantError: "probabilities must sum to 1"},
		{name: "choice probability is not a number", questions: choice, answers: `{"route":{"choice":"a","probabilities":{"a":{"b":1}}}}`, wantError: `probability "a" must be a finite number in [0,1]`},
		{name: "choice probability beyond float64 range", questions: choice, answers: `{"route":{"choice":"a","probabilities":{"a":1e999}}}`, wantError: `probability "a" must be a finite number in [0,1]`},
		{name: "choice probabilities do not sum", questions: choice, answers: `{"route":{"choice":"a","probabilities":{"a":0.2,"b":0.2,"c":0.2}}}`, wantError: "probabilities must sum to 1"},

		// score payload rules.
		{name: "score not a number", questions: score, answers: `{"severity":{"score":"1"}}`, wantError: "score must be a finite number"},
		{name: "score missing", questions: score, answers: `{"severity":{"confidence":0.5}}`, wantError: "score helper is required"},
		{name: "score legend prohibited", questions: score, answers: `{"severity":{"score":1,"legend":{"0":"x"}}}`, wantError: `score helper has unknown field "legend"`},
		{name: "score confidence above range", questions: score, answers: `{"severity":{"score":1,"confidence":1.1}}`, wantError: "confidence must be a finite number in [0,1]"},
		{name: "score probability above range", questions: score, answers: `{"severity":{"score":1,"probabilities":{"0":-0.1,"1":1.1}}}`, wantError: `probability "0" must be a finite number in [0,1]`},
		{name: "score probabilities do not sum", questions: score, answers: `{"severity":{"score":1,"probabilities":{"0":0.5,"1":0.7}}}`, wantError: "probabilities must sum to 1"},
		{name: "score probabilities empty map", questions: score, answers: `{"severity":{"score":1,"probabilities":{}}}`, wantError: "probabilities must sum to 1"},
		{name: "score beyond float64 range", questions: score, answers: `{"severity":{"score":1e999}}`, wantError: "score must be a finite number"},

		// Document and member shapes.
		{name: "answers is an array", questions: noul, answers: `[]`, wantError: "answers must be an object"},
		{name: "answers is null", questions: noul, answers: `null`, wantError: "answers must be an object"},
		{name: "answers member is not an object", questions: noul, answers: `{"n":null}`, wantError: "helper must be an object"},
		{name: "answers member is a number", questions: noul, answers: `{"n":3}`, wantError: "helper must be an object"},
		{name: "wildcard member is not an object", answers: `{"a":3}`, wantError: "helper must be an object"},
		{name: "wildcard declares no helper", answers: `{"a":{"confidence":0.5}}`, wantError: "helper must declare noul, choice, or score"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateFixtureAnswers(test.questions, json.RawMessage(test.answers))
			if test.wantError == "" {
				if err != nil {
					t.Fatalf("ValidateFixtureAnswers() error = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("ValidateFixtureAnswers() succeeded, want an error containing %q", test.wantError)
			}
			if !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("error = %q, want it to contain %q", err, test.wantError)
			}
		})
	}
}

// TestValidateFixtureAnswersReportsUnknownFieldDeterministically pins the
// diagnostic for a payload carrying several unknown fields at once: the
// reported field is the lexicographically first, so the same configuration
// always produces the same stderr line (§42.5) instead of whichever key map
// iteration happened to return.
func TestValidateFixtureAnswersReportsUnknownFieldDeterministically(t *testing.T) {
	answers := json.RawMessage(`{"route":{"choice":"a","z1":1,"z2":2,"z3":3,"z4":4,"z5":5,"z6":6,"z7":7}}`)
	const want = `choice helper has unknown field "z1"`
	for attempt := 0; attempt < 32; attempt++ {
		err := ValidateFixtureAnswers(map[string]string{"route": "choice"}, answers)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("attempt %d: error = %v, want it to contain %q", attempt, err, want)
		}
	}
}

// FuzzValidateFixtureAnswers asserts the validator's boundary properties over
// arbitrary bytes: it never panics, identical input always yields the same
// accept/reject decision and the same diagnostic, and a rejection always
// carries a message, so a violation can never be silently swallowed. The
// question maps cover the configured, wildcard, and unknown-type branches.
func FuzzValidateFixtureAnswers(f *testing.F) {
	seeds := []struct {
		answers string
		name    string
		kind    string
	}{
		{answers: `{"n":{"noul":0.94}}`, name: "n", kind: "noul"},
		{answers: `{"route":{"choice":"a","probabilities":{"a":0.5,"b":0.5}}}`, name: "route", kind: "choice"},
		{answers: `{"severity":{"score":1.5,"probabilities":{"0":0,"1":0.5,"2":0.5}}}`, name: "severity", kind: "score"},
		{answers: `{}`, name: "n", kind: "noul"},
		{answers: `null`, name: "n", kind: "noul"},
		{answers: `{"n":{"noul":1e999}}`, name: "n", kind: "noul"},
		{answers: `{"n":{"noul":true,"choice":"a","z9":1}}`, name: "n", kind: "noul"},
	}
	for _, seed := range seeds {
		f.Add([]byte(seed.answers), seed.name, seed.kind)
	}
	f.Fuzz(func(t *testing.T, answers []byte, name, kind string) {
		questions := []map[string]string{
			nil,
			{name: "noul", name + "-choice": "choice"},
			{name: kind},
		}
		for _, configured := range questions {
			first := ValidateFixtureAnswers(configured, json.RawMessage(answers))
			second := ValidateFixtureAnswers(configured, json.RawMessage(answers))
			if (first == nil) != (second == nil) {
				t.Fatalf("questions %v answers %q: accept/reject differs between identical calls (%v then %v)", configured, answers, first, second)
			}
			if first == nil {
				continue
			}
			if first.Error() == "" {
				t.Fatalf("questions %v answers %q: empty diagnostic", configured, answers)
			}
			if first.Error() != second.Error() {
				t.Fatalf("questions %v answers %q: diagnostics differ between identical calls (%q then %q)", configured, answers, first, second)
			}
		}
	})
}

// TestValidateFixtureAnswersNeverAcceptsWhatGenerationRejects pins the shared
// subset of the static and request-time paths: every fixture response
// generation accepts for a question set must also be statically valid, so
// validate can never reject a working configuration. The fixtures include ones
// generation rejects only for request-relative reasons (unknown choice, missing
// maximum, score range, expected value), which the static path must accept.
func TestValidateFixtureAnswersNeverAcceptsWhatGenerationRejects(t *testing.T) {
	request := fixtureRequest(t, `{"n":{"type":"noul"},"route":{"type":"choice","criteria":{"a":null,"b":null,"c":null}},"severity":{"type":"score","criteria":["low","medium","high"]}}`)
	configured := map[string]string{"n": "noul", "route": "choice", "severity": "score"}
	fixtures := []string{
		`{"n":{"noul":0.94},"route":{"choice":"b"},"severity":{"score":1.5}}`,
		`{"n":{"noul":true},"route":{"choice":"b","confidence":0.91,"probabilities":{"a":0.06,"b":0.91,"c":0.03}},"severity":{"score":1.5,"confidence":0.8,"probabilities":{"0":0,"1":0.5,"2":0.5}}}`,
		`{"n":{"noul":0.2},"route":{"choice":"c","probabilities":{"a":0.2,"b":0.3,"c":0.5}},"severity":{"score":2}}`,
		`{"n":{"noul":0.94},"route":{"choice":"d"},"severity":{"score":1.5}}`,
		`{"n":{"noul":0.94},"route":{"choice":"b","probabilities":{"a":0.8,"b":0.1,"c":0.1}},"severity":{"score":1.5}}`,
		`{"n":{"noul":0.94},"route":{"choice":"b"},"severity":{"score":9}}`,
		`{"n":{"noul":0.94},"route":{"choice":"b"},"severity":{"score":1.5,"probabilities":{"0":0,"1":0,"2":1}}}`,
		`{"n":{"noul":1.5},"route":{"choice":"b"},"severity":{"score":1.5}}`,
	}
	accepted := 0
	for _, fixture := range fixtures {
		if _, err := GenerateAnswers(request, json.RawMessage(fixture)); err != nil {
			continue
		}
		accepted++
		if err := ValidateFixtureAnswers(configured, json.RawMessage(fixture)); err != nil {
			t.Errorf("GenerateAnswers accepts %s but ValidateFixtureAnswers rejects it: %v", fixture, err)
		}
	}
	if accepted < 3 {
		t.Fatalf("only %d fixtures reached the generation-accepting subset, want at least 3", accepted)
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
