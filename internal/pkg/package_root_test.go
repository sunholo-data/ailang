package pkg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/config"
)

func TestPackageRootResolution(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	root := t.TempDir()
	t.Setenv("AILANG_PACKAGE_ROOT", root)
	t.Setenv(config.EnvAgentPolicy, "policy.toml")
	dir := filepath.Join(root, "test", "lib", "0.1.0")
	writeManifest(t, dir, "[package]\nname = \"test/lib\"\nversion = \"0.1.0\"\nedition = \"1\"\n")
	lf := NewLockFile([]LockedPackage{{Name: "test/lib", Version: "0.1.0", Source: "registry", ContentHash: "unused"}}, "test")
	loader := NewPackageLoader(lf, t.TempDir())
	if _, err := loader.LoadManifestByName("test/lib"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".ailang")); !os.IsNotExist(err) {
		t.Fatalf("HOME changed: %v", err)
	}
	// A valid HOME copy and a legacy lock Path must not override a configured root.
	cached, err := CachedPackagePath("test/lib", "0.1.0")
	if err != nil {
		t.Fatal(err)
	}
	writeManifest(t, cached, "[package]\nname = \"test/lib\"\nversion = \"0.1.0\"\nedition = \"1\"\n")
	lf.Packages[0].Path = cached
	t.Setenv("AILANG_PACKAGE_ROOT", filepath.Join(root, "missing"))
	if _, err := NewPackageLoader(lf, t.TempDir()).LoadManifestByName("test/lib"); err == nil {
		t.Fatal("root miss fell back to HOME or legacy Path")
	}
	t.Setenv("AILANG_PACKAGE_ROOT", "")
	if _, err := NewPackageLoader(lf, t.TempDir()).LoadManifestByName("test/lib"); err != nil {
		t.Fatal(err)
	}
}

func TestPackageRootRegistryLockOffline(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	root := t.TempDir()
	t.Setenv("AILANG_PACKAGE_ROOT", root)
	t.Setenv(config.EnvAgentPolicy, "policy.toml")
	app := t.TempDir()
	writeManifest(t, app, "[package]\nname = \"test/app\"\nversion = \"0.1.0\"\nedition = \"1\"\n[dependencies]\n\"test/lib\" = \"0.1.0\"\n")
	lib := filepath.Join(root, "test", "lib", "0.1.0")
	writeManifest(t, lib, "[package]\nname = \"test/lib\"\nversion = \"0.1.0\"\nedition = \"1\"\n[dependencies]\n\"test/leaf\" = \"0.2.0\"\n")
	writeManifest(t, filepath.Join(root, "test", "leaf", "0.2.0"), "[package]\nname = \"test/leaf\"\nversion = \"0.2.0\"\nedition = \"1\"\n")
	m, err := LoadManifest(app)
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := ResolveDependencies(m, app)
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved) != 2 {
		t.Fatal(resolved)
	}
	if _, err := os.Stat(filepath.Join(home, ".ailang")); !os.IsNotExist(err) {
		t.Fatal("HOME changed")
	}
	// Legacy path deps inside a registry package require an index and must refuse.
	writeManifest(t, lib, "[package]\nname = \"test/lib\"\nversion = \"0.1.0\"\nedition = \"1\"\n[dependencies]\n\"test/leaf\" = { path = \"../leaf\" }\n")
	if _, err := ResolveDependencies(m, app); err == nil || !strings.Contains(err.Error(), "confined by AILANG_AGENT_POLICY") {
		t.Fatalf("expected index refusal: %v", err)
	}
}

func TestPackageRootRejectsTraversal(t *testing.T) {
	t.Setenv("AILANG_PACKAGE_ROOT", t.TempDir())
	for _, identity := range [][2]string{{"../lib", "0.1.0"}, {"test/../lib", "0.1.0"}, {"test/lib", "../outside"}, {"test/lib", `..\outside`}, {"test/lib", ""}} {
		if _, err := PackageDir(identity[0], identity[1]); err == nil {
			t.Fatalf("accepted %v", identity)
		}
	}
}
