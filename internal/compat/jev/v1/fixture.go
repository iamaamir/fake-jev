package v1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"sort"
)

const probabilityTolerance = 1e-6

// InvalidStubResponseError identifies a fixture which cannot answer a validated
// request. It is deliberately separate from request validation errors: the
// request is valid, but the selected fixture is not a legal v1 response.
type InvalidStubResponseError struct {
	Reason string
}

func (e *InvalidStubResponseError) Error() string {
	if e == nil || e.Reason == "" {
		return "fake_jev_invalid_stub_response"
	}
	return "fake_jev_invalid_stub_response: " + e.Reason
}

func invalidStubResponse(format string, args ...any) error {
	return &InvalidStubResponseError{Reason: fmt.Sprintf(format, args...)}
}

// GenerateAnswers turns a v1 then.answers document into the wire answer map
// for request. It is intentionally strict: the configured answer names must
// exactly equal request question names, and every helper is checked against the
// corresponding validated question.
func GenerateAnswers(request Request, configured json.RawMessage) (map[string]json.RawMessage, error) {
	answers, ok := objectFields(configured)
	if !ok {
		return nil, invalidStubResponse("answers must be an object")
	}
	if len(answers) != len(request.Questions) {
		return nil, invalidStubResponse("answers must exactly cover request questions")
	}
	questionNames := sortedKeys(request.Questions)
	for _, name := range questionNames {
		if _, exists := answers[name]; !exists {
			return nil, invalidStubResponse("missing answer for question %q", name)
		}
	}
	answerNames := sortedKeys(answers)
	for _, name := range answerNames {
		if _, exists := request.Questions[name]; !exists {
			return nil, invalidStubResponse("answer provided for absent question %q", name)
		}
	}

	result := make(map[string]json.RawMessage, len(answers))
	for _, name := range questionNames {
		question := request.Questions[name]
		answer, err := generateAnswer(question, answers[name])
		if err != nil {
			return nil, invalidStubResponse("question %q: %v", name, err)
		}
		result[name] = answer
	}
	return result, nil
}

func generateAnswer(question Question, configured json.RawMessage) (json.RawMessage, error) {
	fields, ok := objectFields(configured)
	if !ok {
		return nil, fmt.Errorf("helper must be an object")
	}

	switch question.Type {
	case "noul":
		return generateNoulAnswer(fields)
	case "choice":
		return generateChoiceAnswer(question, fields)
	case "score":
		return generateScoreAnswer(question, fields)
	default:
		return nil, fmt.Errorf("unsupported request question type %q", question.Type)
	}
}

func generateNoulAnswer(fields map[string]json.RawMessage) (json.RawMessage, error) {
	value, ok := fields["noul"]
	if !ok {
		return nil, fmt.Errorf("noul helper is required")
	}
	if len(fields) != 1 {
		return nil, fmt.Errorf("noul helper has unknown fields")
	}

	var noul float64
	switch {
	case bytes.Equal(bytes.TrimSpace(value), []byte("true")):
		noul = 1
	case bytes.Equal(bytes.TrimSpace(value), []byte("false")):
		noul = 0
	default:
		if !jsonNumber(value) || json.Unmarshal(value, &noul) != nil || !finiteProbability(noul) {
			return nil, fmt.Errorf("noul must be a finite number in [0,1] or a boolean")
		}
	}
	return json.Marshal(struct {
		Type string  `json:"type"`
		Noul float64 `json:"noul"`
	}{Type: "noul", Noul: noul})
}

func generateChoiceAnswer(question Question, fields map[string]json.RawMessage) (json.RawMessage, error) {
	if err := validateHelperFields(fields, "choice"); err != nil {
		return nil, err
	}
	choiceRaw := fields["choice"]
	var choice string
	if err := json.Unmarshal(choiceRaw, &choice); err != nil {
		return nil, fmt.Errorf("choice must be a string")
	}
	criteria, ok := objectFields(question.Criteria)
	if !question.HasCriteria || !ok || len(criteria) == 0 {
		return nil, fmt.Errorf("request choice criteria must be a non-empty object")
	}
	if _, exists := criteria[choice]; !exists {
		return nil, fmt.Errorf("selected choice %q is not a request criterion", choice)
	}

	probabilities, explicit, err := probabilitiesField(fields)
	if err != nil {
		return nil, err
	}
	if !explicit {
		probabilities = make(map[string]float64, len(criteria))
		for key := range criteria {
			if key == choice {
				probabilities[key] = 1
			} else {
				probabilities[key] = 0
			}
		}
	}
	if explicit {
		if err := validateProbabilityMap(probabilities, criteriaKeys(criteria)); err != nil {
			return nil, err
		}
		selected := probabilities[choice]
		for _, probability := range probabilities {
			if probability > selected {
				return nil, fmt.Errorf("selected choice %q is not a maximum probability", choice)
			}
		}
	}

	confidence, err := confidenceField(fields)
	if err != nil {
		return nil, err
	}
	if confidence == nil {
		if explicit {
			value := probabilities[choice]
			confidence = &value
		} else {
			value := float64(1)
			confidence = &value
		}
	}
	return json.Marshal(struct {
		Type          string             `json:"type"`
		Choice        string             `json:"choice"`
		Confidence    float64            `json:"confidence"`
		Probabilities map[string]float64 `json:"probabilities"`
	}{Type: "choice", Choice: choice, Confidence: *confidence, Probabilities: probabilities})
}

