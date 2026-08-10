package suite

import "log"

// migrateStateToProbes converts deprecated state assertions to http probes
// and logs a deprecation warning.
func migrateStateToProbes(s *Suite) {
	for i := range s.Evals {
		e := &s.Evals[i]
		if len(e.Correctness.State) == 0 {
			continue
		}
		log.Printf("WARNING: eval %q uses deprecated state field; use correctness.probes instead", e.ID)
		for _, sa := range e.Correctness.State {
			probe := Probe{
				HTTP: &HTTPProbe{
					URL:    sa.URL,
					Method: sa.Method,
					Assert: HTTPProbeAssert{
						BodyContains: sa.Expect,
					},
				},
			}
			e.Correctness.Probes = append(e.Correctness.Probes, probe)
		}
		e.Correctness.State = nil
	}
}

// migrateCorrectnessToVerify converts deprecated correctness config to verify steps
// and logs a deprecation warning.
func migrateCorrectnessToVerify(s *Suite) {
	for i := range s.Evals {
		e := &s.Evals[i]
		c := e.Correctness

		if !hasCorrectnessConfig(c) {
			continue
		}

		if len(e.Verify) > 0 {
			log.Printf("WARNING: eval %q has both verify and correctness; correctness is ignored", e.ID)
			e.Correctness = Correctness{}
			continue
		}

		log.Printf("WARNING: eval %q uses deprecated correctness field; use verify instead", e.ID)
		e.Verify = correctnessToVerifySteps(c)
		e.Correctness = Correctness{}
	}
}

// correctnessToVerifySteps translates a deprecated Correctness block into the
// equivalent ordered list of verify steps.
func correctnessToVerifySteps(c Correctness) []VerifyStep {
	var steps []VerifyStep

	if c.AgentExitsOK != nil && *c.AgentExitsOK {
		steps = append(steps, VerifyStep{Type: "agent_exits_ok"})
	}
	if c.Check != "" {
		steps = append(steps, VerifyStep{Type: "check", Run: c.Check})
	}
	if len(c.Output.Contains) > 0 {
		steps = append(steps, VerifyStep{Type: "output_contains", Values: c.Output.Contains})
	}
	steps = append(steps, probesToVerifySteps(c.Probes)...)
	if c.CheckOutput != "" {
		steps = append(steps, VerifyStep{Type: "check_output", Run: c.CheckOutput})
	}
	if len(c.Judge) > 0 {
		step := VerifyStep{Type: "judge", Criteria: c.Judge}
		if c.JudgeModel != "" {
			step.Model = c.JudgeModel
		}
		steps = append(steps, step)
	}
	return steps
}

// probesToVerifySteps translates deprecated probes into verify steps.
func probesToVerifySteps(probes []Probe) []VerifyStep {
	var steps []VerifyStep
	for _, p := range probes {
		switch {
		case p.HTTP != nil:
			steps = append(steps, VerifyStep{
				Type:         "http_check",
				URL:          p.HTTP.URL,
				Method:       p.HTTP.Method,
				Status:       p.HTTP.Assert.Status,
				BodyContains: p.HTTP.Assert.BodyContains,
			})
		case p.File != nil:
			steps = append(steps, VerifyStep{
				Type:     "file_contains",
				Path:     p.File.Path,
				Exists:   p.File.Assert.Exists,
				Contains: p.File.Assert.Contains,
			})
		case p.Command != nil:
			steps = append(steps, VerifyStep{
				Type:           "command",
				Run:            p.Command.Run,
				Exits:          p.Command.Assert.Exits,
				StdoutContains: p.Command.Assert.StdoutContains,
			})
		case p.TCP != nil:
			steps = append(steps, VerifyStep{
				Type: "tcp_check",
				Host: p.TCP.Host,
				Port: p.TCP.Port,
			})
		}
	}
	return steps
}

func hasCorrectnessConfig(c Correctness) bool {
	return c.Check != "" ||
		c.AgentExitsOK != nil ||
		len(c.Output.Contains) > 0 ||
		c.CheckOutput != "" ||
		len(c.State) > 0 ||
		len(c.Probes) > 0 ||
		len(c.Judge) > 0 ||
		c.JudgeModel != ""
}

// migrateAllowedTools moves the deprecated AllowedTools field on variants
// into RunnerConfig["allowed_tools"] and logs a deprecation warning.
func migrateAllowedTools(s *Suite) {
	for i := range s.Evals {
		for j := range s.Evals[i].Variants {
			migrateVariantAllowedTools(&s.Evals[i].Variants[j])
		}
	}
}

func migrateVariantAllowedTools(v *Variant) {
	if len(v.AllowedTools) == 0 {
		return
	}
	log.Printf("WARNING: variant %q uses deprecated allowed_tools field; use runner_config.allowed_tools instead", v.Name)
	if v.RunnerConfig == nil {
		v.RunnerConfig = make(map[string]any)
	}
	if _, ok := v.RunnerConfig["allowed_tools"]; !ok {
		v.RunnerConfig["allowed_tools"] = v.AllowedTools
	}
	v.AllowedTools = nil
}
