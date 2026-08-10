package suite

// mergeDefaults applies suite-level defaults to each eval where the eval
// doesn't provide its own value. RunnerConfig is deep-merged so eval keys
// override default keys while preserving defaults the eval didn't set.
func mergeDefaults(s *Suite) {
	d := s.Defaults
	for i := range s.Evals {
		e := &s.Evals[i]
		if e.Samples == nil && d.Samples != nil {
			e.Samples = d.Samples
		}
		if e.Timeout == nil && d.Timeout != nil {
			e.Timeout = d.Timeout
		}
		if e.Parallel == nil && d.Parallel != nil {
			e.Parallel = d.Parallel
		}
		if e.Model == "" && d.Model != "" {
			e.Model = d.Model
		}
		if e.Runner == "" && d.Runner != "" {
			e.Runner = d.Runner
		}
		applyDefaultJudgeModel(e, d.JudgeModel)
		if e.Isolate == nil {
			t := true
			e.Isolate = &t
		}
		e.RunnerConfig = mergeMaps(d.RunnerConfig, e.RunnerConfig)
		if e.Retry == nil && d.Retry != nil {
			e.Retry = d.Retry
		}
	}

	// Propagate the default judge model to the suite-level compare block too.
	if d.JudgeModel != "" && s.Compare != nil && s.Compare.Model == "" {
		s.Compare.Model = d.JudgeModel
	}
}

// applyDefaultJudgeModel fills in the default judge model on an eval's judge
// verify steps and comparison block where they don't set their own.
func applyDefaultJudgeModel(e *Eval, judgeModel string) {
	if judgeModel == "" {
		return
	}
	for j := range e.Verify {
		if e.Verify[j].Type == "judge" && e.Verify[j].Model == "" {
			e.Verify[j].Model = judgeModel
		}
	}
	if e.Compare != nil && e.Compare.Model == "" {
		e.Compare.Model = judgeModel
	}
}

// resolveRunnerConfig propagates Runner and deep-merges RunnerConfig from each
// eval into its variants. Variant values take precedence over eval values.
func resolveRunnerConfig(s *Suite) {
	for i := range s.Evals {
		e := &s.Evals[i]
		for j := range e.Variants {
			mergeRunnerIntoVariant(e, &e.Variants[j])
		}
	}
}

func mergeRunnerIntoVariant(e *Eval, v *Variant) {
	if v.Runner == "" && e.Runner != "" {
		v.Runner = e.Runner
	}
	if v.Model == "" && e.Model != "" {
		v.Model = e.Model
	}
	v.RunnerConfig = mergeMaps(e.RunnerConfig, v.RunnerConfig)
	if v.Retry == nil && e.Retry != nil {
		v.Retry = e.Retry
	}
}

// mergeMaps deep-merges two maps. Keys in override take precedence over base.
// Returns nil if both inputs are nil.
func mergeMaps(base, override map[string]any) map[string]any {
	if base == nil {
		return override
	}
	if override == nil {
		// Copy base so callers don't share the same map.
		out := make(map[string]any, len(base))
		for k, v := range base {
			out[k] = v
		}
		return out
	}
	out := make(map[string]any, len(base)+len(override))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range override {
		if vMap, ok := v.(map[string]any); ok {
			if baseMap, ok := out[k].(map[string]any); ok {
				out[k] = mergeMaps(baseMap, vMap)
				continue
			}
		}
		out[k] = v
	}
	return out
}
