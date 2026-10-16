package runner

import (
	"os"
	"path/filepath"
	"testing"
)

// #1772: the runtime must see the package directories the lock pins, so
// std/package.assetPath reads the version the program was compiled against.
func TestLockedPackageDirs(t *testing.T) {
	root := t.TempDir()
	dep := filepath.Join(root, "dep")
	app := filepath.Join(root, "app")
	for _, d := range []string{dep, filepath.Join(app, "src")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write := func(path, body string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(app, "ailang.toml"), "[package]\nname = \"probe/app\"\nversion = \"0.1.0\"\nedition = \"1\"\n")
	write(filepath.Join(app, "ailang.lock"), `{"schema":"ailang.lock/v1","schema_version":"1.0.0","generator":"test",
"packages":[{"name":"probe/dep","version":"0.3.0","source":"path","path":"../dep","content_hash":"sha256:test"}]}`)

	dirs := lockedPackageDirs(filepath.Join(app, "src", "main.ail"), "")
	if dirs["probe/dep"] != dep {
		t.Errorf("probe/dep = %q, want %q", dirs["probe/dep"], dep)
	}
	if dirs["probe/app"] != app {
		t.Errorf("root package probe/app = %q, want %q", dirs["probe/app"], app)
	}
	if got := lockedPackageDirs(filepath.Join(t.TempDir(), "x.ail"), ""); got != nil {
		t.Errorf("no manifest: got %v, want nil", got)
	}
}
