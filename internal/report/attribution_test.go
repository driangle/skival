package report

import (
	"bytes"
	"strings"
	"testing"
)

func TestWriteMarkdown_Attribution(t *testing.T) {
	// Attribution is stamped whether or not results were persisted to disk.
	for _, resultsDir := range []string{"", "/tmp/results/xyz"} {
		sr := orderingSuite()
		sr.ResultsDir = resultsDir
		var buf bytes.Buffer
		WriteMarkdown(&buf, sr, DefaultWeights())
		if !strings.Contains(buf.String(), attributionMarkdown) {
			t.Errorf("missing attribution mark (ResultsDir=%q):\n%s", resultsDir, buf.String())
		}
	}
}
