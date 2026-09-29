// Package control implements the local fake-jev control API.
package control

import (
	"encoding/json"
	"net/http"
	"sync"

	"fake-jev/internal/config"
	"fake-jev/internal/engine"
)

const (
	ControlAPIVersion = "v1"
	// ServerVersion is the semver fallback for development hosts. Release hosts
	// supply their product version through Metadata.
	ServerVersion = "0.0.0-dev"
)

// Limits is the resource-limit portion exposed by the metadata endpoint.
type Limits struct {
	DataPlaneBodyBytes    int `json:"dataPlaneBodyBytes"`
	ControlPlaneBodyBytes int `json:"controlPlaneBodyBytes"`
	MaxInteractions       int `json:"maxInteractions"`
	LogBodyBytes          int `json:"logBodyBytes"`
	GracefulShutdown      int `json:"gracefulShutdownSeconds"`
}

// Metadata is the immutable configuration information exposed by /meta.
type Metadata struct {
	ServerVersion       string
	ConfigSchemaVersion int
	Mode                string
	ActiveProfiles      []string
	Limits              Limits
}

// stubMutations keeps registration and index capture atomic with control clears,
// including when multiple API instances share an engine. Engine mutations made
// directly by an embedding host must not race with these control operations.
var stubMutations sync.Mutex

// API owns control-plane state and deliberately never uses the engine journal.
type API struct {
	engine   *engine.Engine
	metadata Metadata
	config   *config.Config
}

// NewAPI creates a control API backed by engine. cfg supplies metadata and is
// retained only to validate dynamically submitted stubs against the same
// configuration contract.
func NewAPI(engineState *engine.Engine, cfg *config.Config) *API {
	api := &API{engine: engineState, metadata: metadataFromConfig(cfg), config: cfg}
	return api
}

// New is a short alias for NewAPI.
func New(engineState *engine.Engine, cfg *config.Config) *API { return NewAPI(engineState, cfg) }

// NewAPIWithMetadata creates an API for hosts that already have compiled
// configuration metadata but do not retain a config.Config value.
func NewAPIWithMetadata(engineState *engine.Engine, metadata Metadata) *API {
	if metadata.ServerVersion == "" {
		metadata.ServerVersion = ServerVersion
	}
	if metadata.ConfigSchemaVersion == 0 {
		metadata.ConfigSchemaVersion = 1
	}
	if metadata.Mode == "" {
		metadata.Mode = "strict"
	}
	metadata.ActiveProfiles = append([]string(nil), metadata.ActiveProfiles...)
	return &API{engine: engineState, metadata: metadata}
}

// Handler returns the net/http handler for all /__fake/v1 endpoints in this
// slice. It does not admit data-plane requests.
func (a *API) Handler() http.Handler { return a }

func metadataFromConfig(cfg *config.Config) Metadata {
	metadata := Metadata{ServerVersion: ServerVersion, ConfigSchemaVersion: 1, Mode: "strict"}
	if cfg == nil {
		metadata.ActiveProfiles = []string{"jev/v1"}
		metadata.Limits = Limits{DataPlaneBodyBytes: 8388608, ControlPlaneBodyBytes: 2097152, MaxInteractions: 10000, LogBodyBytes: 4096, GracefulShutdown: 5}
		return metadata
	}
	metadata.ConfigSchemaVersion = cfg.SchemaVersion
	metadata.Mode = cfg.Mode
	metadata.ActiveProfiles = append([]string(nil), cfg.Compatibility...)
	metadata.Limits = Limits{
		DataPlaneBodyBytes: cfg.Limits.DataPlaneBodyBytes, ControlPlaneBodyBytes: cfg.Limits.ControlPlaneBodyBytes,
		MaxInteractions: cfg.Limits.MaxInteractions, LogBodyBytes: cfg.Limits.LogBodyBytes,
		GracefulShutdown: cfg.Limits.GracefulShutdownSeconds,
	}
	return metadata
}

// configJSON is intentionally private: the control API accepts the same
// schema as a configuration stub, not a second control-specific schema.
type configJSON struct {
	SchemaVersion int                  `json:"schemaVersion"`
	Compatibility []string             `json:"compatibility"`
	Limits        config.LimitsConfig  `json:"limits"`
	Models        []config.ModelConfig `json:"models"`
	Stubs         []json.RawMessage    `json:"stubs"`
	Mode          string               `json:"mode"`
}

func (a *API) validationDocument(stub json.RawMessage) ([]byte, error) {
	base := configJSON{SchemaVersion: 1, Compatibility: append([]string(nil), a.metadata.ActiveProfiles...), Mode: a.metadata.Mode, Stubs: []json.RawMessage{stub}}
	if len(base.Compatibility) == 0 {
		base.Compatibility = []string{"jev/v1"}
	}
	limits := a.metadata.Limits
	if limits.DataPlaneBodyBytes == 0 {
		limits.DataPlaneBodyBytes = 8388608
	}
	if limits.ControlPlaneBodyBytes == 0 {
		limits.ControlPlaneBodyBytes = 2097152
	}
	if limits.MaxInteractions == 0 {
		limits.MaxInteractions = 10000
	}
	if limits.LogBodyBytes == 0 {
		limits.LogBodyBytes = 4096
	}
	if limits.GracefulShutdown == 0 {
		limits.GracefulShutdown = 5
	}
	base.Limits = config.LimitsConfig{DataPlaneBodyBytes: limits.DataPlaneBodyBytes, ControlPlaneBodyBytes: limits.ControlPlaneBodyBytes, MaxInteractions: limits.MaxInteractions, LogBodyBytes: limits.LogBodyBytes, GracefulShutdownSeconds: limits.GracefulShutdown}
	if a.config != nil {
		base.Models = append([]config.ModelConfig(nil), a.config.Models...)
	} else {
		base.Models = []config.ModelConfig{{Name: "jev-latest", Description: "Local deterministic fake model provided by fake-jev.", ReleaseDate: "1970-01-01"}}
	}
	return json.Marshal(base)
}
