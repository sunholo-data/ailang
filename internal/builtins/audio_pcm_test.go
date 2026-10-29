package builtins

import (
	"bytes"
	"encoding/binary"
	"math"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

func TestAudioPCM16_PackedChannelRoundTrip(t *testing.T) {
	pcm := make([]byte, 0, 65536*2)
	for n := -32768; n <= 32767; n++ {
		pcm = binary.LittleEndian.AppendUint16(pcm, uint16(int16(n)))
	}
	v, err := audioPCM16ToFloatsImpl(nil, []eval.Value{&eval.BytesValue{Value: pcm}, &eval.IntValue{Value: 0}, &eval.IntValue{Value: 32768}, &eval.IntValue{Value: 2}})
	if err != nil {
		t.Fatal(err)
	}
	tv := v.(*eval.TaggedValue)
	if tv.CtorName != "Ok" {
		t.Fatal(tv)
	}
	arr := tv.Fields[0].(*eval.ArrayValue)
	xs, packed := arr.Floats()
	if !packed || len(xs) != 65536 || xs[0] != -1 || xs[32768] != 0 || xs[65535] != 32767.0/32768 {
		t.Fatalf("not exact packed interleaved output: %v %d", packed, len(xs))
	}
	v, err = audioFloatsToPCM16Impl(nil, []eval.Value{arr, &eval.IntValue{Value: 2}, &eval.BoolValue{Value: false}})
	if err != nil {
		t.Fatal(err)
	}
	tv = v.(*eval.TaggedValue)
	if tv.CtorName != "Ok" || !bytes.Equal(tv.Fields[0].(*eval.BytesValue).Value, pcm) {
		t.Fatal("all-s16 round trip differs")
	}
}

func TestAudioPCM16_OffsetsAndBounds(t *testing.T) {
	for _, tc := range []struct {
		o, f, c int
		want    string
	}{{-1, 0, 1, "offset"}, {1, 0, 1, "offset"}, {0, -1, 1, "frames"}, {0, 0, 0, "channels"}, {0, 0, 65536, "channels"}, {0, math.MaxInt, 2, "extent"}, {10, 0, 1, "offset"}, {0, 3, 1, "extent"}} {
		v, err := audioPCM16ToFloatsImpl(nil, []eval.Value{&eval.BytesValue{Value: []byte{0, 0, 1, 0}}, &eval.IntValue{Value: tc.o}, &eval.IntValue{Value: tc.f}, &eval.IntValue{Value: tc.c}})
		if err != nil {
			t.Fatal(err)
		}
		tv := v.(*eval.TaggedValue)
		if tv.CtorName != "Err" || !strings.Contains(tv.Fields[0].(*eval.StringValue).Value, tc.want) {
			t.Errorf("%+v: %v", tc, tv)
		}
	}
	v, err := audioPCM16ToFloatsImpl(nil, []eval.Value{&eval.BytesValue{Value: []byte{99, 99, 0, 128, 255, 127}}, &eval.IntValue{Value: 2}, &eval.IntValue{Value: 1}, &eval.IntValue{Value: 2}})
	if err != nil {
		t.Fatal(err)
	}
	arr := v.(*eval.TaggedValue).Fields[0].(*eval.ArrayValue)
	xs, _ := arr.Floats()
	if xs[0] != -1 || xs[1] != 32767.0/32768 {
		t.Fatal(xs)
	}
}

func TestAudioPCM16_QuantizationAndNonfinite(t *testing.T) {
	// Halfway between integers: ties to even; endpoints saturate after rounding.
	samples := []float64{-1, 1, 0.5 / 32768, 1.5 / 32768, -0.5 / 32768, -1.5 / 32768}
	v, err := audioFloatsToPCM16Impl(nil, []eval.Value{eval.NewFloatArray(samples), &eval.IntValue{Value: 1}, &eval.BoolValue{Value: false}})
	if err != nil {
		t.Fatal(err)
	}
	b := v.(*eval.TaggedValue).Fields[0].(*eval.BytesValue).Value
	for i, want := range []int16{-32768, 32767, 0, 2, 0, -2} {
		if got := int16(binary.LittleEndian.Uint16(b[i*2:])); got != want {
			t.Errorf("sample %d: %d want %d", i, got, want)
		}
	}
	for _, clip := range []bool{false, true} {
		for _, x := range []float64{math.NaN(), math.Inf(1), math.Inf(-1), 2, -2} {
			v, err := audioFloatsToPCM16Impl(nil, []eval.Value{eval.NewFloatArray([]float64{x}), &eval.IntValue{Value: 1}, &eval.BoolValue{Value: clip}})
			if err != nil {
				t.Fatal(err)
			}
			tv := v.(*eval.TaggedValue)
			wantOK := clip && !math.IsNaN(x) && !math.IsInf(x, 0)
			if (tv.CtorName == "Ok") != wantOK {
				t.Errorf("x=%v clip=%v got %v", x, clip, tv)
			}
		}
	}
}

func BenchmarkAudioPCM16Packed(b *testing.B) {
	pcm := make([]byte, 10*1024*1024)
	args := []eval.Value{&eval.BytesValue{Value: pcm}, &eval.IntValue{Value: 0}, &eval.IntValue{Value: len(pcm) / 4}, &eval.IntValue{Value: 2}}
	b.ReportAllocs()
	b.SetBytes(int64(len(pcm)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := audioPCM16ToFloatsImpl(nil, args)
		if err != nil {
			b.Fatal(err)
		}
	}
}
