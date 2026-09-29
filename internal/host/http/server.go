package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	nethttp "net/http"
	"time"

	"fake-jev/internal/compat/jev/v1"
	"fake-jev/internal/config"
	"fake-jev/internal/engine"
)

const (
	DefaultDataPlaneBodyBytes    = 8 * 1024 * 1024
	DefaultControlPlaneBodyBytes = 2 * 1024 * 1024
	DefaultMaxInteractions       = engine.DefaultMaxInteractions
	DefaultLogBodyBytes          = 4096
	DefaultGracefulShutdown      = 5
)

// Engine-exchange metadata keys shared with the control API's request-history
// endpoint. The provider-neutral journal carries no HTTP details (§8.6), so
// §41.7's method and path are recorded here. Keep in sync with
// internal/control.
const (
	requestMethodMetadataKey = "method"
	requestTargetMetadataKey = "target"
)

// Limits are the host resource limits. A zero value is filled with the
// specification defaults by NewServer.
type Limits struct {
	DataPlaneBodyBytes    int
	ControlPlaneBodyBytes int
	MaxInteractions       int
	LogBodyBytes          int
	GracefulShutdown      int
}

func defaultLimits() Limits {
	return Limits{
		DataPlaneBodyBytes:    DefaultDataPlaneBodyBytes,
		ControlPlaneBodyBytes: DefaultControlPlaneBodyBytes,
		MaxInteractions:       DefaultMaxInteractions,
		LogBodyBytes:          DefaultLogBodyBytes,
		GracefulShutdown:      DefaultGracefulShutdown,
	}
}

func limitsFromConfig(value config.LimitsConfig) Limits {
	return applyLimitDefaults(Limits{
		DataPlaneBodyBytes:    value.DataPlaneBodyBytes,
		ControlPlaneBodyBytes: value.ControlPlaneBodyBytes,
		MaxInteractions:       value.MaxInteractions,
		LogBodyBytes:          value.LogBodyBytes,
		GracefulShutdown:      value.GracefulShutdownSeconds,
	})
}

func applyLimitDefaults(limits Limits) Limits {
	defaults := defaultLimits()
	if limits.DataPlaneBodyBytes == 0 {
		limits.DataPlaneBodyBytes = defaults.DataPlaneBodyBytes
	}
	if limits.ControlPlaneBodyBytes == 0 {
		limits.ControlPlaneBodyBytes = defaults.ControlPlaneBodyBytes
	}
	if limits.MaxInteractions == 0 {
		limits.MaxInteractions = defaults.MaxInteractions
	}
	if limits.LogBodyBytes == 0 {
		limits.LogBodyBytes = defaults.LogBodyBytes
	}
	if limits.GracefulShutdown == 0 {
		limits.GracefulShutdown = defaults.GracefulShutdown
	}
	return limits
}

// Server is an HTTP handler and a listener-backed host. It deliberately does
// not implement control endpoints; /__fake/ requests are only classified and
// bounded here so a later control API can share the listener safely.
type Server struct {
	router *Router
	limits Limits
}

// NewServer creates a host around an existing engine and router.
func NewServer(router *Router, limits Limits) *Server {
	if router == nil {
		router = NewDefaultRouter(nil)
	}
	return &Server{router: router, limits: applyLimitDefaults(limits)}
}

// NewServerFromConfig compiles the static configuration into the engine and
// creates the HTTP host. Configuration loading and validation remain outside
// this package; this function accepts a validated config model.
func NewServerFromConfig(cfg *config.Config) (*Server, error) {
	if cfg == nil {
		return nil, errors.New("configuration is nil")
	}
	static, err := compileStubs(cfg.Stubs)
	if err != nil {
		return nil, err
	}
	limits := limitsFromConfig(cfg.Limits)
	state, err := engine.NewEngineWithJournal(static, limits.MaxInteractions)
	if err != nil {
		return nil, err
	}
	return NewServer(NewRouter(state, v1.NewProfile(cfg.Models)), limits), nil
}

// NewServerWithConfig is an explicit alias for NewServerFromConfig.
func NewServerWithConfig(cfg *config.Config) (*Server, error) { return NewServerFromConfig(cfg) }

// Engine exposes the host's state for deterministic in-process verification
// and diagnostics. It does not add an HTTP control surface.
func (s *Server) Engine() *engine.Engine {
	if s == nil || s.router == nil {
		return nil
	}
	return s.router.engine
}

