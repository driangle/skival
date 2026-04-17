package suite

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Load reads a suite YAML file, resolves file references, merges defaults, and validates.
func Load(path string) (*Suite, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolving suite path: %w", err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("reading suite file: %w", err)
	}

	var s Suite
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parsing suite YAML: %w", err)
	}

	suiteDir := filepath.Dir(absPath)
	if err := resolveFileRefs(&s, suiteDir); err != nil {
		return nil, err
	}

	if err := validateMatrixExclusive(&s); err != nil {
		return nil, err
	}
	expandMatrices(&s)
	resolvePaths(&s, suiteDir)
	migrateAllowedTools(&s)
	mergeDefaults(&s)
	resolveRunnerConfig(&s)

	if err := validate(&s); err != nil {
		return nil, err
	}

	return &s, nil
}

// resolveFileRefs replaces eval entries that have a `file:` field with the
// contents of the referenced YAML file. Paths are relative to suiteDir.
func resolveFileRefs(s *Suite, suiteDir string) error {
	for i, eval := range s.Evals {
		if eval.File == "" {
			continue
		}

		refPath := eval.File
		if !filepath.IsAbs(refPath) {
			refPath = filepath.Join(suiteDir, refPath)
		}

		data, err := os.ReadFile(refPath)
		if err != nil {
			return fmt.Errorf("reading eval file reference %q: %w", eval.File, err)
		}

		var resolved Eval
		if err := yaml.Unmarshal(data, &resolved); err != nil {
			return fmt.Errorf("parsing eval file %q: %w", eval.File, err)
		}

		resolved.File = ""
		s.Evals[i] = resolved
	}
	return nil
}

// resolvePaths makes relative paths in the suite absolute, anchored to suiteDir.
// This ensures the suite works regardless of the process working directory.
func resolvePaths(s *Suite, suiteDir string) {
	for i := range s.Evals {
		e := &s.Evals[i]

		if e.Dir == "" {
			e.Dir = suiteDir
		} else if !filepath.IsAbs(e.Dir) {
			e.Dir = filepath.Join(suiteDir, e.Dir)
		}

		if e.Correctness.Script != "" && !filepath.IsAbs(e.Correctness.Script) {
			e.Correctness.Script = filepath.Join(suiteDir, e.Correctness.Script)
		}

		resolveTreatmentPaths(&e.Treatments.Control, suiteDir)
		for j := range e.Treatments.Variations {
			resolveTreatmentPaths(&e.Treatments.Variations[j], suiteDir)
		}
	}
}

func resolveTreatmentPaths(t *Treatment, suiteDir string) {
	if t.Skill != "" && !filepath.IsAbs(t.Skill) {
		t.Skill = filepath.Join(suiteDir, t.Skill)
	}
	if t.Dir != "" && !filepath.IsAbs(t.Dir) {
		t.Dir = filepath.Join(suiteDir, t.Dir)
	}
}

// migrateAllowedTools moves the deprecated AllowedTools field on treatments
// into RunnerConfig["allowed_tools"] and logs a deprecation warning.
func migrateAllowedTools(s *Suite) {
	for i := range s.Evals {
		migrateTreatmentAllowedTools(&s.Evals[i].Treatments.Control)
		for j := range s.Evals[i].Treatments.Variations {
			migrateTreatmentAllowedTools(&s.Evals[i].Treatments.Variations[j])
		}
	}
}

func migrateTreatmentAllowedTools(t *Treatment) {
	if len(t.AllowedTools) == 0 {
		return
	}
	log.Printf("WARNING: treatment %q uses deprecated allowed_tools field; use runner_config.allowed_tools instead", t.Name)
	if t.RunnerConfig == nil {
		t.RunnerConfig = make(map[string]any)
	}
	if _, ok := t.RunnerConfig["allowed_tools"]; !ok {
		t.RunnerConfig["allowed_tools"] = t.AllowedTools
	}
	t.AllowedTools = nil
}

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
		if e.Model == "" && d.Model != "" {
			e.Model = d.Model
		}
		if e.Runner == "" && d.Runner != "" {
			e.Runner = d.Runner
		}
		e.RunnerConfig = mergeMaps(d.RunnerConfig, e.RunnerConfig)
	}
}

// resolveRunnerConfig propagates Runner and deep-merges RunnerConfig from each
// eval into its control and variation treatments. Treatment values take
// precedence over eval values.
func resolveRunnerConfig(s *Suite) {
	for i := range s.Evals {
		e := &s.Evals[i]
		mergeRunnerIntoTreatment(e, &e.Treatments.Control)
		for j := range e.Treatments.Variations {
			mergeRunnerIntoTreatment(e, &e.Treatments.Variations[j])
		}
	}
}

func mergeRunnerIntoTreatment(e *Eval, t *Treatment) {
	if t.Runner == "" && e.Runner != "" {
		t.Runner = e.Runner
	}
	t.RunnerConfig = mergeMaps(e.RunnerConfig, t.RunnerConfig)
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
