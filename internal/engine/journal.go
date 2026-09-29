package engine

import (
	"errors"
	"time"
)

const DefaultMaxInteractions = 10000

var (
	ErrJournalFull                  = errors.New("interaction journal full")
	ErrInteractionSequenceExhausted = errors.New("interaction sequence exhausted")
	errNilInteractionJournal        = errors.New("nil interaction journal")
)

// InteractionRequest contains the host-neutral information available before a
// data-plane transition is run. RawBody is copied when it is journaled.
type InteractionRequest struct {
	Profile   string
	Operation string
	RawBody   []byte
	Received  time.Time
}

// InteractionDecision is returned by the route/decode/validate portion of a
// transition. A valid decision is selected by the engine; invalid decisions
// are still journaled without mutating a stub.
type InteractionDecision struct {
	Exchange       Exchange
	Valid          bool
	Outcome        string
	ResponseStatus int
}

// InteractionRecord is the bounded, diagnostic interaction journal entry.
// Sequence positions are nil when no stub sequence applied.
type InteractionRecord struct {
	Sequence               uint64
	Profile                string
	Operation              string
	Received               time.Time
	Request                Exchange
	RawBody                []byte
	Matched                bool
	MatchedStubID          string
	ResponseStatus         int
	Outcome                string
	SequencePositionBefore *uint64
	SequencePositionAfter  *uint64
}

// VerificationFailure is a failure that is not necessarily representable as
// an interaction record. RequestSequence is nil for journal-full failures.
type VerificationFailure struct {
	Code            string
	Message         string
	RequestSequence *uint64
	StubID          *string
}

type interactionJournal struct {
	maxInteractions   int
	entries           []InteractionRecord
	nextSequence      uint64
	sequenceExhausted bool
	failures          []VerificationFailure
}

func newInteractionJournal(maxInteractions int) *interactionJournal {
	return &interactionJournal{maxInteractions: maxInteractions, nextSequence: 1}
}

func (j *interactionJournal) reserve() (uint64, error) {
	if j == nil || j.full() {
		if j != nil {
			j.setJournalFull()
		}
		return 0, ErrJournalFull
	}
	if j.sequenceExhausted {
		return 0, ErrInteractionSequenceExhausted
	}
	sequence := j.nextSequence
	if sequence == 0 {
		// Preserve one-based behavior for a zero-value journal.
		sequence = 1
	}
	if sequence == ^uint64(0) {
		j.sequenceExhausted = true
	} else {
		j.nextSequence = sequence + 1
	}
	return sequence, nil
}

func (j *interactionJournal) full() bool {
	return j == nil || len(j.entries) >= j.maxInteractions
}

func (j *interactionJournal) append(record InteractionRecord) {
	if j == nil || j.full() {
		return
	}
	j.entries = append(j.entries, record)
}

func (j *interactionJournal) setJournalFull() {
	if j == nil {
		return
	}
	for _, failure := range j.failures {
		if failure.Code == "journal_full" {
			return
		}
	}
	j.failures = append(j.failures, VerificationFailure{
		Code:    "journal_full",
		Message: "Interaction journal limit reached.",
	})
}

func (j *interactionJournal) clear() {
	if j == nil {
		return
	}
	j.entries = nil
	j.nextSequence = 1
	j.sequenceExhausted = false
	j.failures = nil
}

func (j *interactionJournal) records() []InteractionRecord {
	if j == nil {
		return nil
	}
	result := make([]InteractionRecord, len(j.entries))
	for i := range j.entries {
		result[i] = cloneInteractionRecord(j.entries[i])
	}
	return result
}

func (j *interactionJournal) verificationFailures() []VerificationFailure {
	if j == nil {
		return nil
	}
	result := make([]VerificationFailure, len(j.failures))
	for i := range j.failures {
		result[i] = cloneVerificationFailure(j.failures[i])
	}
	return result
}

func cloneVerificationFailure(failure VerificationFailure) VerificationFailure {
	if failure.RequestSequence != nil {
		sequence := *failure.RequestSequence
		failure.RequestSequence = &sequence
	}
	if failure.StubID != nil {
		stubID := *failure.StubID
		failure.StubID = &stubID
	}
	return failure
}

func cloneInteractionRecord(record InteractionRecord) InteractionRecord {
	record.Request = cloneExchange(record.Request)
	record.RawBody = append([]byte(nil), record.RawBody...)
	if record.SequencePositionBefore != nil {
		position := *record.SequencePositionBefore
		record.SequencePositionBefore = &position
	}
	if record.SequencePositionAfter != nil {
		position := *record.SequencePositionAfter
		record.SequencePositionAfter = &position
	}
	return record
}

func cloneExchange(exchange Exchange) Exchange {
	if exchange.Questions != nil {
		exchange.Questions = cloneStringMap(exchange.Questions)
	}
	if exchange.Metadata != nil {
		exchange.Metadata = make(map[string]Value, len(exchange.Metadata))
		for key, value := range exchange.Metadata {
			exchange.Metadata[key] = cloneValue(value)
		}
	}
	exchange.State = cloneValue(exchange.State)
	exchange.Payload = cloneValue(exchange.Payload)
	return exchange
}

func cloneStringMap(values map[string]string) map[string]string {
	result := make(map[string]string, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}
