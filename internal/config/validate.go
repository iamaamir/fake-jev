package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
)

var releaseDatePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

var defaultModel = ModelConfig{
	Name:        "jev-latest",
	Description: "Local deterministic fake model provided by fake-jev.",
	ReleaseDate: "1970-01-01",
}

func applyDefaults(cfg *Config, document []byte) error {
	root, err := object(document, "configuration")
	if err != nil {
		return err
	}
	if _, ok := root["server"]; !ok {
		cfg.Server = ServerConfig{Host: DefaultHost, Port: DefaultPort}
	} else {
		server, err := object(root["server"], "server")
		if err != nil {
			return err
		}
		if err := rejectNullFields(server, "server", "host", "port"); err != nil {
			return err
		}
		if _, ok := server["host"]; !ok {
			cfg.Server.Host = DefaultHost
		}
		if _, ok := server["port"]; !ok {
			cfg.Server.Port = DefaultPort
		}
	}
	if _, ok := root["mode"]; !ok {
		cfg.Mode = DefaultMode
	} else if err := rejectNullFields(root, "configuration", "mode"); err != nil {
		return err
	}
	if _, ok := root["compatibility"]; !ok {
		cfg.Compatibility = []string{DefaultProfile}
	}
	if _, ok := root["limits"]; !ok {
		cfg.Limits = LimitsConfig{
			DataPlaneBodyBytes:      8388608,
			ControlPlaneBodyBytes:   2097152,
			MaxInteractions:         10000,
			LogBodyBytes:            4096,
			GracefulShutdownSeconds: 5,
		}
	} else {
		limits, err := object(root["limits"], "limits")
		if err != nil {
			return err
		}
		if err := rejectNullFields(limits, "limits", "dataPlaneBodyBytes", "controlPlaneBodyBytes", "maxInteractions", "logBodyBytes", "gracefulShutdownSeconds"); err != nil {
			return err
		}
		if _, ok := limits["dataPlaneBodyBytes"]; !ok {
			cfg.Limits.DataPlaneBodyBytes = 8388608
		}
		if _, ok := limits["controlPlaneBodyBytes"]; !ok {
			cfg.Limits.ControlPlaneBodyBytes = 2097152
		}
		if _, ok := limits["maxInteractions"]; !ok {
			cfg.Limits.MaxInteractions = 10000
		}
		if _, ok := limits["logBodyBytes"]; !ok {
			cfg.Limits.LogBodyBytes = 4096
		}
		if _, ok := limits["gracefulShutdownSeconds"]; !ok {
			cfg.Limits.GracefulShutdownSeconds = 5
		}
	}
	if _, ok := root["models"]; !ok {
		cfg.Models = []ModelConfig{defaultModel}
	}
	if _, ok := root["stubs"]; !ok {
		cfg.Stubs = []StubConfig{}
	}
	return nil
}

// Validate checks a decoded configuration model. Load should be preferred for
// external input because it also enforces required fields and strict decoding.
func Validate(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("configuration is nil")
	}
	if cfg.SchemaVersion != 1 {
		return fmt.Errorf("schemaVersion must equal 1")
	}
	if err := validateValues(cfg); err != nil {
		return err
	}
	return nil
}

