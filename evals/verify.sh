#!/usr/bin/env bash
# Correctness check for the dogfood suite. Runs in the eval working directory,
# where the agent was asked to produce a `suite.yaml`. The agent output is piped
# to stdin (via the check_output verifier) but is not needed here — correctness
# is judged entirely by whether the produced suite passes `skival validate`,
# the exact self-validation step the skival skill instructs the agent to run.
set -euo pipefail

target="suite.yaml"

if [ ! -f "$target" ]; then
  echo "FAIL: agent did not produce ${target}"
  exit 1
fi

if ! command -v skival >/dev/null 2>&1; then
  echo "FAIL: skival is not on PATH (run 'make install' before the eval)"
  exit 1
fi

# `skival validate` exits non-zero and prints the structural error on failure.
skival validate "$target"
