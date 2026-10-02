package pkg

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ailang#1498: a repo with two packages (sim/ pure, ai/ effectful) where ai/
// depends on sim/ by `{ path = "../sim" }`. The committed ailang.lock must
// survive a CI clone or a bundle unpacked under another HOME.

const simManifest = "[package]\nname = \"game/sim\"\nversion = \"0.1.0\"\nedition = \"1\"\n[exports]\nmodules = [\"game/sim/markers\"]\n"
const simMarkers = "module game/sim/markers\n\nexport pure func marker() -> string = \"joy\"\n"
const aiManifest = "[package]\nname = \"game/ai\"\nversion = \"0.1.0\"\nedition = \"1\"\n[exports]\nmodules = [\"game/ai/main\"]\n[dependencies]\n\"game/sim\" = { path = \"../sim\" }\n"

// buildTwoPackageRepo writes sim/ and ai/ under root and returns ai/'s dir.
func buildTwoPackageRepo(t *testing.T, root string) string {
	t.Helper()
	writeFile(t, filepath.Join(root, "sim", "ailang.toml"), simManifest)
	writeFile(t, filepath.Join(root, "sim", "markers.ail"), simMarkers)
	writeFile(t, filepath.Join(root, "ai", "ailang.toml"), aiManifest)
	return filepath.Join(root, "ai")
}

// copyTree copies src to dst (files only, modes preserved loosely).
func copyTree(t *testing.T, src, dst string) {
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

// writeLock resolves dir's manifest and saves ailang.lock the way `ailang
// lock` does.
func writeLock(t *testing.T, dir string) *LockFile {
	t.Helper()
	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatal(err)
	}
	lf, err := ResolveLock(m, dir, "test")
	if err != nil {
		t.Fatalf("ResolveLock: %v", err)
	}
	if err := lf.Save(dir); err != nil {
		t.Fatal(err)
	}
	return lf
}

// The lock written in one checkout must name no absolute location, and
// `lock --check` (CheckLock) must pass in a copy of the tree at a different
// absolute path — with the original deleted, so nothing can lean on it.
// Kills: portablePathDep returning depDir; CheckLock comparing absolute dirs.
func TestCheckLock_StableAcrossRelocatedCheckout(t *testing.T) {
	root := filepath.Join(t.TempDir(), "checkout-a")
	aiDir := buildTwoPackageRepo(t, root)
	writeLock(t, aiDir)

	raw, err := os.ReadFile(filepath.Join(aiDir, LockFileName))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), filepath.ToSlash(root)) || strings.Contains(string(raw), root) {
		t.Fatalf("ailang.lock bakes in the checkout's absolute path %s:\n%s", root, raw)
	}
	if !strings.Contains(string(raw), `"path": "../sim"`) {
		t.Fatalf("ailang.lock must record the path dep relative to ai/ailang.toml with forward slashes:\n%s", raw)
	}

	if drift, err := CheckLock(aiDir); err != nil || len(drift) != 0 {
		t.Fatalf("fresh lock must be current in its own checkout: drift=%v err=%v", drift, err)
	}

	moved := filepath.Join(t.TempDir(), "elsewhere", "checkout-b")
	copyTree(t, root, moved)
	if err := os.RemoveAll(root); err != nil {
		t.Fatal(err)
	}
	movedAI := filepath.Join(moved, "ai")

	drift, err := CheckLock(movedAI)
	if err != nil || len(drift) != 0 {
		t.Fatalf("lock --check must be stable across checkouts: drift=%v err=%v", drift, err)
	}
	lf, err := LoadLockFile(movedAI)
	if err != nil {
		t.Fatal(err)
	}
	if err := lf.ValidateContentHashesFrom(movedAI); err != nil {
		t.Errorf("content hash must not depend on the absolute location: %v", err)
	}
	loader := NewPackageLoader(lf, movedAI)
	file, err := loader.ResolveImport("game/sim/markers")
	if err != nil {
		t.Fatalf("relocated checkout must resolve the path dep: %v", err)
	}
	if !strings.HasPrefix(file, moved) {
		t.Errorf("resolved %s, want a file under the relocated checkout %s", file, moved)
	}
}

