package executor

import (
	"github.com/driangle/skival/internal/result"
	"github.com/driangle/skival/internal/suite"
	"github.com/driangle/skival/internal/toolaudit"
)

// preflightToolCheck diffs the tools the agent reported in its system/init event
// against the variant's declared allowed_tools and warns about any extras. It is
// meant to run on the first sample of each variant so leaks surface before the
// bulk of the budget is spent. It no-ops silently when the runner reports no tool
// list or no allowed_tools was declared.
func preflightToolCheck(v *suite.Variant, run result.RunResult, prog *progress) {
	available, ok := toolaudit.AvailableTools(run.Conversation)
	if !ok {
		return
	}
	allowed := toStringSlice(v.RunnerConfig["allowed_tools"])
	if extras := toolaudit.Leaks(available, allowed); len(extras) > 0 {
		prog.toolLeak(v.Name, extras)
	}
}
