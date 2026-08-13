package suite

import "testing"

func boolPtr(b bool) *bool { return &b }
func intPtr(i int) *int    { return &i }

func TestResolveCompare_DefaultsEnabledWithCriteria(t *testing.T) {
	c := ResolveCompare(&Compare{Criteria: []string{"clarity"}}, nil, nil)
	if c == nil {
		t.Fatal("expected comparison enabled by default when block present with criteria")
	}
	if len(c.Criteria) != 1 || c.Criteria[0] != "clarity" {
		t.Errorf("criteria not carried through: %v", c.Criteria)
	}
}

func TestResolveCompare_DisabledBlock(t *testing.T) {
	c := ResolveCompare(&Compare{Enabled: boolPtr(false), Criteria: []string{"x"}}, nil, nil)
	if c != nil {
		t.Error("expected nil when block disabled")
	}
}

func TestResolveCompare_NoCriteriaMeansOff(t *testing.T) {
	if ResolveCompare(&Compare{}, nil, nil) != nil {
		t.Error("expected nil when enabled but no criteria")
	}
}

func TestResolveCompare_EvalOverridesSuite(t *testing.T) {
	suiteCmp := &Compare{Criteria: []string{"suite"}, Model: "m1"}
	evalCmp := &Compare{Criteria: []string{"eval"}, Model: "m2"}
	c := ResolveCompare(suiteCmp, evalCmp, nil)
	if c == nil {
		t.Fatal("expected merged config")
	}
	if c.Criteria[0] != "eval" {
		t.Errorf("eval criteria should override suite, got %v", c.Criteria)
	}
	if c.Model != "m2" {
		t.Errorf("eval model should override suite, got %q", c.Model)
	}
}

func TestResolveCompare_EvalCanDisable(t *testing.T) {
	suiteCmp := &Compare{Criteria: []string{"x"}}
	evalCmp := &Compare{Enabled: boolPtr(false)}
	if ResolveCompare(suiteCmp, evalCmp, nil) != nil {
		t.Error("eval should be able to disable suite-level comparison")
	}
}

func TestResolveCompare_OverrideForcesOff(t *testing.T) {
	if ResolveCompare(&Compare{Criteria: []string{"x"}}, nil, boolPtr(false)) != nil {
		t.Error("--no-compare override should force comparison off")
	}
}

func TestResolveCompare_OverrideForcesOn(t *testing.T) {
	// Block disabled in config, but CLI forces it on.
	c := ResolveCompare(&Compare{Enabled: boolPtr(false), Criteria: []string{"x"}}, nil, boolPtr(true))
	if c == nil {
		t.Error("--compare override should force comparison on when criteria exist")
	}
}

func TestResolveCompare_OverrideOnWithoutCriteria(t *testing.T) {
	if ResolveCompare(nil, nil, boolPtr(true)) != nil {
		t.Error("--compare with no configured criteria has nothing to run")
	}
}

func TestEffectiveMaxChars(t *testing.T) {
	if got := (&Compare{}).EffectiveMaxChars(); got != DefaultCompareMaxChars {
		t.Errorf("default max chars = %d, want %d", got, DefaultCompareMaxChars)
	}
	if got := (&Compare{MaxChars: intPtr(50)}).EffectiveMaxChars(); got != 50 {
		t.Errorf("max chars = %d, want 50", got)
	}
	if got := (&Compare{MaxChars: intPtr(-1)}).EffectiveMaxChars(); got != -1 {
		t.Errorf("negative max chars (no truncation) = %d, want -1", got)
	}
}

func TestCompareActive(t *testing.T) {
	s := &Suite{
		Compare: &Compare{Criteria: []string{"x"}},
		Evals:   []Eval{{ID: "e1"}},
	}
	if !s.CompareActive(nil) {
		t.Error("expected active with suite-level compare")
	}
	if s.CompareActive(boolPtr(false)) {
		t.Error("--no-compare should make it inactive")
	}

	// Eval-level only.
	s2 := &Suite{Evals: []Eval{{ID: "e1", Compare: &Compare{Criteria: []string{"y"}}}}}
	if !s2.CompareActive(nil) {
		t.Error("expected active with eval-level compare")
	}
}

func TestCompareWeight(t *testing.T) {
	if got := (&Suite{}).CompareWeight(); got != DefaultCompareWeight {
		t.Errorf("default weight = %g, want %g", got, DefaultCompareWeight)
	}
	w := 0.4
	if got := (&Suite{Compare: &Compare{Weight: &w}}).CompareWeight(); got != 0.4 {
		t.Errorf("configured weight = %g, want 0.4", got)
	}
}

func TestValidateRankingWeights_IncludesQuality(t *testing.T) {
	ok := &Ranking{Weights: RankingWeights{Correctness: 0.5, Cost: 0.2, Duration: 0.1, Quality: 0.2}}
	if errs := validateRankingWeights(ok); len(errs) != 0 {
		t.Errorf("valid weights with quality rejected: %v", errs)
	}
	badSum := &Ranking{Weights: RankingWeights{Correctness: 0.5, Cost: 0.2, Duration: 0.1, Quality: 0.5}}
	if errs := validateRankingWeights(badSum); len(errs) == 0 {
		t.Error("weights summing to 1.3 should be rejected")
	}
	neg := &Ranking{Weights: RankingWeights{Correctness: 1.1, Cost: 0, Duration: 0, Quality: -0.1}}
	if errs := validateRankingWeights(neg); len(errs) == 0 {
		t.Error("negative quality weight should be rejected")
	}
}

func TestValidateRankingWeights_IncludesTokens(t *testing.T) {
	// A suite that ranks on tokens instead of cost: cost 0, tokens carries the
	// economic weight, and the five weights still sum to 1.0.
	ok := &Ranking{Weights: RankingWeights{Correctness: 0.5, Cost: 0, Duration: 0.2, Tokens: 0.3}}
	if errs := validateRankingWeights(ok); len(errs) != 0 {
		t.Errorf("valid weights with tokens rejected: %v", errs)
	}
	badSum := &Ranking{Weights: RankingWeights{Correctness: 0.5, Cost: 0.28, Duration: 0.12, Tokens: 0.3}}
	if errs := validateRankingWeights(badSum); len(errs) == 0 {
		t.Error("weights summing to 1.2 should be rejected")
	}
	neg := &Ranking{Weights: RankingWeights{Correctness: 1.1, Cost: 0, Duration: 0, Tokens: -0.1}}
	if errs := validateRankingWeights(neg); len(errs) == 0 {
		t.Error("negative tokens weight should be rejected")
	}
}

func TestValidateCompare(t *testing.T) {
	if errs := validateCompare(nil, "compare"); len(errs) != 0 {
		t.Errorf("nil compare should be valid, got %v", errs)
	}
	if errs := validateCompare(&Compare{}, "compare"); len(errs) == 0 {
		t.Error("enabled compare without criteria should be invalid")
	}
	if errs := validateCompare(&Compare{Enabled: boolPtr(false)}, "compare"); len(errs) != 0 {
		t.Errorf("disabled compare without criteria should be valid, got %v", errs)
	}
	bad := 1.5
	if errs := validateCompare(&Compare{Criteria: []string{"x"}, Weight: &bad}, "compare"); len(errs) == 0 {
		t.Error("out-of-range weight should be invalid")
	}
}
