package tsscript

import (
	"fmt"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
)

// Diagnostic is a compact, stable representation of an esbuild message.
type Diagnostic struct {
	Text   string
	File   string
	Line   int
	Column int
}

func diagnosticsFromMessages(messages []api.Message) []Diagnostic {
	if len(messages) == 0 {
		return nil
	}
	out := make([]Diagnostic, 0, len(messages))
	for _, msg := range messages {
		d := Diagnostic{Text: msg.Text}
		if msg.Location != nil {
			d.File = msg.Location.File
			d.Line = msg.Location.Line
			d.Column = msg.Location.Column
		}
		out = append(out, d)
	}
	return out
}

func errorFromMessages(op string, messages []api.Message) error {
	if len(messages) == 0 {
		return nil
	}
	parts := make([]string, 0, len(messages))
	for _, msg := range diagnosticsFromMessages(messages) {
		location := strings.TrimSpace(msg.File)
		if msg.Line > 0 {
			if location == "" {
				location = fmt.Sprintf("%d:%d", msg.Line, msg.Column)
			} else {
				location = fmt.Sprintf("%s:%d:%d", location, msg.Line, msg.Column)
			}
		}
		if location == "" {
			parts = append(parts, msg.Text)
			continue
		}
		parts = append(parts, fmt.Sprintf("%s: %s", location, msg.Text))
	}
	return fmt.Errorf("%s failed: %s", op, strings.Join(parts, "; "))
}
