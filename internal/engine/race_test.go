package engine

import (
	"sync"
	"testing"
)

// The tests in this file establish that the engine's serialized state
// transitions survive concurrent registration, selection, journaling,
// evaluation, history clearing, and resetting: admitted interactions are
// conserved exactly, registration indices stay unique and ordered, every
// recorded outcome is a legal one, and a reset leaves a fully rewound registry
// and journal. Every test starts its goroutines from one closed channel and
// joins them with one sync.WaitGroup, so no test waits on a clock or on host
// load. Traces C-HOST-009 and C-QUAL-002.

// concurrentDynamicIDs are the distinct runtime stub identifiers used by the
// concurrent registration workers. Registration indices are asserted to be
// unique and increasing, so the identifiers must not repeat.
var concurrentDynamicIDs = []string{
	"dynamic-0", "dynamic-1", "dynamic-2", "dynamic-3",
	"dynamic-4", "dynamic-5", "dynamic-6", "dynamic-7",
}

// engineReaderSnapshot is one reader goroutine's own detached view of engine
// state. Each reader writes only its own slot, so the test itself adds no
// shared mutable state.
type engineReaderSnapshot struct {
	interactions int
	expectations int
	journalFull  bool
}

// TestConcurrentRegistrationSelectionJournalAndResetAreRaceFree mixes data
// plane selection, dynamic registration and removal, history clearing,
// diagnostic recording, expectation evaluation, and reset on one engine. The
// assertions are conservation and envelope properties that must hold for every
// interleaving: no nil selection, no lost or duplicated registration index, a
// bounded and sequence-ordered journal, a legal outcome/status per record, and
// a completely rewound state after the final reset.
func TestConcurrentRegistrationSelectionJournalAndResetAreRaceFree(t *testing.T) {
	static := []Stub{
		{ID: "static-alpha", Profile: "profile", Priority: 20, Matcher: Matcher{Operation: "alpha"}, Sequence: []ResponseAction{"alpha-1", "alpha-2"}},
		{ID: "static-beta", Profile: "profile", Priority: 10, Matcher: Matcher{Operation: "beta"}, Expect: NewExactExpectation(0)},
		{ID: "static-gamma", Profile: "profile", Priority: 30, Matcher: Matcher{Operation: "gamma"}},
	}
	engineState, err := NewEngineWithJournal(static, 128)
	if err != nil {
		t.Fatalf("NewEngineWithJournal: %v", err)
	}

	const (
		selectors   = 64
		readers     = 8
		dynamics    = 8
		removals    = 8
		clears      = 8
		transitions = 8
	)

	selectionSlots := make([]*Selection, selectors)
	readerSlots := make([]engineReaderSnapshot, readers)
	registrationErrors := make([]error, dynamics)
	transitionErrors := make([]error, transitions)

	start := make(chan struct{})
	var workers sync.WaitGroup
	workers.Add(selectors + readers + dynamics + removals + clears + transitions + 1)

	for slot := 0; slot < selectors; slot++ {
		go func(slot int) {
			defer workers.Done()
			<-start
			operation := "alpha"
			if slot%2 == 1 {
				operation = "beta"
			}
			selectionSlots[slot] = engineState.SelectInvocation(Exchange{Profile: "profile", Operation: operation})
		}(slot)
	}
	for slot := 0; slot < readers; slot++ {
		go func(slot int) {
			defer workers.Done()
			<-start
			readerSlots[slot] = engineReaderSnapshot{
				interactions: len(engineState.Interactions()),
				expectations: len(engineState.EvaluateExpectations()),
				journalFull:  engineState.JournalFull(),
			}
		}(slot)
	}
	for slot := 0; slot < dynamics; slot++ {
		go func(slot int) {
			defer workers.Done()
			<-start
			registrationErrors[slot] = engineState.RegisterDynamic(Stub{ID: concurrentDynamicIDs[slot], Profile: "profile", Priority: 40, Matcher: Matcher{Operation: "dynamic"}})
		}(slot)
	}
	for worker := 0; worker < removals; worker++ {
		go func() {
			defer workers.Done()
			<-start
			engineState.RemoveDynamic()
		}()
	}
	for worker := 0; worker < clears; worker++ {
		go func() {
			defer workers.Done()
			<-start
			engineState.ClearRequestHistory()
		}()
	}
	for slot := 0; slot < transitions; slot++ {
		go func(slot int) {
			defer workers.Done()
			<-start
			record, _, err := engineState.Transition(InteractionRequest{Profile: "profile"}, func(uint64) InteractionDecision {
				return InteractionDecision{Exchange: Exchange{Profile: "profile"}, Valid: true}
			})
			transitionErrors[slot] = err
			if err != nil || record == nil {
				return
			}
			engineState.UpdateInteraction(record.Sequence, "fake_jev_invalid_stub_response", 500)
			engineState.RecordFailure(record.Sequence, "fake_jev_invalid_stub_response", "Concurrent invalid stub response.", "static-alpha", 500)
		}(slot)
	}
	go func() {
		defer workers.Done()
		<-start
		engineState.Reset()
	}()

	close(start)
	workers.Wait()

	// Admission conservation: the journal capacity is 128 and only 72
	// transitions are possible, so no selector can have been rejected and no
	// registration can have been refused for a repeated identifier. A runtime
	// stub matches only the "dynamic" operation, so it can never displace the
	// static stub that owns the operation a selector submitted.
	for slot, selection := range selectionSlots {
		if selection == nil {
			t.Fatalf("selector %d did not select a stub", slot)
		}
		want := "static-alpha"
		if slot%2 == 1 {
			want = "static-beta"
		}
		if selection.Stub.ID != want {
			t.Fatalf("selector %d selected %q, want %q", slot, selection.Stub.ID, want)
		}
	}
	for slot, err := range registrationErrors {
		if err != nil {
			t.Fatalf("dynamic registration %d (%s) failed: %v", slot, concurrentDynamicIDs[slot], err)
		}
	}
	for slot, err := range transitionErrors {
		if err != nil {
			t.Fatalf("transition %d failed: %v", slot, err)
		}
	}
	for slot, snapshot := range readerSlots {
		if snapshot.interactions > 128 {
			t.Fatalf("reader %d observed %d interactions, journal capacity is 128", slot, snapshot.interactions)
		}
		if snapshot.journalFull {
			t.Fatalf("reader %d observed a full journal, but only 72 admissions are possible", slot)
		}
		if snapshot.expectations > 1 {
			t.Fatalf("reader %d observed %d expectation failures, only one stub declares an expectation", slot, snapshot.expectations)
		}
	}

	stubIDs := make(map[string]bool)
	var previousIndex uint64
	for _, stub := range engineState.Registry.Stubs() {
		if stubIDs[stub.ID] {
			t.Fatalf("duplicate stub id %q survived concurrent registration", stub.ID)
		}
		stubIDs[stub.ID] = true
		if stub.RegistrationIndex <= previousIndex {
			t.Fatalf("stub %q has registration index %d after %d, want strictly increasing", stub.ID, stub.RegistrationIndex, previousIndex)
		}
		previousIndex = stub.RegistrationIndex
		if stub.Source != SourceStatic && stub.Source != SourceDynamic {
			t.Fatalf("stub %q has unknown source %q", stub.ID, stub.Source)
		}
		if stub.Sequence != nil && stub.SequencePosition > uint64(len(stub.Sequence)) {
			t.Fatalf("stub %q is at sequence position %d of %d", stub.ID, stub.SequencePosition, len(stub.Sequence))
		}
		if stub.SequencePosition > stub.InvocationCount {
			t.Fatalf("stub %q is at sequence position %d with invocation count %d", stub.ID, stub.SequencePosition, stub.InvocationCount)
		}
	}

	interactions := engineState.Interactions()
	if len(interactions) > 128 {
		t.Fatalf("retained %d interactions, journal capacity is 128", len(interactions))
	}
	legalOutcomes := map[string]bool{
		"matched": true, "unmatched": true, "sequence_exhausted": true, "invalid_stub_response": true,
	}
	legalStatuses := map[int]bool{0: true, 200: true, 409: true, 500: true}
	var previousSequence uint64
	for index, record := range interactions {
		if index > 0 && record.Sequence <= previousSequence {
			t.Fatalf("interaction sequence %d follows %d, want strictly increasing", record.Sequence, previousSequence)
		}
		previousSequence = record.Sequence
		if !legalOutcomes[record.Outcome] {
			t.Fatalf("interaction %d recorded outcome %q", record.Sequence, record.Outcome)
		}
		if !legalStatuses[record.ResponseStatus] {
			t.Fatalf("interaction %d recorded status %d", record.Sequence, record.ResponseStatus)
		}
	}

	// Envelope only: the journal holds 128 records and at most 72 admissions
	// happen here, so reserve can never report a full journal in this test and
	// neither this count nor the readers' JournalFull check above can fire unless
	// the capacity envelope itself breaks. The live journal-full proof is
	// TestConcurrentTransitionsRetainExactlyTheAdmittedSequences, which runs 256
	// callers against capacity 64.
	journalFullFailures := 0
	for _, failure := range engineState.VerificationFailures() {
		if failure.Code == "" || failure.Message == "" {
			t.Fatalf("recorded failure %#v has an empty code or message", failure)
		}
		if failure.Code == "journal_full" {
			journalFullFailures++
			if failure.RequestSequence != nil {
				t.Fatalf("journal_full failure carries request sequence %d, want none", *failure.RequestSequence)
			}
			continue
		}
		if failure.RequestSequence == nil {
			t.Fatalf("failure %q carries no request sequence", failure.Code)
		}
	}
	if journalFullFailures > 1 {
		t.Fatalf("recorded %d journal_full failures, want at most one", journalFullFailures)
	}

	for _, failure := range engineState.EvaluateExpectations() {
		if !stubIDs[failure.StubID] {
			t.Fatalf("expectation failure names unknown stub %q", failure.StubID)
		}
		switch failure.Code {
		case "expect_exactly", "expect_at_least", "expect_at_most":
		default:
			t.Fatalf("expectation failure %q has unknown code", failure.Code)
		}
	}

	// The final reset runs after every worker has joined, so its post-state is
	// deterministic: statics only, rewound counters, and empty derived state.
	engineState.Reset()
	stubs := engineState.Registry.Stubs()
	if len(stubs) != len(static) {
		t.Fatalf("reset left %d stubs, want %d static stubs", len(stubs), len(static))
	}
	for _, stub := range stubs {
		if stub.Source != SourceStatic {
			t.Fatalf("reset left %s stub %q", stub.Source, stub.ID)
		}
		if stub.InvocationCount != 0 || stub.SequencePosition != 0 {
			t.Fatalf("reset left stub %q at count %d, position %d", stub.ID, stub.InvocationCount, stub.SequencePosition)
		}
	}
	if interactions := engineState.Interactions(); len(interactions) != 0 {
		t.Fatalf("reset left %d interactions", len(interactions))
	}
	if failures := engineState.VerificationFailures(); len(failures) != 0 {
		t.Fatalf("reset left %d verification failures", len(failures))
	}
	if failures := engineState.EvaluateExpectations(); len(failures) != 0 {
		t.Fatalf("reset left expectation failures: %#v", failures)
	}
}

