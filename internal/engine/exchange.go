// Package engine contains provider-neutral deterministic matching primitives.
package engine

// Value is a JSON-compatible value tree. Compatibility profiles normalize their
// wire representations into Values before passing them to the engine.
type Value any

// Exchange is a normalized provider exchange. The engine does not interpret
// provider wire formats; profiles populate these generic matching inputs.
type Exchange struct {
	Profile   string
	Operation string
	Model     string
	Questions map[string]string
	State     Value
	Payload   Value
	Metadata  map[string]Value
}

// NewExchange constructs an exchange with the supplied normalized inputs.
// Maps are retained as supplied so callers may choose their own ownership
// policy for payloads and metadata.
func NewExchange(profile, operation string) Exchange {
	return Exchange{
		Profile:   profile,
		Operation: operation,
		Questions: make(map[string]string),
		Metadata:  make(map[string]Value),
	}
}
