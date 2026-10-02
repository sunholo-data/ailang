package eval

import (
	"encoding/json"
	"math"
	"math/rand"
	"strings"
	"testing"
)

// M-JSON-NUMBER-ROUNDTRIP (#1460): canonical JSON number text.

func TestFormatJSONNumber_Table(t *testing.T) {
	cases := []struct {
		f    float64
		want string
	}{
		{math.Copysign(0, -1), "-0.0"},
		{0, "0"},
		{42, "42"},
		{-17, "-17"},
		{3.14, "3.14"},
		{0.001, "0.001"},
		{12345678, "12345678"},
		{1e7, "10000000"},
		{1e10, "10000000000"},
		{100, "100"},
		{9007199254740992, "9007199254740992"},
		{9223372036854775808, "9223372036854776000"}, // 2^63: shortest digits, no int64 saturation
		{-9223372036854775808, "-9223372036854776000"},
		{1e19, "10000000000000000000"},
		{1e20, "100000000000000000000"},
		{1e21, "1e+21"},
		{-1e21, "-1e+21"},
		{1e308, "1e+308"},
		{math.MaxFloat64, "1.7976931348623157e+308"},
		{1e-6, "0.000001"},
		{1e-7, "1e-7"},
		{-1e-7, "-1e-7"},
		{1.5e-10, "1.5e-10"},
		{5e-324, "5e-324"},
		{math.NaN(), "null"},
		{math.Inf(1), "null"},
		{math.Inf(-1), "null"},
	}
	for _, c := range cases {
		if got := FormatJSONNumber(c.f); got != c.want {
			t.Errorf("FormatJSONNumber(%v) = %q, want %q", c.f, got, c.want)
		}
	}
}

// Window boundaries ±1 ulp: below 1e21 stays fixed, at/above goes exponent.
func TestFormatJSONNumber_WindowEdges(t *testing.T) {
	below := math.Nextafter(1e21, 0)
	if s := FormatJSONNumber(below); strings.ContainsAny(s, "e") {
		t.Errorf("just below 1e21 should be fixed notation, got %q", s)
	}
	small := math.Nextafter(1e-6, 0)
	if s := FormatJSONNumber(small); !strings.Contains(s, "e-") {
		t.Errorf("just below 1e-6 should be exponent notation, got %q", s)
	}
}

// Independent oracle: for finite non-zero values the text must be
// byte-identical to Go's encoding/json float encoder.
func TestFormatJSONNumber_MatchesEncodingJSON(t *testing.T) {
	for _, f := range jsonNumberSamples(t) {
		if f == 0 {
			continue
		}
		want, err := json.Marshal(f)
		if err != nil {
			t.Fatal(err)
		}
		if got := FormatJSONNumber(f); got != string(want) {
			t.Fatalf("FormatJSONNumber(%v) = %q, encoding/json = %q", f, got, want)
		}
	}
}

func TestParseJSONNumber(t *testing.T) {
	ok := []struct {
		in   string
		want float64
	}{
		{"10000000000000000000", 1e19},
		{"-10000000000000000000", -1e19},
		{"9007199254740993", 9007199254740992},
		{"-0", math.Copysign(0, -1)},
		{"-0.0", math.Copysign(0, -1)},
		{"1e21", 1e21},
		{"5e-324", 5e-324},
		{"42", 42},
		// Underflow: Go's ParseFloat rounds to zero without an error; the
		// contract is "no silent ±Inf", so this is pinned as Ok(0).
		{"1e-400", 0},
	}
	for _, c := range ok {
		got, err := ParseJSONNumber(json.Number(c.in))
		if err != nil {
			t.Fatalf("ParseJSONNumber(%s): unexpected error %v", c.in, err)
		}
		if math.Float64bits(got) != math.Float64bits(c.want) {
			t.Errorf("ParseJSONNumber(%s) = %v (bits %x), want %v", c.in, got, math.Float64bits(got), c.want)
		}
	}
	for _, in := range []string{"1e400", "-1e400", "1.8e308"} {
		if _, err := ParseJSONNumber(json.Number(in)); err == nil {
			t.Errorf("ParseJSONNumber(%s): expected out-of-range error", in)
		}
	}
}

// Legacy evaluator codec: encodeJSON / legacyDecodeString round-trip.
func TestLegacyJSONNumberRoundTrip(t *testing.T) {
	for _, f := range jsonNumberSamples(t) {
		jv := &TaggedValue{TypeName: "Json", CtorName: "JNumber", Fields: []Value{&FloatValue{Value: f}}}
		text, err := encodeJSON(jv)
		if err != nil {
			t.Fatal(err)
		}
		if text != FormatJSONNumber(f) {
			t.Fatalf("legacy encoder diverges for %v: %q vs %q", f, text, FormatJSONNumber(f))
		}
		res, err := legacyDecodeString(&StringValue{Value: text})
		if err != nil {
			t.Fatal(err)
		}
		tv := res.(*TaggedValue)
		if tv.CtorName != "Ok" {
			t.Fatalf("decode(%q) = %s", text, tv.Fields[0])
		}
		g := tv.Fields[0].(*TaggedValue).Fields[0].(*FloatValue).Value
		if math.Float64bits(g) != math.Float64bits(f) {
			t.Fatalf("round-trip %v -> %q -> %v", f, text, g)
		}
	}
}

func TestLegacyJSONDecodeOutOfRangeIsErr(t *testing.T) {
	for _, in := range []string{"1e400", "[1, -1e400]", `{"x":1e400}`} {
		res, err := legacyDecodeString(&StringValue{Value: in})
		if err != nil {
			t.Fatal(err)
		}
		if tv := res.(*TaggedValue); tv.CtorName != "Err" {
			t.Errorf("decode(%s): expected Err, got %s", in, tv.CtorName)
		}
	}
}

func TestLegacyJSONEncodeNonFiniteIsNull(t *testing.T) {
	for _, f := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		jv := &TaggedValue{TypeName: "Json", CtorName: "JNumber", Fields: []Value{&FloatValue{Value: f}}}
		text, err := encodeJSON(jv)
		if err != nil {
			t.Fatal(err)
		}
		if text != "null" {
			t.Errorf("encode(%v) = %q, want null", f, text)
		}
	}
}

// jsonNumberSamples returns the boundary table plus 10,000 seeded random
// finite float64 bit patterns.
func jsonNumberSamples(t *testing.T) []float64 {
	t.Helper()
	out := []float64{
		0, math.Copysign(0, -1), 1, -1, 0.1, 1e-6, math.Nextafter(1e-6, 0), 1e-7,
		1e21, math.Nextafter(1e21, 0), 1e20, 1e308, math.MaxFloat64, -math.MaxFloat64,
		5e-324, math.SmallestNonzeroFloat64, 2.2250738585072014e-308,
		1 << 53, 1<<53 + 2, 9223372036854775808, -9223372036854775808, 1e19, -1e19,
	}
	r := rand.New(rand.NewSource(1460))
	for len(out) < 10023 {
		f := math.Float64frombits(r.Uint64())
		if math.IsNaN(f) || math.IsInf(f, 0) {
			continue
		}
		out = append(out, f)
	}
	return out
}
