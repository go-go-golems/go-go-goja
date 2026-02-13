package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-go-golems/go-go-goja/cmd/ast-parse-editor/app"
)

const defaultFilename = "untitled.js"

func loadInput(args []string) (string, string, error) {
	switch len(args) {
	case 0:
		return defaultFilename, "", nil
	case 1:
		filename := filepath.Clean(args[0])
		if filename == "." || filename == "" {
			return "", "", fmt.Errorf("invalid input file path")
		}

		// #nosec G304,G703 -- this command intentionally reads a user-provided file path.
		src, err := os.ReadFile(filename)
		if err != nil {
			if os.IsNotExist(err) {
				return filename, "", nil
			}
			return "", "", fmt.Errorf("reading input file %q: %w", filename, err)
		}
		return filename, string(src), nil
	default:
		return "", "", fmt.Errorf("usage: ast-parse-editor [file.js]")
	}
}

func main() {
	filename, src, err := loadInput(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	model := app.NewModel(filename, src)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
