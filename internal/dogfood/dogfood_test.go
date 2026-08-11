package dogfood

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/driangle/skival/internal/suite"
	"gopkg.in/yaml.v3"
)

// Paths are relative to the repository root, located via repoRoot.
const (
	skillRelPath = "claude-code-plugin/skills/skival/SKILL.md"
	suiteRelPath = "evals/suite.yaml"
)

// repoRoot walks up from the test's working directory until it finds go.mod.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not locate repo root (no go.mod found walking up)")
		}
		dir = parent
	}
}

type skillFrontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// parseFrontmatter extracts the leading `---`-delimited YAML block of a skill.
func parseFrontmatter(t *testing.T, content string) skillFrontmatter {
	t.Helper()
	if !strings.HasPrefix(content, "---") {
		t.Fatal("SKILL.md does not start with a `---` frontmatter block")
	}
	rest := content[len("---"):]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		t.Fatal("SKILL.md frontmatter block is not terminated by `---`")
	}
	var fm skillFrontmatter
	if err := yaml.Unmarshal([]byte(rest[:end]), &fm); err != nil {
		t.Fatalf("parsing SKILL.md frontmatter: %v", err)
	}
	return fm
}

// The skill file must exist and carry the frontmatter Claude Code needs to
// discover and trigger it. This catches an accidental rename, deletion, or a
// malformed header before the skill ever reaches an agent.
func TestSkillFileIsWellFormed(t *testing.T) {
	path := filepath.Join(repoRoot(t), skillRelPath)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading skill file %s: %v", skillRelPath, err)
	}
	fm := parseFrontmatter(t, string(data))
	if fm.Name != "skival" {
		t.Errorf("skill name = %q, want %q", fm.Name, "skival")
	}
	if strings.TrimSpace(fm.Description) == "" {
		t.Error("skill description is empty; the description drives skill triggering")
	}
}

// The canonical dogfood suite must load and validate against the current schema
// using the same loader the CLI uses. If the schema evolves in a way that breaks
// this suite, this test fails in the normal `go test` path — no LLM required.
func TestDogfoodSuiteLoadsAndValidates(t *testing.T) {
	root := repoRoot(t)
	s, err := suite.Load(filepath.Join(root, suiteRelPath))
	if err != nil {
		t.Fatalf("loading %s: %v", suiteRelPath, err)
	}
	if len(s.Evals) == 0 {
		t.Fatal("dogfood suite has no evals")
	}
	skillAbs := filepath.Join(root, skillRelPath)
	for _, eval := range s.Evals {
		assertBaselineVsSkill(t, eval, skillAbs)
	}
}

// assertBaselineVsSkill checks an eval compares a no-skill baseline against a
// variant injecting the real skill file, and that the referenced skill exists.
func assertBaselineVsSkill(t *testing.T, eval suite.Eval, skillAbs string) {
	t.Helper()
	var baseline, withSkill *suite.Variant
	for i := range eval.Variants {
		switch eval.Variants[i].Name {
		case "baseline":
			baseline = &eval.Variants[i]
		case "with-skill":
			withSkill = &eval.Variants[i]
		}
	}
	if baseline == nil || withSkill == nil {
		t.Fatalf("eval %q must have a 'baseline' and a 'with-skill' variant", eval.ID)
	}
	if baseline.Skill != "" || len(baseline.Skills) > 0 {
		t.Errorf("eval %q baseline must inject no skill, got %q", eval.ID, baseline.Skill)
	}
	if filepath.Clean(withSkill.Skill) != filepath.Clean(skillAbs) {
		t.Errorf("eval %q with-skill points at %q, want the real skill %q",
			eval.ID, withSkill.Skill, skillAbs)
	}
	if _, err := os.Stat(withSkill.Skill); err != nil {
		t.Errorf("eval %q injects a skill that does not exist: %v", eval.ID, err)
	}
}
