package effects

import (
	"fmt"
	"testing"
)

// #1795: Strict + maxScan > 0 must score the same candidates on every call.
// Before the fix the candidate set was the first maxScan keys of a randomised
// map walk, so the top hit varied between runs.
func TestStrictBoundedScanIsDeterministic(t *testing.T) {
	idx := NewInMemorySharedIndex()
	emb := NewInMemorySharedIndex()
	for i := 1; i <= 6; i++ {
		key := fmt.Sprintf("k%d", i)
		idx.Upsert("ns", key, int64(1)<<uint(i*8), 1, 0)
		emb.UpsertWithEmbedding("ns", key, 0, []float64{float64(i), 1}, 1, 0)
	}
	// A prefix scan of 3 sorted keys sees k1..k3 only.
	for run := 0; run < 50; run++ {
		got := idx.FindSimilarSimHash("ns", int64(1)<<48, 6, 3, DeterminismStrict)
		if len(got) != 3 {
			t.Fatalf("run %d: simhash results %v, want 3 (the k1..k3 prefix)", run, got)
		}
		for _, r := range got {
			if r.Key > "k3" {
				t.Fatalf("run %d: scanned %s outside the k1..k3 prefix: %v", run, r.Key, got)
			}
		}
		e := emb.FindSimilarByEmbedding("ns", []float64{6, 1}, 1, 3, DeterminismStrict)
		if len(e) != 1 || e[0].Key != "k3" {
			t.Fatalf("run %d: embedding top hit %v, want k3 (best within the k1..k3 prefix)", run, e)
		}
	}
}

func TestSharedCacheKeysSorted(t *testing.T) {
	c := NewInMemorySharedCache()
	for _, k := range []string{"d", "a", "c", "b", "e"} {
		c.Put(k, []byte(k))
	}
	for run := 0; run < 20; run++ {
		if got := fmt.Sprint(c.Keys()); got != "[a b c d e]" {
			t.Fatalf("Keys() = %s, want [a b c d e]", got)
		}
	}
}
