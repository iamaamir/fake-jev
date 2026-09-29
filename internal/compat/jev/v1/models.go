package v1

import (
	"encoding/json"

	"fake-jev/internal/config"
)

const (
	DefaultModelName        = "jev-latest"
	DefaultModelDescription = "Local deterministic fake model provided by fake-jev."
	DefaultModelReleaseDate = "1970-01-01"
)

// Model is the public metadata object returned by GET /v1/models.
type Model struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ReleaseDate string `json:"release_date"`
}

// ModelsResponse is the exact models-list envelope.
type ModelsResponse struct {
	Models []Model `json:"models"`
}

// DefaultModels returns a fresh copy of the built-in deterministic model list.
func DefaultModels() []Model {
	return []Model{{
		Name:        DefaultModelName,
		Description: DefaultModelDescription,
		ReleaseDate: DefaultModelReleaseDate,
	}}
}

// ModelsFromConfig preserves model configuration order and values. An omitted
// model list uses the normative built-in list.
func ModelsFromConfig(models []config.ModelConfig) []Model {
	if len(models) == 0 {
		return DefaultModels()
	}
	result := make([]Model, len(models))
	for i, model := range models {
		result[i] = Model{Name: model.Name, Description: model.Description, ReleaseDate: model.ReleaseDate}
	}
	return result
}

// EncodeModels encodes a successful GET /v1/models response. The returned
// body is independent of the input slice and is safe for later mutation.
func EncodeModels(models []Model) (EncodedResponse, error) {
	if len(models) == 0 {
		models = DefaultModels()
	}
	body, err := json.Marshal(ModelsResponse{Models: append([]Model(nil), models...)})
	if err != nil {
		return EncodedResponse{}, err
	}
	return EncodedResponse{Status: 200, Headers: map[string]string{"Content-Type": JSONContentType}, Body: body}, nil
}

// EncodeConfiguredModels converts configuration metadata while preserving its
// order, then encodes the profile's models response.
func EncodeConfiguredModels(models []config.ModelConfig) (EncodedResponse, error) {
	return EncodeModels(ModelsFromConfig(models))
}

// DefaultModelsResponse is the built-in GET /v1/models response.
func DefaultModelsResponse() (EncodedResponse, error) { return EncodeModels(nil) }