func validateDocument(document []byte, cfg *Config) error {
	root, err := object(document, "configuration")
	if err != nil {
		return err
	}
	if err := rejectUnexpectedFields(root, "configuration", "schemaVersion", "server", "mode", "compatibility", "limits", "models", "stubs"); err != nil {
		return err
	}
	version, ok := root["schemaVersion"]
	if !ok {
		return fmt.Errorf("schemaVersion is required")
	}
	var schemaVersion int
	if err := json.Unmarshal(version, &schemaVersion); err != nil || schemaVersion != 1 {
		return fmt.Errorf("schemaVersion must equal integer 1")
	}

	for _, section := range []struct {
		name   string
		fields []string
	}{
		{name: "server", fields: []string{"host", "port"}},
		{name: "limits", fields: []string{"dataPlaneBodyBytes", "controlPlaneBodyBytes", "maxInteractions", "logBodyBytes", "gracefulShutdownSeconds"}},
	} {
		value, exists := root[section.name]
		if !exists {
			continue
		}
		sectionObject, err := object(value, section.name)
		if err != nil {
			return err
		}
		if err := rejectUnexpectedFields(sectionObject, section.name, section.fields...); err != nil {
			return err
		}
	}
	if value, exists := root["compatibility"]; exists {
		if string(value) == "null" {
			return fmt.Errorf("compatibility must be a non-empty array")
		}
	}
	if value, exists := root["models"]; exists && string(value) == "null" {
		return fmt.Errorf("models must be an array")
	}
	if value, exists := root["models"]; exists {
		var models []json.RawMessage
		if err := json.Unmarshal(value, &models); err != nil {
			return fmt.Errorf("models must be an array")
		}
		for i, rawModel := range models {
			modelName := fmt.Sprintf("models[%d]", i)
			model, err := object(rawModel, modelName)
			if err != nil {
				return err
			}
			modelFields := []string{"name", "description", "release_date"}
			if err := rejectUnexpectedFields(model, modelName, modelFields...); err != nil {
				return err
			}
			if err := rejectNullFields(model, modelName, modelFields...); err != nil {
				return err
			}
			for _, field := range modelFields {
				if _, exists := model[field]; !exists {
					return fmt.Errorf("%s.%s is required", modelName, field)
				}
			}
		}
	}
	if value, exists := root["stubs"]; exists && string(value) == "null" {
		return fmt.Errorf("stubs must be an array")
	}
	if value, exists := root["stubs"]; exists {
		var stubs []json.RawMessage
		if err := json.Unmarshal(value, &stubs); err != nil {
			return fmt.Errorf("stubs must be an array")
		}
		for i, rawStub := range stubs {
			stubName := fmt.Sprintf("stubs[%d]", i)
			stub, err := object(rawStub, stubName)
			if err != nil {
				return err
			}
			stubFields := []string{"id", "profile", "priority", "when", "then", "expect"}
			if err := rejectUnexpectedFields(stub, stubName, stubFields...); err != nil {
				return err
			}
			if err := rejectNullFields(stub, stubName, stubFields...); err != nil {
				return err
			}
			when, exists := stub["when"]
			if !exists {
				return fmt.Errorf("stubs[%d].when is required", i)
			}
			whenObject, err := object(when, fmt.Sprintf("stubs[%d].when", i))
			if err != nil {
				return err
			}
			if err := validateWhenDocument(whenObject, i); err != nil {
				return err
			}
			then, exists := stub["then"]
			if !exists {
				return fmt.Errorf("stubs[%d].then is required", i)
			}
			thenObject, err := object(then, fmt.Sprintf("stubs[%d].then", i))
			if err != nil {
				return err
			}
			if err := validateThenDocument(thenObject, i); err != nil {
				return err
			}
			if expect, exists := stub["expect"]; exists {
				expectName := fmt.Sprintf("stubs[%d].expect", i)
				expectObject, err := object(expect, expectName)
				if err != nil {
					return err
				}
				expectFields := []string{"exactly", "atLeast", "atMost"}
				if err := rejectUnexpectedFields(expectObject, expectName, expectFields...); err != nil {
					return err
				}
				if err := rejectNullFields(expectObject, expectName, expectFields...); err != nil {
					return err
				}
			}
		}
	}
	return validateValues(cfg)
}

