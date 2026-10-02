package format

import (
	"strings"
	"testing"
)

// #1481: full-width base-prefixed literals parse to negative int64 values.
// fmt must print them in a form that re-parses to the SAME literal: decimal
// "-7046029254386353131" re-parses as unary minus (a different AST, so the
// round-trip check rejected the file) and "-9223372036854775808" does not
// parse at all. The canonical form is the 64-bit hex pattern.
func TestFullWidthIntLiteralsRoundTrip(t *testing.T) {
	src := "module t\n\n" +
		"pure func f(x: int) -> int = match x {\n" +
		"  0x9E3779B97F4A7C15 => 0x8000000000000000,\n" +
		"  _ => x * 0xffffffffffffffff + 0xff\n" +
		"}\n"
	out := formatSrc(t, src)
	for _, want := range []string{"0x9e3779b97f4a7c15 =>", "0x8000000000000000", "x * 0xffffffffffffffff + 255"} {
		if !strings.Contains(out, want) {
			t.Errorf("formatted output lacks %q:\n%s", want, out)
		}
	}
	if again := formatSrc(t, out); again != out {
		t.Errorf("fmt is not idempotent on full-width literals:\n--- first ---\n%s\n--- second ---\n%s", out, again)
	}
}
