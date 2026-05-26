#!/usr/bin/env bash
# render_to_remarkable.sh — Full pipeline: semantic YAML → HTML → PDF → reMarkable
#
# Usage:
#   ./render_to_remarkable.sh <path-to-file.semantic.yaml>
#
# Produces two PDFs in scripts/output/:
#   - <basename>-full.pdf     (all content)
#   - <basename>-quick.pdf    (importance >= high only)
#
# Then uploads both to reMarkable.

set -euo pipefail

INPUT="$1"

if [ ! -f "$INPUT" ]; then
  echo "ERROR: File not found: $INPUT"
  exit 1
fi

BASENAME="$(basename "$INPUT" .semantic.yaml)"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
OUTPUT_DIR="${SCRIPT_DIR}/output"
mkdir -p "$OUTPUT_DIR"

# ── Detect Chromium-based browser ──

BROWSER=""
for cmd in chromium-browser google-chrome chrome chromium; do
  if command -v "$cmd" &>/dev/null; then
    BROWSER="$cmd"
    break
  fi
done

if [ -z "$BROWSER" ]; then
  echo "ERROR: No Chromium-based browser found. Install chromium-browser."
  echo "  sudo apt install chromium-browser"
  exit 1
fi

echo "Using browser: $BROWSER"
echo "Input: $INPUT"
echo "Output dir: $OUTPUT_DIR"
echo ""

# ── Step 1: Render full guide HTML ──

echo "=== Rendering full guide HTML ==="
python3 "${SCRIPT_DIR}/semantic_render.py" \
  --input "$INPUT" \
  --output "${OUTPUT_DIR}/${BASENAME}-full.html"

# ── Step 2: Render quick reference HTML (importance >= high) ──

echo "=== Rendering quick reference HTML (importance >= high) ==="
python3 "${SCRIPT_DIR}/semantic_render.py" \
  --input "$INPUT" \
  --output "${OUTPUT_DIR}/${BASENAME}-quick.html" \
  --importance-threshold high

# ── Step 3: Convert HTML → PDF ──

echo "=== Converting full guide to PDF ==="
"$BROWSER" --headless --disable-gpu --no-sandbox \
  --print-to-pdf="${OUTPUT_DIR}/${BASENAME}-full.pdf" \
  --no-pdf-header-footer \
  "file://${OUTPUT_DIR}/${BASENAME}-full.html" 2>/dev/null

echo "=== Converting quick reference to PDF ==="
"$BROWSER" --headless --disable-gpu --no-sandbox \
  --print-to-pdf="${OUTPUT_DIR}/${BASENAME}-quick.pdf" \
  --no-pdf-header-footer \
  "file://${OUTPUT_DIR}/${BASENAME}-quick.html" 2>/dev/null

echo ""
echo "=== PDFs generated ==="
ls -lh "${OUTPUT_DIR}/${BASENAME}"-*.pdf

# ── Step 4: Upload to reMarkable ──

REMOTE_DIR="/ai/2026/05/26/LAYOUT-001"

echo ""
echo "=== Uploading to reMarkable ==="
remarquee cloud put "${OUTPUT_DIR}/${BASENAME}-full.pdf" \
  "$REMOTE_DIR" --non-interactive

remarquee cloud put "${OUTPUT_DIR}/${BASENAME}-quick.pdf" \
  "$REMOTE_DIR" --non-interactive

echo ""
echo "=== Verifying upload ==="
remarquee cloud ls "$REMOTE_DIR" --long --non-interactive

echo ""
echo "=== Done ==="
echo "Full guide:   ${OUTPUT_DIR}/${BASENAME}-full.pdf"
echo "Quick ref:    ${OUTPUT_DIR}/${BASENAME}-quick.pdf"
echo "ReMarkable:   $REMOTE_DIR"
