package events

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/modules"
	"github.com/go-go-golems/go-go-goja/pkg/tsgen/spec"
)

type module struct {
	name string
}

var _ modules.NativeModule = (*module)(nil)
var _ modules.TypeScriptDeclarer = (*module)(nil)

// EventEmitter is the Go-native backing object for JavaScript EventEmitter
// instances returned by require("events").
//
// It is not goroutine-safe. All methods that touch listeners or goja values must
// be called on the owning goja runtime goroutine.
type EventEmitter struct {
	vm        *goja.Runtime
	object    *goja.Object
	listeners map[string][]listenerEntry
}

type listenerEntry struct {
	value    goja.Value
	callable goja.Callable
	once     bool
	original goja.Value
}

var eventEmitterType = reflect.TypeOf((*EventEmitter)(nil))

func (m *module) Name() string { return m.name }

func (m *module) Doc() string {
	return `
The events module provides a Go-native subset of Node.js EventEmitter.

Exports:
  EventEmitter / module.exports: constructor for Go-backed EventEmitter objects.

Supported methods:
  on/addListener, once, off/removeListener, removeAllListeners, emit,
  listeners, rawListeners, listenerCount, eventNames.
`
}

func (m *module) TypeScriptModule() *spec.Module {
	return &spec.Module{
		Name: m.name,
		RawDTS: []string{
			"type EventName = string | symbol;",
			"type Listener = (...args: any[]) => void;",
			"class EventEmitter {",
			"  constructor();",
			"  on(name: EventName, listener: Listener): this;",
			"  addListener(name: EventName, listener: Listener): this;",
			"  once(name: EventName, listener: Listener): this;",
			"  off(name: EventName, listener: Listener): this;",
			"  removeListener(name: EventName, listener: Listener): this;",
			"  removeAllListeners(name?: EventName): this;",
			"  emit(name: EventName, ...args: any[]): boolean;",
			"  listeners(name: EventName): Listener[];",
			"  rawListeners(name: EventName): Listener[];",
			"  listenerCount(name: EventName): number;",
			"  eventNames(): EventName[];",
			"}",
			"export = EventEmitter;",
			"export { EventEmitter };",
		},
	}
}

func (m *module) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
	constructor := vm.ToValue(func(call goja.ConstructorCall) *goja.Object {
		emitter := New(vm)
		obj := vm.ToValue(emitter).(*goja.Object)
		obj.SetPrototype(call.This.Prototype())
		emitter.object = obj
		return obj
	}).(*goja.Object)

	proto := vm.NewObject()
	mustSet(vm, proto, "on", func(call goja.FunctionCall) goja.Value {
		return methodOn(vm, call, false)
	})
	mustSet(vm, proto, "addListener", func(call goja.FunctionCall) goja.Value {
		return methodOn(vm, call, false)
	})
	mustSet(vm, proto, "once", func(call goja.FunctionCall) goja.Value {
		return methodOn(vm, call, true)
	})
	mustSet(vm, proto, "off", func(call goja.FunctionCall) goja.Value {
		return methodRemoveListener(vm, call)
	})
	mustSet(vm, proto, "removeListener", func(call goja.FunctionCall) goja.Value {
		return methodRemoveListener(vm, call)
	})
	mustSet(vm, proto, "removeAllListeners", func(call goja.FunctionCall) goja.Value {
		emitter := mustEmitter(vm, call.This)
		if len(call.Arguments) == 0 || goja.IsUndefined(call.Argument(0)) {
			emitter.RemoveAllListeners()
		} else {
			emitter.RemoveAllListeners(eventKey(call.Argument(0)))
		}
		return call.This
	})
	mustSet(vm, proto, "emit", func(call goja.FunctionCall) goja.Value {
		emitter := mustEmitter(vm, call.This)
		name := eventKey(call.Argument(0))
		ok, err := emitter.Emit(name, call.Arguments[1:]...)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return vm.ToValue(ok)
	})
	mustSet(vm, proto, "listeners", func(call goja.FunctionCall) goja.Value {
		emitter := mustEmitter(vm, call.This)
		return vm.ToValue(emitter.Listeners(eventKey(call.Argument(0)), true))
	})
	mustSet(vm, proto, "rawListeners", func(call goja.FunctionCall) goja.Value {
		emitter := mustEmitter(vm, call.This)
		return vm.ToValue(emitter.Listeners(eventKey(call.Argument(0)), false))
	})
	mustSet(vm, proto, "listenerCount", func(call goja.FunctionCall) goja.Value {
		emitter := mustEmitter(vm, call.This)
		return vm.ToValue(emitter.ListenerCount(eventKey(call.Argument(0))))
	})
	mustSet(vm, proto, "eventNames", func(call goja.FunctionCall) goja.Value {
		emitter := mustEmitter(vm, call.This)
		return vm.ToValue(emitter.EventNames())
	})

	mustSet(vm, constructor, "prototype", proto)
	proto.DefineDataProperty("constructor", constructor, goja.FLAG_FALSE, goja.FLAG_FALSE, goja.FLAG_FALSE)
	mustSet(vm, constructor, "EventEmitter", constructor)
	mustSet(vm, constructor, "default", constructor)

	if err := moduleObj.Set("exports", constructor); err != nil {
		panic(vm.NewGoError(fmt.Errorf("events: set exports: %w", err)))
	}
}

