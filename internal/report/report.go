package report

import (
	"fmt"
	"io"

	"github.com/driangle/skival/internal/result"
)

// Write writes a report in the specified format to w.
// Supported formats: "markdown", "json".
func Write(w io.Writer, sr *result.SuiteResult, format string, weights Weights) error {
	switch format {
	case "markdown":
		WriteMarkdown(w, sr, weights)
		return nil
	case "json":
		return WriteJSON(w, sr, weights)
	case "html":
		return WriteHTML(w, sr, weights)
	default:
		return fmt.Errorf("unsupported format: %q (use \"markdown\", \"json\", or \"html\")", format)
	}
}
