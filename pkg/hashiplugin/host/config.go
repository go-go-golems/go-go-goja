package host

import (
	"strings"
	"time"

	"github.com/hashicorp/go-hclog"
)

// Config controls plugin discovery and runtime integration.
type Config struct {
	Directories  []string
	Pattern      string
	Namespace    string
	AllowModules []string
	StartTimeout time.Duration
	CallTimeout  time.Duration
	AutoMTLS     bool
	Logger       hclog.Logger
}

func (c Config) withDefaults() Config {
	if strings.TrimSpace(c.Pattern) == "" {
		c.Pattern = "goja-plugin-*"
	}
	if strings.TrimSpace(c.Namespace) == "" {
		c.Namespace = "plugin:"
	}
	if c.StartTimeout <= 0 {
		c.StartTimeout = 10 * time.Second
	}
	if c.CallTimeout <= 0 {
		c.CallTimeout = 5 * time.Second
	}
	if c.Logger == nil {
		c.Logger = hclog.NewNullLogger()
	}
	return c
}
