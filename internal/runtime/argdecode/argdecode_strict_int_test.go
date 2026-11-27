package argdecode

import (
	"errors"
	"math"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

// TST-RC1-05
func TestDecodeJSON_StrictIntAnd64BitPrecision(t *testing.T) {
	intType := &types.TCon{Name: "int"}

	rejectCases := []struct {
		name string
		json string
	}{
		{"fractional positive", "2.9"},
		{"fractional negative", "-0.5"},
		{"fractional scientific", "1.25e1"},
		{"out of range float", "1e30"},
		{"above MaxInt64", "9223372036854775808"},
		{"below MinInt64", "-9223372036854775809"},
		{"NaN literal", "NaN"},
		{"Infinity literal", "Infinity"},
		{"trailing garbage", "42 extra"},
	}

	for _, tc := range rejectCases {
		t.Run("reject_"+tc.name, func(t *testing.T) {
			val, err := DecodeJSON(tc.json, intType)
			if err == nil {
				t.Fatalf("DecodeJSON(%q, int) succeeded with %#v; want error", tc.json, val)
			}
		})
	}

	// Direct float64 inputs to decodeInt (e.g. NaN / Inf / 2.9) must also fail with *DecodeError.
	for _, bad := range []float64{2.9, -1.1, math.NaN(), math.Inf(1), math.Inf(-1), 1e30} {
		if _, err := decodeInt(bad); err == nil {
			t.Fatalf("decodeInt(%v) succeeded; want DecodeError", bad)
		} else {
			var decErr *DecodeError
			if !errors.As(err, &decErr) {
				t.Fatalf("decodeInt(%v) error = %T (%v); want *DecodeError", bad, err, err)
			}
		}
	}

	acceptCases := []struct {
		name string
		json string
		want int
	}{
		{"small int", "42", 42},
		{"exact integral float 2.0", "2.0", 2},
		{"exact scientific 1e3", "1e3", 1000},
		{"above 2^53 exact 64-bit", "9007199254740993", 9007199254740993},
		{"MaxInt64", "9223372036854775807", math.MaxInt64},
		{"MinInt64", "-9223372036854775808", math.MinInt64},
	}

	for _, tc := range acceptCases {
		t.Run("accept_"+tc.name, func(t *testing.T) {
			val, err := DecodeJSON(tc.json, intType)
			if err != nil {
				t.Fatalf("DecodeJSON(%q, int) unexpected error: %v", tc.json, err)
			}
			iv, ok := val.(*eval.IntValue)
			if !ok {
				t.Fatalf("DecodeJSON(%q, int) = %T; want *eval.IntValue", tc.json, val)
			}
			if iv.Value != tc.want {
				t.Fatalf("DecodeJSON(%q, int) = %d; want %d", tc.json, iv.Value, tc.want)
			}
		})
	}
}
