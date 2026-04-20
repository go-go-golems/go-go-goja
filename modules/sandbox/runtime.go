package sandbox

import (
	"fmt"
	"strings"
	"sync"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/modules"
)

// RuntimeStateContextKey stores the per-runtime sandbox state inside the
// engine runtime values map.
const RuntimeStateContextKey = "sandbox.runtime"

var runtimeStateByVM sync.Map

// RegisterRuntimeState binds a runtime-local sandbox state to a specific VM.
func RegisterRuntimeState(vm *goja.Runtime, state *RuntimeState) {
	if vm == nil || state == nil {
		return
	}
	runtimeStateByVM.Store(vm, state)
}

// UnregisterRuntimeState removes the runtime-local sandbox state for a VM.
func UnregisterRuntimeState(vm *goja.Runtime) {
	if vm == nil {
		return
	}
	runtimeStateByVM.Delete(vm)
}

// RuntimeState captures the runtime-local host state for the sandbox module.
type RuntimeState struct {
	moduleName string
	store      *MemoryStore
}

// NewRuntimeState creates a fresh runtime state with an empty in-memory store.
func NewRuntimeState(moduleName string) *RuntimeState {
	name := strings.TrimSpace(moduleName)
	if name == "" {
		name = "sandbox"
	}
	return &RuntimeState{
		moduleName: name,
		store:      NewMemoryStore(),
	}
}

// ModuleName returns the CommonJS require name for this runtime state.
func (s *RuntimeState) ModuleName() string {
	if s == nil || strings.TrimSpace(s.moduleName) == "" {
		return "sandbox"
	}
	return s.moduleName
}

// Store returns the runtime-local in-memory store.
func (s *RuntimeState) Store() *MemoryStore {
	if s == nil {
		return NewMemoryStore()
	}
	if s.store == nil {
		s.store = NewMemoryStore()
	}
	return s.store
}

// Loader exposes the sandbox module to CommonJS consumers.
func (s *RuntimeState) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
	exports := moduleObj.Get("exports").(*goja.Object)
	modules.SetExport(exports, s.ModuleName(), "defineBot", func(call goja.FunctionCall) goja.Value {
		return s.defineBot(vm, call)
	})
}

func (s *RuntimeState) defineBot(vm *goja.Runtime, call goja.FunctionCall) goja.Value {
	if len(call.Arguments) != 1 {
		panic(vm.NewGoError(fmt.Errorf("sandbox.defineBot expects defineBot(builderFn)")))
	}
	builder, ok := goja.AssertFunction(call.Arguments[0])
	if !ok {
		panic(vm.NewGoError(fmt.Errorf("sandbox.defineBot builder is not a function")))
	}

	draft := newBotDraft(s)
	api := vm.NewObject()
	modules.SetExport(api, s.ModuleName(), "command", func(call goja.FunctionCall) goja.Value {
		return draft.command(vm, call)
	})
	modules.SetExport(api, s.ModuleName(), "event", func(call goja.FunctionCall) goja.Value {
		return draft.event(vm, call)
	})
	modules.SetExport(api, s.ModuleName(), "configure", func(call goja.FunctionCall) goja.Value {
		return draft.configure(vm, call)
	})

	if _, err := builder(goja.Undefined(), api); err != nil {
		panic(vm.NewGoError(err))
	}
	return draft.finalize(vm)
}

// LookupRuntimeState returns the runtime-local sandbox state for a VM.
func LookupRuntimeState(vm *goja.Runtime) *RuntimeState {
	if vm == nil {
		return nil
	}
	value, ok := runtimeStateByVM.Load(vm)
	if !ok {
		return nil
	}
	state, _ := value.(*RuntimeState)
	return state
}

type module struct{}

var _ modules.NativeModule = (*module)(nil)

func (module) Name() string { return "sandbox" }

func (module) Doc() string {
	return `
The sandbox module exposes defineBot(builderFn) for runtime-scoped bot scripting.
`
}

func (module) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
	state := LookupRuntimeState(vm)
	if state == nil {
		panic(vm.NewGoError(fmt.Errorf("sandbox runtime state is not registered for this VM")))
	}
	state.Loader(vm, moduleObj)
}

func init() {
	modules.Register(&module{})
}

// ModuleNameOrDefault resolves the module name for callers that want a trimmed
// fallback.
func ModuleNameOrDefault(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "sandbox"
	}
	return name
}
