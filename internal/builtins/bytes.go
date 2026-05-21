package builtins

import (
	"encoding/base64"
	"fmt"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

// Bytes builtin functions for AILANG
// These provide UTF-8 encoding/decoding and base64 operations
// Part of M-DX15 (Semantic Caching MVP)

func init() {
	registerBytesFromString()
	registerBytesToString()
	registerBytesToBase64()
	registerBytesFromBase64()
	registerBytesLength()
	registerBytesSlice()
	registerBytesConcat()
	registerBytesConcatList()
	registerBytesFromInts()
	registerBytesByteAt()
	registerBytesFilename()
	registerBytesMimeType()
	registerBytesFromBase64URL()
}

// ============================================================================
// Bytes Primitive Builtins
// ============================================================================

// registerBytesFromString registers the _bytes_from_string builtin
func registerBytesFromString() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/bytes",
		Name:    "_bytes_from_string",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "", // Pure function
		Type:    makeBytesFromStringType,
		Impl:    bytesFromStringImpl,

		Metadata: &BuiltinMetadata{
			Description: "Convert a UTF-8 string to bytes",
			LongDesc:    "Encodes the input string as UTF-8 bytes. This is a pure operation that simply exposes the underlying UTF-8 encoding of Go strings.",
			Params: []ParamDoc{
				{Name: "s", Description: "The string to convert to bytes"},
			},
			Returns: "UTF-8 encoded bytes",
			Examples: []Example{
				{Code: `_bytes_from_string("hello")`, Description: "Returns 5 bytes: [104, 101, 108, 108, 111]"},
				{Code: `_bytes_from_string("🎉")`, Description: "Returns 4 bytes (UTF-8 encoding of emoji)"},
			},
			SeeAlso:   []string{"_bytes_to_string", "_bytes_length"},
			Since:     "v0.5.11",
			Stability: StabilityStable,
			Tags:      []string{"bytes", "string", "encoding", "utf8"},
			Category:  "bytes",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _bytes_from_string: %v", err))
	}
}

// makeBytesFromStringType builds the type signature for _bytes_from_string
// Type: string -> bytes
func makeBytesFromStringType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String()).Returns(T.Bytes()).Build()
}

// bytesFromStringImpl is the implementation for _bytes_from_string
func bytesFromStringImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	strVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_bytes_from_string: expected String, got %T", args[0])
	}

	return &eval.BytesValue{Value: []byte(strVal.Value)}, nil
}

// registerBytesToString registers the _bytes_to_string builtin
func registerBytesToString() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/bytes",
		Name:    "_bytes_to_string",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeBytesToStringType,
		Impl:    bytesToStringImpl,

		Metadata: &BuiltinMetadata{
			Description: "Convert bytes to a UTF-8 string",
			LongDesc:    "Decodes bytes as a UTF-8 string. Invalid UTF-8 sequences are replaced with the Unicode replacement character (U+FFFD). For arbitrary binary data, use base64 encoding instead.",
			Params: []ParamDoc{
				{Name: "b", Description: "The bytes to convert to a string"},
			},
			Returns: "UTF-8 decoded string",
			Examples: []Example{
				{Code: `_bytes_to_string(bytes)`, Description: "Decodes bytes as UTF-8 string"},
			},
			SeeAlso:   []string{"_bytes_from_string", "_bytes_to_base64"},
			Since:     "v0.5.11",
			Stability: StabilityStable,
			Tags:      []string{"bytes", "string", "decoding", "utf8"},
			Category:  "bytes",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _bytes_to_string: %v", err))
	}
}

// makeBytesToStringType builds the type signature for _bytes_to_string
// Type: bytes -> string
func makeBytesToStringType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.Bytes()).Returns(T.String()).Build()
}

// bytesToStringImpl is the implementation for _bytes_to_string
func bytesToStringImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	bytesVal, ok := args[0].(*eval.BytesValue)
	if !ok {
		return nil, fmt.Errorf("_bytes_to_string: expected Bytes, got %T", args[0])
	}

	return &eval.StringValue{Value: string(bytesVal.Value)}, nil
}

