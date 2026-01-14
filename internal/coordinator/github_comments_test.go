package coordinator

import (
	"strings"
	"testing"
	"time"
)

// TestIsBotUser tests bot detection with various patterns
func TestIsBotUser_DefaultPatterns(t *testing.T) {
	tests := []struct {
		name     string
		username string
		wantBot  bool
	}{
		{
			name:     "github actions bot",
			username: "github-actions",
			wantBot:  true,
		},
		{
			name:     "dependabot",
			username: "dependabot",
			wantBot:  true,
		},
		{
			name:     "renovate bot",
			username: "renovate",
			wantBot:  true,
		},
		{
			name:     "bot with bracket suffix",
			username: "some-bot[bot]",
			wantBot:  true,
		},
		{
			name:     "codecov bot",
			username: "codecov",
			wantBot:  true,
		},
		{
			name:     "stale bot",
			username: "stale",
			wantBot:  true,
		},
		{
			name:     "ailang agent",
			username: "ailang-agent",
			wantBot:  true,
		},
		{
			name:     "our agent account",
			username: "sunholo-voight-kampff",
			wantBot:  true,
		},
		{
			name:     "regular human user",
			username: "alice-developer",
			wantBot:  false,
		},
		{
			name:     "another human user",
			username: "bob-smith",
			wantBot:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsBotUser(tt.username)
			if got != tt.wantBot {
				t.Errorf("IsBotUser(%q) = %v, want %v", tt.username, got, tt.wantBot)
			}
		})
	}
}

// TestIsBotUser_CaseInsensitive tests bot detection is case-insensitive
func TestIsBotUser_CaseInsensitive(t *testing.T) {
	tests := []struct {
		username string
	}{
		{"GITHUB-ACTIONS"},
		{"GitHub-Actions"},
		{"github-ACTIONS"},
		{"DEPENDABOT"},
		{"DependaBot"},
	}

	for _, tt := range tests {
		t.Run(tt.username, func(t *testing.T) {
			if !IsBotUser(tt.username) {
				t.Errorf("IsBotUser(%q) = false, want true (case-insensitive)", tt.username)
			}
		})
	}
}

// TestIsBotUser_AdditionalPatterns tests custom additional patterns
func TestIsBotUser_AdditionalPatterns(t *testing.T) {
	tests := []struct {
		name     string
		username string
		patterns []string
		wantBot  bool
	}{
		{
			name:     "custom pattern match",
			username: "my-custom-bot",
			patterns: []string{"my-custom"},
			wantBot:  true,
		},
		{
			name:     "multiple custom patterns first match",
			username: "service-reader",
			patterns: []string{"service-", "oauth-"},
			wantBot:  true,
		},
		{
			name:     "multiple custom patterns second match",
			username: "oauth-token-bot",
			patterns: []string{"service-", "oauth-"},
			wantBot:  true,
		},
		{
			name:     "no pattern match",
			username: "alice",
			patterns: []string{"bot-", "automation-"},
			wantBot:  false,
		},
		{
			name:     "case insensitive custom pattern",
			username: "MY-CUSTOM-BOT",
			patterns: []string{"my-custom"},
			wantBot:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsBotUser(tt.username, tt.patterns...)
			if got != tt.wantBot {
				t.Errorf("IsBotUser(%q, %v) = %v, want %v", tt.username, tt.patterns, got, tt.wantBot)
			}
		})
	}
}

// TestExtractFeedbackFromComments_EmptyList tests empty comment list
func TestExtractFeedbackFromComments_EmptyList(t *testing.T) {
	feedback := ExtractFeedbackFromComments([]IssueComment{})
	if feedback != "" {
		t.Errorf("ExtractFeedbackFromComments([]) = %q, want empty string", feedback)
	}
}

// TestExtractFeedbackFromComments_SingleComment tests single comment
func TestExtractFeedbackFromComments_SingleComment(t *testing.T) {
	comments := []IssueComment{
		{
			ID:        1,
			Body:      "This is feedback",
			Author:    "alice",
			CreatedAt: time.Now(),
		},
	}

	feedback := ExtractFeedbackFromComments(comments)
	if feedback != "This is feedback" {
		t.Errorf("ExtractFeedbackFromComments returned %q, want 'This is feedback'", feedback)
	}
}

