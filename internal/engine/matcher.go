package engine

import (
	"bytes"
	"encoding/json"
	"io"
	"math/big"
	"sort"
	"strconv"
	"strings"
)

// Matcher contains only normalized, optional matching fields. A nil Questions
// map is omitted (wildcard); a non-nil empty map matches an exchange with no
// questions. Model and State use explicit presence bits so empty strings and
// JSON null remain representable values.
type Matcher struct {
	Operation string
	Model     string
	HasModel  bool
	Questions map[string]string
	State     Value
	HasState  bool
}

// NewMatcher creates a matcher with all optional fields omitted.
func NewMatcher() Matcher { return Matcher{} }

// WithModel returns a matcher requiring exact model equality.
func (m Matcher) WithModel(model string) Matcher {
	m.Model = model
	m.HasModel = true
	return m
}

// WithState returns a matcher requiring JSON-value equality for state.
func (m Matcher) WithState(state Value) Matcher {
	m.State = state
	m.HasState = true
	return m
}

// Matches reports whether every field present in m matches e. Omitted fields
// are wildcards and all present fields are combined with logical AND.
func (m Matcher) Matches(e Exchange) bool {
	if m.Operation != "" && m.Operation != e.Operation {
		return false
	}
	if (m.HasModel || m.Model != "") && m.Model != e.Model {
		return false
	}
	if m.Questions != nil && !questionTypesEqual(m.Questions, e.Questions) {
		return false
	}
	if m.HasState || valuePresent(m.State) {
		return jsonValuesEqual(m.State, e.State)
	}
	return true
}

func questionTypesEqual(want, got map[string]string) bool {
	if len(want) != len(got) {
		return false
	}
	for name, typ := range want {
		if got[name] != typ {
			return false
		}
	}
	return true
}

// Select returns the first matching stub in the total order required by the
// specification and commits one invocation. The returned Response is the
// selected action; use SelectInvocation when sequence exhaustion must be
// distinguished from an ordinary nil action.
func (r *Registry) Select(exchange Exchange) *Stub {
	selection := r.SelectInvocation(exchange)
	if selection == nil {
		return nil
	}
	return selection.asStub()
}

// Matching returns all matching candidates in selection order. It is useful to
// compatibility layers that need diagnostics while preserving one ordering
// implementation.
func (r *Registry) Matching(exchange Exchange) []Stub {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.matching(exchange)
}

func (r *Registry) matching(exchange Exchange) []Stub {
	candidates := make([]Stub, 0, len(r.stubs))
	for _, stub := range r.stubs {
		if stub.Profile == exchange.Profile && stub.Matcher.Matches(exchange) {
			// Return detached matcher data so callers cannot mutate registry state
			// through Matching or the pointer returned by Select.
			candidates = append(candidates, cloneStub(stub))
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Priority != candidates[j].Priority {
			return candidates[i].Priority > candidates[j].Priority
		}
		return candidates[i].RegistrationIndex < candidates[j].RegistrationIndex
	})
	return candidates
}

func valuePresent(value Value) bool {
	switch value := value.(type) {
	case json.RawMessage:
		return len(value) > 0
	case []byte:
		return len(value) > 0
	default:
		return value != nil
	}
}

func cloneStub(stub Stub) Stub {
	stub.Matcher = cloneMatcher(stub.Matcher)
	if stub.Sequence != nil {
		sequence := make([]ResponseAction, len(stub.Sequence))
		copy(sequence, stub.Sequence)
		stub.Sequence = sequence
	}
	if stub.Expect != nil {
		expectation := *stub.Expect
		if expectation.Exactly != nil {
			value := *expectation.Exactly
			expectation.Exactly = &value
		}
		if expectation.AtLeast != nil {
			value := *expectation.AtLeast
			expectation.AtLeast = &value
		}
		if expectation.AtMost != nil {
			value := *expectation.AtMost
			expectation.AtMost = &value
		}
		stub.Expect = &expectation
	}
	return stub
}

func cloneMatcher(m Matcher) Matcher {
	if m.Questions != nil {
		questions := make(map[string]string, len(m.Questions))
		for name, typ := range m.Questions {
			questions[name] = typ
		}
		m.Questions = questions
	}
	m.State = cloneValue(m.State)
	return m
}

func cloneValue(value Value) Value {
	switch raw := value.(type) {
	case json.RawMessage:
		return append(json.RawMessage(nil), raw...)
	case []byte:
		return append([]byte(nil), raw...)
	}
	decoded, ok := decodeJSONValue(value)
	if !ok {
		return value
	}
	return decoded
}

func jsonValuesEqual(left, right Value) bool {
	l, lok := decodeJSONValue(left)
	r, rok := decodeJSONValue(right)
	return lok && rok && equalJSONValue(l, r)
}

