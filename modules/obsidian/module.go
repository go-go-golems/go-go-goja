package obsidianmod

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/modules"
	obsidianpkg "github.com/go-go-golems/go-go-goja/pkg/obsidian"
	"github.com/go-go-golems/go-go-goja/pkg/obsidiancli"
	"github.com/go-go-golems/go-go-goja/pkg/obsidianmd"
	"github.com/go-go-golems/go-go-goja/pkg/runtimeowner"
	"github.com/pkg/errors"
)

// Options customize runtime-state construction for the module.
type Options struct {
	NewRunner func(cfg obsidiancli.Config) obsidianpkg.Runner
	NewClient func(cfg obsidianpkg.Config, runner obsidianpkg.Runner) *obsidianpkg.Client
	NewOwner  func(vm *goja.Runtime) runtimeowner.Runner
}

// Module adapts the high-level Obsidian client into a goja native module.
type Module struct {
	opts   Options
	states sync.Map // map[*goja.Runtime]*runtimeState
}

type runtimeState struct {
	mu     sync.RWMutex
	cfg    obsidiancli.Config
	runner obsidianpkg.Runner
	client *obsidianpkg.Client
	owner  runtimeowner.Runner
}

var _ modules.NativeModule = (*Module)(nil)

// New creates a new native Obsidian module instance.
func New(opts Options) *Module {
	return &Module{opts: opts}
}

// Name returns the module name exposed to require().
func (m *Module) Name() string { return "obsidian" }

// Doc returns a short module description.
func (m *Module) Doc() string {
	return "Obsidian module with Promise-based vault operations, fluent queries, and markdown helpers."
}

