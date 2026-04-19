package persist

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/driangle/skival/internal/result"
)

// Load reads a persisted results directory and reconstructs a SuiteResult.
func Load(dir string) (*result.SuiteResult, error) {
	summaryPath := filepath.Join(dir, "summary.json")
	data, err := os.ReadFile(summaryPath)
	if err != nil {
		return nil, fmt.Errorf("reading summary.json: %w", err)
	}

	var summary summaryJSON
	if err := json.Unmarshal(data, &summary); err != nil {
		return nil, fmt.Errorf("parsing summary.json: %w", err)
	}

	startedAt, _ := time.Parse(time.RFC3339, summary.StartedAt)
	finishedAt, _ := time.Parse(time.RFC3339, summary.FinishedAt)

	sr := &result.SuiteResult{
		Description: summary.Description,
		StartedAt:   startedAt,
		FinishedAt:  finishedAt,
	}

	evalsDir := filepath.Join(dir, "evals")
	evalEntries, err := os.ReadDir(evalsDir)
	if err != nil {
		return sr, nil // no evals dir is valid (empty suite)
	}

	for _, evalEntry := range evalEntries {
		if !evalEntry.IsDir() {
			continue
		}
		evalResult, err := loadEval(filepath.Join(evalsDir, evalEntry.Name()), evalEntry.Name())
		if err != nil {
			return nil, fmt.Errorf("loading eval %s: %w", evalEntry.Name(), err)
		}
		sr.Evals = append(sr.Evals, evalResult)
	}

	return sr, nil
}

func loadEval(evalDir, evalID string) (result.EvalResult, error) {
	er := result.EvalResult{
		EvalID:   evalID,
		EvalName: evalID,
	}

	varEntries, err := os.ReadDir(evalDir)
	if err != nil {
		return er, fmt.Errorf("reading eval dir: %w", err)
	}

	for _, varEntry := range varEntries {
		if !varEntry.IsDir() {
			continue
		}
		tr, err := loadVariant(filepath.Join(evalDir, varEntry.Name()), varEntry.Name())
		if err != nil {
			return er, fmt.Errorf("loading variant %s: %w", varEntry.Name(), err)
		}
		er.Variants = append(er.Variants, tr)
	}

	return er, nil
}

func loadVariant(varDir, name string) (result.VariantResult, error) {
	tr := result.VariantResult{Name: name}

	entries, err := os.ReadDir(varDir)
	if err != nil {
		return tr, fmt.Errorf("reading variant dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "run-") || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(varDir, entry.Name()))
		if err != nil {
			return tr, fmt.Errorf("reading %s: %w", entry.Name(), err)
		}

		var rj runJSON
		if err := json.Unmarshal(data, &rj); err != nil {
			return tr, fmt.Errorf("parsing %s: %w", entry.Name(), err)
		}

		run := result.RunResult{
			Sample:        rj.Sample,
			Text:          rj.Text,
			IsError:       rj.IsError,
			ExitCode:      rj.ExitCode,
			CostUSD:       rj.CostUSD,
			DurationMs:    rj.DurationMs,
			SessionID:     rj.SessionID,
			Pass:          rj.Pass,
			Attempt:       rj.Attempt,
			TotalAttempts: rj.TotalAttempts,
			Retried:       rj.Retried,
		}
		if rj.Error != "" {
			run.Err = fmt.Errorf("%s", rj.Error)
		}

		// Load conversation JSONL files if they exist.
		baseName := strings.TrimSuffix(entry.Name(), ".json")
		run.Conversation, err = loadConversationJSONL(filepath.Join(varDir, baseName+".conversation.jsonl"))
		if err != nil {
			return tr, fmt.Errorf("loading conversation for %s: %w", entry.Name(), err)
		}
		run.JudgeConversation, err = loadConversationJSONL(filepath.Join(varDir, baseName+".judge.jsonl"))
		if err != nil {
			return tr, fmt.Errorf("loading judge conversation for %s: %w", entry.Name(), err)
		}

		tr.Runs = append(tr.Runs, run)
	}

	// Load aggregate if present.
	aggPath := filepath.Join(varDir, "aggregate.json")
	if data, err := os.ReadFile(aggPath); err == nil {
		var agg result.Aggregate
		if err := json.Unmarshal(data, &agg); err == nil {
			tr.Aggregate = &agg
		}
	}

	return tr, nil
}

// loadConversationJSONL reads a JSONL file and returns each line as json.RawMessage.
// Returns nil, nil if the file doesn't exist (backwards compatible).
func loadConversationJSONL(path string) ([]json.RawMessage, error) {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var messages []json.RawMessage
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		raw := make(json.RawMessage, len(line))
		copy(raw, line)
		messages = append(messages, raw)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return messages, nil
}
