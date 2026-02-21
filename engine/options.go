package engine

import "github.com/dop251/goja_nodejs/require"

type builderSettings struct {
	requireOptions []require.Option
}

func defaultBuilderSettings() builderSettings {
	return builderSettings{}
}

// Option configures engine builder behavior.
type Option func(*builderSettings)

// WithRequireOptions appends require registry options for module loading.
func WithRequireOptions(opts ...require.Option) Option {
	return func(s *builderSettings) {
		s.requireOptions = append(s.requireOptions, opts...)
	}
}
