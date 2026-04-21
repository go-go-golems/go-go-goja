#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/../../../../../.." && pwd)"
OUT_DIR="${1:-/tmp/goja-035}"
mkdir -p "$OUT_DIR"

run_inspector_repro() {
  local session="goja035_inspector"
  tmux kill-session -t "$session" 2>/dev/null || true
  tmux new-session -d -x 96 -y 24 -s "$session" "cd $REPO_ROOT && go run ./cmd/inspector /tmp/long-tree.js"
  sleep 1.2
  tmux capture-pane -t "$session":0.0 -p > "$OUT_DIR/inspector-narrow-initial.txt"
  tmux send-keys -t "$session":0.0 Tab
  for _ in $(seq 1 15); do tmux send-keys -t "$session":0.0 j; done
  sleep 0.3
  tmux capture-pane -t "$session":0.0 -p > "$OUT_DIR/inspector-narrow-scroll.txt"
  tmux send-keys -t "$session":0.0 q
  sleep 0.2
  tmux kill-session -t "$session" 2>/dev/null || true
}

run_smalltalk_repro() {
  local session="goja035_smalltalk"
  tmux kill-session -t "$session" 2>/dev/null || true
  tmux new-session -d -x 120 -y 34 -s "$session" "cd $REPO_ROOT && go run ./cmd/smalltalk-inspector testdata/inspector-test.js"
  sleep 1.2

  for _ in 1 2 3; do tmux send-keys -t "$session":0.0 Tab; done
  tmux send-keys -t "$session":0.0 "const zzzReplFn = function zzzReplFn(value) { return value + 42; }"
  tmux send-keys -t "$session":0.0 Enter
  sleep 0.5

  tmux send-keys -t "$session":0.0 Tab
  sleep 0.2
  tmux send-keys -t "$session":0.0 G
  sleep 0.3
  tmux capture-pane -t "$session":0.0 -p > "$OUT_DIR/smalltalk-globals-bottom.txt"

  tmux send-keys -t "$session":0.0 Enter
  sleep 0.2
  tmux capture-pane -t "$session":0.0 -p > "$OUT_DIR/smalltalk-after-enter-zzz.txt"

  tmux send-keys -t "$session":0.0 q
  sleep 0.2
  tmux kill-session -t "$session" 2>/dev/null || true
}

cat > /tmp/long-tree.js <<'JS'
const veryLongTopLevelBindingNameThatKeepsGoingAndGoingAndGoing = {
  deeplyNestedPropertyWithVerboseNameOne: {
    deeplyNestedPropertyWithVerboseNameTwo: {
      deeplyNestedPropertyWithVerboseNameThree: {
        message: "this is a ridiculously long string literal that should force tree rows to clamp and never wrap across pane boundaries in the inspector tree view"
      }
    }
  }
};

function makeMonsterFunctionWithExtraLongNameAndManyParameters(parameterOneWithLongName, parameterTwoWithLongName, parameterThreeWithLongName) {
  const localBindingWithLongName = parameterOneWithLongName + parameterTwoWithLongName + parameterThreeWithLongName;
  return {
    localBindingWithLongName,
    nested: {
      desc: "another very long descriptor to trigger row overflow behavior in the tree pane and verify vertical scrolling remains stable"
    }
  };
}

for (let i = 0; i < 80; i++) {
  makeMonsterFunctionWithExtraLongNameAndManyParameters(i, i + 1, i + 2);
}
JS

run_inspector_repro
run_smalltalk_repro

echo "Wrote captures to: $OUT_DIR"
