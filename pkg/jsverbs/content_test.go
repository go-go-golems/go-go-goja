package jsverbs

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/go-go-goja/pkg/resultcontent"
	stdrequire "github.com/stretchr/testify/require"
)

type verbLook struct{ name string }

func (l *verbLook) ResultContent(context.Context) ([]resultcontent.Content, error) {
	return []resultcontent.Content{{MIMEType: "image/png", Data: []byte("png:" + l.name), Caption: l.name}}, nil
}

func (l *verbLook) ResultPlaceholder() string { return "look " + l.name }

type verbLookInit struct{}

func (verbLookInit) ID() string { return "jsverbs-test-look" }

func (verbLookInit) InitRuntime(ctx *engine.RuntimeInitializationContext) error {
	if err := ctx.VM.Set("lookList", func() []*verbLook { return []*verbLook{{name: "l1"}, {name: "l2"}} }); err != nil {
		return err
	}
	return ctx.VM.Set("look", func(name string) *verbLook { return &verbLook{name: name} })
}

func lookInvoker(ctx context.Context, registry *Registry, verb *VerbSpec, parsedValues *values.Values) (interface{}, error) {
	factory, err := engine.NewRuntimeFactoryBuilder().
		WithRequireOptions(require.WithLoader(registry.RequireLoader())).
		UseModuleMiddleware(engine.MiddlewareSafe()).
		WithRuntimeInitializers(verbLookInit{}).
		Build()
	if err != nil {
		return nil, err
	}
	rt, err := factory.NewRuntime(engine.WithStartupContext(ctx), engine.WithLifetimeContext(ctx))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rt.Close(context.Background()) }()
	return registry.InvokeInRuntime(ctx, rt, verb, parsedValues)
}

func lookCommand(t *testing.T, source string, verbName string) cmds.Command {
	t.Helper()
	registry, err := ScanSource("looks.js", source)
	stdrequire.NoError(t, err)
	verb, ok := registry.Verb(verbName)
	stdrequire.True(t, ok, "verb %s", verbName)
	command, err := registry.CommandForVerbWithInvoker(verb, lookInvoker)
	stdrequire.NoError(t, err)
	return command
}

const looksSource = `
async function inspect(name) {
  return { name, mean: 0.5, after: look(name) };
}
function grid() {
  return [{ cell: 1, img: look("a") }, { cell: 2, img: look("b") }];
}
function echo(contentDir) {
  return { dir: contentDir, img: look("own") };
}
__verb__("inspect", { fields: { name: { argument: true } } });
__verb__("grid", {});
__verb__("echo", { fields: { contentDir: { help: "verb-owned flag" } } });
`

func TestVerbContentBecomesPlaceholderRows(t *testing.T) {
	rows := runCommand(t, lookCommand(t, looksSource, "looks inspect"), map[string]map[string]interface{}{
		"default": {"name": "after"},
	})
	stdrequire.Len(t, rows, 1)
	stdrequire.Equal(t, "[content 1: look after]", rows[0]["after"])
	stdrequire.Equal(t, 0.5, rows[0]["mean"])
	_, hasSection := rows[0]["jsverbs-content"]
	stdrequire.False(t, hasSection)

	rows = runCommand(t, lookCommand(t, looksSource, "looks grid"), nil)
	stdrequire.Len(t, rows, 2)
	stdrequire.Equal(t, "[content 1: look a]", rows[0]["img"])
	stdrequire.Equal(t, "[content 2: look b]", rows[1]["img"])
}

func TestVerbContentDirWritesFilesAndPutsPathsInRows(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "out")
	rows := runCommand(t, lookCommand(t, looksSource, "looks grid"), map[string]map[string]interface{}{
		ContentSectionSlug: {ContentDirField: dir},
	})
	stdrequire.Len(t, rows, 2)
	for i, name := range []string{"a", "b"} {
		path := filepath.Join(dir, "content-"+string(rune('1'+i))+".png")
		stdrequire.Equal(t, path, rows[i]["img"])
		data, err := os.ReadFile(path)
		stdrequire.NoError(t, err)
		stdrequire.Equal(t, "png:"+name, string(data))
	}
}

func TestVerbOwnedContentDirFieldWins(t *testing.T) {
	command := lookCommand(t, looksSource, "looks echo")
	_, ok := command.Description().Schema.Get(ContentSectionSlug)
	stdrequire.False(t, ok, "content section must not be added when the verb owns --content-dir")
	rows := runCommand(t, command, map[string]map[string]interface{}{"default": {"content-dir": "/verb/value"}})
	stdrequire.Equal(t, "/verb/value", rows[0]["dir"])
	stdrequire.Equal(t, "[content 1: look own]", rows[0]["img"])
}

func TestVerbCommandsHaveContentDirFlag(t *testing.T) {
	command := lookCommand(t, looksSource, "looks grid")
	section, ok := command.Description().Schema.Get(ContentSectionSlug)
	stdrequire.True(t, ok)
	_, ok = section.GetDefinitions().Get(ContentDirField)
	stdrequire.True(t, ok)
	stdrequire.True(t, strings.Contains(section.GetDescription(), "result content"))
}

func TestTextVerbContentPlaceholder(t *testing.T) {
	command := lookCommand(t, `
function show() { return { img: look("t") }; }
__verb__("show", { output: "text" });
`, "looks show")
	out := runWriterCommand(t, command, nil)
	stdrequire.Contains(t, out, `"img": "[content 1: look t]"`)
}

func TestVerbTypedCollectionContent(t *testing.T) {
	rows := runCommand(t, lookCommand(t, `
function typed() { return { imgs: lookList() }; }
__verb__("typed", {});
`, "looks typed"), nil)
	stdrequire.Len(t, rows, 1)
	stdrequire.Equal(t, []any{"[content 1: look l1]", "[content 2: look l2]"}, rows[0]["imgs"])
}
