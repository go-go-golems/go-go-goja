package obsidiancli

import (
	"encoding/json"
	"strings"

	"github.com/pkg/errors"
)

// ParseOutput parses stdout according to the requested output kind.
func ParseOutput(kind OutputKind, stdout string) (any, error) {
	switch kind {
	case OutputRaw:
		return stdout, nil
	case OutputJSON:
		var ret any
		if err := json.Unmarshal([]byte(stdout), &ret); err != nil {
			return nil, errors.Wrap(err, "obsidiancli: parse json output")
		}
		return ret, nil
	case OutputLineList:
		return ParseLineList(stdout), nil
	case OutputKeyValue:
		return ParseKeyValue(stdout)
	default:
		return nil, errors.Errorf("obsidiancli: unknown output kind %q", kind)
	}
}

// ParseLineList parses line-oriented output into a string slice.
func ParseLineList(stdout string) []string {
	lines := strings.Split(stdout, "\n")
	ret := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		ret = append(ret, line)
	}
	return ret
}

// ParseKeyValue parses simple key/value output into a map.
func ParseKeyValue(stdout string) (map[string]string, error) {
	ret := map[string]string{}
	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		sep := strings.IndexAny(line, "=:")
		if sep == -1 {
			return nil, errors.Errorf("obsidiancli: invalid key/value line %q", line)
		}
		key := strings.TrimSpace(line[:sep])
		value := strings.TrimSpace(line[sep+1:])
		if key == "" {
			return nil, errors.Errorf("obsidiancli: invalid key/value line %q", line)
		}
		ret[key] = value
	}
	return ret, nil
}
