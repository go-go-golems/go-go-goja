#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../../../.." && pwd)"
OUT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/out"
BOBATEA_DIR="$ROOT_DIR/bobatea"
GOJA_DIR="$ROOT_DIR/go-go-goja"

mkdir -p "$OUT_DIR"

PKG_SUMMARY="$OUT_DIR/bobatea_pkg_summary.tsv"
PKG_IMPORTS="$OUT_DIR/bobatea_pkg_imports.tsv"
GOJA_BOBATEA_IMPORTS="$OUT_DIR/go_go_goja_imports_from_bobatea.txt"
GOJA_JS_SURFACE="$OUT_DIR/go_go_goja_js_surface.txt"
BOBATEA_JS_SURFACE="$OUT_DIR/bobatea_js_surface.txt"

echo -e "package\tfiles\tgoja\tjsparse\tgo_go_goja\tbubbletea\tbubbles\tjavascript_path\trepl_path" >"$PKG_SUMMARY"
echo -e "package\timport" >"$PKG_IMPORTS"

extract_imports() {
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

while IFS= read -r pkg; do
  rel_pkg="${pkg#$BOBATEA_DIR/}"
  file_count="$(rg --files "$pkg" -g '*.go' | wc -l | tr -d ' ')"
  imports="$(extract_imports "$pkg" || true)"

  has_goja=0
  has_jsparse=0
  has_go_go_goja=0
  has_bubbletea=0
  has_bubbles=0

  if grep -q 'github.com/dop251/goja' <<<"$imports"; then has_goja=1; fi
  if grep -q 'pkg/jsparse' <<<"$imports"; then has_jsparse=1; fi
  if grep -q 'github.com/go-go-golems/go-go-goja' <<<"$imports"; then has_go_go_goja=1; fi
  if grep -q 'github.com/charmbracelet/bubbletea' <<<"$imports"; then has_bubbletea=1; fi
  if grep -q 'github.com/charmbracelet/bubbles' <<<"$imports"; then has_bubbles=1; fi

  javascript_path=0
  repl_path=0
  [[ "$rel_pkg" == *"/javascript"* ]] && javascript_path=1
  [[ "$rel_pkg" == "pkg/repl"* ]] && repl_path=1

  echo -e "${rel_pkg}\t${file_count}\t${has_goja}\t${has_jsparse}\t${has_go_go_goja}\t${has_bubbletea}\t${has_bubbles}\t${javascript_path}\t${repl_path}" >>"$PKG_SUMMARY"

  if [[ -n "$imports" ]]; then
    while IFS= read -r imp; do
      echo -e "${rel_pkg}\t${imp}" >>"$PKG_IMPORTS"
    done <<<"$imports"
  fi
done < <(find "$BOBATEA_DIR/pkg" -type d | sort)

rg -n 'github.com/go-go-golems/bobatea/pkg' "$GOJA_DIR" -g '*.go' >"$GOJA_BOBATEA_IMPORTS" || true
rg -n 'goja|jsparse|javascript|repl|smalltalk-inspector|inspectorapi' "$GOJA_DIR" -g '*.go' >"$GOJA_JS_SURFACE" || true
rg -n 'goja|jsparse|javascript|repl|require\(' "$BOBATEA_DIR" -g '*.go' >"$BOBATEA_JS_SURFACE" || true

echo "wrote:"
echo "  $PKG_SUMMARY"
echo "  $PKG_IMPORTS"
echo "  $GOJA_BOBATEA_IMPORTS"
echo "  $GOJA_JS_SURFACE"
echo "  $BOBATEA_JS_SURFACE"
