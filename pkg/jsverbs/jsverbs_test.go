package jsverbs

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/dop251/goja"
	noderequire "github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/runner"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/go-go-golems/go-go-goja/engine"
	"github.com/stretchr/testify/require"
)

func TestScanDirDiscoversExpectedPaths(t *testing.T) {
	registry := mustRegistry(t)

	paths := []string{}
	for _, verb := range registry.Verbs() {
		paths = append(paths, verb.FullPath())
	}

	require.ElementsMatch(t, []string{
		"advanced numbers add",
		"advanced numbers list-names",
		"advanced numbers multiply",
		"basics banner",
		"basics echo",
		"basics greet",
		"basics list-issues",
		"basics summarize",
		"meta pkg-demo ping",
		"nested with-helper render",
	}, paths)
}

func TestFixtureCommandsExecute(t *testing.T) {
	registry := mustRegistry(t)
	commandMap := mustCommandMap(t, registry)

	t.Run("greet", func(t *testing.T) {
		rows := runCommand(t, commandMap["basics greet"], map[string]map[string]interface{}{
			"default": {
				"name":    "Manuel",
				"excited": true,
			},
		})
		require.Equal(t, "Hello, Manuel!", rows[0]["greeting"])
	})

	t.Run("echo primitive", func(t *testing.T) {
		rows := runCommand(t, commandMap["basics echo"], map[string]map[string]interface{}{
			"default": {
				"value": "plain-text",
			},
		})
		require.Equal(t, "plain-text", rows[0]["value"])
	})

	t.Run("writer output", func(t *testing.T) {
		text := runWriterCommand(t, commandMap["basics banner"], map[string]map[string]interface{}{
			"default": {
				"name": "Manuel",
			},
		})
		require.Equal(t, "=== Manuel ===\n", text)
	})

	t.Run("list issues uses shared section and context", func(t *testing.T) {
		rows := runCommand(t, commandMap["basics list-issues"], map[string]map[string]interface{}{
			"default": {
				"repo": "go-go-golems/go-go-goja",
			},
			"filters": {
				"state":  "closed",
				"labels": []string{"bug", "docs"},
			},
		})
		require.Equal(t, "go-go-golems/go-go-goja", rows[0]["repo"])
		require.Equal(t, "closed", rows[0]["state"])
		require.EqualValues(t, 2, rows[0]["labelCount"])
		require.Equal(t, "decorated:go-go-golems/go-go-goja", rows[0]["helper"])
		require.Equal(t, filepath.ToSlash(filepath.Join(repoRoot(t), "testdata", "jsverbs")), rows[0]["rootDir"])
	})

	t.Run("summarize bind all", func(t *testing.T) {
		rows := runCommand(t, commandMap["basics summarize"], map[string]map[string]interface{}{
			"default": {
				"owner": "go-go-golems",
				"repo":  "go-go-goja",
			},
		})
		require.Equal(t, "go-go-golems/go-go-goja", rows[0]["joined"])
	})

	t.Run("async multiply", func(t *testing.T) {
		rows := runCommand(t, commandMap["advanced numbers multiply"], map[string]map[string]interface{}{
			"default": {
				"a": 6,
				"b": 7,
			},
		})
		require.EqualValues(t, 42, rows[0]["product"])
	})

	t.Run("rest argument list", func(t *testing.T) {
		rows := runCommand(t, commandMap["advanced numbers list-names"], map[string]map[string]interface{}{
			"default": {
				"names": []string{"alice", "bob", "charlie"},
			},
		})
		require.Len(t, rows, 3)
		require.Equal(t, "alice", rows[0]["name"])
		require.Equal(t, "charlie", rows[2]["name"])
	})

	t.Run("relative require helper", func(t *testing.T) {
		rows := runCommand(t, commandMap["nested with-helper render"], map[string]map[string]interface{}{
			"default": {
				"prefix": "repo",
				"target": "glazed",
			},
		})
		require.Equal(t, "repo:glazed", rows[0]["value"])
	})

	t.Run("package metadata auto expose", func(t *testing.T) {
		rows := runCommand(t, commandMap["meta pkg-demo ping"], nil)
		require.Equal(t, true, rows[0]["ok"])
	})
}

func TestScanSourcesSupportsRawJS(t *testing.T) {
	registry, err := ScanSources([]SourceFile{
		{
			Path: "inline.js",
			Source: []byte(`
function ping(name) {
  return { greeting: "hi " + name };
}

__verb__("ping", {
  fields: {
    name: { argument: true }
  }
});
`),
		},
	})
	require.NoError(t, err)

	commandMap := mustCommandMap(t, registry)
	rows := runCommand(t, commandMap["inline ping"], map[string]map[string]interface{}{
		"default": {
			"name": "manuel",
		},
	})
	require.Equal(t, "hi manuel", rows[0]["greeting"])
}

