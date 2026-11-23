package agy

import (
	"encoding/json"
	"fmt"
	"strings"
)

// agy --output-format stream-json emits one JSON object per line with a
// top-level "event" discriminator (agy 1.5.15, fixtures in testdata/):
//
//	init         {model, cwd, tools[], permission_mode}
//	step_update  {step_index, state ACTIVE|DONE, step_type user_input|agent_response|tool,
//	              tool_name, tool_info, text_delta, duration_seconds, usage}
//	result       {status SUCCESS|ERROR, response, error, num_turns, usage,
//	              denied_actions[], finished_reason}
//
// Usage on a DONE agent_response step is that step's delta; result.usage is the
// run total and is authoritative. input_tokens is FRESH input, disjoint from
// cache_read_tokens (step 1: 9240 in / 0 cached; step 2: 2212 in / 7198 cached;
// result: 11452 / 7198). thinking_tokens is reported separately from
// output_tokens.

type agyUsage struct {
	InputTokens     int `json:"input_tokens"`
	OutputTokens    int `json:"output_tokens"`
	ThinkingTokens  int `json:"thinking_tokens"`
	CacheReadTokens int `json:"cache_read_tokens"`
	TotalTokens     int `json:"total_tokens"`
}

type agyInit struct {
	Model          string   `json:"model"`
	Cwd            string   `json:"cwd"`
	Tools          []string `json:"tools"`
	PermissionMode string   `json:"permission_mode"`
}

type agyStep struct {
	ConversationID  string          `json:"conversation_id"`
	StepIndex       int             `json:"step_index"`
	State           string          `json:"state"`
	StepType        string          `json:"step_type"`
	ToolName        string          `json:"tool_name"`
	ToolInfo        json.RawMessage `json:"tool_info"`
	TextDelta       string          `json:"text_delta"`
	DurationSeconds float64         `json:"duration_seconds"`
	Usage           *agyUsage       `json:"usage"`
}

type agyDenied struct {
	Action      string `json:"action"`
	DisplayName string `json:"display_name"`
}

type agyResult struct {
	ConversationID  string      `json:"conversation_id"`
	Status          string      `json:"status"`
	Response        string      `json:"response"`
	Error           string      `json:"error"`
	DurationSeconds float64     `json:"duration_seconds"`
	NumTurns        int         `json:"num_turns"`
	Usage           *agyUsage   `json:"usage"`
	DeniedActions   []agyDenied `json:"denied_actions"`
	// Omitted on a normal finish (agy 1.4.11+).
	FinishedReason string `json:"finished_reason"`
}

type agyEvent struct {
	Event      string     `json:"event"`
	Init       *agyInit   `json:"init"`
	StepUpdate *agyStep   `json:"step_update"`
	Result     *agyResult `json:"result"`
}

// parseAgyEvent decodes one stream line. A non-JSON line, or JSON without an
// "event" field, is an error the caller counts and skips.
func parseAgyEvent(line []byte) (*agyEvent, error) {
	trimmed := strings.TrimSpace(string(line))
	if trimmed == "" || trimmed[0] != '{' {
		return nil, fmt.Errorf("not a JSON object")
	}
	var ev agyEvent
	if err := json.Unmarshal([]byte(trimmed), &ev); err != nil {
		return nil, err
	}
	if ev.Event == "" {
		return nil, fmt.Errorf("no event discriminator")
	}
	return &ev, nil
}

// deniedSummary renders denied_actions for an error message.
func deniedSummary(d []agyDenied) string {
	parts := make([]string, 0, len(d))
	for _, a := range d {
		name := a.Action
		if a.DisplayName != "" {
			name += "(" + a.DisplayName + ")"
		}
		parts = append(parts, name)
	}
	return strings.Join(parts, ", ")
}

// isAuthFailure reports whether a result error is agy's missing/expired
// sign-in, which deserves an actionable message rather than a raw string.
func isAuthFailure(msg string) bool {
	m := strings.ToLower(msg)
	return strings.Contains(m, "authentication failed") || strings.Contains(m, "authentication required") ||
		strings.Contains(m, "please sign in")
}
