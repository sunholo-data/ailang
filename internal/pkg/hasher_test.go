package pkg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContentHash_Deterministic(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "src"), 0755)
	os.WriteFile(filepath.Join(dir, "src", "a.ail"), []byte("module test/a\n"), 0644)
	os.WriteFile(filepath.Join(dir, "src", "b.ail"), []byte("module test/b\n"), 0644)

	h1, err := ContentHash(dir)
	if err != nil {
		t.Fatalf("ContentHash failed: %v", err)
	}
	h2, err := ContentHash(dir)
	if err != nil {
		t.Fatalf("ContentHash failed: %v", err)
	}

	if h1 != h2 {
		t.Errorf("hashes differ: %s != %s", h1, h2)
	}
	if !strings.HasPrefix(h1, "sha256:") {
		t.Errorf("hash should have sha256: prefix, got %s", h1)
	}
}

func TestContentHash_DifferentContent(t *testing.T) {
	dir1 := t.TempDir()
	os.WriteFile(filepath.Join(dir1, "a.ail"), []byte("version 1"), 0644)

	dir2 := t.TempDir()
	os.WriteFile(filepath.Join(dir2, "a.ail"), []byte("version 2"), 0644)

	h1, _ := ContentHash(dir1)
	h2, _ := ContentHash(dir2)

	if h1 == h2 {
		t.Error("different content should produce different hashes")
	}
}

func TestContentHash_DifferentFilenames(t *testing.T) {
	dir1 := t.TempDir()
	os.WriteFile(filepath.Join(dir1, "a.ail"), []byte("content"), 0644)

	dir2 := t.TempDir()
	os.WriteFile(filepath.Join(dir2, "b.ail"), []byte("content"), 0644)

	h1, _ := ContentHash(dir1)
	h2, _ := ContentHash(dir2)

	if h1 == h2 {
		t.Error("different filenames should produce different hashes")
	}
}

func TestContentHash_IgnoresNonAil(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.ail"), []byte("content"), 0644)
	os.WriteFile(filepath.Join(dir, "readme.md"), []byte("docs"), 0644)

	h1, _ := ContentHash(dir)

	// Add another non-.ail file — hash should not change
	os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("more"), 0644)
	h2, _ := ContentHash(dir)

	if h1 != h2 {
		t.Error("non-.ail files should not affect hash")
	}
}

func TestContentHash_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	h, err := ContentHash(dir)
	if err != nil {
		t.Fatalf("ContentHash failed: %v", err)
	}
	if !strings.HasPrefix(h, "sha256:") {
		t.Errorf("empty dir should still produce a valid hash, got %s", h)
	}
}
