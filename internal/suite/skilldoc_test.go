package suite

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// skillVerifyTypeRe matches the documented verifier types in SKILL.md, i.e.
// lines of the form `  - type: <name>` inside the schema examples.
var skillVerifyTypeRe = regexp.MustCompile(`(?m)^\s*-\s*type:\s*(\w+)`)

// findRepoRoot walks up from the working directory until it finds go.mod.
func findRepoRoot(t *testing.T) string {
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

// The skill file documents the verifier schema an agent should emit. If a
// verifier type is added to or removed from the validator (validVerifyTypes)
// without the skill being updated, the skill goes stale and starts teaching
// agents to write suites skival rejects. This test guards that drift in both
// directions using the validator's own set as the source of truth.
func TestSkillDocumentsExactlyTheKnownVerifierTypes(t *testing.T) {
	skillPath := filepath.Join(findRepoRoot(t), "claude-code-plugin", "skills", "skival", "SKILL.md")
	data, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("reading skill file: %v", err)
	}

	documented := map[string]bool{}
	for _, m := range skillVerifyTypeRe.FindAllStringSubmatch(string(data), -1) {
		documented[m[1]] = true
	}

	for vt := range validVerifyTypes {
		if !documented[vt] {
			t.Errorf("verifier type %q is valid but not documented in SKILL.md", vt)
		}
	}
	for d := range documented {
		if !validVerifyTypes[d] {
			t.Errorf("SKILL.md documents verifier type %q that the validator rejects", d)
		}
	}
}
