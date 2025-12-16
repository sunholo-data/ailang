package builtins

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/types"
)

// Embedding builtin functions for AILANG
// These provide IEEE 754 float32 little-endian encoding/decoding for neural embeddings
// Part of M-DX16 (SharedIndex - Deterministic Semantic Retrieval)

func init() {
	registerEmbeddingEncode()
	registerEmbeddingDecode()
}

// ============================================================================
// Embedding Primitive Builtins
// ============================================================================

// registerEmbeddingEncode registers the _embedding_encode builtin
func registerEmbeddingEncode() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/sem",
		Name:    "_embedding_encode",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "", // Pure function
		Type:    makeEmbeddingEncodeType,
		Impl:    embeddingEncodeImpl,

		Metadata: &BuiltinMetadata{
			Description: "Encode a list of floats as packed IEEE 754 float32 little-endian bytes",
			LongDesc:    "Takes a list of float values and packs them into bytes using IEEE 754 float32 little-endian encoding. Each float becomes 4 bytes. Standard embedding dimensions: 384 (MiniLM), 768 (BERT), 1536 (OpenAI).",
			Params: []ParamDoc{
				{Name: "floats", Description: "List of float values to encode"},
			},
			Returns: "Packed bytes (4 * N bytes for N floats)",
			Examples: []Example{
				{Code: `_embedding_encode([1.0, 2.0, 3.0])`, Description: "Returns 12 bytes (3 floats * 4 bytes each)"},
			},
			SeeAlso:   []string{"_embedding_decode"},
			Since:     "v0.5.11",
			Stability: StabilityStable,
			Tags:      []string{"embedding", "bytes", "encoding", "float32"},
			Category:  "semantic",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _embedding_encode: %v", err))
	}
}

// makeEmbeddingEncodeType builds the type signature for _embedding_encode
// Type: list[float] -> bytes
func makeEmbeddingEncodeType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.List(T.Float())).Returns(T.Bytes()).Build()
}

// embeddingEncodeImpl is the implementation for _embedding_encode
func embeddingEncodeImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	listVal, ok := args[0].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("_embedding_encode: expected List, got %T", args[0])
	}

	// Allocate buffer for packed floats
	buf := make([]byte, len(listVal.Elements)*4)

	for i, elem := range listVal.Elements {
		floatVal, ok := elem.(*eval.FloatValue)
		if !ok {
			return nil, fmt.Errorf("_embedding_encode: element %d is not Float, got %T", i, elem)
		}

		// Convert float64 to float32 and pack as little-endian
		bits := math.Float32bits(float32(floatVal.Value))
		binary.LittleEndian.PutUint32(buf[i*4:], bits)
	}

	return &eval.BytesValue{Value: buf}, nil
}

// registerEmbeddingDecode registers the _embedding_decode builtin
func registerEmbeddingDecode() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/sem",
		Name:    "_embedding_decode",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "", // Pure function
		Type:    makeEmbeddingDecodeType,
		Impl:    embeddingDecodeImpl,

		Metadata: &BuiltinMetadata{
			Description: "Decode packed IEEE 754 float32 little-endian bytes to a list of floats",
			LongDesc:    "Takes packed bytes and unpacks them into a list of float values using IEEE 754 float32 little-endian decoding. Returns None if bytes length is not divisible by 4.",
			Params: []ParamDoc{
				{Name: "bytes", Description: "Packed bytes to decode (must be 4*N bytes)"},
			},
			Returns: "Option[list[float]] - Some(floats) if valid, None if invalid length",
			Examples: []Example{
				{Code: `_embedding_decode(_embedding_encode([1.0, 2.0]))`, Description: "Returns Some([1.0, 2.0])"},
				{Code: `_embedding_decode(_bytes_from_string("abc"))`, Description: "Returns None (3 bytes is not divisible by 4)"},
			},
			SeeAlso:   []string{"_embedding_encode"},
			Since:     "v0.5.11",
			Stability: StabilityStable,
			Tags:      []string{"embedding", "bytes", "decoding", "float32"},
			Category:  "semantic",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _embedding_decode: %v", err))
	}
}

// makeEmbeddingDecodeType builds the type signature for _embedding_decode
// Type: bytes -> option[list[float]]
func makeEmbeddingDecodeType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.Bytes()).Returns(
		T.App("Option", T.List(T.Float())),
	).Build()
}

// embeddingDecodeImpl is the implementation for _embedding_decode
func embeddingDecodeImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	bytesVal, ok := args[0].(*eval.BytesValue)
	if !ok {
		return nil, fmt.Errorf("_embedding_decode: expected Bytes, got %T", args[0])
	}

	// Check if length is divisible by 4
	if len(bytesVal.Value)%4 != 0 {
		// Return None for invalid length
		return &eval.TaggedValue{
			ModulePath: "std/option",
			TypeName:   "Option",
			CtorName:   "None",
			Fields:     []eval.Value{},
		}, nil
	}

	// Unpack floats
	numFloats := len(bytesVal.Value) / 4
	floats := make([]eval.Value, numFloats)

	for i := 0; i < numFloats; i++ {
		bits := binary.LittleEndian.Uint32(bytesVal.Value[i*4:])
		floatVal := float64(math.Float32frombits(bits))
		floats[i] = &eval.FloatValue{Value: floatVal}
	}

	// Return Some(list)
	return &eval.TaggedValue{
		ModulePath: "std/option",
		TypeName:   "Option",
		CtorName:   "Some",
		Fields:     []eval.Value{&eval.ListValue{Elements: floats}},
	}, nil
}
