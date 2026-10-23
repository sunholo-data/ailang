package eval_harness

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestMaintenanceOwnedPathsRejectInvalidUTF8(t *testing.T) {
	bad := string([]byte{'x', 0xff})
	if maintenanceOwnedPath(bad) == nil || maintenanceArtifactPath(bad) {
		t.Fatal("invalid UTF-8 could collide under JSON replacement encoding")
	}
	if err := maintenanceOwnedPath("notes/emoji-🧪-&.txt"); err != nil {
		t.Fatal(err)
	}
}

func maintenanceWrite(t *testing.T, root, name, body string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestMaintenanceCheckpointRetainsActualBrokenSourceAndDeletion(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	if err := os.Mkdir(source, 0700); err != nil {
		t.Fatal(err)
	}
	maintenanceWrite(t, source, "main.ail", "deliberately broken source")
	maintenanceWrite(t, source, "ailang.lock", "original lock")
	maintenanceWrite(t, source, "notes/handover.md", "current source fails parsing")
	maintenanceWrite(t, source, "logs/secret.log", "must not carry")
	cp, err := CaptureMaintenanceCheckpoint(source, filepath.Join(root, "checkpoint"), []string{"main.ail", "ailang.lock", "notes/handover.md", "deleted.ail"})
	if err != nil {
		t.Fatal(err)
	}
	if len(cp.Files) != 3 {
		t.Fatalf("files=%+v", cp.Files)
	}
	next := filepath.Join(root, "next")
	if err := RestoreMaintenanceCheckpoint(cp, next); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(next, "main.ail"))
	if err != nil || string(body) != "deliberately broken source" {
		t.Fatalf("healed source: %q %v", body, err)
	}
	for _, p := range []string{"deleted.ail", "logs/secret.log"} {
		if _, err := os.Stat(filepath.Join(next, p)); !os.IsNotExist(err) {
			t.Fatalf("unexpected carried %s: %v", p, err)
		}
	}
	if err := os.Remove(filepath.Join(next, "main.ail")); err != nil {
		t.Fatal(err)
	}
	maintenanceWrite(t, next, "renamed.ail", "broken and renamed")
	cp2, err := CaptureMaintenanceCheckpoint(next, filepath.Join(root, "checkpoint2"), []string{"main.ail", "renamed.ail", "ailang.lock", "notes/handover.md"})
	if err != nil {
		t.Fatal(err)
	}
	final := filepath.Join(root, "final")
	if err := RestoreMaintenanceCheckpoint(cp2, final); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(final, "main.ail")); !os.IsNotExist(err) {
		t.Fatal("deleted source restored")
	}
	if cp.Digest == cp2.Digest {
		t.Fatal("changed artifact retained same digest")
	}
}

func TestMaintenanceCheckpointDeterministicAndCreateOnly(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "source")
	if err := os.Mkdir(src, 0700); err != nil {
		t.Fatal(err)
	}
	maintenanceWrite(t, src, "a", "one")
	maintenanceWrite(t, src, "b", "two")
	a, err := CaptureMaintenanceCheckpoint(src, filepath.Join(root, "a"), []string{"b", "a"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := CaptureMaintenanceCheckpoint(src, filepath.Join(root, "b"), []string{"a", "b"})
	if err != nil {
		t.Fatal(err)
	}
	if a.Digest != b.Digest {
		t.Fatal("input order affected digest")
	}
	if _, err := CaptureMaintenanceCheckpoint(src, a.Root, []string{"a"}); err == nil {
		t.Fatal("overwrote checkpoint")
	}
	dst := filepath.Join(root, "existing")
	if err := os.Mkdir(dst, 0700); err != nil {
		t.Fatal(err)
	}
	if err := RestoreMaintenanceCheckpoint(a, dst); err == nil {
		t.Fatal("overwrote workspace")
	}
}

func TestMaintenanceCheckpointRejectsUnsafePaths(t *testing.T) {
	for _, p := range []string{"../secret", "/absolute", ".", "a/../b", "a\\b", "a//b", "a/", ""} {
		t.Run(strings.ReplaceAll(p, "/", "_"), func(t *testing.T) {
			root := t.TempDir()
			src := filepath.Join(root, "src")
			if err := os.Mkdir(src, 0700); err != nil {
				t.Fatal(err)
			}
			if _, err := CaptureMaintenanceCheckpoint(src, filepath.Join(root, "out"), []string{p}); err == nil {
				t.Fatalf("accepted %q", p)
			}
		})
	}
	root := t.TempDir()
	src := filepath.Join(root, "src")
	if err := os.Mkdir(src, 0700); err != nil {
		t.Fatal(err)
	}
	maintenanceWrite(t, src, "a", "one")
	if _, err := CaptureMaintenanceCheckpoint(src, filepath.Join(root, "dup"), []string{"a", "a"}); err == nil {
		t.Fatal("accepted duplicate")
	}
	if _, err := CaptureMaintenanceCheckpoint(src, filepath.Join(src, "nested"), []string{"a"}); err == nil {
		t.Fatal("checkpoint inside candidate")
	}
}

func TestMaintenanceCheckpointRejectsSymlinkAndTampering(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privilege unavailable in ordinary Windows tests")
	}
	root := t.TempDir()
	src := filepath.Join(root, "source")
	if err := os.Mkdir(src, 0700); err != nil {
		t.Fatal(err)
	}
	maintenanceWrite(t, root, "outside/secret", "oracle")
	if err := os.Symlink(filepath.Join(root, "outside"), filepath.Join(src, "linked")); err != nil {
		t.Fatal(err)
	}
	if _, err := CaptureMaintenanceCheckpoint(src, filepath.Join(root, "bad"), []string{"linked/secret"}); err == nil {
		t.Fatal("followed symlink ancestor")
	}
	if _, err := CaptureMaintenanceCheckpoint(src, filepath.Join(root, "dir"), []string{"linked"}); err == nil {
		t.Fatal("followed symlink file")
	}
	maintenanceWrite(t, src, "main", "candidate")
	cp, err := CaptureMaintenanceCheckpoint(src, filepath.Join(root, "cp"), []string{"main"})
	if err != nil {
		t.Fatal(err)
	}
	maintenanceWrite(t, filepath.Join(cp.Root, "files"), "main", "oracle substituted")
	if err := RestoreMaintenanceCheckpoint(cp, filepath.Join(root, "next")); err == nil {
		t.Fatal("restored tampered source")
	}
}

func TestMaintenanceCheckpointRejectsDirectoryAndSourceRootLink(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "source")
	if err := os.Mkdir(src, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(src, "owned"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := CaptureMaintenanceCheckpoint(src, filepath.Join(root, "directory"), []string{"owned"}); err == nil {
		t.Fatal("directory accepted as source file")
	}
	if runtime.GOOS == "windows" {
		return
	}
	linked := filepath.Join(root, "linked")
	if err := os.Symlink(src, linked); err != nil {
		t.Fatal(err)
	}
	if _, err := CaptureMaintenanceCheckpoint(linked, filepath.Join(root, "root-link"), []string{}); err == nil {
		t.Fatal("symlink source root accepted")
	}
}
