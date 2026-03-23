package modules

import "github.com/go-go-golems/go-go-goja/pkg/tsgen/spec"

// TypeScriptDeclarer is an optional interface for modules that can provide a
// static TypeScript declaration descriptor for generation.
type TypeScriptDeclarer interface {
	TypeScriptModule() *spec.Module
}
