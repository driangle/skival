#!/bin/bash
set -euo pipefail

# Verify fizzbuzz.sh exists and produces correct output
if [[ ! -f fizzbuzz.sh ]]; then
  echo "FAIL: fizzbuzz.sh not found"
  exit 1
fi

output=$(bash fizzbuzz.sh)
expected="1
2
Fizz
4
Buzz
Fizz
7
8
Fizz
Buzz
11
Fizz
13
14
FizzBuzz
16
17
Fizz
19
Buzz"

if [[ "$output" == "$expected" ]]; then
  echo "PASS: output matches expected"
  exit 0
else
  echo "FAIL: output mismatch"
  echo "--- expected ---"
  echo "$expected"
  echo "--- got ---"
  echo "$output"
  exit 1
fi
