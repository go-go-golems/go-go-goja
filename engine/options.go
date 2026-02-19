package engine

import "github.com/dop251/goja_nodejs/require"

type openSettings struct {
	requireOptions []require.Option
}

func defaultOpenSettings() openSettings {
	return openSettings{}
}

// Option configures engine.Open behavior.
type Option func(*openSettings)

// WithRequireOptions appends require registry options for module loading.
func WithRequireOptions(opts ...require.Option) Option {
	return func(s *openSettings) {
		s.requireOptions = append(s.requireOptions, opts...)
	}
}
