package v1

import (
	"encoding/json"
)

// Request is the decoded frozen public request schema. Fields retains unknown
// top-level values so the compatibility boundary does not discard raw data.
type Request struct {
	State     json.RawMessage
	Model     string
	Questions map[string]Question
	Fields    map[string]json.RawMessage
}

// Question is a validated question. Criteria and Instructions remain raw JSON
// because their permitted values intentionally span several JSON kinds.
type Question struct {
	Type            string
	Criteria        json.RawMessage
	HasCriteria     bool
	Instructions    json.RawMessage
	HasInstructions bool
	Fields          map[string]json.RawMessage
}

// Detail is the one deterministic validation detail returned by jev/v1.
type Detail struct {
	Loc  []string `json:"loc"`
	Msg  string   `json:"msg"`
	Type string   `json:"type"`
}

// ErrorEnvelope is the wire body for a validation failure.
type ErrorEnvelope struct {
	Detail []Detail `json:"detail"`
}

// ValidationError carries the exact wire envelope and HTTP status for a
// request validation failure.
type ValidationError struct {
	Envelope ErrorEnvelope
}

func (e *ValidationError) Error() string {
	if e == nil || len(e.Envelope.Detail) == 0 {
		return "request validation failed"
	}
	return e.Envelope.Detail[0].Msg
}

// StatusCode is the status required for every validation failure.
func (e *ValidationError) StatusCode() int { return 422 }

// MarshalJSON makes ValidationError directly serializable as its wire body.
func (e *ValidationError) MarshalJSON() ([]byte, error) {
	if e == nil {
		return json.Marshal(ErrorEnvelope{})
	}
	return json.Marshal(e.Envelope)
}

func missingError(field string) *ValidationError {
	return validationError([]string{"body", field}, "Field required", "missing")
}

func valueError(loc ...string) *ValidationError {
	return validationError(loc, "Invalid value", "value_error")
}

func unsupportedTypeError(loc ...string) *ValidationError {
	return validationError(loc, "Unsupported question type", "literal_error")
}

func invalidJSONError() *ValidationError {
	return validationError([]string{"body"}, "Invalid JSON", "json_invalid")
}

func validationError(loc []string, msg, typ string) *ValidationError {
	return &ValidationError{Envelope: ErrorEnvelope{Detail: []Detail{{
		Loc: append([]string(nil), loc...), Msg: msg, Type: typ,
	}}}}
}
