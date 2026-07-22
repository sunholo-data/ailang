package codex

// Codex CLI event schema, NDJSON parsing, and token accounting.
//
// Split out of codex.go to keep both files under the 800-line
// check-file-sizes gate. See README.md for schema documentation.

import (
	"encoding/json"
	"fmt"
	"strings"
)

// codexTokens captures the flat token structure Codex emits.
type codexTokens struct {
	Input  int `json:"input"`
	Output int `json:"output"`
}

// codexEvent is the normalized Codex NDJSON event shape.
//
// Codex CLI v0.1+ emits a thread/item stream format:
//
//	{"type":"thread.started","thread_id":"..."}
//	{"type":"turn.started"}
//	{"type":"item.started","item":{"id":"...","type":"file_change"|"command_execution"|"agent_message",...}}
//	{"type":"item.completed","item":{...}}
//	{"type":"turn.completed","usage":{"input_tokens":N,"cached_input_tokens":N,"output_tokens":N}}
//
// Older format (pre-v0.1) used flat records:
//
//	{"type":"message","turn_number":N,"text":"...","tokens_used":{"input":N,"output":N}}
//	{"type":"tool_use","tool_name":"...","parameters":{...}}
//
// Both formats are handled. Unknown fields are preserved in Raw for ProviderData.
type codexEvent struct {
	Type       string          `json:"type"`
	TurnNumber int             `json:"turn_number,omitempty"`
	Text       string          `json:"text,omitempty"`
	Tokens     codexTokens     `json:"tokens_used,omitempty"`
	SessionID  string          `json:"session_id,omitempty"`
	ThreadID   string          `json:"thread_id,omitempty"`
	ToolName   string          `json:"tool_name,omitempty"`
	Parameters json.RawMessage `json:"parameters,omitempty"`
	Output     string          `json:"output,omitempty"`
	Role       string          `json:"role,omitempty"`
	Usage      *codexUsage     `json:"usage,omitempty"`
	Item       *codexItem      `json:"item,omitempty"`

	// Raw preserves the full event map for ProviderData (tolerance to schema drift).
	Raw map[string]any `json:"-"`
}

// codexUsage is the token usage block in turn.completed events.
//
// CachedInputTokens is a SUBSET of InputTokens (OpenAI Responses API
// semantics) — see splitCodexInputTokens before using it.
type codexUsage struct {
	InputTokens       int `json:"input_tokens"`
	CachedInputTokens int `json:"cached_input_tokens"`
	OutputTokens      int `json:"output_tokens"`
}

// NOTE (finish reason): the codex CLI's `exec --json` stream exposes NO
// model-level stop reason, in either schema generation. The new format's
// terminal event, turn.completed, carries only a usage block; the old flat
// format's `result` event carries a `status` field that has only ever been
// observed as "success" (and only in this repo's hand-authored fixture — no
// real codex capture exists here, so the failure vocabulary is unknown).
// Neither is modeled as a typed field; `status` survives only in
// ProviderData["codex_events"].
//
// Consequently Result.FinishReason from this executor reflects what the
// EXECUTOR observed (clean termination / timeout / cost kill / non-zero exit),
// never what the MODEL decided. A codex run that refused the task, tripped a
// content filter, or truncated at max tokens reports "stop" like any other
// clean run. Closing that gap requires capturing a real `codex exec --json`
// stream — including a deliberately failing run — not extending the parser
// against the synthetic fixture.

// codexItem is the item payload in item.started / item.completed events.
type codexItem struct {
	ID      string `json:"id"`
	Type    string `json:"type"` // "agent_message", "command_execution", "file_change"
	Text    string `json:"text,omitempty"`
	Command string `json:"command,omitempty"`
}

// parseCodexEvent parses a single NDJSON line into a codexEvent.
// Non-JSON and unparseable lines return an error; callers should skip them
// rather than fail hard (Codex CLI may emit non-JSON preamble on stdout).
func parseCodexEvent(line []byte) (*codexEvent, error) {
	trimmed := strings.TrimSpace(string(line))
	if trimmed == "" {
		return nil, fmt.Errorf("empty line")
	}
	if trimmed[0] != '{' {
		return nil, fmt.Errorf("non-JSON line")
	}
	var ev codexEvent
	if err := json.Unmarshal(line, &ev); err != nil {
		return nil, fmt.Errorf("json: %w", err)
	}
	// Tolerate schema drift: always capture the raw map.
	var raw map[string]any
	if err := json.Unmarshal(line, &raw); err == nil {
		ev.Raw = raw
	}
	return &ev, nil
}

// providerData wraps the list of raw events as the Result.ProviderData map.
// splitCodexInputTokens separates codex's cumulative input_tokens into the
// (fresh, cached) pair that executor.Result models as InputTokens +
// CacheReadInputTokens.
//
// codex is OpenAI's own CLI over the Responses API, where cached_input_tokens
// is a SUBSET of input_tokens (unlike opencode, whose cache counters are
// exclusive of input). Adding it to InputTokens would therefore double-count:
// cmd/ailang/eval_benchmark.go banks agent-mode input as
// InputTokens + CacheCreationInputTokens + CacheReadInputTokens.
//
// Splitting is total-preserving — (input-cached) + cached == input — so the
// banked input total is correct under the documented subset semantics, and no
// worse than today's (cache-blind) total even if a future codex release made
// the counters exclusive. It also lets CostModel.CacheReadCost, declared since
// the executor was written but never applicable, finally apply.
//
// codex reports no cache-CREATION counter, so CacheCreationInputTokens stays 0.
// Not verified against a real codex stream: the repo's only codex fixture is
// synthetic and carries no usage block (see README). Re-check when a real
// capture lands.
func splitCodexInputTokens(inputTokens, cachedInputTokens int) (fresh, cached int) {
	if cachedInputTokens <= 0 {
		return inputTokens, 0
	}
	// Defensive: never let a malformed cached > input yield negative fresh.
	if cachedInputTokens > inputTokens {
		return 0, inputTokens
	}
	return inputTokens - cachedInputTokens, cachedInputTokens
}

func providerData(events []map[string]any) map[string]any {
	if len(events) == 0 {
		return nil
	}
	return map[string]any{
		"codex_events": events,
	}
}
