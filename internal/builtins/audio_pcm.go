package builtins

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

func init() {
	registerAudioPCMPrimitive("_audio_pcm16_to_floats", "Decode whole interleaved s16le frames to a packed float array", 4, func() types.Type {
		T := types.NewBuilder()
		return T.Func(T.Bytes(), T.Int(), T.Int(), T.Int()).Returns(T.App("Result", floatArrayType(), T.String())).Build()
	}, audioPCM16ToFloatsImpl)
	registerAudioPCMPrimitive("_audio_floats_to_pcm16", "Quantize a finite interleaved float array to s16le with explicit clipping policy", 3, func() types.Type {
		T := types.NewBuilder()
		return T.Func(floatArrayType(), T.Int(), T.Bool()).Returns(T.App("Result", T.Bytes(), T.String())).Build()
	}, audioFloatsToPCM16Impl)
}

func audioPCM16ToFloatsImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	const op = "audio.pcm16ToFloats"
	pcm, ok := args[0].(*eval.BytesValue)
	if !ok {
		return nil, fmt.Errorf("%s: expected bytes, got %T", op, args[0])
	}
	ints, err := audioIntegerArgs(op, args, 1)
	if err != nil {
		return nil, err
	}
	offset, frames, channels := ints[0], ints[1], ints[2]
	if offset < 0 || offset%2 != 0 || offset > len(pcm.Value) {
		return wrapErr(op + ": offset must be even and inside the byte buffer"), nil
	}
	if frames < 0 {
		return wrapErr(op + ": frames must be nonnegative"), nil
	}
	if channels <= 0 || channels > 65535 {
		return wrapErr(op + ": channels must be in 1..65535"), nil
	}
	frameBytes := channels * 2
	if frames > (len(pcm.Value)-offset)/frameBytes || frames > audioMaxPCMBytes/frameBytes {
		return wrapErr(op + ": requested PCM extent exceeds the buffer or 100 MiB limit"), nil
	}
	samples := frames * channels
	if samples > int(^uint(0)>>1)/8 {
		return wrapErr(op + ": decoded array length is not representable on this host"), nil
	}
	out := make([]float64, samples)
	for i := range out {
		out[i] = float64(int16(binary.LittleEndian.Uint16(pcm.Value[offset+2*i:]))) / 32768
	}
	return wrapOk(eval.NewFloatArray(out)), nil
}

func audioFloatsToPCM16Impl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	const op = "audio.floatsToPcm16"
	arr, ok := args[0].(*eval.ArrayValue)
	if !ok {
		return nil, fmt.Errorf("%s: expected Array[float], got %T", op, args[0])
	}
	channels, ok := args[1].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("%s: expected int channels, got %T", op, args[1])
	}
	clip, ok := args[2].(*eval.BoolValue)
	if !ok {
		return nil, fmt.Errorf("%s: expected bool clip policy, got %T", op, args[2])
	}
	if channels.Value <= 0 || channels.Value > 65535 {
		return wrapErr(op + ": channels must be in 1..65535"), nil
	}
	if arr.Len()%channels.Value != 0 {
		return wrapErr(op + ": sample array must contain whole interleaved frames"), nil
	}
	if arr.Len() > audioMaxPCMBytes/2 {
		return wrapErr(op + ": encoded PCM exceeds the 100 MiB limit"), nil
	}
	xs, err := arrayFloats(op, arr, 0)
	if err != nil {
		return nil, err
	}
	// Validate the entire input before allocating output: never partially encode a
	// bad array, and never interpret nonfinite data as silence under either policy.
	for i, x := range xs {
		if math.IsNaN(x) || math.IsInf(x, 0) {
			return wrapErr(fmt.Sprintf("%s: sample %d is not finite", op, i)), nil
		}
		if !clip.Value && (x < -1 || x > 1) {
			return wrapErr(fmt.Sprintf("%s: sample %d is outside [-1,1]", op, i)), nil
		}
	}
	out := make([]byte, len(xs)*2)
	for i, x := range xs {
		if x < -1 {
			x = -1
		}
		if x > 1 {
			x = 1
		}
		quantized := math.RoundToEven(x * 32768)
		if quantized < -32768 {
			quantized = -32768
		}
		if quantized > 32767 {
			quantized = 32767
		}
		binary.LittleEndian.PutUint16(out[i*2:], uint16(int16(quantized)))
	}
	return wrapOk(&eval.BytesValue{Value: out}), nil
}
