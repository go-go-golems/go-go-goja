const pb = require("examples.xgoja.protobuf.v1")

const task = pb.Task.builder()
  .id("task-1")
  .title("Ship protobuf builders")
  .addTags("protobuf")
  .addTags("xgoja")
  .putLabels("component", "goja")
  .priority(pb.TaskPriority.TASK_PRIORITY_HIGH)
  .dueAt(new Date("2026-06-12T20:00:00Z"))
  .metadata({ owner: "agent", reviewed: true })
  .build()

exports.task = task
exports.envelope = pb.TaskEnvelope.builder()
  .task(task)
  .source("script")
  .build()
