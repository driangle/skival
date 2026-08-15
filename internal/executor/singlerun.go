package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	agentrunner "github.com/driangle/agentrunner/go"
	"github.com/driangle/skival/internal/result"
	"github.com/driangle/skival/internal/suite"
	"github.com/driangle/skival/internal/verifier"
)

func executeSingleRun(ctx context.Context, eval *suite.Eval, v *suite.Variant, sample int, runner agentrunner.Runner, isolatedDir string, timeoutOverride int) result.RunResult {
	opts, err := buildRunOptions(eval, v, isolatedDir, timeoutOverride)
	if err != nil {
		return result.RunResult{Sample: sample, Err: err}
	}

	session, err := runner.Start(ctx, resolvePrompt(eval, v), opts...)
	if err != nil {
		return result.RunResult{Sample: sample, Err: err}
	}

	conversation := collectConversation(session)

	res, err := session.Result()
	if err != nil {
		return result.RunResult{Sample: sample, Err: err}
	}

	return newRunResult(sample, res, conversation)
}

// resolvePrompt returns the prompt to run: variant prompt takes precedence over eval prompt.
func resolvePrompt(eval *suite.Eval, v *suite.Variant) string {
	if v.Prompt != "" {
		return v.Prompt
	}
	return eval.Prompt
}

// collectConversation drains the session's raw messages into a conversation slice.
func collectConversation(session *agentrunner.Session) []json.RawMessage {
	var conversation []json.RawMessage
	for msg := range session.Messages {
		if msg.Raw != nil {
			conversation = append(conversation, msg.Raw)
		}
	}
	return conversation
}

// newRunResult builds a successful RunResult from a runner result and conversation.
func newRunResult(sample int, res *agentrunner.Result, conversation []json.RawMessage) result.RunResult {
	return result.RunResult{
		Sample:       sample,
		Text:         res.Text,
		IsError:      res.IsError,
		ExitCode:     res.ExitCode,
		CostUSD:      res.CostUSD,
		DurationMs:   res.Duration.Milliseconds(),
		Usage:        res.Usage,
		SessionID:    res.SessionID,
		Conversation: conversation,
		ToolCounts:   verifier.CountToolUses(conversation),
	}
}

func buildRunOptions(eval *suite.Eval, v *suite.Variant, isolatedDir string, timeoutOverride int) ([]agentrunner.Option, error) {
	var opts []agentrunner.Option

	if v.Model != "" {
		opts = append(opts, agentrunner.WithModel(v.Model))
	}

	if dir := resolveWorkdir(eval, v, isolatedDir); dir != "" {
		opts = append(opts, agentrunner.WithWorkingDir(dir))
	}

	// Timeout: CLI override > eval-level.
	if timeoutOverride > 0 {
		opts = append(opts, agentrunner.WithTimeout(time.Duration(timeoutOverride)*time.Second))
	} else if eval.Timeout != nil {
		opts = append(opts, agentrunner.WithTimeout(time.Duration(*eval.Timeout)*time.Second))
	}

	if env := buildRunEnv(v); len(env) > 0 {
		opts = append(opts, agentrunner.WithEnv(env))
	}

	// Runner-specific options from runner_config.
	opts = append(opts, buildRunnerSpecificOpts(v.Runner, v.RunnerConfig)...)

	// Skill file(s) as appended system prompt.
	skillPrompt, err := loadSkillContent(v)
	if err != nil {
		return nil, err
	}
	if skillPrompt != "" {
		opts = append(opts, agentrunner.WithAppendSystemPrompt(skillPrompt))
	}

	// Always skip permissions for automated runs.
	opts = append(opts, agentrunner.WithSkipPermissions())

	return opts, nil
}

// buildRunEnv returns the environment variables for a run: the variant's env
// plus CLAUDE_CONFIG_DIR when config_dir is set. The variant's env map is never mutated.
func buildRunEnv(v *suite.Variant) map[string]string {
	env := v.Env
	if v.ConfigDir == "" {
		return env
	}
	copied := make(map[string]string, len(env)+1)
	for k, val := range env {
		copied[k] = val
	}
	copied["CLAUDE_CONFIG_DIR"] = v.ConfigDir
	return copied
}

// loadSkillContent reads skill file(s) from a variant and returns the concatenated content.
// Returns empty string if no skills are configured.
func loadSkillContent(v *suite.Variant) (string, error) {
	if v.Skill != "" {
		content, err := os.ReadFile(v.Skill)
		if err != nil {
			return "", fmt.Errorf("reading skill file %q: %w", v.Skill, err)
		}
		return string(content), nil
	}

	if len(v.Skills) > 0 {
		var parts []string
		for _, path := range v.Skills {
			content, err := os.ReadFile(path)
			if err != nil {
				return "", fmt.Errorf("reading skill file %q: %w", path, err)
			}
			parts = append(parts, string(content))
		}
		return strings.Join(parts, "\n\n"), nil
	}

	return "", nil
}
