package sandbox

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/pkg/runtimebridge"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type commandDraft struct {
	name    string
	spec    map[string]any
	handler goja.Callable
}

type eventDraft struct {
	name    string
	handler goja.Callable
}

type botDraft struct {
	moduleName string
	store      *MemoryStore
	metadata   map[string]any
	commands   []*commandDraft
	events     []*eventDraft
}

type botSnapshot struct {
	Kind     string           `json:"kind"`
	Metadata map[string]any   `json:"metadata"`
	Commands []map[string]any `json:"commands"`
	Events   []map[string]any `json:"events"`
}

// DispatchRequest contains the data the host passes into a bot dispatch call.
//
// It is intentionally transport-neutral: hosts can map Discord-like, Slack-like,
// or generic event payloads into the same shape.
type DispatchRequest struct {
	Name     string
	Args     map[string]any
	Command  map[string]any
	User     map[string]any
	Guild    map[string]any
	Channel  map[string]any
	Me       map[string]any
	Metadata map[string]any
	Reply    func(context.Context, any) error
	Defer    func(context.Context) error
}

// BotHandle wraps a compiled sandbox bot definition.
//
// The handle keeps the original goja object so the host can dispatch through
// the bot's JS methods from Go without re-parsing the script.
type BotHandle struct {
	vm              *goja.Runtime
	object          *goja.Object
	dispatchCommand goja.Callable
	dispatchEvent   goja.Callable
	describe        goja.Callable
}

// CompileBot validates that the value exported by module.exports is a sandbox
// bot object and returns a Go wrapper around it.
func CompileBot(vm *goja.Runtime, value goja.Value) (*BotHandle, error) {
	if vm == nil {
		return nil, fmt.Errorf("sandbox bot compile: vm is nil")
	}
	if goja.IsUndefined(value) || goja.IsNull(value) {
		return nil, fmt.Errorf("sandbox bot compile: value is nil")
	}
	obj := value.ToObject(vm)
	if obj == nil {
		return nil, fmt.Errorf("sandbox bot compile: value is not an object")
	}

	dispatchCommand, ok := goja.AssertFunction(obj.Get("dispatchCommand"))
	if !ok {
		return nil, fmt.Errorf("sandbox bot compile: missing dispatchCommand method")
	}
	dispatchEvent, ok := goja.AssertFunction(obj.Get("dispatchEvent"))
	if !ok {
		return nil, fmt.Errorf("sandbox bot compile: missing dispatchEvent method")
	}
	describe, ok := goja.AssertFunction(obj.Get("describe"))
	if !ok {
		return nil, fmt.Errorf("sandbox bot compile: missing describe method")
	}

	return &BotHandle{
		vm:              vm,
		object:          obj,
		dispatchCommand: dispatchCommand,
		dispatchEvent:   dispatchEvent,
		describe:        describe,
	}, nil
}

// Describe returns a JSON-like snapshot of the bot definition.
func (h *BotHandle) Describe(ctx context.Context) (map[string]any, error) {
	if h == nil {
		return nil, fmt.Errorf("sandbox bot handle is nil")
	}
	bindings, ok := runtimebridge.Lookup(h.vm)
	if !ok || bindings.Owner == nil {
		return nil, fmt.Errorf("sandbox bot requires runtime owner bindings")
	}

	ret, err := bindings.Owner.Call(ctx, "sandbox.bot.describe", func(context.Context, *goja.Runtime) (any, error) {
		value, err := h.describe(goja.Undefined())
		if err != nil {
			return nil, err
		}
		if goja.IsUndefined(value) || goja.IsNull(value) {
			return map[string]any{}, nil
		}
		if exported, ok := value.Export().(map[string]any); ok {
			return exported, nil
		}
		return map[string]any{"value": value.Export()}, nil
	})
	if err != nil {
		return nil, err
	}
	result, _ := ret.(map[string]any)
	return result, nil
}

// DispatchCommand invokes the named command handler.
func (h *BotHandle) DispatchCommand(ctx context.Context, request DispatchRequest) (any, error) {
	return h.dispatch(ctx, h.dispatchCommand, request)
}

// DispatchEvent invokes the named event handler.
func (h *BotHandle) DispatchEvent(ctx context.Context, request DispatchRequest) (any, error) {
	return h.dispatch(ctx, h.dispatchEvent, request)
}

