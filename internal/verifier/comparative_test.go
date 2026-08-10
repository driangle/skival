package verifier

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"testing"

	agentrunner "github.com/driangle/agentrunner/go"
)

// scriptedRunner returns text computed from the prompt, letting tests react to
// the anonymized/shuffled outputs the comparative judge builds.
type scriptedRunner struct {
	respond        func(prompt string) string
	err            error
	capturedPrompt string
	capturedOpts   agentrunner.Options
}

func (s *scriptedRunner) Run(_ context.Context, _ string, _ ...agentrunner.Option) (*agentrunner.Result, error) {
	return nil, errors.New("not used")
}

func (s *scriptedRunner) Start(_ context.Context, prompt string, opts ...agentrunner.Option) (*agentrunner.Session, error) {
	s.capturedPrompt = prompt
	for _, o := range opts {
		o(&s.capturedOpts)
	}
	if s.err != nil {
		return nil, s.err
	}
	text := s.respond(prompt)
	ctx, cancel := context.WithCancel(context.Background())
	result := &agentrunner.Result{Text: text}
	return agentrunner.NewSession(ctx, cancel, func(_ context.Context, messages chan<- agentrunner.Message) (*agentrunner.Result, error) {
		messages <- agentrunner.Message{Raw: json.RawMessage(`{"type":"assistant"}`)}
		return result, nil
	}), nil
}

var outputBlockRe = regexp.MustCompile(`(?s)### Output (\S+)\n(.*?)(?:\n\n|$)`)

// ratingByMarker builds a judge response that rates each labeled output by a
// marker embedded in its content: "BEST" -> 5, "WORST" -> 1, else 3. This lets
// tests verify labels map back to the right variants after shuffling.
func ratingByMarker(prompt string) string {
	var entries []string
	for _, m := range outputBlockRe.FindAllStringSubmatch(prompt, -1) {
		label, content := m[1], m[2]
		rating := 3
		switch {
		case strings.Contains(content, "BEST"):
			rating = 5
		case strings.Contains(content, "WORST"):
			rating = 1
		}
		entries = append(entries, fmt.Sprintf(`{"label":%q,"rating":%d,"reason":"because"}`, label, rating))
	}
	return `{"scores":[` + strings.Join(entries, ",") + `]}`
}

func TestComparativeJudge_ScoresMapToVariants(t *testing.T) {
	runner := &scriptedRunner{respond: ratingByMarker}
	j := &ComparativeJudge{Runner: runner, Rand: rand.New(rand.NewSource(1))}

	in := ComparativeInput{
		EvalPrompt: "write a poem",
		Criteria:   []string{"clarity"},
		Variants: []ComparativeVariant{
			{Name: "alpha", Output: "this is the BEST answer"},
			{Name: "bravo", Output: "a middling answer"},
			{Name: "charlie", Output: "this is the WORST answer"},
		},
		MaxChars: 4000,
	}

	res := j.Compare(context.Background(), in)
	if res.Err != nil {
		t.Fatalf("unexpected error: %v", res.Err)
	}
	if len(res.Scores) != 3 {
		t.Fatalf("expected 3 scores, got %d", len(res.Scores))
	}

	want := map[string]struct {
		rating int
		score  float64
	}{
		"alpha":   {5, 1.0},
		"bravo":   {3, 0.6},
		"charlie": {1, 0.2},
	}
	for i, s := range res.Scores {
		// Scores must come back in the caller's original variant order.
		if s.Variant != in.Variants[i].Name {
			t.Errorf("score[%d] variant = %q, want %q (order not preserved)", i, s.Variant, in.Variants[i].Name)
		}
		w := want[s.Variant]
		if s.Rating != w.rating {
			t.Errorf("%s rating = %d, want %d", s.Variant, s.Rating, w.rating)
		}
		if s.Score != w.score {
			t.Errorf("%s score = %g, want %g", s.Variant, s.Score, w.score)
		}
	}
}

func TestComparativeJudge_AnonymizesVariantNames(t *testing.T) {
	runner := &scriptedRunner{respond: ratingByMarker}
	j := &ComparativeJudge{Runner: runner, Rand: rand.New(rand.NewSource(2))}

	in := ComparativeInput{
		EvalPrompt: "task",
		Criteria:   []string{"quality"},
		Variants: []ComparativeVariant{
			{Name: "secret-variant-name", Output: "answer one"},
			{Name: "another-name", Output: "answer two"},
		},
	}
	j.Compare(context.Background(), in)

	if strings.Contains(runner.capturedPrompt, "secret-variant-name") {
		t.Error("prompt leaked variant name to the judge")
	}
	if strings.Contains(runner.capturedPrompt, "another-name") {
		t.Error("prompt leaked variant name to the judge")
	}
}

func TestComparativeJudge_TooFewVariants(t *testing.T) {
	j := &ComparativeJudge{Runner: &scriptedRunner{respond: ratingByMarker}}
	res := j.Compare(context.Background(), ComparativeInput{
		Variants: []ComparativeVariant{{Name: "only", Output: "x"}},
	})
	if res.Err == nil {
		t.Error("expected error for fewer than 2 variants")
	}
	if len(res.Scores) != 0 {
		t.Error("expected no scores on error")
	}
}