// registerBytesToBase64 registers the _bytes_to_base64 builtin
func registerBytesToBase64() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/bytes",
		Name:    "_bytes_to_base64",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeBytesToBase64Type,
		Impl:    bytesToBase64Impl,

		Metadata: &BuiltinMetadata{
			Description: "Encode bytes as a base64 string",
			LongDesc:    "Encodes bytes using standard base64 encoding (RFC 4648). This is useful for transmitting binary data as text, e.g., in JSON or URLs.",
			Params: []ParamDoc{
				{Name: "b", Description: "The bytes to encode"},
			},
			Returns: "Base64 encoded string",
			Examples: []Example{
				{Code: `_bytes_to_base64(bytes_from_string("hello"))`, Description: `Returns "aGVsbG8="`},
			},
			SeeAlso:   []string{"_bytes_from_base64", "_bytes_from_string"},
			Since:     "v0.5.11",
			Stability: StabilityStable,
			Tags:      []string{"bytes", "base64", "encoding", "serialization"},
			Category:  "bytes",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _bytes_to_base64: %v", err))
	}
}

// makeBytesToBase64Type builds the type signature for _bytes_to_base64
// Type: bytes -> string
func makeBytesToBase64Type() types.Type {
	T := types.NewBuilder()
	return T.Func(T.Bytes()).Returns(T.String()).Build()
}

// bytesToBase64Impl is the implementation for _bytes_to_base64
func bytesToBase64Impl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	bytesVal, ok := args[0].(*eval.BytesValue)
	if !ok {
		return nil, fmt.Errorf("_bytes_to_base64: expected Bytes, got %T", args[0])
	}

	encoded := base64.StdEncoding.EncodeToString(bytesVal.Value)
	return &eval.StringValue{Value: encoded}, nil
}

// registerBytesFromBase64 registers the _bytes_from_base64 builtin
func registerBytesFromBase64() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/bytes",
		Name:    "_bytes_from_base64",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeBytesFromBase64Type,
		Impl:    bytesFromBase64Impl,

		Metadata: &BuiltinMetadata{
			Description: "Decode a base64 string to bytes",
			LongDesc:    "Decodes a base64 encoded string back to bytes. Returns None if the input is not valid base64. Uses standard base64 encoding (RFC 4648).",
			Params: []ParamDoc{
				{Name: "s", Description: "The base64 string to decode"},
			},
			Returns: "Option[bytes]: Some(decoded) if valid, None if invalid base64",
			Examples: []Example{
				{Code: `_bytes_from_base64("aGVsbG8=")`, Description: "Returns Some(bytes for 'hello')"},
				{Code: `_bytes_from_base64("invalid!!!")`, Description: "Returns None"},
			},
			SeeAlso:   []string{"_bytes_to_base64", "_bytes_to_string"},
			Since:     "v0.5.11",
			Stability: StabilityStable,
			Tags:      []string{"bytes", "base64", "decoding", "option"},
			Category:  "bytes",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _bytes_from_base64: %v", err))
	}
}

// makeBytesFromBase64Type builds the type signature for _bytes_from_base64
// Type: string -> Option[bytes]
func makeBytesFromBase64Type() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String()).Returns(
		T.App("Option", T.Bytes()),
	).Build()
}

// bytesFromBase64Impl is the implementation for _bytes_from_base64
func bytesFromBase64Impl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	strVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_bytes_from_base64: expected String, got %T", args[0])
	}

	decoded, err := base64.StdEncoding.DecodeString(strVal.Value)
	if err != nil {
		// Return None for invalid base64
		return &eval.TaggedValue{
			ModulePath: "std/option",
			TypeName:   "Option",
			CtorName:   "None",
			Fields:     []eval.Value{},
		}, nil
	}

	// Return Some(decoded)
	return &eval.TaggedValue{
		ModulePath: "std/option",
		TypeName:   "Option",
		CtorName:   "Some",
		Fields:     []eval.Value{&eval.BytesValue{Value: decoded}},
	}, nil
}