// Loader wires the module exports for one runtime instance.
func (m *Module) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
	exports := moduleObj.Get("exports").(*goja.Object)
	state := m.ensureState(vm)

	modules.SetExport(exports, m.Name(), "configure", func(call goja.FunctionCall) goja.Value {
		options, err := mapArg(vm, call.Argument(0))
		if err != nil {
			panic(vm.NewTypeError(err.Error()))
		}
		cfg := mergeCLIConfig(state.config(), options)
		state.rebuild(cfg, m.opts)
		return vm.ToValue(configToJSMap(cfg))
	})

	modules.SetExport(exports, m.Name(), "version", func(goja.FunctionCall) goja.Value {
		return m.promise(vm, state, "version", func(ctx context.Context, current *runtimeState) (any, error) {
			return current.clientSnapshot().Version(ctx)
		})
	})

	modules.SetExport(exports, m.Name(), "files", func(call goja.FunctionCall) goja.Value {
		return m.promise(vm, state, "files", func(ctx context.Context, current *runtimeState) (any, error) {
			options, err := mapArg(vm, call.Argument(0))
			if err != nil {
				return nil, err
			}
			return current.clientSnapshot().Files(ctx, fileListOptions(options))
		})
	})

	modules.SetExport(exports, m.Name(), "read", func(call goja.FunctionCall) goja.Value {
		return m.promise(vm, state, "read", func(ctx context.Context, current *runtimeState) (any, error) {
			ref := strings.TrimSpace(call.Argument(0).String())
			return current.clientSnapshot().Read(ctx, ref)
		})
	})

	modules.SetExport(exports, m.Name(), "create", func(call goja.FunctionCall) goja.Value {
		return m.promise(vm, state, "create", func(ctx context.Context, current *runtimeState) (any, error) {
			title := strings.TrimSpace(call.Argument(0).String())
			options, err := mapArg(vm, call.Argument(1))
			if err != nil {
				return nil, err
			}
			return current.clientSnapshot().Create(ctx, title, createOptions(options))
		})
	})

	modules.SetExport(exports, m.Name(), "append", func(call goja.FunctionCall) goja.Value {
		return m.promise(vm, state, "append", func(ctx context.Context, current *runtimeState) (any, error) {
			if err := current.clientSnapshot().Append(ctx, call.Argument(0).String(), call.Argument(1).String()); err != nil {
				return nil, err
			}
			return true, nil
		})
	})

	modules.SetExport(exports, m.Name(), "prepend", func(call goja.FunctionCall) goja.Value {
		return m.promise(vm, state, "prepend", func(ctx context.Context, current *runtimeState) (any, error) {
			if err := current.clientSnapshot().Prepend(ctx, call.Argument(0).String(), call.Argument(1).String()); err != nil {
				return nil, err
			}
			return true, nil
		})
	})

	modules.SetExport(exports, m.Name(), "move", func(call goja.FunctionCall) goja.Value {
		return m.promise(vm, state, "move", func(ctx context.Context, current *runtimeState) (any, error) {
			if err := current.clientSnapshot().Move(ctx, call.Argument(0).String(), call.Argument(1).String()); err != nil {
				return nil, err
			}
			return true, nil
		})
	})

	modules.SetExport(exports, m.Name(), "rename", func(call goja.FunctionCall) goja.Value {
		return m.promise(vm, state, "rename", func(ctx context.Context, current *runtimeState) (any, error) {
			if err := current.clientSnapshot().Rename(ctx, call.Argument(0).String(), call.Argument(1).String()); err != nil {
				return nil, err
			}
			return true, nil
		})
	})

	modules.SetExport(exports, m.Name(), "delete", func(call goja.FunctionCall) goja.Value {
		return m.promise(vm, state, "delete", func(ctx context.Context, current *runtimeState) (any, error) {
			options, err := mapArg(vm, call.Argument(1))
			if err != nil {
				return nil, err
			}
			if err := current.clientSnapshot().Delete(ctx, call.Argument(0).String(), deleteOptions(options)); err != nil {
				return nil, err
			}
			return true, nil
		})
	})

	modules.SetExport(exports, m.Name(), "note", func(call goja.FunctionCall) goja.Value {
		return m.promise(vm, state, "note", func(ctx context.Context, current *runtimeState) (any, error) {
			note, err := current.clientSnapshot().Note(ctx, call.Argument(0).String())
			if err != nil {
				return nil, err
			}
			return noteToMap(ctx, note)
		})
	})

	modules.SetExport(exports, m.Name(), "query", func(call goja.FunctionCall) goja.Value {
		query := state.clientSnapshot().Query()
		options, err := mapArg(vm, call.Argument(0))
		if err == nil {
			applyQueryOptions(query, options)
		}
		return m.newQueryObject(vm, state, query)
	})

	modules.SetExport(exports, m.Name(), "batch", func(call goja.FunctionCall) goja.Value {
		return m.promise(vm, state, "batch", func(ctx context.Context, current *runtimeState) (any, error) {
			options, err := mapArg(vm, call.Argument(0))
			if err != nil {
				return nil, err
			}
			query := current.clientSnapshot().Query()
			applyQueryOptions(query, options)

			var mapper goja.Callable
			if call.Argument(1) != nil && !goja.IsUndefined(call.Argument(1)) && !goja.IsNull(call.Argument(1)) {
				var ok bool
				mapper, ok = goja.AssertFunction(call.Argument(1))
				if !ok {
					return nil, errors.New("obsidian module: batch mapper must be a function")
				}
			}

			notes, err := query.Run(ctx)
			if err != nil {
				return nil, err
			}

			results := make([]any, 0, len(notes))
			for _, note := range notes {
				noteValue, err := noteToMap(ctx, note)
				if err != nil {
					return nil, err
				}
				if mapper == nil {
					results = append(results, noteValue)
					continue
				}
				if current.owner != nil {
					mapped, err := current.owner.Call(ctx, "obsidian.batch.mapper", func(_ context.Context, vm *goja.Runtime) (any, error) {
						value, err := mapper(goja.Undefined(), vm.ToValue(noteValue))
						if err != nil {
							return nil, err
						}
						return value.Export(), nil
					})
					if err != nil {
						return nil, err
					}
					results = append(results, mapped)
					continue
				}
				value, err := mapper(goja.Undefined(), vm.ToValue(noteValue))
				if err != nil {
					return nil, err
				}
				results = append(results, value.Export())
			}
			return results, nil
		})
	})

	modules.SetExport(exports, m.Name(), "exec", func(call goja.FunctionCall) goja.Value {
		return m.promise(vm, state, "exec", func(ctx context.Context, current *runtimeState) (any, error) {
			name := strings.TrimSpace(call.Argument(0).String())
			if name == "" {
				return nil, errors.New("obsidian module: command name is empty")
			}
			parameters, err := mapArg(vm, call.Argument(1))
			if err != nil {
				return nil, err
			}
			flags, err := stringSliceArg(vm, call.Argument(2))
			if err != nil {
				return nil, err
			}
			result, err := current.runnerSnapshot().Run(ctx, obsidiancli.CommandSpec{
				Name:   name,
				Output: obsidiancli.OutputRaw,
			}, obsidiancli.CallOptions{
				Parameters: parameters,
				Flags:      flags,
			})
			if err != nil {
				return nil, err
			}
			return result.Stdout, nil
		})
	})

	if err := exports.Set("md", m.newMarkdownObject(vm)); err != nil {
		panic(vm.NewTypeError(err.Error()))
	}
}

