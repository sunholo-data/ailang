package simhash_test

import (
	"math"
	"os/exec"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/simhash"
)

// The values below were printed by builtins.SimHash at 79da32b60, before the
// algorithm moved here. They are what `inbox_messages.simhash`,
// `brain_frames.simhash` and the Firestore `simhash` field hold today. If
// this test fails, the surviving algorithm no longer matches the persisted
// hash space and every semantic search silently returns nothing — the exact
// failure messaging/simhash.go recorded for the Firestore backend. Do not
// update the numbers; bump the stores' schema versions and re-index instead.
func TestHashIsBitIdenticalToPersistedValues(t *testing.T) {
	golden := map[string]int64{
		"":              0,
		"hello":         -6615550055289275125,
		"Hello, World!": 292971770938951683,
		"the quick brown fox jumps over the lazy dog":                             -3839431810354909714,
		"ailang messages send --to ailang-core: parser panics on empty match arm": 725338547812876438,
		"Zürich café — naïve résumé 日本語 テスト 123":                                  6822815029212014217,
		"a b c d e f g": -5808559072177164832,
		"I have a 1 x":  -5808548077060907380,
	}
	for text, want := range golden {
		if got := simhash.Hash(text); got != want {
			t.Errorf("Hash(%q) = %d, persisted stores hold %d", text, got, want)
		}
	}
}

func TestHashIsLocalitySensitive(t *testing.T) {
	a := simhash.Hash("the coordinator daemon failed to dispatch the message to the inbox")
	b := simhash.Hash("the coordinator daemon failed to dispatch the message to the queue")
	c := simhash.Hash("z3 timed out verifying the contract on line forty two")
	if d := simhash.HammingDistance(a, b); d > 12 {
		t.Errorf("near-duplicate texts differ by %d bits", d)
	}
	if simhash.Similarity(a, b) <= simhash.Similarity(a, c) {
		t.Errorf("unrelated text scored at least as similar: %f vs %f", simhash.Similarity(a, c), simhash.Similarity(a, b))
	}
	if simhash.Hash("Punctuation, CASE and   spacing") != simhash.Hash("punctuation case and spacing") {
		t.Error("tokenizer must ignore case and punctuation")
	}
}

func TestHammingAndSimilarity(t *testing.T) {
	cases := []struct {
		a, b int64
		dist int
	}{
		{0, 0, 0},
		{0, 1, 1},
		{0, -1, 64},
		{0x0F, 0xF0, 8},
		{math.MinInt64, 0, 1},
	}
	for _, c := range cases {
		if got := simhash.HammingDistance(c.a, c.b); got != c.dist {
			t.Errorf("HammingDistance(%d, %d) = %d, want %d", c.a, c.b, got, c.dist)
		}
		want := 1 - float64(c.dist)/64
		if got := simhash.Similarity(c.a, c.b); math.Abs(got-want) > 1e-12 {
			t.Errorf("Similarity(%d, %d) = %f, want %f", c.a, c.b, got, want)
		}
	}
}

func TestCosine(t *testing.T) {
	const eps = 1e-9
	check := func(name string, got, want float64) {
		t.Helper()
		if math.Abs(got-want) > eps {
			t.Errorf("%s = %f, want %f", name, got, want)
		}
	}
	check("identical f64", simhash.Cosine([]float64{1, 0, 0}, []float64{1, 0, 0}), 1)
	check("identical f32", simhash.Cosine([]float32{1, 0, 0}, []float32{1, 0, 0}), 1)
	check("orthogonal", simhash.Cosine([]float64{1, 0, 0}, []float64{0, 1, 0}), 0)
	check("opposite", simhash.Cosine([]float64{1, 0, 0}, []float64{-1, 0, 0}), -1)
	check("zero vector", simhash.Cosine([]float64{0, 0, 0}, []float64{1, 1, 1}), 0)
	check("empty", simhash.Cosine([]float64{}, []float64{}), 0)
	check("mismatched lengths", simhash.Cosine([]float64{1, 2}, []float64{1, 2, 3}), 0)

	check("unit identical", simhash.CosineUnit([]float64{1, 0}, []float64{1, 0}), 1)
	check("unit orthogonal", simhash.CosineUnit([]float64{1, 0}, []float64{0, 1}), 0.5)
	check("unit opposite", simhash.CosineUnit([]float64{1, 0}, []float64{-1, 0}), 0)
	// Degenerate inputs stay 0 on the unit scale too — not 0.5, which would
	// rank an empty embedding as "half similar" to everything.
	check("unit zero vector", simhash.CosineUnit([]float64{0, 0}, []float64{1, 1}), 0)
	check("unit mismatched", simhash.CosineUnit([]float32{1}, []float32{1, 2}), 0)
}

// simhash is a LEAF (stdlib only): both language-core packages (builtins,
// effects) and platform packages (messaging, coordinator, docsearch) call it.
func TestSimhashIsALeaf(t *testing.T) {
	const module = "github.com/sunholo-data/ailang/"
	out, err := exec.Command("go", "list", "-deps", ".").Output()
	if err != nil {
		t.Fatalf("go list -deps: %v", err)
	}
	deps := strings.Split(strings.TrimSpace(string(out)), "\n")
	sawControl := false
	for _, d := range deps {
		if d == "hash/fnv" {
			sawControl = true
		}
		first, _, _ := strings.Cut(d, "/")
		if strings.Contains(first, ".") && d != module+"internal/simhash" {
			t.Errorf("simhash must be stdlib-only but depends on %s", d)
		}
	}
	if !sawControl {
		t.Fatalf("instrument check failed: %d deps returned but hash/fnv absent", len(deps))
	}
}
