package pkg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadManifest_Valid(t *testing.T) {
	dir := t.TempDir()
	content := `
[package]
name = "sunholo/docparse"
version = "0.3.0"
edition = "1"
description = "Parse documents"
license = "MIT"

[exports]
modules = ["sunholo/docparse/parser", "sunholo/docparse/types"]

[dependencies]
"sunholo/json" = "0.3.1"
"shared/utils" = { path = "../utils" }

[effects]
max = ["IO", "FS"]

[metadata]
tags = ["parsing"]

[stability]
level = "experimental"
`
	os.WriteFile(filepath.Join(dir, ManifestFile), []byte(content), 0644)

	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest failed: %v", err)
	}

	if m.Package.Name != "sunholo/docparse" {
		t.Errorf("name = %q, want sunholo/docparse", m.Package.Name)
	}
	if m.Package.Version != "0.3.0" {
		t.Errorf("version = %q, want 0.3.0", m.Package.Version)
	}
	if len(m.Exports.Modules) != 2 {
		t.Errorf("exports = %d, want 2", len(m.Exports.Modules))
	}
	if len(m.Dependencies) != 2 {
		t.Errorf("dependencies = %d, want 2", len(m.Dependencies))
	}

	// Check version dep
	jsonDep := m.Dependencies["sunholo/json"]
	if jsonDep.Version != "0.3.1" {
		t.Errorf("json dep version = %q, want 0.3.1", jsonDep.Version)
	}

	// Check path dep
	utilsDep := m.Dependencies["shared/utils"]
	if utilsDep.Path != "../utils" {
		t.Errorf("utils dep path = %q, want ../utils", utilsDep.Path)
	}

	if len(m.Effects.Max) != 2 {
		t.Errorf("effects = %d, want 2", len(m.Effects.Max))
	}
	if m.Stability.Level != "experimental" {
		t.Errorf("stability = %q, want experimental", m.Stability.Level)
	}
}

func TestLoadManifest_MissingName(t *testing.T) {
	dir := t.TempDir()
	content := `
[package]
version = "0.1.0"
edition = "1"
`
	os.WriteFile(filepath.Join(dir, ManifestFile), []byte(content), 0644)

	_, err := LoadManifest(dir)
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestLoadManifest_BadNameFormat(t *testing.T) {
	dir := t.TempDir()
	content := `
[package]
name = "justname"
version = "0.1.0"
edition = "1"
`
	os.WriteFile(filepath.Join(dir, ManifestFile), []byte(content), 0644)

	_, err := LoadManifest(dir)
	if err == nil {
		t.Fatal("expected error for non-vendor/name format")
	}
}

func TestLoadManifest_BadExportPrefix(t *testing.T) {
	dir := t.TempDir()
	content := `
[package]
name = "sunholo/json"
version = "0.1.0"
edition = "1"

[exports]
modules = ["other/package/mod"]
`
	os.WriteFile(filepath.Join(dir, ManifestFile), []byte(content), 0644)

	_, err := LoadManifest(dir)
	if err == nil {
		t.Fatal("expected error for export prefix mismatch")
	}
}

func TestLoadManifest_BadStability(t *testing.T) {
	dir := t.TempDir()
	content := `
[package]
name = "sunholo/json"
version = "0.1.0"
edition = "1"

[stability]
level = "invalid"
`
	os.WriteFile(filepath.Join(dir, ManifestFile), []byte(content), 0644)

	_, err := LoadManifest(dir)
	if err == nil {
		t.Fatal("expected error for invalid stability level")
	}
}

func TestInitManifest(t *testing.T) {
	dir := t.TempDir()

	if err := InitManifest(dir, "sunholo/mylib", "0.9.9"); err != nil {
		t.Fatalf("InitManifest failed: %v", err)
	}

	// Should be loadable
	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest after init failed: %v", err)
	}
	if m.Package.Name != "sunholo/mylib" {
		t.Errorf("name = %q, want sunholo/mylib", m.Package.Name)
	}
	if m.Package.Version != "0.1.0" {
		t.Errorf("version = %q, want 0.1.0", m.Package.Version)
	}
	if m.Package.AILANG != ">=0.9.9" {
		t.Errorf("ailang = %q, want >=0.9.9", m.Package.AILANG)
	}
}

