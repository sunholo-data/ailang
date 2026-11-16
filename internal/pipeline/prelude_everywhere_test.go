package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/loader"
)

// #2371: the Option/Result prelude (Option, Some, None, Result, Ok, Err) is in
// scope in every module but the standard library's, with the entry-module
// rules: lowest precedence, a module's own declaration of a prelude type or
// constructor drops that prelude module, an explicit import of the name wins.
// Each case is a library module (no main) checked through an importing root.
func TestPreludeInLibraryModules(t *testing.T) {
	const q = "module lib/q\nexport type Q = Some(int) | QN\nexport pure func qmk(i: int) -> Q = if i > 0 then Some(i) else QN\n"
	for _, c := range []struct {
		name, lib string
		want      string // "" = checks
	}{
		{"types and constructors without an import",
			"pure func f(o: Option[int], r: Result[int, string]) -> int = match o { Some(n) => n, None => match r { Ok(m) => m, Err(_) => 0 } }\n" +
				"export pure func run() -> int = f(None, Ok(5))\n", ""},
		{"a helper over Some is generic",
			"export pure func run() -> bool {\n  let isSome = \\o. match o { Some(_) => true, _ => false };\n  isSome(Some(1)) && isSome(Some(\"s\"))\n}\n", ""},
		{"an explicit import of std/option is still valid",
			"import std/option (Option, Some, None)\nexport pure func run() -> Option[int] = Some(1)\n", ""},
		{"a local type declaring Some shadows the Option prelude",
			"type M = Some(int) | Nada\npure func f(m: M) -> int = match m { Some(n) => n, Nada => 0 }\n" +
				"export pure func run() -> int = f(Some(3)) + (match Ok(1) { Ok(n) => n, Err(_) => 0 })\n", ""},
		{"... and drops None with it, as in an entry module",
			"type M = Some(int) | Nada\nexport pure func run(o: Option[int]) -> int = match o { None => 0, _ => 1 }\n", "TC_MATCH_001"},
		{"an explicit import of another type's Some wins",
			"import lib/q (Q, Some, QN)\npure func f(x: Q) -> int = match x { Some(n) => n, QN => 0 }\nexport pure func run() -> int = f(Some(4))\n", ""},
		{"... while Option and None stay the prelude's",
			"import lib/q (Q, Some, QN)\nexport pure func run(o: Option[int]) -> int = match o { Some(n) => n, None => 0 }\n",
			"belongs to ADT 'Q', not 'Option'"},
	} {
		t.Run(c.name, func(t *testing.T) {
			check := byNameFixture(t, map[string]string{
				"lib/q.ail": q,
				"lib/m.ail": "module lib/m\n" + c.lib,
				"main.ail":  "module main\nimport lib/m (run)\nexport pure func go() -> int = 0\n",
			})
			got := check("main.ail")
			switch {
			case c.want == "" && got != "ok":
				t.Fatalf("must check, got %s", got)
			case c.want != "" && !strings.Contains(got, c.want):
				t.Fatalf("want an error containing %q, got %s", c.want, got)
			}
		})
	}
}

// The approved boundary (#2371): exactly the six names, nothing effectful,
// local definitions win, an explicit import is valid without a warning, and a
// library that names none of the six loads neither module.
func TestPreludeBoundaryInLibraryModules(t *testing.T) {
	run := func(t *testing.T, lib string) (Result, error) {
		t.Helper()
		byNameFixture(t, map[string]string{
			"lib/m.ail": "module lib/m\n" + lib,
			"main.ail":  "module main\nimport lib/m (run)\nexport pure func go() -> int = 0\n",
		})
		return runModuleWithCacheDependencies(t.Context(), Config{Mode: ModeCheck}, Source{Filename: "main.ail"}, productionCacheDependencies())
	}
	for _, c := range []struct {
		name, lib, want string // want "" = checks
	}{
		{"a helper function is not in the prelude",
			"export pure func run() -> int = getOrElse(None, 1)\n", "getOrElse"},
		{"println stays entry-only",
			"export func run() -> () ! {IO} = println(\"x\")\n", "println"},
		{"a local type Option is the module's own",
			"type Option[a] = Some(a) | None\npure func f(o: Option[int]) -> int = match o { Some(n) => n, None => 0 }\nexport pure func run() -> int = f(Some(2))\n", ""},
		{"a local type Result drops Ok and Err",
			"type Result = Pending | Done\nexport pure func run() -> int = match Ok(1) { Ok(n) => n, _ => 0 }\n", "Ok"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := run(t, c.lib)
			switch {
			case c.want == "" && err != nil:
				t.Fatalf("must check, got %v", err)
			case c.want != "" && (err == nil || !strings.Contains(err.Error(), c.want)):
				t.Fatalf("want an error naming %q, got %v", c.want, err)
			}
		})
	}
	t.Run("an explicit import is valid without a warning", func(t *testing.T) {
		res, err := run(t, "import std/option (Option, Some, None)\nexport pure func run() -> Option[int] = Some(1)\n")
		if err != nil || len(res.Warnings) != 0 {
			t.Fatalf("want a clean check, got err=%v warnings=%v", err, res.Warnings)
		}
	})
	t.Run("a library naming none of the six loads neither module", func(t *testing.T) {
		res, err := run(t, "-- not Some, None or Ok: a comment\nexport pure func run() -> string = \"Result: none\"\n")
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range loader.PreludeModules() {
			if _, loaded := res.Modules[m]; loaded {
				t.Errorf("%s loaded for a module that never names its types", m)
			}
		}
	})
}

// A library module's compile-cache key covers its implicit imports: its key
// is not the source-only key, a warm run reuses the cold run's entry, and the
// prelude modules are entries of their own.
func TestPreludeLibraryCacheKeyCoversImplicitImports(t *testing.T) {
	src := "module lib/m\nexport pure func run(i: int) -> Option[int] = Some(i)\n"
	byNameFixture(t, map[string]string{
		"lib/m.ail": src,
		"main.ail":  "module main\nimport lib/m (run)\nexport pure func go() -> int = 0\n",
	})
	manifestPath := filepath.Join(os.Getenv("AILANG_CACHE_DIR"), "compile", "manifest.json")
	var keys []string
	for range 2 { // cold, then warm
		if _, err := runModuleWithCacheDependencies(t.Context(), Config{Mode: ModeCheck}, Source{Filename: "main.ail"}, productionCacheDependencies()); err != nil {
			t.Fatal(err)
		}
		m := readCacheManifest(t, manifestPath)
		entry := m.Entries["lib/m"]
		if entry == nil || m.Entries["std/option"] == nil {
			t.Fatalf("manifest lacks lib/m or std/option: %v", m.Entries)
		}
		keys = append(keys, entry.CacheKey)
	}
	if keys[0] != keys[1] {
		t.Errorf("warm key %s differs from cold key %s", keys[1], keys[0])
	}
	if keys[0] == ModuleCacheKey(testCompilerIdentity(t), src, nil) {
		t.Error("lib/m's key is the source-only key: its implicit std/option import is not covered")
	}
}
