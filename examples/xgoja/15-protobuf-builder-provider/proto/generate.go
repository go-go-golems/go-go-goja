package taskpb

//go:generate protoc -I . -I /usr/include --go_out=. --go_opt=paths=source_relative --goja-builder_out=. --goja-builder_opt=paths=source_relative,module_name=examples.xgoja.protobuf.v1 task.proto
