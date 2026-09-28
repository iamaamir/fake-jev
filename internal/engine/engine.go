package engine

// Engine is the provider-neutral selection boundary. Compatibility profiles
// register rules and submit normalized exchanges; the engine returns only the
// selected deterministic action-bearing stub.
type Engine struct {
	Registry *Registry
}

// NewEngine creates an engine and registers static rules in their supplied
// order. A duplicate ID is rejected and leaves the later registration absent.
func NewEngine(static []Stub) (*Engine, error) {
	engine := &Engine{Registry: NewRegistry()}
	for _, stub := range static {
		if err := engine.Registry.RegisterStatic(stub); err != nil {
			return nil, err
		}
	}
	return engine, nil
}

// RegisterStatic registers a configuration rule.
func (e *Engine) RegisterStatic(stub Stub) error {
	if e == nil || e.Registry == nil {
		return nilRegistryError()
	}
	return e.Registry.RegisterStatic(stub)
}

// RegisterDynamic registers a successfully accepted dynamic rule.
func (e *Engine) RegisterDynamic(stub Stub) error {
	if e == nil || e.Registry == nil {
		return nilRegistryError()
	}
	return e.Registry.RegisterDynamic(stub)
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
// and consumes its next response action. No state changes occur for an
// unmatched exchange.
func (e *Engine) SelectInvocation(exchange Exchange) *Selection {
	if e == nil || e.Registry == nil {
		return nil
	}
	return e.Registry.SelectInvocation(exchange)
}

// EvaluateExpectations evaluates all registered invocation expectations in
// registration order.
func (e *Engine) EvaluateExpectations() []ExpectationFailure {
	if e == nil || e.Registry == nil {
		return nil
	}
	return e.Registry.EvaluateExpectations()
}

func nilRegistryError() error { return errNilRegistry }