func (m *Module) ensureState(vm *goja.Runtime) *runtimeState {
	if state, ok := m.states.Load(vm); ok {
		return state.(*runtimeState)
	}

	state := &runtimeState{
		cfg: obsidiancli.DefaultConfig(),
	}
	if m.opts.NewOwner != nil {
		state.owner = m.opts.NewOwner(vm)
	}
	state.rebuild(state.cfg, m.opts)

	actual, _ := m.states.LoadOrStore(vm, state)
	return actual.(*runtimeState)
}

func (m *Module) promise(vm *goja.Runtime, state *runtimeState, op string, fn func(context.Context, *runtimeState) (any, error)) goja.Value {
	promise, resolve, reject := vm.NewPromise()
	if state.owner == nil {
		value, err := fn(context.Background(), state)
		if err != nil {
			_ = reject(vm.ToValue(err.Error()))
		} else {
			_ = resolve(vm.ToValue(value))
		}
		return vm.ToValue(promise)
	}

	go func() {
		value, err := fn(context.Background(), state)
		_ = state.owner.Post(context.Background(), "obsidian."+op+".settle", func(_ context.Context, vm *goja.Runtime) {
			if err != nil {
				_ = reject(vm.ToValue(err.Error()))
				return
			}
			_ = resolve(vm.ToValue(value))
		})
	}()
	return vm.ToValue(promise)
}

func (m *Module) newQueryObject(vm *goja.Runtime, state *runtimeState, query *obsidianpkg.Query) *goja.Object {
	obj := vm.NewObject()
	setQueryMethod := func(name string, fn func() *obsidianpkg.Query) {
		modules.SetExport(obj, m.Name(), name, func(goja.FunctionCall) goja.Value {
			fn()
			return obj
		})
	}

	modules.SetExport(obj, m.Name(), "inFolder", func(call goja.FunctionCall) goja.Value {
		query.InFolder(call.Argument(0).String())
		return obj
	})
	modules.SetExport(obj, m.Name(), "withExtension", func(call goja.FunctionCall) goja.Value {
		query.WithExtension(call.Argument(0).String())
		return obj
	})
	modules.SetExport(obj, m.Name(), "search", func(call goja.FunctionCall) goja.Value {
		query.Search(call.Argument(0).String())
		return obj
	})
	modules.SetExport(obj, m.Name(), "nameContains", func(call goja.FunctionCall) goja.Value {
		query.NameContains(call.Argument(0).String())
		return obj
	})
	modules.SetExport(obj, m.Name(), "tagged", func(call goja.FunctionCall) goja.Value {
		query.Tagged(call.Argument(0).String())
		return obj
	})
	modules.SetExport(obj, m.Name(), "limit", func(call goja.FunctionCall) goja.Value {
		query.Limit(int(call.Argument(0).ToInteger()))
		return obj
	})
	setQueryMethod("orphans", query.Orphans)
	setQueryMethod("deadEnds", query.DeadEnds)
	setQueryMethod("unresolved", query.Unresolved)

	modules.SetExport(obj, m.Name(), "run", func(goja.FunctionCall) goja.Value {
		return m.promise(vm, state, "query.run", func(ctx context.Context, current *runtimeState) (any, error) {
			notes, err := query.Run(ctx)
			if err != nil {
				return nil, err
			}
			return notesToMaps(ctx, notes)
		})
	})
	return obj
}

