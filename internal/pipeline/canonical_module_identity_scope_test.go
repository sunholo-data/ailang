package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/testutil/prefixpkg"
)

// Second review of PR #2224: every import resolves in its importer's scope —
// the importer, its DECLARED dependencies, std — and one file has one
// identity in every build, checked by the pipeline itself. Fixtures follow the
// reviewer's fxA, fxB, fxB2, fxC, fxD, fxG, fxI, fxJ and fxM.

type dep = prefixpkg.Dep

func pkgToml(name, prefix string, exports []string, deps map[string]string) string {
	return prefixpkg.Manifest(name, prefix, exports, deps, nil)
}

// tree writes files, locks app/ against deps and returns the app dir.
func tree(t *testing.T, files map[string]string, deps ...dep) string {
	t.Helper()
	root := prefixpkg.Tree(t, files)
	app := filepath.Join(root, "app")
	prefixpkg.Lock(t, app, deps...)
	return app
}

func wantCheckError(t *testing.T, app, entry string, wants ...string) {
	t.Helper()
	_, err := Run(Config{Mode: ModeCheck, NoCache: true}, Source{Filename: filepath.Join(app, entry)})
	if err == nil {
		t.Fatalf("check %s passed; want an error containing %q", entry, wants)
	}
	for _, w := range wants {
		if !strings.Contains(err.Error(), w) {
			t.Fatalf("check %s error\n got: %v\nwant substring: %s", entry, err, w)
		}
	}
}

// fxJ: a root package without a prefix imports util.ail as `util` (a path)
// and as `acme/app/util` (its bare self-import): one module.
func TestScope_RootPathAndSelfImportOneIdentity(t *testing.T) {
	app := tree(t, map[string]string{
		"app/ailang.toml": pkgToml("acme/app", "", nil, nil),
		"app/util.ail":    "module acme/app/util\n\nexport func mk() -> int = 1\n\nexport func get(n: int) -> int = n + 1\n",
		"app/main.ail": "module acme/app/main\n\nimport std/io (println)\nimport util (mk)\nimport acme/app/util (get)\n\n" +
			"export func main() -> () ! {IO} = println(show(get(mk())))\n",
	})
	t.Chdir(app) // a project-relative spelling resolves from the working directory
	result := checkFixture(t, filepath.Join(app, "main.ail"))
	assertOneIdentityPerFile(t, result.Modules)
	if _, ok := result.Modules["acme/app/util"]; !ok {
		t.Fatalf("util not loaded as acme/app/util; modules %v", sortedKeys(result.Modules))
	}
	if _, dup := result.Modules["util"]; dup {
		t.Fatal("util also loaded under its path spelling")
	}
}

// What the pipeline's file↔identity check still has to catch: a second path
// to a package file that no resolver governs — here a symlink into the
// dependency. The build fails with LDR003 naming the file and both identities.
// (Removing the check in validateAndSort makes this test fail.)
func TestScope_SecondPathToOneFileIsLDR003(t *testing.T) {
	app, shapes := prefixpkg.New(t, map[string]string{
		"app/twice.ail": "module twice\n\nimport std/io (println)\nimport pkg/acme/shapes/types/shape (marked)\n" +
			"import vend/shp/types/shape (mkCircle)\n\nexport func main() -> () ! {IO} = println(show(marked()))\n",
	})
	if err := os.Symlink(shapes, filepath.Join(app, "vend")); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	t.Chdir(app)
	wantCheckError(t, app, "twice.ail", "LDR003: module file ", "is loaded under two identities",
		"pkg/acme/shapes/types/shape", "vend/shp/types/shape")
}

