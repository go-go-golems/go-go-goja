package docaccess

import (
	"fmt"
	"strings"
)

func MatchesQuery(entry Entry, q Query) bool {
	if len(q.Kinds) > 0 && !containsNormalized(q.Kinds, entry.Ref.Kind) {
		return false
	}
	if len(q.Topics) > 0 && !containsAnyNormalized(entry.Topics, q.Topics) {
		return false
	}
	if len(q.Tags) > 0 && !containsAnyNormalized(entry.Tags, q.Tags) {
		return false
	}
	if strings.TrimSpace(q.Text) != "" && !matchesText(entry, q.Text) {
		return false
	}
	return true
}

func matchesText(entry Entry, text string) bool {
	text = strings.ToLower(strings.TrimSpace(text))
	if text == "" {
		return true
	}

	var b strings.Builder
	b.WriteString(entry.Title)
	b.WriteString("\n")
	b.WriteString(entry.Summary)
	b.WriteString("\n")
	b.WriteString(entry.Body)
	b.WriteString("\n")
	b.WriteString(entry.Path)
	for key, value := range entry.Metadata {
		b.WriteString("\n")
		b.WriteString(key)
		b.WriteString(": ")
		_, _ = fmt.Fprint(&b, value)
	}

	return strings.Contains(strings.ToLower(b.String()), text)
}

func containsNormalized(values []string, target string) bool {
	target = strings.TrimSpace(strings.ToLower(target))
	if target == "" {
		return false
	}
	for _, value := range values {
		if strings.TrimSpace(strings.ToLower(value)) == target {
			return true
		}
	}
	return false
}

func containsAnyNormalized(values []string, targets []string) bool {
	for _, value := range values {
		if containsNormalized(targets, value) {
			return true
		}
	}
	return false
}
