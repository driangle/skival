package report

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/driangle/skival/internal/result"
)

// censusSuite builds a two-variant suite where each variant used tools, with
// counts spread across samples so aggregation and sorting are exercised.
func censusSuite() *result.SuiteResult {
	return &result.SuiteResult{
		Evals: []result.EvalResult{{
			EvalID:   "e1",
			EvalName: "e1",
			Variants: []result.VariantResult{
				{
					Name: "no-skill",
					Runs: []result.RunResult{
						{CostUSD: 1, DurationMs: 100, Pass: boolPtr(true), ToolCounts: map[string]int{"Read": 5, "TaskCreate": 4}},
						{CostUSD: 1, DurationMs: 100, Pass: boolPtr(true), ToolCounts: map[string]int{"Read": 7, "TaskCreate": 6, "Grep": 4}},
					},
				},
				{
					Name: "control",
					Runs: []result.RunResult{
						{CostUSD: 1, DurationMs: 100, Pass: boolPtr(true), ToolCounts: map[string]int{"Read": 3}},
					},
				},
			},
		}},
	}
}

func TestVariantRankTools_AggregatedAndSorted(t *testing.T) {
	ranks := RankVariants(censusSuite(), DefaultWeights())

	var noSkill *VariantRank
	for i := range ranks {
		if ranks[i].Name == "no-skill" {
			noSkill = &ranks[i]
		}
	}
	if noSkill == nil {
		t.Fatal("no-skill variant not found in ranks")
	}
	// Read 12, TaskCreate 10, Grep 4 — sorted by count desc.
	want := []ToolCount{
		{Name: "Read", Count: 12},
		{Name: "TaskCreate", Count: 10},
		{Name: "Grep", Count: 4},
	}
	if len(noSkill.Tools) != len(want) {
		t.Fatalf("tools = %v, want %v", noSkill.Tools, want)
	}
	for i, tc := range want {
		if noSkill.Tools[i] != tc {
			t.Errorf("tools[%d] = %v, want %v", i, noSkill.Tools[i], tc)
		}
	}
}

func TestVariantRankTools_TieBrokenByName(t *testing.T) {
	sr := &result.SuiteResult{
		Evals: []result.EvalResult{{
			Variants: []result.VariantResult{{
				Name: "v",
				Runs: []result.RunResult{
					{CostUSD: 1, DurationMs: 1, Pass: boolPtr(true), ToolCounts: map[string]int{"Zed": 2, "Alpha": 2, "Mid": 2}},
				},
			}},
		}},
	}
	tools := RankVariants(sr, DefaultWeights())[0].Tools
	got := []string{tools[0].Name, tools[1].Name, tools[2].Name}
	want := []string{"Alpha", "Mid", "Zed"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("tie order = %v, want %v", got, want)
		}
	}
}

func TestWriteMarkdown_ToolUsageSection(t *testing.T) {
	var buf bytes.Buffer
	WriteMarkdown(&buf, censusSuite(), DefaultWeights())
	out := buf.String()
	if !strings.Contains(out, "## Tool Usage") {
		t.Fatalf("missing Tool Usage section:\n%s", out)
	}
	if !strings.Contains(out, "Read ×12") || !strings.Contains(out, "TaskCreate ×10") {
		t.Errorf("census counts not rendered:\n%s", out)
	}
}

func TestWriteMarkdown_NoToolUsageWhenAbsent(t *testing.T) {
	sr := &result.SuiteResult{
		Evals: []result.EvalResult{{
			Variants: []result.VariantResult{{
				Name: "v",
				Runs: []result.RunResult{{CostUSD: 1, DurationMs: 1, Pass: boolPtr(true)}},
			}},
		}},
	}
	var buf bytes.Buffer
	WriteMarkdown(&buf, sr, DefaultWeights())
	if strings.Contains(buf.String(), "## Tool Usage") {
		t.Errorf("Tool Usage section should be omitted when no tools were used")
	}
}

func TestWriteJSON_VariantToolCounts(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteJSON(&buf, censusSuite(), DefaultWeights()); err != nil {
		t.Fatal(err)
	}
	var report struct {
		Evals []struct {
			Variants []struct {
				Name       string         `json:"name"`
				ToolCounts map[string]int `json:"tool_counts"`
			} `json:"variants"`
		} `json:"evals"`
	}
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	for _, v := range report.Evals[0].Variants {
		if v.Name != "no-skill" {
			continue
		}
		if v.ToolCounts["Read"] != 12 || v.ToolCounts["TaskCreate"] != 10 || v.ToolCounts["Grep"] != 4 {
			t.Errorf("tool_counts = %v, want Read=12 TaskCreate=10 Grep=4", v.ToolCounts)
		}
		return
	}
	t.Fatal("no-skill variant not found in JSON")
}

func TestWriteJSON_ToolCountsOmittedWhenAbsent(t *testing.T) {
	sr := &result.SuiteResult{
		Evals: []result.EvalResult{{
			Variants: []result.VariantResult{{
				Name: "v",
				Runs: []result.RunResult{{CostUSD: 1, DurationMs: 1, Pass: boolPtr(true)}},
			}},
		}},
	}
	var buf bytes.Buffer
	if err := WriteJSON(&buf, sr, DefaultWeights()); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "tool_counts") {
		t.Errorf("tool_counts should be omitted when empty:\n%s", buf.String())
	}
}

func TestWriteHTML_ToolUsageSection(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteHTML(&buf, censusSuite(), DefaultWeights()); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "Tool Usage") {
		t.Fatalf("missing Tool Usage section in HTML")
	}
	if !strings.Contains(out, "Read ×12") {
		t.Errorf("census not rendered in HTML")
	}
}