// An old lock (written before relative paths) names an absolute directory.
// It must still READ while that directory exists, `lock --check` must call it
// drift with a fix-it, and the next lock rewrites it relative.
func TestCheckLock_FlagsAbsolutePathFromOldLock(t *testing.T) {
	root := t.TempDir()
	aiDir := buildTwoPackageRepo(t, root)
	lf := writeLock(t, aiDir)
	simAbs, err := filepath.Abs(filepath.Join(root, "sim"))
	if err != nil {
		t.Fatal(err)
	}
	lf.Packages[0].Path = simAbs // what ailang <= v0.32 wrote
	if err := lf.Save(aiDir); err != nil {
		t.Fatal(err)
	}

	old, err := LoadLockFile(aiDir)
	if err != nil {
		t.Fatalf("old absolute-path lock must still read: %v", err)
	}
	if _, err := NewPackageLoader(old, aiDir).ResolveImport("game/sim/markers"); err != nil {
		t.Errorf("old absolute-path lock must still resolve on the machine that wrote it: %v", err)
	}

	drift, err := CheckLock(aiDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(drift) == 0 {
		t.Fatal("an absolute path dep in ailang.lock must be reported as drift by lock --check")
	}
	joined := strings.Join(drift, "\n")
	if !strings.Contains(joined, "absolute") || !strings.Contains(joined, "../sim") {
		t.Errorf("drift must name the absolute path and the relative replacement, got:\n%s", joined)
	}

	fresh := writeLock(t, aiDir)
	if fresh.Packages[0].Path != "../sim" {
		t.Errorf("next lock must rewrite relative, got %q", fresh.Packages[0].Path)
	}
	if drift, _ := CheckLock(aiDir); len(drift) != 0 {
		t.Errorf("rewritten lock must be current: %v", drift)
	}
}

// When an old lock's absolute path no longer exists (CI clone, another HOME),
// the failure must say why and how to fix it — not just "not found".
func TestLoader_StaleAbsolutePathNamesFix(t *testing.T) {
	gone := filepath.Join(t.TempDir(), "gone", "sim")
	lf := &LockFile{Schema: LockFileSchema, Packages: []LockedPackage{{
		Name: "game/sim", Version: "0.1.0", ContentHash: "sha256:x", Source: "path", Path: gone,
	}}}
	_, err := NewPackageLoader(lf, t.TempDir()).packageDir(&lf.Packages[0])
	if err == nil {
		t.Fatal("missing path dep must error")
	}
	if !strings.Contains(err.Error(), "absolute") || !strings.Contains(err.Error(), "ailang lock") {
		t.Errorf("error must explain the stale absolute path and say to run 'ailang lock', got: %v", err)
	}
	err = lf.ValidateContentHashesFrom(t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "ailang lock") || !strings.Contains(err.Error(), "absolute") {
		t.Errorf("hash validation must carry the same fix-it, got: %v", err)
	}
}

// Content changes in a path dep are drift too.
func TestCheckLock_DetectsContentDrift(t *testing.T) {
	aiDir := buildTwoPackageRepo(t, t.TempDir())
	writeLock(t, aiDir)
	writeFile(t, filepath.Join(aiDir, "..", "sim", "markers.ail"), simMarkers+"\nexport pure func other() -> int = 2\n")
	drift, err := CheckLock(aiDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(drift) == 0 || !strings.Contains(strings.Join(drift, "\n"), "content_hash") {
		t.Errorf("edited path dep must be content_hash drift, got %v", drift)
	}
}

// A missing lock is not "current".
func TestCheckLock_MissingLockIsAnError(t *testing.T) {
	aiDir := buildTwoPackageRepo(t, t.TempDir())
	if _, err := CheckLock(aiDir); err == nil || !strings.Contains(err.Error(), "ailang lock") {
		t.Errorf("missing lock must error with a fix-it, got %v", err)
	}
}

// The content hash names files with forward slashes on every OS, so a lock
// written on Windows validates on Linux and vice versa.
func TestContentHash_SeparatorIndependent(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "sub", "x.ail"), "module a/sub/x\n")
	got, err := ContentHash(dir)
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.New()
	fmt.Fprintf(h, "file:%s\n", "sub/x.ail")
	h.Write([]byte("module a/sub/x\n"))
	if want := "sha256:" + hex.EncodeToString(h.Sum(nil)); got != want {
		t.Errorf("ContentHash must use slash-separated relative names: got %s want %s", got, want)
	}
}

func TestPortablePathDep_ForwardSlashes(t *testing.T) {
	root := filepath.Join("base", "ai")
	dep := filepath.Join("base", "libs", "sim")
	if got := portablePathDep(root, dep); got != "../libs/sim" {
		t.Errorf("portablePathDep = %q, want ../libs/sim", got)
	}
}
