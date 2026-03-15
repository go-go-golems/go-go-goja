package obsidianmd

import (
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// NoteSection is one named markdown section.
type NoteSection struct {
	Title string
	Body  string
}

// NoteTemplate describes a markdown note to render.
type NoteTemplate struct {
	Title       string
	Frontmatter map[string]any
	WikiTags    []string
	Body        string
	Sections    []NoteSection
}

// BuildNote renders a markdown note with optional frontmatter and sections.
func BuildNote(tpl NoteTemplate) (string, error) {
	lines := []string{}

	title := strings.TrimSpace(tpl.Title)
	if title != "" {
		lines = append(lines, "# "+title)
	}

	if len(tpl.Frontmatter) > 0 {
		raw, err := yaml.Marshal(tpl.Frontmatter)
		if err != nil {
			return "", err
		}
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, "---")
		lines = append(lines, strings.TrimRight(string(raw), "\n"))
		lines = append(lines, "---")
	}

	if wikiTags := renderWikiTags(tpl.WikiTags); wikiTags != "" {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, wikiTags)
	}

	body := strings.TrimSpace(tpl.Body)
	if body != "" {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, body)
	}

	sections := normalizeSections(tpl.Sections)
	for _, section := range sections {
		if strings.TrimSpace(section.Title) == "" {
			continue
		}
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, "## "+strings.TrimSpace(section.Title))
		if body := strings.TrimSpace(section.Body); body != "" {
			lines = append(lines, body)
		}
	}

	return strings.Join(lines, "\n") + "\n", nil
}

func renderWikiTags(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	seen := map[string]struct{}{}
	ret := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		tag = strings.Trim(tag, "[]")
		if tag == "" {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		ret = append(ret, "[["+tag+"]]")
	}
	return strings.Join(ret, " ")
}

func normalizeSections(sections []NoteSection) []NoteSection {
	if len(sections) == 0 {
		return nil
	}
	preferred := map[string]int{
		"Brainstorm": 0,
		"Links":      1,
		"Logs":       2,
	}
	ret := append([]NoteSection(nil), sections...)
	sort.SliceStable(ret, func(i, j int) bool {
		pi, okI := preferred[ret[i].Title]
		pj, okJ := preferred[ret[j].Title]
		switch {
		case okI && okJ:
			return pi < pj
		case okI:
			return true
		case okJ:
			return false
		default:
			return ret[i].Title < ret[j].Title
		}
	})
	return ret
}
