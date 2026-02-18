package engine

// RuntimeConfig controls optional runtime features.
type RuntimeConfig struct {
	// CallLogEnabled toggles sqlite call logging.
	CallLogEnabled bool
	// CallLogPath overrides the sqlite database path for call logging.
	CallLogPath string
}

// DefaultRuntimeConfig keeps call logging disabled unless explicitly enabled.
func DefaultRuntimeConfig() RuntimeConfig {
	return RuntimeConfig{
		CallLogEnabled: false,
	}
}
