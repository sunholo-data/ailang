package main

import (
	"fmt"
	"strings"
)

// setAgentField changes ONE field on ONE agent, by line, leaving every other
// byte of the file alone.
//
// Not a YAML round-trip on purpose. config.cloud.yaml is ~1300 lines and its
// comments carry the rulings — who decided a thing, when, and what breaks if you
// widen it ("those patterns are this agent's whole safety scope"). yaml.v3
// re-serialisation reflows the document and moves or drops those, so a one-word
// change would arrive as an unreviewable diff and the reasoning would rot.
//
// Returns the edited document, the previous value (empty if the field was
// absent) and the new one.
func setAgentField(doc, agentID, field, value string) (edited, before, after string, err error) {
	lines := strings.Split(doc, "\n")

	start := -1
	for i, l := range lines {
		if isAgentIDLine(l, agentID) {
			start = i
			break
		}
	}
	if start < 0 {
		return "", "", "", fmt.Errorf("no agent %q in the config — check the id, not the inbox", agentID)
	}

	// The block runs to the next list item at the same indentation, or EOF.
	indent := leadingSpaces(lines[start])
	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		if isListItemAt(lines[i], indent) {
			end = i
			break
		}
	}

	// Fields sit one level in from the "- id:" marker, which itself is indented
	// by `indent` and prefixed "- ".
	fieldIndent := strings.Repeat(" ", indent+2)
	prefix := fieldIndent + field + ":"

	for i := start; i < end; i++ {
		if strings.HasPrefix(lines[i], prefix) {
			before = strings.TrimSpace(strings.TrimPrefix(lines[i], prefix))
			before = strings.Trim(before, `"'`)
			// Preserve any trailing comment on the line: it is usually the
			// reason the value is what it is.
			comment := ""
			if idx := strings.Index(lines[i], " #"); idx > len(prefix) {
				comment = lines[i][idx:]
			}
			lines[i] = prefix + " " + value + comment
			return strings.Join(lines, "\n"), before, value, nil
		}
	}

	// Absent: append at the end of the block, after the last non-blank line so
	// the entry does not grow a stray gap in the middle.
	insertAt := end
	for insertAt > start && strings.TrimSpace(lines[insertAt-1]) == "" {
		insertAt--
	}
	out := make([]string, 0, len(lines)+1)
	out = append(out, lines[:insertAt]...)
	out = append(out, prefix+" "+value)
	out = append(out, lines[insertAt:]...)
	return strings.Join(out, "\n"), "", value, nil
}

// isAgentIDLine matches `- id: <agentID>` at any indentation, tolerating quotes
// and a trailing comment, and rejecting a prefix match (design-doc-creator must
// not match design-doc-creator-daneel).
func isAgentIDLine(line, agentID string) bool {
	t := strings.TrimSpace(line)
	if !strings.HasPrefix(t, "- id:") {
		return false
	}
	v := strings.TrimSpace(strings.TrimPrefix(t, "- id:"))
	if i := strings.Index(v, " #"); i >= 0 {
		v = strings.TrimSpace(v[:i])
	}
	return strings.Trim(v, `"'`) == agentID
}

func isListItemAt(line string, indent int) bool {
	if leadingSpaces(line) != indent {
		return false
	}
	return strings.HasPrefix(strings.TrimSpace(line), "- ")
}

func leadingSpaces(s string) int {
	for i, r := range s {
		if r != ' ' {
			return i
		}
	}
	return len(s)
}
