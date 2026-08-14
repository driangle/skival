package compare

import (
	"encoding/json"
	"fmt"
	"io"
)

// WriteJSON writes a machine-readable JSON comparison report to w. The embedded
// *Comparison promotes its fields to the top level, alongside the attribution.
func WriteJSON(w io.Writer, c *Comparison) error {
	out := struct {
		*Comparison
		MadeWith jsonAttribution `json:"made_with"`
	}{Comparison: c, MadeWith: attribution()}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return fmt.Errorf("encoding comparison JSON: %w", err)
	}
	return nil
}
