package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/dtsgen"
	"github.com/spf13/cobra"
)

func (h *Host) TypeScriptDeclarations(opts dtsgen.Options) (*dtsgen.Result, error) {
	if h == nil {
		return nil, fmt.Errorf("xgoja host is nil")
	}
	return dtsgen.RenderModules(h.Providers, dtsgenModuleInstances(h.RuntimePlan), opts)
}

func dtsgenModuleInstances(runtimePlan *RuntimePlan) []dtsgen.ModuleInstance {
	if runtimePlan == nil || len(runtimePlan.runtimeModules()) == 0 {
		return nil
	}
	out := make([]dtsgen.ModuleInstance, 0, len(runtimePlan.runtimeModules()))
	for _, mod := range runtimePlan.runtimeModules() {
		out = append(out, dtsgen.ModuleInstance{Package: mod.ProviderID(), Name: mod.Name, As: mod.As})
	}
	return out
}

func (h *Host) WriteTypeScriptDeclarations(w io.Writer, opts dtsgen.Options) error {
	if w == nil {
		return fmt.Errorf("writer is nil")
	}
	result, err := h.TypeScriptDeclarations(opts)
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, result.DTS)
	return err
}

func (h *Host) AttachTypes(root *cobra.Command) {
	if root == nil || h == nil {
		return
	}
	out := h.Out
	if out == nil {
		out = root.OutOrStdout()
	}
	cmd, err := buildGlazedCobraCommand(newTypesCommand(h, out), h.MiddlewaresFunc)
	if err != nil {
		root.AddCommand(commandErrorStub("types", "Print TypeScript declarations for this generated xgoja runtime", err))
		return
	}
	root.AddCommand(cmd)
}

type typesCommand struct {
	*cmds.CommandDescription
	host *Host
	out  io.Writer
}

var _ cmds.BareCommand = (*typesCommand)(nil)

type typesSettings struct {
	Out    string `glazed:"out"`
	Check  string `glazed:"check"`
	Strict bool   `glazed:"strict"`
}

func newTypesCommand(host *Host, out io.Writer) cmds.Command {
	return &typesCommand{
		CommandDescription: cmds.NewCommandDescription("types",
			cmds.WithShort("Print TypeScript declarations for this generated xgoja runtime"),
			cmds.WithLong("Print or write a .d.ts declaration file for the require() modules selected into this generated xgoja runtime."),
			cmds.WithFlags(
				fields.New("out", fields.TypeString, fields.WithHelp("Write TypeScript declarations to this .d.ts file instead of stdout")),
				fields.New("check", fields.TypeString, fields.WithHelp("Fail if generated declarations differ from this .d.ts file")),
				fields.New("strict", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Fail if any selected module has no TypeScript descriptor")),
			),
		),
		host: host,
		out:  out,
	}
}

func (c *typesCommand) Run(ctx context.Context, vals *values.Values) error {
	_ = ctx
	settings := typesSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return err
	}
	result, err := c.host.TypeScriptDeclarations(dtsgen.Options{Strict: settings.Strict})
	if err != nil {
		return err
	}
	content := dtsWithTrailingNewline(result.DTS)
	switch {
	case settings.Check != "":
		data, err := os.ReadFile(settings.Check)
		if err != nil {
			return fmt.Errorf("read %s: %w", settings.Check, err)
		}
		if string(data) != content {
			return fmt.Errorf("generated TypeScript declarations differ from %s", settings.Check)
		}
		return nil
	case settings.Out != "":
		return os.WriteFile(settings.Out, []byte(content), 0o644)
	default:
		out := c.out
		if out == nil {
			out = io.Discard
		}
		_, err = io.WriteString(out, content)
		return err
	}
}

func dtsWithTrailingNewline(content string) string {
	if strings.HasSuffix(content, "\n") {
		return content
	}
	return content + "\n"
}
