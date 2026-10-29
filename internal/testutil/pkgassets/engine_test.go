package pkgassets

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/embed"
	"github.com/sunholo-data/ailang/internal/eval"
)

// Exercise the public entrypoint used by MCP/A2A with the shared offline fixture.
// Keep an in-package test so the existing TestGoFiles coverage filter retains
// this package in the submitted profile; external-only suites are not included.
func TestEnginePreserveFloatsLockedAssetBytes(t *testing.T) {
	root, _ := New(t)
	engine := embed.New(root)
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
