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

func TestCreateTarball_IncludesAssets(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ManifestFile), []byte(`[package]
name = "test/pkg"
version = "0.1.0"
edition = "1"

[assets]
files = ["foo.txt", "scripts/bar.mjs"]
`), 0644)
	os.WriteFile(filepath.Join(dir, "core.ail"), []byte("module test/pkg/core\n"), 0644)
	os.MkdirAll(filepath.Join(dir, AssetsDir, "scripts"), 0755)
	os.WriteFile(filepath.Join(dir, AssetsDir, "foo.txt"), []byte("hello\n"), 0644)
	os.WriteFile(filepath.Join(dir, AssetsDir, "scripts", "bar.mjs"), []byte("console.log('x')\n"), 0644)

	data, err := CreateTarball(dir)
	if err != nil {
		t.Fatalf("CreateTarball: %v", err)
	}

	destDir := t.TempDir()
	if err := ExtractTarball(data, destDir); err != nil {
		t.Fatalf("ExtractTarball: %v", err)
	}

	for _, f := range []string{"assets/foo.txt", "assets/scripts/bar.mjs"} {
		if _, err := os.Stat(filepath.Join(destDir, f)); err != nil {
			t.Errorf("expected asset %s to be bundled, got: %v", f, err)
		}
	}
}

func TestCreateTarball_AssetHashDeterministic(t *testing.T) {
	makeDir := func() string {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, ManifestFile), []byte(`[package]
name = "test/pkg"
version = "0.1.0"
edition = "1"
`), 0644)
		os.WriteFile(filepath.Join(dir, "core.ail"), []byte("module test/pkg/core\n"), 0644)
		os.MkdirAll(filepath.Join(dir, AssetsDir), 0755)
		os.WriteFile(filepath.Join(dir, AssetsDir, "foo.txt"), []byte("hello\n"), 0644)
		os.WriteFile(filepath.Join(dir, AssetsDir, "bar.json"), []byte(`{"x":1}`+"\n"), 0644)
		return dir
	}

	d1, err := CreateTarball(makeDir())
	if err != nil {
		t.Fatalf("CreateTarball #1: %v", err)
	}
	d2, err := CreateTarball(makeDir())
	if err != nil {
		t.Fatalf("CreateTarball #2: %v", err)
	}
	if TarballHash(d1) != TarballHash(d2) {
		t.Errorf("tarball with assets not deterministic: %s != %s", TarballHash(d1), TarballHash(d2))
	}
}

func TestVerifyDeclaredAssets(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, AssetsDir), 0755)
	os.WriteFile(filepath.Join(dir, AssetsDir, "foo.txt"), []byte("x"), 0644)

	// Pass: declared file exists
	if err := VerifyDeclaredAssets(dir, &PackageManifest{Assets: AssetConfig{Files: []string{"foo.txt"}}}); err != nil {
		t.Errorf("expected pass for present asset, got: %v", err)
	}

	// Fail: declared file missing
	if err := VerifyDeclaredAssets(dir, &PackageManifest{Assets: AssetConfig{Files: []string{"missing.txt"}}}); err == nil {
		t.Errorf("expected error for missing declared asset")
	}

	// Pass: nil manifest, no assets
	if err := VerifyDeclaredAssets(dir, nil); err != nil {
		t.Errorf("expected pass for nil manifest, got: %v", err)
	}

	// Pass: no declared assets
	if err := VerifyDeclaredAssets(dir, &PackageManifest{}); err != nil {
		t.Errorf("expected pass for empty Assets, got: %v", err)
	}
}

func TestManifest_AssetValidation(t *testing.T) {
	cases := []struct {
		name    string
		assets  []string
		wantErr bool
	}{
		{"empty list ok", nil, false},
		{"valid relative path", []string{"foo.txt", "scripts/bar.mjs"}, false},
		{"empty entry rejected", []string{""}, true},
		{"absolute path rejected", []string{"/etc/passwd"}, true},
		{"parent traversal rejected", []string{"../etc/passwd"}, true},
		{"nested traversal rejected", []string{"a/../../b"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := &PackageManifest{
				Package: PackageInfo{Name: "test/pkg", Version: "0.1.0", Edition: "1"},
				Exports: ExportConfig{Modules: []string{"test/pkg/core"}},
				Assets:  AssetConfig{Files: tc.assets},
			}
			err := m.Validate()
			if tc.wantErr && err == nil {
				t.Errorf("expected validation error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected validation error: %v", err)
			}
		})
	}
}

func TestManifest_SmokeTimeoutValidation(t *testing.T) {
	cases := []struct {
		name           string
		timeoutSeconds int
		wantErr        bool
	}{
		{"unset (zero) ok", 0, false},
		{"valid low boundary", 1, false},
		{"valid mid", 30, false},
		{"valid high boundary", 300, false},
		{"negative rejected", -1, true},
		{"too large rejected", 301, true},
		{"absurdly large rejected", 99999, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := &PackageManifest{
				Package: PackageInfo{Name: "test/pkg", Version: "0.1.0", Edition: "1"},
				Exports: ExportConfig{Modules: []string{"test/pkg/core"}},
				Smoke:   SmokeConfig{TimeoutSeconds: tc.timeoutSeconds},
			}
			err := m.Validate()
			if tc.wantErr && err == nil {
				t.Errorf("expected validation error for %d, got nil", tc.timeoutSeconds)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected validation error for %d: %v", tc.timeoutSeconds, err)
			}
		})
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
