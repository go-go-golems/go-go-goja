#!/usr/bin/env bash
# exp01-api-smoke-test.sh
# Purpose: Basic API lifecycle smoke test against live server
# Usage:   bash exp01-api-smoke-test.sh [BASE_URL]
set -euo pipefail

BASE="${1:-http://localhost:3092}"
echo "=== EXP-01: API Smoke Test against $BASE ==="

# 1. Create a session
echo ""
echo "--- Step 1: Create session ---"
CREATE=$(curl -sf "$BASE/api/sessions" -X POST)
echo "$CREATE" | python3 -m json.tool 2>/dev/null || echo "$CREATE"
SESSION_ID=$(echo "$CREATE" | python3 -c "import sys,json; print(json.load(sys.stdin)['session']['id'])")
echo "Session ID: $SESSION_ID"

# 2. Evaluate simple expression
echo ""
echo "--- Step 2: Evaluate 'const x = 1; x' ---"
EVAL1=$(curl -sf "$BASE/api/sessions/$SESSION_ID/evaluate" -X POST -H 'Content-Type: application/json' -d '{"source":"const x = 1; x"}')
echo "$EVAL1" | python3 -m json.tool 2>/dev/null || echo "$EVAL1"
EVAL1_STATUS=$(echo "$EVAL1" | python3 -c "import sys,json; print(json.load(sys.stdin)['cell']['execution']['status'])")
EVAL1_RESULT=$(echo "$EVAL1" | python3 -c "import sys,json; print(json.load(sys.stdin)['cell']['execution']['result'])")
echo "Status: $EVAL1_STATUS, Result: $EVAL1_RESULT"

# 3. Evaluate expression referencing previous binding
echo ""
echo "--- Step 3: Evaluate 'x + 1' (reference previous binding) ---"
EVAL2=$(curl -sf "$BASE/api/sessions/$SESSION_ID/evaluate" -X POST -H 'Content-Type: application/json' -d '{"source":"x + 1"}')
echo "$EVAL2" | python3 -m json.tool 2>/dev/null || echo "$EVAL2"
EVAL2_STATUS=$(echo "$EVAL2" | python3 -c "import sys,json; print(json.load(sys.stdin)['cell']['execution']['status'])")
EVAL2_RESULT=$(echo "$EVAL2" | python3 -c "import sys,json; print(json.load(sys.stdin)['cell']['execution']['result'])")
echo "Status: $EVAL2_STATUS, Result: $EVAL2_RESULT"

if [ "$EVAL2_STATUS" = "ok" ] && [ "$EVAL2_RESULT" = "2" ]; then
  echo "PASS: Binding persistence across cells works"
else
  echo "FAIL: Expected status=ok result=2, got status=$EVAL2_STATUS result=$EVAL2_RESULT"
fi

# 4. Snapshot session
echo ""
echo "--- Step 4: Snapshot session ---"
SNAP=$(curl -sf "$BASE/api/sessions/$SESSION_ID")
echo "$SNAP" | python3 -c "
import sys, json
d = json.load(sys.stdin)
s = d['session']
print(f'BindingCount: {s[\"bindingCount\"]}')
print(f'CellCount: {s[\"cellCount\"]}')
print(f'Profile: {s[\"profile\"]}')
for b in s.get('bindings', []):
    print(f'  Binding: {b[\"name\"]} ({b[\"kind\"]}) = {b[\"runtime\"][\"preview\"]}')
" 2>/dev/null || echo "$SNAP"

# 5. List sessions
echo ""
echo "--- Step 5: List sessions count ---"
LIST=$(curl -sf "$BASE/api/sessions")
COUNT=$(echo "$LIST" | python3 -c "import sys,json; print(len(json.load(sys.stdin)['sessions']))")
echo "Total sessions: $COUNT"

# 6. History
echo ""
echo "--- Step 6: Get history ---"
HIST=$(curl -sf "$BASE/api/sessions/$SESSION_ID/history")
echo "$HIST" | python3 -c "
import sys, json
d = json.load(sys.stdin)
for h in d['history']:
    print(f'  Cell {h[\"cellId\"]}: [{h[\"status\"]}] {h[\"sourcePreview\"]!r} -> {h[\"resultPreview\"]!r}')
" 2>/dev/null || echo "$HIST"

# 7. Bindings endpoint
echo ""
echo "--- Step 7: Bindings ---"
BINDS=$(curl -sf "$BASE/api/sessions/$SESSION_ID/bindings")
echo "$BINDS" | python3 -m json.tool 2>/dev/null || echo "$BINDS"

echo ""
echo "=== EXP-01 Complete ==="
