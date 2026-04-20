package verifier

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	agentrunner "github.com/driangle/agentrunner/go"
)

// DefaultJudgeModel is the default model used by JudgeVerifier when none is configured.
const DefaultJudgeModel = "claude-haiku-4-5-20251001"

const judgePromptTemplate = `You are an evaluation judge. Determine whether the agent's output satisfies the given criteria.

## Agent Model
The agent under evaluation ran as: %s
This may differ from your own model. When a criterion references "the agent's model name," use the value above — not your own identity.

## Eval Prompt
%s

## Agent Tool Activity
The agent made the following tool calls and received these results during its session (values may be truncated):
%s

## Agent Final Response
%s

## Criteria
%s

## Instructions
Evaluate whether the agent's tool activity and final response together satisfy ALL of the criteria above.
When a criterion asks about tools the agent called or data it queried, rely on the tool activity — not the final response — as the source of truth.
Respond with exactly one of these formats:
PASS: <brief reason>
FAIL: <brief reason explaining which criteria were not met>
`

// JudgeVerifier uses an LLM to evaluate subjective correctness criteria.
type JudgeVerifier struct {
	Runner     agentrunner.Runner
	Criteria   []string
	Prompt     string
	Model      string
	AgentModel string
}

func (v *JudgeVerifier) Verify(ctx context.Context, input VerifyInput) VerifyResult {
	criteria := strings.Join(v.Criteria, "\n- ")
	toolActivity := SummarizeToolActivity(input.Conversation)
	if toolActivity == "" {
		toolActivity = "(no tool calls recorded)"
	}
	judgePrompt := fmt.Sprintf(judgePromptTemplate, v.AgentModel, v.Prompt, toolActivity, input.RunOutput, "- "+criteria)

	model := v.Model
	if model == "" {
		model = DefaultJudgeModel
	}

	session, err := v.Runner.Start(ctx, judgePrompt,
		agentrunner.WithModel(model),
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