// TestExtractFeedbackFromComments_MultipleComments tests multiple comments are joined
func TestExtractFeedbackFromComments_MultipleComments(t *testing.T) {
	comments := []IssueComment{
		{
			ID:        1,
			Body:      "First feedback",
			Author:    "alice",
			CreatedAt: time.Now(),
		},
		{
			ID:        2,
			Body:      "Second feedback",
			Author:    "bob",
			CreatedAt: time.Now(),
		},
		{
			ID:        3,
			Body:      "Third feedback",
			Author:    "charlie",
			CreatedAt: time.Now(),
		},
	}

	feedback := ExtractFeedbackFromComments(comments)

	// Verify all comments are included
	if !strings.Contains(feedback, "First feedback") {
		t.Error("First feedback missing")
	}
	if !strings.Contains(feedback, "Second feedback") {
		t.Error("Second feedback missing")
	}
	if !strings.Contains(feedback, "Third feedback") {
		t.Error("Third feedback missing")
	}

	// Verify separator is present
	if !strings.Contains(feedback, "---") {
		t.Error("Separator missing between comments")
	}
}

// TestExtractFeedbackFromComments_PreservesFormatting tests that formatting is preserved
func TestExtractFeedbackFromComments_PreservesFormatting(t *testing.T) {
	comments := []IssueComment{
		{
			ID:        1,
			Body:      "Need to:\n- Add error handling\n- Improve tests",
			Author:    "alice",
			CreatedAt: time.Now(),
		},
	}

	feedback := ExtractFeedbackFromComments(comments)

	// Verify multi-line formatting is preserved
	if !strings.Contains(feedback, "\n") {
		t.Error("Multi-line formatting not preserved")
	}
	if !strings.Contains(feedback, "Add error handling") {
		t.Error("Content missing")
	}
}

// TestExtractFeedbackFromComments_LargeComment tests handling of large comments
func TestExtractFeedbackFromComments_LargeComment(t *testing.T) {
	largeBody := strings.Repeat("This is a large comment body. ", 100)

	comments := []IssueComment{
		{
			ID:        1,
			Body:      largeBody,
			Author:    "alice",
			CreatedAt: time.Now(),
		},
	}

	feedback := ExtractFeedbackFromComments(comments)

	if feedback != largeBody {
		t.Error("Large comment content not preserved")
	}
}

// TestIsBotUserEmptyUsername tests empty username
func TestIsBotUserEmptyUsername(t *testing.T) {
	if IsBotUser("") {
		t.Error("IsBotUser(\"\") should return false for empty username")
	}
}

// TestIsBotUserWhitespaceUsername tests whitespace-only username
func TestIsBotUserWhitespaceUsername(t *testing.T) {
	if IsBotUser("   ") {
		t.Error("IsBotUser(\"   \") should return false for whitespace username")
	}
}

// TestIsBotUserSingleCharacter tests single character usernames
func TestIsBotUserSingleCharacter(t *testing.T) {
	if IsBotUser("a") {
		t.Error("Single character should not be detected as bot")
	}
}

// TestExtractFeedbackFromComments_NilSlice tests nil slice handling
func TestExtractFeedbackFromComments_NilSlice(t *testing.T) {
	// Should not panic on nil slice
	feedback := ExtractFeedbackFromComments(nil)
	if feedback != "" {
		t.Errorf("ExtractFeedbackFromComments(nil) = %q, want empty string", feedback)
	}
}

// TestExtractFeedbackFromComments_EmptyBodies tests comments with empty bodies
func TestExtractFeedbackFromComments_EmptyBodies(t *testing.T) {
	comments := []IssueComment{
		{
			ID:        1,
			Body:      "",
			Author:    "alice",
			CreatedAt: time.Now(),
		},
		{
			ID:        2,
			Body:      "Real feedback",
			Author:    "bob",
			CreatedAt: time.Now(),
		},
		{
			ID:        3,
			Body:      "",
			Author:    "charlie",
			CreatedAt: time.Now(),
		},
	}

	feedback := ExtractFeedbackFromComments(comments)

	// Should still include empty parts with separators
	if !strings.Contains(feedback, "Real feedback") {
		t.Error("Real feedback missing")
	}
}

// TestExtractFeedbackFromComments_SpecialCharacters tests special characters
func TestExtractFeedbackFromComments_SpecialCharacters(t *testing.T) {
	comments := []IssueComment{
		{
			ID:        1,
			Body:      "Fix: @user mentioned code with `backticks` and **bold**",
			Author:    "alice",
			CreatedAt: time.Now(),
		},
	}

	feedback := ExtractFeedbackFromComments(comments)

	if !strings.Contains(feedback, "@user") {
		t.Error("Mentions not preserved")
	}
	if !strings.Contains(feedback, "`backticks`") {
		t.Error("Backticks not preserved")
	}
	if !strings.Contains(feedback, "**bold**") {
		t.Error("Bold markdown not preserved")
	}
}