// registerBytesFromBase64URL registers _bytes_from_base64url: string -> Option[bytes]
// Decodes base64url (RFC 4648 §5) without padding — used by JWT tokens.
func registerBytesFromBase64URL() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/bytes",
		Name:    "_bytes_from_base64url",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeBytesFromBase64URLType,
		Impl:    bytesFromBase64URLImpl,

		Metadata: &BuiltinMetadata{
			Description: "Decode a base64url string (no padding) to bytes",
			LongDesc:    "Decodes a base64url encoded string (RFC 4648 §5, URL-safe alphabet, no padding) back to bytes. Returns None if the input is not valid base64url. JWT tokens use this encoding for header and payload segments.",
			Params: []ParamDoc{
				{Name: "s", Description: "The base64url string to decode (no padding)"},
			},
			Returns: "Option[bytes]: Some(decoded) if valid, None if invalid base64url",
			Examples: []Example{
				{Code: `_bytes_from_base64url("SGVsbG8")`, Description: "Returns Some(bytes for 'Hello')"},
				{Code: `_bytes_from_base64url("invalid!!!")`, Description: "Returns None"},
			},
			SeeAlso:   []string{"_bytes_from_base64", "_bytes_to_base64"},
			Since:     "v0.9.5",
			Stability: StabilityStable,
			Tags:      []string{"bytes", "base64url", "jwt", "decoding", "option"},
			Category:  "bytes",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _bytes_from_base64url: %v", err))
	}
}

func makeBytesFromBase64URLType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String()).Returns(
		T.App("Option", T.Bytes()),
	).Build()
}

func bytesFromBase64URLImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	strVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_bytes_from_base64url: expected String, got %T", args[0])
	}

	decoded, err := base64.RawURLEncoding.DecodeString(strVal.Value)
	if err != nil {
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
		Fields:     []eval.Value{&eval.BytesValue{Value: decoded}},
	}, nil
}

// registerBytesLength registers the _bytes_length builtin
func registerBytesLength() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/bytes",
		Name:    "_bytes_length",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeBytesLengthType,
		Impl:    bytesLengthImpl,

		Metadata: &BuiltinMetadata{
			Description: "Get the length of a byte slice",
			LongDesc:    "Returns the number of bytes in the slice. This is O(1) and always returns the raw byte count, not the number of UTF-8 characters.",
			Params: []ParamDoc{
				{Name: "b", Description: "The bytes to measure"},
			},
			Returns: "Number of bytes",
			Examples: []Example{
				{Code: `_bytes_length(bytes_from_string("hello"))`, Description: "Returns 5"},
				{Code: `_bytes_length(bytes_from_string("🎉"))`, Description: "Returns 4 (UTF-8 encoding is 4 bytes)"},
			},
			SeeAlso:   []string{"_str_len", "_bytes_from_string"},
			Since:     "v0.5.11",
			Stability: StabilityStable,
			Tags:      []string{"bytes", "length", "size"},
			Category:  "bytes",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _bytes_length: %v", err))
	}
}

// makeBytesLengthType builds the type signature for _bytes_length
// Type: bytes -> int
func makeBytesLengthType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.Bytes()).Returns(T.Int()).Build()
}

// bytesLengthImpl is the implementation for _bytes_length
func bytesLengthImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	bytesVal, ok := args[0].(*eval.BytesValue)
	if !ok {
		return nil, fmt.Errorf("_bytes_length: expected Bytes, got %T", args[0])
	}

	return &eval.IntValue{Value: len(bytesVal.Value)}, nil
}

// registerBytesSlice registers the _bytes_slice builtin
func registerBytesSlice() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/bytes",
		Name:    "_bytes_slice",
		NumArgs: 3,
		IsPure:  true,
		Effect:  "",
		Type:    makeBytesSliceType,
		Impl:    bytesSliceImpl,

		Metadata: &BuiltinMetadata{
			Description: "Extract a sub-slice of bytes",
			LongDesc:    "Returns a sub-slice starting at the given offset with the given length. Returns None if the range is out of bounds (start < 0, length < 0, or start + length > total). This is a pure operation — no mutation.",
			Params: []ParamDoc{
				{Name: "b", Description: "The source byte slice"},
				{Name: "start", Description: "Starting byte offset (0-based)"},
				{Name: "len", Description: "Number of bytes to extract"},
			},
			Returns: "Option[bytes]: Some(sub-slice) if in bounds, None otherwise",
			Examples: []Example{
				{Code: `_bytes_slice(bytes_from_string("hello"), 1, 3)`, Description: "Returns Some(bytes for \"ell\")"},
				{Code: `_bytes_slice(bytes_from_string("hello"), 5, 1)`, Description: "Returns None (out of bounds)"},
				{Code: `_bytes_slice(bytes_from_string(""), 0, 0)`, Description: "Returns Some(empty bytes)"},
			},
			SeeAlso:   []string{"_bytes_length", "_bytes_from_string"},
			Since:     "v0.8.1",
			Stability: StabilityStable,
			Tags:      []string{"bytes", "slice", "substring", "chunk"},
			Category:  "bytes",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _bytes_slice: %v", err))
	}
}

