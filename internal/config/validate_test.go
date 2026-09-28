package config

import (
	"reflect"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load([]byte("schemaVersion: 1\n"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Host != DefaultHost || cfg.Server.Port != DefaultPort || cfg.Mode != DefaultMode {
		t.Fatalf("defaults not applied: %#v", cfg)
	}
	if !reflect.DeepEqual(cfg.Compatibility, []string{DefaultProfile}) || len(cfg.Models) != 1 || len(cfg.Stubs) != 0 {
		t.Fatalf("defaults not applied: %#v", cfg)
	}
}

func TestRejectsMultipleYAMLDocuments(t *testing.T) {
	if _, err := Load([]byte("---\nschemaVersion: 1\n---\nschemaVersion: 1\n")); err == nil {
		t.Fatal("expected multiple YAML documents to be rejected")
	}
}

func TestAcceptsYAMLFlowDocument(t *testing.T) {
	if _, err := Load([]byte("{schemaVersion: 1}")); err != nil {
		t.Fatalf("valid YAML flow document rejected: %v", err)
	}
}

func TestJSONAndYAMLLoadSameModel(t *testing.T) {
	jsonConfig := []byte(`{"schemaVersion":1,"stubs":[{"id":"x","profile":"jev","when":{},"then":{"raw":{"status":200,"headers":{"content-type":"application/json"},"body":{"ok":true}}}}]}`)
	yamlConfig := []byte("schemaVersion: 1\nstubs:\n  - id: x\n    profile: jev\n    when: {}\n    then:\n      raw:\n        status: 200\n        headers:\n          content-type: application/json\n        body:\n          ok: true\n")
	jsonLoaded, err := Load(jsonConfig)
	if err != nil {
		t.Fatal(err)
	}
	yamlLoaded, err := Load(yamlConfig)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(jsonLoaded, yamlLoaded) {
		t.Fatalf("JSON and YAML differ:\nJSON %#v\nYAML %#v", jsonLoaded, yamlLoaded)
	}
}

func TestRejectsUnknownAndInvalidConfiguration(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"missing schema", "mode: strict"},
		{"unknown top level", "schemaVersion: 1\nextra: true"},
		{"unknown nested", "schemaVersion: 1\nserver:\n  nope: true"},
		{"non strict", "schemaVersion: 1\nmode: permissive"},
		{"unknown profile", "schemaVersion: 1\ncompatibility: [other]"},
		{"duplicate id", "schemaVersion: 1\nstubs:\n- {id: x, profile: jev/v1, when: {}, then: {raw: {status: 200, body: null}}}\n- {id: x, profile: jev/v1, when: {}, then: {raw: {status: 200, body: null}}}"},
		{"multiple forms", "schemaVersion: 1\nstubs:\n- id: x\n  profile: jev/v1\n  when: {}\n  then: {answers: {}, raw: {status: 200, body: null}}"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Load([]byte(test.body)); err == nil {
				t.Fatal("expected configuration error")
			}
		})
	}
}

func TestRawBodyMayBeOmitted(t *testing.T) {
	body := `{"schemaVersion":1,"stubs":[{"id":"x","profile":"jev/v1","when":{},"then":{"raw":{"status":204,"headers":{}}}}]}`
	if _, err := Load([]byte(body)); err != nil {
		t.Fatalf("raw response without body rejected: %v", err)
	}
}

func TestRejectsNullAndEmptyStructuredValues(t *testing.T) {
	tests := []string{
		`{"schemaVersion":1,"stubs":[{"id":"x","profile":"jev/v1","when":{"operation":null},"then":{"raw":{"status":200}}}]}`,
		`{"schemaVersion":1,"stubs":[{"id":"x","profile":"jev/v1","when":{"questions":null},"then":{"raw":{"status":200}}}]}`,
		`{"schemaVersion":1,"stubs":[{"id":"x","profile":"jev/v1","when":{},"then":{"sequence":[]}}]}`,
	}
	for _, body := range tests {
		if _, err := Load([]byte(body)); err == nil {
			t.Errorf("expected invalid configuration: %s", body)
		}
	}
}

func TestExpectForms(t *testing.T) {
	valid := []string{
		`{"schemaVersion":1,"stubs":[{"id":"x","profile":"jev/v1","when":{},"then":{"raw":{"status":200,"headers":{},"body":null}},"expect":{"exactly":0}}]}`,
		`{"schemaVersion":1,"stubs":[{"id":"x","profile":"jev/v1","when":{},"then":{"raw":{"status":200,"headers":{},"body":null}},"expect":{"atLeast":1}}]}`,
		`{"schemaVersion":1,"stubs":[{"id":"x","profile":"jev/v1","when":{},"then":{"raw":{"status":200,"headers":{},"body":null}},"expect":{"atLeast":1,"atMost":3}}]}`,
	}
	for _, body := range valid {
		if _, err := Load([]byte(body)); err != nil {
			t.Errorf("valid expect rejected: %v", err)
		}
	}
	invalid := `{"schemaVersion":1,"stubs":[{"id":"x","profile":"jev/v1","when":{},"then":{"raw":{"status":200,"headers":{},"body":null}},"expect":{"exactly":1,"atLeast":1}}]}`
	if _, err := Load([]byte(invalid)); err == nil {
		t.Fatal("expected invalid expect form")
	}
}
