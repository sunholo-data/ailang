package trace

import (
	"encoding/json"
	"io"
)

// WriteJSONL writes all collected events as JSONL (one JSON object per line) to the writer.
func WriteJSONL(w io.Writer, events []TraceEvent) error {
	enc := json.NewEncoder(w)
	for _, event := range events {
		if err := enc.Encode(event); err != nil {
			return err
		}
	}
	return nil
}
