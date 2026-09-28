package engine

import "fmt"

// InvocationExpectation describes the allowed number of times a stub may be
// selected. Exactly is mutually exclusive with the range bounds; nil means a
// bound was not supplied.
type InvocationExpectation struct {
	Exactly *uint64
	AtLeast *uint64
	AtMost  *uint64
}

// NewExactExpectation creates an exact invocation expectation.
func NewExactExpectation(count uint64) *InvocationExpectation {
	return &InvocationExpectation{Exactly: uint64Pointer(count)}
}

// NewRangeExpectation creates an invocation expectation with optional bounds.
// At least one bound must be non-nil.
func NewRangeExpectation(atLeast, atMost *uint64) *InvocationExpectation {
	return &InvocationExpectation{AtLeast: atLeast, AtMost: atMost}
}

// Validate checks the shape required by Section 12.7.
func (e InvocationExpectation) Validate() error {
	if e.Exactly != nil && (e.AtLeast != nil || e.AtMost != nil) {
		return fmt.Errorf("exactly cannot be combined with atLeast or atMost")
	}
	if e.Exactly == nil && e.AtLeast == nil && e.AtMost == nil {
		return fmt.Errorf("expectation requires exactly, atLeast, or atMost")
	}
	if e.AtLeast != nil && e.AtMost != nil && *e.AtLeast > *e.AtMost {
		return fmt.Errorf("atLeast cannot exceed atMost")
	}
	return nil
}

func uint64Pointer(value uint64) *uint64 { return &value }

// Selection is the provider-neutral result of selecting a stub. Action is the
// selected response action, or nil when the sequence is exhausted. The
// compatibility profile maps SequenceExhausted to its wire-level failure.
type Selection struct {
	Stub              Stub
	Action            ResponseAction
	SequenceExhausted bool
}

// asStub preserves the legacy selection API while applying the action chosen
// during the same registry transition.
func (s Selection) asStub() *Stub {
	stub := s.Stub
	stub.Response = s.Action
	return &stub
}

// ExpectationFailure is an engine-level verification failure. Compatibility
// layers can add their interaction sequence and wire representation around it.
type ExpectationFailure struct {
	Code            string
	Message         string
	StubID          string
	InvocationCount uint64
}

// selectInvocation performs matching, mutation, and sequence consumption as
// one registry transition. The invocation count is incremented before action
// selection, including when the sequence is already exhausted.
func (r *Registry) selectInvocation(exchange Exchange) *Selection {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	candidates := r.matching(exchange)
	if len(candidates) == 0 {
		return nil
	}
	selected := candidates[0]
	for i := range r.stubs {
		stub := &r.stubs[i]
		if stub.ID == selected.ID && stub.RegistrationIndex == selected.RegistrationIndex {
			stub.InvocationCount++
			result := Selection{Action: stub.Response}
			if stub.Sequence != nil {
				if stub.SequencePosition >= uint64(len(stub.Sequence)) {
					result.SequenceExhausted = true
				} else {
					result.Action = stub.Sequence[stub.SequencePosition]
					stub.SequencePosition++
				}
			}
			result.Stub = cloneStub(*stub)
			return &result
		}
	}
	return nil
}

// SelectInvocation selects a matching stub and consumes one response action.
// An unmatched exchange returns nil and changes no stub state.
func (r *Registry) SelectInvocation(exchange Exchange) *Selection {
	return r.selectInvocation(exchange)
}

// EvaluateExpectations returns expectation failures in registration order.
func (r *Registry) EvaluateExpectations() []ExpectationFailure {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	failures := make([]ExpectationFailure, 0)
	for _, stub := range r.stubs {
		if stub.Expect == nil {
			continue
		}
		count := stub.InvocationCount
		expectation := *stub.Expect
		if expectation.Exactly != nil && count != *expectation.Exactly {
			failures = append(failures, expectationFailure("expect_exactly", stub))
			continue
		}
		if expectation.AtLeast != nil && count < *expectation.AtLeast {
			failures = append(failures, expectationFailure("expect_at_least", stub))
		}
		if expectation.AtMost != nil && count > *expectation.AtMost {
			failures = append(failures, expectationFailure("expect_at_most", stub))
		}
	}
	return failures
}

func expectationFailure(code string, stub Stub) ExpectationFailure {
	return ExpectationFailure{
		Code:            code,
		Message:         fmt.Sprintf("stub %q invocation expectation was not satisfied", stub.ID),
		StubID:          stub.ID,
		InvocationCount: stub.InvocationCount,
	}
}
