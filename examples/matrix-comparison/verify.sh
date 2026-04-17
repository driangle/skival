#!/usr/bin/env bash
set -euo pipefail

output=$(bash hello.sh)
expected="Hello, World!"

if [[ "$output" == *"$expected"* ]]; then
  exit 0
else
  echo "Expected output containing: $expected"
  echo "Got: $output"
  exit 1
fi
