package trace

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// ReadJSONL reads trace events from JSONL format (one JSON object per line).
// Empty lines are skipped. Returns error on malformed JSON.
func ReadJSONL(r io.Reader) ([]TraceEvent, error) {
	var events []TraceEvent
	scanner := bufio.NewScanner(r)

	// Allow large lines (default 64KB may be too small for traces with large args)
	scanner.Buffer(make([]byte, 0, 256*1024), 1024*1024)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var event TraceEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}

		if event.Version == "" {
			return nil, fmt.Errorf("line %d: missing version field", lineNum)
		}
		if event.Event == "" {
			return nil, fmt.Errorf("line %d: missing event field", lineNum)
		}

		events = append(events, event)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading JSONL: %w", err)
	}

	return events, nil
}

// TraceMetadata extracts metadata from a trace event list.
// Returns the module name and capabilities from the first module_start event.
func TraceMetadata(events []TraceEvent) (moduleName string, caps []string) {
	for _, e := range events {
		if e.Event == EventModuleStart && e.Module != nil {
			return e.Module.Name, e.Module.Caps
		}
	}
	return "", nil
}
