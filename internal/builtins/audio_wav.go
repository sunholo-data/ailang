package builtins

// Checked canonical RIFF/WAVE writing. The pure std/audio reader uses the
// checked header boundary to share the same field/whole-frame validation.

import (
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

const audioMaxRIFFField = uint64(1<<32 - 1)

func init() {
	for _, checked := range []bool{false, true} {
		suffix := ""
		if checked {
			suffix = "_checked"
		}
		registerAudioPCMPrimitive("_audio_wav_header"+suffix, "Construct a validated canonical PCM WAV header", 4,
			func() types.Type {
				T := types.NewBuilder()
				result := T.Bytes()
				if checked {
					result = T.App("Result", T.Bytes(), T.String())
				}
				return T.Func(T.Int(), T.Int(), T.Int(), T.Int()).Returns(result).Build()
			}, audioWavHeaderImpl(checked))
		registerAudioPCMPrimitive("_audio_wav_from_pcm"+suffix, "Wrap complete integer PCM frames as a padded RIFF/WAVE file", 4,
			func() types.Type {
				T := types.NewBuilder()
				result := T.Bytes()
				if checked {
					result = T.App("Result", T.Bytes(), T.String())
				}
				return T.Func(T.Bytes(), T.Int(), T.Int(), T.Int()).Returns(result).Build()
			}, audioWavFromPCMImpl(checked))
	}
}

func registerAudioPCMPrimitive(name, description string, nargs int, typ func() types.Type, impl EffectImpl) {
	if err := RegisterEffectBuiltin(BuiltinSpec{Module: audioModule, Name: name, NumArgs: nargs, IsPure: true, Type: typ, Impl: impl,
		Metadata: audioPCMMetadata(name, description)}); err != nil {
		panic(fmt.Sprintf("register %s: %v", name, err))
	}
}

func audioPCMMetadata(name, description string) *BuiltinMetadata {
	m := &BuiltinMetadata{Description: description, Since: "v0.54.18", Stability: StabilityStable,
		Tags: []string{"audio", "pcm", "pure", "checked"}, Category: "audio"}
	switch {
	case strings.HasPrefix(name, "_audio_wav_header"), strings.HasPrefix(name, "_audio_wav_from_pcm"):
		first := ParamDoc{Name: "dataLen", Description: "PCM bytes, excluding an odd-byte pad; whole frames only"}
		if strings.HasPrefix(name, "_audio_wav_from_pcm") {
			first = ParamDoc{Name: "pcm", Description: "Complete interleaved integer PCM frames, at most 100 MiB"}
		}
		m.Params = []ParamDoc{first, {Name: "rate", Description: "Positive sample rate fitting WAV rate and byteRate fields"},
			{Name: "channels", Description: "1..65535, with blockAlign fitting uint16"}, {Name: "bitsPerSample", Description: "Integer PCM container bits: 8, 16, 24 or 32"}}
		m.Returns = "bytes; invalid fields fail loudly in ordinary execution"
		if strings.HasSuffix(name, "_checked") {
			m.Returns = "Result[bytes, string]; Err names an invalid format, frame or size field"
		}
		m.LongDesc = "Writes canonical format-tag-1 integer PCM. RIFF size includes odd-data padding; data chunk size excludes it. Header-only output is 44 bytes; the complete-file writer appends a zero pad when needed. Validates widths, products, whole frames and RIFF limits before allocation. Float and extensible WAV are not written."
		m.Examples = []Example{{Code: "_audio_wav_header_checked(0, 8000, 1, 16)", Description: "An empty mono PCM16 header"}}
	case name == "_audio_pcm16_to_floats":
		m.Params = []ParamDoc{{Name: "pcm", Description: "s16le bytes"}, {Name: "offset", Description: "Even byte index"},
			{Name: "frames", Description: "Number of complete frames, not scalar samples"}, {Name: "channels", Description: "1..65535; all channels are retained interleaved"}}
		m.Returns = "Result[Array[float], string]; owned packed s16/32768 samples"
		m.LongDesc = "Checks offset, frame count, extent, host size and the 100 MiB PCM limit before allocation. Decoding raw bytes does not validate WAV metadata; use wavInfo's dataOffset, frames and channels for a PCM16 WAV."
		m.Examples = []Example{{Code: "_audio_pcm16_to_floats(pcm, 0, 1, 2)", Description: "Decode one stereo frame to two samples"}}
	case name == "_audio_floats_to_pcm16":
		m.Params = []ParamDoc{{Name: "samples", Description: "Finite interleaved Array[float]"}, {Name: "channels", Description: "1..65535; sample count must be a multiple"},
			{Name: "clip", Description: "false rejects outside [-1,1]; true clamps finite values"}}
		m.Returns = "Result[bytes, string]; interleaved s16le PCM"
		m.LongDesc = "Both policies reject NaN and infinity. Quantizes x*32768 to nearest ties-to-even, then saturates to s16. Checks whole frames and 100 MiB output cap before allocation. Public std/audio exposes an explicit Pcm16Quantization ADT."
		m.Examples = []Example{{Code: "_audio_floats_to_pcm16(samples, 2, false)", Description: "Checked stereo conversion without clipping"}}
	}
	return m
}

// validateAudioWavFormat checks widths before products or narrowing conversions.
// Header-only calls may describe a large buffer without allocating that buffer.
func validateAudioWavFormat(dataLen, rate, channels, bits int) (uint16, uint32, error) {
	if dataLen < 0 {
		return 0, 0, fmt.Errorf("dataLen must be nonnegative (got %d)", dataLen)
	}
	if rate <= 0 || uint64(rate) > audioMaxRIFFField {
		return 0, 0, fmt.Errorf("rate must be in 1..4294967295 (got %d)", rate)
	}
	if channels <= 0 || channels > 65535 {
		return 0, 0, fmt.Errorf("channels must be in 1..65535 (got %d)", channels)
	}
	if bits != 8 && bits != 16 && bits != 24 && bits != 32 {
		return 0, 0, fmt.Errorf("bitsPerSample must be 8, 16, 24 or 32 integer PCM (got %d)", bits)
	}
	bytesPerSample := bits / 8
	if channels > 65535/bytesPerSample {
		return 0, 0, fmt.Errorf("blockAlign exceeds the 16-bit field")
	}
	align := channels * bytesPerSample
	if uint64(rate) > audioMaxRIFFField/uint64(align) {
		return 0, 0, fmt.Errorf("byteRate exceeds the 32-bit field")
	}
	if dataLen%align != 0 {
		return 0, 0, fmt.Errorf("dataLen %d is not a whole %d-byte frame", dataLen, align)
	}
	n := uint64(dataLen)
	pad := n % 2
	if n > audioMaxRIFFField-36-pad {
		return 0, 0, fmt.Errorf("RIFF size exceeds the 32-bit field")
	}
	return uint16(align), uint32(uint64(rate) * uint64(align)), nil
}

func makeAudioWavHeader(dataLen, rate, channels, bits int) ([]byte, error) {
	align, byteRate, err := validateAudioWavFormat(dataLen, rate, channels, bits)
	if err != nil {
		return nil, err
	}
	h := make([]byte, 44)
	copy(h, "RIFF")
	binary.LittleEndian.PutUint32(h[4:], uint32(36+uint64(dataLen)+uint64(dataLen%2)))
	copy(h[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(h[16:], 16)
	binary.LittleEndian.PutUint16(h[20:], 1)
	binary.LittleEndian.PutUint16(h[22:], uint16(channels))
	binary.LittleEndian.PutUint32(h[24:], uint32(rate))
	binary.LittleEndian.PutUint32(h[28:], byteRate)
	binary.LittleEndian.PutUint16(h[32:], align)
	binary.LittleEndian.PutUint16(h[34:], uint16(bits))
	copy(h[36:], "data")
	binary.LittleEndian.PutUint32(h[40:], uint32(dataLen))
	return h, nil
}

func makeAudioWav(pcm []byte, rate, channels, bits int) ([]byte, error) {
	if err := validateAudioWavMaterialization(len(pcm)); err != nil {
		return nil, err
	}
	h, err := makeAudioWavHeader(len(pcm), rate, channels, bits)
	if err != nil {
		return nil, err
	}
	pad := len(pcm) % 2
	maxInt := int(^uint(0) >> 1)
	if len(pcm) > maxInt-44-pad {
		return nil, fmt.Errorf("WAV output length is not representable on this host")
	}
	out := make([]byte, 44+len(pcm)+pad)
	copy(out, h)
	copy(out[44:], pcm)
	return out, nil
}

func validateAudioWavMaterialization(dataLen int) error {
	if dataLen > audioMaxPCMBytes {
		return fmt.Errorf("pcm length %d exceeds %d-byte limit", dataLen, audioMaxPCMBytes)
	}
	return nil
}

func audioIntegerArgs(op string, args []eval.Value, start int) ([]int, error) {
	out := make([]int, len(args)-start)
	for i, a := range args[start:] {
		v, ok := a.(*eval.IntValue)
		if !ok {
			return nil, fmt.Errorf("%s: expected int at argument %d, got %T", op, start+i+1, a)
		}
		out[i] = v.Value
	}
	return out, nil
}

func audioWavResult(op string, b []byte, err error, checked bool) (eval.Value, error) {
	if err != nil {
		if checked {
			return wrapErr(op + ": " + err.Error()), nil
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	value := &eval.BytesValue{Value: b}
	if checked {
		return wrapOk(value), nil
	}
	return value, nil
}

func audioWavHeaderImpl(checked bool) EffectImpl {
	return func(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
		ints, err := audioIntegerArgs("audio.wavHeader", args, 0)
		if err != nil {
			return nil, err
		}
		b, err := makeAudioWavHeader(ints[0], ints[1], ints[2], ints[3])
		return audioWavResult("audio.wavHeader", b, err, checked)
	}
}

func audioWavFromPCMImpl(checked bool) EffectImpl {
	return func(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
		pcm, ok := args[0].(*eval.BytesValue)
		if !ok {
			return nil, fmt.Errorf("audio.wavFromPcm: expected bytes, got %T", args[0])
		}
		ints, err := audioIntegerArgs("audio.wavFromPcm", args, 1)
		if err != nil {
			return nil, err
		}
		b, err := makeAudioWav(pcm.Value, ints[0], ints[1], ints[2])
		return audioWavResult("audio.wavFromPcm", b, err, checked)
	}
}
