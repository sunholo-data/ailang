package builtins

import (
	"encoding/base64"
	"fmt"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/types"
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
