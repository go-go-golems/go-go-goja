package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/migratebuildspec"
	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/specv2"
)

type migrateSpecCommand struct {
	*cmds.CommandDescription
	out io.Writer
}

var _ cmds.BareCommand = (*migrateSpecCommand)(nil)

type migrateSpecSettings struct {
	File    string `glazed:"file"`
	Out     string `glazed:"out"`
	InPlace bool   `glazed:"in-place"`
	Backup  bool   `glazed:"backup"`
	Check   bool   `glazed:"check"`
	From    string `glazed:"from"`
	To      string `glazed:"to"`
}

func newMigrateSpecCommand(out io.Writer) *migrateSpecCommand {
	return &migrateSpecCommand{
		CommandDescription: cmds.NewCommandDescription("migrate-spec",
			cmds.WithShort("Convert legacy xgoja.yaml specs to xgoja/v2"),
			cmds.WithLong(`
Migrate-spec converts legacy v1 xgoja.yaml files into the native xgoja/v2
schema. The hard v2 cutover keeps v1 parsing here as migration input only;
normal build, doctor, and generation paths should be v2-native.

Examples:
  xgoja migrate-spec -f xgoja.yaml --out xgoja.v2.yaml
  xgoja migrate-spec -f xgoja.yaml --in-place --backup
  xgoja migrate-spec -f xgoja.yaml --check
`),
			cmds.WithFlags(
				fields.New("file", fields.TypeString,
					fields.WithDefault("xgoja.yaml"),
					fields.WithShortFlag("f"),
					fields.WithHelp("Path to the xgoja specification to migrate")),
				fields.New("out", fields.TypeString,
					fields.WithHelp("Path to write the migrated v2 specification; stdout when omitted and --in-place is false")),
				fields.New("in-place", fields.TypeBool,
					fields.WithDefault(false),
					fields.WithHelp("Overwrite the input file with the migrated v2 specification")),
				fields.New("backup", fields.TypeBool,
					fields.WithDefault(false),
					fields.WithHelp("When --in-place is set, first write a .bak copy of the original file")),
				fields.New("check", fields.TypeBool,
					fields.WithDefault(false),
					fields.WithHelp("Return non-zero if the file is not already in rendered xgoja/v2 form")),
				fields.New("from", fields.TypeString,
					fields.WithDefault("v1"),
					fields.WithHelp("Source schema version; currently v1")),
				fields.New("to", fields.TypeString,
					fields.WithDefault("v2"),
					fields.WithHelp("Target schema version; currently v2")),
			),
		),
		out: out,
	}
}

func (c *migrateSpecCommand) Run(ctx context.Context, vals *values.Values) error {
	_ = ctx
	settings := migrateSpecSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return err
	}
	if strings.TrimSpace(settings.From) != "v1" {
		return fmt.Errorf("unsupported --from %q; only v1 is supported", settings.From)
	}
	if strings.TrimSpace(settings.To) != "v2" {
		return fmt.Errorf("unsupported --to %q; only v2 is supported", settings.To)
	}
	if settings.InPlace && strings.TrimSpace(settings.Out) != "" {
		return fmt.Errorf("--out and --in-place are mutually exclusive")
	}
	if settings.Backup && !settings.InPlace {
		return fmt.Errorf("--backup requires --in-place")
	}

	inputPath := settings.File
	if strings.TrimSpace(inputPath) == "" {
		inputPath = "xgoja.yaml"
	}
	original, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("read xgoja spec %s: %w", inputPath, err)
	}
	rendered, warnings, err := migrateSpecFileData(inputPath, original)
	if err != nil {
		return err
	}
	for _, warning := range warnings {
		_, _ = fmt.Fprintf(c.out, "warning: %s\n", warning.String())
	}

	changed := !bytes.Equal(bytes.TrimSpace(original), bytes.TrimSpace(rendered))
	if settings.Check {
		if changed {
			return fmt.Errorf("%s is not in rendered xgoja/v2 form", inputPath)
		}
		_, _ = fmt.Fprintf(c.out, "%s is already in rendered xgoja/v2 form\n", inputPath)
		return nil
	}

	if settings.InPlace {
		if settings.Backup {
			backupPath := inputPath + ".bak"
			if err := os.WriteFile(backupPath, original, 0o644); err != nil {
				return fmt.Errorf("write backup %s: %w", backupPath, err)
			}
			_, _ = fmt.Fprintf(c.out, "wrote backup %s\n", backupPath)
		}
		if err := os.WriteFile(inputPath, append(rendered, '\n'), 0o644); err != nil {
			return fmt.Errorf("write migrated spec %s: %w", inputPath, err)
		}
		_, _ = fmt.Fprintf(c.out, "migrated %s to xgoja/v2\n", inputPath)
		return nil
	}

	if strings.TrimSpace(settings.Out) != "" {
		outPath := settings.Out
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil && filepath.Dir(outPath) != "." {
			return fmt.Errorf("create output directory for %s: %w", outPath, err)
		}
		if err := os.WriteFile(outPath, append(rendered, '\n'), 0o644); err != nil {
			return fmt.Errorf("write migrated spec %s: %w", outPath, err)
		}
		_, _ = fmt.Fprintf(c.out, "wrote migrated xgoja/v2 spec %s\n", outPath)
		return nil
	}

	_, err = c.out.Write(append(rendered, '\n'))
	return err
}

func migrateSpecFileData(path string, data []byte) ([]byte, []specv2.MigrationWarning, error) {
	kind, _, err := specv2.DetectSchema(data)
	if err != nil {
		return nil, nil, fmt.Errorf("detect xgoja schema %s: %w", path, err)
	}
	switch kind {
	case specv2.SchemaKindV2:
		cfg, err := specv2.LoadData(data)
		if err != nil {
			return nil, nil, err
		}
		rendered, err := specv2.Render(*cfg)
		return rendered, nil, err
	case specv2.SchemaKindV1:
		legacySpec, _, err := migratebuildspec.LoadFile(path)
		if err != nil {
			return nil, nil, err
		}
		result := specv2.MigrateV1(legacySpec)
		rendered, err := specv2.Render(result.Config)
		return rendered, result.Warnings, err
	case specv2.SchemaKindUnknown:
		return nil, nil, fmt.Errorf("unsupported xgoja schema in %s", path)
	default:
		return nil, nil, fmt.Errorf("unsupported xgoja schema in %s", path)
	}
}