// Serve serves requests from a real net/http listener.
func (s *Server) Serve(listener net.Listener) error {
	if s == nil {
		return errors.New("nil HTTP server")
	}
	if listener == nil {
		return errors.New("nil HTTP listener")
	}
	return (&nethttp.Server{Handler: s}).Serve(listener)
}

// ServeHTTP implements net/http.Handler.
func (s *Server) ServeHTTP(writer nethttp.ResponseWriter, request *nethttp.Request) {
	if request != nil && request.Body != nil {
		defer request.Body.Close()
	}
	if s == nil || s.router == nil || !s.router.active() || request == nil {
		writeJSON(writer, nethttp.StatusInternalServerError, internalErrorBody{})
		return
	}
	control := s.router.isControl(request)
	limit := s.limits.DataPlaneBodyBytes
	if control {
		limit = s.limits.ControlPlaneBodyBytes
	}
	body, tooLarge, err := readBody(request, limit)
	if err != nil {
		writeJSON(writer, nethttp.StatusInternalServerError, internalErrorBody{})
		return
	}
	if tooLarge {
		// Control requests are intentionally not sent through the engine. A data
		// request is admitted through Transition so it receives a normal sequence.
		if control {
			writeJSON(writer, nethttp.StatusRequestEntityTooLarge, failureBody{Error: "fake_jev_payload_too_large", Message: "Request body exceeds the configured limit."})
			return
		}
		s.serveData(writer, request, nil, true)
		return
	}
	if control {
		// The control API is implemented by a later host slice. Keep unsupported
		// control traffic outside the data-plane journal and state.
		writeJSON(writer, nethttp.StatusNotFound, failureBody{Error: "fake_jev_control_not_found", Message: "Unknown control endpoint."})
		return
	}
	s.serveData(writer, request, body, false)
}

