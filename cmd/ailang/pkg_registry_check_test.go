package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/pkg"
	"github.com/sunholo-data/ailang/internal/testutil"
)

func TestPkgRegistryContentHashCheck(t *testing.T) {
	bin := buildAilang(t)
	home := t.TempDir()
	testutil.SetHomeDir(t, home)
	root := t.TempDir()
	t.Setenv(config.EnvPackageRoot, root)
	t.Setenv(config.EnvAgentPolicy, "policy.toml")
	app := t.TempDir()
	lib, err := pkg.PackageDir("test/lib", "0.1.0")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(lib, 0755); err != nil {
		t.Fatal(err)
	}
	for path, data := range map[string]string{
		filepath.Join(app, "ailang.toml"): "[package]\nname = \"test/app\"\nversion = \"0.1.0\"\nedition = \"1\"\n[dependencies]\n\"test/lib\" = \"0.1.0\"\n",
		filepath.Join(app, "main.ail"):    "module test/app/main\nimport pkg/test/lib/core (answer)\nexport pure func main() -> int = answer()\n",
		filepath.Join(lib, "ailang.toml"): "[package]\nname = \"test/lib\"\nversion = \"0.1.0\"\nedition = \"1\"\n",
		filepath.Join(lib, "core.ail"):    "module test/lib/core\nexport pure func answer() -> int = 42\n",
	} {
		if err := os.WriteFile(path, []byte(data), 0644); err != nil {
			t.Fatal(err)
		}
	}
	hash, err := pkg.ContentHash(lib)
	if err != nil {
		t.Fatal(err)
	}
	lf := pkg.NewLockFile([]pkg.LockedPackage{{Name: "test/lib", Version: "0.1.0", Source: "registry", ContentHash: hash}}, "test")
	if err := lf.Save(app); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(lib, 0555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(lib, 0755) })
	before := registryTreeSnapshot(t, root)
	entry := filepath.Join(app, "main.ail")
	out, stderr, exit := runAilangBin(t, bin, "check", entry)
	if exit != 0 {
		t.Fatalf("matching registry check exit %d: %s%s", exit, out, stderr)
	}
	if after := registryTreeSnapshot(t, root); !reflect.DeepEqual(before, after) {
		t.Fatalf("package root changed: %v -> %v", before, after)
	}
	if _, err := os.Stat(filepath.Join(home, ".ailang")); !os.IsNotExist(err) {
		t.Fatalf("HOME changed: %v", err)
	}
	if err := os.Chmod(lib, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(lib, "core.ail"), []byte("module test/lib/core\nexport pure func answer() -> int = 99\n"), 0644); err != nil {
		t.Fatal(err)
	}
	_, stderr, exit = runAilangBin(t, bin, "check", entry)
	if exit != 1 || !strings.Contains(stderr, "test/lib content changed") {
		t.Fatalf("tampered check exit %d: %s", exit, stderr)
	}
}

func registryTreeSnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	snapshot := map[string]string{}
	if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			snapshot[rel] = "dir:" + info.Mode().String()
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		snapshot[rel] = info.Mode().String() + ":" + string(data)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return snapshot
}
