package sharedmem

import (
	"math"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
)

func TestSQLiteSharedIndexSearchParity(t *testing.T) {
	idx, err := NewSQLiteSharedIndex(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer idx.Close()
	mem := effects.NewInMemorySharedIndex()
	for n, key := range []string{"z", "c", "b", "a"} {
		v := []float64{float64(n + 1), 0.25}
		for _, store := range []effects.SharedIndex{idx, mem} {
			store.UpsertWithEmbedding("n", key, int64(n), v, int64(n), 42)
		}
	}
	for _, maxScan := range []int{0, 1, 2, 100} {
		for _, topK := range []int{0, 1, 100} {
			got, err := idx.FindSimilarSimHashChecked("n", 0, topK, maxScan, effects.DeterminismStrict)
			if err != nil {
				t.Fatal(err)
			}
			want := mem.FindSimilarSimHash("n", 0, topK, maxScan, effects.DeterminismStrict)
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("scan %d top %d: %v != %v", maxScan, topK, got, want)
			}
			got, err = idx.FindSimilarByEmbeddingChecked("n", []float64{1, 0}, topK, maxScan, effects.DeterminismStrict)
			if err != nil {
				t.Fatal(err)
			}
			want = mem.FindSimilarByEmbedding("n", []float64{1, 0}, topK, maxScan, effects.DeterminismStrict)
			if len(got) != len(want) {
				t.Fatal("result count")
			}
			for n := range got {
				if got[n].Key != want[n].Key || math.Abs(got[n].Score-want[n].Score) > 1e-6 {
					t.Fatalf("%v != %v", got, want)
				}
			}
		}
	}
	// Plain upsert replaces and clears the previous embedding.
	if err := idx.UpsertChecked("n", "a", 0, 8, 9); err != nil {
		t.Fatal(err)
	}
	hits, err := idx.FindSimilarByEmbeddingChecked("n", []float64{1}, 0, 1, effects.DeterminismStrict)
	if err != nil || len(hits) != 0 {
		t.Fatalf("replacement: %v %v", hits, err)
	}
	// Dimension mismatch preserves existing cosine helper's 0 score.
	hits, err = idx.FindSimilarByEmbeddingChecked("n", []float64{1}, 0, 0, effects.DeterminismStrict)
	if err != nil || len(hits) != 3 {
		t.Fatalf("mismatch: %v %v", hits, err)
	}
	for _, hit := range hits {
		if hit.Score != 0 {
			t.Fatal(hit)
		}
	}
	if err := idx.DeleteChecked("n", "b"); err != nil {
		t.Fatal(err)
	}
	if idx.EntryCount("n") != 3 {
		t.Fatal("delete")
	}
	if _, err := idx.db.Exec("UPDATE sharedindex_entries SET embedding=X'01' WHERE key='c'"); err != nil {
		t.Fatal(err)
	}
	if _, err := idx.FindSimilarByEmbeddingChecked("n", []float64{1, 0}, 0, 0, effects.DeterminismStrict); err == nil {
		t.Fatal("corrupt blob accepted")
	}
}

func TestSQLiteSharedIndexHandlesAndFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shared.db")
	a, err := NewSQLiteSharedIndex(path)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := NewSQLiteSharedIndex(path)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	c, err := NewSQLiteSharedIndex(filepath.Join(t.TempDir(), "other.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if err := a.UpsertChecked("n", "key", -1, 1, 2); err != nil {
		t.Fatal(err)
	}
	if b.EntryCount("n") != 1 || c.EntryCount("n") != 0 {
		t.Fatal("file isolation")
	}
	names, err := b.NamespacesChecked()
	if err != nil || !reflect.DeepEqual(names, []string{"n"}) {
		t.Fatalf("%v %v", names, err)
	}
	if err := b.DeleteChecked("n", "key"); err != nil {
		t.Fatal(err)
	}
	if len(a.Namespaces()) != 0 {
		t.Fatal("empty namespace")
	}
}
