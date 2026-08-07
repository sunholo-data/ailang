// Int <-> byte conversion and byte-level indexing for std/bytes.
//
// Split out of bytes.go: these three builtins are the only ones that cross the
// bytes/int boundary, and they are the ones AILANG code reaches for when it has
// to look at a byte AS A NUMBER — charset transcoding, checksums, binary
// parsing. fromInts and toInts are exact inverses; byteAt is the single-index
// form of toInts.
package builtins

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

// registerBytesFromInts registers the _bytes_from_ints builtin
func registerBytesFromInts() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/bytes",
		Name:    "_bytes_from_ints",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type: func() types.Type {
			T := types.NewBuilder()
			return T.Func(T.List(T.Int())).Returns(T.Bytes()).Build()
		},
		Impl: bytesFromIntsImpl,

		Metadata: &BuiltinMetadata{
			Description: "Construct bytes from a list of integers (0-255)",
			LongDesc:    "Creates a byte slice from integer values. Each integer must be in range 0-255. Useful for building binary headers (WAV, PNG) where specific byte values are needed at exact positions.",
			Params: []ParamDoc{
				{Name: "xs", Description: "List of integers (each 0-255)"},
			},
			Returns: "Byte slice with one byte per integer",
			Examples: []Example{
				{Code: `_bytes_from_ints([0x52, 0x49, 0x46, 0x46])`, Description: "Returns bytes for \"RIFF\" (WAV header magic)"},
			},
			SeeAlso:   []string{"_bytes_from_string", "_bytes_concat"},
			Since:     "v0.8.2",
			Stability: StabilityStable,
			Tags:      []string{"bytes", "construct", "binary", "header"},
			Category:  "bytes",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _bytes_from_ints: %v", err))
	}
}

// bytesFromIntsImpl is the implementation for _bytes_from_ints
func bytesFromIntsImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	listVal, ok := args[0].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("_bytes_from_ints: expected List, got %T", args[0])
	}

	result := make([]byte, len(listVal.Elements))
	for i, elem := range listVal.Elements {
		iv, ok := elem.(*eval.IntValue)
		if !ok {
			return nil, fmt.Errorf("_bytes_from_ints: expected Int element at index %d, got %T", i, elem)
		}
		if iv.Value < 0 || iv.Value > 255 {
			return nil, fmt.Errorf("_bytes_from_ints: value %d at index %d out of byte range (0-255)", iv.Value, i)
		}
		result[i] = byte(iv.Value)
	}
	return &eval.BytesValue{Value: result}, nil
}

// registerBytesByteAt registers the _bytes_byte_at builtin
// Returns the byte value (0-255) at the given index, or None if out of bounds.
// Inverse of fromInts: byteAt(fromInts([65]), 0) == Some(65).
func registerBytesByteAt() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/bytes",
		Name:    "_bytes_byte_at",
		NumArgs: 2,
		IsPure:  true,
		Effect:  "",
		Type: func() types.Type {
			T := types.NewBuilder()
			return T.Func(T.Bytes(), T.Int()).Returns(
				T.App("Option", T.Int()),
			).Build()
		},
		Impl: func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
			bytesVal, ok := args[0].(*eval.BytesValue)
			if !ok {
				return nil, fmt.Errorf("_bytes_byte_at: expected Bytes, got %T", args[0])
			}
			idxVal, ok := args[1].(*eval.IntValue)
			if !ok {
				return nil, fmt.Errorf("_bytes_byte_at: expected Int for index, got %T", args[1])
			}
			i := idxVal.Value
			if i < 0 || i >= len(bytesVal.Value) {
				return &eval.TaggedValue{
					ModulePath: "std/option",
					TypeName:   "Option",
					CtorName:   "None",
					Fields:     []eval.Value{},
				}, nil
			}
			return &eval.TaggedValue{
				ModulePath: "std/option",
				TypeName:   "Option",
				CtorName:   "Some",
				Fields:     []eval.Value{&eval.IntValue{Value: int(bytesVal.Value[i])}},
			}, nil
		},

		Metadata: &BuiltinMetadata{
			Description: "Get the byte value at the given index",
			LongDesc:    "Returns Some(b[i]) where 0 <= b[i] <= 255 if the index is in bounds, otherwise None. The inverse of _bytes_from_ints: round-trip through fromInts/byteAt preserves byte values exactly. For UTF-8 strings, byteAt returns the raw byte at that position — not the Unicode codepoint (e.g. byteAt(fromString(\"é\"), 0) == Some(195), the first UTF-8 byte).",
			Params: []ParamDoc{
				{Name: "b", Description: "The byte slice to index into"},
				{Name: "i", Description: "Zero-based byte offset"},
			},
			Returns: "Option[int]: Some(byte value 0-255) if in bounds, None otherwise",
			Examples: []Example{
				{Code: `_bytes_byte_at(_bytes_from_string("A"), 0)`, Description: "Returns Some(65) (ASCII 'A')"},
				{Code: `_bytes_byte_at(_bytes_from_string("hello"), 4)`, Description: "Returns Some(111) (ASCII 'o')"},
				{Code: `_bytes_byte_at(_bytes_from_string("abc"), 10)`, Description: "Returns None (out of bounds)"},
			},
			SeeAlso:   []string{"_bytes_from_ints", "_bytes_length", "_bytes_slice"},
			Since:     "v0.21.0",
			Stability: StabilityStable,
			Tags:      []string{"bytes", "index", "ascii", "char-code"},
			Category:  "bytes",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _bytes_byte_at: %v", err))
	}
}

