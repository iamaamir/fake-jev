package engine

import (
	"fmt"
	"sync"
)

var (
	errNilRegistry                = fmt.Errorf("nil registry")
	errRegistrationIndexExhausted = fmt.Errorf("registration index exhausted")
)

// StubSource records how a stub entered the registry. Source is provenance,
// not an additional precedence rule.
type StubSource string

const (
	SourceStatic  StubSource = "static"
	SourceDynamic StubSource = "dynamic"
)

// ResponseAction is deliberately opaque to the engine. A compatibility
// profile supplies and consumes the action; matching only selects its stub.
type ResponseAction any

// Stub is a provider-neutral deterministic rule.
type Stub struct {
	ID                string
	Profile           string
	Priority          int
	Matcher           Matcher
	Response          ResponseAction
	RegistrationIndex uint64
	Source            StubSource
}

// Registry stores static and dynamic rules and assigns one registration index
// to each successful registration. Indices start at one, matching the
// externally visible registration contract.
type Registry struct {
	mu                         sync.RWMutex
	stubs                      []Stub
	nextRegistrationIndex      uint64
	registrationIndexExhausted bool
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{nextRegistrationIndex: 1}
}

// RegisterStatic appends a configuration stub in static registration order.
func (r *Registry) RegisterStatic(stub Stub) error {
	return r.register(stub, SourceStatic)
}

// RegisterDynamic appends a runtime stub in successful registration order.
// Failed registrations do not consume an index.
func (r *Registry) RegisterDynamic(stub Stub) error {
	return r.register(stub, SourceDynamic)
}

func (r *Registry) register(stub Stub, source StubSource) error {
	if r == nil {
		return errNilRegistry
	}
	if stub.ID == "" {
		return fmt.Errorf("stub id must not be empty")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.registrationIndexExhausted {
		return errRegistrationIndexExhausted
	}
	for _, existing := range r.stubs {
		if existing.ID == stub.ID {
			return fmt.Errorf("duplicate stub id %q", stub.ID)
		}
	}
	if stub.Profile == "" {
		return fmt.Errorf("stub profile must not be empty")
	}
	if r.nextRegistrationIndex == 0 {
		// Preserve the documented one-based contract for a zero-value Registry.
		r.nextRegistrationIndex = 1
	}
	stub.Source = source
	stub.RegistrationIndex = r.nextRegistrationIndex
	if r.nextRegistrationIndex == ^uint64(0) {
		r.registrationIndexExhausted = true
	} else {
		r.nextRegistrationIndex++
	}
	stub = cloneStub(stub)
	r.stubs = append(r.stubs, stub)
	return nil
}

// Register is a convenience for callers that already know a stub's source.
// Registration indices are always assigned by the registry; caller-provided
// indices never affect ordering.
func (r *Registry) Register(stub Stub, source StubSource) error {
	switch source {
	case SourceStatic:
		return r.RegisterStatic(stub)
	case SourceDynamic:
		return r.RegisterDynamic(stub)
	default:
		return fmt.Errorf("unknown stub source %q", source)
	}
}

// Stubs returns all registered stubs in registration order.
func (r *Registry) Stubs() []Stub {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Stub, len(r.stubs))
	for i, stub := range r.stubs {
		result[i] = cloneStub(stub)
	}
	return result
}

// RemoveDynamic removes all dynamic stubs without changing the static
// registration indices or the next index.
func (r *Registry) RemoveDynamic() {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	kept := make([]Stub, 0, len(r.stubs))
	for _, stub := range r.stubs {
		if stub.Source != SourceDynamic {
			kept = append(kept, stub)
		}
	}
	r.stubs = kept
}
