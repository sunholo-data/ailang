package openrouter

import (
	"encoding/json"
	"fmt"

	"github.com/sunholo-data/ailang/internal/ai"
)

// OpenRouter Broadcast correlation fields (M-OPENROUTER-BROADCAST-INGEST M3).
//
// OpenRouter accepts three optional body fields — `user`, `session_id` and
// `trace` — and attaches them to the trace it broadcasts to configured
// destinations. They do not change the completion.
//
// There are three request-build sites in this package (chat.go, step.go,
// streamstep.go) and all three must carry these, or a broadcast trace from that
// path arrives un-joinable. The two helpers below are the single source of
// truth so the sites cannot drift.

// applyCorrelation stamps the correlation fields onto a chatRequest.
//
// A nil or empty Correlation leaves the struct untouched, so the marshalled
// body is byte-identical to one built before these fields existed. That
// property is asserted by the golden-body tests at every build site.
func applyCorrelation(apiReq *chatRequest, c *ai.Correlation) error {
	if c.IsEmpty() {
		return nil
	}
	if err := c.Validate(); err != nil {
		return err
	}
	apiReq.User = c.User
	apiReq.SessionID = c.SessionID
	apiReq.Trace = c.Trace
	return nil
}

// correlationExtras renders the correlation fields as pre-marshalled JSON
// fragments for the splice-based build sites (step.go, streamstep.go), which
// assemble their body from an OpenAI struct plus OpenRouter-only extensions
// rather than from chatRequest.
//
// Returns nil when there is nothing to add — the splice helper then emits the
// unmodified OpenAI body, preserving the flag-off wire bytes exactly.
func correlationExtras(c *ai.Correlation) ([][]byte, error) {
	if c.IsEmpty() {
		return nil, nil
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}

	var extras [][]byte
	appendField := func(name string, value any) error {
		encoded, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("marshal correlation %s: %w", name, err)
		}
		extras = append(extras, fmt.Appendf(nil, "%q:%s", name, encoded))
		return nil
	}

	if c.User != "" {
		if err := appendField("user", c.User); err != nil {
			return nil, err
		}
	}
	if c.SessionID != "" {
		if err := appendField("session_id", c.SessionID); err != nil {
			return nil, err
		}
	}
	if len(c.Trace) > 0 {
		if err := appendField("trace", c.Trace); err != nil {
			return nil, err
		}
	}
	return extras, nil
}
