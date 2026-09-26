package builtins

import (
	"bytes"
	"strings"
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

func TestBytesToInts(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  []int
	}{
		{
			name:  "ascii",
			input: []byte("AB"),
			want:  []int{65, 66},
		},
		{
			name:  "empty",
			input: []byte{},
			want:  []int{},
		},
		{
			name:  "full byte range boundaries",
			input: []byte{0x00, 0x7F, 0x80, 0xFF},
			want:  []int{0, 127, 128, 255},
		},
		{
			name:  "high bytes are unsigned, not sign-extended",
			input: []byte{0xC5},
			want:  []int{197},
		},
		{
			name:  "multi-byte UTF-8 yields raw bytes, not codepoints",
			input: []byte("é"),
			want:  []int{195, 169},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []eval.Value{&eval.BytesValue{Value: tt.input}}
			result, err := bytesToIntsImpl(nil, args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			listVal, ok := result.(*eval.ListValue)
			if !ok {
				t.Fatalf("expected ListValue, got %T", result)
			}
			if len(listVal.Elements) != len(tt.want) {
				t.Fatalf("got len %d, want %d", len(listVal.Elements), len(tt.want))
			}
			for i, elem := range listVal.Elements {
				iv, ok := elem.(*eval.IntValue)
				if !ok {
					t.Fatalf("element %d: expected IntValue, got %T", i, elem)
				}
				if iv.Value != tt.want[i] {
					t.Errorf("element %d: got %d, want %d", i, iv.Value, tt.want[i])
				}
			}
		})
	}
}

func TestBytesToIntsWrongType(t *testing.T) {
	args := []eval.Value{&eval.StringValue{Value: "not bytes"}}
	if _, err := bytesToIntsImpl(nil, args); err == nil {
		t.Fatal("expected error for non-Bytes argument, got nil")
	}
}

func TestBytesIntsRoundTrip(t *testing.T) {
	// Property: toInts(fromInts(xs)) == xs for every value in 0..255.
	// This is the acceptance criterion from design_docs/implemented/v0_21_0/
	// m-bytes-toints-byteAt.md that shipped with byteAt but never with toInts.
	elements := make([]eval.Value, 256)
	for i := range elements {
		elements[i] = &eval.IntValue{Value: i}
	}

	bytesResult, err := bytesFromIntsImpl(nil, []eval.Value{&eval.ListValue{Elements: elements}})
	if err != nil {
		t.Fatalf("_bytes_from_ints failed: %v", err)
	}

	intsResult, err := bytesToIntsImpl(nil, []eval.Value{bytesResult})
	if err != nil {
		t.Fatalf("_bytes_to_ints failed: %v", err)
	}

	got := intsResult.(*eval.ListValue).Elements
	if len(got) != 256 {
		t.Fatalf("round trip changed length: got %d, want 256", len(got))
	}
	for i, elem := range got {
		if v := elem.(*eval.IntValue).Value; v != i {
			t.Errorf("round trip corrupted index %d: got %d", i, v)
		}
	}
}

// ============================================================================
// base64url (M-STD-BASE64URL-ENCODE)
// ============================================================================

// encodeB64URL is a test helper: runs _bytes_to_base64url and returns the string.
func encodeB64URL(t *testing.T, b []byte) string {
	t.Helper()
	result, err := bytesToBase64URLImpl(nil, []eval.Value{&eval.BytesValue{Value: b}})
	if err != nil {
		t.Fatalf("_bytes_to_base64url(%v) failed: %v", b, err)
	}
	strVal, ok := result.(*eval.StringValue)
	if !ok {
		t.Fatalf("expected StringValue, got %T", result)
	}
	return strVal.Value
}

// decodeB64URL is a test helper: runs _bytes_from_base64url, requiring Some.
func decodeB64URL(t *testing.T, s string) []byte {
	t.Helper()
	result, err := bytesFromBase64URLImpl(nil, []eval.Value{&eval.StringValue{Value: s}})
	if err != nil {
		t.Fatalf("_bytes_from_base64url(%q) failed: %v", s, err)
	}
	tagged, ok := result.(*eval.TaggedValue)
	if !ok {
		t.Fatalf("expected TaggedValue, got %T", result)
	}
	if tagged.CtorName != "Some" {
		t.Fatalf("_bytes_from_base64url(%q): expected Some, got %s", s, tagged.CtorName)
	}
	return tagged.Fields[0].(*eval.BytesValue).Value
}

