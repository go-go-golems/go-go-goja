#!/usr/bin/env bash
# exp02-edge-case-evaluation.sh
# Purpose: Test edge cases in the rewrite and evaluation pipeline
# Usage:   bash exp02-edge-case-evaluation.sh [BASE_URL]
set -euo pipefail

BASE="${1:-http://localhost:3092}"
echo "=== EXP-02: Edge Case Evaluation ==="

# Create session
CREATE=$(curl -sf "$BASE/api/sessions" -X POST)
SESSION_ID=$(echo "$CREATE" | python3 -c "import sys,json; print(json.load(sys.stdin)['session']['id'])")
echo "Session ID: $SESSION_ID"

test_eval() {
  local label="$1"
  local source="$2"
  local expect_status="${3:-ok}"
  local expect_result="$4"
  
  echo ""
  echo "--- Test: $label ---"
  echo "Source: $source"
  
  RESP=$(curl -sf "$BASE/api/sessions/$SESSION_ID/evaluate" \
    -X POST -H 'Content-Type: application/json' \
    -d "{\"source\":$(echo "$source" | python3 -c 'import sys,json; print(json.dumps(sys.stdin.read()))')}")
  
  STATUS=$(echo "$RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['cell']['execution']['status'])" 2>/dev/null || echo "PARSE_ERROR")
  RESULT=$(echo "$RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['cell']['execution']['result'])" 2>/dev/null || echo "PARSE_ERROR")
  
  echo "Status: $STATUS, Result: $RESULT"
  
  if [ "$STATUS" != "$expect_status" ]; then
    echo "FAIL: Expected status=$expect_status, got $STATUS"
  elif [ -n "$expect_result" ] && [ "$RESULT" != "$expect_result" ]; then
    echo "FAIL: Expected result=$expect_result, got $RESULT"
  else
    echo "PASS"
  fi
}

# BUG HUNT 1: Re-declaring a const in a new cell
test_eval "const x in cell 1" "const x = 10; x" "ok" "10"
test_eval "const x re-declared in cell 2" "const x = 20; x" "ok" "20"

# BUG HUNT 2: let re-declaration
# (New session)
CREATE=$(curl -sf "$BASE/api/sessions" -X POST)
SESSION_ID=$(echo "$CREATE" | python3 -c "import sys,json; print(json.load(sys.stdin)['session']['id'])")
echo ""
echo "New Session ID for let tests: $SESSION_ID"
test_eval "let y in cell 1" "let y = 'hello'; y" "ok" "hello"
test_eval "let y reassigned in cell 2" "y = 'world'; y" "ok" "world"

# BUG HUNT 3: Function declarations
CREATE=$(curl -sf "$BASE/api/sessions" -X POST)
SESSION_ID=$(echo "$CREATE" | python3 -c "import sys,json; print(json.load(sys.stdin)['session']['id'])")
echo ""
echo "New Session ID for function tests: $SESSION_ID"
test_eval "function declaration" "function add(a, b) { return a + b; }; add(2, 3)" "ok" "5"
test_eval "use function from previous cell" "add(10, 20)" "ok" "30"

# BUG HUNT 4: Class declarations
CREATE=$(curl -sf "$BASE/api/sessions" -X POST)
SESSION_ID=$(echo "$CREATE" | python3 -c "import sys,json; print(json.load(sys.stdin)['session']['id'])")
echo ""
echo "New Session ID for class tests: $SESSION_ID"
test_eval "class declaration" "class Foo { constructor(x) { this.x = x; } getValue() { return this.x; } }; new Foo(42).getValue()" "ok" "42"
test_eval "use class from previous cell" "new Foo(99).getValue()" "ok" "99"

# BUG HUNT 5: Syntax error recovery
CREATE=$(curl -sf "$BASE/api/sessions" -X POST)
SESSION_ID=$(echo "$CREATE" | python3 -c "import sys,json; print(json.load(sys.stdin)['session']['id'])")
echo ""
echo "New Session ID for syntax error tests: $SESSION_ID"
test_eval "valid code before error" "const a = 1; a" "ok" "1"
test_eval "syntax error" "const = broken{" "parse-error" ""
test_eval "session still usable after error" "a + 10" "ok" "11"

# BUG HUNT 6: Empty source
test_eval "empty source" "" "ok" "undefined"

# BUG HUNT 7: Only whitespace
test_eval "whitespace only" "   " "ok" "undefined"

# BUG HUNT 8: Deeply nested async
CREATE=$(curl -sf "$BASE/api/sessions" -X POST)
SESSION_ID=$(echo "$CREATE" | python3 -c "import sys,json; print(json.load(sys.stdin)['session']['id'])")
echo ""
echo "New Session ID for async tests: $SESSION_ID"
test_eval "await expression" "await Promise.resolve(42)" "ok" "42"
test_eval "await after await" "await Promise.resolve(100)" "ok" "100"

# BUG HUNT 9: console capture
CREATE=$(curl -sf "$BASE/api/sessions" -X POST)
SESSION_ID=$(echo "$CREATE" | python3 -c "import sys,json; print(json.load(sys.stdin)['session']['id'])")
echo ""
echo "New Session ID for console tests: $SESSION_ID"
echo ""
echo "--- Test: console.log capture ---"
RESP=$(curl -sf "$BASE/api/sessions/$SESSION_ID/evaluate" \
  -X POST -H 'Content-Type: application/json' \
  -d '{"source":"console.log(\"hello\", 42); console.warn(\"danger\"); 1"}')
CONSOLE=$(echo "$RESP" | python3 -c "
import sys,json
cell = json.load(sys.stdin)['cell']
events = cell['execution']['console']
print(f'Console events: {len(events)}')
for e in events:
    print(f'  {e[\"kind\"]}: {e[\"message\"]!r}')
" 2>/dev/null)
echo "$CONSOLE"

# BUG HUNT 10: Var declaration (should it be global?)
CREATE=$(curl -sf "$BASE/api/sessions" -X POST)
SESSION_ID=$(echo "$CREATE" | python3 -c "import sys,json; print(json.load(sys.stdin)['session']['id'])")
echo ""
echo "New Session ID for var tests: $SESSION_ID"
test_eval "var declaration" "var z = 999; z" "ok" "999"
test_eval "var usage in next cell" "z + 1" "ok" "1000"

echo ""
echo "=== EXP-02 Complete ==="
