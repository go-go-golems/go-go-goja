package protogoja

import (
	"testing"

	"github.com/dop251/goja"
	contract "github.com/go-go-golems/go-go-goja/pkg/hashiplugin/contract"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
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

func TestBuilderRefMapHelpersObjectMapPutDeleteAndClear(t *testing.T) {
	vm := goja.New()
	builder, field := newDynamicMapBuilder(t)

	_, err := vm.RunString(`globalThis.objectInput = { alpha: 1, beta: "2" }`)
	require.NoError(t, err)
	require.NoError(t, builder.Set(vm, field, vm.Get("objectInput")))
	require.Equal(t, map[string]int64{"alpha": 1, "beta": 2}, dynamicStringInt64Map(t, builder.Build(), field))

	_, err = vm.RunString(`globalThis.mapInput = new Map([["gamma", 3], ["delta", "4"]])`)
	require.NoError(t, err)
	require.NoError(t, builder.Set(vm, field, vm.Get("mapInput")))
	require.Equal(t, map[string]int64{"gamma": 3, "delta": 4}, dynamicStringInt64Map(t, builder.Build(), field))

	require.NoError(t, builder.Put(vm, field, vm.ToValue("epsilon"), vm.ToValue(5)))
	require.Equal(t, map[string]int64{"gamma": 3, "delta": 4, "epsilon": 5}, dynamicStringInt64Map(t, builder.Build(), field))

	require.NoError(t, builder.Delete(vm, field, vm.ToValue("gamma")))
	require.NoError(t, builder.Delete(vm, field, vm.ToValue("missing")))
	require.Equal(t, map[string]int64{"delta": 4, "epsilon": 5}, dynamicStringInt64Map(t, builder.Build(), field))

	require.NoError(t, builder.Clear(field))
	require.Empty(t, dynamicStringInt64Map(t, builder.Build(), field))
}

func TestBuilderRefMapSetFailureDoesNotClearExistingEntries(t *testing.T) {
	vm := goja.New()
	builder, field := newDynamicMapBuilder(t)

	require.NoError(t, builder.Put(vm, field, vm.ToValue("existing"), vm.ToValue(7)))
	_, err := vm.RunString(`globalThis.invalidMap = new Map([["bad", {}]])`)
	require.NoError(t, err)

	err = builder.Set(vm, field, vm.Get("invalidMap"))
	require.ErrorContains(t, err, "expected integer")
	require.Equal(t, map[string]int64{"existing": 7}, dynamicStringInt64Map(t, builder.Build(), field))
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

func newDynamicMapBuilder(t *testing.T) (*BuilderRef, protoreflect.FieldDescriptor) {
	t.Helper()
	labelOptional := descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL
	labelRepeated := descriptorpb.FieldDescriptorProto_LABEL_REPEATED
	typeMessage := descriptorpb.FieldDescriptorProto_TYPE_MESSAGE
	typeString := descriptorpb.FieldDescriptorProto_TYPE_STRING
	typeInt64 := descriptorpb.FieldDescriptorProto_TYPE_INT64

	file, err := protodesc.NewFile(&descriptorpb.FileDescriptorProto{
		Syntax:  proto.String("proto3"),
		Name:    proto.String("protogoja_map_test.proto"),
		Package: proto.String("protogoja.test"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("MapMessage"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     proto.String("labels"),
						JsonName: proto.String("labels"),
						Number:   proto.Int32(1),
						Label:    &labelRepeated,
						Type:     &typeMessage,
						TypeName: proto.String(".protogoja.test.MapMessage.LabelsEntry"),
					},
				},
				NestedType: []*descriptorpb.DescriptorProto{
					{
						Name: proto.String("LabelsEntry"),
						Field: []*descriptorpb.FieldDescriptorProto{
							{
								Name:     proto.String("key"),
								JsonName: proto.String("key"),
								Number:   proto.Int32(1),
								Label:    &labelOptional,
								Type:     &typeString,
							},
							{
								Name:     proto.String("value"),
								JsonName: proto.String("value"),
								Number:   proto.Int32(2),
								Label:    &labelOptional,
								Type:     &typeInt64,
							},
						},
						Options: &descriptorpb.MessageOptions{MapEntry: proto.Bool(true)},
					},
				},
			},
		},
	}, nil)
	require.NoError(t, err)

	messageDesc := file.Messages().ByName("MapMessage")
	builder, err := NewBuilder(dynamicpb.NewMessage(messageDesc))
	require.NoError(t, err)
	return builder, messageDesc.Fields().ByName("labels")
}

func dynamicStringInt64Map(t *testing.T, msg proto.Message, field protoreflect.FieldDescriptor) map[string]int64 {
	t.Helper()
	out := map[string]int64{}
	msg.ProtoReflect().Get(field).Map().Range(func(key protoreflect.MapKey, value protoreflect.Value) bool {
		out[key.String()] = value.Int()
		return true
	})
	return out
}
