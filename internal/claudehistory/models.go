// Package claudehistory reads Claude Code conversation history from ~/.claude/projects/
package claudehistory

import (
	"time"
)

// Session represents a Claude Code conversation session.
type Session struct {
	ID          string    `json:"id"`
	ProjectPath string    `json:"project_path"` // Escaped path (e.g., "-Users-mark-dev-sunholo-ailang")
	ProjectName string    `json:"project_name"` // Human-readable name
	Messages    []Message `json:"messages"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	TurnCount   int       `json:"turn_count"`
	TotalIn     int       `json:"total_in"`    // Total input tokens
	TotalOut    int       `json:"total_out"`   // Total output tokens
	CacheRead   int       `json:"cache_read"`  // Total cache read tokens
	CacheWrite  int       `json:"cache_write"` // Total cache creation tokens
	Model       string    `json:"model"`       // Primary model used
	GitBranch   string    `json:"git_branch"`  // Git branch if available
	Cwd         string    `json:"cwd"`         // Working directory
}

// SessionMeta is a lightweight summary of a session (for listing).
type SessionMeta struct {
	ID        string    `json:"id"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	TurnCount int       `json:"turn_count"`
	TotalIn   int       `json:"total_in"`
	TotalOut  int       `json:"total_out"`
	Model     string    `json:"model"`
	FilePath  string    `json:"file_path"` // Full path to JSONL file
	FileSize  int64     `json:"file_size"` // Size in bytes
}

// Project represents a Claude Code project directory.
type Project struct {
	Path          string        `json:"path"`           // Escaped path
	Name          string        `json:"name"`           // Human-readable name
	Sessions      []SessionMeta `json:"sessions"`       // Session summaries
	TotalSessions int           `json:"total_sessions"` // Count of sessions
}

// Message represents a single message in a conversation.
type Message struct {
	UUID       string         `json:"uuid"`
	ParentUUID string         `json:"parent_uuid,omitempty"`
	SessionID  string         `json:"session_id"`
	Type       string         `json:"type"` // "user" or "assistant"
	Timestamp  time.Time      `json:"timestamp"`
	Model      string         `json:"model,omitempty"`
	MessageID  string         `json:"message_id,omitempty"` // Anthropic message ID (msg_...)
	RequestID  string         `json:"request_id,omitempty"` // Anthropic request ID (req_...)
	Content    []ContentBlock `json:"content"`
	Usage      *TokenUsage    `json:"usage,omitempty"`
	GitBranch  string         `json:"git_branch,omitempty"`
	Cwd        string         `json:"cwd,omitempty"`
	StopReason string         `json:"stop_reason,omitempty"`
}

// ContentBlock represents a block of content within a message.
// Only one of the content fields will be populated based on Type.
type ContentBlock struct {
	Type string `json:"type"` // "thinking", "text", "tool_use", "tool_result"

	// For type="text"
	Text string `json:"text,omitempty"`

	// For type="thinking"
	Thinking string `json:"thinking,omitempty"`

	// For type="tool_use"
	ToolUse *ToolUseBlock `json:"tool_use,omitempty"`

	// For type="tool_result"
	ToolResult *ToolResultBlock `json:"tool_result,omitempty"`
}

// ToolUseBlock represents a tool call made by Claude.
type ToolUseBlock struct {
	ID    string      `json:"id"`    // Tool use ID (toolu_...)
	Name  string      `json:"name"`  // Tool name (e.g., "Read", "Write", "Bash")
	Input interface{} `json:"input"` // Tool input (varies by tool)
}

// ToolResultBlock represents the result of a tool call.
type ToolResultBlock struct {
	ToolUseID string `json:"tool_use_id"` // References ToolUseBlock.ID
	Content   string `json:"content"`     // Result content
	IsError   bool   `json:"is_error"`    // Whether the tool call failed
}

// TokenUsage tracks token consumption for a message.
type TokenUsage struct {
	InputTokens         int `json:"input_tokens"`
	OutputTokens        int `json:"output_tokens"`
	CacheReadTokens     int `json:"cache_read_input_tokens,omitempty"`
	CacheCreationTokens int `json:"cache_creation_input_tokens,omitempty"`
}

// JSONLEntry represents a raw entry from the Claude Code JSONL file.
// This is the format stored on disk.
type JSONLEntry struct {
	SessionID  string `json:"sessionId"`
	Type       string `json:"type"` // "user", "assistant"
	ParentUUID string `json:"parentUuid,omitempty"`
	Timestamp  string `json:"timestamp"`
	UUID       string `json:"uuid,omitempty"`
	Message    *struct {
		Model   string `json:"model,omitempty"`
		ID      string `json:"id,omitempty"` // msg_...
		Role    string `json:"role,omitempty"`
		Content []struct {
			Type     string      `json:"type"`
			Text     string      `json:"text,omitempty"`
			Thinking string      `json:"thinking,omitempty"`
			ID       string      `json:"id,omitempty"`   // For tool_use
			Name     string      `json:"name,omitempty"` // For tool_use
			Input    interface{} `json:"input,omitempty"`
			// For tool_result
			ToolUseID string `json:"tool_use_id,omitempty"`
			Content   string `json:"content,omitempty"`
			IsError   bool   `json:"is_error,omitempty"`
		} `json:"content,omitempty"`
		StopReason string `json:"stop_reason,omitempty"`
		Usage      *struct {
			InputTokens          int `json:"input_tokens"`
			OutputTokens         int `json:"output_tokens"`
			CacheReadInputTokens int `json:"cache_read_input_tokens,omitempty"`
			CacheCreationTokens  int `json:"cache_creation_input_tokens,omitempty"`
		} `json:"usage,omitempty"`
	} `json:"message,omitempty"`
	RequestID string `json:"requestId,omitempty"`
	GitBranch string `json:"gitBranch,omitempty"`
	Cwd       string `json:"cwd,omitempty"`
}