// TestConcurrentTransitionsRetainExactlyTheAdmittedSequences proves the
// journal retains exactly the admitted admission window under contention. The
// engine serializes reserve-and-append, so the retained records must be the
// first 64 sequences in order, the stub must show exactly 64 invocations, and
// the single journal-full report must carry no request sequence.
func TestConcurrentTransitionsRetainExactlyTheAdmittedSequences(t *testing.T) {
	const journalCapacity = 64
	engineState, err := NewEngineWithJournal([]Stub{{ID: "static", Profile: "profile"}}, journalCapacity)
	if err != nil {
		t.Fatalf("NewEngineWithJournal: %v", err)
	}

	const callers = 256
	selectionSlots := make([]*Selection, callers)
	start := make(chan struct{})
	var workers sync.WaitGroup
	workers.Add(callers)
	for slot := 0; slot < callers; slot++ {
		go func(slot int) {
			defer workers.Done()
			<-start
			selectionSlots[slot] = engineState.SelectInvocation(Exchange{Profile: "profile"})
		}(slot)
	}
	close(start)
	workers.Wait()

	admitted := 0
	for _, selection := range selectionSlots {
		if selection != nil {
			admitted++
		}
	}
	if admitted != journalCapacity {
		t.Fatalf("admitted %d of %d calls, want exactly %d", admitted, callers, journalCapacity)
	}

	interactions := engineState.Interactions()
	if len(interactions) != journalCapacity {
		t.Fatalf("retained %d interactions, want %d", len(interactions), journalCapacity)
	}
	for index, record := range interactions {
		if record.Sequence != uint64(index+1) {
			t.Fatalf("interaction %d has sequence %d, want %d", index, record.Sequence, index+1)
		}
		if !record.Matched || record.MatchedStubID != "static" || record.Outcome != "matched" {
			t.Fatalf("interaction %d = %#v, want a matched static record", index, record)
		}
	}

	stubs := engineState.Registry.Stubs()
	if len(stubs) != 1 {
		t.Fatalf("registry holds %d stubs, want 1", len(stubs))
	}
	if stubs[0].InvocationCount != journalCapacity {
		t.Fatalf("invocation count is %d, want %d", stubs[0].InvocationCount, journalCapacity)
	}

	failures := engineState.VerificationFailures()
	if len(failures) != 1 {
		t.Fatalf("recorded %d verification failures, want exactly one: %#v", len(failures), failures)
	}
	if failures[0].Code != "journal_full" || failures[0].RequestSequence != nil {
		t.Fatalf("journal-full failure = %#v", failures[0])
	}
}

