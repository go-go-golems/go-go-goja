package protogoja

import (
	"testing"

	"github.com/dop251/goja"
	contract "github.com/go-go-golems/go-go-goja/pkg/hashiplugin/contract"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestBuilderRefSetBuildCloneAndClear(t *testing.T) {
	vm := goja.New()
	builder, err := NewBuilder(&contract.ModuleManifest{})
	require.NoError(t, err)
	desc := builder.Descriptor()

	require.NoError(t, builder.Set(vm, desc.Fields().ByName("module_name"), vm.ToValue("demo")))
	require.NoError(t, builder.Set(vm, desc.Fields().ByName("version"), vm.ToValue("v1")))

	built := builder.Build().(*contract.ModuleManifest)
	require.Equal(t, "demo", built.GetModuleName())
	require.Equal(t, "v1", built.GetVersion())

	built.ModuleName = "mutated"
	require.Equal(t, "demo", builder.Build().(*contract.ModuleManifest).GetModuleName())

	clone, err := builder.Clone()
	require.NoError(t, err)
	require.NoError(t, clone.Set(vm, desc.Fields().ByName("module_name"), vm.ToValue("clone")))
	require.Equal(t, "demo", builder.Build().(*contract.ModuleManifest).GetModuleName())
	require.Equal(t, "clone", clone.Build().(*contract.ModuleManifest).GetModuleName())

	require.NoError(t, builder.Clear(desc.Fields().ByName("version")))
	require.Empty(t, builder.Build().(*contract.ModuleManifest).GetVersion())
}

func TestBuilderRefRepeatedAndMessageFields(t *testing.T) {
	vm := goja.New()
	builder, err := NewBuilder(&contract.ModuleManifest{})
	require.NoError(t, err)
	desc := builder.Descriptor()

	_, err = vm.RunString(`globalThis.capabilities = ["tools", "storage"]`)
	require.NoError(t, err)
	require.NoError(t, builder.Set(vm, desc.Fields().ByName("capabilities"), vm.Get("capabilities")))

	exportValue, err := ToValue(vm, &contract.ExportSpec{Name: "run", Kind: contract.ExportKind_EXPORT_KIND_FUNCTION})
	require.NoError(t, err)
	require.NoError(t, builder.Add(vm, desc.Fields().ByName("exports"), exportValue))

	built := builder.Build().(*contract.ModuleManifest)
	require.Equal(t, []string{"tools", "storage"}, built.GetCapabilities())
	require.Len(t, built.GetExports(), 1)
	require.Equal(t, "run", built.GetExports()[0].GetName())
	require.Equal(t, contract.ExportKind_EXPORT_KIND_FUNCTION, built.GetExports()[0].GetKind())
}

func TestBuilderRefAcceptsBuilderRefsForMessageFields(t *testing.T) {
	vm := goja.New()
	parent, err := NewBuilder(&contract.ModuleManifest{})
	require.NoError(t, err)
	parentDesc := parent.Descriptor()

	child, err := NewBuilder(&contract.ExportSpec{})
	require.NoError(t, err)
	childDesc := child.Descriptor()
	require.NoError(t, child.Set(vm, childDesc.Fields().ByName("name"), vm.ToValue("run")))
	require.NoError(t, child.Set(vm, childDesc.Fields().ByName("kind"), vm.ToValue("EXPORT_KIND_FUNCTION")))

	builderObject := vm.NewObject()
	require.NoError(t, AttachBuilderRef(vm, builderObject, child))
	require.Empty(t, builderObject.Keys())

	require.NoError(t, parent.Add(vm, parentDesc.Fields().ByName("exports"), builderObject))

	// Mutating the child builder after conversion must not mutate the value that
	// was appended to the parent builder.
	require.NoError(t, child.Set(vm, childDesc.Fields().ByName("name"), vm.ToValue("mutated")))

	built := parent.Build().(*contract.ModuleManifest)
	require.Len(t, built.GetExports(), 1)
	require.Equal(t, "run", built.GetExports()[0].GetName())
	require.Equal(t, contract.ExportKind_EXPORT_KIND_FUNCTION, built.GetExports()[0].GetKind())

	wrongChild, err := NewBuilder(&contract.InvokeRequest{ExportName: "bad"})
	require.NoError(t, err)
	wrongObject := vm.NewObject()
	require.NoError(t, AttachBuilderRef(vm, wrongObject, wrongChild))
	err = parent.Add(vm, parentDesc.Fields().ByName("exports"), wrongObject)
	require.ErrorContains(t, err, "expected hashiplugin.contract.v1.ExportSpec ProtoMessage or builder")
}

func TestBuilderRefEnumSetters(t *testing.T) {
	vm := goja.New()
	builder, err := NewBuilder(&contract.ExportSpec{})
	require.NoError(t, err)
	desc := builder.Descriptor()
	kindField := desc.Fields().ByName("kind")

	require.NoError(t, builder.Set(vm, kindField, vm.ToValue("EXPORT_KIND_FUNCTION")))
	require.Equal(t, contract.ExportKind_EXPORT_KIND_FUNCTION, builder.Build().(*contract.ExportSpec).GetKind())

	require.NoError(t, builder.Set(vm, kindField, vm.ToValue(int64(contract.ExportKind_EXPORT_KIND_OBJECT))))
	require.Equal(t, contract.ExportKind_EXPORT_KIND_OBJECT, builder.Build().(*contract.ExportSpec).GetKind())

	err = builder.Set(vm, kindField, vm.ToValue("EXPORT_KIND_MISSING"))
	require.ErrorContains(t, err, "unknown enum name")
}

func TestBuilderRefRejectsWrongFieldAndWrongMessageType(t *testing.T) {
	vm := goja.New()
	builder, err := NewBuilder(&contract.ModuleManifest{})
	require.NoError(t, err)
	otherDesc := (&contract.InvokeRequest{}).ProtoReflect().Descriptor()

	err = builder.Set(vm, otherDesc.Fields().ByName("export_name"), vm.ToValue("bad"))
	require.ErrorContains(t, err, "does not belong")

	exportsField := builder.Descriptor().Fields().ByName("exports")
	wrongMessage, err := ToValue(vm, &contract.InvokeRequest{ExportName: "bad"})
	require.NoError(t, err)
	err = builder.Add(vm, exportsField, wrongMessage)
	require.ErrorContains(t, err, "expected hashiplugin.contract.v1.ExportSpec ProtoMessage")
}

func TestBuilderRefIntegerSafety(t *testing.T) {
	vm := goja.New()
	builder, err := NewBuilder(&contract.MethodSpec{})
	require.NoError(t, err)
	// MethodSpec has no integer fields in the current fixture, so exercise the
	// conversion helper directly with a string field descriptor only for error
	// path stability.
	field := builder.Descriptor().Fields().ByName("name")
	_, err = valueForField(vm, field, vm.ToValue(123))
	require.ErrorContains(t, err, "expected string")
}

func TestNewBuilderRejectsNilAndBuildClone(t *testing.T) {
	_, err := NewBuilder(nil)
	require.ErrorContains(t, err, "nil proto message")

	builder, err := NewBuilder(&contract.ModuleManifest{ModuleName: "demo"})
	require.NoError(t, err)
	first := builder.Build()
	second := builder.Build()
	require.True(t, proto.Equal(first, second))
	first.(*contract.ModuleManifest).ModuleName = "changed"
	require.Equal(t, "demo", second.(*contract.ModuleManifest).GetModuleName())
}
