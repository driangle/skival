package compare

import (
	"fmt"
	"io"
)

// Write writes a comparison report in the specified format to w.
// Supported formats: "markdown", "json".
func Write(w io.Writer, c *Comparison, format string) error {
	switch format {
	case "markdown":
		WriteMarkdown(w, c)
		return nil
	case "json":
		return WriteJSON(w, c)
	default:
		return fmt.Errorf("unsupported format: %q (use \"markdown\" or \"json\")", format)
	}
}
