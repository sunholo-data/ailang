package effects

import (
	"os"
	"testing"
)

// TestIsSensitiveVarName tests sensitive variable name detection
func TestIsSensitiveVarName(t *testing.T) {
	tests := []struct {
		name      string
		sensitive bool
	}{
		// Sensitive
		{"API_KEY", true},
		{"SECRET", true},
		{"TOKEN", true},
		{"PASSWORD", true},
		{"CREDENTIAL", true},
		{"CLIENT_SECRET", true},
		{"AUTH_TOKEN", true},
		{"DB_PASSWORD", true},
		{"api_key", true}, // Case-insensitive
		{"MySecret", true},

		// Not sensitive
		{"DEBUG", false},
		{"PATH", false},
		{"HOME", false},
		{"LANG", false},
		{"USER", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSensitiveVarName(tt.name)
			if result != tt.sensitive {
				t.Errorf("IsSensitiveVarName(%q) = %v, want %v", tt.name, result, tt.sensitive)
			}
		})
	}
}

// TestRedactEnvValue tests value redaction
func TestRedactEnvValue(t *testing.T) {
	// Ensure redaction is enabled for tests
	os.Setenv("AILANG_REDACT_ENV", "on")
	defer os.Unsetenv("AILANG_REDACT_ENV")

	tests := []struct {
		name     string
		varName  string
		value    string
		expected string
	}{
		// Sensitive variables should be redacted
		{"API key", "API_KEY", "sk-proj-abc123", "[REDACTED]"},
		{"Secret", "SECRET", "my-secret-value", "[REDACTED]"},
		{"Token", "AUTH_TOKEN", "bearer-token-123", "[REDACTED]"},
		{"Password", "PASSWORD", "hunter2", "[REDACTED]"},

		// Non-sensitive variables should pass through
		{"Debug flag", "DEBUG", "true", "true"},
		{"Path", "PATH", "/usr/bin:/bin", "/usr/bin:/bin"},
		{"Home", "HOME", "/Users/mark", "/Users/mark"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactEnvValue(tt.varName, tt.value)
			if result != tt.expected {
				t.Errorf("RedactEnvValue(%q, %q) = %q, want %q", tt.varName, tt.value, result, tt.expected)
			}
		})
	}
}

// TestRedactEnvValue_Disabled tests redaction can be disabled
func TestRedactEnvValue_Disabled(t *testing.T) {
	// Disable redaction
	os.Setenv("AILANG_REDACT_ENV", "off")
	defer os.Unsetenv("AILANG_REDACT_ENV")

	// Reload redaction setting (normally happens at package init)
	original := redactionEnabled
	redactionEnabled = (os.Getenv("AILANG_REDACT_ENV") != "off")
	defer func() { redactionEnabled = original }()

	// Sensitive value should NOT be redacted when disabled
	result := RedactEnvValue("API_KEY", "sk-proj-secret123")
	if result != "sk-proj-secret123" {
		t.Errorf("expected value to pass through when redaction disabled, got %q", result)
	}
}

