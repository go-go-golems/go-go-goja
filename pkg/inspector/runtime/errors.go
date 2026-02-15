package runtime

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/dop251/goja"
)

// StackFrame represents a single frame from a JS exception stack trace.
type StackFrame struct {
	FunctionName string
	FileName     string
	Line         int
	Column       int
}

// ErrorInfo holds parsed information from a goja exception.
type ErrorInfo struct {
	Message string
	Frames  []StackFrame
	Raw     string
}

// ParseException extracts structured error info from a goja Exception.
func ParseException(ex *goja.Exception) ErrorInfo {
	raw := ex.String()
	lines := strings.Split(raw, "\n")

	info := ErrorInfo{Raw: raw}
	if len(lines) > 0 {
		info.Message = strings.TrimSpace(lines[0])
	}

	for _, line := range lines[1:] {
		frame := parseStackLine(strings.TrimSpace(line))
		if frame != nil {
			info.Frames = append(info.Frames, *frame)
		}
	}

	return info
}

// stackLineRe matches goja stack trace lines like:
//
//	at functionName (file:line:col(offset))
//	at functionName (<eval>:line:col(offset))
var stackLineRe = regexp.MustCompile(
	`^at\s+(\S+)\s+\(([^:]+):(\d+):(\d+)`,
)

func parseStackLine(line string) *StackFrame {
	m := stackLineRe.FindStringSubmatch(line)
	if m == nil {
		return nil
	}

	lineNum, _ := strconv.Atoi(m[3])
	colNum, _ := strconv.Atoi(m[4])

	return &StackFrame{
		FunctionName: m[1],
		FileName:     m[2],
		Line:         lineNum,
		Column:       colNum,
	}
}
