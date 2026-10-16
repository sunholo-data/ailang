package sharedmem

import (
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
)

func TestSQLiteSharedIndexPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "library.db")
	idx, err := NewSQLiteSharedIndex(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := idx.UpsertWithEmbeddingChecked("a", "same", -1, []float64{1, 0}, 7, 42); err != nil {
		t.Fatal(err)
	}
	if err := idx.UpsertChecked("b", "same", 0, 1, 2); err != nil {
		t.Fatal(err)
	}
	if err := idx.Close(); err != nil {
		t.Fatal(err)
	}
	idx, err = NewSQLiteSharedIndex(path)
	if err != nil {
		t.Fatal(err)
	}
	defer idx.Close()
	got, err := idx.FindSimilarSimHashChecked("a", -1, 1, 0, effects.DeterminismStrict)
	if err != nil || len(got) != 1 || got[0].Score != 1 || got[0].Version != 7 || got[0].Timestamp != 42 {
		t.Fatalf("got %v: %v", got, err)
	}
	emb, err := idx.FindSimilarByEmbeddingChecked("a", []float64{1, 0}, 1, 0, effects.DeterminismStrict)
	if err != nil || len(emb) != 1 || emb[0].Score != 1 {
		t.Fatalf("reopened embedding: %v %v", emb, err)
	}
	if idx.EntryCount("a") != 1 || idx.EntryCount("b") != 1 {
		t.Fatal("namespace isolation")
	}
	idx.Close()
	if err := idx.DeleteChecked("a", "same"); err == nil {
		t.Fatal("closed delete succeeded")
	}
	if _, err := idx.EntryCountChecked("a"); err == nil {
		t.Fatal("closed count succeeded")
	}
	if _, err := idx.NamespacesChecked(); err == nil {
		t.Fatal("closed namespaces succeeded")
	}
	if _, err := idx.FindSimilarByEmbeddingChecked("a", []float64{1}, 1, 0, effects.DeterminismStrict); err == nil {
		t.Fatal("closed embedding search succeeded")
	}
	if err := idx.UpsertChecked("a", "bad", 0, 0, 0); err == nil {
		t.Fatal("closed write succeeded")
	}
	if _, err := idx.FindSimilarSimHashChecked("a", 0, 1, 0, effects.DeterminismStrict); err == nil {
		t.Fatal("closed read succeeded")
	}
}
