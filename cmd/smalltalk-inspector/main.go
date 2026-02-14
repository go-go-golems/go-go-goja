package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-go-golems/go-go-goja/cmd/smalltalk-inspector/app"
)

func main() {
	var filename string
	if len(os.Args) > 1 {
		filename = os.Args[1]
	}

	model := app.NewModel(filename)

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
