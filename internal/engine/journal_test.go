package engine

import (
	"sync"
	"testing"
)

func TestTransitionAssignsSequenceBeforeRouteAndRecordsRejectedRequests(t *testing.T) {
	engine, err := NewEngineWithJournal(nil, 4)
	if err != nil {
		t.Fatal(err)
	}
	var seen []uint64
	tests := []struct {
		name   string
		input  InteractionRequest
		valid  bool
		status int
		result string
	}{
		{name: "unknown route", input: InteractionRequest{Profile: "none", Operation: "missing"}, result: "unknown_route", status: 404},
		{name: "malformed", input: InteractionRequest{Profile: "jev/v1", Operation: "systemone", RawBody: []byte("{")}, result: "json_invalid", status: 422},
		{name: "valid", input: InteractionRequest{Profile: "jev/v1", Operation: "systemone"}, valid: true, result: "matched", status: 200},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			record, selection, err := engine.Transition(test.input, func(sequence uint64) InteractionDecision {
				seen = append(seen, sequence)
				return InteractionDecision{
					Exchange: Exchange{Profile: test.input.Profile, Operation: test.input.Operation},
					Valid:    test.valid, Outcome: test.result, ResponseStatus: test.status,
				}
			})
			if err != nil {
				t.Fatal(err)
			}
			if selection != nil {
				t.Fatalf("selection = %#v, want nil", selection)
			}
			if record == nil || record.Sequence != uint64(len(seen)) {
				t.Fatalf("record = %#v, want sequence %d", record, len(seen))
			}
		})
	}
	for i, want := range []uint64{1, 2, 3} {
		if seen[i] != want {
			t.Fatalf("handler sequence[%d] = %d, want %d", i, seen[i], want)
		}
	}
}

