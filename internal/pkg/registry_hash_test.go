package pkg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/testutil"
)

func TestRegistryContentHash(t *testing.T) {
	t.Setenv("AILANG_PACKAGE_ROOT", t.TempDir())
	dir, err := PackageDir("test/lib", "0.1.0")
	if err != nil {
		t.Fatal(err)
	}
	writeManifest(t, dir, "[package]\nname = \"test/lib\"\nversion = \"0.1.0\"\nedition = \"1\"\n")
	source := filepath.Join(dir, "core.ail")
	if err := os.WriteFile(source, []byte("module test/lib/core\n"), 0644); err != nil {
		t.Fatal(err)
	}
	hash, err := ContentHash(dir)
	if err != nil {
		t.Fatal(err)
	}
	lf := NewLockFile([]LockedPackage{{Name: "test/lib", Version: "0.1.0", Source: "registry", ContentHash: hash}}, "test")
	if err := lf.ValidateContentHashesFrom(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("module test/lib/changed\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := lf.ValidateContentHashesFrom(t.TempDir()); err == nil || !strings.Contains(err.Error(), "content changed") {
		t.Fatalf("tampering accepted: %v", err)
	}
	lf.Packages[0].ContentHash = "x"
	if err := lf.ValidateContentHashes(); err == nil {
		t.Fatal("short hash accepted")
	}
	t.Setenv("AILANG_PACKAGE_ROOT", t.TempDir())
	if err := lf.ValidateContentHashes(); err == nil || !strings.Contains(err.Error(), "not provisioned in AILANG_PACKAGE_ROOT") || strings.Contains(err.Error(), "lstat") {
		t.Fatalf("missing registry directory: want named not-provisioned error, got %v", err)
	}
	lf.Packages[0].ContentHash = ""
	if err := lf.Validate(); err == nil {
		t.Fatal("empty hash accepted")
	}
}

func TestRegistryContentHashCannotBeMaskedByPathDrift(t *testing.T) {
	t.Setenv("AILANG_PACKAGE_ROOT", t.TempDir())
	path := t.TempDir()
	lf := NewLockFile([]LockedPackage{
		{Name: "a/path", Source: "path", Path: path, ContentHash: "x"},
		{Name: "test/lib", Version: "0.1.0", Source: "registry", ContentHash: "x"},
	}, "test")
	err := lf.ValidateContentHashes()
	if _, ok := err.(*RegistryContentError); !ok {
		t.Fatalf("registry trust failure masked: %T %v", err, err)
	}
}

func TestRegistryContentHashHomeCache(t *testing.T) {
	home := t.TempDir()
	testutil.SetHomeDir(t, home)
	t.Setenv("AILANG_PACKAGE_ROOT", "")
	dir, err := CachedPackagePath("test/lib", "0.1.0")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(dir, "core.ail")
	if err := os.WriteFile(source, []byte("module test/lib/core\n"), 0644); err != nil {
		t.Fatal(err)
	}
	hash, err := ContentHash(dir)
	if err != nil {
		t.Fatal(err)
	}
	lf := NewLockFile([]LockedPackage{{Name: "test/lib", Version: "0.1.0", Source: "registry", ContentHash: hash}}, "test")
	if err := lf.ValidateContentHashes(); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AILANG_PACKAGE_ROOT", t.TempDir())
	if err := lf.ValidateContentHashes(); err == nil {
		t.Fatal("root-only hash fell back to HOME")
	}
	t.Setenv("AILANG_PACKAGE_ROOT", "")
	if err := os.WriteFile(source, []byte("module test/lib/changed\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := lf.ValidateContentHashes(); err == nil {
		t.Fatal("HOME tampering accepted")
	}
}
