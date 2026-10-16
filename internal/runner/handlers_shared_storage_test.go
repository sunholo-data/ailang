package runner

import (
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/platform/sharedmem"
)

func TestSharedStoragePersistence(t *testing.T) {
	sharedmem.Register()
	t.Setenv("AILANG_SHAREDMEM_DB", filepath.Join(t.TempDir(), "env.db"))
	path := filepath.Join(t.TempDir(), "flag.db")
	ctx := effects.NewEffContext(nil)
	ctx.Grant(effects.NewCapability("SharedMem"))
	closeStore, err := SetupSharedStorage(ctx, SharedStorageOptions{SharedMemDB: path})
	if err != nil {
		t.Fatal(err)
	}
	ctx.SharedMem.Cache.Put("universe", []byte("voyage"))
	if err := closeStore(); err != nil {
		t.Fatal(err)
	}
	ctx2 := effects.NewEffContext(nil)
	ctx2.Grant(effects.NewCapability("SharedMem"))
	closeStore, err = SetupSharedStorage(ctx2, SharedStorageOptions{SharedMemDB: path})
	if err != nil {
		t.Fatal(err)
	}
	defer closeStore()
	value, ok := ctx2.SharedMem.Cache.Get("universe")
	if !ok || string(value) != "voyage" {
		t.Fatalf("persisted value %q, %v", value, ok)
	}
}

func TestSharedStorageCapabilityAndOpenErrors(t *testing.T) {
	sharedmem.Register()
	t.Setenv("AILANG_SHAREDMEM_DB", "")
	t.Setenv("AILANG_SHAREDINDEX_DB", "")
	ctx := effects.NewEffContext(nil)
	if _, err := SetupSharedStorage(ctx, SharedStorageOptions{SharedMemDB: filepath.Join(t.TempDir(), "store.db")}); err == nil {
		t.Fatal("missing capability accepted")
	}
	ctx.Grant(effects.NewCapability("SharedMem"))
	if _, err := SetupSharedStorage(ctx, SharedStorageOptions{SharedMemDB: t.TempDir()}); err == nil {
		t.Fatal("directory accepted as database")
	}
	closeStore, err := SetupSharedStorage(ctx, SharedStorageOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer closeStore()
	if _, ok := ctx.SharedMem.Cache.(*effects.InMemorySharedCache); !ok {
		t.Fatal("default is not in memory")
	}
}

func TestSharedStorageIndexPrecedenceAndIsolation(t *testing.T) {
	sharedmem.Register()
	envPath := filepath.Join(t.TempDir(), "env.db")
	t.Setenv("AILANG_SHAREDINDEX_DB", envPath)
	t.Setenv("AILANG_SHAREDMEM_DB", "")
	path := filepath.Join(t.TempDir(), "flag.db")
	ctx := effects.NewEffContext(nil)
	ctx.Grant(effects.NewCapability("SharedIndex"))
	closeStores, err := SetupSharedStorage(ctx, SharedStorageOptions{SharedIndexDB: path})
	if err != nil {
		t.Fatal(err)
	}
	ctx.SharedIndex.Index.Upsert("archive", "key", -1, 2, 1000)
	if err := closeStores(); err != nil {
		t.Fatal(err)
	}
	closeStores, err = SetupSharedStorage(ctx, SharedStorageOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if n := ctx.SharedIndex.Index.EntryCount("archive"); n != 0 {
		t.Fatalf("flag leaked into env DB: %d", n)
	}
	closeStores()
	closeStores, err = SetupSharedStorage(ctx, SharedStorageOptions{SharedIndexDB: path})
	if err != nil {
		t.Fatal(err)
	}
	defer closeStores()
	if n := ctx.SharedIndex.Index.EntryCount("archive"); n != 1 {
		t.Fatalf("flag DB not durable: %d", n)
	}
	if _, err := SetupSharedStorage(effects.NewEffContext(nil), SharedStorageOptions{}); err == nil {
		t.Fatal("environment path without cap accepted")
	}
}

func TestSharedStorageClosesPartialSetup(t *testing.T) {
	sharedmem.Register()
	t.Setenv("AILANG_SHAREDMEM_DB", "")
	t.Setenv("AILANG_SHAREDINDEX_DB", "")
	ctx := effects.NewEffContext(nil)
	ctx.Grant(effects.NewCapability("SharedMem"))
	ctx.Grant(effects.NewCapability("SharedIndex"))
	_, err := SetupSharedStorage(ctx, SharedStorageOptions{SharedMemDB: filepath.Join(t.TempDir(), "mem.db"), SharedIndexDB: t.TempDir()})
	if err == nil {
		t.Fatal("bad index path accepted")
	}
	cache := ctx.SharedMem.Cache.(effects.CheckedSharedCache)
	if err := cache.PutChecked("x", []byte("y")); err == nil {
		t.Fatal("partially opened cache leaked")
	}
}
