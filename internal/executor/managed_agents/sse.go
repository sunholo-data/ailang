package managed_agents

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// sseEvent is one parsed SSE message: `event: <name>\ndata: <json>\n\n`.
type sseEvent struct {
	Name    string // The event name from the `event:` line ("interaction.created", etc.)
	RawData []byte // The raw bytes after `data:`. Empty if no data line.
	Done    bool   // True for the terminal `data: [DONE]` sentinel
}

// streamState accumulates information across SSE events for one interaction.
type streamState struct {
	InteractionID string
	EnvironmentID string
	Text          strings.Builder // Concatenated step.delta text (the actual response)
	Usage         Usage           // Final token counts from interaction.completed
	Status        string          // Final status (e.g. "completed", "failed")
	StepCount     int             // Number of step.start events seen
	UnknownEvents []map[string]any
}

// parseSSE reads an SSE stream from r, dispatching each event to handler.
// The handler is called once per `event:` block (including the terminal
// `done` sentinel). Non-matching / blank lines are tolerated per the SSE
// spec.
//
// Returns io.EOF when the stream closes cleanly, or any underlying read /
// JSON-parse error otherwise.
func parseSSE(r io.Reader, handler func(sseEvent) error) error {
	scanner := bufio.NewScanner(r)
	// SSE events can carry large JSON payloads; bump the buffer ceiling so a
	// big interaction.completed event (with full usage payload + IDs) doesn't
	// fall off the end of the scanner.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var (
		curName string
		curData strings.Builder
	)

	flush := func() error {
		if curName == "" && curData.Len() == 0 {
			return nil
		}
		ev := sseEvent{Name: curName}
		dataStr := curData.String()
		if strings.TrimSpace(dataStr) == "[DONE]" {
			ev.Done = true
		} else if dataStr != "" {
			ev.RawData = []byte(dataStr)
		}
		curName = ""
		curData.Reset()
		return handler(ev)
	}

	for scanner.Scan() {
		line := scanner.Text()

		// Blank line = event boundary
		if line == "" {
			if err := flush(); err != nil {
				return err
			}
			continue
		}

		// Comment lines (start with ":") are heartbeats per the SSE spec; skip.
		if strings.HasPrefix(line, ":") {
			continue
		}

		// Parse SSE field: "name: value" or "name:value"
		colon := strings.IndexByte(line, ':')
		if colon < 0 {
			continue // malformed line, ignore
		}
		field := line[:colon]
		value := line[colon+1:]
		if len(value) > 0 && value[0] == ' ' {
			value = value[1:]
		}

		switch field {
		case "event":
			curName = value
		case "data":
			// Multi-line `data:` accumulates with newlines per SSE spec.
			if curData.Len() > 0 {
				curData.WriteByte('\n')
			}
			curData.WriteString(value)
		}
	}

	// Flush any final event missing a trailing blank line.
	if curName != "" || curData.Len() > 0 {
		if err := flush(); err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return fmt.Errorf("sse scan error: %w", err)
	}
	return nil
}

// foldEvent updates the streamState from one parsed SSE event. Unknown event
// types are captured into UnknownEvents so forward-compat changes from
// Google's side don't silently drop information.
func foldEvent(s *streamState, ev sseEvent) error {
	if ev.Done {
		return nil
	}
	if len(ev.RawData) == 0 {
		return nil
	}

	switch ev.Name {
	case "interaction.created":
		var p createdPayload
		if err := json.Unmarshal(ev.RawData, &p); err != nil {
			return fmt.Errorf("parse interaction.created: %w", err)
		}
		s.InteractionID = p.Interaction.ID
		s.Status = p.Interaction.Status

	case "interaction.status_update":
		// Heartbeat; the interaction.completed event has the final status.
		// Nothing to capture here that isn't superseded later.

	case "step.start":
		s.StepCount++

	case "step.delta":
		var p stepDeltaPayload
		if err := json.Unmarshal(ev.RawData, &p); err != nil {
			return fmt.Errorf("parse step.delta: %w", err)
		}
		if p.Delta.Type == "text" || p.Delta.Type == "" {
			s.Text.WriteString(p.Delta.Text)
		}

	case "step.stop":
		// End of a step; nothing to fold yet.

	case "interaction.completed":
		var p completedPayload
		if err := json.Unmarshal(ev.RawData, &p); err != nil {
			return fmt.Errorf("parse interaction.completed: %w", err)
		}
		s.Status = p.Interaction.Status
		s.Usage = p.Interaction.Usage
		if p.Interaction.EnvironmentID != "" {
			s.EnvironmentID = p.Interaction.EnvironmentID
		}
		if p.Interaction.ID != "" {
			s.InteractionID = p.Interaction.ID
		}

	case "interaction.failed", "interaction.cancelled":
		// Capture failure path so the executor can surface a clear error.
		var p completedPayload
		_ = json.Unmarshal(ev.RawData, &p) // best-effort; failure may have minimal payload
		s.Status = p.Interaction.Status
		if s.Status == "" {
			s.Status = ev.Name // fallback to event name as the status
		}

	default:
		// Unknown event type — capture for forward-compat.
		var raw map[string]any
		if err := json.Unmarshal(ev.RawData, &raw); err == nil {
			raw["_event"] = ev.Name
			s.UnknownEvents = append(s.UnknownEvents, raw)
		}
	}
	return nil
}
