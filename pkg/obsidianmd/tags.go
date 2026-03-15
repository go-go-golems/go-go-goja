package obsidianmd

import (
	"bufio"
	"regexp"
	"strings"
)

var tagPattern = regexp.MustCompile(`(^|\s)#([A-Za-z0-9][A-Za-z0-9_-]*)`)

// ExtractTags returns hashtag tokens without the leading '#'.
func ExtractTags(raw string) []string {
	scanner := bufio.NewScanner(strings.NewReader(raw))
	ret := []string{}
	seen := map[string]struct{}{}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") {
			if headingPattern.MatchString(line) {
				continue
			}
		}
		matches := tagPattern.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			tag := match[2]
			if _, ok := seen[tag]; ok {
				continue
			}
			seen[tag] = struct{}{}
			ret = append(ret, tag)
		}
	}
	return ret
}
