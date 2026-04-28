#!/usr/bin/env bash
set -euo pipefail

# Validate that jsverbs can expose a Glazed objectFromFile flag and receive the
# parsed JSON/YAML object as a JavaScript object.
#
# Run from anywhere:
#   bash ttmp/2026/04/25/GOJA-053--add-fs-primitive-module-and-ensure-all-goja-nodejs-modules-are-require-able/scripts/validate-jsverbs-objectfromfile.sh

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd -- "${SCRIPT_DIR}/../../../../../.." && pwd)
TMP_DIR=$(mktemp -d)
trap 'rm -rf "${TMP_DIR}"' EXIT

mkdir -p "${TMP_DIR}/verbs"

cat > "${TMP_DIR}/verbs/objectfile.js" <<'JS'
function inspectConfig(config) {
  return [{
    type: typeof config,
    name: config && config.name,
    nested: config && config.nested && config.nested.value,
    listLength: config && config.items && config.items.length,
    firstItem: config && config.items && config.items[0]
  }];
}

__verb__("inspectConfig", {
  short: "Inspect JSON object loaded from a file",
  fields: {
    config: {
      type: "objectFromFile",
      help: "Path to JSON config file"
    }
  }
});
JS

cat > "${TMP_DIR}/config.json" <<'JSON'
{"name":"demo","nested":{"value":42},"items":["a","b","c"]}
JSON

cd "${REPO_ROOT}"

output=$(go run ./cmd/jsverbs-example \
  --dir "${TMP_DIR}/verbs" \
  objectfile inspect-config \
  --config "${TMP_DIR}/config.json")

printf '%s\n' "${output}"

if ! grep -q 'object' <<<"${output}"; then
  echo "expected output to show typeof config === object" >&2
  exit 1
fi
if ! grep -q 'demo' <<<"${output}"; then
  echo "expected output to include JSON field name=demo" >&2
  exit 1
fi
if ! grep -q '42' <<<"${output}"; then
  echo "expected output to include nested JSON value 42" >&2
  exit 1
fi
if ! grep -q '3' <<<"${output}"; then
  echo "expected output to include list length 3" >&2
  exit 1
fi
if ! grep -q 'a' <<<"${output}"; then
  echo "expected output to include first array item a" >&2
  exit 1
fi

echo "OK: objectFromFile JSON was parsed by Glazed and delivered to jsverbs as a JavaScript object."
