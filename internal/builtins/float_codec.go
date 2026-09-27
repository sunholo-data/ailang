package builtins

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

// Packed little-endian float codecs (M-NUMERICS Phase 0, D4).
//
// The width is in the name so the lossy one says it is lossy: F32 rounds every
// float64 to the nearest float32, F64 is exact. Decoding rejects a byte count
// that is not a whole number of floats with an Err naming the count, instead of
// the Option the older _embedding_decode returns.
//
// The encoders over [float] are _embedding_encode (F32) and _vec_encode_f64le;
// the decoders return Result[[float], string].

func init() {
	registerVecEncodeF64LE()
	registerVecDecodeLE("_vec_decode_f32le", 4)
	registerVecDecodeLE("_vec_decode_f64le", 8)
}

func resultOk(v eval.Value) eval.Value {
	return &eval.TaggedValue{ModulePath: "std/result", TypeName: "Result", CtorName: "Ok", Fields: []eval.Value{v}}
}

func resultErr(msg string) eval.Value {
	return &eval.TaggedValue{ModulePath: "std/result", TypeName: "Result", CtorName: "Err", Fields: []eval.Value{&eval.StringValue{Value: msg}}}
}

// encodeFloatsLE packs xs as little-endian float32 (width 4) or float64 (width 8).
func encodeFloatsLE(xs []float64, width int) []byte {
	buf := make([]byte, len(xs)*width)
	for i, x := range xs {
		if width == 4 {
			binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(float32(x)))
		} else {
			binary.LittleEndian.PutUint64(buf[i*8:], math.Float64bits(x))
		}
	}
	return buf
}

// decodeFloatsLE unpacks little-endian floats of the given width. A length that
// is not a multiple of width is an error naming the count.
func decodeFloatsLE(b []byte, width int) ([]float64, error) {
	if len(b)%width != 0 {
		return nil, fmt.Errorf("decodeF%dLE: %d bytes is not a whole number of %d-byte floats", width*8, len(b), width)
	}
	out := make([]float64, len(b)/width)
	for i := range out {
		if width == 4 {
			out[i] = float64(math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:])))
		} else {
			out[i] = math.Float64frombits(binary.LittleEndian.Uint64(b[i*8:]))
		}
	}
	return out, nil
}

func registerVecEncodeF64LE() {
	registerVec("_vec_encode_f64le", "Pack a float vector as little-endian float64 bytes (exact, 8 bytes per float)", 1,
		func() types.Type {
			T := types.NewBuilder()
			return T.Func(floatListType()).Returns(T.Bytes()).Build()
		},
		func(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
			xs, err := asFloats("encodeF64LE", args[0], 0)
			if err != nil {
				return nil, err
			}
			return &eval.BytesValue{Value: encodeFloatsLE(xs, 8)}, nil
		})
}

func registerVecDecodeLE(name string, width int) {
	desc := fmt.Sprintf("Unpack little-endian float%d bytes into a float vector; Err if the length is not a multiple of %d", width*8, width)
	registerVec(name, desc, 1,
		func() types.Type {
			T := types.NewBuilder()
			return T.Func(T.Bytes()).Returns(T.App("Result", floatListType(), T.String())).Build()
		},
		func(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
			b, ok := args[0].(*eval.BytesValue)
			if !ok {
				return nil, fmt.Errorf("%s: expected bytes, got %T", name, args[0])
			}
			xs, err := decodeFloatsLE(b.Value, width)
			if err != nil {
				return resultErr(err.Error()), nil
			}
			return resultOk(floatsValue(xs)), nil
		})
}