func TestBytesToBase64URL(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{
			name:  "hello — unpadded where standard base64 pads",
			input: []byte("Hello"),
			want:  "SGVsbG8",
		},
		{
			name:  "empty",
			input: []byte{},
			want:  "",
		},
		{
			// The exact case from fb_dfb699d91224be9c. Standard base64 gives
			// "YStiL2M/" — the '/' is what the Gmail API rejects.
			name:  "reported case: URL-safe alphabet differs from standard",
			input: []byte("a+b/c?"),
			want:  "YStiL2M_",
		},
		{
			// 0xFB 0xFF exercises both substituted characters at once:
			// standard base64 of these bytes contains '+' and '/'.
			name:  "binary exercising both '-' and '_'",
			input: []byte{0xFB, 0xFF, 0xFE},
			want:  "-__-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := encodeB64URL(t, tt.input); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// TestBytesToBase64URLDiffersFromStandard pins the difference the feature
// exists for, asserting both encoders side by side on the reported input.
func TestBytesToBase64URLDiffersFromStandard(t *testing.T) {
	input := []byte("a+b/c?")

	stdResult, err := bytesToBase64Impl(nil, []eval.Value{&eval.BytesValue{Value: input}})
	if err != nil {
		t.Fatalf("_bytes_to_base64 failed: %v", err)
	}
	std := stdResult.(*eval.StringValue).Value
	url := encodeB64URL(t, input)

	if std != "YStiL2M/" {
		t.Errorf("standard base64: got %q, want %q", std, "YStiL2M/")
	}
	if url != "YStiL2M_" {
		t.Errorf("base64url: got %q, want %q", url, "YStiL2M_")
	}
	if std == url {
		t.Error("standard and URL-safe encodings must differ on this input")
	}
}

// TestBytesBase64URLAlphabet asserts the encoder never emits a character that
// is invalid in a URL or in a JWT segment: '+', '/' or '='.
func TestBytesBase64URLAlphabet(t *testing.T) {
	// Deterministic pseudorandom inputs — no seed dependence across Go versions.
	var state uint32 = 0x12345678
	next := func() byte {
		state = state*1664525 + 1013904223
		return byte(state >> 24)
	}

	for length := 0; length < 64; length++ {
		buf := make([]byte, length)
		for i := range buf {
			buf[i] = next()
		}
		encoded := encodeB64URL(t, buf)
		for _, bad := range []rune{'+', '/', '='} {
			if strings.ContainsRune(encoded, bad) {
				t.Fatalf("encoding of %v contains %q (not URL-safe): %q", buf, bad, encoded)
			}
		}
	}
}

// TestBytesBase64URLRoundTrip covers all 256 byte values and all three
// length-mod-3 padding residues — the residues are where a hand-rolled
// encoder characteristically breaks.
func TestBytesBase64URLRoundTrip(t *testing.T) {
	t.Run("all 256 single byte values", func(t *testing.T) {
		for i := 0; i < 256; i++ {
			original := []byte{byte(i)}
			got := decodeB64URL(t, encodeB64URL(t, original))
			if !bytes.Equal(got, original) {
				t.Errorf("byte %d: round trip gave %v, want %v", i, got, original)
			}
		}
	})

	t.Run("all padding residues", func(t *testing.T) {
		// Lengths 0..6 cover every length mod 3, twice.
		for length := 0; length <= 6; length++ {
			original := make([]byte, length)
			for i := range original {
				original[i] = byte(0xF0 + i)
			}
			encoded := encodeB64URL(t, original)
			if strings.Contains(encoded, "=") {
				t.Errorf("length %d: encoding %q must not be padded", length, encoded)
			}
			got := decodeB64URL(t, encoded)
			if !bytes.Equal(got, original) {
				t.Errorf("length %d: round trip gave %v, want %v", length, got, original)
			}
		}
	})

	t.Run("RFC 5322 message with CRLF and UTF-8 subject", func(t *testing.T) {
		// The Gmail `raw` field shape: CRLF line endings, a UTF-8 subject.
		original := []byte("To: a@example.com\r\n" +
			"Subject: Café — ünïcode\r\n" +
			"Content-Type: text/plain; charset=UTF-8\r\n" +
			"\r\n" +
			"Body with a + and a / in it.\r\n")
		encoded := encodeB64URL(t, original)
		if strings.ContainsAny(encoded, "+/=") {
			t.Errorf("encoding is not URL-safe: %q", encoded)
		}
		if got := decodeB64URL(t, encoded); !bytes.Equal(got, original) {
			t.Errorf("round trip failed:\n got %q\nwant %q", got, original)
		}
	})
}

func TestBytesToBase64URLWrongType(t *testing.T) {
	_, err := bytesToBase64URLImpl(nil, []eval.Value{&eval.StringValue{Value: "not bytes"}})
	if err == nil {
		t.Fatal("expected an error for a non-Bytes argument, got nil")
	}
}
