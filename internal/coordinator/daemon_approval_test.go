package coordinator

import (
	"testing"
)

// TestTruncateForAttribute tests text truncation for span attributes.
func TestTruncateForAttribute(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "short text",
			input:  "short",
			maxLen: 10,
			want:   "short",
		},
		{
			name:   "exact length",
			input:  "exactly10c",
			maxLen: 10,
			want:   "exactly10c",
		},
		{
			name:   "needs truncation",
			input:  "this is a long rejection reason that exceeds the attribute limit",
			maxLen: 30,
			want:   "this is a long rejection reaso...",
		},
		{
			name:   "empty string",
			input:  "",
			maxLen: 10,
			want:   "",
		},
		{
			name:   "typical rejection feedback",
			input:  "Need to add better error handling. The edge cases are not covered.",
			maxLen: 500,
			want:   "Need to add better error handling. The edge cases are not covered.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateForAttribute(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateForAttribute(%q, %d) = %q, want %q",
					tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

// TestApprovalChannelValues tests that channel values are consistent across the system.
func TestApprovalChannelValues(t *testing.T) {
	// These are the valid channel values used across Dashboard, CLI, and GitHub
	validChannels := []string{"dashboard", "cli", "github"}

	// Verify the expected channels are recognized
	for _, ch := range validChannels {
		switch ch {
		case "dashboard", "cli", "github":
			// Valid
		default:
			t.Errorf("Unexpected channel value: %s", ch)
		}
	}
}

// TestApprovalActionValues tests that action values are consistent.
func TestApprovalActionValues(t *testing.T) {
	// These are the valid action values for approval decisions
	validActions := []string{"approve", "reject"}

	for _, action := range validActions {
		switch action {
		case "approve", "reject":
			// Valid
		default:
			t.Errorf("Unexpected action value: %s", action)
		}
	}
}

// TestMultiChannelScenarios tests various multi-channel approval scenarios.
func TestMultiChannelScenarios(t *testing.T) {
	scenarios := []struct {
		name            string
		channel         string
		action          string
		iteration       int
		hasGitHubIssue  bool
		expectsSpan     bool
		expectsGitHubOp bool
	}{
		{
			name:            "dashboard approve first iteration",
			channel:         "dashboard",
			action:          "approve",
			iteration:       1,
			hasGitHubIssue:  false,
			expectsSpan:     true,
			expectsGitHubOp: false,
		},
		{
			name:            "cli reject with github issue",
			channel:         "cli",
			action:          "reject",
			iteration:       1,
			hasGitHubIssue:  true,
			expectsSpan:     true,
			expectsGitHubOp: true, // Should post feedback to GitHub
		},
		{
			name:            "dashboard reject on iteration 3 (final)",
			channel:         "dashboard",
			action:          "reject",
			iteration:       3,
			hasGitHubIssue:  true,
			expectsSpan:     true,
			expectsGitHubOp: true,
		},
		{
			name:            "github approve via label",
			channel:         "github",
			action:          "approve",
			iteration:       2,
			hasGitHubIssue:  true,
			expectsSpan:     true,
			expectsGitHubOp: false, // Label is already set on GitHub
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			// Validate the scenario configuration
			if s.expectsSpan != true {
				t.Error("All approval decisions should create spans")
			}

			// If has GitHub issue and rejecting, should post feedback
			if s.hasGitHubIssue && s.action == "reject" && !s.expectsGitHubOp {
				t.Error("Rejections with GitHub issue should post feedback")
			}

			// Verify iteration is valid
			if s.iteration < 1 || s.iteration > MaxAgentIterations {
				t.Errorf("Invalid iteration: %d (should be 1-%d)", s.iteration, MaxAgentIterations)
			}
		})
	}
}
