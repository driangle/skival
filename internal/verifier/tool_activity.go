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

// streamEnvelope is a minimal view of a session message that exposes only the
// fields needed to summarize tool activity. It covers two shapes: the nested
// claude-code stream-json message (tool blocks under message.content) and the
// flat exec-runner event (type/name/input/content at the top level).
type streamEnvelope struct {
	Type    string `json:"type"`
	Message *struct {
		Content []streamContentBlock `json:"content,omitempty"`
	} `json:"message,omitempty"`

	// Flat top-level fields for exec-runner events.
	Name    string          `json:"name,omitempty"`
	Input   json.RawMessage `json:"input,omitempty"`
	Content json.RawMessage `json:"content,omitempty"`
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
		if env.Message != nil {
			writeNestedBlocks(&sb, env.Message.Content)
			continue
		}
		writeFlatEvent(&sb, env)
	}
	return sb.String()
}

// writeNestedBlocks renders tool blocks nested under a claude-code
// message.content array.
func writeNestedBlocks(sb *strings.Builder, blocks []streamContentBlock) {
	for _, b := range blocks {
		switch b.Type {
		case "tool_use":
			fmt.Fprintf(sb, "-> tool_use %s: %s\n", b.Name, truncate(string(b.Input), maxToolBlockLen))
		case "tool_result":
			fmt.Fprintf(sb, "<- tool_result: %s\n", truncate(stringifyToolResult(b.Content), maxToolBlockLen))
		}
	}
}

// writeFlatEvent renders a flat exec-runner event whose type/name/input/content
// live at the top level of the JSON object.
func writeFlatEvent(sb *strings.Builder, env streamEnvelope) {
	switch env.Type {
	case "tool_use":
		fmt.Fprintf(sb, "-> tool_use %s: %s\n", env.Name, truncate(string(env.Input), maxToolBlockLen))
	case "tool_result":
		fmt.Fprintf(sb, "<- tool_result: %s\n", truncate(stringifyToolResult(env.Content), maxToolBlockLen))
	}
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
