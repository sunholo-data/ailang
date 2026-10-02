package vm

import (
	"math"
	"math/rand"
	"testing"

	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/eval"
)

// M-JSON-NUMBER-ROUNDTRIP (#1460): VM codec (__json_encode/__json_decode).

func vmJNum(f float64) bytecode.Value {
	return bytecode.NewADT(jsonTagJNumber, "JNumber", []bytecode.Value{bytecode.NewFloat(f)})
}

func TestVMJSONNumberEncodeTable(t *testing.T) {
	cases := map[float64]string{
		math.Copysign(0, -1): "-0.0",
		1e21:                 "1e+21",
		1e308:                "1e+308",
		5e-324:               "5e-324",
		1e-7:                 "1e-7",
		100:                  "100",
		9223372036854775808:  "9223372036854776000",
		math.NaN():           "null",
	}
	for f, want := range cases {
		got, err := builtinJsonEncode([]bytecode.Value{vmJNum(f)})
		if err != nil {
			t.Fatal(err)
		}
		if got.AsString() != want {
			t.Errorf("encode(%v) = %q, want %q", f, got.AsString(), want)
		}
	}
}

func TestVMJSONNumberRoundTrip(t *testing.T) {
	samples := []float64{0, math.Copysign(0, -1), 1e21, 1e308, 5e-324, 1e19, -1e19, 9223372036854775808}
	r := rand.New(rand.NewSource(1460))
	for len(samples) < 10008 {
		f := math.Float64frombits(r.Uint64())
		if !math.IsNaN(f) && !math.IsInf(f, 0) {
			samples = append(samples, f)
		}
	}
	for _, f := range samples {
		enc, err := builtinJsonEncode([]bytecode.Value{vmJNum(f)})
		if err != nil {
			t.Fatal(err)
		}
		text := enc.AsString()
		if text != eval.FormatJSONNumber(f) {
			t.Fatalf("VM encoder diverges for %v: %q vs %q", f, text, eval.FormatJSONNumber(f))
		}
		dec, err := builtinJsonDecode([]bytecode.Value{bytecode.NewString(text)})
		if err != nil {
			t.Fatal(err)
		}
		assertResultOk(t, dec)
		g := dec.AsADT().Fields[0].AsADT().Fields[0].Flt
		if math.Float64bits(g) != math.Float64bits(f) {
			t.Fatalf("round-trip %v -> %q -> %v", f, text, g)
		}
	}
}

func TestVMJSONDecodeLargeIntegersExact(t *testing.T) {
	for in, want := range map[string]float64{
		"10000000000000000000":  1e19,
		"-10000000000000000000": -1e19,
		"-0":                    math.Copysign(0, -1),
	} {
		dec, err := builtinJsonDecode([]bytecode.Value{bytecode.NewString(in)})
		if err != nil {
			t.Fatal(err)
		}
		assertResultOk(t, dec)
		g := dec.AsADT().Fields[0].AsADT().Fields[0].Flt
		if math.Float64bits(g) != math.Float64bits(want) {
			t.Errorf("decode(%s) = %v, want %v", in, g, want)
		}
	}
}

func TestVMJSONDecodeOutOfRangeIsErr(t *testing.T) {
	for _, in := range []string{`1e400`, `[1, -1e400]`, `{"x":{"y":[1e400]}}`} {
		dec, err := builtinJsonDecode([]bytecode.Value{bytecode.NewString(in)})
		if err != nil {
			t.Fatal(err)
		}
		if dec.AsADT().Tag == resultTagOk {
			t.Errorf("decode(%s): expected Err, got Ok", in)
		}
	}
}
