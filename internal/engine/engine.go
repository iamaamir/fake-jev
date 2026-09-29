package engine

import (
	"errors"
	"sync"
	"time"
)

// Engine is the provider-neutral selection boundary. Compatibility profiles
// register rules and submit normalized exchanges; the engine returns only the
// selected deterministic action-bearing stub.
type Engine struct {
	Registry *Registry

	mu      sync.Mutex
	journal *interactionJournal
}

var errNilTransitionHandler = errors.New("nil transition handler")

// NewEngine creates an engine and registers static rules in their supplied
// order. A duplicate ID is rejected and leaves the later registration absent.
func NewEngine(static []Stub) (*Engine, error) {
	return NewEngineWithJournal(static, DefaultMaxInteractions)
}

// NewEngineWithJournal creates an engine with a bounded interaction journal.
// A zero capacity is valid and causes every data-plane admission to return
// ErrJournalFull without assigning a sequence or mutating registry state.
func NewEngineWithJournal(static []Stub, maxInteractions int) (*Engine, error) {
	if maxInteractions < 0 {
		maxInteractions = 0
	}
	engine := &Engine{Registry: NewRegistry(), journal: newInteractionJournal(maxInteractions)}
	for _, stub := range static {
		if err := engine.Registry.RegisterStatic(stub); err != nil {
			return nil, err
		}
	}
	return engine, nil
}

// NewEngineWithCapacity is an explicit alias for NewEngineWithJournal.
func NewEngineWithCapacity(static []Stub, maxInteractions int) (*Engine, error) {
	return NewEngineWithJournal(static, maxInteractions)
}

