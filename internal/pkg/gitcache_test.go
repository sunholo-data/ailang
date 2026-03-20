package pkg

import (
	"strings"
	"testing"
)

func TestGitCache_CacheDir_Deterministic(t *testing.T) {
	cache := &GitCache{baseDir: "/tmp/test-cache"}

	dir1 := cache.CacheDir("https://github.com/sunholo-data/ailang-packages")
	dir2 := cache.CacheDir("https://github.com/sunholo-data/ailang-packages")

	if dir1 != dir2 {
		t.Errorf("cache dir not deterministic: %s != %s", dir1, dir2)
	}
	if !strings.HasPrefix(dir1, "/tmp/test-cache/") {
		t.Errorf("cache dir should be under base: %s", dir1)
	}
}

func TestGitCache_CacheDir_DifferentURLs(t *testing.T) {
	cache := &GitCache{baseDir: "/tmp/test-cache"}

	dir1 := cache.CacheDir("https://github.com/sunholo-data/ailang-packages")
	dir2 := cache.CacheDir("https://github.com/other-org/other-repo")

	if dir1 == dir2 {
		t.Error("different URLs should have different cache dirs")
	}
}

func TestGitCache_Resolve_RequiresTagOrRev(t *testing.T) {
	cache := &GitCache{baseDir: t.TempDir()}

	_, _, err := cache.Resolve("https://example.com/repo", "", "", "")
	if err == nil {
		t.Fatal("should require tag or rev")
	}
	if !strings.Contains(err.Error(), "must specify tag or rev") {
		t.Errorf("error should mention tag or rev, got: %v", err)
	}
}

// Integration test — requires git and network access
func TestGitCache_Resolve_RealRepo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	cache := &GitCache{baseDir: t.TempDir()}

	// Use the actual ailang-packages repo
	localPath, rev, err := cache.Resolve(
		"https://github.com/sunholo-data/ailang-packages",
		"main", "", "packages/auth",
	)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if localPath == "" {
		t.Error("localPath should not be empty")
	}
	if rev == "" {
		t.Error("resolved rev should not be empty")
	}
	if len(rev) != 40 {
		t.Errorf("resolved rev should be 40-char hash, got %d chars: %s", len(rev), rev)
	}

	// Should be able to load manifest from the resolved path
	m, err := LoadManifest(localPath)
	if err != nil {
		t.Fatalf("LoadManifest from git cache: %v", err)
	}
	if m.Package.Name != "sunholo/auth" {
		t.Errorf("package name = %q, want sunholo/auth", m.Package.Name)
	}
}
