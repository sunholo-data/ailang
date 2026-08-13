package ai

import "fmt"

// Correlation carries the identifiers that let a provider-side trace be joined
// back to the run that caused it.
//
// This exists for OpenRouter Broadcast, which pushes a trace for every request
// — input/output, token counts, cost, timing — to a configured destination.
// Without these fields those traces arrive as an undifferentiated stream that
// cannot be matched to an eval run, chain, or benchmark, which is the whole
// reason to ingest them.
//
// See https://openrouter.ai/docs/guides/features/broadcast
type Correlation struct {
	// User identifies the end user. Optional; <= MaxUserLen chars.
	User string

	// SessionID groups related requests. For AILANG this is the chain ID, the
	// finest grain already persisted per run. Optional; <= MaxSessionIDLen.
	SessionID string

	// Trace is free-form metadata attached to the provider-side trace, e.g.
	// {"trace_name": "eval:fizzbuzz", "benchmark": "fizzbuzz", "tier": "core"}.
	Trace map[string]any
}

// OpenRouter's documented caps on the correlation fields.
const (
	MaxUserLen      = 128
	MaxSessionIDLen = 256
)

// Validate rejects over-cap values.
//
// Truncating instead would be worse than dropping the field: a shortened
// session id is still a syntactically fine id that silently joins to nothing,
// which is the class of silent data corruption CLAUDE.md Principle 2 exists to
// prevent. Callers get a typed error before any request is dispatched.
func (c *Correlation) Validate() error {
	if c == nil {
		return nil
	}
	if len(c.User) > MaxUserLen {
		return fmt.Errorf("correlation user is %d chars, exceeds the OpenRouter limit of %d", len(c.User), MaxUserLen)
	}
	if len(c.SessionID) > MaxSessionIDLen {
		return fmt.Errorf("correlation session_id is %d chars, exceeds the OpenRouter limit of %d", len(c.SessionID), MaxSessionIDLen)
	}
	return nil
}

// IsEmpty reports whether there is nothing to send.
//
// Providers use this to keep the wire bytes untouched when no correlation was
// requested.
func (c *Correlation) IsEmpty() bool {
	return c == nil || (c.User == "" && c.SessionID == "" && len(c.Trace) == 0)
}
