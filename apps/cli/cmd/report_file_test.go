package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/driangle/skival/internal/report"
	"github.com/driangle/skival/internal/result"
)

// TestWriteReportFile verifies the helper that drops report.html into the
// results dir (so its relative session links resolve) actually writes valid HTML.
func TestWriteReportFile(t *testing.T) {
	sr := &result.SuiteResult{StartedAt: time.Now(), FinishedAt: time.Now()}
	path := filepath.Join(t.TempDir(), "report.html")

	if err := writeReportFile(path, sr, report.DefaultWeights()); err != nil {
		t.Fatalf("writeReportFile: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading report: %v", err)
	}
	if !strings.Contains(string(data), "<!DOCTYPE html>") {
		t.Errorf("report.html does not look like HTML: %.40q", string(data))
	}
}