func TestComparativeJudge_RunnerError(t *testing.T) {
	j := &ComparativeJudge{Runner: &scriptedRunner{err: errors.New("boom")}}
	res := j.Compare(context.Background(), ComparativeInput{
		Variants: []ComparativeVariant{{Name: "a", Output: "x"}, {Name: "b", Output: "y"}},
	})
	if res.Err == nil {
		t.Error("expected error when runner fails")
	}
	if len(res.Scores) != 0 {
		t.Error("expected no scores on runner error")
	}
}

func TestComparativeJudge_UnparseableResponse(t *testing.T) {
	j := &ComparativeJudge{Runner: &scriptedRunner{respond: func(string) string { return "I liked them all equally" }}}
	res := j.Compare(context.Background(), ComparativeInput{
		Variants: []ComparativeVariant{{Name: "a", Output: "x"}, {Name: "b", Output: "y"}},
	})
	if res.Err == nil {
		t.Error("expected error for unparseable response")
	}
}

func TestComparativeJudge_MissingLabel(t *testing.T) {
	// Judge only scores one of two outputs.
	j := &ComparativeJudge{Runner: &scriptedRunner{respond: func(string) string {
		return `{"scores":[{"label":"A","rating":4,"reason":"ok"}]}`
	}}}
	res := j.Compare(context.Background(), ComparativeInput{
		Variants: []ComparativeVariant{{Name: "a", Output: "x"}, {Name: "b", Output: "y"}},
		MaxChars: 4000,
	})
	if res.Err == nil {
		t.Error("expected error when a variant score is missing")
	}
}

func TestComparativeJudge_OutOfRangeRating(t *testing.T) {
	j := &ComparativeJudge{Runner: &scriptedRunner{respond: func(prompt string) string {
		var entries []string
		for _, m := range outputBlockRe.FindAllStringSubmatch(prompt, -1) {
			entries = append(entries, fmt.Sprintf(`{"label":%q,"rating":7,"reason":"x"}`, m[1]))
		}
		return `{"scores":[` + strings.Join(entries, ",") + `]}`
	}}}
	res := j.Compare(context.Background(), ComparativeInput{
		Variants: []ComparativeVariant{{Name: "a", Output: "x"}, {Name: "b", Output: "y"}},
		MaxChars: 4000,
	})
	if res.Err == nil {
		t.Error("expected error for out-of-range rating")
	}
}

func TestComparativeJudge_TruncatesLongOutputs(t *testing.T) {
	runner := &scriptedRunner{respond: ratingByMarker}
	j := &ComparativeJudge{Runner: runner, Rand: rand.New(rand.NewSource(3))}

	long := strings.Repeat("x", 5000)
	j.Compare(context.Background(), ComparativeInput{
		EvalPrompt: "t",
		Criteria:   []string{"c"},
		Variants: []ComparativeVariant{
			{Name: "a", Output: long},
			{Name: "b", Output: "short"},
		},
		MaxChars: 100,
	})
	if strings.Contains(runner.capturedPrompt, strings.Repeat("x", 200)) {
		t.Error("expected long output to be truncated in the prompt")
	}
	if !strings.Contains(runner.capturedPrompt, "...") {
		t.Error("expected truncation marker in the prompt")
	}
}

func TestComparativeJudge_UsesConfiguredModel(t *testing.T) {
	runner := &scriptedRunner{respond: ratingByMarker}
	j := &ComparativeJudge{Runner: runner, Model: "claude-sonnet-5", Rand: rand.New(rand.NewSource(4))}
	j.Compare(context.Background(), ComparativeInput{
		Variants: []ComparativeVariant{{Name: "a", Output: "x"}, {Name: "b", Output: "y"}},
		MaxChars: 4000,
	})
	if runner.capturedOpts.Model != "claude-sonnet-5" {
		t.Errorf("model = %q, want claude-sonnet-5", runner.capturedOpts.Model)
	}
}

func TestComparativeJudge_DefaultModel(t *testing.T) {
	runner := &scriptedRunner{respond: ratingByMarker}
	j := &ComparativeJudge{Runner: runner, Rand: rand.New(rand.NewSource(5))}
	j.Compare(context.Background(), ComparativeInput{
		Variants: []ComparativeVariant{{Name: "a", Output: "x"}, {Name: "b", Output: "y"}},
		MaxChars: 4000,
	})
	if runner.capturedOpts.Model != DefaultJudgeModel {
		t.Errorf("model = %q, want default %q", runner.capturedOpts.Model, DefaultJudgeModel)
	}
}

func TestExtractJSONObject(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{`{"a":1}`, `{"a":1}`},
		{"prose before {\"a\":1} and after", `{"a":1}`},
		{"```json\n{\"a\":1}\n```", `{"a":1}`},
		{"no json here", ""},
	}
	for _, c := range cases {
		if got := extractJSONObject(c.in); got != c.want {
			t.Errorf("extractJSONObject(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestComparativeLabel(t *testing.T) {
	if comparativeLabel(0) != "A" {
		t.Errorf("label(0) = %q, want A", comparativeLabel(0))
	}
	if comparativeLabel(25) != "Z" {
		t.Errorf("label(25) = %q, want Z", comparativeLabel(25))
	}
	if comparativeLabel(26) != "V27" {
		t.Errorf("label(26) = %q, want V27", comparativeLabel(26))
	}
}
