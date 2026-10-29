package effects

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/testutil"
)

func assetResult(t *testing.T, ctx *EffContext, name, rel string) (string, string) {
	t.Helper()
	v, err := pkgAssetPath(ctx, []eval.Value{&eval.StringValue{Value: name}, &eval.StringValue{Value: rel}})
	if err != nil {
		t.Fatal(err)
	}
	tagged := v.(*eval.TaggedValue)
	return tagged.CtorName, tagged.Fields[0].(*eval.StringValue).Value
}

func TestPkgAssetPath_LockedTargets(t *testing.T) {
	home := t.TempDir()
	testutil.SetHomeDir(t, home)
	base := filepath.Join(home, ".ailang", "cache", "registry", "test", "pkg")
	for _, version := range []string{"0.3.0", "0.4.0"} {
		dir := filepath.Join(base, version, AssetsDir)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "data"), []byte(version), 0600); err != nil {
			t.Fatal(err)
		}
	}
	locked := filepath.Join(base, "0.3.0")
	ctx := NewEffContext(nil)
	ctx.PkgAssets = map[string]PkgAssetTarget{"test/pkg": {Dir: locked, Version: "0.3.0"}}
	for _, clone := range []*EffContext{ctx, ctx.WithBudget(nil), ctx.Clone().(*EffContext)} {
		tag, path := assetResult(t, clone, "test/pkg", "data")
		if tag != "Ok" || path != filepath.Join(locked, AssetsDir, "data") {
			t.Fatalf("%s %s", tag, path)
		}
	}
	if err := os.RemoveAll(locked); err != nil {
		t.Fatal(err)
	}
	tag, msg := assetResult(t, ctx, "test/pkg", "data")
	if tag != "Err" || !strings.Contains(msg, "test/pkg@0.3.0") || !strings.Contains(msg, "ailang install") {
		t.Fatalf("%s %s", tag, msg)
	}
	// A present unresolved target must also block fallback.
	ctx.PkgAssets = map[string]PkgAssetTarget{"test/pkg": {Version: "0.3.0"}}
	tag, msg = assetResult(t, ctx, "test/pkg", "data")
	if tag != "Err" {
		t.Fatalf("unresolved target fell back: %s", msg)
	}
	// A dependency absent from the map may use compatibility fallback.
	ctx.PkgAssets = map[string]PkgAssetTarget{}
	tag, path := assetResult(t, ctx, "test/pkg", "data")
	if tag != "Ok" || path != filepath.Join(base, "0.4.0", AssetsDir, "data") {
		t.Fatalf("%s %s", tag, path)
	}
}

func TestPkgAssetPath_LocalTargetAndValidation(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, AssetsDir), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, AssetsDir, "data"), []byte("local"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx := NewEffContext(nil)
	ctx.PkgAssets = map[string]PkgAssetTarget{"test/pkg": {Dir: dir, Version: "git-or-path"}}
	tag, path := assetResult(t, ctx, "test/pkg", "data")
	if tag != "Ok" || path != filepath.Join(dir, AssetsDir, "data") {
		t.Fatalf("%s %s", tag, path)
	}
	for _, rel := range []string{"../data", "/data", `C:\data`, `nested\..\..\escape`, ""} {
		tag, msg := assetResult(t, ctx, "test/pkg", rel)
		if tag != "Err" || !strings.HasPrefix(msg, "invalid relative path:") {
			t.Fatalf("%s %s", tag, msg)
		}
	}
	ctx.Env.Sandbox = dir
	tag, msg := assetResult(t, ctx, "bad", "../data")
	if tag != "Err" || !strings.Contains(msg, "outside the FS sandbox") {
		t.Fatalf("%s %s", tag, msg)
	}
}

func TestPkgAssetPath_InvalidVersions(t *testing.T) {
	home := t.TempDir()
	testutil.SetHomeDir(t, home)
	base := filepath.Join(home, ".ailang", "cache", "registry", "test", "pkg")
	if err := os.MkdirAll(filepath.Join(base, "garbage"), 0755); err != nil {
		t.Fatal(err)
	}
	tag, msg := assetResult(t, nil, "test/pkg", "data")
	if tag != "Err" || !strings.Contains(msg, "no valid installed versions") {
		t.Fatalf("%s %s", tag, msg)
	}
}

// Existing embedders using PkgDirs must keep lock authority in budget scopes.
func TestPkgAssetPath_LegacyBudgetContext(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, AssetsDir), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, AssetsDir, "data"), []byte("locked"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx := NewEffContext(nil)
	ctx.PkgDirs = map[string]string{"test/pkg": dir}
	for _, derived := range []*EffContext{ctx, ctx.WithBudget(nil), ctx.Clone().(*EffContext)} {
		tag, got := assetResult(t, derived, "test/pkg", "data")
		if tag != "Ok" || got != filepath.Join(dir, AssetsDir, "data") {
			t.Fatalf("lost legacy locked target: %s %s", tag, got)
		}
	}
}
