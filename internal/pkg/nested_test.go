package pkg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeNestedFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"ailang.toml":                         "[package]\nname = \"demo/root\"\nversion = \"0.1.0\"\nedition = \"1\"\nmodule_prefix = \"root\"\n[exports]\nmodules = [\"root/a\"]\n[effects]\nmax = []\n",
		"root/a.ail":                          "module root/a\nexport func f() -> int = 1\n",
		"packages/sub/ailang.toml":            "[package]\nname = \"demo/sub\"\nversion = \"0.1.0\"\nedition = \"1\"\nmodule_prefix = \"sub\"\n[exports]\nmodules = [\"sub/b\"]\n[effects]\nmax = []\n",
		"packages/sub/sub/b.ail":              "module sub/b\nexport func g() -> int = 2\n",
		"packages/sub/_namedtest_body_12.ail": "module x\n",
		"docs/notes/plain.ail":                "module root/docs/notes/plain\nexport func h() -> int = 3\n",
	}
	for rel, body := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// TST-PKGPAR-05
// A directory holding its own ailang.toml is another package: the outer
// package's tarball, content hash, staging copy, source discovery and
// stale-test-body scan all leave it out. A plain subdirectory stays in.
func TestNestedPackage_IsABoundary(t *testing.T) {
	root := writeNestedFixture(t)

	data, err := CreateTarball(root)
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if err := ExtractTarball(data, out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(out, "packages")); !os.IsNotExist(err) {
		t.Errorf("nested package shipped in the outer tarball (stat err=%v)", err)
	}
	if _, err := os.Stat(filepath.Join(out, "docs", "notes", "plain.ail")); err != nil {
		t.Errorf("a plain subdirectory must still ship: %v", err)
	}

	before, err := ContentHash(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "packages", "sub", "sub", "b.ail"), []byte("module sub/b\nexport func g() -> int = 99\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	after, err := ContentHash(root)
	if err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Errorf("editing a nested package changed the outer content hash: %s -> %s", before, after)
	}

	stage := t.TempDir()
	if err := copyPackageContents(root, stage); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(stage, "packages", "sub")); !os.IsNotExist(err) {
		t.Errorf("publish staging copied the nested package (stat err=%v)", err)
	}

	src, err := DiscoverPackageSources(root)
	if err != nil {
		t.Fatal(err)
	}
	for f := range src.SourceFiles {
		if strings.Contains(filepath.ToSlash(f), "packages/sub") {
			t.Errorf("discovery picked up nested source %s", f)
		}
	}

	stale, err := FindNamedTestBodyFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) != 0 {
		t.Errorf("the nested package's leftovers are its own, not PUB024 for the root: %v", stale)
	}
}

func TestIsNestedPackageRoot(t *testing.T) {
	root := writeNestedFixture(t)
	if IsNestedPackageRoot(root, root) || IsNestedPackageRoot(root+string(filepath.Separator), root) {
		t.Error("the package's own root is not nested")
	}
	if !IsNestedPackageRoot(filepath.Join(root, "packages", "sub"), root) {
		t.Error("packages/sub has its own ailang.toml")
	}
	if IsNestedPackageRoot(filepath.Join(root, "docs"), root) {
		t.Error("a plain subdirectory is not a package")
	}
}
