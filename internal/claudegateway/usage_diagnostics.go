package claudegateway

import (
	"bytes"
	"encoding/json"
	"log"

	"github.com/sunholo-data/ailang/internal/modelreg"
)

// UsageDiagnostic contains billing observations only, never message content,
// headers, credentials, raw responses or arbitrary provider fields. Observations
// on a failed stream are diagnostic evidence, not a verified settlement receipt.
type UsageDiagnostic struct {
	Event             string     `json:"event"`
	AccountID         string     `json:"account_id"`
	TaskID            string     `json:"task_id"`
	RequestID         string     `json:"request_id"`
	ProviderRequestID string     `json:"provider_request_id"`
	MessageID         string     `json:"message_id"`
	Stage             string     `json:"stage"`
	Reason            string     `json:"reason"`
	UnresolvedMarked  bool       `json:"unresolved_marked"`
	Stream            usageTrace `json:"usage_evidence"`
}

type usageTrace struct {
	Start        map[string]any        `json:"start,omitempty"`
	Final        map[string]any        `json:"last_delta,omitempty"`
	LastVerified *modelreg.ClaudeUsage `json:"last_verified_totals,omitempty"`
	Started      bool                  `json:"started"`
	FinalSeen    bool                  `json:"final_seen"`
	Stopped      bool                  `json:"stopped"`
}

func (trace *usageTrace) observeVerified(usage modelreg.ClaudeUsage) {
	trace.LastVerified = &usage
}

func usageObservation(raw json.RawMessage) map[string]any {
	obj, err := decodeObject(raw)
	if err != nil {
		return map[string]any{"invalid_object": true}
	}
	result := make(map[string]any)
	for _, field := range []string{"input_tokens", "output_tokens", "cache_read_input_tokens", "cache_creation_input_tokens"} {
		if value, exists := obj[field]; exists {
			result[field] = observedCounter(value)
		}
	}
	for field, children := range map[string][]string{
		"cache_creation":        {"ephemeral_5m_input_tokens", "ephemeral_1h_input_tokens"},
		"server_tool_use":       {"web_search_requests", "web_fetch_requests"},
		"output_tokens_details": {"thinking_tokens"},
	} {
		if raw, exists := obj[field]; exists {
			if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
				result[field] = nil
				continue
			}
			child, err := decodeObject(raw)
			if err != nil {
				result[field] = "invalid"
				continue
			}
			counts := make(map[string]any)
			for _, name := range children {
				if value, exists := child[name]; exists {
					counts[name] = observedCounter(value)
				}
			}
			result[field] = counts
		}
	}
	// Count categories without including arbitrary field names or values.
	result["category_count"] = len(obj)
	return result
}

func observedCounter(raw json.RawMessage) any {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil
	}
	n, err := usageCounter(raw)
	if err != nil {
		return "invalid"
	}
	return n
}

func diagnosticID(id string) string {
	if len(id) > 128 {
		return "invalid"
	}
	for _, ch := range id {
		if !(ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || ch == '-' || ch == '_') {
			return "invalid"
		}
	}
	return id
}

func diagnosticReason(err error) string {
	if err == nil {
		return "unverifiable_response"
	}
	// Decoder/transport errors can contain arbitrary upstream data. Only these
	// fixed validation errors are permitted verbatim in operational logs.
	switch err.Error() {
	case "unsupported usage category", "invalid usage counter", "incomplete usage", "cache TTL usage missing", "cache TTL categories unsupported", "cache usage inconsistent", "unsupported paid tool usage", "unsupported output usage breakdown", "invalid output usage breakdown", "unsupported usage service tier", "unsupported inference residency", "unsupported stream usage category", "missing or impossible stream output usage", "invalid stream identity", "duplicate message_start", "unexpected message_delta", "truncated stream", "event after message_stop", "event type mismatch", "invalid content sequence", "unsupported stream event", "stream event too large":
		return err.Error()
	default:
		return "unverifiable_response"
	}
}

func (g *Gateway) reportUnresolved(d UsageDiagnostic) {
	if g.Diagnostic != nil {
		g.Diagnostic(d)
		return
	}
	encoded, _ := json.Marshal(d)
	log.Printf("%s", encoded)
}
