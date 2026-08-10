package result

import (
	"encoding/json"
	"time"

	agentrunner "github.com/driangle/agentrunner/go"
)

// RunResult captures the outcome of a single sample run.
type RunResult struct {
	Sample            int
	Text              string
	IsError           bool
	ExitCode          int
	CostUSD           float64
	DurationMs        int64
	Usage             agentrunner.Usage
	SessionID         string
	Err               error
	Pass              *bool
	Conversation      []json.RawMessage
	JudgeConversation []json.RawMessage
	WorkDir           string // resolved working directory for this sample
	Attempt           int    // 1-indexed attempt number (0 means no retry was configured)
	TotalAttempts     int    // total attempts made for this sample
	Retried           bool   // true if this result came from a retry (attempt > 1)
}

// VariantResult groups runs for one variant.
type VariantResult struct {
	Name      string `json:"name"`
	Runner    string `json:"runner,omitempty"`
	Model     string `json:"model,omitempty"`
	IsControl bool   `json:"is_control"`
	Runs      []RunResult
	Aggregate *Aggregate
}

// SkippedVariant records a variant that was not executed due to a hook failure.
type SkippedVariant struct {
	Name   string // variant name
	Reason string // why it was skipped (e.g., "before hook failed")
}

// EvalResult groups treatments for one eval.
type EvalResult struct {
	EvalID   string
	EvalName string
	Variants []VariantResult
	Skipped  []SkippedVariant
	Err      error
	// Comparison holds comparative-judge results for this eval, when comparative
	// judging ran. Nil when the feature was disabled or skipped for the eval.
	Comparison *Comparison
}

// Comparison holds the outcome of comparative judging across the passing
// variants of a single eval. It carries a per-variant quality score on a
// normalized [0,1] scale (derived from the judge's 1-5 rating).
type Comparison struct {
	// Model is the judge model that produced the scores.
	Model string `json:"model,omitempty"`
	// Scores is the per-variant quality result, one entry per compared variant.
	Scores []ComparativeScore `json:"scores,omitempty"`
	// Skipped, when non-empty, explains why comparison did not produce scores
	// (e.g. too few passing variants, judge error). Scores is empty in that case.
	Skipped string `json:"skipped,omitempty"`
	// Conversation holds the raw JSON messages from the judge run, if any. It is
	// persisted separately (as JSONL), not inlined into the comparison record.
	Conversation []json.RawMessage `json:"-"`
}

// ComparativeScore is one variant's comparative quality result for an eval.
type ComparativeScore struct {
	// Variant is the variant name this score belongs to.
	Variant string `json:"variant"`
	// Rating is the judge's raw 1-5 quality rating.
	Rating int `json:"rating"`
	// Score is Rating normalized to [0,1] (Rating/5), the value fed to ranking.
	Score float64 `json:"score"`
	// Reason is the judge's brief justification for the rating.
	Reason string `json:"reason,omitempty"`
}

// SuiteResult is the top-level result for an entire suite execution.
type SuiteResult struct {
	Description string
	StartedAt   time.Time
	FinishedAt  time.Time
	Evals       []EvalResult
}
