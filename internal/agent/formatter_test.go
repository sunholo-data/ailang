package agent

import (
	"strings"
	"testing"
)

// TestFormatResult tests result formatting
func TestFormatResult(t *testing.T) {
	result := &DirectiveResult{
		Success:    true,
		DurationMS: 4500,
		NumTurns:   3,
		Cost:       0.0125,
		SessionID:  "test-session-123",
		Output:     "Task completed successfully!",
		FilesCreated: []string{
			"hello.txt",
			"world.txt",
		},
		TokensUsed: TokenUsage{
			InputTokens:  1000,
			OutputTokens: 500,
		},
		Workspace: "/tmp/test-workspace",
	}

	formatted := FormatResult(result)

	// Verify key elements are present
	tests := []struct {
		name     string
		contains string
	}{
		{"Success header", "✅ Directive Completed Successfully"},
		{"Duration", "4.5s"},
		{"Cost", "$0.0125"},
		{"Turns", "3"},
		{"Session ID", "test-session-123"},
		{"Output", "Task completed successfully!"},
		{"File 1", "hello.txt"},
		{"File 2", "world.txt"},
		{"Input tokens", "1,000 tokens"},
		{"Output tokens", "500 tokens"},
		{"Workspace", "/tmp/test-workspace"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(formatted, tt.contains) {
				t.Errorf("Expected formatted result to contain %q, but it didn't.\nFormatted:\n%s",
					tt.contains, formatted)
			}
		})
	}
}

// TestFormatResult_Error tests error formatting
func TestFormatResult_Error(t *testing.T) {
	result := &DirectiveResult{
		Success:    false,
		DurationMS: 2000,
		NumTurns:   1,
		Cost:       0.005,
		SessionID:  "error-session",
		Error:      "Execution timed out after 300 seconds",
		TokensUsed: TokenUsage{
			InputTokens:  500,
			OutputTokens: 0,
		},
	}

	formatted := FormatResult(result)

	// Verify error formatting
	if !strings.Contains(formatted, "❌ Directive Failed") {
		t.Error("Expected failure header")
	}

	if !strings.Contains(formatted, "Execution timed out") {
		t.Error("Expected error message")
	}

	// Should have error in code block
	if !strings.Contains(formatted, "```\nExecution timed out") {
		t.Error("Expected error in code block")
	}
}

// TestFormatResultCompact tests compact formatting
func TestFormatResultCompact(t *testing.T) {
	tests := []struct {
		name   string
		result *DirectiveResult
		want   string
	}{
		{
			name: "Success",
			result: &DirectiveResult{
				Success:      true,
				DurationMS:   1500,
				Cost:         0.003,
				NumTurns:     2,
				FilesCreated: []string{"file1.txt", "file2.txt"},
			},
			want: "✅ Completed in 1.5s (cost: $0.0030, 2 turns, 2 files created)",
		},
		{
			name: "Failure",
			result: &DirectiveResult{
				Success:    false,
				DurationMS: 500,
				Cost:       0.001,
				NumTurns:   1,
			},
			want: "❌ Completed in 500ms (cost: $0.0010, 1 turns, 0 files created)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatResultCompact(tt.result)
			if got != tt.want {
				t.Errorf("FormatResultCompact() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestFormatDuration tests duration formatting
func TestFormatDuration(t *testing.T) {
	tests := []struct {
		ms   int
		want string
	}{
		{ms: 50, want: "50ms"},
		{ms: 500, want: "500ms"},
		{ms: 1000, want: "1.0s"},
		{ms: 1500, want: "1.5s"},
		{ms: 45000, want: "45.0s"},
		{ms: 60000, want: "1.0m"},
		{ms: 90000, want: "1.5m"},
		{ms: 300000, want: "5.0m"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatDuration(tt.ms)
			if got != tt.want {
				t.Errorf("formatDuration(%d) = %q, want %q", tt.ms, got, tt.want)
			}
		})
	}
}

// TestFormatNumber tests number formatting
func TestFormatNumber(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{n: 0, want: "0"},
		{n: 42, want: "42"},
		{n: 999, want: "999"},
		{n: 1000, want: "1,000"},
		{n: 1234, want: "1,234"},
		{n: 999999, want: "999,999"},
		{n: 1000000, want: "1,000,000"},
		{n: 1234567, want: "1,234,567"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatNumber(tt.n)
			if got != tt.want {
				t.Errorf("formatNumber(%d) = %q, want %q", tt.n, got, tt.want)
			}
		})
	}
}

// TestFormatResultWithTranscript tests transcript inclusion
func TestFormatResultWithTranscript(t *testing.T) {
	result := &DirectiveResult{
		Success:    true,
		DurationMS: 1000,
		Cost:       0.01,
		NumTurns:   2,
		SessionID:  "test",
		Output:     "Success!",
		Transcript: "Turn 1: User asked...\nTurn 2: Agent responded...",
	}

	formatted := FormatResultWithTranscript(result)

	// Should include standard formatting
	if !strings.Contains(formatted, "Success!") {
		t.Error("Expected output in result")
	}

	// Should include transcript
	if !strings.Contains(formatted, "Full Transcript") {
		t.Error("Expected transcript section")
	}

	if !strings.Contains(formatted, "Turn 1: User asked") {
		t.Error("Expected transcript content")
	}

	// Transcript should be in collapsible details
	if !strings.Contains(formatted, "<details>") {
		t.Error("Expected transcript in details tag")
	}
}

// TestFormatResult_WithCache tests cache token formatting
func TestFormatResult_WithCache(t *testing.T) {
	result := &DirectiveResult{
		Success:    true,
		DurationMS: 1000,
		Cost:       0.005,
		NumTurns:   1,
		SessionID:  "cache-test",
		Output:     "Done",
		TokensUsed: TokenUsage{
			InputTokens:              5000,
			OutputTokens:             1000,
			CacheReadInputTokens:     4000,
			CacheCreationInputTokens: 500,
		},
	}

	formatted := FormatResult(result)

	// Should show cache hits
	if !strings.Contains(formatted, "4,000 from cache") {
		t.Error("Expected cache read tokens")
	}

	// Should show cache creation
	if !strings.Contains(formatted, "Cache Created") {
		t.Error("Expected cache creation tokens")
	}
}
