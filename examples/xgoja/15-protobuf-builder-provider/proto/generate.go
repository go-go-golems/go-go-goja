package taskpb

//go:generate protoc -I . --go_out=. --go_opt=paths=source_relative --goja-builder_out=. --goja-builder_opt=paths=source_relative,module_name=examples.xgoja.protobuf.v1 task.proto
