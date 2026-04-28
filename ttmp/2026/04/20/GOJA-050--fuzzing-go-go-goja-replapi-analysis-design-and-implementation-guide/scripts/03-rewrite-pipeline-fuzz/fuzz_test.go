// 03-rewrite-pipeline-fuzz: Fuzzer targeting the source rewrite (async IIFE wrapping) pipeline.
//
// This experiment fuzzes the instrumented execution path, which includes:
//  1. jsparse.Analyze (parsing)
//  2. buildRewrite (async IIFE wrapping + binding capture)
//  3. executeWrapped (runtime execution of rewritten code)
//  4. persistWrappedReturn (binding extraction from IIFE result)
//
// Run:
//
//	cd ttmp/.../scripts/03-rewrite-pipeline-fuzz/
//	go test -fuzz=FuzzRewritePipeline -v -fuzztime=30s
package fuzz

import (
	"context"
	"testing"
	"time"

	"github.com/go-go-golems/go-go-goja/engine"
	"github.com/go-go-golems/go-go-goja/pkg/replapi"
	"github.com/rs/zerolog"
)

func newFactory(t *testing.T) *engine.Factory {
	t.Helper()
	factory, err := engine.NewBuilder().WithModules(engine.DefaultRegistryModules()).Build() //nolint:staticcheck
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}
	return factory
}

// FuzzRewritePipeline fuzzes the full instrumented pipeline including rewriting.
func FuzzRewritePipeline(f *testing.F) {
	seeds := []string{
		// Declarations that the rewrite captures
		"const x = 1",
		"let y = 2; y",
		"var z = 3",
		"const a = 1, b = 2",
		"function hello() { return 'world' }",
		"class Foo { bar() { return 1 } }",
		// Expression-only (last expression capture)
		"1 + 2",
		"'hello'.toUpperCase()",
		// Mixed
		"const x = 1; x + 1",
		"let a = [1,2,3]; a.map(x => x * 2)",
		// Async/await
		"async function delay() { return 1 }; delay()",
		// Destructuring
		"const { a, b } = { a: 1, b: 2 }; a + b",
		"const [x, y] = [1, 2]; x + y",
		// Spread
		"const obj = { ...{ a: 1 }, b: 2 }; obj",
		"const arr = [...[1, 2], 3]; arr",
		// Template literals
		"const name = 'world'; `hello ${name}`",
		// Try/catch
		"try { throw 'err' } catch(e) { e }",
		// With semicolons
		"const x = 1;; x",
		// Getter/setter
		"const obj = { get x() { return 1 } }; obj.x",
		// Computed property
		"const key = 'a'; const obj = { [key]: 1 }; obj",
		// Symbol
		"const s = Symbol('test'); const obj = { [s]: 1 }; obj[s]",
		// Generator
		"function* gen() { yield 1; yield 2 }",
		// Default parameters
		"const f = (x = 1) => x; f()",
		// Rest parameters
		"const f = (...args) => args; f(1, 2, 3)",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, source string) {
		ctx := context.Background()
		factory := newFactory(t)
		app, err := replapi.New(factory, zerolog.Nop(), replapi.WithProfile(replapi.ProfileInteractive))
		if err != nil {
			t.Fatalf("create app: %v", err)
		}

		session, err := app.CreateSession(ctx)
		if err != nil {
			t.Fatalf("create session: %v", err)
		}

		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("PANIC in rewrite pipeline source=%q: %v", truncate(source, 80), r)
				}
			}()

			timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			_, err := app.Evaluate(timeoutCtx, session.ID, source)
			// Errors are expected for invalid JS, but panics are not
			_ = err
		}()
	})
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
