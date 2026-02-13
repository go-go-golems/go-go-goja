package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-go-golems/go-go-goja/cmd/ast-parse-editor/app"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: ast-parse-editor <file.js>\n")
		os.Exit(1)
	}

	filename := filepath.Clean(os.Args[1])
	if filename == "." {
		fmt.Fprintln(os.Stderr, "Error: invalid input file path")
		os.Exit(1)
	}

	// #nosec G304,G703 -- this command intentionally reads a user-provided file path.
	src, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input file: %v\n", err)
		os.Exit(1)
	}

	model := app.NewModel(filename, string(src))
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
