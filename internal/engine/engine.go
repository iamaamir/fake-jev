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

// Select finds the highest-priority matching rule for an exchange.
func (e *Engine) Select(exchange Exchange) *Stub {
	if e == nil || e.Registry == nil {
		return nil
	}
	return e.Registry.Select(exchange)
}

func nilRegistryError() error { return errNilRegistry }
