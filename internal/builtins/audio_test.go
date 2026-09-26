package builtins

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/third_party/goopus/oggopus"
)

// synthSpeechPCM returns deterministic speech-like s16le PCM: a pitch-glided
// square-ish carrier under a syllable envelope plus LCG noise. Integer-only on
// purpose, so the fixture is identical on every architecture and the golden
// hash below is a statement about the encoder, not about math.Sin.
func synthSpeechPCM(rate, channels int, ms int) []byte {
	n := rate * ms / 1000
	out := make([]byte, 0, n*channels*2)
	seed := uint32(12345)
	phase := 0
	for i := range n {
		period := rate / (110 + (i/(rate/8))%5*20) // 110..190 Hz pitch steps
		phase = (phase + 1) % period
		carrier := 6000
		if phase > period/3 {
			carrier = -3000
		}
		syll := i % (rate / 4) // 250 ms syllables
		env := syll
		if half := rate / 8; syll > half {
			env = rate/4 - syll
		}
		env = env * 1024 / (rate / 8) // 0..1024
		seed = seed*1664525 + 1013904223
		noise := int(seed>>20) - 2048
		s := (carrier*env)/1024 + noise
		for c := range channels {
			v := s
			if c == 1 {
				v = s / 2
			}
			out = binary.LittleEndian.AppendUint16(out, uint16(int16(v)))
		}
	}
	return out
}

func mustEncode(t *testing.T, pcm []byte, rate, channels, bitrate int) []byte {
	t.Helper()
	v, err := audioEncodeOggOpusImpl(nil, []eval.Value{
		&eval.BytesValue{Value: pcm},
		&eval.IntValue{Value: rate}, &eval.IntValue{Value: channels}, &eval.IntValue{Value: bitrate},
	})
	if err != nil {
		t.Fatalf("impl error: %v", err)
	}
	tv := v.(*eval.TaggedValue)
	if tv.CtorName != "Ok" {
		t.Fatalf("encode returned %s(%v)", tv.CtorName, tv.Fields[0])
	}
	return tv.Fields[0].(*eval.BytesValue).Value
}

func encodeErr(t *testing.T, pcm []byte, rate, channels, bitrate int) string {
	t.Helper()
	v, err := audioEncodeOggOpusImpl(nil, []eval.Value{
		&eval.BytesValue{Value: pcm},
		&eval.IntValue{Value: rate}, &eval.IntValue{Value: channels}, &eval.IntValue{Value: bitrate},
	})
	if err != nil {
		t.Fatalf("impl error: %v", err)
	}
	tv := v.(*eval.TaggedValue)
	if tv.CtorName != "Err" {
		t.Fatalf("expected Err for rate=%d channels=%d bitrate=%d len=%d, got Ok", rate, channels, bitrate, len(pcm))
	}
	return tv.Fields[0].(*eval.StringValue).Value
}

// goldenOggOpus5sSHA256 pins the encoder output for the 5 s fixture. The codec
// is fixed-point, so this must hold on amd64 and arm64 alike. A change here
// means the bytes std/audio.encode produces changed: bump it deliberately.
const goldenOggOpus5sSHA256 = "76dab0bc613b06d3002f0c4327743b19118ca349ed93e9c418b9d78c4d404322"

func TestAudioEncode_DeterministicAndGolden(t *testing.T) {
	pcm := synthSpeechPCM(24000, 1, 5000)
	a := mustEncode(t, pcm, 24000, 1, 24000)
	b := mustEncode(t, pcm, 24000, 1, 24000)
	if !bytes.Equal(a, b) {
		t.Fatal("two encodes of the same input differ")
	}
	sum := sha256.Sum256(a)
	if got := hex.EncodeToString(sum[:]); got != goldenOggOpus5sSHA256 {
		t.Errorf("golden hash mismatch:\n got  %s\n want %s", got, goldenOggOpus5sSHA256)
	}
	if len(a) >= 40*1024 {
		t.Errorf("5 s at 24 kbps is %d bytes, want < 40 KB", len(a))
	}
}

func TestAudioEncode_DifferentInputsGetDifferentSerials(t *testing.T) {
	a := mustEncode(t, synthSpeechPCM(24000, 1, 200), 24000, 1, 24000)
	b := mustEncode(t, synthSpeechPCM(24000, 1, 220), 24000, 1, 24000)
	if bytes.Equal(a[14:18], b[14:18]) {
		t.Error("different inputs produced the same Ogg serial")
	}
}

