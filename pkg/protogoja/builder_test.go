package protogoja

import (
	"testing"

	"github.com/dop251/goja"
	contract "github.com/go-go-golems/go-go-goja/pkg/hashiplugin/contract"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
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
	require.ErrorContains(t, err, `labels["bad"]`)
	require.ErrorContains(t, err, "expected integer")
	require.Equal(t, map[string]int64{"existing": 7}, dynamicStringInt64Map(t, builder.Build(), field))
}

func TestBuilderRefWellKnownTypeConversions(t *testing.T) {
	vm := goja.New()
	builder, fields := newDynamicWKTBuilder(t)

	require.NoError(t, builder.Set(vm, fields["timestamp"], vm.ToValue("2026-06-12T16:00:00Z")))
	require.NoError(t, builder.Set(vm, fields["duration"], vm.ToValue("1h2m3s")))
	_, err := vm.RunString(`
		globalThis.timestampDate = new Date("2026-06-12T16:00:00Z");
		globalThis.structInput = { enabled: true, count: 3, nested: { name: "demo" } };
		globalThis.valueInput = { labels: ["a", "b"] };
		globalThis.listInput = [1, "two", false];
	`)
	require.NoError(t, err)
	require.NoError(t, builder.Set(vm, fields["timestamp"], vm.Get("timestampDate")))
	require.NoError(t, err)
	require.NoError(t, builder.Set(vm, fields["struct"], vm.Get("structInput")))
	require.NoError(t, builder.Set(vm, fields["value"], vm.Get("valueInput")))
	require.NoError(t, builder.Set(vm, fields["list"], vm.Get("listInput")))
	require.NoError(t, builder.Set(vm, fields["string_wrapper"], vm.ToValue("wrapped")))
	require.NoError(t, builder.Set(vm, fields["field_mask"], vm.ToValue("module_name,version")))

	manifestValue, err := ToValue(vm, &contract.ModuleManifest{ModuleName: "wrapped-any"})
	require.NoError(t, err)
	require.NoError(t, builder.Set(vm, fields["any"], manifestValue))

	built := builder.Build().ProtoReflect()
	timestamp := knownField(t, built, fields["timestamp"], &timestamppb.Timestamp{})
	duration := knownField(t, built, fields["duration"], &durationpb.Duration{})
	stringWrapper := knownField(t, built, fields["string_wrapper"], &wrapperspb.StringValue{})
	fieldMask := knownField(t, built, fields["field_mask"], &fieldmaskpb.FieldMask{})
	structValue := knownField(t, built, fields["struct"], &structpb.Struct{})
	listValue := knownField(t, built, fields["list"], &structpb.ListValue{})
	value := knownField(t, built, fields["value"], &structpb.Value{})
	anyValue := knownField(t, built, fields["any"], &anypb.Any{})

	require.Equal(t, "2026-06-12T16:00:00Z", timestamp.AsTime().UTC().Format("2006-01-02T15:04:05Z"))
	require.Equal(t, int64(3723), duration.GetSeconds())
	require.Equal(t, int32(0), duration.GetNanos())
	require.Equal(t, "wrapped", stringWrapper.GetValue())
	require.Equal(t, []string{"module_name", "version"}, fieldMask.GetPaths())
	require.Equal(t, true, structValue.Fields["enabled"].GetBoolValue())
	require.Len(t, listValue.Values, 3)
	require.Contains(t, value.GetStructValue().Fields, "labels")
	require.Equal(t, "type.googleapis.com/hashiplugin.contract.v1.ModuleManifest", anyValue.GetTypeUrl())
}

func TestBuilderRefFieldPathRichErrors(t *testing.T) {
	vm := goja.New()
	tests := []struct {
		name     string
		run      func() error
		contains []string
	}{
		{
			name: "repeated index",
			run: func() error {
				builder, err := NewBuilder(&contract.ModuleManifest{})
				require.NoError(t, err)
				field := builder.Descriptor().Fields().ByName("capabilities")
				_, err = vm.RunString(`globalThis.invalidRepeated = ["ok", 123]`)
				require.NoError(t, err)
				return builder.Set(vm, field, vm.Get("invalidRepeated"))
			},
			contains: []string{"capabilities[1]", "expected string"},
		},
		{
			name: "map key",
			run: func() error {
				builder, field := newDynamicMapBuilder(t)
				_, err := vm.RunString(`globalThis.invalidObjectMap = { bad: {} }`)
				require.NoError(t, err)
				return builder.Set(vm, field, vm.Get("invalidObjectMap"))
			},
			contains: []string{`labels["bad"]`, "expected integer"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			require.Error(t, err)
			for _, expected := range tt.contains {
				require.ErrorContains(t, err, expected)
			}
		})
	}
}

