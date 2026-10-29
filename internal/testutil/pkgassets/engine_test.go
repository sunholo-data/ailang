package pkgassets_test

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/embed"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/testutil/pkgassets"
)

// Keep this public Engine regression external: besides exercising the same
// entrypoint used by MCP/A2A, it verifies the shared offline fixture end to end.
func TestEnginePreserveFloatsLockedAssetBytes(t *testing.T) {
	root, _ := pkgassets.New(t)
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
