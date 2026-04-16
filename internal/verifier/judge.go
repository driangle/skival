package verifier

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	agentrunner "github.com/driangle/agentrunner/go"
)

const judgeModel = "claude-haiku-4-5-20251001"

const judgePromptTemplate = `You are an evaluation judge. Determine whether the agent's output satisfies the given criteria.

## Eval Prompt
%s

## Agent Output
%s

## Criteria
%s

## Instructions
Evaluate whether the agent's output satisfies ALL of the criteria above.
Respond with exactly one of these formats:
PASS: <brief reason>
FAIL: <brief reason explaining which criteria were not met>
`

// JudgeVerifier uses an LLM to evaluate subjective correctness criteria.
type JudgeVerifier struct {
	Runner   agentrunner.Runner
	Criteria []string
	Prompt   string
}

func (v *JudgeVerifier) Verify(ctx context.Context, input VerifyInput) VerifyResult {
	criteria := strings.Join(v.Criteria, "\n- ")
	judgePrompt := fmt.Sprintf(judgePromptTemplate, v.Prompt, input.RunOutput, "- "+criteria)

	session, err := v.Runner.Start(ctx, judgePrompt,
		agentrunner.WithModel(judgeModel),
		agentrunner.WithSkipPermissions(),
	)
	if err != nil {
		return VerifyResult{
			Pass:   false,
			Reason: fmt.Sprintf("judge invocation failed: %v", err),
		}
	}

	var conversation []json.RawMessage
	for msg := range session.Messages {
		if msg.Raw != nil {
			conversation = append(conversation, msg.Raw)
		}
	}

	res, err := session.Result()
	if err != nil {
		return VerifyResult{
			Pass:   false,
			Reason: fmt.Sprintf("judge invocation failed: %v", err),
		}
	}

	vr := parseJudgeResponse(res.Text)
	vr.Conversation = conversation
	return vr
}

func parseJudgeResponse(text string) VerifyResult {
	text = strings.TrimSpace(text)

	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		upper := strings.ToUpper(line)

		if strings.HasPrefix(upper, "PASS") {
			reason := strings.TrimSpace(strings.TrimPrefix(line, line[:4]))
			reason = strings.TrimPrefix(reason, ":")
			reason = strings.TrimSpace(reason)
			if reason == "" {
				reason = "judge passed"
			}
			return VerifyResult{Pass: true, Reason: reason}
		}

		if strings.HasPrefix(upper, "FAIL") {
			reason := strings.TrimSpace(strings.TrimPrefix(line, line[:4]))
			reason = strings.TrimPrefix(reason, ":")
			reason = strings.TrimSpace(reason)
			if reason == "" {
				reason = "judge failed"
			}
			return VerifyResult{Pass: false, Reason: reason}
		}
	}

	return VerifyResult{
		Pass:   false,
		Reason: fmt.Sprintf("judge response could not be parsed: %s", truncate(text, 200)),
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