func TestAudioEncode_50sUnder400KB(t *testing.T) {
	pcm := synthSpeechPCM(24000, 1, 50000)
	out := mustEncode(t, pcm, 24000, 1, 24000)
	t.Logf("50 s: %d PCM bytes -> %d Ogg Opus bytes", len(pcm), len(out))
	if len(out) >= 400*1024 {
		t.Errorf("50 s at 24 kbps is %d bytes, want < 400 KB", len(out))
	}
}

// Decode with the codec's own reader: CRCs, page structure and granule
// accounting must all check out, and the decoded length must equal the input
// length exactly (end trim) at the 48 kHz output rate.
func TestAudioEncode_RoundTrip(t *testing.T) {
	cases := []struct {
		rate, channels, ms int
	}{
		{24000, 1, 5000},
		{24000, 1, 1234}, // partial final frame
		{48000, 2, 1000},
		{8000, 1, 700},
		{16000, 2, 20}, // exactly one frame
		{24000, 1, 0},  // empty input still yields a valid stream
	}
	for _, c := range cases {
		pcm := synthSpeechPCM(c.rate, c.channels, c.ms)
		out := mustEncode(t, pcm, c.rate, c.channels, 32000)

		if string(out[0:4]) != "OggS" || out[5]&oggFlagBOS == 0 {
			t.Fatalf("%+v: first page is not a BOS Ogg page", c)
		}
		head := out[28:47]
		if string(head[:8]) != "OpusHead" || head[8] != 1 || int(head[9]) != c.channels ||
			int(binary.LittleEndian.Uint32(head[12:16])) != c.rate {
			t.Fatalf("%+v: bad OpusHead % x", c, head)
		}

		dec, info, err := oggopus.DecodeInterleaved(bytes.NewReader(out))
		if err != nil {
			t.Fatalf("%+v: decode: %v", c, err)
		}
		if info.Channels != c.channels || int(info.InputSampleRate) != c.rate {
			t.Errorf("%+v: info %+v", c, info)
		}
		inSamples := len(pcm) / (2 * c.channels)
		want48k := inSamples * 48000 / c.rate
		got48k := len(dec) / (2 * c.channels)
		if got48k != want48k {
			t.Errorf("%+v: decoded %d samples at 48 kHz, want %d", c, got48k, want48k)
		}
	}
}

func TestAudioEncode_RejectsBadInput(t *testing.T) {
	good := synthSpeechPCM(24000, 1, 100)
	cases := []struct {
		name                    string
		pcm                     []byte
		rate, channels, bitrate int
		field                   string
	}{
		{"rate 44100", good, 44100, 1, 24000, "rate"},
		{"3 channels", synthSpeechPCM(24000, 1, 60)[:2*3*100], 24000, 3, 24000, "channels"},
		{"odd length", good[:len(good)-1], 24000, 1, 24000, "pcm length"},
		{"stereo half-sample", good[:len(good)-2], 24000, 2, 24000, "pcm length"},
		{"bitrate 0", good, 24000, 1, 0, "bitrate"},
		{"bitrate 1_000_000", good, 24000, 1, 1000000, "bitrate"},
	}
	for _, c := range cases {
		msg := encodeErr(t, c.pcm, c.rate, c.channels, c.bitrate)
		if !strings.Contains(msg, c.field) {
			t.Errorf("%s: error %q does not name %q", c.name, msg, c.field)
		}
	}
}

func TestAudioEncode_RejectsOversizePCM(t *testing.T) {
	if err := validateAudioEncode(audioMaxPCMBytes+2, 24000, 1, 24000); err == nil ||
		!strings.Contains(err.Error(), "pcm length") {
		t.Fatalf("oversize PCM: got %v", err)
	}
}

// RFC 3533 CRC: a known vector keeps us from drifting to hash/crc32's reflected
// IEEE variant, which would produce pages every reader rejects.
func TestOggCRC_KnownVector(t *testing.T) {
	if got := oggCRC([]byte("123456789")); got != 0x89A1897F {
		t.Fatalf("oggCRC(\"123456789\") = %#08x, want 0x89a1897f", got)
	}
}
