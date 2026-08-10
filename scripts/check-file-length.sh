#!/usr/bin/env bash
#
# check-file-length.sh — enforce a maximum line count per Go source file.
#
# Non-test files (*.go) may be at most $MAX_NONTEST lines; test files
# (*_test.go) may be at most $MAX_TEST lines. Prints one line per violation
# ("path: N > LIMIT") and exits non-zero if any file is over budget.
#
# Usage:
#   check-file-length.sh [root]      # root defaults to the current directory
#
# Env overrides (used by tests and for tuning the limits):
#   MAX_NONTEST   line budget for non-test .go files (default 300)
#   MAX_TEST      line budget for *_test.go files    (default 500)
set -euo pipefail

MAX_NONTEST="${MAX_NONTEST:-300}"
MAX_TEST="${MAX_TEST:-500}"
ROOT="${1:-.}"

status=0

while IFS= read -r file; do
	lines=$(wc -l <"$file" | tr -d '[:space:]')
	case "$file" in
	*_test.go) limit="$MAX_TEST" ;;
	*) limit="$MAX_NONTEST" ;;
	esac
	if [ "$lines" -gt "$limit" ]; then
		echo "$file: $lines > $limit"
		status=1
	fi
done < <(find "$ROOT" -type f -name '*.go' -not -path '*/vendor/*' | sort)

if [ "$status" -ne 0 ]; then
	echo "" >&2
	echo "file-length limit exceeded (non-test: $MAX_NONTEST, test: $MAX_TEST lines)." >&2
	echo "Split the file into cohesive smaller files, or add a documented exemption." >&2
fi

exit "$status"
