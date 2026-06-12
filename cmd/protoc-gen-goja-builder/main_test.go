package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/go-go-golems/go-go-goja/cmd/protoc-gen-goja-builder/internal/generator"
	contract "github.com/go-go-golems/go-go-goja/pkg/hashiplugin/contract"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

func TestGeneratorProducesFirstCompanionGoFile(t *testing.T) {
	resp := generateFixtureResponse(t)
	require.Equal(t, "fixture/v1/fixture_goja.pb.go", resp.File[0].GetName())

	goldenPath := "testdata/fixture_goja.pb.go.golden"
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		require.NoError(t, os.WriteFile(goldenPath, []byte(resp.File[0].GetContent()), 0o644))
	}
	golden, err := os.ReadFile(goldenPath)
	require.NoError(t, err)
	require.Equal(t, string(golden), resp.File[0].GetContent())
}

func TestGeneratedCompanionFileCompiles(t *testing.T) {
	req := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{contract.File_jsmodule_proto.Path()},
		Parameter:      proto.String("paths=source_relative,module_name=hashiplugin.contract.v1"),
		ProtoFile:      descriptorsForFile(t, contract.File_jsmodule_proto),
	}
	plugin, err := protogen.Options{ParamFunc: func(_, _ string) error { return nil }}.New(req)
	require.NoError(t, err)
	opts, err := generator.ParseParameter(req.GetParameter())
	require.NoError(t, err)
	require.NoError(t, generator.Generate(plugin, opts))
	resp := plugin.Response()
	require.Empty(t, resp.GetError())
	require.Len(t, resp.File, 1)

	repoRoot, err := filepath.Abs("../..")
	require.NoError(t, err)
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module example.com/compiletest\n\ngo 1.26\n\nrequire github.com/go-go-golems/go-go-goja v0.0.0\n\nreplace github.com/go-go-golems/go-go-goja => "+repoRoot+"\n"), 0o644))
	pkgDir := filepath.Join(tmp, "pkg", "hashiplugin", "contract")
	require.NoError(t, os.MkdirAll(pkgDir, 0o755))
	pbGo, err := os.ReadFile(filepath.Join(repoRoot, "pkg", "hashiplugin", "contract", "jsmodule.pb.go"))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "jsmodule.pb.go"), pbGo, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, filepath.Base(resp.File[0].GetName())), []byte(resp.File[0].GetContent()), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "generated_runtime_test.go"), []byte(generatedRuntimeTestSource()), 0o644))

	cmd := exec.Command("go", "test", "-mod=mod", ".")
	cmd.Dir = pkgDir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))
}

func generateFixtureResponse(t *testing.T) *pluginpb.CodeGeneratorResponse {
	t.Helper()
	req := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"fixture/v1/fixture.proto"},
		Parameter:      proto.String("paths=source_relative,module_name=fixture.custom,emit_dts=false,emit_provider=false,register_global=false,builder_suffix=Builder,message_ref_name=ProtoMessage"),
		ProtoFile: []*descriptorpb.FileDescriptorProto{
			fixtureFileDescriptor(),
		},
	}

	var params []string
	plugin, err := protogen.Options{ParamFunc: func(name, value string) error {
		if value == "" {
			params = append(params, name)
			return nil
		}
		params = append(params, name+"="+value)
		return nil
	}}.New(req)
	require.NoError(t, err)

	opts, err := generator.ParseParameter(req.GetParameter())
	require.NoError(t, err)
	require.NoError(t, generator.Generate(plugin, opts))
	require.NotEmpty(t, params)

	resp := plugin.Response()
	require.Empty(t, resp.GetError())
	require.Len(t, resp.File, 1)
	return resp
}

func descriptorsForFile(t *testing.T, file protoreflect.FileDescriptor) []*descriptorpb.FileDescriptorProto {
	t.Helper()
	seen := map[string]bool{}
	var out []*descriptorpb.FileDescriptorProto
	var visit func(protoreflect.FileDescriptor)
	visit = func(fd protoreflect.FileDescriptor) {
		if seen[fd.Path()] {
			return
		}
		seen[fd.Path()] = true
		for i := 0; i < fd.Imports().Len(); i++ {
			visit(fd.Imports().Get(i).FileDescriptor)
		}
		protoFile := protodesc.ToFileDescriptorProto(fd)
		out = append(out, protoFile)
	}
	visit(file)
	return out
}

