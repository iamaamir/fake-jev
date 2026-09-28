package engine

import (
	"reflect"
	"sync"
	"testing"
)

func TestSequenceConsumptionAndExhaustionGoldenVector(t *testing.T) {
	engine, err := NewEngine([]Stub{{
		ID:       "retry",
		Profile:  "jev/v1",
		Sequence: []ResponseAction{"yes", "no"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	exchange := Exchange{Profile: "jev/v1"}

	first := engine.SelectInvocation(exchange)
	second := engine.SelectInvocation(exchange)
	third := engine.SelectInvocation(exchange)
	if first == nil || second == nil || third == nil {
		t.Fatalf("sequence selections = %#v, %#v, %#v; want three selected calls", first, second, third)
	}
	if first.SequenceExhausted || second.SequenceExhausted || !third.SequenceExhausted {
		t.Fatalf("exhaustion flags = %v, %v, %v; want false, false, true", first.SequenceExhausted, second.SequenceExhausted, third.SequenceExhausted)
	}
	if !reflect.DeepEqual([]ResponseAction{first.Action, second.Action}, []ResponseAction{"yes", "no"}) {
		t.Fatalf("sequence actions = %#v, %#v; want yes, no", first.Action, second.Action)
	}
	if got := third.Stub.InvocationCount; got != 3 {
		t.Fatalf("invocation count after exhaustion = %d, want 3", got)
	}
	if got := third.Stub.SequencePosition; got != 2 {
		t.Fatalf("sequence position after exhaustion = %d, want 2", got)
	}
	if got := engine.EvaluateExpectations(); len(got) != 0 {
		t.Fatalf("unexpected expectation failures: %#v", got)
	}
}

func TestEmptySequenceExhaustsWithoutRepeating(t *testing.T) {
	engine, err := NewEngine([]Stub{{
		ID:       "empty",
		Profile:  "profile",
		Sequence: []ResponseAction{},
	}})
	if err != nil {
		t.Fatal(err)
	}
	first := engine.SelectInvocation(Exchange{Profile: "profile"})
	second := engine.SelectInvocation(Exchange{Profile: "profile"})
	if first == nil || second == nil || !first.SequenceExhausted || !second.SequenceExhausted {
		t.Fatalf("empty-sequence selections = %#v, %#v; want exhausted selections", first, second)
	}
	if first.Action != nil || second.Action != nil {
		t.Fatalf("empty-sequence actions = %#v, %#v; want nil actions", first.Action, second.Action)
	}
	if first.Stub.InvocationCount != 1 || second.Stub.InvocationCount != 2 {
		t.Fatalf("empty-sequence counts = %d, %d; want 1, 2", first.Stub.InvocationCount, second.Stub.InvocationCount)
	}
	if first.Stub.SequencePosition != 0 || second.Stub.SequencePosition != 0 {
		t.Fatalf("empty-sequence positions = %d, %d; want 0, 0", first.Stub.SequencePosition, second.Stub.SequencePosition)
	}
}

func TestConcurrentSelectionSerializesState(t *testing.T) {
	const calls = 32
	sequence := make([]ResponseAction, calls)
	for i := range sequence {
		sequence[i] = i
	}
	engine, err := NewEngine([]Stub{{ID: "concurrent", Profile: "profile", Sequence: sequence}})
	if err != nil {
		t.Fatal(err)
	}

	start := make(chan struct{})
	results := make(chan *Selection, calls)
	var workers sync.WaitGroup
	workers.Add(calls)
	for i := 0; i < calls; i++ {
		go func() {
			defer workers.Done()
			<-start
			results <- engine.SelectInvocation(Exchange{Profile: "profile"})
		}()
	}
	close(start)
	workers.Wait()
	close(results)
	for selection := range results {
		if selection == nil || selection.SequenceExhausted {
			t.Fatalf("concurrent selection = %#v; want non-exhausted result", selection)
		}
	}
	stub := engine.Registry.Stubs()[0]
	if stub.InvocationCount != calls || stub.SequencePosition != calls {
		t.Fatalf("concurrent state = count %d, position %d; want %d, %d", stub.InvocationCount, stub.SequencePosition, calls, calls)
	}
}

func TestInvalidExpectationDoesNotRegister(t *testing.T) {
	one := uint64(1)
	two := uint64(2)
	registry := NewRegistry()
	err := registry.RegisterStatic(Stub{
		ID:      "invalid",
		Profile: "profile",
		Expect:  &InvocationExpectation{Exactly: &one, AtMost: &two},
	})
	if err == nil {
		t.Fatal("invalid expectation unexpectedly registered")
	}
	if len(registry.Stubs()) != 0 {
		t.Fatal("invalid expectation mutated registry")
	}
	if err := registry.RegisterStatic(Stub{ID: "valid", Profile: "profile"}); err != nil {
		t.Fatal(err)
	}
	if got := registry.Stubs()[0].RegistrationIndex; got != 1 {
		t.Fatalf("invalid expectation consumed registration index: got %d, want 1", got)
	}
}

func TestNilEngineAndRegistryAreSafe(t *testing.T) {
	var engine *Engine
	if engine.Select(Exchange{}) != nil || engine.SelectInvocation(Exchange{}) != nil {
		t.Fatal("nil engine selection unexpectedly returned a result")
	}
	if engine.EvaluateExpectations() != nil {
		t.Fatal("nil engine expectations unexpectedly returned failures")
	}
	var registry *Registry
	if registry.Select(Exchange{}) != nil || registry.SelectInvocation(Exchange{}) != nil {
		t.Fatal("nil registry selection unexpectedly returned a result")
	}
	if registry.EvaluateExpectations() != nil {
		t.Fatal("nil registry expectations unexpectedly returned failures")
	}
}

func TestSelectionCountsBeforeResponseFailure(t *testing.T) {
	engine, err := NewEngine([]Stub{{
		ID:       "fails-later",
		Profile:  "profile",
		Sequence: []ResponseAction{"response"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	selection := engine.SelectInvocation(Exchange{Profile: "profile"})
	if selection == nil || selection.Action != "response" {
		t.Fatalf("selection = %#v, want response action", selection)
	}
	// A provider may fail while consuming the opaque action. The engine has
	// already committed the invocation count at selection time.
	if got := engine.Registry.Stubs()[0].InvocationCount; got != 1 {
		t.Fatalf("invocation count after later failure = %d, want 1", got)
	}
}

func TestUnmatchedSelectionDoesNotAdvanceSequence(t *testing.T) {
	engine, err := NewEngine([]Stub{{
		ID:       "profiled",
		Profile:  "profile",
		Sequence: []ResponseAction{"first"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if got := engine.SelectInvocation(Exchange{Profile: "other"}); got != nil {
		t.Fatalf("unmatched selection = %#v, want nil", got)
	}
	selection := engine.SelectInvocation(Exchange{Profile: "profile"})
	if selection == nil || selection.Action != "first" || selection.Stub.SequencePosition != 1 {
		t.Fatalf("first matched selection = %#v, want first action at position 1", selection)
	}
}

func TestInvocationExpectationValidation(t *testing.T) {
	one := uint64(1)
	two := uint64(2)
	cases := []struct {
		name   string
		expect InvocationExpectation
		want   bool
	}{
		{name: "exactly", expect: InvocationExpectation{Exactly: &one}, want: true},
		{name: "at least", expect: InvocationExpectation{AtLeast: &one}, want: true},
		{name: "at most", expect: InvocationExpectation{AtMost: &two}, want: true},
		{name: "bounded range", expect: InvocationExpectation{AtLeast: &one, AtMost: &two}, want: true},
		{name: "exactly with range rejected", expect: InvocationExpectation{Exactly: &one, AtMost: &two}, want: false},
		{name: "empty rejected", expect: InvocationExpectation{}, want: false},
		{name: "reversed range rejected", expect: InvocationExpectation{AtLeast: &two, AtMost: &one}, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.expect.Validate() == nil
			if got != tc.want {
				t.Fatalf("Validate() success = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestInvocationExpectationsEvaluateTable(t *testing.T) {
	one := uint64(1)
	two := uint64(2)
	three := uint64(3)
	cases := []struct {
		name       string
		expect     InvocationExpectation
		calls      int
		wantCode   string
		wantErrors int
	}{
		{name: "exactly unsatisfied", expect: InvocationExpectation{Exactly: &two}, calls: 1, wantCode: "expect_exactly", wantErrors: 1},
		{name: "atLeast unsatisfied", expect: InvocationExpectation{AtLeast: &two}, calls: 1, wantCode: "expect_at_least", wantErrors: 1},
		{name: "atMost unsatisfied", expect: InvocationExpectation{AtMost: &one}, calls: 2, wantCode: "expect_at_most", wantErrors: 1},
		{name: "range lower bound", expect: InvocationExpectation{AtLeast: &two, AtMost: &three}, calls: 1, wantCode: "expect_at_least", wantErrors: 1},
		{name: "range upper bound", expect: InvocationExpectation{AtLeast: &one, AtMost: &two}, calls: 3, wantCode: "expect_at_most", wantErrors: 1},
		{name: "range satisfied", expect: InvocationExpectation{AtLeast: &one, AtMost: &two}, calls: 2, wantErrors: 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			engine, err := NewEngine([]Stub{{
				ID: "stub", Profile: "profile", Matcher: Matcher{Operation: "run"}, Expect: &tc.expect,
			}})
			if err != nil {
				t.Fatal(err)
			}
			for i := 0; i < tc.calls; i++ {
				if engine.SelectInvocation(Exchange{Profile: "profile", Operation: "run"}) == nil {
					t.Fatal("expected matching selection")
				}
			}
			failures := engine.EvaluateExpectations()
			if len(failures) != tc.wantErrors {
				t.Fatalf("failures = %#v, want %d", failures, tc.wantErrors)
			}
			if tc.wantCode != "" && failures[0].Code != tc.wantCode {
				t.Fatalf("failure code = %q, want %q", failures[0].Code, tc.wantCode)
			}
		})
	}
}

func TestInvocationExpectationsEvaluateByRegistrationOrder(t *testing.T) {
	one := uint64(1)
	engine, err := NewEngine([]Stub{
		{ID: "exact", Profile: "profile", Matcher: Matcher{Operation: "exact"}, Expect: &InvocationExpectation{Exactly: &one}},
		{ID: "lower", Profile: "profile", Matcher: Matcher{Operation: "lower"}, Expect: &InvocationExpectation{AtLeast: &one}},
		{ID: "upper", Profile: "profile", Matcher: Matcher{Operation: "upper"}, Expect: &InvocationExpectation{AtMost: &one}},
	})
	if err != nil {
		t.Fatal(err)
	}
	// Select exact once, then select upper twice. Lower remains unsatisfied.
	engine.SelectInvocation(Exchange{Profile: "profile", Operation: "exact"})
	engine.SelectInvocation(Exchange{Profile: "profile", Operation: "upper"})
	engine.SelectInvocation(Exchange{Profile: "profile", Operation: "upper"})
	failures := engine.EvaluateExpectations()
	if len(failures) != 2 {
		t.Fatalf("expectation failures = %#v, want two", failures)
	}
	if failures[0].Code != "expect_at_least" || failures[0].StubID != "lower" {
		t.Fatalf("first failure = %#v, want lower bound failure", failures[0])
	}
	if failures[1].Code != "expect_at_most" || failures[1].StubID != "upper" {
		t.Fatalf("second failure = %#v, want upper bound failure", failures[1])
	}
}
