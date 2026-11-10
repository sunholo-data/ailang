package pipeline

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/testutil/prefixpkg"
)

// Diagnostics that name a module whose identity is not the spelling its
// author wrote must show both (review of #2224, item 5).

const shapesNoExports = "[package]\nname = \"acme/shapes\"\nmodule_prefix = \"shp\"\nversion = \"0.1.0\"\nedition = \"1\"\n"

func checkErr(t *testing.T, app, entry string) string {
	t.Helper()
	_, err := Run(Config{Mode: ModeCheck, NoCache: true}, Source{Filename: filepath.Join(app, entry)})
	if err == nil {
		t.Fatalf("check %s: expected an error", entry)
	}
	return err.Error()
}

// LDR002: a cycle inside a dependency, written across two spellings
// (shp/cyc/a imports shp/cyc/b; b imports pkg/acme/shapes/cyc/a).
func TestCanonicalModuleIdentity_CycleErrorNamesSpelling(t *testing.T) {
	app, _ := prefixpkg.New(t, map[string]string{
		"shapes/ailang.toml":   shapesNoExports,
		"shapes/shp/cyc/a.ail": "module shp/cyc/a\n\nimport shp/cyc/b (fb)\n\nexport func fa() -> int = 1\n",
		"shapes/shp/cyc/b.ail": "module shp/cyc/b\n\nimport pkg/acme/shapes/cyc/a (fa)\n\nexport func fb() -> int = fa()\n",
		"app/cycle.ail":        "module cycle\n\nimport pkg/acme/shapes/cyc/a (fa)\n\nexport func main() -> int = fa()\n",
	})
	msg := checkErr(t, app, "cycle.ail")
	want := "LDR002: dependency cycle detected: pkg/acme/shapes/cyc/a -> pkg/acme/shapes/cyc/b (imported as shp/cyc/b) -> pkg/acme/shapes/cyc/a"
	if msg != "dependency cycle: "+want {
		t.Fatalf("cycle error should name the canonical modules and the spelling written\n got: %s\nwant: %s", msg, want)
	}
}

// A type error inside a dependency module names its identity and the module
// path its file declares.
func TestCanonicalModuleIdentity_TypeErrorNamesDeclaredModule(t *testing.T) {
	app, _ := prefixpkg.New(t, map[string]string{
		"shapes/shp/ops/area.ail": "module shp/ops/area\n\nimport shp/types/shape (Shape, Circle, Square)\n\n" +
			"export func area(s: Shape) -> int = \"not an int\"\n\nexport func unitSquare() -> Shape = Square(1)\n",
	})
	msg := checkErr(t, app, "main.ail")
	want := "type error in pkg/acme/shapes/ops/area (module shp/ops/area)"
	if !strings.Contains(msg, want) {
		t.Fatalf("type error should name the dependency module and its declared path\n got: %s\nwant substring: %s", msg, want)
	}
}

// The root's own modules and entry files are named as before: no suffix.
func TestCanonicalModuleIdentity_RootDiagnosticsUnchanged(t *testing.T) {
	app, _ := prefixpkg.New(t, map[string]string{
		"app/bad.ail": "module bad\n\nexport func f() -> int = \"x\"\n",
	})
	msg := checkErr(t, app, "bad.ail")
	if strings.Contains(msg, "(module ") || strings.Contains(msg, "imported as") {
		t.Fatalf("a root module's diagnostic gained an identity note: %s", msg)
	}
}
