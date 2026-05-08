package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeManifest writes a minimal ailang.toml for tests.
func writeNotifyTestManifest(t *testing.T, dir, name, version string) {
	t.Helper()
	content := "[package]\nname = \"" + name + "\"\nversion = \"" + version + "\"\nedition = \"1\"\n"
	if err := os.WriteFile(filepath.Join(dir, "ailang.toml"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

// writeNotifyTestLockFile writes a minimal ailang.lock (JSON format) for tests.
func writeNotifyTestLockFile(t *testing.T, dir, pkgName, version, interfaceHash string) {
	t.Helper()
	content := `{
  "schema": "ailang.lock/v1",
  "schema_version": "1",
  "generated_at": "2026-01-01T00:00:00Z",
  "generator": "test",
  "packages": [
    {
      "name": "` + pkgName + `",
      "version": "` + version + `",
      "content_hash": "sha256:aaaa",
      "interface_hash": "` + interfaceHash + `",
      "source": "path",
      "effects": [],
      "exports": []
    }
  ]
}
`
	if err := os.WriteFile(filepath.Join(dir, "ailang.lock"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

// TestNotifyUpgrade_FromPackageDir verifies that notify-upgrade --dry-run succeeds
// when run from inside the package directory itself.
func TestNotifyUpgrade_FromPackageDir(t *testing.T) {
	dir := t.TempDir()
	writeNotifyTestManifest(t, dir, "sunholo/myext", "0.1.1")

	orig, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(orig) })

	var err error
	out := captureStdout(t, func() {
		err = pkgNotifyUpgradeCommand([]string{"--dry-run", "sunholo/myext@0.1.1"})
	})

	if err != nil {
		t.Fatalf("expected no error from package dir, got: %v", err)
	}
	if !strings.Contains(out, `"to_interface_hash": "sha256:`) {
		t.Errorf("expected to_interface_hash sha256: in output, got:\n%s", out)
	}
}

// TestNotifyUpgrade_FromConsumerWithPathDep verifies that notify-upgrade --dry-run
// succeeds when run from a consumer workspace that has the package as a path dep.
func TestNotifyUpgrade_FromConsumerWithPathDep(t *testing.T) {
	// Extension package directory
	extDir := t.TempDir()
	writeNotifyTestManifest(t, extDir, "sunholo/myext", "0.1.1")

	// Consumer workspace with a path dep
	consumerDir := t.TempDir()
	tomlContent := "[package]\nname = \"myapp/src\"\nversion = \"1.0.0\"\nedition = \"1\"\n\n[dependencies]\n\"sunholo/myext\" = { path = \"" + filepath.ToSlash(extDir) + "\" }\n"
	if err := os.WriteFile(filepath.Join(consumerDir, "ailang.toml"), []byte(tomlContent), 0600); err != nil {
		t.Fatal(err)
	}
	writeNotifyTestLockFile(t, consumerDir, "sunholo/myext", "0.1.0", "sha256:oldoldhash")

	orig, _ := os.Getwd()
	if err := os.Chdir(consumerDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(orig) })

	var err error
	out := captureStdout(t, func() {
		err = pkgNotifyUpgradeCommand([]string{"--dry-run", "sunholo/myext@0.1.1"})
	})

	if err != nil {
		t.Fatalf("expected no error from consumer with path dep, got: %v", err)
	}
	if !strings.Contains(out, `"to_interface_hash": "sha256:`) {
		t.Errorf("expected to_interface_hash sha256: in output, got:\n%s", out)
	}
	if !strings.Contains(out, `"from_interface_hash": "sha256:oldoldhash"`) {
		t.Errorf("expected from_interface_hash from lockfile, got:\n%s", out)
	}
}

// TestNotifyUpgrade_MissingPackage verifies that a package not present in the
// manifest still produces a dry-run message (hashes empty for unknown packages).
func TestNotifyUpgrade_MissingPackage(t *testing.T) {
	dir := t.TempDir()
	writeNotifyTestManifest(t, dir, "myapp/src", "1.0.0")

	orig, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(orig) })

	var err error
	out := captureStdout(t, func() {
		err = pkgNotifyUpgradeCommand([]string{"--dry-run", "unknown/pkg@1.0.0"})
	})

	if err != nil {
		t.Fatalf("dry-run for unknown package should not error, got: %v", err)
	}
	if !strings.Contains(out, `"package"`) {
		t.Errorf("expected JSON output, got:\n%s", out)
	}
}
