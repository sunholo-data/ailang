package pipeline

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/pkg"
)

// ailang#1498: two packages in one repo — sim/ (pure) and ai/ (depends on
// sim by `{ path = "../sim" }`). Lock in one checkout, copy the whole tree to
// another absolute location, delete the original, and type-check ai/ there.
// Before relative path deps, the lock named checkout-a and the copy failed
// with "package directory not found".
func TestPathDep_LockSurvivesRelocatedCheckout(t *testing.T) {
	root := filepath.Join(t.TempDir(), "checkout-a")
	for _, d := range []string{"sim", "ai"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0755); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite(t, filepath.Join(root, "sim"), "ailang.toml", `[package]
name = "game/sim"
version = "0.1.0"
edition = "1"

[exports]
modules = ["game/sim/markers"]
`)
	mustWrite(t, filepath.Join(root, "sim"), "markers.ail", `module game/sim/markers

export pure func marker() -> string = "joy"
`)
	mustWrite(t, filepath.Join(root, "ai"), "ailang.toml", `[package]
name = "game/ai"
version = "0.1.0"
edition = "1"

[exports]
modules = ["game/ai/main"]

[dependencies]
"game/sim" = { path = "../sim" }
`)
	mustWrite(t, filepath.Join(root, "ai"), "main.ail", `module game/ai/main

import pkg/game/sim/markers (marker)

export pure func describe() -> string = "feeling ${marker()}"
`)

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
	cfg := Config{Mode: ModeCheck, RelaxModules: true}
	if _, err := Run(cfg, Source{Filename: filepath.Join(aiDir, "main.ail")}); err != nil {
		t.Fatalf("instrument: check must pass in the original checkout: %v", err)
	}

	moved := filepath.Join(t.TempDir(), "other-home", "checkout-b")
	copyTreeForTest(t, root, moved)
	if err := os.RemoveAll(root); err != nil {
		t.Fatal(err)
	}
	movedAI := filepath.Join(moved, "ai")
	chdirTo(t, movedAI)

	if _, err := Run(cfg, Source{Filename: filepath.Join(movedAI, "main.ail")}); err != nil {
		t.Fatalf("check must pass in a relocated checkout with the committed lock: %v", err)
	}
	if drift, err := pkg.CheckLock(movedAI); err != nil || len(drift) != 0 {
		t.Fatalf("lock --check must be clean in the relocated checkout: drift=%v err=%v", drift, err)
	}
}

func copyTreeForTest(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		if strings.HasPrefix(info.Name(), ".") {
			return nil // skip caches; the copy is a fresh clone
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.Create(target)
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, in)
		return err
	})
	if err != nil {
		t.Fatalf("copy %s -> %s: %v", src, dst, err)
	}
}
