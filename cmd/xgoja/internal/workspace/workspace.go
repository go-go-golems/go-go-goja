package workspace

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Mode string

const (
	ModeOff  Mode = "off"
	ModeAuto Mode = "auto"
	ModePath Mode = "path"
)

type Spec struct {
	Mode Mode
	File string
}

type Module struct {
	Path string
	Dir  string
}

type ResolutionKind string

const (
	ResolutionVersioned ResolutionKind = "versioned"
	ResolutionReplace   ResolutionKind = "replace"
	ResolutionWorkspace ResolutionKind = "workspace"
)

type ResolutionSource string

const (
	SourceVersion         ResolutionSource = "version"
	SourceExplicitReplace ResolutionSource = "explicit-replace"
	SourceCLIReplace      ResolutionSource = "cli-replace"
	SourceGoWork          ResolutionSource = "go-work"
)

type Requirement struct {
	ModulePath      string
	Version         string
	ExplicitReplace string
	RequiredBy      []string
}

type Options struct {
	Spec       Spec
	StartDir   string
	CLIReplace map[string]string
}

type GoModulePlan struct {
	ModulePath       string
	Version          string
	LocalDir         string
	RequiredBy       []string
	ResolutionKind   ResolutionKind
	ResolutionSource ResolutionSource
}

type Plan struct {
	WorkspaceFile string
	Modules       []GoModulePlan
}

func Resolve(requirements []Requirement, opts Options) (*Plan, error) {
	workspaceModules, workspaceFile, err := loadWorkspaceModules(opts.Spec, opts.StartDir)
	if err != nil {
		return nil, err
	}
	out := &Plan{WorkspaceFile: workspaceFile}
	for _, req := range requirements {
		modulePath := strings.TrimSpace(req.ModulePath)
		if modulePath == "" {
			continue
		}
		planned := GoModulePlan{
			ModulePath:       modulePath,
			Version:          strings.TrimSpace(req.Version),
			RequiredBy:       append([]string(nil), req.RequiredBy...),
			ResolutionKind:   ResolutionVersioned,
			ResolutionSource: SourceVersion,
		}
		if replacement := strings.TrimSpace(req.ExplicitReplace); replacement != "" {
			planned.LocalDir = absMaybe(opts.StartDir, replacement)
			planned.ResolutionKind = ResolutionReplace
			planned.ResolutionSource = SourceExplicitReplace
		} else if replacement := strings.TrimSpace(opts.CLIReplace[modulePath]); replacement != "" {
			planned.LocalDir = absMaybe(opts.StartDir, replacement)
			planned.ResolutionKind = ResolutionReplace
			planned.ResolutionSource = SourceCLIReplace
		} else if module, ok := workspaceModules[modulePath]; ok {
			planned.LocalDir = module.Dir
			planned.ResolutionKind = ResolutionWorkspace
			planned.ResolutionSource = SourceGoWork
		}
		out.Modules = append(out.Modules, planned)
	}
	return out, nil
}

func loadWorkspaceModules(spec Spec, startDir string) (map[string]Module, string, error) {
	mode := spec.Mode
	if mode == "" {
		mode = ModeAuto
	}
	switch mode {
	case ModeOff:
		return map[string]Module{}, "", nil
	case ModeAuto:
		file, err := FindGoWork(startDir)
		if err != nil || file == "" {
			return map[string]Module{}, "", err
		}
		modules, err := ParseGoWork(file)
		return modules, file, err
	case ModePath:
		if strings.TrimSpace(spec.File) == "" {
			return nil, "", fmt.Errorf("workspace.file is required when workspace.mode is path")
		}
		file := absMaybe(startDir, spec.File)
		modules, err := ParseGoWork(file)
		return modules, file, err
	default:
		return nil, "", fmt.Errorf("unsupported workspace mode %q", spec.Mode)
	}
}

func FindGoWork(startDir string) (string, error) {
	if strings.TrimSpace(startDir) == "" {
		startDir = "."
	}
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(dir, "go.work")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		} else if err != nil && !os.IsNotExist(err) {
			return "", err
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", nil
		}
		dir = parent
	}
}

type goWorkJSON struct {
	Use []struct {
		DiskPath string
	} `json:"Use"`
}

func ParseGoWork(path string) (map[string]Module, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("go.work path is required")
	}
	cmd := exec.Command("go", "work", "edit", "-json", path)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	data, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("parse go.work %s: %w: %s", path, err, strings.TrimSpace(stderr.String()))
	}
	parsed := goWorkJSON{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("decode go.work json %s: %w", path, err)
	}
	base := filepath.Dir(path)
	modules := map[string]Module{}
	for _, use := range parsed.Use {
		dir := use.DiskPath
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(base, dir)
		}
		modulePath, err := ReadGoModModulePath(filepath.Join(dir, "go.mod"))
		if err != nil {
			return nil, err
		}
		modules[modulePath] = Module{Path: modulePath, Dir: filepath.Clean(dir)}
	}
	return modules, nil
}

func ReadGoModModulePath(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read go.mod %s: %w", path, err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			modulePath := strings.TrimSpace(strings.TrimPrefix(line, "module "))
			modulePath = strings.Trim(modulePath, "\"")
			if modulePath == "" {
				return "", fmt.Errorf("go.mod %s has empty module path", path)
			}
			return modulePath, nil
		}
	}
	return "", fmt.Errorf("go.mod %s has no module directive", path)
}

func absMaybe(baseDir, path string) string {
	path = strings.TrimSpace(path)
	if path == "" || filepath.IsAbs(path) || strings.TrimSpace(baseDir) == "" {
		return filepath.Clean(path)
	}
	return filepath.Clean(filepath.Join(baseDir, path))
}
