package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-go-golems/go-go-goja/modules"
	_ "github.com/go-go-golems/go-go-goja/modules/database"
	_ "github.com/go-go-golems/go-go-goja/modules/exec"
	_ "github.com/go-go-golems/go-go-goja/modules/fs"
	"github.com/go-go-golems/go-go-goja/pkg/tsgen/render"
	"github.com/go-go-golems/go-go-goja/pkg/tsgen/spec"
	"github.com/go-go-golems/go-go-goja/pkg/tsgen/validate"
)

type options struct {
	Out     string
	Modules []string
	Strict  bool
	Check   bool
	Header  string
}

func main() {
	if err := run(os.Args[1:], os.Stderr); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "gen-dts: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, stderr io.Writer) error {
	opts, err := parseOptions(args)
	if err != nil {
		return err
	}

	out, err := generateFromModules(modules.ListDefaultModules(), opts)
	if err != nil {
		return err
	}

	if !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	return writeOrCheck(opts.Out, out, opts.Check)
}

func parseOptions(args []string) (options, error) {
	fs := flag.NewFlagSet("gen-dts", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	out := fs.String("out", "", "Output path for generated .d.ts file")
	moduleCSV := fs.String("module", "", "Comma-separated module names to include (optional)")
	strict := fs.Bool("strict", false, "Fail if selected module has no descriptor")
	check := fs.Bool("check", false, "Check mode: fail if generated output differs from --out")
	header := fs.String("header", "", "Optional generated file header comment")

	if err := fs.Parse(args); err != nil {
		return options{}, err
	}
	if strings.TrimSpace(*out) == "" {
		return options{}, fmt.Errorf("--out is required")
	}

	modules := parseModuleCSV(*moduleCSV)

	return options{
		Out:     strings.TrimSpace(*out),
		Modules: modules,
		Strict:  *strict,
		Check:   *check,
		Header:  strings.TrimSpace(*header),
	}, nil
}

func parseModuleCSV(moduleCSV string) []string {
	if strings.TrimSpace(moduleCSV) == "" {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0)
	for _, raw := range strings.Split(moduleCSV, ",") {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func generateFromModules(allModules []modules.NativeModule, opts options) (string, error) {
	selectedSet := map[string]struct{}{}
	for _, name := range opts.Modules {
		selectedSet[name] = struct{}{}
	}

	foundSelected := map[string]struct{}{}
	descriptors := make([]*spec.Module, 0)

	for _, module := range allModules {
		if module == nil {
			continue
		}
		name := strings.TrimSpace(module.Name())
		if name == "" {
			continue
		}

		if len(selectedSet) > 0 {
			if _, ok := selectedSet[name]; !ok {
				continue
			}
			foundSelected[name] = struct{}{}
		}

		typedModule, ok := module.(modules.TypeScriptDeclarer)
		if !ok {
			if opts.Strict {
				return "", fmt.Errorf("module %q has no TypeScript descriptor", name)
			}
			continue
		}

		descriptor := typedModule.TypeScriptModule()
		if descriptor == nil {
			return "", fmt.Errorf("module %q returned nil TypeScript descriptor", name)
		}
		if strings.TrimSpace(descriptor.Name) == "" {
			descriptor.Name = name
		}
		if strings.TrimSpace(descriptor.Name) != name {
			return "", fmt.Errorf("descriptor name %q does not match module name %q", descriptor.Name, name)
		}
		if err := validate.Module(descriptor); err != nil {
			return "", err
		}
		descriptors = append(descriptors, descriptor)
	}

	if len(selectedSet) > 0 {
		missing := make([]string, 0)
		for name := range selectedSet {
			if _, ok := foundSelected[name]; !ok {
				missing = append(missing, name)
			}
		}
		if len(missing) > 0 {
			sort.Strings(missing)
			return "", fmt.Errorf("requested module(s) not found: %s", strings.Join(missing, ", "))
		}
	}

	bundle := &spec.Bundle{
		HeaderComment: opts.Header,
		Modules:       descriptors,
	}
	if err := validate.Bundle(bundle); err != nil {
		return "", err
	}
	return render.Bundle(bundle)
}

func writeOrCheck(path string, content string, check bool) error {
	if check {
		existing, err := os.ReadFile(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("--check failed: %s does not exist", path)
			}
			return fmt.Errorf("read %s: %w", path, err)
		}
		if string(existing) != content {
			return fmt.Errorf("--check failed: generated output differs from %s", path)
		}
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
