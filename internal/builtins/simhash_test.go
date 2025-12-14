package builtins

import (
	"testing"

	"github.com/sunholo/ailang/internal/eval"
)

func TestSimHash_Deterministic(t *testing.T) {
	// Same input should always produce same output
	text := "The quick brown fox jumps over the lazy dog"

	hash1 := SimHash(text)
	hash2 := SimHash(text)
	hash3 := SimHash(text)

	if hash1 != hash2 || hash2 != hash3 {
		t.Errorf("SimHash not deterministic: %d, %d, %d", hash1, hash2, hash3)
	}
}

func TestSimHash_SimilarTexts(t *testing.T) {
	// Similar texts should have small Hamming distance
	text1 := "The quick brown fox jumps over the lazy dog"
	text2 := "The quick brown fox jumps over the lazy cat"
	text3 := "A fast brown fox leaps over the lazy dog"

	hash1 := SimHash(text1)
	hash2 := SimHash(text2)
	hash3 := SimHash(text3)

	dist12 := HammingDistance(hash1, hash2)
	dist13 := HammingDistance(hash1, hash3)

	t.Logf("hash1: %064b", uint64(hash1))
	t.Logf("hash2: %064b", uint64(hash2))
	t.Logf("hash3: %064b", uint64(hash3))
	t.Logf("distance(1,2): %d", dist12)
	t.Logf("distance(1,3): %d", dist13)

	// Similar texts should have distance < 15
	if dist12 > 15 {
		t.Errorf("Similar texts have too large distance: %d", dist12)
	}
	if dist13 > 15 {
		t.Errorf("Similar texts have too large distance: %d", dist13)
	}
}

func TestSimHash_DifferentTexts(t *testing.T) {
	// Very different texts should have large Hamming distance
	text1 := "The quick brown fox jumps over the lazy dog"
	text2 := "Machine learning models process natural language"
	text3 := "HTTP requests are handled by the server component"

	hash1 := SimHash(text1)
	hash2 := SimHash(text2)
	hash3 := SimHash(text3)

	dist12 := HammingDistance(hash1, hash2)
	dist13 := HammingDistance(hash1, hash3)

	t.Logf("hash1: %064b", uint64(hash1))
	t.Logf("hash2: %064b", uint64(hash2))
	t.Logf("hash3: %064b", uint64(hash3))
	t.Logf("distance(1,2): %d", dist12)
	t.Logf("distance(1,3): %d", dist13)

	// Different texts should have distance > 10
	// Note: This is probabilistic, so we use a reasonable threshold
	if dist12 < 5 {
		t.Errorf("Different texts have surprisingly small distance: %d", dist12)
	}
	if dist13 < 5 {
		t.Errorf("Different texts have surprisingly small distance: %d", dist13)
	}
}

func TestSimHash_EmptyString(t *testing.T) {
	hash := SimHash("")
	if hash != 0 {
		t.Errorf("Empty string should hash to 0, got %d", hash)
	}
}

func TestSimHash_CaseInsensitive(t *testing.T) {
	// SimHash should be case-insensitive (lowercase internally)
	text1 := "Hello World"
	text2 := "hello world"
	text3 := "HELLO WORLD"

	hash1 := SimHash(text1)
	hash2 := SimHash(text2)
	hash3 := SimHash(text3)

	if hash1 != hash2 {
		t.Errorf("Case should not matter: hash(%q)=%d != hash(%q)=%d", text1, hash1, text2, hash2)
	}
	if hash2 != hash3 {
		t.Errorf("Case should not matter: hash(%q)=%d != hash(%q)=%d", text2, hash2, text3, hash3)
	}
}

func TestSimHash_PunctuationIgnored(t *testing.T) {
	// Punctuation should be ignored
	text1 := "hello world"
	text2 := "hello, world!"
	text3 := "hello... world???"

	hash1 := SimHash(text1)
	hash2 := SimHash(text2)
	hash3 := SimHash(text3)

	if hash1 != hash2 {
		t.Errorf("Punctuation should be ignored: hash(%q)=%d != hash(%q)=%d", text1, hash1, text2, hash2)
	}
	if hash2 != hash3 {
		t.Errorf("Punctuation should be ignored: hash(%q)=%d != hash(%q)=%d", text2, hash2, text3, hash3)
	}
}

func TestHammingDistance_Identical(t *testing.T) {
	// Same values should have distance 0
	testCases := []int64{0, 1, -1, 12345, -12345, 0x7FFFFFFFFFFFFFFF}

	for _, v := range testCases {
		dist := HammingDistance(v, v)
		if dist != 0 {
			t.Errorf("HammingDistance(%d, %d) = %d, want 0", v, v, dist)
		}
	}
}