// TestConcurrentDuplicateRegistrationAdmitsExactlyOneStub proves that a
// duplicate identifier can never be admitted twice: exactly one of the racing
// registrations succeeds, and the registry holds that one stub at index one.
func TestConcurrentDuplicateRegistrationAdmitsExactlyOneStub(t *testing.T) {
	engineState, err := NewEngineWithJournal(nil, 16)
	if err != nil {
		t.Fatalf("NewEngineWithJournal: %v", err)
	}

	const registrars = 32
	registrationErrors := make([]error, registrars)
	start := make(chan struct{})
	var workers sync.WaitGroup
	workers.Add(registrars)
	for slot := 0; slot < registrars; slot++ {
		go func(slot int) {
			defer workers.Done()
			<-start
			registrationErrors[slot] = engineState.RegisterDynamic(Stub{ID: "dynamic", Profile: "profile"})
		}(slot)
	}
	close(start)
	workers.Wait()

	admitted := 0
	for slot, err := range registrationErrors {
		if err == nil {
			admitted++
			continue
		}
		if err.Error() == "" {
			t.Fatalf("registration %d failed with an empty error", slot)
		}
	}
	if admitted != 1 {
		t.Fatalf("admitted %d duplicate registrations, want exactly one", admitted)
	}

	stubs := engineState.Registry.Stubs()
	if len(stubs) != 1 {
		t.Fatalf("registry holds %d stubs, want 1", len(stubs))
	}
	if stubs[0].ID != "dynamic" || stubs[0].Source != SourceDynamic || stubs[0].RegistrationIndex != 1 {
		t.Fatalf("admitted stub = %#v, want dynamic source at index 1", stubs[0])
	}
}