func (m *Module) newMarkdownObject(vm *goja.Runtime) *goja.Object {
	obj := vm.NewObject()
	modules.SetExport(obj, m.Name(), "parseFrontmatter", func(call goja.FunctionCall) goja.Value {
		doc, err := obsidianmd.ParseDocument(call.Argument(0).String())
		if err != nil {
			panic(vm.NewTypeError(err.Error()))
		}
		return vm.ToValue(map[string]any{
			"frontmatter": doc.Frontmatter,
			"body":        doc.Body,
		})
	})
	modules.SetExport(obj, m.Name(), "wikilinks", func(call goja.FunctionCall) []string {
		return obsidianmd.ExtractWikilinks(call.Argument(0).String())
	})
	modules.SetExport(obj, m.Name(), "headings", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(headingsToMaps(obsidianmd.ExtractHeadings(call.Argument(0).String())))
	})
	modules.SetExport(obj, m.Name(), "tags", func(call goja.FunctionCall) []string {
		return obsidianmd.ExtractTags(call.Argument(0).String())
	})
	modules.SetExport(obj, m.Name(), "tasks", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(tasksToMaps(obsidianmd.ExtractTasks(call.Argument(0).String())))
	})
	modules.SetExport(obj, m.Name(), "note", func(call goja.FunctionCall) goja.Value {
		options, err := mapArg(vm, call.Argument(0))
		if err != nil {
			panic(vm.NewTypeError(err.Error()))
		}
		note, err := obsidianmd.BuildNote(noteTemplate(options))
		if err != nil {
			panic(vm.NewTypeError(err.Error()))
		}
		return vm.ToValue(note)
	})
	return obj
}

func (s *runtimeState) rebuild(cfg obsidiancli.Config, opts Options) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if opts.NewRunner != nil {
		s.runner = opts.NewRunner(cfg)
	} else {
		s.runner = obsidiancli.NewRunner(cfg, nil)
	}
	if opts.NewClient != nil {
		s.client = opts.NewClient(obsidianpkg.Config{CLI: cfg}, s.runner)
	} else {
		s.client = obsidianpkg.NewClient(obsidianpkg.Config{CLI: cfg}, s.runner)
	}
	s.cfg = cfg
}

func (s *runtimeState) config() obsidiancli.Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

func (s *runtimeState) clientSnapshot() *obsidianpkg.Client {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.client
}

func (s *runtimeState) runnerSnapshot() obsidianpkg.Runner {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.runner
}

func mergeCLIConfig(cfg obsidiancli.Config, options map[string]any) obsidiancli.Config {
	if len(options) == 0 {
		return cfg
	}
	if value := stringOption(options, "binaryPath"); value != "" {
		cfg.BinaryPath = value
	}
	if value := stringOption(options, "vault"); value != "" {
		cfg.Vault = value
	}
	if value := stringOption(options, "workingDir"); value != "" {
		cfg.WorkingDir = value
	}
	if timeoutMS := intOption(options, "timeoutMs"); timeoutMS > 0 {
		cfg.Timeout = time.Duration(timeoutMS) * time.Millisecond
	}
	if env := stringSliceOption(options, "env"); len(env) > 0 {
		cfg.Env = env
	}
	return cfg
}

func configToJSMap(cfg obsidiancli.Config) map[string]any {
	timeoutMS := int64(0)
	if cfg.Timeout > 0 {
		timeoutMS = cfg.Timeout.Milliseconds()
	}
	return map[string]any{
		"binaryPath": cfg.BinaryPath,
		"vault":      cfg.Vault,
		"workingDir": cfg.WorkingDir,
		"timeoutMs":  timeoutMS,
		"env":        append([]string(nil), cfg.Env...),
	}
}

func fileListOptions(options map[string]any) obsidianpkg.FileListOptions {
	return obsidianpkg.FileListOptions{
		Folder: stringOption(options, "folder"),
		Ext:    stringOption(options, "ext"),
		Limit:  intOption(options, "limit"),
		Vault:  stringOption(options, "vault"),
	}
}

func createOptions(options map[string]any) obsidianpkg.CreateOptions {
	return obsidianpkg.CreateOptions{
		Content:  stringOption(options, "content"),
		Folder:   stringOption(options, "folder"),
		Template: stringOption(options, "template"),
		Vault:    stringOption(options, "vault"),
	}
}

func deleteOptions(options map[string]any) obsidianpkg.DeleteOptions {
	return obsidianpkg.DeleteOptions{
		Permanent: boolOption(options, "permanent"),
		Vault:     stringOption(options, "vault"),
	}
}

func applyQueryOptions(query *obsidianpkg.Query, options map[string]any) {
	if query == nil || len(options) == 0 {
		return
	}
	if value := stringOption(options, "folder"); value != "" {
		query.InFolder(value)
	}
	if value := stringOption(options, "ext"); value != "" {
		query.WithExtension(value)
	}
	if value := stringOption(options, "search"); value != "" {
		query.Search(value)
	}
	if value := stringOption(options, "nameContains"); value != "" {
		query.NameContains(value)
	}
	if value := stringOption(options, "tag"); value != "" {
		query.Tagged(value)
	}
	if value := intOption(options, "limit"); value > 0 {
		query.Limit(value)
	}
	switch strings.ToLower(stringOption(options, "mode")) {
	case "orphans":
		query.Orphans()
	case "deadends", "dead-ends":
		query.DeadEnds()
	case "unresolved":
		query.Unresolved()
	}
}

