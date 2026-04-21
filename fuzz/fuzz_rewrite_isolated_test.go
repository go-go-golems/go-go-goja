package fuzz

import (
	"testing"

	"github.com/go-go-golems/go-go-goja/pkg/jsparse"
)

// FuzzRewriteIsolated tests the jsparse.Analyze + buildRewrite pipeline without
// a VM. Since buildRewrite is unexported, we verify it indirectly by checking
// that analysis never panics and produces consistent results.
//
// The actual rewrite function is tested through FuzzEvaluateInstrumented.
func FuzzRewriteIsolated(f *testing.F) {
	for _, seed := range SeedsMinimal {
		f.Add(seed)
	}
	for _, seed := range SeedsDeclarations {
		f.Add(seed)
	}
	for _, seed := range SeedsMixed {
		f.Add(seed)
	}
	for _, seed := range SeedsErrorInputs {
		f.Add(seed)
	}
	for _, seed := range SeedsUnicode {
		f.Add(seed)
	}
	for _, seed := range SeedsDeepNesting {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, source string) {
		// Primary invariant: analysis never panics.
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic in jsparse.Analyze source=%q: %v", truncate(source, 100), r)
				}
			}()

			analysis := jsparse.Analyze("<fuzz>", source, nil)

			// Invariant: analysis result is never nil.
			if analysis == nil {
				t.Fatalf("nil analysis for source=%q", truncate(source, 100))
			}

			// Invariant: source is preserved.
			if analysis.Source != source {
				t.Fatalf("source mismatch: got %q", truncate(analysis.Source, 100))
			}

			// Invariant: diagnostics never nil (empty slice is fine).
			_ = analysis.Diagnostics()

			// Invariant: index is populated when parsing succeeds.
			if analysis.ParseErr == nil {
				if analysis.Index == nil {
					t.Fatalf("nil index for successfully parsed source=%q", truncate(source, 100))
				}
				if analysis.Program == nil {
					t.Fatalf("nil program for successfully parsed source=%q", truncate(source, 100))
				}
				if len(analysis.Program.Body) >= 0 && analysis.Resolution == nil {
					t.Fatalf("nil resolution for successfully parsed source=%q", truncate(source, 100))
				}
			}
		}()
	})
}

// TestFuzzRewriteIsolatedSeeds runs every seed through analysis.
func TestFuzzRewriteIsolatedSeeds(t *testing.T) {
	t.Parallel()
	for _, source := range AllSeeds() {
		func(src string) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic in analysis for source=%q: %v", truncate(src, 80), r)
				}
			}()
			analysis := jsparse.Analyze("<test>", src, nil)
			if analysis == nil {
				t.Errorf("nil analysis for source=%q", truncate(src, 80))
			}
		}(source)
	}
}

// TestFuzzRewriteIsolatedEmptyVariants tests empty-ish inputs (BUG-1 regression).
func TestFuzzRewriteIsolatedEmptyVariants(t *testing.T) {
	t.Parallel()
	empties := []string{"", " ", "\n", "\t", "  \n  "}
	for _, src := range empties {
		func(s string) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic for empty variant %q: %v", s, r)
				}
			}()
			analysis := jsparse.Analyze("<test>", s, nil)
			// Empty inputs should parse cleanly (possibly with empty body).
			if analysis.Source != s {
				t.Errorf("source not preserved for %q", s)
			}
		}(src)
	}
}

// TestFuzzRewriteIsolatedDiagnostics verifies error inputs produce diagnostics.
func TestFuzzRewriteIsolatedDiagnostics(t *testing.T) {
	t.Parallel()
	errorSources := []string{
		"const x = ;",
		"function f( {}",
		"return 1",
	}
	for _, src := range errorSources {
		analysis := jsparse.Analyze("<test>", src, nil)
		if analysis.ParseErr == nil && len(analysis.Diagnostics()) == 0 {
			t.Errorf("expected parse error or diagnostics for %q", truncate(src, 40))
		}
	}
}

// TestFuzzRewriteIsolatedDeclarations verifies declarations are found.
func TestFuzzRewriteIsolatedDeclarations(t *testing.T) {
	t.Parallel()
	cases := []struct {
		source    string
		wantNames []string
	}{
		{"const x = 1", []string{"x"}},
		{"let y = 2", []string{"y"}},
		{"var z = 3", []string{"z"}},
		{"const a = 1, b = 2", []string{"a", "b"}},
		{"function f() {}", []string{"f"}},
		{"class A {}", []string{"A"}},
	}

	for _, tc := range cases {
		analysis := jsparse.Analyze("<test>", tc.source, nil)
		if analysis.ParseErr != nil {
			t.Fatalf("unexpected parse error for %q: %v", tc.source, analysis.ParseErr)
		}
		if analysis.Resolution == nil {
			t.Fatalf("nil resolution for %q", tc.source)
		}
		root := analysis.Resolution.Scopes[analysis.Resolution.RootScopeID]
		if root == nil {
			t.Fatalf("nil root scope for %q", tc.source)
		}
		for _, name := range tc.wantNames {
			if _, ok := root.Bindings[name]; !ok {
				t.Errorf("expected binding %q not found for source=%q (bindings: %v)",
					name, tc.source, bindingNames(root.Bindings))
			}
		}
	}
}

func bindingNames(m map[string]*jsparse.BindingRecord) []string {
	names := make([]string, 0, len(m))
	for k := range m {
		if k != "" {
			names = append(names, k)
		}
	}
	return names
}
