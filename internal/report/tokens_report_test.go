package report

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/driangle/skival/internal/result"
)

// tokenSuite builds a two-variant suite where the variants differ on token usage,
// used to check how each report surfaces the token dimension.
func tokenSuite() *result.SuiteResult {
	return &result.SuiteResult{
		StartedAt:  time.Now(),
		FinishedAt: time.Now(),
		Evals: []result.EvalResult{{
			Variants: []result.VariantResult{
				{Name: "hungry", Runs: []result.RunResult{tokenRun(9000, 1000)}},
				{Name: "terse", Runs: []result.RunResult{tokenRun(900, 100)}},
			},
		}},
	}
}

func tokenWeights() Weights {
	return Weights{Correctness: 0.5, Duration: 0.2, Tokens: 0.3}
}

func TestMarkdown_TokenColumnGatedByWeight(t *testing.T) {
	sr := tokenSuite()

	var withTokens bytes.Buffer
	WriteMarkdown(&withTokens, sr, tokenWeights())
	if !strings.Contains(withTokens.String(), "MEDIAN TOKENS") {
		t.Error("expected MEDIAN TOKENS column when tokens weight is active")
	}
	if !strings.Contains(withTokens.String(), "1.0k") {
		t.Errorf("expected terse variant's 1000 total tokens rendered, got:\n%s", withTokens.String())
	}

	var noTokens bytes.Buffer
	WriteMarkdown(&noTokens, sr, DefaultWeights())
	if strings.Contains(noTokens.String(), "MEDIAN TOKENS") {
		t.Error("default weights must not add the MEDIAN TOKENS column")
	}
}

func TestJSON_TokenFieldGatedByWeight(t *testing.T) {
	sr := tokenSuite()

	var withTokens bytes.Buffer
	if err := WriteJSON(&withTokens, sr, tokenWeights()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(withTokens.String(), "median_total_tokens") {
		t.Error("expected median_total_tokens in rankings when tokens weight is active")
	}

	var noTokens bytes.Buffer
	if err := WriteJSON(&noTokens, sr, DefaultWeights()); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(noTokens.String(), "median_total_tokens") {
		t.Error("default weights must omit median_total_tokens from rankings")
	}
}

func TestHTML_TokenMetricGatedByWeight(t *testing.T) {
	sr := tokenSuite()

	var withTokens bytes.Buffer
	if err := WriteHTML(&withTokens, sr, tokenWeights()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(withTokens.String(), "median tokens") {
		t.Error("expected 'median tokens' metric in rankings when tokens weight is active")
	}
	if !strings.Contains(withTokens.String(), "tokens") ||
		!strings.Contains(withTokens.String(), "0.3 tokens") {
		t.Error("expected token weight noted in the composite formula")
	}

	var noTokens bytes.Buffer
	if err := WriteHTML(&noTokens, sr, DefaultWeights()); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(noTokens.String(), "median tokens") {
		t.Error("default weights must not render the 'median tokens' metric")
	}
}
