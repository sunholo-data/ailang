// Package simhash is the ONE locality-sensitive fingerprint and the ONE pair
// of similarity measures (Hamming over fingerprints, cosine over embedding
// vectors) for everything in AILANG that asks "is this near that".
//
// Three SimHash algorithms used to exist — FNV-1a over Unicode word tokens
// (builtins, the `_simhash` builtin), MD5 over whitespace tokens (docsearch)
// and FNV-1a over ASCII-only tokens dropping 1-char words (coordinator) —
// with four Hamming distances and three cosines beside them. The survivor is
// the builtins algorithm, unchanged bit for bit, because it is the one whose
// hashes are PERSISTED: `inbox_messages.simhash` (messaging SQLite), the
// `simhash` field on Firestore inbox documents, `brain_frames.simhash`
// (sharedmem SQLite) and every value the `_simhash` builtin ever returned
// to a program. TestHashIsBitIdenticalToPersistedValues pins fixed vectors
// captured from the pre-move implementation so a store written before this
// package existed still searches. M-V1-SIMPLIFY-S3 M5.
//
// It is a LEAF: stdlib only. Language-core packages (builtins, effects) and
// platform packages (messaging, coordinator, docsearch) both call it.
package simhash

import (
	"hash/fnv"
	"math"
	"math/bits"
	"strings"
	"unicode"
)

// Bits is the fingerprint width; Similarity divides by it.
const Bits = 64

// Hash computes the 64-bit SimHash of text.
//
//  1. Lower-case; split into runs of Unicode letters/digits (everything else
//     is a separator).
//  2. FNV-1a 64 each token.
//  3. For each bit position, +1 when the token's bit is set, -1 when clear.
//  4. The fingerprint bit is set where the sum is positive.
//
// Empty input (no tokens) is 0. The result is int64 because that is what
// every store column holds; compare with HammingDistance, never numerically.
func Hash(text string) int64 {
	tokens := tokenize(text)
	if len(tokens) == 0 {
		return 0
	}
	var vector [Bits]int
	for _, token := range tokens {
		h := fnv.New64a()
		_, _ = h.Write([]byte(token))
		sum := h.Sum64()
		for i := 0; i < Bits; i++ {
			if (sum>>i)&1 == 1 {
				vector[i]++
			} else {
				vector[i]--
			}
		}
	}
	var result uint64
	for i := 0; i < Bits; i++ {
		if vector[i] > 0 {
			result |= 1 << i
		}
	}
	return int64(result)
}

// tokenize splits text into lower-cased runs of letters and digits.
func tokenize(text string) []string {
	text = strings.ToLower(text)
	var tokens []string
	var current strings.Builder
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			current.WriteRune(r)
		} else if current.Len() > 0 {
			tokens = append(tokens, current.String())
			current.Reset()
		}
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}
	return tokens
}

// HammingDistance is the number of bit positions where a and b differ
// (0 = identical fingerprints, 64 = every bit differs). Typical reading:
// 0-3 near-duplicate, 4-10 related, more than 10 different.
func HammingDistance(a, b int64) int {
	return bits.OnesCount64(uint64(a) ^ uint64(b))
}

// Similarity maps HammingDistance onto [0, 1]: 1 - distance/64. Every
// search threshold in the codebase (messaging 0.70, coordinator dedup 0.80,
// brain 3-tier) is expressed against this scale.
func Similarity(a, b int64) float64 {
	return 1.0 - float64(HammingDistance(a, b))/float64(Bits)
}

// Cosine is the cosine similarity of two embedding vectors in [-1, 1].
// Degenerate inputs — empty, different lengths, or a zero vector — score 0:
// there is no direction to compare, and every caller treats 0 as "no match".
func Cosine[T ~float32 | ~float64](a, b []T) float64 {
	c, _ := cosine(a, b)
	return c
}

// CosineUnit is Cosine rescaled onto [0, 1] ((c+1)/2) so embedding scores
// compare with Similarity's scale. Degenerate inputs still score 0, not 0.5.
func CosineUnit[T ~float32 | ~float64](a, b []T) float64 {
	c, ok := cosine(a, b)
	if !ok {
		return 0
	}
	return (c + 1) / 2
}

func cosine[T ~float32 | ~float64](a, b []T) (float64, bool) {
	if len(a) == 0 || len(a) != len(b) {
		return 0, false
	}
	var dot, normA, normB float64
	for i := range a {
		ai, bi := float64(a[i]), float64(b[i])
		dot += ai * bi
		normA += ai * ai
		normB += bi * bi
	}
	if normA == 0 || normB == 0 {
		return 0, false
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB)), true
}
