package pkg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateTarball_Roundtrip(t *testing.T) {
	// Create a mock package
	srcDir := t.TempDir()
	os.WriteFile(filepath.Join(srcDir, ManifestFile), []byte(`[package]
name = "test/pkg"
version = "0.1.0"
edition = "1"
`), 0644)
	os.WriteFile(filepath.Join(srcDir, "core.ail"), []byte("module test/pkg/core\n"), 0644)
	os.MkdirAll(filepath.Join(srcDir, "src"), 0755)
	os.WriteFile(filepath.Join(srcDir, "src", "helper.ail"), []byte("module test/pkg/helper\n"), 0644)
	os.WriteFile(filepath.Join(srcDir, "AGENT.md"), []byte("# test/pkg\n"), 0644)
	// This should be excluded
	os.WriteFile(filepath.Join(srcDir, "readme.md"), []byte("not included\n"), 0644)

	data, err := CreateTarball(srcDir)
	if err != nil {
		t.Fatalf("CreateTarball: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("tarball should not be empty")
	}

	// Extract to new dir
	destDir := t.TempDir()
	if err := ExtractTarball(data, destDir); err != nil {
		t.Fatalf("ExtractTarball: %v", err)
	}

	// Verify files exist
	for _, f := range []string{ManifestFile, "core.ail", "src/helper.ail", "AGENT.md"} {
		if _, err := os.Stat(filepath.Join(destDir, f)); err != nil {
			t.Errorf("expected file %s to exist after extraction", f)
		}
	}

	// readme.md should NOT be extracted (not included)
	if _, err := os.Stat(filepath.Join(destDir, "readme.md")); err == nil {
		t.Error("readme.md should not be in tarball")
	}
}

func TestCreateTarball_Deterministic(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ManifestFile), []byte(`[package]
name = "test/pkg"
version = "0.1.0"
edition = "1"
`), 0644)
	os.WriteFile(filepath.Join(dir, "a.ail"), []byte("module test/pkg/a\n"), 0644)
	os.WriteFile(filepath.Join(dir, "b.ail"), []byte("module test/pkg/b\n"), 0644)

	data1, _ := CreateTarball(dir)
	data2, _ := CreateTarball(dir)

	h1 := TarballHash(data1)
	h2 := TarballHash(data2)

	if h1 != h2 {
		t.Errorf("tarball not deterministic: %s != %s", h1, h2)
	}
}

func TestCreateTarball_ExcludesGitAndTests(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ManifestFile), []byte(`[package]
name = "test/pkg"
version = "0.1.0"
edition = "1"
`), 0644)
	os.WriteFile(filepath.Join(dir, "core.ail"), []byte("module test/pkg/core\n"), 0644)

	// Create .git and tests dirs with files
	os.MkdirAll(filepath.Join(dir, ".git", "objects"), 0755)
	os.WriteFile(filepath.Join(dir, ".git", "HEAD"), []byte("ref: refs/heads/main\n"), 0644)
	os.MkdirAll(filepath.Join(dir, "tests"), 0755)
	os.WriteFile(filepath.Join(dir, "tests", "core_test.ail"), []byte("test\n"), 0644)

	data, err := CreateTarball(dir)
	if err != nil {
		t.Fatalf("CreateTarball: %v", err)
	}

	// Extract and verify exclusions
	destDir := t.TempDir()
	ExtractTarball(data, destDir)

	if _, err := os.Stat(filepath.Join(destDir, ".git")); err == nil {
		t.Error(".git should be excluded from tarball")
	}
	if _, err := os.Stat(filepath.Join(destDir, "tests")); err == nil {
		t.Error("tests/ should be excluded from tarball")
	}
}

func TestTarballHash(t *testing.T) {
	h := TarballHash([]byte("test data"))
	if !strings.HasPrefix(h, "sha256:") {
		t.Errorf("should have sha256: prefix, got %s", h)
	}
	if len(h) != 7+64 { // "sha256:" + 64 hex chars
		t.Errorf("unexpected hash length: %d", len(h))
	}
}
