package builtins

// std/audio builtins. M-STD-AUDIO (v0.44.0).
//
// The WAV helpers in std/audio are pure AILANG over std/bytes; only the codec
// lives in Go, because a CELT encoder in AILANG would be orders of magnitude
// too slow (a single map pass over 50 s of PCM already costs 3.5 s).
//
// The codec is a vendored copy of github.com/tphakala/go-opus v1.1.0
// (BSD-3-Clause, third_party/goopus): pure Go, no cgo, fixed-point, so it
// builds for js/wasm and gives the same bytes on every architecture.

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

const (
	audioModule      = "std/audio"
	audioMinBitrate  = 6000
	audioMaxBitrate  = 510000
	audioMaxPCMBytes = 100 * 1024 * 1024 // same cap as std/deflate and std/gzip
)

func init() {
	registerAudioEncodeOggOpus()
}

func registerAudioEncodeOggOpus() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  audioModule,
		Name:    "_audio_encode_ogg_opus",
		NumArgs: 4,
		IsPure:  true,
		Effect:  "",
		Type: func() types.Type {
			T := types.NewBuilder()
			return T.Func(T.Bytes(), T.Int(), T.Int(), T.Int()).Returns(
				T.App("Result", T.Bytes(), T.String()),
			).Build()
		},
		Impl: audioEncodeOggOpusImpl,
		Metadata: &BuiltinMetadata{
			Description: "Encode 16-bit PCM to a deterministic Ogg Opus file",
			LongDesc:    "Compresses interleaved 16-bit little-endian PCM to an Ogg Opus stream (RFC 7845). Rate must be 8000, 12000, 16000, 24000 or 48000 Hz; channels 1 or 2; bitrate 6000..510000 bits/s; PCM a whole number of samples and at most 100 MB. Invalid input returns Err naming the field, and is never clamped. The same input always gives the same bytes: the Ogg serial is a hash of the input rather than random.",
			Params: []ParamDoc{
				{Name: "pcm", Description: "Interleaved s16le PCM"},
				{Name: "rate", Description: "Sample rate in Hz"},
				{Name: "channels", Description: "1 (mono) or 2 (stereo)"},
				{Name: "bitrate", Description: "Target bitrate in bits/s"},
			},
			Returns: "Result[bytes, string] — Ok(Ogg Opus file bytes) or Err(message)",
			Examples: []Example{
				{Code: `_audio_encode_ogg_opus(pcm, 24000, 1, 24000)`, Description: "50 s of 24 kHz mono speech becomes about 230 KB"},
			},
			SeeAlso:   []string{"_bytes_from_ints"},
			Since:     "v0.44.0",
			Stability: StabilityStable,
			Tags:      []string{"audio", "opus", "ogg", "encode", "pure"},
			Category:  "audio",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _audio_encode_ogg_opus: %v", err))
	}
}

func audioEncodeOggOpusImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	pcm, ok := args[0].(*eval.BytesValue)
	if !ok {
		return nil, fmt.Errorf("_audio_encode_ogg_opus: expected Bytes, got %T", args[0])
	}
	ints := make([]int, 3)
	for i, a := range args[1:] {
		iv, ok := a.(*eval.IntValue)
		if !ok {
			return nil, fmt.Errorf("_audio_encode_ogg_opus: expected Int for argument %d, got %T", i+2, a)
		}
		ints[i] = iv.Value
	}
	rate, channels, bitrate := ints[0], ints[1], ints[2]
	if err := validateAudioEncode(len(pcm.Value), rate, channels, bitrate); err != nil {
		return wrapErr("audio.encode: " + err.Error()), nil
	}
	out, err := encodeOggOpus(pcm.Value, rate, channels, bitrate)
	if err != nil {
		return wrapErr("audio.encode: " + err.Error()), nil
	}
	return wrapOk(&eval.BytesValue{Value: out}), nil
}