// fxA: dependency aaa/a declares ccc/c (prefix cc), which lacks cc/x; the
// root's other dependency eee/e (also prefix cc) has it. aaa/a's cc/x is not
// found in aaa/a's scope — not "dependency cycle", and never eee/e's file.
// fxC: the ROOT has prefix cc and cc/x, and ccc/c's cc/x is unexported: aaa/a
// must not get the root's file either.
func TestScope_DependencyMissTailsInItsOwnScope(t *testing.T) {
	common := map[string]string{
		"a/ailang.toml": pkgToml("aaa/a", "pa", nil, map[string]string{"ccc/c": "../c"}),
		"a/pa/m.ail":    "module pa/m\n\nimport cc/x (who)\n\nexport func whoA() -> string = who()\n",
		"c/cc/y.ail":    "module cc/y\n\nexport func y() -> int = 1\n",
		"app/main.ail": "module main\n\nimport std/io (println)\nimport pkg/aaa/a/m (whoA)\n\n" +
			"export func main() -> () ! {IO} = println(whoA())\n",
	}
	t.Run("fxA", func(t *testing.T) {
		files := map[string]string{
			"app/ailang.toml": pkgToml("acme/app", "", nil, map[string]string{"aaa/a": "../a", "eee/e": "../e"}),
			"c/ailang.toml":   pkgToml("ccc/c", "cc", nil, nil),
			"e/ailang.toml":   pkgToml("eee/e", "cc", nil, nil),
			"e/cc/x.ail":      "module cc/x\n\nexport func who() -> string = \"eee/e\"\n",
		}
		for k, v := range common {
			files[k] = v
		}
		app := tree(t, files, dep{Name: "aaa/a", Dir: "../a"}, dep{Name: "ccc/c", Dir: "../c"}, dep{Name: "eee/e", Dir: "../e"})
		wantCheckError(t, app, "main.ail", "failed to load pkg/aaa/a/m: LDR001: module pkg/ccc/c/x (imported as cc/x), imported by package aaa/a, was not found in its scope",
			`not found in package "ccc/c"`)
	})
	t.Run("fxC", func(t *testing.T) {
		files := map[string]string{
			"app/ailang.toml": pkgToml("acme/app", "cc", nil, map[string]string{"aaa/a": "../a"}),
			"app/cc/x.ail":    "module cc/x\n\nexport func who() -> string = \"ROOT\"\n",
			"c/ailang.toml":   pkgToml("ccc/c", "cc", []string{"cc/y"}, nil),
			"c/cc/x.ail":      "module cc/x\n\nexport func who() -> string = \"ccc/c unexported\"\n",
		}
		for k, v := range common {
			files[k] = v
		}
		app := tree(t, files, dep{Name: "aaa/a", Dir: "../a"}, dep{Name: "ccc/c", Dir: "../c"})
		wantCheckError(t, app, "main.ail", "LDR001: module pkg/ccc/c/x", "imported by package aaa/a, was not found in its scope", "not exported")
	})
}

