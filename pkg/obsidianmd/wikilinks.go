package obsidianmd

import (
	"regexp"
	"strings"
)

var wikilinkPattern = regexp.MustCompile(`\[\[([^\]]+)\]\]`)

// ExtractWikilinks returns bare wikilink targets from markdown text.
func ExtractWikilinks(raw string) []string {
	matches := wikilinkPattern.FindAllStringSubmatch(raw, -1)
	if len(matches) == 0 {
		return nil
	}

	ret := make([]string, 0, len(matches))
	for _, match := range matches {
		target := strings.TrimSpace(match[1])
		if target == "" {
			continue
		}
		if idx := strings.Index(target, "|"); idx >= 0 {
			target = strings.TrimSpace(target[:idx])
		}
		if idx := strings.Index(target, "#"); idx >= 0 {
			target = strings.TrimSpace(target[:idx])
		}
		if target == "" {
			continue
		}
		ret = append(ret, target)
	}
	return ret
}
