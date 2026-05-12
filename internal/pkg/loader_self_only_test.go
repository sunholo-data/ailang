package pkg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeSelfOnlyPackage writes a minimal package layout (ailang.toml + a few
// .ail files) into a fresh tempdir and returns the package root. Unlike
// setupTestPackage, no lock file is involved — this mirrors the state a
// human author is in *before* running `ailang lock`.
func writeSelfOnlyPackage(t *testing.T, name string, exports []string, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()

	quoted := make([]string, len(exports))
	for i, e := range exports {
		quoted[i] = `"` + e + `"`
	}
	manifest := `[package]
name = "` + name + `"
version = "0.1.0"
edition = "1"

[exports]
modules = [` + strings.Join(quoted, ", ") + `]
`
	if err := os.WriteFile(filepath.Join(dir, ManifestFile), []byte(manifest), 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	for path, content := range files {
		fullPath := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("write file: %v", err)
		}
	}
	return dir
}

func TestSelfOnlyPackageLoader_ResolvesSibling(t *testing.T) {
	dir := writeSelfOnlyPackage(t, "sunholo/linkedin",
		[]string{"sunholo/linkedin/types", "sunholo/linkedin/auth"},
		map[string]string{
			"types.ail": "module sunholo/linkedin/types\n",
			"auth.ail":  "module sunholo/linkedin/auth\n",
		})

	loader, err := NewSelfOnlyPackageLoader(dir)
	if err != nil {
		t.Fatalf("NewSelfOnlyPackageLoader: %v", err)
	}

	got, err := loader.ResolveImport("sunholo/linkedin/types")
	if err != nil {
		t.Fatalf("self-reference should resolve without lock file: %v", err)
	}
	if !strings.HasSuffix(got, "types.ail") {
		t.Errorf("resolved = %q, want suffix types.ail", got)
	}
}

func TestSelfOnlyPackageLoader_ExternalImportHasClearError(t *testing.T) {
	dir := writeSelfOnlyPackage(t, "sunholo/linkedin", nil, map[string]string{
		"types.ail": "module sunholo/linkedin/types\n",
	})

	loader, err := NewSelfOnlyPackageLoader(dir)
	if err != nil {
		t.Fatalf("NewSelfOnlyPackageLoader: %v", err)
	}

	_, err = loader.ResolveImport("sunholo/firestore/client")
	if err == nil {
		t.Fatal("external import must error when no lock file exists")
	}
	if !strings.Contains(err.Error(), "ailang.lock") {
		t.Errorf("error %q must mention ailang.lock so authors know what to run", err.Error())
	}
}

func TestSelfOnlyPackageLoader_NotAPackage(t *testing.T) {
	dir := t.TempDir() // no ailang.toml

	_, err := NewSelfOnlyPackageLoader(dir)
	if err == nil {
		t.Fatal("constructor must fail for a directory without ailang.toml")
	}
	if !strings.Contains(err.Error(), "ailang.toml") {
		t.Errorf("error %q must mention ailang.toml", err.Error())
	}
}

func TestSelfOnlyPackageLoader_RespectsExportsList(t *testing.T) {
	// types is exported; internal_helper is not.
	dir := writeSelfOnlyPackage(t, "sunholo/linkedin",
		[]string{"sunholo/linkedin/types"},
		map[string]string{
			"types.ail":           "module sunholo/linkedin/types\n",
			"internal_helper.ail": "module sunholo/linkedin/internal_helper\n",
		})

	loader, err := NewSelfOnlyPackageLoader(dir)
	if err != nil {
		t.Fatalf("NewSelfOnlyPackageLoader: %v", err)
	}

	// Exported sibling resolves
	if _, err := loader.ResolveImport("sunholo/linkedin/types"); err != nil {
		t.Errorf("exported self-import should succeed: %v", err)
	}

	// Non-exported sibling: the package loader checks export visibility even
	// for self-references. This matches the behavior consumers see, so an
	// author can't accidentally rely on importing an internal-only module
	// from a sibling and have it break only after publishing.
	if _, err := loader.ResolveImport("sunholo/linkedin/internal_helper"); err == nil {
		t.Error("non-exported self-import should error to match consumer-side behavior")
	}
}
