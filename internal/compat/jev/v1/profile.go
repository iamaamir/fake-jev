package v1

import (
	"bytes"
	"encoding/json"
	"fmt"

	"fake-jev/internal/config"
	"fake-jev/internal/engine"
)

// RequestMeta is the host-neutral routing metadata needed by a profile.
type RequestMeta struct {
	Method string
	Target string
}

// HostRequest is the host-neutral input passed to a compatibility profile.
// Headers are intentionally opaque to jev/v1: Authorization is accepted as-is
// and is neither validated nor logged.
type HostRequest struct {
	Meta    RequestMeta
	Headers map[string]string
	Body    []byte
}

// HostResponse is the host-neutral output returned by a compatibility profile.
type HostResponse = EncodedResponse

// Profile is the small compatibility boundary described by §9. It contains no
// HTTP server or listener behavior.
type Profile interface {
	ID() string
	MatchRoute(RequestMeta) bool
	Decode(HostRequest) (engine.Exchange, error)
	Encode(ResponseResult) (HostResponse, error)
}

// JevV1Profile owns jev/v1 protocol behavior while leaving request admission,
// journal transitions, and stub selection to the host and engine.
type JevV1Profile struct {
	models []Model
}

// NewProfile constructs a jev/v1 profile. Configuration order is retained.
func NewProfile(models []config.ModelConfig) *JevV1Profile {
	configured := ModelsFromConfig(models)
	return &JevV1Profile{models: append([]Model(nil), configured...)}
}

// DefaultProfile returns a profile using the built-in model metadata.
func DefaultProfile() *JevV1Profile { return NewProfile(nil) }

// ID returns the compile-time registered profile identifier.
func (p *JevV1Profile) ID() string { return "jev/v1" }

// MatchRoute reports whether the profile owns the exact route and method.
func (p *JevV1Profile) MatchRoute(meta RequestMeta) bool {
	_, ok := MatchRoute(meta.Method, meta.Target)
	return ok
}

// Route returns the recognized route without performing decoding.
func (p *JevV1Profile) Route(meta RequestMeta) Route { return RecognizeRoute(meta.Method, meta.Target) }

// Decode validates systemone JSON and normalizes it for the provider-neutral
// engine. Models has no request body and therefore returns an empty exchange.
// Headers, including Authorization, are deliberately not inspected.
func (p *JevV1Profile) Decode(input HostRequest) (engine.Exchange, error) {
	route := RecognizeRoute(input.Meta.Method, input.Meta.Target)
	switch route {
	case ModelsRoute:
		return engine.NewExchange(p.ID(), route.Operation()), nil
	case SystemOneRoute:
		request, validationErr := ValidateRequest(input.Body)
		if validationErr != nil {
			return engine.Exchange{}, validationErr
		}
		state, err := decodeJSONValue(request.State)
		if err != nil {
			return engine.Exchange{}, fmt.Errorf("decode request state: %w", err)
		}
		exchange := engine.NewExchange(p.ID(), route.Operation())
		exchange.Model = request.Model
		exchange.State = state
		for name, question := range request.Questions {
			exchange.Questions[name] = question.Type
		}
		exchange.Payload = request
		return exchange, nil
	default:
		return engine.Exchange{}, fmt.Errorf("unknown jev/v1 route")
	}
}

// Encode serializes a selected response result. Models responses are exposed
// separately through Models because they do not require a stub result.
func (p *JevV1Profile) Encode(result ResponseResult) (HostResponse, error) {
	return EncodeResult(result)
}

// Models returns the successful GET /v1/models response for this profile.
func (p *JevV1Profile) Models() (HostResponse, error) {
	return EncodeModels(p.models)
}

// ModelsResponse is an explicit alias useful to hosts dispatching a models
// route without selecting a stub.
func (p *JevV1Profile) ModelsResponse() (HostResponse, error) { return p.Models() }

func decodeJSONValue(raw json.RawMessage) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	return value, nil
}
