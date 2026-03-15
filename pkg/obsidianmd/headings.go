package obsidianmd

import (
	"bufio"
	"regexp"
	"strings"
)

var headingPattern = regexp.MustCompile(`^(#{1,6})\s+(.+?)\s*$`)

// Heading is one markdown heading.
type Heading struct {
	Level int
	Text  string
	Line  int
}

// ExtractHeadings returns markdown headings in source order.
func ExtractHeadings(raw string) []Heading {
	scanner := bufio.NewScanner(strings.NewReader(raw))
	ret := []Heading{}
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		match := headingPattern.FindStringSubmatch(line)
		if len(match) == 0 {
			continue
		}
		ret = append(ret, Heading{
			Level: len(match[1]),
			Text:  strings.TrimSpace(match[2]),
			Line:  lineNo,
		})
	}
	return ret
}
