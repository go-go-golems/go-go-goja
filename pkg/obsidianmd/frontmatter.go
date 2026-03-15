package obsidianmd

import (
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// Document is a parsed markdown document split into frontmatter and body.
type Document struct {
	Frontmatter map[string]any
	Body        string
}

// ParseDocument splits optional YAML frontmatter from the markdown body.
func ParseDocument(raw string) (Document, error) {
	if !strings.HasPrefix(raw, "---\n") && raw != "---" {
		return Document{Body: raw}, nil
	}

	lines := strings.Split(raw, "\n")
	if len(lines) == 0 || lines[0] != "---" {
		return Document{Body: raw}, nil
	}

	end := -1
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			end = i
			break
		}
	}
	if end == -1 {
		return Document{}, errors.New("obsidianmd: unterminated frontmatter block")
	}

	fmRaw := strings.Join(lines[1:end], "\n")
	body := strings.Join(lines[end+1:], "\n")
	if fmRaw == "" {
		return Document{Frontmatter: map[string]any{}, Body: body}, nil
	}

	var fm map[string]any
	if err := yaml.Unmarshal([]byte(fmRaw), &fm); err != nil {
		return Document{}, errors.Wrap(err, "obsidianmd: parse frontmatter")
	}
	return Document{
		Frontmatter: fm,
		Body:        body,
	}, nil
}
