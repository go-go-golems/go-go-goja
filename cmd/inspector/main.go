package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dop251/goja/parser"
	"github.com/go-go-golems/go-go-goja/cmd/inspector/app"
	"github.com/go-go-golems/go-go-goja/pkg/jsparse"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: goja-inspector <file.js>\n")
		os.Exit(1)
	}

	filename := os.Args[1]
	filename = filepath.Clean(filename)
	if filename == "." {
		fmt.Fprintln(os.Stderr, "Error: invalid input file path")
		os.Exit(1)
	}
	// #nosec G304,G703 -- inspector intentionally reads the user-selected source file path.
	src, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input file: %v\n", err)
		os.Exit(1)
	}

	sourceText := string(src)

	// Parse the source
	program, parseErr := parser.ParseFile(nil, filename, sourceText, 0)

	// Build the index and resolve scopes (even partial parse can produce an AST)
	var index *jsparse.Index
	if program != nil {
		index = jsparse.BuildIndex(program, sourceText)
		index.Resolution = jsparse.Resolve(program, index)
	}

	model := app.NewModel(filename, sourceText, program, parseErr, index)

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
