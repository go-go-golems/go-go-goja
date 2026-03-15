package obsidiancli

import "time"

// Config configures how the Obsidian CLI is invoked.
type Config struct {
	BinaryPath string
	Vault      string
	WorkingDir string
	Timeout    time.Duration
	Env        []string
}

// DefaultConfig returns the default Obsidian CLI configuration.
func DefaultConfig() Config {
	return Config{
		BinaryPath: "obsidian",
	}
}
