package config

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"
)

// The fuzz target in this file establishes that malformed configuration bytes
// are answered with a controlled decode error instead of a panic or a
// half-applied model, and that a successful load is reproducible, validated,
// defaulted, and non-amplifying. Only Load (bytes) is exercised; LoadFile's
// unbounded read of a caller-supplied path is a recorded out-of-scope defect.
// Traces C-QUAL-004 and C-QUAL-002.

// configFuzzMaxBytes caps the input the fuzzer may hand to the decoder. It
// bounds the input, not the cost of decoding it: Load's work is not bounded by
// the byte length of a document, because YAML anchors and aliases let a small
// document expand to a large decoded value. Bounding the cost of decoding a
// document is a product change and is not made here; the closest tracked item is
// the unbounded configuration read logged after FJ-030. This cap only keeps the
// fuzzer's inputs comparable in size.
const configFuzzMaxBytes = 64 << 10

// configFuzzMaxMappingKeys bounds how many mapping entries a document handed to
// the YAML decoder may declare before this target stops calling Load on it. It
// bounds the work the target asks for, not the cost of Load.
//
// The bound is still needed after FJ-065, and it applies only to the YAML path.
// FJ-065 made Load answer a JSON document with a repeated key from its own JSON
// scan, so that shape no longer reaches yaml.v3 at all; it also keeps only the
// first duplicate-key diagnostic. yaml.v3 still formats one diagnostic string
// per repeated-key pair *inside* decodeYAML, before Load's filter can drop them,
// and it still compares every pair of mapping keys, so both its time and its
// allocation stay quadratic in the number of keys of a mapping that is decoded
// by YAML. Measured with Load at this revision: a 32,000-byte YAML document
// repeating one key 6,400 times costs 3.1 s and 3.8 GiB, 26,203 bytes repeating
// it 13,100 times costs 17.4 s and 13.9 GiB, and 65,535 bytes costs 21.4 s and
// 15.2 GiB. Each of those is inside configFuzzMaxBytes, so the byte cap alone
// does not bound them. Malformed input of that shape stays covered up to this
// bound; the quadratic decode cost itself is a suspected product defect, not a
// property this test asserts.
const configFuzzMaxMappingKeys = 256

// configFuzzMappingKeys counts the separators that can introduce a mapping
// entry: ":" for entries written with a value in either style, and "," for
// flow-style entries, which a YAML mapping may write without a value at all
// ("{a, a}" is a two-entry mapping with no colon). Flow sequences are counted by
// the same "," and are cheaper to decode than mappings, so the count is a
// conservative proxy rather than an exact key count.
func configFuzzMappingKeys(data []byte) int {
	return bytes.Count(data, []byte{':'}) + bytes.Count(data, []byte{','})
}

// configFuzzYAMLPath reports whether Load would hand data to the YAML decoder,
// by repeating Load's own routing in toJSON: a document that starts with "{" or
// "[" is decoded by encoding/json when the in-package JSON scan accepts it or
// rejects it as a repeated key, and reaches the YAML decoder in every other
// case. Reusing the scan keeps the routing decision identical to the product's,
// including decoded-key equality ("\u0061" counts as a repeat of "a").
func configFuzzYAMLPath(data []byte) bool {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return false
	}
	if trimmed[0] != '{' && trimmed[0] != '[' {
		return true
	}
	err := rejectDuplicateJSON(trimmed)
	if err == nil {
		return false
	}
	var duplicate *duplicateKeyError
	return !errors.As(err, &duplicate)
}

// configFuzzBeyondKeyBound reports whether a document declares more mapping
// entries than configFuzzMaxMappingKeys and is decoded by the YAML decoder. A
// document the JSON scan accepts, or rejects as a repeated key, is decoded by
// encoding/json in linear time and stays fuzzable at any size, so the duplicate
// key handling FJ-065 added is exercised here up to configFuzzMaxBytes.
func configFuzzBeyondKeyBound(data []byte) bool {
	if configFuzzMappingKeys(data) <= configFuzzMaxMappingKeys {
		return false
	}
	return configFuzzYAMLPath(data)
}

