#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
cd "$repo_root"

tag="replapi_hardening"

cases=()

failed=0
for case_spec in "${cases[@]}"; do
  IFS='|' read -r package test_name expected <<<"$case_spec"
  output="$(mktemp)"
  if go test -tags "$tag" "./$package" -run "^${test_name}$" -count=1 -v >"$output" 2>&1; then
    echo "UNEXPECTED PASS: $package/$test_name"
    failed=1
  elif ! grep -Fq "$expected" "$output"; then
    echo "WRONG FAILURE: $package/$test_name"
    cat "$output"
    failed=1
  else
    echo "EXPECTED RED: $package/$test_name"
  fi
  rm -f "$output"
done

if (( failed != 0 )); then
  echo "Phase 0 red baseline did not match the expected failure set." >&2
  exit 1
fi

failure_label="failures"
if (( ${#cases[@]} == 1 )); then
  failure_label="failure"
fi
echo "Phase 0 red baseline confirmed: ${#cases[@]} expected ${failure_label}."