func TestScanFSSupportsVirtualFiles(t *testing.T) {
	registry, err := ScanFS(fstest.MapFS{
		"nested/entry.js": {
			Data: []byte(`
function render(prefix, target) {
  const helper = require("./sub/helper");
  return { value: helper.decorate(prefix, target) };
}

__verb__("render", {
  fields: {
    prefix: { argument: true },
    target: { argument: true }
  }
});
`),
		},
		"nested/sub/helper.js": {
			Data: []byte(`exports.decorate = (prefix, target) => prefix + ":" + target;`),
		},
	}, ".", ScanOptions{
		IncludePublicFunctions: true,
		Extensions:             []string{".js"},
		FailOnErrorDiagnostics: true,
	})
	require.NoError(t, err)

	commandMap := mustCommandMap(t, registry)
	rows := runCommand(t, commandMap["nested entry render"], map[string]map[string]interface{}{
		"default": {
			"prefix": "repo",
			"target": "glazed",
		},
	})
	require.Equal(t, "repo:glazed", rows[0]["value"])
}

func TestScanDiagnosticsSurfaceInvalidMetadata(t *testing.T) {
	registry, err := ScanSource("broken.js", `
function greet() {
  return { ok: true };
}

__verb__("greet", {
  short: helper()
});
`)
	require.Error(t, err)
	require.NotNil(t, registry)
	require.Contains(t, err.Error(), "unsupported metadata literal")
	require.Len(t, registry.ErrorDiagnostics(), 1)
	require.Contains(t, registry.ErrorDiagnostics()[0].Message, "invalid __verb__ metadata")
}

func TestCommandsFailForUnknownBoundSection(t *testing.T) {
	registry, err := ScanSource("broken.js", `
function summarize(filters) {
  return { ok: !!filters };
}

__verb__("summarize", {
  fields: {
    filters: { bind: "missing" }
  }
});
`)
	require.NoError(t, err)

	_, err = registry.Commands()
	require.Error(t, err)
	require.Contains(t, err.Error(), `unknown section "missing"`)
}

func TestCommandsFailForObjectParamWithoutBind(t *testing.T) {
	registry, err := ScanSource("broken.js", `
function summarize({ owner }) {
  return { owner };
}
`)
	require.NoError(t, err)

	_, err = registry.Commands()
	require.Error(t, err)
	require.Contains(t, err.Error(), "requires a bind")
}

func TestAddSharedSectionRejectsDuplicateSlug(t *testing.T) {
	registry := &Registry{}

	err := registry.AddSharedSection(&SectionSpec{
		Slug:  "filters",
		Title: "Filters",
		Fields: map[string]*FieldSpec{
			"state": {Type: "string"},
		},
	})
	require.NoError(t, err)

	err = registry.AddSharedSection(&SectionSpec{
		Slug:  "filters",
		Title: "Other Filters",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), `duplicate shared section "filters"`)
}

