package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTryLoadPackageResolver_BrokenManifestSurfacesParseError(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "ailang.toml")
	manifest := `[package]
name = "sunholo/broken"
version = "0.1.0"
edition = "1"

[dependencies]
"sunholo/duplicate" = "0.1.0"
"sunholo/duplicate" = "0.2.0"
`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	_, err := tryLoadPackageResolver(dir)
	if err == nil {
		t.Fatal("broken manifest unexpectedly loaded")
	}
	message := err.Error()
	if !strings.Contains(message, manifestPath) {
		t.Fatalf("error does not name manifest path %q: %s", manifestPath, message)
	}
	if !strings.Contains(message, "already been defined") || !strings.Contains(message, "sunholo/duplicate") {
		t.Fatalf("error does not carry duplicate-key TOML detail: %s", message)
	}
}

func TestTryLoadPackageResolver_NoManifestUsesLegacyResolution(t *testing.T) {
	dir := t.TempDir()
	resolver, err := tryLoadPackageResolver(dir)
	if err != nil || resolver != nil {
		t.Fatalf("no manifest: resolver = %T, error = %v; want nil, nil", resolver, err)
	}

	sourcePath := filepath.Join(dir, "main.ail")
	source := "module main\n\npure func answer() -> int = 42\n"
	if err := os.WriteFile(sourcePath, []byte(source), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := Run(Config{Mode: ModeCheck, PackageDir: dir}, Source{Filename: sourcePath}); err != nil {
		t.Fatalf("legacy bare-project pipeline failed: %v", err)
	}
}
