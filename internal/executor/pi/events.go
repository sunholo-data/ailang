package pi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/executor"
)

// piUsageCost mirrors message_end.message.usage.cost.
type piUsageCost struct {
	Input      float64 `json:"input"`
	Output     float64 `json:"output"`
	CacheRead  float64 `json:"cacheRead"`
	CacheWrite float64 `json:"cacheWrite"`
	Total      float64 `json:"total"`
}

// piUsage mirrors message_end.message.usage.
type piUsage struct {
	Input       int         `json:"input"`
	Output      int         `json:"output"`
	CacheRead   int         `json:"cacheRead"`
	CacheWrite  int         `json:"cacheWrite"`
	TotalTokens int         `json:"totalTokens"`
	Cost        piUsageCost `json:"cost"`
}

// piMessage captures the per-message envelope.
type piMessage struct {
	Role string `json:"role"`
	// StopReason is pi's per-message stop signal, present on message_start /
	// message_end / turn_end (and mirrored on assistantMessageEvent.partial).
	// Observed values: "stop", "toolUse" (fixtures, pi 0.70.2). A tool-calling
	// turn ends "toolUse" and the run's FINAL turn ends "stop", so only the
	// last value seen is meaningful as a run-level finish reason.
	StopReason string   `json:"stopReason,omitempty"`
	Usage      *piUsage `json:"usage,omitempty"`
}

// piAssistantMessageEvent captures the inner discriminator for message_update.
type piAssistantMessageEvent struct {
	Type         string `json:"type"`
	ContentIndex int    `json:"contentIndex,omitempty"`
	Delta        string `json:"delta,omitempty"`
	Content      string `json:"content,omitempty"`
}

// piToolResult captures the tool_execution_end result envelope.
type piToolResult struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

// piEvent is the top-level NDJSON event wrapper.
type piEvent struct {
	Type                  string                   `json:"type"`
	Message               *piMessage               `json:"message,omitempty"`
	AssistantMessageEvent *piAssistantMessageEvent `json:"assistantMessageEvent,omitempty"`

	// session event
	SessionID string `json:"id,omitempty"`

	// tool_execution_start / tool_execution_end
	ToolCallID string          `json:"toolCallId,omitempty"`
	ToolName   string          `json:"toolName,omitempty"`
	Args       json.RawMessage `json:"args,omitempty"`
	Result     *piToolResult   `json:"result,omitempty"`
	IsError    bool            `json:"isError,omitempty"`

	// auto_retry_start / auto_retry_end (pi >= 0.84)
	Attempt     int  `json:"attempt,omitempty"`
	MaxAttempts int  `json:"maxAttempts,omitempty"`
	Success     bool `json:"success,omitempty"`

	// Raw preserves full event for ProviderData (schema-drift tolerance).
	Raw map[string]any `json:"-"`
}

// piRetries tallies pi's internal auto-retry loop (0.84+): bounded upstream by
// maxRetries (default 3), announced on the wire as maxAttempts. A run that
// reaches maxAttempts and then fails is a NAMED outcome, not a slow success.
type piRetries struct {
	Count       int
	MaxAttempts int
	Exhausted   bool
}

func (r *piRetries) observeStart(attempt, maxAttempts int) {
	r.Count++
	if maxAttempts > r.MaxAttempts {
		r.MaxAttempts = maxAttempts
	}
}

func (r *piRetries) observeEnd(attempt int, success bool) {
	if !success && r.MaxAttempts > 0 && attempt >= r.MaxAttempts {
		r.Exhausted = true
	}
}

func (r *piRetries) providerData() map[string]any {
	if r.Count == 0 {
		return nil
	}
	return map[string]any{"count": r.Count, "max_attempts": r.MaxAttempts, "exhausted": r.Exhausted}
}

// parsePiEvent parses a single NDJSON line.
// Returns error for non-JSON lines or parse failures; callers skip them.
func parsePiEvent(line []byte) (*piEvent, error) {
	trimmed := strings.TrimSpace(string(line))
	if trimmed == "" {
		return nil, fmt.Errorf("empty line")
	}
	if trimmed[0] != '{' {
		return nil, fmt.Errorf("non-JSON line")
	}
	var ev piEvent
	if err := json.Unmarshal(line, &ev); err != nil {
		return nil, fmt.Errorf("json: %w", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(line, &raw); err == nil {
		ev.Raw = raw
	}
	return &ev, nil
}

// flattenPiToolResult joins all text content blocks from a tool_execution_end.
func flattenPiToolResult(r *piToolResult) string {
	if r == nil {
		return ""
	}
	if len(r.Content) == 0 {
		return ""
	}
	if len(r.Content) == 1 {
		return r.Content[0].Text
	}
	var b strings.Builder
	for i, c := range r.Content {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(c.Text)
	}
	return b.String()
}

// piCancelFinishReason distinguishes a deadline-driven context cancellation
// (a timeout by another name) from a caller-driven one (an abort).
func piCancelFinishReason(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return executor.FinishTimeout
	}
	return executor.FinishError
}

// normalizePiFinishReason maps pi's camelCase stopReason vocabulary onto the
// canonical executor.Finish* values.
//
// Pi is pre-1.0 (0.70.x) and its stop-reason vocabulary is NOT documented
// upstream; "stop" and "toolUse" are the only values observed in captured
// fixtures. Rather than guess at the rest, unrecognized values are passed
// through verbatim so they surface in the banked JSON instead of being
// silently coerced to "stop" (CategorizeAgentError ignores values it doesn't
// know, so pass-through cannot misclassify a run). Re-check this mapping when
// bumping the pinned pi version.
func normalizePiFinishReason(raw string) string {
	switch raw {
	case "stop", "endTurn", "end_turn":
		return executor.FinishStop
	case "toolUse", "tool_use", "toolCalls":
		return executor.FinishToolCalls
	case "maxTokens", "max_tokens", "length":
		return executor.FinishLength
	case "refusal", "safety", "contentFilter":
		return executor.FinishContentFilter
	case "aborted", "error":
		return executor.FinishError
	default:
		return raw
	}
}

// piProviderData wraps raw events as Result.ProviderData, plus the M2 drift
// signals: pi_unknown_events {type: count}, pi_unparsed_lines and pi_retries, each only when
// non-empty so a clean stream banks neither.
func piProviderData(events []map[string]any, unknown map[string]int, unparsed int, retries *piRetries) map[string]any {
	pd := map[string]any{}
	if len(events) > 0 {
		pd["pi_events"] = events
	}
	if len(unknown) > 0 {
		pd["pi_unknown_events"] = unknown
	}
	if unparsed > 0 {
		pd["pi_unparsed_lines"] = unparsed
	}
	if r := retries.providerData(); r != nil {
		pd["pi_retries"] = r
	}
	if len(pd) == 0 {
		return nil
	}
	return pd
}
