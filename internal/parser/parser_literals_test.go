package parser

import "testing"

// TestParseIntLiteralValue covers hex/binary/octal base-detection and the regression that
// leading-zero decimals stay decimal (NOT reinterpreted as octal by Go's base-0 parser).
func TestParseIntLiteralValue(t *testing.T) {
	cases := []struct {
		in   string
		want int64
	}{
		{"0xE000", 57344},
		{"0xff", 255},
		{"0XaB", 171},
		{"0b1010", 10},
		{"0o17", 15},
		{"0O755", 493},
		{"42", 42},
		{"0", 0},
		{"0123", 123}, // leading-zero decimal stays decimal, not octal 83
	}
	for _, c := range cases {
		got, err := parseIntLiteralValue(c.in)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", c.in, err)
		}
		if got != c.want {
			t.Errorf("%s: got %d, want %d", c.in, got, c.want)
		}
	}
}