func TestAddSharedSectionRejectsNilFieldSpec(t *testing.T) {
	registry := &Registry{}

	err := registry.AddSharedSection(&SectionSpec{
		Slug:  "filters",
		Title: "Filters",
		Fields: map[string]*FieldSpec{
			"state": nil,
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), `shared section "filters" field "state" is nil`)
}

func TestResolveSectionPrefersLocalSectionOverSharedSection(t *testing.T) {
	registry, err := ScanSource("example.js", `
__section__("filters", {
  title: "Local Filters",
  fields: {
    state: { type: "string", default: "closed" }
  }
});

function summarize(filters) {
  return { ok: !!filters };
}

__verb__("summarize", {
  fields: {
    filters: { bind: "filters" }
  }
});
`)
	require.NoError(t, err)
	require.NoError(t, registry.AddSharedSection(&SectionSpec{
		Slug:  "filters",
		Title: "Shared Filters",
		Fields: map[string]*FieldSpec{
			"state": {Type: "string", Default: "open"},
		},
	}))

	verb, ok := registry.Verb("example summarize")
	require.True(t, ok)

	section, ok := registry.ResolveSection(verb, "filters")
	require.True(t, ok)
	require.Equal(t, "Local Filters", section.Title)
	require.Equal(t, "closed", section.Fields["state"].Default)
}

func TestCommandsUseRegistrySharedSection(t *testing.T) {
	registry, err := ScanSource("shared.js", `
function summarize(filters) {
  return {
    state: filters.state,
    labelCount: filters.labels.length
  };
}

__verb__("summarize", {
  fields: {
    filters: { bind: "filters" }
  }
});
`)
	require.NoError(t, err)
	require.NoError(t, registry.AddSharedSection(&SectionSpec{
		Slug:  "filters",
		Title: "Filters",
		Fields: map[string]*FieldSpec{
			"state": {
				Type:    "choice",
				Choices: []string{"open", "closed"},
			},
			"labels": {
				Type: "stringList",
			},
		},
	}))

	commandMap := mustCommandMap(t, registry)
	rows := runCommand(t, commandMap["shared summarize"], map[string]map[string]interface{}{
		"filters": {
			"state":  "closed",
			"labels": []string{"bug", "docs"},
		},
	})
	require.Equal(t, "closed", rows[0]["state"])
	require.EqualValues(t, 2, rows[0]["labelCount"])
}

func TestLocalSectionOverridesRegistrySharedSectionDuringCommandExecution(t *testing.T) {
	registry, err := ScanSource("local.js", `
__section__("filters", {
  fields: {
    localOnly: { type: "string" }
  }
});

function summarize(filters) {
  return {
    local: filters.localOnly,
    shared: filters.sharedOnly
  };
}

__verb__("summarize", {
  fields: {
    filters: { bind: "filters" }
  }
});
`)
	require.NoError(t, err)
	require.NoError(t, registry.AddSharedSection(&SectionSpec{
		Slug:  "filters",
		Title: "Shared Filters",
		Fields: map[string]*FieldSpec{
			"sharedOnly": {Type: "string"},
		},
	}))

	commandMap := mustCommandMap(t, registry)
	rows := runCommand(t, commandMap["local summarize"], map[string]map[string]interface{}{
		"filters": {
			"localOnly": "from-local",
		},
	})
	require.Equal(t, "from-local", rows[0]["local"])
	require.Nil(t, rows[0]["shared"])
}

func TestSharedSectionsWorkWithScanFS(t *testing.T) {
	registry, err := ScanFS(fstest.MapFS{
		"nested/entry.js": {
			Data: []byte(`
function render(filters) {
  const helper = require("./sub/helper");
  return { state: filters.state, decorated: helper.decorate(filters.state) };
}

__verb__("render", {
  fields: {
    filters: { bind: "filters" }
  }
});
`),
		},
		"nested/sub/helper.js": {
			Data: []byte(`exports.decorate = (value) => "decorated:" + value;`),
		},
	}, ".", ScanOptions{
		IncludePublicFunctions: true,
		Extensions:             []string{".js"},
		FailOnErrorDiagnostics: true,
	})
	require.NoError(t, err)
	require.NoError(t, registry.AddSharedSection(&SectionSpec{
		Slug:  "filters",
		Title: "Filters",
		Fields: map[string]*FieldSpec{
			"state": {Type: "string"},
		},
	}))

	commandMap := mustCommandMap(t, registry)
	rows := runCommand(t, commandMap["nested entry render"], map[string]map[string]interface{}{
		"filters": {
			"state": "closed",
		},
	})
	require.Equal(t, "closed", rows[0]["state"])
	require.Equal(t, "decorated:closed", rows[0]["decorated"])
}

func TestCommandDescriptionForVerb(t *testing.T) {
	registry := mustRegistry(t)
	verb, ok := registry.Verb("basics greet")
	require.True(t, ok)
	desc, err := registry.CommandDescriptionForVerb(verb)
	require.NoError(t, err)
	require.Equal(t, "greet", desc.Name)
	require.NotNil(t, desc.Schema)
}

func TestInvokeInRuntimeReusesLiveRuntime(t *testing.T) {
	registry := mustRegistry(t)
	verb, ok := registry.Verb("basics list-issues")
	require.True(t, ok)

	requireOpt, err := engine.RequireOptionWithModuleRootsFromScript(
		filepath.Join(repoRoot(t), "testdata", "jsverbs", "basics.js"),
		engine.DefaultModuleRootsOptions(),
	)
	require.NoError(t, err)

	builder := engine.NewBuilder().
		WithRequireOptions(noderequire.WithLoader(registry.RequireLoader())).
		WithModules(engine.DefaultRegistryModules())
	if requireOpt != nil {
		builder = builder.WithRequireOptions(requireOpt)
	}
	factory, err := builder.Build()
	require.NoError(t, err)
	rt, err := factory.NewRuntime(context.Background())
	require.NoError(t, err)
	defer func() { _ = rt.Close(context.Background()) }()

	desc, err := registry.CommandDescriptionForVerb(verb)
	require.NoError(t, err)
	cmd := &Command{CommandDescription: desc, registry: registry, verb: verb}
	parsedValues, err := runner.ParseCommandValues(cmd, runner.WithValuesForSections(map[string]map[string]interface{}{
		"default": {
			"repo": "go-go-golems/go-go-goja",
		},
		"filters": {
			"state":  "closed",
			"labels": []string{"bug", "docs"},
		},
	}))
	require.NoError(t, err)

	result, err := registry.InvokeInRuntime(context.Background(), rt, verb, parsedValues)
	require.NoError(t, err)
	rows, err := rowsFromResult(result)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	row := rowToMap(rows[0])
	require.Equal(t, "go-go-golems/go-go-goja", row["repo"])
	require.Equal(t, "closed", row["state"])
	require.EqualValues(t, 2, row["labelCount"])

	stillAlive, err := rt.Owner.Call(context.Background(), "test.still-alive", func(_ context.Context, vm *goja.Runtime) (any, error) {
		return vm.RunString(`(() => 42)()`)
	})
	require.NoError(t, err)
	value, ok := stillAlive.(goja.Value)
	require.True(t, ok)
	require.EqualValues(t, 42, value.ToInteger())
}

func TestUnknownSectionStillFailsWhenAbsentFromBothCatalogs(t *testing.T) {
	registry, err := ScanSource("broken.js", `
function summarize(filters) {
  return { ok: !!filters };
}

__verb__("summarize", {
  fields: {
    filters: { bind: "missing" }
  }
});
`)
	require.NoError(t, err)
	require.NoError(t, registry.AddSharedSection(&SectionSpec{
		Slug:  "filters",
		Title: "Filters",
		Fields: map[string]*FieldSpec{
			"state": {Type: "string"},
		},
	}))

	_, err = registry.Commands()
	require.Error(t, err)
	require.Contains(t, err.Error(), `unknown section "missing"`)
}

func mustRegistry(t *testing.T) *Registry {
	t.Helper()
	registry, err := ScanDir(filepath.Join(repoRoot(t), "testdata", "jsverbs"))
	require.NoError(t, err)
	return registry
}

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	return root
}

