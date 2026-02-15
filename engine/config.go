package engine

// RuntimeConfig controls optional runtime features.
type RuntimeConfig struct{}

// DefaultRuntimeConfig returns the default runtime configuration.
func DefaultRuntimeConfig() RuntimeConfig {
	return RuntimeConfig{}
}
