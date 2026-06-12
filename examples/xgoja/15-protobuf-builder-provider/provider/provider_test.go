package provider

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	taskpb "github.com/go-go-golems/go-go-goja/examples/xgoja/15-protobuf-builder-provider/proto"
	"github.com/go-go-golems/go-go-goja/pkg/protogoja"
	"github.com/go-go-golems/go-go-goja/pkg/tsgen/render"
	"github.com/go-go-golems/go-go-goja/pkg/tsgen/spec"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/dtsgen"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestProviderRegistersGeneratedProtobufBuilderModule(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	if err := Register(registry); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	module, ok := registry.ResolveModule(PackageID, ModuleName)
	if !ok {
		t.Fatalf("provider module %s/%s not registered", PackageID, ModuleName)
	}
	if module.TypeScript == nil {
		t.Fatalf("provider module did not expose generated TypeScript descriptor")
	}

	dts, err := render.Bundle(&spec.Bundle{Modules: []*spec.Module{module.TypeScript}})
	if err != nil {
		t.Fatalf("render generated DTS: %v", err)
	}
	for _, want := range []string{
		"declare module \"examples.xgoja.protobuf.v1\"",
		"export interface TaskBuilder",
		"addTags(value: string): this;",
		"putLabels(key: string, value: string): this;",
		"export const TaskPriority",
	} {
		if !strings.Contains(dts, want) {
			t.Fatalf("generated DTS missing %q in:\n%s", want, dts)
		}
	}

	result, err := dtsgen.RenderModules(registry, []dtsgen.ModuleInstance{{Package: PackageID, Name: ModuleName}}, dtsgen.Options{})
	if err != nil {
		t.Fatalf("render xgoja DTS bundle: %v", err)
	}
	if !strings.Contains(result.DTS, "declare module \"examples.xgoja.protobuf.v1\"") {
		t.Fatalf("xgoja DTS bundle missing protobuf module:\n%s", result.DTS)
	}

	loader, err := module.NewModuleFactory(providerapi.ModuleSetupContext{Name: ModuleName})
	if err != nil {
		t.Fatalf("module factory: %v", err)
	}
	vm := goja.New()
	requireRegistry := require.NewRegistry()
	requireRegistry.RegisterNativeModule(ModuleName, loader)
	requireRegistry.Enable(vm)

	buildTaskScript, err := os.ReadFile(filepath.Join("..", "scripts", "build-task.js"))
	if err != nil {
		t.Fatalf("read build-task.js: %v", err)
	}
	moduleObj := vm.NewObject()
	exports := vm.NewObject()
	if err := moduleObj.Set("exports", exports); err != nil {
		t.Fatalf("set script exports: %v", err)
	}
	wrapped := "(function(exports, require) {\n" + string(buildTaskScript) + "\n})"
	fn, err := vm.RunString(wrapped)
	if err != nil {
		t.Fatalf("compile script: %v", err)
	}
	call, ok := goja.AssertFunction(fn)
	if !ok {
		t.Fatalf("wrapped script is not callable")
	}
	if _, err := call(goja.Undefined(), exports, vm.Get("require")); err != nil {
		t.Fatalf("execute script: %v", err)
	}

	taskMsg, ok := protogoja.MessageFromValue(exports.Get("task"))
	if !ok {
		t.Fatalf("script task export is not a generated ProtoMessage")
	}
	task, ok := taskMsg.(*taskpb.Task)
	if !ok {
		t.Fatalf("task export type = %T", taskMsg)
	}
	if task.GetId() != "task-1" || task.GetTitle() != "Ship protobuf builders" {
		t.Fatalf("unexpected task: %v", task)
	}
	if got := task.GetTags(); len(got) != 2 || got[0] != "protobuf" || got[1] != "xgoja" {
		t.Fatalf("unexpected tags: %v", got)
	}
	if task.GetLabels()["component"] != "goja" {
		t.Fatalf("unexpected labels: %v", task.GetLabels())
	}
	if task.GetPriority() != taskpb.TaskPriority_TASK_PRIORITY_HIGH {
		t.Fatalf("priority = %v", task.GetPriority())
	}
	if timestamppb.New(task.GetDueAt().AsTime()).AsTime().Format("2006-01-02T15:04:05Z") != "2026-06-12T20:00:00Z" {
		t.Fatalf("unexpected due_at: %v", task.GetDueAt())
	}
	if task.GetMetadata().GetFields()["owner"].GetStringValue() != "agent" {
		t.Fatalf("unexpected metadata: %v", task.GetMetadata())
	}

	envelopeMsg, ok := protogoja.MessageFromValue(exports.Get("envelope"))
	if !ok {
		t.Fatalf("script envelope export is not a generated ProtoMessage")
	}
	envelope, ok := envelopeMsg.(*taskpb.TaskEnvelope)
	if !ok {
		t.Fatalf("envelope export type = %T", envelopeMsg)
	}
	if envelope.GetSource() != "script" || envelope.GetTask().GetId() != "task-1" {
		t.Fatalf("unexpected envelope: %v", envelope)
	}
}