func validateValues(cfg *Config) error {
	if cfg.Mode != "strict" {
		return fmt.Errorf("mode %q is not supported; only strict is valid", cfg.Mode)
	}
	if len(cfg.Compatibility) == 0 {
		return fmt.Errorf("compatibility must not be empty")
	}
	enabled := make(map[string]struct{}, len(cfg.Compatibility))
	for i, profile := range cfg.Compatibility {
		normalized, err := normalizeProfile(profile)
		if err != nil {
			return err
		}
		if _, exists := enabled[normalized]; exists {
			return fmt.Errorf("duplicate compatibility profile %q", normalized)
		}
		cfg.Compatibility[i] = normalized
		enabled[normalized] = struct{}{}
	}
	if err := validateServer(cfg.Server); err != nil {
		return err
	}
	if err := validateLimits(cfg.Limits); err != nil {
		return err
	}
	if err := validateModels(cfg.Models); err != nil {
		return err
	}
	ids := make(map[string]struct{}, len(cfg.Stubs))
	for i := range cfg.Stubs {
		stub := &cfg.Stubs[i]
		if stub.ID == "" {
			return fmt.Errorf("stub %d id must be non-empty", i)
		}
		if _, exists := ids[stub.ID]; exists {
			return fmt.Errorf("duplicate static stub id %q", stub.ID)
		}
		ids[stub.ID] = struct{}{}
		profile, err := normalizeProfile(stub.Profile)
		if err != nil {
			return fmt.Errorf("stub %q: %w", stub.ID, err)
		}
		if _, exists := enabled[profile]; !exists {
			return fmt.Errorf("stub %q uses profile %q which is not enabled", stub.ID, profile)
		}
		stub.Profile = profile
		if stub.When.Operation == "" {
			stub.When.Operation = "systemone"
		}
		if err := validateWhen(stub.When); err != nil {
			return fmt.Errorf("stub %q when: %w", stub.ID, err)
		}
		if err := validateThen(stub.Then); err != nil {
			return fmt.Errorf("stub %q then: %w", stub.ID, err)
		}
		if err := validateExpect(stub.Expect); err != nil {
			return fmt.Errorf("stub %q expect: %w", stub.ID, err)
		}
	}
	return nil
}

func normalizeProfile(profile string) (string, error) {
	if profile == "jev" {
		return DefaultProfile, nil
	}
	if profile != DefaultProfile {
		return "", fmt.Errorf("unknown profile %q", profile)
	}
	return profile, nil
}

func validateServer(server ServerConfig) error {
	if server.Port < 0 || server.Port > 65535 {
		return fmt.Errorf("server.port must be between 0 and 65535")
	}
	return nil
}

func validateLimits(limits LimitsConfig) error {
	if limits.DataPlaneBodyBytes < 1024 {
		return fmt.Errorf("limits.dataPlaneBodyBytes must be at least 1024")
	}
	if limits.ControlPlaneBodyBytes < 1024 {
		return fmt.Errorf("limits.controlPlaneBodyBytes must be at least 1024")
	}
	if limits.MaxInteractions < 1 {
		return fmt.Errorf("limits.maxInteractions must be at least 1")
	}
	if limits.LogBodyBytes < 0 {
		return fmt.Errorf("limits.logBodyBytes must be non-negative")
	}
	if limits.GracefulShutdownSeconds < 1 || limits.GracefulShutdownSeconds > 60 {
		return fmt.Errorf("limits.gracefulShutdownSeconds must be between 1 and 60")
	}
	return nil
}

func validateModels(models []ModelConfig) error {
	if len(models) == 0 {
		return fmt.Errorf("models must contain at least one model")
	}
	names := make(map[string]struct{}, len(models))
	for i, model := range models {
		if model.Name == "" {
			return fmt.Errorf("models[%d].name must be non-empty", i)
		}
		if _, exists := names[model.Name]; exists {
			return fmt.Errorf("duplicate model name %q", model.Name)
		}
		names[model.Name] = struct{}{}
		if !releaseDatePattern.MatchString(model.ReleaseDate) {
			return fmt.Errorf("models[%d].release_date must match YYYY-MM-DD", i)
		}
	}
	return nil
}

