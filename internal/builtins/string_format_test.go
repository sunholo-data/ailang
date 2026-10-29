package builtins

import (
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

func TestFormatFixedRegistered(t *testing.T) {
	spec, ok := GetSpec("_str_formatFixed")
	if !ok {
		t.Fatal("_str_formatFixed is not registered")
	}
	if spec.Module != "std/string" || spec.NumArgs != 2 || !spec.IsPure || spec.Effect != "" {
		t.Fatalf("unexpected registration: %+v", spec)
	}
	if got := spec.Type().String(); got != "(float, int) -> string" {
		t.Fatalf("signature = %q", got)
	}
}

func TestFormatFixedGoldens(t *testing.T) {
	spec, ok := GetSpec("_str_formatFixed")
	if !ok {
		t.Fatal("_str_formatFixed is not registered")
	}
	tests := []struct {
		name string
		x    float64
		d    int
		want string
	}{
		{"audio_report", -27.060000000000002, 2, "-27.06"},
		{"zero_precision", 3.14159, 0, "3"},
		{"trailing_zero", 42, 2, "42.00"},
		{"tie_even", 0.125, 2, "0.12"},
		{"below_tie", 2.675, 2, "2.67"},
		{"binary_digits", 0.1, 20, "0.10000000000000000555"},
		{"negative_zero", math.Copysign(0, -1), 2, "-0.00"},
		{"nan", math.NaN(), 2, "NaN"},
		{"positive_infinity", math.Inf(1), 2, "+Inf"},
		{"negative_infinity", math.Inf(-1), 2, "-Inf"},
		{"max_precision", 1, 1000, "1." + strings.Repeat("0", 1000)},
		{"large", math.MaxFloat64, 2, strconv.FormatFloat(math.MaxFloat64, 'f', 2, 64)},
		{"subnormal", math.SmallestNonzeroFloat64, 1000, strconv.FormatFloat(math.SmallestNonzeroFloat64, 'f', 1000, 64)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := spec.Impl(nil, []eval.Value{&eval.FloatValue{Value: tt.x}, &eval.IntValue{Value: tt.d}})
			if err != nil {
				t.Fatal(err)
			}
			s, ok := got.(*eval.StringValue)
			if !ok || s.Value != tt.want {
				t.Fatalf("formatFixed(%v, %d) = %v, want %q", tt.x, tt.d, got, tt.want)
			}
		})
	}
}

func TestFormatFixedRejectsInvalidPrecision(t *testing.T) {
	spec, ok := GetSpec("_str_formatFixed")
	if !ok {
		t.Fatal("_str_formatFixed is not registered")
	}
	for _, d := range []int{-1, math.MinInt, 1001, math.MaxInt} {
		for _, x := range []float64{1.25, math.NaN()} {
			got, err := spec.Impl(nil, []eval.Value{&eval.FloatValue{Value: x}, &eval.IntValue{Value: d}})
			if got != nil || err == nil || !strings.Contains(err.Error(), "decimals") || !strings.Contains(err.Error(), "0..1000") {
				t.Errorf("formatFixed(%v, %d) = %v, %v; want precision error", x, d, got, err)
			}
		}
	}
}

func TestFormatFixedRejectsWrongTypes(t *testing.T) {
	spec, ok := GetSpec("_str_formatFixed")
	if !ok {
		t.Fatal("_str_formatFixed is not registered")
	}
	for _, args := range [][]eval.Value{
		{&eval.IntValue{Value: 1}, &eval.IntValue{Value: 2}},
		{&eval.FloatValue{Value: 1}, &eval.StringValue{Value: "2"}},
	} {
		if _, err := spec.Impl(nil, args); err == nil || !strings.Contains(err.Error(), "formatFixed") {
			t.Errorf("invalid arguments %v: %v", args, err)
		}
	}
}
