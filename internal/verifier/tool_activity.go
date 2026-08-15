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
	visitToolBlocks(conversation,
		func(name string, input json.RawMessage) {
			fmt.Fprintf(&sb, "-> tool_use %s: %s\n", name, truncate(string(input), maxToolBlockLen))
		},
		func(content json.RawMessage) {
			fmt.Fprintf(&sb, "<- tool_result: %s\n", truncate(stringifyToolResult(content), maxToolBlockLen))
		})
	return sb.String()
}

// CountToolUses tallies how many times each tool was invoked across a
// conversation, keyed by tool name. It walks the same two conversation shapes
// as SummarizeToolActivity. Returns an empty map when there is no tool activity.
func CountToolUses(conversation []json.RawMessage) map[string]int {
	counts := make(map[string]int)
	visitToolBlocks(conversation,
		func(name string, _ json.RawMessage) {
			if name != "" {
				counts[name]++
			}
		},
		func(json.RawMessage) {})
	return counts
}

// visitToolBlocks walks a conversation once and invokes onUse for every
// tool_use block and onResult for every tool_result block, across both the
// nested claude-code (message.content) and flat exec-runner shapes. Messages
// that do not parse are silently skipped.
func visitToolBlocks(conversation []json.RawMessage, onUse func(name string, input json.RawMessage), onResult func(content json.RawMessage)) {
	for _, raw := range conversation {
		var env streamEnvelope
		if err := json.Unmarshal(raw, &env); err != nil {
			continue
		}
		if env.Message != nil {
			for _, b := range env.Message.Content {
				switch b.Type {
				case "tool_use":
					onUse(b.Name, b.Input)
				case "tool_result":
					onResult(b.Content)
				}
			}
			continue
		}
		switch env.Type {
		case "tool_use":
			onUse(env.Name, env.Input)
		case "tool_result":
			onResult(env.Content)
		}
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