// fxB, fxB2, fxG: an import satisfied only by a package the importer does not
// declare is LDR006 — by the build-wide prefix table, by a published path, or
// (a dependency importing the root) by the root's prefix or bare name.
func TestScope_UndeclaredPackageIsLDR006(t *testing.T) {
	base := map[string]string{
		"app/ailang.toml": pkgToml("acme/app", "", nil, map[string]string{"aaa/a": "../a", "eee/e": "../e"}),
		"a/ailang.toml":   pkgToml("aaa/a", "pa", nil, nil),
		"e/ailang.toml":   pkgToml("eee/e", "cc", nil, nil),
		"e/cc/x.ail":      "module cc/x\n\nexport func who() -> string = \"eee/e\"\n",
		"app/main.ail": "module main\n\nimport std/io (println)\nimport pkg/aaa/a/m (whoA)\n\n" +
			"export func main() -> () ! {IO} = println(whoA())\n",
	}
	for name, spelling := range map[string]string{"fxB": "cc/x", "fxB2": "pkg/eee/e/x"} {
		t.Run(name, func(t *testing.T) {
			files := map[string]string{"a/pa/m.ail": "module pa/m\n\nimport " + spelling + " (who)\n\nexport func whoA() -> string = who()\n"}
			for k, v := range base {
				files[k] = v
			}
			app := tree(t, files, dep{Name: "aaa/a", Dir: "../a"}, dep{Name: "eee/e", Dir: "../e"})
			wantCheckError(t, app, "main.ail", "LDR006: package aaa/a imports \""+spelling+"\" from package eee/e, which it does not declare",
				"add \"eee/e\" to [dependencies] in aaa/a's ailang.toml")
		})
	}
	for name, spelling := range map[string]string{"fxG-prefix": "ap/util", "fxG-bare": "acme/app/util"} {
		t.Run(name, func(t *testing.T) {
			app := tree(t, map[string]string{
				"app/ailang.toml": pkgToml("acme/app", "ap", nil, map[string]string{"aaa/a": "../a"}),
				"app/ap/util.ail": "module ap/util\n\nexport func rootSecret() -> string = \"ROOT-SECRET\"\n",
				"a/ailang.toml":   pkgToml("aaa/a", "pa", nil, nil),
				"a/pa/m.ail":      "module pa/m\n\nimport " + spelling + " (rootSecret)\n\nexport func fromA() -> string = rootSecret()\n",
				"app/ap/main.ail": "module ap/main\n\nimport std/io (println)\nimport pkg/aaa/a/m (fromA)\n\n" +
					"export func main() -> () ! {IO} = println(fromA())\n",
			}, dep{Name: "aaa/a", Dir: "../a"})
			wantCheckError(t, app, filepath.Join("ap", "main.ail"),
				"LDR006: package aaa/a imports \""+spelling+"\", which is a module of the root package acme/app")
		})
	}
}

// fxD: two declared dependencies share prefix "sh" and both have sh/x. The
// root's `sh/x` is ambiguous (both named, sorted), whatever order the manifest
// lists them in.
func TestScope_AmbiguousPrefixIsLDR005(t *testing.T) {
	app := tree(t, map[string]string{
		"app/ailang.toml": pkgToml("acme/app", "", nil, map[string]string{"zzz/two": "../two", "aaa/one": "../one"}),
		"one/ailang.toml": pkgToml("aaa/one", "sh", nil, nil),
		"two/ailang.toml": pkgToml("zzz/two", "sh", nil, nil),
		"one/sh/x.ail":    "module sh/x\n\nexport func who() -> string = \"aaa/one\"\n",
		"two/sh/x.ail":    "module sh/x\n\nexport func who() -> string = \"zzz/two\"\n",
		"app/main.ail":    "module main\n\nimport std/io (println)\nimport sh/x (who)\n\nexport func main() -> () ! {IO} = println(who())\n",
		"app/pick.ail":    "module pick\n\nimport std/io (println)\nimport pkg/zzz/two/x (who)\n\nexport func main() -> () ! {IO} = println(who())\n",
	}, dep{Name: "aaa/one", Dir: "../one"}, dep{Name: "zzz/two", Dir: "../two"})
	wantCheckError(t, app, "main.ail", "LDR005: package acme/app imports \"sh/x\", but its dependencies aaa/one and zzz/two all provide it under module_prefix \"sh\"",
		"(pkg/aaa/one/x)")
	checkFixture(t, filepath.Join(app, "pick.ail")) // the published path is the way out
}

// A dependency that shares its prefix with one of ITS dependencies gets its
// own module (own package before declared dependencies).
func TestScope_OwnPrefixBeforeDependencies(t *testing.T) {
	app := tree(t, map[string]string{
		"app/ailang.toml": pkgToml("acme/app", "", nil, map[string]string{"bbb/two": "../two"}),
		"one/ailang.toml": pkgToml("aaa/one", "px", nil, nil),
		"two/ailang.toml": pkgToml("bbb/two", "px", nil, map[string]string{"aaa/one": "../one"}),
		"one/px/t.ail":    "module px/t\n\nexport func who() -> string = \"aaa/one\"\n",
		"two/px/t.ail":    "module px/t\n\nexport func who() -> string = \"bbb/two\"\n",
		"two/px/use.ail":  "module px/use\n\nimport px/t (who)\n\nexport func twoSays() -> string = who()\n",
		"app/main.ail": "module main\n\nimport std/io (println)\nimport pkg/bbb/two/use (twoSays)\n\n" +
			"export func main() -> () ! {IO} = println(twoSays())\n",
	}, dep{Name: "aaa/one", Dir: "../one"}, dep{Name: "bbb/two", Dir: "../two"})
	result := checkFixture(t, filepath.Join(app, "main.ail"))
	if _, ok := result.Modules["pkg/bbb/two/t"]; !ok {
		t.Fatalf("bbb/two's px/t must be its own; modules %v", sortedKeys(result.Modules))
	}
	if _, wrong := result.Modules["pkg/aaa/one/t"]; wrong {
		t.Fatal("bbb/two's px/t loaded its dependency's module")
	}
}

