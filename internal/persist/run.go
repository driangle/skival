package persist

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	agentrunner "github.com/driangle/agentrunner/go"
	"github.com/driangle/skival/internal/result"
)

// runJSON is the serialized form of a single run.
type runJSON struct {
	Sample        int        `json:"sample"`
	Text          string     `json:"text"`
	IsError       bool       `json:"is_error"`
	ExitCode      int        `json:"exit_code"`
	CostUSD       float64    `json:"cost_usd"`
	DurationMs    int64      `json:"duration_ms"`
	Usage         *usageJSON `json:"usage,omitempty"`
	SessionID     string     `json:"session_id,omitempty"`
	SessionPage   string     `json:"session_page,omitempty"`
	Pass          *bool      `json:"pass"`
	Error         string     `json:"error,omitempty"`
	Attempt       int        `json:"attempt,omitempty"`
	TotalAttempts int        `json:"total_attempts,omitempty"`
	Retried       bool       `json:"retried,omitempty"`
}

// usageJSON is the serialized token usage for a single run. It is a pointer on
// runJSON so runners that report no tokens (e.g. the exec runner) omit it
// entirely, and older result dirs without the field load as zero usage.
type usageJSON struct {
	Input         int `json:"input"`
	Output        int `json:"output"`
	CacheCreation int `json:"cache_creation"`
	CacheRead     int `json:"cache_read"`
}

// toUsageJSON converts an agentrunner.Usage into its serialized form, returning
// nil when every field is zero so no empty usage object is written.
func toUsageJSON(u agentrunner.Usage) *usageJSON {
	if u.InputTokens == 0 && u.OutputTokens == 0 &&
		u.CacheCreationInputTokens == 0 && u.CacheReadInputTokens == 0 {
		return nil
	}
	return &usageJSON{
		Input:         u.InputTokens,
		Output:        u.OutputTokens,
		CacheCreation: u.CacheCreationInputTokens,
		CacheRead:     u.CacheReadInputTokens,
	}
}

// toUsage converts serialized usage back into agentrunner.Usage. A nil pointer
// (older data or a no-token runner) yields the zero value.
func (u *usageJSON) toUsage() agentrunner.Usage {
	if u == nil {
		return agentrunner.Usage{}
	}
	return agentrunner.Usage{
		InputTokens:              u.Input,
		OutputTokens:             u.Output,
		CacheCreationInputTokens: u.CacheCreation,
		CacheReadInputTokens:     u.CacheRead,
	}
}

// toRunResult maps the serialized run fields (everything except the conversation
// sidecars, which loadRun attaches) back onto a RunResult.
func (rj runJSON) toRunResult() result.RunResult {
	run := result.RunResult{
		Sample:        rj.Sample,
		Text:          rj.Text,
		IsError:       rj.IsError,
		ExitCode:      rj.ExitCode,
		CostUSD:       rj.CostUSD,
		DurationMs:    rj.DurationMs,
		Usage:         rj.Usage.toUsage(),
		SessionID:     rj.SessionID,
		SessionPage:   rj.SessionPage,
		Pass:          rj.Pass,
		Attempt:       rj.Attempt,
		TotalAttempts: rj.TotalAttempts,
		Retried:       rj.Retried,
	}
	if rj.Error != "" {
		run.Err = fmt.Errorf("%s", rj.Error)
	}
	return run
}

// writeRunMeta writes run-N.json for a single run.
func writeRunMeta(variantDir string, run result.RunResult) error {
	r := runJSON{
		Sample:        run.Sample,
		Text:          run.Text,
		IsError:       run.IsError,
		ExitCode:      run.ExitCode,
		CostUSD:       run.CostUSD,
		DurationMs:    run.DurationMs,
		Usage:         toUsageJSON(run.Usage),
		SessionID:     run.SessionID,
		SessionPage:   run.SessionPage,
		Pass:          run.Pass,
		Attempt:       run.Attempt,
		TotalAttempts: run.TotalAttempts,
		Retried:       run.Retried,
	}
	if run.Err != nil {
		r.Error = run.Err.Error()
	}

	filename := fmt.Sprintf("run-%d.json", run.Sample)
	return writeAtomicJSON(filepath.Join(variantDir, filename), r)
}

// writeRunSidecars writes a run's transcript JSONL files (conversation and, if
// present, the judge conversation).
func writeRunSidecars(variantDir string, run result.RunResult) error {
	if len(run.Conversation) > 0 {
		convPath := filepath.Join(variantDir, fmt.Sprintf("run-%d.conversation.jsonl", run.Sample))
		if err := writeConversationJSONL(convPath, run.Conversation); err != nil {
			return err
		}
	}

	if len(run.JudgeConversation) > 0 {
		judgePath := filepath.Join(variantDir, fmt.Sprintf("run-%d.judge.jsonl", run.Sample))
		if err := writeConversationJSONL(judgePath, run.JudgeConversation); err != nil {
			return err
		}
	}

	return nil
}

// writeConversationJSONL writes raw JSON messages as JSONL via atomic temp+rename.
func writeConversationJSONL(path string, messages []json.RawMessage) error {
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, ".tmp-*.jsonl")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}

	for _, raw := range messages {
		if _, err := f.Write(raw); err != nil {
			f.Close()
			os.Remove(f.Name())
			return fmt.Errorf("writing JSONL line: %w", err)
		}
		if _, err := f.Write([]byte("\n")); err != nil {
			f.Close()
			os.Remove(f.Name())
			return fmt.Errorf("writing newline: %w", err)
		}
	}
	f.Close()

	if err := os.Rename(f.Name(), path); err != nil {
		os.Remove(f.Name())
		return err
	}
	return nil
}
