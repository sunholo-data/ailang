package server

import (
	"bytes"
	"encoding/json"
	"testing"
)

// --- Payload parsing tests ---

func TestClaudeHookPayload_AllFields(t *testing.T) {
	raw := `{
		"session_id": "s1",
		"transcript_path": "/path/to/transcript",
		"cwd": "/home/user",
		"permission_mode": "plan",
		"hook_event_name": "PreToolUse",
		"agent_id": "agent-1",
		"agent_type": "general-purpose",
		"tool_name": "Bash",
		"tool_input": {"command": "ls"},
		"tool_response": "output",
		"tool_use_id": "tu-1",
		"source": "vscode",
		"model": "claude-opus-4-6"
	}`

	var p ClaudeHookPayload
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if p.SessionID != "s1" {
		t.Errorf("SessionID: want s1, got %s", p.SessionID)
	}
	if p.TranscriptPath != "/path/to/transcript" {
		t.Errorf("TranscriptPath mismatch")
	}
	if p.Cwd != "/home/user" {
		t.Errorf("Cwd mismatch")
	}
	if p.PermissionMode != "plan" {
		t.Errorf("PermissionMode mismatch")
	}
	if p.HookEventName != "PreToolUse" {
		t.Errorf("HookEventName mismatch")
	}
	if p.AgentID != "agent-1" {
		t.Errorf("AgentID mismatch")
	}
	if p.AgentType != "general-purpose" {
		t.Errorf("AgentType mismatch")
	}
	if p.ToolName != "Bash" {
		t.Errorf("ToolName mismatch")
	}
	if p.ToolUseID != "tu-1" {
		t.Errorf("ToolUseID mismatch")
	}
	if p.Source != "vscode" {
		t.Errorf("Source mismatch")
	}
	if p.Model != "claude-opus-4-6" {
		t.Errorf("Model mismatch")
	}
	if p.ToolInput == nil {
		t.Error("ToolInput should not be nil")
	}
	if p.ToolResponse == nil {
		t.Error("ToolResponse should not be nil")
	}
}

func TestClaudeHookPayload_OmittedFields(t *testing.T) {
	raw := `{"session_id":"s1","hook_event_name":"Stop","cwd":"/tmp"}`
	var p ClaudeHookPayload
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if p.ToolName != "" {
		t.Errorf("ToolName should be empty, got %s", p.ToolName)
	}
	if p.AgentID != "" {
		t.Errorf("AgentID should be empty, got %s", p.AgentID)
	}
	if p.ToolInput != nil {
		t.Errorf("ToolInput should be nil")
	}
}

func TestClaudeHookPayload_CorrelationIDsNotInJSON(t *testing.T) {
	// Correlation IDs use `json:"-"` so they must not appear in JSON output
	p := ClaudeHookPayload{
		SessionID:     "s1",
		HookEventName: "SessionStart",
		TaskID:        "task-secret",
		ChainID:       "chain-secret",
	}
	body, _ := json.Marshal(p)
	if bytes.Contains(body, []byte("task-secret")) {
		t.Error("TaskID should not be in JSON output (has json:\"-\" tag)")
	}
	if bytes.Contains(body, []byte("chain-secret")) {
		t.Error("ChainID should not be in JSON output (has json:\"-\" tag)")
	}
}
