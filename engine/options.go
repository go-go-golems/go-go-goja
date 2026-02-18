package engine

import "github.com/dop251/goja_nodejs/require"

type callLogMode int

const (
	callLogModeDisabled callLogMode = iota
	callLogModeEnabled
)

type openSettings struct {
	requireOptions []require.Option
	callLogMode    callLogMode
	callLogPath    string
}

func defaultOpenSettings() openSettings {
	return openSettings{
		callLogMode: callLogModeDisabled,
	}
}

// Option configures engine.Open behavior.
type Option func(*openSettings)

// WithRequireOptions appends require registry options for module loading.
func WithRequireOptions(opts ...require.Option) Option {
	return func(s *openSettings) {
		s.requireOptions = append(s.requireOptions, opts...)
	}
}

// WithCallLog enables call logging for the opened runtime.
// If path is empty, the default sqlite location is used.
func WithCallLog(path string) Option {
	return func(s *openSettings) {
		s.callLogMode = callLogModeEnabled
		s.callLogPath = path
	}
}

// WithCallLogDisabled disables call logging for the opened runtime.
func WithCallLogDisabled() Option {
	return func(s *openSettings) {
		s.callLogMode = callLogModeDisabled
		s.callLogPath = ""
	}
}
