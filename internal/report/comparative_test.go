package report

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/driangle/skival/internal/result"
)

func comparisonResult() *result.SuiteResult {
	return &result.SuiteResult{
		Description: "cmp",
		Evals: []result.EvalResult{{
			EvalID:   "eval-1",
			EvalName: "Eval One",
			Variants: []result.VariantResult{
				{Name: "control", Runs: []result.RunResult{{CostUSD: 1, DurationMs: 100, Pass: boolPtr(true)}}},
				{Name: "treatment", Runs: []result.RunResult{{CostUSD: 1, DurationMs: 100, Pass: boolPtr(true)}}},
			},
			Comparison: &result.Comparison{
				Model: "judge-model",
				Scores: []result.ComparativeScore{
					{Variant: "control", Rating: 3, Score: 0.6, Reason: "ok"},
					{Variant: "treatment", Rating: 5, Score: 1.0, Reason: "great"},
				},
			},
		}},
	}
}

func TestMarkdown_RendersComparison(t *testing.T) {
	var buf bytes.Buffer
	WriteMarkdown(&buf, comparisonResult(), DefaultWeights())
	out := buf.String()

	if !strings.Contains(out, "## Comparative Quality") {
		t.Error("markdown missing comparative quality section")
	}
	if !strings.Contains(out, "judge-model") {
		t.Error("markdown missing judge model")
	}
	if !strings.Contains(out, "great") {
		t.Error("markdown missing score reason")
	}
	if !strings.Contains(out, "QUALITY") {
		t.Error("markdown ranking table missing QUALITY column")
	}
}

func TestMarkdown_RendersSkippedComparison(t *testing.T) {
	sr := comparisonResult()
	sr.Evals[0].Comparison = &result.Comparison{Skipped: "judge errored"}

	var buf bytes.Buffer
	WriteMarkdown(&buf, sr, DefaultWeights())
	out := buf.String()
	if !strings.Contains(out, "judge errored") {
		t.Error("markdown should surface the skip reason")
	}
	// No scores -> no QUALITY column in rankings.
	if strings.Contains(out, "QUALITY") {
		t.Error("skipped comparison should not add a QUALITY column")
	}
}

func TestMarkdown_NoComparisonSectionWhenAbsent(t *testing.T) {
	sr := comparisonResult()
	sr.Evals[0].Comparison = nil

	var buf bytes.Buffer
	WriteMarkdown(&buf, sr, DefaultWeights())
	if strings.Contains(buf.String(), "Comparative Quality") {
		t.Error("no comparison section expected when no comparison ran")
	}
}

func TestJSON_RendersComparison(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteJSON(&buf, comparisonResult(), Weights{Correctness: 0.7, Cost: 0.1, Duration: 0.1, Quality: 0.1}); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}

	var parsed struct {
		Evals []struct {
			Comparison *struct {
				Model  string `json:"model"`
				Scores []struct {
					Variant string  `json:"variant"`
					Rating  int     `json:"rating"`
					Score   float64 `json:"score"`
				} `json:"scores"`
			} `json:"comparison"`
		} `json:"evals"`
		Rankings []struct {
			Name         string   `json:"name"`
			QualityScore *float64 `json:"quality_score"`
		} `json:"rankings"`
	}
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	c := parsed.Evals[0].Comparison
	if c == nil {
		t.Fatal("json missing comparison")
	}
	if c.Model != "judge-model" || len(c.Scores) != 2 {
		t.Errorf("unexpected comparison: %+v", c)
	}
	if len(parsed.Rankings) == 0 || parsed.Rankings[0].QualityScore == nil {
		t.Fatal("json rankings missing quality_score")
	}
}

func TestJSON_OmitsQualityScoreWithoutComparison(t *testing.T) {
	sr := comparisonResult()
	sr.Evals[0].Comparison = nil

	var buf bytes.Buffer
	if err := WriteJSON(&buf, sr, DefaultWeights()); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	if strings.Contains(buf.String(), "quality_score") {
		t.Error("quality_score should be omitted when no comparison ran")
	}
}

func TestHTML_RendersComparison(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteHTML(&buf, comparisonResult(), DefaultWeights()); err != nil {
		t.Fatalf("WriteHTML: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Comparative Quality") {
		t.Error("html missing comparative quality section")
	}
	if !strings.Contains(out, ">Quality<") {
		t.Error("html rankings missing Quality column header")
	}
	if !strings.Contains(out, "great") {
		t.Error("html missing score reason")
	}
}
