package suite

// DefaultCompareMaxChars caps how many characters of each variant output are
// passed to the comparative judge, bounding judge context cost. Outputs longer
// than this are truncated before comparison.
const DefaultCompareMaxChars = 4000

// DefaultCompareWeight is the ranking weight given to comparative quality when
// comparison is enabled but the suite does not set explicit ranking weights.
// The base correctness/cost/duration weights are renormalized to make room.
const DefaultCompareWeight = 0.15

// Compare configures comparative judging: an LLM scores the outputs of the
// variants that passed an eval, on a 1-5 quality scale, producing a signal
// that feeds ranking. It is opt-in and never affects suites that omit it.
type Compare struct {
	// Enabled toggles comparison. When the block is present, comparison
	// defaults to on; set enabled: false to define criteria without running it
	// (e.g. a suite-level block disabled for one eval via an override).
	Enabled *bool `yaml:"enabled,omitempty"`
	// Criteria are the qualities the judge weighs when scoring outputs.
	Criteria []string `yaml:"criteria,omitempty"`
	// Model overrides the judge model. Defaults to DefaultJudgeModel.
	Model string `yaml:"model,omitempty"`
	// MaxChars caps per-output length passed to the judge. Defaults to
	// DefaultCompareMaxChars. Set to a negative value to disable truncation.
	MaxChars *int `yaml:"max_chars,omitempty"`
	// Weight is the ranking weight for comparative quality. Only meaningful at
	// suite level. Defaults to DefaultCompareWeight when comparison is enabled.
	Weight *float64 `yaml:"weight,omitempty"`
}

// isEnabled reports whether the block requests comparison to run. A nil block
// is never enabled; a present block defaults to enabled unless set to false.
func (c *Compare) isEnabled() bool {
	if c == nil {
		return false
	}
	if c.Enabled != nil {
		return *c.Enabled
	}
	return true
}

// mergeCompare combines suite-level and eval-level fields into one block, with
// eval-level values overriding suite-level. The result may be disabled; callers
// decide enablement separately. Returns nil only when both inputs are nil.
func mergeCompare(suiteCmp, evalCmp *Compare) *Compare {
	if suiteCmp == nil && evalCmp == nil {
		return nil
	}
	merged := &Compare{}
	if suiteCmp != nil {
		*merged = *suiteCmp
	}
	if evalCmp != nil {
		if evalCmp.Enabled != nil {
			merged.Enabled = evalCmp.Enabled
		}
		if len(evalCmp.Criteria) > 0 {
			merged.Criteria = evalCmp.Criteria
		}
		if evalCmp.Model != "" {
			merged.Model = evalCmp.Model
		}
		if evalCmp.MaxChars != nil {
			merged.MaxChars = evalCmp.MaxChars
		}
	}
	return merged
}

// EffectiveCompare merges suite-level and eval-level comparison config for one
// eval and returns the resolved settings, or nil when comparison is not enabled
// for the eval. Eval-level fields override suite-level fields; either level can
// disable comparison via enabled: false.
func EffectiveCompare(suiteCmp, evalCmp *Compare) *Compare {
	return ResolveCompare(suiteCmp, evalCmp, nil)
}

// ResolveCompare is EffectiveCompare with a CLI override: override=false forces
// comparison off; override=true forces it on wherever criteria are configured
// (overriding enabled: false). A nil override defers to the config. Returns nil
// when comparison should not run for the eval.
func ResolveCompare(suiteCmp, evalCmp *Compare, override *bool) *Compare {
	if override != nil && !*override {
		return nil
	}
	merged := mergeCompare(suiteCmp, evalCmp)
	if merged == nil {
		return nil
	}
	enabled := merged.isEnabled()
	if override != nil {
		enabled = *override
	}
	if !enabled || len(merged.Criteria) == 0 {
		return nil
	}
	return merged
}

// CompareActive reports whether comparison will run for any eval given the CLI
// override. Used to decide default ranking weights.
func (s *Suite) CompareActive(override *bool) bool {
	for i := range s.Evals {
		if ResolveCompare(s.Compare, s.Evals[i].Compare, override) != nil {
			return true
		}
	}
	return false
}

// CompareWeight returns the ranking weight to allocate to comparative quality
// when the suite does not set explicit ranking weights. It honors a suite-level
// compare.weight override, otherwise DefaultCompareWeight.
func (s *Suite) CompareWeight() float64 {
	if s.Compare != nil && s.Compare.Weight != nil {
		return *s.Compare.Weight
	}
	return DefaultCompareWeight
}

// EffectiveMaxChars returns the per-output truncation cap, applying the default
// when unset. A negative configured value means "do not truncate".
func (c *Compare) EffectiveMaxChars() int {
	if c == nil || c.MaxChars == nil {
		return DefaultCompareMaxChars
	}
	return *c.MaxChars
}