// New creates a Go-native EventEmitter backing value for vm. The caller is
// responsible for wrapping it in a goja object when exposing it to JavaScript.
func New(vm *goja.Runtime) *EventEmitter {
	return &EventEmitter{
		vm:        vm,
		listeners: map[string][]listenerEntry{},
	}
}

// FromValue unwraps a JavaScript value created by the Go-native EventEmitter
// constructor.
func FromValue(value goja.Value) (*EventEmitter, *goja.Object, bool) {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return nil, nil, false
	}
	if value.ExportType() != eventEmitterType {
		return nil, nil, false
	}
	emitter, ok := value.Export().(*EventEmitter)
	if !ok || emitter == nil || emitter.vm == nil {
		return nil, nil, false
	}
	obj := value.ToObject(emitter.vm)
	if emitter.object == nil {
		emitter.object = obj
	}
	return emitter, obj, true
}

// AddListenerValue registers a JavaScript callable value as a listener.
func (e *EventEmitter) AddListenerValue(name string, value goja.Value) error {
	callable, ok := goja.AssertFunction(value)
	if !ok {
		return fmt.Errorf("listener must be a function")
	}
	e.addListener(name, listenerEntry{value: value, callable: callable})
	return nil
}

// AddGoListener registers a Go function as a listener. It must be called on the
// owner goroutine for e's runtime.
func (e *EventEmitter) AddGoListener(name string, fn func(goja.FunctionCall) goja.Value) error {
	if e == nil || e.vm == nil {
		return fmt.Errorf("events: nil emitter")
	}
	return e.AddListenerValue(name, e.vm.ToValue(fn))
}

// Emit invokes all listeners for name synchronously on the owner goroutine.
func (e *EventEmitter) Emit(name string, args ...goja.Value) (bool, error) {
	if e == nil {
		return false, fmt.Errorf("events: nil emitter")
	}
	return e.emit(name, args)
}

func (e *EventEmitter) addListener(name string, entry listenerEntry) {
	if e.listeners == nil {
		e.listeners = map[string][]listenerEntry{}
	}
	e.listeners[name] = append(e.listeners[name], entry)
}

func (e *EventEmitter) emit(name string, args []goja.Value) (bool, error) {
	list := append([]listenerEntry(nil), e.listeners[name]...)
	if len(list) == 0 {
		if name == "error" {
			return false, e.unhandledError(args)
		}
		return false, nil
	}
	for _, entry := range list {
		if entry.once {
			e.removeListenerEntry(name, entry.value)
		}
		if _, err := entry.callable(e.thisObject(), args...); err != nil {
			return true, err
		}
	}
	return true, nil
}

func (e *EventEmitter) unhandledError(args []goja.Value) error {
	if len(args) == 0 || goja.IsUndefined(args[0]) || goja.IsNull(args[0]) {
		return fmt.Errorf("Unhandled error event")
	}
	obj := args[0].ToObject(e.vm)
	if msg := obj.Get("message"); msg != nil && !goja.IsUndefined(msg) && !goja.IsNull(msg) {
		return fmt.Errorf("Unhandled error event: %s", msg.String())
	}
	return fmt.Errorf("Unhandled error event: %s", args[0].String())
}

