package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// minimalManifestTOML builds a minimal ailang.toml with an [extensions] block.
func minimalManifestTOML(extras string) string {
	return `[package]
name = "vendor/myapp"
version = "0.1.0"
edition = "2025"
` + extras
}

// twoPackageLockJSON is a synthetic ailang.lock with the two test extension packages.
const twoPackageLockJSON = `{
  "schema": "ailang.lock/v1",
  "schema_version": "1.0.0",
  "generated_at": "2026-01-01T00:00:00Z",
  "generator": "test",
  "packages": [
    {
      "name": "motoko-ext-compaction",
      "version": "0.2.0",
      "content_hash": "sha256:abc123",
      "source": "registry",
      "effects": [],
      "exports": ["motoko-ext-compaction/register"]
    },
    {
      "name": "motoko-ext-exa-search",
      "version": "0.4.1",
      "content_hash": "sha256:def456",
      "source": "registry",
      "effects": [],
      "exports": ["motoko-ext-exa-search/register"]
    }
  ]
}`

// writeFixture writes ailang.toml and ailang.lock into a temp dir.
func writeFixture(t *testing.T, tomlContent, lockContent string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ailang.toml"), []byte(tomlContent), 0644); err != nil {
		t.Fatalf("write ailang.toml: %v", err)
	}
	if lockContent != "" {
		if err := os.WriteFile(filepath.Join(dir, "ailang.lock"), []byte(lockContent), 0644); err != nil {
			t.Fatalf("write ailang.lock: %v", err)
		}
	}
	return dir
}

func TestExtRegistryGen_TwoPackages(t *testing.T) {
	toml := minimalManifestTOML(`
[extensions]
packages      = ["motoko-ext-compaction@0.2.0", "motoko-ext-exa-search@0.4.1"]
config_import = "src/core/config.RuntimeConfig"
hooks_import  = "src/core/ext/types.ExtensionHooks"
output        = "src/core/ext/registry_generated.ail"
`)
	dir := writeFixture(t, toml, twoPackageLockJSON)

	outPath := filepath.Join(dir, "src", "core", "ext", "registry_generated.ail")
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if err := extRegistryGenCommand([]string{
		"-config", filepath.Join(dir, "ailang.toml"),
		"-output", outPath,
	}); err != nil {
		t.Fatalf("extRegistryGenCommand: %v", err)
	}

	got, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	goldenPath := filepath.Join("testdata", "ext_registry_gen", "two_packages.ail.golden")
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}

	if string(got) != string(want) {
		t.Errorf("output does not match golden file %s\n\nGOT:\n%s\n\nWANT:\n%s", goldenPath, got, want)
	}
}

func TestExtRegistryGen_DryRun(t *testing.T) {
	toml := minimalManifestTOML(`
[extensions]
packages      = ["motoko-ext-compaction@0.2.0", "motoko-ext-exa-search@0.4.1"]
config_import = "src/core/config.RuntimeConfig"
hooks_import  = "src/core/ext/types.ExtensionHooks"
`)
	dir := writeFixture(t, toml, twoPackageLockJSON)

	// Redirect stdout to capture dry-run output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := extRegistryGenCommand([]string{
		"-config", filepath.Join(dir, "ailang.toml"),
		"-dry-run",
	})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("extRegistryGenCommand: %v", err)
	}

	var buf strings.Builder
	b := make([]byte, 4096)
	for {
		n, rerr := r.Read(b)
		buf.Write(b[:n])
		if rerr != nil {
			break
		}
	}
	out := buf.String()

	if !strings.Contains(out, "module") {
		t.Errorf("dry-run output missing 'module' declaration, got:\n%s", out)
	}
	if !strings.Contains(out, "register_compaction") {
		t.Errorf("dry-run output missing 'register_compaction', got:\n%s", out)
	}

	// Verify no file was written
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".ail") {
			t.Errorf("dry-run wrote unexpected file: %s", e.Name())
		}
	}
}

func TestExtRegistryGen_PackageNotLocked(t *testing.T) {
	toml := minimalManifestTOML(`
[extensions]
packages      = ["motoko-ext-unlisted@1.0.0"]
config_import = "src/core/config.RuntimeConfig"
hooks_import  = "src/core/ext/types.ExtensionHooks"
`)
	dir := writeFixture(t, toml, twoPackageLockJSON)

	err := extRegistryGenCommand([]string{
		"-config", filepath.Join(dir, "ailang.toml"),
		"-dry-run",
	})
	if err == nil {
		t.Fatal("expected error for unlisted package, got nil")
	}
	if !strings.Contains(err.Error(), "motoko-ext-unlisted") {
		t.Errorf("error should name the missing package, got: %v", err)
	}
	if !strings.Contains(err.Error(), "ailang.lock") {
		t.Errorf("error should mention ailang.lock, got: %v", err)
	}
}

func TestExtRegistryGen_MissingConfigImport(t *testing.T) {
	// Missing config_import triggers manifest Validate() before we reach the generator.
	toml := minimalManifestTOML(`
[extensions]
packages     = ["motoko-ext-compaction@0.2.0"]
hooks_import = "src/core/ext/types.ExtensionHooks"
`)
	dir := writeFixture(t, toml, twoPackageLockJSON)

	err := extRegistryGenCommand([]string{
		"-config", filepath.Join(dir, "ailang.toml"),
		"-dry-run",
	})
	if err == nil {
		t.Fatal("expected error for missing config_import, got nil")
	}
	if !strings.Contains(err.Error(), "config_import") {
		t.Errorf("error should mention config_import, got: %v", err)
	}
}

func TestExtRegistryGen_EmptyPackages(t *testing.T) {
	toml := minimalManifestTOML(`
[extensions]
packages = []
`)
	dir := writeFixture(t, toml, twoPackageLockJSON)

	// Should succeed with no output file written
	if err := extRegistryGenCommand([]string{
		"-config", filepath.Join(dir, "ailang.toml"),
	}); err != nil {
		t.Fatalf("extRegistryGenCommand: %v", err)
	}

	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".ail") {
			t.Errorf("empty packages wrote unexpected .ail file: %s", e.Name())
		}
	}
}

func TestDeriveShortName(t *testing.T) {
	cases := []struct{ input, want string }{
		{"motoko-ext-exa-search@0.4.1", "exa_search"},
		{"motoko-ext-mcp", "mcp"},
		{"motoko-ext-compaction@0.2.0", "compaction"},
		{"my-tool", "my_tool"},
		{"simple", "simple"},
	}
	for _, tc := range cases {
		got := deriveShortName(tc.input)
		if got != tc.want {
			t.Errorf("deriveShortName(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
