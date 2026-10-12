package claudegateway

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/sunholo-data/ailang/internal/modelreg"
)

func usageCounter(raw json.RawMessage) (int64, error) {
	var n *int64
	if json.Unmarshal(raw, &n) != nil || n == nil || *n < 0 {
		return 0, errors.New("invalid usage counter")
	}
	return *n, nil
}

// Anthropic's SDK replaces cumulative whole-message counters when present;
// optional null/omitted fields preserve the preceding observation. Validate the
// complete merged object once, so cache total/split updates are order independent.
// https://github.com/anthropics/anthropic-sdk-typescript/blob/main/src/lib/MessageStream.ts
func mergeStreamUsage(previous modelreg.ClaudeUsage, delta map[string]json.RawMessage, maxTokens int) (modelreg.ClaudeUsage, error) {
	output, err := usageCounter(delta["output_tokens"])
	if err != nil || output < previous.OutputTokens || output > int64(maxTokens) {
		return previous, errors.New("missing or impossible stream output usage")
	}
	merged := map[string]any{
		"input_tokens": previous.InputTokens, "output_tokens": output,
		"cache_read_input_tokens":     previous.CacheReadTokens,
		"cache_creation_input_tokens": previous.CacheWrite5mTokens + previous.CacheWrite1hTokens,
		"cache_creation":              map[string]int64{"ephemeral_5m_input_tokens": previous.CacheWrite5mTokens, "ephemeral_1h_input_tokens": previous.CacheWrite1hTokens},
	}
	for field, raw := range delta {
		switch field {
		case "output_tokens", "input_tokens", "cache_read_input_tokens", "cache_creation_input_tokens", "cache_creation", "server_tool_use", "service_tier", "inference_geo", "output_tokens_details":
			if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
				continue
			}
			merged[field] = raw
		default:
			return previous, errors.New("unsupported stream usage category")
		}
	}
	raw, err := json.Marshal(merged)
	if err != nil {
		return previous, err
	}
	updated, err := parseUsage(raw, true)
	if err != nil {
		return previous, err
	}
	return updated, nil
}
