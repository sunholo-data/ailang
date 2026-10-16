package effects

import (
	"errors"
	"testing"
)

func TestSharedIndexOpenerRegistration(t *testing.T) {
	indexOpenerMu.RLock()
	old := indexOpener
	indexOpenerMu.RUnlock()
	defer RegisterSharedIndexOpener(old)
	RegisterSharedIndexOpener(nil)
	if _, err := OpenSharedIndex("test.db"); !errors.Is(err, ErrBackendNotRegistered) {
		t.Fatalf("got %v", err)
	}
	sentinel := errors.New("cannot open")
	RegisterSharedIndexOpener(func(path string) (PersistentSharedIndex, error) {
		if path != "test.db" {
			t.Fatal(path)
		}
		return nil, sentinel
	})
	if _, err := OpenSharedIndex("test.db"); !errors.Is(err, sentinel) {
		t.Fatal(err)
	}
}

func TestCheckedIndexLegacyAdapter(t *testing.T) {
	mem := NewInMemorySharedIndex()
	idx := CheckedIndex(mem)
	if err := idx.UpsertChecked("n", "a", -1, 3, 4); err != nil {
		t.Fatal(err)
	}
	if err := idx.UpsertWithEmbeddingChecked("n", "b", 0, []float64{1}, 4, 5); err != nil {
		t.Fatal(err)
	}
	if n, err := idx.EntryCountChecked("n"); err != nil || n != 2 {
		t.Fatalf("%d %v", n, err)
	}
	if ns, err := idx.NamespacesChecked(); err != nil || len(ns) != 1 {
		t.Fatalf("%v %v", ns, err)
	}
	if hits, err := idx.FindSimilarSimHashChecked("n", -1, 1, 0, DeterminismStrict); err != nil || len(hits) != 1 || hits[0].Key != "a" {
		t.Fatalf("%v %v", hits, err)
	}
	if hits, err := idx.FindSimilarByEmbeddingChecked("n", []float64{1}, 1, 0, DeterminismStrict); err != nil || len(hits) != 1 || hits[0].Key != "b" {
		t.Fatalf("%v %v", hits, err)
	}
	if err := idx.DeleteChecked("n", "a"); err != nil {
		t.Fatal(err)
	}
}
