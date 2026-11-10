package pipeline

import (
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/testutil/prefixpkg"
)

// Review of PR #2224: identity must come from the RESOLVED module, and a
// module_prefix spelling must be read in the importing package's scope.

const shapeSrc = `module shp/types/shape

export type Shape = Circle(int) | Square(int)

export func mkCircle(r: int) -> Shape = Circle(r)

export func marked() -> int = 1
`

const areaSrc = `module shp/ops/area

import shp/types/shape (Shape, Circle, Square)

export func area(s: Shape) -> int =
  match s { Circle(r) => 3 * r * r, Square(w) => w * w }
`

// A package without an [exports] list accepts any path that names a file:
// pkg/acme/shapes/types/shape and pkg/acme/shapes/shp/types/shape both reach
// shp/types/shape.ail. Two spellings, one resolved file, one identity.
func TestCanonicalModuleIdentity_UnexportedPackageSpellingsOfOneFile(t *testing.T) {
	root := prefixpkg.Tree(t, map[string]string{
		"shapes/ailang.toml":         prefixpkg.Manifest("acme/shapes", "shp", nil, nil, nil),
		"shapes/shp/types/shape.ail": shapeSrc,
		"shapes/shp/ops/area.ail":    areaSrc,
		"app/ailang.toml":            prefixpkg.Manifest("acme/app", "", nil, map[string]string{"acme/shapes": "../shapes"}, nil),
		"app/main.ail": `module main

import std/io (println)
import pkg/acme/shapes/types/shape (marked)
import pkg/acme/shapes/shp/types/shape (mkCircle)
import pkg/acme/shapes/ops/area (area)

export func main() -> () ! {IO} = println(show(area(mkCircle(1)) + marked()))
`,
	})
	app := filepath.Join(root, "app")
	prefixpkg.Lock(t, app, prefixpkg.Dep{Name: "acme/shapes", Dir: "../shapes"})
	result := checkFixture(t, filepath.Join(app, "main.ail"))
	assertOneIdentityPerFile(t, result.Modules)
	assertImportsResolveToLoaded(t, result.Modules)
	if _, ok := result.Modules[prefixpkg.Canonical]; !ok {
		t.Fatalf("want the shape module under %s; modules: %v", prefixpkg.Canonical, sortedKeys(result.Modules))
	}
}

// Two dependencies declare the same module_prefix "px" and both have
// px/types/t. bbb/two's own module importing px/types/t means bbb/two's file —
// whichever way the two package names sort.
func TestCanonicalModuleIdentity_SharedPrefixScopedToImportingPackage(t *testing.T) {
	for _, other := range []string{"aaa/one", "zzz/one"} {
		t.Run(other, func(t *testing.T) {
			root := prefixpkg.Tree(t, map[string]string{
				"one/ailang.toml":    prefixpkg.Manifest(other, "px", nil, nil, nil),
				"one/px/types/t.ail": "module px/types/t\n\nexport func who() -> string = \"" + other + "\"\n",
				"two/ailang.toml":    prefixpkg.Manifest("bbb/two", "px", nil, nil, nil),
				"two/px/types/t.ail": "module px/types/t\n\nexport func who() -> string = \"bbb/two\"\n",
				"two/px/use.ail":     "module px/use\n\nimport px/types/t (who)\n\nexport func twoSays() -> string = who()\n",
				"app/ailang.toml": prefixpkg.Manifest("acme/app", "", nil,
					map[string]string{other: "../one", "bbb/two": "../two"}, nil),
				"app/main.ail": "module main\n\nimport std/io (println)\nimport pkg/bbb/two/use (twoSays)\n\n" +
					"export func main() -> () ! {IO} = println(twoSays())\n",
			})
			app := filepath.Join(root, "app")
			prefixpkg.Lock(t, app, prefixpkg.Dep{Name: other, Dir: "../one"}, prefixpkg.Dep{Name: "bbb/two", Dir: "../two"})
			result := checkFixture(t, filepath.Join(app, "main.ail"))
			assertOneIdentityPerFile(t, result.Modules)
			want := canonicalFilePath(filepath.Join(root, "two", "px", "types", "t.ail"))
			wrong := canonicalFilePath(filepath.Join(root, "one", "px", "types", "t.ail"))
			found := false
			for id, mod := range result.Modules {
				if mod.File == nil || mod.File.Path == "" {
					continue
				}
				switch canonicalFilePath(mod.File.Path) {
				case wrong:
					t.Fatalf("bbb/two's px/types/t loaded %s's file as %s", other, id)
				case want:
					found = true
					if id != "pkg/bbb/two/types/t" {
						t.Errorf("bbb/two's t has identity %s, want pkg/bbb/two/types/t", id)
					}
				}
			}
			if !found {
				t.Fatalf("bbb/two's own px/types/t was not loaded; modules: %v", sortedKeys(result.Modules))
			}
		})
	}
}

// A prefixed package that is only a TRANSITIVE dependency (app2 -> mid ->
// shapes, mid declaring module_prefix "md") resolves its own md/ imports.
func TestCanonicalModuleIdentity_TransitivePrefixedPackage(t *testing.T) {
	root := prefixpkg.Tree(t, map[string]string{
		"shapes/ailang.toml":         prefixpkg.Manifest("acme/shapes", "shp", []string{"shp/types/shape", "shp/ops/area"}, nil, nil),
		"shapes/shp/types/shape.ail": shapeSrc,
		"shapes/shp/ops/area.ail":    areaSrc,
		"mid/ailang.toml": prefixpkg.Manifest("acme/mid", "md", []string{"md/api", "md/inner"},
			map[string]string{"acme/shapes": "../shapes"}, nil),
		"mid/md/inner.ail": "module md/inner\n\nimport pkg/acme/shapes/types/shape (Shape, mkCircle)\n\n" +
			"export func one() -> Shape = mkCircle(1)\n",
		"mid/md/api.ail": "module md/api\n\nimport md/inner (one)\nimport pkg/acme/shapes/ops/area (area)\n\n" +
			"export func midArea() -> int = area(one())\n",
		"app/ailang.toml": prefixpkg.Manifest("acme/app2", "", nil, map[string]string{"acme/mid": "../mid"}, nil),
		"app/main.ail": "module main\n\nimport std/io (println)\nimport pkg/acme/mid/api (midArea)\n\n" +
			"export func main() -> () ! {IO} = println(show(midArea()))\n",
	})
	app := filepath.Join(root, "app")
	prefixpkg.Lock(t, app, prefixpkg.Dep{Name: "acme/mid", Dir: "../mid"}, prefixpkg.Dep{Name: "acme/shapes", Dir: "../shapes"})
	result := checkFixture(t, filepath.Join(app, "main.ail"))
	assertOneIdentityPerFile(t, result.Modules)
	assertImportsResolveToLoaded(t, result.Modules)
	for _, id := range []string{"pkg/acme/mid/api", "pkg/acme/mid/inner", prefixpkg.Canonical, "pkg/acme/shapes/ops/area"} {
		if _, ok := result.Modules[id]; !ok {
			t.Errorf("module %s not loaded; modules: %v", id, sortedKeys(result.Modules))
		}
	}
}
