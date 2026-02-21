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
	finalModel, err := p.Run()
	if finalModel != nil {
		switch fm := finalModel.(type) {
		case app.Model:
			fm.Close()
		case *app.Model:
			fm.Close()
		}
	} else {
		model.Close()
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