func TestInitManifest_AlreadyExists(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ManifestFile), []byte("existing"), 0644)

	err := InitManifest(dir, "sunholo/mylib", "0.9.9")
	if err == nil {
		t.Fatal("expected error when manifest already exists")
	}
}

func TestFindManifest(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "src", "deep")
	os.MkdirAll(sub, 0755)
	os.WriteFile(filepath.Join(root, ManifestFile), []byte(`
[package]
name = "test/pkg"
version = "0.1.0"
edition = "1"
`), 0644)

	found := FindManifest(sub)
	if found != root {
		t.Errorf("FindManifest = %q, want %q", found, root)
	}
}

func TestFindManifest_NotFound(t *testing.T) {
	dir := t.TempDir()
	found := FindManifest(dir)
	if found != "" {
		t.Errorf("FindManifest = %q, want empty", found)
	}
}

func TestIsPathDep(t *testing.T) {
	m := &PackageManifest{
		Dependencies: map[string]Dependency{
			"sunholo/json": {Version: "0.3.1"},
			"shared/utils": {Path: "../utils"},
		},
	}

	if m.IsPathDep("sunholo/json") {
		t.Error("version dep should not be path dep")
	}
	if !m.IsPathDep("shared/utils") {
		t.Error("path dep should be path dep")
	}
	if m.IsPathDep("nonexistent") {
		t.Error("missing dep should not be path dep")
	}
}

func TestLoadManifest_GitDep(t *testing.T) {
	dir := t.TempDir()
	content := `
[package]
name = "test/app"
version = "0.1.0"
edition = "1"

[dependencies]
"sunholo/auth" = { git = "https://github.com/sunholo-data/ailang-packages", subdir = "packages/auth", tag = "auth-v0.1.0" }
`
	os.WriteFile(filepath.Join(dir, ManifestFile), []byte(content), 0644)

	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest failed: %v", err)
	}

	dep := m.Dependencies["sunholo/auth"]
	if dep.Git != "https://github.com/sunholo-data/ailang-packages" {
		t.Errorf("git = %q", dep.Git)
	}
	if dep.Tag != "auth-v0.1.0" {
		t.Errorf("tag = %q", dep.Tag)
	}
	if dep.Subdir != "packages/auth" {
		t.Errorf("subdir = %q", dep.Subdir)
	}
	if !m.IsGitDep("sunholo/auth") {
		t.Error("should be git dep")
	}
}

func TestLoadManifest_GitDepRequiresTagOrRev(t *testing.T) {
	dir := t.TempDir()
	content := `
[package]
name = "test/app"
version = "0.1.0"
edition = "1"

[dependencies]
"sunholo/auth" = { git = "https://github.com/example/repo" }
`
	os.WriteFile(filepath.Join(dir, ManifestFile), []byte(content), 0644)

	_, err := LoadManifest(dir)
	if err == nil {
		t.Fatal("expected error for git dep without tag or rev")
	}
}

func TestLoadManifest_GitAndPathConflict(t *testing.T) {
	dir := t.TempDir()
	content := `
[package]
name = "test/app"
version = "0.1.0"
edition = "1"

[dependencies]
"sunholo/auth" = { git = "https://example.com/repo", path = "../local", tag = "v1" }
`
	os.WriteFile(filepath.Join(dir, ManifestFile), []byte(content), 0644)

	_, err := LoadManifest(dir)
	if err == nil {
		t.Fatal("expected error for dep with both path and git")
	}
}

func TestLoadManifest_ModulePrefix_Valid(t *testing.T) {
	dir := t.TempDir()
	content := `
[package]
name = "sunholo/docparse"
version = "0.1.0"
edition = "1"
module_prefix = "docparse"

[exports]
modules = ["docparse/services/api", "docparse/handlers/parse"]
`
	os.WriteFile(filepath.Join(dir, ManifestFile), []byte(content), 0644)

	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Package.ModulePrefix != "docparse" {
		t.Errorf("expected module_prefix 'docparse', got %q", m.Package.ModulePrefix)
	}
}

