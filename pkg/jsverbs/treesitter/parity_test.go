package treesitter_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/go-go-golems/go-go-goja/pkg/jsverbs"
	"github.com/go-go-golems/go-go-goja/pkg/jsverbs/treesitter"
	"github.com/stretchr/testify/require"
)

// The two parsers must produce the same registry for every verb tree in the
// repository and for the edge cases below: same files, functions, parameters,
// packages, sections, docs, verbs and diagnostics.

func repoRoot(t testing.TB) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

// verbTrees lists every directory in the repository (outside ttmp) whose
// JavaScript declares verbs.
func verbTrees(t testing.TB) []string {
	t.Helper()
	root := repoRoot(t)
	dirs := map[string]bool{}
	for _, top := range []string{"examples", "pkg", "cmd"} {
		err := filepath.WalkDir(filepath.Join(root, top), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() && (d.Name() == "node_modules" || strings.HasPrefix(d.Name(), ".")) {
				return filepath.SkipDir
			}
			if d.IsDir() || (filepath.Ext(path) != ".js" && filepath.Ext(path) != ".cjs") {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if strings.Contains(string(data), jsverbs.VerbSentinel) {
				dirs[filepath.Dir(path)] = true
			}
			return nil
		})
		require.NoError(t, err)
	}
	out := make([]string, 0, len(dirs))
	for d := range dirs {
		out = append(out, d)
	}
	sort.Strings(out)
	return out
}

type fileSnap struct {
	RelPath    string
	ModulePath string
	Package    jsverbs.PackageSpec
	Functions  []jsverbs.FunctionSpec
	Sections   []jsverbs.SectionSpec
	Docs       map[string]string
}

type verbSnap struct {
	Path        string
	Function    string
	Short       string
	Long        string
	OutputMode  string
	Tags        []string
	UseSections []string
	Fields      map[string]*jsverbs.FieldSpec
	Params      []jsverbs.ParameterSpec
	File        string
}

type registrySnap struct {
	Files       []fileSnap
	Verbs       []verbSnap
	Diagnostics []jsverbs.Diagnostic
	Err         string
}

func snapshot(reg *jsverbs.Registry, err error) registrySnap {
	s := registrySnap{}
	if err != nil {
		s.Err = err.Error()
	}
	if reg == nil {
		return s
	}
	for _, f := range reg.Files {
		fs := fileSnap{RelPath: f.RelPath, ModulePath: f.ModulePath, Package: f.Package, Docs: f.Docs}
		for _, fn := range f.Functions {
			fs.Functions = append(fs.Functions, *fn)
		}
		for _, slug := range f.SectionOrder {
			fs.Sections = append(fs.Sections, *f.Sections[slug])
		}
		s.Files = append(s.Files, fs)
	}
	for _, v := range reg.Verbs() {
		s.Verbs = append(s.Verbs, verbSnap{
			Path: v.FullPath(), Function: v.FunctionName, Short: v.Short, Long: v.Long, OutputMode: v.OutputMode,
			Tags: v.Tags, UseSections: v.UseSections, Fields: v.Fields, Params: v.Params, File: v.File.RelPath,
		})
	}
	sort.Slice(s.Verbs, func(i, j int) bool { return s.Verbs[i].Path < s.Verbs[j].Path })
	s.Diagnostics = reg.Diagnostics
	return s
}

func asJSON(t *testing.T, v interface{}) string {
	t.Helper()
	b, err := json.MarshalIndent(v, "", "  ")
	require.NoError(t, err)
	return string(b)
}

func scanBoth(t *testing.T, scan func(jsverbs.ScanOptions) (*jsverbs.Registry, error)) (registrySnap, registrySnap) {
	t.Helper()
	opts := jsverbs.DefaultScanOptions()
	opts.FailOnErrorDiagnostics = false
	gojaReg, gojaErr := scan(opts)
	opts.Parser = treesitter.Parser{}
	tsReg, tsErr := scan(opts)
	return snapshot(gojaReg, gojaErr), snapshot(tsReg, tsErr)
}

