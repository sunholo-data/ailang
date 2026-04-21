package builtins

import (
	"fmt"
	"hash/fnv"
	"strings"
	"unicode"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

// SimHash builtin functions for AILANG
// SimHash is a locality-sensitive hashing algorithm for near-duplicate detection.
// Similar documents produce similar hashes with low Hamming distance.
// Part of M-DX15 (Semantic Caching MVP)

func init() {
	registerSimHash()
	registerHammingDistance()
}

// ============================================================================
// SimHash Algorithm Implementation
// ============================================================================

// SimHash computes a 64-bit locality-sensitive hash of a string.
// The algorithm:
// 1. Tokenize the input into words (or n-grams)
// 2. Hash each token to a 64-bit value using FNV-1a
// 3. For each bit position, sum +1 if bit is 1, -1 if bit is 0
// 4. Final hash has bit=1 if sum>0, bit=0 otherwise
//
// This produces a fingerprint where similar documents have similar hashes.
func SimHash(text string) int64 {
	// Tokenize: split into lowercase words, remove punctuation
	tokens := tokenize(text)
	if len(tokens) == 0 {
		return 0
	}

	// Vector to accumulate bit weights
	// Each position tracks sum of +1/-1 for that bit across all tokens
	var vector [64]int

	for _, token := range tokens {
		// Hash the token using FNV-1a (fast, good distribution)
		h := fnv.New64a()
		h.Write([]byte(token))
		hash := h.Sum64()

		// Update vector: +1 for bit=1, -1 for bit=0
		for i := 0; i < 64; i++ {
			bit := (hash >> i) & 1
			if bit == 1 {
				vector[i]++
			} else {
				vector[i]--
			}
		}
	}

	// Build final hash: bit=1 if sum>0, bit=0 otherwise
	var result uint64
	for i := 0; i < 64; i++ {
		if vector[i] > 0 {
			result |= (1 << i)
		}
	}

	return int64(result)
}

// tokenize splits text into normalized tokens for hashing.
// Returns lowercase words with punctuation removed.
func tokenize(text string) []string {
	// Convert to lowercase
	text = strings.ToLower(text)

	// Split into words, removing punctuation
	var tokens []string
	var current strings.Builder

	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			current.WriteRune(r)
		} else {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		}
	}

	// Don't forget the last word
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

// HammingDistance computes the number of differing bits between two 64-bit integers.
// Lower distance = more similar documents.
// Typical thresholds:
//   - 0-3: Very similar (likely near-duplicates)
//   - 4-10: Somewhat similar
//   - 10+: Different documents
func HammingDistance(a, b int64) int {
	// XOR gives us bits that differ
	diff := uint64(a) ^ uint64(b)

	// Count the number of set bits (popcount)
	count := 0
	for diff != 0 {
		count++
		diff &= diff - 1 // Clear lowest set bit
	}

	return count
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

	hash := SimHash(strVal.Value)
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

	dist := HammingDistance(int64(aVal.Value), int64(bVal.Value))
	return &eval.IntValue{Value: dist}, nil
}
