package engine

// RuntimeConfig controls optional runtime features.
type RuntimeConfig struct {
	// CallLogEnabled toggles sqlite call logging.
	CallLogEnabled bool
	// CallLogPath overrides the sqlite database path for call logging.
	CallLogPath string
}

// DefaultRuntimeConfig enables call logging to the default sqlite path.
func DefaultRuntimeConfig() RuntimeConfig {
	return RuntimeConfig{
		CallLogEnabled: true,
	}
}
