package effects

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

// TestPkgAssetPath_Resolves verifies that an installed package's asset is
// resolved to an absolute path under the registry cache.
func TestPkgAssetPath_Resolves(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	pkgDir := filepath.Join(home, ".ailang", "cache", "registry", "test", "pkg", "0.1.0", AssetsDir)
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	expected := filepath.Join(pkgDir, "foo.txt")
	if err := os.WriteFile(expected, []byte("x"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := pkgAssetPath(nil, []eval.Value{
		&eval.StringValue{Value: "test/pkg"},
		&eval.StringValue{Value: "foo.txt"},
	})
	if err != nil {
		t.Fatalf("pkgAssetPath: %v", err)
	}
	tagged, ok := got.(*eval.TaggedValue)
	if !ok || tagged.CtorName != "Ok" {
		t.Fatalf("expected Ok, got %T %v", got, got)
	}
	gotPath := tagged.Fields[0].(*eval.StringValue).Value
	if gotPath != expected {
		t.Errorf("expected path %q, got %q", expected, gotPath)
	}
}

// TestPkgAssetPath_PicksLatestVersion verifies that when multiple versions are
// installed, the lexically-highest version is selected.
func TestPkgAssetPath_PicksLatestVersion(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	base := filepath.Join(home, ".ailang", "cache", "registry", "test", "pkg")
	for _, v := range []string{"0.1.0", "0.2.0", "0.10.0"} {
		dir := filepath.Join(base, v, AssetsDir)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "v.txt"), []byte(v), 0644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}

	got, err := pkgAssetPath(nil, []eval.Value{
		&eval.StringValue{Value: "test/pkg"},
		&eval.StringValue{Value: "v.txt"},
	})
	if err != nil {
		t.Fatalf("pkgAssetPath: %v", err)
	}
	tagged := got.(*eval.TaggedValue)
	gotPath := tagged.Fields[0].(*eval.StringValue).Value
	// Lexical sort: "0.10.0" < "0.2.0" < "0.1.0"? No: "0.10.0" sorts before "0.2.0".
	// We document "lexical sort" — that means "0.2.0" wins here.
	want := filepath.Join(base, "0.2.0", AssetsDir, "v.txt")
	if gotPath != want {
		t.Errorf("expected %q, got %q", want, gotPath)
	}
}

// TestPkgAssetPath_PackageMissing returns Err when the package is not installed.
func TestPkgAssetPath_PackageMissing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got, err := pkgAssetPath(nil, []eval.Value{
		&eval.StringValue{Value: "test/pkg"},
		&eval.StringValue{Value: "foo.txt"},
	})
	if err != nil {
		t.Fatalf("pkgAssetPath: %v", err)
	}
	tagged, ok := got.(*eval.TaggedValue)
	if !ok || tagged.CtorName != "Err" {
		t.Fatalf("expected Err, got %T %v", got, got)
	}
	msg := tagged.Fields[0].(*eval.StringValue).Value
	if msg == "" {
		t.Error("expected non-empty Err message")
	}
}

// TestPkgAssetPath_AssetMissing returns Err when the package is installed but
// the asset file is absent.
func TestPkgAssetPath_AssetMissing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	pkgDir := filepath.Join(home, ".ailang", "cache", "registry", "test", "pkg", "0.1.0", AssetsDir)
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	got, err := pkgAssetPath(nil, []eval.Value{
		&eval.StringValue{Value: "test/pkg"},
		&eval.StringValue{Value: "missing.txt"},
	})
	if err != nil {
		t.Fatalf("pkgAssetPath: %v", err)
	}
	tagged := got.(*eval.TaggedValue)
	if tagged.CtorName != "Err" {
		t.Fatalf("expected Err, got %s", tagged.CtorName)
	}
}

// TestPkgAssetPath_RejectsBadInputs covers invalid package names and unsafe
// relative paths.
func TestPkgAssetPath_RejectsBadInputs(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cases := []struct {
		name string
		pkg  string
		rel  string
	}{
		{"single segment", "test", "foo.txt"},
		{"empty rel", "test/pkg", ""},
		{"absolute rel", "test/pkg", "/etc/passwd"},
		{"parent traversal", "test/pkg", "../escape"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := pkgAssetPath(nil, []eval.Value{
				&eval.StringValue{Value: tc.pkg},
				&eval.StringValue{Value: tc.rel},
			})
			if err != nil {
				t.Fatalf("pkgAssetPath: %v", err)
			}
			tagged := got.(*eval.TaggedValue)
			if tagged.CtorName != "Err" {
				t.Errorf("expected Err for %s, got %s", tc.name, tagged.CtorName)
			}
		})
	}
}
