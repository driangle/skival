package exec

import (
	"bufio"
	"context"
	"encoding/json"
	"os"

	agentrunner "github.com/driangle/agentrunner/go"
)

// maxEventLine bounds a single JSONL event line to guard against unbounded
// memory use when a program writes very large tool payloads.
const maxEventLine = 10 * 1024 * 1024

// finalEvent is the terminal event a program may emit to report token usage and
// cost. Its text is used only as a fallback when the program writes no stdout.
type finalEvent struct {
	Text    string  `json:"text"`
	CostUSD float64 `json:"cost_usd"`
	Usage   struct {
		InputTokens              int `json:"input_tokens"`
		OutputTokens             int `json:"output_tokens"`
		CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
		CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	} `json:"usage"`
}

// forwardEvents reads the JSONL events file (if present), forwards each event to
// the session as a raw message, and returns the terminal final event if one was
// seen. A missing or unreadable file is tolerated and yields a nil result.
func forwardEvents(ctx context.Context, path string, msgCh chan<- agentrunner.Message) *finalEvent {
	if path == "" {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var final *finalEvent
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), maxEventLine)
	for scanner.Scan() {
		line := trimmedCopy(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		kind, ok := eventType(line)
		if !ok {
			continue
		}
		if kind == "final" {
			if fe := parseFinal(line); fe != nil {
				final = fe
			}
			continue
		}
		if !send(ctx, msgCh, agentrunner.Message{Type: mapEventType(kind), Raw: line}) {
			return final
		}
	}
	return final
}

// eventType returns the "type" field of a JSONL event line.
func eventType(line []byte) (string, bool) {
	var head struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(line, &head); err != nil {
		return "", false
	}
	return head.Type, true
}

// parseFinal decodes a final event, returning nil if it does not parse.
func parseFinal(line []byte) *finalEvent {
	var fe finalEvent
	if err := json.Unmarshal(line, &fe); err != nil {
		return nil
	}
	return &fe
}

// send delivers a message unless the context is cancelled, returning false when
// it was cancelled.
func send(ctx context.Context, msgCh chan<- agentrunner.Message, msg agentrunner.Message) bool {
	select {
	case msgCh <- msg:
		return true
	case <-ctx.Done():
		return false
	}
}

// trimmedCopy returns a copy of b with trailing newline/carriage-return removed.
// A copy is required because bufio.Scanner reuses its buffer.
func trimmedCopy(b []byte) []byte {
	out := make([]byte, len(b))
	copy(out, b)
	return out
}

// mapEventType maps an event's type string to an agentrunner MessageType. The
// mapping is cosmetic — verifiers read the raw JSON, not the type.
func mapEventType(kind string) agentrunner.MessageType {
	switch kind {
	case "tool_use":
		return agentrunner.MessageTypeToolUse
	case "tool_result":
		return agentrunner.MessageTypeToolResult
	case "message":
		return agentrunner.MessageTypeAssistant
	default:
		return agentrunner.MessageType(kind)
	}
}
