# Shell Scripting Best Practices

When writing shell scripts, follow these rules:

1. Always start with `#!/usr/bin/env bash` and `set -euo pipefail`
2. Use functions to organize logic — don't put everything at the top level
3. Quote all variable expansions: `"$var"` not `$var`
4. Use `(( ))` for arithmetic comparisons, not `[ ]` with `-eq`
5. Prefer `printf` over `echo` for portable output
