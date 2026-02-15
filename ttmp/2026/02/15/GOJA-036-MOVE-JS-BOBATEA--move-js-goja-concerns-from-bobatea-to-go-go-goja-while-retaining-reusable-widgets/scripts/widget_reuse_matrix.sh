#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../../../.." && pwd)"
OUT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/out"
BOBATEA_DIR="$ROOT_DIR/bobatea"
GOJA_DIR="$ROOT_DIR/go-go-goja"

mkdir -p "$OUT_DIR"

MATRIX="$OUT_DIR/widget_reuse_matrix.tsv"
GOJA_IMPORTS="$OUT_DIR/go_go_goja_bobatea_widget_imports.tsv"

echo -e "package\tgo_files\ttest_files\tloc\timports_goja\timports_jsparse\timports_go_go_goja\tused_in_go_go_goja\tusage_sites" >"$MATRIX"
echo -e "import_path\tsite" >"$GOJA_IMPORTS"

imports_of_pkg() {
  local target_dir="$1"
  awk '
    BEGIN { in_block=0 }
    /^import[[:space:]]*\(/ { in_block=1; next }
    in_block && /^\)/ { in_block=0; next }
    /^import[[:space:]]+"/ {
      line=$0
      gsub(/^import[[:space:]]+/, "", line)
      gsub(/"/, "", line)
      print line
      next
    }
    in_block {
      if (match($0, /"[^"]+"/)) {
        imp=substr($0, RSTART+1, RLENGTH-2)
        print imp
      }
    }
  ' "$target_dir"/*.go 2>/dev/null | sort -u
}

while IFS= read -r dir; do
  rel="${dir#$BOBATEA_DIR/}"
  go_files="$(find "$dir" -maxdepth 1 -type f -name '*.go' | wc -l | tr -d ' ')"
  test_files="$(find "$dir" -maxdepth 1 -type f -name '*_test.go' | wc -l | tr -d ' ')"
  loc="$(find "$dir" -maxdepth 1 -type f -name '*.go' -print0 | xargs -0r cat | wc -l | tr -d ' ')"
  imports="$(imports_of_pkg "$dir" || true)"

  has_goja=0
  has_jsparse=0
  has_go_go_goja=0
  if grep -q 'github.com/dop251/goja' <<<"$imports"; then has_goja=1; fi
  if grep -q 'pkg/jsparse' <<<"$imports"; then has_jsparse=1; fi
  if grep -q 'github.com/go-go-golems/go-go-goja' <<<"$imports"; then has_go_go_goja=1; fi

  import_path="github.com/go-go-golems/bobatea/${rel}"
  usages="$(rg -n --no-heading "\"$import_path\"" "$GOJA_DIR" -g '*.go' || true)"
  usage_count=0
  usage_sites="-"
  if [[ -n "$usages" ]]; then
    usage_count="$(printf "%s\n" "$usages" | wc -l | tr -d ' ')"
    usage_sites="$(printf "%s\n" "$usages" | cut -d: -f1-2 | paste -sd';' -)"
    while IFS= read -r line; do
      [[ -z "$line" ]] && continue
      echo -e "${import_path}\t${line}" >>"$GOJA_IMPORTS"
    done <<<"$usages"
  fi

  echo -e "${rel}\t${go_files}\t${test_files}\t${loc}\t${has_goja}\t${has_jsparse}\t${has_go_go_goja}\t${usage_count}\t${usage_sites}" >>"$MATRIX"
done < <(find "$BOBATEA_DIR/pkg" -type d | sort)

echo "wrote:"
echo "  $MATRIX"
echo "  $GOJA_IMPORTS"
