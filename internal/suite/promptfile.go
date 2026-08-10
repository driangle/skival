package suite

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// placeholderRe matches {{ name }} template placeholders. The name may be
// surrounded by optional whitespace and consists of letters, digits, and
// underscores.
var placeholderRe = regexp.MustCompile(`{{\s*([A-Za-z0-9_]+)\s*}}`)

// resolvePromptFiles reads prompt_file references on each eval and its variants
// into the corresponding Prompt field, applying {{var}} substitution from the
// merged vars maps. baseDirs[i] is the directory that eval i's relative
// prompt_file paths resolve against (the suite dir, or the eval file's dir when
// the eval was loaded via file:).
func resolvePromptFiles(s *Suite, baseDirs []string) error {
	for i := range s.Evals {
		if err := resolveEvalPromptFile(&s.Evals[i], baseDirs[i]); err != nil {
			return fmt.Errorf("eval[%d] (%s): %w", i, s.Evals[i].ID, err)
		}
	}
	return nil
}

// resolveEvalPromptFile resolves the eval-level template and every variant's
// prompt. When the eval sets prompt_file, its contents become the shared
// fallback template for variants that don't define their own prompt.
func resolveEvalPromptFile(e *Eval, baseDir string) error {
	var evalTemplate string
	if e.PromptFile != "" {
		if e.Prompt != "" {
			return fmt.Errorf("cannot set both prompt and prompt_file")
		}
		raw, err := readPromptFile(e.PromptFile, baseDir)
		if err != nil {
			return err
		}
		evalTemplate = raw
		// Render the eval-level fallback leniently: variant-only vars may fill
		// remaining placeholders, which are validated strictly per variant.
		e.Prompt = substitute(raw, e.Vars)
	}

	for j := range e.Variants {
		if err := resolveVariantPromptFile(&e.Variants[j], e, evalTemplate, baseDir); err != nil {
			return fmt.Errorf("variant[%d] (%s): %w", j, e.Variants[j].Name, err)
		}
	}
	return nil
}

// resolveVariantPromptFile populates a variant's Prompt from whichever template
// applies to it, substituting the eval's vars overlaid by the variant's vars.
func resolveVariantPromptFile(v *Variant, e *Eval, evalTemplate, baseDir string) error {
	raw, ok, err := variantTemplate(v, evalTemplate, baseDir)
	if err != nil {
		return err
	}
	if !ok {
		return nil // no template applies; runtime falls back to eval.Prompt
	}
	rendered, err := renderTemplate(raw, mergeVars(e.Vars, v.Vars))
	if err != nil {
		return err
	}
	v.Prompt = rendered
	return nil
}

// variantTemplate returns the raw template text that applies to a variant and
// whether one applies at all. A variant's own prompt_file wins; an inline
// variant prompt is used verbatim (no templating); otherwise the eval-level
// template (if any) is inherited.
func variantTemplate(v *Variant, evalTemplate, baseDir string) (string, bool, error) {
	if v.PromptFile != "" {
		if v.Prompt != "" {
			return "", false, fmt.Errorf("cannot set both prompt and prompt_file")
		}
		raw, err := readPromptFile(v.PromptFile, baseDir)
		if err != nil {
			return "", false, err
		}
		return raw, true, nil
	}
	if v.Prompt != "" {
		return "", false, nil // inline prompt, no substitution
	}
	if evalTemplate != "" {
		return evalTemplate, true, nil
	}
	return "", false, nil
}

// readPromptFile reads a prompt file, resolving a relative path against baseDir.
func readPromptFile(path, baseDir string) (string, error) {
	resolved := path
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(baseDir, resolved)
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return "", fmt.Errorf("reading prompt_file %q: %w", path, err)
	}
	return string(data), nil
}

// renderTemplate substitutes vars into tmpl. When vars is non-empty it runs in
// strict mode: any placeholder left unresolved is an error, catching typos in
// either the template or the vars map. An empty vars map means no templating is
// intended, so the file is returned verbatim (literal {{...}} is preserved).
func renderTemplate(tmpl string, vars map[string]string) (string, error) {
	if len(vars) == 0 {
		return tmpl, nil
	}
	out := substitute(tmpl, vars)
	if leftover := unresolvedPlaceholders(out); len(leftover) > 0 {
		return "", fmt.Errorf("unresolved placeholder(s): %s", strings.Join(leftover, ", "))
	}
	return out, nil
}

// substitute replaces {{name}} placeholders in tmpl with values from vars.
// Placeholders with no matching var are left untouched.
func substitute(tmpl string, vars map[string]string) string {
	return placeholderRe.ReplaceAllStringFunc(tmpl, func(m string) string {
		if val, ok := vars[placeholderName(m)]; ok {
			return val
		}
		return m
	})
}

// placeholderName extracts the variable name from a {{ name }} match.
func placeholderName(match string) string {
	return placeholderRe.FindStringSubmatch(match)[1]
}

// unresolvedPlaceholders returns the sorted, de-duplicated set of remaining
// {{name}} placeholders in s, formatted for error messages.
func unresolvedPlaceholders(s string) []string {
	seen := make(map[string]bool)
	var names []string
	for _, m := range placeholderRe.FindAllString(s, -1) {
		name := placeholderName(m)
		if !seen[name] {
			seen[name] = true
			names = append(names, "{{"+name+"}}")
		}
	}
	sort.Strings(names)
	return names
}

// mergeVars returns a new map with base entries overlaid by override entries.
// Returns nil when both are empty.
func mergeVars(base, override map[string]string) map[string]string {
	if len(base) == 0 && len(override) == 0 {
		return nil
	}
	out := make(map[string]string, len(base)+len(override))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range override {
		out[k] = v
	}
	return out
}
