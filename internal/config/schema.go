// Package config defines and loads the fake-jev configuration contract.
package config

import "encoding/json"

const (
	DefaultHost    = "127.0.0.1"
	DefaultPort    = 8787
	DefaultMode    = "strict"
	DefaultProfile = "jev/v1"
)

type Config struct {
	SchemaVersion int           `json:"schemaVersion"`
	Server        ServerConfig  `json:"server"`
	Mode          string        `json:"mode"`
	Compatibility []string      `json:"compatibility"`
	Limits        LimitsConfig  `json:"limits"`
	Models        []ModelConfig `json:"models"`
	Stubs         []StubConfig  `json:"stubs"`
}

type ServerConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type LimitsConfig struct {
	DataPlaneBodyBytes      int `json:"dataPlaneBodyBytes"`
	ControlPlaneBodyBytes   int `json:"controlPlaneBodyBytes"`
	MaxInteractions         int `json:"maxInteractions"`
	LogBodyBytes            int `json:"logBodyBytes"`
	GracefulShutdownSeconds int `json:"gracefulShutdownSeconds"`
}

type ModelConfig struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ReleaseDate string `json:"release_date"`
}

type StubConfig struct {
	ID       string        `json:"id"`
	Profile  string        `json:"profile"`
	Priority int32         `json:"priority"`
	When     WhenConfig    `json:"when"`
	Then     ThenConfig    `json:"then"`
	Expect   *ExpectConfig `json:"expect,omitempty"`
}

type WhenConfig struct {
	Operation string            `json:"operation"`
	Model     string            `json:"model,omitempty"`
	State     json.RawMessage   `json:"state,omitempty"`
	Questions map[string]string `json:"questions,omitempty"`
}

type ThenConfig struct {
	Answers  json.RawMessage  `json:"answers,omitempty"`
	Sequence []ResponseConfig `json:"sequence,omitempty"`
	Raw      *RawResponse     `json:"raw,omitempty"`
	Model    string           `json:"model,omitempty"`
	Usage    *UsageConfig     `json:"usage,omitempty"`
}

type ResponseConfig struct {
	Answers json.RawMessage `json:"answers,omitempty"`
	Raw     *RawResponse    `json:"raw,omitempty"`
	Model   string          `json:"model,omitempty"`
	Usage   *UsageConfig    `json:"usage,omitempty"`
}

type RawResponse struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	Body    json.RawMessage   `json:"body"`
}

type UsageConfig struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type ExpectConfig struct {
	Exactly *int `json:"exactly,omitempty"`
	AtLeast *int `json:"atLeast,omitempty"`
	AtMost  *int `json:"atMost,omitempty"`
}