func generatedRuntimeTestSource() string {
	return `package contract

import (
	"testing"

	"strings"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/pkg/protogoja"
	"github.com/go-go-golems/go-go-goja/pkg/tsgen/render"
	"github.com/go-go-golems/go-go-goja/pkg/tsgen/spec"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/dtsgen"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

func TestGeneratedTypeScriptModuleRenders(t *testing.T) {
	module := GojaBuilderFileJsmoduleProtoTypeScriptModule("hashiplugin.contract.v1")
	if module == nil {
		t.Fatalf("TypeScript module is nil")
	}
	out, err := render.Bundle(&spec.Bundle{Modules: []*spec.Module{module}})
	if err != nil {
		t.Fatalf("render DTS: %v", err)
	}
	for _, want := range []string{
		"declare module \"hashiplugin.contract.v1\"",
		"export interface ProtoMessage",
		"export interface ModuleManifestBuilder",
		"moduleName(value: string): this;",
		"export const ExportKind",
		"export type ExportKindValue",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("rendered DTS missing %q in:\n%s", want, out)
		}
	}

	registry := providerapi.NewProviderRegistry()
	if err := registry.Package("fixture", providerapi.Module{
		Name:       "hashiplugin.contract.v1",
		TypeScript: module,
		NewModuleFactory: func(providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
			return nil, nil
		},
	}); err != nil {
		t.Fatalf("register provider module: %v", err)
	}
	result, err := dtsgen.RenderModules(registry, []dtsgen.ModuleInstance{{Package: "fixture", Name: "hashiplugin.contract.v1"}}, dtsgen.Options{})
	if err != nil {
		t.Fatalf("xgoja dtsgen render: %v", err)
	}
	if !strings.Contains(result.DTS, "declare module \"hashiplugin.contract.v1\"") {
		t.Fatalf("xgoja DTS missing generated module declaration:\n%s", result.DTS)
	}
}

func TestGeneratedModuleManifestBuilderRuntime(t *testing.T) {
	vm := goja.New()
	ns, err := NewModuleManifestGojaNamespace(vm)
	if err != nil {
		t.Fatalf("namespace: %v", err)
	}
	prototype, ok := protogoja.MessagePrototypeFromValue(ns)
	if !ok {
		t.Fatalf("namespace has no message prototype")
	}
	if string(prototype.TypeName()) != "hashiplugin.contract.v1.ModuleManifest" {
		t.Fatalf("prototype type = %s", prototype.TypeName())
	}
	if _, ok := prototype.NewMessage().(*ModuleManifest); !ok {
		t.Fatalf("prototype created wrong message type")
	}
	builderFn, ok := goja.AssertFunction(ns.Get("builder"))
	if !ok {
		t.Fatalf("builder is not callable")
	}
	builderValue, err := builderFn(goja.Undefined())
	if err != nil {
		t.Fatalf("builder call: %v", err)
	}
	builder := builderValue.ToObject(vm)
	moduleNameFn, ok := goja.AssertFunction(builder.Get("moduleName"))
	if !ok {
		t.Fatalf("moduleName is not callable")
	}
	if _, err := moduleNameFn(builder, vm.ToValue("demo")); err != nil {
		t.Fatalf("moduleName call: %v", err)
	}
	versionFn, ok := goja.AssertFunction(builder.Get("version"))
	if !ok {
		t.Fatalf("version is not callable")
	}
	if _, err := versionFn(builder, vm.ToValue("v1")); err != nil {
		t.Fatalf("version call: %v", err)
	}
	buildFn, ok := goja.AssertFunction(builder.Get("build"))
	if !ok {
		t.Fatalf("build is not callable")
	}
	builtValue, err := buildFn(builder)
	if err != nil {
		t.Fatalf("build call: %v", err)
	}
	msg, ok := protogoja.MessageFromValue(builtValue)
	if !ok {
		t.Fatalf("built value is not a ProtoMessage")
	}
	manifest, ok := msg.(*ModuleManifest)
	if !ok {
		t.Fatalf("built message type = %T", msg)
	}
	if manifest.GetModuleName() != "demo" || manifest.GetVersion() != "v1" {
		t.Fatalf("unexpected manifest: module_name=%q version=%q", manifest.GetModuleName(), manifest.GetVersion())
	}
	isFn, ok := goja.AssertFunction(ns.Get("is"))
	if !ok {
		t.Fatalf("is is not callable")
	}
	isValue, err := isFn(ns, builtValue)
	if err != nil {
		t.Fatalf("is call: %v", err)
	}
	if !isValue.ToBoolean() {
		t.Fatalf("namespace did not recognize built message")
	}
}

func TestGeneratedExportSpecEnumSetterRuntime(t *testing.T) {
	vm := goja.New()
	enumObj, err := NewExportKindGojaEnum(vm)
	if err != nil {
		t.Fatalf("enum: %v", err)
	}
	ns, err := NewExportSpecGojaNamespace(vm)
	if err != nil {
		t.Fatalf("namespace: %v", err)
	}
	builderFn, ok := goja.AssertFunction(ns.Get("builder"))
	if !ok {
		t.Fatalf("builder is not callable")
	}
	builderValue, err := builderFn(goja.Undefined())
	if err != nil {
		t.Fatalf("builder call: %v", err)
	}
	builder := builderValue.ToObject(vm)
	nameFn, ok := goja.AssertFunction(builder.Get("name"))
	if !ok {
		t.Fatalf("name is not callable")
	}
	if _, err := nameFn(builder, vm.ToValue("run")); err != nil {
		t.Fatalf("name call: %v", err)
	}
	kindFn, ok := goja.AssertFunction(builder.Get("kind"))
	if !ok {
		t.Fatalf("kind is not callable")
	}
	if _, err := kindFn(builder, enumObj.Get("EXPORT_KIND_FUNCTION")); err != nil {
		t.Fatalf("kind call: %v", err)
	}
	buildFn, ok := goja.AssertFunction(builder.Get("build"))
	if !ok {
		t.Fatalf("build is not callable")
	}
	builtValue, err := buildFn(builder)
	if err != nil {
		t.Fatalf("build call: %v", err)
	}
	msg, ok := protogoja.MessageFromValue(builtValue)
	if !ok {
		t.Fatalf("built value is not a ProtoMessage")
	}
	exportSpec, ok := msg.(*ExportSpec)
	if !ok {
		t.Fatalf("built message type = %T", msg)
	}
	if exportSpec.GetName() != "run" || exportSpec.GetKind() != ExportKind_EXPORT_KIND_FUNCTION {
		t.Fatalf("unexpected export spec: name=%q kind=%v", exportSpec.GetName(), exportSpec.GetKind())
	}
}
`
}

