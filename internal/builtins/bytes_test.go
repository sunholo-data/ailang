package builtins

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

func TestBytesFromString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantLen int
		wantHex string
	}{
		{
			name:    "simple ascii",
			input:   "hello",
			wantLen: 5,
			wantHex: "<bytes:68656c6c6f>",
		},
		{
			name:    "emoji (4 bytes UTF-8)",
			input:   "🎉",
			wantLen: 4,
		},
		{
			name:    "empty string",
			input:   "",
			wantLen: 0,
		},
		{
			name:    "unicode characters",
			input:   "日本語",
			wantLen: 9, // 3 bytes per character
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []eval.Value{&eval.StringValue{Value: tt.input}}
			result, err := bytesFromStringImpl(nil, args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			bytesVal, ok := result.(*eval.BytesValue)
			if !ok {
				t.Fatalf("expected BytesValue, got %T", result)
			}

			if len(bytesVal.Value) != tt.wantLen {
				t.Errorf("got len %d, want %d", len(bytesVal.Value), tt.wantLen)
			}

			if tt.wantHex != "" && bytesVal.String() != tt.wantHex {
				t.Errorf("got %s, want %s", bytesVal.String(), tt.wantHex)
			}
		})
	}
}

func TestBytesToString(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{
			name:  "simple ascii",
			input: []byte("hello"),
			want:  "hello",
		},
		{
			name:  "empty bytes",
			input: []byte{},
			want:  "",
		},
		{
			name:  "unicode",
			input: []byte("日本語"),
			want:  "日本語",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []eval.Value{&eval.BytesValue{Value: tt.input}}
			result, err := bytesToStringImpl(nil, args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			strVal, ok := result.(*eval.StringValue)
			if !ok {
				t.Fatalf("expected StringValue, got %T", result)
			}

			if strVal.Value != tt.want {
				t.Errorf("got %q, want %q", strVal.Value, tt.want)
			}
		})
	}
}

func TestBytesToBase64(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{
			name:  "hello",
			input: []byte("hello"),
			want:  "aGVsbG8=",
		},
		{
			name:  "empty",
			input: []byte{},
			want:  "",
		},
		{
			name:  "binary data",
			input: []byte{0x00, 0x01, 0x02, 0xFF},
			want:  "AAEC/w==",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []eval.Value{&eval.BytesValue{Value: tt.input}}
			result, err := bytesToBase64Impl(nil, args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			strVal, ok := result.(*eval.StringValue)
			if !ok {
				t.Fatalf("expected StringValue, got %T", result)
			}

			if strVal.Value != tt.want {
				t.Errorf("got %q, want %q", strVal.Value, tt.want)
			}
		})
	}
}

func TestBytesFromBase64(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantSome  bool
		wantBytes []byte
	}{
		{
			name:      "valid base64",
			input:     "aGVsbG8=",
			wantSome:  true,
			wantBytes: []byte("hello"),
		},
		{
			name:      "empty string",
			input:     "",
			wantSome:  true,
			wantBytes: []byte{},
		},
		{
			name:     "invalid base64",
			input:    "!!!invalid!!!",
			wantSome: false,
		},
		{
			name:     "truncated base64",
			input:    "aGVsbG8", // missing padding
			wantSome: false,     // strict mode rejects this
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []eval.Value{&eval.StringValue{Value: tt.input}}
			result, err := bytesFromBase64Impl(nil, args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			taggedVal, ok := result.(*eval.TaggedValue)
			if !ok {
				t.Fatalf("expected TaggedValue, got %T", result)
			}

			if tt.wantSome {
				if taggedVal.CtorName != "Some" {
					t.Errorf("expected Some, got %s", taggedVal.CtorName)
				}
				if len(taggedVal.Fields) != 1 {
					t.Fatalf("expected 1 field, got %d", len(taggedVal.Fields))
				}
				bytesVal, ok := taggedVal.Fields[0].(*eval.BytesValue)
				if !ok {
					t.Fatalf("expected BytesValue, got %T", taggedVal.Fields[0])
				}
				if string(bytesVal.Value) != string(tt.wantBytes) {
					t.Errorf("got %v, want %v", bytesVal.Value, tt.wantBytes)
				}
			} else {
				if taggedVal.CtorName != "None" {
					t.Errorf("expected None, got %s", taggedVal.CtorName)
				}
			}
		})
	}
}

func TestBytesLength(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  int
	}{
		{
			name:  "simple",
			input: []byte("hello"),
			want:  5,
		},
		{
			name:  "empty",
			input: []byte{},
			want:  0,
		},
		{
			name:  "emoji (4 bytes)",
			input: []byte("🎉"),
			want:  4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []eval.Value{&eval.BytesValue{Value: tt.input}}
			result, err := bytesLengthImpl(nil, args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			intVal, ok := result.(*eval.IntValue)
			if !ok {
				t.Fatalf("expected IntValue, got %T", result)
			}

			if intVal.Value != tt.want {
				t.Errorf("got %d, want %d", intVal.Value, tt.want)
			}
		})
	}
}

func TestBytesRoundTrip(t *testing.T) {
	// Test that bytes_to_string(bytes_from_string(s)) == s
	original := "Hello, 世界! 🌍"

	// bytes_from_string
	args1 := []eval.Value{&eval.StringValue{Value: original}}
	bytesResult, err := bytesFromStringImpl(nil, args1)
	if err != nil {
		t.Fatalf("bytes_from_string failed: %v", err)
	}

	// bytes_to_string
	args2 := []eval.Value{bytesResult}
	stringResult, err := bytesToStringImpl(nil, args2)
	if err != nil {
		t.Fatalf("bytes_to_string failed: %v", err)
	}

	finalStr := stringResult.(*eval.StringValue).Value
	if finalStr != original {
		t.Errorf("round trip failed: got %q, want %q", finalStr, original)
	}
}

func TestBase64RoundTrip(t *testing.T) {
	// Test that bytes_from_base64(bytes_to_base64(b)) == Some(b)
	original := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}

	// bytes_to_base64
	args1 := []eval.Value{&eval.BytesValue{Value: original}}
	base64Result, err := bytesToBase64Impl(nil, args1)
	if err != nil {
		t.Fatalf("bytes_to_base64 failed: %v", err)
	}

	// bytes_from_base64
	args2 := []eval.Value{base64Result}
	decodedResult, err := bytesFromBase64Impl(nil, args2)
	if err != nil {
		t.Fatalf("bytes_from_base64 failed: %v", err)
	}

	taggedVal := decodedResult.(*eval.TaggedValue)
	if taggedVal.CtorName != "Some" {
		t.Fatalf("expected Some, got %s", taggedVal.CtorName)
	}

	finalBytes := taggedVal.Fields[0].(*eval.BytesValue).Value
	if string(finalBytes) != string(original) {
		t.Errorf("round trip failed: got %v, want %v", finalBytes, original)
	}
}
