// Package analysis provides reusable helpers for working with jsparse analysis results
// in the context of an inspector-style UI.
package analysis

import (
	"github.com/go-go-golems/go-go-goja/pkg/jsparse"
)

// Session wraps an AnalysisResult with inspector-specific convenience methods.
type Session struct {
	Result *jsparse.AnalysisResult
}

// NewSession creates a new analysis session from a filename and source string.
func NewSession(filename, source string) *Session {
	result := jsparse.Analyze(filename, source, nil)
	return &Session{Result: result}
}

// NewSessionFromResult wraps an existing analysis result.
func NewSessionFromResult(result *jsparse.AnalysisResult) *Session {
	return &Session{Result: result}
}

// GlobalBindings returns the top-level bindings from the root scope.
func (s *Session) GlobalBindings() map[string]*jsparse.BindingRecord {
	if s.Result == nil || s.Result.Resolution == nil {
		return nil
	}
	rootScope := s.Result.Resolution.Scopes[s.Result.Resolution.RootScopeID]
	if rootScope == nil {
		return nil
	}
	return rootScope.Bindings
}