// makeBytesSliceType builds the type signature for _bytes_slice
// Type: (bytes, int, int) -> Option[bytes]
func makeBytesSliceType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.Bytes(), T.Int(), T.Int()).Returns(
		T.App("Option", T.Bytes()),
	).Build()
}

// bytesSliceImpl is the implementation for _bytes_slice
func bytesSliceImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	bytesVal, ok := args[0].(*eval.BytesValue)
	if !ok {
		return nil, fmt.Errorf("_bytes_slice: expected Bytes, got %T", args[0])
	}
	startVal, ok := args[1].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("_bytes_slice: expected Int for start, got %T", args[1])
	}
	lenVal, ok := args[2].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("_bytes_slice: expected Int for len, got %T", args[2])
	}

	start := startVal.Value
	length := lenVal.Value
	total := len(bytesVal.Value)

	// Bounds check — return None for invalid ranges
	if start < 0 || length < 0 || start+length > total {
		return &eval.TaggedValue{
			ModulePath: "std/option",
			TypeName:   "Option",
			CtorName:   "None",
			Fields:     []eval.Value{},
		}, nil
	}

	result := make([]byte, length)
	copy(result, bytesVal.Value[start:start+length])

	return &eval.TaggedValue{
		ModulePath: "std/option",
		TypeName:   "Option",
		CtorName:   "Some",
		Fields:     []eval.Value{&eval.BytesValue{Value: result}},
	}, nil
}

// ============================================================================
// Bytes Concatenation & Construction
// ============================================================================

// registerBytesConcat registers the _bytes_concat builtin
func registerBytesConcat() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/bytes",
		Name:    "_bytes_concat",
		NumArgs: 2,
		IsPure:  true,
		Effect:  "",
		Type: func() types.Type {
			T := types.NewBuilder()
			return T.Func(T.Bytes(), T.Bytes()).Returns(T.Bytes()).Build()
		},
		Impl: func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
			a, ok := args[0].(*eval.BytesValue)
			if !ok {
				return nil, fmt.Errorf("_bytes_concat: expected Bytes for first argument, got %T", args[0])
			}
			b, ok := args[1].(*eval.BytesValue)
			if !ok {
				return nil, fmt.Errorf("_bytes_concat: expected Bytes for second argument, got %T", args[1])
			}
			result := make([]byte, len(a.Value)+len(b.Value))
			copy(result, a.Value)
			copy(result[len(a.Value):], b.Value)
			return &eval.BytesValue{Value: result}, nil
		},

		Metadata: &BuiltinMetadata{
			Description: "Concatenate two byte slices",
			LongDesc:    "Returns a new byte slice containing a followed by b. Pure operation — inputs are not modified.",
			Params: []ParamDoc{
				{Name: "a", Description: "First byte slice"},
				{Name: "b", Description: "Second byte slice"},
			},
			Returns: "Concatenated bytes",
			Examples: []Example{
				{Code: `_bytes_concat(fromString("hello"), fromString(" world"))`, Description: "Returns bytes for \"hello world\""},
			},
			SeeAlso:   []string{"_bytes_concat_list", "_bytes_from_string"},
			Since:     "v0.8.2",
			Stability: StabilityStable,
			Tags:      []string{"bytes", "concat", "combine"},
			Category:  "bytes",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _bytes_concat: %v", err))
	}
}

