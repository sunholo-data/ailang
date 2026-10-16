// Package pkgassets provides an offline, two-version asset fixture for runtime
// integration tests. The lock pins older bytes while newer bytes are cached.
package pkgassets

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/pkg"
	"github.com/sunholo-data/ailang/internal/testutil"
)

// Source performs actual asset reads; external state is asserted by the Go
// harness instead of contracts or inline tests. All effects remain explicit.
const Source = `module asset_fixture
import std/package (assetPath)
import std/fs (readFile)
import std/result (Ok, Err)
import std/io (exit)
export func read() -> string ! {FS} =
 match assetPath("test/data", "data.txt") {
  Ok(p) => readFile(p),
  Err(e) => e
 }
export func main() -> () ! {FS, IO} =
 if read() == "locked" then () else exit(23)
`

// New returns the consuming project directory and locked dependency directory.
func New(t *testing.T) (root, locked string) {
	t.Helper()
	testutil.SetHomeDir(t, t.TempDir())
	t.Setenv("AILANG_PACKAGE_ROOT", "")
	root = t.TempDir()
	write := func(path, content string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, v := range []string{"0.3.0", "0.4.0"} {
		dir, err := pkg.PackageDir("test/data", v)
		if err != nil {
			t.Fatal(err)
		}
		content := "newer"
		if v == "0.3.0" {
			content = "locked"
			locked = dir
		}
		write(filepath.Join(dir, "assets", "data.txt"), content)
		write(filepath.Join(dir, pkg.ManifestFile), fmt.Sprintf("[package]\nname = \"test/data\"\nversion = %q\nedition = \"1\"\n", v))
	}
	write(filepath.Join(root, pkg.ManifestFile), "[package]\nname = \"test/consumer\"\nversion = \"0.1.0\"\nedition = \"1\"\n[dependencies]\n\"test/data\" = \"0.3.0\"\n")
	hash, err := pkg.ContentHash(locked)
	if err != nil {
		t.Fatal(err)
	}
	lf := pkg.NewLockFile([]pkg.LockedPackage{{Name: "test/data", Source: "registry", Version: "0.3.0", ContentHash: hash}}, "asset-test")
	if err := lf.Save(root); err != nil {
		t.Fatal(err)
	}
	write(filepath.Join(root, "asset_fixture.ail"), Source)
	return root, locked
}
