package telemetry

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// Truncate safely truncates a string to maxLen runes (characters), appending "..." if truncated.
// Handles UTF-8 correctly to avoid breaking multi-byte characters.
func Truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}

	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}

	// For very short maxLen, just return what we can
	if maxLen <= 3 {
		return string(runes[:maxLen])
	}

	// Leave room for "..."
	targetLen := maxLen - 3
	return string(runes[:targetLen]) + "..."
}

// ErrorCategory represents categories of errors for filtering in traces.
type ErrorCategory string

const (
	ErrorCategoryParse   ErrorCategory = "parse_error"
	ErrorCategoryType    ErrorCategory = "type_error"
	ErrorCategoryModule  ErrorCategory = "module_error"
	ErrorCategoryRuntime ErrorCategory = "runtime_error"
	ErrorCategoryAPI     ErrorCategory = "api_error"
	ErrorCategoryTimeout ErrorCategory = "timeout"
	ErrorCategoryUnknown ErrorCategory = "unknown"
)

// CategorizeError determines the category of an error for trace filtering.
// Categories help users quickly filter traces by error type.
func CategorizeError(err error) string {
	if err == nil {
		return string(ErrorCategoryUnknown)
	}

	msg := strings.ToLower(err.Error())

	// Check for timeout first (most specific)
	if strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline exceeded") ||
		strings.Contains(msg, "context canceled") {
		return string(ErrorCategoryTimeout)
	}

	// Type errors - check before parse errors because "expected" can appear in both
	if strings.Contains(msg, "type") || strings.Contains(msg, "cannot unify") ||
		strings.Contains(msg, "mismatch") || strings.Contains(msg, "inference") {
		return string(ErrorCategoryType)
	}

	// Parse errors
	if strings.Contains(msg, "parse") || strings.Contains(msg, "syntax") ||
		strings.Contains(msg, "unexpected token") || strings.Contains(msg, "expected") {
		return string(ErrorCategoryParse)
	}

	// Module/import errors
	if strings.Contains(msg, "module") || strings.Contains(msg, "import") ||
		strings.Contains(msg, "not found") || strings.Contains(msg, "ldr") {
		return string(ErrorCategoryModule)
	}

	// API errors
	if strings.Contains(msg, "api") || strings.Contains(msg, "http") ||
		strings.Contains(msg, "request") || strings.Contains(msg, "response") ||
		strings.Contains(msg, "status code") || strings.Contains(msg, "rate limit") {
		return string(ErrorCategoryAPI)
	}

	// Runtime errors
	if strings.Contains(msg, "runtime") || strings.Contains(msg, "panic") ||
		strings.Contains(msg, "nil pointer") || strings.Contains(msg, "index out") {
		return string(ErrorCategoryRuntime)
	}

	return string(ErrorCategoryUnknown)
}

// ShortHash returns a short hex-encoded hash of the content.
// Useful for deduplication without storing full content.
func ShortHash(content string, length int) string {
	if length <= 0 {
		return ""
	}

	hash := sha256.Sum256([]byte(content))
	hexHash := hex.EncodeToString(hash[:])

	if length >= len(hexHash) {
		return hexHash
	}
	return hexHash[:length]
}

// LineSnippet extracts a snippet of source code around the given line number.
// Returns at most maxLen characters, centered on the target line.
func LineSnippet(source string, lineNum int, maxLen int) string {
	if source == "" || lineNum <= 0 || maxLen <= 0 {
		return ""
	}

	lines := strings.Split(source, "\n")
	if lineNum > len(lines) {
		return ""
	}

	// Get the target line (1-indexed)
	targetLine := lines[lineNum-1]

	// Trim whitespace and truncate
	snippet := strings.TrimSpace(targetLine)
	if len(snippet) > maxLen {
		return Truncate(snippet, maxLen)
	}

	return snippet
}
