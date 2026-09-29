package control

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"fake-jev/internal/config"
	"fake-jev/internal/engine"
)

var errRegisteredStubMissing = errors.New("registered stub disappeared outside control mutation boundary")

// Engine-exchange metadata keys recorded by the HTTP host on every data-plane
// interaction. The provider-neutral journal carries no HTTP details (§8.6), so
// §41.7's method and path travel through Exchange.Metadata. internal/host/http
// writes the same keys.
const (
	requestMethodMetadataKey = "method"
	requestTargetMetadataKey = "target"
)

func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r == nil || r.URL == nil || !strings.HasPrefix(r.URL.Path, "/__fake/") {
		writeJSON(w, http.StatusNotFound, controlError{"fake_jev_control_not_found", "Unknown control endpoint.", ""})
		return
	}
	if r.Body != nil {
		limit := a.metadata.Limits.ControlPlaneBodyBytes
		if limit == 0 {
			limit = 2097152
		}
		r.Body = http.MaxBytesReader(w, r.Body, int64(limit))
	}
	if r.URL.Path == "/__fake/v1/health" {
		if r.Method != http.MethodGet {
			a.methodNotAllowed(w)
			return
		}
		a.health(w)
		return
	}
	if r.URL.Path == "/__fake/v1/meta" {
		if r.Method != http.MethodGet {
			a.methodNotAllowed(w)
			return
		}
		a.meta(w)
		return
	}
	if r.URL.Path == "/__fake/v1/stubs" {
		switch r.Method {
		case http.MethodGet:
			a.listStubs(w)
		case http.MethodPost:
			a.createStub(w, r)
		case http.MethodDelete:
			a.clearStubs(w)
		default:
			a.methodNotAllowed(w)
		}
		return
	}
	if r.URL.Path == "/__fake/v1/requests" {
		switch r.Method {
		case http.MethodGet:
			a.listRequests(w)
		case http.MethodDelete:
			a.clearRequests(w)
		default:
			a.methodNotAllowed(w)
		}
		return
	}
	if r.URL.Path == "/__fake/v1/verify" {
		if r.Method != http.MethodGet {
			a.methodNotAllowed(w)
			return
		}
		a.verify(w)
		return
	}
	if r.URL.Path == "/__fake/v1/reset" {
		if r.Method != http.MethodPost {
			a.methodNotAllowed(w)
			return
		}
		a.reset(w)
		return
	}
	writeJSON(w, http.StatusNotFound, controlError{"fake_jev_control_not_found", "Unknown control endpoint.", ""})
}

func (a *API) health(w http.ResponseWriter) {
	writeJSON(w, http.StatusOK, struct {
		Status            string `json:"status"`
		ServerVersion     string `json:"serverVersion"`
		ControlAPIVersion string `json:"controlApiVersion"`
	}{"ok", a.metadata.ServerVersion, ControlAPIVersion})
}

func (a *API) meta(w http.ResponseWriter) {
	profiles := append([]string(nil), a.metadata.ActiveProfiles...)
	writeJSON(w, http.StatusOK, struct {
		ServerVersion       string   `json:"serverVersion"`
		ControlAPIVersion   string   `json:"controlApiVersion"`
		ConfigSchemaVersion int      `json:"configSchemaVersion"`
		Mode                string   `json:"mode"`
		ActiveProfiles      []string `json:"activeProfiles"`
		Limits              Limits   `json:"limits"`
	}{a.metadata.ServerVersion, ControlAPIVersion, a.metadata.ConfigSchemaVersion, a.metadata.Mode, profiles, a.metadata.Limits})
}

type stubItem struct {
	ID                string            `json:"id"`
	Profile           string            `json:"profile"`
	Priority          int               `json:"priority"`
	Source            engine.StubSource `json:"source"`
	RegistrationIndex uint64            `json:"registrationIndex"`
	Invocations       uint64            `json:"invocations"`
}

func (a *API) listStubs(w http.ResponseWriter) {
	items := make([]stubItem, 0)
	if a.engine != nil && a.engine.Registry != nil {
		stubs := a.engine.Registry.Stubs()
		items = make([]stubItem, 0, len(stubs))
		for _, stub := range stubs {
			items = append(items, stubItem{ID: stub.ID, Profile: stub.Profile, Priority: stub.Priority, Source: stub.Source, RegistrationIndex: stub.RegistrationIndex, Invocations: stub.InvocationCount})
		}
	}
	writeJSON(w, http.StatusOK, struct {
		Stubs []stubItem `json:"stubs"`
	}{items})
}

