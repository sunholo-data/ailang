package builtins

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/simhash"
	"github.com/sunholo-data/ailang/internal/types"
)

// SimHash builtin functions for AILANG (M-DX15 Semantic Caching MVP).
// The algorithm lives in internal/simhash — the one fingerprint every store
// persists; this file only registers `_simhash` and `_hamming_distance`.

func init() {
	registerSimHash()
	registerHammingDistance()
}

// ============================================================================
// Builtin Registration
// ============================================================================

// registerSimHash registers the _simhash builtin
func registerSimHash() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/simhash",
		Name:    "_simhash",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "", // Pure function
		Type:    makeSimHashType,
		Impl:    simHashImpl,

		Metadata: &BuiltinMetadata{
			Description: "Compute a 64-bit locality-sensitive hash of a string",
			LongDesc: `SimHash produces a fingerprint where similar documents have similar hashes.
This is useful for near-duplicate detection without expensive embedding models.

The algorithm tokenizes the input, hashes each token, and combines them into
a single 64-bit value. Documents with similar content will have hashes with
low Hamming distance (few differing bits).

Typical thresholds:
- 0-3 bits different: Very similar (likely near-duplicates)
- 4-10 bits different: Somewhat similar
- 10+ bits different: Different documents`,
			Params: []ParamDoc{
				{Name: "text", Description: "The text to hash"},
			},
			Returns: "64-bit integer hash (simhash64 type alias)",
			Examples: []Example{
				{Code: `_simhash("hello world")`, Description: "Returns consistent 64-bit hash"},
				{Code: `_simhash("hello world!") -- similar hash`, Description: "Small Hamming distance from above"},
				{Code: `_simhash("goodbye mars") -- different hash`, Description: "Large Hamming distance from above"},
			},
			SeeAlso:   []string{"_hamming_distance", "_bytes_from_string"},
			Since:     "v0.5.11",
			Stability: StabilityStable,
			Tags:      []string{"hash", "simhash", "similarity", "fingerprint", "lsh"},
			Category:  "simhash",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _simhash: %v", err))
	}
}

// makeSimHashType builds the type signature for _simhash
// Type: string -> int (simhash64 is type alias for int)
func makeSimHashType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String()).Returns(T.Int()).Build()
}

// simHashImpl is the implementation for _simhash
func simHashImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	strVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_simhash: expected String, got %T", args[0])
	}

	hash := simhash.Hash(strVal.Value)
	return &eval.IntValue{Value: int(hash)}, nil
}

// registerHammingDistance registers the _hamming_distance builtin
func registerHammingDistance() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/simhash",
		Name:    "_hamming_distance",
		NumArgs: 2,
		IsPure:  true,
		Effect:  "",
		Type:    makeHammingDistanceType,
		Impl:    hammingDistanceImpl,

		Metadata: &BuiltinMetadata{
			Description: "Compute the Hamming distance between two 64-bit hashes",
			LongDesc: `Returns the number of differing bits between two SimHash values.
Lower distance means more similar documents.

Typical interpretation:
- 0-3: Very similar (likely near-duplicates)
- 4-10: Somewhat similar
- 10+: Different documents

This is much faster than computing cosine similarity on embeddings
and works well for detecting near-duplicate text.`,
			Params: []ParamDoc{
				{Name: "a", Description: "First SimHash value"},
				{Name: "b", Description: "Second SimHash value"},
			},
			Returns: "Number of differing bits (0-64)",
			Examples: []Example{
				{Code: `_hamming_distance(h1, h1)`, Description: "Returns 0 (identical)"},
				{Code: `_hamming_distance(simhash("hello"), simhash("helo"))`, Description: "Returns small number"},
			},
			SeeAlso:   []string{"_simhash"},
			Since:     "v0.5.11",
			Stability: StabilityStable,
			Tags:      []string{"hash", "simhash", "distance", "similarity"},
			Category:  "simhash",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _hamming_distance: %v", err))
	}
}

// makeHammingDistanceType builds the type signature for _hamming_distance
// Type: int -> int -> int
func makeHammingDistanceType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.Int(), T.Int()).Returns(T.Int()).Build()
}

// hammingDistanceImpl is the implementation for _hamming_distance
func hammingDistanceImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	aVal, ok := args[0].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("_hamming_distance: first argument must be int, got %T", args[0])
	}

	bVal, ok := args[1].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("_hamming_distance: second argument must be int, got %T", args[1])
	}

	dist := simhash.HammingDistance(int64(aVal.Value), int64(bVal.Value))
	return &eval.IntValue{Value: dist}, nil
}
