package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Load decodes a YAML or JSON configuration and applies the specification's
// built-in defaults before validating it.
func Load(data []byte) (*Config, error) {
	jsonData, err := toJSON(data)
	if err != nil {
		return nil, err
	}

	var cfg Config
	decoder := json.NewDecoder(bytes.NewReader(jsonData))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode configuration: %w", err)
	}
	if err := requireEOF(decoder); err != nil {
		return nil, fmt.Errorf("decode configuration: %w", err)
	}
	if err := applyDefaults(&cfg, jsonData); err != nil {
		return nil, err
	}
	if err := validateDocument(jsonData, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// LoadConfig is the named form retained for callers that prefer an explicit
// function name at configuration boundaries.
func LoadConfig(data []byte) (*Config, error) { return Load(data) }

// LoadFile loads a configuration from path. The parser accepts either YAML or
// JSON based on document content, so file extensions do not affect behavior.
func LoadFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read configuration %q: %w", path, err)
	}
	return Load(data)
}

func requireEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON documents")
		}
		return err
	}
	return nil
}

func toJSON(data []byte) ([]byte, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, errors.New("configuration is empty")
	}
	if trimmed[0] == '{' || trimmed[0] == '[' {
		if err := rejectDuplicateJSON(trimmed); err == nil {
			return trimmed, nil
		}
		// YAML flow mappings and sequences may begin with the same delimiters as
		// JSON. Fall through so valid YAML is not rejected merely because it uses
		// flow syntax; decodeYAML still rejects trailing and multiple documents.
	}
	value, err := decodeYAML(trimmed)
	if err != nil {
		return nil, fmt.Errorf("decode YAML configuration: %w", err)
	}
	value, err = normalizeYAMLValue(value)
	if err != nil {
		return nil, fmt.Errorf("convert YAML configuration: %w", err)
	}
	result, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("convert YAML configuration to JSON: %w", err)
	}
	return result, nil
}

func decodeYAML(data []byte) (any, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	var extra any
	err := decoder.Decode(&extra)
	if err == nil {
		return nil, errors.New("multiple YAML documents")
	}
	if !errors.Is(err, io.EOF) {
		return nil, err
	}
	return value, nil
}

func rejectDuplicateJSON(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := scanJSONValue(decoder); err != nil {
		return fmt.Errorf("decode JSON configuration: %w", err)
	}
	if err := requireEOF(decoder); err != nil {
		return fmt.Errorf("decode JSON configuration: %w", err)
	}
	return nil
}

func scanJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delim, isDelim := token.(json.Delim)
	if !isDelim {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]struct{}{}
		for decoder.More() {
			key, err := decoder.Token()
			if err != nil {
				return err
			}
			name, ok := key.(string)
			if !ok {
				return fmt.Errorf("object key is not a string")
			}
			if _, exists := seen[name]; exists {
				return fmt.Errorf("duplicate object key %q", name)
			}
			seen[name] = struct{}{}
			if err := scanJSONValue(decoder); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil {
			return err
		}
		if end != json.Delim('}') {
			return fmt.Errorf("malformed object")
		}
	case '[':
		for decoder.More() {
			if err := scanJSONValue(decoder); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil {
			return err
		}
		if end != json.Delim(']') {
			return fmt.Errorf("malformed array")
		}
	default:
		return fmt.Errorf("unexpected delimiter %q", delim)
	}
	return nil
}

func normalizeYAMLValue(value any) (any, error) {
	switch value := value.(type) {
	case map[string]any:
		result := make(map[string]any, len(value))
		for key, item := range value {
			converted, err := normalizeYAMLValue(item)
			if err != nil {
				return nil, err
			}
			result[key] = converted
		}
		return result, nil
	case map[any]any:
		result := make(map[string]any, len(value))
		for key, item := range value {
			name, ok := key.(string)
			if !ok {
				return nil, fmt.Errorf("object key %v is not a string", key)
			}
			converted, err := normalizeYAMLValue(item)
			if err != nil {
				return nil, err
			}
			result[name] = converted
		}
		return result, nil
	case time.Time:
		return value.Format("2006-01-02"), nil
	case []any:
		result := make([]any, len(value))
		for i, item := range value {
			converted, err := normalizeYAMLValue(item)
			if err != nil {
				return nil, err
			}
			result[i] = converted
		}
		return result, nil
	default:
		return value, nil
	}
}