// TestRedactErrorMessage tests error message redaction
func TestRedactErrorMessage(t *testing.T) {
	// Ensure redaction is enabled
	os.Setenv("AILANG_REDACT_ENV", "on")
	defer os.Unsetenv("AILANG_REDACT_ENV")
	original := redactionEnabled
	redactionEnabled = true
	defer func() { redactionEnabled = original }()

	tests := []struct {
		name     string
		message  string
		contains string // redacted message should contain this
		notContain string // redacted message should NOT contain this
	}{
		{
			name:       "OpenAI API key",
			message:    "failed to authenticate with key sk-proj-abc123def456ghi789",
			contains:   "[REDACTED]",
			notContain: "sk-proj-abc123",
		},
		{
			name:       "Anthropic API key",
			message:    "invalid token: sk-ant-xyz789abc456def123",
			contains:   "[REDACTED]",
			notContain: "sk-ant-xyz789",
		},
		{
			name:       "Long token",
			message:    "bearer token 1234567890abcdefghij1234567890",
			contains:   "[REDACTED]",
			notContain: "1234567890abcdefghij",
		},
		{
			name:       "Base64 string",
			message:    "encoded secret: YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXo=",
			contains:   "[REDACTED]",
			notContain: "YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXo=",
		},
		{
			name:       "Safe message",
			message:    "variable not found in environment",
			contains:   "variable not found",
			notContain: "[REDACTED]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactErrorMessage(tt.message)

			if !contains(result, tt.contains) {
				t.Errorf("RedactErrorMessage() = %q, should contain %q", result, tt.contains)
			}

			if tt.notContain != "" && contains(result, tt.notContain) {
				t.Errorf("RedactErrorMessage() = %q, should NOT contain %q", result, tt.notContain)
			}
		})
	}
}

// TestRedactEnvSnapshot tests snapshot redaction
func TestRedactEnvSnapshot(t *testing.T) {
	// Ensure redaction is enabled
	os.Setenv("AILANG_REDACT_ENV", "on")
	defer os.Unsetenv("AILANG_REDACT_ENV")
	original := redactionEnabled
	redactionEnabled = true
	defer func() { redactionEnabled = original }()

	snapshot := map[string]string{
		"API_KEY":  "sk-proj-secret123",
		"SECRET":   "my-secret",
		"DEBUG":    "true",
		"PATH":     "/usr/bin:/bin",
		"PASSWORD": "hunter2",
	}

	redacted := RedactEnvSnapshot(snapshot)

	// Sensitive values should be redacted
	if redacted["API_KEY"] != "[REDACTED]" {
		t.Errorf("API_KEY should be redacted, got %q", redacted["API_KEY"])
	}
	if redacted["SECRET"] != "[REDACTED]" {
		t.Errorf("SECRET should be redacted, got %q", redacted["SECRET"])
	}
	if redacted["PASSWORD"] != "[REDACTED]" {
		t.Errorf("PASSWORD should be redacted, got %q", redacted["PASSWORD"])
	}

	// Non-sensitive values should pass through
	if redacted["DEBUG"] != "true" {
		t.Errorf("DEBUG should not be redacted, got %q", redacted["DEBUG"])
	}
	if redacted["PATH"] != "/usr/bin:/bin" {
		t.Errorf("PATH should not be redacted, got %q", redacted["PATH"])
	}

	// Original snapshot should be unchanged
	if snapshot["API_KEY"] != "sk-proj-secret123" {
		t.Error("RedactEnvSnapshot should not mutate original")
	}
}

// TestFormatEnvError tests error formatting
func TestFormatEnvError(t *testing.T) {
	// Ensure redaction is enabled
	os.Setenv("AILANG_REDACT_ENV", "on")
	defer os.Unsetenv("AILANG_REDACT_ENV")
	original := redactionEnabled
	redactionEnabled = true
	defer func() { redactionEnabled = original }()

	tests := []struct {
		name      string
		varName   string
		errMsg    string
		wantContains []string
	}{
		{
			name:    "Not found error",
			varName: "API_KEY",
			errMsg:  "not found",
			wantContains: []string{
				"not found",
				"export API_KEY=",
			},
		},
		{
			name:    "Not allowed error",
			varName: "SECRET_TOKEN",
			errMsg:  "not in allowlist",
			wantContains: []string{
				"not in allowlist",
				"--allow-env SECRET_TOKEN",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatEnvError(tt.varName, &envError{msg: tt.errMsg})

			for _, want := range tt.wantContains {
				if !contains(result, want) {
					t.Errorf("FormatEnvError() = %q, should contain %q", result, want)
				}
			}
		})
	}
}

// Helper error type for testing
type envError struct {
	msg string
}

func (e *envError) Error() string {
	return e.msg
}
