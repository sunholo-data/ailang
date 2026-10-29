package pkg

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/sunholo-data/ailang/internal/testutil"
)

func TestPackageLoaderAssetTargets(t *testing.T) {
	home := t.TempDir()
	testutil.SetHomeDir(t, home)
	root := t.TempDir()
	registry, err := PackageDir("test/registry", "0.3.0")
	if err != nil {
		t.Fatal(err)
	}
	local := filepath.Join(root, "local")
	git := filepath.Join(root, "git")
	for _, dir := range []string{registry, local, git} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	lf := NewLockFile([]LockedPackage{
		{Name: "test/registry", Source: "registry", Version: "0.3.0"},
		{Name: "test/local", Source: "path", Path: "local", Version: "0.1.0"},
		{Name: "test/git", Source: "git", Path: git, Version: "0.2.0"},
		{Name: "test/missing", Source: "path", Path: "missing", Version: "0.5.0"},
	}, "test")
	pl := NewPackageLoader(lf, root)
	for name, want := range map[string]string{"test/registry": registry, "test/local": local, "test/git": git} {
		dir, version, found := pl.AssetTarget(name)
		if !found || dir != want || version == "" {
			t.Errorf("%s: %q %q %v", name, dir, version, found)
		}
	}
	dir, version, found := pl.AssetTarget("test/missing")
	if !found || dir != "" || version != "0.5.0" {
		t.Fatalf("missing target: %q %q %v", dir, version, found)
	}
	if _, _, found := pl.AssetTarget("test/absent"); found {
		t.Fatal("absent package reported locked")
	}
	want := []string{"test/git", "test/local", "test/missing", "test/registry"}
	if !reflect.DeepEqual(pl.LockedPackageNames(), want) {
		t.Fatal(pl.LockedPackageNames())
	}
}