func validateWhenDocument(when map[string]json.RawMessage, index int) error {
	name := fmt.Sprintf("stubs[%d].when", index)
	whenFields := []string{"operation", "model", "state", "questions"}
	if err := rejectUnexpectedFields(when, name, whenFields...); err != nil {
		return err
	}
	if err := rejectNullFields(when, name, whenFields...); err != nil {
		return err
	}
	if questions, exists := when["questions"]; exists {
		questionObject, err := object(questions, name+".questions")
		if err != nil {
			return err
		}
		for _, question := range sortedKeys(questionObject) {
			if err := rejectNullFields(map[string]json.RawMessage{"value": questionObject[question]}, name+".questions."+question, "value"); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateThenDocument(then map[string]json.RawMessage, index int) error {
	name := fmt.Sprintf("stubs[%d].then", index)
	thenFields := []string{"answers", "sequence", "raw", "model", "usage"}
	if err := rejectUnexpectedFields(then, name, thenFields...); err != nil {
		return err
	}
	if err := rejectNullFields(then, name, thenFields...); err != nil {
		return err
	}
	if raw, exists := then["raw"]; exists {
		rawObject, err := object(raw, name+".raw")
		if err != nil {
			return err
		}
		if err := rejectUnexpectedFields(rawObject, name+".raw", "status", "headers", "body"); err != nil {
			return err
		}
		if err := rejectNullFields(rawObject, name+".raw", "status", "headers"); err != nil {
			return err
		}
	}
	if usage, exists := then["usage"]; exists {
		if err := validateUsageDocument(usage, name+".usage"); err != nil {
			return err
		}
	}
	if sequence, exists := then["sequence"]; exists {
		var responses []json.RawMessage
		if err := json.Unmarshal(sequence, &responses); err != nil {
			return fmt.Errorf("%s.sequence must be an array", name)
		}
		for i, response := range responses {
			if err := validateResponseDocument(response, fmt.Sprintf("%s.sequence[%d]", name, i)); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateResponseDocument(raw json.RawMessage, name string) error {
	response, err := object(raw, name)
	if err != nil {
		return err
	}
	responseFields := []string{"answers", "raw", "model", "usage"}
	if err := rejectUnexpectedFields(response, name, responseFields...); err != nil {
		return err
	}
	if err := rejectNullFields(response, name, responseFields...); err != nil {
		return err
	}
	if rawResponse, exists := response["raw"]; exists {
		rawObject, err := object(rawResponse, name+".raw")
		if err != nil {
			return err
		}
		if err := rejectUnexpectedFields(rawObject, name+".raw", "status", "headers", "body"); err != nil {
			return err
		}
		if err := rejectNullFields(rawObject, name+".raw", "status", "headers"); err != nil {
			return err
		}
	}
	if usage, exists := response["usage"]; exists {
		if err := validateUsageDocument(usage, name+".usage"); err != nil {
			return err
		}
	}
	return nil
}

func validateUsageDocument(raw json.RawMessage, name string) error {
	usage, err := object(raw, name)
	if err != nil {
		return err
	}
	usageFields := []string{"input_tokens", "output_tokens"}
	if err := rejectUnexpectedFields(usage, name, usageFields...); err != nil {
		return err
	}
	for _, field := range usageFields {
		if _, exists := usage[field]; !exists {
			return fmt.Errorf("%s.%s is required", name, field)
		}
	}
	if err := rejectNullFields(usage, name, usageFields...); err != nil {
		return err
	}
	return nil
}

func validateWhen(when WhenConfig) error {
	if when.Operation == "" {
		when.Operation = "systemone"
	}
	if when.Operation != "systemone" {
		return fmt.Errorf("operation %q is invalid", when.Operation)
	}
	if len(when.State) > 0 {
		trimmed := bytes.TrimSpace(when.State)
		if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) || (trimmed[0] != '"' && trimmed[0] != '{' && trimmed[0] != '[') {
			return fmt.Errorf("state must be a string, object, or array")
		}
	}
	if when.Questions != nil {
		if len(when.Questions) == 0 {
			return fmt.Errorf("questions must not be empty")
		}
		for _, name := range sortedKeys(when.Questions) {
			kind := when.Questions[name]
			if name == "" {
				return fmt.Errorf("question names must be non-empty")
			}
			switch kind {
			case "noul", "choice", "score":
			default:
				return fmt.Errorf("question %q has invalid type %q", name, kind)
			}
		}
	}
	return nil
}

func validateThen(then ThenConfig) error {
	forms := 0
	if len(then.Answers) > 0 {
		forms++
		if err := requireJSONObject(then.Answers, "answers"); err != nil {
			return err
		}
	}
	if then.Sequence != nil {
		forms++
		if len(then.Sequence) == 0 {
			return fmt.Errorf("sequence must contain at least one response")
		}
		for i, response := range then.Sequence {
			if err := validateResponse(response); err != nil {
				return fmt.Errorf("sequence[%d]: %w", i, err)
			}
		}
	}
	if then.Raw != nil {
		forms++
		if then.Model != "" || then.Usage != nil {
			return fmt.Errorf("raw cannot have model or usage siblings")
		}
		if err := validateRaw(*then.Raw); err != nil {
			return err
		}
	}
	if forms != 1 {
		return fmt.Errorf("exactly one of answers, sequence, or raw is required")
	}
	if then.Usage != nil {
		if err := validateUsage(*then.Usage); err != nil {
			return err
		}
	}
	return nil
}

func validateResponse(response ResponseConfig) error {
	forms := 0
	if len(response.Answers) > 0 {
		forms++
		if err := requireJSONObject(response.Answers, "answers"); err != nil {
			return err
		}
	}
	if response.Raw != nil {
		forms++
		if response.Model != "" || response.Usage != nil {
			return fmt.Errorf("raw cannot have model or usage siblings")
		}
		if err := validateRaw(*response.Raw); err != nil {
			return err
		}
	}
	if forms != 1 {
		return fmt.Errorf("exactly one of answers or raw is required")
	}
	if response.Usage != nil {
		if err := validateUsage(*response.Usage); err != nil {
			return err
		}
	}
	return nil
}

func validateRaw(raw RawResponse) error {
	if raw.Status < 100 || raw.Status > 599 {
		return fmt.Errorf("raw.status must be between 100 and 599")
	}
	for name := range raw.Headers {
		if name == "" {
			return fmt.Errorf("raw header names must be non-empty")
		}
	}
	return nil
}

func validateUsage(usage UsageConfig) error {
	if usage.InputTokens < 0 || usage.OutputTokens < 0 {
		return fmt.Errorf("usage token counts must be non-negative")
	}
	return nil
}

func validateExpect(expect *ExpectConfig) error {
	if expect == nil {
		return nil
	}
	forms := 0
	if expect.Exactly != nil {
		forms++
	}
	if expect.AtLeast != nil || expect.AtMost != nil {
		forms++
	}
	if forms != 1 {
		return fmt.Errorf("exactly one of exactly or a range is required")
	}
	if expect.Exactly != nil && *expect.Exactly < 0 {
		return fmt.Errorf("exactly must be non-negative")
	}
	if expect.AtLeast != nil && *expect.AtLeast < 0 {
		return fmt.Errorf("atLeast must be non-negative")
	}
	if expect.AtMost != nil && *expect.AtMost < 0 {
		return fmt.Errorf("atMost must be non-negative")
	}
	if expect.AtLeast != nil && expect.AtMost != nil && *expect.AtLeast > *expect.AtMost {
		return fmt.Errorf("atLeast must be less than or equal to atMost")
	}
	return nil
}

// rejectUnexpectedFields enforces the exact-key half of specification §12.3 on
// one project-owned schema object: a key that is not spelled exactly like a
// known field is an unknown key and fails validation. encoding/json matches
// object keys to struct fields case-insensitively, so DisallowUnknownFields
// alone accepts a key that differs from a field name only in case, writes the
// field with it, and leaves the decoded configuration disagreeing with the raw
// document that validateDocument reads. The walk stops at the intentionally open
// containers of §38.8 (raw bodies, state, answers, question names, header names),
// so their content is never compared against project-owned field names.
func rejectUnexpectedFields(value map[string]json.RawMessage, name string, fields ...string) error {
	known := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		known[field] = struct{}{}
	}
	for _, key := range sortedKeys(value) {
		if _, isKnown := known[key]; !isKnown {
			return fmt.Errorf("%s.%s is not a known field", name, key)
		}
	}
	return nil
}

func rejectNullFields(object map[string]json.RawMessage, name string, fields ...string) error {
	for _, field := range fields {
		raw, exists := object[field]
		if exists && bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return fmt.Errorf("%s.%s must not be null", name, field)
		}
	}
	return nil
}

func object(raw []byte, name string) (map[string]json.RawMessage, error) {
	var value map[string]json.RawMessage
	if err := json.Unmarshal(raw, &value); err != nil || value == nil {
		return nil, fmt.Errorf("%s must be an object", name)
	}
	return value, nil
}

func requireJSONObject(raw json.RawMessage, name string) error {
	if _, err := object(raw, name); err != nil {
		return err
	}
	return nil
}
