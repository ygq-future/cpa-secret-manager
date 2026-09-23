package runtime

import (
	"errors"
	"io"
	"time"
)

// Methods exchanged with the CPA host over the C ABI JSON envelope.
const (
	MethodPluginRegister     = "plugin.register"
	MethodPluginReconfigure  = "plugin.reconfigure"
	MethodPluginShutdown     = "plugin.shutdown"
	MethodManagementRegister = "management.register"
	MethodManagementHandle   = "management.handle"
	MethodUsageHandle        = "usage.handle"
)

// SchemaVersion is the plugin contract version declared to the host. Keep it
// below the host's newest schema unless a newer capability is actually needed.
const SchemaVersion uint32 = 1

var (
	// ErrInvalidRequest indicates a malformed host request envelope.
	ErrInvalidRequest = errors.New("runtime: invalid request")
	// ErrShutdown indicates that the runtime has been shut down.
	ErrShutdown = errors.New("runtime: shutdown")
)

// Clock abstracts time for testing.
type Clock interface {
	Now() time.Time
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }

// Options provides injectable dependencies for the runtime.
type Options struct {
	// Clock overrides the time source; nil uses the system clock.
	Clock Clock
	// StatePath overrides the configured plugin state document path. It is used
	// by local harnesses and tests; production uses the host configuration.
	StatePath string
	// Random overrides the key generation entropy source; nil uses crypto/rand.
	Random io.Reader
	// FlushInterval bounds how much usage accounting a crash can lose; zero uses
	// the default.
	FlushInterval time.Duration
}

// RegisterResult is the metadata and capability declaration returned on
// registration.
type RegisterResult struct {
	SchemaVersion uint32       `json:"schema_version"`
	Metadata      Metadata     `json:"metadata"`
	Capabilities  Capabilities `json:"capabilities"`
}

// Capabilities declares the host integration points implemented by the plugin.
type Capabilities struct {
	ManagementAPI bool `json:"management_api"`
	UsagePlugin   bool `json:"usage_plugin"`
}

// Metadata describes non-sensitive plugin information displayed by the host.
type Metadata struct {
	Name             string `json:"Name"`
	Version          string `json:"Version"`
	Author           string `json:"Author"`
	GitHubRepository string `json:"GitHubRepository"`
	Description      string `json:"Description,omitempty"`
	Logo             string `json:"Logo,omitempty"`
	// ConfigFields stays empty on purpose: business settings live in the plugin
	// state document so the host config drawer cannot rewrite config.yaml.
	ConfigFields []ConfigField `json:"ConfigFields,omitempty"`
}

// ConfigField describes one host-rendered configuration field.
type ConfigField struct {
	Name        string `json:"Name"`
	Type        string `json:"Type"`
	Description string `json:"Description"`
}
