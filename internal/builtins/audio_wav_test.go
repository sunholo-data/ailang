package builtins

import (
	"bytes"
	"encoding/binary"
	"math"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

func TestAudioWavHeader_CheckedBoundsAndGoldens(t *testing.T) {
	h, err := makeAudioWavHeader(1, 8000, 1, 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(h) != 44 || binary.LittleEndian.Uint32(h[4:8]) != 38 || binary.LittleEndian.Uint32(h[40:44]) != 1 {
		t.Fatalf("odd header: %v", h)
	}
	golden := []byte{82, 73, 70, 70, 136, 0, 0, 0, 87, 65, 86, 69, 102, 109, 116, 32, 16, 0, 0, 0, 1, 0, 2, 0, 128, 187, 0, 0, 0, 238, 2, 0, 4, 0, 16, 0, 100, 97, 116, 97, 100, 0, 0, 0}
	h, err = makeAudioWavHeader(100, 48000, 2, 16)
	if err != nil || !bytes.Equal(h, golden) {
		t.Fatalf("stereo golden: %v %v", h, err)
	}
	for _, tc := range []struct {
		n, r, c, b int
		field      string
	}{
		{-1, 8000, 1, 8, "dataLen"}, {0, 0, 1, 8, "rate"}, {0, 8000, 65536, 8, "channels"},
		{0, 8000, 1, 0, "bitsPerSample"}, {0, 8000, 1, 40, "bitsPerSample"}, {1, 8000, 1, 16, "frame"},
		{0, 8000, 65535, 16, "blockAlign"}, {0, math.MaxInt, 1, 8, "rate"},
		{0, math.MaxUint32, 2, 8, "byteRate"}, {math.MaxInt, 8000, 1, 8, "RIFF"},
	} {
		_, err := makeAudioWavHeader(tc.n, tc.r, tc.c, tc.b)
		if err == nil || !strings.Contains(err.Error(), tc.field) {
			t.Errorf("%+v: %v", tc, err)
		}
	}
	// Pure header-only boundaries must not allocate the conceptual PCM buffer.
	for _, n := range []int{4294967258, 4294967259} {
		_, err := makeAudioWavHeader(n, 8000, 1, 8)
		if (n == 4294967258) != (err == nil) {
			t.Errorf("boundary %d: %v", n, err)
		}
	}
	for _, bits := range []int{8, 16, 24, 32} {
		if _, err := makeAudioWavHeader(bits/8, 8000, 1, bits); err != nil {
			t.Fatal(err)
		}
	}
}

func TestAudioWavFromPCM_PadAndFailures(t *testing.T) {
	// Exercise the allocation boundary without allocating an oversized buffer.
	if err := validateAudioWavMaterialization(audioMaxPCMBytes); err != nil {
		t.Fatal(err)
	}
	if err := validateAudioWavMaterialization(audioMaxPCMBytes + 1); err == nil {
		t.Fatal("oversized materialization accepted")
	}
	b, err := makeAudioWav([]byte{128}, 8000, 1, 8)
	if err != nil || len(b) != 46 || b[44] != 128 || b[45] != 0 {
		t.Fatalf("odd PCM: %v %v", b, err)
	}
	b, err = makeAudioWav(nil, 24000, 1, 16)
	if err != nil || len(b) != 44 {
		t.Fatalf("empty: %v %v", b, err)
	}
	for _, checked := range []bool{false, true} {
		v, err := audioWavFromPCMImpl(checked)(nil, []eval.Value{&eval.BytesValue{Value: []byte{0}}, &eval.IntValue{Value: 8000}, &eval.IntValue{Value: 1}, &eval.IntValue{Value: 16}})
		if checked {
			if err != nil || v.(*eval.TaggedValue).CtorName != "Err" {
				t.Fatalf("checked: %v %v", v, err)
			}
		} else if err == nil {
			t.Fatal("legacy accepted incomplete frame without contract flag")
		}
	}
}
