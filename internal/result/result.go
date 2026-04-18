package result

import (
	"encoding/json"
	"time"

	agentrunner "github.com/driangle/agentrunner/go"
)

// RunResult captures the outcome of a single sample run.
type RunResult struct {
	Sample     int
	Text       string
	IsError    bool
	ExitCode   int
	CostUSD    float64
	DurationMs int64
	Usage      agentrunner.Usage
	SessionID  string
	Err                error
	Pass               *bool
	Conversation       []json.RawMessage
	JudgeConversation  []json.RawMessage
	Attempt            int  // 1-indexed attempt number (0 means no retry was configured)
	TotalAttempts      int  // total attempts made for this sample
	Retried            bool // true if this result came from a retry (attempt > 1)
}

// TreatmentResult groups runs for one treatment.
type TreatmentResult struct {
	Name      string `json:"name"`
	Runner    string `json:"runner,omitempty"`
	Model     string `json:"model,omitempty"`
	IsControl bool   `json:"is_control"`
	Runs      []RunResult
	Aggregate *Aggregate
}

// SkippedTreatment records a treatment that was not executed due to a hook failure.
type SkippedTreatment struct {
	Name   string // treatment name
	Reason string // why it was skipped (e.g., "before hook failed")
}

// EvalResult groups treatments for one eval.
type EvalResult struct {
	EvalID     string
	EvalName   string
	Treatments []TreatmentResult
	Skipped    []SkippedTreatment
	Err        error
}

// SuiteResult is the top-level result for an entire suite execution.
type SuiteResult struct {
	Description string
	StartedAt   time.Time
	FinishedAt  time.Time
	Evals       []EvalResult
}
