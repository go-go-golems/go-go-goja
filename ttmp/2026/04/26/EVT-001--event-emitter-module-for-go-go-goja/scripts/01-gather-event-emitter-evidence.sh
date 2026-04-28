#!/usr/bin/env bash
set -euo pipefail

# Evidence helper for EVT-001. Run from the go-go-goja repository root.
# It captures the file/line anchors that matter for the event-emitter design.

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
cd "$repo_root"

echo "# go-go-goja event-emitter evidence"
echo

echo "## Repository runtime/module architecture"
nl -ba README.md | sed -n '1,130p;150,185p;227,288p'
nl -ba engine/factory.go | sed -n '156,246p'
nl -ba engine/runtime.go | sed -n '1,36p;69,115p'
nl -ba engine/module_specs.go | sed -n '1,40p;64,85p;96,180p'
nl -ba engine/runtime_modules.go | sed -n '1,50p'
nl -ba modules/common.go | sed -n '1,120p'
nl -ba modules/timer/timer.go | sed -n '1,70p'
nl -ba modules/fs/fs_async.go | sed -n '1,80p'
nl -ba pkg/runtimeowner/types.go | sed -n '1,35p'
nl -ba pkg/runtimeowner/runner.go | sed -n '62,160p'
nl -ba pkg/runtimebridge/runtimebridge.go | sed -n '1,45p'

echo

echo "## External dependency contracts"
if [[ -f "../goja_nodejs/eventloop/eventloop.go" ]]; then
  nl -ba "../goja_nodejs/eventloop/eventloop.go" | sed -n '314,321p'
else
  nl -ba "$(go env GOPATH)/pkg/mod/github.com/dop251/goja_nodejs@v0.0.0-20250409162600-f7acab6894b0/eventloop/eventloop.go" | sed -n '314,321p'
fi
nl -ba "$(go env GOPATH)/pkg/mod/github.com/!three!dots!labs/watermill@v1.5.1/message/pubsub.go" | sed -n '25,39p'
nl -ba "$(go env GOPATH)/pkg/mod/github.com/!three!dots!labs/watermill@v1.5.1/message/message.go" | sed -n '96,147p'
nl -ba "$(go env GOPATH)/pkg/mod/github.com/fsnotify/fsnotify@v1.9.0/fsnotify.go" | sed -n '100,144p;278,300p'
