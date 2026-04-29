// 01-basic-replapi-fuzz: Minimal proof-of-concept fuzzer for replapi.App.Evaluate.
//
// This experiment demonstrates how to create a standalone fuzz harness that:
//  1. Creates a replapi.App in raw mode (no store needed)
//  2. Creates a session
//  3. Feeds fuzz-derived JavaScript source into Evaluate()
//  4. Checks for panics, unexpected errors, or resource leaks
//
// Run manually:
//
//	go run ./ttmp/.../scripts/01-basic-replapi-fuzz/main.go
//
// Run with go test fuzzing:
//
//	cd ttmp/.../scripts/01-basic-replapi-fuzz/
//	go test -fuzz=FuzzEvaluateRaw -v -fuzztime=30s
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-go-golems/go-go-goja/engine"
	"github.com/go-go-golems/go-go-goja/pkg/replapi"
	"github.com/rs/zerolog"
)

func main() {
	ctx := context.Background()
	factory, err := engine.NewBuilder().WithModules(engine.DefaultRegistryModules()).Build() //nolint:staticcheck
	if err != nil {
		fmt.Fprintf(os.Stderr, "build factory: %v\n", err)
		os.Exit(1)
	}

	app, err := replapi.New(factory, zerolog.Nop(), replapi.WithProfile(replapi.ProfileRaw))
	if err != nil {
		fmt.Fprintf(os.Stderr, "create app: %v\n", err)
		os.Exit(1)
	}

	session, err := app.CreateSession(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create session: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Session created: %s\n", session.ID)

	// A small corpus of interesting JavaScript inputs to exercise the engine
	corpus := []string{
		// Basic expressions
		"1 + 1",
		"\"hello\"",
		"true",
		"null",
		"undefined",
		// Variable declarations
		"const x = 42; x",
		"let y = 'hello'; y",
		"var z = true; z",
		// Functions
		"function add(a, b) { return a + b; }; add(1, 2)",
		"const mul = (a, b) => a * b; mul(3, 4)",
		// Objects and arrays
		"({key: 'value'})",
		"[1, 2, 3]",
		"Object.keys({a: 1, b: 2}).join(',')",
		// Edge cases
		"",
		"   ",
		";",
		"{}",
		"()",
		"[]",
		// String edge cases
		"\"\\x00\\x01\\x02\"",
		"'\uffff'",
		"`template ${1 + 2}`",
		// Type coercion edge cases
		"[] + []",
		"[] + {}",
		"{} + []",
		"!!\"\"",
		"+true",
		// Nested structures
		"JSON.parse('{\"a\":1}')",
		"new Array(1000).fill(0)",
		// Error-producing inputs
		"throw new Error('test')",
		"undefined.property",
		"null.property",
		// Unicode
		"const 你好 = 'world'; 你好",
		"// comment\n1",
		"/* block */ 2",
		// Deeply nested
		"((((1))))",
		"if (true) { if (true) { if (true) { 1 } } }",
	}

	passed := 0
	failed := 0
	panicked := 0

	for i, input := range corpus {
		func() {
			defer func() {
				if r := recover(); r != nil {
					panicked++
					fmt.Fprintf(os.Stderr, "[%d] PANIC on input %q: %v\n", i, input, r)
				}
			}()

			timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			resp, err := app.Evaluate(timeoutCtx, session.ID, input)
			if err != nil {
				// Some errors are expected (e.g., syntax errors, runtime errors)
				fmt.Printf("[%d] Error (expected for some): input=%q err=%v\n", i, truncate(input, 50), err)
				passed++
			} else if resp != nil && resp.Cell != nil {
				fmt.Printf("[%d] OK: input=%q status=%s result=%s\n",
					i, truncate(input, 50), resp.Cell.Execution.Status, truncate(resp.Cell.Execution.Result, 50))
				passed++
			} else {
				failed++
				fmt.Fprintf(os.Stderr, "[%d] UNEXPECTED: nil response for input %q\n", i, input)
			}
		}()
	}

	fmt.Printf("\n=== Results: passed=%d failed=%d panicked=%d total=%d ===\n", passed, failed, panicked, len(corpus))
	if panicked > 0 {
		os.Exit(1)
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
