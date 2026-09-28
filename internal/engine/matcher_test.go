package engine

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestMatcherAcceptanceCriteria(t *testing.T) {
	bigNumber := json.RawMessage(`9007199254740993`)
	cases := []struct {
		name     string
		matcher  Matcher
		exchange Exchange
		want     bool
	}{
		{
			name:    "omitted fields are wildcards",
			matcher: Matcher{}, exchange: Exchange{Operation: "any", Model: "model", Questions: map[string]string{"q": "type-a"}}, want: true,
		},
		{
			name:     "all present fields are ANDed",
			matcher:  Matcher{Operation: "run", Model: "m", Questions: map[string]string{"q": "type-a"}},
			exchange: Exchange{Operation: "run", Model: "m", Questions: map[string]string{"q": "type-a"}}, want: true,
		},
		{
			name:    "operation mismatch rejects",
			matcher: Matcher{Operation: "run"}, exchange: Exchange{Operation: "other"}, want: false,
		},
		{
			name:    "model is exact and has no aliases",
			matcher: Matcher{Model: "exact"}, exchange: Exchange{Model: "alias"}, want: false,
		},
		{
			name:     "questions reject extra names",
			matcher:  Matcher{Questions: map[string]string{"q1": "type-a"}},
			exchange: Exchange{Questions: map[string]string{"q1": "type-a", "q2": "type-b"}}, want: false,
		},
		{
			name:     "questions reject missing names",
			matcher:  Matcher{Questions: map[string]string{"q1": "type-a", "q2": "type-b"}},
			exchange: Exchange{Questions: map[string]string{"q1": "type-a"}}, want: false,
		},
		{
			name:     "questions require exact type",
			matcher:  Matcher{Questions: map[string]string{"q1": "type-a"}},
			exchange: Exchange{Questions: map[string]string{"q1": "type-b"}}, want: false,
		},
		{
			name:     "question contents are ignored",
			matcher:  Matcher{Questions: map[string]string{"q1": "type-a"}},
			exchange: Exchange{Questions: map[string]string{"q1": "type-a"}, Payload: map[string]any{"instructions": "different", "criteria": []string{"a"}}}, want: true,
		},
		{
			name:     "state objects ignore key order",
			matcher:  Matcher{State: json.RawMessage(`{"a":1,"b":[true,"x"]}`)},
			exchange: Exchange{State: json.RawMessage(`{"b":[true,"x"],"a":1.0}`)}, want: true,
		},
		{
			name:    "state numbers compare numerically without float precision loss",
			matcher: Matcher{State: bigNumber}, exchange: Exchange{State: json.RawMessage(`9007199254740993.0`)}, want: true,
		},
		{
			name:    "state exponent equals integer",
			matcher: Matcher{State: json.RawMessage(`1`)}, exchange: Exchange{State: json.RawMessage(`1e0`)}, want: true,
		},
		{
			name:    "state arrays preserve order",
			matcher: Matcher{State: json.RawMessage(`[1,2]`)}, exchange: Exchange{State: json.RawMessage(`[2,1]`)}, want: false,
		},
		{
			name:    "state null compares as JSON null",
			matcher: Matcher{State: nil, HasState: true}, exchange: Exchange{State: nil}, want: true,
		},
		{
			name:    "omitted raw state is a wildcard",
			matcher: Matcher{State: json.RawMessage(nil)}, exchange: Exchange{State: json.RawMessage(`{"present":true}`)}, want: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.matcher.Matches(tc.exchange)
			if got != tc.want {
				t.Fatalf("Matches() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRegistryRegistrationProvenanceAndIndices(t *testing.T) {
	registry := NewRegistry()
	static := Stub{ID: "static", Profile: "profile"}
	dynamic := Stub{ID: "dynamic", Profile: "profile"}
	if err := registry.RegisterStatic(static); err != nil {
		t.Fatal(err)
	}
	if err := registry.RegisterDynamic(dynamic); err != nil {
		t.Fatal(err)
	}
	stubs := registry.Stubs()
	if len(stubs) != 2 {
		t.Fatalf("registered %d stubs, want 2", len(stubs))
	}
	if stubs[0].Source != SourceStatic || stubs[0].RegistrationIndex != 1 {
		t.Fatalf("static provenance/index = %#v, want static/1", stubs[0])
	}
	if stubs[1].Source != SourceDynamic || stubs[1].RegistrationIndex != 2 {
		t.Fatalf("dynamic provenance/index = %#v, want dynamic/2", stubs[1])
	}
	if err := registry.RegisterDynamic(Stub{ID: "dynamic", Profile: "profile"}); err == nil {
		t.Fatal("duplicate registration unexpectedly succeeded")
	}
	if err := registry.RegisterDynamic(Stub{ID: "third", Profile: "profile"}); err != nil {
		t.Fatal(err)
	}
	if got := registry.Stubs()[2].RegistrationIndex; got != 3 {
		t.Fatalf("failed registration consumed index: got %d, want 3", got)
	}
}

func TestRegistryRejectsInvalidAndExhaustedRegistrations(t *testing.T) {
	var zero Registry
	if err := zero.RegisterStatic(Stub{ID: "zero", Profile: "profile"}); err != nil {
		t.Fatal(err)
	}
	if got := zero.Stubs()[0].RegistrationIndex; got != 1 {
		t.Fatalf("zero-value registry index = %d, want 1", got)
	}
	for _, stub := range []Stub{{Profile: "profile"}, {ID: "missing-profile"}} {
		if err := zero.RegisterStatic(stub); err == nil {
			t.Fatalf("invalid stub %#v unexpectedly registered", stub)
		}
	}

	exhausted := Registry{nextRegistrationIndex: ^uint64(0)}
	if err := exhausted.RegisterStatic(Stub{ID: "last", Profile: "profile"}); err != nil {
		t.Fatal(err)
	}
	if got := exhausted.Stubs()[0].RegistrationIndex; got != ^uint64(0) {
		t.Fatalf("maximum registration index = %d, want %d", got, ^uint64(0))
	}
	if err := exhausted.RegisterStatic(Stub{ID: "after", Profile: "profile"}); err == nil {
		t.Fatal("registration after index exhaustion unexpectedly succeeded")
	}
	if got := len(exhausted.Stubs()); got != 1 {
		t.Fatalf("exhausted registry contains %d stubs, want 1", got)
	}
}

func TestRegistryDetachesMatcherInputsAndOutputs(t *testing.T) {
	questions := map[string]string{"q": "choice"}
	state := map[string]any{"nested": map[string]any{"value": "original"}}
	registry := NewRegistry()
	if err := registry.RegisterStatic(Stub{
		ID:      "detached",
		Profile: "profile",
		Matcher: Matcher{Questions: questions, State: state, HasState: true},
	}); err != nil {
		t.Fatal(err)
	}
	questions["q"] = "changed"
	setNestedStateValue(t, state, "changed")
	exchange := Exchange{Profile: "profile", Questions: map[string]string{"q": "choice"}, State: map[string]any{"nested": map[string]any{"value": "original"}}}
	if registry.Select(exchange) == nil {
		t.Fatal("registration retained mutable caller state")
	}

	returned := registry.Stubs()
	returned[0].Matcher.Questions["q"] = "changed-again"
	setNestedStateValue(t, returned[0].Matcher.State, "changed-again")
	if registry.Select(exchange) == nil {
		t.Fatal("Stubs exposed mutable registry matcher state")
	}
	selected := registry.Select(exchange)
	if selected == nil {
		t.Fatal("selection unexpectedly disappeared")
	}
	selected.Matcher.Questions["q"] = "changed-selected"
	if registry.Select(exchange) == nil {
		t.Fatal("Select exposed mutable registry matcher state")
	}
}

func setNestedStateValue(t *testing.T, value Value, replacement string) {
	t.Helper()
	root, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("state root type = %T, want map[string]any", value)
	}
	nested, ok := root["nested"].(map[string]any)
	if !ok {
		t.Fatalf("nested state type = %T, want map[string]any", root["nested"])
	}
	nested["value"] = replacement
}

func TestMatcherRejectsOverflowingJSONExponent(t *testing.T) {
	matcher := Matcher{State: json.RawMessage(`1e` + strings.Repeat("9", 128)), HasState: true}
	if matcher.Matches(Exchange{State: json.RawMessage(`1`)}) {
		t.Fatal("overflowing exponent unexpectedly matched")
	}
}

func TestMatcherRejectsCyclicStateWithoutPanic(t *testing.T) {
	cyclic := map[string]any{}
	cyclic["self"] = cyclic
	matcher := Matcher{State: cyclic, HasState: true}
	if matcher.Matches(Exchange{State: map[string]any{}}) {
		t.Fatal("cyclic state unexpectedly matched")
	}
}

func TestGoldenMatchingAndOrderingVectors(t *testing.T) {
	all := Matcher{}
	registry := NewRegistry()
	for _, stub := range []Stub{
		{ID: "A", Profile: "jev/v1", Priority: 0, Matcher: all},
		{ID: "B", Profile: "jev/v1", Priority: 10, Matcher: all},
		{ID: "C", Profile: "jev/v1", Priority: 10, Matcher: all},
		{ID: "other-profile", Profile: "other", Priority: 100, Matcher: all},
	} {
		if err := registry.RegisterStatic(stub); err != nil {
			t.Fatal(err)
		}
	}
	request := Exchange{Profile: "jev/v1"}
	if got := registry.Select(request); got == nil || got.ID != "B" {
		t.Fatalf("priority vector selected %#v, want B", got)
	}
	if got := registry.Matching(request); !reflect.DeepEqual(ids(got), []string{"B", "C", "A"}) {
		t.Fatalf("ordered matches = %v, want [B C A]", ids(got))
	}
	removed := NewRegistry()
	for _, stub := range []Stub{
		{ID: "A", Profile: "jev/v1", Priority: 0, Matcher: all},
		{ID: "C", Profile: "jev/v1", Priority: 10, Matcher: all},
	} {
		if err := removed.RegisterStatic(stub); err != nil {
			t.Fatal(err)
		}
	}
	if got := removed.Select(request); got == nil || got.ID != "C" {
		t.Fatalf("remaining tie vector selected %#v, want C", got)
	}

	dynamic := NewRegistry()
	for _, stub := range []Stub{
		{ID: "static", Profile: "jev/v1", Priority: 0, Matcher: all},
		{ID: "dynamic-equal", Profile: "jev/v1", Priority: 0, Matcher: all},
	} {
		var err error
		if stub.ID == "static" {
			err = dynamic.RegisterStatic(stub)
		} else {
			err = dynamic.RegisterDynamic(stub)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	if got := dynamic.Select(request); got == nil || got.ID != "static" {
		t.Fatalf("equal-priority dynamic vector selected %#v, want static", got)
	}
	if err := dynamic.RegisterDynamic(Stub{ID: "dynamic-greater", Profile: "jev/v1", Priority: 1, Matcher: all}); err != nil {
		t.Fatal(err)
	}
	if got := dynamic.Select(request); got == nil || got.ID != "dynamic-greater" {
		t.Fatalf("greater-priority dynamic vector selected %#v, want dynamic-greater", got)
	}
}

func ids(stubs []Stub) []string {
	result := make([]string, len(stubs))
	for i := range stubs {
		result[i] = stubs[i].ID
	}
	return result
}