func noteToMap(ctx context.Context, note *obsidianpkg.Note) (map[string]any, error) {
	content, err := note.Content(ctx)
	if err != nil {
		return nil, err
	}
	frontmatter, err := note.Frontmatter(ctx)
	if err != nil {
		return nil, err
	}
	wikilinks, err := note.Wikilinks(ctx)
	if err != nil {
		return nil, err
	}
	headings, err := note.Headings(ctx)
	if err != nil {
		return nil, err
	}
	tags, err := note.Tags(ctx)
	if err != nil {
		return nil, err
	}
	tasks, err := note.Tasks(ctx)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"path":        note.Path,
		"title":       note.Title,
		"content":     content,
		"frontmatter": frontmatter,
		"wikilinks":   wikilinks,
		"headings":    headingsToMaps(headings),
		"tags":        tags,
		"tasks":       tasksToMaps(tasks),
	}, nil
}

func notesToMaps(ctx context.Context, notes []*obsidianpkg.Note) ([]map[string]any, error) {
	ret := make([]map[string]any, 0, len(notes))
	for _, note := range notes {
		value, err := noteToMap(ctx, note)
		if err != nil {
			return nil, err
		}
		ret = append(ret, value)
	}
	return ret, nil
}

func headingsToMaps(headings []obsidianmd.Heading) []map[string]any {
	ret := make([]map[string]any, 0, len(headings))
	for _, heading := range headings {
		ret = append(ret, map[string]any{
			"level": heading.Level,
			"text":  heading.Text,
			"line":  heading.Line,
		})
	}
	return ret
}

func tasksToMaps(tasks []obsidianmd.Task) []map[string]any {
	ret := make([]map[string]any, 0, len(tasks))
	for _, task := range tasks {
		ret = append(ret, map[string]any{
			"text": task.Text,
			"done": task.Done,
			"line": task.Line,
		})
	}
	return ret
}

func noteTemplate(options map[string]any) obsidianmd.NoteTemplate {
	sections := make([]obsidianmd.NoteSection, 0)
	for _, raw := range sliceOption(options, "sections") {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		sections = append(sections, obsidianmd.NoteSection{
			Title: stringOption(row, "title"),
			Body:  stringOption(row, "body"),
		})
	}
	return obsidianmd.NoteTemplate{
		Title:    stringOption(options, "title"),
		WikiTags: stringSliceOption(options, "wikiTags"),
		Body:     stringOption(options, "body"),
		Sections: sections,
	}
}

func mapArg(vm *goja.Runtime, value goja.Value) (map[string]any, error) {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return nil, nil
	}
	ret := map[string]any{}
	if err := vm.ExportTo(value, &ret); err != nil {
		return nil, errors.Wrap(err, "obsidian module: export object")
	}
	return ret, nil
}

func stringSliceArg(vm *goja.Runtime, value goja.Value) ([]string, error) {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return nil, nil
	}
	var ret []string
	if err := vm.ExportTo(value, &ret); err != nil {
		return nil, errors.Wrap(err, "obsidian module: export string slice")
	}
	return ret, nil
}

func stringOption(options map[string]any, key string) string {
	if len(options) == 0 {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(options[key]))
}

func intOption(options map[string]any, key string) int {
	if len(options) == 0 {
		return 0
	}
	switch value := options[key].(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	}
	return 0
}

func boolOption(options map[string]any, key string) bool {
	if len(options) == 0 {
		return false
	}
	value, ok := options[key].(bool)
	return ok && value
}

func stringSliceOption(options map[string]any, key string) []string {
	if len(options) == 0 {
		return nil
	}
	raw := sliceOption(options, key)
	ret := make([]string, 0, len(raw))
	for _, item := range raw {
		text := strings.TrimSpace(fmt.Sprint(item))
		if text == "" {
			continue
		}
		ret = append(ret, text)
	}
	return ret
}

func sliceOption(options map[string]any, key string) []any {
	if len(options) == 0 {
		return nil
	}
	switch value := options[key].(type) {
	case []any:
		return value
	default:
		return nil
	}
}

func init() {
	modules.Register(New(Options{}))
}