func TestLoadManifest_ModulePrefix_ExportsWithPkgName(t *testing.T) {
	dir := t.TempDir()
	// Exports can also use the full vendor/name prefix
	content := `
[package]
name = "sunholo/docparse"
version = "0.1.0"
edition = "1"
module_prefix = "docparse"

[exports]
modules = ["sunholo/docparse/new_module", "docparse/legacy_module"]
`
	os.WriteFile(filepath.Join(dir, ManifestFile), []byte(content), 0644)

	_, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadManifest_ModulePrefix_RejectsSlashes(t *testing.T) {
	dir := t.TempDir()
	content := `
[package]
name = "sunholo/docparse"
version = "0.1.0"
edition = "1"
module_prefix = "my/prefix"

[exports]
modules = ["my/prefix/module"]
`
	os.WriteFile(filepath.Join(dir, ManifestFile), []byte(content), 0644)

	_, err := LoadManifest(dir)
	if err == nil {
		t.Fatal("expected error for module_prefix with slashes")
	}
}

func TestLoadManifest_ModulePrefix_BadExportStillFails(t *testing.T) {
	dir := t.TempDir()
	content := `
[package]
name = "sunholo/docparse"
version = "0.1.0"
edition = "1"
module_prefix = "docparse"

[exports]
modules = ["other/something"]
`
	os.WriteFile(filepath.Join(dir, ManifestFile), []byte(content), 0644)

	_, err := LoadManifest(dir)
	if err == nil {
		t.Fatal("expected error for export not matching package name or module_prefix")
	}
}

func TestLoadManifest_RejectsNonExactVersions(t *testing.T) {
	tests := []struct {
		version string
		wantErr bool
	}{
		{"0.1.0", false},
		{"1.2.3", false},
		{"latest", true},
		{"^0.1.0", true},
		{"~0.1.0", true},
		{">=0.1.0", true},
		{"<=0.1.0", true},
		{">0.1.0", true},
	}

	for _, tc := range tests {
		t.Run(tc.version, func(t *testing.T) {
			dir := t.TempDir()
			content := `
[package]
name = "test/pkg"
version = "0.1.0"
edition = "1"

[dependencies]
"sunholo/auth" = { version = "` + tc.version + `" }
`
			os.WriteFile(filepath.Join(dir, ManifestFile), []byte(content), 0644)

			_, err := LoadManifest(dir)
			if tc.wantErr && err == nil {
				t.Errorf("expected error for version %q, got nil", tc.version)
			}
			if tc.wantErr && err != nil && !strings.Contains(err.Error(), "non-exact version") {
				t.Errorf("expected 'non-exact version' error, got: %v", err)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error for version %q: %v", tc.version, err)
			}
		})
	}
}

func TestManifest_MapImportToModulePath(t *testing.T) {
	m := &PackageManifest{
		Package: PackageInfo{
			Name:         "sunholo/docparse",
			ModulePrefix: "docparse",
		},
	}

	// With module_prefix: remap vendor/name/sub → prefix/sub
	got := m.MapImportToModulePath("sunholo/docparse/services/api")
	if got != "docparse/services/api" {
		t.Errorf("expected 'docparse/services/api', got %q", got)
	}

	// Already prefix-based: return as-is
	got = m.MapImportToModulePath("docparse/services/api")
	if got != "docparse/services/api" {
		t.Errorf("expected 'docparse/services/api', got %q", got)
	}

	// No prefix set: return unchanged
	m2 := &PackageManifest{
		Package: PackageInfo{Name: "sunholo/auth"},
	}
	got = m2.MapImportToModulePath("sunholo/auth/keys")
	if got != "sunholo/auth/keys" {
		t.Errorf("expected 'sunholo/auth/keys', got %q", got)
	}
}

func TestManifest_MapModuleToImportPath(t *testing.T) {
	m := &PackageManifest{
		Package: PackageInfo{
			Name:         "sunholo/docparse",
			ModulePrefix: "docparse",
		},
	}

	got := m.MapModuleToImportPath("docparse/services/api")
	if got != "sunholo/docparse/services/api" {
		t.Errorf("expected 'sunholo/docparse/services/api', got %q", got)
	}
}
