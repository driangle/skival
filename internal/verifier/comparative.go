package verifier

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"

	agentrunner "github.com/driangle/agentrunner/go"
)

const comparativePromptTemplate = `You are an evaluation judge comparing several agents' outputs for the same task.

## Task Prompt
%s

## Comparison Criteria
Rate each output on how well it satisfies these criteria:
%s

## Outputs
%s

## Instructions
Score EVERY output on a 1-5 quality scale, where 5 is excellent and 1 is poor,
judging only by the criteria above. Compare the outputs against each other — do
not award every output the same score unless they are genuinely indistinguishable.
Respond with ONLY a JSON object in exactly this form, and nothing else:

{"scores": [{"label": "A", "rating": 4, "reason": "brief justification"}, ...]}

Include one entry for every output label shown above.`

// ComparativeVariant is one variant's output to be compared.
type ComparativeVariant struct {
	Name   string
	Output string
}

// ComparativeInput holds everything the comparative judge needs to score a set
// of variant outputs for a single eval.
type ComparativeInput struct {
	// EvalPrompt is the task prompt the variants were run against.
	EvalPrompt string
	// Criteria are the qualities the judge weighs.
	Criteria []string
	// Variants are the outputs to compare, in caller order. Scores are returned
	// in this same order regardless of the shuffled order shown to the judge.
	Variants []ComparativeVariant
	// MaxChars caps each output's length before it is shown to the judge. A
	// value <= 0 disables truncation.
	MaxChars int
}

// ComparativeScore is one variant's comparative quality result.
type ComparativeScore struct {
	Variant string
	Rating  int     // raw 1-5 judge rating
	Score   float64 // Rating normalized to [0,1]
	Reason  string
}

// ComparativeResult is the outcome of one comparative judging call.
type ComparativeResult struct {
	Scores       []ComparativeScore
	Conversation []json.RawMessage
	RawText      string
	// Err is set when the judge failed or returned output that could not be
	// mapped to a score for every variant. Callers degrade to per-run pass/fail.
	Err error
}

// ComparativeJudge scores the outputs of multiple variants against shared
// criteria in a single LLM call. Output order is shuffled and outputs are shown
// under anonymous labels so neither position nor variant name biases the judge.
type ComparativeJudge struct {
	Runner agentrunner.Runner
	Model  string
	// Rand, when set, controls output-order shuffling. Leave nil in production
	// (uses the global source); set it in tests for deterministic ordering.
	Rand *rand.Rand
}

// Compare scores the given variant outputs and returns per-variant results in
// the caller's original order. On any failure it returns a result with Err set
// and no scores, so the caller can fall back to per-run pass/fail.
func (j *ComparativeJudge) Compare(ctx context.Context, in ComparativeInput) ComparativeResult {
	if len(in.Variants) < 2 {
		return ComparativeResult{Err: fmt.Errorf("comparative judge needs at least 2 variants, got %d", len(in.Variants))}
	}

	// Shuffle the display order so position does not bias the judge.
	perm := j.perm(len(in.Variants))
	labels := make([]string, len(in.Variants))
	var outputs strings.Builder
	for pos, orig := range perm {
		label := comparativeLabel(pos)
		labels[orig] = label
		out := in.Variants[orig].Output
		if in.MaxChars > 0 {
			out = truncate(out, in.MaxChars)
		}
		if strings.TrimSpace(out) == "" {
			out = "(empty output)"
		}
		fmt.Fprintf(&outputs, "### Output %s\n%s\n\n", label, out)
	}

	criteria := "- " + strings.Join(in.Criteria, "\n- ")
	prompt := fmt.Sprintf(comparativePromptTemplate, in.EvalPrompt, criteria, strings.TrimRight(outputs.String(), "\n"))

	model := j.Model
	if model == "" {
		model = DefaultJudgeModel
	}

	session, err := j.Runner.Start(ctx, prompt,
		agentrunner.WithModel(model),
		agentrunner.WithSkipPermissions(),
	)
	if err != nil {
		return ComparativeResult{Err: fmt.Errorf("comparative judge invocation failed: %w", err)}
	}

	var conversation []json.RawMessage
	for msg := range session.Messages {
		if msg.Raw != nil {
			conversation = append(conversation, msg.Raw)
		}
	}

	res, err := session.Result()
	if err != nil {
		return ComparativeResult{Conversation: conversation, Err: fmt.Errorf("comparative judge invocation failed: %w", err)}
	}

	scores, err := parseComparativeResponse(res.Text, labels, in.Variants)
	return ComparativeResult{
		Scores:       scores,
		Conversation: conversation,
		RawText:      res.Text,
		Err:          err,
	}
}

// perm returns a permutation of [0,n) using the judge's Rand if set, else the
// global source.
func (j *ComparativeJudge) perm(n int) []int {
	if j.Rand != nil {
		return j.Rand.Perm(n)
	}
	return rand.Perm(n)
}

// comparativeLabel maps a zero-based position to a stable anonymous label:
// A..Z, then V27, V28, ... for the (rare) case of more than 26 variants.
func comparativeLabel(pos int) string {
	if pos < 26 {
		return string(rune('A' + pos))
	}
	return fmt.Sprintf("V%d", pos+1)
}

// judgeScores is the JSON shape the comparative judge is asked to return.
type judgeScores struct {
	Scores []struct {
		Label  string `json:"label"`
		Rating int    `json:"rating"`
		Reason string `json:"reason"`
	} `json:"scores"`
}

// parseComparativeResponse maps the judge's JSON back to per-variant scores in
// the caller's original order. It fails if the JSON is unparseable or if any
// variant is missing a valid (1-5) rating, so scoring is all-or-nothing.
func parseComparativeResponse(text string, labels []string, variants []ComparativeVariant) ([]ComparativeScore, error) {
	raw := extractJSONObject(text)
	if raw == "" {
		return nil, fmt.Errorf("comparative judge response contained no JSON object: %s", truncate(strings.TrimSpace(text), 200))
	}

	var parsed judgeScores
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, fmt.Errorf("comparative judge response could not be parsed: %w", err)
	}

	byLabel := make(map[string]struct {
		rating int
		reason string
	}, len(parsed.Scores))
	for _, s := range parsed.Scores {
		byLabel[strings.ToUpper(strings.TrimSpace(s.Label))] = struct {
			rating int
			reason string
		}{s.Rating, s.Reason}
	}

	scores := make([]ComparativeScore, len(variants))
	for i := range variants {
		entry, ok := byLabel[strings.ToUpper(labels[i])]
		if !ok {
			return nil, fmt.Errorf("comparative judge omitted a score for output %s", labels[i])
		}
		if entry.rating < 1 || entry.rating > 5 {
			return nil, fmt.Errorf("comparative judge returned out-of-range rating %d for output %s", entry.rating, labels[i])
		}
		scores[i] = ComparativeScore{
			Variant: variants[i].Name,
			Rating:  entry.rating,
			Score:   float64(entry.rating) / 5.0,
			Reason:  strings.TrimSpace(entry.reason),
		}
	}
	return scores, nil
}

// extractJSONObject returns the substring spanning the first '{' to the last
// '}' in text, tolerating prose or code fences around the JSON. Returns "" when
// no brace pair is present.
func extractJSONObject(text string) string {
	start := strings.IndexByte(text, '{')
	end := strings.LastIndexByte(text, '}')
	if start < 0 || end < 0 || end < start {
		return ""
	}
	return text[start : end+1]
}
