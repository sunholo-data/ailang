package sharedmem

import (
	"fmt"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
)

func TestEmbeddingCacheReopenAndCorruption(t *testing.T) {
	path := filepath.Join(t.TempDir(), "library.db")
	c, err := OpenEmbeddingCache(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = c.Put("model", "hash", []float32{1, 2}); err != nil {
		t.Fatal(err)
	}
	var createdAt int64
	if err := c.db.QueryRow("SELECT created_at FROM embed_cache WHERE model=? AND text_hash=?", "model", "hash").Scan(&createdAt); err != nil || createdAt <= 0 {
		t.Fatalf("missing creation timestamp: %d %v", createdAt, err)
	}
	c.Close()
	c, err = OpenEmbeddingCache(path)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	v, ok, err := c.Get("model", "hash", 2)
	if err != nil || !ok || len(v) != 2 || v[1] != 2 {
		t.Fatalf("%v %v %v", v, ok, err)
	}
	if _, _, err = c.Get("model", "hash", 3); err == nil {
		t.Fatal("dimension mismatch ignored")
	}
	_, err = c.db.Exec("UPDATE embed_cache SET vector=x'01'")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = c.Get("model", "hash", 2); err == nil {
		t.Fatal("corrupt blob ignored")
	}
	c.Close()
	if _, _, err = c.Get("x", "y", 2); err == nil {
		t.Fatal("closed cache disguised as miss")
	}
}

type persistentCountingEmbedder struct {
	calls int
	fail  bool
}

func (e *persistentCountingEmbedder) Embed(string) ([]float32, error) {
	e.calls++
	if e.fail {
		return nil, fmt.Errorf("provider unavailable")
	}
	return []float32{0.125, 0.75}, nil
}
func (e *persistentCountingEmbedder) EmbedBatch(texts []string) ([][]float32, error) {
	return nil, fmt.Errorf("unused")
}
func (e *persistentCountingEmbedder) ModelName() string { return "provider/model-v1" }
func (e *persistentCountingEmbedder) Dimension() int    { return 2 }
func TestEmbeddingCacheRestartSkipsProviderAndIsolatesBrain(t *testing.T) {
	path := filepath.Join(t.TempDir(), "all.db")
	brain, err := NewSQLiteSharedCache(path)
	if err != nil {
		t.Fatal(err)
	}
	defer brain.Close()
	store, err := OpenEmbeddingCache(path)
	if err != nil {
		t.Fatal(err)
	}
	provider := &persistentCountingEmbedder{}
	c := effects.NewCachingEmbedder(provider, store)
	original, err := c.Embed("remember this")
	if err != nil || provider.calls != 1 {
		t.Fatalf("%v calls=%d", err, provider.calls)
	}
	store.Close()
	reopened, err := OpenEmbeddingCache(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	offline := &persistentCountingEmbedder{fail: true}
	cached := effects.NewCachingEmbedder(offline, reopened)
	vector, err := cached.Embed("remember this")
	if err != nil || offline.calls != 0 || !reflect.DeepEqual(original, vector) {
		t.Fatalf("%v %v calls=%d", vector, err, offline.calls)
	}
	if _, err := cached.Embed("different"); err == nil || offline.calls != 1 {
		t.Fatal("different text did not miss")
	}
	var count int
	if err := brain.db.QueryRow("SELECT COUNT(*) FROM brain_frames").Scan(&count); err != nil || count != 0 {
		t.Fatalf("cache leaked into brain: %d %v", count, err)
	}
}
