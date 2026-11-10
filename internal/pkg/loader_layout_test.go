package pkg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPackageLayout_LockedDependencyAndRoot(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("dep/ailang.toml", "[package]\nname = \"acme/dep\"\nmodule_prefix = \"dp\"\nversion = \"0.1.0\"\nedition = \"1\"\n\n[dependencies]\n\"z/b\" = \"0.1.0\"\n\"a/a\" = \"0.1.0\"\n")
	write("app/ailang.toml", "[package]\nname = \"acme/app\"\nmodule_prefix = \"ap\"\nversion = \"0.1.0\"\nedition = \"1\"\n")
	lf := NewLockFile([]LockedPackage{{Name: "acme/dep", Version: "0.1.0", Source: "path", Path: "../dep"}}, "test")
	pl := NewPackageLoader(lf, filepath.Join(root, "app"))

	dir, prefix, deps, ok := pl.PackageLayout("acme/dep")
	if !ok || prefix != "dp" || !strings.HasSuffix(filepath.ToSlash(dir), "/dep") || strings.Join(deps, ",") != "a/a,z/b" {
		t.Fatalf("dep layout = (%q, %q, %v, %v)", dir, prefix, deps, ok)
	}
	dir, prefix, deps, ok = pl.PackageLayout("acme/app")
	if !ok || prefix != "ap" || dir != filepath.Join(root, "app") || len(deps) != 0 {
		t.Fatalf("root layout = (%q, %q, %v, %v)", dir, prefix, deps, ok)
	}
	if _, _, _, ok := pl.PackageLayout("acme/unknown"); ok {
		t.Fatal("an unknown package has no layout")
	}
	broken := NewLockFile([]LockedPackage{{Name: "acme/gone", Version: "0.1.0", Source: "path", Path: "../gone"}}, "test")
	if _, _, _, ok := NewPackageLoader(broken, filepath.Join(root, "app")).PackageLayout("acme/gone"); ok {
		t.Fatal("a locked package whose manifest does not load has no layout")
	}
}