func (s *Server) serveData(writer nethttp.ResponseWriter, request *nethttp.Request, body []byte, tooLarge bool) {
	route := s.router.route(request)
	operation := route.Operation()
	requestInfo := engine.InteractionRequest{
		Profile: s.router.profileID(), Operation: operation, Received: time.Now(), RawBody: body,
	}
	if tooLarge {
		requestInfo.RawBody = nil
	}
	var decoded v1.Request
	var decodeErr error
	decide := func(sequence uint64) engine.InteractionDecision {
		_ = sequence
		if tooLarge {
			return engine.InteractionDecision{
				Exchange: errorExchange(s.router.profileID(), operation, "fake_jev_payload_too_large"),
				Valid:    false, Outcome: "payload_too_large", ResponseStatus: nethttp.StatusRequestEntityTooLarge,
				Failure: &engine.VerificationFailure{Code: "payload_too_large", Message: "Request exceeded the configured payload limit."},
			}
		}
		if route == v1.UnknownRoute {
			return engine.InteractionDecision{
				Exchange: errorExchange(s.router.profileID(), "", "fake_jev_unknown_route"),
				Valid:    false, Outcome: "unknown_route", ResponseStatus: nethttp.StatusNotFound,
				Failure: &engine.VerificationFailure{Code: "unknown_route", Message: "Request used an unknown route."},
			}
		}
		exchange, err := s.router.profile.Decode(v1.HostRequest{
			Meta: v1.RequestMeta{Method: request.Method, Target: requestTarget(request)}, Body: body,
		})
		decodeErr = err
		if decodeErr != nil {
			status := nethttp.StatusUnprocessableEntity
			if validation, ok := decodeErr.(*v1.ValidationError); ok {
				status = validation.StatusCode()
			}
			return engine.InteractionDecision{Exchange: errorExchange(s.router.profileID(), operation, "validation_error"), Valid: false, Outcome: "validation_error", ResponseStatus: status,
				Failure: &engine.VerificationFailure{Code: "validation_error", Message: "Request failed validation."}}
		}
		if operation == "models" {
			return engine.InteractionDecision{Exchange: exchange, Valid: true, Outcome: "matched", ResponseStatus: nethttp.StatusOK}
		}
		if operation == "systemone" {
			var ok bool
			decoded, ok = exchange.Payload.(v1.Request)
			if !ok {
				return engine.InteractionDecision{Exchange: errorExchange(s.router.profileID(), operation, "fake_jev_internal_error"), Valid: false, Outcome: "internal_error", ResponseStatus: nethttp.StatusInternalServerError}
			}
			if len(s.router.engine.Registry.Matching(exchange)) == 0 {
				return engine.InteractionDecision{Exchange: exchange, Valid: true, Outcome: "unmatched", ResponseStatus: nethttp.StatusNotImplemented,
					Failure: &engine.VerificationFailure{Code: "unmatched_request", Message: "Request did not match any configured stub."}}
			}
		}
		return engine.InteractionDecision{Exchange: exchange, Valid: true, Outcome: "matched"}
	}
	record, selection, err := s.router.engine.Transition(requestInfo, func(sequence uint64) engine.InteractionDecision {
		decision := decide(sequence)
		stampRequestMetadata(&decision.Exchange, request)
		return decision
	})
	if err != nil {
		if errors.Is(err, engine.ErrJournalFull) {
			writeJSON(writer, nethttp.StatusInsufficientStorage, failureBody{Error: "fake_jev_journal_full", Message: "Interaction journal limit reached. Reset or increase maxInteractions."})
			return
		}
		writeJSON(writer, nethttp.StatusInternalServerError, internalErrorBody{})
		return
	}
	if record == nil {
		writeJSON(writer, nethttp.StatusInternalServerError, internalErrorBody{})
		return
	}
	if tooLarge {
		writeJSON(writer, nethttp.StatusRequestEntityTooLarge, failureBody{Error: "fake_jev_payload_too_large", Message: "Request body exceeds the configured limit."})
		return
	}
	if route == v1.UnknownRoute {
		writeJSON(writer, nethttp.StatusNotFound, failureBody{Error: "fake_jev_unknown_route", Message: "No active compatibility profile handles this route."})
		return
	}
	if decodeErr != nil {
		if validation, ok := decodeErr.(*v1.ValidationError); ok {
			writeJSON(writer, validation.StatusCode(), validation.Envelope)
		} else {
			writeJSON(writer, nethttp.StatusInternalServerError, internalErrorBody{})
		}
		return
	}
	if operation == "models" {
		response, encodeErr := s.router.profile.ModelsResponse()
		if encodeErr != nil {
			writeJSON(writer, nethttp.StatusInternalServerError, internalErrorBody{})
			return
		}
		writeEncoded(writer, response)
		return
	}
	if selection == nil {
		writeJSON(writer, nethttp.StatusNotImplemented, failureBody{Error: "fake_jev_unmatched_request", Message: "No configured stub matched this request.", Profile: s.router.profileID(), Operation: operation})
		return
	}
	if selection.SequenceExhausted {
		writeJSON(writer, nethttp.StatusConflict, failureWithStub("fake_jev_sequence_exhausted", "Configured response sequence is exhausted.", selection.Stub.ID))
		return
	}
	responseConfig, ok := selection.Action.(config.ResponseConfig)
	if !ok {
		writeJSON(writer, nethttp.StatusInternalServerError, failureWithStub("fake_jev_invalid_stub_response", "Configured stub cannot produce a valid response for this request.", selection.Stub.ID))
		return
	}
	response, encodeErr := s.router.profile.Encode(v1.ResponseResult{Request: decoded, Response: responseConfig})
	if encodeErr != nil {
		s.router.engine.RecordFailure(record.Sequence, "invalid_stub_response", "Configured stub cannot produce a valid response for this request.", selection.Stub.ID, nethttp.StatusInternalServerError)
		writeJSON(writer, nethttp.StatusInternalServerError, failureWithStub("fake_jev_invalid_stub_response", "Configured stub cannot produce a valid response for this request.", selection.Stub.ID))
		return
	}
	s.router.engine.UpdateInteraction(record.Sequence, "matched", response.Status)
	writeEncoded(writer, response)
}

func requestTarget(request *nethttp.Request) string {
	if request == nil || request.URL == nil {
		return ""
	}
	return request.URL.RequestURI()
}

// stampRequestMetadata records the HTTP method and the URL path (query string
// excluded, §41.7 / §39.8) on the exchange so the control API can render the
// request-history record. Every data-plane transition passes through here,
// including route, validation, payload, and internal failures.
func stampRequestMetadata(exchange *engine.Exchange, request *nethttp.Request) {
	if exchange == nil || request == nil {
		return
	}
	if exchange.Metadata == nil {
		exchange.Metadata = make(map[string]engine.Value, 2)
	}
	exchange.Metadata[requestMethodMetadataKey] = request.Method
	path := ""
	if request.URL != nil {
		path = request.URL.Path
	}
	exchange.Metadata[requestTargetMetadataKey] = path
}

func readBody(request *nethttp.Request, limit int) ([]byte, bool, error) {
	if request == nil || request.Body == nil {
		return nil, false, nil
	}
	if limit < 0 {
		limit = 0
	}
	if request.ContentLength > int64(limit) {
		return nil, true, nil
	}
	reader := io.LimitReader(request.Body, int64(limit)+1)
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, false, err
	}
	if len(body) > limit {
		return nil, true, nil
	}
	return body, false, nil
}

