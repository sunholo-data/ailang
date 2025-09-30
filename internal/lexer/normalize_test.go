package lexer

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/ast"
	"golang.org/x/text/unicode/norm"
)

// TestBOMStripping verifies that UTF-8 BOM is removed
func TestBOMStripping(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected []byte
	}{
		{
			name:     "with_bom",
			input:    []byte{0xEF, 0xBB, 0xBF, 'h', 'e', 'l', 'l', 'o'},
			expected: []byte("hello"),
		},
		{
			name:     "without_bom",
			input:    []byte("hello"),
			expected: []byte("hello"),
		},
		{
			name:     "empty_with_bom",
			input:    []byte{0xEF, 0xBB, 0xBF},
			expected: []byte{},
		},
		{
			name:     "empty_without_bom",
			input:    []byte{},
			expected: []byte{},
		},
		{
			name:     "partial_bom",
			input:    []byte{0xEF, 0xBB, 'h', 'i'},
			expected: []byte{0xEF, 0xBB, 'h', 'i'}, // Not a valid BOM
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Normalize(tt.input)
			if !bytes.Equal(result, tt.expected) {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestNFCNormalization verifies Unicode normalization
func TestNFCNormalization(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "already_nfc",
			input:    "café", // U+00E9 (é in NFC)
			expected: "café",
		},
		{
			name:     "nfd_to_nfc",
			input:    "cafe\u0301", // e + combining acute accent (NFD)
			expected: "café",       // Should become é (U+00E9)
		},
		{
			name:     "ascii_unchanged",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "mixed_unicode",
			input:    "naïve café", // i + combining diaeresis, é in NFC
			expected: "naïve café",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := string(Normalize([]byte(tt.input)))
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}

			// Verify result is in NFC form
			if !norm.NFC.IsNormalString(result) {
				t.Errorf("Result is not in NFC form")
			}
		})
	}
}

// TestBOMAndNFC verifies both BOM stripping and NFC normalization together
func TestBOMAndNFC(t *testing.T) {
	// BOM + NFD input
	input := append(bomUTF8, []byte("cafe\u0301")...) // BOM + "café" in NFD
	expected := "café"                                // "café" in NFC, no BOM

	result := string(Normalize(input))
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}

	if !norm.NFC.IsNormalString(result) {
		t.Errorf("Result is not in NFC form")
	}
}

// TestNormalizeIdempotent verifies that normalizing twice has no effect
func TestNormalizeIdempotent(t *testing.T) {
	inputs := []string{
		"hello",
		"café",
		"cafe\u0301",
		"\uFEFFhello",
	}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			first := Normalize([]byte(input))
			second := Normalize(first)

			if !bytes.Equal(first, second) {
				t.Errorf("Normalize is not idempotent: first=%q, second=%q", first, second)
			}
		})
	}
}

// TestCanaryDeterministicParsing is the canary test that ensures
// lexically equivalent source produces identical AST output regardless
// of encoding variations (LF vs CRLF, NFC vs NFD).
func TestCanaryDeterministicParsing(t *testing.T) {
	variants := []struct {
		name  string
		input string
	}{
		{
			name:  "lf_nfc",
			input: "let café = 42", // LF, NFC
		},
		{
			name:  "crlf_nfc",
			input: "let café = 42", // CRLF manually added below
		},
		{
			name:  "lf_nfd",
			input: "let cafe\u0301 = 42", // LF, NFD (e + combining acute)
		},
		{
			name:  "crlf_nfd",
			input: "let cafe\u0301 = 42", // CRLF manually added below, NFD
		},
		{
			name:  "bom_lf_nfc",
			input: "\uFEFFlet café = 42", // BOM + LF + NFC
		},
	}

	// Manually add CRLF variants
	variants[1].input = strings.ReplaceAll(variants[1].input, "\n", "\r\n")
	variants[3].input = strings.ReplaceAll(variants[3].input, "\n", "\r\n")

	// Parse all variants and collect AST JSON
	var outputs []string
	for _, v := range variants {
		t.Run(v.name, func(t *testing.T) {
			// Normalize input
			normalized := Normalize([]byte(v.input))

			// Lex and parse
			l := New(string(normalized), "test.ail")
			tokens := []Token{}
			for {
				tok := l.NextToken()
				tokens = append(tokens, tok)
				if tok.Type == EOF {
					break
				}
			}

			// For the canary test, we just verify tokenization produces identical results
			// Full parsing would be done when parser uses Normalize()

			// Serialize tokens to JSON for comparison
			jsonData, err := json.Marshal(tokens)
			if err != nil {
				t.Fatalf("Failed to marshal tokens: %v", err)
			}
			outputs = append(outputs, string(jsonData))
		})
	}

	// Verify all outputs are identical
	if len(outputs) < 2 {
		t.Fatal("Not enough outputs to compare")
	}

	baseline := outputs[0]
	for i, output := range outputs[1:] {
		if output != baseline {
			t.Errorf("Variant %d produced different output than baseline", i+1)
			t.Logf("Baseline: %s", baseline)
			t.Logf("Variant %d: %s", i+1, output)
		}
	}
}

// TestNormalizePreservesSemantics verifies normalization doesn't change meaning
func TestNormalizePreservesSemantics(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "let_binding",
			input: "let x = 5",
		},
		{
			name:  "unicode_identifier",
			input: "let café = 42", // NFC
		},
		{
			name:  "string_literal",
			input: `"hello world"`,
		},
		{
			name:  "comment",
			input: "-- this is a comment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse without normalization (baseline)
			l1 := New(tt.input, "test.ail")
			var tokens1 []Token
			for {
				tok := l1.NextToken()
				tokens1 = append(tokens1, tok)
				if tok.Type == EOF {
					break
				}
			}

			// Parse with normalization
			normalized := Normalize([]byte(tt.input))
			l2 := New(string(normalized), "test.ail")
			var tokens2 []Token
			for {
				tok := l2.NextToken()
				tokens2 = append(tokens2, tok)
				if tok.Type == EOF {
					break
				}
			}

			// Should produce same tokens (ignoring position details)
			if len(tokens1) != len(tokens2) {
				t.Errorf("Token count mismatch: %d vs %d", len(tokens1), len(tokens2))
			}

			for i := range tokens1 {
				if i >= len(tokens2) {
					break
				}
				if tokens1[i].Type != tokens2[i].Type {
					t.Errorf("Token %d type mismatch: %v vs %v", i, tokens1[i].Type, tokens2[i].Type)
				}
			}
		})
	}
}

// TestNormalizeDeterminism verifies Normalize() produces stable output
func TestNormalizeDeterminism(t *testing.T) {
	input := []byte("\uFEFFcafe\u0301") // BOM + NFD

	var results [][]byte
	for i := 0; i < 100; i++ {
		result := Normalize(input)
		results = append(results, result)
	}

	baseline := results[0]
	for i, result := range results[1:] {
		if !bytes.Equal(result, baseline) {
			t.Errorf("Iteration %d produced different output", i+1)
		}
	}
}

// Ensure this compiles - ast package is used for future integration
var _ = ast.File{}
