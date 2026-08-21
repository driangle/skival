package verifier

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// ToolNotUsedVerifier fails a sample when the agent invoked any tool named in
// Forbidden. It is the hard-assertion backstop to tool-access enforcement and the
// pre-flight leak warning: even when a tool leaks past the advisory allow list,
// this turns its use into a failed run — e.g. proving a "no tools" variant
// genuinely used no tools.
type ToolNotUsedVerifier struct {
	// Forbidden lists tool names that must not be used. Names are matched by
	// base name, so "Bash" also catches a scoped "Bash(git:*)" invocation.
	Forbidden []string
}

func (v *ToolNotUsedVerifier) Verify(_ context.Context, input VerifyInput) VerifyResult {
	counts := CountToolUses(input.Conversation)
	if len(counts) == 0 || len(v.Forbidden) == 0 {
		return VerifyResult{Pass: true, Reason: "no forbidden tools were used"}
	}

	used := make(map[string]int, len(counts))
	for name, n := range counts {
		used[baseToolName(name)] += n
	}

	var violations []string
	for _, tool := range v.Forbidden {
		if n := used[baseToolName(tool)]; n > 0 {
			violations = append(violations, fmt.Sprintf("%s ×%d", tool, n))
		}
	}
	if len(violations) == 0 {
		return VerifyResult{Pass: true, Reason: "no forbidden tools were used"}
	}
	sort.Strings(violations)
	return VerifyResult{
		Pass:   false,
		Reason: fmt.Sprintf("forbidden tool(s) used: %s", strings.Join(violations, ", ")),
	}
}

// baseToolName strips a trailing scope suffix ("Bash(git:*)" -> "Bash") so a
// forbidden base name matches a scoped invocation. Mirrors the matching done in
// the toolaudit package for the pre-flight leak check.
func baseToolName(tool string) string {
	if i := strings.IndexByte(tool, '('); i >= 0 {
		return tool[:i]
	}
	return tool
}
