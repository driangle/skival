package suite

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// decodeStrict unmarshals YAML into v with unknown fields rejected. Unlike
// yaml.Unmarshal, a key that maps to no struct field (a typo like `varaints:`
// or an unsupported block like `treatments:`) produces a loud error at load
// time instead of being silently dropped.
func decodeStrict(data []byte, v any) error {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	return dec.Decode(v)
}

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
	if err := decodeStrict(data, &s); err != nil {
		return nil, fmt.Errorf("parsing suite YAML: %w", err)
	}

	if err := transform(&s, filepath.Dir(absPath)); err != nil {
		return nil, err
	}

	if err := validate(&s); err != nil {
		return nil, err
	}

	warnModelRunnerCompat(&s)

	return &s, nil
}

// transform resolves references and normalizes a freshly-parsed suite in place:
// eval file refs, matrix expansion, prompt files, path resolution, deprecated-
// field migrations, and default merging. suiteDir anchors relative paths.
func transform(s *Suite, suiteDir string) error {
	baseDirs, err := resolveFileRefs(s, suiteDir)
	if err != nil {
		return err
	}
	if err := validateMatrixExclusive(s); err != nil {
		return err
	}
	expandMatrices(s)
	if err := resolvePromptFiles(s, baseDirs); err != nil {
		return err
	}
	resolvePaths(s, suiteDir)
	mergeDefaults(s)
	resolveRunnerConfig(s)
	return nil
}

// resolveFileRefs replaces eval entries that have a `file:` field with the
// contents of the referenced YAML file. It returns, per eval, the base
// directory that eval-local references (such as prompt_file) resolve against:
// the eval file's directory for file-referenced evals, else suiteDir.
func resolveFileRefs(s *Suite, suiteDir string) ([]string, error) {
	baseDirs := make([]string, len(s.Evals))
	for i, eval := range s.Evals {
		baseDirs[i] = suiteDir
		if eval.File == "" {
			continue
		}

		refPath := eval.File
		if !filepath.IsAbs(refPath) {
			refPath = filepath.Join(suiteDir, refPath)
		}

		data, err := os.ReadFile(refPath)
		if err != nil {
			return nil, fmt.Errorf("reading eval file reference %q: %w", eval.File, err)
		}

		var resolved Eval
		if err := decodeStrict(data, &resolved); err != nil {
			return nil, fmt.Errorf("parsing eval file %q (eval index %d): %w", eval.File, i, err)
		}

		resolved.File = ""
		s.Evals[i] = resolved
		baseDirs[i] = filepath.Dir(refPath)
	}
	return baseDirs, nil
}

// resolvePaths makes relative paths in the suite absolute, anchored to suiteDir.
// This ensures the suite works regardless of the process working directory.
func resolvePaths(s *Suite, suiteDir string) {
	for i := range s.Evals {
		e := &s.Evals[i]
		e.SuiteDir = suiteDir

		if e.Dir == "" {
			e.Dir = suiteDir
		} else if !filepath.IsAbs(e.Dir) {
			e.Dir = filepath.Join(suiteDir, e.Dir)
		}

		for j := range e.Verify {
			step := &e.Verify[j]
			switch step.Type {
			case "check_output":
				if step.Run != "" && !filepath.IsAbs(step.Run) && !hasPathVar(step.Run) {
					step.Run = filepath.Join(suiteDir, step.Run)
				}
			case "file_contains":
				// file_contains paths are resolved at runtime against the workdir,
				// not at load time against the suite dir.
			}
		}

		for j := range e.Variants {
			resolveVariantPaths(&e.Variants[j], suiteDir)
		}
	}
}

// hasPathVar reports whether a path references a ${SKIVAL_...} substitution
// variable. Such paths are left raw at load time so they can be expanded at
// pipeline-build time against the suite and working directories.
func hasPathVar(raw string) bool {
	return strings.Contains(raw, "${SKIVAL_")
}

func resolveVariantPaths(v *Variant, suiteDir string) {
	if v.Skill != "" && !filepath.IsAbs(v.Skill) {
		v.Skill = filepath.Join(suiteDir, v.Skill)
	}
	for i, s := range v.Skills {
		if s != "" && !filepath.IsAbs(s) {
			v.Skills[i] = filepath.Join(suiteDir, s)
		}
	}
	if v.Dir != "" && !filepath.IsAbs(v.Dir) {
		v.Dir = filepath.Join(suiteDir, v.Dir)
	}
	if v.ConfigDir != "" && !filepath.IsAbs(v.ConfigDir) {
		v.ConfigDir = filepath.Join(suiteDir, v.ConfigDir)
	}
}