func TestBuilderRefOptionalPresenceHelpersHasAndClear(t *testing.T) {
	vm := goja.New()
	builder, optionalField, implicitField := newDynamicOptionalBuilder(t)

	has, err := builder.Has(optionalField)
	require.NoError(t, err)
	require.False(t, has)

	// Explicit presence is independent of the scalar zero value.
	require.NoError(t, builder.Set(vm, optionalField, vm.ToValue("")))
	has, err = builder.Has(optionalField)
	require.NoError(t, err)
	require.True(t, has)
	require.True(t, builder.Build().ProtoReflect().Has(optionalField))

	require.NoError(t, builder.Clear(optionalField))
	has, err = builder.Has(optionalField)
	require.NoError(t, err)
	require.False(t, has)
	require.False(t, builder.Build().ProtoReflect().Has(optionalField))

	// Implicit proto3 scalar fields still follow protobuf reflection semantics:
	// zero values are not present, non-zero values are present.
	require.NoError(t, builder.Set(vm, implicitField, vm.ToValue("")))
	has, err = builder.Has(implicitField)
	require.NoError(t, err)
	require.False(t, has)
	require.NoError(t, builder.Set(vm, implicitField, vm.ToValue("value")))
	has, err = builder.Has(implicitField)
	require.NoError(t, err)
	require.True(t, has)

	otherBuilder, err := NewBuilder(&contract.ModuleManifest{})
	require.NoError(t, err)
	_, err = otherBuilder.Has(optionalField)
	require.ErrorContains(t, err, "does not belong")
}

func TestBuilderRefOneofHelpersWhichAndClear(t *testing.T) {
	vm := goja.New()
	builder, oneof, textField, countField := newDynamicOneofBuilder(t)

	selected, err := builder.WhichOneof(oneof)
	require.NoError(t, err)
	require.Nil(t, selected)

	require.NoError(t, builder.Set(vm, textField, vm.ToValue("hello")))
	selected, err = builder.WhichOneof(oneof)
	require.NoError(t, err)
	require.Equal(t, textField.FullName(), selected.FullName())

	require.NoError(t, builder.Set(vm, countField, vm.ToValue(42)))
	selected, err = builder.WhichOneof(oneof)
	require.NoError(t, err)
	require.Equal(t, countField.FullName(), selected.FullName())
	require.False(t, builder.Build().ProtoReflect().Has(textField))
	require.True(t, builder.Build().ProtoReflect().Has(countField))

	require.NoError(t, builder.ClearOneof(oneof))
	selected, err = builder.WhichOneof(oneof)
	require.NoError(t, err)
	require.Nil(t, selected)
	require.False(t, builder.Build().ProtoReflect().Has(countField))

	otherBuilder, err := NewBuilder(&contract.ModuleManifest{})
	require.NoError(t, err)
	_, err = otherBuilder.WhichOneof(oneof)
	require.ErrorContains(t, err, "does not belong")
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

func knownField[M proto.Message](t *testing.T, msg protoreflect.Message, field protoreflect.FieldDescriptor, target M) M {
	t.Helper()
	encoded, err := proto.Marshal(msg.Get(field).Message().Interface())
	require.NoError(t, err)
	require.NoError(t, proto.Unmarshal(encoded, target))
	return target
}

func newDynamicWKTBuilder(t *testing.T) (*BuilderRef, map[string]protoreflect.FieldDescriptor) {
	t.Helper()
	labelOptional := descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL
	typeMessage := descriptorpb.FieldDescriptorProto_TYPE_MESSAGE
	fields := []*descriptorpb.FieldDescriptorProto{
		wktField("timestamp", 1, ".google.protobuf.Timestamp", &labelOptional, &typeMessage),
		wktField("duration", 2, ".google.protobuf.Duration", &labelOptional, &typeMessage),
		wktField("any", 3, ".google.protobuf.Any", &labelOptional, &typeMessage),
		wktField("struct", 4, ".google.protobuf.Struct", &labelOptional, &typeMessage),
		wktField("value", 5, ".google.protobuf.Value", &labelOptional, &typeMessage),
		wktField("list", 6, ".google.protobuf.ListValue", &labelOptional, &typeMessage),
		wktField("string_wrapper", 7, ".google.protobuf.StringValue", &labelOptional, &typeMessage),
		wktField("field_mask", 8, ".google.protobuf.FieldMask", &labelOptional, &typeMessage),
	}
	file, err := protodesc.NewFile(&descriptorpb.FileDescriptorProto{
		Syntax:  proto.String("proto3"),
		Name:    proto.String("protogoja_wkt_test.proto"),
		Package: proto.String("protogoja.test"),
		Dependency: []string{
			"google/protobuf/any.proto",
			"google/protobuf/duration.proto",
			"google/protobuf/field_mask.proto",
			"google/protobuf/struct.proto",
			"google/protobuf/timestamp.proto",
			"google/protobuf/wrappers.proto",
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name:  proto.String("WKTMessage"),
				Field: fields,
			},
		},
	}, protoregistry.GlobalFiles)
	require.NoError(t, err)

	messageDesc := file.Messages().ByName("WKTMessage")
	builder, err := NewBuilder(dynamicpb.NewMessage(messageDesc))
	require.NoError(t, err)
	out := map[string]protoreflect.FieldDescriptor{}
	for i := 0; i < messageDesc.Fields().Len(); i++ {
		field := messageDesc.Fields().Get(i)
		out[string(field.Name())] = field
	}
	return builder, out
}

