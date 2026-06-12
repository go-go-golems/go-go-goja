package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/buildexec"
	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/generate"
	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/plan"
	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/workspace"
)

type genDTSCommand struct {
	*cmds.CommandDescription
	out io.Writer
}

var _ cmds.BareCommand = (*genDTSCommand)(nil)

type genDTSSettings struct {
	File         string `glazed:"file"`
	Output       string `glazed:"out"`
	Check        bool   `glazed:"check"`
	Strict       bool   `glazed:"strict"`
	WorkDir      string `glazed:"work-dir"`
	KeepWork     bool   `glazed:"keep-work"`
	XGojaVersion string `glazed:"xgoja-version"`
	XGojaReplace string `glazed:"xgoja-replace"`
}

func newGenDTSCommand(out io.Writer) *genDTSCommand {
	return &genDTSCommand{
		CommandDescription: cmds.NewCommandDescription("gen-dts",
			cmds.WithShort("Generate TypeScript declarations for an xgoja build spec"),
			cmds.WithLong(`
Generate a .d.ts declaration file for the require() modules selected by an
xgoja.yaml build spec.

The command uses a generated sidecar Go program so provider packages listed in
the build spec are imported and registered exactly like generated xgoja binaries.
This makes gen-dts work for third-party providers that are not linked into the
precompiled xgoja CLI.

Examples:
  xgoja gen-dts -f xgoja.yaml --out js/types/xgoja-modules.d.ts
  xgoja gen-dts -f xgoja.yaml --out js/types/xgoja-modules.d.ts --strict
  xgoja gen-dts -f xgoja.yaml --out js/types/xgoja-modules.d.ts --check
  xgoja gen-dts -f xgoja.yaml --xgoja-replace /path/to/go-go-goja --keep-work
`),
			cmds.WithFlags(
				fields.New("file", fields.TypeString,
					fields.WithDefault("xgoja.yaml"),
					fields.WithShortFlag("f"),
					fields.WithHelp("Path to the xgoja build specification")),
				fields.New("out", fields.TypeString,
					fields.WithHelp("Output path for generated .d.ts file; v2 defaults to the first dts artifact output")),
				fields.New("check", fields.TypeBool,
					fields.WithDefault(false),
					fields.WithHelp("Check mode: fail if generated output differs from --out")),
				fields.New("strict", fields.TypeBool,
					fields.WithDefault(false),
					fields.WithHelp("Fail if any selected module has no TypeScript descriptor")),
				fields.New("work-dir", fields.TypeString,
					fields.WithHelp("Directory for generated sidecar files; defaults to a temporary directory")),
				fields.New("keep-work", fields.TypeBool,
					fields.WithDefault(false),
					fields.WithHelp("Keep the generated sidecar directory after completion or failure")),
				fields.New("xgoja-version", fields.TypeString,
					fields.WithDefault(defaultXGojaModuleVersion()),
					fields.WithHelp("go-go-goja module version required by generated sidecar go.mod when --xgoja-replace is not set")),
				fields.New("xgoja-replace", fields.TypeString,
					fields.WithHelp("Optional local replacement path for github.com/go-go-golems/go-go-goja in generated sidecar go.mod")),
			),
		),
		out: out,
	}
}

func (c *genDTSCommand) Run(ctx context.Context, vals *values.Values) error {
	settings := genDTSSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return err
	}
	compiledPlan, err := loadV2Plan(settings.File)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(c.out, "validated xgoja/v2 plan for %s\n", settings.File)
	applyV2DTSArtifactDefaults(&settings, compiledPlan)

	workDir := strings.TrimSpace(settings.WorkDir)
	cleanup := func() {}
	if workDir == "" {
		tmp, err := os.MkdirTemp("", "xgoja-dts-*_")
		if err != nil {
			return fmt.Errorf("create temporary dts workspace: %w", err)
		}
		workDir = tmp
		if !settings.KeepWork {
			cleanup = func() { _ = os.RemoveAll(tmp) }
		}
	}
	defer cleanup()

	goModules := compiledPlan.GoModules
	if strings.TrimSpace(settings.Output) == "" {
		return fmt.Errorf("--out is required unless the v2 spec has a dts artifact with output")
	}
	if err := writeDTSSidecar(workDir, compiledPlan, settings, goModules); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(c.out, "generated dts sidecar workspace: %s\n", workDir)
	if settings.WorkDir == "" && !settings.KeepWork {
		_, _ = fmt.Fprintln(c.out, "use --keep-work to inspect generated dts sidecar files")
	}
	if _, err := buildexec.GoModTidy(ctx, workDir); err != nil {
		return err
	}
	result, err := buildexec.GoRun(ctx, workDir, compiledPlan.Config.Go.Env)
	if err != nil {
		return err
	}
	return writeOrCheckDTS(settings.Output, result.Stdout, settings.Check)
}

func applyV2DTSArtifactDefaults(settings *genDTSSettings, compiledPlan *plan.Plan) {
	if settings == nil || compiledPlan == nil {
		return
	}
	for _, artifact := range compiledPlan.Artifacts {
		if artifact.Spec.Type != "dts" {
			continue
		}
		if strings.TrimSpace(settings.Output) == "" {
			settings.Output = artifact.Spec.Output
		}
		if artifact.Spec.Strict {
			settings.Strict = true
		}
		return
	}
}

func writeDTSSidecar(dir string, compiledPlan *plan.Plan, settings genDTSSettings, goModules *workspace.Plan) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create dts sidecar directory %s: %w", dir, err)
	}
	files := map[string]string{
		"go.mod":  generate.RenderGoMod(generate.BuildSpecFromPlan(compiledPlan), generate.Options{XGojaModuleVersion: settings.XGojaVersion, XGojaReplace: settings.XGojaReplace, GoModules: goModules}),
		"main.go": generate.RenderDTSGenMainPlan(compiledPlan, settings.Strict),
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			return fmt.Errorf("write dts sidecar %s: %w", name, err)
		}
	}
	return nil
}

func writeOrCheckDTS(path string, content string, check bool) error {
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
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