func errorExchange(profile, operation, code string) engine.Exchange {
	exchange := engine.NewExchange(profile, operation)
	exchange.Metadata["error"] = code
	return exchange
}

func writeEncoded(writer nethttp.ResponseWriter, response v1.HostResponse) {
	for name, value := range response.Headers {
		writer.Header().Set(name, value)
	}
	writer.WriteHeader(response.Status)
	if _, err := writer.Write(response.Body); err != nil {
		return
	}
}

func writeJSON(writer nethttp.ResponseWriter, status int, value any) {
	body, err := json.Marshal(value)
	if err != nil {
		status = nethttp.StatusInternalServerError
		body = []byte(`{"error":"fake_jev_internal_error","message":"Internal fake-jev error."}`)
	}
	writer.Header().Set("Content-Type", v1.JSONContentType)
	writer.WriteHeader(status)
	if _, err := writer.Write(body); err != nil {
		return
	}
}

type failureBody struct {
	Error     string  `json:"error"`
	Message   string  `json:"message"`
	Profile   string  `json:"profile,omitempty"`
	Operation string  `json:"operation,omitempty"`
	StubID    *string `json:"stubId,omitempty"`
}

func failureWithStub(code, message, stubID string) failureBody {
	return failureBody{Error: code, Message: message, StubID: &stubID}
}

type internalErrorBody struct{}

func (internalErrorBody) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}{"fake_jev_internal_error", "Internal fake-jev error."})
}

func compileStubs(stubs []config.StubConfig) ([]engine.Stub, error) {
	compiled := make([]engine.Stub, 0, len(stubs))
	for _, source := range stubs {
		matcher := engine.NewMatcher()
		matcher.Operation = source.When.Operation
		if source.When.Model != "" {
			matcher = matcher.WithModel(source.When.Model)
		}
		if len(source.When.State) > 0 {
			state, err := decodeJSONValue(source.When.State)
			if err != nil {
				return nil, fmt.Errorf("stub %q state: %w", source.ID, err)
			}
			matcher = matcher.WithState(state)
		}
		if source.When.Questions != nil {
			matcher.Questions = cloneStringMap(source.When.Questions)
		}
		stub := engine.Stub{ID: source.ID, Profile: source.Profile, Priority: int(source.Priority), Matcher: matcher, Response: thenResponse(source.Then)}
		if source.Then.Sequence != nil {
			stub.Sequence = make([]engine.ResponseAction, len(source.Then.Sequence))
			for index, response := range source.Then.Sequence {
				stub.Sequence[index] = response
			}
		}
		if source.Expect != nil {
			expectation, err := compileExpectation(source.Expect)
			if err != nil {
				return nil, fmt.Errorf("stub %q: %w", source.ID, err)
			}
			stub.Expect = expectation
		}
		compiled = append(compiled, stub)
	}
	return compiled, nil
}

func compileExpectation(source *config.ExpectConfig) (*engine.InvocationExpectation, error) {
	if source.Exactly != nil {
		if *source.Exactly < 0 {
			return nil, errors.New("expect.exactly must be non-negative")
		}
		return engine.NewExactExpectation(uint64(*source.Exactly)), nil
	}
	if source.AtLeast == nil && source.AtMost == nil {
		return nil, errors.New("expectation requires exactly, atLeast, or atMost")
	}
	var atLeast, atMost *uint64
	if source.AtLeast != nil {
		if *source.AtLeast < 0 {
			return nil, errors.New("expect.atLeast must be non-negative")
		}
		value := uint64(*source.AtLeast)
		atLeast = &value
	}
	if source.AtMost != nil {
		if *source.AtMost < 0 {
			return nil, errors.New("expect.atMost must be non-negative")
		}
		value := uint64(*source.AtMost)
		atMost = &value
	}
	return engine.NewRangeExpectation(atLeast, atMost), nil
}

func decodeJSONValue(raw []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	return value, nil
}

func cloneStringMap(source map[string]string) map[string]string {
	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

// thenResponse keeps compatibility-specific response data opaque to engine.
func thenResponse(source config.ThenConfig) config.ResponseConfig {
	return config.ResponseConfig{Answers: source.Answers, Raw: source.Raw, Model: source.Model, Usage: source.Usage}
}

var _ nethttp.Handler = (*Server)(nil)
