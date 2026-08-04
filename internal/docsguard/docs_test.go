// Package docsguard holds tests that keep the project documentation in sync
// with the suite schema. It has no runtime code — the tests read README.md and
// the published docs and assert their YAML examples still load and validate.
package docsguard

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/driangle/skival/internal/suite"
)

// docFiles are the user-facing docs (README + the published VitePress pages).
// PLAN.md and specs/** are excluded from the site via srcExclude and are not
// checked here.
var docFiles = []string{
	"README.md",
	"docs/index.md",
	"docs/getting-started.md",
	"docs/configuration.md",
	"docs/cli.md",
	"docs/verifiers.md",
	"docs/examples.md",
}

// repoRoot walks up from this test file to the directory containing go.mod.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine test file path")
	}
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not locate repo root (go.mod not found)")
		}
		dir = parent
	}
}

// forbiddenLine matches removed schema tokens that must not reappear in the
// docs after the migration to the flat variants schema.
var forbiddenLine = []struct {
	name string
	re   *regexp.Regexp
}{
	{"treatments:", regexp.MustCompile(`^[ \t-]*treatments:`)},
	{"control:", regexp.MustCompile(`^[ \t-]*control:`)},
	{"variations:", regexp.MustCompile(`^[ \t-]*variations:`)},
	{"--treatments", regexp.MustCompile(`--treatments\b`)},
}

// TestDocsHaveNoRemovedSchemaTokens fails if any doc reintroduces the removed
// treatments/control/variations schema or the --treatments flag.
func TestDocsHaveNoRemovedSchemaTokens(t *testing.T) {
	root := repoRoot(t)
	for _, rel := range docFiles {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("reading %s: %v", rel, err)
		}
		for i, line := range strings.Split(string(data), "\n") {
			for _, f := range forbiddenLine {
				if f.re.MatchString(line) {
					t.Errorf("%s:%d uses removed token %q; use the flat variants schema instead:\n  %s",
						rel, i+1, f.name, strings.TrimSpace(line))
				}
			}
		}
	}
}

// TestDocSuitesValidate loads every complete suite YAML example in the docs and
// requires it to pass the suite loader + validator, so copy-pasteable examples
// can't drift from the schema.
func TestDocSuitesValidate(t *testing.T) {
	root := repoRoot(t)
	checked := 0
	for _, rel := range docFiles {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("reading %s: %v", rel, err)
		}
		for j, block := range yamlBlocks(string(data)) {
			reason, ok := skipReason(block)
			if !ok {
				t.Logf("%s: skipping yaml block #%d (%s)", rel, j+1, reason)
				continue
			}
			dir := t.TempDir()
			path := filepath.Join(dir, "suite.yaml")
			if err := os.WriteFile(path, []byte(block), 0o644); err != nil {
				t.Fatalf("writing temp suite: %v", err)
			}
			if _, err := suite.Load(path); err != nil {
				t.Errorf("%s: yaml block #%d does not load/validate:\n%v\n--- block ---\n%s",
					rel, j+1, err, block)
				continue
			}
			checked++
		}
	}
	if checked == 0 {
		t.Fatal("no full-suite doc examples were validated; extraction likely broke")
	}
	t.Logf("validated %d full-suite doc examples", checked)
}

// yamlBlocks returns the contents of every ```yaml fenced code block.
func yamlBlocks(content string) []string {
	var blocks []string
	var cur []string
	inBlock := false
	for _, ln := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(ln)
		if !inBlock {
			if trimmed == "```yaml" {
				inBlock = true
				cur = nil
			}
			continue
		}
		if trimmed == "```" {
			blocks = append(blocks, strings.Join(cur, "\n"))
			inBlock = false
			continue
		}
		cur = append(cur, ln)
	}
	return blocks
}

var (
	reTopVersion = regexp.MustCompile(`(?m)^version:`)
	reTopEvals   = regexp.MustCompile(`(?m)^evals:`)
)

// skipReason decides whether a YAML block is a complete, self-contained suite
// worth loading. Fragments and blocks with external filesystem dependencies
// (file refs, config_dir) are skipped with an explanation rather than failed.
func skipReason(block string) (reason string, checkable bool) {
	switch {
	case !reTopVersion.MatchString(block) || !reTopEvals.MatchString(block):
		return "partial fragment (no top-level version/evals)", false
	case !strings.Contains(block, "variants:") && !strings.Contains(block, "matrix:"):
		return "no variants or matrix", false
	case strings.Contains(block, "file:"):
		return "references external files", false
	case strings.Contains(block, "config_dir:"):
		return "references on-disk config_dir", false
	}
	return "", true
}