func (h *BotHandle) dispatch(ctx context.Context, fn goja.Callable, request DispatchRequest) (any, error) {
	if h == nil {
		return nil, fmt.Errorf("sandbox bot handle is nil")
	}
	bindings, ok := runtimebridge.Lookup(h.vm)
	if !ok || bindings.Owner == nil {
		return nil, fmt.Errorf("sandbox bot requires runtime owner bindings")
	}

	ret, err := bindings.Owner.Call(ctx, "sandbox.bot.dispatch", func(callCtx context.Context, vm *goja.Runtime) (any, error) {
		input := buildDispatchInput(vm, callCtx, request)
		result, err := fn(goja.Undefined(), input)
		if err != nil {
			return nil, err
		}
		if goja.IsUndefined(result) || goja.IsNull(result) {
			return nil, nil
		}
		return result.Export(), nil
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func resolveValue(_ *goja.Runtime, value goja.Value) (any, error) {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return nil, nil
	}
	return value.Export(), nil
}

func buildDispatchInput(vm *goja.Runtime, ctx context.Context, request DispatchRequest) *goja.Object {
	input := vm.NewObject()
	setObjectField(vm, input, "name", request.Name)
	setObjectField(vm, input, "args", request.Args)
	setObjectField(vm, input, "command", request.Command)
	setObjectField(vm, input, "user", request.User)
	setObjectField(vm, input, "guild", request.Guild)
	setObjectField(vm, input, "channel", request.Channel)
	setObjectField(vm, input, "me", request.Me)
	setObjectField(vm, input, "metadata", request.Metadata)

	if request.Reply != nil {
		_ = input.Set("reply", func(message any) error {
			return request.Reply(ctx, message)
		})
	} else {
		_ = input.Set("reply", func(message any) error { return nil })
	}
	if request.Defer != nil {
		_ = input.Set("defer", func() error {
			return request.Defer(ctx)
		})
	} else {
		_ = input.Set("defer", func() error { return nil })
	}

	return input
}

func exportMap(value goja.Value) map[string]any {
	if goja.IsUndefined(value) || goja.IsNull(value) {
		return map[string]any{}
	}
	if exported, ok := value.Export().(map[string]any); ok {
		return exported
	}
	return map[string]any{"value": value.Export()}
}

func newBotDraft(state *RuntimeState) *botDraft {
	return &botDraft{
		moduleName: state.ModuleName(),
		store:      state.Store(),
		metadata:   map[string]any{},
		commands:   []*commandDraft{},
		events:     []*eventDraft{},
	}
}

func (d *botDraft) command(vm *goja.Runtime, call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 2 || len(call.Arguments) > 3 {
		panic(vm.NewGoError(fmt.Errorf("sandbox.command expects command(name, [spec], handler)")))
	}
	name := strings.TrimSpace(call.Arguments[0].String())
	if name == "" {
		panic(vm.NewGoError(fmt.Errorf("sandbox.command name is empty")))
	}

	var (
		spec    map[string]any
		handler goja.Callable
	)
	if len(call.Arguments) == 2 {
		var ok bool
		handler, ok = goja.AssertFunction(call.Arguments[1])
		if !ok {
			panic(vm.NewGoError(fmt.Errorf("sandbox.command %q handler is not a function", name)))
		}
	} else {
		spec = exportMap(call.Arguments[1])
		var ok bool
		handler, ok = goja.AssertFunction(call.Arguments[2])
		if !ok {
			panic(vm.NewGoError(fmt.Errorf("sandbox.command %q handler is not a function", name)))
		}
	}

	d.commands = append(d.commands, &commandDraft{name: name, spec: spec, handler: handler})
	return goja.Undefined()
}

func (d *botDraft) event(vm *goja.Runtime, call goja.FunctionCall) goja.Value {
	if len(call.Arguments) != 2 {
		panic(vm.NewGoError(fmt.Errorf("sandbox.event expects event(name, handler)")))
	}
	name := strings.TrimSpace(call.Arguments[0].String())
	if name == "" {
		panic(vm.NewGoError(fmt.Errorf("sandbox.event name is empty")))
	}

	handler, ok := goja.AssertFunction(call.Arguments[1])
	if !ok {
		panic(vm.NewGoError(fmt.Errorf("sandbox.event %q handler is not a function", name)))
	}

	d.events = append(d.events, &eventDraft{name: name, handler: handler})
	return goja.Undefined()
}

func (d *botDraft) configure(vm *goja.Runtime, call goja.FunctionCall) goja.Value {
	if len(call.Arguments) != 1 {
		panic(vm.NewGoError(fmt.Errorf("sandbox.configure expects configure(options)")))
	}
	options := exportMap(call.Arguments[0])
	if len(options) == 0 {
		return goja.Undefined()
	}
	for key, value := range options {
		d.metadata[key] = value
	}
	return goja.Undefined()
}

func (d *botDraft) finalize(vm *goja.Runtime) goja.Value {
	bot := vm.NewObject()
	_ = bot.Set("kind", "sandbox.bot")
	_ = bot.Set("metadata", cloneMap(d.metadata))
	_ = bot.Set("commands", d.commandSnapshots())
	_ = bot.Set("events", d.eventSnapshots())

	commands := append([]*commandDraft(nil), d.commands...)
	events := append([]*eventDraft(nil), d.events...)
	store := d.store
	moduleName := d.moduleName
	metadata := cloneMap(d.metadata)

	_ = bot.Set("describe", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(map[string]any{
			"kind":     "sandbox.bot",
			"metadata": cloneMap(metadata),
			"commands": commandSnapshotsFromDrafts(commands),
			"events":   eventSnapshotsFromDrafts(events),
		})
	})
	_ = bot.Set("dispatchCommand", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) != 1 {
			panic(vm.NewGoError(fmt.Errorf("sandbox.bot.dispatchCommand expects one input object")))
		}
		input := objectFromValue(vm, call.Arguments[0])
		name := strings.TrimSpace(input.Get("name").String())
		if name == "" {
			panic(vm.NewGoError(fmt.Errorf("sandbox.bot.dispatchCommand input name is empty")))
		}
		command := findCommand(commands, name)
		if command == nil {
			panic(vm.NewGoError(fmt.Errorf("sandbox bot %q has no command named %q", moduleName, name)))
		}
		ctx := buildContext(vm, store, input, "command", name, metadata)
		result, err := command.handler(goja.Undefined(), ctx)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		resolved, err := resolveValue(vm, result)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return vm.ToValue(resolved)
	})
	_ = bot.Set("dispatchEvent", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) != 1 {
			panic(vm.NewGoError(fmt.Errorf("sandbox.bot.dispatchEvent expects one input object")))
		}
		input := objectFromValue(vm, call.Arguments[0])
		name := strings.TrimSpace(input.Get("name").String())
		if name == "" {
			panic(vm.NewGoError(fmt.Errorf("sandbox.bot.dispatchEvent input name is empty")))
		}
		matches := findEvents(events, name)
		if len(matches) == 0 {
			panic(vm.NewGoError(fmt.Errorf("sandbox bot %q has no event named %q", moduleName, name)))
		}
		ctx := buildContext(vm, store, input, "event", name, metadata)
		results := make([]any, 0, len(matches))
		for _, ev := range matches {
			result, err := ev.handler(goja.Undefined(), ctx)
			if err != nil {
				panic(vm.NewGoError(err))
			}
			resolved, err := resolveValue(vm, result)
			if err != nil {
				panic(vm.NewGoError(err))
			}
			if resolved != nil {
				results = append(results, resolved)
			}
		}
		return vm.ToValue(results)
	})

	return bot
}

