package specv2

import (
	"path/filepath"
	"strings"
)

func ApplyDefaults(cfg *Config) {
	if cfg == nil {
		return
	}
	cfg.Schema = strings.TrimSpace(cfg.Schema)
	cfg.Name = strings.TrimSpace(cfg.Name)
	cfg.App.Name = strings.TrimSpace(cfg.App.Name)
	cfg.App.EnvPrefix = strings.TrimSpace(cfg.App.EnvPrefix)
	if cfg.Schema == "" {
		cfg.Schema = Schema
	}
	if cfg.Name == "" {
		cfg.Name = "xgoja-app"
	}
	if strings.TrimSpace(cfg.App.Name) == "" {
		cfg.App.Name = cfg.Name
	}
	if cfg.App.ConfigFile != nil && cfg.App.ConfigFile.Enabled && strings.TrimSpace(cfg.App.ConfigFile.FileName) == "" {
		cfg.App.ConfigFile.FileName = "config.yaml"
	}
	if strings.TrimSpace(cfg.Go.Version) == "" {
		cfg.Go.Version = "1.26"
	}
	if strings.TrimSpace(cfg.Go.Module) == "" {
		cfg.Go.Module = "xgoja.generated/" + sanitizeModulePathPart(cfg.Name)
	}
	if strings.TrimSpace(cfg.Workspace.Mode) == "" {
		cfg.Workspace.Mode = "auto"
	}
	for i := range cfg.Providers {
		cfg.Providers[i].ID = strings.TrimSpace(cfg.Providers[i].ID)
		cfg.Providers[i].Import = strings.TrimSpace(cfg.Providers[i].Import)
		cfg.Providers[i].Register = strings.TrimSpace(cfg.Providers[i].Register)
		if cfg.Providers[i].Register == "" {
			cfg.Providers[i].Register = "Register"
		}
	}
	for i := range cfg.Sources {
		applySourceDefaults(&cfg.Sources[i])
	}
	for i := range cfg.Artifacts {
		cfg.Artifacts[i].ID = strings.TrimSpace(cfg.Artifacts[i].ID)
		cfg.Artifacts[i].Type = strings.TrimSpace(cfg.Artifacts[i].Type)
		if cfg.Artifacts[i].Type == "binary" && strings.TrimSpace(cfg.Artifacts[i].Output) == "" {
			cfg.Artifacts[i].Output = filepath.ToSlash(filepath.Join("dist", sanitizeModulePathPart(cfg.Name)))
		}
	}
}

func applySourceDefaults(source *SourceSpec) {
	if source == nil {
		return
	}
	source.ID = strings.TrimSpace(source.ID)
	source.Language = strings.TrimSpace(source.Language)
	if source.Language == "" && source.Kind == SourceKindJSVerbs {
		source.Language = inferLanguageFromExtensions(source.Extensions)
	}
	if source.Compile == nil && sourceNeedsCompilePolicy(source.Kind, source.Language) {
		source.Compile = &CompileSpec{}
	}
	if source.Compile != nil && strings.TrimSpace(source.Compile.Mode) == "" {
		switch source.Kind {
		case SourceKindJSVerbs, SourceKindScript:
			source.Compile.Mode = "runtime"
		case SourceKindAssets, SourceKindHelp:
			source.Compile.Mode = "preserve"
		default:
			source.Compile.Mode = "preserve"
		}
	}
}

func sourceNeedsCompilePolicy(kind SourceKind, language string) bool {
	switch kind {
	case SourceKindJSVerbs, SourceKindScript:
		return strings.EqualFold(language, "typescript")
	case SourceKindAssets, SourceKindHelp:
		return false
	default:
		return false
	}
}

func inferLanguageFromExtensions(exts []string) string {
	for _, ext := range exts {
		switch strings.ToLower(strings.TrimSpace(ext)) {
		case ".ts", ".tsx", ".mts", ".cts":
			return "typescript"
		}
	}
	return "javascript"
}

func sanitizeModulePathPart(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	lastDash := false
	for _, r := range s {
		valid := r >= 'a' && r <= 'z' || r >= '0' && r <= '9'
		if valid {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash && b.Len() > 0 {
			b.WriteRune('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "xgoja-app"
	}
	return out
}
