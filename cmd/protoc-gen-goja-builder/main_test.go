package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/go-go-golems/go-go-goja/cmd/protoc-gen-goja-builder/internal/generator"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

func TestGeneratorProducesFirstCompanionGoFile(t *testing.T) {
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
	require.Equal(t, "fixture/v1/fixture_goja.pb.go", resp.File[0].GetName())

	golden, err := os.ReadFile("testdata/fixture_goja.pb.go.golden")
	require.NoError(t, err)
	require.Equal(t, string(golden), resp.File[0].GetContent())
}

func TestGeneratedCompanionFileCompiles(t *testing.T) {
	req := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"fixture/v1/fixture.proto"},
		Parameter:      proto.String("paths=source_relative,module_name=fixture.custom"),
		ProtoFile: []*descriptorpb.FileDescriptorProto{
			fixtureFileDescriptor(),
		},
	}
	plugin, err := protogen.Options{ParamFunc: func(_, _ string) error { return nil }}.New(req)
	require.NoError(t, err)
	opts, err := generator.ParseParameter(req.GetParameter())
	require.NoError(t, err)
	require.NoError(t, generator.Generate(plugin, opts))
	resp := plugin.Response()
	require.Empty(t, resp.GetError())
	require.Len(t, resp.File, 1)

	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module example.com/fixture\n\ngo 1.26\n"), 0o644))
	outPath := filepath.Join(tmp, resp.File[0].GetName())
	require.NoError(t, os.MkdirAll(filepath.Dir(outPath), 0o755))
	require.NoError(t, os.WriteFile(outPath, []byte(resp.File[0].GetContent()), 0o644))

	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = tmp
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))
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
