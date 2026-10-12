package pipeline

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

func TestCachePurity_DerivedValuesSurviveArtifacts(t *testing.T) {
	ifc := buildIface(t, map[string]string{"purity.ail": `module purity
export func quiet(x: int) -> int ! {} = x
export func effectful(x: int) -> int ! {IO} = x
export func declared[e](f: int -> int ! {e}, x: int) -> int ! {e} = f(x)
export pure func latent(f: int -> int, x: int) -> int = f(x)
`}, "purity.ail")
	root := t.TempDir()
	t.Setenv("AILANG_CACHE_DIR", "")
	cs, err := NewCacheStore(root)
	if err != nil {
		t.Fatal(err)
	}
	key := ModuleCacheKey("same-build", "purity source", nil)
	cm := &CachedModule{Core: &core.Program{}, CoreTI: types.NewCoreTypeInfo(), Iface: ifc}
	if err := cs.StoreArtifacts("purity", key, cm); err != nil {
		t.Fatal(err)
	}
	loaded, err := cs.LoadArtifacts("purity", key)
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]bool{"quiet": true, "effectful": false, "declared": false, "latent": true} {
		if ifc.Exports[name].Purity != want || loaded.Iface.Exports[name].Purity != want {
			t.Errorf("%s cold=%v cached=%v want=%v", name, ifc.Exports[name].Purity, loaded.Iface.Exports[name].Purity, want)
		}
	}
}

func TestCachePurity_RejectsV6AlwaysTrueMetadata(t *testing.T) {
	t.Setenv("AILANG_CACHE_DIR", "")
	root := t.TempDir()
	cs, err := NewCacheStore(root)
	if err != nil {
		t.Fatal(err)
	}
	// Identical compiler identity and key: the format version alone must evict
	// old always-true interface metadata, even for same-build dev binaries.
	key := ModuleCacheKey("same-build", "source", nil)
	cs.Store("purity", &CacheEntry{CacheKey: key, IfaceJSON: []byte(`{"exports":{"effectful":{"purity":true}}}`)})
	if err := cs.Save(); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(root, ".ailang", "cache", "compile", "manifest.json")
	bytes, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest CacheManifest
	if err := json.Unmarshal(bytes, &manifest); err != nil {
		t.Fatal(err)
	}
	manifest.Version = "v6"
	bytes, err = json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, bytes, 0o644); err != nil {
		t.Fatal(err)
	}
	fresh, err := NewCacheStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, hit := fresh.Lookup("purity", key); hit {
		t.Fatal("v6 always-true metadata survived purity migration")
	}
}