func (d *botDraft) commandSnapshots() []map[string]any {
	return commandSnapshotsFromDrafts(d.commands)
}

func (d *botDraft) eventSnapshots() []map[string]any {
	return eventSnapshotsFromDrafts(d.events)
}

func commandSnapshotsFromDrafts(commands []*commandDraft) []map[string]any {
	out := make([]map[string]any, 0, len(commands))
	for _, command := range commands {
		snapshot := map[string]any{"name": command.name}
		if len(command.spec) > 0 {
			snapshot["spec"] = cloneMap(command.spec)
		}
		out = append(out, snapshot)
	}
	return out
}

func eventSnapshotsFromDrafts(events []*eventDraft) []map[string]any {
	out := make([]map[string]any, 0, len(events))
	for _, event := range events {
		out = append(out, map[string]any{"name": event.name})
	}
	return out
}

func cloneMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func findCommand(commands []*commandDraft, name string) *commandDraft {
	for _, command := range commands {
		if command != nil && command.name == name {
			return command
		}
	}
	return nil
}

func findEvents(events []*eventDraft, name string) []*eventDraft {
	matches := make([]*eventDraft, 0, 1)
	for _, event := range events {
		if event != nil && event.name == name {
			matches = append(matches, event)
		}
	}
	return matches
}

