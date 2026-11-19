package runner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/testutil/prefixpkg"
)

// `ailang run` of a root capped at [effects].max = ["IO"] whose main calls a
// dependency's FS function is refused before anything runs — under both
// spellings of the dependency module, on the evaluator and with
// --strict-bytecode — and the file the call would write never appears.
func TestCanonicalModuleIdentity_RunRefusesDependencyEffectBeyondRootCeiling(t *testing.T) {
	app := prefixpkg.Ceiling(t)
	for sp := range prefixpkg.Spellings {
		for _, engine := range []string{"eval", "strict-bytecode"} {
			t.Run(sp+"/"+engine, func(t *testing.T) {
				code := Run(t.Context(), Options{
					Filename: filepath.Join(app, "call_"+sp+".ail"),
					Entry:    "main", ArgsJSON: "null", Caps: "IO,FS", Quiet: true, NoPrint: true,
					BytecodeMode: engine != "eval", StrictBytecode: engine != "eval",
				})
				if code == 0 {
					t.Fatalf("run exited 0; the root's [effects].max = [IO] must refuse the dependency's FS call")
				}
				if _, err := os.Stat(prefixpkg.CeilingMarker(app, sp)); !os.IsNotExist(err) {
					t.Fatalf("the refused program performed its FS effect: %s exists (stat err %v)", prefixpkg.CeilingMarker(app, sp), err)
				}
			})
		}
	}
	// Control: with FS inside the ceiling the same program runs and writes the
	// marker, so the absence above is the refusal, not a broken fixture.
	t.Run("control", func(t *testing.T) {
		ok := prefixpkg.Ceiling(t)
		manifest := prefixpkg.Manifest("acme/app", "", nil, map[string]string{"evil/dep": "../dep"}, []string{"IO", "FS"})
		if err := os.WriteFile(filepath.Join(ok, "ailang.toml"), []byte(manifest), 0o600); err != nil {
			t.Fatal(err)
		}
		code := Run(t.Context(), Options{Filename: filepath.Join(ok, "call_pfx.ail"), Entry: "main", ArgsJSON: "null", Caps: "IO,FS", Quiet: true, NoPrint: true})
		if code != 0 {
			t.Fatalf("control run exited %d", code)
		}
		if _, err := os.Stat(prefixpkg.CeilingMarker(ok, "pfx")); err != nil {
			t.Fatalf("control run did not write its marker: %v", err)
		}
	})
}
