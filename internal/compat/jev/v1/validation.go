package v1

import (
	"bytes"
	"encoding/json"
	"io"
	"sort"
)

// ValidateRequest decodes and validates a POST /v1/systemone body. It returns
// exactly the first frozen-schema error, or a populated Request on success.
func ValidateRequest(body []byte) (Request, *ValidationError) {
	fields, validJSON, object := decodeObject(body)
	if !validJSON {
		return Request{}, invalidJSONError()
	}
	if !object {
		return Request{}, valueError("body")
	}

	state, exists := fields["state"]
	if !exists {
		return Request{}, missingError("state")
	}
	if !validState(state) {
		return Request{}, valueError("body", "state")
	}

	modelRaw, exists := fields["model"]
	if !exists {
		return Request{}, missingError("model")
	}
	var model string
	if err := json.Unmarshal(modelRaw, &model); err != nil || !isString(modelRaw) {
		return Request{}, valueError("body", "model")
	}

	questionsRaw, exists := fields["questions"]
	if !exists {
		return Request{}, missingError("questions")
	}
	questionsFields, ok := objectFields(questionsRaw)
	if !ok || len(questionsFields) == 0 {
		return Request{}, valueError("body", "questions")
	}

	questions := make(map[string]Question, len(questionsFields))
	names := make([]string, 0, len(questionsFields))
	for name := range questionsFields {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		question, err := validateQuestion(name, questionsFields[name])
		if err != nil {
			return Request{}, err
		}
		questions[name] = question
	}

	return Request{
		State:     cloneRaw(state),
		Model:     model,
		Questions: questions,
		Fields:    cloneFields(fields),
	}, nil
}

// ParseRequest is the parsing-oriented spelling of ValidateRequest.
func ParseRequest(body []byte) (Request, *ValidationError) {
	return ValidateRequest(body)
}

// DecodeRequest is retained as an explicit wire-decoding spelling.
func DecodeRequest(body []byte) (Request, *ValidationError) {
	return ValidateRequest(body)
}

func validateQuestion(name string, raw json.RawMessage) (Question, *ValidationError) {
	loc := []string{"body", "questions", name}
	fields, ok := objectFields(raw)
	if !ok {
		return Question{}, valueError(loc...)
	}

	typeRaw, exists := fields["type"]
	if !exists || !isString(typeRaw) {
		return Question{}, valueError(append(loc, "type")...)
	}
	var questionType string
	if err := json.Unmarshal(typeRaw, &questionType); err != nil {
		return Question{}, valueError(append(loc, "type")...)
	}
	switch questionType {
	case "noul", "choice", "score":
	default:
		return Question{}, unsupportedTypeError(append(loc, "type")...)
	}

	question := Question{Type: questionType, Fields: cloneFields(fields)}
	if instructions, exists := fields["instructions"]; exists {
		if !validFlexibleValue(instructions) {
			return Question{}, valueError(append(loc, "instructions")...)
		}
		question.HasInstructions = true
		question.Instructions = cloneRaw(instructions)
	}

	criteria, hasCriteria := fields["criteria"]
	if hasCriteria {
		question.HasCriteria = true
		question.Criteria = cloneRaw(criteria)
	}
	switch questionType {
	case "noul":
		if hasCriteria && !validNoulCriteria(criteria) {
			return Question{}, valueError(append(loc, "criteria")...)
		}
	case "choice":
		criteriaFields, valid := objectFields(criteria)
		if !hasCriteria || !valid || len(criteriaFields) == 0 || !validCriteriaObject(criteriaFields) {
			return Question{}, valueError(append(loc, "criteria")...)
		}
	case "score":
		var values []json.RawMessage
		if !hasCriteria || !isArray(criteria) || json.Unmarshal(criteria, &values) != nil || len(values) == 0 {
			return Question{}, valueError(append(loc, "criteria")...)
		}
		for _, value := range values {
			if !validScoreCriteriaValue(value) {
				return Question{}, valueError(append(loc, "criteria")...)
			}
		}
	}
	return question, nil
}

func decodeObject(body []byte) (map[string]json.RawMessage, bool, bool) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	var value json.RawMessage
	if err := decoder.Decode(&value); err != nil {
		return nil, false, false
	}
	// Keep a second value raw: decoding it into any would recursively build
	// maps and slices for an input that is rejected regardless.
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, false, false
	}
	fields, object := objectFields(value)
	return fields, true, object
}

func objectFields(raw json.RawMessage) (map[string]json.RawMessage, bool) {
	if !isObject(raw) {
		return nil, false
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil || fields == nil {
		return nil, false
	}
	return fields, true
}

func validState(raw json.RawMessage) bool {
	return isString(raw) || isObject(raw) || isArray(raw)
}

func validNoulCriteria(raw json.RawMessage) bool {
	if isNull(raw) {
		return true
	}
	fields, ok := objectFields(raw)
	return ok && validCriteriaObject(fields)
}

func validCriteriaObject(fields map[string]json.RawMessage) bool {
	for _, value := range fields {
		if !validFlexibleValue(value) {
			return false
		}
	}
	return true
}

func validFlexibleValue(raw json.RawMessage) bool {
	return isNull(raw) || isString(raw) || isObject(raw) || isArray(raw)
}

func validScoreCriteriaValue(raw json.RawMessage) bool {
	return isString(raw) || isObject(raw) || isArray(raw)
}

func isString(raw json.RawMessage) bool {
	return len(bytes.TrimSpace(raw)) > 0 && bytes.TrimSpace(raw)[0] == '"'
}

func isObject(raw json.RawMessage) bool {
	trimmed := bytes.TrimSpace(raw)
	return len(trimmed) > 0 && trimmed[0] == '{'
}

func isArray(raw json.RawMessage) bool {
	trimmed := bytes.TrimSpace(raw)
	return len(trimmed) > 0 && trimmed[0] == '['
}

func isNull(raw json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(raw), []byte("null"))
}

func cloneRaw(raw json.RawMessage) json.RawMessage {
	return append(json.RawMessage(nil), raw...)
}

func cloneFields(fields map[string]json.RawMessage) map[string]json.RawMessage {
	clone := make(map[string]json.RawMessage, len(fields))
	for key, value := range fields {
		clone[key] = cloneRaw(value)
	}
	return clone
}
