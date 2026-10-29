package embed

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/pkg"
	"github.com/sunholo-data/ailang/internal/testutil/pkgassets"
)

func TestEngineLockedAssetBytes(t *testing.T) {
	root, locked := pkgassets.New(t)
	engine := New(root)
	defer engine.Close()
	base := effects.NewEffContext(nil)
	base.Grant(effects.NewCapability("FS"))
	engine.SetEffContext(base)
	for i := 0; i < 2; i++ {
		v, err := engine.Call("asset_fixture", "read")
		if err != nil {
			t.Fatal(err)
		}
		if v.(*eval.StringValue).Value != "locked" {
			t.Fatal(v)
		}
		engine.SetEffContext(base) // Replacement must retain the engine's lock.
	}
	if base.PkgAssets != nil {
		t.Fatal("mutated caller's context")
	}
	// A second engine with a different lock must have its own targets.
	lf, err := pkg.LoadLockFile(root)
	if err != nil {
		t.Fatal(err)
	}
	newer := filepath.Join(filepath.Dir(locked), "0.4.0")
	lf.Packages[0].Version = "0.4.0"
	lf.Packages[0].ContentHash, err = pkg.ContentHash(newer)
	if err != nil {
		t.Fatal(err)
	}
	if err := lf.Save(root); err != nil {
		t.Fatal(err)
	}
	second := New(root)
	defer second.Close()
	second.SetEffContext(base)
	v, err := second.Call("asset_fixture", "read")
	if err != nil {
		t.Fatal(err)
	}
	if v.(*eval.StringValue).Value != "newer" {
		t.Fatal(v)
	}
	v, err = engine.Call("asset_fixture", "read")
	if err != nil {
		t.Fatal(err)
	}
	if v.(*eval.StringValue).Value != "locked" {
		t.Fatal("engines shared targets")
	}
	if err := os.RemoveAll(locked); err != nil {
		t.Fatal(err)
	}
	v, err = engine.Call("asset_fixture", "read")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(v.(*eval.StringValue).Value, "test/data@0.3.0") {
		t.Fatal(v)
	}
}

func TestEnginePreserveFloatsLockedAssetBytes(t *testing.T) {
	root, _ := pkgassets.New(t)
	engine := New(root)
	defer engine.Close()
	ctx := effects.NewEffContext(nil)
	ctx.Grant(effects.NewCapability("FS"))
	engine.SetEffContext(ctx)
	for i := 0; i < 2; i++ {
		v, err := engine.CallPreserveFloats("asset_fixture", "read")
		if err != nil {
			t.Fatal(err)
		}
		if v.(*eval.StringValue).Value != "locked" {
			t.Fatalf("CallPreserveFloats selected unlocked bytes: %v", v)
		}
		engine.SetEffContext(ctx)
	}
}

func TestEngineMalformedAssetLockFailsBeforeEvaluation(t *testing.T) {
	root, _ := pkgassets.New(t)
	if err := os.WriteFile(filepath.Join(root, pkg.LockFileName), []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	engine := New(root)
	defer engine.Close()
	ctx := effects.NewEffContext(nil)
	ctx.Grant(effects.NewCapability("FS"))
	engine.SetEffContext(ctx)
	if v, err := engine.CallPreserveFloats("asset_fixture", "read"); err == nil || !strings.Contains(err.Error(), pkg.LockFileName) || v != nil {
		t.Fatalf("malformed lock on CallPreserveFloats: value=%v error=%v", v, err)
	}
	if _, err := engine.Call("asset_fixture", "read"); err == nil || !strings.Contains(err.Error(), pkg.LockFileName) {
		t.Fatalf("malformed lock: %v", err)
	}
	if err := engine.Load("asset_fixture"); err == nil || !strings.Contains(err.Error(), pkg.LockFileName) {
		t.Fatalf("malformed lock on Load: %v", err)
	}
}
