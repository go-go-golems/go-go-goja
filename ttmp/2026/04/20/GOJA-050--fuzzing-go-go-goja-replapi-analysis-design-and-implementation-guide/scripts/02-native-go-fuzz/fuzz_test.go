// 02-native-go-fuzz: Go 1.18+ native fuzz test for replapi.Evaluate.
//
// This experiment uses the built-in Go fuzzing framework to exercise the
// replapi evaluation pipeline with automatically generated inputs.
//
// Run:
//
//	cd ttmp/.../scripts/02-native-go-fuzz/
//	go test -fuzz=FuzzEvaluateRaw -v -fuzztime=30s
//	go test -fuzz=FuzzEvaluateInstrumented -v -fuzztime=30s
//	go test -fuzz=FuzzSessionLifecycle -v -fuzztime=30s
package fuzz

import (
	"context"
	"testing"
	"time"

	"github.com/go-go-golems/go-go-goja/engine"
	"github.com/go-go-golems/go-go-goja/pkg/replapi"
	"github.com/rs/zerolog"
)

// newTestFactory creates a factory with default modules for testing.
func newTestFactory(t *testing.T) *engine.Factory {
	t.Helper()
	factory, err := engine.NewBuilder().WithModules(engine.DefaultRegistryModules()).Build() //nolint:staticcheck
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}
	return factory
}

// newRawApp creates an App in raw mode (no store needed).
func newRawApp(t *testing.T) *replapi.App {
	t.Helper()
	app, err := replapi.New(newTestFactory(t), zerolog.Nop(), replapi.WithProfile(replapi.ProfileRaw))
	if err != nil {
		t.Fatalf("create raw app: %v", err)
	}
	return app
}

// newInteractiveApp creates an App in interactive mode.
func newInteractiveApp(t *testing.T) *replapi.App {
	t.Helper()
	app, err := replapi.New(newTestFactory(t), zerolog.Nop(), replapi.WithProfile(replapi.ProfileInteractive))
	if err != nil {
		t.Fatalf("create interactive app: %v", err)
	}
	return app
}

// safeEvaluate runs Evaluate with panic recovery and a timeout.
func safeEvaluate(ctx context.Context, app *replapi.App, sessionID, source string) (bool, any) {
	var didPanic bool
	var panicVal any
	defer func() {
		if r := recover(); r != nil {
			didPanic = true
			panicVal = r
		}
	}()

	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, _ = app.Evaluate(timeoutCtx, sessionID, source)
	return didPanic, panicVal
}

// FuzzEvaluateRaw fuzzes the raw evaluation path.
func FuzzEvaluateRaw(f *testing.F) {
	// Seed corpus: interesting edge cases
	seeds := []string{
		"",
		"1",
		"const x = 1; x",
		"function f() {}; f()",
		"throw new Error('x')",
		"undefined.property",
		"null.toString()",
		"({})",
		"[]",
		"JSON.parse('null')",
		"new Promise(r => r(1))",
		"while(true){}",
		"for(var i=0;i<100;i++){}",
		"try { throw 1 } catch(e) { e }",
		"class A {}; new A()",
		"async function af() { return 1 }; af()",
		"const a = []; a.length = 1000000; a",
		"Object.create(null)",
		"String.fromCharCode(0, 0xFFFF, 0x10FFFF)",
		"eval('1+1')",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, source string) {
		ctx := context.Background()
		app := newRawApp(t)
		session, err := app.CreateSession(ctx)
		if err != nil {
			t.Fatalf("create session: %v", err)
		}

		panicked, panicVal := safeEvaluate(ctx, app, session.ID, source)
		if panicked {
			t.Fatalf("panic on raw evaluate with source %q: %v", truncate(source, 100), panicVal)
		}
	})
}

// FuzzEvaluateInstrumented fuzzes the instrumented (interactive) evaluation path.
func FuzzEvaluateInstrumented(f *testing.F) {
	seeds := []string{
		"const x = 1; x",
		"let a = 1; let b = 2; a + b",
		"function add(a,b) { return a + b }; add(1,2)",
		"class Point { constructor(x,y) { this.x=x; this.y=y } }; new Point(1,2)",
		"var messy = [1, 'two', { three: 3 }]; messy",
		"console.log('hello'); 42",
		"const obj = { get prop() { return 1 } }; obj.prop",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, source string) {
		ctx := context.Background()
		app := newInteractiveApp(t)
		session, err := app.CreateSession(ctx)
		if err != nil {
			t.Fatalf("create session: %v", err)
		}

		panicked, panicVal := safeEvaluate(ctx, app, session.ID, source)
		if panicked {
			t.Fatalf("panic on instrumented evaluate with source %q: %v", truncate(source, 100), panicVal)
		}
	})
}

// FuzzSessionLifecycle fuzzes multi-evaluate session sequences.
func FuzzSessionLifecycle(f *testing.F) {
	// Seeds are pairs: (first evaluate, second evaluate)
	type pair struct{ a, b string }
	seeds := []pair{
		{"const x = 1", "x + 1"},
		{"let y = 'hello'", "y + ' world'"},
		{"function f() { return 42 }", "f()"},
		{"var z = null", "z"},
	}
	for _, s := range seeds {
		f.Add(s.a, s.b)
	}

	f.Fuzz(func(t *testing.T, first, second string) {
		ctx := context.Background()
		app := newRawApp(t)
		session, err := app.CreateSession(ctx)
		if err != nil {
			t.Fatalf("create session: %v", err)
		}

		panicked, _ := safeEvaluate(ctx, app, session.ID, first)
		if panicked {
			return // first input caused an expected error; skip second
		}

		panicked, panicVal := safeEvaluate(ctx, app, session.ID, second)
		if panicked {
			t.Fatalf("panic on second evaluate first=%q second=%q: %v",
				truncate(first, 50), truncate(second, 50), panicVal)
		}
	})
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
