package suite

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Load reads a suite YAML file, resolves file references, merges defaults, and validates.
func Load(path string) (*Suite, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading suite file: %w", err)
	}

	var s Suite
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parsing suite YAML: %w", err)
	}

	suiteDir := filepath.Dir(path)
	if err := resolveFileRefs(&s, suiteDir); err != nil {
		return nil, err
	}

	mergeDefaults(&s)

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

// mergeDefaults applies suite-level defaults to each eval where the eval
// doesn't provide its own value.
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
	}
}