func TestTransitionRejectsNilHandlerWithoutAdmission(t *testing.T) {
	engine, err := NewEngineWithJournal(nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := engine.Transition(InteractionRequest{Profile: "profile"}, nil); err == nil {
		t.Fatal("nil handler should return an error")
	}
	if got := engine.Interactions(); len(got) != 0 {
		t.Fatalf("interactions after nil handler = %#v, want empty", got)
	}
}

func TestZeroValueJournalStartsAtOne(t *testing.T) {
	journal := &interactionJournal{maxInteractions: 1}
	sequence, err := journal.reserve()
	if err != nil || sequence != 1 {
		t.Fatalf("zero-value journal reserve = %d, err %v; want sequence 1", sequence, err)
	}
}

func TestJournalSequenceExhaustionDoesNotWrap(t *testing.T) {
	engine, err := NewEngineWithJournal(nil, 2)
	if err != nil {
		t.Fatal(err)
	}
	engine.journal.nextSequence = ^uint64(0)
	first, _, err := engine.Transition(InteractionRequest{Profile: "profile"}, func(sequence uint64) InteractionDecision {
		if sequence != ^uint64(0) {
			t.Fatalf("sequence = %d, want max uint64", sequence)
		}
		return InteractionDecision{Valid: false, Outcome: "invalid"}
	})
	if err != nil || first == nil || first.Sequence != ^uint64(0) {
		t.Fatalf("first transition = %#v, err %v", first, err)
	}
	if _, _, err := engine.Transition(InteractionRequest{Profile: "profile"}, func(uint64) InteractionDecision {
		t.Fatal("handler ran after sequence exhaustion")
		return InteractionDecision{}
	}); err != ErrInteractionSequenceExhausted {
		t.Fatalf("second transition error = %v, want %v", err, ErrInteractionSequenceExhausted)
	}
	entries := engine.Interactions()
	if len(entries) != 1 || entries[0].Sequence != ^uint64(0) {
		t.Fatalf("entries after exhaustion = %#v", entries)
	}
}

func TestJournalFullIsBoundedStickyAndDoesNotMutateState(t *testing.T) {
	engine, err := NewEngineWithJournal([]Stub{{
		ID: "once", Profile: "profile", Sequence: []ResponseAction{"first"},
	}}, 1)
	if err != nil {
		t.Fatal(err)
	}
	first := engine.SelectInvocation(Exchange{Profile: "profile"})
	if first == nil || first.Action != "first" {
		t.Fatalf("first selection = %#v, want first action", first)
	}
	second := engine.SelectInvocation(Exchange{Profile: "profile"})
	if second != nil {
		t.Fatalf("full-journal selection = %#v, want nil", second)
	}
	stub := engine.Registry.Stubs()[0]
	if stub.InvocationCount != 1 || stub.SequencePosition != 1 {
		t.Fatalf("state after full journal = count %d position %d, want 1 and 1", stub.InvocationCount, stub.SequencePosition)
	}
	if got := engine.Interactions(); len(got) != 1 || got[0].Sequence != 1 {
		t.Fatalf("journal = %#v, want one entry sequence 1", got)
	}
	failures := engine.VerificationFailures()
	if len(failures) != 1 || failures[0].Code != "journal_full" || failures[0].RequestSequence != nil {
		t.Fatalf("failures = %#v, want sticky journal_full with nil sequence", failures)
	}
	if !engine.JournalFull() {
		t.Fatal("journal should report full")
	}

	engine.ClearRequestHistory()
	if len(engine.Interactions()) != 0 || len(engine.VerificationFailures()) != 0 {
		t.Fatal("clear did not remove interactions and derived failures")
	}
	if next := engine.SelectInvocation(Exchange{Profile: "profile"}); next == nil || next.Stub.InvocationCount != 2 {
		t.Fatalf("post-clear selection = %#v, want preserved counter at 2", next)
	}
	if entries := engine.Interactions(); len(entries) != 1 || entries[0].Sequence != 1 {
		t.Fatalf("post-clear journal = %#v, want sequence reset to 1", entries)
	}
}

func TestResetAtomicallyRestoresStaticAndDynamicState(t *testing.T) {
	engine, err := NewEngineWithJournal([]Stub{{
		ID: "static", Profile: "profile", Sequence: []ResponseAction{"static-action"},
	}}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if err := engine.RegisterDynamic(Stub{ID: "dynamic", Profile: "profile", Response: "dynamic-action"}); err != nil {
		t.Fatal(err)
	}
	if engine.SelectInvocation(Exchange{Profile: "profile"}) == nil {
		t.Fatal("initial selection failed")
	}
	engine.Reset()
	stubs := engine.Registry.Stubs()
	if len(stubs) != 1 || stubs[0].ID != "static" || stubs[0].InvocationCount != 0 || stubs[0].SequencePosition != 0 {
		t.Fatalf("stubs after reset = %#v, want rewound static only", stubs)
	}
	if got := engine.Interactions(); len(got) != 0 {
		t.Fatalf("journal after reset = %#v, want empty", got)
	}
	if err := engine.RegisterDynamic(Stub{ID: "new", Profile: "profile"}); err != nil {
		t.Fatal(err)
	}
	stubs = engine.Registry.Stubs()
	if len(stubs) != 2 || stubs[1].RegistrationIndex != 2 {
		t.Fatalf("dynamic index after reset = %#v, want 2", stubs)
	}
}

func TestJournalCopiesMutableRequestData(t *testing.T) {
	engine, err := NewEngineWithJournal(nil, 2)
	if err != nil {
		t.Fatal(err)
	}
	questions := map[string]string{"q": "choice"}
	exchange := Exchange{Profile: "profile", Questions: questions, State: map[string]any{"nested": "before"}}
	if _, _, err := engine.Transition(InteractionRequest{Profile: "profile"}, func(uint64) InteractionDecision {
		return InteractionDecision{Exchange: exchange, Valid: false, Outcome: "invalid"}
	}); err != nil {
		t.Fatal(err)
	}
	questions["new"] = "score"
	state, ok := exchange.State.(map[string]any)
	if !ok {
		t.Fatal("test state has unexpected type")
	}
	state["nested"] = "after"
	got := engine.Interactions()[0]
	if len(got.Request.Questions) != 1 || got.Request.Questions["q"] != "choice" {
		t.Fatalf("journal questions = %#v, want detached copy", got.Request.Questions)
	}
	journalState, ok := got.Request.State.(map[string]any)
	if !ok || journalState["nested"] != "before" {
		t.Fatalf("journal state = %#v, want detached copy", got.Request.State)
	}
}

func TestConcurrentTransitionsAndResetAreRaceFree(t *testing.T) {
	engine, err := NewEngineWithJournal([]Stub{{ID: "safe", Profile: "profile", Response: "ok"}}, 256)
	if err != nil {
		t.Fatal(err)
	}
	const calls = 128
	var workers sync.WaitGroup
	workers.Add(calls + 1)
	for i := 0; i < calls; i++ {
		go func() {
			defer workers.Done()
			engine.SelectInvocation(Exchange{Profile: "profile"})
		}()
	}
	go func() {
		defer workers.Done()
		engine.Reset()
	}()
	workers.Wait()
	stubs := engine.Registry.Stubs()
	if len(stubs) != 1 || stubs[0].InvocationCount > calls {
		t.Fatalf("state after concurrent reset = %#v", stubs)
	}
	entries := engine.Interactions()
	for i := 1; i < len(entries); i++ {
		if entries[i-1].Sequence >= entries[i].Sequence {
			t.Fatalf("journal not sequence ordered: %#v", entries)
		}
	}
}
