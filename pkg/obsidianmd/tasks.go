package obsidianmd

import (
	"bufio"
	"regexp"
	"strings"
)

var taskPattern = regexp.MustCompile(`^\s*[-*]\s+\[([ xX])\]\s+(.+?)\s*$`)

// Task is one markdown checkbox item.
type Task struct {
	Text string
	Done bool
	Line int
}

// ExtractTasks returns markdown checkbox tasks in source order.
func ExtractTasks(raw string) []Task {
	scanner := bufio.NewScanner(strings.NewReader(raw))
	ret := []Task{}
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		match := taskPattern.FindStringSubmatch(scanner.Text())
		if len(match) == 0 {
			continue
		}
		ret = append(ret, Task{
			Text: strings.TrimSpace(match[2]),
			Done: strings.EqualFold(match[1], "x"),
			Line: lineNo,
		})
	}
	return ret
}
