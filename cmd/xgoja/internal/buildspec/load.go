package buildspec

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func LoadFile(path string) (*Spec, *Report, error) {
	if strings.TrimSpace(path) == "" {
		path = "xgoja.yaml"
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve spec path %q: %w", path, err)
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, nil, fmt.Errorf("read spec %s: %w", abs, err)
	}

	spec := &Spec{}
	if err := yaml.Unmarshal(data, spec); err != nil {
		return nil, nil, fmt.Errorf("parse spec %s: %w", abs, err)
	}
	spec.BaseDir = filepath.Dir(abs)
	applyDefaults(spec)

	report := Validate(spec)
	if report.HasErrors() {
		return spec, report, &ValidationError{Report: report}
	}
	return spec, report, nil
}

func applyDefaults(spec *Spec) {
	if spec == nil {
		return
	}
	spec.Name = strings.TrimSpace(spec.Name)
	if spec.Name == "" {
		spec.Name = "xgoja-app"
	}
	if strings.TrimSpace(spec.Go.Version) == "" {
		spec.Go.Version = "1.26"
	}
	if strings.TrimSpace(spec.Go.Module) == "" {
		spec.Go.Module = "example.com/generated/" + sanitizeModulePathPart(spec.Name)
	}
	if strings.TrimSpace(spec.Target.Kind) == "" {
		spec.Target.Kind = "xgoja"
	}
	if strings.TrimSpace(spec.Target.Output) == "" {
		spec.Target.Output = filepath.ToSlash(filepath.Join("dist", sanitizeModulePathPart(spec.Name)))
	}
	for i := range spec.Packages {
		if strings.TrimSpace(spec.Packages[i].Register) == "" {
			spec.Packages[i].Register = "Register"
		}
	}
	if spec.Commands.Eval.Enabled && strings.TrimSpace(spec.Commands.Eval.Name) == "" {
		spec.Commands.Eval.Name = "eval"
	}
	if spec.Commands.Run.Enabled && strings.TrimSpace(spec.Commands.Run.Name) == "" {
		spec.Commands.Run.Name = "run"
	}
	if spec.Commands.Repl.Enabled && strings.TrimSpace(spec.Commands.Repl.Name) == "" {
		spec.Commands.Repl.Name = "repl"
	}
	if spec.Commands.JSVerbs.Enabled && strings.TrimSpace(spec.Commands.JSVerbs.Name) == "" {
		spec.Commands.JSVerbs.Name = "verbs"
	}
}

func sanitizeModulePathPart(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	lastDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == '-' || r == '_' || r == ' ' || r == '.':
			if !lastDash && b.Len() > 0 {
				b.WriteRune('-')
				lastDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "xgoja-app"
	}
	return out
}
