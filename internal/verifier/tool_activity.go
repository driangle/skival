package verifier

import (
	"encoding/json"
	"fmt"
	"strings"
)

// maxToolBlockLen caps the rendered length of a single tool_use input or
// tool_result content block in the summary. Long blocks are truncated with
// an ellipsis to keep the judge prompt bounded.
const maxToolBlockLen = 800

// streamEnvelope is a minimal view of a claude-code stream-json message
// that exposes only the fields needed to summarize tool activity.
type streamEnvelope struct {
	Type    string `json:"type"`
	Message *struct {
		Content []streamContentBlock `json:"content,omitempty"`
	} `json:"message,omitempty"`
}

type streamContentBlock struct {
	Type    string          `json:"type"`
	Name    string          `json:"name,omitempty"`
	Input   json.RawMessage `json:"input,omitempty"`
	Content json.RawMessage `json:"content,omitempty"`
}

// SummarizeToolActivity renders a human-readable summary of the tool calls
// and tool results found in a claude-code stream conversation. Messages that
// do not parse or contain no tool blocks are silently skipped. Returns an
// empty string when there is no tool activity to report.
func SummarizeToolActivity(conversation []json.RawMessage) string {
	var sb strings.Builder
	for _, raw := range conversation {
		var env streamEnvelope
		if err := json.Unmarshal(raw, &env); err != nil {
			continue
		}
		if env.Message == nil {
			continue
		}
		for _, b := range env.Message.Content {
			switch b.Type {
			case "tool_use":
				fmt.Fprintf(&sb, "-> tool_use %s: %s\n", b.Name, truncate(string(b.Input), maxToolBlockLen))
			case "tool_result":
				fmt.Fprintf(&sb, "<- tool_result: %s\n", truncate(stringifyToolResult(b.Content), maxToolBlockLen))
			}
		}
	}
	return sb.String()
}

// stringifyToolResult flattens a tool_result content payload into a plain
// string. The content may be a JSON string, an array of {type, text} blocks,
// or arbitrary JSON — in that last case we return the raw JSON bytes.
func stringifyToolResult(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &blocks); err == nil {
		var sb strings.Builder
		for i, b := range blocks {
			if b.Type != "text" {
				continue
			}
			if i > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString(b.Text)
		}
		return sb.String()
	}
	return string(raw)
}
