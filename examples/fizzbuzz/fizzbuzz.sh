#!/usr/bin/env bash
set -euo pipefail

for ((i = 1; i <= 20; i++)); do
  if (( i % 15 == 0 )); then
    printf "FizzBuzz\n"
  elif (( i % 3 == 0 )); then
    printf "Fizz\n"
  elif (( i % 5 == 0 )); then
    printf "Buzz\n"
  else
    printf "%d\n" "$i"
  fi
done
