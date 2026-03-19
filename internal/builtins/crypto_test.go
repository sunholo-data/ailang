package builtins

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
)

func TestCryptoSha256Hex(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"},
		{"", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
		{"abc", "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"},
		{"The quick brown fox jumps over the lazy dog", "d7a8fbb307d7809469ca9abcb0082e4f8d5651e46d3cdb762d02d0bf37c9e592"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := sha256HexImpl(nil, []eval.Value{
				&eval.StringValue{Value: tt.input},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := result.(*eval.StringValue).Value
			if got != tt.expected {
				t.Errorf("sha256Hex(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestCryptoSha256Bytes(t *testing.T) {
	input := []byte("hello")
	expected := sha256.Sum256(input)
	expectedHex := hex.EncodeToString(expected[:])

	result, err := sha256BytesImpl(nil, []eval.Value{
		&eval.BytesValue{Value: input},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := result.(*eval.StringValue).Value
	if got != expectedHex {
		t.Errorf("sha256Bytes(hello) = %q, want %q", got, expectedHex)
	}
}

func TestCryptoHmacSha256(t *testing.T) {
	tests := []struct {
		message string
		key     string
	}{
		{"message", "secret"},
		{"", "key"},
		{"data", ""},
		{"The quick brown fox jumps over the lazy dog", "key"},
	}

	for _, tt := range tests {
		t.Run(tt.message+"_"+tt.key, func(t *testing.T) {
			// Compute expected HMAC using Go stdlib directly
			mac := hmac.New(sha256.New, []byte(tt.key))
			mac.Write([]byte(tt.message))
			expected := hex.EncodeToString(mac.Sum(nil))

			result, err := hmacSha256Impl(nil, []eval.Value{
				&eval.StringValue{Value: tt.message},
				&eval.StringValue{Value: tt.key},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := result.(*eval.StringValue).Value
			if got != expected {
				t.Errorf("hmacSha256(%q, %q) = %q, want %q", tt.message, tt.key, got, expected)
			}
		})
	}
}

func TestCryptoConstantTimeEqual(t *testing.T) {
	tests := []struct {
		a, b     string
		expected bool
	}{
		{"abc", "abc", true},
		{"abc", "def", false},
		{"", "", true},
		{"a", "", false},
		{"", "a", false},
		{"hello world", "hello world", true},
		{"hello world", "hello worlD", false},
	}

	for _, tt := range tests {
		t.Run(tt.a+"_vs_"+tt.b, func(t *testing.T) {
			result, err := constantTimeEqualImpl(nil, []eval.Value{
				&eval.StringValue{Value: tt.a},
				&eval.StringValue{Value: tt.b},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := result.(*eval.BoolValue).Value
			if got != tt.expected {
				t.Errorf("constantTimeEqual(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.expected)
			}
		})
	}
}

func TestCryptoSha256Deterministic(t *testing.T) {
	// Run 20 times to catch any nondeterminism (per milestone checklist)
	for i := 0; i < 20; i++ {
		result, err := sha256HexImpl(nil, []eval.Value{
			&eval.StringValue{Value: "determinism test"},
		})
		if err != nil {
			t.Fatalf("run %d: unexpected error: %v", i, err)
		}
		got := result.(*eval.StringValue).Value
		expected := "ae08fdbbe31ef075d8d2452dae93cac75efdd967b87b7f73574d918d393f8f45"
		if got != expected {
			t.Errorf("run %d: sha256Hex('determinism test') = %q, want %q", i, got, expected)
		}
	}
}

// Verify nil EffContext is fine for pure builtins (no capability check)
func TestCryptoPureNoEffectContext(t *testing.T) {
	var nilCtx *effects.EffContext

	_, err := sha256HexImpl(nilCtx, []eval.Value{&eval.StringValue{Value: "test"}})
	if err != nil {
		t.Errorf("sha256Hex should work with nil EffContext: %v", err)
	}

	_, err = hmacSha256Impl(nilCtx, []eval.Value{
		&eval.StringValue{Value: "msg"},
		&eval.StringValue{Value: "key"},
	})
	if err != nil {
		t.Errorf("hmacSha256 should work with nil EffContext: %v", err)
	}

	_, err = constantTimeEqualImpl(nilCtx, []eval.Value{
		&eval.StringValue{Value: "a"},
		&eval.StringValue{Value: "b"},
	})
	if err != nil {
		t.Errorf("constantTimeEqual should work with nil EffContext: %v", err)
	}
}