// TestConcurrentClearHistoryPreservesInvocationCounters proves that clearing
// request history never drops or duplicates a stub mutation: selection counts
// survive, every retained record is a matched record of the single static stub,
// and the retained window stays sequence ordered across interleaved clears.
func TestConcurrentClearHistoryPreservesInvocationCounters(t *testing.T) {
	const (
		selectors        = 32
		callsPerSelector = 8
		clearers         = 8
		clearsPerClearer = 8
		totalSelections  = selectors * callsPerSelector
		journalCapacity  = 4096
	)
	engineState, err := NewEngineWithJournal([]Stub{{ID: "static", Profile: "profile"}}, journalCapacity)
	if err != nil {
		t.Fatalf("NewEngineWithJournal: %v", err)
	}

	selectedSlots := make([]int, selectors)
	start := make(chan struct{})
	var workers sync.WaitGroup
	workers.Add(selectors + clearers)
	for slot := 0; slot < selectors; slot++ {
		go func(slot int) {
			defer workers.Done()
			<-start
			selected := 0
			for call := 0; call < callsPerSelector; call++ {
				if engineState.SelectInvocation(Exchange{Profile: "profile"}) != nil {
					selected++
				}
			}
			selectedSlots[slot] = selected
		}(slot)
	}
	for worker := 0; worker < clearers; worker++ {
		go func() {
			defer workers.Done()
			<-start
			for call := 0; call < clearsPerClearer; call++ {
				engineState.ClearRequestHistory()
			}
		}()
	}
	close(start)
	workers.Wait()

	selected := 0
	for _, count := range selectedSlots {
		selected += count
	}
	if selected != totalSelections {
		t.Fatalf("selected %d of %d exchanges, want all of them", selected, totalSelections)
	}

	stubs := engineState.Registry.Stubs()
	if len(stubs) != 1 {
		t.Fatalf("registry holds %d stubs, want 1", len(stubs))
	}
	if stubs[0].InvocationCount != totalSelections {
		t.Fatalf("invocation count is %d, want %d", stubs[0].InvocationCount, totalSelections)
	}
	if stubs[0].SequencePosition != 0 {
		t.Fatalf("sequence position is %d, want 0 for a stub without a sequence", stubs[0].SequencePosition)
	}

	interactions := engineState.Interactions()
	if len(interactions) > totalSelections {
		t.Fatalf("retained %d interactions, want at most %d", len(interactions), totalSelections)
	}
	for index, record := range interactions {
		if index > 0 && record.Sequence <= interactions[index-1].Sequence {
			t.Fatalf("interaction sequence %d follows %d, want strictly increasing", record.Sequence, interactions[index-1].Sequence)
		}
		if !record.Matched || record.MatchedStubID != "static" || record.Outcome != "matched" {
			t.Fatalf("interaction %d = %#v, want a matched static record", index, record)
		}
	}
	if failures := engineState.VerificationFailures(); len(failures) != 0 {
		t.Fatalf("clear left %d verification failures: %#v", len(failures), failures)
	}
}