func generateScoreAnswer(question Question, fields map[string]json.RawMessage) (json.RawMessage, error) {
	if err := validateHelperFields(fields, "score"); err != nil {
		return nil, err
	}
	scoreRaw := fields["score"]
	var score float64
	if !jsonNumber(scoreRaw) || json.Unmarshal(scoreRaw, &score) != nil || !finiteNumber(score) {
		return nil, fmt.Errorf("score must be a finite number")
	}

	var criteria []json.RawMessage
	if !question.HasCriteria || json.Unmarshal(question.Criteria, &criteria) != nil || len(criteria) == 0 {
		return nil, fmt.Errorf("request score criteria must be a non-empty array")
	}
	maxScore := float64(len(criteria) - 1)
	if score < 0 || score > maxScore {
		return nil, fmt.Errorf("score must be within [0,%d]", len(criteria)-1)
	}

	keys := make([]string, len(criteria))
	legend := make(map[string]json.RawMessage, len(criteria))
	for index, criterion := range criteria {
		key := fmt.Sprint(index)
		keys[index] = key
		legend[key] = cloneRaw(criterion)
	}
	probabilities, explicit, err := probabilitiesField(fields)
	if err != nil {
		return nil, err
	}
	if !explicit {
		probabilities = make(map[string]float64, len(criteria))
		for _, key := range keys {
			probabilities[key] = 0
		}
		lo := math.Floor(score)
		hi := math.Ceil(score)
		loKey := fmt.Sprint(int(lo))
		hiKey := fmt.Sprint(int(hi))
		if lo == hi {
			probabilities[loKey] = 1
		} else {
			probabilities[loKey] = hi - score
			probabilities[hiKey] = score - lo
		}
	}
	if explicit {
		if err := validateProbabilityMap(probabilities, keys); err != nil {
			return nil, err
		}
		var expected float64
		for _, key := range keys {
			probability := probabilities[key]
			var level int
			if _, err := fmt.Sscanf(key, "%d", &level); err != nil {
				return nil, fmt.Errorf("invalid score probability key %q", key)
			}
			expected += float64(level) * probability
		}
		if math.Abs(expected-score) > probabilityTolerance {
			return nil, fmt.Errorf("probability expected value does not equal score")
		}
	}

	confidence, err := confidenceField(fields)
	if err != nil {
		return nil, err
	}
	if confidence == nil {
		value := float64(1)
		confidence = &value
	}
	return json.Marshal(struct {
		Type          string                     `json:"type"`
		Score         float64                    `json:"score"`
		Confidence    float64                    `json:"confidence"`
		Probabilities map[string]float64         `json:"probabilities"`
		Legend        map[string]json.RawMessage `json:"legend"`
	}{Type: "score", Score: score, Confidence: *confidence, Probabilities: probabilities, Legend: legend})
}

func validateHelperFields(fields map[string]json.RawMessage, helper string) error {
	if _, ok := fields[helper]; !ok {
		return fmt.Errorf("%s helper is required", helper)
	}
	for key := range fields {
		if key != helper && key != "confidence" && key != "probabilities" {
			return fmt.Errorf("%s helper has unknown field %q", helper, key)
		}
	}
	return nil
}

func probabilitiesField(fields map[string]json.RawMessage) (map[string]float64, bool, error) {
	raw, exists := fields["probabilities"]
	if !exists {
		return nil, false, nil
	}
	probabilitiesRaw, ok := objectFields(raw)
	if !ok {
		return nil, false, fmt.Errorf("probabilities must be an object")
	}
	probabilities := make(map[string]float64, len(probabilitiesRaw))
	for _, key := range sortedKeys(probabilitiesRaw) {
		value := probabilitiesRaw[key]
		var probability float64
		if !jsonNumber(value) || json.Unmarshal(value, &probability) != nil || !finiteProbability(probability) {
			return nil, false, fmt.Errorf("probability %q must be a finite number in [0,1]", key)
		}
		probabilities[key] = probability
	}
	return probabilities, true, nil
}

func confidenceField(fields map[string]json.RawMessage) (*float64, error) {
	raw, exists := fields["confidence"]
	if !exists {
		return nil, nil
	}
	var confidence float64
	if !jsonNumber(raw) || json.Unmarshal(raw, &confidence) != nil || !finiteProbability(confidence) {
		return nil, fmt.Errorf("confidence must be a finite number in [0,1]")
	}
	return &confidence, nil
}

func validateProbabilityMap(probabilities map[string]float64, expectedKeys []string) error {
	if len(probabilities) != len(expectedKeys) {
		return fmt.Errorf("probabilities must exactly cover request criteria")
	}
	expected := make(map[string]struct{}, len(expectedKeys))
	for _, key := range expectedKeys {
		expected[key] = struct{}{}
	}
	sum := float64(0)
	for _, key := range sortedKeys(probabilities) {
		probability := probabilities[key]
		if _, ok := expected[key]; !ok {
			return fmt.Errorf("probabilities contain unknown key %q", key)
		}
		if !finiteProbability(probability) {
			return fmt.Errorf("probability %q must be a finite number in [0,1]", key)
		}
		sum += probability
	}
	for _, key := range expectedKeys {
		if _, ok := probabilities[key]; !ok {
			return fmt.Errorf("probabilities are missing key %q", key)
		}
	}
	if math.Abs(sum-1) > probabilityTolerance {
		return fmt.Errorf("probabilities must sum to 1")
	}
	return nil
}

func finiteProbability(value float64) bool {
	return finiteNumber(value) && value >= 0 && value <= 1
}

func finiteNumber(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

func jsonNumber(raw json.RawMessage) bool {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || (trimmed[0] != '-' && (trimmed[0] < '0' || trimmed[0] > '9')) {
		return false
	}
	return true
}

func sortedKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func criteriaKeys(criteria map[string]json.RawMessage) []string {
	return sortedKeys(criteria)
}
