package claudehistory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewReader(t *testing.T) {
	r := NewReader()
	homeDir, _ := os.UserHomeDir()
	expected := filepath.Join(homeDir, ".claude", "projects")

	if r.BaseDir() != expected {
		t.Errorf("expected baseDir %s, got %s", expected, r.BaseDir())
	}
}

func TestNewReaderWithBase(t *testing.T) {
	r := NewReaderWithBase("/custom/path")
	if r.BaseDir() != "/custom/path" {
		t.Errorf("expected baseDir /custom/path, got %s", r.BaseDir())
	}
}

func TestUnescapePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"-Users-mark-dev-sunholo-ailang", "/Users/mark/dev/sunholo/ailang"},
		{"-home-user-project", "/home/user/project"},
		{"simple", "simple"},
	}

	for _, tc := range tests {
		result := unescapePath(tc.input)
		if result != tc.expected {
			t.Errorf("unescapePath(%q) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}

func TestConvertEntry(t *testing.T) {
	entry := JSONLEntry{
		SessionID:  "test-session-123",
		Type:       "assistant",
		ParentUUID: "parent-uuid",
		UUID:       "this-uuid",
		Timestamp:  "2026-01-25T10:00:00Z",
		RequestID:  "req_123",
		GitBranch:  "main",
		Cwd:        "/path/to/project",
		Message: &struct {
			Model   string `json:"model,omitempty"`
			ID      string `json:"id,omitempty"`
			Role    string `json:"role,omitempty"`
			Content []struct {
				Type      string      `json:"type"`
				Text      string      `json:"text,omitempty"`
				Thinking  string      `json:"thinking,omitempty"`
				ID        string      `json:"id,omitempty"`
				Name      string      `json:"name,omitempty"`
				Input     interface{} `json:"input,omitempty"`
				ToolUseID string      `json:"tool_use_id,omitempty"`
				Content   string      `json:"content,omitempty"`
				IsError   bool        `json:"is_error,omitempty"`
			} `json:"content,omitempty"`
			StopReason string `json:"stop_reason,omitempty"`
			Usage      *struct {
				InputTokens          int `json:"input_tokens"`
				OutputTokens         int `json:"output_tokens"`
				CacheReadInputTokens int `json:"cache_read_input_tokens,omitempty"`
				CacheCreationTokens  int `json:"cache_creation_input_tokens,omitempty"`
			} `json:"usage,omitempty"`
		}{
			Model: "claude-opus-4-5-20251101",
			ID:    "msg_123",
			Content: []struct {
				Type      string      `json:"type"`
				Text      string      `json:"text,omitempty"`
				Thinking  string      `json:"thinking,omitempty"`
				ID        string      `json:"id,omitempty"`
				Name      string      `json:"name,omitempty"`
				Input     interface{} `json:"input,omitempty"`
				ToolUseID string      `json:"tool_use_id,omitempty"`
				Content   string      `json:"content,omitempty"`
				IsError   bool        `json:"is_error,omitempty"`
			}{
				{Type: "thinking", Thinking: "Let me think..."},
				{Type: "text", Text: "Here is my response"},
				{Type: "tool_use", ID: "toolu_123", Name: "Read", Input: map[string]string{"file_path": "/test.txt"}},
			},
			StopReason: "end_turn",
			Usage: &struct {
				InputTokens          int `json:"input_tokens"`
				OutputTokens         int `json:"output_tokens"`
				CacheReadInputTokens int `json:"cache_read_input_tokens,omitempty"`
				CacheCreationTokens  int `json:"cache_creation_input_tokens,omitempty"`
			}{
				InputTokens:          100,
				OutputTokens:         50,
				CacheReadInputTokens: 1000,
			},
		},
	}

	msg := convertEntry(entry)

	if msg.SessionID != "test-session-123" {
		t.Errorf("expected sessionID 'test-session-123', got %q", msg.SessionID)
	}
	if msg.Type != "assistant" {
		t.Errorf("expected type 'assistant', got %q", msg.Type)
	}
	if msg.Model != "claude-opus-4-5-20251101" {
		t.Errorf("expected model 'claude-opus-4-5-20251101', got %q", msg.Model)
	}
	if len(msg.Content) != 3 {
		t.Errorf("expected 3 content blocks, got %d", len(msg.Content))
	}

	// Check thinking block
	if msg.Content[0].Type != "thinking" {
		t.Errorf("expected first block type 'thinking', got %q", msg.Content[0].Type)
	}
	if msg.Content[0].Thinking != "Let me think..." {
		t.Errorf("expected thinking content 'Let me think...', got %q", msg.Content[0].Thinking)
	}

	// Check text block
	if msg.Content[1].Type != "text" {
		t.Errorf("expected second block type 'text', got %q", msg.Content[1].Type)
	}
	if msg.Content[1].Text != "Here is my response" {
		t.Errorf("expected text content 'Here is my response', got %q", msg.Content[1].Text)
	}

	// Check tool_use block
	if msg.Content[2].Type != "tool_use" {
		t.Errorf("expected third block type 'tool_use', got %q", msg.Content[2].Type)
	}
	if msg.Content[2].ToolUse == nil {
		t.Fatal("expected ToolUse to be non-nil")
	}
	if msg.Content[2].ToolUse.Name != "Read" {
		t.Errorf("expected tool name 'Read', got %q", msg.Content[2].ToolUse.Name)
	}

	// Check usage
	if msg.Usage == nil {
		t.Fatal("expected Usage to be non-nil")
	}
	if msg.Usage.InputTokens != 100 {
		t.Errorf("expected 100 input tokens, got %d", msg.Usage.InputTokens)
	}
	if msg.Usage.OutputTokens != 50 {
		t.Errorf("expected 50 output tokens, got %d", msg.Usage.OutputTokens)
	}
	if msg.Usage.CacheReadTokens != 1000 {
		t.Errorf("expected 1000 cache read tokens, got %d", msg.Usage.CacheReadTokens)
	}
}

func TestReadSessionWithTestData(t *testing.T) {
	// Create a temp directory with test data
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "-test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	// Create a test JSONL file
	sessionFile := filepath.Join(projectDir, "test-session-abc.jsonl")
	f, err := os.Create(sessionFile)
	if err != nil {
		t.Fatalf("failed to create session file: %v", err)
	}

	// Write test entries
	entries := []JSONLEntry{
		{
			SessionID: "test-session-abc",
			Type:      "user",
			Timestamp: "2026-01-25T10:00:00Z",
			Message: &struct {
				Model   string `json:"model,omitempty"`
				ID      string `json:"id,omitempty"`
				Role    string `json:"role,omitempty"`
				Content []struct {
					Type      string      `json:"type"`
					Text      string      `json:"text,omitempty"`
					Thinking  string      `json:"thinking,omitempty"`
					ID        string      `json:"id,omitempty"`
					Name      string      `json:"name,omitempty"`
					Input     interface{} `json:"input,omitempty"`
					ToolUseID string      `json:"tool_use_id,omitempty"`
					Content   string      `json:"content,omitempty"`
					IsError   bool        `json:"is_error,omitempty"`
				} `json:"content,omitempty"`
				StopReason string `json:"stop_reason,omitempty"`
				Usage      *struct {
					InputTokens          int `json:"input_tokens"`
					OutputTokens         int `json:"output_tokens"`
					CacheReadInputTokens int `json:"cache_read_input_tokens,omitempty"`
					CacheCreationTokens  int `json:"cache_creation_input_tokens,omitempty"`
				} `json:"usage,omitempty"`
			}{
				Role: "user",
				Content: []struct {
					Type      string      `json:"type"`
					Text      string      `json:"text,omitempty"`
					Thinking  string      `json:"thinking,omitempty"`
					ID        string      `json:"id,omitempty"`
					Name      string      `json:"name,omitempty"`
					Input     interface{} `json:"input,omitempty"`
					ToolUseID string      `json:"tool_use_id,omitempty"`
					Content   string      `json:"content,omitempty"`
					IsError   bool        `json:"is_error,omitempty"`
				}{
					{Type: "text", Text: "Hello, Claude!"},
				},
			},
		},
		{
			SessionID: "test-session-abc",
			Type:      "assistant",
			Timestamp: "2026-01-25T10:00:05Z",
			Message: &struct {
				Model   string `json:"model,omitempty"`
				ID      string `json:"id,omitempty"`
				Role    string `json:"role,omitempty"`
				Content []struct {
					Type      string      `json:"type"`
					Text      string      `json:"text,omitempty"`
					Thinking  string      `json:"thinking,omitempty"`
					ID        string      `json:"id,omitempty"`
					Name      string      `json:"name,omitempty"`
					Input     interface{} `json:"input,omitempty"`
					ToolUseID string      `json:"tool_use_id,omitempty"`
					Content   string      `json:"content,omitempty"`
					IsError   bool        `json:"is_error,omitempty"`
				} `json:"content,omitempty"`
				StopReason string `json:"stop_reason,omitempty"`
				Usage      *struct {
					InputTokens          int `json:"input_tokens"`
					OutputTokens         int `json:"output_tokens"`
					CacheReadInputTokens int `json:"cache_read_input_tokens,omitempty"`
					CacheCreationTokens  int `json:"cache_creation_input_tokens,omitempty"`
				} `json:"usage,omitempty"`
			}{
				Model: "claude-3-opus",
				ID:    "msg_test",
				Role:  "assistant",
				Content: []struct {
					Type      string      `json:"type"`
					Text      string      `json:"text,omitempty"`
					Thinking  string      `json:"thinking,omitempty"`
					ID        string      `json:"id,omitempty"`
					Name      string      `json:"name,omitempty"`
					Input     interface{} `json:"input,omitempty"`
					ToolUseID string      `json:"tool_use_id,omitempty"`
					Content   string      `json:"content,omitempty"`
					IsError   bool        `json:"is_error,omitempty"`
				}{
					{Type: "thinking", Thinking: "This is a greeting"},
					{Type: "text", Text: "Hello! How can I help you?"},
				},
				Usage: &struct {
					InputTokens          int `json:"input_tokens"`
					OutputTokens         int `json:"output_tokens"`
					CacheReadInputTokens int `json:"cache_read_input_tokens,omitempty"`
					CacheCreationTokens  int `json:"cache_creation_input_tokens,omitempty"`
				}{
					InputTokens:  10,
					OutputTokens: 20,
				},
			},
			GitBranch: "main",
			Cwd:       "/test/project",
		},
	}

	encoder := json.NewEncoder(f)
	for _, entry := range entries {
		if err := encoder.Encode(entry); err != nil {
			t.Fatalf("failed to write entry: %v", err)
		}
	}
	f.Close()

	// Test reading
	r := NewReaderWithBase(tmpDir)

	// Test ListProjects
	projects, err := r.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects failed: %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(projects))
	}
	if projects[0].Path != "-test-project" {
		t.Errorf("expected project path '-test-project', got %q", projects[0].Path)
	}
	if projects[0].Name != "/test/project" {
		t.Errorf("expected project name '/test/project', got %q", projects[0].Name)
	}

	// Test ListSessions
	sessions, err := r.ListSessions("-test-project")
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].ID != "test-session-abc" {
		t.Errorf("expected session ID 'test-session-abc', got %q", sessions[0].ID)
	}

	// Test GetSession
	session, err := r.GetSession("test-session-abc")
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if session.ID != "test-session-abc" {
		t.Errorf("expected session ID 'test-session-abc', got %q", session.ID)
	}
	if len(session.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(session.Messages))
	}
	if session.Model != "claude-3-opus" {
		t.Errorf("expected model 'claude-3-opus', got %q", session.Model)
	}
	if session.TotalIn != 10 {
		t.Errorf("expected 10 total input tokens, got %d", session.TotalIn)
	}
	if session.TotalOut != 20 {
		t.Errorf("expected 20 total output tokens, got %d", session.TotalOut)
	}
	if session.GitBranch != "main" {
		t.Errorf("expected git branch 'main', got %q", session.GitBranch)
	}

	// Test GetMessagesByTimeRange
	start, _ := time.Parse(time.RFC3339, "2026-01-25T10:00:00Z")
	end, _ := time.Parse(time.RFC3339, "2026-01-25T10:00:03Z")
	rangeMessages, err := r.GetMessagesByTimeRange("test-session-abc", start, end)
	if err != nil {
		t.Fatalf("GetMessagesByTimeRange failed: %v", err)
	}
	if len(rangeMessages) != 1 {
		t.Errorf("expected 1 message in range, got %d", len(rangeMessages))
	}
}

func TestListProjectsEmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	r := NewReaderWithBase(tmpDir)

	projects, err := r.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects failed: %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("expected 0 projects for empty dir, got %d", len(projects))
	}
}

func TestListProjectsNonExistentDir(t *testing.T) {
	r := NewReaderWithBase("/nonexistent/path/that/does/not/exist")

	projects, err := r.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects should not error on nonexistent dir, got: %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("expected empty projects, got %d", len(projects))
	}
}

func TestGetSessionNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	r := NewReaderWithBase(tmpDir)

	_, err := r.GetSession("nonexistent-session")
	if err == nil {
		t.Error("expected error for nonexistent session")
	}
}

// TestWithRealData tests against actual Claude Code history if available.
// This test is skipped in CI but useful for local development.
func TestWithRealData(t *testing.T) {
	r := NewReader()

	// Check if Claude Code data exists
	if _, err := os.Stat(r.BaseDir()); os.IsNotExist(err) {
		t.Skip("Skipping real data test - no Claude Code history found")
	}

	projects, err := r.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects failed: %v", err)
	}

	if len(projects) == 0 {
		t.Skip("No projects found in Claude Code history")
	}

	t.Logf("Found %d projects", len(projects))

	// Test first project
	project := projects[0]
	t.Logf("Testing project: %s (%s)", project.Name, project.Path)

	sessions, err := r.ListSessions(project.Path)
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}

	if len(sessions) == 0 {
		t.Skip("No sessions found in first project")
	}

	t.Logf("Found %d sessions", len(sessions))

	// Test first session
	sessionMeta := sessions[0]
	t.Logf("Testing session: %s (model: %s, turns: %d)", sessionMeta.ID, sessionMeta.Model, sessionMeta.TurnCount)

	session, err := r.GetSession(sessionMeta.ID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}

	t.Logf("Session has %d messages, %d input tokens, %d output tokens",
		len(session.Messages), session.TotalIn, session.TotalOut)

	// Check that we got valid content blocks
	var thinkingCount, textCount, toolUseCount int
	for _, msg := range session.Messages {
		for _, block := range msg.Content {
			switch block.Type {
			case "thinking":
				thinkingCount++
			case "text":
				textCount++
			case "tool_use":
				toolUseCount++
			}
		}
	}

	t.Logf("Content blocks - thinking: %d, text: %d, tool_use: %d",
		thinkingCount, textCount, toolUseCount)
}