// registerBytesToInts registers the _bytes_to_ints builtin
// Returns every byte value (0-255) as a list of ints — the whole-slice
// counterpart to _bytes_byte_at. Exact inverse of _bytes_from_ints.
func registerBytesToInts() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/bytes",
		Name:    "_bytes_to_ints",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type: func() types.Type {
			T := types.NewBuilder()
			return T.Func(T.Bytes()).Returns(T.List(T.Int())).Build()
		},
		Impl: bytesToIntsImpl,

		Metadata: &BuiltinMetadata{
			Description: "Get all byte values as a list of ints (0-255)",
			LongDesc:    "Returns every byte of the slice as an int in 0-255, in order. The exact inverse of _bytes_from_ints: _bytes_to_ints(_bytes_from_ints(xs)) == xs for any list of ints in range. Where _bytes_byte_at reads one index, this exposes the whole slice to the standard list combinators (map/filter/foldl) — the shape needed for transcoding, checksums, and byte-level parsing. For UTF-8 strings, these are raw bytes, NOT Unicode codepoints (e.g. _bytes_to_ints(_bytes_from_string(\"é\")) == [195, 169], two bytes for one character).",
			Params: []ParamDoc{
				{Name: "b", Description: "The byte slice to expand"},
			},
			Returns: "List of ints, one per byte, each 0-255 (empty list for empty bytes)",
			Examples: []Example{
				{Code: `_bytes_to_ints(_bytes_from_string("AB"))`, Description: "Returns [65, 66]"},
				{Code: `_bytes_to_ints(_bytes_from_ints([0x52, 0x49, 0x46, 0x46]))`, Description: "Returns [82, 73, 70, 70] (round-trips exactly)"},
				{Code: `_bytes_to_ints(_bytes_from_string(""))`, Description: "Returns [] (empty list)"},
			},
			SeeAlso:   []string{"_bytes_from_ints", "_bytes_byte_at", "_bytes_length"},
			Since:     "v0.34.0",
			Stability: StabilityStable,
			Tags:      []string{"bytes", "list", "transcode", "char-code"},
			Category:  "bytes",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _bytes_to_ints: %v", err))
	}
}

// bytesToIntsImpl is the implementation for _bytes_to_ints
func bytesToIntsImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	bytesVal, ok := args[0].(*eval.BytesValue)
	if !ok {
		return nil, fmt.Errorf("_bytes_to_ints: expected Bytes, got %T", args[0])
	}

	elements := make([]eval.Value, len(bytesVal.Value))
	for i, b := range bytesVal.Value {
		elements[i] = &eval.IntValue{Value: int(b)}
	}
	return &eval.ListValue{Elements: elements}, nil
}