// assertLoadedConfigDefaults checks the §12 defaults and the non-amplification
// property that a loaded configuration cannot declare more stubs than the bytes
// it was built from.
func assertLoadedConfigDefaults(t *testing.T, document []byte, cfg *Config) {
	t.Helper()
	if cfg.SchemaVersion != 1 {
		t.Fatalf("loaded schemaVersion is %d, want 1", cfg.SchemaVersion)
	}
	if err := Validate(cfg); err != nil {
		t.Fatalf("Validate rejected a loaded configuration: %v", err)
	}
	if cfg.Server.Host != DefaultHost || cfg.Server.Port != DefaultPort {
		t.Fatalf("server defaults are %q:%d, want %q:%d", cfg.Server.Host, cfg.Server.Port, DefaultHost, DefaultPort)
	}
	if cfg.Mode != DefaultMode {
		t.Fatalf("mode default is %q, want %q", cfg.Mode, DefaultMode)
	}
	if len(cfg.Compatibility) < 1 {
		t.Fatal("loaded configuration has no compatibility profile")
	}
	if cfg.Limits.DataPlaneBodyBytes <= 0 || cfg.Limits.ControlPlaneBodyBytes <= 0 ||
		cfg.Limits.MaxInteractions <= 0 || cfg.Limits.LogBodyBytes <= 0 ||
		cfg.Limits.GracefulShutdownSeconds <= 0 {
		t.Fatalf("loaded limits are not all positive: %#v", cfg.Limits)
	}
	if len(cfg.Models) < 1 {
		t.Fatal("loaded configuration has no models")
	}
	if cfg.Stubs == nil {
		t.Fatal("loaded configuration stubs are nil")
	}
	if len(cfg.Stubs) > len(document) {
		t.Fatalf("loaded %d stubs from %d configuration bytes", len(cfg.Stubs), len(document))
	}
}

// FuzzConfigMalformedInput feeds arbitrary bytes to the configuration loader.
// Every input must decode to a validated, defaulted configuration or to a
// non-empty error, and identical bytes must always produce the same outcome so
// no partially applied state can leak into a caller.
func FuzzConfigMalformedInput(f *testing.F) {
	seeds := []string{
		`{"schemaVersion":1}`,
		`{}`,
		`null`,
		`[]`,
		`"x"`,
		`0`,
		"schemaVersion: 1\n",
		"schemaVersion: 1\n---\nschemaVersion: 1\n",
		`{"schemaVersion":1,"schemaVersion":1}`,
		`{"schemaVersion":1,"unknown":true}`,
		`{"schemaVersion":2}`,
		`{"schemaVersion":1,"limits":{"maxInteractions":-1}}`,
		`{"schemaVersion":1,"stubs":[{"id":"s","profile":"jev/v1","when":{},"raw":{"status":200,"headers":{},"body":"x"}}]}`,
		`{"schemaVersion":1,"mode":"auto"}`,
		strings.Repeat("<", 1<<10),
		`{"schemaVersion":1,"models":[{"name":"müde","description":"ok","release_date":"1970-01-01"}]}`,
		"{\"schemaVersion\":1}\x00",
		strings.Repeat("[", 256) + strings.Repeat("]", 256),
	}
	for _, seed := range seeds {
		f.Add([]byte(seed))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > configFuzzMaxBytes || configFuzzBeyondKeyBound(data) {
			return
		}
		dataBefore := append([]byte(nil), data...)

		first, firstErr := Load(data)
		second, secondErr := Load(data)

		if (firstErr == nil) != (secondErr == nil) {
			t.Fatalf("Load accept/reject differs between identical calls: %v then %v", firstErr, secondErr)
		}
		if firstErr != nil {
			if firstErr.Error() == "" {
				t.Fatal("Load reported an empty diagnostic")
			}
			if firstErr.Error() != secondErr.Error() {
				t.Fatalf("Load diagnostics differ between identical calls: %q then %q", firstErr, secondErr)
			}
			if first != nil {
				t.Fatalf("Load returned a configuration together with error %v", firstErr)
			}
		} else {
			if first == nil || second == nil {
				t.Fatal("Load returned a nil configuration without an error")
			}
			if !reflect.DeepEqual(first, second) {
				t.Fatalf("Load is not reproducible: %#v then %#v", first, second)
			}
			assertLoadedConfigDefaults(t, data, first)
		}

		if !bytes.Equal(dataBefore, data) {
			t.Fatalf("Load mutated the caller's configuration bytes %q into %q", dataBefore, data)
		}
	})
}
