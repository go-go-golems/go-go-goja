// Package tsscript wraps esbuild's Go API for TypeScript sources that are
// executed by goja-based runtimes.
package tsscript

import (
	"path/filepath"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
)

// Options controls TypeScript transpilation and bundling.
//
// The zero value is usable: output targets ES2015 JavaScript for the neutral
// platform with CommonJS bundle output when bundling and IIFE output when
// transforming a source string.
type Options struct {
	Target    api.Target
	Format    api.Format
	Platform  api.Platform
	Sourcemap api.SourceMap
	JSX       api.JSX

	// External module names are preserved as require()/import references during
	// bundling. xgoja native module aliases should be listed here.
	External []string
	Define   map[string]string

	// Tsconfig is used by Build/BundleEntry. TsconfigRaw is used by Transform.
	Tsconfig    string
	TsconfigRaw string
	SourceRoot  string

	// LogLevel defaults to silent so callers get structured errors instead of
	// esbuild printing directly to stderr.
	LogLevel api.LogLevel
}

// Source describes one TypeScript/JavaScript input.
type Source struct {
	Path       string
	AbsPath    string
	ResolveDir string
	Contents   []byte
}

// Artifact is JavaScript output produced by esbuild.
type Artifact struct {
	Path       string
	Code       []byte
	SourceMap  []byte
	Warnings   []Diagnostic
	LoaderUsed api.Loader
	Bundled    bool
}

func defaultTarget(target api.Target) api.Target {
	if target == 0 {
		return api.ES2015
	}
	return target
}

func defaultPlatform(platform api.Platform) api.Platform {
	if platform == 0 {
		return api.PlatformNeutral
	}
	return platform
}

func defaultLogLevel(level api.LogLevel) api.LogLevel {
	if level == 0 {
		return api.LogLevelSilent
	}
	return level
}

func defaultTransformFormat(format api.Format) api.Format {
	if format == 0 {
		return api.FormatIIFE
	}
	return format
}

func defaultBundleFormat(format api.Format) api.Format {
	if format == 0 {
		return api.FormatCommonJS
	}
	return format
}

// IsTypeScriptPath reports whether path has a TypeScript-family extension.
func IsTypeScriptPath(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".ts", ".tsx", ".mts", ".cts":
		return true
	default:
		return false
	}
}

// LoaderForPath returns the esbuild loader implied by path's extension.
func LoaderForPath(path string) api.Loader {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".tsx":
		return api.LoaderTSX
	case ".ts", ".mts", ".cts":
		return api.LoaderTS
	case ".jsx":
		return api.LoaderJSX
	case ".json":
		return api.LoaderJSON
	default:
		return api.LoaderJS
	}
}
