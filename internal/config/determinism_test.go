package config

import "testing"

// TestLoadDiagnosticsAreDeterministic pins that identical input always produces
// an identical outcome. An outcome is both whether Load failed and the exact
// message it returned, so a diagnostic that depends on Go's randomized map
// iteration order fails here. The number of calls is large enough that an
// iteration-order dependence cannot survive by chance: a two-way randomized
// choice survives 1000 calls with probability about 2^-999.
func TestLoadDiagnosticsAreDeterministic(t *testing.T) {
	const calls = 1000
	tests := []struct {
		name string
		body string
	}{
		{"yaml non string keys", `{07,2}`},
		{"nested yaml non string keys", `{a: {1: x}, b: {2: y}}`},
		{"yaml invalid question types", "schemaVersion: 1\nstubs:\n- id: x\n  profile: jev/v1\n  when:\n    questions:\n      a: bogus\n      b: bogus2\n  then:\n    answers: {}\n"},
		{"json null questions", `{"schemaVersion":1,"stubs":[{"id":"x","profile":"jev/v1","when":{"questions":{"a":null,"b":null}},"then":{"answers":{}}}]}`},
		{"json raw header names", `{"schemaVersion":1,"stubs":[{"id":"r","profile":"jev/v1","when":{},"then":{"raw":{"status":200,"headers":{"":"x"}}}}]}`},
		{"accepted yaml", "{schemaVersion: 1}"},
		{"accepted json", `{"schemaVersion":1}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			type outcome struct {
				failed  bool
				message string
			}
			load := func() outcome {
				_, err := Load([]byte(test.body))
				if err == nil {
					return outcome{}
				}
				return outcome{failed: true, message: err.Error()}
			}
			first := load()
			for call := 2; call <= calls; call++ {
				if got := load(); got != first {
					t.Fatalf("call %d returned %+v, call 1 returned %+v", call, got, first)
				}
			}
		})
	}
}