func (a *API) clearStubs(w http.ResponseWriter) {
	stubMutations.Lock()
	if a.engine != nil {
		a.engine.RemoveDynamic()
	}
	stubMutations.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

// requestRecord is the exact §41.7 interaction record shape. Nullable string
// members use pointers so absence renders as JSON null rather than "".
type requestRecord struct {
	Sequence      uint64  `json:"sequence"`
	Timestamp     string  `json:"timestamp"`
	Method        string  `json:"method"`
	Path          string  `json:"path"`
	Profile       *string `json:"profile"`
	Operation     *string `json:"operation"`
	Outcome       string  `json:"outcome"`
	MatchedStubID *string `json:"matchedStubId"`
	Status        int     `json:"status"`
	Error         *string `json:"error"`
	RequestBody   any     `json:"requestBody"`
}

func (a *API) listRequests(w http.ResponseWriter) {
	records := a.engine.Interactions()
	items := make([]requestRecord, 0, len(records))
	for _, record := range records {
		items = append(items, newRequestRecord(record))
	}
	writeJSON(w, http.StatusOK, struct {
		Requests []requestRecord `json:"requests"`
	}{items})
}

func newRequestRecord(record engine.InteractionRecord) requestRecord {
	item := requestRecord{
		Sequence:    record.Sequence,
		Timestamp:   record.Received.UTC().Format(time.RFC3339Nano),
		Method:      metadataString(record.Request.Metadata, requestMethodMetadataKey),
		Path:        metadataString(record.Request.Metadata, requestTargetMetadataKey),
		Outcome:     record.Outcome,
		Status:      record.ResponseStatus,
		RequestBody: requestBodyValue(record.RawBody),
	}
	item.Profile = stringPointer(record.Profile)
	item.Operation = stringPointer(record.Operation)
	if record.MatchedStubID != "" {
		stubID := record.MatchedStubID
		item.MatchedStubID = &stubID
	}
	if record.Outcome != "matched" {
		code := failureCodeForOutcome(record.Outcome)
		item.Error = &code
	}
	return item
}

// failureCodeForOutcome maps an interaction outcome to the §43 fake failure
// code recorded in the §41.7 error member. Only a matched request, including an
// intentionally configured raw non-2xx response, carries a null error.
func failureCodeForOutcome(outcome string) string {
	if outcome == "unmatched" {
		return "fake_jev_unmatched_request"
	}
	return "fake_jev_" + outcome
}

// clearRequests clears the interaction journal and interaction-derived
// verification failures and restarts the interaction sequence at 1. It leaves
// stub invocation counts and response-sequence positions untouched (§41.8).
func (a *API) clearRequests(w http.ResponseWriter) {
	if a.engine != nil {
		a.engine.ClearRequestHistory()
	}
	w.WriteHeader(http.StatusNoContent)
}

// reset performs the six atomic §41.10 effects through the engine's reset
// boundary and serializes with control-plane stub mutations so a concurrent
// registration cannot capture an index across the reset.
func (a *API) reset(w http.ResponseWriter) {
	stubMutations.Lock()
	if a.engine != nil {
		a.engine.Reset()
	}
	stubMutations.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func metadataString(metadata map[string]engine.Value, key string) string {
	if value, ok := metadata[key].(string); ok {
		return value
	}
	return ""
}

func stringPointer(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

// requestBodyValue applies the §41.7 rule: the parsed JSON value for a valid
// body, the UTF-8-decoded raw string (invalid sequences replaced) for a
// malformed body, and null for a request with no body or an intentionally
// unretained oversized body.
func requestBodyValue(raw []byte) any {
	if len(raw) == 0 {
		return nil
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return strings.ToValidUTF8(string(raw), "\uFFFD")
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return strings.ToValidUTF8(string(raw), "\uFFFD")
	}
	return value
}

func (a *API) createStub(w http.ResponseWriter, r *http.Request) {
	var source json.RawMessage
	if err := decodeJSON(r.Body, &source); err != nil || !utf8.Valid(source) {
		var oversized *http.MaxBytesError
		if errors.As(err, &oversized) {
			writeJSON(w, http.StatusRequestEntityTooLarge, controlError{"fake_jev_payload_too_large", "Request body exceeds the configured limit.", ""})
			return
		}
		badRequest(w, "Invalid control request.")
		return
	}
	document, err := a.validationDocument(source)
	if err != nil {
		badRequest(w, err.Error())
		return
	}
	validated, err := config.Load(document)
	if err != nil || len(validated.Stubs) != 1 {
		if err == nil {
			err = errors.New("dynamic stub validation failed")
		}
		badRequest(w, err.Error())
		return
	}
	stub, err := compileStub(validated.Stubs[0])
	if err != nil {
		badRequest(w, err.Error())
		return
	}
	if a.engine == nil {
		writeJSON(w, http.StatusInternalServerError, controlError{"fake_jev_internal_error", "Internal fake-jev error.", ""})
		return
	}
	index, err := a.registerStub(stub)
	if err != nil {
		if errors.Is(err, errRegisteredStubMissing) {
			writeJSON(w, http.StatusInternalServerError, controlError{"fake_jev_internal_error", "Internal fake-jev error.", ""})
			return
		}
		if strings.HasPrefix(err.Error(), "duplicate stub id") {
			writeJSON(w, http.StatusConflict, controlError{"fake_jev_duplicate_stub_id", fmt.Sprintf("A stub with id '%s' already exists.", stub.ID), stub.ID})
			return
		}
		badRequest(w, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, struct {
		ID                string `json:"id"`
		RegistrationIndex uint64 `json:"registrationIndex"`
	}{stub.ID, index})
}

func (a *API) registerStub(stub engine.Stub) (uint64, error) {
	stubMutations.Lock()
	defer stubMutations.Unlock()
	if err := a.engine.RegisterDynamic(stub); err != nil {
		return 0, err
	}
	for _, item := range a.engine.Registry.Stubs() {
		if item.ID == stub.ID {
			return item.RegistrationIndex, nil
		}
	}
	return 0, errRegisteredStubMissing
}

func compileStub(source config.StubConfig) (engine.Stub, error) {
	matcher := engine.NewMatcher()
	matcher.Operation = source.When.Operation
	if source.When.Model != "" {
		matcher = matcher.WithModel(source.When.Model)
	}
	if len(source.When.State) > 0 {
		value, err := decodeValue(source.When.State)
		if err != nil {
			return engine.Stub{}, fmt.Errorf("stub %q state: %w", source.ID, err)
		}
		matcher = matcher.WithState(value)
	}
	if source.When.Questions != nil {
		matcher.Questions = make(map[string]string, len(source.When.Questions))
		for key, value := range source.When.Questions {
			matcher.Questions[key] = value
		}
	}
	stub := engine.Stub{ID: source.ID, Profile: source.Profile, Priority: int(source.Priority), Matcher: matcher, Response: responseAction(source.Then)}
	if source.Then.Sequence != nil {
		stub.Sequence = make([]engine.ResponseAction, len(source.Then.Sequence))
		for i, response := range source.Then.Sequence {
			stub.Sequence[i] = response
		}
	}
	if source.Expect != nil {
		expectation, err := compileExpectation(source.Expect)
		if err != nil {
			return engine.Stub{}, err
		}
		stub.Expect = expectation
	}
	return stub, nil
}

func responseAction(source config.ThenConfig) config.ResponseConfig {
	return config.ResponseConfig{Answers: source.Answers, Raw: source.Raw, Model: source.Model, Usage: source.Usage}
}

func compileExpectation(source *config.ExpectConfig) (*engine.InvocationExpectation, error) {
	if source.Exactly != nil {
		return engine.NewExactExpectation(uint64(*source.Exactly)), nil
	}
	var atLeast, atMost *uint64
	if source.AtLeast != nil {
		value := uint64(*source.AtLeast)
		atLeast = &value
	}
	if source.AtMost != nil {
		value := uint64(*source.AtMost)
		atMost = &value
	}
	return engine.NewRangeExpectation(atLeast, atMost), nil
}

func decodeValue(raw []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	return value, nil
}

func decodeJSON(reader io.Reader, target any) error {
	if reader == nil {
		return errors.New("empty body")
	}
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

type controlError struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	StubID  string `json:"stubId,omitempty"`
}

func badRequest(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusBadRequest, controlError{"fake_jev_bad_control_request", message, ""})
}

func (a *API) methodNotAllowed(w http.ResponseWriter) {
	writeJSON(w, http.StatusMethodNotAllowed, controlError{"fake_jev_control_method_not_allowed", "Method is not allowed.", ""})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	body, err := json.Marshal(value)
	if err != nil {
		status = http.StatusInternalServerError
		body = []byte(`{"error":"fake_jev_internal_error","message":"Internal fake-jev error."}`)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(body); err != nil {
		return
	}
}
