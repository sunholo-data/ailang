package messaging

import "testing"

// The Firestore backend carried its own `simhashText` that XOR-folded runes into
// eight shift positions. It compiled, ran, and produced numbers — they just were
// not comparable with the stored hashes, so every cloud-side search scored noise.
// Pin the property that actually matters: related text must score higher than
// unrelated text, which the old fold did not guarantee.
func TestComputeSimhash_RelatedTextScoresAboveUnrelated(t *testing.T) {
	query := ComputeSimhash("otel exporter ships tokens to the collector", "")
	related := ComputeSimhash(
		"SECURITY: -emit-trace otel ships access AND refresh tokens to the collector",
		"Stood up a local OTLP collector and the exporter received the refresh token.")
	unrelated := ComputeSimhash(
		"Release v0.36.0",
		"Version bump, changelog entries and dashboard refresh.")

	relScore := simhashSimilarityForTest(query, related)
	unrelScore := simhashSimilarityForTest(query, unrelated)

	if relScore <= unrelScore {
		t.Fatalf("related text must outscore unrelated: related=%.3f unrelated=%.3f", relScore, unrelScore)
	}
	if relScore < 0.60 {
		t.Errorf("related text scored %.3f, below any usable search threshold", relScore)
	}
}

// A message with an empty payload must still hash its title rather than
// degenerating to zero — otherwise every title-only message collides.
func TestComputeSimhash_TitleOnlyIsNotZero(t *testing.T) {
	if got := ComputeSimhash("Daneel watchdog: ALERT", ""); got == 0 {
		t.Fatal("title-only message hashed to 0; every such message would collide")
	}
}

func TestSearchText_JoinsTitleAndPayload(t *testing.T) {
	if got := SearchText("title", "payload"); got != "title payload" {
		t.Errorf("SearchText = %q, want %q", got, "title payload")
	}
	if got := SearchText("title", ""); got != "title" {
		t.Errorf("SearchText with empty payload = %q, want %q", got, "title")
	}
}

func simhashSimilarityForTest(a, b int64) float64 {
	xor := uint64(a ^ b)
	var distance int
	for i := 0; i < 64; i++ {
		if (xor>>uint(i))&1 == 1 {
			distance++
		}
	}
	return 1.0 - float64(distance)/64.0
}
