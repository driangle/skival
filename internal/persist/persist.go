package persist

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/driangle/skival/internal/report"
	"github.com/driangle/skival/internal/result"
)

// Save writes all result data to a timestamped directory under baseDir.
// Returns the created directory path.
func Save(baseDir string, sr *result.SuiteResult, weights report.Weights) (string, error) {
	timestamp := sr.StartedAt.Format("20060102-150405")
	dir := filepath.Join(baseDir, timestamp)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating results dir: %w", err)
	}

	if err := writeEvals(dir, sr); err != nil {
		return "", err
	}

	if err := writeSummary(dir, sr, weights); err != nil {
		return "", err
	}

	return dir, nil
}

func writeEvals(dir string, sr *result.SuiteResult) error {
	evalsDir := filepath.Join(dir, "evals")

	for _, eval := range sr.Evals {
		for _, variant := range eval.Variants {
			variantDir := filepath.Join(evalsDir, eval.EvalID, variant.Name)
			if err := os.MkdirAll(variantDir, 0o755); err != nil {
				return fmt.Errorf("creating variant dir: %w", err)
			}

			for _, run := range variant.Runs {
				if err := writeRunJSON(variantDir, run); err != nil {
					return err
				}
			}

			if variant.Aggregate != nil {
				if err := writeAtomicJSON(filepath.Join(variantDir, "aggregate.json"), variant.Aggregate); err != nil {
					return fmt.Errorf("writing aggregate.json: %w", err)
				}
			}
		}
	}

	return nil
}

func writeSummary(dir string, sr *result.SuiteResult, weights report.Weights) error {
	// summary.json
	if err := writeAtomicJSON(filepath.Join(dir, "summary.json"), buildSummaryJSON(sr, weights)); err != nil {
		return fmt.Errorf("writing summary.json: %w", err)
	}

	// summary.md
	summaryPath := filepath.Join(dir, "summary.md")
	f, err := os.CreateTemp(filepath.Dir(summaryPath), ".summary-*.md")
	if err != nil {
		return fmt.Errorf("creating temp file for summary.md: %w", err)
	}
	report.WriteMarkdown(f, sr, weights)
	f.Close()
	if err := os.Rename(f.Name(), summaryPath); err != nil {
		os.Remove(f.Name())
		return fmt.Errorf("writing summary.md: %w", err)
	}

	return nil
}

// runJSON is the serialized form of a single run.
type runJSON struct {
	Sample        int     `json:"sample"`
	Text          string  `json:"text"`
	IsError       bool    `json:"is_error"`
	ExitCode      int     `json:"exit_code"`
	CostUSD       float64 `json:"cost_usd"`
	DurationMs    int64   `json:"duration_ms"`
	SessionID     string  `json:"session_id,omitempty"`
	Pass          *bool   `json:"pass"`
	Error         string  `json:"error,omitempty"`
	Attempt       int     `json:"attempt,omitempty"`
	TotalAttempts int     `json:"total_attempts,omitempty"`
	Retried       bool    `json:"retried,omitempty"`
}

func writeRunJSON(variantDir string, run result.RunResult) error {
	r := runJSON{
		Sample:        run.Sample,
		Text:          run.Text,
		IsError:       run.IsError,
		ExitCode:      run.ExitCode,
		CostUSD:       run.CostUSD,
		DurationMs:    run.DurationMs,
		SessionID:     run.SessionID,
		Pass:          run.Pass,
		Attempt:       run.Attempt,
		TotalAttempts: run.TotalAttempts,
		Retried:       run.Retried,
	}
	if run.Err != nil {
		r.Error = run.Err.Error()
	}

	filename := fmt.Sprintf("run-%d.json", run.Sample)
	if err := writeAtomicJSON(filepath.Join(variantDir, filename), r); err != nil {
		return err
	}

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

type summaryJSON struct {
	Description string               `json:"description"`
	StartedAt   string               `json:"started_at"`
	FinishedAt  string               `json:"finished_at"`
	Rankings    []report.VariantRank `json:"rankings,omitempty"`
}

func buildSummaryJSON(sr *result.SuiteResult, weights report.Weights) summaryJSON {
	return summaryJSON{
		Description: sr.Description,
		StartedAt:   sr.StartedAt.Format(time.RFC3339),
		FinishedAt:  sr.FinishedAt.Format(time.RFC3339),
		Rankings:    report.RankVariants(sr, weights),
	}
}

// writeAtomicJSON writes data as JSON to path via a temp file + rename.
func writeAtomicJSON(path string, data any) error {
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, ".tmp-*.json")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		f.Close()
		os.Remove(f.Name())
		return fmt.Errorf("encoding JSON: %w", err)
	}
	f.Close()

	if err := os.Rename(f.Name(), path); err != nil {
		os.Remove(f.Name())
		return err
	}
	return nil
}
