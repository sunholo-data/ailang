package builtins

import (
	"math"
	"math/rand"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

// M-JSON-NUMBER-ROUNDTRIP (#1460): registry codec (_json_encode/_json_decode).

func registryNumberSamples() []float64 {
	out := []float64{
		0, math.Copysign(0, -1), 1, -1, 0.1, 1e-6, 1e-7, 1e21, math.Nextafter(1e21, 0),
		1e308, math.MaxFloat64, 5e-324, 1 << 53, 9223372036854775808, -9223372036854775808, 1e19, -1e19,
	}
	r := rand.New(rand.NewSource(1460))
	for len(out) < 10017 {
		f := math.Float64frombits(r.Uint64())
		if math.IsNaN(f) || math.IsInf(f, 0) {
			continue
		}
		out = append(out, f)
	}
	return out
}

func TestRegistryJSONNumberEncodeTable(t *testing.T) {
	cases := map[float64]string{
		math.Copysign(0, -1): "-0.0",
		1e21:                 "1e+21",
		1e308:                "1e+308",
		5e-324:               "5e-324",
		1e-7:                 "1e-7",
		100:                  "100",
		9223372036854775808:  "9223372036854776000",
		math.Inf(1):          "null",
	}
	for f, want := range cases {
		res, err := jsonEncodeImpl(nil, []eval.Value{makeJNumberFloat(f)})
		if err != nil {
			t.Fatal(err)
		}
		if got := res.(*eval.StringValue).Value; got != want {
			t.Errorf("encode(%v) = %q, want %q", f, got, want)
		}
	}
}

func TestRegistryJSONNumberRoundTrip(t *testing.T) {
	for _, f := range registryNumberSamples() {
		res, err := jsonEncodeImpl(nil, []eval.Value{makeJNumberFloat(f)})
		if err != nil {
			t.Fatal(err)
		}
		text := res.(*eval.StringValue).Value
		if text != eval.FormatJSONNumber(f) {
			t.Fatalf("registry encoder diverges for %v: %q vs %q", f, text, eval.FormatJSONNumber(f))
		}
		dec, err := jsonDecodeString(&eval.StringValue{Value: text})
		if err != nil {
			t.Fatal(err)
		}
		inner := extractOk(dec)
		if inner == nil {
			t.Fatalf("decode(%q) not Ok: %v", text, dec)
		}
		g := inner.(*eval.TaggedValue).Fields[0].(*eval.FloatValue).Value
		if math.Float64bits(g) != math.Float64bits(f) {
			t.Fatalf("round-trip %v -> %q -> %v", f, text, g)
		}
	}
}

func TestRegistryJSONDecodeLargeIntegersExact(t *testing.T) {
	for in, want := range map[string]float64{
		`10000000000000000000`:  1e19,
		`-10000000000000000000`: -1e19,
		`-0`:                    math.Copysign(0, -1),
	} {
		inner := extractOk(mustDecode(t, in))
		if inner == nil {
			t.Fatalf("decode(%s) not Ok", in)
		}
		g := inner.(*eval.TaggedValue).Fields[0].(*eval.FloatValue).Value
		if math.Float64bits(g) != math.Float64bits(want) {
			t.Errorf("decode(%s) = %v, want %v", in, g, want)
		}
	}
}

func TestRegistryJSONDecodeOutOfRangeIsErr(t *testing.T) {
	for _, in := range []string{`1e400`, `[1, -1e400]`, `{"x":{"y":[1e400]}}`} {
		res := mustDecode(t, in)
		if tv := res.(*eval.TaggedValue); tv.CtorName != "Err" {
			t.Errorf("decode(%s): expected Err, got %s", in, tv.CtorName)
		}
	}
}

func mustDecode(t *testing.T, in string) eval.Value {
	t.Helper()
	res, err := jsonDecodeString(&eval.StringValue{Value: in})
	if err != nil {
		t.Fatal(err)
	}
	return res
}
