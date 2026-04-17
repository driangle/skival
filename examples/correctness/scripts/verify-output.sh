#!/bin/bash
# Verify that output.txt contains exactly "hello world"
expected="hello world"
actual=$(cat output.txt 2>/dev/null)
if [ "$actual" = "$expected" ]; then
  echo "PASS"
  exit 0
else
  echo "FAIL: expected '$expected', got '$actual'"
  exit 1
fi