func decodeJSONValue(value Value) (any, bool) {
	switch raw := value.(type) {
	case json.RawMessage:
		return decodeRawJSON(raw)
	case []byte:
		return decodeRawJSON(raw)
	}
	// Marshal first so all supported JSON-compatible containers are copied and
	// normalized consistently. UseNumber in decodeRawJSON preserves precision.
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, false
	}
	return decodeRawJSON(raw)
}

func decodeRawJSON(raw []byte) (any, bool) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, false
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, false
	}
	return value, true
}

func equalJSONValue(left, right any) bool {
	if leftNumber, ok := jsonNumber(left); ok {
		rightNumber, rightOK := jsonNumber(right)
		return rightOK && leftNumber.Cmp(rightNumber) == 0
	}
	if _, ok := jsonNumber(right); ok {
		return false
	}
	switch left := left.(type) {
	case nil:
		return right == nil
	case string:
		right, ok := right.(string)
		return ok && left == right
	case bool:
		right, ok := right.(bool)
		return ok && left == right
	case []any:
		right, ok := right.([]any)
		if !ok || len(left) != len(right) {
			return false
		}
		for i := range left {
			if !equalJSONValue(left[i], right[i]) {
				return false
			}
		}
		return true
	case map[string]any:
		right, ok := right.(map[string]any)
		if !ok || len(left) != len(right) {
			return false
		}
		for key, item := range left {
			other, exists := right[key]
			if !exists || !equalJSONValue(item, other) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func jsonNumber(value any) (*big.Rat, bool) {
	var text string
	switch value := value.(type) {
	case json.Number:
		text = string(value)
	case int:
		text = formatInt(int64(value))
	case int8:
		text = formatInt(int64(value))
	case int16:
		text = formatInt(int64(value))
	case int32:
		text = formatInt(int64(value))
	case int64:
		text = formatInt(value)
	case uint:
		text = formatUint(uint64(value))
	case uint8:
		text = formatUint(uint64(value))
	case uint16:
		text = formatUint(uint64(value))
	case uint32:
		text = formatUint(uint64(value))
	case uint64:
		text = formatUint(value)
	case float32:
		text = formatFloat(float64(value))
	case float64:
		text = formatFloat(value)
	default:
		return nil, false
	}
	return decimalRat(text)
}

func decimalRat(text string) (*big.Rat, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, false
	}
	negative := false
	if text[0] == '+' || text[0] == '-' {
		negative = text[0] == '-'
		text = text[1:]
	}
	exponent := 0
	if index := strings.IndexAny(text, "eE"); index >= 0 {
		parsed, ok := parseExponent(text[index+1:])
		if !ok {
			return nil, false
		}
		exponent = parsed
		text = text[:index]
	}
	point := strings.IndexByte(text, '.')
	fractionDigits := 0
	if point >= 0 {
		fractionDigits = len(text) - point - 1
		text = text[:point] + text[point+1:]
	}
	if text == "" {
		return nil, false
	}
	for _, digit := range text {
		if digit < '0' || digit > '9' {
			return nil, false
		}
	}
	coefficient := new(big.Int)
	if _, ok := coefficient.SetString(text, 10); !ok {
		return nil, false
	}
	if negative {
		coefficient.Neg(coefficient)
	}
	rat := new(big.Rat).SetInt(coefficient)
	shift := exponent - fractionDigits
	if shift >= 0 {
		rat.Mul(rat, new(big.Rat).SetInt(pow10(shift)))
	} else {
		rat.Quo(rat, new(big.Rat).SetInt(pow10(-shift)))
	}
	return rat, true
}

func parseExponent(text string) (int, bool) {
	if text == "" {
		return 0, false
	}
	negative := false
	if text[0] == '+' || text[0] == '-' {
		negative = text[0] == '-'
		text = text[1:]
	}
	if text == "" {
		return 0, false
	}
	value := 0
	for _, digit := range text {
		if digit < '0' || digit > '9' {
			return 0, false
		}
		digitValue := int(digit - '0')
		if value > (100000-digitValue)/10 {
			return 0, false
		}
		value = value*10 + digitValue
	}
	if negative {
		value = -value
	}
	return value, true
}

func pow10(power int) *big.Int {
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(power)), nil)
}

func formatInt(value int64) string   { return new(big.Int).SetInt64(value).String() }
func formatUint(value uint64) string { return new(big.Int).SetUint64(value).String() }
func formatFloat(value float64) string {
	// Values decoded from JSON use json.Number. This path only handles callers
	// that already supplied a binary float, preserving its shortest decimal form.
	return strconv.FormatFloat(value, 'g', -1, 64)
}