func objectFromValue(vm *goja.Runtime, value goja.Value) *goja.Object {
	if goja.IsUndefined(value) || goja.IsNull(value) {
		return vm.NewObject()
	}
	obj := value.ToObject(vm)
	if obj == nil {
		return vm.NewObject()
	}
	return obj
}

func setObjectField(vm *goja.Runtime, obj *goja.Object, name string, value any) {
	if obj == nil {
		return
	}
	if value == nil {
		_ = obj.Set(name, vm.ToValue(nil))
		return
	}
	_ = obj.Set(name, value)
}

func buildContext(vm *goja.Runtime, store *MemoryStore, input *goja.Object, kind, name string, metadata map[string]any) *goja.Object {
	ctx := vm.NewObject()
	setObjectField(vm, ctx, "args", input.Get("args"))
	setObjectField(vm, ctx, "command", input.Get("command"))
	setObjectField(vm, ctx, "user", input.Get("user"))
	setObjectField(vm, ctx, "guild", input.Get("guild"))
	setObjectField(vm, ctx, "channel", input.Get("channel"))
	setObjectField(vm, ctx, "me", input.Get("me"))
	setObjectField(vm, ctx, "metadata", input.Get("metadata"))
	_ = ctx.Set("store", storeObject(vm, store))
	_ = ctx.Set("log", loggerObject(vm, kind, name, metadata))

	if reply := input.Get("reply"); !goja.IsUndefined(reply) && !goja.IsNull(reply) {
		_ = ctx.Set("reply", reply)
	} else {
		_ = ctx.Set("reply", func(message any) error { return nil })
	}
	if def := input.Get("defer"); !goja.IsUndefined(def) && !goja.IsNull(def) {
		_ = ctx.Set("defer", def)
	} else {
		_ = ctx.Set("defer", func() error { return nil })
	}
	return ctx
}

func storeObject(vm *goja.Runtime, store *MemoryStore) *goja.Object {
	obj := vm.NewObject()
	_ = obj.Set("get", func(key string, defaultValue any) any {
		if store == nil {
			return defaultValue
		}
		return store.Get(key, defaultValue)
	})
	_ = obj.Set("set", func(key string, value any) {
		if store != nil {
			store.Set(key, value)
		}
	})
	_ = obj.Set("delete", func(key string) bool {
		if store == nil {
			return false
		}
		return store.Delete(key)
	})
	_ = obj.Set("keys", func(prefix string) []string {
		if store == nil {
			return nil
		}
		return store.Keys(prefix)
	})
	_ = obj.Set("namespace", func(parts ...string) any {
		if store == nil {
			return storeObject(vm, NewMemoryStore().Namespace(parts...))
		}
		return storeObject(vm, store.Namespace(parts...))
	})
	return obj
}

func loggerObject(vm *goja.Runtime, kind, name string, metadata map[string]any) *goja.Object {
	obj := vm.NewObject()
	baseFields := map[string]any{"sandboxKind": kind, "sandboxName": name}
	for key, value := range metadata {
		baseFields["meta."+key] = value
	}
	setLogMethod := func(level string, fn func(msg string, fields map[string]any)) {
		_ = obj.Set(level, fn)
	}
	setLogMethod("info", func(msg string, fields map[string]any) {
		logEvent := log.Info()
		applyFields(logEvent, baseFields)
		applyFields(logEvent, fields)
		logEvent.Msg(msg)
	})
	setLogMethod("debug", func(msg string, fields map[string]any) {
		logEvent := log.Debug()
		applyFields(logEvent, baseFields)
		applyFields(logEvent, fields)
		logEvent.Msg(msg)
	})
	setLogMethod("warn", func(msg string, fields map[string]any) {
		logEvent := log.Warn()
		applyFields(logEvent, baseFields)
		applyFields(logEvent, fields)
		logEvent.Msg(msg)
	})
	setLogMethod("error", func(msg string, fields map[string]any) {
		logEvent := log.Error()
		applyFields(logEvent, baseFields)
		applyFields(logEvent, fields)
		logEvent.Msg(msg)
	})
	return obj
}

func applyFields(event *zerolog.Event, fields map[string]any) {
	if event == nil || len(fields) == 0 {
		return
	}
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		event.Interface(key, fields[key])
	}
}
