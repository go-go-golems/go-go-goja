package obsidiancli

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/pkg/errors"
)

// CallOptions describes one invocation of an Obsidian CLI command.
type CallOptions struct {
	Parameters map[string]any
	Flags      []string
	Positional []string
	Vault      string
}

// BuildArgs serializes a command spec and call options into argv entries.
func BuildArgs(cfg Config, spec CommandSpec, call CallOptions) ([]string, error) {
	if strings.TrimSpace(spec.Name) == "" {
		return nil, errors.New("obsidiancli: command spec name is empty")
	}

	args := make([]string, 0, 2+len(call.Parameters)+len(call.Flags)+len(call.Positional))
	vault := strings.TrimSpace(call.Vault)
	if vault == "" {
		vault = strings.TrimSpace(cfg.Vault)
	}
	if vault != "" {
		args = append(args, "vault="+vault)
	}
	args = append(args, spec.Name)

	if len(call.Parameters) > 0 {
		keys := make([]string, 0, len(call.Parameters))
		for key := range call.Parameters {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			formatted, err := formatParameter(key, call.Parameters[key])
			if err != nil {
				return nil, err
			}
			if formatted != "" {
				args = append(args, formatted)
			}
		}
	}

	flags := normalizeFlags(call.Flags)
	args = append(args, flags...)
	args = append(args, call.Positional...)
	return args, nil
}

func normalizeFlags(flags []string) []string {
	if len(flags) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	ret := make([]string, 0, len(flags))
	for _, flag := range flags {
		flag = strings.TrimSpace(flag)
		if flag == "" {
			continue
		}
		if _, ok := seen[flag]; ok {
			continue
		}
		seen[flag] = struct{}{}
		ret = append(ret, flag)
	}
	sort.Strings(ret)
	return ret
}

func formatParameter(key string, value any) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", errors.New("obsidiancli: parameter key is empty")
	}
	if value == nil {
		return "", nil
	}

	formatted, err := stringifyValue(value)
	if err != nil {
		return "", errors.Wrapf(err, "obsidiancli: format parameter %q", key)
	}
	return key + "=" + formatted, nil
}

func stringifyValue(value any) (string, error) {
	if value == nil {
		return "", nil
	}

	switch v := value.(type) {
	case string:
		return v, nil
	case fmt.Stringer:
		return v.String(), nil
	case bool:
		if v {
			return "true", nil
		}
		return "false", nil
	case []string:
		return strings.Join(v, ","), nil
	case []byte:
		return string(v), nil
	}

	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		parts := make([]string, 0, rv.Len())
		for i := range rv.Len() {
			part, err := stringifyValue(rv.Index(i).Interface())
			if err != nil {
				return "", err
			}
			parts = append(parts, part)
		}
		return strings.Join(parts, ","), nil
	case reflect.Map, reflect.Struct:
		raw, err := json.Marshal(value)
		if err != nil {
			return "", err
		}
		return string(raw), nil
	}
	return fmt.Sprint(value), nil
}
