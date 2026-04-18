package compare

import (
	"encoding/json"
	"fmt"
	"io"
)

// WriteJSON writes a machine-readable JSON comparison report to w.
func WriteJSON(w io.Writer, c *Comparison) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(c); err != nil {
		return fmt.Errorf("encoding comparison JSON: %w", err)
	}
	return nil
}