func wktField(name string, number int32, typeName string, label *descriptorpb.FieldDescriptorProto_Label, fieldType *descriptorpb.FieldDescriptorProto_Type) *descriptorpb.FieldDescriptorProto {
	return &descriptorpb.FieldDescriptorProto{
		Name:     proto.String(name),
		JsonName: proto.String(name),
		Number:   proto.Int32(number),
		Label:    label,
		Type:     fieldType,
		TypeName: proto.String(typeName),
	}
}

func newDynamicOptionalBuilder(t *testing.T) (*BuilderRef, protoreflect.FieldDescriptor, protoreflect.FieldDescriptor) {
	t.Helper()
	labelOptional := descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL
	typeString := descriptorpb.FieldDescriptorProto_TYPE_STRING

	file, err := protodesc.NewFile(&descriptorpb.FileDescriptorProto{
		Syntax:  proto.String("proto3"),
		Name:    proto.String("protogoja_optional_test.proto"),
		Package: proto.String("protogoja.test"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("OptionalMessage"),
				OneofDecl: []*descriptorpb.OneofDescriptorProto{
					{Name: proto.String("_optional_name")},
				},
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:           proto.String("optional_name"),
						JsonName:       proto.String("optionalName"),
						Number:         proto.Int32(1),
						Label:          &labelOptional,
						Type:           &typeString,
						OneofIndex:     proto.Int32(0),
						Proto3Optional: proto.Bool(true),
					},
					{
						Name:     proto.String("implicit_name"),
						JsonName: proto.String("implicitName"),
						Number:   proto.Int32(2),
						Label:    &labelOptional,
						Type:     &typeString,
					},
				},
			},
		},
	}, nil)
	require.NoError(t, err)

	messageDesc := file.Messages().ByName("OptionalMessage")
	builder, err := NewBuilder(dynamicpb.NewMessage(messageDesc))
	require.NoError(t, err)
	return builder,
		messageDesc.Fields().ByName("optional_name"),
		messageDesc.Fields().ByName("implicit_name")
}

func newDynamicOneofBuilder(t *testing.T) (*BuilderRef, protoreflect.OneofDescriptor, protoreflect.FieldDescriptor, protoreflect.FieldDescriptor) {
	t.Helper()
	labelOptional := descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL
	typeString := descriptorpb.FieldDescriptorProto_TYPE_STRING
	typeInt32 := descriptorpb.FieldDescriptorProto_TYPE_INT32

	file, err := protodesc.NewFile(&descriptorpb.FileDescriptorProto{
		Syntax:  proto.String("proto3"),
		Name:    proto.String("protogoja_oneof_test.proto"),
		Package: proto.String("protogoja.test"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("OneofMessage"),
				OneofDecl: []*descriptorpb.OneofDescriptorProto{
					{Name: proto.String("choice")},
				},
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:       proto.String("text"),
						JsonName:   proto.String("text"),
						Number:     proto.Int32(1),
						Label:      &labelOptional,
						Type:       &typeString,
						OneofIndex: proto.Int32(0),
					},
					{
						Name:       proto.String("count"),
						JsonName:   proto.String("count"),
						Number:     proto.Int32(2),
						Label:      &labelOptional,
						Type:       &typeInt32,
						OneofIndex: proto.Int32(0),
					},
				},
			},
		},
	}, nil)
	require.NoError(t, err)

	messageDesc := file.Messages().ByName("OneofMessage")
	builder, err := NewBuilder(dynamicpb.NewMessage(messageDesc))
	require.NoError(t, err)
	return builder,
		messageDesc.Oneofs().ByName("choice"),
		messageDesc.Fields().ByName("text"),
		messageDesc.Fields().ByName("count")
}
