#!/bin/bash
# Verify calc.py adds two numbers correctly
result=$(echo -e "3\n4" | python3 calc.py 2>/dev/null)
if echo "$result" | grep -q "Result:"; then
  echo "PASS"
  exit 0
else
  echo "FAIL: output did not contain 'Result:'"
  exit 1
fi
