package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/pkg"
)

// `ailang lock --check` (ailang#1498): clean on a fresh lock, non-nil error
// naming the drift and the fix when the lock carries an old absolute path.
func TestPkgLockCheck_FlagsAbsolutePathDrift(t *testing.T) {
	root := t.TempDir()
	write := func(rel, body string) {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("sim/ailang.toml", "[package]\nname = \"game/sim\"\nversion = \"0.1.0\"\nedition = \"1\"\n")
	write("sim/markers.ail", "module game/sim/markers\n")
	write("ai/ailang.toml", "[package]\nname = \"game/ai\"\nversion = \"0.1.0\"\nedition = \"1\"\n[dependencies]\n\"game/sim\" = { path = \"../sim\" }\n")
	aiDir := filepath.Join(root, "ai")

	m, err := pkg.LoadManifest(aiDir)
	if err != nil {
		t.Fatal(err)
	}
	lf, err := pkg.ResolveLock(m, aiDir, "test")
	if err != nil {
		t.Fatal(err)
	}
	if err := lf.Save(aiDir); err != nil {
		t.Fatal(err)
	}
	if err := pkgLockCheck(aiDir); err != nil {
		t.Fatalf("fresh lock must pass --check: %v", err)
	}

	lf.Packages[0].Path = filepath.Join(root, "sim")
	if err := lf.Save(aiDir); err != nil {
		t.Fatal(err)
	}
	err = pkgLockCheck(aiDir)
	if err == nil || !strings.Contains(err.Error(), "absolute") || !strings.Contains(err.Error(), "ailang lock") {
		t.Fatalf("--check must fail naming the absolute path and the fix, got %v", err)
	}
}
