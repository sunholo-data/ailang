package parser

import (
	"math"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/lexer"
)

// #1481: base-prefixed literals are 64-bit patterns. Values in [2^63, 2^64-1]
// wrap to the two's-complement int64 (like Go's int64(uint64(x))), so hash
// constants such as SplitMix64's 0x9e3779b97f4a7c15 are writable verbatim.
// Decimal literals stay signed: above MaxInt64 they are still an error.
func TestParseIntLiteralValueFullWidth(t *testing.T) {
	cases := []struct {
		in   string
		want int64
	}{
		{"0x9e3779b97f4a7c15", -7046029254386353131},
		{"0xbf58476d1ce4e5b9", -4658895280553007687},
		{"0x94d049bb133111eb", -7723592293110705685},
		{"0xcbf29ce484222325", -3750763034362895579}, // FNV-1a 64 offset basis
		{"0x8000000000000000", math.MinInt64},
		{"0xffffffffffffffff", -1},
		{"0XFFFFFFFFFFFFFFFF", -1},
		{"0x7fffffffffffffff", math.MaxInt64},
		{"0b1111111111111111111111111111111111111111111111111111111111111111", -1},
		{"0b1000000000000000000000000000000000000000000000000000000000000000", math.MinInt64},
		{"0o1777777777777777777777", -1},
		{"0o1000000000000000000000", math.MinInt64},
		{"9223372036854775807", math.MaxInt64}, // largest decimal, unchanged
	}
	for _, c := range cases {
		got, err := parseIntLiteralValue(c.in)
		if err != nil {
			t.Errorf("%s: unexpected error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s: got %d, want %d", c.in, got, c.want)
		}
	}
	for _, bad := range []string{
		"9223372036854775808",      // decimal above MaxInt64 stays an error
		"18446744073709551615",     // decimal MaxUint64: NOT silently -1
		"0x10000000000000000",      // 65 bits
		"0x1fffffffffffffffff",     // 69 bits
		"0o2000000000000000000000", // 65 bits
	} {
		if v, err := parseIntLiteralValue(bad); err == nil {
			t.Errorf("%s: expected an error, got %d", bad, v)
		}
	}
}

func TestPAR021_IntLiteralOutOfRange(t *testing.T) {
	t.Run("decimal above MaxInt64 suggests the hex pattern and signed decimal", func(t *testing.T) {
		input := "module t\npure func f() -> int = 18446744073709551615\n"
		err := firstParserErrorWithCode(t, input, "PAR021")
		if err == nil {
			t.Fatal("expected PAR021 for a decimal literal above MaxInt64")
		}
		if err.Pos.Line != 2 || err.Pos.Column == 0 {
			t.Errorf("PAR021 should be positioned at the literal, got %+v", err.Pos)
		}
		sugg := strings.Join(err.Suggestions, "\n")
		for _, want := range []string{"0xFFFFFFFFFFFFFFFF", "signed decimal -1"} {
			if !strings.Contains(sugg, want) {
				t.Errorf("PAR021 should suggest %s; got:\n%s", want, sugg)
			}
		}
	})
	t.Run("decimal 2^63 suggests only the hex form (-9223372036854775808 does not parse)", func(t *testing.T) {
		input := "module t\npure func f() -> int = 9223372036854775808\n"
		err := firstParserErrorWithCode(t, input, "PAR021")
		if err == nil {
			t.Fatal("expected PAR021 for 2^63")
		}
		sugg := strings.Join(err.Suggestions, "\n")
		if !strings.Contains(sugg, "0x8000000000000000") {
			t.Errorf("PAR021 should suggest 0x8000000000000000; got:\n%s", sugg)
		}
		if strings.Contains(sugg, "signed decimal") {
			t.Errorf("PAR021 must not suggest an unparseable signed decimal for MinInt64; got:\n%s", sugg)
		}
	})
	t.Run("hex wider than 64 bits", func(t *testing.T) {
		input := "module t\npure func f() -> int = 0x1fffffffffffffffff\n"
		err := firstParserErrorWithCode(t, input, "PAR021")
		if err == nil {
			t.Fatal("expected PAR021 for a 69-bit hex literal")
		}
		if !strings.Contains(err.Error()+strings.Join(err.Suggestions, "\n"), "0xFFFFFFFFFFFFFFFF") {
			t.Errorf("PAR021 should name the largest writable pattern: %s", err.Error())
		}
	})
	t.Run("pattern position: out-of-range int is an error, not a clamp", func(t *testing.T) {
		input := "module t\npure func f(x: int) -> int = match x { 0x1fffffffffffffffff => 1, _ => 3 }\n"
		if firstParserErrorWithCode(t, input, "PAR021") == nil {
			t.Fatal("expected PAR021 for an out-of-range int pattern (was a silent clamp)")
		}
	})
	t.Run("pattern position: overflowing float is an error, not +Inf", func(t *testing.T) {
		input := "module t\npure func f(x: float) -> int = match x { 1e999 => 1, _ => 3 }\n"
		if firstParserErrorWithCode(t, input, "PAR021") == nil {
			t.Fatal("expected PAR021 for an overflowing float pattern (was a silent +Inf)")
		}
	})
	t.Run("in-range literals do not trigger PAR021", func(t *testing.T) {
		input := "module t\npure func f(x: int) -> int = match x { 0xff => 0x9e3779b97f4a7c15, 0123 => 0b1010, _ => 9223372036854775807 }\n"
		l := lexer.New(input, "<test>")
		p := New(l)
		_ = p.Parse()
		if len(p.errors) != 0 {
			t.Fatalf("unexpected parse errors: %v", p.errors)
		}
	})
}

// The full-width literal in pattern position must carry the wrapped value
// (it used to silently become MaxInt64).
func TestFullWidthHexPatternValue(t *testing.T) {
	p := New(lexer.New("0x9e3779b97f4a7c15", "<test>"))
	v, err := p.literalValue()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != int64(-7046029254386353131) {
		t.Fatalf("pattern literal value = %v, want -7046029254386353131", v)
	}
	p = New(lexer.New("0x1fffffffffffffffff", "<test>"))
	if v, err := p.literalValue(); err == nil {
		t.Fatalf("69-bit pattern literal must be an error, got %v", v)
	}
}
