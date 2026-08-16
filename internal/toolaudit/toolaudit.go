// Package toolaudit inspects an agent's own report of the tools it has access
// to (the stream's system/init event) and diffs it against a variant's declared
// allowed_tools, so tool-access leakage can be caught on the first sample.
package toolaudit

import (
	"encoding/json"
	"sort"
	"strings"
)

// initEvent is the minimal shape of the claude-code stream's system/init line.
// Only the fields needed to extract the reported tool list are decoded.
type initEvent struct {
	Type    string            `json:"type"`
	Subtype string            `json:"subtype"`
	Tools   []json.RawMessage `json:"tools"`
}

// AvailableTools returns the tools an agent reported in its system/init event,
// read from a run's raw conversation. ok is false when no init event carries a
// non-empty tools array (e.g. runners that don't emit one), so callers can
// no-op quietly rather than assuming a fixed built-in set.
func AvailableTools(conversation []json.RawMessage) (tools []string, ok bool) {
	for _, raw := range conversation {
		var ev initEvent
		if err := json.Unmarshal(raw, &ev); err != nil {
			continue
		}
		if ev.Type != "system" || ev.Subtype != "init" || len(ev.Tools) == 0 {
			continue
		}
		return decodeToolNames(ev.Tools), true
	}
	return nil, false
}

// decodeToolNames turns each raw tools entry into a name. Entries are plain JSON
// strings in practice; an object with a "name" field is also accepted so a
// runner reporting richer tool descriptors still yields usable names.
func decodeToolNames(raw []json.RawMessage) []string {
	names := make([]string, 0, len(raw))
	for _, entry := range raw {
		var name string
		if err := json.Unmarshal(entry, &name); err == nil {
			if name != "" {
				names = append(names, name)
			}
			continue
		}
		var obj struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(entry, &obj); err == nil && obj.Name != "" {
			names = append(names, obj.Name)
		}
	}
	return names
}

// Leaks returns the available tools not covered by allowed, sorted for stable
// output. allowed entries are matched by base tool name, so a scoped entry like
// "Bash(git:*)" covers the base "Bash" the init event reports. It returns nil
// when allowed is empty (nothing was declared to enforce) or nothing leaked.
func Leaks(available, allowed []string) []string {
	if len(allowed) == 0 {
		return nil
	}
	allowedBase := make(map[string]bool, len(allowed))
	for _, a := range allowed {
		allowedBase[baseToolName(a)] = true
	}
	var extras []string
	for _, tool := range available {
		if !allowedBase[baseToolName(tool)] {
			extras = append(extras, tool)
		}
	}
	sort.Strings(extras)
	return extras
}

// baseToolName strips a trailing scope suffix ("Bash(git:*)" -> "Bash") so a
// scoped allow entry matches the unscoped name the agent reports.
func baseToolName(tool string) string {
	if i := strings.IndexByte(tool, '('); i >= 0 {
		return tool[:i]
	}
	return tool
}

// BuiltinWhitelist derives the exclusive --tools value from a variant's
// allowed_tools. It reduces each entry to its base name ("Bash(git:*)" ->
// "Bash"), drops MCP entries (mcp__*, which are governed by mcp_config, not the
// built-in --tools set), and dedupes while preserving order.
//
// When allowed_tools names no built-ins (e.g. only MCP tools, or an empty list),
// it returns [""] — which the runner emits as `--tools ""` to disable every
// built-in — so the allow list stays an exclusive whitelist rather than silently
// leaving built-ins enabled.
func BuiltinWhitelist(allowed []string) []string {
	seen := make(map[string]bool, len(allowed))
	var out []string
	for _, entry := range allowed {
		base := baseToolName(entry)
		if base == "" || strings.HasPrefix(base, "mcp__") || seen[base] {
			continue
		}
		seen[base] = true
		out = append(out, base)
	}
	if len(out) == 0 {
		return []string{""}
	}
	return out
}
