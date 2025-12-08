package effects

import (
	"os"
	"regexp"
	"strings"
)

// Redaction patterns for sensitive environment variable values
var (
	// Patterns that match sensitive variable names
	sensitiveKeyPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)key`),        // API_KEY, SECRET_KEY, etc.
		regexp.MustCompile(`(?i)secret`),     // SECRET, CLIENT_SECRET, etc.
		regexp.MustCompile(`(?i)token`),      // TOKEN, AUTH_TOKEN, etc.
		regexp.MustCompile(`(?i)password`),   // PASSWORD, DB_PASSWORD, etc.
		regexp.MustCompile(`(?i)credential`), // CREDENTIAL, CREDENTIALS, etc.
	}

	// Redaction enabled by default (disable with AILANG_REDACT_ENV=off)
	redactionEnabled = os.Getenv("AILANG_REDACT_ENV") != "off"
)

// RedactEnvValue redacts sensitive environment variable values
//
// Checks if the variable name matches sensitive patterns (key, secret, token, password).
// If redaction is enabled and the name is sensitive, returns "[REDACTED]".
// Otherwise returns the original value.
//
// Redaction can be disabled with: AILANG_REDACT_ENV=off
//
// Parameters:
//   - name: Environment variable name
//   - value: Environment variable value
//
// Returns:
//   - Original value if not sensitive or redaction disabled
//   - "[REDACTED]" if sensitive and redaction enabled
//
// Example:
//
//	RedactEnvValue("API_KEY", "sk-proj-abc123")     // "[REDACTED]"
//	RedactEnvValue("DEBUG", "true")                  // "true"
//	RedactEnvValue("SECRET_TOKEN", "sensitive-data") // "[REDACTED]"
func RedactEnvValue(name, value string) string {
	if !redactionEnabled {
		return value
	}

	if IsSensitiveVarName(name) {
		return "[REDACTED]"
	}

	return value
}

// IsSensitiveVarName checks if a variable name matches sensitive patterns
//
// Checks against patterns: key, secret, token, password, credential (case-insensitive).
//
// Parameters:
//   - name: Environment variable name
//
// Returns:
//   - true if name matches any sensitive pattern
//   - false otherwise
//
// Example:
//
//	IsSensitiveVarName("API_KEY")      // true
//	IsSensitiveVarName("SECRET")       // true
//	IsSensitiveVarName("DEBUG")        // false
//	IsSensitiveVarName("PATH")         // false
func IsSensitiveVarName(name string) bool {
	for _, pattern := range sensitiveKeyPatterns {
		if pattern.MatchString(name) {
			return true
		}
	}
	return false
}

// RedactErrorMessage redacts sensitive values from error messages
//
// Scans error messages for patterns that look like sensitive values
// (long alphanumeric strings, base64-like patterns) and replaces them
// with "[REDACTED]".
//
// Patterns redacted:
//   - API keys: sk-proj-*, sk-ant-*, etc.
//   - Long tokens: strings with 20+ alphanumeric/dash/underscore chars
//   - Base64-like strings: 16+ chars of [A-Za-z0-9+/=]
//
// Parameters:
//   - message: Error message that may contain sensitive values
//
// Returns:
//   - Error message with sensitive values replaced by "[REDACTED]"
//
// Example:
//
//	RedactErrorMessage("failed to auth with key sk-proj-abc123def456")
//	// "failed to auth with key [REDACTED]"
func RedactErrorMessage(message string) string {
	if !redactionEnabled {
		return message
	}

	// Pattern 1: API key prefixes (OpenAI, Anthropic, etc.)
	apiKeyPattern := regexp.MustCompile(`(sk-proj-|sk-ant-|Bearer\s+)[A-Za-z0-9_-]{20,}`)
	message = apiKeyPattern.ReplaceAllString(message, "[REDACTED]")

	// Pattern 2: Long tokens (20+ alphanumeric with dashes/underscores)
	longTokenPattern := regexp.MustCompile(`\b[A-Za-z0-9_-]{20,}\b`)
	message = longTokenPattern.ReplaceAllString(message, "[REDACTED]")

	// Pattern 3: Base64-like strings (16+ chars)
	base64Pattern := regexp.MustCompile(`[A-Za-z0-9+/]{16,}={0,2}`)
	message = base64Pattern.ReplaceAllString(message, "[REDACTED]")

	return message
}

// RedactEnvSnapshot redacts sensitive values in an environment snapshot
//
// Returns a new map with sensitive values replaced by "[REDACTED]".
// Used for safe logging/debugging of environment snapshots.
//
// Parameters:
//   - snapshot: Original environment snapshot
//
// Returns:
//   - New map with sensitive values redacted
//
// Example:
//
//	original := map[string]string{
//	    "API_KEY": "sk-proj-secret",
//	    "DEBUG": "true",
//	}
//	redacted := RedactEnvSnapshot(original)
//	// {"API_KEY": "[REDACTED]", "DEBUG": "true"}
func RedactEnvSnapshot(snapshot map[string]string) map[string]string {
	if !redactionEnabled {
		return snapshot
	}

	redacted := make(map[string]string, len(snapshot))
	for key, value := range snapshot {
		redacted[key] = RedactEnvValue(key, value)
	}
	return redacted
}

// FormatEnvError formats environment errors with redaction
//
// Creates user-friendly error messages while redacting sensitive values.
// Provides actionable suggestions for common error cases.
//
// Parameters:
//   - varName: Environment variable name
//   - err: The error that occurred
//
// Returns:
//   - Formatted error message with redacted sensitive data
//
// Example:
//
//	FormatEnvError("API_KEY", errors.New("not found"))
//	// "environment variable 'API_KEY' not found. Set with: export API_KEY=<value>"
func FormatEnvError(varName string, err error) string {
	message := err.Error()

	// Add helpful suggestions
	if strings.Contains(message, "not found") || strings.Contains(message, "NotFound") {
		return message + "\n  Suggestion: Set with: export " + varName + "=<value>"
	}

	if strings.Contains(message, "not in allowlist") || strings.Contains(message, "NotAllowed") {
		return message + "\n  Suggestion: Add to allowlist: --allow-env " + varName
	}

	// Redact any sensitive values that might be in the error message
	return RedactErrorMessage(message)
}
