package v1

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strconv"
	"testing"
)

// The fuzz target in this file establishes that untrusted jev/v1 input cannot
// panic the profile boundary and cannot produce state that disagrees with the
// profile's own validator: decoding, validation, and answer generation are
// deterministic, a rejection is always reported with the validator's own
// diagnostic, a generated answer always matches the question it answers, and
// neither the request body nor the configured answers are mutated. Traces
// C-QUAL-004 and C-QUAL-002.

const (
	profileFuzzMaxMethod        = 64
	profileFuzzMaxTarget        = 512
	profileFuzzMaxAuthorization = 512
	profileFuzzMaxBody          = 16 << 10
	profileFuzzMaxConfigured    = 16 << 10
	profileFuzzMaxQuestions     = 32
)

// assertGeneratedAnswers checks the request-relative answer contract: the
// answer key set equals the question name set, every answer declares its
// question's type and carries the matching helper, and a rejection is always a
// non-empty InvalidStubResponseError. Generation is run twice so that a hidden
// generator state would show up as differing output.
func assertGeneratedAnswers(t *testing.T, request Request, configured []byte) {
	t.Helper()
	first, firstErr := GenerateAnswers(request, json.RawMessage(configured))
	second, secondErr := GenerateAnswers(request, json.RawMessage(configured))
	if (firstErr == nil) != (secondErr == nil) {
		t.Fatalf("GenerateAnswers accept/reject differs between identical calls: %v then %v", firstErr, secondErr)
	}
	if firstErr != nil {
		if firstErr.Error() == "" {
			t.Fatal("GenerateAnswers reported an empty diagnostic")
		}
		var invalid *InvalidStubResponseError
		if !errors.As(firstErr, &invalid) {
			t.Fatalf("GenerateAnswers error %v is not an InvalidStubResponseError", firstErr)
		}
		return
	}
	firstJSON, err := json.Marshal(first)
	if err != nil {
		t.Fatalf("marshal generated answers: %v", err)
	}
	secondJSON, err := json.Marshal(second)
	if err != nil {
		t.Fatalf("marshal generated answers: %v", err)
	}
	if !bytes.Equal(firstJSON, secondJSON) {
		t.Fatalf("GenerateAnswers is not reproducible: %s then %s", firstJSON, secondJSON)
	}
	if len(first) != len(request.Questions) {
		t.Fatalf("generated %d answers for %d questions", len(first), len(request.Questions))
	}
	for name, question := range request.Questions {
		answer, exists := first[name]
		if !exists {
			t.Fatalf("question %q has no generated answer", name)
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(answer, &fields); err != nil || fields == nil {
			t.Fatalf("answer %q is not a JSON object: %s", name, answer)
		}
		typeRaw, exists := fields["type"]
		if !exists {
			t.Fatalf("answer %q declares no type: %s", name, answer)
		}
		var answerType string
		if err := json.Unmarshal(typeRaw, &answerType); err != nil {
			t.Fatalf("answer %q has an undecodable type: %s", name, answer)
		}
		if answerType != question.Type {
			t.Fatalf("answer %q has type %q, want %q", name, answerType, question.Type)
		}
		if _, exists := fields[question.Type]; !exists {
			t.Fatalf("answer %q carries no %q helper: %s", name, question.Type, answer)
		}
	}
}

// FuzzProfileUntrustedInput feeds arbitrary routing metadata, request bodies,
// and configured answers to the jev/v1 profile. For every input the profile
// must either accept the request or report a controlled error, must agree
// exactly with ValidateRequest on the systemone route, must be reproducible, and
// must not corrupt the caller-owned input slices.
func FuzzProfileUntrustedInput(f *testing.F) {
	const canonicalBody = `{"state":{},"model":"jev-latest","questions":{"urgent":{"type":"noul"}}}`
	// A body with more questions than the generation cap, so the seed corpus
	// covers the branch that skips answer generation.
	var manyQuestions []byte
	manyQuestions = append(manyQuestions, `{"state":{},"model":"jev-latest","questions":{`...)
	for i := 0; i < 64; i++ {
		if i > 0 {
			manyQuestions = append(manyQuestions, ',')
		}
		manyQuestions = append(manyQuestions, `"q`...)
		manyQuestions = strconv.AppendInt(manyQuestions, int64(i), 10)
		manyQuestions = append(manyQuestions, `":{"type":"noul"}`...)
	}
	manyQuestions = append(manyQuestions, '}', '}')

	seeds := []struct {
		name          string
		method        string
		target        string
		authorization string
		body          string
		configured    string
	}{
		{"models", "GET", "/v1/models", "", "", ""},
		{"systemone-valid", "POST", "/v1/systemone", "Bearer secret", canonicalBody, `{"urgent":{"noul":0.9}}`},
		{"systemone-invalid-answer", "POST", "/v1/systemone", "Bearer secret", canonicalBody, `{"urgent":{"noul":1.5}}`},
		{"configured-null", "POST", "/v1/systemone", "", canonicalBody, `null`},
		{"configured-array", "POST", "/v1/systemone", "", canonicalBody, `[]`},
		{"configured-empty-object", "POST", "/v1/systemone", "", canonicalBody, `{}`},
		{"body-truncated", "POST", "/v1/systemone", "", `{`, `{"urgent":{"noul":0.9}}`},
		{"body-null", "POST", "/v1/systemone", "", `null`, ""},
		{"body-array", "POST", "/v1/systemone", "", `[]`, ""},
		{"body-duplicate-questions", "POST", "/v1/systemone", "", `{"state":{},"model":"m","questions":{"a":{"type":"noul"}},"questions":{"b":{"type":"noul"}}}`, `{"b":{"noul":1}}`},
		{"body-many-questions", "POST", "/v1/systemone", "", string(manyQuestions), `{"q0":{"noul":1}}`},
		{"body-invalid-utf8", "POST", "/v1/systemone", "", "\x80\x81{\"state\":{}}", ""},
		{"unknown-target", "POST", "/v1/other", "", canonicalBody, `{"urgent":{"noul":0.9}}`},
	}
	for _, seed := range seeds {
		f.Add(seed.method, seed.target, seed.authorization, []byte(seed.body), []byte(seed.configured))
	}

	f.Fuzz(func(t *testing.T, method, target, authorization string, body, configured []byte) {
		if len(method) > profileFuzzMaxMethod || len(target) > profileFuzzMaxTarget ||
			len(authorization) > profileFuzzMaxAuthorization || len(body) > profileFuzzMaxBody ||
			len(configured) > profileFuzzMaxConfigured {
			return
		}
		bodyBefore := append([]byte(nil), body...)
		configuredBefore := append([]byte(nil), configured...)

		profile := DefaultProfile()
		meta := RequestMeta{Method: method, Target: target}
		hostRequest := HostRequest{Meta: meta, Headers: map[string]string{"Authorization": authorization}, Body: body}

		exchange, decodeErr := profile.Decode(hostRequest)
		repeated, repeatedErr := profile.Decode(hostRequest)

		switch route := profile.Route(meta); route {
		case SystemOneRoute:
			request, validationErr := ValidateRequest(body)
			if validationErr == nil && decodeErr != nil {
				t.Fatalf("Decode rejected %q, which ValidateRequest accepted: %v", body, decodeErr)
			}
			if validationErr != nil {
				if decodeErr == nil {
					t.Fatalf("Decode accepted %q, which ValidateRequest rejected: %v", body, validationErr)
				}
				if decodeErr.Error() != validationErr.Error() {
					t.Fatalf("Decode diagnostic %q differs from the validator diagnostic %q", decodeErr, validationErr)
				}
			}
			if decodeErr == nil {
				if exchange.Profile != "jev/v1" {
					t.Fatalf("decoded exchange profile is %q, want jev/v1", exchange.Profile)
				}
				if exchange.Operation != SystemOneRoute.Operation() {
					t.Fatalf("decoded exchange operation is %q, want %q", exchange.Operation, SystemOneRoute.Operation())
				}
				if len(request.Questions) <= profileFuzzMaxQuestions {
					assertGeneratedAnswers(t, request, configured)
				}
			}
		case ModelsRoute:
			if decodeErr != nil {
				t.Fatalf("Decode rejected the models route: %v", decodeErr)
			}
			if exchange.Profile != "jev/v1" || exchange.Operation != ModelsRoute.Operation() || len(exchange.Questions) != 0 {
				t.Fatalf("models exchange = %#v", exchange)
			}
			withoutBody, err := profile.Decode(HostRequest{Meta: meta, Headers: hostRequest.Headers})
			if err != nil || !reflect.DeepEqual(exchange, withoutBody) {
				t.Fatalf("models decoding depends on the request body: %v, %#v", err, withoutBody)
			}
		default:
			if decodeErr == nil {
				t.Fatalf("Decode accepted the unknown route %s %q", method, target)
			}
		}

		if (decodeErr == nil) != (repeatedErr == nil) {
			t.Fatalf("Decode accept/reject differs between identical calls: %v then %v", decodeErr, repeatedErr)
		}
		if decodeErr != nil {
			if decodeErr.Error() == "" {
				t.Fatal("Decode reported an empty diagnostic")
			}
			if decodeErr.Error() != repeatedErr.Error() {
				t.Fatalf("Decode diagnostics differ between identical calls: %q then %q", decodeErr, repeatedErr)
			}
		}
		if !reflect.DeepEqual(exchange, repeated) {
			t.Fatalf("Decode is not reproducible: %#v then %#v", exchange, repeated)
		}
		if !bytes.Equal(bodyBefore, body) || !bytes.Equal(configuredBefore, configured) {
			t.Fatalf("profile decoding mutated the caller input: body %q, configured %q", body, configured)
		}
	})
}
