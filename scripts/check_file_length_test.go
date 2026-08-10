package scripts

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// runChecker runs check-file-length.sh against root with the given per-path
// budgets and returns combined output plus whether it exited zero.
func runChecker(t *testing.T, root string, maxNonTest, maxTest int) (string, bool) {
	t.Helper()
	cmd := exec.Command("bash", "check-file-length.sh", root)
	cmd.Env = append(cmd.Environ(),
		"MAX_NONTEST="+strconv.Itoa(maxNonTest),
		"MAX_TEST="+strconv.Itoa(maxTest),
	)
	out, err := cmd.CombinedOutput()
	return string(out), err == nil
}

// writeGo writes a Go fixture file with exactly the given number of lines.
func writeGo(t *testing.T, path string, lines int) {
	t.Helper()
	var b strings.Builder
	b.WriteString("package fixture\n")
	for i := 2; i <= lines; i++ {
		b.WriteString("// filler line\n")
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestCheckFileLength_OverLimitNonTestFails(t *testing.T) {
	dir := t.TempDir()
	writeGo(t, filepath.Join(dir, "big.go"), 301)

	out, ok := runChecker(t, dir, 300, 500)
	if ok {
		t.Fatalf("expected failure for 301-line non-test file, got success:\n%s", out)
	}
	if !strings.Contains(out, "big.go") || !strings.Contains(out, "301 > 300") {
		t.Fatalf("expected violation for big.go (301 > 300), got:\n%s", out)
	}
}

func TestCheckFileLength_UnderLimitPasses(t *testing.T) {
	dir := t.TempDir()
	writeGo(t, filepath.Join(dir, "small.go"), 300)
	writeGo(t, filepath.Join(dir, "small_test.go"), 500)

	out, ok := runChecker(t, dir, 300, 500)
	if !ok {
		t.Fatalf("expected success for at-limit files, got failure:\n%s", out)
	}
}

func TestCheckFileLength_TestFileGetsLargerBudget(t *testing.T) {
	dir := t.TempDir()
	// 400 lines: over the non-test budget (300) but under the test budget (500).
	writeGo(t, filepath.Join(dir, "wide_test.go"), 400)

	out, ok := runChecker(t, dir, 300, 500)
	if !ok {
		t.Fatalf("expected 400-line test file to pass under test budget, got failure:\n%s", out)
	}
	if strings.Contains(out, "wide_test.go") {
		t.Fatalf("did not expect a violation for wide_test.go, got:\n%s", out)
	}
}

func TestCheckFileLength_OverLimitTestFails(t *testing.T) {
	dir := t.TempDir()
	writeGo(t, filepath.Join(dir, "huge_test.go"), 501)

	out, ok := runChecker(t, dir, 300, 500)
	if ok {
		t.Fatalf("expected failure for 501-line test file, got success:\n%s", out)
	}
	if !strings.Contains(out, "huge_test.go") || !strings.Contains(out, "501 > 500") {
		t.Fatalf("expected violation for huge_test.go (501 > 500), got:\n%s", out)
	}
}
