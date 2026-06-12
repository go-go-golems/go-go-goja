package main

import (
	"github.com/go-go-golems/go-go-goja/cmd/protoc-gen-goja-builder/internal/generator"
	"google.golang.org/protobuf/compiler/protogen"
)

func main() {
	opts := protogen.Options{
		ParamFunc: func(_, _ string) error {
			return nil
		},
	}
	opts.Run(func(plugin *protogen.Plugin) error {
		parsed, err := generator.ParseParameter(plugin.Request.GetParameter())
		if err != nil {
			return err
		}
		return generator.Generate(plugin, parsed)
	})
}
