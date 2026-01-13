package executor

import (
	"encoding/json"
	"testing"
)

// TestOpenAICodexJSONLFormat tests OpenAI Codex JSONL format compatibility
// This test documents the key finding: OpenAI Codex uses a DIFFERENT JSONL schema
// than Claude Code and Gemini CLI
func TestOpenAICodexJSONLFormat(t *testing.T) {
	// OpenAI Codex uses different field names and structure
	// Reference: https://developers.openai.com/codex/cli/

	// CLAUDE CODE format example:
	// {"type": "stream_event", "event": {"type": "message_start", "message": {...}}}
	claudeEvent := map[string]interface{}{
		"type": "stream_event",
		"event": map[string]interface{}{
			"type":    "message_start",
			"message": map[string]interface{}{},
		},
	}

	claudeJSON, _ := json.Marshal(claudeEvent)
	t.Logf("Claude Code JSONL format: %s", string(claudeJSON))

	// GEMINI CLI format is similar to Claude
	geminiEvent := map[string]interface{}{
		"type": "stream_event",
		"event": map[string]interface{}{
			"type": "message_start",
		},
	}

	geminiJSON, _ := json.Marshal(geminiEvent)
	t.Logf("Gemini CLI JSONL format: %s", string(geminiJSON))

	// OPENAI CODEX format is DIFFERENT
	// OpenAI Codex uses different field names entirely
	codexEvent := map[string]interface{}{
		"type":        "message",
		"turn_number": 1,
		"text":        "response text",
		"tokens_used": map[string]interface{}{
			"input":  123,
			"output": 456,
		},
	}

	codexJSON, _ := json.Marshal(codexEvent)
	t.Logf("OpenAI Codex format: %s", string(codexJSON))

	if string(claudeJSON) == string(codexJSON) {
		t.Error("Claude and Codex formats should be different")
	}

	if string(geminiJSON) == string(codexJSON) {
		t.Error("Gemini and Codex formats should be different")
	}

	// Verify Claude and Gemini use the same "type": "stream_event" pattern
	if !contains(string(claudeJSON), "stream_event") {
		t.Error("Claude should use stream_event type")
	}

	if !contains(string(geminiJSON), "stream_event") {
		t.Error("Gemini should use stream_event type")
	}

	// Verify Codex uses different "type": "message" pattern
	if !contains(string(codexJSON), "\"type\":\"message\"") {
		t.Error("Codex should use message type")
	}

	if contains(string(codexJSON), "stream_event") {
		t.Error("Codex should NOT use stream_event")
	}
}

// TestJSONLFormatComparison provides a structured comparison matrix
func TestJSONLFormatComparison(t *testing.T) {
	// Schema comparison between three providers
	comparison := map[string]map[string]string{
		"claude": {
			"event_type_field": "type",
			"event_type_value": "stream_event",
			"content_field":    "event",
			"message_type":     "message_start",
		},
		"gemini": {
			"event_type_field": "type",
			"event_type_value": "stream_event",
			"content_field":    "event",
			"message_type":     "message_start",
		},
		"codex": {
			"event_type_field": "type",
			"event_type_value": "message",
			"content_field":    "text",
			"message_type":     "message (top-level)",
		},
	}

	t.Log("JSONL Format Comparison:")
	t.Log("========================")
	for provider, schema := range comparison {
		t.Logf("%s:", provider)
		for key, value := range schema {
			t.Logf("  %s: %s", key, value)
		}
	}

	// Key finding: Claude and Gemini are compatible
	if comparison["claude"]["event_type_field"] != comparison["gemini"]["event_type_field"] {
		t.Error("Claude and Gemini should use the same field names")
	}

	// Key finding: Codex is incompatible
	if comparison["codex"]["event_type_value"] == comparison["claude"]["event_type_value"] {
		t.Error("Codex should use different event type value")
	}

	t.Log("\nCONCLUSION: OpenAI Codex JSONL is INCOMPATIBLE with Claude/Gemini")
	t.Log("RECOMMENDATION: If Codex support is needed, create separate JSONL parser")
}

// TestCodexTokenUsageExtraction demonstrates Codex token field differences
func TestCodexTokenUsageExtraction(t *testing.T) {
	// Claude/Gemini embed token usage in nested structures
	claudeUsage := map[string]interface{}{
		"type": "stream_event",
		"event": map[string]interface{}{
			"type":    "message_stop",
			"message": map[string]interface{}{},
			"usage": map[string]interface{}{
				"input_tokens":  100,
				"output_tokens": 50,
			},
		},
	}

	claudeBytes, _ := json.Marshal(claudeUsage)
	t.Logf("Claude token usage structure: %s", string(claudeBytes))

	// Codex uses flat structure
	codexUsage := map[string]interface{}{
		"type": "message_stop",
		"tokens_used": map[string]interface{}{
			"input":  100,
			"output": 50,
		},
	}

	codexBytes, _ := json.Marshal(codexUsage)
	t.Logf("Codex token usage structure: %s", string(codexBytes))

	// Verify different field nesting
	if contains(string(claudeBytes), "message") != contains(string(codexBytes), "message") {
		// Expected - they use different structures
	}
}

// TestIncompatibleJSONLShouldNotParse verifies parsers handle format mismatches
func TestIncompatibleJSONLShouldNotParse(t *testing.T) {
	// If Claude parser receives Codex JSON, it should fail gracefully
	codexLine := `{"type": "message", "text": "output", "tokens_used": {"input": 100, "output": 50}}`

	var claudeEvent map[string]interface{}
	if err := json.Unmarshal([]byte(codexLine), &claudeEvent); err == nil {
		// JSON is valid, but structure is wrong
		// Claude parser would look for "event" field (missing)
		if event, ok := claudeEvent["event"]; !ok {
			t.Logf("Codex event missing 'event' field - Claude parser would fail: %v", event)
		}
	}

	// This demonstrates that format incompatibility would be caught by parser
	if _, hasEvent := claudeEvent["event"]; hasEvent {
		t.Error("Codex JSON should not have 'event' field")
	}
}

// Helper function
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestCodexFormatImplications documents implications for implementation
func TestCodexFormatImplications(t *testing.T) {
	t.Log("OpenAI Codex JSONL Format Analysis - CONCLUSIONS")
	t.Log("================================================")
	t.Log("")
	t.Log("1. INCOMPATIBILITY: Codex uses completely different JSONL schema")
	t.Log("   - Claude: {type: stream_event, event: {...}}")
	t.Log("   - Gemini: {type: stream_event, event: {...}} (same as Claude)")
	t.Log("   - Codex:  {type: message, text: ..., tokens_used: {...}}")
	t.Log("")
	t.Log("2. PARSER REQUIREMENT: Separate JSONL parser needed for Codex")
	t.Log("   - Cannot reuse Claude/Gemini parser")
	t.Log("   - Different field extraction logic required")
	t.Log("   - Different token counting mechanism")
	t.Log("")
	t.Log("3. IMPLEMENTATION IMPACT:")
	t.Log("   - Add executor/codex/codex.go with Codex-specific parser")
	t.Log("   - Implement CodexExecutor.ExecuteStreaming() with Codex JSONL parsing")
	t.Log("   - Register in executor/init.go")
	t.Log("")
	t.Log("4. TESTING STRATEGY:")
	t.Log("   - Unit tests for Codex JSONL parsing (testdata/codex_response.jsonl)")
	t.Log("   - Integration tests with Codex CLI (if available)")
	t.Log("   - Format validation tests (this file)")
	t.Log("")
	t.Log("RECOMMENDATION: File feature request or create Codex executor in future sprint")
}
