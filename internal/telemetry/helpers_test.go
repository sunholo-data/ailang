package telemetry

import (
	"errors"
	"testing"
)

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"empty string", "", 10, ""},
		{"short string", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"needs truncation", "hello world", 8, "hello..."},
		{"very short max", "hello", 3, "hel"},
		{"zero max", "hello", 0, ""},
		{"negative max", "hello", -1, ""},
		{"unicode string", "hello 世界", 10, "hello 世界"},       // 8 runes, no truncation needed
		{"unicode needs truncation", "hello 世界!", 8, "hello..."},   // 9 runes, truncate
		{"long unicode", "世界世界世界世界世界", 8, "世界世界世..."},              // 10 runes, truncate to 5+...
		{"emoji", "hello 👋 world", 12, "hello 👋 w..."},             // 13 runes, truncate to 9+...
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Truncate(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("Truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

func TestCategorizeError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"nil error", nil, "unknown"},
		{"parse error", errors.New("parse error: unexpected token"), "parse_error"},
		{"syntax error", errors.New("syntax error at line 5"), "parse_error"},
		{"expected token", errors.New("expected ')' but got '}'"), "parse_error"},
		{"type mismatch", errors.New("type mismatch: expected int, got string"), "type_error"},
		{"cannot unify", errors.New("cannot unify types: Int and String"), "type_error"},
		{"module not found", errors.New("module not found: std/missing"), "module_error"},
		{"import error", errors.New("failed to import module"), "module_error"},
		{"ldr error", errors.New("LDR001: module resolution failed"), "module_error"},
		{"api error", errors.New("API request failed: status code 429"), "api_error"},
		{"http error", errors.New("http: server closed connection"), "api_error"},
		{"rate limit", errors.New("rate limit exceeded"), "api_error"},
		{"timeout", errors.New("operation timeout"), "timeout"},
		{"deadline exceeded", errors.New("context deadline exceeded"), "timeout"},
		{"context canceled", errors.New("context canceled"), "timeout"},
		{"runtime panic", errors.New("runtime panic: nil pointer dereference"), "runtime_error"},
		{"nil pointer", errors.New("nil pointer"), "runtime_error"},
		{"unknown error", errors.New("something went wrong"), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CategorizeError(tt.err)
			if result != tt.expected {
				t.Errorf("CategorizeError(%v) = %q, want %q", tt.err, result, tt.expected)
			}
		})
	}
}

func TestShortHash(t *testing.T) {
	tests := []struct {
		name    string
		content string
		length  int
	}{
		{"empty content", "", 8},
		{"short content", "hello", 8},
		{"long content", "hello world this is a longer string", 8},
		{"zero length", "hello", 0},
		{"negative length", "hello", -1},
		{"very long length", "hello", 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShortHash(tt.content, tt.length)

			// Check length constraints
			if tt.length <= 0 {
				if result != "" {
					t.Errorf("ShortHash with length %d should return empty string, got %q", tt.length, result)
				}
				return
			}

			if tt.length < 64 && len(result) != tt.length {
				t.Errorf("ShortHash length = %d, want %d", len(result), tt.length)
			}

			// Check determinism
			result2 := ShortHash(tt.content, tt.length)
			if result != result2 {
				t.Errorf("ShortHash not deterministic: %q != %q", result, result2)
			}

			// Check it's valid hex
			for _, c := range result {
				if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
					t.Errorf("ShortHash contains non-hex character: %c", c)
				}
			}
		})
	}

	// Test different content produces different hashes
	hash1 := ShortHash("hello", 8)
	hash2 := ShortHash("world", 8)
	if hash1 == hash2 {
		t.Error("Different content should produce different hashes")
	}
}

func TestLineSnippet(t *testing.T) {
	source := `package main

func main() {
    fmt.Println("Hello, World!")
}
`

	tests := []struct {
		name     string
		source   string
		lineNum  int
		maxLen   int
		expected string
	}{
		{"first line", source, 1, 50, "package main"},
		{"middle line", source, 4, 50, `fmt.Println("Hello, World!")`},
		{"last line", source, 5, 50, "}"},
		{"empty source", "", 1, 50, ""},
		{"line out of range", source, 100, 50, ""},
		{"zero line", source, 0, 50, ""},
		{"negative line", source, -1, 50, ""},
		{"zero maxLen", source, 1, 0, ""},
		{"truncated line", source, 4, 15, `fmt.Println(...`}, // 28 runes, truncate to 12+...
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LineSnippet(tt.source, tt.lineNum, tt.maxLen)
			if result != tt.expected {
				t.Errorf("LineSnippet(source, %d, %d) = %q, want %q", tt.lineNum, tt.maxLen, result, tt.expected)
			}
		})
	}
}
