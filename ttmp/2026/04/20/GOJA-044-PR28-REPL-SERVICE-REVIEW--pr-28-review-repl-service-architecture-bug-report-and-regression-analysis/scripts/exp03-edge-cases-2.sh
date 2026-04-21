#!/usr/bin/env bash
# exp03-edge-cases-2.sh
# Purpose: More edge case tests (skip empty/whitespace which crash)
set -euo pipefail

BASE="${1:-http://localhost:3092}"
echo "=== EXP-03: More Edge Cases ==="

create_session() {
  curl -sf "$BASE/api/sessions" -X POST | python3 -c "import sys,json; print(json.load(sys.stdin)['session']['id'])"
}

eval_source() {
  local sid="$1" source="$2"
  curl -sf "$BASE/api/sessions/$sid/evaluate" \
    -X POST -H 'Content-Type: application/json' \
    -d "{\"source\":$(python3 -c "import sys,json; print(json.dumps('$source'))")}"
}

eval_raw() {
  local sid="$1" source="$2"
  curl -sf "$BASE/api/sessions/$sid/evaluate" \
    -X POST -H 'Content-Type: application/json' \
    -d "{\"source\":$(python3 -c "import json,sys; print(json.dumps(sys.stdin.read()))" <<< "$source")}"
}

# BUG HUNT 11: Delete and recreate session with same ID
echo ""
echo "--- Test: Delete and recreate session ---"
SID=$(create_session)
echo "Created: $SID"
eval_raw "$SID" "const d = 100; d"
echo "Evaluated, deleting..."
curl -sf "$BASE/api/sessions/$SID" -X DELETE | python3 -m json.tool
echo "Trying to restore deleted session..."
curl -sf "$BASE/api/sessions/$SID/restore" -X POST 2>&1 || echo "RESTORE FAILED (expected if hard-deleted)"
echo "Trying to get snapshot of deleted session..."
curl -sf "$BASE/api/sessions/$SID" 2>&1 || echo "SNAPSHOT FAILED (expected if deleted)"

# BUG HUNT 12: Race condition - evaluate same session concurrently
echo ""
echo "--- Test: Concurrent evaluation ---"
SID=$(create_session)
echo "Session: $SID"
eval_raw "$SID" "const base = 1; base" > /dev/null
# Fire 5 concurrent evals
for i in $(seq 1 5); do
  eval_raw "$SID" "base + $i" > /tmp/exp03-concurrent-$i.txt 2>&1 &
done
wait
echo "Concurrent results:"
for i in $(seq 1 5); do
  STATUS=$(python3 -c "import json; d=json.load(open('/tmp/exp03-concurrent-$i.txt')); print(d['cell']['execution']['status'])" 2>/dev/null || echo "PARSE_ERROR")
  RESULT=$(python3 -c "import json; d=json.load(open('/tmp/exp03-concurrent-$i.txt')); print(d['cell']['execution']['result'])" 2>/dev/null || echo "PARSE_ERROR")
  echo "  Request $i: status=$STATUS result=$RESULT"
done

# BUG HUNT 13: Very large expression (response size)
echo ""
echo "--- Test: Large object literal ---"
SID=$(create_session)
PAYLOAD=$(python3 -c "props = ','.join([f'key{i}: {i}' for i in range(200)]); print(f'const obj = {{{props}}}; Object.keys(obj).length')")
eval_raw "$SID" "$PAYLOAD" | python3 -c "
import sys,json
d = json.load(sys.stdin)
print(f'Status: {d[\"cell\"][\"execution\"][\"status\"]}')
print(f'Result: {d[\"cell\"][\"execution\"][\"result\"]}')
print(f'Binding count: {d[\"session\"][\"bindingCount\"]}')
print(f'Binding own props: {len(d[\"session\"][\"bindings\"][0][\"runtime\"].get(\"ownProperties\", []))}')
" 2>/dev/null || echo "FAILED"

# BUG HUNT 14: Export and restore cycle
echo ""
echo "--- Test: Export/Restore cycle ---"
SID=$(create_session)
eval_raw "$SID" 'const ex = 42; ex' > /dev/null
eval_raw "$SID" 'const ey = ex * 2; ey' > /dev/null
echo "Exporting session $SID..."
EXPORT=$(curl -sf "$BASE/api/sessions/$SID/export")
EVAL_COUNT=$(echo "$EXPORT" | python3 -c "import sys,json; print(len(json.load(sys.stdin)['evaluations']))")
echo "Export has $EVAL_COUNT evaluations"

# BUG HUNT 15: Null/undefined edge cases
echo ""
echo "--- Test: Null/undefined results ---"
SID=$(create_session)
eval_raw "$SID" "null" | python3 -c "
import sys,json
d = json.load(sys.stdin)
print(f'null -> status={d[\"cell\"][\"execution\"][\"status\"]} result={d[\"cell\"][\"execution\"][\"result\"]!r}')
" 2>/dev/null
eval_raw "$SID" "undefined" | python3 -c "
import sys,json
d = json.load(sys.stdin)
print(f'undefined -> status={d[\"cell\"][\"execution\"][\"status\"]} result={d[\"cell\"][\"execution\"][\"result\"]!r}')
" 2>/dev/null

# BUG HUNT 16: Try re-declaring with let
echo ""
echo "--- Test: let re-declaration ---"
SID=$(create_session)
eval_raw "$SID" "let lx = 1; lx" | python3 -c "import sys,json; d=json.load(sys.stdin); print(f'let lx=1: {d[\"cell\"][\"execution\"]}')" 2>/dev/null
eval_raw "$SID" "let lx = 2; lx" | python3 -c "import sys,json; d=json.load(sys.stdin); print(f'let lx=2 (re-declare): status={d[\"cell\"][\"execution\"][\"status\"]} result={d[\"cell\"][\"execution\"][\"result\"]}')" 2>/dev/null || echo "re-declaration failed (crash or error)"

# BUG HUNT 17: Special characters in source
echo ""
echo "--- Test: Template literals ---"
SID=$(create_session)
eval_raw "$SID" 'const name = "world"; `hello ${name}`' | python3 -c "
import sys,json
d = json.load(sys.stdin)
print(f'template literal -> status={d[\"cell\"][\"execution\"][\"status\"]} result={d[\"cell\"][\"execution\"][\"result\"]!r}')
" 2>/dev/null

# BUG HUNT 18: Arrow functions
echo ""
echo "--- Test: Arrow functions ---"
SID=$(create_session)
eval_raw "$SID" 'const double = x => x * 2; double(21)' | python3 -c "
import sys,json
d = json.load(sys.stdin)
print(f'arrow fn -> status={d[\"cell\"][\"execution\"][\"status\"]} result={d[\"cell\"][\"execution\"][\"result\"]}')
print(f'binding kind: {d[\"session\"][\"bindings\"][0][\"kind\"]}')
print(f'binding valueKind: {d[\"session\"][\"bindings\"][0][\"runtime\"][\"valueKind\"]}')
" 2>/dev/null

echo ""
echo "=== EXP-03 Complete ==="