// fxM: a dependency whose module_prefix is the root's vendor ("acme") imports
// its own acme/app/util; the root acme/app also has util. The bare self-import
// is scoped to the importer, so the dependency gets its own.
func TestScope_BareSelfImportScopedToImporter(t *testing.T) {
	app := tree(t, map[string]string{
		"app/ailang.toml":     pkgToml("acme/app", "", nil, map[string]string{"ddd/d": "../d"}),
		"app/util.ail":        "module acme/app/util\n\nexport func who() -> string = \"ROOT\"\n",
		"d/ailang.toml":       pkgToml("ddd/d", "acme", nil, nil),
		"d/acme/app/util.ail": "module acme/app/util\n\nexport func who() -> string = \"ddd/d\"\n",
		"d/acme/m.ail":        "module acme/m\n\nimport acme/app/util (who)\n\nexport func whoD() -> string = who()\n",
		"app/main.ail": "module acme/app/main\n\nimport std/io (println)\nimport pkg/ddd/d/m (whoD)\n\n" +
			"export func main() -> () ! {IO} = println(whoD())\n",
	}, dep{Name: "ddd/d", Dir: "../d"})
	result := checkFixture(t, filepath.Join(app, "main.ail"))
	want := canonicalFilePath(filepath.Join(filepath.Dir(app), "d", "acme", "app", "util.ail"))
	for id, mod := range result.Modules {
		if strings.HasSuffix(id, "app/util") && canonicalFilePath(mod.File.Path) != want {
			t.Fatalf("ddd/d's acme/app/util loaded %s as %s", mod.File.Path, id)
		}
	}
	if _, ok := result.Modules["pkg/ddd/d/app/util"]; !ok {
		t.Fatalf("ddd/d's util not loaded as pkg/ddd/d/app/util; modules %v", sortedKeys(result.Modules))
	}
}

// fxI: `cc/Init` for cc/init.ail is rejected on every host — on a
// case-insensitive filesystem it would resolve and be a second identity.
func TestScope_CaseMismatchRejected(t *testing.T) {
	app := tree(t, map[string]string{
		"app/ailang.toml": pkgToml("acme/app", "", nil, map[string]string{"ccc/c": "../c"}),
		"c/ailang.toml":   pkgToml("ccc/c", "cc", nil, nil),
		"c/cc/init.ail":   "module cc/init\n\nexport func mk() -> int = 41\n",
		"app/main.ail":    "module main\n\nimport std/io (println)\nimport cc/init (mk)\nimport cc/Init (mk as mk2)\n\nexport func main() -> () ! {IO} = println(show(mk() + mk2()))\n",
	}, dep{Name: "ccc/c", Dir: "../c"})
	_, err := Run(Config{Mode: ModeCheck, NoCache: true}, Source{Filename: filepath.Join(app, "main.ail")})
	if err == nil {
		t.Fatal("cc/Init for cc/init.ail must be rejected")
	}
	msg := err.Error()
	// Case-insensitive disk: our message. Case-sensitive disk: not found at all.
	if !strings.Contains(msg, "the file on disk is spelled \"init.ail\"") && !strings.Contains(msg, "LDR001") {
		t.Fatalf("unexpected error: %v", msg)
	}
}
