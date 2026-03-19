#!/usr/bin/env bash
set -euo pipefail

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

if [ "$output" = "$expected" ]; then
  echo "PASS: output matches expected FizzBuzz sequence"
  exit 0
else
  echo "FAIL: output does not match"
  echo "--- expected ---"
  echo "$expected"
  echo "--- got ---"
  echo "$output"
  exit 1
fi