// RegisterStatic registers a configuration rule atomically with data-plane
// transitions.
func (e *Engine) RegisterStatic(stub Stub) error {
	if e == nil || e.Registry == nil {
		return nilRegistryError()
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.Registry.RegisterStatic(stub)
}

// RegisterDynamic registers a successfully accepted dynamic rule atomically
// with data-plane transitions.
func (e *Engine) RegisterDynamic(stub Stub) error {
	if e == nil || e.Registry == nil {
		return nilRegistryError()
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.Registry.RegisterDynamic(stub)
}

// RemoveDynamic removes runtime stubs atomically with data-plane transitions.
func (e *Engine) RemoveDynamic() {
	if e == nil || e.Registry == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.Registry.RemoveDynamic()
}

// Select selects the highest-priority matching rule and consumes one
// invocation. The returned stub contains the selected action in Response;
// callers that need to distinguish sequence exhaustion should use
// SelectInvocation.
func (e *Engine) Select(exchange Exchange) *Stub {
	selection := e.SelectInvocation(exchange)
	if selection == nil {
		return nil
	}
	return selection.asStub()
}

// SelectInvocation selects a matching rule, increments its invocation count,
// consumes its next response action, and records the normalized interaction.
// An unmatched exchange is also admitted and recorded. Call Transition when
// routing or validation must happen after sequence assignment.
func (e *Engine) SelectInvocation(exchange Exchange) *Selection {
	if e == nil || e.Registry == nil || e.journal == nil {
		return nil
	}
	var selection *Selection
	_, err := e.transition(InteractionRequest{Profile: exchange.Profile, Operation: exchange.Operation}, func(uint64) (InteractionDecision, *Selection) {
		selection = e.Registry.SelectInvocation(exchange)
		decision := InteractionDecision{Exchange: exchange, Valid: true}
		if selection == nil {
			decision.Outcome = "unmatched"
		} else {
			decision.Outcome = "matched"
		}
		return decision, selection
	})
	if err != nil {
		return nil
	}
	return selection
}

// Transition serializes admission, route/decode/validation, selection,
// mutation, and journaling as one state transition. The handler runs after a
// sequence has been reserved and before the transition is recorded. It must
// return Valid=false for malformed or otherwise rejected requests; such a
// request is still recorded and does not select a stub.
func (e *Engine) Transition(request InteractionRequest, handler func(sequence uint64) InteractionDecision) (*InteractionRecord, *Selection, error) {
	if e == nil || e.Registry == nil {
		return nil, nil, nilRegistryError()
	}
	if e.journal == nil {
		return nil, nil, errNilInteractionJournal
	}
	if handler == nil {
		return nil, nil, errNilTransitionHandler
	}
	var selection *Selection
	record, err := e.transition(request, func(sequence uint64) (InteractionDecision, *Selection) {
		decision := handler(sequence)
		if decision.Valid {
			selection = e.Registry.SelectInvocation(decision.Exchange)
			if selection == nil {
				if decision.Outcome == "" {
					decision.Outcome = "unmatched"
				}
			} else {
				decision.Outcome = "matched"
				if decision.ResponseStatus == 0 {
					decision.ResponseStatus = 200
				}
			}
		}
		return decision, selection
	})
	return record, selection, err
}

func (e *Engine) transition(request InteractionRequest, handler func(uint64) (InteractionDecision, *Selection)) (*InteractionRecord, error) {
	if e == nil || e.journal == nil {
		return nil, errNilInteractionJournal
	}
	if handler == nil {
		return nil, errNilTransitionHandler
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	sequence, err := e.journal.reserve()
	if err != nil {
		return nil, err
	}
	decision, selection := handler(sequence)
	if selection != nil && selection.SequenceExhausted {
		decision.Outcome = "sequence_exhausted"
		decision.ResponseStatus = 409
		if decision.Failure == nil {
			stubID := selection.Stub.ID
			decision.Failure = &VerificationFailure{
				Code:    "sequence_exhausted",
				Message: "Configured response sequence is exhausted.",
				StubID:  &stubID,
			}
		}
	}
	if decision.Outcome == "" {
		if decision.Valid {
			decision.Outcome = "unmatched"
		} else {
			decision.Outcome = "invalid"
		}
	}
	recordedRequest := decision.Exchange
	if recordedRequest.Profile == "" {
		recordedRequest.Profile = request.Profile
	}
	if recordedRequest.Operation == "" {
		recordedRequest.Operation = request.Operation
	}
	profile := request.Profile
	if profile == "" {
		profile = recordedRequest.Profile
	}
	operation := request.Operation
	if operation == "" {
		operation = recordedRequest.Operation
	}
	record := InteractionRecord{
		Sequence:       sequence,
		Profile:        profile,
		Operation:      operation,
		Received:       request.Received,
		Request:        cloneExchange(recordedRequest),
		RawBody:        append([]byte(nil), request.RawBody...),
		ResponseStatus: decision.ResponseStatus,
		Outcome:        decision.Outcome,
	}
	if record.Received.IsZero() {
		// This is diagnostic data only and is never used by matching or output.
		record.Received = time.Now()
	}
	if selection != nil {
		record.Matched = true
		record.MatchedStubID = selection.Stub.ID
		if selection.Stub.Sequence != nil {
			before := selection.Stub.SequencePosition
			if !selection.SequenceExhausted && before > 0 {
				before--
			}
			record.SequencePositionBefore = uint64Pointer(before)
			record.SequencePositionAfter = uint64Pointer(selection.Stub.SequencePosition)
		}
	}
	e.journal.append(record)
	if decision.Failure != nil {
		failure := cloneVerificationFailure(*decision.Failure)
		if failure.RequestSequence == nil {
			failure.RequestSequence = uint64Pointer(sequence)
		}
		e.journal.addFailure(failure)
	}
	return &record, nil
}

// UpdateInteraction completes diagnostic fields that are only known after the
// compatibility profile has encoded the selected response.
func (e *Engine) UpdateInteraction(sequence uint64, outcome string, status int) {
	if e == nil || e.journal == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.journal.update(sequence, outcome, status)
}

// RecordFailure records a fake-server failure discovered after a transition,
// such as an invalid configured response. The interaction remains retained
// while its outcome and status are updated atomically with the failure state.
func (e *Engine) RecordFailure(sequence uint64, code, message, stubID string, status int) {
	if e == nil || e.journal == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	outcome := code
	if len("fake_jev_") <= len(outcome) && outcome[:len("fake_jev_")] == "fake_jev_" {
		outcome = outcome[len("fake_jev_"):]
	}
	e.journal.update(sequence, outcome, status)
	failure := VerificationFailure{Code: outcome, Message: message, RequestSequence: uint64Pointer(sequence)}
	if stubID != "" {
		failure.StubID = &stubID
	}
	e.journal.addFailure(failure)
}

// ClearRequestHistory atomically clears retained interactions and all
// interaction-derived failures while preserving stub counters and sequences.
func (e *Engine) ClearRequestHistory() {
	if e == nil || e.journal == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.journal.clear()
}

// ClearHistory is an alias for ClearRequestHistory.
func (e *Engine) ClearHistory() { e.ClearRequestHistory() }

// ClearRequests is an alias for ClearRequestHistory.
func (e *Engine) ClearRequests() { e.ClearRequestHistory() }

// Reset atomically restores all engine state to its post-construction state:
// static counters and sequences are rewound, dynamic stubs are removed, and
// registration numbering resumes immediately after the static stubs.
func (e *Engine) Reset() {
	if e == nil || e.Registry == nil || e.journal == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.Registry.mu.Lock()
	staticCount := 0
	stubs := make([]Stub, 0, len(e.Registry.stubs))
	for _, stub := range e.Registry.stubs {
		if stub.Source == SourceDynamic {
			continue
		}
		stub.InvocationCount = 0
		stub.SequencePosition = 0
		stubs = append(stubs, stub)
		staticCount++
	}
	e.Registry.stubs = stubs
	e.Registry.nextRegistrationIndex = uint64(staticCount + 1)
	e.Registry.registrationIndexExhausted = false
	e.Registry.mu.Unlock()
	e.journal.clear()
}

// FullReset is an alias for Reset.
func (e *Engine) FullReset() { e.Reset() }

// Interactions returns a detached sequence-ordered copy of the journal.
func (e *Engine) Interactions() []InteractionRecord {
	if e == nil || e.journal == nil {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.journal.records()
}

// Journal returns a detached copy of the retained interactions.
func (e *Engine) Journal() []InteractionRecord { return e.Interactions() }

// VerificationFailures returns sticky interaction-derived failures. The
// journal-full failure remains until ClearRequestHistory or Reset.
func (e *Engine) VerificationFailures() []VerificationFailure {
	if e == nil || e.journal == nil {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.journal.verificationFailures()
}

// EvaluateExpectations evaluates all registered invocation expectations in
// registration order.
func (e *Engine) EvaluateExpectations() []ExpectationFailure {
	if e == nil || e.Registry == nil {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.Registry.EvaluateExpectations()
}

func (e *Engine) JournalFull() bool {
	if e == nil || e.journal == nil {
		return false
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.journal.full()
}

func nilRegistryError() error { return errNilRegistry }