// registerBytesConcatList registers the _bytes_concat_list builtin
func registerBytesConcatList() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/bytes",
		Name:    "_bytes_concat_list",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type: func() types.Type {
			T := types.NewBuilder()
			return T.Func(T.List(T.Bytes())).Returns(T.Bytes()).Build()
		},
		Impl: func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
			listVal, ok := args[0].(*eval.ListValue)
			if !ok {
				return nil, fmt.Errorf("_bytes_concat_list: expected List, got %T", args[0])
			}
			// Calculate total length first for single allocation
			total := 0
			for _, elem := range listVal.Elements {
				bv, ok := elem.(*eval.BytesValue)
				if !ok {
					return nil, fmt.Errorf("_bytes_concat_list: expected Bytes element, got %T", elem)
				}
				total += len(bv.Value)
			}
			result := make([]byte, 0, total)
			for _, elem := range listVal.Elements {
				result = append(result, elem.(*eval.BytesValue).Value...)
			}
			return &eval.BytesValue{Value: result}, nil
		},

		Metadata: &BuiltinMetadata{
			Description: "Concatenate a list of byte slices",
			LongDesc:    "Concatenates all byte slices in the list into a single byte slice. Single allocation for efficiency.",
			Params: []ParamDoc{
				{Name: "xs", Description: "List of byte slices to concatenate"},
			},
			Returns: "Concatenated bytes",
			Examples: []Example{
				{Code: `_bytes_concat_list([header, body, footer])`, Description: "Combines header + body + footer into one byte slice"},
			},
			SeeAlso:   []string{"_bytes_concat", "_bytes_from_string"},
			Since:     "v0.8.2",
			Stability: StabilityStable,
			Tags:      []string{"bytes", "concat", "list", "combine"},
			Category:  "bytes",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _bytes_concat_list: %v", err))
	}
}

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
		Impl: func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
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
		},

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

// registerBytesFilename registers the _bytes_filename builtin
func registerBytesFilename() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/bytes",
		Name:    "_bytes_filename",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type: func() types.Type {
			T := types.NewBuilder()
			return T.Func(T.Bytes()).Returns(T.String()).Build()
		},
		Impl: func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
			bytesVal, ok := args[0].(*eval.BytesValue)
			if !ok {
				return nil, fmt.Errorf("_bytes_filename: expected Bytes, got %T", args[0])
			}
			return &eval.StringValue{Value: bytesVal.Filename}, nil
		},
		Metadata: &BuiltinMetadata{
			Description: "Get the original filename of uploaded bytes",
			LongDesc:    "Returns the original filename from a file upload. Returns empty string if the bytes were not from an upload.",
			Params:      []ParamDoc{{Name: "b", Description: "Bytes value (typically from file upload)"}},
			Returns:     "Original filename or empty string",
			Since:       "v0.9.4",
			Stability:   StabilityStable,
			Tags:        []string{"bytes", "upload", "filename"},
			Category:    "bytes",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _bytes_filename: %v", err))
	}
}

// registerBytesMimeType registers the _bytes_mime_type builtin
func registerBytesMimeType() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/bytes",
		Name:    "_bytes_mime_type",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type: func() types.Type {
			T := types.NewBuilder()
			return T.Func(T.Bytes()).Returns(T.String()).Build()
		},
		Impl: func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
			bytesVal, ok := args[0].(*eval.BytesValue)
			if !ok {
				return nil, fmt.Errorf("_bytes_mime_type: expected Bytes, got %T", args[0])
			}
			return &eval.StringValue{Value: bytesVal.MimeType}, nil
		},
		Metadata: &BuiltinMetadata{
			Description: "Get the MIME type of uploaded bytes",
			LongDesc:    "Returns the MIME type from a file upload. Returns empty string if unknown or not from an upload.",
			Params:      []ParamDoc{{Name: "b", Description: "Bytes value (typically from file upload)"}},
			Returns:     "MIME type string or empty string",
			Since:       "v0.9.4",
			Stability:   StabilityStable,
			Tags:        []string{"bytes", "upload", "mime", "content-type"},
			Category:    "bytes",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _bytes_mime_type: %v", err))
	}
}