func mustCommandMap(t *testing.T, registry *Registry) map[string]cmds.Command {
	t.Helper()
	commands, err := registry.Commands()
	require.NoError(t, err)
	ret := map[string]cmds.Command{}
	for i, command := range commands {
		ret[registry.Verbs()[i].FullPath()] = command
	}
	return ret
}

func runCommand(t *testing.T, command cmds.Command, valuesBySection map[string]map[string]interface{}) []map[string]interface{} {
	t.Helper()
	parsedValues, err := runner.ParseCommandValues(command, runner.WithValuesForSections(valuesBySection))
	require.NoError(t, err)

	glazeCommand, ok := command.(cmds.GlazeCommand)
	require.True(t, ok)

	gp := &captureProcessor{}
	err = glazeCommand.RunIntoGlazeProcessor(context.Background(), parsedValues, gp)
	require.NoError(t, err)
	err = gp.Close(context.Background())
	require.NoError(t, err)

	rows := make([]map[string]interface{}, 0, len(gp.rows))
	for _, row := range gp.rows {
		rows = append(rows, rowToMap(row))
	}
	return rows
}

func runWriterCommand(t *testing.T, command cmds.Command, valuesBySection map[string]map[string]interface{}) string {
	t.Helper()
	parsedValues, err := runner.ParseCommandValues(command, runner.WithValuesForSections(valuesBySection))
	require.NoError(t, err)

	writerCommand, ok := command.(cmds.WriterCommand)
	require.True(t, ok)

	var b strings.Builder
	err = writerCommand.RunIntoWriter(context.Background(), parsedValues, &b)
	require.NoError(t, err)
	return b.String()
}

func rowToMap(row types.Row) map[string]interface{} {
	ret := map[string]interface{}{}
	for pair := row.Oldest(); pair != nil; pair = pair.Next() {
		ret[pair.Key] = pair.Value
	}
	return ret
}

type captureProcessor struct {
	rows []types.Row
}

func (c *captureProcessor) AddRow(_ context.Context, row types.Row) error {
	c.rows = append(c.rows, row)
	return nil
}

func (c *captureProcessor) Close(context.Context) error {
	return nil
}
