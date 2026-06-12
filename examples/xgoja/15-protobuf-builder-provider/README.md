# xgoja protobuf builder provider example

This example exercises `protoc-gen-goja-builder` as an xgoja provider module from proto schema to usable Goja application code.

It contains:

- `proto/task.proto` — example protobuf schema with enum, repeated, map, nested message, `Timestamp`, and `Struct` fields.
- `proto/task.pb.go` — generated Go protobuf types.
- `proto/task_goja.pb.go` — generated Goja fluent builder companion.
- `provider/provider.go` — xgoja provider registration for the generated protobuf builder module.
- `provider/provider_test.go` — compiled end-to-end test that registers the provider module, requires it from JavaScript, builds protobuf messages, and extracts concrete Go protobuf values with `protogoja.MessageFromValue`.
- `scripts/build-task.js` — representative JavaScript consumer script.

Regenerate protobuf files from the repository root with:

```bash
go generate ./examples/xgoja/15-protobuf-builder-provider/proto
```

Run the compiled example test with:

```bash
go test ./examples/xgoja/15-protobuf-builder-provider/... -count=1
```