func TestParseParameterRejectsUnknownOption(t *testing.T) {
	_, err := generator.ParseParameter("module_name=fixture,unknown=true")
	require.ErrorContains(t, err, "unknown option")
}

func fixtureFileDescriptor() *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{
		Name:    proto.String("fixture/v1/fixture.proto"),
		Syntax:  proto.String("proto3"),
		Package: proto.String("fixture.v1"),
		Options: &descriptorpb.FileOptions{
			GoPackage: proto.String("example.com/fixture/v1;fixturev1"),
		},
		EnumType: []*descriptorpb.EnumDescriptorProto{
			{
				Name: proto.String("ExampleKind"),
				Value: []*descriptorpb.EnumValueDescriptorProto{
					{Name: proto.String("EXAMPLE_KIND_UNSPECIFIED"), Number: proto.Int32(0)},
					{Name: proto.String("EXAMPLE_KIND_PRIMARY"), Number: proto.Int32(1)},
				},
			},
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("Example"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     proto.String("name"),
						Number:   proto.Int32(1),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
						JsonName: proto.String("name"),
					},
					{
						Name:     proto.String("kind"),
						Number:   proto.Int32(2),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(),
						TypeName: proto.String(".fixture.v1.ExampleKind"),
						JsonName: proto.String("kind"),
					},
				},
				NestedType: []*descriptorpb.DescriptorProto{
					{
						Name: proto.String("Nested"),
						Field: []*descriptorpb.FieldDescriptorProto{
							{
								Name:     proto.String("enabled"),
								Number:   proto.Int32(1),
								Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
								Type:     descriptorpb.FieldDescriptorProto_TYPE_BOOL.Enum(),
								JsonName: proto.String("enabled"),
							},
						},
					},
				},
			},
		},
	}
}