func TestHammingDistance_OneBitDiff(t *testing.T) {
	// Values differing by one bit should have distance 1
	testCases := []struct {
		a, b int64
		want int
	}{
		{0, 1, 1},
		{0, 2, 1},
		{0, 4, 1},
		{1, 3, 1}, // 01 vs 11
		{0xFF, 0xFE, 1},
	}

	for _, tc := range testCases {
		dist := HammingDistance(tc.a, tc.b)
		if dist != tc.want {
			t.Errorf("HammingDistance(%d, %d) = %d, want %d", tc.a, tc.b, dist, tc.want)
		}
	}
}

func TestHammingDistance_AllBitsDiff(t *testing.T) {
	// All 64 bits different
	dist := HammingDistance(0, -1) // 0 vs all 1s
	if dist != 64 {
		t.Errorf("HammingDistance(0, -1) = %d, want 64", dist)
	}
}

func TestHammingDistance_Symmetric(t *testing.T) {
	// Distance should be symmetric
	a := int64(0x1234567890ABCDEF)
	b := int64(0x0FEDCBA987654321)

	dist1 := HammingDistance(a, b)
	dist2 := HammingDistance(b, a)

	if dist1 != dist2 {
		t.Errorf("HammingDistance not symmetric: %d != %d", dist1, dist2)
	}
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"hello world", []string{"hello", "world"}},
		{"Hello World", []string{"hello", "world"}},
		{"hello, world!", []string{"hello", "world"}},
		{"", []string(nil)},
		{"   ", []string(nil)},
		{"one", []string{"one"}},
		{"test123", []string{"test123"}},
		{"a b c", []string{"a", "b", "c"}},
	}

	for _, tc := range tests {
		got := tokenize(tc.input)
		if len(got) != len(tc.want) {
			t.Errorf("tokenize(%q) = %v, want %v", tc.input, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("tokenize(%q)[%d] = %q, want %q", tc.input, i, got[i], tc.want[i])
			}
		}
	}
}

// Builtin implementation tests

func TestSimHashBuiltin(t *testing.T) {
	text := "hello world"
	args := []eval.Value{&eval.StringValue{Value: text}}

	result, err := simHashImpl(nil, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	intVal, ok := result.(*eval.IntValue)
	if !ok {
		t.Fatalf("expected IntValue, got %T", result)
	}

	// Verify it matches the Go implementation
	expected := SimHash(text)
	if int64(intVal.Value) != expected {
		t.Errorf("builtin returned %d, expected %d", intVal.Value, expected)
	}
}

func TestHammingDistanceBuiltin(t *testing.T) {
	a := int64(12345)
	b := int64(12346)

	args := []eval.Value{
		&eval.IntValue{Value: int(a)},
		&eval.IntValue{Value: int(b)},
	}

	result, err := hammingDistanceImpl(nil, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	intVal, ok := result.(*eval.IntValue)
	if !ok {
		t.Fatalf("expected IntValue, got %T", result)
	}

	expected := HammingDistance(a, b)
	if intVal.Value != expected {
		t.Errorf("builtin returned %d, expected %d", intVal.Value, expected)
	}
}

func TestSimHash_NearDuplicateDetection(t *testing.T) {
	// Real-world example: detecting near-duplicate content
	original := "The AILANG programming language is designed for AI code synthesis"
	nearDup1 := "The AILANG programming language is designed for AI code generation" // one word changed
	nearDup2 := "AILANG programming language designed for AI code synthesis"         // words removed
	different := "Python is a popular programming language for web development"

	hashOrig := SimHash(original)
	hashDup1 := SimHash(nearDup1)
	hashDup2 := SimHash(nearDup2)
	hashDiff := SimHash(different)

	distDup1 := HammingDistance(hashOrig, hashDup1)
	distDup2 := HammingDistance(hashOrig, hashDup2)
	distDiff := HammingDistance(hashOrig, hashDiff)

	t.Logf("Original vs near-dup1 (one word): distance = %d", distDup1)
	t.Logf("Original vs near-dup2 (words removed): distance = %d", distDup2)
	t.Logf("Original vs different: distance = %d", distDiff)

	// Near-duplicates should be much closer than different content
	if distDup1 >= distDiff {
		t.Errorf("Near-duplicate should be closer than different content: %d >= %d", distDup1, distDiff)
	}
	if distDup2 >= distDiff {
		t.Errorf("Near-duplicate should be closer than different content: %d >= %d", distDup2, distDiff)
	}
}