func TestParsersAgreeOnRepositoryVerbTrees(t *testing.T) {
	trees := verbTrees(t)
	require.NotEmpty(t, trees)
	root := repoRoot(t)
	totalVerbs := 0
	for _, dir := range trees {
		rel, _ := filepath.Rel(root, dir)
		t.Run(rel, func(t *testing.T) {
			gojaSnap, tsSnap := scanBoth(t, func(o jsverbs.ScanOptions) (*jsverbs.Registry, error) { return jsverbs.ScanDir(dir, o) })
			require.Equal(t, asJSON(t, tsSnap), asJSON(t, gojaSnap))
			require.Empty(t, gojaSnap.Err)
			totalVerbs += len(gojaSnap.Verbs)
		})
	}
	t.Logf("%d verb trees, %d verbs, identical with both parsers", len(trees), totalVerbs)
	require.Greater(t, totalVerbs, 20)
}

// edgeCases exercise declaration forms, literal forms and metadata errors.
var edgeCases = map[string]string{
	"forms.js": `
function plain(a, b = 2, {c, d}, [e], ...rest) { return {}; }
async function later(x) { return x; }
const arrow = (x, y) => ({ x, y });
const single = x => x;
let fnExpr = function(q) { return q; };
var asyncArrow = async (z) => z;
function* gen() {}
const notAFunction = 42, alsoArrow = () => 1;
class Thing {}
function _private() {}

__package__({ name: "forms", short: 'Forms', long: "line\nbreak", parents: ["top"], tags: ["a", "b",] });

__section__("filters", {
  title: "Filters", // a comment
  description: ` + "`raw \\n text`" + `,
  fields: { state: { type: "choice", choices: ["open", "closed"], default: "open" }, limit: { type: "int", default: 10 } },
});

doc` + "`" + `---
verb: plain
---
Plain does things.

With a second paragraph.` + "`" + `;

__verb__("plain", { short: "Plain", output: "text", sections: ["filters"], fields: { a: { argument: true }, c: { bind: "all" }, e: { bind: "all" } } });
__verb__("later", { command: "do-later", tags: ["x"], fields: { x: { type: "float", default: 1.5e3 } } });
__verb__("arrow", { name: "Arrow Thing", fields: { x: {}, y: { required: true, help: 'it\'s y' } } });
__verb__("single", {});
__verb__("fnExpr", { fields: { q: { type: "bool", default: false, short: "q" } } });
__verb__("asyncArrow", { fields: { z: { default: null } } });
__verb__({ function: "alsoArrow", short: "named in the object" });
`,
	"errors.js": `
function one() {}
function two() {}
function three() {}
function four() {}
function five() {}
function six() {}
function seven() {}
__verb__("one", { short: helper() });
__verb__("two", { fields: { a: { default: undefined } } });
__verb__("three", { short });
__verb__("four", { ...base });
__verb__("five", { ["computed"]: 1 });
__verb__("six", { method() {} });
__verb__("seven", { short: ` + "`a ${b}`" + ` });
__verb__("one");
__section__({ title: "no name" });
__section__("", { title: "empty" });
__package__("not an object");
__verb__(name, {});
`,
	"nested/index.js": `
function helperOnly() {}
const value = [1, , 3];
function act(n) { return n; }
__verb__("act", { fields: { n: { type: "int", argument: true, default: 3 } }, tags: [] });
`,
	"unicode.js": `
// héllo wörld 日本語 — offsets after multi-byte text must stay right
function größe({ä, ö}, [ü]) { return {}; }
doc` + "`" + `---
verb: größe
---
Größe misst — 日本語.` + "`" + `;
__verb__("größe", { short: "Größe", fields: { ä: { bind: "all" }, ü: { bind: "all" } } });
`,
	"early-return.js": `
// require() runs a module as a function body: a top-level return is legal.
if (typeof module === "undefined") return;
function early() {}
__verb__("early", {});
`,
	"strings.js": `
function s() {}
__verb__("s", { short: "tab\there é \"q\"", long: 'single \'quoted\' \\ back', tags: ["ünï", "日本"] });
`,
}

func edgeSources() []jsverbs.SourceFile {
	names := make([]string, 0, len(edgeCases))
	for name := range edgeCases {
		names = append(names, name)
	}
	sort.Strings(names)
	files := []jsverbs.SourceFile{}
	for _, name := range names {
		files = append(files, jsverbs.SourceFile{Path: name, Source: []byte(edgeCases[name])})
	}
	return files
}

