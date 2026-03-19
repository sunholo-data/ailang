package pkg

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLockFile_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()

	packages := []LockedPackage{
		{
			Name:        "sunholo/json",
			Version:     "0.3.1",
			ContentHash: "sha256:abc123",
			Source:      "path",
			Path:        "../json",
			Effects:     []string{"IO"},
			Exports:     []string{"parseJson", "encodeJson"},
		},
		{
			Name:        "acme/xml",
			Version:     "0.1.0",
			ContentHash: "sha256:def456",
			Source:      "path",
			Effects:     []string{},
			Exports:     []string{"parseXml"},
		},
	}

	lf := NewLockFile(packages, "ailang lock v1.0.0")
	if err := lf.Save(dir); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load it back
	loaded, err := LoadLockFile(dir)
	if err != nil {
		t.Fatalf("LoadLockFile failed: %v", err)
	}

	if len(loaded.Packages) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(loaded.Packages))
	}

	// Should be sorted by name (acme/xml before sunholo/json)
	if loaded.Packages[0].Name != "acme/xml" {
		t.Errorf("first package = %q, want acme/xml (sorted)", loaded.Packages[0].Name)
	}
	if loaded.Packages[1].Name != "sunholo/json" {
		t.Errorf("second package = %q, want sunholo/json (sorted)", loaded.Packages[1].Name)
	}
}

func TestLockFile_Deterministic(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	packages := []LockedPackage{
		{Name: "z/pkg", Version: "1.0", ContentHash: "sha256:111", Source: "path", Effects: []string{"Net", "IO"}, Exports: []string{"b", "a"}},
		{Name: "a/pkg", Version: "2.0", ContentHash: "sha256:222", Source: "path", Effects: []string{}, Exports: []string{}},
	}

	lf1 := NewLockFile(packages, "test")
	lf1.Save(dir1)

	lf2 := NewLockFile(packages, "test")
	lf2.GeneratedAt = lf1.GeneratedAt // use same timestamp for determinism comparison
	lf2.Save(dir2)

	data1, _ := os.ReadFile(filepath.Join(dir1, LockFileName))
	data2, _ := os.ReadFile(filepath.Join(dir2, LockFileName))

	if string(data1) != string(data2) {
		t.Error("lock files should be identical for same input")
	}

	// Effects should be sorted within packages
	var lf LockFile
	json.Unmarshal(data1, &lf)
	pkg := lf.Packages[1] // z/pkg after sorting
	if pkg.Effects[0] != "IO" || pkg.Effects[1] != "Net" {
		t.Errorf("effects should be sorted: got %v", pkg.Effects)
	}
	if pkg.Exports[0] != "a" || pkg.Exports[1] != "b" {
		t.Errorf("exports should be sorted: got %v", pkg.Exports)
	}
}

func TestLockFile_ValidateDuplicate(t *testing.T) {
	lf := &LockFile{
		Schema:  LockFileSchema,
		Version: "1.0.0",
		Packages: []LockedPackage{
			{Name: "a/b", ContentHash: "sha256:1", Source: "path"},
			{Name: "a/b", ContentHash: "sha256:2", Source: "path"},
		},
	}
	if err := lf.Validate(); err == nil {
		t.Error("expected error for duplicate package names")
	}
}

func TestLockFile_ValidateMissingHash(t *testing.T) {
	lf := &LockFile{
		Schema:  LockFileSchema,
		Version: "1.0.0",
		Packages: []LockedPackage{
			{Name: "a/b", ContentHash: "", Source: "path"},
		},
	}
	if err := lf.Validate(); err == nil {
		t.Error("expected error for missing content_hash")
	}
}

func TestLockFile_FindPackage(t *testing.T) {
	lf := &LockFile{
		Packages: []LockedPackage{
			{Name: "a/one", ContentHash: "sha256:1", Source: "path"},
			{Name: "b/two", ContentHash: "sha256:2", Source: "path"},
		},
	}

	p, found := lf.FindPackage("a/one")
	if !found || p.Name != "a/one" {
		t.Error("should find a/one")
	}

	_, found = lf.FindPackage("c/three")
	if found {
		t.Error("should not find c/three")
	}
}

func TestLockFile_ValidateAgainstManifest(t *testing.T) {
	lf := &LockFile{
		Packages: []LockedPackage{
			{Name: "sunholo/json", ContentHash: "sha256:1", Source: "path"},
		},
	}

	// Manifest matches lock file — should pass
	m1 := &PackageManifest{
		Dependencies: map[string]Dependency{
			"sunholo/json": {Version: "0.3.1"},
		},
	}
	if err := lf.ValidateAgainstManifest(m1); err != nil {
		t.Errorf("should pass: %v", err)
	}

	// Manifest has extra dep — should fail
	m2 := &PackageManifest{
		Dependencies: map[string]Dependency{
			"sunholo/json": {Version: "0.3.1"},
			"sunholo/xml":  {Version: "0.1.0"},
		},
	}
	if err := lf.ValidateAgainstManifest(m2); err == nil {
		t.Error("should fail when manifest has dep not in lock file")
	}
}
