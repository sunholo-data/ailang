package builtins

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"

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
	// M6: the same codecs on Array[float], building the packed store directly.
	registerArrayEncodeLE("_array_encode_f32le", 4)
	registerArrayEncodeLE("_array_encode_f64le", 8)
	registerArrayDecodeLE("_array_decode_f32le", 4)
	registerArrayDecodeLE("_array_decode_f64le", 8)
	registerJSONDecodeFloatArray()
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

func registerArrayEncodeLE(name string, width int) {
	registerArrayBuiltin(name, fmt.Sprintf("Pack a float array as little-endian float%d bytes", width*8), 1,
		func() types.Type {
			T := types.NewBuilder()
			return T.Func(floatArrayType()).Returns(T.Bytes()).Build()
		},
		func(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
			xs, err := arrayFloats(fmt.Sprintf("encodeF%dLE", width*8), args[0], 0)
			if err != nil {
				return nil, err
			}
			return &eval.BytesValue{Value: encodeFloatsLE(xs, width)}, nil
		})
}

func registerArrayDecodeLE(name string, width int) {
	registerArrayBuiltin(name, fmt.Sprintf("Unpack little-endian float%d bytes into a packed float array; Err on a partial float", width*8), 1,
		func() types.Type {
			T := types.NewBuilder()
			return T.Func(T.Bytes()).Returns(T.App("Result", floatArrayType(), T.String())).Build()
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
			return resultOk(eval.NewFloatArray(xs)), nil
		})
}

// _json_decode_float_array parses a flat JSON array of numbers straight into a
// packed Array[float], without building a Json tree (the tree costs ~30x the
// payload in memory). Anything else - nesting, strings, null, a trailing comma
// - is an Err naming the byte offset.
func registerJSONDecodeFloatArray() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/json",
		Name:    "_json_decode_float_array",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type: func() types.Type {
			T := types.NewBuilder()
			return T.Func(T.String()).Returns(T.App("Result", floatArrayType(), T.String())).Build()
		},
		Impl: func(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
			s, ok := args[0].(*eval.StringValue)
			if !ok {
				return nil, fmt.Errorf("decodeFloatArray: expected string, got %T", args[0])
			}
			xs, err := parseFlatJSONFloats(s.Value)
			if err != nil {
				return resultErr("decodeFloatArray: " + err.Error()), nil
			}
			return resultOk(eval.NewFloatArray(xs)), nil
		},
		Metadata: &BuiltinMetadata{
			Description: "Parse a flat JSON array of numbers into a packed Array[float] without a Json tree",
			Since:       "v0.47.0",
			Stability:   StabilityExperimental,
			Tags:        []string{"json", "array", "float", "numeric"},
			Category:    "json",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _json_decode_float_array: %v", err))
	}
}

func isJSONSpace(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' }

// parseFlatJSONFloats accepts `[` number (`,` number)* `]` with JSON
// whitespace. Numbers follow the JSON grammar (no leading '+', no leading
// zeros, no NaN/Infinity, no hex).
func parseFlatJSONFloats(s string) ([]float64, error) {
	i := 0
	skip := func() {
		for i < len(s) && isJSONSpace(s[i]) {
			i++
		}
	}
	skip()
	if i >= len(s) || s[i] != '[' {
		return nil, fmt.Errorf("expected '[' at offset %d", i)
	}
	i++
	skip()
	var out []float64
	if i < len(s) && s[i] == ']' {
		i++
	} else {
		for {
			skip()
			start := i
			end, ok := scanJSONNumber(s, i)
			if !ok {
				return nil, fmt.Errorf("expected a number at offset %d", start)
			}
			x, err := strconv.ParseFloat(s[start:end], 64)
			if err != nil {
				return nil, fmt.Errorf("number at offset %d: %v", start, err)
			}
			out = append(out, x)
			i = end
			skip()
			if i < len(s) && s[i] == ',' {
				i++
				continue
			}
			if i < len(s) && s[i] == ']' {
				i++
				break
			}
			return nil, fmt.Errorf("expected ',' or ']' at offset %d", i)
		}
	}
	skip()
	if i != len(s) {
		return nil, fmt.Errorf("trailing data at offset %d", i)
	}
	return out, nil
}

// scanJSONNumber returns the end of the JSON number starting at i.
func scanJSONNumber(s string, i int) (int, bool) {
	digits := func() int {
		n := 0
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
			n++
		}
		return n
	}
	if i < len(s) && s[i] == '-' {
		i++
	}
	if i < len(s) && s[i] == '0' {
		i++
	} else if digits() == 0 {
		return 0, false
	}
	if i < len(s) && s[i] == '.' {
		i++
		if digits() == 0 {
			return 0, false
		}
	}
	if i < len(s) && (s[i] == 'e' || s[i] == 'E') {
		i++
		if i < len(s) && (s[i] == '+' || s[i] == '-') {
			i++
		}
		if digits() == 0 {
			return 0, false
		}
	}
	return i, true
}