func TestParsersAgreeOnEdgeCases(t *testing.T) {
	gojaSnap, tsSnap := scanBoth(t, func(o jsverbs.ScanOptions) (*jsverbs.Registry, error) {
		return jsverbs.ScanSources(edgeSources(), o)
	})
	require.Equal(t, asJSON(t, tsSnap), asJSON(t, gojaSnap))

	// Spot checks that the shared result is the documented one.
	byPath := map[string]verbSnap{}
	for _, v := range gojaSnap.Verbs {
		byPath[v.Path] = v
	}
	require.Contains(t, byPath, "top forms plain")
	require.Contains(t, byPath, "top forms fn-expr", "const name = function(...) {} is a declaration form")
	require.Contains(t, byPath, "top forms also-arrow")
	require.Contains(t, byPath, "top forms do-later")
	require.Contains(t, byPath, "top forms arrow-thing")
	require.Contains(t, byPath, "early-return early", "a top-level return does not hide what follows it")
	require.Equal(t, "Plain does things.\n\nWith a second paragraph.", byPath["top forms plain"].Long)
	require.Equal(t, []jsverbs.ParameterSpec{
		{Name: "a", Kind: jsverbs.ParameterIdentifier},
		{Name: "b", Kind: jsverbs.ParameterIdentifier},
		{Name: "{c, d}", Kind: jsverbs.ParameterObject},
		{Name: "[e]", Kind: jsverbs.ParameterArray},
		{Name: "rest", Kind: jsverbs.ParameterIdentifier, Rest: true},
	}, byPath["top forms plain"].Params)
	require.Equal(t, `tab	here é "q"`, byPath["strings s"].Short)
	require.Equal(t, `single 'quoted' \ back`, byPath["strings s"].Long)
	require.Equal(t, float64(1500), byPath["top forms do-later"].Fields["x"].Default)
	require.Len(t, gojaSnap.Diagnostics, 12, asJSON(t, gojaSnap.Diagnostics))
	require.Equal(t, "Größe misst — 日本語.", byPath["unicode größe"].Long)
	require.Equal(t, "{ä, ö}", byPath["unicode größe"].Params[0].Name)
	for _, f := range gojaSnap.Files {
		if f.RelPath == "forms.js" {
			require.Equal(t, `raw \n text`, f.Sections[0].Description, "template metadata is raw text")
		}
	}
}

func TestDefaultParserReportsUnparseableSources(t *testing.T) {
	files := []jsverbs.SourceFile{
		{Path: "verbs.js", Source: []byte("export function hi() {}\n__verb__(\"hi\", {});\n")},
		{Path: "asset.js", Source: []byte("import x from './x.js';\nexport default x;\n")},
		{Path: "ok.js", Source: []byte("#!/usr/bin/env goja\nfunction ok() {}\n__verb__(\"ok\", {});\n")},
	}

	reg, err := jsverbs.ScanSources(files)
	require.Error(t, err, "a verb file the default parser cannot read fails the scan")
	require.Len(t, reg.Files, 3, "unparseable files stay in the registry for require()")
	diags := map[string]jsverbs.Diagnostic{}
	for _, d := range reg.Diagnostics {
		diags[d.Path] = d
	}
	require.Equal(t, jsverbs.DiagnosticSeverityError, diags["verbs.js"].Severity)
	require.Contains(t, diags["verbs.js"].Message, "pkg/jsverbs/treesitter")
	require.Equal(t, jsverbs.DiagnosticSeverityWarning, diags["asset.js"].Severity)
	require.NotContains(t, diags, "ok.js", "a hashbang line is accepted")
	_, ok := reg.Verb("ok ok")
	require.True(t, ok)

	// The tree-sitter parser reads module syntax.
	reg, err = jsverbs.ScanSources(files, jsverbs.ScanOptions{Extensions: []string{".js"}, FailOnErrorDiagnostics: true, Parser: treesitter.Parser{}})
	require.NoError(t, err)
	_, ok = reg.Verb("verbs hi")
	require.True(t, ok)
}

func BenchmarkScanRepositoryVerbTrees(b *testing.B) {
	trees := verbTrees(b)
	for _, tc := range []struct {
		name   string
		parser jsverbs.SourceParser
	}{{"goja", nil}, {"treesitter", treesitter.Parser{}}} {
		b.Run(tc.name, func(b *testing.B) {
			opts := jsverbs.DefaultScanOptions()
			opts.FailOnErrorDiagnostics = false
			opts.Parser = tc.parser
			for i := 0; i < b.N; i++ {
				for _, dir := range trees {
					if _, err := jsverbs.ScanDir(dir, opts); err != nil {
						b.Fatal(err)
					}
				}
			}
		})
	}
}