func (e *EventEmitter) removeListenerEntry(name string, value goja.Value) (listenerEntry, bool) {
	list := e.listeners[name]
	for i, entry := range list {
		if sameListener(entry, value) {
			copy(list[i:], list[i+1:])
			list = list[:len(list)-1]
			if len(list) == 0 {
				delete(e.listeners, name)
			} else {
				e.listeners[name] = list
			}
			return entry, true
		}
	}
	return listenerEntry{}, false
}

// RemoveListener removes the first listener matching value.
func (e *EventEmitter) RemoveListener(name string, value goja.Value) bool {
	removed, ok := e.removeListenerEntry(name, value)
	if ok && name != "removeListener" {
		listener := removed.value
		if removed.original != nil {
			listener = removed.original
		}
		_, _ = e.emit("removeListener", []goja.Value{e.vm.ToValue(name), listener})
	}
	return ok
}

// RemoveAllListeners clears either one event's listeners or every listener.
func (e *EventEmitter) RemoveAllListeners(names ...string) {
	if len(names) == 0 {
		e.listeners = map[string][]listenerEntry{}
		return
	}
	delete(e.listeners, names[0])
}

// Listeners returns a copy of listener values for name.
func (e *EventEmitter) Listeners(name string, unwrapOnce bool) []goja.Value {
	list := e.listeners[name]
	out := make([]goja.Value, 0, len(list))
	for _, entry := range list {
		if unwrapOnce && entry.once && entry.original != nil {
			out = append(out, entry.original)
			continue
		}
		out = append(out, entry.value)
	}
	return out
}

// ListenerCount returns the number of registered listeners for name.
func (e *EventEmitter) ListenerCount(name string) int {
	return len(e.listeners[name])
}

// EventNames returns sorted event names with at least one listener.
func (e *EventEmitter) EventNames() []string {
	names := make([]string, 0, len(e.listeners))
	for name, list := range e.listeners {
		if len(list) > 0 {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func methodOn(vm *goja.Runtime, call goja.FunctionCall, once bool) goja.Value {
	emitter := mustEmitter(vm, call.This)
	name := eventKey(call.Argument(0))
	listenerValue := call.Argument(1)
	callable, ok := goja.AssertFunction(listenerValue)
	if !ok {
		panic(vm.NewTypeError("listener must be a function"))
	}
	if name != "newListener" {
		if _, err := emitter.emit("newListener", []goja.Value{vm.ToValue(name), listenerValue}); err != nil {
			panic(err)
		}
	}
	entry := listenerEntry{value: listenerValue, callable: callable, once: once}
	if once {
		entry.original = listenerValue
	}
	emitter.addListener(name, entry)
	return call.This
}

func methodRemoveListener(vm *goja.Runtime, call goja.FunctionCall) goja.Value {
	emitter := mustEmitter(vm, call.This)
	emitter.RemoveListener(eventKey(call.Argument(0)), call.Argument(1))
	return call.This
}

func mustEmitter(vm *goja.Runtime, value goja.Value) *EventEmitter {
	emitter, _, ok := FromValue(value)
	if !ok {
		panic(vm.NewTypeError("Value of this must be an events.EventEmitter"))
	}
	return emitter
}

func sameListener(entry listenerEntry, value goja.Value) bool {
	if entry.value != nil && entry.value.StrictEquals(value) {
		return true
	}
	return entry.original != nil && entry.original.StrictEquals(value)
}

func eventKey(value goja.Value) string {
	if value == nil || goja.IsUndefined(value) {
		return "undefined"
	}
	return value.String()
}

func (e *EventEmitter) thisObject() goja.Value {
	if e.object != nil {
		return e.object
	}
	return e.vm.ToValue(e)
}

func mustSet(vm *goja.Runtime, obj *goja.Object, name string, value interface{}) {
	if err := obj.Set(name, value); err != nil {
		panic(vm.NewGoError(fmt.Errorf("events: set %s: %w", name, err)))
	}
}

func init() {
	modules.Register(&module{name: "events"})
	modules.Register(&module{name: "node:events"})
}
