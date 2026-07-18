#!/usr/bin/env bash
set -u

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../../.." && pwd)
work_dir=$(mktemp -d)
trap 'rm -rf "$work_dir"' EXIT

write_spec() {
  local path=$1
  local first_type=$2
  local first_id=$3
  local second_type=$4
  local second_id=$5
  cat >"$path" <<YAML
schema: xgoja/v2
name: artifact-order-reproduction
go:
  module: xgoja.generated/artifact-order-reproduction
  version: "1.26"
workspace:
  mode: off
artifacts:
  - id: $first_id
    type: $first_type
    output: $work_dir/$first_id
    package: xgojaruntime
  - id: $second_id
    type: $second_type
    output: $work_dir/$second_id
    package: xgojaruntime
YAML
}

run_case() {
  local label=$1
  local spec=$2
  shift 2
  echo "=== $label ==="
  set +e
  (cd "$repo_root" && go run ./cmd/xgoja "$@" -f "$spec") 2>&1
  local status=$?
  set -e
  echo "exit=$status"
  echo
}

set -e
binary_first="$work_dir/binary-first.yaml"
package_first="$work_dir/package-first.yaml"
support_first="$work_dir/support-first.yaml"
write_spec "$binary_first" binary binary runtime-package runtime-package
write_spec "$package_first" runtime-package runtime-package binary binary
write_spec "$support_first" embedded-assets assets binary binary

run_case "binary first: build succeeds" "$binary_first" build --dry-run --work-dir "$work_dir/build-binary-first"
run_case "binary first: generate fails" "$binary_first" generate --dry-run
run_case "runtime-package first: build fails" "$package_first" build --dry-run --work-dir "$work_dir/build-package-first"
run_case "runtime-package first: generate succeeds" "$package_first" generate --dry-run
run_case "support artifact first: build command selects binary" "$support_first" build --dry-run --work-dir "$work_dir/build-support-first"
echo "=== support artifact first: generated runtime metadata still uses support artifact ==="
grep -A3 '"target"' "$work_dir/build-support-first/xgoja.runtime.json"
echo
