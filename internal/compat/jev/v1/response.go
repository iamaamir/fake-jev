package v1

import (
	"encoding/json"
	"fmt"
	"strings"

	"fake-jev/internal/config"
)

const JSONContentType = "application/json"

// Usage is the usage object in a successful non-raw response.
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// Response is the successful jev/v1 response shape. Answers are already
// encoded answer objects; response construction never interprets them.
type Response struct {
	Model   string                     `json:"model"`
	Answers map[string]json.RawMessage `json:"answers"`
	Usage   Usage                      `json:"usage"`
}

// EncodedResponse is the profile-neutral HTTP response produced by encoding.
// The host is responsible for writing its status, headers, and body.
type EncodedResponse struct {
	Status  int
	Headers map[string]string
	Body    []byte
}

// ResponseResult is the selected response supplied to the compatibility
// profile. Response contains the configured response form; Answers can be
// supplied by a provider-neutral result after fixture generation.
type ResponseResult struct {
	Request  Request
	Response config.ResponseConfig
	Answers  map[string]json.RawMessage
}

// EncodeResponse encodes a configured response for request. Raw responses use
// the raw wire rules; configured answers are converted using the strict v1
// fixture helpers before the success envelope is serialized.
func EncodeResponse(request Request, configured config.ResponseConfig) (EncodedResponse, error) {
	return encodeResponse(request, configured, nil)
}

// EncodeResult is the Profile.Encode-compatible response encoder.
func EncodeResult(result ResponseResult) (EncodedResponse, error) {
	return encodeResponse(result.Request, result.Response, result.Answers)
}

func encodeResponse(request Request, configured config.ResponseConfig, answers map[string]json.RawMessage) (EncodedResponse, error) {
	if configured.Raw != nil {
		return EncodeRawResponse(*configured.Raw)
	}
	if answers == nil {
		var err error
		answers, err = GenerateAnswers(request, configured.Answers)
		if err != nil {
			return EncodedResponse{}, err
		}
	}
	return EncodeSuccess(request, answers, configured.Model, configured.Usage)
}

// EncodeSuccess creates the exact non-raw §39.4 response shape. An empty model
// override means that the request model is echoed, as required by §13.6.
func EncodeSuccess(request Request, answers map[string]json.RawMessage, model string, usage *config.UsageConfig) (EncodedResponse, error) {
	if err := validateResponseAnswers(request, answers); err != nil {
		return EncodedResponse{}, err
	}
	if usage != nil && (usage.InputTokens < 0 || usage.OutputTokens < 0) {
		return EncodedResponse{}, invalidStubResponse("usage token counts must be non-negative")
	}
	if model == "" {
		model = request.Model
	}
	response := Response{Model: model, Answers: cloneAnswers(answers)}
	if usage != nil {
		response.Usage = Usage{InputTokens: usage.InputTokens, OutputTokens: usage.OutputTokens}
	}
	body, err := json.Marshal(response)
	if err != nil {
		return EncodedResponse{}, err
	}
	return EncodedResponse{Status: 200, Headers: map[string]string{"Content-Type": JSONContentType}, Body: body}, nil
}

// EncodeJSONError serializes any non-raw JSON error body with the profile's
// required content type. The host supplies the status code.
func EncodeJSONError(status int, value any) (EncodedResponse, error) {
	body, err := json.Marshal(value)
	if err != nil {
		return EncodedResponse{}, err
	}
	return EncodedResponse{Status: status, Headers: map[string]string{"Content-Type": JSONContentType}, Body: body}, nil
}

// EncodeRawResponse applies §39.7. Body is serialized as JSON exactly once;
// an omitted body remains an empty response body.
func EncodeRawResponse(raw config.RawResponse) (EncodedResponse, error) {
	if raw.Status < 100 || raw.Status > 599 {
		return EncodedResponse{}, fmt.Errorf("raw.status must be between 100 and 599")
	}
	headers := cloneHeaders(raw.Headers)
	var body []byte
	if raw.Body != nil {
		encoded, err := json.Marshal(raw.Body)
		if err != nil {
			return EncodedResponse{}, err
		}
		body = encoded
		if !hasContentType(headers) {
			headers["Content-Type"] = JSONContentType
		}
	}
	return EncodedResponse{Status: raw.Status, Headers: headers, Body: body}, nil
}

func validateResponseAnswers(request Request, answers map[string]json.RawMessage) error {
	if len(answers) != len(request.Questions) {
		return invalidStubResponse("answers must exactly cover request questions")
	}
	for name := range request.Questions {
		answer, ok := answers[name]
		if !ok {
			return invalidStubResponse("missing answer for question %q", name)
		}
		if !json.Valid(answer) {
			return invalidStubResponse("answer for question %q must be valid JSON", name)
		}
	}
	for name := range answers {
		if _, ok := request.Questions[name]; !ok {
			return invalidStubResponse("answer provided for absent question %q", name)
		}
	}
	return nil
}

func cloneAnswers(answers map[string]json.RawMessage) map[string]json.RawMessage {
	result := make(map[string]json.RawMessage, len(answers))
	for name, answer := range answers {
		result[name] = append(json.RawMessage(nil), answer...)
	}
	return result
}

func cloneHeaders(headers map[string]string) map[string]string {
	result := make(map[string]string, len(headers))
	for name, value := range headers {
		result[name] = value
	}
	return result
}

func hasContentType(headers map[string]string) bool {
	for name := range headers {
		if strings.EqualFold(name, "Content-Type") {
			return true
		}
	}
	return false
}
